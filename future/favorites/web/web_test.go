package web_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/internal/testkit"
	"github.com/djshellshoxxx/circula/future/favorites/memstore"
	"github.com/djshellshoxxx/circula/future/favorites/web"
)

type env struct {
	h   *web.Handler
	dir *testkit.Directory
	svc *favorites.Service
	srv *httptest.Server
}

// users maps the test cookie value to a session.
var users = map[string]web.Session{
	"alice":    {Actor: favorites.Actor{Network: "n1", Member: "alice"}, CSRF: "csrf-alice"},
	"bob":      {Actor: favorites.Actor{Network: "n1", Member: "bob"}, CSRF: "csrf-bob"},
	"alice-n2": {Actor: favorites.Actor{Network: "n2", Member: "alice"}, CSRF: "csrf-alice-n2"},
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := testkit.NewDirectory()
	for _, n := range []string{"n1", "n2"} {
		for _, id := range []string{"alice", "bob", "carol", "dave", "erin"} {
			dir.Add(n, id)
		}
	}
	dir.AddTrade("t1", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "bob", Description: "bike <b>repair</b>", Completed: true})
	svc, err := favorites.NewService(favorites.Config{Store: memstore.New(), Policy: dir, Trades: dir})
	if err != nil {
		t.Fatal(err)
	}
	h, err := web.New(web.Config{Service: svc, DirectoryURL: "/directory", Proposal: web.ProposalFields{Action: "/trades/new"},
		Auth: web.AuthFunc(func(r *http.Request) (web.Session, error) {
			c, err := r.Cookie("sid")
			if err != nil {
				return web.Session{}, errors.New("no session")
			}
			s, ok := users[c.Value]
			if !ok {
				return web.Session{}, errors.New("bad session")
			}
			return s, nil
		})})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &env{h: h, dir: dir, svc: svc, srv: srv}
}

type resp struct {
	*http.Response
	Body string
}

var client = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func (e *env) do(t *testing.T, user, method, path string, body string, hdr map[string]string) resp {
	t.Helper()
	req, err := http.NewRequest(method, e.srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if user != "" {
		req.AddCookie(&http.Cookie{Name: "sid", Value: user})
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	r, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp{r, sb.String()}
}

func (e *env) form(t *testing.T, user, path string, kv ...string) resp {
	t.Helper()
	v := url.Values{}
	if s, ok := users[user]; ok {
		v.Set("csrf_token", s.CSRF)
	}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return e.do(t, user, "POST", path, v.Encode(), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
}

func (e *env) get(t *testing.T, user, path string) resp { return e.do(t, user, "GET", path, "", nil) }

func (e *env) api(t *testing.T, user, method, path string, body any) resp {
	t.Helper()
	var b string
	switch x := body.(type) {
	case nil:
	case string:
		b = x
	default:
		j, _ := json.Marshal(x)
		b = string(j)
	}
	h := map[string]string{"Content-Type": "application/json"}
	if s, ok := users[user]; ok {
		h["X-CSRF-Token"] = s.CSRF
	}
	return e.do(t, user, method, path, b, h)
}

func wantStatus(t *testing.T, r resp, code int) {
	t.Helper()
	if r.StatusCode != code {
		t.Fatalf("status %d want %d; body: %.300s", r.StatusCode, code, r.Body)
	}
}

func (e *env) add(t *testing.T, user, target, nick string) {
	t.Helper()
	wantStatus(t, e.form(t, user, "/favorites/add", "target", target, "nickname", nick), 303)
}

func TestNoSessionIs401AndNoStore(t *testing.T) {
	e := newEnv(t)
	for _, c := range [][2]string{{"GET", "/favorites"}, {"POST", "/favorites/add"}, {"GET", "/api/v1/me/favorites"}, {"DELETE", "/api/v1/me/favorites/bob"}, {"GET", "/favorites/drafts/x"}} {
		r := e.do(t, "", c[0], c[1], "", nil)
		if r.StatusCode != 401 {
			t.Errorf("%v: %d", c, r.StatusCode)
		}
		if !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
			t.Errorf("%v: missing no-store", c)
		}
	}
	if r := e.do(t, "forged", "GET", "/favorites", "", nil); r.StatusCode != 401 {
		t.Errorf("forged session: %d", r.StatusCode)
	}
}

var styleRe = regexp.MustCompile(`(?s)<style>(.*?)</style>`)

func TestHeadersAndCSPHashMatchInlineStyle(t *testing.T) {
	e := newEnv(t)
	r := e.get(t, "alice", "/favorites")
	wantStatus(t, r, 200)
	m := styleRe.FindStringSubmatch(r.Body)
	if m == nil {
		t.Fatal("no style block")
	}
	sum := sha256.Sum256([]byte(m[1]))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	csp := r.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "style-src "+want) || strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "script-src") || !strings.Contains(csp, "default-src 'none'") {
		t.Fatalf("CSP %q does not pin the inline stylesheet %s", csp, want)
	}
	for k, v := range map[string]string{"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "X-Frame-Options": "DENY"} {
		if !strings.Contains(r.Header.Get(k), v) {
			t.Errorf("%s = %q", k, r.Header.Get(k))
		}
	}
	if !strings.Contains(r.Header.Get("Vary"), "Cookie") {
		t.Error("Vary must include Cookie")
	}
	if strings.Contains(strings.ToLower(r.Body), "<script") {
		t.Error("page must be script-free")
	}
}

func TestAccessibleMarkup(t *testing.T) {
	e := newEnv(t)
	e.add(t, "alice", "bob", "Bike guy")
	e.add(t, "alice", "carol", "")
	r := e.get(t, "alice", "/favorites")
	b := r.Body
	for _, must := range []string{`<html lang="en">`, `name="viewport"`, `<title>Saved trading partners - Circula</title>`, `class="skip" href="#main"`, `<main id="main"`,
		`<h1>Saved trading partners</h1>`, `<label for="target">`, `<label for="nickname"`, `aria-describedby="target-h"`, `href="/directory"`, `<ul class="fav">`} {
		if !strings.Contains(b, must) {
			t.Errorf("missing %s", must)
		}
	}
	if strings.Count(b, "<h1") != 1 {
		t.Error("exactly one h1")
	}
	// every text input has an id that a <label for> points to
	for _, id := range regexp.MustCompile(`<input type="text" id="([^"]+)"`).FindAllStringSubmatch(b, -1) {
		if !strings.Contains(b, `for="`+id[1]+`"`) {
			t.Errorf("input %s has no label", id[1])
		}
	}
	// action buttons carry the member's name for assistive technology
	for _, must := range []string{`Remove<span class="vh"> Member bob from saved partners</span>`, `Pin<span class="vh"> Member bob</span>`, `Start a proposal<span class="vh"> with Member bob</span>`, `Move down<span class="vh"> Member bob</span>`} {
		if !strings.Contains(b, must) {
			t.Errorf("missing accessible name %q", must)
		}
	}
	// no ambiguous controls: no link-styled buttons, no onclick/handlers
	if regexp.MustCompile(`(?i)\son[a-z]+=`).MatchString(b) || strings.Contains(b, `role="button"`) {
		t.Error("inline handlers or role=button found")
	}
	// the first item has no "Move up", the last no "Move down"
	items := strings.Split(b, `<li class="item">`)
	if strings.Contains(items[1], "Move up") || !strings.Contains(items[1], "Move down") || strings.Contains(items[2], "Move down") || !strings.Contains(items[2], "Move up") {
		t.Error("edge move buttons")
	}
	// error summary is announced and links to the field
	r = e.form(t, "alice", "/favorites/add", "target", "nobody")
	wantStatus(t, r, 422)
	for _, must := range []string{`role="alert"`, `href="#target"`, `That member can&#39;t be saved`, `value="nobody"`} {
		if !strings.Contains(r.Body, must) {
			t.Errorf("error page missing %q", must)
		}
	}
	// empty state has a directory fallback
	r = e.get(t, "bob", "/favorites")
	if !strings.Contains(r.Body, "no saved trading partners yet") || !strings.Contains(r.Body, `href="/directory"`) {
		t.Error("empty state")
	}
}

func TestCSRFAndOriginProtection(t *testing.T) {
	e := newEnv(t)
	ct := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	// missing and wrong token
	wantStatus(t, e.do(t, "alice", "POST", "/favorites/add", "target=bob", ct), 403)
	wantStatus(t, e.do(t, "alice", "POST", "/favorites/add", "target=bob&csrf_token=nope", ct), 403)
	// another user's token does not work
	wantStatus(t, e.do(t, "alice", "POST", "/favorites/add", "target=bob&csrf_token=csrf-bob", ct), 403)
	// cross-site browser signals are refused even with a right token
	for _, h := range []map[string]string{{"Origin": "https://evil.example"}, {"Sec-Fetch-Site": "cross-site"}, {"Sec-Fetch-Site": "same-site"}} {
		hh := map[string]string{"Content-Type": ct["Content-Type"]}
		for k, v := range h {
			hh[k] = v
		}
		wantStatus(t, e.do(t, "alice", "POST", "/favorites/add", "target=bob&csrf_token=csrf-alice", hh), 403)
	}
	p, _ := e.svc.List(ctxb, users["alice"].Actor, favorites.ListOptions{})
	if len(p.Items) != 0 {
		t.Fatal("refused request changed state")
	}
	// same-origin passes
	u, _ := url.Parse(e.srv.URL)
	wantStatus(t, e.do(t, "alice", "POST", "/favorites/add", "target=bob&csrf_token=csrf-alice", map[string]string{"Content-Type": ct["Content-Type"], "Origin": "http://" + u.Host, "Sec-Fetch-Site": "same-origin"}), 303)
	// GET never mutates
	for _, p := range []string{"/favorites/bob/remove", "/favorites/bob/pin", "/favorites/add"} {
		if r := e.get(t, "alice", p); r.StatusCode != 404 && r.StatusCode != 405 {
			t.Errorf("GET %s: %d", p, r.StatusCode)
		}
	}
	// oversized body
	big := "target=bob&csrf_token=csrf-alice&nickname=" + strings.Repeat("a", 70000)
	if r := e.do(t, "alice", "POST", "/favorites/add", big, ct); r.StatusCode != 400 {
		t.Errorf("oversize: %d", r.StatusCode)
	}
}

func TestMaliciousTextIsEscaped(t *testing.T) {
	e := newEnv(t)
	e.dir.Set(favorites.Member{ID: "xss", Network: "n1", Active: true, Visible: true, DisplayName: `<img src=x onerror=alert(1)>"'`})
	e.dir.Set(favorites.Member{ID: `q"><b>`, Network: "n1", Active: true, Visible: true, DisplayName: "quote"})
	e.add(t, "alice", "xss", `<script>alert(1)</script>`)
	e.add(t, "alice", `q"><b>`, `a"onfocus="x`)
	b := e.get(t, "alice", "/favorites").Body
	for _, bad := range []string{"<script>alert", "<img src=x", `"><b>`, `onfocus="x`} {
		if strings.Contains(b, bad) {
			t.Errorf("unescaped %q in page", bad)
		}
	}
	if !strings.Contains(b, "&lt;script&gt;") || !strings.Contains(b, "%22%3E%3Cb%3E") {
		t.Error("expected escaped text and path-escaped member id")
	}
	// the draft page too
	e.dir.AddTrade("tx", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "xss", Description: `</textarea><script>1</script>`, Completed: true})
	r := e.form(t, "alice", "/favorites/repeat", "trade_id", "tx", "request_key", "k")
	wantStatus(t, r, 303)
	d := e.get(t, "alice", r.Header.Get("Location"))
	wantStatus(t, d, 200)
	if strings.Contains(d.Body, "</textarea><script>") || strings.Contains(d.Body, "<img src=x") {
		t.Error("draft page escaping")
	}
}

func TestListLifecycleThroughForms(t *testing.T) {
	e := newEnv(t)
	for _, id := range []string{"bob", "carol", "dave"} {
		e.add(t, "alice", id, "n-"+id)
	}
	order := func() []string {
		p, _ := e.svc.List(ctxb, users["alice"].Actor, favorites.ListOptions{})
		var o []string
		for _, v := range p.Items {
			o = append(o, v.Target)
		}
		return o
	}
	rev := func() string {
		p, _ := e.svc.List(ctxb, users["alice"].Actor, favorites.ListOptions{})
		return fmt.Sprint(p.Revision)
	}
	r := e.form(t, "alice", "/favorites/dave/move", "dir", "up", "revision", rev())
	wantStatus(t, r, 303)
	if got := strings.Join(order(), ","); got != "bob,dave,carol" {
		t.Fatalf("move up: %s", got)
	}
	// replay the stale form: refused, nothing changes
	stale := "1"
	r = e.form(t, "alice", "/favorites/dave/move", "dir", "up", "revision", stale)
	wantStatus(t, r, 409)
	if !strings.Contains(r.Body, "Your list changed") {
		t.Error("stale message")
	}
	if got := strings.Join(order(), ","); got != "bob,dave,carol" {
		t.Fatalf("stale changed order: %s", got)
	}
	wantStatus(t, e.form(t, "alice", "/favorites/carol/move", "dir", "down", "revision", rev()), 303) // edge no-op
	if got := strings.Join(order(), ","); got != "bob,dave,carol" {
		t.Fatalf("edge: %s", got)
	}
	wantStatus(t, e.form(t, "alice", "/favorites/carol/move", "dir", "sideways", "revision", rev()), 400)
	wantStatus(t, e.form(t, "alice", "/favorites/carol/move", "dir", "up", "revision", "x"), 400)
	wantStatus(t, e.form(t, "alice", "/favorites/carol/pin", "pinned", "1"), 303)
	if got := strings.Join(order(), ","); got != "carol,bob,dave" {
		t.Fatalf("pin: %s", got)
	}
	// moving across the pinned boundary is a no-op
	wantStatus(t, e.form(t, "alice", "/favorites/bob/move", "dir", "up", "revision", rev()), 303)
	if got := strings.Join(order(), ","); got != "carol,bob,dave" {
		t.Fatalf("boundary: %s", got)
	}
	wantStatus(t, e.form(t, "alice", "/favorites/bob/nickname", "nickname", "Bobby"), 303)
	wantStatus(t, e.form(t, "alice", "/favorites/bob/nickname", "nickname", "a‮b"), 400)
	wantStatus(t, e.form(t, "alice", "/favorites/zed/nickname", "nickname", "x"), 404)
	wantStatus(t, e.form(t, "alice", "/favorites/carol/pin", "pinned", "0"), 303)
	wantStatus(t, e.form(t, "alice", "/favorites/dave/remove"), 303)
	wantStatus(t, e.form(t, "alice", "/favorites/dave/remove"), 303) // idempotent
	if got := strings.Join(order(), ","); got != "bob,carol" {
		t.Fatalf("final: %s", got)
	}
	// success notices use a fixed vocabulary; arbitrary text in ?n= is ignored
	r = e.get(t, "alice", "/favorites?n=%3Cscript%3E")
	if strings.Contains(r.Body, "<script>") || strings.Contains(r.Body, `role="status"`) {
		t.Error("notice injection")
	}
	r = e.get(t, "alice", "/favorites?n=removed")
	if !strings.Contains(r.Body, `role="status"`) {
		t.Error("notice")
	}
}

func TestAccountAndNetworkSwitchingNeverLeaks(t *testing.T) {
	e := newEnv(t)
	e.add(t, "alice", "bob", "SECRET-NICK-ALICE")
	e.add(t, "alice-n2", "carol", "SECRET-NICK-N2")
	// same browser switches account: bob's page has nothing of alice's
	b := e.get(t, "bob", "/favorites").Body
	if strings.Contains(b, "SECRET") || strings.Contains(b, "Member bob") && strings.Contains(b, "<li") {
		t.Fatalf("bob sees alice's list: %s", b)
	}
	// alice in network 2 sees only her n2 list
	b = e.get(t, "alice-n2", "/favorites").Body
	if strings.Contains(b, "SECRET-NICK-ALICE") || !strings.Contains(b, "SECRET-NICK-N2") {
		t.Fatal("network switch leak")
	}
	a := e.get(t, "alice", "/favorites").Body
	if strings.Contains(a, "SECRET-NICK-N2") || !strings.Contains(a, "SECRET-NICK-ALICE") {
		t.Fatal("network switch leak (n1)")
	}
	// API too, and X-Network cannot select another network
	r := e.api(t, "alice", "GET", "/api/v1/me/favorites", nil)
	if strings.Contains(r.Body, "N2") {
		t.Fatal("api leak")
	}
	r = e.do(t, "alice", "GET", "/api/v1/me/favorites", "", map[string]string{"X-Network": "n2"})
	wantStatus(t, r, 403)
	// bob cannot act on alice's relation through any route
	wantStatus(t, e.form(t, "bob", "/favorites/bob/remove"), 303) // his own (nonexistent) -> no-op
	if p, _ := e.svc.List(ctxb, users["alice"].Actor, favorites.ListOptions{}); len(p.Items) != 1 {
		t.Fatal("bob removed alice's favorite")
	}
	// private pages carry no-store so back/forward cache cannot resurrect them after switching
	if !strings.Contains(e.get(t, "alice", "/favorites").Header.Get("Cache-Control"), "no-store") {
		t.Fatal("no-store")
	}
}

func TestDraftFlowHTML(t *testing.T) {
	e := newEnv(t)
	e.add(t, "alice", "bob", "")
	list := e.get(t, "alice", "/favorites").Body
	keys := regexp.MustCompile(`name="request_key" value="([0-9a-f]{32})"`).FindAllStringSubmatch(list, -1)
	if len(keys) == 0 {
		t.Fatal("no request key rendered")
	}
	if again := regexp.MustCompile(`name="request_key" value="([0-9a-f]{32})"`).FindStringSubmatch(e.get(t, "alice", "/favorites").Body); again[1] == keys[0][1] {
		t.Fatal("each page render must mint a new request key")
	}
	r := e.form(t, "alice", "/favorites/bob/draft", "request_key", keys[0][1])
	wantStatus(t, r, 303)
	loc := r.Header.Get("Location")
	if loc != "/favorites/drafts/"+keys[0][1] {
		t.Fatal(loc)
	}
	d := e.get(t, "alice", loc)
	wantStatus(t, d, 200)
	intent := regexp.MustCompile(`name="idempotency_key" value="([0-9a-f]{32})"`).FindStringSubmatch(d.Body)
	if intent == nil {
		t.Fatalf("no intent: %s", d.Body)
	}
	for _, must := range []string{`action="/trades/new"`, `name="recipient" value="bob"`, `name="amount" value=""`, `name="csrf_token" value="csrf-alice"`, `<label for="amount">`, "Nothing has been proposed"} {
		if !strings.Contains(d.Body, must) {
			t.Errorf("draft page missing %q", must)
		}
	}
	// a double submit of the same form returns the same draft and key
	r2 := e.form(t, "alice", "/favorites/bob/draft", "request_key", keys[0][1])
	if r2.Header.Get("Location") != loc {
		t.Fatal("double submit")
	}
	if d2 := e.get(t, "alice", loc); !strings.Contains(d2.Body, intent[1]) {
		t.Fatal("same draft must keep its idempotency key")
	}
	// a second deliberate repeat is independent
	r3 := e.form(t, "alice", "/favorites/bob/draft", "request_key", "second-intent")
	d3 := e.get(t, "alice", r3.Header.Get("Location"))
	if strings.Contains(d3.Body, intent[1]) {
		t.Fatal("independent drafts share an idempotency key")
	}
	// other users cannot read the draft
	wantStatus(t, e.get(t, "bob", loc), 404)
	wantStatus(t, e.get(t, "alice-n2", loc), 404)
	// partner blocked after the draft was made
	e.dir.Block("n1", "alice", "bob")
	wantStatus(t, e.get(t, "alice", loc), 404)
	wantStatus(t, e.form(t, "alice", "/favorites/bob/draft", "request_key", "k9"), 422)
}

func TestRepeatRouteAndPicker(t *testing.T) {
	e := newEnv(t)
	r := e.form(t, "alice", "/favorites/repeat", "trade_id", "t1", "request_key", "rk1")
	wantStatus(t, r, 303)
	d := e.get(t, "alice", r.Header.Get("Location"))
	if !strings.Contains(d.Body, "bike &lt;b&gt;repair&lt;/b&gt;") || !strings.Contains(d.Body, `name="amount" value=""`) {
		t.Fatalf("draft body %s", d.Body)
	}
	wantStatus(t, e.form(t, "bob", "/favorites/repeat", "trade_id", "t1"), 422) // bob is the payee: repeat unsupported
}

var ctxb = context.Background()

func TestPickerFragment(t *testing.T) {
	e := newEnv(t)
	for _, id := range []string{"bob", "carol"} {
		e.add(t, "alice", id, "")
	}
	r := httptest.NewRequest("GET", "/trades/new", nil)
	r.AddCookie(&http.Cookie{Name: "sid", Value: "alice"})
	frag := string(e.h.Picker(r))
	for _, must := range []string{`aria-labelledby="fav-picker-h"`, "Member bob", "Member carol", `href="/directory"`, `action="/favorites/bob/draft"`, `name="csrf_token" value="csrf-alice"`} {
		if !strings.Contains(frag, must) {
			t.Errorf("picker missing %q", must)
		}
	}
	// blocked partner is not suggested; unauthenticated callers only get the fallback
	e.dir.Block("n1", "alice", "carol")
	if f := string(e.h.Picker(r)); strings.Contains(f, "carol") {
		t.Error("blocked partner suggested")
	}
	anon := httptest.NewRequest("GET", "/trades/new", nil)
	if f := string(e.h.Picker(anon)); strings.Contains(f, "bob") || !strings.Contains(f, `href="/directory"`) {
		t.Error("anonymous picker")
	}
	e.dir.Fail.Store(true)
	if f := string(e.h.Picker(r)); strings.Contains(f, "bob") || !strings.Contains(f, `href="/directory"`) {
		t.Error("picker must fail closed to the directory link")
	}
}

func TestPolicyOutageFailsClosedInHTML(t *testing.T) {
	e := newEnv(t)
	e.add(t, "alice", "bob", "")
	e.dir.Fail.Store(true)
	r := e.get(t, "alice", "/favorites")
	wantStatus(t, r, 503)
	if strings.Contains(r.Body, "Member bob") || !strings.Contains(r.Body, "temporarily unavailable") {
		t.Error("outage page")
	}
	wantStatus(t, e.form(t, "alice", "/favorites/add", "target", "carol"), 503)
	e.dir.Fail.Store(false)
	if p, _ := e.svc.List(ctxb, users["alice"].Actor, favorites.ListOptions{}); len(p.Items) != 1 {
		t.Fatal("state changed during outage")
	}
}

func TestSuspendedPartnerDisappearsFromPageImmediately(t *testing.T) {
	e := newEnv(t)
	e.add(t, "alice", "bob", "")
	m := e.dir.Get("n1", "bob")
	m.Blocked = true
	e.dir.Set(m)
	if b := e.get(t, "alice", "/favorites").Body; strings.Contains(b, "Member bob") {
		t.Fatal("suspended partner still listed")
	}
	// but the owner can still delete the stale row
	wantStatus(t, e.form(t, "alice", "/favorites/bob/remove"), 303)
}

func TestNewValidatesConfig(t *testing.T) {
	e := newEnv(t)
	ok := web.AuthFunc(func(*http.Request) (web.Session, error) { return web.Session{}, errors.New("x") })
	for name, c := range map[string]web.Config{
		"no service": {Auth: ok}, "no auth": {Service: e.svc}, "bad base": {Service: e.svc, Auth: ok, BasePath: "favorites"},
		"trailing slash": {Service: e.svc, Auth: ok, APIPath: "/api/"}, "open redirect dir": {Service: e.svc, Auth: ok, DirectoryURL: "//evil.example"},
		"absolute proposal": {Service: e.svc, Auth: ok, Proposal: web.ProposalFields{Action: "https://evil.example/x"}},
	} {
		if _, err := web.New(c); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
