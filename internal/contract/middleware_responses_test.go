package contract

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-08: errors produced before an operation's handler are still part of that
// operation's contract. Keep the shared guards visible to generated clients.
func TestCommonErrorResponsesDeclared(t *testing.T) {
	h := newHarness(t)
	for path, item := range h.doc.Paths.Map() {
		for method, op := range item.Operations() {
			want := map[int]string{413: "PayloadTooLarge", 429: "RateLimited", 503: "Unavailable"}
			switch method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				want[423] = "NetworkFrozen"
			}
			readsBody := op.RequestBody != nil
			for _, p := range append(item.Parameters, op.Parameters...) {
				if p.Value != nil && p.Value.In == "header" && p.Value.Name == "Idempotency-Key" {
					readsBody = true // replay fingerprinting also validates the body
				}
			}
			if readsBody {
				want[415] = "UnsupportedMediaType"
			}
			for status, component := range want {
				r := op.Responses.Status(status)
				if r == nil || r.Ref != "#/components/responses/"+component {
					t.Errorf("%s %s: missing shared %d %s response", method, path, status, component)
				}
			}
		}
	}
}

func TestCommonErrorResponsesMatchOpenAPI(t *testing.T) {
	for _, limiter := range []string{"IP", "principal"} {
		t.Run("429_"+limiter, func(t *testing.T) {
			h := newHarness(t)
			token := h.token(h.env.Person("alice"), domain.ScopeRead)
			opt := api.Options{CSRFKey: csrfKey, IPPerMinute: -1, PrincipalPerMinute: -1, Now: h.env.Clock.Now}
			if limiter == "IP" {
				opt.IPPerMinute = 1
			} else {
				opt.PrincipalPerMinute = 1
			}
			h.h = api.NewHandlerWithOptions(h.env.Svc, opt)
			c := call{method: "GET", path: "/me", token: token}
			h.do(200, c)
			r := h.do(429, c)
			if r.Header().Get("Retry-After") != "60" {
				t.Errorf("rate limit Retry-After = %q", r.Header().Get("Retry-After"))
			}
		})
	}
	t.Run("413_on_reads_and_writes", func(t *testing.T) {
		h := newHarness(t)
		h.h = api.NewHandlerWithOptions(h.env.Svc, api.Options{CSRFKey: csrfKey, IPPerMinute: -1, PrincipalPerMinute: -1, MaxBodyBytes: 8})
		for _, c := range []call{
			{method: "GET", path: "/me"},
			{method: "POST", path: "/listings"},
		} {
			c.body, c.skipRequest = m{"oversized": "body"}, true
			h.do(413, c)
		}
	})
	t.Run("415_for_body_decoding_and_replay", func(t *testing.T) {
		h := newHarness(t)
		admin := h.env.Person("admin")
		h.env.Network(admin, "acme", nil)
		token := h.token(admin, domain.ScopeRead, domain.ScopeWrite)
		for _, c := range []call{
			{method: "POST", path: "/listings", idem: true, hdr: m2("Content-Type", "text/plain")},
			{method: "PATCH", path: "/me", hdr: m2("Content-Type", "application/json")},
			{method: "POST", path: "/transactions/01950980-7b00-7000-8000-000000000001/accept", idem: true, hdr: m2("Content-Type", "text/plain")},
		} {
			c.token, c.body, c.skipRequest = token, m{}, true
			h.do(415, c)
		}
	})
	t.Run("423_on_frozen_write", func(t *testing.T) {
		h := newHarness(t)
		admin := h.env.Person("admin")
		net, _ := h.env.Network(admin, "acme", nil)
		token := h.token(admin, domain.ScopeRead, domain.ScopeWrite)
		h.env.Svc.FreezeNetwork(net.ID, "fixture integrity failure")
		h.do(200, call{method: "GET", path: "/network", token: token})
		c := call{method: "POST", path: "/listings", token: token, idem: true, body: m{"kind": "offer", "title": "ladder"}}
		r := h.do(423, c)
		if got := get[struct{ Code string }](t, r).Code; got != "network-frozen" {
			t.Errorf("423 code = %q", got)
		}
		h.env.Svc.UnfreezeNetwork(net.ID)
		h.do(201, c)
	})
	t.Run("503_during_authentication", func(t *testing.T) {
		h := newHarness(t)
		var fault *unavailableStore
		h.env = testutil.NewEnvWith(t, func(st store.Store) store.Store {
			fault = &unavailableStore{Store: st}
			return fault
		})
		admin := h.env.Person("admin")
		h.env.Network(admin, "acme", nil)
		token := h.token(admin, domain.ScopeRead, domain.ScopeWrite)
		session, _, err := h.env.Svc.Login(context.Background(), "admin", "correct horse battery")
		if err != nil {
			t.Fatal(err)
		}
		h.h = api.NewHandlerWithOptions(h.env.Svc, api.Options{CSRFKey: csrfKey, IPPerMinute: -1, PrincipalPerMinute: -1})
		for _, failure := range []error{domain.ErrUnavailable, errors.New("database is locked (5) (SQLITE_BUSY)")} {
			fault.err = failure
			for _, c := range []call{
				{method: "GET", path: "/me", token: token},
				{method: "GET", path: "/me", hdr: m2("Cookie", httpx.SessionCookie+"="+session)},
				{method: "POST", path: "/listings", token: token, idem: true, body: m{"kind": "offer", "title": "ladder"}},
			} {
				r := h.do(503, c)
				if r.Header().Get("Retry-After") == "" {
					t.Error("503 lacks Retry-After")
				}
			}
		}
		fault.err = nil
		h.do(200, call{method: "GET", path: "/me", token: token})
		h.do(401, call{method: "GET", path: "/me", token: "cir_invalid"})
		h.do(401, call{method: "GET", path: "/me", hdr: m2("Cookie", httpx.SessionCookie+"=invalid")})
	})
}

type unavailableStore struct {
	store.Store
	err error
}

func (s *unavailableStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if s.err != nil {
		return s.err
	}
	return s.Store.View(ctx, fn)
}
