package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// world is a network with an admin, two members and bearer tokens for each.
type world struct {
	t      *testing.T
	env    *testutil.Env
	h      http.Handler
	net    *domain.Network
	admin  domain.Actor
	alice  domain.Actor
	bob    domain.Actor
	adminM *domain.Member
	aliceM *domain.Member
	bobM   *domain.Member
	adminT string
	aliceT string
	bobT   string
	keySeq int
}

var testCSRFKey = []byte("0123456789abcdef0123456789abcdef")

// csrfFor is the CSRF token the API expects for a browser session.
func csrfFor(sess string) string { return httpx.CSRF{Key: testCSRFKey}.Token(sess) }

func newWorld(t *testing.T) *world { return newWorldOpts(t, api.Options{CSRFKey: testCSRFKey}) }

func newWorldOpts(t *testing.T, opt api.Options) *world {
	t.Helper()
	env := testutil.NewEnv(t)
	w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, opt)}
	w.admin = env.Person("admin")
	w.net, w.adminM = env.Network(w.admin, "acme", nil)
	w.alice, w.aliceM = env.Member(w.admin, w.net, "alice")
	w.bob, w.bobM = env.Member(w.admin, w.net, "bob")
	w.adminT = w.token(w.admin, domain.ScopeRead, domain.ScopeWrite)
	w.aliceT = w.token(w.alice, domain.ScopeRead, domain.ScopeWrite)
	w.bobT = w.token(w.bob, domain.ScopeRead, domain.ScopeWrite)
	return w
}

func (w *world) token(a domain.Actor, scopes ...domain.Scope) string {
	w.t.Helper()
	raw, _, err := w.env.Svc.CreateAPIToken(context.Background(), a, service.TokenInput{Name: "t", Scopes: scopes})
	if err != nil {
		w.t.Fatal(err)
	}
	return raw
}

func (w *world) key() string {
	w.keySeq++
	return "k-" + strings.Repeat("x", 8) + string(rune('a'+w.keySeq%26)) + itoa(w.keySeq)
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

type req struct {
	method, path, token string
	body                any
	raw                 string
	hdr                 map[string]string
	ctype               string
}

func (w *world) do(r req) *httptest.ResponseRecorder {
	w.t.Helper()
	var rd *bytes.Reader
	switch {
	case r.raw != "":
		rd = bytes.NewReader([]byte(r.raw))
	case r.body != nil:
		b, _ := json.Marshal(r.body)
		rd = bytes.NewReader(b)
	default:
		rd = bytes.NewReader(nil)
	}
	hr := httptest.NewRequest(r.method, r.path, rd)
	if r.body != nil || r.raw != "" {
		ct := r.ctype
		if ct == "" {
			ct = "application/json; charset=utf-8"
			if r.method == http.MethodPatch {
				ct = "application/merge-patch+json"
			}
		}
		hr.Header.Set("Content-Type", ct)
	}
	if r.token != "" {
		hr.Header.Set("Authorization", "Bearer "+r.token)
	}
	for k, v := range r.hdr {
		hr.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, hr)
	return rec
}

func (w *world) get(token, path string) *httptest.ResponseRecorder {
	return w.do(req{method: "GET", path: "/api/v1" + path, token: token})
}

func (w *world) post(token, path string, body any, key string) *httptest.ResponseRecorder {
	h := map[string]string{}
	if key != "" {
		h["Idempotency-Key"] = key
	}
	return w.do(req{method: "POST", path: "/api/v1" + path, token: token, body: body, hdr: h})
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
	}
	return v
}

type m = map[string]any

type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
	Errors    []struct {
		Pointer string `json:"pointer"`
		Code    string `json:"code"`
		Detail  string `json:"detail"`
	} `json:"errors"`
	CurrentStatus string `json:"current_status"`
}

// wantProblem asserts a problem+json response with status and code.
func wantProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) problem {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("content-type = %q", ct)
	}
	p := decode[problem](t, rec)
	if p.Code != code || p.Type != "urn:circula:problem:"+code || p.Status != status || p.Title == "" || p.Instance == "" {
		t.Fatalf("problem = %+v, want code %s", p, code)
	}
	if p.RequestID == "" || p.RequestID != rec.Header().Get("X-Request-Id") {
		t.Fatalf("request id mismatch: %q vs %q", p.RequestID, rec.Header().Get("X-Request-Id"))
	}
	return p
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, status, rec.Body.String())
	}
}

func testutilInvite(t *testing.T, w *world, admin domain.Actor, netID string) (string, *domain.Invitation) {
	t.Helper()
	tok, inv, err := w.env.Svc.CreateInvitation(context.Background(), admin, netID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	return tok, inv
}

func optsRate(n int) api.Options { return api.Options{PrincipalPerMinute: n, IPPerMinute: -1} }
