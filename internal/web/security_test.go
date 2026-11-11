package web

import (
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

func TestCSRFRejections(t *testing.T) {
	w := newWorld(t)
	b := w.base()

	// No token at all.
	anon := w.browser(t)
	p := anon.post("/login", nil, url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	if p.Status != 403 {
		t.Errorf("login without csrf: %d", p.Status)
	}
	anon.mustContain(p, "Form expired", "Nothing changed")

	// Token from another browser's session does not verify against this cookie.
	other := w.browser(t)
	otherTok := other.get("/login").csrf()
	f := anon.get("/login")
	p = anon.post("/login", f, url.Values{"login": {"alice@example.org"}, "password": {testPW}, "csrf": {otherTok}})
	if p.Status != 403 {
		t.Errorf("foreign token accepted: %d", p.Status)
	}

	// Signed in: state-changing posts without a token do nothing.
	a := w.browser(t)
	a.login("alice@example.org")
	form := a.get(b + "/trades/new")
	vals := url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"}, "amount": {"5.00"}, "step": {"confirm"}, "csrf": {"bogus"}}
	if p = a.post(b+"/trades/new", form, vals); p.Status != 403 {
		t.Errorf("bogus csrf accepted: %d", p.Status)
	}
	vals.Del("csrf")
	req, _ := http.NewRequest("POST", a.url+b+"/trades/new", strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if p = a.do(req); p.Status != 403 {
		t.Errorf("missing csrf accepted: %d", p.Status)
	}
	txs, _ := w.env.Svc.ListTransactions(w.env.Ctx, w.alice, w.net.ID, nil, service.Page{})
	if len(txs) != 0 {
		t.Fatalf("a rejected form created %d trades", len(txs))
	}

	// Valid token but cross-site browser signals.
	for name, h := range map[string][2]string{"origin": {"Origin", "http://evil.example"}, "fetch-site": {"Sec-Fetch-Site", "cross-site"}, "null-origin": {"Origin", "null"}} {
		good := url.Values{"csrf": {form.csrf()}, "counterparty_id": {w.bobM.ID}, "direction": {"pay"}, "amount": {"5.00"}, "step": {"confirm"}}
		req, _ := http.NewRequest("POST", a.url+b+"/trades/new", strings.NewReader(good.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set(h[0], h[1])
		if p = a.do(req); p.Status != 403 {
			t.Errorf("%s: cross-site post accepted: %d", name, p.Status)
		}
	}

	// Logout is POST-only and token-protected.
	if p = a.get("/logout"); p.Status == 200 && strings.Contains(p.Body, "signed out") {
		t.Error("GET /logout signed out")
	}
	if p = a.post("/logout", nil, nil); p.Status != 403 {
		t.Errorf("logout without csrf: %d", p.Status)
	}
	if p = a.get(b); p.URL == "/login" {
		t.Error("session was ended by a rejected logout")
	}
}

func TestHeadersAndCookies(t *testing.T) {
	// The httptest client connects from loopback, so trust it as the proxy to
	// exercise X-Forwarded-Proto.
	w := newWorldWith(t, func(o *Options) { o.Edge.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")} })
	a := w.browser(t)
	p := a.get("/login")
	csp := p.Header.Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "script-src 'self'", "style-src 'self'", "frame-ancestors 'none'", "form-action 'self'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP lacks %q: %s", want, csp)
		}
	}
	if strings.Contains(csp, "unsafe") || strings.Contains(csp, "http") {
		t.Errorf("CSP too loose: %s", csp)
	}
	if p.Header.Get("X-Content-Type-Options") != "nosniff" || p.Header.Get("X-Frame-Options") != "DENY" || p.Header.Get("X-Request-Id") == "" {
		t.Errorf("missing hardening headers: %v", p.Header)
	}
	if strings.Contains(p.Body, "<script>") || strings.Contains(p.Body, " style=") || regexp.MustCompile(`(src|href)="https?://`).MatchString(p.Body) {
		t.Error("inline script/style or external resource on login page")
	}

	for _, secure := range []bool{false, true} {
		req, _ := http.NewRequest("POST", w.srv.URL+"/login", strings.NewReader(url.Values{"login": {"alice@example.org"}, "password": {testPW}, "csrf": {p.csrf()}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if secure {
			req.Header.Set("X-Forwarded-Proto", "https")
		}
		for _, c := range a.c.Jar.Cookies(req.URL) {
			if secure && c.Name == httpx.CSRFCookie { // over HTTPS only the __Host- spelling counts (SA2-14)
				c = &http.Cookie{Name: httpx.CSRFCookieSecure, Value: c.Value}
			}
			req.AddCookie(c)
		}
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		found := false
		for _, c := range resp.Cookies() {
			if c.Name != httpx.SessionCookieName(secure) {
				continue
			}
			found = true
			if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure != secure || c.Path != "/" {
				t.Errorf("session cookie flags wrong (secure=%v): %+v", secure, c)
			}
		}
		if !found {
			t.Errorf("no session cookie set (secure=%v), status %d", secure, resp.StatusCode)
		}
	}
	// The csrf cookie carries the same flags.
	fresh := w.browser(t)
	r := fresh.get("/login")
	for _, c := range r.Header.Values("Set-Cookie") {
		if strings.HasPrefix(c, httpx.CSRFCookie+"=") && (!strings.Contains(c, "HttpOnly") || !strings.Contains(c, "SameSite=Lax")) {
			t.Errorf("csrf cookie flags: %s", c)
		}
	}
	if s := fresh.get("/static/style.css"); s.Status != 200 || !strings.Contains(s.Header.Get("Content-Type"), "css") {
		t.Errorf("static css: %d %s", s.Status, s.Header.Get("Content-Type"))
	}
}

func title(p *pg) string {
	m := regexp.MustCompile(`<h1[^>]*>([^<]*)</h1>`).FindStringSubmatch(p.Body)
	if m == nil {
		return ""
	}
	return m[1]
}

func TestAuthorisationAndIsolation(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	a := w.browser(t)
	a.login("alice@example.org")
	for _, path := range []string{"/admin", "/admin/members", "/admin/policy", "/admin/adjust", "/admin/audit", "/admin/exports", "/admin/invitations", "/admin/members/" + w.bobM.ID + "/suspend"} {
		if p := a.get(b + path); p.Status != 403 {
			t.Errorf("member got %d for %s", p.Status, path)
		}
	}
	if p := a.get(b + "/review"); p.Status != 403 {
		t.Errorf("review queue for plain member: %d", p.Status)
	}
	// Member cannot use admin POSTs even with a valid token.
	f := a.get(b)
	if p := a.post(b+"/admin/adjust", f, url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"5"}, "reason": {"x"}, "step": {"confirm"}}); p.Status != 403 {
		t.Errorf("member POST adjust: %d", p.Status)
	}

	// A signed-in outsider sees the same thing for a real network and a missing one.
	w.env.Person("eve@example.org")
	e := w.browser(t)
	e.login("eve@example.org")
	real, fake := e.get(b), e.get("/n/does-not-exist")
	if real.Status != 404 || fake.Status != 404 || title(real) != title(fake) {
		t.Errorf("isolation leak: %d %q vs %d %q", real.Status, title(real), fake.Status, title(fake))
	}
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	if p := e.get(b + "/trades/" + tx.ID); p.Status != 404 {
		t.Errorf("outsider trade view: %d", p.Status)
	}
	// A member who is not a party cannot see someone else's trade either.
	c, _ := w.env.Member(w.admin, w.net, "carol@example.org")
	_ = c
	cb := w.browser(t)
	cb.login("carol@example.org")
	if p := cb.get(b + "/trades/" + tx.ID); p.Status != 404 {
		t.Errorf("non-party trade view: %d", p.Status)
	}
	// Unknown trade and a hidden one look the same.
	if p := cb.get(b + "/trades/nope"); p.Status != 404 || title(p) != "Page not found" {
		t.Errorf("unknown trade: %d %q", p.Status, title(p))
	}
}

func TestSafeNextAndRateLimit(t *testing.T) {
	for in, want := range map[string]string{"/n/x": "/n/x", "//evil.com": "", "https://evil.com": "", "/\\evil": "", "": "", "/a\r\nb": ""} {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q)=%q want %q", in, got, want)
		}
	}
	w := newWorld(t)
	b := w.browser(t)
	f := b.get("/login")
	p := b.post("/login", f, url.Values{"login": {"alice@example.org"}, "password": {testPW}, "next": {"//evil.com"}})
	if p.URL == "//evil.com" || strings.Contains(p.URL, "evil") {
		t.Errorf("open redirect to %s", p.URL)
	}
	// Per-address limiter engages.
	rl := w.browser(t)
	f = rl.get("/login")
	var last *pg
	for i := 0; i < 14; i++ {
		last = rl.post("/login", f, url.Values{"login": {"nobody@example.org"}, "password": {"wrong wrong wrong"}})
	}
	if last.Status != 429 {
		t.Errorf("no rate limit after repeated failures: %d", last.Status)
	}
}

func TestErrorPagesHideInternals(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	p := b.get("/no/such/page")
	if p.Status != 404 {
		t.Fatalf("status %d", p.Status)
	}
	b.mustContain(p, "Page not found", "Return home", `lang="en"`)
}
