package service_test

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-007: over-subscribed concurrent accepts against one payer, repeated for
// 100 random interleavings (random amounts, counts, limits and goroutine start
// order). In every round: the payer is never observed below its limit, the
// final balance is the sum of the accepted trades, nothing is lost or posted
// twice, and every refused trade would not fit even after the accepted ones,
// so the outcome is consistent with some serial order.
func TestQA007OverSubscriptionInterleavings(t *testing.T) {
	rounds := 100
	switch {
	case testing.Short():
		rounds = 10
	case raceOn:
		rounds = 8
	}
	for r := 0; r < rounds; r++ {
		seed := int64(7000 + r)
		if !qa007Round(t, seed) {
			t.Fatalf("round with seed %d failed; rerun this seed to reproduce", seed)
		}
	}
}

func qa007Round(t *testing.T, seed int64) bool {
	rng := rand.New(rand.NewSource(seed))
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 1_000_000 })
	n := 8 + rng.Intn(17)
	amounts := make([]int64, n)
	ids := make([]string, n)
	var total int64
	for i := range ids {
		amounts[i] = 100 + rng.Int63n(1900)
		total += amounts[i]
		ids[i] = tu.Must(w.propose(w.a, w.bM, service.ProposerPays, amounts[i])).ID
	}
	limit := total/4 + rng.Int63n(total/2) // always over-subscribed
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &limit, Reason: "qa-007"}))

	order := rng.Perm(n)
	accepted := make([]atomic.Bool, n)
	refused := make([]atomic.Bool, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	stop := make(chan struct{})
	var lowest atomic.Int64
	observer := make(chan struct{})
	go func() { // reads the payer's balance while the accepts race
		defer close(observer)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if b, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err == nil && b.Balance < lowest.Load() {
				lowest.Store(b.Balance)
			}
		}
	}()
	for _, i := range order {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: ids[i], IdempotencyKey: "k-" + ids[i]})
			switch {
			case err == nil:
				accepted[i].Store(true)
			case errors.Is(err, domain.ErrInsufficientCredit):
				refused[i].Store(true)
			default:
				t.Errorf("seed %d: unexpected error: %v", seed, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(stop)
	<-observer

	var used int64
	okCount := 0
	for i := range ids {
		if accepted[i].Load() == refused[i].Load() {
			t.Errorf("seed %d: trade %d accepted=%v refused=%v", seed, i, accepted[i].Load(), refused[i].Load())
		}
		if accepted[i].Load() {
			used += amounts[i]
			okCount++
		}
	}
	if used > limit {
		t.Errorf("seed %d: accepted %d over limit %d", seed, used, limit)
	}
	for i := range ids {
		if refused[i].Load() && amounts[i] <= limit-used {
			t.Errorf("seed %d: trade of %d refused although %d credit was still free", seed, amounts[i], limit-used)
		}
	}
	if got := w.balance(w.a).Balance; got != -used {
		t.Errorf("seed %d: payer balance %d, want %d", seed, got, -used)
	}
	if lowest.Load() < -limit {
		t.Errorf("seed %d: balance %d observed below the limit %d", seed, lowest.Load(), -limit)
	}
	done := tu.Must(w.Svc.ListTransactions(w.Ctx, w.b, w.net.ID, []domain.TxStatus{domain.TxCompleted}, service.Page{Limit: 500}))
	if len(done) != okCount {
		t.Errorf("seed %d: %d completed trades, %d accepts succeeded", seed, len(done), okCount)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Errorf("seed %d: ledger: %v", seed, rep.Findings)
	}
	return !t.Failed()
}
