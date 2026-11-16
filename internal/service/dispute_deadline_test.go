package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// DOM-017 / DAT-051: the dispute deadline is fixed at completion from the
// window in force then. Lengthening the window later does not reopen an
// expired right, shortening it does not cut a running one, and opening the
// dispute is allowed up to and including the stored instant.
func TestDisputeDeadlineFixedAtCompletion(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "deadline", nil)
	a, aM := e.Member(admin, net, "a@example.org")
	b, bM := e.Member(admin, net, "b@example.org")
	const d = 24 * time.Hour

	setWindow := func(days int) {
		t.Helper()
		n := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID))
		pol := n.Policy
		pol.DisputeWindowDays = days
		tu.Must(e.Svc.UpdatePolicy(e.Ctx, admin, net.ID, service.PolicyChange{Policy: pol, Reason: "window change", IdempotencyKey: e.Key()}))
	}
	open := func(txID string) error {
		_, err := e.Svc.OpenDispute(e.Ctx, a, net.ID, service.OpenDisputeInput{TxID: txID, Category: "not_delivered",
			Statement: "never arrived", IdempotencyKey: e.Key()})
		return err
	}
	deadline := func(txID string) time.Time {
		t.Helper()
		v := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, txID))
		if v.Tx.DisputeDeadlineAt == nil {
			t.Fatalf("trade %s has no stored dispute deadline", txID)
		}
		return *v.Tx.DisputeDeadlineAt
	}
	window := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID)).Policy.DisputeWindowDays

	t0 := e.Clock.Now()
	t1 := e.Trade(net.ID, b, a, bM, aM, 100)
	if got, want := deadline(t1.ID), t0.Add(time.Duration(window)*d); !got.Equal(want) {
		t.Fatalf("stored deadline %s, want completion + %d days = %s", got, window, want)
	}
	if v := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, t1.ID)).Tx.DisputePolicyVersion; v != net.PolicyVersion {
		t.Fatalf("stored policy version %d, want %d", v, net.PolicyVersion)
	}

	// Lengthening the window after completion does not extend t1's right.
	setWindow(window + 30)
	e.Clock.Advance(time.Duration(window)*d + time.Second)
	if err := open(t1.ID); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("dispute after the stored deadline: err = %v, want invalid state", err)
	}
	if got := deadline(t1.ID); !got.Equal(t0.Add(time.Duration(window) * d)) {
		t.Fatalf("policy change moved the stored deadline to %s", got)
	}

	// A trade completed under the longer window keeps it after the window is cut.
	t2 := e.Trade(net.ID, b, a, bM, aM, 100)
	t2Deadline := deadline(t2.ID)
	if want := e.Clock.Now().Add(time.Duration(window+30) * d); !t2Deadline.Equal(want) {
		t.Fatalf("t2 deadline %s, want %s", t2Deadline, want)
	}
	setWindow(2)            // adverse: scheduled after the default notice
	e.Clock.Advance(20 * d) // the shorter window is in force now
	if n := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID)); n.Policy.DisputeWindowDays != 2 {
		t.Fatalf("shorter window not in force after notice: %d", n.Policy.DisputeWindowDays)
	}
	if err := open(t2.ID); err != nil {
		t.Fatalf("dispute within t2's stored deadline refused: %v", err)
	}
	if got := deadline(t2.ID); !got.Equal(t2Deadline) {
		t.Fatalf("t2 deadline changed to %s", got)
	}

	// Boundary: at exactly the stored instant the dispute may still be opened.
	t3 := e.Trade(net.ID, b, a, bM, aM, 100)
	e.Clock.Advance(deadline(t3.ID).Sub(e.Clock.Now()))
	if err := open(t3.ID); err != nil {
		t.Fatalf("dispute at the exact deadline refused: %v", err)
	}

	// Disputes never post: balances are what the three trades made them.
	if bal := tu.Must(e.Svc.GetBalance(e.Ctx, a, net.ID)); bal.Balance != 300 {
		t.Fatalf("balance %d after disputes, want 300", bal.Balance)
	}
	requireVerified(t, e, admin, net.ID)
}
