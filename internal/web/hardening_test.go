package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

// SA-01: no control character, backslash or second slash may survive.
func TestSafeNextRejectsBrowserTricks(t *testing.T) {
	bad := []string{"/\t/evil.example", "/\v/evil.example", "/ /evil.example", "/\n/evil", "/\r/evil", "/%09/evil.example",
		"/%0d%0a/evil", "/%5cevil", "/\\evil", "//evil", "///evil", "/%2f/evil", "http://evil", "evil", "/a\x00b", "/a\x7fb", "/a\tb",
		"/" + strings.Repeat("a", 3000)}
	for _, in := range bad {
		if got := safeNext(in); got != "" {
			t.Errorf("safeNext(%q) = %q, want empty", in, got)
		}
	}
	for _, in := range []string{"/", "/n/town/trades?x=1", "/n/town#frag"} {
		if got := safeNext(in); got != in {
			t.Errorf("safeNext(%q) = %q", in, got)
		}
	}
	w := newWorld(t)
	for _, next := range []string{"/\t/evil.example", "/%09/evil.example", "/\v/evil.example"} {
		b := w.browser(t)
		f := b.get("/login")
		req, _ := http.NewRequest("POST", b.url+"/login", strings.NewReader(url.Values{"login": {"alice@example.org"},
			"password": {testPW}, "csrf": {f.csrf()}, "next": {next}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		b.c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		resp, err := b.c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if loc := resp.Header.Get("Location"); loc != "/" {
			t.Errorf("next=%q redirected to %q", next, loc)
		}
	}
}

// SA-02: cookie-secure forces Secure and the __Host- name; untrusted
// forwarding headers are ignored; HSTS follows HTTPS.
func TestSecureCookiesAndHSTS(t *testing.T) {
	w := newWorldWith(t, func(o *Options) { o.Edge.ForceSecure, o.Edge.PublicHTTPS = true, true })
	b := w.browser(t)
	f := b.get("/login")
	if f.Header.Get("Strict-Transport-Security") == "" {
		t.Error("no HSTS on https deployment")
	}
	// The jar will not return Secure cookies over http, so replay them by hand.
	var csrfCookie string
	for _, sc := range f.Header.Values("Set-Cookie") {
		if strings.HasPrefix(sc, httpx.CSRFCookieSecure+"=") {
			csrfCookie = strings.TrimPrefix(strings.SplitN(sc, ";", 2)[0], httpx.CSRFCookieSecure+"=")
		}
	}
	token := httpx.CSRF{Key: []byte("0123456789abcdef0123456789abcdef")}.Token(csrfCookie)
	req, _ := http.NewRequest("POST", b.url+"/login", strings.NewReader(url.Values{"login": {"alice@example.org"},
		"password": {testPW}, "csrf": {token}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: httpx.CSRFCookieSecure, Value: csrfCookie})
	rt, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = rt.Body.Close()
	var got bool
	for _, c := range rt.Cookies() {
		if c.Name == httpx.SessionCookieSecure {
			got = c.Secure && c.Path == "/" && c.Domain == "" && c.HttpOnly
		}
		if c.Name == httpx.SessionCookie {
			t.Error("insecure cookie name used in secure mode")
		}
	}
	if !got {
		t.Errorf("no valid __Host- session cookie: %v", rt.Cookies())
	}

	// Plain deployment: X-Forwarded-Proto from an untrusted peer is ignored.
	plain := newWorld(t)
	pb := plain.browser(t)
	r, _ := http.NewRequest("GET", pb.url+"/login", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	rt, err = http.DefaultTransport.RoundTrip(r)
	if err != nil {
		t.Fatal(err)
	}
	_ = rt.Body.Close()
	for _, c := range rt.Cookies() {
		if c.Secure {
			t.Errorf("client-supplied X-Forwarded-Proto made %s Secure", c.Name)
		}
	}
	if rt.Header.Get("Strict-Transport-Security") != "" {
		t.Error("HSTS on plain http")
	}
}

// WEB-01: the public base URL decides which Origin is accepted and what
// invitation links look like.
func TestPublicBaseURLOrigin(t *testing.T) {
	w := newWorldWith(t, func(o *Options) { o.PublicBaseURL = "https://circula.example" })
	b := w.browser(t)
	f := b.get("/login")
	post := func(origin string) int {
		req, _ := http.NewRequest("POST", b.url+"/login", strings.NewReader(url.Values{"login": {"nobody"}, "password": {"x"}, "csrf": {f.csrf()}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return b.do(req).Status
	}
	if st := post("https://circula.example"); st == 403 {
		t.Error("public origin refused")
	}
	if st := post(b.url); st != 403 { // the Host-matching origin is not the public one
		t.Errorf("non-public origin accepted: %d", st)
	}
	if st := post("https://evil.example"); st != 403 {
		t.Errorf("foreign origin accepted: %d", st)
	}
}

// SA-03: with a setup secret, only the holder creates a network, once.
func TestSetupSecretEnforced(t *testing.T) {
	w := newWorldWith(t, func(o *Options) { o.SetupSecret = "s3cret-value" })
	w.env.Person("founder@example.org")
	b := w.browser(t)
	b.login("founder@example.org")
	form := b.get("/setup")
	b.mustContain(form, "Setup secret")
	vals := func(secret, slug string) url.Values {
		return url.Values{"setup_secret": {secret}, "name": {"New"}, "slug": {slug}, "display_name": {"Founder"}, "currency_name": {"credit"},
			"decimals": {"2"}, "default_credit_limit": {"0.00"}, "join_mode": {"invite"}, "signup_grant": {"0.00"}}
	}
	if p := b.post("/setup", form, vals("", "one")); p.Status != 422 {
		t.Errorf("missing secret: %d", p.Status)
	}
	if p := b.post("/setup", form, vals("wrong", "one")); p.Status != 422 {
		t.Errorf("wrong secret: %d", p.Status)
	}
	if _, err := w.env.Svc.NetworkBySlug(w.env.Ctx, "one"); err == nil {
		t.Fatal("network created without the secret")
	}
	if p := b.post("/setup", form, vals("s3cret-value", "one")); p.URL != "/n/one/" {
		t.Fatalf("right secret: %d %s", p.Status, p.URL)
	}
	if p := b.post("/setup", form, vals("s3cret-value", "two")); p.Status != 403 {
		t.Errorf("secret reusable: %d", p.Status)
	}
	if _, err := w.env.Svc.NetworkBySlug(w.env.Ctx, "two"); err == nil {
		t.Fatal("second network created with a used secret")
	}
	if !secretEqual("a", "a") || secretEqual("a", "b") || secretEqual("a", "aa") {
		t.Error("secretEqual")
	}
}

func xffReq(b *browser, path, ip string, vals url.Values) *http.Request {
	req, _ := http.NewRequest("POST", b.url+path, strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", ip)
	return req
}

// SA-05: behind a trusted proxy, one client's failures do not block others;
// without one, a forged X-Forwarded-For does not reset the counter.
func TestLoginLimitPerClientBehindProxy(t *testing.T) {
	w := newWorldWith(t, func(o *Options) { o.Edge.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")} })
	attacker, victim := w.browser(t), w.browser(t)
	af, vf := attacker.get("/login"), victim.get("/login")
	var last *pg
	for i := 0; i < 8; i++ {
		last = attacker.do(xffReq(attacker, "/login", "198.51.100.7", url.Values{"login": {"alice@example.org"}, "password": {"wrong wrong wrong"}, "csrf": {af.csrf()}}))
	}
	if last.Status != 429 {
		t.Fatalf("attacker not limited: %d", last.Status)
	}
	p := victim.do(xffReq(victim, "/login", "203.0.113.20", url.Values{"login": {"bob@example.org"}, "password": {testPW}, "csrf": {vf.csrf()}}))
	if p.Status != 200 || strings.Contains(p.Body, "error-summary") {
		t.Errorf("other client locked out by attacker: %d", p.Status)
	}
	// Right-most untrusted entry decides: a forged leading entry changes nothing.
	again := attacker.do(xffReq(attacker, "/login", "9.9.9.9, 198.51.100.7", url.Values{"login": {"alice@example.org"}, "password": {testPW}, "csrf": {af.csrf()}}))
	if again.Status != 429 {
		t.Errorf("forged XFF entry bypassed the limit: %d", again.Status)
	}

	// No trusted proxy configured: XFF is ignored entirely.
	n := newWorld(t)
	nb := n.browser(t)
	nf := nb.get("/login")
	for i := 0; i < 8; i++ {
		last = nb.do(xffReq(nb, "/login", "10.0.0."+string(rune('1'+i)), url.Values{"login": {"alice@example.org"}, "password": {"wrong wrong wrong"}, "csrf": {nf.csrf()}}))
	}
	if last.Status != 429 {
		t.Errorf("rotating X-Forwarded-For evaded the limit: %d", last.Status)
	}
}

func TestSuccessfulLoginsDoNotUseTheBudget(t *testing.T) {
	w := newWorld(t)
	for i := 0; i < 8; i++ {
		b := w.browser(t)
		b.login("alice@example.org")
	}
}

// AUTH-03 / SEC-003: password change page.
func TestPasswordChange(t *testing.T) {
	w := newWorld(t)
	a, other := w.browser(t), w.browser(t)
	a.login("alice@example.org")
	other.login("alice@example.org")
	form := a.get("/account/password")
	a.mustContain(form, "Current password", "New password", "Confirm new password")

	set := func(cur, n, n2 string) *pg {
		return a.post("/account/password", form, url.Values{"current_password": {cur}, "password": {n}, "password2": {n2}})
	}
	a.mustContain(set("wrong wrong wrong", "a brand new password", "a brand new password"), "Current password: not correct")
	a.mustContain(set(testPW, "a brand new password", "different one entered"), "do not match")
	a.mustContain(set(testPW, "short", "short"), "at least 12 characters")
	if p := a.get(w.base()); p.URL == "/login" {
		t.Fatal("failed attempts ended the session")
	}
	p := set(testPW, "a brand new password", "a brand new password")
	if p.URL != "/login" {
		t.Fatalf("after change landed on %s (%d)", p.URL, p.Status)
	}
	a.mustContain(p, "Password changed")
	if o := other.get(w.base()); o.URL != "/login" {
		t.Error("another session survived the password change")
	}
	if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", testPW); err == nil {
		t.Error("old password still works")
	}
	if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", "a brand new password"); err != nil {
		t.Errorf("new password: %v", err)
	}
	// Anonymous callers are sent to sign in.
	if p := w.browser(t).get("/account/password"); p.URL != "/login" {
		t.Errorf("anonymous: %s", p.URL)
	}
}

func TestPasswordChangeGuessesLimited(t *testing.T) {
	w := newWorld(t)
	a := w.browser(t)
	a.login("alice@example.org")
	form := a.get("/account/password")
	var p *pg
	for i := 0; i < 7; i++ {
		p = a.post("/account/password", form, url.Values{"current_password": {"nope nope nope"}, "password": {"a brand new password"}, "password2": {"a brand new password"}})
	}
	if p.Status != 429 {
		t.Errorf("no limit on current-password guesses: %d", p.Status)
	}
}

// UX-030: the page carries the expiry data, the extend endpoint answers.
func TestSessionExpiryWarning(t *testing.T) {
	w := newWorldWith(t, func(o *Options) { o.SessionIdle, o.SessionMax = 2*time.Hour, time.Hour })
	a := w.browser(t)
	home := a.login("alice@example.org")
	a.mustContain(home, `id="session-warning"`, `data-expires="`, `data-extendable="false"`) // absolute limit (1h) < idle (2h)
	if !strings.Contains(a.get("/static/app.js").Body, "session-extend") {
		t.Error("app.js lacks the warning script")
	}
	f := a.get(w.base())
	req, _ := http.NewRequest("POST", a.url+"/session/extend", nil)
	req.Header.Set("X-CSRF-Token", f.csrf())
	p := a.do(req)
	if p.Status != 200 || !strings.Contains(p.Body, `"expires"`) || !strings.Contains(p.Body, `"extendable":false`) {
		t.Errorf("extend: %d %s", p.Status, p.Body)
	}
	req, _ = http.NewRequest("POST", a.url+"/session/extend", nil)
	if p := a.do(req); p.Status != 403 {
		t.Errorf("extend without csrf: %d", p.Status)
	}
	// Disabled when the lifetimes are not given.
	n := newWorld(t)
	nb := n.browser(t)
	if p := nb.login("alice@example.org"); strings.Contains(p.Body, "session-warning") {
		t.Error("warning without configured lifetimes")
	}
}

// CMD-05: lock contention is a retryable 503.
func TestBusyDatabaseIs503(t *testing.T) {
	w := newWorld(t)
	s := newServer(w.env.Svc, Options{})
	for _, e := range []error{errors.New("store: begin: database is locked (5) (SQLITE_BUSY)"), errors.New("x: SQLITE_BUSY")} {
		rec := httptest.NewRecorder()
		s.fail(rec, httptest.NewRequest("GET", "/", nil), e)
		if rec.Code != 503 || rec.Header().Get("Retry-After") == "" || strings.Contains(rec.Body.String(), "SQLITE") {
			t.Errorf("busy: %d %v %q", rec.Code, rec.Header(), rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	s.fail(rec, httptest.NewRequest("GET", "/", nil), errors.New("boom"))
	if rec.Code != 500 {
		t.Errorf("plain error: %d", rec.Code)
	}
}

// SA-10: registering a taken login looks exactly like registering a new one
// and never signs anyone in.
func TestRegisterIsUniform(t *testing.T) {
	w := newWorld(t)
	reg := func(login string) (*pg, *browser) {
		b := w.browser(t)
		f := b.get("/register")
		return b.post("/register", f, url.Values{"login": {login}, "password": {"another long password"}, "password2": {"another long password"}}), b
	}
	fresh, fb := reg("newcomer@example.org")
	taken, tb := reg("alice@example.org")
	if fresh.URL != "/login" || taken.URL != "/login" || fresh.Status != taken.Status {
		t.Fatalf("responses differ: %s %d vs %s %d", fresh.URL, fresh.Status, taken.URL, taken.Status)
	}
	for _, b := range []*browser{fb, tb} {
		if p := b.get("/"); p.URL != "/login" {
			t.Errorf("registration signed someone in: %s", p.URL)
		}
	}
	// The existing account's password is untouched.
	if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", testPW); err != nil {
		t.Errorf("existing account changed: %v", err)
	}
}

// WEB-02: a proposal waiting for me is shown however much newer activity exists.
func TestDashboardShowsOldProposalBehindManyTrades(t *testing.T) {
	w := newWorld(t)
	old, err := w.env.Svc.ProposeTransaction(w.env.Ctx, w.bob, w.net.ID, service.ProposeInput{CounterpartyID: w.aliceM.ID,
		Direction: service.ProposerPays, Amount: 777, Note: "old-proposal-marker", IdempotencyKey: w.env.Key()})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 105; i++ {
		w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1)
	}
	a := w.browser(t)
	a.login("alice@example.org")
	p := a.get(w.base())
	if !strings.Contains(p.Body, "/trades/"+old.ID) {
		t.Errorf("old proposal %s missing from the dashboard", old.ID)
	}
}
