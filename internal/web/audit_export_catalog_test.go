package web

import (
	"bytes"
	"html"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

func renderAuditExportPage(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantAuditExportText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

// FR-082, DAT-033, UX-047: both audit tables retain exact sequence/target
// references, time markup, filters, pagination and escaped row details.
func TestAuditCatalogueDataAndControls(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x onerror=alert(1)>`
	const target = "transaction 018f0000-0000-7000-8000-000000000001"
	const seq = int64(9223372036854775807)
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name, path, caption, region, targetLabel, empty string
	}{
		{"admin_audit", "/admin/audit", "Audit log", "Audit log table", "Target", "No audit events match."},
		{"my_audit", "/audit", "Activity log", "Activity log table", "About", "No events match."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &page{Slug: "town", CSRF: "csrf-value", V: map[string]string{"action": "trade.completed"},
				D: map[string]any{"Rows": []auditRow{{Seq: seq, When: stamp, Actor: unsafe, Action: "trade.completed", Target: target, Reason: unsafe}},
					"Next": "9007199254740993"}}
			body := renderAuditExportPage(t, tc.name, p)
			wantAuditExportText(t, body, `<form method="get" action="/n/town`+tc.path+`">`,
				`<label for="f-action">Filter by action</label>`, `name="action"`, `value="trade.completed"`,
				"For example trade.completed. Leave empty for everything.", ">Filter</button>",
				`role="region" aria-label="`+tc.region+`" tabindex="0"`, "<caption>"+tc.caption+"</caption>",
				`<th scope="col">`+tc.targetLabel+"</th>", `<th scope="row" data-label="No.">9223372036854775807</th>`,
				`<td data-label="When">`+string(stamp)+"</td>", `<td data-label="`+tc.targetLabel+`">`+target+"</td>",
				`href="/n/town`+tc.path+`?action=trade.completed&after=9007199254740993"`)
			if strings.Contains(body, "<img") || strings.Count(body, template.HTMLEscapeString(unsafe)) != 2 {
				t.Fatal("audit actor or reason crossed the HTML escaping boundary")
			}
			if tc.name == "admin_audit" {
				wantAuditExportText(t, body, "Consistency check", "Recomputes balances and checks the audit chain. The run is recorded.",
					`action="/n/town/admin/check"`, `name="csrf" value="csrf-value"`, ">Run consistency check</button>")
			} else {
				wantAuditExportText(t, body, "Events recorded about your account. Other people's names and details are left out.")
				if strings.Contains(body, `action="/n/town/admin/check"`) {
					t.Fatal("member audit view gained the administrator consistency action")
				}
			}
			p.D = map[string]any{"Rows": []auditRow{}, "Next": ""}
			body = renderAuditExportPage(t, tc.name, p)
			wantAuditExportText(t, body, tc.empty)
			if strings.Contains(body, "<table") || strings.Contains(body, "after=") {
				t.Fatal("empty audit view gained a table or continuation")
			}
		})
	}
}

// FR-052, DAT-016: count formatting never changes ledger references or
// service-generated findings, amounts, identifiers and remediation text.
func TestLedgerCheckCatalogueCountsAndFindings(t *testing.T) {
	const seq = int64(9223372036854775807)
	const finding = `account 018f0000-0000-7000-8000-000000000001 cached balance 9007199254740993 != sum of entries 9007199254740992; "><img src=x> 100% $&`
	for _, tc := range []struct {
		count int64
		ok    bool
		want  string
	}{
		{0, true, "Ledger sequence 9223372036854775807; 0 audit events checked."},
		{1, true, "Ledger sequence 9223372036854775807; 1 audit event checked."},
		{1000, false, "Ledger sequence 9223372036854775807; 1,000 audit events checked."},
	} {
		rep := &service.LedgerReport{OK: tc.ok, LedgerSeq: seq, AuditEvents: tc.count}
		status := "Check passed: no problems found."
		if !tc.ok {
			status = "Check failed: problems found."
			rep.Findings = []string{finding}
		}
		body := renderAuditExportPage(t, "admin_check", &page{Slug: "town", D: rep})
		wantAuditExportText(t, body, status, tc.want, `href="/n/town/admin/audit">Back to audit and checks</a>`)
		if !tc.ok && !strings.Contains(body, "<li>"+template.HTMLEscapeString(finding)+"</li>") {
			t.Fatal("service-generated finding was reformatted or not escaped")
		}
		if strings.Contains(body, "<img") {
			t.Fatal("consistency finding became trusted HTML")
		}
	}
}

// FR-091..094, UX-047: only the four administrator scopes and two wire formats
// are offered; the audited POST action and CSRF protection stay intact.
func TestAdminExportsCatalogueControls(t *testing.T) {
	body := renderAuditExportPage(t, "admin_exports", &page{Slug: "town", CSRF: "csrf-value"})
	wantAuditExportText(t, body, "Exports are recorded in the audit log and download as one zip file.",
		`<form method="post" action="/n/town/admin/exports" data-loading-ignore>`, `name="csrf" value="csrf-value"`,
		`<label for="f-scope">What to export</label>`, `<select id="f-scope" name="scope">`,
		`<label for="f-format">File format</label>`, `<select id="f-format" name="format">`,
		">Download export</button>")
	for value, label := range map[string]string{
		"ledger": "Ledger: accounts, trades, entries", "members": "Members and roles", "audit": "Audit history", "full": "Everything",
		"csv": "CSV", "json": "JSON",
	} {
		wantAuditExportText(t, body, `<option value="`+value+`">`+label+"</option>")
	}
	if strings.Count(body, "<option ") != 6 {
		t.Fatal("export scope or format choices changed")
	}
}
