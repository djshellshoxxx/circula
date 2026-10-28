package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestUnauthenticatedIs401WithChallenge(t *testing.T) {
	w := newWorld(t)
	rec := w.get("", "/network")
	wantProblem(t, rec, 401, "unauthenticated")
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("missing WWW-Authenticate")
	}
	wantProblem(t, w.get("cir_bogus", "/network"), 401, "unauthenticated")
	wantProblem(t, w.get("not-a-token", "/network"), 401, "unauthenticated")
}

func TestSecurityHeadersAndRequestID(t *testing.T) {
	w := newWorld(t)
	rec := w.get(w.aliceT, "/network")
	wantStatus(t, rec, 200)
	for k, v := range map[string]string{"X-Content-Type-Options": "nosniff", "Cache-Control": "no-store",
		"X-Frame-Options": "DENY", "Referrer-Policy": "no-referrer"} {
		if rec.Header().Get(k) != v {
			t.Errorf("%s = %q", k, rec.Header().Get(k))
		}
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") || rec.Header().Get("X-Request-Id") == "" {
		t.Errorf("headers: %v", rec.Header())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS must not be enabled")
	}
	// A well-formed inbound id is echoed; a hostile one is replaced.
	rec = w.do(req{method: "GET", path: "/api/v1/network", token: w.aliceT, hdr: map[string]string{"X-Request-Id": "trace-12345678"}})
	if rec.Header().Get("X-Request-Id") != "trace-12345678" {
		t.Error("request id not echoed")
	}
	hostileID := "bad id\x01"
	rec = w.do(req{method: "GET", path: "/api/v1/network", token: w.aliceT, hdr: map[string]string{"X-Request-Id": hostileID}})
	returnedID := rec.Header().Get("X-Request-Id")
	if returnedID == hostileID {
		t.Error("hostile request id was echoed")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`).MatchString(returnedID) {
		t.Errorf("replacement request id is empty or malformed: %q", returnedID)
	}
}

func TestUnknownRouteAndMethod(t *testing.T) {
	w := newWorld(t)
	wantProblem(t, w.get(w.aliceT, "/nope"), 404, "not-found")
	rec := w.do(req{method: "DELETE", path: "/api/v1/network", token: w.aliceT})
	wantProblem(t, rec, 405, "method-not-allowed")
	if rec.Header().Get("Allow") != "GET" {
		t.Errorf("Allow = %q", rec.Header().Get("Allow"))
	}
}

func TestCredentialsInQueryRejected(t *testing.T) {
	w := newWorld(t)
	for _, q := range []string{"?token=abc", "?access_token=x", "?foo=" + w.aliceT, "?session=1"} {
		rec := w.do(req{method: "GET", path: "/api/v1/network" + q, token: w.aliceT})
		p := wantProblem(t, rec, 400, "credentials-in-query")
		if strings.Contains(rec.Body.String(), w.aliceT) || strings.Contains(p.Detail, "abc") {
			t.Fatal("credential echoed")
		}
	}
}

func TestCookieSessionNeedsCSRF(t *testing.T) {
	w := newWorld(t)
	sess, _, err := w.env.Svc.Login(context.Background(), "alice", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	ck := &http.Cookie{Name: "circula_session", Value: sess}
	do := func(method, path, csrf string, body any) *httptest.ResponseRecorder {
		rec := w.do(req{method: method, path: path, body: body, hdr: map[string]string{"Cookie": ck.String(), "X-CSRF-Token": csrf,
			"Idempotency-Key": w.key()}})
		return rec
	}
	rec := do("GET", "/api/v1/accounts/me", "", nil)
	wantStatus(t, rec, 200)
	if rec.Header().Get("X-CSRF-Token") != csrfFor(sess) {
		t.Fatal("csrf token not advertised")
	}
	body := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "100"}
	wantProblem(t, do("POST", "/api/v1/transactions", "", body), 403, "csrf-invalid")
	wantProblem(t, do("POST", "/api/v1/transactions", "wrong", body), 403, "csrf-invalid")
	wantStatus(t, do("POST", "/api/v1/transactions", csrfFor(sess), body), 201)
	// Bearer-authenticated unsafe requests are exempt.
	wantStatus(t, w.post(w.aliceT, "/transactions", body, w.key()), 201)
}

func TestTokenScopesAndRevocation(t *testing.T) {
	w := newWorld(t)
	ro := w.token(w.alice, domain.ScopeRead)
	wantStatus(t, w.get(ro, "/accounts/me"), 200)
	body := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "100"}
	wantProblem(t, w.post(ro, "/transactions", body, w.key()), 403, "forbidden")

	// Token management: create needs a session, not a token (API-011).
	wantProblem(t, w.post(w.aliceT, "/tokens", m{"name": "x", "scopes": []string{"read"}}, ""), 403, "forbidden")
	sess, _, _ := w.env.Svc.Login(context.Background(), "alice", "correct horse battery")
	hdr := map[string]string{"Cookie": (&http.Cookie{Name: "circula_session", Value: sess}).String(), "X-CSRF-Token": csrfFor(sess)}
	rec := w.do(req{method: "POST", path: "/api/v1/tokens", body: m{"name": "ci", "scopes": []string{"read"}}, hdr: hdr})
	wantStatus(t, rec, 201)
	created := decode[struct {
		ID, Token string
		ExpiresAt string `json:"expires_at"`
	}](t, rec)
	if !strings.HasPrefix(created.Token, "cir_") || created.ExpiresAt == "" {
		t.Fatalf("token = %+v", created)
	}
	wantStatus(t, w.get(created.Token, "/accounts/me"), 200)
	list := w.get(w.aliceT, "/tokens")
	wantStatus(t, list, 200)
	if strings.Contains(list.Body.String(), created.Token) || strings.Contains(list.Body.String(), "token_hash") {
		t.Fatal("token list leaks secrets")
	}
	wantProblem(t, w.do(req{method: "POST", path: "/api/v1/tokens", body: m{"name": "a", "scopes": []string{"admin"}}, hdr: hdr}), 501, "not-implemented")
	wantStatus(t, w.do(req{method: "DELETE", path: "/api/v1/tokens/" + created.ID, token: w.aliceT}), 204)
	wantProblem(t, w.get(created.Token, "/accounts/me"), 401, "unauthenticated")
}

func TestAuthorisation(t *testing.T) {
	w := newWorld(t)
	// Members cannot use admin routes.
	wantProblem(t, w.get(w.aliceT, "/admin/members"), 403, "forbidden")
	wantProblem(t, w.get(w.aliceT, "/admin/policy"), 403, "forbidden")
	wantProblem(t, w.get(w.aliceT, "/admin/audit-events"), 403, "forbidden")
	wantProblem(t, w.post(w.aliceT, "/admin/adjustments", m{"member_id": w.bobM.ID, "direction": "credit", "amount_minor": "5", "reason": "x"}, w.key()), 403, "forbidden")
	wantStatus(t, w.get(w.adminT, "/admin/members"), 200)

	// A person with no membership sees nothing: the network does not exist for them.
	outsider := w.env.Person("eve")
	et := w.token(outsider, domain.ScopeRead, domain.ScopeWrite)
	wantProblem(t, w.get(et, "/network"), 404, "not-found")
	wantProblem(t, w.get(et, "/accounts/me"), 404, "not-found")
	wantStatus(t, w.get(et, "/me"), 200)

	// Another network's transaction is indistinguishable from a missing one.
	// Another network's data is indistinguishable from a missing id (API-017).
	adm2 := w.env.Person("admin2")
	net2, m2 := w.env.Network(adm2, "other", nil)
	z, zm := w.env.Member(adm2, net2, "zed")
	tx := w.env.Trade(net2.ID, z, adm2, zm, m2, 100)
	zt := w.token(z, domain.ScopeRead, domain.ScopeWrite)
	wantStatus(t, w.get(zt, "/transactions/"+tx.ID), 200)
	wantProblem(t, w.get(w.aliceT, "/transactions/"+tx.ID), 404, "not-found")
	wantProblem(t, w.get(w.aliceT, "/transactions/does-not-exist"), 404, "not-found")
	wantProblem(t, w.get(w.aliceT, "/members/"+zm.ID), 404, "not-found")

	// A member of two networks must name one.
	_, _ = w.env.Member(adm2, net2, "alice2")
	multi := w.env.Person("multi")
	tok, _ := testutilInvite(t, w, adm2, net2.ID)
	if _, err := w.env.Svc.RedeemInvitation(context.Background(), multi, tok, "multi"); err != nil {
		t.Fatal(err)
	}
	tok2, _ := testutilInvite(t, w, w.admin, w.net.ID)
	if _, err := w.env.Svc.RedeemInvitation(context.Background(), multi, tok2, "multi"); err != nil {
		t.Fatal(err)
	}
	mt := w.token(multi, domain.ScopeRead, domain.ScopeWrite)
	wantProblem(t, w.get(mt, "/network"), 400, "network-required")
	rec := w.do(req{method: "GET", path: "/api/v1/network", token: mt, hdr: map[string]string{"X-Network": "other"}})
	wantStatus(t, rec, 200)
	wantProblem(t, w.do(req{method: "GET", path: "/api/v1/network", token: w.aliceT, hdr: map[string]string{"X-Network": "other"}}), 404, "not-found")
}
