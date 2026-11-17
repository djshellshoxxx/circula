package web_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/qrfixture"
	"github.com/djshellshoxxx/circula/future/qr/web"
)

const (
	origin = "https://circula.example"
	csrf   = "csrf-token-0123456789"
)

var secret = []byte("0123456789abcdef0123456789abcdef-test-secret")

type auth struct{}

func (auth) Authenticate(r *http.Request) (web.Session, error) {
	m, n := r.Header.Get("X-Test-Member"), r.Header.Get("X-Test-Network")
	if m == "" || n == "" {
		return web.Session{}, web.ErrUnauthenticated
	}
	names := map[string]string{"a": "Alice", "b": "Bob", "c": "Carol", "x": "Xena"}
	return web.Session{Principal: qr.Principal{NetworkID: n, MemberID: m}, CookieAuth: r.Header.Get("X-Test-Bearer") == "", CSRF: csrf, DisplayName: names[m]}, nil
}

type fixture struct {
	t      *testing.T
	db     *sql.DB
	svc    *qr.Service
	fx     *qrfixture.Fixture
	store  *qr.Store
	h      http.Handler
	mu     sync.Mutex
	errs   []string
	failTx bool
}

func setup(t *testing.T) *fixture {
	t.Helper()
	if !haveDriver {
		t.Skip("no SQLite driver linked: run with -tags sqlite_cgo_test")
	}
	db, err := sql.Open(driverName, "file:"+filepath.Join(t.TempDir(), "w.db")+dsnOpts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	f := &fixture{t: t, db: db, fx: &qrfixture.Fixture{}}
	must(t, qrfixture.Migrate(ctx, db))
	must(t, qrfixture.AddNetwork(ctx, db, "n1", "Alpha", true))
	must(t, qrfixture.AddNetwork(ctx, db, "n2", "Beta", true))
	for _, m := range [][2]string{{"a", "Alice"}, {"b", "Bob"}, {"c", "Carol"}} {
		must(t, qrfixture.AddMember(ctx, db, "n1", m[0], m[1], 1000))
	}
	must(t, qrfixture.AddMember(ctx, db, "n2", "x", "Xena", 1000))
	f.store = qr.NewStore(db)
	must(t, f.store.Migrate(ctx))
	f.svc, err = qr.NewService(qr.Config{Origin: origin, AliasSecret: secret, NetworkEnabled: func(context.Context, qr.DBTX, string) (bool, error) { return true, nil }}, f.store, f.fx)
	must(t, err)
	f.h, err = web.NewHandler(web.Config{Service: f.svc, Auth: auth{}, LoginURL: "/login",
		OnError: func(p string, err error) { f.mu.Lock(); f.errs = append(f.errs, p+" :: "+err.Error()); f.mu.Unlock() }})
	must(t, err)
	return f
}

func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type req struct {
	method, path string
	member, net  string
	body         string
	ctype        string
	headers      map[string]string
	bearer       bool
}

func (f *fixture) do(r req) *httptest.ResponseRecorder {
	f.t.Helper()
	var body io.Reader
	if r.body != "" {
		body = strings.NewReader(r.body)
	}
	hr := httptest.NewRequest(r.method, r.path, body)
	if r.member != "" {
		hr.Header.Set("X-Test-Member", r.member)
		hr.Header.Set("X-Test-Network", orStr(r.net, "n1"))
	}
	if r.bearer {
		hr.Header.Set("X-Test-Bearer", "1")
	}
	if r.ctype != "" {
		hr.Header.Set("Content-Type", r.ctype)
	}
	for k, v := range r.headers {
		hr.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, hr)
	return w
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (f *fixture) api(method, path, member string, body any, hdr map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	r := req{method: method, path: path, member: member, bearer: true, headers: hdr}
	if body != nil {
		b, _ := json.Marshal(body)
		r.body, r.ctype = string(b), "application/json"
	}
	return f.do(r)
}

func (f *fixture) form(path, member string, v url.Values) *httptest.ResponseRecorder {
	f.t.Helper()
	if v.Get("csrf") == "" {
		v.Set("csrf", csrf)
	}
	return f.do(req{method: "POST", path: path, member: member, body: v.Encode(), ctype: "application/x-www-form-urlencoded"})
}

func jsonBody(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("%v in %q", err, w.Body.String())
	}
}

func wantStatus(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("status %d want %d: %s", w.Code, code, w.Body.String())
	}
}

func checkHeaders(t *testing.T, w *httptest.ResponseRecorder, what string) {
	t.Helper()
	h := w.Header()
	if h.Get("Cache-Control") != "no-store" || h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("%s: privacy headers missing: %v", what, h)
	}
}

func idem(k string) map[string]string { return map[string]string{"Idempotency-Key": k} }

func TestAPIFlow(t *testing.T) {
	f := setup(t)
	// static QR
	w := f.api("GET", "/api/v1/me/qr", "a", nil, nil)
	wantStatus(t, w, 200)
	checkHeaders(t, w, "me/qr")
	var me struct {
		Alias struct {
			URL, CompactCode string `json:"-"`
		}
		Raw map[string]any
	}
	var raw map[string]any
	jsonBody(t, w, &raw)
	aliasURL := raw["alias"].(map[string]any)["url"].(string)
	if !strings.HasPrefix(aliasURL, origin+"/qr/v1/a/") || strings.Contains(w.Body.String(), "balance") {
		t.Fatal(w.Body.String())
	}
	_ = me
	pw := f.api("GET", "/api/v1/me/qr.png", "a", nil, nil)
	wantStatus(t, pw, 200)
	if pw.Header().Get("Content-Type") != "image/png" || !strings.HasPrefix(pw.Header().Get("Content-Disposition"), "attachment") || !bytes.HasPrefix(pw.Body.Bytes(), []byte("\x89PNG")) {
		t.Fatal(pw.Header())
	}
	if strings.Contains(pw.Header().Get("Content-Disposition"), "/") {
		t.Fatal("filename leaks")
	}
	// rotate needs a key and is idempotent
	wantStatus(t, f.api("POST", "/api/v1/me/qr/rotate", "a", nil, nil), 400)
	r1 := f.api("POST", "/api/v1/me/qr/rotate", "a", nil, idem("rot-key-0001"))
	r2 := f.api("POST", "/api/v1/me/qr/rotate", "a", nil, idem("rot-key-0001"))
	wantStatus(t, r1, 200)
	if r1.Body.String() != r2.Body.String() {
		t.Fatal("rotate not idempotent")
	}
	// request lifecycle
	body := map[string]any{"amount_minor": 400, "note": "lunch", "expires_in_seconds": 3600}
	c := f.api("POST", "/api/v1/qr-requests", "a", body, idem("create-0001"))
	wantStatus(t, c, 201)
	var cr struct {
		Request struct {
			ID, Status string
			Completed  bool
		}
		Token, URL, CompactCode string `json:"-"`
	}
	var craw map[string]any
	jsonBody(t, c, &craw)
	token, link := craw["token"].(string), craw["url"].(string)
	id := craw["request"].(map[string]any)["id"].(string)
	if _, err := base64.StdEncoding.DecodeString(craw["png_base64"].(string)); err != nil || link != origin+"/qr/v1/"+token {
		t.Fatal(craw)
	}
	_ = cr
	again := f.api("POST", "/api/v1/qr-requests", "a", body, idem("create-0001"))
	wantStatus(t, again, 200)
	var araw map[string]any
	jsonBody(t, again, &araw)
	if araw["token"] != nil || araw["token_available"] != false {
		t.Fatalf("replay must not return a token: %v", araw)
	}
	// preview via resolve (no code in URL) and via the proposed path
	pv := f.api("POST", "/api/v1/qr-resolve", "b", map[string]string{"code": link}, nil)
	wantStatus(t, pv, 200)
	var pj map[string]any
	jsonBody(t, pv, &pj)
	if pj["payee"] != "Alice" || pj["payer"] != "Bob" || pj["network"] != "Alpha" || pj["amount_minor"].(float64) != 400 || pj["estimated_available_credit_minor"].(float64) != 1000 || !strings.Contains(pj["notice"].(string), "Nothing is paid") {
		t.Fatal(pj)
	}
	wantStatus(t, f.api("GET", "/api/v1/qr-requests/"+token, "b", nil, nil), 200)
	// claim
	cl := f.api("POST", "/api/v1/qr-claims", "b", map[string]string{"code": link}, idem("claim-key-01"))
	wantStatus(t, cl, 201)
	var clj map[string]any
	jsonBody(t, cl, &clj)
	if clj["status"] != "pending" || clj["replayed"] != false {
		t.Fatal(clj)
	}
	cl2 := f.api("POST", "/api/v1/qr-claims", "b", map[string]string{"code": link}, idem("claim-key-01"))
	wantStatus(t, cl2, 200)
	var cl2j map[string]any
	jsonBody(t, cl2, &cl2j)
	if cl2j["transaction_id"] != clj["transaction_id"] || cl2j["replayed"] != true {
		t.Fatal(cl2j)
	}
	// someone else
	wantStatus(t, f.api("POST", "/api/v1/qr-claims", "c", map[string]string{"code": link}, idem("claim-key-01")), 409)
	// status: pending until accepted, then completed
	st := f.api("GET", "/api/v1/qr-requests/"+id, "a", nil, nil)
	var stj map[string]any
	jsonBody(t, st, &stj)
	if stj["status"] != "pending" || stj["completed"] != false {
		t.Fatal(stj)
	}
	must(t, f.store.Update(context.Background(), func(ctx context.Context, q qr.DBTX) error {
		return f.fx.Accept(ctx, q, "n1", clj["transaction_id"].(string), "a")
	}))
	jsonBody(t, f.api("GET", "/api/v1/qr-requests/"+id, "a", nil, nil), &stj)
	if stj["status"] != "completed" || stj["completed"] != true {
		t.Fatal(stj)
	}
	// revoke a claimed one is a conflict; list
	wantStatus(t, f.api("POST", "/api/v1/qr-requests/"+id+"/revoke", "a", nil, nil), 409)
	lw := f.api("GET", "/api/v1/qr-requests?limit=1", "a", nil, nil)
	wantStatus(t, lw, 200)
	wantStatus(t, f.api("GET", "/api/v1/qr-requests?limit=0", "a", nil, nil), 400)
	wantStatus(t, f.api("GET", "/api/v1/qr-requests?cursor=!!!", "a", nil, nil), 400)
	// alias flow through the API
	wantStatus(t, f.api("GET", "/api/v1/qr-aliases/"+strings.TrimPrefix(aliasURL, origin+"/qr/v1/a/"), "b", nil, nil), 410) // rotated: revoked
	var rot map[string]any
	jsonBody(t, r1, &rot)
	newAlias := rot["alias"].(map[string]any)["url"].(string)
	ap := f.api("POST", "/api/v1/qr-claims", "b", map[string]any{"code": newAlias, "amount_minor": 75, "note": "tip"}, idem("alias-key-01"))
	wantStatus(t, ap, 201)
	// a request fixes its own amount/note: client values are refused
	c2 := f.api("POST", "/api/v1/qr-requests", "a", body, idem("create-0002"))
	var c2j map[string]any
	jsonBody(t, c2, &c2j)
	wantStatus(t, f.api("POST", "/api/v1/qr-claims", "c", map[string]any{"code": c2j["url"], "amount_minor": 1}, idem("claim-key-02")), 400)
	if got := f.fx.Proposed.Load(); got != 2 {
		t.Fatalf("proposals %d", got)
	}
}

func TestProblemShapesAndStatusMapping(t *testing.T) {
	f := setup(t)
	tok, _ := qr.NewToken()
	cases := []struct {
		name   string
		r      *httptest.ResponseRecorder
		status int
		code   string
	}{
		{"unauth", f.do(req{method: "GET", path: "/api/v1/me/qr"}), 401, "unauthenticated"},
		{"unknown token", f.api("GET", "/api/v1/qr-requests/"+tok, "b", nil, nil), 404, "not_found"},
		{"malformed token", f.api("GET", "/api/v1/qr-requests/zzz", "b", nil, nil), 404, "not_found"},
		{"bad json", f.do(req{method: "POST", path: "/api/v1/qr-resolve", member: "b", bearer: true, body: "{", ctype: "application/json"}), 400, "invalid_request"},
		{"unknown field", f.do(req{method: "POST", path: "/api/v1/qr-resolve", member: "b", bearer: true, body: `{"code":"x","admin":true}`, ctype: "application/json"}), 400, "invalid_request"},
		{"trailing data", f.do(req{method: "POST", path: "/api/v1/qr-resolve", member: "b", bearer: true, body: `{"code":"x"} {}`, ctype: "application/json"}), 400, "invalid_request"},
		{"media type", f.do(req{method: "POST", path: "/api/v1/qr-resolve", member: "b", bearer: true, body: `code=x`, ctype: "application/x-www-form-urlencoded"}), 415, "unsupported_media"},
		{"too large", f.do(req{method: "POST", path: "/api/v1/qr-resolve", member: "b", bearer: true, body: `{"code":"` + strings.Repeat("a", 70000) + `"}`, ctype: "application/json"}), 413, "payload_too_large"},
		{"not our code", f.api("POST", "/api/v1/qr-resolve", "b", map[string]string{"code": "https://evil.example/qr/v1/" + tok}, nil), 400, "invalid_request"},
		{"no idempotency", f.api("POST", "/api/v1/qr-requests", "a", map[string]any{"amount_minor": 5}, nil), 400, "invalid_request"},
		{"bad amount", f.api("POST", "/api/v1/qr-requests", "a", map[string]any{"amount_minor": -5}, idem("bad-amount-01")), 400, "invalid_request"},
		{"cross-network revoke", f.api("POST", "/api/v1/qr-requests/qrr_00000000000000000000000000000000/revoke", "x", nil, nil), 404, "not_found"},
		{"method", f.api("DELETE", "/api/v1/me/qr", "a", nil, nil), 405, ""},
	}
	for _, c := range cases {
		wantStatus(t, c.r, c.status)
		if c.code == "" {
			continue
		}
		if ct := c.r.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Fatalf("%s: content type %q", c.name, ct)
		}
		checkHeaders(t, c.r, c.name)
		var p map[string]any
		jsonBody(t, c.r, &p)
		if p["code"] != c.code || int(p["status"].(float64)) != c.status || p["title"] == "" || p["type"] == "" {
			t.Fatalf("%s: %v", c.name, p)
		}
		if strings.Contains(c.r.Body.String(), tok) || strings.Contains(strings.ToLower(c.r.Body.String()), "sql") || strings.Contains(c.r.Body.String(), "goroutine") {
			t.Fatalf("%s leaks: %s", c.name, c.r.Body.String())
		}
	}
	if w := f.do(req{method: "GET", path: "/api/v1/me/qr"}); w.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("401 needs WWW-Authenticate")
	}
}

func TestCSRFForCookieSessionsOnly(t *testing.T) {
	f := setup(t)
	body := `{"amount_minor":5}`
	mk := func(token string, bearer bool, key string) *httptest.ResponseRecorder {
		h := map[string]string{"Idempotency-Key": key}
		if token != "" {
			h["X-CSRF-Token"] = token
		}
		return f.do(req{method: "POST", path: "/api/v1/qr-requests", member: "a", body: body, ctype: "application/json", headers: h, bearer: bearer})
	}
	wantStatus(t, mk("", false, "csrf-key-0001"), 403)
	wantStatus(t, mk("wrong-token-0123456789", false, "csrf-key-0001"), 403)
	wantStatus(t, mk(csrf[:5], false, "csrf-key-0001"), 403)
	wantStatus(t, mk(csrf, false, "csrf-key-0001"), 201)
	wantStatus(t, mk("", true, "csrf-key-0002"), 201) // bearer/API token sessions have no ambient credentials
	var p map[string]any
	jsonBody(t, mk("", false, "csrf-key-0003"), &p)
	if p["code"] != "csrf" {
		t.Fatal(p)
	}
	// GET never needs CSRF; HTML form posts do
	wantStatus(t, f.do(req{method: "GET", path: "/api/v1/me/qr", member: "a"}), 200)
	w := f.do(req{method: "POST", path: "/qr/rotate", member: "a", body: "idempotency_key=rot-form-0001", ctype: "application/x-www-form-urlencoded"})
	wantStatus(t, w, 403)
}

func TestHTMLNoJavaScriptTradeFlow(t *testing.T) {
	f := setup(t)
	w := f.do(req{method: "GET", path: "/qr", member: "a"})
	wantStatus(t, w, 200)
	checkHeaders(t, w, "home")
	home := w.Body.String()
	a11y(t, "home", w)
	if !strings.Contains(home, "alt=\"QR code for Alice") || !strings.Contains(home, "r.") && !strings.Contains(home, "a.") {
		t.Fatal(home)
	}
	// create a request via the plain form
	cw := f.form("/qr/requests", "a", url.Values{"amount": {"800"}, "note": {"consulting"}, "ttl": {"3600"}, "idempotency_key": {"form-create-0001"}})
	wantStatus(t, cw, 201)
	a11y(t, "created", cw)
	page := cw.Body.String()
	link := regexp.MustCompile(`value="(https://circula\.example/qr/v1/[A-Za-z0-9_-]{22})"`).FindStringSubmatch(page)
	if link == nil || !strings.Contains(page, `src="data:image/png;base64,`) || !strings.Contains(page, "download=\"circula-request.png\"") || !strings.Contains(page, "Open: not paid") {
		t.Fatal(page)
	}
	if !strings.Contains(cw.Header().Get("Content-Security-Policy"), "script-src 'self'") || strings.Contains(page, "<script") {
		t.Fatal("created page must not include scripts")
	}
	// double submit = same page is NOT re-shown (the code is unrecoverable): redirect to status
	dw := f.form("/qr/requests", "a", url.Values{"amount": {"800"}, "note": {"consulting"}, "ttl": {"3600"}, "idempotency_key": {"form-create-0001"}})
	wantStatus(t, dw, 303)
	if !strings.HasPrefix(dw.Header().Get("Location"), "/qr/requests/qrr_") {
		t.Fatal(dw.Header())
	}
	// payer pastes the link into the manual-entry form (with stray whitespace)
	mw := f.form("/qr/manual", "b", url.Values{"code": {"  " + link[1] + "\n"}})
	wantStatus(t, mw, 303)
	loc := mw.Header().Get("Location")
	if loc != strings.TrimPrefix(link[1], origin) {
		t.Fatal(loc)
	}
	// confirmation screen
	vw := f.do(req{method: "GET", path: loc, member: "b"})
	wantStatus(t, vw, 200)
	a11y(t, "confirm", vw)
	v := vw.Body.String()
	for _, want := range []string{"Alpha", "Bob", "Alice", "800 units", "consulting", "1,000 units", "Nothing is paid until you confirm", "Send trade proposal to Alice"} {
		if !strings.Contains(v, want) {
			t.Fatalf("confirmation missing %q", want)
		}
	}
	key := regexp.MustCompile(`name="idempotency_key" value="([^"]+)"`).FindStringSubmatch(v)[1]
	code := regexp.MustCompile(`name="code" value="([^"]+)"`).FindStringSubmatch(v)[1]
	form := url.Values{"kind": {"request"}, "code": {code}, "idempotency_key": {key}, "amount": {"1"}, "note": {"tampered"}}
	pw := f.form("/qr/confirm", "b", form)
	wantStatus(t, pw, 200)
	if !strings.Contains(pw.Body.String(), "Pending: waiting for the other member") || strings.Contains(pw.Body.String(), "Completed") {
		t.Fatal(pw.Body.String())
	}
	// browser retry of the same POST: replay, still exactly one proposal
	pw2 := f.form("/qr/confirm", "b", form)
	wantStatus(t, pw2, 200)
	if !strings.Contains(pw2.Body.String(), "Proposal already sent") || f.fx.Proposed.Load() != 1 {
		t.Fatal(pw2.Body.String(), f.fx.Proposed.Load())
	}
	var amount int64
	must(t, f.db.QueryRow(`SELECT amount FROM fx_tx`).Scan(&amount))
	if amount != 800 {
		t.Fatal("tampered amount reached the ledger fixture:", amount)
	}
	// the recipient's own scan goes to their status page, never a second confirm; others see claimed
	ow := f.do(req{method: "GET", path: loc, member: "a"})
	wantStatus(t, ow, 303)
	wantStatus(t, f.do(req{method: "GET", path: loc, member: "c"}), 409)
	sw := f.do(req{method: "GET", path: ow.Header().Get("Location"), member: "a"})
	wantStatus(t, sw, 200)
	if !strings.Contains(sw.Body.String(), "Pending: a member confirmed") {
		t.Fatal(sw.Body.String())
	}
	// cross network
	wantStatus(t, f.do(req{method: "GET", path: loc, member: "x", net: "n2"}), 404)
}

func TestHTMLAliasFlowAndValidation(t *testing.T) {
	f := setup(t)
	pw := f.do(req{method: "GET", path: "/qr/print", member: "a"})
	wantStatus(t, pw, 200)
	a11y(t, "print", pw)
	re := regexp.MustCompile(`Code: <code>(a\.[A-Za-z0-9_-]{22})</code>`)
	compact := re.FindStringSubmatch(pw.Body.String())
	if compact == nil || !strings.Contains(pw.Body.String(), "@media print") && !strings.Contains(pw.Body.String(), "qr-print") {
		t.Fatal(pw.Body.String())
	}
	mw := f.form("/qr/manual", "b", url.Values{"code": {compact[1]}}) // compact code works too
	wantStatus(t, mw, 303)
	loc := mw.Header().Get("Location")
	if !strings.HasPrefix(loc, "/qr/v1/a/") {
		t.Fatal(loc)
	}
	aw := f.do(req{method: "GET", path: loc, member: "b"})
	wantStatus(t, aw, 200)
	a11y(t, "alias confirm", aw)
	body := aw.Body.String()
	if !strings.Contains(body, `<label for="amount">`) || !strings.Contains(body, "Alice") {
		t.Fatal(body)
	}
	key := regexp.MustCompile(`name="idempotency_key" value="([^"]+)"`).FindStringSubmatch(body)[1]
	code := regexp.MustCompile(`name="code" value="([^"]+)"`).FindStringSubmatch(body)[1]
	for _, bad := range []string{"", "12.5", "-5", "1e3", "0x10", "99999999999999999999", " "} {
		w := f.form("/qr/confirm", "b", url.Values{"kind": {"alias"}, "code": {code}, "idempotency_key": {key}, "amount": {bad}})
		if w.Code != 400 {
			t.Fatalf("amount %q: %d", bad, w.Code)
		}
		if !strings.Contains(w.Body.String(), `role="alert"`) {
			t.Fatalf("amount %q: error not announced", bad)
		}
	}
	if f.fx.Proposed.Load() != 0 {
		t.Fatal("proposal created from invalid amount")
	}
	ok := f.form("/qr/confirm", "b", url.Values{"kind": {"alias"}, "code": {code}, "idempotency_key": {key}, "amount": {"125"}, "note": {"thanks"}})
	wantStatus(t, ok, 200)
	// the owner scanning their own alias is refused
	wantStatus(t, f.do(req{method: "GET", path: loc, member: "a"}), 400)
	// unknown kind
	wantStatus(t, f.form("/qr/confirm", "b", url.Values{"kind": {"transfer"}, "code": {code}}), 400)
}

func TestManualEntryRejectsTamperedAndHostileInput(t *testing.T) {
	f := setup(t)
	tok, _ := qr.NewToken()
	good := origin + "/qr/v1/" + tok
	for _, bad := range []string{
		"", "   ", "javascript:alert(1)", "data:text/html,<script>alert(1)</script>", "//evil.example/qr/v1/" + tok,
		"https://evil.example/qr/v1/" + tok, "http://circula.example/qr/v1/" + tok, "https://circula.example.evil.example/qr/v1/" + tok,
		"https://user:pw@circula.example/qr/v1/" + tok, good + "?amount=9999", good + "#frag", good + "/", good + "%2e", origin + "/qr/v2/" + tok,
		origin + "/qr/v1/../v1/" + tok, strings.Repeat("a", 5000), "r." + tok + "x", "x." + tok, good + "\n" + good,
	} {
		w := f.form("/qr/manual", "b", url.Values{"code": {bad}})
		if w.Code != 400 {
			t.Fatalf("%q -> %d %v", bad, w.Code, w.Header().Get("Location"))
		}
		if w.Header().Get("Location") != "" || !strings.Contains(w.Body.String(), `role="alert"`) {
			t.Fatalf("%q: redirected or unannounced", bad)
		}
	}
	// a well-formed but unknown code redirects to the canonical path, which then reads as not found
	w := f.form("/qr/manual", "b", url.Values{"code": {good}})
	wantStatus(t, w, 303)
	if l := w.Header().Get("Location"); l != "/qr/v1/"+tok {
		t.Fatal(l)
	}
	wantStatus(t, f.do(req{method: "GET", path: w.Header().Get("Location"), member: "b"}), 404)
	// path-level abuse
	for _, p := range []string{"/qr/v1/" + tok + "/x", "/qr/v1/", "/qr/v1/a/", "/qr/static/../go.mod", "/qr/static/vendor/qr-scanner/LICENSE", "/qr/static/nope.js"} {
		if w := f.do(req{method: "GET", path: p, member: "b"}); w.Code != 404 && w.Code != 301 && w.Code != 307 && w.Code != 308 {
			t.Fatalf("%s -> %d", p, w.Code)
		}
	}
}

func TestUnauthenticatedHTML(t *testing.T) {
	f := setup(t)
	w := f.do(req{method: "GET", path: "/qr"})
	wantStatus(t, w, 303)
	if w.Header().Get("Location") != "/login" {
		t.Fatal(w.Header())
	}
	p := f.do(req{method: "POST", path: "/qr/manual", body: "code=x", ctype: "application/x-www-form-urlencoded"})
	wantStatus(t, p, 401)
	if strings.Contains(p.Body.String(), "Alice") {
		t.Fatal("leak")
	}
}

func TestStaticAssets(t *testing.T) {
	f := setup(t)
	for path, ct := range map[string]string{
		"/qr/static/qr.css": "text/css", "/qr/static/scan.js": "text/javascript",
		"/qr/static/vendor/qr-scanner/qr-scanner.min.js":        "text/javascript",
		"/qr/static/vendor/qr-scanner/qr-scanner-worker.min.js": "text/javascript",
	} {
		w := f.do(req{method: "GET", path: path})
		wantStatus(t, w, 200)
		if !strings.HasPrefix(w.Header().Get("Content-Type"), ct) || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Body.Len() == 0 {
			t.Fatalf("%s %v", path, w.Header())
		}
	}
	// the scan page allows its worker/media sources; other pages do not
	sw := f.do(req{method: "GET", path: "/qr/scan", member: "b"})
	wantStatus(t, sw, 200)
	a11y(t, "scan", sw)
	if !strings.Contains(sw.Header().Get("Content-Security-Policy"), "worker-src 'self' blob:") || !strings.Contains(sw.Body.String(), `<script src="/qr/static/scan.js" defer>`) {
		t.Fatal("scan page")
	}
	if strings.Contains(sw.Body.String(), `<form method="post" action="/qr/manual" id="manual-form"`) == false || !strings.Contains(sw.Body.String(), "<noscript>") {
		t.Fatal("manual entry and noscript fallback must exist")
	}
	hw := f.do(req{method: "GET", path: "/qr", member: "b"})
	if strings.Contains(hw.Header().Get("Content-Security-Policy"), "blob:") || strings.Contains(hw.Header().Get("Content-Security-Policy"), "unsafe") {
		t.Fatal("home CSP")
	}
	// scanner script must not reference remote origins or storage
	js := f.do(req{method: "GET", path: "/qr/static/scan.js"}).Body.String()
	for _, bad := range []string{"http://", "https://", "localStorage", "sessionStorage", "fetch(", "XMLHttpRequest", "sendBeacon", "eval("} {
		if strings.Contains(js, bad) {
			t.Fatalf("scan.js contains %q", bad)
		}
	}
}

type failCore struct{ *qrfixture.Fixture }

func (failCore) Propose(context.Context, qr.DBTX, qr.ProposalInput) (qr.ProposalResult, error) {
	return qr.ProposalResult{}, errors.New("database exploded: SECRET-INTERNAL-DETAIL")
}

func TestInternalErrorsAreOpaqueAndLogsAreRedacted(t *testing.T) {
	f := setup(t)
	svc, err := qr.NewService(qr.Config{Origin: origin, AliasSecret: secret, NetworkEnabled: func(context.Context, qr.DBTX, string) (bool, error) { return true, nil }}, f.store, failCore{f.fx})
	must(t, err)
	var logged []string
	h, err := web.NewHandler(web.Config{Service: svc, Auth: auth{}, OnError: func(p string, err error) { logged = append(logged, p+" "+err.Error()) }})
	must(t, err)
	f.h = h
	c := f.api("POST", "/api/v1/qr-requests", "a", map[string]any{"amount_minor": 5}, idem("create-0001"))
	var cj map[string]any
	jsonBody(t, c, &cj)
	token := cj["token"].(string)
	w := f.api("POST", "/api/v1/qr-claims", "b", map[string]string{"code": token[:0] + "r." + token}, idem("claim-key-01"))
	wantStatus(t, w, 500)
	if strings.Contains(w.Body.String(), "SECRET-INTERNAL-DETAIL") || strings.Contains(w.Body.String(), token) {
		t.Fatal(w.Body.String())
	}
	// the HTML page is equally opaque
	hw := f.do(req{method: "GET", path: "/qr/v1/" + token, member: "b"})
	wantStatus(t, hw, 200) // preview does not propose
	pw := f.form("/qr/confirm", "b", url.Values{"kind": {"request"}, "code": {token}, "idempotency_key": {"form-claim-0001"}})
	wantStatus(t, pw, 500)
	if strings.Contains(pw.Body.String(), "SECRET-INTERNAL-DETAIL") || strings.Contains(pw.Body.String(), "database") {
		t.Fatal(pw.Body.String())
	}
	if len(logged) == 0 {
		t.Fatal("unexpected errors must reach OnError")
	}
	for _, l := range logged {
		if strings.Contains(l, token) {
			t.Fatalf("token in log path: %s", l)
		}
	}
	// the failed claim burned nothing
	var n int
	must(t, f.db.QueryRow(`SELECT COUNT(*) FROM qr_trade_request WHERE claimed_tx_id IS NOT NULL`).Scan(&n))
	if n != 0 {
		t.Fatal("claimed despite failure")
	}
}

func TestRedactPath(t *testing.T) {
	tok, _ := qr.NewToken()
	for in, want := range map[string]string{
		"/qr/v1/" + tok:                     "/qr/v1/[redacted]",
		"/qr/v1/a/" + tok:                   "/qr/v1/a/[redacted]",
		"/api/v1/qr-requests/" + tok:        "/api/v1/qr-requests/[redacted]",
		"/api/v1/qr-requests/qrr_ab/revoke": "/api/v1/qr-requests/[redacted]/revoke",
		"/api/v1/qr-aliases/" + tok:         "/api/v1/qr-aliases/[redacted]",
		"/api/v1/me/qr":                     "/api/v1/me/qr",
		"/qr/requests/qrr_0":                "/qr/requests/qrr_0",
		"/proxy/qr/v1/" + tok + "?x":        "/proxy/qr/v1/[redacted]",
	} {
		if got := qr.RedactPath(in); got != want {
			t.Errorf("%s -> %s want %s", in, got, want)
		}
	}
}

func TestRequestBodyLimitOnForms(t *testing.T) {
	f := setup(t)
	w := f.do(req{method: "POST", path: "/qr/manual", member: "b", body: "csrf=" + csrf + "&code=" + strings.Repeat("a", 70000), ctype: "application/x-www-form-urlencoded"})
	wantStatus(t, w, 413)
	w = f.do(req{method: "POST", path: "/qr/manual", member: "b", body: "code=x", ctype: "multipart/form-data; boundary=x"})
	wantStatus(t, w, 415)
}

// a11y applies structural accessibility assertions that can be automated without a browser.
func a11y(t *testing.T, name string, w *httptest.ResponseRecorder) {
	t.Helper()
	b := w.Body.String()
	must1 := func(ok bool, msg string) {
		t.Helper()
		if !ok {
			t.Errorf("%s: %s", name, msg)
		}
	}
	must1(strings.HasPrefix(b, "<!doctype html>") && strings.Contains(b, `<html lang="en">`), "doctype and lang")
	must1(regexp.MustCompile(`<title>[^<]+ - Circula QR</title>`).MatchString(b), "descriptive title")
	must1(strings.Contains(b, `<a class="skip" href="#main">`) && strings.Contains(b, `<main id="main"`), "skip link and main landmark")
	must1(strings.Count(b, "<h1") == 1, "exactly one h1")
	must1(strings.Contains(b, `<nav aria-label=`), "labelled nav")
	must1(!regexp.MustCompile(`(?i)\son[a-z]+=`).MatchString(b) && !strings.Contains(b, "style="), "no inline handlers or styles (CSP)")
	for _, m := range regexp.MustCompile(`<img [^>]*>`).FindAllString(b, -1) {
		must1(regexp.MustCompile(`alt="[^"]{8,}"`).MatchString(m), "img needs meaningful alt: "+m)
	}
	// every visible input has an associated label
	for _, m := range regexp.MustCompile(`<(?:input|select)\s[^>]*>`).FindAllString(b, -1) {
		if strings.Contains(m, `type="hidden"`) || strings.Contains(m, "readonly") && false {
			continue
		}
		id := regexp.MustCompile(`id="([^"]+)"`).FindStringSubmatch(m)
		must1(id != nil && strings.Contains(b, `<label for="`+id[1]+`">`), "input without label: "+m)
	}
	// every aria-describedby points at an existing id
	for _, m := range regexp.MustCompile(`aria-describedby="([^"]+)"`).FindAllStringSubmatch(b, -1) {
		must1(strings.Contains(b, `id="`+m[1]+`"`), "dangling aria-describedby "+m[1])
	}
	// tables have captions and scoped headers
	if strings.Contains(b, "<table") {
		must1(strings.Contains(b, "<caption>") && strings.Contains(b, `<th scope="col">`), "table caption/scope")
	}
	_ = time.Now
}

func TestNotesAreEscapedInHTML(t *testing.T) {
	f := setup(t)
	evil := `<img src=x onerror=alert(1)>"'&`
	c := f.form("/qr/requests", "a", url.Values{"amount": {"5"}, "note": {evil}, "ttl": {"3600"}, "idempotency_key": {"form-xss-00001"}})
	wantStatus(t, c, 201)
	link := regexp.MustCompile(`value="(https://circula\.example/qr/v1/[A-Za-z0-9_-]{22})"`).FindStringSubmatch(c.Body.String())[1]
	for _, w := range []*httptest.ResponseRecorder{c, f.do(req{method: "GET", path: strings.TrimPrefix(link, origin), member: "b"})} {
		b := w.Body.String()
		if strings.Contains(b, "<img src=x") || strings.Contains(b, "onerror=alert(1)>") || !strings.Contains(b, "&lt;img src=x onerror=alert(1)&gt;") {
			t.Fatalf("note not escaped: %s", b)
		}
	}
}
