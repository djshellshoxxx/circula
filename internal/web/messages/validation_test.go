package messages

import "testing"

// UX-047: validation templates keep complete sentences, exact amount strings,
// positional arguments and service-provided details without changing wording.
func TestValidationMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"validation.label.related_transaction_id", nil, "Related trade reference"},
		{"validation.detail.choice_required", nil, "nothing was chosen"},
		{"validation.detail.choice_invalid", nil, "that choice is not on the list"},
		{"validation.choice.related_transaction_id", nil, "Choose a posted trade involving this member, or leave the reference empty."},
		{"validation.field.message", []any{"Amount", "not a number", "Enter digits only, for example 12.50."}, "Amount: not a number. Enter digits only, for example 12.50."},
		{"validation.field.message", []any{"Name", "Zoë <img src=x> 100%", "Check the value and try again."}, "Name: Zoë <img src=x> 100%. Check the value and try again."},
		{"validation.fix.decimals", []any{0}, "Use at most 0 decimal places and try again."},
		{"validation.fix.decimals", []any{1}, "Use at most 1 decimal place and try again."},
		{"validation.fix.decimals", []any{2}, "Use at most 2 decimal places and try again."},
		{"validation.fix.number", []any{"90071992547409.93"}, "Enter digits only, for example 90071992547409.93."},
		{"validation.refusal.insufficient_credit", nil, "Not sent. This trade would take your available credit below zero. Try a smaller amount. Nothing changed."},
		{"validation.refusal.insufficient_credit_available", []any{"90071992547409.93"}, "Not sent. This trade would take your available credit below zero. Available to spend is 90071992547409.93. Try a smaller amount. Nothing changed."},
		{"validation.refusal.amount_exceeds", []any{"90071992547409.93"}, "Not sent. The amount is above the largest single trade this network allows (90071992547409.93). Nothing changed."},
		{"validation.refusal.blocked", []any{"first <img src=x>; second"}, "Not done. first <img src=x>; second. Nothing changed."},
		{"validation.refusal.fields", nil, "Not saved. Fix the problems below. Nothing changed."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAdminResetMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"admin.reset.action", nil, "Reset password"},
		{"admin.reset.confirm_title", []any{"Zoë 100%"}, "Reset password: Zoë 100%?"},
		{"admin.member_action.heading", []any{"Reset password", "Zoë 100%"}, "Reset password: Zoë 100%"},
		{"admin.reset.for_member", []any{"Zoë <img src=x>"}, "Temporary password for Zoë <img src=x>"},
		{"admin.reset.instructions", []any{"Zoë 100%"}, "Give it to Zoë 100% in person or by a channel you trust. They must choose a new password when they sign in. All their sessions and API tokens were ended."},
		{"admin.reset.forbidden", nil, "Not done. This password cannot be reset here. Use Change password for your own account, or ask the platform operator for help. Nothing changed."},
		{"admin.reset.throttled", nil, "Not done. You have reset too many passwords in the last hour. Try again later. Nothing changed."},
		{"admin.reset.shown_once", nil, "This is the only time it is shown."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
