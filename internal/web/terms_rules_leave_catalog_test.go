package web

import (
	"bytes"
	"html"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func renderGovCatalogue(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantGovCatalogue(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(body, "<img") || strings.Contains(body, "<script>") {
		t.Fatal("member text became markup")
	}
}

// UX-047, GOV-009, SEC-016: version IDs, safe return fields and consent gates
// remain exact while authored terms and retained editor values stay escaped.
func TestTermsPagesCatalogue(t *testing.T) {
	const unsafe = `"><img src=x> & Zoë 100% $& </textarea>` + "\x00%!"
	when := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	terms := &domain.Terms{Version: 1000, Body: unsafe + "\r\n\r\nSecond paragraph.", PublishedAt: when}
	for _, accept := range []bool{false, true} {
		p := &page{Slug: "town", CSRF: "csrf-token", D: &termsData{Terms: terms, Paras: paragraphs(terms.Body), Accepted: 1000,
			NeedsAcceptance: accept, Next: "/n/town/trades/new?to=member-id&keep=1"}}
		if accept {
			p.Summary = messages.Text("terms.accept_required")
			p.Errors = []fieldErr{{"accept", messages.Text("terms.accept_field")}}
		}
		body := renderGovCatalogue(t, "terms", p)
		wantGovCatalogue(t, body, "Version 1000, published "+string(fmtTime(when))+".", "Terms text", "Second paragraph.", `href="/n/town/policy"`)
		if !strings.Contains(body, template.HTMLEscapeString(unsafe)) || strings.Contains(body, "1,000") {
			t.Fatal("terms text or version ID changed")
		}
		if accept {
			wantGovCatalogue(t, body, `action="/n/town/terms"`, `name="csrf" value="csrf-token"`, `name="version" value="1000"`,
				`name="next" value="/n/town/trades/new?to=member-id&keep=1"`, `name="accept" value="1"`, `href="#f-accept"`, `id="e-accept"`,
				"I have read and accept version 1000 of these terms", "You can read your balance and history without accepting, but you must accept before you propose or accept a trade.", "Accept these terms")
		} else if strings.Contains(body, `name="accept"`) || !strings.Contains(body, "You accepted version 1000.") {
			t.Fatal("accepted terms gained another acceptance form")
		}
	}
	for _, current := range []*domain.Terms{nil, terms} {
		body := renderGovCatalogue(t, "admin_terms", &page{Slug: "town", CSRF: "csrf-token", V: map[string]string{"body": unsafe}, D: &termsData{Terms: current}})
		wantGovCatalogue(t, body, `action="/n/town/admin/terms"`, `name="csrf" value="csrf-token"`, `for="f-body">Terms text</label>`,
			`name="body" rows="4" maxlength="20000" aria-describedby="h-body" required`, "Up to 20000 characters. Leave a blank line between paragraphs.",
			"Publish new version", `href="/n/town/admin"`)
		if !strings.Contains(body, template.HTMLEscapeString(unsafe)+"</textarea>") {
			t.Fatal("retained terms were changed")
		}
		if current == nil {
			wantGovCatalogue(t, body, "No terms are published yet; joining needs no acceptance.")
		} else {
			wantGovCatalogue(t, body, "Current: version 1000, published "+string(fmtTime(when))+".")
		}
	}
	body := renderGovCatalogue(t, "terms", &page{Slug: "town", D: &termsData{}})
	wantGovCatalogue(t, body, "This network has not published terms yet.")
	if strings.Contains(body, `name="accept"`) {
		t.Fatal("unpublished terms gained an acceptance form")
	}
}

// UX-047, GOV-002, GOV-003: exact amounts and sorted change rows keep their
// current policy meaning; timestamps and raw reasons never enter string slots.
func TestRulesCatalogueRowsAndStates(t *testing.T) {
	const unsafe = `"><img src=x> & Zoë 100% $&` + "\x00%!"
	when := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	later := when.Add(14 * 24 * time.Hour)
	rows := ruleRows([]service.PolicyLogEntry{{Version: 1000, AnnouncedAt: when, EffectiveAt: later, Pending: true, NoticeDays: 14,
		Reason: unsafe, NoticeReason: unsafe, Changes: map[string]any{"signup_grant": int64(9007199254740993), "custom": unsafe, "enabled": true},
		Previous: map[string]any{"signup_grant": int64(-9007199254740993), "enabled": false}}}, 2)
	wantLines := []string{"custom: " + unsafe, "enabled: no to yes", "Sign-up grant: -90071992547409.93 to 90071992547409.93"}
	if len(rows[0].Lines) != len(wantLines) {
		t.Fatal("change rows were lost")
	}
	for i, want := range wantLines {
		if rows[0].Lines[i] != want {
			t.Fatalf("change row %d: got %q, want %q", i, rows[0].Lines[i], want)
		}
	}
	rows = append(rows, ruleRow{Version: 1001, Pending: true, Announced: fmtTime(when), Effective: fmtTime(later)},
		ruleRow{Version: 1002, Pending: true, NoticeDays: 1, Announced: fmtTime(when), Effective: fmtTime(later)},
		ruleRow{Version: 999, Withdrawn: true, Announced: fmtTime(when)},
		ruleRow{Version: 998, Announced: fmtTime(when), Effective: fmtTime(later)}, ruleRow{Version: 1, Init: true})
	for _, canLeave := range []bool{false, true} {
		body := renderGovCatalogue(t, "rules", &page{Slug: "town", D: &rulesData{Rows: rows, Terms: &domain.Terms{Version: 1000}, CanLeave: canLeave}})
		wantGovCatalogue(t, body, "Read the network terms (version 1000)", "Version 1000", "Scheduled", "Cancelled before it took effect", "Initial rules when the network was created.",
			"Announced "+string(fmtTime(when))+"; takes effect "+string(fmtTime(later))+" (14 days' notice).",
			"Announced "+string(fmtTime(when))+"; takes effect "+string(fmtTime(later))+" (1 day's notice).",
			"Announced "+string(fmtTime(when))+"; takes effect "+string(fmtTime(later))+".",
			"Announced "+string(fmtTime(when))+"; in force from "+string(fmtTime(later))+".", "Announced "+string(fmtTime(when))+".")
		for _, prefix := range []string{"Reason: ", "Shorter notice because: "} {
			if !strings.Contains(body, prefix+template.HTMLEscapeString(unsafe)) {
				t.Fatal("rule reason was not escaped")
			}
		}
		if strings.Contains(body, `href="/n/town/leave"`) != canLeave || strings.Contains(body, "1,000") {
			t.Fatal("leave gate or version ID changed")
		}
	}
	body := renderGovCatalogue(t, "rules", &page{Slug: "town", D: &rulesData{}})
	wantGovCatalogue(t, body, "This network has not published terms.", "Change log")

	t.Run("policy quantities use catalogue formatting", func(t *testing.T) {
		for _, tc := range []struct {
			value any
			want  string
		}{
			{int(0), "0"}, {int(1), "1"}, {int(1000), "1,000"}, {int(3650), "3,650"},
			{int64(0), "0"}, {int64(1), "1"}, {int64(1000), "1,000"}, {int64(3650), "3,650"},
		} {
			if got := fmtChangeValue("inactivity_days", tc.value, 2); got != tc.want {
				t.Errorf("quantity %T(%v): got %q, want %q", tc.value, tc.value, got, tc.want)
			}
		}
		rows := ruleRows([]service.PolicyLogEntry{{Version: 1000,
			Changes:  map[string]any{"inactivity_days": 3650, "dispute_window_days": 1000},
			Previous: map[string]any{"inactivity_days": 1000},
		}}, 2)
		body := renderGovCatalogue(t, "rules", &page{Slug: "town", D: &rulesData{Rows: rows}})
		wantGovCatalogue(t, body, "Version 1000", label("inactivity_days")+": 1,000 to 3,650",
			label("dispute_window_days")+": 1,000")
		if strings.Contains(body, "Version 1,000") {
			t.Fatal("version identifier was formatted as a quantity")
		}
	})
}

// UX-047, FR-014, GOV-010: blockers still suppress departure while export stays
// first; fixed-point balances, confirmation fields and retained reasons survive.
func TestLeaveCataloguePreviewAndControls(t *testing.T) {
	const unsafe = `"><img src=x> & Zoë 100% $&` + "\x00%!"
	for _, rule := range []domain.DepartureRule{domain.DepartRetain, domain.DepartSettleToZero} {
		for _, count := range []int{0, 1, 2} {
			for _, blocked := range []bool{false, true} {
				pv := &service.DepartPreview{Rule: rule, Balance: -9007199254740993, Decimals: 2, OpenProposals: count}
				if blocked {
					pv.Blockers = []string{unsafe}
				}
				p := &page{Slug: "town", NetName: unsafe, CSRF: "csrf-token", D: pv, V: map[string]string{"reason": unsafe},
					Summary: messages.Text("leave.confirm_required"), Errors: []fieldErr{{"confirm", messages.Text("leave.confirm_error")}}}
				body := renderGovCatalogue(t, "leave", p)
				wantGovCatalogue(t, body, "Your balance is −90071992547409.93.", `action="/n/town/export"`, `name="csrf" value="csrf-token"`,
					`for="f-format">File format</label>`, `<option value="csv">CSV (spreadsheet)</option>`, `<option value="json">JSON</option>`,
					"Download your data first", "Download my data", `href="/n/town/history">History page</a>`, "You cannot rejoin this network with the same account afterwards.")
				if rule == domain.DepartSettleToZero {
					wantGovCatalogue(t, body, "When you leave, your balance is settled to zero by a recorded adjustment.")
				} else {
					wantGovCatalogue(t, body, "It stays on your closed account and your history is kept.")
				}
				switch count {
				case 0:
					wantGovCatalogue(t, body, "You have no open trade proposals.")
				case 1:
					wantGovCatalogue(t, body, "1 open trade proposal involving you will be cancelled, and the other members are told.")
				default:
					wantGovCatalogue(t, body, "2 open trade proposals involving you will be cancelled, and the other members are told.")
				}
				leaveAt := strings.Index(body, `action="/n/town/leave"`)
				if blocked {
					wantGovCatalogue(t, body, "You cannot leave yet:", "Resolve these first")
					if leaveAt >= 0 || strings.Contains(body, `name="confirm"`) {
						t.Fatal("blocked member received a leave form")
					}
				} else {
					if leaveAt < 0 || strings.Index(body, `action="/n/town/export"`) > leaveAt {
						t.Fatal("export is not offered before leaving")
					}
					wantGovCatalogue(t, body, `name="confirm" value="1"`, `href="#f-confirm"`, `id="e-confirm"`, `for="f-reason">Reason (optional)</label>`, `maxlength="500"`, `href="/n/town">Cancel</a>`)
					if !strings.Contains(body, `name="reason" type="text" value="`+template.HTMLEscapeString(unsafe)+`"`) {
						t.Fatal("retained reason changed")
					}
				}
				if !strings.Contains(body, template.HTMLEscapeString(unsafe)) {
					t.Fatal("network name or blocker lost escaping")
				}
			}
		}
	}
}

func TestTermsRulesLeaveCatalogueEscapesMessages(t *testing.T) {
	const unsafe = `"><img src=x> & catalogue`
	for _, tc := range []struct {
		name, want string
		data       any
	}{
		{"terms", `class="sr">`, &termsData{Terms: &domain.Terms{Version: 1}, NeedsAcceptance: true}},
		{"admin_terms", `for="f-body">`, &termsData{}},
		{"rules", `<h2>`, &rulesData{}},
		{"leave", `for="f-format">`, &service.DepartPreview{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tpl := mustLoadTemplates()[tc.name].Funcs(template.FuncMap{"msg": func(key string, args ...any) string {
				if strings.HasPrefix(key, "terms.") || strings.HasPrefix(key, "rules.") || strings.HasPrefix(key, "leave.") {
					return unsafe
				}
				return messages.Text(key, args...)
			}})
			var out bytes.Buffer
			if err := tpl.ExecuteTemplate(&out, "base", &page{Slug: "town", D: tc.data}); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "<img") || !strings.Contains(out.String(), tc.want+template.HTMLEscapeString(unsafe)) {
				t.Fatal("page ignored catalogue text or trusted its markup")
			}
		})
	}
}
