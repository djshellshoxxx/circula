package service_test

import (
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// UX-017: ReversalsOf lists what reversed a transaction, oldest first, to whoever may see
// the original, and nothing to anyone else.
func TestReversalsOf(t *testing.T) {
	w := newWorld(t, nil)
	orig := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 300)
	plain := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 50)
	r1 := tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: orig.ID, Amount: 100, Reason: "part", IdempotencyKey: w.Key()}))
	r2 := tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: orig.ID, Reason: "rest", IdempotencyKey: w.Key()}))

	for name, who := range map[string]domain.Actor{"payer": w.a, "payee": w.b, "admin": w.admin} {
		got := tu.Must(w.Svc.ReversalsOf(w.Ctx, who, w.net.ID, orig.ID))
		if len(got) != 2 || got[0].ID != r1.ID || got[1].ID != r2.ID {
			t.Fatalf("%s: reversals = %v", name, got)
		}
		if got[0].OriginalTxID != orig.ID || got[0].Kind != domain.KindReversal || got[0].Amount != 100 || got[1].Amount != 200 {
			t.Errorf("%s: unexpected reversal rows %+v %+v", name, got[0], got[1])
		}
		if none := tu.Must(w.Svc.ReversalsOf(w.Ctx, who, w.net.ID, plain.ID)); len(none) != 0 {
			t.Errorf("%s: unreversed trade lists %d reversals", name, len(none))
		}
		if none := tu.Must(w.Svc.ReversalsOf(w.Ctx, who, w.net.ID, r1.ID)); len(none) != 0 {
			t.Errorf("%s: a reversal lists %d reversals", name, len(none))
		}
	}

	// A member who is not a party, a missing id and another network's administrator see the
	// same not-found (SEC-009).
	c, _ := w.Member(w.admin, w.net, "cid@example.org")
	other := w.Person("boss@other.example")
	w.Network(other, "elsewhere", nil)
	for name, call := range map[string]func() error{
		"outsider":      func() error { _, err := w.Svc.ReversalsOf(w.Ctx, c, w.net.ID, orig.ID); return err },
		"missing":       func() error { _, err := w.Svc.ReversalsOf(w.Ctx, w.a, w.net.ID, "no-such-id"); return err },
		"foreign admin": func() error { _, err := w.Svc.ReversalsOf(w.Ctx, other, w.net.ID, orig.ID); return err },
	} {
		if err := call(); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: err = %v, want not-found", name, err)
		}
	}
}
