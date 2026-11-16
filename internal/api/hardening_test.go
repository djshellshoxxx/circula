package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web"
)

// bridge mounts the real HTML and JSON adapters on one server with the shared
// CSRF key and edge policy, as `circula serve` does.
func bridge(t *testing.T, edge httpx.Edge) (*testutil.Env, *httptest.Server) {
	t.Helper()
	env := testutil.NewEnv(t)
	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: testCSRFKey, Edge: edge}))
	mux.Handle("/", web.NewHandlerWith(env.Svc, web.Options{CSRFKey: testCSRFKey, Edge: edge, Clock: env.Clock}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return env, srv
}

var formCSRF = regexp.MustCompile(`name="csrf" value="([^"]+)"`)

// signIn signs in through the HTML form and returns the session cookie.
func signIn(t *testing.T, srv *httptest.Server, login string) (*http.Client, *http.Cookie) {
	t.Helper()
	jar := newJar()
	c := &http.Client{Jar: jar}
	resp, err := c.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	m := formCSRF.FindSubmatch(b)
	if m == nil {
		t.Fatal("no csrf token on login page")
	}
	resp, err = c.PostForm(srv.URL+"/login", url.Values{"login": {login}, "password": {"correct horse battery"}, "csrf": {string(m[1])}})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	u, _ := url.Parse(srv.URL)
	for _, ck := range jar.Cookies(u) {
		if ck.Name == httpx.SessionCookie || ck.Name == httpx.SessionCookieSecure {
			return c, ck
		}
	}
	t.Fatalf("sign-in set no session cookie (status %d)", resp.StatusCode)
	return nil, nil
}

// API-01: the cookie the web sign-in sets authenticates the API, and the CSRF
// token is the same scheme on both sides.
func TestWebSessionWorksForAPI(t *testing.T) {
	env, srv := bridge(t, httpx.Edge{})
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	_, bobM := env.Member(admin, net, "bob")
	env.Member(admin, net, "alice")
	c, sess := signIn(t, srv, "alice")

	do := func(method, path, csrf string, body string) *http.Response {
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "k-"+strings.ReplaceAll(path, "/", "")+csrf[:min(4, len(csrf))])
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := do("GET", "/api/v1/accounts/me", "", "")
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("API rejected the web session cookie: %d", resp.StatusCode)
	}
	token := resp.Header.Get("X-CSRF-Token")
	if token == "" || token != (httpx.CSRF{Key: testCSRFKey}).Token(sess.Value) {
		t.Fatalf("csrf header %q is not the shared scheme", token)
	}
	body := `{"counterparty_member_id":"` + bobM.ID + `","direction":"pay","amount_minor":"100"}`
	resp = do("POST", "/api/v1/transactions", "", body)
	_ = resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("unsafe call without csrf: %d", resp.StatusCode)
	}
	resp = do("POST", "/api/v1/transactions", token, body)
	_ = resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Errorf("unsafe call with csrf: %d", resp.StatusCode)
	}
	// The token printed into an HTML form is the same value.
	page, _ := c.Get(srv.URL + "/")
	pb, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if m := formCSRF.FindSubmatch(pb); m == nil || string(m[1]) != token {
		t.Errorf("web form token differs from the API token")
	}
}

// SA-02: with cookie-secure the sign-in cookie is __Host- prefixed and Secure,
// and the API accepts it.
func TestSecureSessionCookieWorksForAPI(t *testing.T) {
	env, srv := bridge(t, httpx.Edge{ForceSecure: true, PublicHTTPS: true})
	env.Person("alice")
	resp, err := http.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Error("no HSTS on the HTML adapter")
	}
	var csrf string
	for _, ck := range resp.Cookies() {
		if ck.Name == httpx.CSRFCookieSecure {
			csrf = ck.Value
		}
	}
	form := formCSRF.FindSubmatch(b)[1]
	req, _ := http.NewRequest("POST", srv.URL+"/login", strings.NewReader(url.Values{"login": {"alice"}, "password": {"correct horse battery"}, "csrf": {string(form)}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: httpx.CSRFCookieSecure, Value: csrf})
	resp, err = http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	var sess *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == httpx.SessionCookieSecure {
			sess = ck
		}
	}
	if sess == nil || !sess.Secure || sess.Path != "/" {
		t.Fatalf("no secure __Host- cookie: %v", resp.Cookies())
	}
	r2, _ := http.NewRequest("GET", srv.URL+"/api/v1/me", nil)
	r2.AddCookie(&http.Cookie{Name: sess.Name, Value: sess.Value})
	resp, err = http.DefaultClient.Do(r2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 || resp.Header.Get("Strict-Transport-Security") == "" {
		t.Errorf("API: %d, HSTS %q", resp.StatusCode, resp.Header.Get("Strict-Transport-Security"))
	}
}

// AUTH-03: password change over the API needs a session and revokes it.
func TestAPIChangePassword(t *testing.T) {
	w := newWorld(t)
	sess, _, err := w.env.Svc.Login(context.Background(), "alice", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	hdr := map[string]string{"Cookie": (&http.Cookie{Name: httpx.SessionCookie, Value: sess}).String(), "X-CSRF-Token": csrfFor(sess)}
	call := func(h map[string]string, token, cur, nw string) *httptest.ResponseRecorder {
		return w.do(req{method: "POST", path: "/api/v1/me/password", token: token, hdr: h, body: m{"current_password": cur, "new_password": nw}})
	}
	// A bearer token may not change the password.
	wantProblem(t, call(nil, w.aliceT, "correct horse battery", "another long password"), 403, "forbidden")
	// Wrong current password: field error, and the session survives.
	p := wantProblem(t, call(hdr, "", "wrong wrong wrong", "another long password"), 422, "validation-failed")
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/current_password" {
		t.Errorf("errors: %+v", p.Errors)
	}
	p = wantProblem(t, call(hdr, "", "correct horse battery", "short"), 422, "validation-failed")
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/new_password" {
		t.Errorf("errors: %+v", p.Errors)
	}
	wantStatus(t, w.do(req{method: "GET", path: "/api/v1/me", hdr: hdr}), 200)
	wantStatus(t, call(hdr, "", "correct horse battery", "another long password"), 204)
	wantProblem(t, w.do(req{method: "GET", path: "/api/v1/me", hdr: hdr}), 401, "unauthenticated")
	if _, _, err := w.env.Svc.Login(context.Background(), "alice", "another long password"); err != nil {
		t.Errorf("new password: %v", err)
	}
}

func TestAPIPasswordGuessesLimited(t *testing.T) {
	w := newWorld(t)
	sess, _, _ := w.env.Svc.Login(context.Background(), "alice", "correct horse battery")
	hdr := map[string]string{"Cookie": (&http.Cookie{Name: httpx.SessionCookie, Value: sess}).String(), "X-CSRF-Token": csrfFor(sess)}
	var last *httptest.ResponseRecorder
	for i := 0; i < 7; i++ {
		last = w.do(req{method: "POST", path: "/api/v1/me/password", hdr: hdr, body: m{"current_password": "nope nope nope", "new_password": "another long password"}})
	}
	wantProblem(t, last, 429, "rate-limited")
}

// FR-011 / API-059: deny a join request.
func TestAPIDenyJoinRequest(t *testing.T) {
	w := newWorld(t)
	an, _ := w.env.Network(w.admin, "appr", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	z := w.env.Person("zed")
	zm, err := w.env.Svc.JoinNetwork(context.Background(), z, an.ID, "Zed")
	if err != nil || zm.Status != domain.MemberPending {
		t.Fatalf("join: %v %+v", err, zm)
	}
	rec := w.do(req{method: "POST", path: "/api/v1/admin/members/" + zm.ID + "/deny", token: w.adminT,
		hdr: map[string]string{"X-Network": "appr", "Idempotency-Key": w.key()}, body: m{"reason": "not a neighbour"}})
	wantStatus(t, rec, 200)
	ms, _ := w.env.Svc.AdminListMembers(context.Background(), w.admin, an.ID, []domain.MemberStatus{domain.MemberPending}, service.Page{})
	if len(ms) != 0 {
		t.Errorf("still pending: %d", len(ms))
	}
	// A member cannot deny.
	wantProblem(t, w.do(req{method: "POST", path: "/api/v1/admin/members/" + zm.ID + "/deny", token: w.aliceT,
		hdr: map[string]string{"Idempotency-Key": w.key()}}), 403, "forbidden")
}

// SA-11: one member cannot fill the export store.
func TestExportCapPerPerson(t *testing.T) {
	w := newWorld(t)
	for i := 0; i < 5; i++ {
		wantStatus(t, w.post(w.aliceT, "/exports", m{"scope": "mine", "format": "csv"}, w.key()), 201)
	}
	wantProblem(t, w.post(w.aliceT, "/exports", m{"scope": "mine", "format": "csv"}, w.key()), 429, "rate-limited")
	wantStatus(t, w.post(w.bobT, "/exports", m{"scope": "mine", "format": "csv"}, w.key()), 201)
}

// API-09 / SA-05: client IP comes from X-Forwarded-For only for trusted proxies.
func TestIPRateLimitUsesTrustedProxyClientIP(t *testing.T) {
	get := func(w *world, xff string) int {
		return w.do(req{method: "GET", path: "/api/v1/me", token: w.aliceT, hdr: map[string]string{"X-Forwarded-For": xff}}).Code
	}
	trusted := newWorldOpts(t, api.Options{IPPerMinute: 3, PrincipalPerMinute: -1,
		Edge: httpx.Edge{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}}})
	for i := 0; i < 3; i++ {
		if c := get(trusted, "198.51.100.1"); c != 200 {
			t.Fatalf("request %d: %d", i, c)
		}
	}
	if c := get(trusted, "198.51.100.1"); c != 429 {
		t.Errorf("client not limited: %d", c)
	}
	if c := get(trusted, "198.51.100.2"); c != 200 {
		t.Errorf("another client behind the proxy shares the bucket: %d", c)
	}
	if c := get(trusted, "6.6.6.6, 198.51.100.1"); c != 429 {
		t.Errorf("forged left-most entry evaded the limit: %d", c)
	}
	plain := newWorldOpts(t, api.Options{IPPerMinute: 3, PrincipalPerMinute: -1})
	for i := 0; i < 3; i++ {
		get(plain, "10.0.0.1")
	}
	if c := get(plain, "10.0.0.99"); c != 429 {
		t.Errorf("untrusted X-Forwarded-For changed the bucket: %d", c)
	}
}

// API-08: malformed idempotency key has its own code.
func TestIdempotencyKeyHeaderCode(t *testing.T) {
	w := newWorld(t)
	p := wantProblem(t, w.post(w.aliceT, "/transactions", m{}, "bad\tkey"), 400, "invalid-header")
	if !strings.Contains(p.Detail, "Idempotency-Key") {
		t.Errorf("detail %q", p.Detail)
	}
}

func TestInvalidHeaderTitleIsJSON(t *testing.T) {
	var v map[string]any
	w := newWorld(t)
	rec := w.post(w.aliceT, "/transactions", m{}, "bad\tkey")
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil || v["title"] != "Invalid request header" {
		t.Errorf("%v %v", err, v)
	}
}

// API-03: entries come in ledger_seq order, newest posting first, even when
// transaction ids (assigned at proposal time) say otherwise; the cursor pages by seq.
func TestEntriesOrderedByLedgerSeq(t *testing.T) {
	w := newWorld(t)
	var ids []string
	for _, amt := range []string{"100", "200", "300"} {
		rec := w.post(w.aliceT, "/transactions", m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": amt}, w.key())
		wantStatus(t, rec, 201)
		ids = append(ids, decode[struct{ ID string }](t, rec).ID)
	}
	// Bob accepts in reverse proposal order: posting order is 3rd, 1st, 2nd.
	for _, i := range []int{2, 0, 1} {
		wantStatus(t, w.post(w.bobT, "/transactions/"+ids[i]+"/accept", nil, w.key()), 200)
	}
	type page struct {
		Data []struct {
			TransactionID string `json:"transaction_id"`
			Seq           int64  `json:"ledger_seq"`
			Name          string `json:"counterparty_display_name"`
		}
		Next *string `json:"next_cursor"`
	}
	rec := w.get(w.aliceT, "/accounts/me/entries?limit=2")
	wantStatus(t, rec, 200)
	p1 := decode[page](t, rec)
	if len(p1.Data) != 2 || p1.Next == nil || p1.Data[0].Seq <= p1.Data[1].Seq || p1.Data[0].TransactionID != ids[1] || p1.Data[1].TransactionID != ids[0] {
		t.Fatalf("first page: %+v", p1)
	}
	if p1.Data[0].Name == "" {
		t.Error("counterparty name missing")
	}
	p2 := decode[page](t, w.get(w.aliceT, "/accounts/me/entries?limit=2&cursor="+url.QueryEscape(*p1.Next)))
	if len(p2.Data) != 1 || p2.Data[0].TransactionID != ids[2] || p2.Next != nil || p2.Data[0].Seq >= p1.Data[1].Seq {
		t.Fatalf("second page: %+v", p2)
	}
}

// API-04: single reads, not scans, find members and listings.
func TestSingleReadsByID(t *testing.T) {
	w := newWorld(t)
	wantStatus(t, w.get(w.aliceT, "/members/"+w.bobM.ID), 200)
	wantProblem(t, w.get(w.aliceT, "/members/nope"), 404, "not-found")
	rec := w.post(w.aliceT, "/listings", m{"kind": "offer", "title": "Eggs", "description": "fresh", "price_minor": "100"}, w.key())
	wantStatus(t, rec, 201)
	id := decode[struct{ ID string }](t, rec).ID
	etag := rec.Header().Get("ETag")
	patch := func(tok string) *httptest.ResponseRecorder {
		return w.do(req{method: "PATCH", path: "/api/v1/listings/" + id, token: tok, body: m{"title": "Free eggs"}, hdr: map[string]string{"If-Match": etag}})
	}
	wantStatus(t, patch(w.aliceT), 200)
	wantProblem(t, patch(w.bobT), 404, "not-found") // not the owner
}
