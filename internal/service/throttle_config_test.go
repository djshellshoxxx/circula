package service_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func configuredThrottleEnv(t *testing.T) *tu.Env {
	t.Helper()
	return tu.NewEnvConfig(t, nil, func(c *service.Config) {
		c.AuthThrottleAttempts, c.AuthThrottleGlobalAttempts, c.AuthThrottleLock = 4, 8, 30*time.Minute
	})
}

// SEC-006 / OPS-008: configuration reaches real login counters, is still
// address-isolated, and honors the configured lock at its exact boundary.
func TestConfiguredAuthenticationThrottleLoginAndGlobal(t *testing.T) {
	for _, address := range []string{"", "203.0.113.8"} {
		t.Run("address="+address, func(t *testing.T) {
			e := configuredThrottleEnv(t)
			e.Person("ana@example.org")
			ctx := e.Ctx
			if address != "" {
				ctx = service.WithClientIP(ctx, address)
			}
			for i := 0; i < 4; i++ {
				_, err := e.Svc.LoginStep(ctx, "ana@example.org", bad)
				wantCode(t, err, domain.ErrUnauthenticated)
			}
			_, err := e.Svc.LoginStep(ctx, "ana@example.org", pw)
			wantCode(t, err, domain.ErrRateLimited)
			e.Clock.Advance(30*time.Minute - time.Millisecond)
			_, err = e.Svc.LoginStep(ctx, "ana@example.org", pw)
			wantCode(t, err, domain.ErrRateLimited)
			e.Clock.Advance(time.Millisecond)
			_, err = e.Svc.LoginStep(ctx, "ana@example.org", bad)
			wantCode(t, err, domain.ErrUnauthenticated)
			// One failure at expiry must not immediately lock the owner again.
			if _, err := e.Svc.LoginStep(ctx, "ana@example.org", pw); err != nil {
				t.Fatalf("no allowance after one failure at lock expiry: %v", err)
			}
		})
	}
	e := configuredThrottleEnv(t)
	e.Person("ana@example.org")
	for i := 0; i < 8; i++ {
		ctx := service.WithClientIP(e.Ctx, fmt.Sprintf("203.0.113.%d", i+1))
		_, err := e.Svc.LoginStep(ctx, "ana@example.org", bad)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, err := e.Svc.LoginStep(service.WithClientIP(e.Ctx, "198.51.100.9"), "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	tu.Must(e.Svc.OperatorUnlockLogin(e.Ctx, domain.SystemActor("operator"), "ana@example.org"))
	if _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatal(err)
	}
}

func TestConfiguredAuthenticationThrottleRecoveryAndSensitivePassword(t *testing.T) {
	for _, operation := range []string{"recovery", "generate-codes"} {
		t.Run(operation, func(t *testing.T) {
			e := configuredThrottleEnv(t)
			actor := e.Person("ana@example.org")
			codes := codesFor(t, e, actor)
			attempt := func(valid bool) error {
				if operation == "recovery" {
					code := "AAAA-AAAA-AAAA-AAAA"
					if valid {
						code = codes[0]
					}
					return e.Svc.RecoverPassword(e.Ctx, "ana@example.org", code, newPW)
				}
				password := bad
				if valid {
					password = pw
				}
				_, err := e.Svc.GenerateRecoveryCodes(e.Ctx, actor, password)
				return err
			}
			for i := 0; i < 4; i++ {
				wantCode(t, attempt(false), domain.ErrUnauthenticated)
			}
			wantCode(t, attempt(true), domain.ErrRateLimited)
			e.Clock.Advance(30 * time.Minute)
			wantCode(t, attempt(false), domain.ErrUnauthenticated)
			noErr(t, attempt(true))
		})
	}
}

func TestConfiguredAuthenticationThrottleMFA(t *testing.T) {
	e := configuredThrottleEnv(t)
	actor := e.Person("ana@example.org")
	secret, _ := enrol(t, e, actor)
	challenge := login(e, "ana@example.org").MFAChallenge
	for i := 0; i < 4; i++ {
		_, _, err := e.Svc.CompleteMFA(e.Ctx, challenge, "not-a-code")
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, _, err := e.Svc.CompleteMFA(e.Ctx, challenge, totpAt(t, secret, now(e)))
	wantCode(t, err, domain.ErrRateLimited)
	e.Clock.Advance(30 * time.Minute)
	challenge = login(e, "ana@example.org").MFAChallenge // old challenge expired
	_, _, err = e.Svc.CompleteMFA(e.Ctx, challenge, "not-a-code")
	wantCode(t, err, domain.ErrUnauthenticated)
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, challenge, totpAt(t, secret, now(e))); err != nil {
		t.Fatalf("configured MFA lock did not release at boundary: %v", err)
	}
}

func TestAuthenticationThrottleRejectsUnsafeServiceConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  service.Config
	}{
		{"auth-throttle-attempts", service.Config{AuthThrottleAttempts: 3}},
		{"auth-throttle-attempts", service.Config{AuthThrottleAttempts: 6}},
		{"auth-throttle-attempts", service.Config{AuthThrottleAttempts: -1}},
		{"auth-throttle-global-attempts", service.Config{AuthThrottleGlobalAttempts: 4}},
		{"auth-throttle-global-attempts", service.Config{AuthThrottleGlobalAttempts: 101}},
		{"auth-throttle-global-attempts", service.Config{AuthThrottleGlobalAttempts: -1}},
		{"auth-throttle-lock", service.Config{AuthThrottleLock: 14 * time.Minute}},
		{"auth-throttle-lock", service.Config{AuthThrottleLock: 24*time.Hour + time.Second}},
		{"auth-throttle-lock", service.Config{AuthThrottleLock: 15*time.Minute + time.Millisecond}},
	} {
		if _, err := service.New(nil, tc.cfg); err == nil || !strings.Contains(err.Error(), tc.name) {
			t.Fatalf("invalid %+v: %v", tc.cfg, err)
		}
	}
}
