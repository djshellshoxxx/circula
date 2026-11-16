package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

func TestPersistentAuthThrottleSettings(t *testing.T) {
	cfg, _, err := resolveWith(t, nil, baseEnv(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthThrottleAttempts != 5 || cfg.AuthThrottleGlobalAttempts != 100 || cfg.AuthThrottleLock != 15*time.Minute {
		t.Fatalf("defaults changed: %d/%d/%s", cfg.AuthThrottleAttempts, cfg.AuthThrottleGlobalAttempts, cfg.AuthThrottleLock)
	}
	for key, values := range map[string][]string{
		"auth-throttle-attempts":        {"0", "3", "6", "off", "4.5", "-1"},
		"auth-throttle-global-attempts": {"0", "4", "101", "off"},
		"auth-throttle-lock":            {"0", "14m59s", "24h1s", "15m1ms", "off"},
	} {
		for _, value := range values {
			_, _, err := resolveWith(t, []string{"-" + key, value}, baseEnv(t, nil))
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("accepted invalid %s=%s: %v", key, value, err)
			}
		}
	}
	// Flag beats env; the generic every-source test covers JSON, env and CLI
	// for every setting and requires a sample for each new throttle setting.
	cfg, vals, err := resolveWith(t, []string{"-auth-throttle-attempts", "4", "-auth-throttle-global-attempts", "4", "-auth-throttle-lock", "24h"},
		baseEnv(t, map[string]string{envName("auth-throttle-attempts"): "5"}))
	if err != nil || cfg.AuthThrottleAttempts != 4 || cfg.AuthThrottleGlobalAttempts != 4 || cfg.AuthThrottleLock != 24*time.Hour || vals["auth-throttle-attempts"].source != "flag" {
		t.Fatalf("configured limits: %+v / %v", cfg, err)
	}
}

// Exercise the actual command composition root: parsed settings must reach
// the service; a parser-only test would miss missing wiring in newService.
func TestPersistentAuthThrottleServiceWiring(t *testing.T) {
	cfg, _, err := resolveWith(t, []string{"-auth-throttle-attempts", "4", "-auth-throttle-global-attempts", "8", "-auth-throttle-lock", "30m"}, baseEnv(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "throttle.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc, err := newService(st, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		_, err := svc.LoginStep(service.WithClientIP(ctx, "203.0.113.1"), "nobody", "wrong-password")
		if domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	_, err = svc.LoginStep(service.WithClientIP(ctx, "203.0.113.1"), "nobody", "wrong-password")
	if domain.CodeOf(err) != domain.CodeRateLimited {
		t.Fatalf("configured service threshold ignored: %v", err)
	}
	var lockedUntil int64
	if err := st.DB().QueryRow(`SELECT locked_until FROM login_throttle WHERE login = ?`, "nobody ip:203.0.113.1").Scan(&lockedUntil); err != nil {
		t.Fatal(err)
	}
	if left := time.Until(time.UnixMilli(lockedUntil)); left < 29*time.Minute || left > 30*time.Minute {
		t.Fatalf("configured lock ignored: remaining %s", left)
	}
}
