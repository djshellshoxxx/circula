package main

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web"
)

// SEC-003, SEC-006, SEC-032: the new settings default to the behaviour that
// existed before them.
func TestAuthSettingsDefaultsKeepBehaviour(t *testing.T) {
	cfg, vals, err := resolveWith(t, nil, baseEnv(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionIdleAdmin != cfg.SessionIdle || cfg.SessionMaxAdmin != cfg.SessionMax ||
		cfg.SessionIdleReviewer != cfg.SessionIdle || cfg.SessionMaxReviewer != cfg.SessionMax {
		t.Errorf("role lifetimes differ from member lifetimes by default: %+v", cfg)
	}
	for _, k := range []string{"session-idle-admin", "session-idle-reviewer"} {
		if r := vals[k]; r.value != "720h" || r.source != "derived" {
			t.Errorf("%s = %+v, want 720h derived from session-idle", k, r)
		}
	}
	if r := vals["session-max-admin"]; r.value != "2160h" || r.source != "derived" {
		t.Errorf("session-max-admin = %+v", r)
	}
	if got := webLimits(cfg); got != web.DefaultLimits {
		t.Errorf("web limits %+v, want the built-in %+v", got, web.DefaultLimits)
	}
	if cfg.APIRatePrincipal != 600 || cfg.APIRateAddress != 1200 {
		t.Errorf("API rates %d/%d, want 600/1200", cfg.APIRatePrincipal, cfg.APIRateAddress)
	}
	if cfg.NetworkJoinLimit != (rateSpec{}) {
		t.Errorf("network-join-limit on by default: %+v", cfg.NetworkJoinLimit)
	}
	if cfg.SignupGrantCap != domain.MaxPolicyAmount {
		t.Errorf("signup-grant-cap default %d, want the policy bound %d", cfg.SignupGrantCap, domain.MaxPolicyAmount)
	}
}

func TestAuthSettingsParsing(t *testing.T) {
	get := func(kv ...string) (*Config, error) {
		t.Helper()
		extra := map[string]string{}
		for i := 0; i < len(kv); i += 2 {
			extra[envName(kv[i])] = kv[i+1]
		}
		cfg, _, err := resolveWith(t, nil, baseEnv(t, extra))
		return cfg, err
	}
	cfg, err := get("session-idle-admin", "60m", "session-max-admin", "12h", "session-idle-reviewer", "2h")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SessionIdleAdmin != time.Hour || cfg.SessionMaxAdmin != 12*time.Hour || cfg.SessionIdleReviewer != 2*time.Hour ||
		cfg.SessionMaxReviewer != cfg.SessionMax {
		t.Errorf("role lifetimes: %+v", cfg)
	}
	// a role lifetime may not exceed the member one, and has the same floor
	if _, err := get("session-idle", "1h", "session-idle-admin", "2h"); err == nil || !strings.Contains(err.Error(), "session-idle-admin") {
		t.Errorf("admin idle longer than session-idle accepted: %v", err)
	}
	if _, err := get("session-max-reviewer", "30s"); err == nil {
		t.Error("30s reviewer lifetime accepted")
	}

	cfg, err = get("rate-register", "3/30s", "network-join-limit", "50/24h", "api-rate-principal", "10")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RateRegister != (rateSpec{3, 30 * time.Second}) || cfg.NetworkJoinLimit != (rateSpec{50, 24 * time.Hour}) || cfg.APIRatePrincipal != 10 {
		t.Errorf("rates: %+v %+v %d", cfg.RateRegister, cfg.NetworkJoinLimit, cfg.APIRatePrincipal)
	}
	if l := webLimits(cfg); l.Register != (web.Limit{Max: 3, Window: 30 * time.Second}) || l.Join != web.DefaultLimits.Join {
		t.Errorf("web limits: %+v", l)
	}
	for _, bad := range [][2]string{
		{"rate-join", "off"}, {"rate-join", "0/1m"}, {"rate-join", "10"}, {"rate-join", "10/500ms"}, {"rate-join", "10/1.5s"},
		{"rate-signin-account", "x/1m"}, {"network-join-limit", "5/forever"}, {"api-rate-address", "0"},
		{"signup-grant-cap", "-1"}, {"signup-grant-cap", "1.5"},
	} {
		if _, err := get(bad[0], bad[1]); err == nil || !strings.Contains(err.Error(), bad[0]) {
			t.Errorf("%s=%s: %v", bad[0], bad[1], err)
		}
	}
	if cfg, err = get("network-join-limit", "0"); err != nil || cfg.NetworkJoinLimit != (rateSpec{}) {
		t.Errorf("network-join-limit=0: %+v %v", cfg, err)
	}
	// 0 forbids grants: the service sees a negative cap
	if cfg, err = get("signup-grant-cap", "0"); err != nil || cfg.SignupGrantCap != -1 {
		t.Errorf("signup-grant-cap=0: %+v %v", cfg, err)
	}
	if cfg, err = get("signup-grant-cap", "2500"); err != nil || cfg.SignupGrantCap != 2500 {
		t.Errorf("signup-grant-cap=2500: %+v %v", cfg, err)
	}
}
