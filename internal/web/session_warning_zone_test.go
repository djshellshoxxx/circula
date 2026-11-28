package web

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// The real shared shell supplies catalogue-formatted expiry text in the
// member's selected zone while retaining its numeric epoch.
func TestSessionWarningTemplateUsesDisplayZone(t *testing.T) {
	for _, tc := range []struct {
		name, zone, want string
		at               time.Time
	}{
		{"default UTC", "", "28 Nov 2026, 00:00 UTC", time.Date(2026, time.November, 28, 0, 0, 0, 0, time.UTC)},
		{"explicit UTC", "UTC", "28 Nov 2026, 00:00 UTC", time.Date(2026, time.November, 28, 0, 0, 0, 0, time.UTC)},
		{"half-hour zone", "Asia/Kolkata", "28 Nov 2026, 05:30 IST", time.Date(2026, time.November, 28, 0, 0, 0, 0, time.UTC)},
		{"date rollover", "Asia/Kolkata", "29 Nov 2026, 02:30 IST", time.Date(2026, time.November, 28, 21, 0, 0, 0, time.UTC)},
		{"winter", "Europe/Berlin", "2 Jan 2026, 01:00 CET", time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)},
		{"summer", "Europe/Berlin", "2 Jul 2026, 02:00 CEST", time.Date(2026, time.July, 2, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &page{Title: "Session warning", Authed: true, ExpiresAt: tc.at.Unix(), Extendable: true}
			if tc.zone != "" {
				loc, err := time.LoadLocation(tc.zone)
				if err != nil {
					t.Fatal(err)
				}
				p.displayZone = loc
			}
			var body bytes.Buffer
			if err := mustLoadTemplates()["login"].ExecuteTemplate(&body, "base", p); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				`data-expires="` + strconv.FormatInt(tc.at.Unix(), 10) + `"`,
				`data-expires-text="` + template.HTMLEscapeString(tc.want) + `"`,
				`data-extendable="true"`,
			} {
				if !strings.Contains(body.String(), want) {
					t.Errorf("shared shell missing %q", want)
				}
			}
		})
	}
}

// Account-page rendering and the real authenticated, CSRF-protected extension
// route use the saved preference without changing the machine-readable expiry.
func TestSessionWarningHTTPUsesSavedDisplayZone(t *testing.T) {
	const idle = time.Hour
	for _, tc := range []struct {
		name, zone, want string
		at               time.Time
	}{
		{"default UTC", "", "28 Nov 2026, 00:00 UTC", time.Date(2026, time.November, 28, 0, 0, 0, 0, time.UTC)},
		{"half-hour zone", "Asia/Kolkata", "28 Nov 2026, 05:30 IST", time.Date(2026, time.November, 28, 0, 0, 0, 0, time.UTC)},
		{"winter", "Europe/Berlin", "2 Jan 2026, 01:00 CET", time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)},
		{"summer", "Europe/Berlin", "2 Jul 2026, 02:00 CEST", time.Date(2026, time.July, 2, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := testutil.NewEnv(t)
			env.Clock.Advance(tc.at.Add(-idle).Sub(env.Clock.Now()))
			actor := env.Person("session-zone@example.org")
			if err := env.Svc.SetTimeZone(env.Ctx, actor, tc.zone); err != nil {
				t.Fatal(err)
			}
			session, _, err := env.Svc.Login(env.Ctx, "session-zone@example.org", "correct horse battery")
			if err != nil {
				t.Fatal(err)
			}
			s := newServer(env.Svc, Options{
				CSRFKey: []byte("0123456789abcdef0123456789abcdef"),
				Clock:   env.Clock, SessionIdle: idle, SessionMax: 24 * time.Hour,
			})
			h := s.wrap(s.mux)
			request := func(method, path string, csrf bool) *httptest.ResponseRecorder {
				t.Helper()
				r := httptest.NewRequest(method, path, nil)
				r.AddCookie(&http.Cookie{Name: httpx.SessionCookie, Value: session})
				if csrf {
					r.Header.Set("X-CSRF-Token", s.csrf.Token(session))
				}
				out := httptest.NewRecorder()
				h.ServeHTTP(out, r)
				return out
			}
			pageResponse := request(http.MethodGet, "/account/sessions", false)
			if pageResponse.Code != http.StatusOK {
				t.Fatalf("sessions page returned %d: %s", pageResponse.Code, pageResponse.Body.String())
			}
			wantAttribute := `data-expires-text="` + template.HTMLEscapeString(tc.want) + `"`
			if !strings.Contains(pageResponse.Body.String(), wantAttribute) {
				t.Errorf("account page omitted saved-zone expiry %q", wantAttribute)
			}
			// The password handler deliberately renders without a viewer context.
			// This bounded change keeps that existing UTC fallback explicit.
			fallback := request(http.MethodGet, "/account/password", false)
			if fallback.Code != http.StatusOK {
				t.Fatalf("password page returned %d: %s", fallback.Code, fallback.Body.String())
			}
			wantUTC := `data-expires-text="` + tc.at.UTC().Format("2 Jan 2006, 15:04 MST") + `"`
			if !strings.Contains(fallback.Body.String(), wantUTC) {
				t.Errorf("nil-viewer page omitted UTC fallback %q", wantUTC)
			}
			extension := request(http.MethodPost, "/session/extend", true)
			if extension.Code != http.StatusOK {
				t.Fatalf("extension returned %d: %s", extension.Code, extension.Body.String())
			}
			var response struct {
				Expires     int64  `json:"expires"`
				Extendable  bool   `json:"extendable"`
				ExpiresText string `json:"expires_text"`
			}
			if err := json.Unmarshal(extension.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Expires != tc.at.Unix() || !response.Extendable {
				t.Fatalf("numeric session behavior changed: %#v", response)
			}
			if response.ExpiresText != tc.want {
				t.Errorf("extension expiry text %q, want saved-zone %q", response.ExpiresText, tc.want)
			}
			if denied := request(http.MethodPost, "/session/extend", false); denied.Code != http.StatusForbidden {
				t.Fatalf("extension without CSRF returned %d", denied.Code)
			}
		})
	}
}
