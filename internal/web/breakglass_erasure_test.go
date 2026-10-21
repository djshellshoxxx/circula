package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-013: the history page is read only, admin only, and per network.
func TestBreakGlassPage(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	sys := domain.SystemActor("t")
	testutil.Must(w.env.Svc.OpenBreakGlass(w.env.Ctx, sys, service.OpenBreakGlassInput{Network: "town", Operator: "Olga",
		Reason: "restore after disk fault", TTL: time.Hour}))

	adm := w.browser(t)
	adm.login("admin@example.org")
	p := adm.get(b + "/admin/break-glass")
	adm.mustContain(p, "Operator access", "Olga", "restore after disk fault", "open")
	adm.mustContain(adm.get(b+"/admin"), "Operator access")
	// No way to change anything from the page.
	if strings.Contains(p.Body, "<form") && strings.Contains(p.Body, `action="/n/town/admin/break-glass`) {
		t.Error("the page offers a form")
	}
	if r := adm.post(b+"/admin/break-glass", p, nil); r.Status == 200 {
		t.Errorf("POST accepted: %d", r.Status)
	}
	// After expiry the page says so, without any job having run.
	w.env.Clock.Advance(2 * time.Hour)
	adm.mustContain(adm.get(b+"/admin/break-glass"), "expired")

	al := w.browser(t)
	al.login("alice@example.org")
	if r := al.get(b + "/admin/break-glass"); r.Status != 403 {
		t.Errorf("member got %d", r.Status)
	}
	anon := w.browser(t)
	if r := anon.get(b + "/admin/break-glass"); strings.Contains(r.Body, "Olga") {
		t.Errorf("anonymous saw the history (%d %s)", r.Status, r.URL)
	}
}

// SEC-018 through the pages: admin erase of a departed member, and self erase.
func TestErasurePages(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx := w.env.Ctx
	testutil.Must(w.env.Svc.DepartMember(ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.aliceM.ID, Reason: "left"}))
	testutil.Must(w.env.Svc.DepartMember(ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.bobM.ID, Reason: "left"}))

	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.mustContain(adm.get(b+"/admin/members?status=departed"), "Erase personal data")
	// Active members cannot be erased from here.
	if r := adm.get(b + "/admin/members/" + w.adminM.ID + "/erase"); r.Status != 404 {
		t.Errorf("erase form for an active member: %d", r.Status)
	}
	f := adm.get(b + "/admin/members/" + w.aliceM.ID + "/erase")
	adm.mustContain(f, "This cannot be undone", "Reason")
	noConfirm := adm.post(b+"/admin/members/"+w.aliceM.ID+"/erase", f, url.Values{"reason": {"request"}})
	adm.mustContain(noConfirm, `id="error-summary"`, "tick the box")
	noReason := adm.post(b+"/admin/members/"+w.aliceM.ID+"/erase", f, url.Values{"confirm": {"yes"}})
	adm.mustContain(noReason, `id="error-summary"`, "Reason")
	done := adm.post(b+"/admin/members/"+w.aliceM.ID+"/erase", f, url.Values{"confirm": {"yes"}, "reason": {"request 12"}})
	adm.mustContain(done, "Personal data erased", "Former member")
	if strings.Contains(done.Body, "alice@example.org") {
		t.Error("the erased name is still shown")
	}
	// Not twice.
	again := adm.post(b+"/admin/members/"+w.aliceM.ID+"/erase", f, url.Values{"confirm": {"yes"}, "reason": {"request 12"}})
	if again.Status != 409 {
		t.Errorf("second erase: %d", again.Status)
	}
	// Plain members have no such page.
	al := w.browser(t)
	al.login("bob@example.org")
	if r := al.get(b + "/admin/members/" + w.aliceM.ID + "/erase"); r.Status != 403 {
		t.Errorf("non-admin got %d", r.Status)
	}

	// Self erase by a departed member; the form is not for active members.
	sf := al.get(b + "/erase-my-data")
	al.mustContain(sf, "Erase my personal data", "Optional")
	bad := al.post(b+"/erase-my-data", sf, url.Values{})
	al.mustContain(bad, "tick the box")
	ok := al.post(b+"/erase-my-data", sf, url.Values{"confirm": {"yes"}})
	al.mustContain(ok, "erased")
	m := testutil.Must(w.env.Svc.GetMember(ctx, w.admin, w.net.ID, w.bobM.ID))
	if !strings.HasPrefix(m.DisplayName, "Former member ") {
		t.Errorf("name after self erase: %q", m.DisplayName)
	}
	act := w.browser(t)
	act.login("admin@example.org")
	if r := act.get(b + "/erase-my-data"); r.Status != 403 {
		t.Errorf("active member got %d", r.Status)
	}
}
