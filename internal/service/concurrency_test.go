package service_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// LED-5 / DOM-014: many accepts racing for the last available credit. Proposals
// are made while the limit is high (each passes the soft reservation guard); the
// limit is then reduced so that only 7 of 25 can post. Exactly 7 may succeed,
// the rest are refused and stay proposed, and no balance ever breaches the limit.
func TestParallelAcceptsNeverBreachCreditLimit(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 50000 })
	const n, amt = 25, int64(2000)
	ids := make([]string, n)
	for i := range ids {
		ids[i] = tu.Must(w.propose(w.a, w.bM, service.ProposerPays, amt)).ID
	}
	limit := int64(15000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &limit, Reason: "race"}))

	var ok, refused atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-start
			_, err := w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: id, IdempotencyKey: "k-" + id})
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, domain.ErrInsufficientCredit):
				refused.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}(id)
	}
	close(start)
	wg.Wait()
	if ok.Load() != 7 || refused.Load() != n-7 {
		t.Fatalf("succeeded=%d refused=%d, want 7 and %d", ok.Load(), refused.Load(), n-7)
	}
	if got := w.balance(w.a).Balance; got != -7*amt || got < -limit {
		t.Errorf("payer balance = %d, limit %d", got, limit)
	}
	proposed := tu.Must(w.Svc.ListTransactions(w.Ctx, w.b, w.net.ID, []domain.TxStatus{domain.TxProposed}, service.Page{Limit: 500}))
	if len(proposed) != n-7 {
		t.Errorf("%d proposals remain, want %d", len(proposed), n-7)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Errorf("ledger: %v", rep.Findings)
	}
}

// Two racing accepts of the same proposal by different keys: exactly one posts.
func TestParallelDoubleAcceptSingleSuccess(t *testing.T) {
	w := newWorld(t, nil)
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1000))
	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := w.accept(w.b, p.ID); err == nil {
				ok.Add(1)
			} else if !errors.Is(err, domain.ErrInvalidState) {
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("%d accepts succeeded, want 1", ok.Load())
	}
	if w.balance(w.a).Balance != -1000 {
		t.Errorf("posted more than once: %d", w.balance(w.a).Balance)
	}
}

// DOM-016 acceptance: ten concurrent identical requests with one key produce
// one transaction and one entry pair.
func TestIdempotentConcurrentPropose(t *testing.T) {
	w := newWorld(t, nil)
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: "same-key"}
	ids := make(chan string, 10)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in)
			if err != nil {
				t.Errorf("propose: %v", err)
				return
			}
			ids <- tx.ID
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		seen[id] = true
	}
	if len(seen) != 1 {
		t.Fatalf("got %d distinct transactions, want 1", len(seen))
	}
	if all := tu.Must(w.Svc.ListTransactions(w.Ctx, w.a, w.net.ID, nil, service.Page{})); len(all) != 1 {
		t.Errorf("%d transactions stored", len(all))
	}
}
