package messages

import "testing"

// UX-047: display strings do not reinterpret member data as formatting, round
// financial values, or turn a pending reversal into a completed posting.
func TestDisputeMessagesPreserveDataAndPostingState(t *testing.T) {
	const name = `Zoë 100% %! $& "><img src=x>`
	const amount = "90071992547409.93"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"dispute.trade_sentence", []any{name, "Ana", amount, "Hours"}, name + " paid Ana " + amount + " Hours"},
		{"dispute.outcome.upheld_pending", []any{amount, "Hours"}, "Upheld: a reversal of " + amount + " Hours is decided but not yet posted."},
		{"dispute.outcome.upheld_posted", []any{amount, "Hours"}, "Upheld: a reversal of " + amount + " Hours has been posted."},
		{"dispute.outcome.partial_pending", []any{"1", "credit"}, "Partly upheld: a reversal of 1 credit is decided but not yet posted."},
		{"dispute.outcome.partial_posted", []any{"1", "credit"}, "Partly upheld: a reversal of 1 credit has been posted."},
		{"dispute.response_from", []any{name}, "Response from " + name},
		{"dispute.event_with_outcome", []any{name, "unexpected_%!"}, name + " (unexpected_%!)"},
		{"dispute.not_eligible_deadline", []any{"20 Nov 2026, 12:00 UTC"}, "Not sent. This trade cannot be disputed now: it must be completed, on or before its dispute deadline (20 Nov 2026, 12:00 UTC), and you must be a party to it. Nothing changed."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	for _, key := range []string{"dispute.form.deadline", "dispute.opened_at", "dispute.rereview_explanation", "dispute.empty"} {
		parts := Around(key)
		if got, want := parts.Before+"element"+parts.After, Text(key, "element"); got != want {
			t.Fatalf("%s lost its complete sentence around the template-owned element", key)
		}
	}
}
