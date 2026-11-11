package service_test

import (
	"reflect"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) reverse(actor domain.Actor, id string, amount int64, reason string) (*domain.Transaction, error) {
	return w.Svc.ReverseTransaction(w.Ctx, actor, w.net.ID, service.ReverseInput{TxID: id, Amount: amount, Reason: reason, IdempotencyKey: w.Key()})
}

func (w *world) entries(id string) []*domain.Entry {
	return tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, id)).Entries
}

// DOM-019: after a full reversal each balance equals its pre-trade value and
// the original's entries are untouched.
func TestFullReversal(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	origEntries := w.entries(tx.ID)
	rev := tu.Must(w.reverse(w.admin, tx.ID, 0, "wrong amount recorded"))
	if rev.Kind != domain.KindReversal || rev.OriginalTxID != tx.ID || rev.Amount != 2500 || rev.Status != domain.TxCompleted {
		t.Fatalf("reversal = %+v", rev)
	}
	if w.balance(w.a).Balance != 0 || w.balance(w.b).Balance != 0 {
		t.Errorf("balances not restored: %d %d", w.balance(w.a).Balance, w.balance(w.b).Balance)
	}
	if got := w.entries(tx.ID); !reflect.DeepEqual(got, origEntries) {
		t.Errorf("original entries changed:\n%v\n%v", origEntries, got)
	}
	orig := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, tx.ID)).Tx
	if orig.Status != domain.TxReversed || orig.ReversedTotal != 2500 {
		t.Errorf("original = %s reversed=%d", orig.Status, orig.ReversedTotal)
	}
	// terminal: nothing more to reverse; a reversal cannot be reversed
	_, err := w.reverse(w.admin, tx.ID, 1, "again")
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.reverse(w.admin, rev.ID, 0, "reverse the reversal")
	wantCode(t, err, domain.ErrInvalidState)
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

// DOM-020: 25.00 trade: reverse 20.00 -> completed, total 20.00; +10.00 refused; +5.00 -> reversed.
func TestPartialReversalCumulativeCap(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	tu.Must(w.reverse(w.admin, tx.ID, 2000, "partial"))
	got := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, tx.ID)).Tx
	if got.Status != domain.TxCompleted || got.ReversedTotal != 2000 {
		t.Fatalf("after partial: %s %d", got.Status, got.ReversedTotal)
	}
	_, err := w.reverse(w.admin, tx.ID, 1000, "too much")
	wantCode(t, err, domain.ErrValidation)
	tu.Must(w.reverse(w.admin, tx.ID, 500, "rest"))
	got = tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, tx.ID)).Tx
	if got.Status != domain.TxReversed || got.ReversedTotal != 2500 {
		t.Fatalf("after full: %s %d", got.Status, got.ReversedTotal)
	}
	if w.balance(w.a).Balance != 0 {
		t.Error("cumulative reversals should restore the balance")
	}
}

func TestReversalAuthorization(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	_, err := w.reverse(w.a, tx.ID, 0, "I want my money back") // party, not admin
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.reverse(w.admin, tx.ID, 0, "")
	wantCode(t, err, domain.ErrValidation)
	// DOM-037: an administrator who is a party cannot reverse
	own := w.Trade(w.net.ID, w.admin, w.b, w.adminM, w.bM, 100)
	_, err = w.reverse(w.admin, own.ID, 0, "undo my own trade")
	wantCode(t, err, domain.ErrForbidden)
	// but a different administrator can
	admin2, m2 := w.Member2(w.admin, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, m2.ID, domain.RoleAdmin))
	tu.Must(w.reverse(admin2, own.ID, 0, "second admin undoes it"))
}

// Member2 is a helper: a new active member.
func (w *world) Member2(admin domain.Actor, login string) (domain.Actor, *domain.Member) {
	return w.Member(admin, w.net, login)
}

// DOM-021 / LED-5: a reversal that would push the debited account below its limit is refused.
func TestReversalBlockedByLimit(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 2500 })
	c, cM := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500) // B = +2500
	w.Trade(w.net.ID, w.b, c, w.bM, cM, 5000)           // B spends it and all its credit: -2500
	_, err := w.reverse(w.admin, tx.ID, 2500, "refund")
	wantCode(t, err, domain.ErrInsufficientCredit)
	got := tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, tx.ID))
	if got.Tx.ReversedTotal != 0 || got.Tx.Status != domain.TxCompleted {
		t.Errorf("failed reversal changed the original: %+v", got.Tx)
	}
	if w.balance(w.b).Balance != -2500 {
		t.Errorf("balance changed: %d", w.balance(w.b).Balance)
	}
}
