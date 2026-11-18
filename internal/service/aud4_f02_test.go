package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// DOM-038 / AUD4-F02: an administrator who is a party to a dispute must not
// be able to take over a reviewer's account with a password reset and decide
// the dispute as that reviewer. Reviewer resets need the platform operator.
func TestAdminCannotResetReviewerPassword(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 })
	_, revM := w.Member(w.admin, w.net, "rita@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, revM.ID, domain.RoleReviewer))
	tx := w.Trade(w.net.ID, w.admin, w.a, w.adminM, w.aM, 5000)
	tu.Must(w.Svc.OpenDispute(w.Ctx, w.admin, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "not_delivered", IdempotencyKey: w.Key()}))
	_, err := w.Svc.AdminResetPassword(w.Ctx, w.admin, w.net.ID, revM.ID, "lost password, verified")
	if domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("administrator reset a reviewer's password: err = %v", err)
	}
	// An ordinary member can still be reset by an administrator.
	if _, err := w.Svc.AdminResetPassword(w.Ctx, w.admin, w.net.ID, w.bM.ID, "lost password, verified"); err != nil {
		t.Fatalf("ordinary member reset refused: %v", err)
	}
}
