package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// UX-047, PER-022, UX-030: the actual shell supplies complete text messages,
// keeps its server time element, and requires no script for the snapshot label.
func TestProgressiveCatalogueTemplate(t *testing.T) {
	when := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	stamp := fmtTime(when)
	p := &page{Title: "Example", Authed: true, ExpiresAt: when.Unix(), Extendable: true, SnapshotAt: stamp}
	var b bytes.Buffer
	if err := mustLoadTemplates()["login"].ExecuteTemplate(&b, "base", p); err != nil {
		t.Fatal(err)
	}
	body := b.String()
	for _, want := range []string{
		`<p id="figure-freshness">Page generated ` + string(stamp) + `. Figures are a snapshot and do not update automatically.</p>`,
		`id="figure-stale" class="notice" role="status" hidden`,
		`data-offline-message="The browser reports that the connection is unavailable. The figures have not refreshed. {snapshot}"`,
		`data-history-message="This earlier page was restored from browser history. The figures have not refreshed. {snapshot}"`,
		`id="session-warning" class="notice" role="alert" tabindex="-1" hidden data-expires="1795089600" data-extendable="true"`,
		`data-extendable-message="It ends around {time}. Choose Stay signed in to keep working."`,
		`data-fixed-message="It ends around {time} and cannot be extended. Finish what you are doing, then sign in again."`,
		`data-ended-message="Your session has ended. Sign in again to continue. Your changes on this page have not been sent."`,
		`data-failed-message="Could not extend the session. Sign in again."`,
		`<strong>Your session is about to end.</strong>`,
		`id="session-extend">Stay signed in</button>`,
		`id="session-login" href="/login">Sign in again</a>`,
		`id="session-dismiss" class="link">Dismiss</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, "%!") {
		t.Fatal("catalogue interpolation corrupted")
	}
}

func TestProgressiveCatalogueEscapesAttributesAndTimeSurroundings(t *testing.T) {
	const unsafe = `"><img src=x onerror="alert('x')"> & $&`
	tpl := mustLoadTemplates()["login"].Funcs(template.FuncMap{
		"msg": func(key string, args ...any) string {
			if strings.HasPrefix(key, "freshness.") || strings.HasPrefix(key, "session_warning.") {
				return unsafe + messages.Text(key, args...)
			}
			return messages.Text(key, args...)
		},
		"msgAround": func(key string) messages.Parts {
			parts := messages.Around(key)
			parts.Before = unsafe + parts.Before
			parts.After += unsafe
			return parts
		},
	})
	stamp := fmtTime(time.Unix(0, 0).UTC())
	p := &page{Title: "Example", Authed: true, ExpiresAt: 1, SnapshotAt: stamp}
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, "base", p); err != nil {
		t.Fatal(err)
	}
	body := b.String()
	escaped := template.HTMLEscapeString(unsafe)
	for _, attribute := range []string{"data-offline-message", "data-history-message", "data-extendable-message", "data-fixed-message", "data-ended-message", "data-failed-message"} {
		if !strings.Contains(body, attribute+`="`+escaped) {
			t.Errorf("%s was not escaped", attribute)
		}
	}
	if !strings.Contains(body, `<p id="figure-freshness">`+escaped+"Page generated "+string(stamp)) ||
		!strings.Contains(body, "automatically."+escaped+"</p>") {
		t.Error("time surroundings were not escaped or the existing time markup was lost")
	}
	if strings.Contains(body, "<img") || strings.Count(body, "<time ") != 1 {
		t.Error("catalogue text crossed the template escaping boundary")
	}
}
