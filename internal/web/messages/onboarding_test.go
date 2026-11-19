package messages

import "testing"

func TestOnboardingMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"setup.amount.invalid", []any{"Default credit limit", 4, "0.1250"}, "Default credit limit: enter a number with up to 4 decimal places, for example 0.1250."},
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
