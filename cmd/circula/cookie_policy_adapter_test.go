package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/djshellshoxxx/circula/internal/httpx"
)

// SEC-022/SA2-14: the operator cookie-secure setting must reach the actual
// HTML adapter, not merely parse successfully in the CLI configuration.
// The test deliberately uses a plain HTTP backend request: a TLS-terminating
// reverse proxy cannot make r.TLS non-nil inside the Go service.
func TestServeCookieSecurePolicyReachesHTMLAdapter(t *testing.T) {
	cases := []struct {
		name       string
		secure     bool
		publicURL  string
		cookieName string
	}{
		{"secure behind TLS proxy", true, "https://circula.example", httpx.CSRFCookieSecure},
		{"plain HTTP development", false, "http://127.0.0.1", httpx.CSRFCookie},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				StateDir:      t.TempDir(),
				CookieSecure:  tc.secure,
				PublicBaseURL: tc.publicURL,
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			_, webHandler, err := newHandlers(nil, cfg, logger)
			if err != nil {
				t.Fatalf("construct handlers: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/login", nil)
			rec := httptest.NewRecorder()
			webHandler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("login page status %d, want 200", rec.Code)
			}

			var csrf *http.Cookie
			for _, c := range rec.Result().Cookies() {
				if c.Name == tc.cookieName {
					csrf = c
				}
				if c.Name == httpx.CSRFCookie || c.Name == httpx.CSRFCookieSecure {
					if c.Name != tc.cookieName {
						t.Errorf("unexpected CSRF cookie variant %q", c.Name)
					}
				}
			}
			if csrf == nil {
				t.Fatalf("CSRF cookie %q missing from login response", tc.cookieName)
			}
			if csrf.Secure != tc.secure {
				t.Errorf("CSRF cookie Secure=%v, want %v", csrf.Secure, tc.secure)
			}
			if !csrf.HttpOnly || csrf.Path != "/" || csrf.Domain != "" {
				t.Errorf("CSRF cookie attributes do not meet host-only policy: HttpOnly=%v Path=%q Domain=%q",
					csrf.HttpOnly, csrf.Path, csrf.Domain)
			}
			if got := rec.Header().Get("Strict-Transport-Security"); (got != "") != tc.secure {
				t.Errorf("HSTS present=%v, want %v", got != "", tc.secure)
			}
		})
	}
}
