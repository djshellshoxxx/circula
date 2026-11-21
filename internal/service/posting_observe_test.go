package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type postingCounts struct {
	countObs
	mu       sync.Mutex
	attempts int
	failures map[string]int
}

func (o *postingCounts) PostingAttempted(n int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.attempts += n
}

func (o *postingCounts) PostingFailed(n int, reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.failures == nil {
		o.failures = make(map[string]int)
	}
	o.failures[reason] += n
}

func (o *postingCounts) require(t *testing.T, attempts, committed, replayed int, reason string, failures int) {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	total := 0
	for _, n := range o.failures {
		total += n
	}
	if o.attempts != attempts || o.posted.Load() != int64(committed) || o.replayed.Load() != int64(replayed) ||
		o.failures[reason] != failures || total != failures {
		t.Fatalf("attempts=%d committed=%d replays=%d failures=%v", o.attempts, o.posted.Load(), o.replayed.Load(), o.failures)
	}
}

// OPS-014 / ARC-022: a successful attempt counts once, a durable replay adds
// no posting attempt, and a credit refusal changes neither balance nor success.
func TestPostingObserverCountsCommitCreditRefusalAndReplay(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 2500 })
	o := &postingCounts{}
	w.Svc.SetObserver(o)
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	in := service.TxRef{TxID: p.ID, IdempotencyKey: w.Key()}
	tu.Must(w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, in))
	o.require(t, 1, 1, 0, "", 0)
	tu.Must(w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, in))
	o.require(t, 1, 1, 1, "", 0)

	request := tu.Must(w.propose(w.b, w.aM, service.ProposerIsPaid, 2400))
	lower := int64(2000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &lower, Reason: "capacity changed before acceptance"}))
	before := dbChecksum(t, w.Env)
	_, err := w.accept(w.a, request.ID)
	wantCode(t, err, domain.ErrInsufficientCredit)
	o.require(t, 2, 1, 1, "credit_limit", 1)
	if after := dbChecksum(t, w.Env); after != before {
		t.Fatal("refused posting changed stored state")
	}
}

type postingUnitFailureStore struct {
	store.Store
	fail bool
	err  error
}

func (s *postingUnitFailureStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		if s.fail {
			return s.err
		}
		return nil
	})
}

// OPS-014: failure after postTx must count as an unsuccessful attempt, never
// as a committed posting; the ordinary retry can still complete exactly once.
func TestPostingObserverCountsEnclosingUnitRollbackAndRetry(t *testing.T) {
	fault := &postingUnitFailureStore{err: errors.New("deliberate unit failure")}
	e := tu.NewEnvWith(t, func(st store.Store) store.Store { fault.Store = st; return fault })
	a := e.Person("admin@example.org")
	net, _ := e.Network(a, "metrics", nil)
	payer, _ := e.Member(a, net, "payer@example.org")
	payee, qm := e.Member(a, net, "payee@example.org")
	p := tu.Must(e.Svc.ProposeTransaction(e.Ctx, payer, net.ID, service.ProposeInput{
		CounterpartyID: qm.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: e.Key()}))
	o := &postingCounts{}
	e.Svc.SetObserver(o)
	in := service.TxRef{TxID: p.ID, IdempotencyKey: e.Key()}
	before := dbChecksum(t, e)
	fault.fail = true
	if _, err := e.Svc.AcceptTransaction(e.Ctx, payee, net.ID, in); !errors.Is(err, fault.err) {
		t.Fatalf("expected controlled rollback, got %v", err)
	}
	o.require(t, 1, 0, 0, "internal", 1)
	if after := dbChecksum(t, e); after != before {
		t.Fatal("failed write unit changed stored state")
	}
	fault.fail = false
	tu.Must(e.Svc.AcceptTransaction(e.Ctx, payee, net.ID, in))
	o.require(t, 2, 1, 0, "internal", 1)
}

// A dispute can commit its decision while the reversal is posting_pending.
// OPS-014 must preserve the failed financial attempt within that successful unit.
func TestPostingObserverCountsPendingDisputeRefusal(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 2500 })
	reviewer, _ := w.reviewer()
	c, cm := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	w.Trade(w.net.ID, w.b, c, w.bM, cm, 5000)
	d := tu.Must(w.open(w.a, tx.ID))
	o := &postingCounts{}
	w.Svc.SetObserver(o)
	got := tu.Must(w.resolve(reviewer, d.ID, domain.OutcomeUpheld, 0, "not delivered"))
	if got.Status != domain.DisputePostingPending {
		t.Fatalf("dispute status=%s", got.Status)
	}
	o.require(t, 1, 0, 0, "credit_limit", 1)
}
