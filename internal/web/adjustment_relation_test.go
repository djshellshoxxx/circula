package web

import (
	"fmt"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAdjustmentRelationWebReviewAndHistory(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	original := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	admin := w.browser(t)
	admin.login("admin@example.org")
	vals := url.Values{"member_id": {w.aliceM.ID}, "direction": {"credit"}, "amount": {"0.25"}, "reason": {"rounding correction"}, "related_transaction_id": {original.ID}, "step": {"review"}}
	form := admin.get(b + "/admin/adjust")
	admin.mustContain(form, "Related transaction (optional)")
	review := admin.post(b+"/admin/adjust", form, vals)
	link := fmt.Sprintf(`<a href="%s/trades/%s">The original entry</a>`, b, original.ID)
	admin.mustContain(review, link, `name="related_transaction_id" value="`+original.ID+`"`)
	vals.Set("step", "back")
	back := admin.post(b+"/admin/adjust", review, vals)
	admin.mustContain(back, `value="`+original.ID+`"`)
	vals.Set("step", "review")
	review = admin.post(b+"/admin/adjust", back, vals)
	vals.Set("step", "confirm")
	done := admin.post(b+"/admin/adjust", review, vals)
	if done.Status != 200 {
		t.Fatalf("adjustment status=%d %s", done.Status, done.Body)
	}
	rows := testutil.Must(w.env.Svc.ListTransactions(w.env.Ctx, w.alice, w.net.ID, nil, service.Page{}))
	adjustment := rows[0]
	if adjustment.RelatedTxID != original.ID {
		t.Fatal("form lost relation")
	}
	member := w.browser(t)
	member.login("alice@example.org")
	row := rowOf(member.get(b+"/history").Body, b+"/trades/"+adjustment.ID)
	for _, want := range []string{fmt.Sprintf(`Adjusts <a href="%s/trades/%s">the original entry</a>`, b, original.ID), "Reason: rounding correction", "&#43;0.25", "<time datetime="} {
		if !strings.Contains(row, want) {
			t.Fatalf("history missing %q: %s", want, row)
		}
	}
	member.mustContain(member.get(b+"/trades/"+adjustment.ID), "<dt>Adjusts</dt>", link)
	origRow := rowOf(member.get(b+"/history").Body, b+"/trades/"+original.ID)
	if strings.Contains(origRow, "Adjusts") || strings.Contains(origRow, "rounding correction") {
		t.Fatal("original line altered")
	}

	// Unrelated references fail before the review and create nothing.
	unrelated := w.env.Trade(w.net.ID, w.admin, w.bob, w.adminM, w.bobM, 50)
	vals.Set("step", "review")
	vals.Set("related_transaction_id", unrelated.ID)
	vals.Del("idem")
	bad := admin.post(b+"/admin/adjust", admin.get(b+"/admin/adjust"), vals)
	if bad.Status != 422 {
		t.Fatalf("unrelated review accepted: %d", bad.Status)
	}
	admin.mustContain(bad, "Choose a posted transaction involving the adjusted member")
}

func TestAdjustmentRelationWebApprovalShowsStoredContext(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	original := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	_, secondM := w.env.Member(w.admin, w.net, "second@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, domain.SystemActor("fixture"), w.net.ID, secondM.ID, domain.RoleAdmin))
	w.env.Clock.Advance(25 * time.Hour)
	pol := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID)).Policy
	pol.DualControlThreshold = 1000
	testutil.Must(w.env.Svc.UpdatePolicy(w.env.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dual", IdempotencyKey: w.env.Key()}))
	pa := testutil.Must(w.env.Svc.CreateAdjustmentRequest(w.env.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aliceM.ID, Amount: 2000, Direction: service.AdjustCredit, Reason: "larger correction", RelatedTxID: original.ID, IdempotencyKey: w.env.Key()})).Pending
	two := w.browser(t)
	two.login("second@example.org")
	path := b + "/admin/adjustments/" + pa.ID + "/approve"
	review := two.post(path, two.get(b+"/admin/adjustments/pending"), nil)
	two.mustContain(review, "Related transaction", fmt.Sprintf(`<a href="%s/trades/%s">The original entry</a>`, b, original.ID))
	confirm := confirmationValues(t, review, "confirm")
	confirm.Set("related_transaction_id", "forged")
	two.mustContain(two.post(path, review, confirm), "Adjustment request approved")
	posted := testutil.Must(w.env.Svc.GetPendingAdjustment(w.env.Ctx, w.admin, w.net.ID, pa.ID))
	if testutil.Must(w.env.Svc.GetTransaction(w.env.Ctx, w.alice, w.net.ID, posted.TxID)).Tx.RelatedTxID != original.ID {
		t.Fatal("approval trusted posted relation")
	}
}
