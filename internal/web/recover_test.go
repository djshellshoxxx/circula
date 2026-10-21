package web

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
)

var (
	recCodeRe = regexp.MustCompile(`<code>([A-Z2-7]{4}(?:-[A-Z2-7]{4}){3})</code>`)
	tempRe = regexp.MustCompile(`<code class="temp">([^<]+)</code>`)
)

// SEC-008 web flow: generate codes (shown once), recover on /recover with
// uniform failures, old password dead, nothing signed in.
func TestWebRecoveryFlow(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")

	f := b.get("/account/recovery-codes")
	b.mustContain(f, "0</strong> unused codes")
	bad := b.post("/account/recovery-codes", f, url.Values{"password": {"not the password!"}})
	b.mustContain(bad, "Current password: not correct")
	if recCodeRe.MatchString(bad.Body) {
		t.Fatal("codes shown after a wrong password")
	}
	ok := b.post("/account/recovery-codes", f, url.Values{"password": {testPW}})
	codes := recCodeRe.FindAllStringSubmatch(ok.Body, -1)
	if len(codes) != 10 {
		t.Fatalf("got %d codes\n%s", len(codes), snippet(ok.Body))
	}
	// shown once: reloading shows only the count
	again := b.get("/account/recovery-codes")
	if recCodeRe.MatchString(again.Body) {
		t.Fatal("codes must not be shown a second time")
	}
	b.mustContain(again, "10</strong> unused codes")

	// a stranger on /recover
	anon := w.browser(t)
	f = anon.get("/recover")
	b.mustContain(f, "Recover your account")
	newPW := "a brand new passphrase"
	form := func(login, code string) url.Values {
		return url.Values{"login": {login}, "recovery_code": {code}, "password": {newPW}, "password2": {newPW}}
	}
	wrong := anon.post("/recover", f, form("alice@example.org", "AAAA-AAAA-AAAA-AAAA"))
	unknown := anon.post("/recover", f, form("nobody@example.org", "AAAA-AAAA-AAAA-AAAA"))
	if wrong.Status != 401 || unknown.Status != 401 {
		t.Fatalf("statuses %d %d", wrong.Status, unknown.Status)
	}
	if strip(wrong.Body) != strip(unknown.Body) {
		t.Fatal("unknown login must look the same as a wrong code")
	}
	mismatch := form("alice@example.org", codes[0][1])
	mismatch.Set("password2", "different")
	b.mustContain(anon.post("/recover", f, mismatch), "the two passwords do not match")

	done := anon.post("/recover", f, form("alice@example.org", codes[0][1]))
	if done.URL != "/login" {
		t.Fatalf("expected redirect to /login, at %s\n%s", done.URL, snippet(done.Body))
	}
	b.mustContain(done, "Your password was changed")
	replay := anon.post("/recover", f, form("alice@example.org", codes[0][1]))
	if replay.Status != 401 {
		t.Fatalf("replay status %d", replay.Status)
	}
	// the old browser session is dead, the old password too
	if p := b.get("/account/recovery-codes"); p.URL != "/login" {
		t.Fatalf("session must be revoked, at %s", p.URL)
	}
	lf := anon.get("/login")
	if p := anon.post("/login", lf, url.Values{"login": {"alice@example.org"}, "password": {testPW}}); p.Status != 401 {
		t.Fatalf("old password status %d", p.Status)
	}
	if p := anon.post("/login", lf, url.Values{"login": {"alice@example.org"}, "password": {newPW}}); p.Status != 200 || p.URL != "/n/town/" {
		t.Fatalf("new password login: %d %s", p.Status, p.URL)
	}
}

func strip(s string) string {
	s = regexp.MustCompile(`name="csrf" value="[^"]*"`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`name="idem" value="[^"]*"`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`name="login" type="text" value="[^"]*"`).ReplaceAllString(s, "")
	return regexp.MustCompile(`[0-9a-f]{16}`).ReplaceAllString(s, "")
}

// Admin reset page: confirm, temp password shown once, forced change redirect,
// self and cross-network refused.
func TestWebAdminReset(t *testing.T) {
	w := newWorld(t)
	adm := w.browser(t)
	adm.login("admin@example.org")
	path := w.base() + "/admin/members/" + w.aliceM.ID + "/reset-password"

	members := adm.get(w.base() + "/admin/members")
	adm.mustContain(members, "Reset password")
	confirm := adm.get(path)
	adm.mustContain(confirm, "Reset password: alice@example.org", "Reason")

	noReason := adm.post(path, confirm, url.Values{"reason": {""}})
	if noReason.Status != 422 {
		t.Fatalf("missing reason: %d", noReason.Status)
	}
	done := adm.post(path, confirm, url.Values{"reason": {"lost password and codes"}})
	m := tempRe.FindStringSubmatch(done.Body)
	if done.Status != 200 || m == nil {
		t.Fatalf("no temp password: %d\n%s", done.Status, snippet(done.Body))
	}
	temp := m[1]
	if again := adm.get(w.base() + "/admin/members"); strings.Contains(again.Body, temp) {
		t.Fatal("temp password shown twice")
	}

	// self: refused
	self := adm.get(w.base() + "/admin/members/" + w.adminM.ID + "/reset-password")
	if p := adm.post(w.base()+"/admin/members/"+w.adminM.ID+"/reset-password", self, url.Values{"reason": {"my own"}}); p.Status != 403 {
		t.Fatalf("self reset: %d", p.Status)
	}
	// a plain member cannot reach the page
	bob := w.browser(t)
	bob.login("bob@example.org")
	if p := bob.get(path); p.Status != 403 {
		t.Fatalf("member access: %d", p.Status)
	}

	// alice signs in with the temp password and is sent to the password page
	al := w.browser(t)
	lf := al.get("/login")
	p := al.post("/login", lf, url.Values{"login": {"alice@example.org"}, "password": {temp}})
	if p.URL != "/account/password" {
		t.Fatalf("forced change redirect, at %s\n%s", p.URL, snippet(p.Body))
	}
	al.mustContain(p, "Choose a new password to continue")
	for _, to := range []string{"/", w.base(), "/account/recovery-codes"} {
		if q := al.get(to); q.URL != "/account/password" {
			t.Errorf("GET %s landed on %s", to, q.URL)
		}
	}
	pf := al.get("/account/password")
	np := "my own new passphrase"
	ch := al.post("/account/password", pf, url.Values{"current_password": {temp}, "password": {np}, "password2": {np}})
	if ch.URL != "/login" {
		t.Fatalf("after change: %s\n%s", ch.URL, snippet(ch.Body))
	}
	lf = al.get("/login")
	if q := al.post("/login", lf, url.Values{"login": {"alice@example.org"}, "password": {np}}); q.URL != "/n/town/" {
		t.Fatalf("login after change at %s", q.URL)
	}
}
