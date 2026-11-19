package web

import (
	"html"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// UX-047: shared form refusals keep the original correction and accessible
// field association, while service details remain escaped by the real page.
func TestValidationCatalogueRendersEscapedFieldDetails(t *testing.T) {
	c := &rc{Net: &domain.Network{Policy: domain.Policy{Decimals: 2}}}
	detail := "Zoë 100% " + evil
	summary, fields, ok := c.refuse(domain.Invalid("reason", detail))
	if !ok || summary != "Not saved. Fix the problems below. Nothing changed." || len(fields) != 1 {
		t.Fatalf("unexpected refusal: %q %+v %v", summary, fields, ok)
	}
	def := memberActions["reset-password"]
	body := renderAuthPage(t, "admin_member_act", &page{
		Title: messages.Text("admin.reset.confirm_title", "Zoë 100%"), Slug: "town", CSRF: "csrf-value",
		Summary: summary, Errors: fields, V: map[string]string{"reason": detail},
		D: &memberAct{Key: "reset-password", Def: def, M: &domain.Member{ID: "member-id", DisplayName: "Zoë 100%"}},
	})
	wantAll(t, "validation", body, html.EscapeString("Reason: "+detail+". Check the value and try again."),
		`href="#f-reason"`, `aria-invalid="true"`, `id="e-reason"`, `name="csrf" value="csrf-value"`,
		`action="/n/town/admin/members/member-id/reset-password"`)
	if strings.Contains(body, "<script>alert") {
		t.Fatal("service detail or retained form value crossed the escaping boundary")
	}
	if got := label("future_field"); got != "future field" {
		t.Fatalf("unknown-field fallback changed: %q", got)
	}
}

// UX-047: render both administrator reset screens through the actual shared
// templates; names and the one-time password remain plain escaped values.
func TestAdminResetPagesCatalogue(t *testing.T) {
	def := memberActions["reset-password"]
	for _, name := range []string{"Zoë 100%", evil} {
		t.Run(name, func(t *testing.T) {
			body := renderAuthPage(t, "admin_member_act", &page{
				Title: messages.Text("admin.reset.confirm_title", name), Slug: "town", CSRF: "csrf-value",
				D: &memberAct{Key: "reset-password", Def: def, M: &domain.Member{ID: "member-id", DisplayName: name}},
			})
			wantAll(t, "reset confirmation", body, html.EscapeString("Reset password: "+name+"?"),
				html.EscapeString("Reset password: "+name), html.EscapeString(def.Consequence),
				`for="f-reason">Reason`, "Required. It is recorded in the audit log.", ">Reset password</button>",
				">Go back</a>", `name="csrf" value="csrf-value"`)
			if strings.Contains(body, "<script>alert") {
				t.Fatal("confirmation member name crossed the escaping boundary")
			}
			body = renderAuthPage(t, "admin_reset_done", &page{
				Title: messages.Text("admin.reset.done_title"), Slug: "town", D: &resetDone{Name: name, Temp: evil},
			})
			wantAll(t, "temporary password", body, html.EscapeString("Temporary password for "+name),
				"<strong>This is the only time it is shown.</strong>",
				html.EscapeString("Give it to "+name+" in person or by a channel you trust. They must choose a new password when they sign in. All their sessions and API tokens were ended."),
				`<code class="temp">`+html.EscapeString(evil)+"</code>", ">Back to members</a>")
			if strings.Contains(body, "<script>alert") {
				t.Fatal("temporary password or member name crossed the escaping boundary")
			}
		})
	}
}
