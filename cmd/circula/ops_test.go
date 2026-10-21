package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

func testOps(t *testing.T) (*opsServer, *store.SQLite) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	st, err := store.Open(ctx, filepath.Join(dir, "circula.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := &Config{
		Database:       filepath.Join(dir, "circula.db"),
		MaxBodyBytes:   1024,
		RequestTimeout: 5 * time.Second,
		MetricsAllow:   []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128")},
	}
	ops := &opsServer{cfg: cfg, db: st.DB(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), metrics: newMetrics(time.Now())}
	ops.ready.Store(true)
	return ops, st
}

// stubHandler stands in for the API and UI, echoing the path it received.
func stubHandler(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, name+" "+r.URL.Path)
	})
}

func do(t *testing.T, h http.Handler, method, path, remote string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = remote
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthzReportsCoarseStatusOnly(t *testing.T) {
	ops, _ := testOps(t)
	h := ops.routes(stubHandler("api"), stubHandler("web"))

	rec := do(t, h, http.MethodGet, "/healthz", "203.0.113.9:4000", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz code = %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["version"] == "" || len(body) != 2 {
		t.Errorf("healthz body = %v; want only status and version", body)
	}
}

func TestHealthzUnhealthyWhenDatabaseClosed(t *testing.T) {
	ops, st := testOps(t)
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	rec := do(t, h, http.MethodGet, "/healthz", "127.0.0.1:1", nil)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "unavailable") {
		t.Errorf("healthz with closed database: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, http.MethodGet, "/readyz", "127.0.0.1:1", nil); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz with closed database: %d", rec.Code)
	}
}

func TestReadyzFollowsServingState(t *testing.T) {
	ops, _ := testOps(t)
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	if rec := do(t, h, http.MethodGet, "/readyz", "127.0.0.1:1", nil); rec.Code != http.StatusOK {
		t.Fatalf("ready: %d", rec.Code)
	}
	ops.ready.Store(false)
	if rec := do(t, h, http.MethodGet, "/readyz", "127.0.0.1:1", nil); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("draining readyz: %d", rec.Code)
	}
}

func TestMetricsRefusedOutsideAllowedRange(t *testing.T) {
	ops, _ := testOps(t)
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	if rec := do(t, h, http.MethodGet, "/metrics", "203.0.113.9:4000", nil); rec.Code != http.StatusForbidden {
		t.Errorf("external metrics code = %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/metrics", "127.0.0.1:4000", map[string]string{"X-Forwarded-For": "198.51.100.7"}); rec.Code != http.StatusForbidden {
		t.Errorf("forwarded metrics code = %d; a proxy must not expose it", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/metrics", "[::1]:4000", nil); rec.Code != http.StatusOK {
		t.Errorf("loopback v6 metrics code = %d", rec.Code)
	}
}

func TestMetricsExposeRequiredSeriesWithRouteTemplates(t *testing.T) {
	ops, _ := testOps(t)
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	do(t, h, http.MethodGet, "/api/v1/networks/SECRET-NET-ID/members", "203.0.113.1:1", nil)
	do(t, h, http.MethodGet, "/api/v1/networks/SECRET-NET-ID/members", "203.0.113.1:1", nil)
	ops.metrics.ledgerResult(false, 2, time.Now())
	ops.metrics.jobRan("expiry", "ok")
	rec := do(t, h, http.MethodGet, "/metrics", "127.0.0.1:1", nil)
	body := rec.Body.String()
	for _, want := range []string{
		`circula_http_requests_total{route="/api/",class="2xx"} 2`,
		`circula_ledger_verification_ok 0`,
		`circula_ledger_verification_failing_networks 2`,
		`circula_job_runs_total{job="expiry",result="ok"} 1`,
		`circula_schema_version`,
		`circula_http_request_duration_seconds_count{route="/api/"} 2`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics lack %q", want)
		}
	}
	if strings.Contains(body, "SECRET-NET-ID") {
		t.Error("metrics leak a path identifier")
	}
}

func TestRequestIDAndRoutesMounted(t *testing.T) {
	ops, _ := testOps(t)
	h := ops.routes(stubHandler("api"), stubHandler("web"))
	rec := do(t, h, http.MethodGet, "/api/v1/ping", "127.0.0.1:1", map[string]string{"X-Request-Id": "abc-123"})
	if rec.Body.String() != "api /api/v1/ping" {
		t.Errorf("api mount = %q", rec.Body.String())
	}
	if rec.Header().Get("X-Request-Id") != "abc-123" {
		t.Errorf("request id not echoed: %q", rec.Header().Get("X-Request-Id"))
	}
	rec = do(t, h, http.MethodGet, "/trades", "127.0.0.1:1", map[string]string{"X-Request-Id": "bad id\x01"})
	if rec.Body.String() != "web /trades" || rec.Header().Get("X-Request-Id") == "" || rec.Header().Get("X-Request-Id") == "bad id\x01" {
		t.Errorf("web mount or request id replacement wrong: %q %q", rec.Body.String(), rec.Header().Get("X-Request-Id"))
	}
}

func TestBodyLimitApplied(t *testing.T) {
	ops, _ := testOps(t)
	reader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := ops.routes(reader, stubHandler("web"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/x", strings.NewReader(strings.Repeat("a", 4096)))
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body code = %d", rec.Code)
	}
}

func TestPanicRecoveredAsInternalError(t *testing.T) {
	ops, _ := testOps(t)
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := ops.routes(boom, stubHandler("web"))
	rec := do(t, h, http.MethodGet, "/api/v1/x", "127.0.0.1:1", nil)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("panic response = %d %q", rec.Code, rec.Body.String())
	}
}

func TestRouteLabelsAreTemplates(t *testing.T) {
	for in, want := range map[string]string{
		"/api/v1/x/123": "/api/",
		"/healthz":      "/healthz",
		"/metrics":      "/metrics",
		"/trades/42":    "/",
	} {
		if got := routeOf(in); got != want {
			t.Errorf("routeOf(%q) = %q, want %q", in, got, want)
		}
	}
}
