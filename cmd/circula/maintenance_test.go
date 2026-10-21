package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func maintOps(t *testing.T, cfgOn bool) (*opsServer, string) {
	t.Helper()
	ops, _ := testOps(t)
	ops.cfg.Maintenance = cfgOn
	ops.maint = newMaintenance(ops.cfg)
	return ops, maintenancePath(ops.cfg.Database)
}

// OPS-024: web and API answer 503 with Retry-After; health stays up.
func TestMaintenanceModeServes503ExceptOps(t *testing.T) {
	ops, _ := maintOps(t, true)
	h := ops.routes(stubHandler("api"), stubHandler("web"))

	api := do(t, h, "GET", "/api/v1/me", "127.0.0.1:1", nil)
	if api.Code != 503 || api.Header().Get("Retry-After") != "300" ||
		!strings.HasPrefix(api.Header().Get("Content-Type"), "application/problem+json") ||
		!strings.Contains(api.Body.String(), `"code":"maintenance"`) {
		t.Errorf("api: %d %v %s", api.Code, api.Header(), api.Body.String())
	}
	web := do(t, h, "POST", "/login", "127.0.0.1:1", nil)
	if web.Code != 503 || web.Header().Get("Retry-After") != "300" ||
		!strings.Contains(web.Body.String(), "Back soon") || strings.Contains(web.Body.String(), "web ") {
		t.Errorf("web: %d %v %s", web.Code, web.Header(), web.Body.String())
	}
	if hz := do(t, h, "GET", "/healthz", "127.0.0.1:1", nil); hz.Code != 200 {
		t.Errorf("healthz in maintenance: %d", hz.Code)
	}
	if rz := do(t, h, "GET", "/readyz", "127.0.0.1:1", nil); rz.Code != 200 {
		t.Errorf("readyz in maintenance: %d", rz.Code)
	}
	if m := do(t, h, "GET", "/metrics", "127.0.0.1:1", nil); m.Code != 200 || !strings.Contains(m.Body.String(), "circula_maintenance_mode 1") {
		t.Errorf("metrics in maintenance: %d %s", m.Code, m.Body.String())
	}
}

func TestMaintenanceOffPassesThrough(t *testing.T) {
	ops, _ := maintOps(t, false)
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	if r := do(t, h, "GET", "/api/v1/me", "127.0.0.1:1", nil); r.Code != 200 || r.Body.String() != "api /api/v1/me" {
		t.Errorf("api: %d %s", r.Code, r.Body.String())
	}
	if r := do(t, h, "GET", "/x", "127.0.0.1:1", nil); r.Code != 200 {
		t.Errorf("web: %d", r.Code)
	}
}

// The file switch toggles a running server without a restart.
func TestMaintenanceFileSwitch(t *testing.T) {
	ops, path := maintOps(t, false)
	now := time.Now()
	ops.maint.now = func() time.Time { return now }
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	var out, errb bytes.Buffer
	args := []string{"-database", ops.cfg.Database, "-public-base-url", "https://x.test"}

	if code := cmdMaintenance(context.Background(), append(append([]string{}, args...), "-retry-after", "120", "on"), &out, &errb); code != exitOK {
		t.Fatalf("on: %d %s", code, errb.String())
	}
	now = now.Add(2 * time.Second)
	r := do(t, h, "GET", "/api/v1/me", "127.0.0.1:1", nil)
	if r.Code != 503 || r.Header().Get("Retry-After") != "120" {
		t.Fatalf("after on: %d %v", r.Code, r.Header())
	}
	if code := cmdMaintenance(context.Background(), append(append([]string{}, args...), "off"), &out, &errb); code != exitOK {
		t.Fatalf("off: %d %s", code, errb.String())
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("file remains after off")
	}
	now = now.Add(2 * time.Second)
	if r := do(t, h, "GET", "/api/v1/me", "127.0.0.1:1", nil); r.Code != 200 {
		t.Fatalf("after off: %d", r.Code)
	}
	if code := cmdMaintenance(context.Background(), append(append([]string{}, args...), "sideways"), &out, &errb); code != exitUsage {
		t.Errorf("bad action: %d", code)
	}
	if code := cmdMaintenance(context.Background(), append(append([]string{}, args...), "-retry-after", "0", "on"), &out, &errb); code != exitUsage {
		t.Errorf("bad retry-after: %d", code)
	}
}
