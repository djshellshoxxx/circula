package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

// SEC-020, OPS-032: the process logger escapes control characters, redacts
// credentials and stamps UTC in both log formats.
func TestServeLoggerEscapesRedactsAndUsesUTC(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		var buf bytes.Buffer
		l := newLogger(&Config{LogLevel: slog.LevelDebug, LogFormat: format}, &buf)
		l.Error("scheduled job failed", "job", "backup", "password", "Zq81canaryPassword",
			"error", errors.New("open /x: refused\nlevel=INFO msg=\"forged entry\" token=Zq81canaryToken"))
		out := buf.String()
		if n := strings.Count(out, "\n"); n != 1 {
			t.Fatalf("%s: want one line, got %d:\n%s", format, n, out)
		}
		if strings.Contains(out, "Zq81canary") {
			t.Errorf("%s: credential logged:\n%s", format, out)
		}
		if format == "json" {
			var rec map[string]any
			if err := json.Unmarshal([]byte(out), &rec); err != nil {
				t.Fatalf("json: %v\n%s", err, out)
			}
			if ts, _ := rec["time"].(string); !strings.HasSuffix(ts, "Z") || rec["level"] != "ERROR" || rec["msg"] != "scheduled job failed" {
				t.Errorf("json entry: %v", rec)
			}
		} else if !regexp.MustCompile(`^time=\S+Z level=ERROR msg="scheduled job failed"`).MatchString(out) {
			t.Errorf("text entry: %s", out)
		}
	}
}

// SEC-020: the access log of the full serve stack (edge, API, UI) records no
// credential sent in a path, query, header, cookie or form, and every entry
// has a UTC time, severity, event name and request id.
func TestAccessLogCarriesNoCredentials(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	cfg := testConfigFor(db)
	svc, err := newService(st, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var logs syncBuf
	logger := newLogger(&Config{LogLevel: slog.LevelDebug, LogFormat: "json"}, &logs)
	apiH, webH, err := newHandlers(svc, cfg, logger)
	if err != nil {
		t.Fatal(err)
	}
	ops := &opsServer{cfg: &Config{MaxBodyBytes: 1 << 20, RequestTimeout: 5 * time.Second}, db: st.DB(), logger: logger, metrics: newMetrics(time.Now())}
	ops.ready.Store(true)
	srv := httptest.NewServer(ops.routes(apiH, webH))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	sent := 0
	send := func(method, path string, hdr map[string]string, form url.Values) string {
		t.Helper()
		var body io.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req, _ := http.NewRequest(method, srv.URL+path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		sent++
		return string(b)
	}
	canaries := map[string]string{
		"password":         "correct horse battery",
		"invitation token": "inv_Zq81canaryInvite0123",
		"query token":      "cir_Zq81canaryQueryToken0123456",
		"bearer token":     "cir_Zq81canaryHeaderToken012345",
		"forged cookie":    "Zq81canarySessionCookie",
		"form password":    "Zq81 canary form password",
	}
	send("GET", "/join/"+canaries["invitation token"], nil, nil)
	send("GET", "/api/v1/me?access_token="+canaries["query token"], nil, nil)
	send("GET", "/api/v1/me", map[string]string{"Authorization": "Bearer " + canaries["bearer token"]}, nil)
	send("GET", "/", map[string]string{"Cookie": "circula_session=" + canaries["forged cookie"]}, nil)
	send("POST", "/login", nil, url.Values{"login": {"founder01"}, "password": {canaries["form password"]}})
	page := send("GET", "/login", nil, nil)
	m := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(page)
	if m == nil {
		t.Fatal("no csrf token on /login")
	}
	canaries["csrf token"] = m[1]
	send("POST", "/login", nil, url.Values{"csrf": {m[1]}, "login": {"founder01"}, "password": {canaries["password"]}})
	u, _ := url.Parse(srv.URL)
	for _, ck := range jar.Cookies(u) {
		canaries["cookie "+ck.Name] = ck.Value
	}
	if canaries["cookie circula_session"] == "" {
		t.Fatalf("sign-in set no session cookie: %v", jar.Cookies(u))
	}
	send("GET", "/n/test-net", nil, nil)
	send("GET", "/healthz", nil, nil)

	// The access-log line is written after the response; wait for all of them.
	deadline := time.Now().Add(2 * time.Second)
	for strings.Count(logs.String(), `"msg":"request"`) < sent && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	out := logs.String()
	if got := strings.Count(out, `"msg":"request"`); got < sent {
		t.Fatalf("want %d access-log entries, got %d:\n%s", sent, got, out)
	}
	for kind, v := range canaries {
		if strings.Contains(out, v) {
			t.Errorf("%s reached the log", kind)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("not a JSON record: %v\n%s", err, line)
		}
		ts, _ := rec["time"].(string)
		rid, _ := rec["request_id"].(string)
		if !strings.HasSuffix(ts, "Z") || rec["level"] == nil || rec["msg"] == "" || rid == "" {
			t.Errorf("entry lacks UTC time, level, event name or request id: %s", line)
		}
	}
}
