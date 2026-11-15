package web

import (
	"regexp"
	"strings"
	"testing"
)

// UX-002: screens of the account, sign-in and administration sections must be
// reachable through rendered links, not only by typing their address. These
// three were reachable by URL alone (found by e2e/tests/navigation.spec.js).

func mustLinkTo(t *testing.T, p *pg, href string) {
	t.Helper()
	if !strings.Contains(p.Body, `href="`+href+`"`) {
		t.Errorf("page %s (status %d) has no link to %s\n%s", p.URL, p.Status, href, snippet(p.Body))
	}
}

func TestUX002SecurityPageLinksPasswordRecoveryCodes(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	sec := b.get("/account/security")
	if sec.Status != 200 {
		t.Fatalf("security page: %d", sec.Status)
	}
	mustLinkTo(t, sec, "/account/recovery-codes")
	if p := b.get("/account/recovery-codes"); p.Status != 200 {
		t.Fatalf("recovery codes page: %d", p.Status)
	}
}

func TestUX002SignInLinksAccountRecovery(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	login := b.get("/login")
	mustLinkTo(t, login, "/recover")
	if p := b.get("/recover"); p.Status != 200 {
		t.Fatalf("recover page: %d", p.Status)
	}
}

func TestUX002AdminOverviewLinksNetworkTerms(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("admin@example.org")
	ov := b.get(w.base() + "/admin")
	if ov.Status != 200 {
		t.Fatalf("admin overview: %d", ov.Status)
	}
	mustLinkTo(t, ov, w.base()+"/admin/terms")
	if p := b.get(w.base() + "/admin/terms"); p.Status != 200 {
		t.Fatalf("admin terms page: %d", p.Status)
	}
}

// UX-034: a long word (a heading such as "Announcements" at 200 percent text
// on a 320 pixel screen, or an identifier) must wrap instead of widening the
// page. Every element inherits the wrap from body. Found by
// e2e/tests/resize.spec.js.
func TestUX034BodyWrapsLongWords(t *testing.T) {
	w := newWorld(t)
	css := w.browser(t).get("/static/style.css")
	if css.Status != 200 {
		t.Fatalf("style.css: %d", css.Status)
	}
	body := regexp.MustCompile(`(?m)^body\{[^}]*\}`).FindString(css.Body)
	if !strings.Contains(body, "overflow-wrap:break-word") {
		t.Errorf("the body rule does not wrap long words: %q", body)
	}
}
