package messages

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// UX-047: administrator money messages retain exact fixed-point strings and
// complete action names; only the precision count uses plural selection.
func TestAdminAmountMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"admin.adjust.amount_value", []any{"90071992547409.93", "Hours"}, "90071992547409.93 Hours"},
		{"admin.adjust.amount_hint", []any{1}, "Amount in credits, up to 1 decimal place."},
		{"admin.adjust.amount_hint", []any{4}, "Amount in credits, up to 4 decimal places."},
		{"admin.adjust.queued", []any{"Zoë <img src=x> 100%"}, "Adjustment for Zoë <img src=x> 100% is above the dual-control threshold. Nothing is posted yet: another administrator must approve it before it expires."},
		{"admin.adjust.posted", []any{"Zoë", "−90071992547409.93", "In debt"}, "Adjustment posted for Zoë. Their balance is now −90071992547409.93 · In debt."},
		{"admin.adjust.threshold", []any{"1.00", "credit"}, "Adjustments above 1.00 credit need approval from a second administrator before anything is posted. A request that is not decided in time expires and posts nothing."},
		{"admin.adjust.caption", []any{"Town & Country 100%"}, "Adjustment requests in Town & Country 100%"},
		{"admin.adjust.row_amount", []any{"Debit (balance goes down)", "90071992547409.93", "Hours"}, "Debit (balance goes down): 90071992547409.93 Hours"},
		{"admin.adjust.status_by", []any{"approved", "Zoë\x00%! <img>"}, "approved by Zoë\x00%! <img>"},
		{"admin.adjust.review_named", []any{"Zoë 100%"}, "Review approval for Zoë 100%"},
		{"admin.adjust.reject_named", []any{"Zoë 100%"}, "Reject adjustment for Zoë 100%"},
		{"admin.adjust.cancel_named", []any{"Zoë 100%"}, "Cancel request for Zoë 100%"},
		{"admin.adjust.related_sentence", []any{"the original entry"}, "Adjusts the original entry"},
		{"admin.reverse.amount_value", []any{"1.00", "credit"}, "1.00 credit"},
		{"admin.reverse.amount_hint", []any{0}, "Optional. Up to 0 decimal places. Leave empty to reverse everything that is left."},
		{"admin.reverse.amount_hint", []any{1}, "Optional. Up to 1 decimal place. Leave empty to reverse everything that is left."},
		{"admin.reverse.amount_hint", []any{4}, "Optional. Up to 4 decimal places. Leave empty to reverse everything that is left."},
		{"admin.reverse.amount_help", []any{1, "125.0"}, "Amount: enter a number with up to 1 decimal place, for example 125.0. Leave it empty to reverse everything that is left."},
		{"admin.reverse.amount_help", []any{2, "12.50"}, "Amount: enter a number with up to 2 decimal places, for example 12.50. Leave it empty to reverse everything that is left."},
		{"admin.reverse.amount_max", []any{"90071992547409.93"}, "Amount: at most 90071992547409.93 is left to reverse on this trade."},
		{"admin.reverse.reference_status", []any{"Pending"}, "Trade reference: only a completed trade can be reversed. This one is Pending."},
		{"admin.reverse.insufficient_credit", []any{"Zoë <img src=x>"}, "Not reversed. Zoë <img src=x> does not have enough available credit to give this amount back. Nothing changed."},
		{"admin.reverse.trade_sentence", []any{"Ana", "Ben", "90071992547409.93", "Hours"}, "Ana paid Ben 90071992547409.93 Hours"},
		{"admin.reverse.full_sentence", []any{"1.00", "credit", "Ben", "Ana"}, "1.00 credit go back from Ben to Ana. This reverses everything that is left, so the trade becomes Reversed."},
		{"admin.reverse.partial_sentence", []any{"90071992547409.93", "Hours", "Ben", "Ana"}, "90071992547409.93 Hours go back from Ben to Ana. This is a partial reversal; the trade stays Completed."},
		{"admin.reverse.balance_now", []any{"Zoë 100%"}, "Zoë 100% now"},
		{"admin.reverse.balance_after", []any{"Zoë 100%"}, "Zoë 100% after"},
		{"admin.reverse.posted", []any{"90071992547409.93", "Hours", "Ben", "Ana"}, "Reversal posted: 90071992547409.93 Hours went back from Ben to Ana."},
	} {
		t.Run(tc.key+"/"+tc.want, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAdminAmountMessagesRemainPlainText(t *testing.T) {
	var out bytes.Buffer
	msg := Text("admin.reverse.posted", "90071992547409.93", "<Hours>&", `<img src=x onerror="alert(1)">`, "Zoë 100%")
	if err := template.Must(template.New("message").Parse("<p>{{.}}</p>")).Execute(&out, msg); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<Hours>") || strings.Contains(out.String(), "<img") ||
		!strings.Contains(out.String(), "&lt;Hours&gt;&amp;") || !strings.Contains(out.String(), "&lt;img") {
		t.Fatal("administrator financial message crossed the escaping boundary")
	}
	for _, key := range []string{"admin.adjust.related_sentence", "admin.adjust.expires_suffix"} {
		parts := Around(key)
		if got := parts.Before + "element" + parts.After; got != Text(key, "element") {
			t.Fatalf("%s lost its template-owned element position", key)
		}
	}
}
