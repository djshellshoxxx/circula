package web

import (
	"bytes"
	"html"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func renderPolicyNetworkCatalogue(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func wantPolicyNetworkCatalogue(t *testing.T, body string, wants ...string) {
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

// UX-047, GOV-003: exact policy versions and amounts, scheduled timestamps,
// cancellation, consent and the complete submitted field contract survive.
func TestPolicyCatalogueControls(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	pol := domain.DefaultPolicy()
	pol.DefaultCreditLimit, pol.MaxTxAmount, pol.SignupGrant = 9007199254740993, 9007199254740992, 1
	pol.CurrencyName, pol.JoinMode, pol.DepartureRule = unsafe, domain.JoinOpen, domain.DepartSettleToZero
	pol.DefaultFlagBlocksPayer = true
	v := policyValues(pol)
	v["reason"], v["notice_reason"], v["notice_days"], v["confirm_open_grant_risk"] = unsafe, unsafe, "7", "1"
	when := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	later := when.Add(14 * 24 * time.Hour)
	pending := &ruleRow{Version: 1001, Announced: fmtTime(when), Effective: fmtTime(later), Lines: []string{unsafe}}
	for _, scheduled := range []*ruleRow{nil, pending} {
		p := &page{Slug: "town", CSRF: "csrf-value", Idem: "idem-value", Dec: pol.Decimals, Unit: pol.CurrencyName, V: v,
			D:      &policyPageData{Version: 1000, Pending: scheduled},
			Errors: []fieldErr{{"confirm_open_grant_risk", "Confirmation: " + unsafe}}}
		body := renderPolicyNetworkCatalogue(t, "admin_policy", p)
		wantPolicyNetworkCatalogue(t, body,
			"Current policy is version 1000. Saving creates a new version; earlier versions are kept. Decimal places (2) cannot be changed.",
			`<a href="/n/town/policy">change log</a>`, `<form method="post" action="/n/town/admin/policy" novalidate>`,
			`name="csrf" value="csrf-value"`, `name="idem" value="idem-value"`,
			`name="default_credit_limit" type="text" value="90071992547409.93" inputmode="decimal"`,
			`name="max_tx_amount" type="text" value="90071992547409.92" inputmode="decimal"`,
			`name="signup_grant" type="text" value="0.01" inputmode="decimal"`,
			`<select id="f-join_mode" name="join_mode">`, `<option value="open" selected>Open</option>`,
			`<select id="f-departure_rule" name="departure_rule">`, `<option value="settle-to-zero" selected>Balance is settled to zero</option>`,
			`id="f-confirm_open_grant_risk" name="confirm_open_grant_risk" value="1" checked aria-invalid="true" aria-describedby="e-confirm_open_grant_risk"`,
			`<p class="error" id="e-confirm_open_grant_risk"><span class="sr">Error: </span>`,
			"I accept the risk of open joining together with a grant: everyone who joins receives it, including people who join by themselves, so anyone can collect it again by opening new accounts. Only needed for that combination.",
			`name="default_flag_blocks_payer" value="1" checked`, `name="notice_days" type="text" value="7" inputmode="numeric"`,
			`name="notice_reason" type="text"`, `maxlength="500"`, "Save new version")
		for _, name := range []string{"currency_name", "default_credit_limit", "max_tx_amount", "signup_grant", "proposal_expiry_days", "dispute_window_days", "limit_warning_days", "response_period_days", "re_review_window_days", "limit_warning_percent", "inactivity_days", "reason", "notice_days", "notice_reason"} {
			wantPolicyNetworkCatalogue(t, body, `for="f-`+name+`"`, `id="f-`+name+`" name="`+name+`"`)
		}
		for _, value := range []string{"invite", "approval", "open", "retain", "settle-to-zero"} {
			if strings.Count(body, `<option value="`+value+`"`) != 1 {
				t.Fatalf("policy choice %q changed", value)
			}
		}
		if strings.Count(body, "<option ") != 5 || strings.Contains(body, `name="decimals"`) {
			t.Fatal("policy options or immutable decimal precision changed")
		}
		if !strings.Contains(body, `value="`+template.HTMLEscapeString(unsafe)+`"`) {
			t.Fatal("retained name or reason was not escaped")
		}
		if scheduled == nil {
			if strings.Contains(body, `/admin/policy/cancel`) || strings.Count(body, `name="idem"`) != 1 {
				t.Fatal("unscheduled policy gained a cancellation form")
			}
		} else {
			wantPolicyNetworkCatalogue(t, body, `aria-labelledby="pend-h"`, "Scheduled change",
				"Version 1001 was announced "+string(fmtTime(when))+" and takes effect "+string(fmtTime(later))+". Until then the rules in force are unchanged.",
				`action="/n/town/admin/policy/cancel"`, "Cancel the scheduled change", "While a change is scheduled, make no other policy change: cancel it first.")
			if strings.Count(body, `name="csrf" value="csrf-value"`) != 2 || strings.Count(body, `name="idem" value="idem-value"`) != 2 ||
				!strings.Contains(body, "<li>"+template.HTMLEscapeString(unsafe)+"</li>") {
				t.Fatal("scheduled change data or cancellation protection changed")
			}
		}
		if strings.Contains(body, "1,000") || strings.Contains(body, "1,001") || strings.Contains(body, "90,071,992") {
			t.Fatal("an exact version or monetary value was grouped as a count")
		}
	}
}

// The amount hints use the network's existing unit inflector, including an
// already-plural name, while keeping user-provided unit text escaped.
func TestPolicyCatalogueUnitHints(t *testing.T) {
	for _, unit := range []string{"credit", "Hours", `Zoë 100% $& "><img src=x>` + "\x00%!"} {
		body := renderPolicyNetworkCatalogue(t, "admin_policy", &page{Slug: "town", Dec: 2, Unit: unit, D: &policyPageData{Version: 1}})
		want := template.HTMLEscapeString(unitFor(unit, 0, 2))
		if strings.Count(body, "Amount in "+want+".") != 2 || !strings.Contains(body, "Amount in "+want+"; 0 for none.") {
			t.Errorf("amount hints do not use the inflected network unit %q", unit)
		}
		if strings.Contains(body, "Hourss") || strings.Contains(body, "<img") {
			t.Fatal("amount hint changed or trusted the unit name")
		}
	}
}

// GOV-023, FR-068, GOV-025: these are presentation gates, not authorization.
// The production template retains each current action, CSRF field, required
// reason, closure confirmation and the service-generated blockers as data.
func TestNetworkCatalogueStateControls(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	when := fmtTime(time.Date(2026, time.December, 19, 12, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name    string
		data    networkData
		actions []string
		wants   []string
	}{
		{"active", networkData{}, []string{"suspend", "close"}, []string{"Every member is told, with your reason.", "Reason for pausing", "Reason for closing", "At least 30. Leave empty for 30."}},
		{"administrator pause", networkData{AdminSuspended: true}, []string{"resume", "close"}, []string{"Reason for resuming", "Resume trading"}},
		{"both pauses", networkData{AdminSuspended: true, PlatformHolds: true}, []string{"resume", "close"}, []string{"The platform operator has also paused this network. Only the operator can lift that."}},
		{"platform pause", networkData{PlatformHolds: true}, []string{"close"}, []string{"Only the platform operator can lift the platform's pause."}},
		{"blocked", networkData{Blockers: []string{unsafe}}, []string{"suspend"}, []string{"Closing cannot be announced until these are resolved:", `<a href="/n/town/review">dispute review queue</a>`}},
		{"announced", networkData{Closing: true, CanCancel: true, ClosingAt: when, ClosureReason: unsafe}, []string{"suspend", "close/cancel"}, []string{"Reason for cancelling", "Cancel the closure"}},
		{"awaiting disputes", networkData{Closing: true, ClosingAt: when, ClosureReason: unsafe}, []string{"suspend"}, []string{"The closing date has passed. The network completes closing once every dispute is resolved."}},
		{"closed", networkData{Closed: true, FinalSHA256: strings.Repeat("a012", 16)}, nil, []string{"This network is closed. History stays readable and exports are available to members and administrators.", "Final export checksum (SHA-256): <code>" + strings.Repeat("a012", 16) + "</code>"}},
		{"closed retention", networkData{Closed: true, ReadOnlyUntil: when}, nil, []string{"This network is closed. History stays readable and exports are available to members and administrators until " + string(when) + "."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.data.MinNoticeDays = domain.MinClosureNoticeDays
			if tc.data.AdminSuspended || tc.data.PlatformHolds {
				tc.data.Suspensions = []suspensionRow{{Who: "An administrator", Category: unsafe, Reason: unsafe, At: when}}
			}
			body := renderPolicyNetworkCatalogue(t, "admin_network", &page{Slug: "town", CSRF: "csrf-value", D: &tc.data,
				V: map[string]string{"reason": unsafe, "notice_days": "45", "confirm": "town"}})
			wantPolicyNetworkCatalogue(t, body, tc.wants...)
			for _, action := range []string{"suspend", "resume", "close", "close/cancel"} {
				want := false
				for _, offered := range tc.actions {
					want = want || offered == action
				}
				if strings.Contains(body, `action="/n/town/admin/network/`+action+`"`) != want {
					t.Errorf("action %q visibility changed", action)
				}
			}
			if strings.Count(body, `name="csrf" value="csrf-value"`) != len(tc.actions) || strings.Count(body, `name="reason"`) != len(tc.actions) ||
				strings.Count(body, `maxlength="500" aria-describedby="h-reason" required`) != len(tc.actions) {
				t.Fatal("CSRF or required reason control changed")
			}
			if strings.Contains(body, `action="/n/town/admin/network/close"`) {
				wantPolicyNetworkCatalogue(t, body, `name="notice_days" type="text" value="45" maxlength="3"`,
					`name="confirm" type="text" value="town" maxlength="40" aria-describedby="h-confirm" required`,
					"Type the network address name to confirm", "Members are told at once.", "Announce the closure",
					`href="/n/town/admin/exports">Exports</a>`)
			}
			if tc.data.Closing {
				if !strings.Contains(body, "The network closes on <strong>"+string(when)+"</strong>: "+template.HTMLEscapeString(unsafe)+". Trading continues until then.") {
					t.Fatal("closing timestamp or untrusted reason changed")
				}
			}
			if tc.data.Suspensions != nil {
				if !strings.Contains(body, "<strong>An administrator</strong> paused trading, "+string(when)+". Category: "+template.HTMLEscapeString(unsafe)+".") {
					t.Fatal("suspension timestamp or untrusted category changed")
				}
			}
			if tc.data.Blockers != nil && !strings.Contains(body, "<li>"+template.HTMLEscapeString(unsafe)+"</li>") {
				t.Fatal("service-provided blocker was changed")
			}
		})
	}
}

func TestNetworkCatalogueCounts(t *testing.T) {
	for _, tc := range []struct {
		count     int
		formatted string
		members   string
		proposals string
		accounts  string
	}{
		{count: 0, formatted: "0", members: "All 0 active members are told now."},
		{count: 1, formatted: "1", members: "1 active member is told now.", proposals: "open proposal will be cancelled", accounts: "account has a balance that is not zero"},
		{count: 1000, formatted: "1,000", members: "All 1,000 active members are told now.", proposals: "open proposals will be cancelled", accounts: "accounts have a balance that is not zero"},
	} {
		d := &networkData{MinNoticeDays: 30, Members: tc.count, OpenProposals: tc.count, NonZeroBalances: tc.count}
		body := renderPolicyNetworkCatalogue(t, "admin_network", &page{Slug: "town", D: d})
		wantPolicyNetworkCatalogue(t, body, tc.members, "Closing is announced at least 30 days ahead.")
		if tc.count == 0 {
			if strings.Contains(body, "<strong>0</strong>") || strings.Contains(body, "balance that is not zero") {
				t.Fatal("empty warnings became visible")
			}
		} else {
			wantPolicyNetworkCatalogue(t, body, "<strong>"+tc.formatted+"</strong> "+tc.proposals,
				"<strong>"+tc.formatted+"</strong> "+tc.accounts,
				"Closing does not settle balances. Tell members how you expect to settle them before you close.")
		}
	}
}

func TestNetworkCatalogueSuspensionCategories(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	for _, tc := range []struct{ category, label string }{
		{domain.SuspendLegalOrder, "Legal order"},
		{domain.SuspendSecurityIncident, "Security incident"},
		{domain.SuspendPlatformAbuse, "Platform abuse"},
		{domain.SuspendHostingAgreement, "Hosting agreement"},
		{"future_category", "future_category"},
		{unsafe, unsafe},
		{"", ""},
	} {
		label := networkSuspensionCategory(tc.category)
		if label != tc.label {
			t.Fatalf("suspension category %q became %q, want %q", tc.category, label, tc.label)
		}
		body := renderPolicyNetworkCatalogue(t, "admin_network", &page{Slug: "town", D: &networkData{
			Closed: true, Suspensions: []suspensionRow{{Who: "The platform operator", Category: label, Reason: unsafe}},
		}})
		wantPolicyNetworkCatalogue(t, body, "<strong>The platform operator</strong>")
		if label == "" {
			if strings.Contains(body, "Category:") {
				t.Fatal("an empty category gained a visible label")
			}
		} else if !strings.Contains(body, "Category: "+template.HTMLEscapeString(tc.label)+".") {
			t.Fatal("category text changed or became trusted markup")
		}
	}
}

// A catalogue entry remains plain text even around template-owned timestamps
// and emphasized counts. This also catches a template ignoring its catalogue.
func TestPolicyNetworkCatalogueEscapesMessages(t *testing.T) {
	const unsafe = `CATALOGUE "><img src=x> & 100% $&` + "\x00%!"
	stamp := fmtTime(time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name, key string
		data      any
	}{
		{"admin_policy", "admin.policy.save", &policyPageData{Version: 1000, Pending: &ruleRow{Version: 1001, Announced: stamp, Effective: stamp}}},
		{"admin_network", "admin.network.announce", &networkData{MinNoticeDays: 30, Members: 1, OpenProposals: 1, NonZeroBalances: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := mustLoadTemplates()[tc.name].Funcs(template.FuncMap{
				"msg": func(key string, args ...any) string {
					if key == tc.key {
						return unsafe
					}
					return messages.Text(key, args...)
				},
				"msgAroundCount": func(key string, count int) messages.Parts {
					parts := messages.AroundCount(key, count)
					parts.Before, parts.After = unsafe+parts.Before, parts.After+unsafe
					return parts
				},
				"msgAroundTwoVersion": func(key string, version int) messages.TwoParts {
					parts := messages.AroundTwoVersion(key, version)
					parts.Before, parts.Between, parts.After = unsafe+parts.Before, unsafe+parts.Between, parts.After+unsafe
					return parts
				},
			})
			var out bytes.Buffer
			if err := tmpl.ExecuteTemplate(&out, "base", &page{Slug: "town", CSRF: "csrf-value", Idem: "idem-value", Unit: "Hours", Dec: 2, D: tc.data}); err != nil {
				t.Fatal(err)
			}
			body, escaped := out.String(), template.HTMLEscapeString(unsafe)
			if !strings.Contains(body, ">"+escaped+"</button>") || strings.Count(body, escaped) < 4 || strings.Contains(body, "<img") {
				t.Fatal("catalogue labels or surrounding parts were ignored or became trusted markup")
			}
			if tc.name == "admin_policy" && strings.Count(body, string(stamp)) != 2 {
				t.Fatal("timestamp markup was lost")
			}
			if tc.name == "admin_network" && strings.Count(body, "<strong>1</strong>") != 2 {
				t.Fatal("count emphasis was lost")
			}
		})
	}
}
