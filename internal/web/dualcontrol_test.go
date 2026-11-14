package web

import (
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

var approveRe = regexp.MustCompile(`/admin/adjustments/([^/"]+)/approve`)

// SEC-011 web flow: an adjustment above the threshold is queued, not posted;
// the proposer sees no approve button; a second administrator approves it.
func TestDualControlWebFlow(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	_, adm2M := w.env.Member(w.admin, w.net, "admin2@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, domain.SystemActor("fixture"), w.net.ID, adm2M.ID, domain.RoleAdmin))
	w.env.Clock.Advance(25 * time.Hour) // A3-01: a new administrator waits before deciding
	n := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000 // 10.00
	testutil.Must(w.env.Svc.UpdatePolicy(w.env.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dual", IdempotencyKey: w.env.Key()}))

	adm := w.browser(t)
	adm.login("admin@example.org")
	vals := url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"25.00"}, "reason": {"big correction"}}
	f := adm.get(b + "/admin/adjust")
	vals.Set("step", "review")
	rev := adm.post(b+"/admin/adjust", f, vals)
	vals.Set("step", "confirm")
	done := adm.post(b+"/admin/adjust", rev, vals)
	adm.mustContain(done, "Adjustments awaiting approval", "above the dual-control threshold", "big correction", "Cancel request")
	if bal, _ := w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID); bal.Balance != 0 {
		t.Fatalf("posted before approval: %d", bal.Balance)
	}
	list := adm.get(b + "/admin/adjustments/pending")
	if approveRe.MatchString(list.Body) {
		t.Fatal("proposer is offered an approve button")
	}
	adm.mustContain(list, "Cancel request")

	// The proposer posting the approval by hand is refused and nothing posts.
	pa := testutil.Must(w.env.Svc.ListPendingAdjustments(w.env.Ctx, w.admin, w.net.ID, nil, service.Page{}))[0]
	self := adm.post(b+"/admin/adjustments/"+pa.ID+"/approve", list, nil)
	if self.Status != 409 {
		t.Errorf("self approve status %d", self.Status)
	}
	adm.mustContain(self, "needs a different administrator")

	// A plain member cannot reach the page.
	al := w.browser(t)
	al.login("alice@example.org")
	if p := al.get(b + "/admin/adjustments/pending"); p.Status != 403 {
		t.Errorf("member status %d", p.Status)
	}

	two := w.browser(t)
	two.login("admin2@example.org")
	l2 := two.get(b + "/admin/adjustments/pending")
	two.mustContain(l2, "Approve and post", "Reject", "big correction", "25.00")
	ok := two.post(b+"/admin/adjustments/"+pa.ID+"/approve", l2, nil)
	two.mustContain(ok, "Adjustment request approved")
	if bal, _ := w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID); bal.Balance != 2500 {
		t.Errorf("bob balance %d", bal.Balance)
	}
	// A decision on an already decided request is a clean refusal.
	again := two.post(b+"/admin/adjustments/"+pa.ID+"/reject", l2, nil)
	if again.Status != 409 {
		t.Errorf("reject after approve status %d", again.Status)
	}
	// CSRF is enforced.
	if p := two.post(b+"/admin/adjustments/"+pa.ID+"/approve", nil, nil); p.Status == 200 || p.Status == 303 {
		t.Errorf("post without csrf accepted: %d", p.Status)
	}
	two.mustContain(two.get(b+"/admin/adjustments/pending?status=approved"), "approved by", "admin2@example.org")
}

// A3-01 in the list: no approve/reject button where the service would refuse
// (founder-appointed administrator on the founder's request; any administrator
// in the 24 h after their grant). The honest direction keeps its button.
func TestDualControlButtonsFollowA3_01(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	_, apM := w.env.Member(w.admin, w.net, "appointed@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, apM.ID, domain.RoleAdmin))
	n := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	testutil.Must(w.env.Svc.UpdatePolicy(w.env.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dual", IdempotencyKey: w.env.Key()}))
	ap := w.browser(t)
	ap.login("appointed@example.org")
	adm := w.browser(t)
	adm.login("admin@example.org")
	request := func(br *browser, reason string) {
		vals := url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"25.00"}, "reason": {reason}}
		f := br.get(b + "/admin/adjust")
		vals.Set("step", "review")
		rev := br.post(b+"/admin/adjust", f, vals)
		vals.Set("step", "confirm")
		br.mustContain(br.post(b+"/admin/adjust", rev, vals), "Adjustments awaiting approval", reason)
	}
	request(adm, "founder asks")

	// Within 24 h of the grant: nothing to decide, and the page says so.
	l := ap.get(b + "/admin/adjustments/pending")
	if approveRe.MatchString(l.Body) || regexp.MustCompile(`/reject"`).MatchString(l.Body) {
		t.Fatal("new administrator offered a decision")
	}
	ap.mustContain(l, "founder asks", "Another administrator must decide this.")

	// After the wait: still nothing on the founder's request (appointed by the requester).
	w.env.Clock.Advance(25 * time.Hour)
	if l = ap.get(b + "/admin/adjustments/pending"); approveRe.MatchString(l.Body) {
		t.Fatal("founder-appointed administrator offered the founder's request")
	}

	// The appointed administrator asks; the founder gets the button, and it works.
	request(ap, "appointed asks")
	fl := adm.get(b + "/admin/adjustments/pending")
	m := approveRe.FindAllStringSubmatch(fl.Body, -1)
	pending := testutil.Must(w.env.Svc.ListPendingAdjustments(w.env.Ctx, w.admin, w.net.ID, []domain.PendingStatus{domain.PendingOpen}, service.Page{}))
	var theirs string
	for _, pa := range pending {
		if pa.Proposer == apM.ID {
			theirs = pa.ID
		}
	}
	if len(m) != 1 || m[0][1] != theirs {
		t.Fatalf("founder offered %v, want only %s", m, theirs)
	}
	adm.mustContain(adm.post(b+"/admin/adjustments/"+theirs+"/approve", fl, nil), "Adjustment request approved")
}
