package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// The service's text limit counts Unicode code points, not UTF-8 bytes. Mix
// two-, three-, and four-byte characters at its exact 2,000-character bound.
func disputeTextAtLimit() string { return strings.Repeat("é文😀", 666) + "é文" }

// FR-070, UX-021: valid non-ASCII evidence follows the same review/confirm
// path as ASCII. Refusing a longer statement must not open a dispute.
func TestDisputeStatementCharacterLimitMatchesService(t *testing.T) {
	w := newWorld(t)
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	br := w.browser(t)
	br.login("alice@example.org")
	path := w.base() + "/trades/" + tx.ID + "/dispute"
	form := br.get(path)
	statement := disputeTextAtLimit()
	for _, step := range []string{"review", "confirm"} {
		invalid := br.post(path, form, url.Values{"category": {"quality"}, "statement": {statement + "文"}, "step": {step}})
		if invalid.Status != 422 {
			t.Fatalf("over-limit %s status = %d", step, invalid.Status)
		}
		br.mustContain(invalid, `href="#f-statement"`)
		v := testutil.Must(w.env.Svc.GetTransaction(w.env.Ctx, w.alice, w.net.ID, tx.ID))
		if v.Tx.Status != domain.TxCompleted {
			t.Fatal("over-limit statement changed the trade")
		}
	}
	values := url.Values{"category": {"quality"}, "statement": {statement}, "step": {"review"}}
	review := br.post(path, form, values)
	if review.Status != 200 {
		t.Fatalf("2,000-character statement review status = %d", review.Status)
	}
	br.mustContain(review, statement, `name="step" value="confirm"`)
	values.Set("step", "confirm")
	opened := br.post(path, review, values)
	if opened.Status != 200 {
		t.Fatalf("statement confirmation status = %d", opened.Status)
	}
	disputes := testutil.Must(w.env.Svc.FindDisputes(w.env.Ctx, w.alice, w.net.ID, service.DisputeQuery{}))
	if len(disputes) != 1 || disputes[0].Statement != statement {
		t.Fatal("the confirmed statement was not preserved exactly")
	}
}

// FR-073, UX-021: the first decision's rationale must not have a stricter
// byte-based limit in its review screen than in the service command.
func TestDisputeRationaleCharacterLimitMatchesService(t *testing.T) {
	w := newWorld(t)
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	d := testutil.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: tx.ID, Category: "quality", Statement: "Please review the evidence", IdempotencyKey: w.env.Key()}))
	br := w.browser(t)
	br.login("admin@example.org")
	path := w.base() + "/disputes/" + d.ID
	form := br.get(path)
	rationale := disputeTextAtLimit()
	for _, step := range []string{"review", "confirm"} {
		invalid := br.post(path+"/resolve", form, url.Values{"outcome": {"dismissed"}, "rationale": {rationale + "文"}, "step": {step}})
		if invalid.Status != 422 {
			t.Fatalf("over-limit %s status = %d", step, invalid.Status)
		}
		br.mustContain(invalid, `href="#f-rationale"`)
		view := testutil.Must(w.env.Svc.GetDispute(w.env.Ctx, w.admin, w.net.ID, d.ID))
		if view.Dispute.Status != domain.DisputeOpen {
			t.Fatal("over-limit rationale changed the dispute")
		}
	}
	values := url.Values{"outcome": {"dismissed"}, "rationale": {rationale}, "step": {"review"}}
	review := br.post(path+"/resolve", form, values)
	if review.Status != 200 {
		t.Fatalf("2,000-character rationale review status = %d", review.Status)
	}
	br.mustContain(review, rationale, `name="step" value="confirm"`)
	values.Set("step", "confirm")
	decided := br.post(path+"/resolve", review, values)
	if decided.Status != 200 {
		t.Fatalf("decision confirmation status = %d", decided.Status)
	}
	view := testutil.Must(w.env.Svc.GetDispute(w.env.Ctx, w.admin, w.net.ID, d.ID))
	if view.Dispute.Status != domain.DisputeResolved || view.Dispute.Rationale != rationale {
		t.Fatal("the confirmed rationale was not preserved exactly")
	}
}

// FR-078, UX-021: the separately confirmed re-review uses the same character
// bound, with its proof bound to the entire accepted Unicode rationale.
func TestReReviewRationaleCharacterLimitMatchesService(t *testing.T) {
	w, _, _, id := reReviewWorld(t)
	br := w.browser(t)
	br.login("second@example.org")
	path := w.base() + "/disputes/" + id
	form := br.get(path)
	rationale := disputeTextAtLimit()
	invalid := br.post(path+"/rereview-decide", form, url.Values{"outcome": {"confirmed"}, "rationale": {rationale + "文"}})
	if invalid.Status != 422 {
		t.Fatalf("over-limit review status = %d", invalid.Status)
	}
	br.mustContain(invalid, `href="#f-rationale"`)
	assertReReviewPending(t, w, id, 0)
	review := br.post(path+"/rereview-decide", form, url.Values{"outcome": {"confirmed"}, "rationale": {rationale}})
	if review.Status != 200 {
		t.Fatalf("2,000-character re-review status = %d", review.Status)
	}
	br.mustContain(review, rationale)
	assertReReviewPending(t, w, id, 0)
	decided := br.post(path+"/rereview-decide", review, confirmationValues(t, review, "confirm"))
	if decided.Status != 200 {
		t.Fatalf("re-review confirmation status = %d", decided.Status)
	}
	view := testutil.Must(w.env.Svc.GetDispute(w.env.Ctx, w.admin, w.net.ID, id))
	if view.ReReview.Status != service.ReReviewDecided || view.ReReview.Rationale != rationale {
		t.Fatal("the confirmed re-review rationale was not preserved exactly")
	}
}
