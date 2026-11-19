package web

import (
	"net/url"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-001, FR-078 (AUD4-F03): the dispute page shows the re-review deadline
// the service enforces, which is the window of the policy in force when the
// dispute was opened. A later, shorter window neither hides the request form
// nor moves the date shown with it.
func TestDisputePageReReviewDeadlineFixedAtOpen(t *testing.T) {
	w := newWorld(t) // re-review window: the default 14 days
	env := w.env
	rev, revM := env.Member(w.admin, w.net, "rev1@example.org")
	testutil.Must(0, env.Svc.AssignRole(env.Ctx, w.admin, w.net.ID, revM.ID, domain.RoleReviewer))
	tx := env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	d := testutil.Must(env.Svc.OpenDispute(env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "quality",
		Statement: "broken", IdempotencyKey: env.Key()}))
	d = testutil.Must(env.Svc.ResolveDispute(env.Ctx, rev, w.net.ID, service.ResolveInput{DisputeID: d.ID,
		Outcome: domain.OutcomeDismissed, Rationale: "not enough evidence", IdempotencyKey: env.Key()}))
	until := d.DecidedAt.Add(time.Duration(domain.DefaultPolicy().ReReviewWindowDays) * 24 * time.Hour)

	env.Clock.Advance(time.Hour)
	pol := testutil.Must(env.Svc.GetNetwork(env.Ctx, w.admin, w.net.ID)).Policy
	pol.ReReviewWindowDays = 1
	testutil.Must(env.Svc.UpdatePolicy(env.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "tidy", IdempotencyKey: env.Key()}))
	env.Clock.Advance(3 * 24 * time.Hour)

	alice := w.browser(t)
	alice.login("alice@example.org")
	path := w.base() + "/disputes/" + d.ID
	page := alice.get(path)
	alice.mustContain(page, "Ask for a re-review", `datetime="`+until.UTC().Format(time.RFC3339)+`"`)
	alice.mustContain(alice.post(path+"/rereview", page, url.Values{"reason": {"I have photos"}}), "Re-review requested")
}
