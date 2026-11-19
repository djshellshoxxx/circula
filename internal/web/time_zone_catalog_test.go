package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func renderTimeZoneCatalogue(t *testing.T, tpl *pageTemplate, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := tpl.ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// UX-047, SEC-016: the form preserves machine-readable UTC time, the display
// zone, raw input and accessible validation associations without trusting text.
func TestTimeZoneCatalogueRetainsValuesAndMarkup(t *testing.T) {
	when := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	const unsafe = `  Mars/Base "><img src=x onerror=alert(1)> %! ` + "\x00  "
	for _, tc := range []struct{ name, current, input, display string }{
		{"default UTC", "UTC", "", "19 Nov 2026, 12:00 UTC"},
		{"saved IANA ID", "Asia/Kolkata", "Asia/Kolkata", "19 Nov 2026, 17:30 IST"},
		{"invalid retained input", "UTC", unsafe, "19 Nov 2026, 12:00 UTC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stamp := fmtTime(when.In(service.Location(tc.current)))
			p := &page{Title: messages.Text("account.time_zone"), CSRF: "csrf-token", V: map[string]string{"time_zone": tc.input},
				D: &timeZoneData{Current: tc.current, Example: stamp}}
			if tc.name == "invalid retained input" {
				p.Summary = messages.Text("time_zone.invalid_summary")
				p.Errors = []fieldErr{{"time_zone", messages.Text("time_zone.invalid_field")}}
			}
			body := renderTimeZoneCatalogue(t, mustLoadTemplates()["time_zone"], p)
			for _, want := range []string{
				`<h1 id="page-title">Time zone</h1>`,
				`<p>Pages show times in your time zone. The API and the audit log always use UTC. Now it is ` + string(stamp) + ` in <strong>` + tc.current + `</strong>.</p>`,
				`datetime="2026-11-19T12:00:00Z">` + tc.display + `</time>`,
				`method="post" action="/account/time-zone" novalidate`, `name="csrf" value="csrf-token"`,
				`<label for="f-time_zone">Time zone</label>`,
				`id="f-time_zone" name="time_zone" type="text" value="` + template.HTMLEscapeString(tc.input) + `" maxlength="64"`,
				`id="h-time_zone">An IANA name such as Europe/Berlin, Asia/Kolkata or America/Argentina/Buenos_Aires. Leave empty for UTC.</p>`,
				`>Save time zone</button>`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if tc.name == "invalid retained input" {
				for _, want := range []string{`id="error-summary"`, `href="#f-time_zone"`, `id="e-time_zone"`,
					`aria-invalid="true"`, `aria-describedby="h-time_zone e-time_zone"`, "Not saved. Fix the problem below. Nothing changed."} {
					if !strings.Contains(body, want) {
						t.Errorf("invalid form lost %q", want)
					}
				}
			} else if strings.Contains(body, `aria-invalid="true"`) || !strings.Contains(body, `aria-describedby="h-time_zone"`) {
				t.Fatal("valid input gained an error or lost its hint association")
			}
			if strings.Contains(body, "<img") || strings.Contains(body, "<select") {
				t.Fatal("retained text became markup or the existing input changed to a selector")
			}
		})
	}
}

func TestTimeZoneCatalogueLabelsAreEscaped(t *testing.T) {
	const unsafe = `"><img src=x onerror="alert(1)"> & catalogue`
	tpl := mustLoadTemplates()["time_zone"].Funcs(template.FuncMap{
		"msg": func(key string, args ...any) string {
			if key == "account.time_zone" || strings.HasPrefix(key, "time_zone.") {
				return unsafe
			}
			return messages.Text(key, args...)
		},
	})
	body := renderTimeZoneCatalogue(t, tpl, &page{D: &timeZoneData{Current: "UTC", Example: fmtTime(time.Unix(0, 0).UTC())}})
	escaped := template.HTMLEscapeString(unsafe)
	for _, want := range []string{`<label for="f-time_zone">` + escaped + `</label>`, `id="h-time_zone">` + escaped + `</p>`, `>` + escaped + `</button>`} {
		if !strings.Contains(body, want) {
			t.Errorf("label ignored the catalogue or was not escaped: %q", want)
		}
	}
	if strings.Contains(body, "<img") {
		t.Fatal("catalogue content became markup")
	}
}

func TestTimeZoneCataloguePartsAreEscaped(t *testing.T) {
	const unsafe = `"><img src=x onerror="alert(1)"> & %!` + "\x00"
	tpl := mustLoadTemplates()["time_zone"].Funcs(template.FuncMap{
		"msgAroundTwo": func(string) messages.TwoParts {
			return messages.TwoParts{Before: unsafe, Between: unsafe, After: unsafe}
		},
	})
	stamp := fmtTime(time.Unix(0, 0).UTC())
	body := renderTimeZoneCatalogue(t, tpl, &page{D: &timeZoneData{Current: unsafe, Example: stamp}})
	escaped := template.HTMLEscapeString(unsafe)
	want := "<p>" + escaped + string(stamp) + escaped + "<strong>" + escaped + "</strong>" + escaped + "</p>"
	if !strings.Contains(body, want) || strings.Contains(body, "<img") || strings.Count(body, "<time ") != 1 {
		t.Fatal("status text or zone ID became markup, or the existing time element was lost")
	}
}
