package web

import (
	"html/template"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// UX-047: the actual administrator reset templates preserve complete
// catalogue text, native form labels and contextual escaping. These render
// tests use no store; TestWebAdminReset covers the real reset and forced change.
func TestAdministratorResetPagesCatalogue(t *testing.T) {
	name := "Zoë 100% " + evil
	escapedName := template.HTMLEscapeString(name)
	def := memberActions["reset-password"]
	newPage := func() *page {
		return &page{Title: messages.Text("auth.admin_reset.title", name), Slug: "town",
			Authed: true, CSRF: "test-csrf", V: map[string]string{"reason": evil},
			D: &memberAct{Key: "reset-password", Def: def, M: &domain.Member{ID: "member-1", DisplayName: name}}}
	}

	t.Run("confirmation", func(t *testing.T) {
		body := renderAuthPage(t, "admin_member_act", newPage())
		wantAll(t, "reset confirmation", body,
			"<h1 id=\"page-title\">Reset password: "+escapedName+"?</h1>",
			"<p class=\"big\">Reset password: "+escapedName+"</p>",
			"A one-time temporary password is created and shown to you once. The member&#39;s sessions and API tokens end, their recovery codes stop working, and they must choose a new password at next sign-in. Give them the temporary password in person or by a channel you trust.",
			`method="post" action="/n/town/admin/members/member-1/reset-password"`,
			`name="csrf" value="test-csrf"`, `<label for="f-reason">Reason</label>`,
			`id="h-reason">Required. It is recorded in the audit log.</p>`,
			`aria-describedby="h-reason" required`, ">Reset password</button>", ">Go back</a>")
		if strings.Contains(body, "<script>alert") || !strings.Contains(body, template.HTMLEscapeString(evil)) {
			t.Fatal("reset confirmation did not escape the member name and submitted reason")
		}
	})

	t.Run("reason error", func(t *testing.T) {
		p := newPage()
		// Shared field validation is unchanged by this catalogue continuation.
		p.Summary = "Not saved. Fix the problems below. Nothing changed."
		p.Errors = []fieldErr{{"reason", "Reason: required. Fill it in and try again."}}
		body := renderAuthPage(t, "admin_member_act", p)
		wantAll(t, "reset reason error", body, `href="#f-reason"`,
			`aria-invalid="true"`, `aria-describedby="h-reason e-reason"`,
			`id="e-reason"`, "Reason: required. Fill it in and try again.", ">Reset password</button>")
	})

	for _, key := range []string{"auth.admin_reset.forbidden", "auth.admin_reset.rate_limited"} {
		t.Run(key, func(t *testing.T) {
			p := newPage()
			p.Summary = messages.Text(key)
			body := renderAuthPage(t, "admin_member_act", p)
			wantAll(t, key, body, `id="error-summary"`, template.HTMLEscapeString(p.Summary),
				`<label for="f-reason">Reason</label>`, ">Go back</a>")
			if strings.Contains(body, `<code class="temp">`) {
				t.Fatal("a refused reset rendered a temporary password")
			}
		})
	}

	t.Run("temporary password", func(t *testing.T) {
		temp := "one-time-100% " + evil
		body := renderAuthPage(t, "admin_reset_done", &page{
			Title: messages.Text("auth.admin_reset.done.title"), Slug: "town", Authed: true,
			D: &resetDone{Name: name, Temp: temp}})
		wantAll(t, "reset result", body, `<h1 id="page-title">Temporary password</h1>`,
			"Temporary password for "+escapedName, "<strong>This is the only time it is shown.</strong>",
			"Give it to "+escapedName+" in person or by a channel you trust. They must choose a new password when they sign in. All their sessions and API tokens were ended.",
			`<code class="temp">`+template.HTMLEscapeString(temp)+"</code>",
			`href="/n/town/admin/members">Back to members</a>`)
		if strings.Contains(body, "<script>alert") {
			t.Fatal("reset result rendered the member name or temporary password as markup")
		}
	})

	t.Run("other member action", func(t *testing.T) {
		body := renderAuthPage(t, "admin_member_act", &page{Title: "Suspend: Ana?", Slug: "town",
			D: &memberAct{Key: "suspend", Def: memberActions["suspend"],
				M: &domain.Member{ID: "member-2", DisplayName: "Ana"}}})
		wantAll(t, "shared action", body, "<p class=\"big\">Suspend: Ana</p>",
			"The member can no longer propose, accept or post. They can still view history and decline trades.",
			`<label for="f-reason">Reason</label>`, ">Suspend</button>", ">Go back</a>")
	})
}
