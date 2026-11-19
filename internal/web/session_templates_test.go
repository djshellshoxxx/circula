package web

import (
	"bytes"
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
	for _, tc := range []struct {
		name, template string
		data           any
		errors         []fieldErr
		want           []string
	}{
		{"list", "sessions", []service.SessionInfo{current, other}, nil,
			[]string{"This session", "Other session", "Sign out here", "/account/sessions/" + other.ID + "/revoke"}},
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
		})
	}
}
