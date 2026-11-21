package messages

import (
	"strconv"
	"testing"
)

// UX-047, GOV-003, GOV-025: counts use plural rules; versions, examples,
// confirmation slugs and entered text remain exact string arguments.
func TestPolicyNetworkMessages(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"admin.policy.current", []any{"1000", 2}, "Current policy is version 1000. Saving creates a new version; earlier versions are kept. Decimal places (2) cannot be changed."},
		{"admin.policy.scheduled", []any{"1001", "ANNOUNCED", "EFFECTIVE"}, "Version 1001 was announced ANNOUNCED and takes effect EFFECTIVE. Until then the rules in force are unchanged."},
		{"admin.policy.amount_invalid", []any{0, "Largest single trade", "1250"}, "Largest single trade: enter a number with up to 0 decimal places, for example 1250."},
		{"admin.policy.amount_invalid", []any{1, "Largest single trade", "125.0"}, "Largest single trade: enter a number with up to 1 decimal place, for example 125.0."},
		{"admin.policy.amount_invalid", []any{2, "Largest single trade", "90071992547409.93"}, "Largest single trade: enter a number with up to 2 decimal places, for example 90071992547409.93."},
		{"admin.policy.integer_invalid", []any{"Limit warning (percent of limit used)"}, "Limit warning (percent of limit used): enter a whole number."},
		{"admin.policy.amount_hint", []any{unsafe}, "Amount in " + unsafe + "."},
		{"admin.policy.grant_hint", []any{"Hours"}, "Amount in Hours; 0 for none."},
		{"admin.policy.notice_invalid", nil, "Notice (days): enter a whole number, or leave it empty."},
		{"admin.policy.invalid", nil, "Not saved. Fix the problems below. Nothing changed."},
		{"admin.policy.announced", nil, "Change announced to members. It takes effect after its notice period; see the scheduled change below."},
		{"admin.policy.saved", nil, "Policy saved as a new version."},
		{"admin.network.confirm_invalid", []any{unsafe}, "Confirmation: type " + unsafe + " exactly."},
		{"admin.network.close_invalid", nil, "Not closed. Type the network address name exactly to confirm. Nothing changed."},
		{"admin.network.suspended", nil, "Trading is paused. Members were told. Balances, history and exports stay available."},
		{"admin.network.resumed", nil, "Your suspension is lifted."},
		{"admin.network.closure_announced", nil, "The closure date is announced. Members were told. Trading continues until that date."},
		{"admin.network.closure_cancelled", nil, "The closure is cancelled. Members were told."},
		{"admin.network.category", []any{unsafe}, "Category: " + unsafe + "."},
		{"admin.network.closing", []any{"TIME", unsafe}, "The network closes on TIME: " + unsafe + ". Trading continues until then. On that date open proposals are cancelled and new trades stop. Members keep read and export access afterwards."},
		{"admin.network.members_told", []any{0}, "All 0 active members are told now."},
		{"admin.network.members_told", []any{1}, "1 active member is told now."},
		{"admin.network.members_told", []any{1000}, "All 1,000 active members are told now."},
		{"admin.network.count", []any{0}, "0"},
		{"admin.network.count", []any{1}, "1"},
		{"admin.network.count", []any{1000}, "1,000"},
		{"admin.network.proposals", []any{0, "0"}, "0 open proposals will be cancelled on the closing date."},
		{"admin.network.proposals", []any{1, "1"}, "1 open proposal will be cancelled on the closing date."},
		{"admin.network.proposals", []any{1000, "1,000"}, "1,000 open proposals will be cancelled on the closing date."},
		{"admin.network.nonzero", []any{0, "0"}, "0 accounts have a balance that is not zero. Closing does not settle balances. Tell members how you expect to settle them before you close."},
		{"admin.network.nonzero", []any{1, "1"}, "1 account has a balance that is not zero. Closing does not settle balances. Tell members how you expect to settle them before you close."},
		{"admin.network.nonzero", []any{1000, "1,000"}, "1,000 accounts have a balance that is not zero. Closing does not settle balances. Tell members how you expect to settle them before you close."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPolicyNetworkMarkupSlots(t *testing.T) {
	for _, version := range []int{1, 1000, int(^uint(0) >> 1)} {
		parts := AroundTwoVersion("admin.policy.scheduled", version)
		if parts.Before != "Version "+strconv.Itoa(version)+" was announced " || parts.Between != " and takes effect " ||
			parts.After != ". Until then the rules in force are unchanged." {
			t.Fatalf("version or timestamp slots changed: %#v", parts)
		}
	}
	for _, count := range []int{0, 1, 1000} {
		for _, key := range []string{"admin.network.proposals", "admin.network.nonzero"} {
			parts := AroundCount(key, count)
			if parts.Before+Text("admin.network.count", count)+parts.After != Text(key, count, Text("admin.network.count", count)) {
				t.Fatalf("count or emphasis slot changed for %q", key)
			}
		}
	}
	for name, call := range map[string]func(){
		"count without a slot": func() { AroundCount("admin.network.count", 1) },
		"count with two slots": func() { AroundCount("admin.network.suspension", 1) },
		"version without slots": func() {
			AroundTwoVersion("admin.policy.title", 1000)
		},
		"version with too few arguments": func() {
			AroundTwoVersion("admin.network.suspension", 1000)
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid message slots were accepted")
				}
			}()
			call()
		})
	}
}

func TestPolicyNetworkNoticeDays(t *testing.T) {
	const after = " Until the date, trading continues. On the date, open proposals are cancelled and new trades, memberships, adjustments and grants stop for good. History stays readable and exports stay available for 90 days."
	for _, tc := range []struct {
		days int
		want string
	}{
		{0, "Closing is announced at least 0 days ahead."},
		{1, "Closing is announced at least 1 day ahead."},
		{30, "Closing is announced at least 30 days ahead."},
		{1000, "Closing is announced at least 1,000 days ahead."},
	} {
		if got := Text("admin.network.close_explanation", tc.days); got != tc.want+after {
			t.Fatalf("notice count %d: got %q", tc.days, got)
		}
	}
}
