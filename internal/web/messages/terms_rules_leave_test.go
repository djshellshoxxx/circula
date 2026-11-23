package messages

import (
	"strconv"
	"testing"
)

// UX-047, GOV-009, FR-014: identifiers and fixed-point amounts remain literal;
// only actual counts use plural/count formatting.
func TestTermsRulesLeaveMessages(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"terms.accepted_flash", []any{"1000"}, "You accepted version 1000 of the network terms."},
		{"terms.admin.published_flash", []any{"1000"}, "Published version 1000. Members were told and must accept it before their next trade."},
		{"terms.accept_version", []any{"1000"}, "I have read and accept version 1000 of these terms"},
		{"rules.version", []any{"1000"}, "Version 1000"},
		{"rules.quantity", []any{0}, "0"},
		{"rules.quantity", []any{1}, "1"},
		{"rules.quantity", []any{1000}, "1,000"},
		{"rules.quantity", []any{3650}, "3,650"},
		{"rules.quantity", []any{int64(9007199254740993)}, "9,007,199,254,740,993"},
		{"rules.change", []any{"Credit limit", "90071992547409.93", "0.00"}, "Credit limit: 90071992547409.93 to 0.00"},
		{"rules.initial_value", []any{"Name", unsafe}, "Name: " + unsafe},
		{"rules.reason", []any{unsafe}, "Reason: " + unsafe},
		{"rules.shorter_notice", []any{unsafe}, "Shorter notice because: " + unsafe},
		{"leave.balance_retain", []any{"−90071992547409.93"}, "Your balance is −90071992547409.93. It stays on your closed account and your history is kept."},
		{"leave.balance_settle", []any{"−90071992547409.93"}, "Your balance is −90071992547409.93. When you leave, your balance is settled to zero by a recorded adjustment."},
		{"leave.confirm", []any{unsafe}, "I understand and want to leave " + unsafe},
		{"leave.left", []any{unsafe}, "You have left " + unsafe + ". You can still read your history and download your data from the History page."},
		{"leave.proposals", []any{0}, "0 open trade proposals involving you will be cancelled, and the other members are told."},
		{"leave.proposals", []any{1}, "1 open trade proposal involving you will be cancelled, and the other members are told."},
		{"leave.proposals", []any{1000}, "1,000 open trade proposals involving you will be cancelled, and the other members are told."},
		{"terms.changed", nil, "These terms changed while you were reading. Please read the new version."},
		{"terms.accept_required", nil, "Not saved. Tick the box to say you accept these terms. Nothing changed."},
		{"leave.confirm_required", nil, "Not done. Tick the box to confirm that you want to leave. Nothing changed."},
		{"rules.cancelled_flash", nil, "The scheduled change was cancelled. The rules in force stay as they are."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTermsRulesMarkupSlots(t *testing.T) {
	for _, key := range []string{"terms.published", "terms.admin.current"} {
		parts := Around(key, strconv.Itoa(1000))
		if parts.Before+"TIME"+parts.After != Text(key, "1000", "TIME") {
			t.Fatal("version or publication timestamp slot changed")
		}
	}
	for _, tc := range []struct {
		count int
		after string
	}{{0, " (0 days' notice)."}, {1, " (1 day's notice)."}, {14, " (14 days' notice)."}} {
		parts := AroundTwoCount("rules.announced_scheduled_notice", tc.count)
		if parts.Before != "Announced " || parts.Between != "; takes effect " || parts.After != tc.after {
			t.Fatalf("notice sentence changed: %#v", parts)
		}
	}
	for _, key := range []string{"rules.announced_scheduled", "rules.in_force"} {
		parts := AroundTwo(key)
		if parts.Before+"ANNOUNCED"+parts.Between+"EFFECTIVE"+parts.After != Text(key, "ANNOUNCED", "EFFECTIVE") {
			t.Fatal("rule timestamps changed")
		}
	}
	for _, key := range []string{"rules.version", "rules.announced_scheduled", "leave.title"} {
		t.Run("reject counted misuse/"+key, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid counted element message was accepted")
				}
			}()
			AroundTwoCount(key, 14)
		})
	}
}
