package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/testutil"
)

const forgedName = "Mallory\n{\"time\":\"2026-03-01T12:00:00Z\",\"level\":\"INFO\",\"msg\":\"login ok\",\"request_id\":\"forged01\"}\r\n" +
	"time=2026-03-01T12:00:00Z level=INFO msg=\"login ok\"\u2028"

// SEC-020: user-controlled text (a display name with a newline and a fake log
// line) and credentials that reach the API's internal-error and panic logging
// produce one escaped entry each, with the response's request id, UTC time,
// severity and event name, and no credential.
func TestAPILogForgeryAndCredentialsInErrors(t *testing.T) {
	env := testutil.NewEnv(t)
	var buf bytes.Buffer
	h := NewHandlerWithOptions(env.Svc, Options{Logger: slog.New(slog.NewJSONHandler(&buf, nil))}).(*server)
	h.add(route{pattern: "GET /api/v1/zz-forge", public: true, noNet: true, h: func(c *reqCtx) error {
		return fmt.Errorf("member %s: upstream refused %s with %s", forgedName,
			c.r.Header.Get("Authorization"), c.r.Header.Get("Cookie"))
	}})
	h.add(route{pattern: "GET /api/v1/zz-forge-panic", public: true, noNet: true, h: func(c *reqCtx) error {
		panic("member " + forgedName + " token=" + c.r.Header.Get("X-Canary"))
	}})
	var ids []string
	for _, p := range []string{"/api/v1/zz-forge", "/api/v1/zz-forge-panic"} {
		r := httptest.NewRequest("GET", p, nil)
		r.Header.Set("Authorization", "Bearer cir_Zq81canaryApiToken0123456789")
		r.Header.Set("Cookie", "circula_session=Zq81canarySession; cm_csrf=Zq81canaryCsrf")
		r.Header.Set("X-Canary", "Zq81canaryPanicToken")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != 500 {
			t.Fatalf("%s: %d %s", p, rec.Code, rec.Body.String())
		}
		ids = append(ids, rec.Header().Get("X-Request-Id"))
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 log entries, got %d:\n%s", len(lines), out)
	}
	for i, want := range []string{"api internal error", "api panic"} {
		var rec map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &rec); err != nil {
			t.Fatalf("entry %d is not one JSON record: %v\n%s", i, err, lines[i])
		}
		if ts, _ := rec["time"].(string); rec["msg"] != want || rec["request_id"] != ids[i] || rec["level"] != "ERROR" || !strings.HasSuffix(ts, "Z") {
			t.Errorf("entry %d: %v (want %s, request id %s)", i, rec, want, ids[i])
		}
	}
	if strings.Contains(out, "Zq81canary") || strings.Contains(out, "\u2028") {
		t.Errorf("credential or raw separator logged:\n%s", out)
	}
}
