package service_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type acctSnap struct {
	bal, limit int64
	system     bool
}

// snapshot reads accounts with raw SQL, independently of the service.
func snapshot(t *testing.T, w *world) map[string]acctSnap {
	t.Helper()
	rows, err := w.Store.DB().Query(`SELECT id, balance, is_system, limit_override, limit_override_until FROM accounts WHERE network_id = ?`, w.net.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]acctSnap{}
	now := w.Clock.Now().UnixMilli()
	for rows.Next() {
		var id string
		var bal int64
		var sys int
		var lim, until *int64
		if err := rows.Scan(&id, &bal, &sys, &lim, &until); err != nil {
			t.Fatal(err)
		}
		l := w.net.Policy.DefaultCreditLimit
		if lim != nil && (until == nil || now < *until) {
			l = *lim
		}
		out[id] = acctSnap{bal: bal, limit: l, system: sys == 1}
	}
	return out
}

// checkInvariants verifies LED-1 and LED-3 with queries that do not use the service.
func checkInvariants(t *testing.T, w *world, ctx string) {
	t.Helper()
	db := w.Store.DB()
	var sumBal, sumEnt int64
	if err := db.QueryRow(`SELECT COALESCE(SUM(balance),0) FROM accounts WHERE network_id = ?`, w.net.ID).Scan(&sumBal); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM entries WHERE network_id = ?`, w.net.ID).Scan(&sumEnt); err != nil {
		t.Fatal(err)
	}
	if sumBal != 0 || sumEnt != 0 {
		t.Fatalf("%s: LED-1 broken: balances %d entries %d", ctx, sumBal, sumEnt)
	}
	var bad int
	if err := db.QueryRow(`SELECT COUNT(*) FROM accounts a WHERE a.balance <> (SELECT COALESCE(SUM(e.amount),0) FROM entries e WHERE e.account_id = a.id)`).Scan(&bad); err != nil || bad != 0 {
		t.Fatalf("%s: LED-3 broken for %d accounts (err=%v)", ctx, bad, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM (SELECT tx_id FROM entries GROUP BY tx_id HAVING COUNT(*) <> 2 OR SUM(amount) <> 0)`).Scan(&bad); err != nil || bad != 0 {
		t.Fatalf("%s: %d transactions without two balanced entries", ctx, bad)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE status IN ('completed','disputed','reversed') AND id NOT IN (SELECT tx_id FROM entries)`).Scan(&bad); err != nil || bad != 0 {
		t.Fatalf("%s: %d posted transactions without entries", ctx, bad)
	}
}

// LED-1, LED-3, LED-5 under random command sequences, including failed and refused ones.
func TestRandomisedLedgerInvariants(t *testing.T) {
	var reversals, disputes, trades int
	for seed := int64(1); seed <= 6; seed++ {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			steps := 400
			if raceOn { // the race detector slows the pure-Go driver ~10x
				steps = 100
			}
			r, d, tr := runRandom(t, seed, steps)
			reversals, disputes, trades = reversals+r, disputes+d, trades+tr
		})
	}
	// guard against a generator that never reaches the interesting paths
	if reversals == 0 || disputes == 0 || trades < 20 {
		t.Errorf("weak coverage: %d trades, %d reversals, %d disputes", trades, reversals, disputes)
	}
}

func runRandom(t *testing.T, seed int64, steps int) (reversals, disputes, trades int) {
	rng := rand.New(rand.NewSource(seed))
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000; p.DisputeWindowDays = 5; p.ProposalExpiryDays = 2 })
	rev, _ := w.reviewer()
	actors := []domain.Actor{w.a, w.b}
	members := []*domain.Member{w.aM, w.bM}
	for i := 0; i < 4; i++ {
		a, m := w.Member2(w.admin, fmt.Sprintf("m%d@example.org", i))
		actors, members = append(actors, a), append(members, m)
	}
	actorOf := map[string]domain.Actor{}
	for i, m := range members {
		actorOf[m.ID] = actors[i]
	}
	pick := func() int { return rng.Intn(len(members)) }
	txIn := func(st ...domain.TxStatus) *domain.Transaction {
		l := tu.Must(w.Svc.AdminListTransactions(w.Ctx, w.admin, w.net.ID, "", st, service.Page{Limit: 200}))
		if len(l) == 0 {
			return nil
		}
		return l[rng.Intn(len(l))]
	}
	check := func(err error, what string) {
		if err != nil && domain.CodeOf(err) == domain.CodeInternal {
			t.Fatalf("step %s: internal error: %v", what, err)
		}
	}
	for step := 0; step < steps; step++ {
		before := snapshot(t, w)
		op := rng.Intn(14)
		what := fmt.Sprintf("seed%d/step%d/op%d", seed, step, op)
		switch op {
		case 0, 1, 2: // propose (weighted up)
			i, j := pick(), pick()
			dir := service.ProposerPays
			if rng.Intn(2) == 0 {
				dir = service.ProposerIsPaid
			}
			_, err := w.Svc.ProposeTransaction(w.Ctx, actors[i], w.net.ID, service.ProposeInput{CounterpartyID: members[j].ID,
				Direction: dir, Amount: 1 + rng.Int63n(3000), IdempotencyKey: w.Key()})
			check(err, what)
		case 3, 4, 5: // accept
			if tx := txIn(domain.TxProposed); tx != nil {
				cp := tx.PayeeMemberID
				if tx.CreatedBy == tx.PayeeMemberID {
					cp = tx.PayerMemberID
				}
				_, err := w.Svc.AcceptTransaction(w.Ctx, actorOf[cp], w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: w.Key()})
				check(err, what)
			}
		case 6: // decline or cancel
			if tx := txIn(domain.TxProposed); tx != nil {
				if rng.Intn(2) == 0 {
					cp := tx.PayeeMemberID
					if tx.CreatedBy == tx.PayeeMemberID {
						cp = tx.PayerMemberID
					}
					_, err := w.Svc.DeclineTransaction(w.Ctx, actorOf[cp], w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: w.Key()})
					check(err, what)
				} else {
					_, err := w.Svc.CancelTransaction(w.Ctx, actorOf[tx.CreatedBy], w.net.ID, service.CancelInput{TxID: tx.ID, IdempotencyKey: w.Key()})
					check(err, what)
				}
			}
		case 7: // time passes, sweep
			w.Clock.Advance(time.Duration(rng.Int63n(int64(50 * time.Hour))))
			_, err := w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("sweep"), 100)
			check(err, what)
		case 8: // administrative (partial) reversal
			if tx := txIn(domain.TxCompleted); tx != nil && tx.Kind != domain.KindReversal {
				amt := int64(0)
				if rng.Intn(2) == 0 {
					amt = 1 + rng.Int63n(tx.Amount)
				}
				_, err := w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: tx.ID, Amount: amt, Reason: "r", IdempotencyKey: w.Key()})
				check(err, what)
			}
		case 9: // adjustment
			m := members[pick()]
			dir := service.AdjustCredit
			if rng.Intn(2) == 0 {
				dir = service.AdjustDebit
			}
			_, err := w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: m.ID, Amount: 1 + rng.Int63n(4000), Direction: dir, Reason: "r", IdempotencyKey: w.Key()})
			check(err, what)
		case 10: // open dispute
			if tx := txIn(domain.TxCompleted); tx != nil && tx.Kind == domain.KindTrade {
				who := tx.PayerMemberID
				if rng.Intn(2) == 0 {
					who = tx.PayeeMemberID
				}
				_, err := w.Svc.OpenDispute(w.Ctx, actorOf[who], w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other", IdempotencyKey: w.Key()})
				check(err, what)
			}
		case 11: // resolve or retry
			ds := tu.Must(w.Svc.ListDisputes(w.Ctx, w.admin, w.net.ID, nil, service.Page{Limit: 100}))
			if len(ds) > 0 {
				d := ds[rng.Intn(len(ds))]
				if d.Status == domain.DisputePostingPending {
					_, err := w.Svc.RetryDisputePosting(w.Ctx, rev, w.net.ID, d.ID)
					check(err, what)
					break
				}
				outcome := []domain.DisputeOutcome{domain.OutcomeDismissed, domain.OutcomeUpheld, domain.OutcomePartiallyUpheld}[rng.Intn(3)]
				var amt int64
				if outcome == domain.OutcomePartiallyUpheld {
					amt = 1 + rng.Int63n(1500)
				}
				_, err := w.Svc.ResolveDispute(w.Ctx, rev, w.net.ID, service.ResolveInput{DisputeID: d.ID, Outcome: outcome, Amount: amt, Rationale: "r", IdempotencyKey: w.Key()})
				check(err, what)
			}
		case 12: // limit change
			l := rng.Int63n(8000)
			err := w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: members[pick()].ID, Limit: &l, Reason: "r"})
			check(err, what)
		case 13: // suspend / restore
			m := members[pick()]
			if rng.Intn(2) == 0 {
				check(w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, m.ID, "r"), what)
			} else {
				check(w.Svc.RestoreMember(w.Ctx, w.admin, w.net.ID, m.ID, "r"), what)
			}
		}
		// LED-5: any non-system balance that went down must not be below the limit in force.
		after := snapshot(t, w)
		for id, a := range after {
			if b := before[id]; a.bal < b.bal && !a.system && a.bal < -a.limit {
				t.Fatalf("%s: LED-5 breached on %s: %d -> %d with limit %d", what, id, b.bal, a.bal, a.limit)
			}
		}
		checkInvariants(t, w, what)
		if step%30 == 29 {
			if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
				t.Fatalf("%s: VerifyLedger: %v", what, rep.Findings)
			}
		}
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("final VerifyLedger: %v", rep.Findings)
	}
	db := w.Store.DB()
	_ = db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind = 'reversal'`).Scan(&reversals)
	_ = db.QueryRow(`SELECT COUNT(*) FROM disputes`).Scan(&disputes)
	_ = db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind = 'trade' AND status IN ('completed','disputed','reversed')`).Scan(&trades)
	t.Logf("seed %d: %d posted trades, %d reversals, %d disputes, ledger_seq=%d", seed, trades, reversals, disputes, w.balance(w.a).LedgerSeq)
	return
}
