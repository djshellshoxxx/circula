package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestIdempotentRetryReturnsOriginal(t *testing.T) {
	w := newWorld(t, nil)
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 700, IdempotencyKey: "k1"}
	first := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	again := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	if first.ID != again.ID {
		t.Fatalf("retry created %s, original %s", again.ID, first.ID)
	}
	if got := w.balance(w.a); got.ReservedOutgoing != 700 {
		t.Errorf("reserved twice? %d", got.ReservedOutgoing)
	}

	// accept retry with the same key returns the original result and posts once
	ak := service.TxRef{TxID: first.ID, IdempotencyKey: "acc"}
	done := tu.Must(w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, ak))
	retry := tu.Must(w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, ak))
	if done.ID != retry.ID || retry.Status != domain.TxCompleted {
		t.Fatalf("retry result differs: %+v", retry)
	}
	if got := w.balance(w.a).Balance; got != -700 {
		t.Errorf("balance = %d, posted twice?", got)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK || rep.LedgerSeq != 1 {
		t.Errorf("report %+v", rep)
	}
}

// DOM-016: reusing a key with a different payload is refused and changes nothing.
func TestIdempotencyKeyReusedWithDifferentPayload(t *testing.T) {
	w := newWorld(t, nil)
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 700, IdempotencyKey: "k1"}
	tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	in.Amount = 800
	_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in)
	wantCode(t, err, domain.ErrKeyReused)
	if got := tu.Must(w.Svc.ListTransactions(w.Ctx, w.a, w.net.ID, nil, service.Page{})); len(got) != 1 || got[0].Amount != 700 {
		t.Errorf("state changed: %+v", got)
	}
}

func TestIdempotencyKeyRequiredAndScoped(t *testing.T) {
	w := newWorld(t, nil)
	_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 1})
	wantCode(t, err, domain.ErrKeyRequired)

	// keys are scoped per actor: the same key string from another member is independent
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 5, IdempotencyKey: "shared"}
	t1 := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	in2 := service.ProposeInput{CounterpartyID: w.aM.ID, Direction: service.ProposerPays, Amount: 5, IdempotencyKey: "shared"}
	t2 := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.b, w.net.ID, in2))
	if t1.ID == t2.ID {
		t.Error("idempotency keys must be scoped to the acting principal")
	}
}

// API-025: a failed command is not recorded, so the key is reusable.
func TestFailedCommandDoesNotConsumeKey(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100 })
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 500, IdempotencyKey: "k"}
	_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in)
	wantCode(t, err, domain.ErrInsufficientCredit)
	in.Amount = 50
	if _, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in); err != nil {
		t.Fatalf("key should be reusable after a failure: %v", err)
	}
}

// DAT-028: expired idempotency records free the key.
func TestIdempotencyRecordExpires(t *testing.T) {
	w := newWorld(t, nil)
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 5, IdempotencyKey: "k"}
	first := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	w.Clock.Advance(8 * 24 * time.Hour)
	second := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	if first.ID == second.ID {
		t.Error("after retention the key is free and a new transaction results")
	}
}

func TestOtherCommandsAreIdempotent(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)

	adj := service.AdjustInput{MemberID: w.aM.ID, Amount: 300, Direction: service.AdjustCredit, Reason: "fix", IdempotencyKey: "adj"}
	a1 := tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, adj))
	a2 := tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, adj))
	if a1.ID != a2.ID || w.balance(w.a).Balance != -700 {
		t.Errorf("adjustment replay: %s vs %s, balance %d", a1.ID, a2.ID, w.balance(w.a).Balance)
	}

	rev := service.ReverseInput{TxID: tx.ID, Amount: 400, Reason: "partial", IdempotencyKey: "rev"}
	r1 := tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, rev))
	r2 := tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, rev))
	if r1.ID != r2.ID {
		t.Error("reversal replay created a second reversal")
	}
	v := tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, tx.ID))
	if v.Tx.ReversedTotal != 400 {
		t.Errorf("reversed total = %d", v.Tx.ReversedTotal)
	}

	od := service.OpenDisputeInput{TxID: tx.ID, Category: "quality", IdempotencyKey: "dsp"}
	d1 := tu.Must(w.Svc.OpenDispute(w.Ctx, w.a, w.net.ID, od))
	d2 := tu.Must(w.Svc.OpenDispute(w.Ctx, w.a, w.net.ID, od))
	if d1.ID != d2.ID {
		t.Error("dispute replay created a second dispute")
	}

	l := service.ListingInput{Type: domain.ListingOffer, Title: "x", IdempotencyKey: "lst"}
	l1 := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, l))
	l2 := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, l))
	if l1.ID != l2.ID {
		t.Error("listing replay created a second listing")
	}
}
