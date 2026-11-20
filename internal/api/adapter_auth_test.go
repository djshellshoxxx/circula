package api_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func TestAdapterAdminReadsRequireCurrentMFA(t *testing.T) {
	env := testutil.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
	admin := env.Person("admin")
	codes := enrollReplayMFA(t, env, admin)
	net, member := env.Network(admin, "admin-read-mfa", nil)
	w := &world{t: t, env: env, net: net, admin: admin, adminM: member, h: api.NewHandler(env.Svc)}
	t.Cleanup(func() { api.CloseHandler(w.h) })
	w.adminT = w.token(admin, domain.ScopeRead, domain.ScopeWrite)
	readOnly := w.token(admin, domain.ScopeRead)
	writeOnlyToken := w.token(admin, domain.ScopeWrite)
	check := w.post(w.adminT, "/admin/consistency-checks", nil, w.key())
	wantStatus(t, check, 201)
	paths := []string{"/api/v1/admin/policy", "/api/v1/admin/audit-events", check.Header().Get("Location")}
	env.Svc.FreezeNetwork(net.ID, "Fixture ledger investigation")
	for _, path := range paths {
		wantStatus(t, w.do(req{method: "GET", path: path, token: readOnly}), 200)
		wantProblem(t, w.do(req{method: "GET", path: path, token: writeOnlyToken}), 403, "forbidden")
	}
	writeOnly := admin
	writeOnly.Scopes = []domain.Scope{domain.ScopeWrite}
	if err := env.Svc.AuthorizeAdminRead(env.Ctx, writeOnly, net.ID); domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("admin read accepted a write-only actor: %v", err)
	}
	if _, err := env.Svc.GetNetworkAccess(env.Ctx, writeOnly, net.ID); domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("network access accepted a write-only actor: %v", err)
	}
	if err := env.Svc.DisableMFA(env.Ctx, admin, replayTestPassword, codes[0]); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		wantProblem(t, w.do(req{method: "GET", path: path, token: readOnly}), 403, "forbidden")
	}
	access := testutil.Must(env.Svc.GetNetworkAccess(env.Ctx, admin, net.ID))
	if access.CanAdmin || access.CanReview || !access.Member.HasRole(domain.RoleAdmin) {
		t.Fatal("missing MFA must remove privileged capabilities without rewriting assigned roles")
	}
	enrollReplayMFA(t, env, admin)
	for _, path := range paths {
		wantStatus(t, w.do(req{method: "GET", path: path, token: readOnly}), 200)
	}
}

func TestAdapterAdminReadsUseLiveBreakGlass(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	check := w.post(w.adminT, "/admin/consistency-checks", nil, w.key())
	wantStatus(t, check, 201)
	paths := []string{"/api/v1/admin/policy", "/api/v1/admin/audit-events", check.Header().Get("Location")}
	readOnly := w.token(w.alice, domain.ScopeRead)
	for _, path := range paths {
		wantProblem(t, w.do(req{method: "GET", path: path, token: readOnly}), 403, "forbidden")
	}
	sys := domain.SystemActor("adapter-test")
	testutil.Must(w.env.Svc.OpenBreakGlass(w.env.Ctx, sys, service.OpenBreakGlassInput{
		Network: w.net.ID, Operator: "Test operator", Reason: "Read authorization test", TTL: time.Minute}))
	testutil.Must(w.env.Svc.GrantBreakGlassAdmin(w.env.Ctx, sys, service.GrantBreakGlassInput{
		Network: w.net.ID, Login: "alice", Reason: "Temporary recovery access"}))
	w.env.Svc.FreezeNetwork(w.net.ID, "Fixture ledger investigation")
	for _, path := range paths {
		wantStatus(t, w.do(req{method: "GET", path: path, token: readOnly}), 200)
	}
	access := testutil.Must(w.env.Svc.GetNetworkAccess(w.env.Ctx, w.alice, w.net.ID))
	if !access.CanAdmin || !access.CanReview || !access.Member.HasRole(domain.RoleAdmin) {
		t.Fatal("live grant was not reflected in effective access")
	}
	w.env.Clock.Advance(time.Minute)
	for _, path := range paths {
		wantProblem(t, w.do(req{method: "GET", path: path, token: readOnly}), 403, "forbidden")
	}
	access = testutil.Must(w.env.Svc.GetNetworkAccess(w.env.Ctx, w.alice, w.net.ID))
	if access.CanAdmin || access.CanReview || access.Member.HasRole(domain.RoleAdmin) {
		t.Fatal("expired grant survived in effective access without the cleanup job")
	}
}
