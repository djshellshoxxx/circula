package main

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

var _ service.AuthenticationObserver = (*metrics)(nil)

// OPS-014: real credential operations emit metrics without an HTTP response;
// a throttled call adds no failed-proof count and raw labels cannot escape.
func TestServiceAuthenticationObserverWiredToMetrics(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	wrongPassword := e.Key()
	m := newMetrics(time.Now())
	e.Svc.SetObserver(m)
	for range 5 {
		_, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", wrongPassword)
		if domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("expected a credential refusal, got %v", err)
		}
	}
	if _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", wrongPassword); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Fatalf("expected a throttle refusal, got %v", err)
	}
	m.AuthenticationFailed("private-credential-canary")
	m.AuthenticationThrottled("private-identity-canary")
	var b strings.Builder
	m.write(&b, dynamicMetrics{now: time.Now()})
	for _, want := range []string{
		`circula_authentication_failures_total{reason="password"} 5`,
		`circula_authentication_throttled_total{reason="password"} 1`,
		`circula_authentication_failures_total{reason="other"} 1`,
		`circula_authentication_throttled_total{reason="other"} 1`,
		`circula_authentication_failures_total{reason="mfa"} 0`,
		"circula_http_auth_failures_total 0",
		"circula_http_rate_limited_total 0",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing credential metric %q", want)
		}
	}
	for _, absent := range []string{"private-credential-canary", "private-identity-canary", "ana@example.org", wrongPassword} {
		if strings.Contains(b.String(), absent) {
			t.Fatal("credential or identity text reached metric output")
		}
	}
}
