package messages

import (
	"strconv"
	"testing"
)

func TestOnboardingMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"setup.amount.invalid", []any{4, "Default credit limit", "0.1250"}, "Default credit limit: enter a number with up to 4 decimal places, for example 0.1250."},
		{"join.network.title", []any{"Zoë 100%"}, "Join Zoë 100%"},
		{"join.network.invite_only", []any{"Town <swap>"}, "Town <swap> is invitation only. Ask an administrator for an invitation link."},
		{"join.flash.joined", []any{"Zoë 100%"}, "You have joined Zoë 100%."},
		// A terms version identifies a record; it is not a formatted quantity.
		{"join.terms.heading", []any{"10000"}, "Network terms (version 10000)"},
		{"join.throttle.ten_minutes", nil, "Not sent. Too many join attempts. Wait ten minutes and try again. Nothing changed."},
		{"join.throttle.other", nil, "Not sent. Too many join attempts. Wait a while and try again. Nothing changed."},
	} {
		if got := Text(tc.key, tc.args...); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.key, got, tc.want)
		}
	}
}

// UX-047: setup correction sentences use the selected precision's plural form.
func TestSetupAmountValidationMessages(t *testing.T) {
	for _, tc := range []struct {
		precision int
		example   string
		want      string
	}{
		{0, "1250", "enter a number with up to 0 decimal places, for example 1250."},
		{1, "125.0", "enter a number with up to 1 decimal place, for example 125.0."},
		{2, "12.50", "enter a number with up to 2 decimal places, for example 12.50."},
		{4, "0.1250", "enter a number with up to 4 decimal places, for example 0.1250."},
	} {
		for _, label := range []string{"Default credit limit", "Sign-up grant"} {
			t.Run(label+"/"+strconv.Itoa(tc.precision), func(t *testing.T) {
				want := label + ": " + tc.want
				if got := Text("setup.amount.invalid", tc.precision, label, tc.example); got != want {
					t.Errorf("setup amount correction = %q, want %q", got, want)
				}
			})
		}
	}
}
