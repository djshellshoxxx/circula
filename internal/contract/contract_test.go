// Package contract checks the running JSON API against docs/api/openapi.yaml:
// every response (status code, headers declared required, JSON body) produced
// while exercising the API is validated against the document, and every
// documented operation must be exercised at least once.
package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

var csrfKey = []byte("0123456789abcdef0123456789abcdef")

type m = map[string]any

type harness struct {
	t      *testing.T
	env    *testutil.Env
	h      http.Handler
	doc    *openapi3.T
	router routers.Router
	seen   map[string]bool // "METHOD /path status"
	ops    map[string]bool // operations exercised
	keys   int
}

type call struct {
	method, path, token string
	body                any
	hdr                 map[string]string
	idem                bool
	skipRequest         bool // deliberately invalid request
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	l := openapi3.NewLoader()
	doc, err := l.LoadFromFile("../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(l.Context); err != nil {
		t.Fatalf("openapi.yaml is not a valid OpenAPI document: %v", err)
	}
	rt, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	env := testutil.NewEnv(t)
	return &harness{t: t, env: env, doc: doc, router: rt,
		h:    api.NewHandlerWithOptions(env.Svc, api.Options{CSRFKey: csrfKey, IPPerMinute: -1, PrincipalPerMinute: -1}),
		seen: map[string]bool{}, ops: map[string]bool{}}
}

func (h *harness) key() string { h.keys++; return fmt.Sprintf("contract-key-%d", h.keys) }

func (h *harness) token(a domain.Actor, scopes ...domain.Scope) string {
	raw, _, err := h.env.Svc.CreateAPIToken(context.Background(), a, service.TokenInput{Name: "t", Scopes: scopes})
	if err != nil {
		h.t.Fatal(err)
	}
	return raw
}

// do performs the call and validates request and response against the document.
// want is asserted so a silent behaviour change is not mistaken for a pass.
func (h *harness) do(want int, c call) *httptest.ResponseRecorder {
	h.t.Helper()
	var rd io.Reader = bytes.NewReader(nil)
	if c.body != nil {
		b, _ := json.Marshal(c.body)
		rd = bytes.NewReader(b)
	}
	path := "/api/v1" + c.path
	r := httptest.NewRequest(c.method, path, rd)
	if c.body != nil {
		ct := "application/json; charset=utf-8"
		if c.method == http.MethodPatch {
			ct = "application/merge-patch+json"
		}
		r.Header.Set("Content-Type", ct)
	}
	if c.token != "" {
		r.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.idem {
		r.Header.Set("Idempotency-Key", h.key())
	}
	for k, v := range c.hdr {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, r)
	if rec.Code != want {
		h.t.Errorf("%s %s: status %d, want %d; body %s", c.method, path, rec.Code, want, rec.Body.String())
	}

	// Routing: the concrete path must be documented.
	vr := httptest.NewRequest(c.method, path, bytes.NewReader(mustJSON(c.body)))
	vr.Header = r.Header.Clone()
	route, params, err := h.router.FindRoute(vr)
	if err != nil {
		h.t.Errorf("%s %s: not documented in openapi.yaml: %v", c.method, path, err)
		return rec
	}
	h.ops[route.Method+" "+route.Path] = true
	h.seen[fmt.Sprintf("%s %s %d", route.Method, route.Path, rec.Code)] = true

	noAuth := func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
	rin := &openapi3filter.RequestValidationInput{Request: vr, PathParams: params, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: noAuth}}
	if !c.skipRequest {
		if err := openapi3filter.ValidateRequest(context.Background(), rin); err != nil {
			h.t.Errorf("%s %s: request does not match openapi.yaml: %s", c.method, path, short(err))
		}
	}
	out := &openapi3filter.ResponseValidationInput{RequestValidationInput: rin, Status: rec.Code,
		Header: rec.Header(), Options: &openapi3filter.Options{IncludeResponseStatus: true}}
	out.SetBodyBytes(rec.Body.Bytes())
	if err := openapi3filter.ValidateResponse(context.Background(), out); err != nil {
		h.t.Errorf("%s %s -> %d: response does not match openapi.yaml: %s\nbody: %s", c.method, path, rec.Code, short(err), trunc(rec.Body.String()))
	}
	h.checkHeaders(route, rec)
	return rec
}

// checkHeaders asserts that headers the document marks required are present.
func (h *harness) checkHeaders(route *routers.Route, rec *httptest.ResponseRecorder) {
	h.t.Helper()
	ref := route.Operation.Responses.Status(rec.Code)
	if ref == nil {
		ref = route.Operation.Responses.Default()
	}
	if ref == nil || ref.Value == nil {
		return
	}
	for name, hr := range ref.Value.Headers {
		if hr.Value != nil && hr.Value.Required && rec.Header().Get(name) == "" {
			h.t.Errorf("%s %s -> %d: documented required header %s missing", route.Method, route.Path, rec.Code, name)
		}
	}
}

func mustJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

// short drops the schema dump kin-openapi appends to errors.
func short(err error) string {
	s := err.Error()
	if i := strings.Index(s, "\nSchema:"); i >= 0 {
		s = s[:i]
	}
	return s
}

func trunc(s string) string {
	if len(s) > 600 {
		return s[:600] + "..."
	}
	return s
}

func get[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
	}
	return v
}

type idOnly struct {
	ID       string `json:"id"`
	Token    string `json:"token"`
	MemberID string `json:"member_id"`
}

func TestAPIMatchesOpenAPI(t *testing.T) {
	h := newHarness(t)
	env := h.env
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	alice, aliceM := env.Member(admin, net, "alice")
	bob, bobM := env.Member(admin, net, "bob")
	rw := []domain.Scope{domain.ScopeRead, domain.ScopeWrite}
	adminT, aliceT, bobT := h.token(admin, rw...), h.token(alice, rw...), h.token(bob, rw...)
	roT := h.token(alice, domain.ScopeRead)
	outsiderT := h.token(env.Person("eve"), rw...)

	// Network, identity, directory, account.
	h.do(200, call{method: "GET", path: "/network", token: aliceT})
	h.do(401, call{method: "GET", path: "/network"})
	h.do(200, call{method: "GET", path: "/me", token: aliceT})
	h.do(200, call{method: "GET", path: "/members", token: aliceT})
	h.do(200, call{method: "GET", path: "/members?limit=1", token: aliceT})
	h.do(200, call{method: "GET", path: "/members/" + bobM.ID, token: aliceT})
	h.do(404, call{method: "GET", path: "/members/nope", token: aliceT})
	h.do(200, call{method: "GET", path: "/accounts/me", token: aliceT})
	h.do(200, call{method: "GET", path: "/accounts/me/entries", token: aliceT})
	h.do(400, call{method: "GET", path: "/accounts/me/entries?limit=0", token: aliceT, skipRequest: true})
	h.do(200, call{method: "GET", path: "/network/announcements", token: aliceT})
	h.do(200, call{method: "PATCH", path: "/me", token: aliceT, body: m{"time_zone": "Europe/Stockholm"}})
	h.do(422, call{method: "PATCH", path: "/me", token: aliceT, body: m{"time_zone": "Mars/Base"}, skipRequest: true})
	h.do(422, call{method: "PATCH", path: "/me", token: aliceT, body: m{}, skipRequest: true})

	// Listings.
	rec := h.do(201, call{method: "POST", path: "/listings", token: aliceT, idem: true,
		body: m{"kind": "offer", "title": "bike repair", "description": "d", "price_minor": "500"}})
	lid := get[idOnly](t, rec).ID
	letag := rec.Header().Get("ETag")
	h.do(422, call{method: "POST", path: "/listings", token: aliceT, idem: true, body: m{"kind": "offer", "title": ""}, skipRequest: true})
	h.do(200, call{method: "GET", path: "/listings", token: bobT})
	h.do(200, call{method: "GET", path: "/listings/" + lid, token: bobT})
	h.do(304, call{method: "GET", path: "/listings/" + lid, token: bobT, hdr: m2("If-None-Match", letag)})
	h.do(404, call{method: "GET", path: "/listings/nope", token: bobT})
	h.do(200, call{method: "PATCH", path: "/listings/" + lid, token: aliceT, hdr: m2("If-Match", letag), body: m{"title": "bike fix"}})
	h.do(428, call{method: "PATCH", path: "/listings/" + lid, token: aliceT, body: m{"title": "x"}, skipRequest: true})
	h.do(412, call{method: "PATCH", path: "/listings/" + lid, token: aliceT, hdr: m2("If-Match", `"stale"`), body: m{"title": "x"}})

	// Transactions.
	rec = h.do(201, call{method: "POST", path: "/transactions", token: aliceT, idem: true,
		body: m{"counterparty_member_id": bobM.ID, "direction": "pay", "amount_minor": "1000", "description": "repair"}})
	tx := get[idOnly](t, rec).ID
	h.do(400, call{method: "POST", path: "/transactions", token: aliceT, skipRequest: true,
		body: m{"counterparty_member_id": bobM.ID, "direction": "pay", "amount_minor": "1"}})
	h.do(422, call{method: "POST", path: "/transactions", token: aliceT, idem: true, skipRequest: true,
		body: m{"counterparty_member_id": bobM.ID, "direction": "pay", "amount_minor": "999999999999"}})
	h.do(200, call{method: "GET", path: "/transactions", token: aliceT})
	h.do(200, call{method: "GET", path: "/transactions?status=proposed&role=payer", token: aliceT})
	h.do(400, call{method: "GET", path: "/transactions?status=bogus", token: aliceT, skipRequest: true})
	h.do(200, call{method: "GET", path: "/transactions/" + tx, token: bobT})
	h.do(404, call{method: "GET", path: "/transactions/" + tx, token: outsiderT})
	h.do(200, call{method: "GET", path: "/transactions/" + tx + "/events", token: bobT})
	h.do(403, call{method: "POST", path: "/transactions/" + tx + "/accept", token: aliceT, idem: true})
	h.do(200, call{method: "POST", path: "/transactions/" + tx + "/accept", token: bobT, idem: true})
	h.do(409, call{method: "POST", path: "/transactions/" + tx + "/accept", token: bobT, idem: true})
	h.do(403, call{method: "POST", path: "/transactions", token: roT, idem: true,
		body: m{"counterparty_member_id": bobM.ID, "direction": "pay", "amount_minor": "1"}})
	rec = h.do(201, call{method: "POST", path: "/transactions", token: bobT, idem: true,
		body: m{"counterparty_member_id": aliceM.ID, "direction": "pay", "amount_minor": "100"}})
	t2 := get[idOnly](t, rec).ID
	h.do(200, call{method: "POST", path: "/transactions/" + t2 + "/decline", token: aliceT, idem: true, body: m{"reason": "no"}})
	rec = h.do(201, call{method: "POST", path: "/transactions", token: aliceT, idem: true,
		body: m{"counterparty_member_id": bobM.ID, "direction": "pay", "amount_minor": "100"}})
	t3 := get[idOnly](t, rec).ID
	h.do(200, call{method: "POST", path: "/transactions/" + t3 + "/cancel", token: aliceT, idem: true})

	// Disputes.
	rec = h.do(201, call{method: "POST", path: "/disputes", token: aliceT, idem: true,
		body: m{"transaction_id": tx, "category": "not_delivered", "statement": "nothing came"}})
	d := get[idOnly](t, rec).ID
	h.do(409, call{method: "POST", path: "/disputes", token: aliceT, idem: true,
		body: m{"transaction_id": tx, "category": "other", "statement": "again"}})
	h.do(200, call{method: "GET", path: "/disputes", token: aliceT})
	h.do(200, call{method: "GET", path: "/disputes?status=open", token: adminT})
	h.do(200, call{method: "GET", path: "/disputes/" + d, token: bobT})
	h.do(404, call{method: "GET", path: "/disputes/" + d, token: outsiderT})
	h.do(200, call{method: "POST", path: "/disputes/" + d + "/review", token: adminT})
	h.do(501, call{method: "POST", path: "/disputes/" + d + "/resolve", token: adminT, idem: true, body: m{"outcome": "adjustment", "reason": "x"}})
	h.do(403, call{method: "POST", path: "/disputes/" + d + "/response", token: aliceT, idem: true, body: m{"response": "mine"}})
	h.do(422, call{method: "POST", path: "/disputes/" + d + "/response", token: bobT, idem: true, body: m{"response": " "}, skipRequest: true})
	h.do(201, call{method: "POST", path: "/disputes/" + d + "/response", token: bobT, idem: true, body: m{"response": "I handed it over"}})
	h.do(409, call{method: "POST", path: "/disputes/" + d + "/response", token: bobT, idem: true, body: m{"response": "again"}})
	h.do(409, call{method: "POST", path: "/disputes/" + d + "/re-review", token: aliceT, idem: true, body: m{"reason": "too early"}})
	h.do(200, call{method: "POST", path: "/disputes/" + d + "/resolve", token: adminT, idem: true,
		body: m{"outcome": "dismissed", "reason": "evidence shows delivery"}})
	// Re-review: requested by a party, decided by a reviewer other than the original one.
	rev, revM := env.Member(admin, net, "rev")
	h.do(200, call{method: "PUT", path: "/admin/members/" + revM.ID + "/roles/reviewer", token: adminT})
	revT := h.token(rev, rw...)
	h.do(404, call{method: "POST", path: "/disputes/" + d + "/re-review", token: revT, idem: true, body: m{"reason": "not mine"}})
	h.do(201, call{method: "POST", path: "/disputes/" + d + "/re-review", token: aliceT, idem: true, body: m{"reason": "I have photos"}})
	h.do(409, call{method: "POST", path: "/disputes/" + d + "/re-review", token: aliceT, idem: true, body: m{"reason": "again"}})
	h.do(403, call{method: "POST", path: "/disputes/" + d + "/re-review/decision", token: adminT, idem: true,
		body: m{"outcome": "confirmed", "reason": "same reviewer"}})
	h.do(200, call{method: "POST", path: "/disputes/" + d + "/re-review/decision", token: revT, idem: true,
		body: m{"outcome": "confirmed", "reason": "decision stands"}})
	tx2 := env.Trade(net.ID, alice, bob, aliceM, bobM, 700)
	rec = h.do(201, call{method: "POST", path: "/disputes", token: aliceT, idem: true,
		body: m{"transaction_id": tx2.ID, "category": "not_delivered", "statement": "nothing"}})
	d2 := get[idOnly](t, rec).ID
	h.do(200, call{method: "POST", path: "/disputes/" + d2 + "/withdraw", token: aliceT})
	rec = h.do(201, call{method: "POST", path: "/disputes", token: aliceT, idem: true,
		body: m{"transaction_id": tx2.ID, "category": "not_delivered", "statement": "nothing again"}})
	d3 := get[idOnly](t, rec).ID
	h.do(409, call{method: "POST", path: "/disputes/" + d3 + "/retry-posting", token: adminT})

	// Admin.
	rec = h.do(200, call{method: "GET", path: "/admin/policy", token: adminT})
	petag := rec.Header().Get("ETag")
	h.do(403, call{method: "GET", path: "/admin/policy", token: aliceT})
	h.do(200, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", petag),
		body: m{"reason": "raise limit", "default_credit_limit_minor": "20000",
			"response_period_days": 5, "re_review_window_days": 10, "limit_warning_percent": 90}})
	h.do(412, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", petag), body: m{"reason": "again"}})
	h.do(200, call{method: "GET", path: "/admin/members", token: adminT})
	h.do(400, call{method: "GET", path: "/admin/members?status=weird", token: adminT, skipRequest: true})
	h.do(200, call{method: "GET", path: "/admin/accounts/" + bobM.ID, token: adminT})
	rec = h.do(200, call{method: "GET", path: "/admin/accounts/" + aliceM.ID, token: adminT})
	h.do(200, call{method: "PUT", path: "/admin/accounts/" + aliceM.ID + "/credit-limit", token: adminT,
		hdr: m2("If-Match", rec.Header().Get("ETag")), body: m{"credit_limit_minor": "500", "reason": "risk review"}})
	h.do(428, call{method: "PUT", path: "/admin/accounts/" + aliceM.ID + "/credit-limit", token: adminT, skipRequest: true,
		body: m{"credit_limit_minor": "500", "reason": "x"}})
	h.do(201, call{method: "POST", path: "/admin/adjustments", token: adminT, idem: true,
		body: m{"member_id": bobM.ID, "direction": "credit", "amount_minor": "250", "reason": "gift"}})
	h.do(422, call{method: "POST", path: "/admin/adjustments", token: adminT, idem: true, skipRequest: true,
		body: m{"member_id": bobM.ID, "direction": "sideways", "amount_minor": "250", "reason": "gift"}})
	h.do(200, call{method: "POST", path: "/admin/members/" + bobM.ID + "/suspend", token: adminT, idem: true, body: m{"reason": "review"}})
	h.do(200, call{method: "POST", path: "/admin/members/" + bobM.ID + "/reinstate", token: adminT, idem: true, body: m{"reason": "ok"}})
	h.do(200, call{method: "PUT", path: "/admin/members/" + bobM.ID + "/roles/reviewer", token: adminT})
	h.do(200, call{method: "DELETE", path: "/admin/members/" + bobM.ID + "/roles/reviewer", token: adminT})
	rec = h.do(201, call{method: "POST", path: "/admin/invitations", token: adminT, idem: true, body: m{"ttl_seconds": 3600}})
	inv := get[idOnly](t, rec).Token
	carol := env.Person("carol")
	rec = h.do(201, call{method: "POST", path: "/invitations/" + inv + "/accept", token: h.token(carol, rw...), idem: true, body: m{"display_name": "Carol"}})
	carolM := get[idOnly](t, rec).MemberID
	h.do(404, call{method: "POST", path: "/invitations/inv_nope/accept", token: h.token(env.Person("dan"), rw...), idem: true, body: m{"display_name": "Dan"}})
	h.do(409, call{method: "POST", path: "/admin/members/" + carolM + "/approve", token: adminT, idem: true, body: m{"reason": "ok"}})
	h.do(409, call{method: "POST", path: "/admin/members/" + carolM + "/deny", token: adminT, idem: true, body: m{"reason": "no"}})
	h.do(200, call{method: "POST", path: "/admin/members/" + carolM + "/remove", token: adminT, idem: true, body: m{"reason": "left"}})
	h.do(200, call{method: "GET", path: "/admin/audit-events?limit=2", token: adminT})
	h.do(400, call{method: "GET", path: "/admin/audit-events?from=yesterday", token: adminT, skipRequest: true})
	h.do(200, call{method: "GET", path: "/audit-events?limit=2", token: h.token(bob, rw...)})
	h.do(400, call{method: "GET", path: "/audit-events?actor_id=x", token: adminT, skipRequest: true})
	rec = h.do(201, call{method: "POST", path: "/admin/consistency-checks", token: adminT, idem: true})
	h.do(200, call{method: "GET", path: "/admin/consistency-checks/" + get[idOnly](t, rec).ID, token: adminT})
	h.do(404, call{method: "GET", path: "/admin/consistency-checks/nope", token: adminT})
	h.do(400, call{method: "POST", path: "/admin/announcements", token: adminT, body: m{"title": "Closed", "body": "x"}, skipRequest: true})
	rec = h.do(201, call{method: "POST", path: "/admin/announcements", token: adminT, idem: true,
		body: m{"title": "Market day", "body": "Saturday at the hall"}})
	ann := get[idOnly](t, rec).ID
	h.do(200, call{method: "GET", path: "/admin/announcements", token: adminT})
	h.do(403, call{method: "GET", path: "/admin/announcements", token: aliceT})
	h.do(204, call{method: "DELETE", path: "/admin/announcements/" + ann, token: adminT})
	h.do(404, call{method: "DELETE", path: "/admin/announcements/" + ann, token: adminT})

	// Reports and notifications.
	h.do(422, call{method: "POST", path: "/reports", token: aliceT, idem: true, skipRequest: true,
		body: m{"target_type": "listing", "target_id": lid, "category": "scam"}})
	rec = h.do(201, call{method: "POST", path: "/reports", token: bobT, idem: true,
		body: m{"target_type": "listing", "target_id": lid, "category": "scam", "description": "looks fake"}})
	rep := get[idOnly](t, rec).ID
	h.do(409, call{method: "POST", path: "/reports", token: bobT, idem: true,
		body: m{"target_type": "listing", "target_id": lid, "category": "spam"}})
	h.do(200, call{method: "GET", path: "/admin/reports", token: adminT})
	h.do(200, call{method: "GET", path: "/admin/reports?status=open", token: adminT})
	h.do(403, call{method: "GET", path: "/admin/reports", token: aliceT})
	h.do(403, call{method: "POST", path: "/admin/reports/" + rep + "/handle", token: aliceT, idem: true})
	h.do(200, call{method: "POST", path: "/admin/reports/" + rep + "/handle", token: adminT, idem: true, body: m{"note": "checked"}})
	h.do(409, call{method: "POST", path: "/admin/reports/" + rep + "/handle", token: adminT, idem: true})
	h.do(200, call{method: "GET", path: "/notifications", token: aliceT})
	h.do(200, call{method: "GET", path: "/notifications?unread=true", token: bobT})
	h.do(200, call{method: "GET", path: "/notifications/unread-count", token: bobT})
	h.do(200, call{method: "POST", path: "/notifications/read", token: bobT, body: m{"all": true}})
	h.do(200, call{method: "POST", path: "/notifications/read", token: bobT, body: m{"ids": []string{"nope"}}})
	h.do(422, call{method: "POST", path: "/notifications/read", token: bobT, body: m{}, skipRequest: true})

	// Reversal (admin) on a fresh trade.
	tx3 := env.Trade(net.ID, bob, alice, bobM, aliceM, 300)
	h.do(201, call{method: "POST", path: "/admin/adjustments", token: adminT, idem: true,
		body: m{"member_id": aliceM.ID, "direction": "credit", "amount_minor": "5000", "reason": "fund reversal"}})
	h.do(201, call{method: "POST", path: "/admin/transactions/" + tx3.ID + "/reversal", token: adminT, idem: true, body: m{"reason": "agreed undo", "amount_minor": "100"}})
	h.do(422, call{method: "POST", path: "/admin/transactions/" + tx3.ID + "/reversal", token: adminT, idem: true, body: m{"reason": "too much", "amount_minor": "999999"}})

	// Exports.
	rec = h.do(201, call{method: "POST", path: "/exports", token: aliceT, body: m{"scope": "mine", "format": "csv"}})
	ex := get[struct {
		ID          string `json:"id"`
		ArtefactURL string `json:"artefact_url"`
	}](t, rec)
	h.do(200, call{method: "GET", path: "/exports/" + ex.ID, token: aliceT})
	h.do(404, call{method: "GET", path: "/exports/" + ex.ID, token: bobT})
	h.do(200, call{method: "GET", path: strings.TrimPrefix(ex.ArtefactURL, "/api/v1"), token: aliceT})
	h.do(422, call{method: "POST", path: "/exports", token: aliceT, body: m{"scope": "everything"}, skipRequest: true})
	h.do(403, call{method: "POST", path: "/exports", token: aliceT, body: m{"scope": "audit"}})

	// Tokens (create/password need a browser session and CSRF token).
	sess, _, err := env.Svc.Login(context.Background(), "alice", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	cookie := m2("Cookie", (&http.Cookie{Name: "circula_session", Value: sess}).String(), "X-CSRF-Token", httpx.CSRF{Key: csrfKey}.Token(sess))
	rec = h.do(201, call{method: "POST", path: "/tokens", hdr: cookie, body: m{"name": "ci", "scopes": []string{"read"}}})
	tokID := get[idOnly](t, rec).ID
	h.do(403, call{method: "POST", path: "/tokens", token: aliceT, body: m{"name": "x", "scopes": []string{"read"}}})
	h.do(200, call{method: "GET", path: "/tokens", token: aliceT})
	h.do(204, call{method: "DELETE", path: "/tokens/" + tokID, token: aliceT})
	h.do(404, call{method: "DELETE", path: "/tokens/" + tokID, token: aliceT})
	h.do(403, call{method: "POST", path: "/me/password", token: aliceT, body: m{"current_password": "a", "new_password": "bbbbbbbbbbbb"}})
	h.do(422, call{method: "POST", path: "/me/password", hdr: cookie, body: m{"current_password": "wrong wrong wrong", "new_password": "another long password"}})
	h.do(204, call{method: "POST", path: "/me/password", hdr: cookie, body: m{"current_password": "correct horse battery", "new_password": "another long password"}})

	// Departure last (needs no open disputes).
	h.do(200, call{method: "POST", path: "/disputes/" + d3 + "/resolve", token: adminT, idem: true, body: m{"outcome": "dismissed", "reason": "no evidence", "waive_response": true, "waive_reason": "other party did not engage"}})
	h.do(200, call{method: "POST", path: "/me/membership/departure", token: bobT, idem: true, body: m{"reason": "moving"}})
	h.do(409, call{method: "POST", path: "/me/membership/departure", token: adminT, idem: true})

	// Dual control (SEC-011): above the threshold an adjustment waits for a second administrator.
	adm2, adm2M := env.Member(admin, net, "admin2")
	if err := env.Svc.AssignRole(env.Ctx, admin, net.ID, adm2M.ID, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	adm2T := h.token(adm2, rw...)
	rec = h.do(200, call{method: "GET", path: "/admin/policy", token: adminT})
	h.do(200, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", rec.Header().Get("ETag")),
		body: m{"reason": "dual control", "dual_control_threshold_minor": "1000"}})
	rec = h.do(202, call{method: "POST", path: "/admin/adjustments", token: adminT, idem: true,
		body: m{"member_id": aliceM.ID, "direction": "credit", "amount_minor": "5000", "reason": "large"}})
	pa := get[idOnly](t, rec).ID
	rec = h.do(202, call{method: "POST", path: "/admin/adjustments", token: adminT, idem: true,
		body: m{"member_id": aliceM.ID, "direction": "credit", "amount_minor": "6000", "reason": "large two"}})
	pb := get[idOnly](t, rec).ID
	rec = h.do(202, call{method: "POST", path: "/admin/adjustments", token: adminT, idem: true,
		body: m{"member_id": aliceM.ID, "direction": "credit", "amount_minor": "7000", "reason": "large three"}})
	pc := get[idOnly](t, rec).ID
	h.do(200, call{method: "GET", path: "/admin/adjustments?status=pending", token: adm2T})
	h.do(400, call{method: "GET", path: "/admin/adjustments?status=weird", token: adm2T, skipRequest: true})
	h.do(200, call{method: "GET", path: "/admin/adjustments/" + pa, token: adm2T})
	h.do(403, call{method: "POST", path: "/admin/adjustments/" + pa + "/approve", token: adminT})
	h.do(200, call{method: "POST", path: "/admin/adjustments/" + pa + "/approve", token: adm2T})
	h.do(409, call{method: "POST", path: "/admin/adjustments/" + pa + "/reject", token: adm2T})
	h.do(200, call{method: "POST", path: "/admin/adjustments/" + pb + "/reject", token: adm2T})
	h.do(200, call{method: "POST", path: "/admin/adjustments/" + pc + "/cancel", token: adminT})
	h.do(404, call{method: "GET", path: "/admin/adjustments/nope", token: adm2T})

	// Governance: notice before an adverse policy change (GOV-003), change log (GOV-002), terms (GOV-009).
	rec = h.do(200, call{method: "GET", path: "/admin/policy", token: adminT})
	petag = rec.Header().Get("ETag")
	h.do(422, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", petag), skipRequest: true,
		body: m{"reason": "lower", "default_credit_limit_minor": "100", "notice_days": 13}})
	rec = h.do(200, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", petag),
		body: m{"reason": "lower", "default_credit_limit_minor": "100"}})
	if got := get[struct {
		Pending *struct {
			NoticeDays int `json:"notice_days"`
		} `json:"pending"`
	}](t, rec); got.Pending == nil || got.Pending.NoticeDays != 14 {
		t.Errorf("adverse change was not scheduled with 14 days' notice: %s", rec.Body.String())
	}
	h.do(409, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", petag),
		body: m{"reason": "more", "dispute_window_days": 10}})
	rec = h.do(200, call{method: "GET", path: "/admin/policy", token: adminT})
	if !strings.Contains(rec.Body.String(), `"pending":{`) {
		t.Errorf("scheduled change not shown: %s", rec.Body.String())
	}
	rec = h.do(200, call{method: "GET", path: "/network/policy-log", token: aliceT})
	if !strings.Contains(rec.Body.String(), `"pending":true`) {
		t.Errorf("policy log: %s", rec.Body.String())
	}
	h.do(200, call{method: "PATCH", path: "/admin/policy", token: adminT, hdr: m2("If-Match", petag), body: m{"reason": "cancel"}})

	h.do(200, call{method: "GET", path: "/network/terms", token: aliceT})
	h.do(403, call{method: "POST", path: "/admin/terms", token: aliceT, idem: true, body: m{"body": "x"}})
	h.do(422, call{method: "POST", path: "/admin/terms", token: adminT, idem: true, skipRequest: true, body: m{"body": ""}})
	h.do(201, call{method: "POST", path: "/admin/terms", token: adminT, idem: true, body: m{"body": "Be kind.\n\nPay promptly."}})
	rec = h.do(200, call{method: "GET", path: "/network/terms", token: aliceT})
	if !strings.Contains(rec.Body.String(), `"needs_acceptance":true`) {
		t.Errorf("terms state: %s", rec.Body.String())
	}
	rec = h.do(409, call{method: "POST", path: "/transactions", token: aliceT, idem: true,
		body: m{"counterparty_member_id": adm2M.ID, "direction": "pay", "amount_minor": "10"}})
	if !strings.Contains(rec.Body.String(), "terms-acceptance-required") {
		t.Errorf("want terms-acceptance-required: %s", rec.Body.String())
	}
	h.do(422, call{method: "POST", path: "/me/terms/accept", token: aliceT, idem: true, body: m{"version": 9}})
	h.do(200, call{method: "POST", path: "/me/terms/accept", token: aliceT, idem: true, body: m{"version": 1}})
	rec = h.do(201, call{method: "POST", path: "/transactions", token: aliceT, idem: true,
		body: m{"counterparty_member_id": adm2M.ID, "direction": "pay", "amount_minor": "10"}})
	openTx := get[idOnly](t, rec).ID
	rec = h.do(201, call{method: "POST", path: "/admin/invitations", token: adminT, idem: true, body: m{"ttl_seconds": 3600}})
	inv2 := get[idOnly](t, rec).Token
	erin := env.Person("erin")
	h.do(422, call{method: "POST", path: "/invitations/" + inv2 + "/accept", token: h.token(erin, rw...), idem: true, body: m{"display_name": "Erin"}})
	h.do(201, call{method: "POST", path: "/invitations/" + inv2 + "/accept", token: h.token(erin, rw...), idem: true,
		body: m{"display_name": "Erin", "terms_version": 1}})
	// Suspension and closure (GOV-023, FR-068).
	h.do(403, call{method: "POST", path: "/admin/network/suspend", token: aliceT, body: m{"reason": "x"}})
	h.do(422, call{method: "POST", path: "/admin/network/suspend", token: adminT, body: m{"reason": ""}, skipRequest: true})
	h.do(204, call{method: "POST", path: "/admin/network/suspend", token: adminT, body: m{"reason": "security review"}})
	h.do(409, call{method: "POST", path: "/admin/network/suspend", token: adminT, body: m{"reason": "again"}})
	h.do(200, call{method: "GET", path: "/accounts/me", token: aliceT})
	h.do(204, call{method: "POST", path: "/admin/network/resume", token: adminT, body: m{"reason": "review done"}})
	h.do(409, call{method: "POST", path: "/admin/network/resume", token: adminT, body: m{"reason": "again"}})
	h.do(200, call{method: "GET", path: "/admin/network/closure", token: adminT})
	h.do(403, call{method: "GET", path: "/admin/network/closure", token: aliceT})
	h.do(422, call{method: "POST", path: "/admin/network/close", token: adminT, body: m{"reason": ""}, skipRequest: true})
	// An open proposal blocks closure (FR-068); withdraw it, then close.
	h.do(409, call{method: "POST", path: "/admin/network/close", token: adminT, body: m{"reason": "winding down"}})
	h.do(200, call{method: "POST", path: "/transactions/" + openTx + "/cancel", token: aliceT, idem: true})
	h.do(204, call{method: "POST", path: "/admin/network/close", token: adminT, body: m{"reason": "winding down"}})

	// Generic problems.
	h.do(401, call{method: "GET", path: "/accounts/me", token: "cir_bogus"})
	h.do(400, call{method: "GET", path: "/accounts/me?access_token=x", token: aliceT, skipRequest: true})

	// Coverage: every documented operation was exercised.
	var missing []string
	for p, item := range h.doc.Paths.Map() {
		for method := range item.Operations() {
			if !h.ops[method+" "+p] {
				missing = append(missing, method+" "+p)
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("documented operations never exercised:\n  %s", strings.Join(missing, "\n  "))
	}
	t.Logf("operations exercised: %d; distinct (operation,status) pairs: %d", len(h.ops), len(h.seen))
}

func m2(kv ...string) map[string]string {
	out := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}
