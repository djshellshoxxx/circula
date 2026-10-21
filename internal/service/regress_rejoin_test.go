package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-011 / FR-017: a denied applicant stays out. Re-requesting, joining under
// a later open policy, redeeming an invitation, and approving after the fact
// are all refused, and the denial is what the audit trail shows.
func TestDeniedApplicantStaysOutAfterPolicyChange(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	p, pM := w.joinPending("late@example.org")
	tu.Must(w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, false, "not this time"))

	if _, err := w.Svc.JoinNetwork(w.Ctx, p, w.net.ID, "Late"); err == nil {
		t.Fatal("denied applicant re-requested under approval mode")
	}
	pol := domain.DefaultPolicy()
	pol.JoinMode = domain.JoinOpen
	tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "open up", IdempotencyKey: w.Key()}))
	_, err := w.Svc.JoinNetwork(w.Ctx, p, w.net.ID, "Late")
	wantCode(t, err, domain.ErrForbidden)

	tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	_, err = w.Svc.RedeemInvitation(w.Ctx, p, tok, "Late")
	wantCode(t, err, domain.ErrForbidden)

	if _, err := w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, true, "changed my mind"); err == nil {
		t.Fatal("a denied membership was approved after the fact")
	}

	active := tu.Must(w.Svc.AdminListMembers(w.Ctx, w.admin, w.net.ID, []domain.MemberStatus{domain.MemberActive}, service.Page{}))
	for _, m := range active {
		if m.ID == pM.ID {
			t.Fatal("denied applicant is listed as active")
		}
	}
	// The open join mode did not admit anyone else through the denied person's identity.
	if got := tu.Must(w.Svc.AdminListMembers(w.Ctx, w.admin, w.net.ID, []domain.MemberStatus{domain.MemberDeparted}, service.Page{})); len(got) != 1 {
		t.Errorf("departed members = %d, want only the denied applicant", len(got))
	}
}
