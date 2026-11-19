package web

import (
	"bytes"
	"html"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// These cases use the real embedded templates and catalogue without a store.
// They can run alongside production web files when database compilation is
// unavailable; the handler/CSRF tests remain a separate integration gate.
func TestSessionTemplatesRender(t *testing.T) {
	templates := mustLoadTemplates()
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	current := service.SessionInfo{ID: strings.Repeat("a", 64), Current: true, CreatedAt: now,
		LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	other := current
	other.ID, other.Current = strings.Repeat("b", 64), false
	position := "1763553600000:" + other.ID
	for _, tc := range []struct {
		name, template string
		data           any
		errors         []fieldErr
		want           []string
	}{
		{"list", "sessions", service.SessionPage{Rows: []service.SessionInfo{current, other}, Next: position}, nil,
			[]string{"This session", "Other session", "Sign out here", "/account/sessions/" + other.ID + "/revoke"}},
		{"empty continuation", "sessions", service.SessionPage{Next: position}, nil,
			[]string{"There are no active sessions on this page.", "Next page", "/account/sessions?after="}},
		{"current confirmation", "session_revoke", &current, nil,
			[]string{"You will be signed out here", `for="f-confirm"`, `name="csrf"`, `value="yes"`}},
		{"other confirmation", "session_revoke", &other, nil,
			[]string{"You will stay signed in here", "Keep session", "End session"}},
		{"confirmation error", "session_revoke", &other, []fieldErr{{"confirm", messages.Text("sessions.confirm_label")}},
			[]string{`aria-invalid="true"`, `aria-describedby="e-confirm"`, `id="e-confirm"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			p := &page{Title: messages.Text("sessions.title"), D: tc.data, Authed: true,
				CSRF: "test-csrf", Errors: tc.errors, V: map[string]string{}}
			if err := templates[tc.template].ExecuteTemplate(&out, "base", p); err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(out.String(), want) {
					t.Errorf("rendered template lacks %q", want)
				}
			}
			if pg, ok := tc.data.(service.SessionPage); ok && pg.Next != "" {
				// The literal query context must preserve the composite cursor.
				// A dynamic URL prefix can instead filter its colon as a scheme.
				const prefix = `href="/account/sessions?after=`
				_, rest, ok := strings.Cut(out.String(), prefix)
				if !ok {
					t.Fatal("session page has no continuation URL")
				}
				query, _, ok := strings.Cut(rest, `"`)
				if !ok {
					t.Fatal("session continuation URL is not quote-delimited")
				}
				link, err := url.Parse("/account/sessions?after=" + html.UnescapeString(query))
				if err != nil {
					t.Fatalf("session continuation URL: %v", err)
				}
				if link.Fragment != "" || link.Query().Get("after") != pg.Next || strings.Contains(out.String(), "ZgotmplZ") {
					t.Fatalf("session cursor did not round trip through href: %q", link.String())
				}
			}
		})
	}
}
