package messages

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// UX-047: plural selection and English number formatting belong to the
// catalogue, including zero, exact singular, and grouped large counts.
func TestOverviewMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"overview.pending", []any{0}, "0 membership requests are waiting"},
		{"overview.pending", []any{1}, "1 membership request is waiting"},
		{"overview.pending", []any{1000}, "1,000 membership requests are waiting"},
		{"overview.stale", []any{1, 48}, "1 open proposal has waited at least 48 hours for an answer"},
		{"overview.stale", []any{2, 72}, "2 open proposals have waited at least 72 hours for an answer"},
		{"overview.disputes", []any{1}, "1 dispute is open or under review"},
		{"overview.disputes", []any{0}, "0 disputes are open or under review"},
		{"overview.posting", []any{1}, "1 dispute is decided but the reversal is not yet posted"},
		{"overview.posting", []any{2}, "2 disputes are decided but the reversal is not yet posted"},
		{"overview.limit", []any{1}, "1 active member is at or beyond their credit limit"},
		{"overview.limit", []any{2}, "2 active members are at or beyond their credit limit"},
		{"overview.invites", []any{1, 72}, "1 invitation ends within 72 hours"},
		{"overview.invites", []any{2, 48}, "2 invitations end within 48 hours"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := Text(tc.key, tc.args...); got != tc.want+"." {
				t.Fatalf("got %q, want %q", got, tc.want+".")
			}
		})
	}
}

// SEC-026 / UX-047: activity signal sentences select the plural on the count,
// reorder their arguments, group large numbers and keep names as plain text.
func TestSEC026SignalMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"signals.velocity", []any{21, "Ana", 60, 20}, "Ana made 21 trade proposals within a 60-minute window, above the review threshold of 20."},
		{"signals.velocity", []any{1, "Ana", 1, 0}, "Ana made 1 trade proposal within a 1-minute window, above the review threshold of 0."},
		{"signals.velocity", []any{1001, "<b>Zoë</b>", 10080, 1000}, "<b>Zoë</b> made 1,001 trade proposals within a 10,080-minute window, above the review threshold of 1,000."},
		{"signals.reciprocal", []any{11, "Ana", "Ben", 30, 10}, "Ana and Ben completed 11 trades with each other, in both directions, within a 30-day window, above the review threshold of 10."},
		{"signals.settings", []any{20, 60, 10, 30}, "This network raises a signal when the trade proposals one member makes within a 60-minute window exceed 20, or when the trades two members complete with each other, in both directions, within a 30-day window exceed 10."},
	} {
		if got := Text(tc.key, tc.args...); got != tc.want {
			t.Errorf("%s%v = %q, want %q", tc.key, tc.args, got, tc.want)
		}
	}
}

// UX-047: catalogue interpolation preserves exact amounts, Unicode names and
// literal percent signs; the result remains subject to HTML escaping.
func TestProposalMessageIsPlainText(t *testing.T) {
	got := Text("queue.party_proposal", "<img src=x>", "Zoë 100%", "90071992547409.93", "credits")
	if got != "<img src=x> proposes to pay Zoë 100% 90071992547409.93 credits" {
		t.Fatal(got)
	}
	var b bytes.Buffer
	if err := template.Must(template.New("test").Parse("{{.}}")).Execute(&b, got); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "<img") || !strings.Contains(b.String(), "&lt;img") {
		t.Fatal("message crossed the HTML escaping boundary")
	}
}

func TestUnknownMessageFailsVisibly(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("unknown key was silently printed")
		}
	}()
	Text("missing.message")
}

func TestEveryCatalogueEntryFormats(t *testing.T) {
	for key, e := range keys {
		var args []any
		switch key {
		case "balance.spoken_credit", "balance.spoken_debt", "balance.limit_used":
			args = []any{"12.50", "credits"}
		case "notice.closing_date", "balance.spoken_level", "status.waiting_other", "reversal.ledger", "row.ledger", "join.network.title", "join.network.invite_only", "join.flash.joined", "join.terms.heading":
			args = []any{"example"}
		case "setup.amount.invalid":
			args = []any{"Default credit limit", 2, "12.50"}
		case "freshness.snapshot", "freshness.offline", "freshness.history", "session_warning.extendable", "session_warning.fixed":
			args = []any{"example"}
		case "reversal.by":
			args = []any{"ledger reference 12345", "12.50", "credits"}
		case "queue.party_proposal":
			args = []any{"Ana", "Ben", "1.00", "credit"}
		case "common.unread", "common.new", "security.notices.unseen":
			args = []any{1000}
		case "overview.stale", "overview.invites":
			args = []any{2, 48}
		case "signals.settings":
			args = []any{20, 60, 10, 30}
		case "signals.velocity":
			args = []any{21, "Ana", 60, 20}
		case "signals.reciprocal":
			args = []any{11, "Ana", "Ben", 30, 10}
		case "validation.fix.decimals":
			args = []any{2}
		case "validation.fix.number", "validation.refusal.insufficient_credit_available", "validation.refusal.amount_exceeds":
			args = []any{"90071992547409.93"}
		case "validation.refusal.blocked", "admin.reset.confirm_title", "admin.reset.for_member", "admin.reset.instructions":
			args = []any{"example"}
		case "validation.field.message":
			args = []any{"Amount", "not a number", "Enter digits only, for example 12.50."}
		case "admin.member_action.heading":
			args = []any{"Reset password", "Zoë"}
		case "listing.owner_price":
			args = []any{"Zoë", "90071992547409.93", "Hours"}
		case "listing.price":
			args = []any{"1.00", "credit"}
		case "listing.price_help":
			args = []any{2, "12.50"}
		case "listing.by_owner", "listing.edit_named", "listing.report_named", "listing.propose_about",
			"profile.listings_by", "profile.empty", "profile.propose_with", "profile.report",
			"directory.member_reference", "directory.propose_with":
			args = []any{"Zoë"}
		default:
			if strings.HasPrefix(key, "financial.row.") {
				args = []any{"Ana", "12.50", "credits"}
			}
			if e.One != "" {
				args = []any{2}
			}
		}
		if got := Text(key, args...); got == key || strings.Contains(got, "%!") {
			t.Fatalf("unresolved message %q: %q", key, got)
		}
	}
}
