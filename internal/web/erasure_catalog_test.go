package web

import (
	"bytes"
	"html"
	"html/template"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

const erasureCatalogueUnsafe = `Zoë 100% $& "><img src=x> </textarea>` + "\x00%!"

func renderErasureCatalogue(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func requireErasureCatalogue(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(body, "<img") || strings.Contains(body, "<script>") {
		t.Fatal("erasure text became markup")
	}
}

// SEC-018, UX-047: both existing scopes keep their exact target/consent fields,
// reason requirements, retained values and irreversible/retention explanations.
func TestErasureCatalogueForms(t *testing.T) {
	for _, self := range []bool{false, true} {
		for _, invalid := range []bool{false, true} {
			p := &page{Slug: "town", CSRF: "csrf-token", V: map[string]string{"reason": erasureCatalogueUnsafe},
				Title: messages.Text("erasure.member_title", erasureCatalogueUnsafe),
				D:     &eraseData{MemberID: "member-1000", Name: erasureCatalogueUnsafe, Self: self}}
			action, back := "/n/town/admin/members/member-1000/erase", "/n/town/admin/members?status=departed"
			heading, hint := "Erase the personal data of "+erasureCatalogueUnsafe, messages.Text("erasure.reason_required")
			if self {
				p.Title = messages.Text("erasure.self_title")
				action, back = "/n/town/erase-my-data", "/n/town"
				heading, hint = messages.Text("erasure.self_heading"), messages.Text("erasure.reason_optional")
			}
			if invalid {
				p.Summary = messages.Text("erasure.confirm_required")
				p.Errors = []fieldErr{{"confirm", messages.Text("erasure.confirm_error")}, {"reason", erasureCatalogueUnsafe}}
			}
			body := renderErasureCatalogue(t, "erase_confirm", p)
			requireErasureCatalogue(t, body, `<h1 id="page-title">`+template.HTMLEscapeString(p.Title)+`</h1>`,
				`<p class="big">`+template.HTMLEscapeString(heading)+`</p>`, `method="post" action="`+action+`" novalidate`,
				`name="csrf" value="csrf-token"`, `for="f-reason">Reason</label>`, `id="h-reason">`+template.HTMLEscapeString(hint)+`</p>`,
				`value="`+template.HTMLEscapeString(erasureCatalogueUnsafe)+`"`,
				`<label><input type="checkbox" id="f-confirm" name="confirm" value="yes"`,
				`> I understand this cannot be undone</label>`, `class="danger">Erase personal data</button>`, `href="`+back+`">Go back</a>`)
			for _, key := range []string{"erasure.irreversible", "erasure.name_effect", "erasure.listings_effect", "erasure.notes_effect", "erasure.login_effect", "erasure.records_effect"} {
				if !strings.Contains(html.UnescapeString(body), messages.Text(key)) {
					t.Errorf("erasure consequence was omitted: %s", key)
				}
			}
			if strings.Contains(body, ` required`) != !self || strings.Contains(body, `name="member_id"`) || strings.Contains(body, ` checked`) {
				t.Fatal("reason requirement, target scope or unchecked consent changed")
			}
			if invalid {
				requireErasureCatalogue(t, body, `id="error-summary"`, `href="#f-confirm"`, `id="e-confirm"`,
					`name="confirm" value="yes" aria-invalid="true" aria-describedby="e-confirm"`,
					`href="#f-reason"`, `aria-describedby="h-reason e-reason"`,
					messages.Text("erasure.confirm_required"), messages.Text("erasure.confirm_error"))
			} else if strings.Contains(body, `aria-invalid="true"`) {
				t.Fatal("ordinary confirmation acquired error attributes")
			}
		}
	}
}

func TestErasureActionCatalogue(t *testing.T) {
	def := memberActions["erase"]
	if def.Verb != "Erase personal data" || def.Consequence != messages.Text("erasure.consequence") || !def.Reason || def.Amount {
		t.Fatal("erasure action definition changed beyond its catalogue source")
	}
	m := &domain.Member{ID: "member-1000", DisplayName: erasureCatalogueUnsafe, Status: domain.MemberDeparted}
	body := renderErasureCatalogue(t, "admin_members", &page{Slug: "town", D: map[string]any{
		"Rows": []memberRow{newMemberRow(m)}, "Pages": pageNav{Total: 1}}})
	requireErasureCatalogue(t, body, `href="/n/town/admin/members/member-1000/erase" aria-label="`+
		template.HTMLEscapeString("Erase personal data "+erasureCatalogueUnsafe)+`">Erase personal data</a>`)
}

func TestErasureCatalogueEscapesMessages(t *testing.T) {
	const injected = `"><img src=x> & catalogue`
	for _, self := range []bool{false, true} {
		tpl := mustLoadTemplates()["erase_confirm"].Funcs(template.FuncMap{"msg": func(key string, args ...any) string {
			if strings.HasPrefix(key, "erasure.") {
				return injected
			}
			return messages.Text(key, args...)
		}})
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, "base", &page{D: &eraseData{Self: self, Name: "Ana"}}); err != nil {
			t.Fatal(err)
		}
		body, escaped := out.String(), template.HTMLEscapeString(injected)
		requireErasureCatalogue(t, body, `<p class="big">`+escaped+`</p>`, `id="h-reason">`+escaped+`</p>`,
			`> `+escaped+`</label>`, `class="danger">`+escaped+`</button>`)
	}
}
