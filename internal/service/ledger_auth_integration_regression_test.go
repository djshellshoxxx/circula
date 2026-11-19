package service_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A1-04 (GOV-020), service level: a reviewer decision must not post a
// reversal while the network is read-only because it has no active
// administrator. The unit test TestReverseRejectsAdministratorRequiredNetwork
// covers the helper; this one covers the ResolveDispute entry point.
func TestResolveDisputeRefusesReversalWhenAdministratorRequired(t *testing.T) {
	w := newWorld(t, nil)
	rev, _ := w.reviewer()
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.Svc.StartDisputeReview(w.Ctx, rev, w.net.ID, d.ID))
	// As in TestDepartureSettlementRefusedWhenAdminRequired: the only
	// administrator is no longer active.
	if _, err := w.Store.DB().Exec(`UPDATE members SET status='suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	if !tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)).AdminRequired {
		t.Fatal("precondition: administrator required")
	}
	beforeA, beforeB := w.balance(w.a).Balance, w.balance(w.b).Balance
	_, err := w.resolve(rev, d.ID, domain.OutcomeUpheld, 0, "not delivered")
	wantCode(t, err, domain.ErrInvalidState)
	if got := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, tx.ID)).Tx; got.Status != domain.TxDisputed || got.ReversedTotal != 0 {
		t.Fatalf("refused decision changed the trade: status %s reversed %d", got.Status, got.ReversedTotal)
	}
	if w.balance(w.a).Balance != beforeA || w.balance(w.b).Balance != beforeB {
		t.Fatal("refused decision moved balances")
	}
}

// A3-F03 (SEC-007), service level: with require-mfa-admins an administrator
// without a second factor cannot use reviewer/decider powers, while a
// reviewer who is not an administrator is unaffected.
func TestDeciderEntryPointsRequireAdministratorMFA(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
	admin := e.Person("admin@example.org")
	_, rc := enrol(t, e, admin) // set-up needs admin calls
	net, _ := e.Network(admin, "demo", nil)
	a, aM := e.Member(admin, net, "ana@example.org")
	b, bM := e.Member(admin, net, "ben@example.org")
	rev, revM := e.Member(admin, net, "rev@example.org")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, admin, net.ID, revM.ID, domain.RoleReviewer))
	tx := e.Trade(net.ID, a, b, aM, bM, 1000)
	d := tu.Must(e.Svc.OpenDispute(e.Ctx, a, net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "not_delivered",
		Statement: "never arrived", IdempotencyKey: e.Key()}))
	if err := e.Svc.DisableMFA(e.Ctx, admin, pw, rc[0]); err != nil {
		t.Fatal(err)
	}
	mfaRefused := func(what string, err error) {
		t.Helper()
		if domain.CodeOf(err) != domain.CodeForbidden || !strings.Contains(err.Error(), "mfa-enrolment-required") {
			t.Fatalf("%s by administrator without MFA: %v", what, err)
		}
	}
	_, err := e.Svc.StartDisputeReview(e.Ctx, admin, net.ID, d.ID)
	mfaRefused("StartDisputeReview", err)
	_, err = e.Svc.ResolveDispute(e.Ctx, admin, net.ID, service.ResolveInput{DisputeID: d.ID, Outcome: domain.OutcomeUpheld,
		Rationale: "not delivered", IdempotencyKey: e.Key()})
	mfaRefused("ResolveDispute", err)
	if got := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, tx.ID)).Tx; got.Status != domain.TxDisputed || got.ReversedTotal != 0 {
		t.Fatalf("refused decision changed the trade: status %s reversed %d", got.Status, got.ReversedTotal)
	}
	// The MFA requirement is for administrators only.
	if _, err := e.Svc.StartDisputeReview(e.Ctx, rev, net.ID, d.ID); err != nil {
		t.Fatalf("reviewer without the administrator role: %v", err)
	}
}
