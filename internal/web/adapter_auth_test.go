package web

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func TestWebAdminReadsAndNavigationRequireCurrentMFA(t *testing.T) {
	env := testutil.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
	admin := env.Person("admin@example.org")
	enrollment := testutil.Must(env.Svc.BeginMFAEnrolment(env.Ctx, admin, testPW))
	codes := testutil.Must(env.Svc.ConfirmMFAEnrolment(env.Ctx, admin, testPW,
		webTOTP(t, enrollment.Secret, env.Clock.Now())))
	env.Clock.Advance(31 * time.Second)
	net, member := env.Network(admin, "admin-read-mfa", nil)
	w := &world{env: env, net: net, admin: admin, adminM: member}
	w.srv = httptest.NewServer(NewHandlerWith(env.Svc, Options{
		CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: env.Clock}))
	t.Cleanup(w.srv.Close)
	b := w.browser(t)
	step := b.login("admin@example.org")
	loggedIn := b.post("/login/mfa", step, url.Values{"challenge": {challengeOf(t, step)},
		"code": {webTOTP(t, enrollment.Secret, env.Clock.Now())}})
	if loggedIn.Status != 200 || loggedIn.URL == "/login/mfa" {
		t.Fatalf("MFA login failed: %d %s", loggedIn.Status, loggedIn.URL)
	}
	env.Svc.FreezeNetwork(net.ID, "Fixture ledger investigation")
	paths := []string{"/admin", "/admin/policy", "/admin/exports", "/admin/invitations"}
	for _, path := range paths {
		if p := b.get(w.base() + path); p.Status != 200 {
			t.Fatalf("authorized frozen-network read %s returned %d", path, p.Status)
		}
	}
	b.mustContain(b.get(w.base()+"/"), "Network admin", "Review queue")
	if err := env.Svc.DisableMFA(env.Ctx, admin, testPW, codes[0]); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if p := b.get(w.base() + path); p.Status != 403 {
			t.Errorf("missing MFA privileged read %s returned %d", path, p.Status)
		}
	}
	home := b.get(w.base() + "/")
	if home.Status != 200 || strings.Contains(home.Body, "Network admin") || strings.Contains(home.Body, "Review queue") {
		t.Fatalf("member home exposed privileged navigation without MFA: %d %s", home.Status, snippet(home.Body))
	}
}

func TestWebAdminReadsAndNavigationUseLiveBreakGlass(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	if p := b.get(w.base() + "/admin"); p.Status != 403 {
		t.Fatalf("ordinary member reached admin page: %d", p.Status)
	}
	sys := domain.SystemActor("adapter-test")
	testutil.Must(w.env.Svc.OpenBreakGlass(w.env.Ctx, sys, service.OpenBreakGlassInput{
		Network: w.net.ID, Operator: "Test operator", Reason: "Read authorization test", TTL: time.Minute}))
	testutil.Must(w.env.Svc.GrantBreakGlassAdmin(w.env.Ctx, sys, service.GrantBreakGlassInput{
		Network: w.net.ID, Login: "alice@example.org", Reason: "Temporary recovery access"}))
	w.env.Svc.FreezeNetwork(w.net.ID, "Fixture ledger investigation")
	for _, path := range []string{"/admin", "/admin/policy", "/admin/invitations"} {
		if p := b.get(w.base() + path); p.Status != 200 {
			t.Fatalf("live break-glass read %s returned %d", path, p.Status)
		}
	}
	b.mustContain(b.get(w.base()+"/"), "Network admin", "Review queue")
	w.env.Clock.Advance(time.Minute)
	if p := b.get(w.base() + "/admin"); p.Status != 403 {
		t.Fatalf("expired break-glass read returned %d", p.Status)
	}
	home := b.get(w.base() + "/")
	if home.Status != 200 || strings.Contains(home.Body, "Network admin") || strings.Contains(home.Body, "Review queue") {
		t.Fatalf("expired grant left privileged navigation: %d %s", home.Status, snippet(home.Body))
	}
}
