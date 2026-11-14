package logx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/logx"
	"github.com/djshellshoxxx/circula/internal/service"
)

// forged is a display name that tries to end the current entry and append a
// second, fake one (SEC-020 acceptance: newline forgery).
const forged = "Mallory\n{\"time\":\"2026-01-01T00:00:00Z\",\"level\":\"INFO\",\"msg\":\"login ok\",\"user\":\"admin\"}\r\n" +
	"time=2026-01-01T00:00:00Z level=INFO msg=\"login ok\"\u2028\u2029\u0085\u202e\x1b[2J"

type sink struct{ bytes.Buffer }

func newJSON(t *testing.T) (*slog.Logger, *sink) {
	t.Helper()
	var s sink
	return logx.New(slog.New(slog.NewJSONHandler(&s, &slog.HandlerOptions{Level: slog.LevelDebug}))), &s
}

func newText(t *testing.T) (*slog.Logger, *sink) {
	t.Helper()
	var s sink
	return logx.New(slog.New(slog.NewTextHandler(&s, &slog.HandlerOptions{Level: slog.LevelDebug}))), &s
}

// emit writes one entry that carries the forged value in every position a
// caller could put user-controlled text.
func emit(l *slog.Logger) {
	l.With("display_name", forged).WithGroup(forged).Error(forged,
		"request_id", "rid-1234567",
		"display_name", forged,
		"err", errors.New("render failed for "+forged),
		"panic", any([]string{forged}),
		slog.Group("member", "name", forged),
	)
}

func TestNewlineForgeryIsOneJSONEntry(t *testing.T) {
	l, s := newJSON(t)
	emit(l)
	out := s.String()
	if n := strings.Count(out, "\n"); n != 1 {
		t.Fatalf("want exactly one log line, got %d:\n%s", n, out)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("entry is not one JSON object: %v\n%s", err, out)
	}
	if strings.Count(out, `"msg":`) != 1 || strings.Contains(out, `"msg":"login ok"`) {
		t.Fatalf("forged entry survived: %s", out)
	}
	for _, r := range []string{"\u2028", "\u2029", "\u0085", "\u202e", "\x1b"} {
		if strings.Contains(out, r) {
			t.Errorf("raw %q in output", r)
		}
	}
	if !strings.Contains(rec["msg"].(string), `Mallory\n{`) {
		t.Errorf("newline not visibly escaped in msg: %q", rec["msg"])
	}
}

func TestNewlineForgeryIsOneTextEntry(t *testing.T) {
	l, s := newText(t)
	emit(l)
	out := s.String()
	if n := strings.Count(out, "\n"); n != 1 {
		t.Fatalf("want exactly one log line, got %d:\n%s", n, out)
	}
	// The forged "level=INFO" text stays inside a quoted value of the one line.
	if !strings.HasPrefix(out, "time=") || !strings.Contains(out, ` level=ERROR msg="Mallory\\n{`) || strings.Contains(out, "\r") {
		t.Fatalf("forged entry survived: %s", out)
	}
}

func TestLiteralBackslashIsDistinguishable(t *testing.T) {
	if got := logx.Clean(`a\nb`); got != `a\\nb` {
		t.Errorf("literal backslash-n: %q", got)
	}
	if got := logx.Clean("a\nb"); got != `a\nb` {
		t.Errorf("newline: %q", got)
	}
	if got := logx.Clean("plain text 42"); got != "plain text 42" {
		t.Errorf("plain: %q", got)
	}
	if got := logx.Clean("bad \xff byte"); got != `bad \ufffd byte` {
		t.Errorf("invalid utf-8: %q", got)
	}
}

func TestSensitiveKeysRedacted(t *testing.T) {
	l, s := newJSON(t)
	canary := "canary-Zq81-secret-value"
	keys := []string{"password", "Password", "new_password", "session", "session_id", "SessionID", "csrf", "csrf_token",
		"totp_secret", "mfa_secret", "api_token", "token", "Authorization", "cookie", "recovery_code", "otp", "api-key"}
	args := []any{}
	for _, k := range keys {
		args = append(args, k, canary)
	}
	l.Info("event", args...)
	l.With("secret", canary).Info("event", slog.Group("auth", "password", canary))
	out := s.String()
	if strings.Contains(out, canary) {
		t.Fatalf("canary under a sensitive key reached the log:\n%s", out)
	}
	if c := strings.Count(out, logx.Redacted); c != len(keys)+2 {
		t.Errorf("want %d redactions, got %d:\n%s", len(keys)+2, c, out)
	}
}

func TestNeutralKeysKept(t *testing.T) {
	l, s := newJSON(t)
	l.Info("request", "request_id", "abc12345", "route", "/api/", "status", 200, "duration_ms", int64(3), "footprint", "ok")
	out := s.String()
	for _, want := range []string{`"request_id":"abc12345"`, `"route":"/api/"`, `"status":200`, `"duration_ms":3`, `"footprint":"ok"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}

func TestSecretsInFreeTextRedacted(t *testing.T) {
	cases := map[string]string{
		"invitation token":    "GET /join/inv_Zq81canaryInvite01 failed",
		"api token":           "bearer cir_Zq81canaryApiToken0123456789 rejected",
		"bearer header":       "authorization: Bearer Zq81canaryBearer.value",
		"password field":      `decode {"login":"x","password":"Zq81canaryPass"}`,
		"password kv":         "password=Zq81canaryPass2 and more",
		"session cookie":      "__Host-circula_session=Zq81canarySession; Path=/",
		"session id kv":       "session_id: Zq81canarySid",
		"csrf kv":             "csrf=Zq81canaryCsrf&x=1",
		"totp kv":             "totp_secret=Zq81CANARYTOTPSECRET",
		"recovery code kv":    "recovery code: Zq81-canary",
		"email address":       "no person with login alice.Zq81canary@example.org",
		"international phone": "call +44 20 7946 0958 now",
	}
	for name, in := range cases {
		out := logx.Clean(in)
		for _, frag := range []string{"Zq81canary", "Zq81CANARY", "Zq81-canary", "7946 0958"} {
			if strings.Contains(out, frag) {
				t.Errorf("%s: %q leaked in %q", name, frag, out)
			}
		}
	}
	// Agreement with the service's own redaction (service.RedactSecrets).
	for _, s := range []string{"/join/inv_abcdefgh1234", "cir_abcdefghijklmnop", "x inv_short"} {
		if strings.Contains(service.RedactSecrets(s), "_REDACTED") && !strings.Contains(logx.Clean(s), "_REDACTED") {
			t.Errorf("logx.Clean keeps a token service.RedactSecrets removes: %q -> %q", s, logx.Clean(s))
		}
	}
	// Ordinary text is untouched.
	for _, s := range []string{"ledger verification failed; network frozen", "invalid session expired", "2026-11-04 12:00:00 +0000 UTC"} {
		if got := logx.Clean(s); got != s {
			t.Errorf("ordinary text changed: %q -> %q", s, got)
		}
	}
}

func TestTimeIsUTC(t *testing.T) {
	l, s := newJSON(t)
	berlin := time.FixedZone("CET", 3600)
	l.Info("event", "at", time.Date(2026, 1, 1, 12, 0, 0, 0, berlin))
	var rec map[string]any
	if err := json.Unmarshal(s.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"time", "at"} {
		if v, _ := rec[k].(string); !strings.HasSuffix(v, "Z") {
			t.Errorf("%s not UTC: %q", k, v)
		}
	}
	if rec["at"] != "2026-01-01T11:00:00Z" || rec["level"] != "INFO" || rec["msg"] != "event" {
		t.Errorf("entry: %v", rec)
	}
}

func TestWrapIdempotentAndLevels(t *testing.T) {
	var s sink
	inner := slog.NewJSONHandler(&s, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := logx.Wrap(inner)
	if logx.Wrap(h) != h {
		t.Error("Wrap of a wrapped handler must not add a layer")
	}
	l := slog.New(h)
	if logx.New(l) != l {
		t.Error("New of a wrapped logger must return it unchanged")
	}
	l.Info("dropped")
	l.Warn("kept")
	if strings.Contains(s.String(), "dropped") || !strings.Contains(s.String(), "kept") {
		t.Errorf("level not delegated: %s", s.String())
	}
	if logx.New(nil) == nil {
		t.Error("New(nil) must fall back to the default logger")
	}
}
