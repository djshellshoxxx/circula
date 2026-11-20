package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// DOM-038 / SEC-010: an earlier reversal author remains conflicted after more
// than one repository page of newer reversals. Authorization must search for
// that actor's participation, not just inspect the newest 500 linked rows.
func TestDisputeIndependenceBeyondReversalPage(t *testing.T) {
	w := newWorld(t, nil)
	priorAdmin, priorM := w.Member2(w.admin, "prior-admin@example.org")
	noErr(t, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, priorM.ID, domain.RoleAdmin))
	original := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	tu.Must(w.reverse(priorAdmin, original.ID, 1, "earlier participation"))
	for i := 0; i < 500; i++ {
		tu.Must(w.reverse(w.admin, original.ID, 1, "later partial reversal"))
	}
	d := tu.Must(w.open(w.a, original.ID))
	before := dbChecksum(t, w.Env)
	_, err := w.Svc.ResolveDispute(w.Ctx, priorAdmin, w.net.ID, service.ResolveInput{
		DisputeID: d.ID, Outcome: domain.OutcomeDismissed, Rationale: "attempted self-review", IdempotencyKey: w.Key(),
	})
	wantCode(t, err, domain.ErrForbidden)
	if after := dbChecksum(t, w.Env); after != before {
		t.Fatal("conflicted decision changed the ledger, dispute or audit")
	}
	// A reviewer with no linked postings is still allowed to decide.
	reviewer, _ := w.reviewer()
	done := tu.Must(w.resolve(reviewer, d.ID, domain.OutcomeDismissed, 0, "independent review"))
	if done.Status != domain.DisputeResolved {
		t.Fatalf("independent decision = %s", done.Status)
	}
}
