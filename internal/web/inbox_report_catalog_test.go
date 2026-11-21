package web

import (
	"bytes"
	"html"
	"html/template"
	"strings"
	"testing"
	"time"
)

func renderInboxReportPage(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantInboxReportText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

// UX-047, FR-038: the actual inbox preserves read actions, filter/cursor links,
// the dispute time element and escaped complete action names.
func TestInboxCatalogueCountsAndActions(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x onerror=alert(1)>`
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	body := renderInboxReportPage(t, "inbox", &page{Slug: "town", CSRF: "csrf-value", D: &inboxData{
		Unread: 1000, Filter: "unread", Next: "next-id",
		Items: []inboxRow{
			{ID: "notice-id", Title: unsafe, Body: unsafe, Href: "/n/town/trades/trade-id", Unread: true, When: stamp},
			{ID: "read-id", Title: "Already read", Body: "Kept notification", When: stamp},
		},
		Disputes: []awaitingDispute{{ID: "dispute-id", Category: unsafe, Opened: stamp}},
	}})
	wantInboxReportText(t, body, "Waiting for your action", "Notifications", "1,000 unread.",
		`href="/n/town/inbox">Show all</a>`, `href="/n/town/inbox?show=unread&after=next-id"`,
		`href="/n/town/disputes/dispute-id"`, "A dispute was opened on a trade with you ("+unsafe+")",
		`<p>Opened `+string(stamp)+". You can add your response.</p>",
		`class="tag">Unread</span>`, `class="tag">Read</span>`, "Already read", "Kept notification",
		`action="/n/town/inbox/read"`, `name="csrf" value="csrf-value"`, `name="id" value="notice-id"`,
		">Mark all as read</button>", `aria-label="Mark as read: `+unsafe+`">Mark as read</button>`)
	if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
		t.Fatal("inbox content or an action name crossed the HTML escaping boundary")
	}
	if strings.Contains(body, `name="id" value="read-id"`) {
		t.Fatal("a read notification gained a mark-read form")
	}
	for _, tc := range []struct {
		count int
		want  string
	}{{0, "No unread notifications."}, {1, "1 unread."}} {
		body = renderInboxReportPage(t, "inbox", &page{Slug: "town", D: &inboxData{Unread: tc.count}})
		wantInboxReportText(t, body, tc.want, "Nothing is waiting for you. Trade proposals sent to you and disputes you can answer will appear here.",
			"No notifications to show.", `href="/n/town/inbox?show=unread">Show unread only</a>`)
		if tc.count == 0 && strings.Contains(body, `action="/n/town/inbox/read"`) {
			t.Fatal("an empty inbox gained a mark-read form")
		}
	}
}

// UX-047, FR-024, SEC-016: target names and retained report details are escaped,
// while category values, CSRF, idempotency and form constraints stay unchanged.
func TestReportFormCatalogue(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x onerror=alert(1)>`
	for _, typ := range []string{"listing", "member"} {
		label := "Listing: " + unsafe
		if typ == "member" {
			label = "Member: " + unsafe
		}
		body := renderInboxReportPage(t, "report_form", &page{Slug: "town", CSRF: "csrf-value", Idem: "idem-value",
			V: map[string]string{"category": "scam", "note": unsafe},
			D: &reportFormData{Type: typ, ID: "target-id", Label: label, Cats: reportCats}})
		wantInboxReportText(t, body, "You are reporting: <strong>"+label+"</strong>.",
			"A report goes to the network administrators. It does not hide, suspend or change anything by itself, and the person you report is not told who reported.",
			`action="/n/town/report" novalidate`, `name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`name="type" value="`+typ+`"`, `name="id" value="target-id"`,
			`<label for="f-category">What is the problem?</label>`, `<option value="scam" selected>Looks like a scam</option>`,
			`<label for="f-note">Details (optional)</label>`, "Up to 500 characters.", `maxlength="500"`,
			unsafe+"</textarea>", ">Send report</button>", `href="/n/town">Cancel</a>`)
		if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
			t.Fatal("report target or retained note was not escaped")
		}
	}
}

func TestReportCatalogueCategoryValues(t *testing.T) {
	want := []option{{"spam", "Spam or advertising"}, {"scam", "Looks like a scam"}, {"inappropriate", "Inappropriate content"},
		{"harassment", "Harassment or abuse"}, {"other", "Something else"}}
	if len(reportCats) != len(want) {
		t.Fatal("report category choices changed")
	}
	for i, option := range want {
		if reportCats[i] != option || categoryOf(option.Value) != option.Label {
			t.Fatalf("report category %d changed: %#v", i, reportCats[i])
		}
	}
}

// UX-047, FR-024: only fixed category/type labels enter Around. Arbitrary target
// names remain template data; unknown enums use a complete plain-text fallback.
func TestReportCatalogueSubjectEnumProvenance(t *testing.T) {
	const unsafe = "\x00%!" + ` "><img src=x> Zoë 100% $&`
	for _, category := range reportCats {
		for _, typ := range []string{"listing", "member"} {
			parts, plain := reportAbout(category.Value, typ, unsafe)
			if plain != "" || parts.Before != category.Label+" about "+typ+" " || parts.After != "" {
				t.Fatalf("known enum labels did not preserve the target slot: %#v, %q", parts, plain)
			}
		}
	}
	for _, tc := range []struct{ category, typ string }{
		{unsafe, "member"}, {"scam", unsafe}, {unsafe, unsafe},
	} {
		parts, plain := reportAbout(tc.category, tc.typ, unsafe)
		if parts.Before != "" || parts.After != "" || plain != categoryOf(tc.category)+" about "+tc.typ+" "+unsafe {
			t.Fatalf("unknown enum values did not remain plain text: %#v, %q", parts, plain)
		}
		body := renderInboxReportPage(t, "admin_reports", &page{Slug: "town", D: &reportsData{
			Rows: []reportRow{{PlainAbout: plain}},
		}})
		if strings.Contains(body, "<img") || !strings.Contains(body, template.HTMLEscapeString(plain)) {
			t.Fatal("unknown report values were not escaped")
		}
	}
}

// UX-047, FR-024, SEC-016: open/handled queues retain their controls, target
// markup, timestamps and complete accessible action names without trusting text.
func TestReportQueueCatalogue(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x onerror=alert(1)>` + "\x00%!"
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	open := reportRow{ID: "open-id", Reporter: unsafe, Target: unsafe, TargetHref: "/n/town/members/member-id", Note: unsafe, Open: true, Created: stamp}
	open.About, open.PlainAbout = reportAbout("scam", "member", open.Target)
	handled := reportRow{ID: "handled-id", Reporter: unsafe, Target: unsafe, HandlingNote: unsafe, Created: stamp}
	handled.About, handled.PlainAbout = reportAbout("spam", "listing", handled.Target)
	body := renderInboxReportPage(t, "admin_reports", &page{Slug: "town", CSRF: "csrf-value", D: &reportsData{
		Rows: []reportRow{open, handled}, Filter: "all", Next: "next-id",
	}})
	wantInboxReportText(t, body,
		"Reports from members about listings, members and transactions. Handling a report here only records that you looked at it; use the ordinary member, policy and dispute tools to act. The reporter is shown only to administrators who are not the report subject.",
		`aria-label="Report filter"`, `href="/n/town/admin/reports?status=all" aria-current="page">All</a>`,
		`class="tag">Open</span>`, `class="tag">Handled</span>`,
		`Looks like a scam about member <a href="/n/town/members/member-id">`, `Spam or advertising about listing <strong>`,
		`action="/n/town/admin/reports/open-id/handle"`, `name="csrf" value="csrf-value"`,
		`<label for="n-open-id">Note (optional, administrators only)</label>`, `id="n-open-id" name="note" maxlength="500"`,
		`href="/n/town/admin/reports?status=all&after=next-id"`)
	for _, escaped := range []string{
		template.HTMLEscapeString(unsafe) + "</a>",
		template.HTMLEscapeString(unsafe) + "</strong>",
		"Reported by " + template.HTMLEscapeString(unsafe) + " &middot; " + string(stamp),
		`aria-label="` + template.HTMLEscapeString("Mark handled: report from "+unsafe) + `">Mark handled</button>`,
		"Note: " + template.HTMLEscapeString(unsafe),
	} {
		if !strings.Contains(body, escaped) {
			t.Errorf("queue lacks escaped complete message %q", escaped)
		}
	}
	if strings.Contains(body, "<img") || strings.Contains(body, "/reports/handled-id/handle") {
		t.Fatal("queue trusted member text or gave a handled report another handle form")
	}
	for _, filter := range []string{"open", "handled", "all"} {
		body = renderInboxReportPage(t, "admin_reports", &page{Slug: "town", D: &reportsData{Filter: filter}})
		wantInboxReportText(t, body, "No reports to show.", `aria-current="page"`)
		if strings.Contains(body, "/handle\"") {
			t.Fatal("an empty report queue gained a handling form")
		}
	}
}
