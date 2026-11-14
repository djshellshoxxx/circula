package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// logBuf is a goroutine-safe log sink.
type logBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *logBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *logBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// captureLogs returns a JSON logger writing to app, and routes slog's default
// logger and the standard log package into stray until the test ends, so
// output that bypasses the configured logger is scanned too.
func captureLogs(t *testing.T) (logger *slog.Logger, app, stray *logBuf) {
	t.Helper()
	app, stray = &logBuf{}, &logBuf{}
	prev, prevOut, prevFlags := slog.Default(), log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewTextHandler(stray, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() {
		slog.SetDefault(prev)
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return slog.New(slog.NewJSONHandler(app, &slog.HandlerOptions{Level: slog.LevelDebug})), app, stray
}

// logEntries parses JSON log output and checks the SEC-020 required fields:
// UTC timestamp, severity, event name and request identifier.
func logEntries(t *testing.T, out string) []map[string]any {
	t.Helper()
	var recs []map[string]any
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not one JSON record: %v\n%s", err, line)
		}
		ts, _ := rec["time"].(string)
		lvl, _ := rec["level"].(string)
		msg, _ := rec["msg"].(string)
		rid, _ := rec["request_id"].(string)
		if !strings.HasSuffix(ts, "Z") || lvl == "" || msg == "" || rid == "" {
			t.Errorf("log entry lacks UTC time, level, event name or request id: %s", line)
		}
		recs = append(recs, rec)
	}
	return recs
}

const forgedName = "Mallory\n{\"time\":\"2026-03-01T12:00:00Z\",\"level\":\"INFO\",\"msg\":\"login ok\",\"request_id\":\"forged01\"}\r\n" +
	"time=2026-03-01T12:00:00Z level=INFO msg=\"login ok\"\u2028"

// SEC-020: a display name carrying a newline and a fake log line, reaching
// the web adapter's panic and internal-error logging, produces one escaped
// entry each, tagged with the response's request id.
func TestWebLogForgedDisplayNameIsOneEntry(t *testing.T) {
	env := testutil.NewEnv(t)
	logger, app, stray := captureLogs(t)
	s := newServer(env.Svc, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Logger: logger})
	s.mux.HandleFunc("GET /zz-panic", func(http.ResponseWriter, *http.Request) {
		panic("rendering member " + forgedName)
	})
	s.mux.HandleFunc("GET /zz-fail", func(w http.ResponseWriter, r *http.Request) {
		s.fail(w, r, fmt.Errorf("load member %q: %w", forgedName, errors.New("disk I/O error\n"+forgedName)))
	})
	h := s.wrap(s.mux)
	var ids []string
	for _, p := range []string{"/zz-panic", "/zz-fail"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != 500 {
			t.Fatalf("%s: %d", p, rec.Code)
		}
		ids = append(ids, rec.Header().Get("X-Request-Id"))
	}
	recs := logEntries(t, app.String())
	if len(recs) != 2 {
		t.Fatalf("want 2 log entries, got %d:\n%s", len(recs), app.String())
	}
	for i, want := range []string{"web panic", "web internal error"} {
		if recs[i]["msg"] != want || recs[i]["request_id"] != ids[i] {
			t.Errorf("entry %d: %v (want %s, request id %s)", i, recs[i], want, ids[i])
		}
	}
	if strings.Contains(app.String(), "\u2028") || strings.Contains(app.String(), "forged01\",\"level") {
		t.Errorf("forgery not neutralised:\n%s", app.String())
	}
	if stray.String() != "" {
		t.Errorf("web logged outside its configured logger:\n%s", stray.String())
	}
	// The serve edge's request id is kept, so these lines match the access log.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/zz-fail", nil)
	req.Header.Set("X-Request-Id", "edge-request-0001")
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-Id"); got != "edge-request-0001" || !strings.Contains(app.String(), `"request_id":"edge-request-0001"`) {
		t.Errorf("upstream request id not kept: header %q\n%s", got, app.String())
	}
}

// canaryWorld is newWorld with the JSON API mounted beside the UI (as serve
// does), both logging to one captured logger, over a fault-injecting store.
func canaryWorld(t *testing.T) (*world, *testutil.FaultStore, *logBuf, *logBuf) {
	t.Helper()
	logger, app, stray := captureLogs(t)
	var fs *testutil.FaultStore
	env := testutil.NewEnvWith(t, func(s store.Store) store.Store { fs = testutil.NewFaultStore(s); return fs })
	w := &world{env: env}
	w.admin = env.Person("admin@example.org")
	w.net, w.adminM = env.Network(w.admin, "town", nil)
	w.alice, w.aliceM = env.Member(w.admin, w.net, "alice@example.org")
	w.bob, w.bobM = env.Member(w.admin, w.net, "bob@example.org")
	key := []byte("0123456789abcdef0123456789abcdef")
	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewHandlerWithOptions(env.Svc, api.Options{Logger: logger, CSRFKey: key}))
	mux.Handle("/", NewHandlerWith(env.Svc, Options{CSRFKey: key, Clock: env.Clock, Logger: logger}))
	w.srv = httptest.NewServer(mux)
	t.Cleanup(w.srv.Close)
	return w, fs, app, stray
}

// apiDo sends a JSON API request with the browser's cookie jar.
func (b *browser) apiDo(method, path string, hdr map[string]string, body any) (int, http.Header, map[string]any) {
	b.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, b.url+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := b.c.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, resp.Header, out
}

func (b *browser) cookie(name string) string {
	u, _ := url.Parse(b.url)
	for _, c := range b.c.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// SEC-020 acceptance: canary values for every forbidden category (password,
// session id, CSRF token, recovery codes, API and invitation tokens, TOTP
// secret) go through login, recovery, trade and export flows on the web UI and
// the JSON API, including requests that fail and are logged; a scan of all log
// output finds none of them, and every entry carries the required fields.
func TestLogsCarryNoCanaryAcrossFlows(t *testing.T) {
	w, fs, app, stray := canaryWorld(t)
	base := w.base()
	canaries := map[string]string{"password": testPW}
	add := func(kind, v string) {
		if v == "" {
			t.Fatalf("no %s captured", kind)
		}
		canaries[kind+" "+v] = v
	}

	// Login: a wrong password, then MFA enrolment (TOTP secret, MFA recovery codes).
	alice := w.browser(t)
	wrongPW := "Zq81 canary wrong password"
	add("wrong password", wrongPW)
	f := alice.get("/login")
	if p := alice.post("/login", f, url.Values{"login": {"alice@example.org"}, "password": {wrongPW}}); p.Status != 401 {
		t.Fatalf("wrong password: %d", p.Status)
	}
	secret, mfaCodes := enrolWeb(t, w, alice)
	add("totp secret", secret)
	for _, c := range mfaCodes {
		add("mfa recovery code", c)
	}
	add("session id", alice.cookie("circula_session"))
	add("csrf cookie", alice.cookie("cm_csrf"))
	add("form csrf", alice.get("/account/security").csrf())

	// A second sign-in through the second factor.
	alice2 := w.browser(t)
	f = alice2.get("/login")
	step := alice2.post("/login", f, url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	challenge := challengeOf(t, step)
	add("mfa challenge", challenge)
	if p := alice2.post("/login/mfa", step, url.Values{"challenge": {challenge}, "code": {webTOTP(t, secret, w.env.Clock.Now())}}); p.Status != 200 || p.URL == "/login/mfa" {
		t.Fatalf("second factor sign-in: %d %s", p.Status, p.URL)
	}
	add("second session id", alice2.cookie("circula_session"))
	w.env.Clock.Advance(31 * time.Second)

	// API token created from the session (cookie + CSRF header), then used.
	st, hdr, _ := alice.apiDo("GET", "/api/v1/me", nil, nil)
	if st != 200 {
		t.Fatalf("api me with session: %d", st)
	}
	apiCSRF := hdr.Get("X-CSRF-Token")
	add("api csrf", apiCSRF)
	st, _, tok := alice.apiDo("POST", "/api/v1/tokens", map[string]string{"X-CSRF-Token": apiCSRF},
		map[string]any{"name": "Zq81 token label", "scopes": []string{"read", "write"}})
	if st != 201 {
		t.Fatalf("create token: %d %v", st, tok)
	}
	aliceTok, _ := tok["token"].(string)
	add("api token", aliceTok)
	bobTok, _ := testutil.Must2(w.env.Svc.CreateAPIToken(context.Background(), w.bob,
		service.TokenInput{Name: "bob", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}))
	add("api token", bobTok)
	bearer := func(tok, idem string) map[string]string {
		h := map[string]string{"Authorization": "Bearer " + tok, "X-Network": w.net.Slug}
		if idem != "" {
			h["Idempotency-Key"] = idem
		}
		return h
	}

	// Credentials sent where they must not be: refused, never logged.
	bogus := "cir_Zq81canaryBogusToken0123456789"
	add("bogus token", bogus)
	anon := w.browser(t)
	if st, _, _ := anon.apiDo("GET", "/api/v1/me", bearer(bogus, ""), nil); st != 401 {
		t.Errorf("bogus token: %d", st)
	}
	if st, _, _ := anon.apiDo("GET", "/api/v1/me?access_token="+aliceTok, nil, nil); st != 400 {
		t.Errorf("token in query: %d", st)
	}
	inv := "inv_Zq81canaryInvitation0123456789"
	add("invitation token", inv)
	anon.get("/join/" + inv)
	anon.apiDo("POST", "/api/v1/invitations/"+inv+"/accept", bearer(bobTok, "inv-1"), map[string]any{})

	// Trade over the API; the first accept fails inside the store and is logged.
	propose := map[string]any{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "500", "description": "garden help"}
	st, _, tx := alice.apiDo("POST", "/api/v1/transactions", bearer(aliceTok, "p-1"), propose)
	if st != 201 {
		t.Fatalf("propose: %d %v", st, tx)
	}
	txID, _ := tx["id"].(string)
	fs.Arm("InsertEntry", 1)
	if st, _, _ := anon.apiDo("POST", "/api/v1/transactions/"+txID+"/accept", bearer(bobTok, "a-1"), map[string]any{}); st != 500 {
		t.Fatalf("faulted accept: %d", st)
	}
	fs.Disarm()
	if st, _, body := anon.apiDo("POST", "/api/v1/transactions/"+txID+"/accept", bearer(bobTok, "a-2"), map[string]any{}); st != 200 {
		t.Fatalf("accept: %d %v", st, body)
	}

	// Trade on the web; the accept fails inside the store and is logged.
	st, _, tx = alice.apiDo("POST", "/api/v1/transactions", bearer(aliceTok, "p-2"), propose)
	if st != 201 {
		t.Fatalf("propose 2: %d", st)
	}
	txID, _ = tx["id"].(string)
	bob := w.browser(t)
	bob.login("bob@example.org")
	add("bob session id", bob.cookie("circula_session"))
	rev := bob.get(base + "/trades/" + txID + "/accept")
	fs.Arm("InsertEntry", 1)
	if p := bob.post(base+"/trades/"+txID+"/accept", rev, nil); p.Status != 500 {
		t.Fatalf("faulted web accept: %d", p.Status)
	}
	fs.Disarm()

	// Export over the API, downloaded.
	st, _, ex := alice.apiDo("POST", "/api/v1/exports", bearer(aliceTok, "e-1"), map[string]any{"scope": "mine", "format": "csv"})
	if st != 201 {
		t.Fatalf("export: %d %v", st, ex)
	}
	link, _ := ex["artefact_url"].(string)
	req, _ := http.NewRequest("GET", w.srv.URL+link, nil)
	for k, v := range bearer(aliceTok, "") {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if link == "" || resp.StatusCode != 200 {
		t.Fatalf("artefact %q: %d", link, resp.StatusCode)
	}

	// Account recovery: codes issued, a wrong code, then a successful reset.
	f = bob.get("/account/recovery-codes")
	codes := recCodeRe.FindAllStringSubmatch(bob.post("/account/recovery-codes", f, url.Values{"password": {testPW}}).Body, -1)
	if len(codes) == 0 {
		t.Fatal("no account recovery codes")
	}
	for _, c := range codes {
		add("account recovery code", c[1])
	}
	newPW := "Zq81 canary new passphrase"
	add("new password", newPW)
	rec := w.browser(t)
	f = rec.get("/recover")
	rec.post("/recover", f, url.Values{"login": {"bob@example.org"}, "recovery_code": {"AAAA-AAAA-AAAA-AAAA"}, "password": {newPW}, "password2": {newPW}})
	if p := rec.post("/recover", f, url.Values{"login": {"bob@example.org"}, "recovery_code": {codes[0][1]},
		"password": {newPW}, "password2": {newPW}}); p.Status != 200 {
		t.Fatalf("recovery: %d %s", p.Status, snippet(p.Body))
	}

	out := app.String() + stray.String()
	for kind, v := range canaries {
		if strings.Contains(out, v) {
			t.Errorf("%s reached the log", kind)
		}
	}
	recs := logEntries(t, app.String())
	got := map[string]bool{}
	for _, r := range recs {
		got[r["msg"].(string)] = true
	}
	if !got["api internal error"] || !got["web internal error"] {
		t.Fatalf("the flow logged no internal error, so the scan proves little:\n%s", app.String())
	}
}
