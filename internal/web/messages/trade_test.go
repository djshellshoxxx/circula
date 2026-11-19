package messages

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// UX-047: trade sentences keep exact amount and reference strings. Only counts
// use locale number formatting; interpolated member content remains plain text.
func TestTradeMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"trade.sentence.pay", []any{"Zoë 100% <img src=x>", "90071992547409.93", "Hours"}, "You pay Zoë 100% <img src=x> 90071992547409.93 Hours"},
		{"trade.sentence.receive", []any{"Zoë", "1.00", "credit"}, "Zoë pays you 1.00 credit"},
		{"trade.sentence.other", []any{"Ana", "Ben", "90071992547409.93", "credits"}, "Ana pays Ben 90071992547409.93 credits"},
		{"trade.member_with_reference", []any{"Zoë 100%", "1234-5678"}, "Zoë 100% (member reference 1234-5678)"},
		{"trade.amount_value", []any{"90071992547409.93", "<hours>&"}, "90071992547409.93 <hours>&"},
		{"trade.propose.amount_help", []any{0, "1250"}, "Amount: enter a number with up to 0 decimal places, for example 1250."},
		{"trade.propose.amount_help", []any{1, "125.0"}, "Amount: enter a number with up to 1 decimal place, for example 125.0."},
		{"trade.propose.amount_help", []any{2, "12.50"}, "Amount: enter a number with up to 2 decimal places, for example 12.50."},
		{"trade.propose.amount_hint", []any{1}, "Amount in credits, up to 1 decimal place."},
		{"trade.propose.amount_hint", []any{2}, "Amount in credits, up to 2 decimal places."},
		{"trade.propose.amount_decimals", []any{1}, "Amount: this network allows at most 1 decimal place. Remove the extra digits."},
		{"trade.propose.amount_decimals", []any{2}, "Amount: this network allows at most 2 decimal places. Remove the extra digits."},
		{"trade.propose.sent", []any{"Zoë <b>100%</b>"}, "Trade sent to Zoë <b>100%</b>. It is pending until they respond; pending trades are not part of your balance."},
		{"trade.review.too_much", []any{"90071992547409.93"}, "This is more than your available to spend (90071992547409.93). It will be refused if you send it. Go back and enter a smaller amount."},
		{"trade.review.reserved", []any{"−90071992547409.93"}, "−90071992547409.93 (the amount is set aside while the trade is pending)"},
		{"trade.review.awaiting", []any{"90071992547409.93"}, "90071992547409.93 (it rises only once the trade is accepted)"},
		{"trade.review.expiry", []any{0}, "after 0 days if not answered"},
		{"trade.review.expiry", []any{1}, "after 1 day if not answered"},
		{"trade.review.expiry", []any{1000}, "after 1,000 days if not answered"},
		{"trade.note_value", []any{"Care & repair 100%"}, "Note: Care & repair 100%"},
		{"trade.action.title", []any{"Cancel"}, "Cancel this trade?"},
		{"trade.action.confirm", []any{"Decline"}, "Decline this trade"},
		{"trade.action.completed_balance", []any{"−90071992547409.93", "In debt"}, "Trade completed. Your balance is now −90071992547409.93 · In debt."},
		{"trade.history.reason", []any{"<script>100%</script>"}, "Reason: <script>100%</script>"},
		{"trade.history.reverses", []any{"the original entry"}, "Reverses the original entry"},
		{"trade.history.adjusts", []any{"the original entry"}, "Adjusts the original entry"},
	} {
		t.Run(tc.key+"/"+tc.want, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTradeMessagesRemainPlainText(t *testing.T) {
	msg := Text("trade.sentence.other", `<img src=x onerror="alert(1)">`, "Zoë 100%", "90071992547409.93", "<Hours>&")
	var out bytes.Buffer
	if err := template.Must(template.New("message").Parse("<p>{{.}}</p>")).Execute(&out, msg); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<img") || strings.Contains(out.String(), "<Hours>") ||
		!strings.Contains(out.String(), "&lt;img") || !strings.Contains(out.String(), "&lt;Hours&gt;&amp;") {
		t.Fatal("trade sentence crossed the HTML escaping boundary")
	}
	for _, key := range []string{"trade.history.reverses", "trade.history.adjusts"} {
		parts := Around(key)
		if got := parts.Before + "the original entry" + parts.After; got != Text(key, "the original entry") {
			t.Fatalf("%s lost its template-owned link position: %q", key, got)
		}
	}
}
