package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// UX-047, SEC-016: the picker preserves names, status meaning and fixed local
// links, with contextual escaping for both user values and catalogue output.
func TestNetworkPickerCatalogue(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x onerror=alert(1)>`
	var links []netLink
	for _, status := range domain.AllMemberStatuses {
		links = append(links, netLink{Name: unsafe, Slug: "town-" + string(status), Status: string(status)})
	}
	links = append(links, netLink{Name: "Future membership state", Slug: "town-future", Status: unsafe})
	for _, tc := range []struct {
		name  string
		links []netLink
	}{{"empty", nil}, {"memberships", links}} {
		t.Run(tc.name, func(t *testing.T) {
			tpl := mustLoadTemplates()["networks"]
			var out bytes.Buffer
			if err := tpl.ExecuteTemplate(&out, "base", &page{Title: messages.Text("onboarding.networks.title"), D: tc.links}); err != nil {
				t.Fatal(err)
			}
			body := out.String()
			for _, want := range []string{`<h1 id="page-title">Your networks</h1>`, `href="/setup">Create a network</a>`} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if tc.links == nil {
				if !strings.Contains(body, messages.Text("onboarding.networks.empty")) || strings.Contains(body, `<ul class="rows">`) {
					t.Fatal("empty picker lost its guidance or gained membership rows")
				}
				return
			}
			for _, status := range domain.AllMemberStatuses {
				want := `href="/n/town-` + string(status) + `">` + template.HTMLEscapeString(unsafe) + `</a> <span class="tag">` + string(status) + `</span>`
				if !strings.Contains(body, want) {
					t.Errorf("membership link or status changed: %q", want)
				}
			}
			if strings.Contains(body, "<img") || strings.Contains(body, messages.Text("onboarding.networks.empty")) ||
				!strings.Contains(body, `<span class="tag">`+template.HTMLEscapeString(unsafe)+`</span>`) {
				t.Fatal("picker failed to escape an unknown status/name or showed empty-state guidance with rows")
			}
		})
	}
}

func TestNetworkPickerCatalogueTextIsEscaped(t *testing.T) {
	const unsafe = `"><img src=x onerror="alert(1)"> & catalogue`
	for _, data := range [][]netLink{nil, {{Name: "Town", Slug: "town", Status: "active"}}} {
		tpl := mustLoadTemplates()["networks"].Funcs(template.FuncMap{
			"msg": func(key string, args ...any) string {
				if strings.HasPrefix(key, "onboarding.networks.") {
					return unsafe
				}
				return messages.Text(key, args...)
			},
		})
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, "base", &page{D: data}); err != nil {
			t.Fatal(err)
		}
		body := out.String()
		escaped := template.HTMLEscapeString(unsafe)
		if strings.Contains(body, "<img") || !strings.Contains(body, `href="/setup">`+escaped+`</a>`) {
			t.Fatal("picker creation label ignored the catalogue or trusted its markup")
		}
		if data == nil && !strings.Contains(body, "<p>"+escaped+"</p>") {
			t.Fatal("empty-state guidance ignored the catalogue or trusted its markup")
		}
		if data != nil && !strings.Contains(body, `<span class="tag">`+escaped+`</span>`) {
			t.Fatal("membership label ignored the catalogue or trusted its markup")
		}
	}
}
