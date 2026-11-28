package web

import (
	"bytes"
	"html"
	"html/template"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

const memberCatalogueUnsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"

func renderMemberCatalogue(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantMemberCatalogue(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(body, "<img") || strings.Contains(body, "<script>") {
		t.Fatal("ordinary text became markup")
	}
}

// Presentation labels must not change the domain values or the action set.
func TestMemberCatalogueRowsAndActions(t *testing.T) {
	var rows []memberRow
	for _, tc := range []struct {
		status  domain.MemberStatus
		roles   []domain.Role
		flag    bool
		actions []string
	}{
		{domain.MemberPending, nil, false, []string{"approve", "deny"}},
		{domain.MemberActive, nil, false, []string{"limit", "suspend", "remove", "reset-password", "grant-admin", "grant-reviewer", "flag"}},
		{domain.MemberActive, []domain.Role{domain.RoleAdmin, domain.RoleReviewer}, true, []string{"limit", "suspend", "remove", "reset-password", "revoke-admin", "revoke-reviewer", "unflag"}},
		{domain.MemberSuspended, nil, false, []string{"restore", "remove"}},
		{domain.MemberDeparted, nil, false, []string{"erase"}},
		{domain.MemberStatus(memberCatalogueUnsafe), []domain.Role{domain.Role(memberCatalogueUnsafe)}, false, nil},
	} {
		m := &domain.Member{ID: "member-1000", DisplayName: memberCatalogueUnsafe, Status: tc.status, Roles: tc.roles, DefaultFlag: tc.flag}
		row := newMemberRow(m)
		var actions []string
		for _, a := range row.Actions {
			actions = append(actions, a.Value)
		}
		if row.Member != m || row.Status != tc.status || !reflect.DeepEqual(actions, tc.actions) || row.StatusLabel != string(tc.status) {
			t.Fatalf("member state/action selection changed: %#v", row)
		}
		if len(tc.roles) == 2 && !reflect.DeepEqual(row.Roles, []string{"admin", "reviewer"}) {
			t.Fatal("known role labels or their order changed")
		}
		rows = append(rows, row)
	}
	if memberRoleLabel(domain.Role(memberCatalogueUnsafe)) != memberCatalogueUnsafe || removalReasonLabel(memberCatalogueUnsafe) != memberCatalogueUnsafe {
		t.Fatal("unknown display values were changed")
	}
	for _, status := range []string{"", "pending", "active", "suspended", "departed"} {
		p := &page{Slug: "town", NetName: memberCatalogueUnsafe, V: map[string]string{"status": status},
			D: map[string]any{"Rows": rows, "Pages": pageNav{Total: 1000, First: "/n/town/admin/members?status=" + status,
				Prev: "/n/town/admin/members?status=" + status + "&before=member-1000", Next: "/n/town/admin/members?status=" + status + "&after=member-1000"}}}
		body := renderMemberCatalogue(t, "admin_members", p)
		wantMemberCatalogue(t, body, `method="get" action="/n/town/admin/members"`, `aria-label="Members table"`,
			`data-label="Name"`, `data-label="Status"`, `data-label="Roles"`, `data-label="Actions"`, `class="tag">in default</span>`,
			"1,000 members match this filter.", `aria-label="Member list pages"`, "before=member-1000", "after=member-1000",
			`href="/n/town/admin/members/member-1000/approve" aria-label="`+template.HTMLEscapeString("Approve "+memberCatalogueUnsafe)+`">Approve</a>`,
			`<caption>Members of `+template.HTMLEscapeString(memberCatalogueUnsafe)+`</caption>`,
			`data-label="Status">`+template.HTMLEscapeString(memberCatalogueUnsafe)+`</td>`,
			`data-label="Roles"><ul class="role-list"><li>`+template.HTMLEscapeString(memberCatalogueUnsafe)+`</li></ul></td>`)
		for _, value := range []string{"pending", "active", "suspended", "departed"} {
			selected := strings.Contains(body, `<option value="`+value+`" selected>`)
			if selected != (value == status) {
				t.Fatalf("filter %q changed option %q", status, value)
			}
		}
	}
	body := renderMemberCatalogue(t, "admin_members", &page{D: map[string]any{"Rows": []memberRow(nil), "Pages": pageNav{}}})
	wantMemberCatalogue(t, body, "No members match this filter.")
	if strings.Contains(body, `<table`) {
		t.Fatal("empty membership list gained a table")
	}
}

func TestMemberActionCatalogueControls(t *testing.T) {
	m := &domain.Member{ID: "member-1000", DisplayName: memberCatalogueUnsafe}
	for _, key := range []string{"approve", "deny", "suspend", "restore", "remove", "grant-admin", "revoke-admin", "grant-reviewer", "revoke-reviewer", "flag", "unflag", "limit", "reset-password"} {
		t.Run(key, func(t *testing.T) {
			def := memberActions[key]
			p := &page{Slug: "town", CSRF: "csrf-token", Title: messages.Text("admin.member_action.confirm_title", def.Verb, m.DisplayName),
				D: newMemberAct(key, def, m), V: map[string]string{"amount": "90071992547409.93", "reason": memberCatalogueUnsafe,
					"transaction_id": memberCatalogueUnsafe, "reason_code": domain.RemovalConfirmedFraud}}
			if key == "remove" {
				p.Errors = []fieldErr{{"reason_code", "Choose a reason."}}
			}
			body := renderMemberCatalogue(t, "admin_member_act", p)
			wantMemberCatalogue(t, body, `method="post" action="/n/town/admin/members/member-1000/`+key+`"`, `name="csrf" value="csrf-token"`,
				template.HTMLEscapeString(def.Consequence), template.HTMLEscapeString(m.DisplayName), ">"+template.HTMLEscapeString(def.Verb)+"</button>", ">Go back</a>")
			if strings.Contains(body, `id="f-amount"`) != def.Amount || strings.Contains(body, `id="f-reason_code"`) != (key == "remove") {
				t.Fatal("action gained or lost its amount/removal fields")
			}
			if def.Amount {
				wantMemberCatalogue(t, body, `for="f-amount">Credit limit</label>`, `value="90071992547409.93" inputmode="decimal"`,
					"Amount in credits. Leave empty to use the network default.")
			}
			if def.Reason {
				wantMemberCatalogue(t, body, `aria-describedby="h-reason" required`, `value="`+template.HTMLEscapeString(memberCatalogueUnsafe)+`"`)
			}
			if key == "remove" {
				wantMemberCatalogue(t, body, `href="#f-reason_code"`, `aria-invalid="true" aria-describedby="h-reason_code e-reason_code"`,
					`<option value="confirmed_fraud" selected>Confirmed fraud</option>`, `maxlength="500"`, `for="f-transaction_id">Trade reference (optional)</label>`)
				for _, code := range domain.RemovalReasons {
					wantMemberCatalogue(t, body, `value="`+code+`"`, template.HTMLEscapeString(domain.RemovalReasonLabel(code)))
				}
			}
			if key == "reset-password" {
				if def.Verb != messages.Text("admin.reset.action") || def.Consequence != messages.Text("admin.reset.consequence") || !def.Reason || def.Amount {
					t.Fatal("completed reset action definition changed")
				}
				wantMemberCatalogue(t, body, `for="f-reason">Reason</label>`, "Required. It is recorded in the audit log.", ">Reset password</button>")
			}
		})
	}
	// A future reason label remains ordinary data; this does not change the allowed codes.
	d := newMemberAct("remove", memberActions["remove"], m)
	d.Codes = append(d.Codes, option{Value: "future_code", Label: removalReasonLabel(memberCatalogueUnsafe)})
	body := renderMemberCatalogue(t, "admin_member_act", &page{D: d})
	wantMemberCatalogue(t, body, `value="future_code">`+template.HTMLEscapeString(memberCatalogueUnsafe)+`</option>`)
}

func TestInvitationCatalogueStateAndValues(t *testing.T) {
	now := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	c := &rc{loc: time.FixedZone("IST", 5*3600+30*60)}
	var rows []invRow
	for _, tc := range []struct {
		id     string
		i      domain.Invitation
		label  string
		revoke bool
	}{
		{"unused-1000", domain.Invitation{ExpiresAt: now.Add(time.Hour)}, "Unused", true},
		{"boundary-1000", domain.Invitation{ExpiresAt: now}, "Unused", true},
		{"expired-1000", domain.Invitation{ExpiresAt: now.Add(-time.Nanosecond)}, "Expired", false},
		{"revoked-1000", domain.Invitation{ExpiresAt: now.Add(time.Hour), RevokedAt: &now}, "Revoked", false},
		{"revoked-expired-1000", domain.Invitation{ExpiresAt: now.Add(-time.Hour), RevokedAt: &now}, "Revoked", false},
		{"used-1000", domain.Invitation{ExpiresAt: now.Add(time.Hour), UsedAt: &now}, "Used", false},
		{"used-revoked-expired-1000", domain.Invitation{ExpiresAt: now.Add(-time.Hour), UsedAt: &now, RevokedAt: &now}, "Used", false},
	} {
		tc.i.ID = tc.id
		row := c.invitationRow(&tc.i, now)
		if row.Status != tc.label || row.CanRevoke != tc.revoke || row.ID != tc.id {
			t.Fatalf("invitation state/precedence changed: %#v", row)
		}
		if !strings.Contains(string(row.Expires.(template.HTML)), ">"+row.ExpiresLabel+"</time>") || !strings.HasSuffix(row.ExpiresLabel, " IST") {
			t.Fatal("expiration time and accessible name disagree")
		}
		rows = append(rows, row)
	}
	p := &page{Slug: "town", CSRF: "csrf-token", V: map[string]string{"days": memberCatalogueUnsafe},
		Summary: messages.Text("admin.invitations.invalid"), Errors: []fieldErr{{"days", messages.Text("admin.invitations.days_error")}},
		D: &invData{List: rows, NewLink: "https://example.test/join/1000?value=" + memberCatalogueUnsafe}}
	body := renderMemberCatalogue(t, "admin_invites", p)
	wantMemberCatalogue(t, body, "Invitation created. Copy this link now: it is shown only once.", `name="csrf" value="csrf-token"`,
		`for="f-days">Valid for (days)</label>`, `value="`+template.HTMLEscapeString(memberCatalogueUnsafe)+`"`,
		`aria-invalid="true" aria-describedby="h-days e-days" required`, `href="#f-days"`, "Valid for: enter a whole number of days from 1 to 90.",
		`aria-label="Revoke invitation expiring 19 Nov 2026, 18:30 IST">Revoke</button>`)
	for _, row := range rows {
		if strings.Contains(body, `action="/n/town/admin/invitations/`+row.ID+`/revoke"`) != row.CanRevoke {
			t.Fatalf("revocation control changed for %s", row.ID)
		}
	}
	body = renderMemberCatalogue(t, "admin_invites", &page{D: &invData{}})
	wantMemberCatalogue(t, body, "No invitations yet.")
	if strings.Contains(body, "Invitation created.") || strings.Contains(body, "/revoke") {
		t.Fatal("empty invitation list gained a result or revocation")
	}
}

// Deliberately disagreeing labels prove that display wording is not the gate.
func TestInvitationRevocationUsesState(t *testing.T) {
	for _, tc := range []struct {
		label string
		allow bool
	}{{`Translated <img src=x>`, true}, {"Unused", false}} {
		body := renderMemberCatalogue(t, "admin_invites", &page{Slug: "town", D: &invData{List: []invRow{{ID: "invite-1000", Status: tc.label,
			Expires: fmtTime(time.Unix(1000, 0)), ExpiresLabel: memberCatalogueUnsafe, CanRevoke: tc.allow}}}})
		if strings.Contains(body, `action="/n/town/admin/invitations/invite-1000/revoke"`) != tc.allow {
			t.Fatal("invitation label still controls revocation")
		}
		wantMemberCatalogue(t, body, template.HTMLEscapeString(tc.label)+", expires ")
		if tc.allow {
			wantMemberCatalogue(t, body, `aria-label="`+template.HTMLEscapeString("Revoke invitation expiring "+memberCatalogueUnsafe)+`">Revoke</button>`)
		}
	}
}

func TestInactiveMembersCatalogue(t *testing.T) {
	m := &domain.Member{ID: "member-1000", DisplayName: memberCatalogueUnsafe, Status: domain.MemberActive, JoinedAt: time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		days int
		rows []memberRow
	}{{0, nil}, {60, nil}, {60, []memberRow{newMemberRow(m)}}} {
		body := renderMemberCatalogue(t, "admin_inactive", &page{Slug: "town", NetName: memberCatalogueUnsafe,
			D: map[string]any{"Days": tc.days, "Rows": tc.rows}})
		wantMemberCatalogue(t, body, `href="/n/town/admin/members">Back to members</a>`)
		switch {
		case tc.days == 0:
			wantMemberCatalogue(t, body, `This network has no inactivity rule. You can set one in the <a href="/n/town/admin/policy">policy</a>.`)
		case len(tc.rows) == 0:
			wantMemberCatalogue(t, body, "No member is flagged as inactive.", "60 days.")
		default:
			wantMemberCatalogue(t, body, "60 days.", `aria-label="Inactive members table"`, `data-label="Joined"><time datetime="2026-11-19">2026-11-19</time></td>`,
				`<caption>Inactive members of `+template.HTMLEscapeString(memberCatalogueUnsafe)+`</caption>`,
				`href="/n/town/admin/members/member-1000/limit" aria-label="`+template.HTMLEscapeString("Set credit limit "+memberCatalogueUnsafe)+`">Set credit limit</a>`)
		}
	}
}

func TestMemberInvitationCatalogueEscapesMessages(t *testing.T) {
	const injected = `"><img src=x> & catalogue`
	m := &domain.Member{ID: "member-1000", DisplayName: "Ana", Status: domain.MemberActive}
	for _, tc := range []struct {
		name, marker string
		data         any
	}{
		{"admin_members", `for="f-status">`, map[string]any{"Rows": []memberRow{newMemberRow(m)}, "Pages": pageNav{Total: 1}}},
		{"admin_member_act", `id="h-amount">`, newMemberAct("limit", memberActions["limit"], m)},
		{"admin_invites", `for="f-days">`, &invData{}},
		{"admin_inactive", `data-label="`, map[string]any{"Days": 60, "Rows": []memberRow{newMemberRow(m)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tpl := mustLoadTemplates()[tc.name].Funcs(template.FuncMap{"msg": func(key string, args ...any) string {
				if strings.HasPrefix(key, "admin.members.") || strings.HasPrefix(key, "admin.member_action.") || strings.HasPrefix(key, "admin.invitations.") || strings.HasPrefix(key, "admin.inactive.") {
					return injected
				}
				return messages.Text(key, args...)
			}})
			var out bytes.Buffer
			if err := tpl.ExecuteTemplate(&out, "base", &page{D: tc.data}); err != nil {
				t.Fatal(err)
			}
			body := out.String()
			if !strings.Contains(body, tc.marker+template.HTMLEscapeString(injected)) || strings.Contains(body, "<img") || !strings.Contains(html.UnescapeString(body), injected) {
				t.Fatal("page ignored catalogue text or trusted its markup")
			}
		})
	}
}
