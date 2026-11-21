package web

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

var confirmationInputRE = regexp.MustCompile(`<input type="hidden" name="([^"]+)" value="([^"]*)"`)

// Submit the values rendered by the review page, including its signed proof.
func confirmationValues(t *testing.T, p *pg, step string) url.Values {
	t.Helper()
	v := url.Values{"step": {step}}
	for _, m := range confirmationInputRE.FindAllStringSubmatch(p.Body, -1) {
		v.Set(m[1], html.UnescapeString(m[2]))
	}
	if v.Get("review_proof") == "" {
		t.Fatalf("review page lacks confirmation proof: status %d\n%s", p.Status, snippet(p.Body))
	}
	return v
}

func reReviewWorld(t *testing.T) (*world, *domain.Member, string, string) {
	t.Helper()
	w := newWorld(t)
	first, firstM := w.env.Member(w.admin, w.net, "first@example.org")
	_, secondM := w.env.Member(w.admin, w.net, "second@example.org")
	for _, m := range []*domain.Member{firstM, secondM} {
		testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, m.ID, domain.RoleReviewer))
	}
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	d := testutil.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: tx.ID, Category: "quality", Statement: "broken", IdempotencyKey: w.env.Key()}))
	testutil.Must(w.env.Svc.ResolveDispute(w.env.Ctx, first, w.net.ID, service.ResolveInput{
		DisputeID: d.ID, Outcome: domain.OutcomeDismissed, Rationale: "initial evidence",
		WaiveResponse: true, WaiveReason: "both parties asked for a decision", IdempotencyKey: w.env.Key()}))
	testutil.Must(w.env.Svc.RequestReReview(w.env.Ctx, w.alice, w.net.ID, service.ReReviewRequestInput{
		DisputeID: d.ID, Reason: "new evidence", IdempotencyKey: w.env.Key()}))
	return w, secondM, tx.ID, d.ID
}

func assertReReviewPending(t *testing.T, w *world, disputeID string, reversed int64) {
	t.Helper()
	v := testutil.Must(w.env.Svc.GetDispute(w.env.Ctx, w.admin, w.net.ID, disputeID))
	if v.ReReview.Status != service.ReReviewRequested || v.Tx.ReversedTotal != reversed || v.Dispute.Outcome != domain.OutcomeDismissed {
		t.Fatalf("unexpected mutation: re-review=%+v transaction=%+v original=%+v", v.ReReview, v.Tx, v.Dispute)
	}
	if b := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID)); b.Balance != -1000+reversed {
		t.Fatalf("payer balance = %d; want %d", b.Balance, -1000+reversed)
	}
	if b := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); b.Balance != 1000-reversed {
		t.Fatalf("payee balance = %d; want %d", b.Balance, 1000-reversed)
	}
}

func TestReReviewRequiresReviewedConfirmation(t *testing.T) {
	for _, tc := range []struct {
		outcome, amount string
		reversed        int64
	}{{"upheld", "", 1000}, {"partially_upheld", "3.25", 325}, {"confirmed", "", 0}} {
		t.Run(tc.outcome, func(t *testing.T) {
			w, _, _, id := reReviewWorld(t)
			br := w.browser(t)
			br.login("second@example.org")
			path := w.base() + "/disputes/" + id
			invalid := br.post(path+"/rereview-decide", br.get(path), url.Values{
				"outcome": {"partially_upheld"}, "amount": {"not an amount"}, "rationale": {"new evidence"}})
			if invalid.Status != 422 {
				t.Fatalf("invalid re-review amount status = %d", invalid.Status)
			}
			br.mustContain(invalid, "enter the amount to reverse, with up to 2 decimal places", `href="#f-amount"`)
			assertReReviewPending(t, w, id, 0)
			review := br.post(path+"/rereview-decide", br.get(path), url.Values{
				"outcome": {tc.outcome}, "amount": {tc.amount}, "rationale": {`New "photos" & testimony`}})
			br.mustContain(review, "Review the re-review decision", "alice@example.org", "bob@example.org", `New "photos" & testimony`)
			// PER-003: reviewing a dispute permits signed effects, not a read of
			// either party's unrelated account balance.
			if strings.Contains(review.Body, "balance now") || strings.Contains(review.Body, "balance after") {
				t.Fatal("reviewer was shown private account balances")
			}
			br.mustContain(review, makeBalance(tc.reversed, 2, "credit").Signed, makeBalance(-tc.reversed, 2, "credit").Signed)
			assertReReviewPending(t, w, id, 0)
			back := br.post(path+"/rereview-decide", review, confirmationValues(t, review, "back"))
			br.mustContain(back, "Review re-review decision", `New "photos" & testimony`)
			assertReReviewPending(t, w, id, 0)

			for _, change := range []struct{ field, value string }{
				{"review_proof", ""}, {"amount", "1.23"}, {"outcome", "other"}, {"rationale", "changed after review"},
			} {
				v := confirmationValues(t, review, "confirm")
				v.Set(change.field, change.value)
				bad := br.post(path+"/rereview-decide", review, v)
				if bad.Status != 422 {
					t.Fatalf("altered %s status = %d", change.field, bad.Status)
				}
				if change.field == "review_proof" {
					br.mustContain(bad, "review this decision again before confirming; nothing changed")
				}
				assertReReviewPending(t, w, id, 0)
			}
			v := confirmationValues(t, review, "confirm")
			br.mustContain(br.post(path+"/rereview-decide", review, v), "Re-review decision recorded")
			// Reusing the rendered submission must not post a second reversal.
			br.mustContain(br.post(path+"/rereview-decide", review, v), "Re-review decision recorded")
			d := testutil.Must(w.env.Svc.GetDispute(w.env.Ctx, w.admin, w.net.ID, id))
			if d.ReReview.ReversalAmount != tc.reversed || d.Tx.ReversedTotal != tc.reversed || d.ReReview.Outcome != tc.outcome {
				t.Fatalf("confirmed different effects: re-review=%+v transaction=%+v", d.ReReview, d.Tx)
			}
		})
	}
}

func TestReReviewConfirmRechecksLiveStateAndAuthority(t *testing.T) {
	for _, change := range []string{"remaining amount", "reviewer role"} {
		t.Run(change, func(t *testing.T) {
			w, reviewer, txID, id := reReviewWorld(t)
			br := w.browser(t)
			br.login("second@example.org")
			path := w.base() + "/disputes/" + id
			review := br.post(path+"/rereview-decide", br.get(path), url.Values{"outcome": {"upheld"}, "rationale": {"new evidence"}})
			var reversed int64
			if change == "remaining amount" {
				testutil.Must(w.env.Svc.ReverseTransaction(w.env.Ctx, w.admin, w.net.ID, service.ReverseInput{
					TxID: txID, Amount: 200, Reason: "separate correction", IdempotencyKey: w.env.Key()}))
				reversed = 200
			} else {
				testutil.Must(0, w.env.Svc.RevokeRole(w.env.Ctx, w.admin, w.net.ID, reviewer.ID, domain.RoleReviewer))
			}
			bad := br.post(path+"/rereview-decide", review, confirmationValues(t, review, "confirm"))
			if bad.Status < 400 || strings.Contains(bad.Body, "Re-review decision recorded") {
				t.Fatalf("stale review accepted: %d", bad.Status)
			}
			assertReReviewPending(t, w, id, reversed)
		})
	}
}

func TestReReviewAdminPreviewIncludesProjectedBalances(t *testing.T) {
	w, _, _, id := reReviewWorld(t)
	testutil.Must(w.env.Svc.CreateAdjustment(w.env.Ctx, w.admin, w.net.ID, service.AdjustInput{
		MemberID: w.aliceM.ID, Amount: 2345, Direction: service.AdjustCredit, Reason: "separate credit", IdempotencyKey: w.env.Key()}))
	br := w.browser(t)
	br.login("admin@example.org")
	path := w.base() + "/disputes/" + id
	review := br.post(path+"/rereview-decide", br.get(path), url.Values{"outcome": {"upheld"}, "rationale": {"new evidence"}})
	br.mustContain(review, "alice@example.org: balance now", "+13.45", "alice@example.org: balance after", "+23.45",
		"bob@example.org: balance now", "+10.00", "bob@example.org: balance after", "0.00")
	if b := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID)); b.Balance != 1345 {
		t.Fatalf("admin preview posted: %d", b.Balance)
	}
}

func approvalWorld(t *testing.T) (*world, *domain.Member, *domain.PendingAdjustment) {
	t.Helper()
	w := newWorld(t)
	_, admin2 := w.env.Member(w.admin, w.net, "admin2@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, domain.SystemActor("fixture"), w.net.ID, admin2.ID, domain.RoleAdmin))
	w.env.Clock.Advance(25 * time.Hour)
	pol := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID)).Policy
	pol.DualControlThreshold = 1000
	testutil.Must(w.env.Svc.UpdatePolicy(w.env.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dual", IdempotencyKey: w.env.Key()}))
	r := testutil.Must(w.env.Svc.CreateAdjustmentRequest(w.env.Ctx, w.admin, w.net.ID, service.AdjustInput{
		MemberID: w.bobM.ID, Amount: 2500, Direction: service.AdjustCredit, Reason: "reviewed correction", IdempotencyKey: w.env.Key()}))
	return w, admin2, r.Pending
}

func assertApprovalUnposted(t *testing.T, w *world, id string, status domain.PendingStatus) {
	t.Helper()
	p := testutil.Must(w.env.Svc.GetPendingAdjustment(w.env.Ctx, w.admin, w.net.ID, id))
	if p.Status != status {
		t.Fatalf("request status = %s, want %s", p.Status, status)
	}
	for _, actor := range []domain.Actor{w.alice, w.bob} {
		if b := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, actor, w.net.ID)); b.Balance != 0 {
			t.Fatalf("unconfirmed approval moved credit: %d", b.Balance)
		}
	}
}

func TestAdjustmentApprovalRequiresReviewedConfirmation(t *testing.T) {
	w, _, pa := approvalWorld(t)
	br := w.browser(t)
	br.login("admin2@example.org")
	path := w.base() + "/admin/adjustments/" + pa.ID + "/approve"
	list := br.get(w.base() + "/admin/adjustments/pending")
	review := br.post(path, list, nil)
	br.mustContain(review, "Review adjustment approval", "bob@example.org", "Founder", "25.00", "Their balance now", "Their balance after", "+25.00")
	assertApprovalUnposted(t, w, pa.ID, domain.PendingOpen)
	br.mustContain(br.post(path, review, confirmationValues(t, review, "back")), "Adjustments awaiting approval", "Review approval")
	assertApprovalUnposted(t, w, pa.ID, domain.PendingOpen)
	bad := br.post(path, list, url.Values{"step": {"confirm"}})
	if bad.Status != 409 {
		t.Fatalf("unreviewed confirm status = %d", bad.Status)
	}
	assertApprovalUnposted(t, w, pa.ID, domain.PendingOpen)
	vals := confirmationValues(t, review, "confirm")
	// Posted amount/member fields cannot replace the stored approved request.
	vals.Set("member_id", w.aliceM.ID)
	vals.Set("amount", "999.00")
	vals.Set("direction", "debit")
	br.mustContain(br.post(path, review, vals), "Adjustment request approved")
	br.mustContain(br.post(path, review, vals), "Adjustment request approved")
	if b := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); b.Balance != 2500 {
		t.Fatalf("reviewed request posted incorrectly: %d", b.Balance)
	}
	if b := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID)); b.Balance != 0 {
		t.Fatalf("forged member was credited: %d", b.Balance)
	}
}

func TestAdjustmentApprovalConfirmRechecksLiveStateAndAuthority(t *testing.T) {
	for _, change := range []string{"cancelled", "expired", "admin role"} {
		t.Run(change, func(t *testing.T) {
			w, admin2, pa := approvalWorld(t)
			br := w.browser(t)
			br.login("admin2@example.org")
			path := w.base() + "/admin/adjustments/" + pa.ID + "/approve"
			review := br.post(path, br.get(w.base()+"/admin/adjustments/pending"), nil)
			status := domain.PendingOpen
			switch change {
			case "cancelled":
				testutil.Must(w.env.Svc.CancelAdjustment(w.env.Ctx, w.admin, w.net.ID, pa.ID))
				status = domain.PendingCancelled
			case "expired":
				w.env.Clock.Advance(pa.ExpiresAt.Sub(w.env.Clock.Now()))
				br.login("admin2@example.org") // advance may also have expired the session
				status = domain.PendingExpired
			case "admin role":
				testutil.Must(0, w.env.Svc.RevokeRole(w.env.Ctx, w.admin, w.net.ID, admin2.ID, domain.RoleAdmin))
			}
			current := br.get(w.base() + "/admin/adjustments/pending")
			vals := confirmationValues(t, review, "confirm")
			if change == "expired" {
				vals.Set("csrf", current.csrf())
			}
			bad := br.post(path, current, vals)
			wantStatus := 409
			if change == "admin role" {
				wantStatus = 403
			}
			if bad.Status != wantStatus || strings.Contains(bad.Body, "Adjustment request approved") {
				t.Fatalf("stale approval accepted: %d", bad.Status)
			}
			assertApprovalUnposted(t, w, pa.ID, status)
		})
	}
}
