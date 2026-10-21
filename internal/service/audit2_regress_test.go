package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SA2-01: one administrator cannot loosen or disable dual control; the
// operator can.
func TestPolicyCannotLoosenDualControl(t *testing.T) {
	d := newDC(t)
	if err := d.setThreshold(0); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("disabling dual control by an admin: %v", err)
	}
	if err := d.setThreshold(domain.MaxPolicyAmount); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("raising the threshold by an admin: %v", err)
	}
	if err := d.setThreshold(500); err != nil {
		t.Fatalf("tightening must stay allowed: %v", err)
	}
	r, err := d.request(d.admin, d.aM, 5_000_000, service.AdjustCredit)
	if err != nil || r.Pending == nil || r.Tx != nil {
		t.Fatalf("large adjustment must stay pending: %+v %v", r, err)
	}
	if err := d.Svc.OperatorSetDualControl(d.Ctx, domain.SystemActor("t"), d.net.Slug, 0, "wind down"); err != nil {
		t.Fatalf("operator: %v", err)
	}
	if err := d.Svc.OperatorSetDualControl(d.Ctx, d.admin, d.net.Slug, 0, "x"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-system operator call: %v", err)
	}
}

// SA2-02: an administrator removing a member cannot settle more than the
// threshold alone.
func TestDepartureSettlementRespectsDualControl(t *testing.T) {
	d := newDC(t)
	n := tu.Must(d.Svc.GetNetwork(d.Ctx, d.admin, d.net.ID))
	pol := n.Policy
	pol.DepartureRule = domain.DepartSettleToZero
	if _, err := d.Svc.UpdatePolicy(d.Ctx, d.admin, d.net.ID, service.PolicyChange{Policy: pol, Reason: "x", IdempotencyKey: d.Key()}); err != nil {
		t.Fatal(err)
	}
	tx := tu.Must(d.propose(d.b, d.aM, service.ProposerPays, 50000))
	tu.Must(d.accept(d.a, tx.ID))
	_, err := d.Svc.DepartMember(d.Ctx, d.admin, d.net.ID, service.DepartInput{MemberID: d.aM.ID, Reason: "r"})
	if domain.CodeOf(err) != domain.CodeApprovalRequired {
		t.Fatalf("departure over the threshold: %v", err)
	}
	// The member can still leave on their own terms.
	if _, err := d.Svc.DepartMember(d.Ctx, d.a, d.net.ID, service.DepartInput{Reason: "moving"}); err != nil {
		t.Fatalf("self departure: %v", err)
	}
}

// SA2-04: a break-glass administrator cannot create lasting administrators.
func TestBreakGlassCannotMakeLastingChanges(t *testing.T) {
	w := newWorld(t, nil)
	w.openBG(time.Hour)
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "support"}))
	if err := w.Svc.AssignRole(w.Ctx, w.a, w.net.ID, w.bM.ID, domain.RoleAdmin); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("break-glass AssignRole: %v", err)
	}
	if _, err := w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{Limit: 5}); err != nil {
		t.Fatalf("break-glass read access must still work: %v", err)
	}
}
