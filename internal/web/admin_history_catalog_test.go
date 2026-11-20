package web

import (
	"bytes"
	"html"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func renderAdminHistoryCatalogue(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantAdminHistoryCatalogue(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(body, "<img") || strings.Contains(body, "<script>") {
		t.Fatal("entered text became markup")
	}
}

// API-064: the editor retains its scheduling/replacement controls, exact
// announcement identifiers, CSRF/idempotency and append-only/delete actions.
func TestAnnouncementsCatalogueControls(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x> </textarea>` + "\x00%!"
	const currentID = "018f0000-0000-7000-8000-000000000001"
	const futureID = "018f0000-0000-7000-8000-000000000002"
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	for _, populated := range []bool{false, true} {
		d := &annAdminData{}
		if populated {
			d.Rows = []annAdminRow{{annRow{ID: currentID, Title: unsafe, Body: unsafe, At: stamp}, false},
				{annRow{ID: futureID, Title: "Next month", Body: "Future notice", At: stamp}, true}}
			d.Options = []option{{Value: currentID, Label: unsafe}, {Value: futureID, Label: "Next month"}}
		}
		p := &page{Slug: "town", CSRF: "csrf-value", Idem: "idem-value", D: d,
			V: map[string]string{"title": unsafe, "body": unsafe, "days": "365", "supersedes_id": currentID}}
		body := renderAdminHistoryCatalogue(t, "admin_announcements", p)
		wantAdminHistoryCatalogue(t, body, "Announcements appear on every member's home page from their start time.",
			"They cannot be edited; delete one and publish a new one to correct it. Deleted announcements stay in the audit log.",
			`<form method="post" action="/n/town/admin/announcements" novalidate>`,
			`name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`<label for="f-title">Title</label>`, `maxlength="120" aria-describedby="h-title" required`,
			`<label for="f-body">Message</label>`, `name="body" rows="4" maxlength="4000" aria-describedby="h-body" required`,
			`<label for="f-days">Start in (days)</label>`, `name="days" type="text" value="365" inputmode="numeric" aria-describedby="h-days"`,
			"Optional. Leave empty to publish now.", ">Publish announcement</button>", "Current announcements")
		if !strings.Contains(body, `value="`+template.HTMLEscapeString(unsafe)+`"`) || !strings.Contains(body, template.HTMLEscapeString(unsafe)+"</textarea>") {
			t.Fatal("retained announcement text changed or was not escaped")
		}
		if strings.Count(body, `name="idem"`) != 1 || strings.Contains(body, `name="effective_at"`) {
			t.Fatal("announcement idempotency or scheduling field contract changed")
		}
		if !populated {
			wantAdminHistoryCatalogue(t, body, "No announcements yet.")
			if strings.Contains(body, `name="supersedes_id"`) || strings.Contains(body, "/delete") {
				t.Fatal("empty announcement page gained replacement/delete controls")
			}
			continue
		}
		wantAdminHistoryCatalogue(t, body, `<label for="f-supersedes_id">Replaces an earlier announcement</label>`,
			`<select id="f-supersedes_id" name="supersedes_id">`, `<option value="">None</option>`,
			`<option value="`+currentID+`" selected>`, " · since "+string(stamp), " · starts "+string(stamp),
			`action="/n/town/admin/announcements/`+currentID+`/delete"`, `action="/n/town/admin/announcements/`+futureID+`/delete"`)
		if strings.Count(body, `name="csrf" value="csrf-value"`) != 3 || strings.Count(body, `class="secondary"`) != 2 {
			t.Fatal("delete controls or CSRF protection changed")
		}
	}
}

func TestAnnouncementCatalogueDeleteNames(t *testing.T) {
	const title = `Zoë 100% $& "><img src=x>` + "\x00%!"
	body := renderAdminHistoryCatalogue(t, "admin_announcements", &page{Slug: "town", CSRF: "csrf-value", D: &annAdminData{
		Rows: []annAdminRow{{annRow: annRow{ID: "announcement-exact-id", Title: title}}},
	}})
	wantAdminHistoryCatalogue(t, body, `action="/n/town/admin/announcements/announcement-exact-id/delete"`,
		`name="csrf" value="csrf-value"`)
	if !strings.Contains(body, `aria-label="`+template.HTMLEscapeString("Delete announcement "+title)+`">Delete</button>`) {
		t.Fatal("delete action lost its complete accessible name or trusted the announcement title")
	}
}

// FR-067: zero-count attention rows retain their text but no action link;
// every management route and frozen-state explanation remains available.
func TestAdminOverviewCatalogueControls(t *testing.T) {
	const unsafe = `ledger check: Zoë 100% $& "><img src=x>` + "\x00%!"
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	for _, frozen := range []bool{false, true} {
		d := &overviewData{AsOf: stamp, Frozen: frozen, FrozenSince: stamp, FrozenReason: unsafe,
			Check: "Check passed: no problems found.", CheckAt: stamp,
			Rows: []attentionRow{{0, "0 membership requests are waiting.", "/zero-count-action", "Invisible action"},
				{1, "1 dispute is open or under review.", "/one-count-action", "Review one"},
				{1000, "1,000 open proposals have waited at least 48 hours for an answer.", "/many-count-action", "Review many"}}}
		body := renderAdminHistoryCatalogue(t, "admin_home", &page{Slug: "town", D: d})
		wantAdminHistoryCatalogue(t, body, "Needs attention", "Counted from the ledger and the dispute records when this page loaded, "+string(stamp)+".",
			"0 membership requests are waiting.", `href="/one-count-action">Review one</a>`, `href="/many-count-action">Review many</a>`,
			"1,000 open proposals have waited at least 48 hours for an answer.", "Manage the network", "Check passed: no problems found. "+string(stamp))
		if strings.Contains(body, "/zero-count-action") || strings.Contains(body, "Invisible action") {
			t.Fatal("zero-count attention row gained an action link")
		}
		for _, path := range []string{"members", "invitations", "policy", "terms", "adjust", "reverse", "announcements", "adjustments/pending", "audit", "network", "exports", "break-glass", "reports"} {
			wantAdminHistoryCatalogue(t, body, `href="/n/town/admin/`+path+`"`)
		}
		wantAdminHistoryCatalogue(t, body, `href="/n/town/review">Dispute review queue</a>`, `href="/help#support-boundaries">Support boundaries</a>`)
		if frozen {
			wantAdminHistoryCatalogue(t, body, `role="alert" aria-labelledby="frz-h"`, "This network is read-only",
				"Frozen since "+string(stamp)+". Members can read balances and history, but every change is refused (status 423) until the ledger check passes again.")
			if !strings.Contains(body, "Reason: "+template.HTMLEscapeString(unsafe)) {
				t.Fatal("frozen reason was changed or not escaped")
			}
		} else if strings.Contains(body, "frz-h") || strings.Contains(body, unsafe) {
			t.Fatal("unfrozen overview gained a frozen notice")
		}
	}
}

func TestStaleQueueCatalogueCountsAndControls(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	const id = "018f0000-0000-7000-8000-000000000001"
	const next = "018f0000-0000-7000-8000-000000000050"
	when := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	stamp := fmtTime(when)
	for _, tc := range []struct {
		hours int
		words string
	}{{0, "0 hours"}, {1, "1 hour"}, {48, "48 hours"}, {1000, "1,000 hours"}} {
		for _, populated := range []bool{false, true} {
			d := map[string]any{"Hours": tc.hours, "AsOf": stamp, "Rows": []txRow{}, "Next": ""}
			if populated {
				d["Rows"] = []txRow{{ID: id, Text: unsafe, Signed: "90071992547409.93", Status: "Waiting for an answer", When: when}}
				d["Next"] = next
			}
			body := renderAdminHistoryCatalogue(t, "admin_stale_proposals", &page{Slug: "town", D: d})
			wantAdminHistoryCatalogue(t, body, "Open proposals waiting at least "+tc.words+" for an answer, as of "+string(stamp)+". Expired proposals are excluded.",
				`href="/n/town/admin">Administrator overview</a>`)
			if populated {
				wantAdminHistoryCatalogue(t, body, `href="/n/town/trades/`+id+`"`, "90071992547409.93", "Waiting for an answer",
					`href="/n/town/admin/proposals/stale?after=`+next+`"`)
				if !strings.Contains(body, template.HTMLEscapeString(unsafe)) || strings.Contains(body, "90,071,992") {
					t.Fatal("queue data was reinterpreted or the exact amount was grouped")
				}
			} else {
				wantAdminHistoryCatalogue(t, body, "No proposals are waiting this long.", `href="/n/town/admin">Return to the administrator overview</a>`)
				if strings.Contains(body, "?after=") {
					t.Fatal("empty queue gained a continuation")
				}
			}
		}
	}
}

// SEC-013 b: this page is a read-only history, with stored operator/reason
// data kept literal and an exact continuation reference. It offers no grant,
// close or extension action, including after a session expires.
func TestBreakGlassCatalogueReadOnlyControls(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	const next = "018f0000-0000-7000-8000-000000000050"
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	rows := []breakGlassRow{
		{ID: "not-displayed-id", Operator: unsafe, Reason: unsafe, Status: "open", Opened: stamp, Expires: stamp},
		{Operator: "Operator 2", Reason: "Stored reason", Status: "closed", Opened: stamp, Expires: stamp, Closed: stamp, ClosedReason: unsafe},
		{Operator: "Operator 3", Reason: "Stored reason", Status: "expired", Opened: stamp, Expires: stamp},
		{Operator: "Operator 4", Reason: "Stored reason", Status: unsafe, Opened: stamp, Expires: stamp},
	}
	body := renderAdminHistoryCatalogue(t, "admin_break_glass", &page{Slug: "town", D: map[string]any{"Rows": rows, "Next": next}})
	wantAdminHistoryCatalogue(t, body, "A session lasts at most four hours. You cannot open or close sessions from this page.",
		`role="region" aria-label="Operator access table" tabindex="0"`, "<caption>Operator access sessions, newest first</caption>",
		`href="/n/town/admin/break-glass?after=`+next+`"`, "Not closed")
	for _, label := range []string{"Operator", "Status", "Opened", "Expires", "Closed", "Reason"} {
		wantAdminHistoryCatalogue(t, body, `<th scope="col">`+label+`</th>`, `data-label="`+label+`"`)
	}
	for _, status := range []string{"open", "closed", "expired", unsafe} {
		if !strings.Contains(body, `<td data-label="Status">`+template.HTMLEscapeString(status)+`</td>`) {
			t.Fatal("read-only status changed or was not escaped")
		}
	}
	if !strings.Contains(body, string(stamp)+" ("+template.HTMLEscapeString(unsafe)+")") || strings.Contains(body, "not-displayed-id") || strings.Contains(body, "<form") {
		t.Fatal("stored close reason, identifier visibility or read-only controls changed")
	}
	body = renderAdminHistoryCatalogue(t, "admin_break_glass", &page{Slug: "town", D: map[string]any{"Rows": []breakGlassRow{}, "Next": ""}})
	wantAdminHistoryCatalogue(t, body, "No operator access has been recorded for this network.")
	if strings.Contains(body, "<table") || strings.Contains(body, "?after=") || strings.Contains(body, "<form") {
		t.Fatal("empty access history gained rows, pagination or mutation controls")
	}
}

func TestBreakGlassCatalogueStatusLabels(t *testing.T) {
	for _, status := range []string{"open", "closed", "expired", "future_status", "", `Zoë 100% $& "><img src=x>` + "\x00%!"} {
		label := breakGlassStatusText(status)
		if label != status {
			t.Fatalf("status %q changed to %q", status, label)
		}
		body := renderAdminHistoryCatalogue(t, "admin_break_glass", &page{Slug: "town", D: map[string]any{
			"Rows": []breakGlassRow{{Status: label}}, "Next": "",
		}})
		if !strings.Contains(body, `<td data-label="Status">`+template.HTMLEscapeString(status)+`</td>`) || strings.Contains(body, "<img") {
			t.Fatal("status fallback was reinterpreted or not escaped")
		}
	}
}

func TestAdminHistoryCatalogueEscapesMessages(t *testing.T) {
	const unsafe = `CATALOGUE "><img src=x> & 100% $&` + "\x00%!"
	for _, tc := range []struct {
		name, key string
		data      any
	}{
		{"admin_announcements", "admin.announcements.publish", &annAdminData{}},
		{"admin_home", "overview.manage_title", &overviewData{}},
		{"admin_stale_proposals", "queue.overview", map[string]any{"Hours": 48, "AsOf": "TIME", "Rows": []txRow{}, "Next": ""}},
		{"admin_break_glass", "admin.break_glass.caption", map[string]any{"Rows": []breakGlassRow{{Status: "open"}}, "Next": ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := mustLoadTemplates()[tc.name].Funcs(template.FuncMap{"msg": func(key string, args ...any) string {
				if key == tc.key {
					return unsafe
				}
				return messages.Text(key, args...)
			}})
			var out bytes.Buffer
			if err := tmpl.ExecuteTemplate(&out, "base", &page{Slug: "town", D: tc.data}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), template.HTMLEscapeString(unsafe)) || strings.Contains(out.String(), "<img") {
				t.Fatal("template ignored its catalogue message or trusted it as markup")
			}
		})
	}
}
