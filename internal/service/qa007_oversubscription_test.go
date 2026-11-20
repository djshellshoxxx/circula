package service_test

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// qa007Rounds is the number of random interleavings QA-007 requires per CI
// configuration (at least 100). CIRCULA_QA007_ROUNDS raises it for long runs.
func qa007Rounds(t *testing.T) int {
	n := 100
	if v := os.Getenv("CIRCULA_QA007_ROUNDS"); v != "" {
		x, err := strconv.Atoi(v)
		if err != nil || x < 100 {
			t.Fatalf("CIRCULA_QA007_ROUNDS=%q: want an integer >= 100", v)
		}
		n = x
	}
	return n
}

// QA-007 over-subscription: in each of at least 100 seeded interleavings a
// fresh payer has N proposals open whose total exceeds its credit, and all N
// are accepted at once, by the payee (payer-initiated) or by the payer itself
// (payee-initiated), with seeded start jitter. An observer samples the payer's
// balance throughout. Per round:
//   - the balance is never seen below the negative limit (LED-5);
//   - successes fit in the credit and every refusal was for an amount that no
//     longer fit, so the outcome matches a serial order (rounds with equal
//     amounts also check the exact number that fit);
//   - nothing is lost or double-applied: each success has exactly two entries,
//     each refusal none and stays proposed, and the final balance is minus the
//     sum of the accepted amounts (QA-002).
func TestQA007OversubscriptionInterleavings(t *testing.T) {
	rounds := qa007Rounds(t)
	// The open-request caps are anti-spam guards, not under test here.
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 1_000_000 }, func(c *service.Config) { c.MaxOpenRequestsPerPair, c.MaxOpenProposalsPerMember = 1_000_000, 1_000_000 })
	n := 12
	if raceOn {
		n = 6
	}
	db := w.Store.DB()
	var equalRounds, mixedRounds int
	for r := 0; r < rounds; r++ {
		seed := int64(7000 + r)
		rng := rand.New(rand.NewSource(seed))
		what := fmt.Sprintf("round %d (seed %d)", r, seed)
		payer, payerM := w.Member2(w.admin, fmt.Sprintf("qa007-payer-%d@example.org", r))
		equal := r%2 == 0
		amounts := make([]int64, n)
		var total int64
		for i := range amounts {
			amounts[i] = 1000
			if !equal {
				amounts[i] = 100 + rng.Int63n(1900)
			}
			total += amounts[i]
		}
		// Credit for roughly 40-70 percent of the total: always over-subscribed.
		limit := total * int64(40+rng.Intn(31)) / 100
		if equal {
			limit = 1000*int64(n/2) + rng.Int63n(1000) // fits exactly n/2
			equalRounds++
		} else {
			mixedRounds++
		}
		type req struct {
			id       string
			amount   int64
			acceptBy domain.Actor
		}
		reqs := make([]req, n)
		for i := range reqs {
			if rng.Intn(2) == 0 { // payer proposes to pay B; B accepts
				tx := tu.Must(w.propose(payer, w.bM, service.ProposerPays, amounts[i]))
				reqs[i] = req{tx.ID, amounts[i], w.b}
			} else { // B asks the payer to pay; the payer accepts
				tx := tu.Must(w.propose(w.b, payerM, service.ProposerIsPaid, amounts[i]))
				reqs[i] = req{tx.ID, amounts[i], payer}
			}
		}
		tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: payerM.ID, Limit: &limit, Reason: "qa007"}))

		jitter := make([]time.Duration, n)
		for i := range jitter {
			jitter[i] = time.Duration(rng.Intn(300)) * time.Microsecond
		}
		var stop atomic.Bool
		var minSeen atomic.Int64
		observed := make(chan struct{})
		go func() {
			defer close(observed)
			for !stop.Load() {
				var bal int64
				if err := db.QueryRow(`SELECT balance FROM accounts WHERE member_id = ?`, payerM.ID).Scan(&bal); err == nil && bal < minSeen.Load() {
					minSeen.Store(bal)
				}
				runtime.Gosched()
			}
		}()
		ok := make([]bool, n)
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range reqs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				time.Sleep(jitter[i])
				_, err := w.Svc.AcceptTransaction(w.Ctx, reqs[i].acceptBy, w.net.ID, service.TxRef{TxID: reqs[i].id, IdempotencyKey: "qa007-" + reqs[i].id})
				switch {
				case err == nil:
					ok[i] = true
				case errors.Is(err, domain.ErrInsufficientCredit):
				default:
					t.Errorf("%s: accept %d: unexpected error %v", what, i, err)
				}
			}(i)
		}
		close(start)
		wg.Wait()
		stop.Store(true)
		<-observed

		var accepted int64
		var succeeded int
		for i, rq := range reqs {
			var entries int
			if err := db.QueryRow(`SELECT COUNT(*) FROM entries WHERE tx_id = ?`, rq.id).Scan(&entries); err != nil {
				t.Fatal(err)
			}
			st := w.txStatus(rq.id).Status
			if ok[i] {
				succeeded++
				accepted += rq.amount
				if entries != 2 || st != domain.TxCompleted {
					t.Errorf("%s: accepted tx %s: %d entries, status %s", what, rq.id, entries, st)
				}
			} else if entries != 0 || st != domain.TxProposed {
				t.Errorf("%s: refused tx %s: %d entries, status %s", what, rq.id, entries, st)
			}
		}
		if accepted > limit {
			t.Fatalf("%s: accepted %d over limit %d", what, accepted, limit)
		}
		// Debits only lower the remaining credit, so a refusal is consistent
		// with a serial order only if its amount exceeds what remains at the end.
		for i, rq := range reqs {
			if !ok[i] && rq.amount <= limit-accepted {
				t.Errorf("%s: refused %d although %d credit remains", what, rq.amount, limit-accepted)
			}
		}
		if equal && succeeded != n/2 {
			t.Errorf("%s: %d equal-amount postings succeeded, %d fit", what, succeeded, n/2)
		}
		if succeeded == 0 || succeeded == n {
			t.Errorf("%s: %d of %d succeeded; the scenario must be over-subscribed and partly fill", what, succeeded, n)
		}
		var bal int64
		if err := db.QueryRow(`SELECT balance FROM accounts WHERE member_id = ?`, payerM.ID).Scan(&bal); err != nil {
			t.Fatal(err)
		}
		if bal != -accepted {
			t.Errorf("%s: final balance %d, want %d", what, bal, -accepted)
		}
		if m := minSeen.Load(); m < -limit {
			t.Errorf("%s: observed balance %d below limit %d", what, m, -limit)
		}
		if t.Failed() {
			t.FailNow()
		}
	}
	sumInvariants(t, w, "after QA-007 rounds")
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("ledger: %v", rep.Findings)
	}
	t.Logf("QA-007: %d interleavings (%d equal-amount, %d mixed), %d concurrent accepts each", rounds, equalRounds, mixedRounds, n)
}
