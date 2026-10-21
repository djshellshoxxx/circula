package web

import (
	"archive/zip"
	"bytes"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func TestAdminMembersAdjustPolicy(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")

	// Members list and a confirmation step that names the object and consequence.
	ml := adm.get(b + "/admin/members")
	adm.mustContain(ml, "alice@example.org", "bob@example.org", `scope="row"`, "Suspend")
	conf := adm.get(b + "/admin/members/" + w.aliceM.ID + "/suspend")
	adm.mustContain(conf, "Suspend: alice@example.org", "can no longer propose", "Reason")
	noReason := adm.post(b+"/admin/members/"+w.aliceM.ID+"/suspend", conf, url.Values{"reason": {""}})
	adm.mustContain(noReason, `id="error-summary"`, "Reason")
	done := adm.post(b+"/admin/members/"+w.aliceM.ID+"/suspend", conf, url.Values{"reason": {"rules"}})
	adm.mustContain(done, "Suspend: done for alice@example.org")

	al := w.browser(t)
	al.login("alice@example.org")
	home := al.get(b)
	al.mustContain(home, "Your membership is suspended", "cannot propose, accept or post")
	if p := al.get(b + "/directory"); p.Status != 403 {
		t.Errorf("suspended member directory: %d", p.Status)
	}
	adm.post(b+"/admin/members/"+w.aliceM.ID+"/restore", adm.get(b+"/admin/members/"+w.aliceM.ID+"/restore"), url.Values{"reason": {"ok"}})

	// Adjustment: needs a reason, shows the effect, posts after confirmation.
	f := adm.get(b + "/admin/adjust")
	bad := adm.post(b+"/admin/adjust", f, url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"10.00"}, "reason": {""}, "step": {"review"}})
	adm.mustContain(bad, `href="#f-reason"`, "a reason is required")
	rev := adm.post(b+"/admin/adjust", f, url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"10.00"}, "reason": {"welcome gift"}, "step": {"review"}})
	adm.mustContain(rev, "Their balance after", "+10.00", "welcome gift", "Go back and edit")
	if bal, _ := w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID); bal.Balance != 0 {
		t.Fatal("review posted early")
	}
	ok := adm.post(b+"/admin/adjust", rev, url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"10.00"}, "reason": {"welcome gift"}, "step": {"confirm"}})
	adm.mustContain(ok, "Adjustment posted for bob@example.org", "+10.00")
	if bal, _ := w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID); bal.Balance != 1000 {
		t.Errorf("bob balance %d", bal.Balance)
	}
	bobB := w.browser(t)
	bobB.login("bob@example.org")
	bobB.mustContain(bobB.get(b+"/history"), "Adjustment: You received 10.00 credits")

	// Policy: reason required, new version saved.
	pf := adm.get(b + "/admin/policy")
	adm.mustContain(pf, `value="100.00"`, "version 1")
	np := url.Values{"currency_name": {"credit"}, "default_credit_limit": {"150.00"}, "max_tx_amount": {"1000.00"}, "signup_grant": {"0.00"},
		"proposal_expiry_days": {"7"}, "dispute_window_days": {"30"}, "limit_warning_days": {"0"}, "join_mode": {"invite"}, "departure_rule": {"retain"}}
	if p := adm.post(b+"/admin/policy", pf, np); !strings.Contains(p.Body, `id="error-summary"`) {
		t.Error("policy without reason accepted")
	}
	np.Set("reason", "community vote")
	adm.mustContain(adm.post(b+"/admin/policy", pf, np), "Policy saved as a new version", `value="150.00"`, "version 2")
	np.Set("default_credit_limit", "abc")
	adm.mustContain(adm.post(b+"/admin/policy", pf, np), `href="#f-default_credit_limit"`)

	// Audit log and consistency check.
	au := adm.get(b + "/admin/audit")
	adm.mustContain(au, "adjustment", "member.suspended", `scope="col"`)
	chk := adm.post(b+"/admin/check", au, nil)
	adm.mustContain(chk, "Check passed")
}

func TestInvitationsAndApprovalJoin(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	f := adm.get(b + "/admin/invitations")
	p := adm.post(b+"/admin/invitations", f, url.Values{"days": {"3"}})
	adm.mustContain(p, "Invitation created", "/join/", "Unused")
	bad := adm.post(b+"/admin/invitations", f, url.Values{"days": {"500"}})
	adm.mustContain(bad, `href="#f-days"`)

	// Approval-mode network: request, pending page, admin approves.
	w.env.Person("zed@example.org")
	an, _ := w.env.Network(w.admin, "appr", func(pol *domain.Policy) { pol.JoinMode = domain.JoinApproval })
	z := w.browser(t)
	z.login("zed@example.org")
	jf := z.get("/n/appr/join")
	z.mustContain(jf, "Request to join")
	req := z.post("/n/appr/join", jf, url.Values{"display_name": {"Zed"}})
	z.mustContain(req, "Waiting for approval", "pending until an administrator decides")
	if p := z.get("/n/appr/trades"); !strings.Contains(p.Body, "Waiting for approval") {
		t.Error("pending member reached trades")
	}
	ab := w.browser(t)
	ab.login("admin@example.org")
	ms, _ := w.env.Svc.AdminListMembers(w.env.Ctx, w.admin, an.ID, []domain.MemberStatus{domain.MemberPending}, service.Page{})
	if len(ms) != 1 {
		t.Fatalf("pending members: %d", len(ms))
	}
	hm := ab.get("/n/appr/admin")
	ab.mustContain(hm, "1 membership request is waiting")
	cf := ab.get("/n/appr/admin/members/" + ms[0].ID + "/approve")
	ab.mustContain(cf, "Approve: Zed")
	ab.mustContain(ab.post("/n/appr/admin/members/"+ms[0].ID+"/approve", cf, nil), "Approve: done for Zed")
	z.mustContain(z.get("/n/appr"), "Available to spend")
}

func TestListingsDirectoryExports(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	a, bb := w.browser(t), w.browser(t)
	a.login("alice@example.org")
	bb.login("bob@example.org")

	f := a.get(b + "/listings/new")
	empty := a.post(b+"/listings/new", f, url.Values{"type": {"offer"}, "title": {""}})
	a.mustContain(empty, `id="error-summary"`)
	badPrice := a.post(b+"/listings/new", f, url.Values{"type": {"offer"}, "title": {"Bread"}, "price": {"1.234"}})
	a.mustContain(badPrice, `href="#f-price"`, "up to 2 decimal places")
	made := a.post(b+"/listings/new", f, url.Values{"type": {"offer"}, "title": {"Fresh bread"}, "description": {"Sourdough"}, "price": {"3.50"}})
	a.mustContain(made, "Listing published", "Fresh bread", "Offer")
	seen := bb.get(b + "/listings?type=offer&q=bread")
	bb.mustContain(seen, "Fresh bread", "3.50", "Propose a trade", "alice@example.org")
	if p := bb.get(b + "/listings?type=request"); strings.Contains(p.Body, "Fresh bread") {
		t.Error("type filter ignored")
	}
	ls, _ := w.env.Svc.ListListings(w.env.Ctx, w.alice, w.net.ID, service.ListingQuery{MemberID: w.aliceM.ID})
	ef := a.get(b + "/listings/" + ls[0].ID + "/edit")
	a.mustContain(ef, `value="Fresh bread"`, "Paused")
	a.post(b+"/listings/"+ls[0].ID+"/edit", ef, url.Values{"title": {"Fresh bread"}, "status": {"paused"}})
	if p := bb.get(b + "/listings"); strings.Contains(p.Body, "Fresh bread") {
		t.Error("paused listing still visible")
	}
	if p := bb.get(b + "/listings/" + ls[0].ID + "/edit"); p.Status != 404 {
		t.Errorf("editing another member's listing: %d", p.Status)
	}

	d := a.get(b + "/directory?q=BOB")
	a.mustContain(d, "1 member found", "bob@example.org")
	a.mustContain(a.get(b+"/directory?q=zzz"), "No members match")
	a.mustContain(a.get(b+"/members/"+w.bobM.ID), "Listings by bob@example.org", "has no active listings")

	// Exports: member's own, and admin ledger as a zip.
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	h := a.get(b + "/history")
	mine := a.post(b+"/export", h, url.Values{"format": {"csv"}})
	if mine.Status != 200 || mine.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("own export: %d %s", mine.Status, mine.Header.Get("Content-Type"))
	}
	if zr, err := zip.NewReader(bytes.NewReader([]byte(mine.Body)), int64(len(mine.Body))); err != nil || len(zr.File) == 0 {
		t.Errorf("own export is not a zip with files: %v", err)
	}
	adm := w.browser(t)
	adm.login("admin@example.org")
	ex := adm.post(b+"/admin/exports", adm.get(b+"/admin/exports"), url.Values{"scope": {"full"}, "format": {"json"}})
	if ex.Header.Get("Content-Type") != "application/zip" || !strings.Contains(ex.Header.Get("Content-Disposition"), "circula-town-full-json.zip") {
		t.Errorf("admin export headers: %v", ex.Header)
	}
	// Members cannot use the admin export route.
	if p := a.post(b+"/admin/exports", h, url.Values{"scope": {"full"}, "format": {"csv"}}); p.Status != 403 {
		t.Errorf("member admin export: %d", p.Status)
	}
}
