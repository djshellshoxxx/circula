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
		case "notice.credit_limit_used":
			args = []any{"90071992547409.93", "90071992547409.94", "90"}
		case "error.fallback.reference", "error.reference", "help.balance.credit", "help.balance.debt", "help.balance.level",
			"notice.membership_pending.body", "notice.frozen", "histbal.hint", "histbal.field_error", "histbal.at_utc", "histbal.reason.format":
			args = []any{"example"}
		case "balance.spoken_credit", "balance.spoken_debt", "balance.limit_used":
			args = []any{"12.50", "credits"}
		case "notice.closing_date", "balance.spoken_level", "status.waiting_other", "reversal.ledger", "row.ledger", "join.network.title", "join.network.invite_only", "join.flash.joined", "join.terms.heading":
			args = []any{"example"}
		case "setup.amount.invalid":
			args = []any{"Default credit limit", 2, "12.50"}
		case "freshness.snapshot", "freshness.offline", "freshness.history", "session_warning.extendable", "session_warning.fixed":
			args = []any{"example"}
		case "time_zone.display":
			args = []any{"19 Nov 2026, 17:30 IST", "Asia/Kolkata"}
		case "terms.accepted_flash", "terms.accept_version", "terms.accepted_version", "terms.admin.published_flash",
			"rules.terms_link", "rules.version", "rules.announced", "rules.reason", "rules.shorter_notice",
			"leave.left", "leave.intro", "leave.balance_settle", "leave.balance_retain", "leave.download_explanation", "leave.confirm":
			args = []any{"1000"}
		case "terms.published", "terms.admin.current", "rules.announced_scheduled", "rules.in_force", "rules.initial_value":
			args = []any{"1000", "example"}
		case "rules.change":
			args = []any{"Default credit limit", "90071992547409.93", "0.00"}
		case "rules.announced_scheduled_notice":
			args = []any{14, "announced", "effective"}
		case "inbox.dispute_opened", "inbox.dispute_response", "inbox.mark_read_named", "report.target_listing", "report.target_member",
			"report.subject", "report.reported_by", "report.mark_handled_named", "report.handling_note":
			args = []any{"example"}
		case "inbox.unread_count":
			args = []any{1000}
		case "admin.check.summary":
			args = []any{1000, "9223372036854775807"}
		case "admin.announcements.starts", "admin.announcements.since", "admin.break_glass.closed_with_reason":
			args = []any{"example", "example"}
		case "admin.announcements.delete_named", "overview.frozen", "overview.frozen_reason", "overview.counted",
			"overview.manage_members", "overview.manage_invitations", "overview.manage_policy", "overview.manage_terms",
			"overview.manage_adjust", "overview.manage_reverse", "overview.manage_announcements", "overview.manage_pending",
			"overview.manage_network", "overview.manage_operator", "overview.manage_reports", "overview.manage_support", "queue.empty":
			args = []any{"example"}
		case "queue.explanation":
			args = []any{48, "example"}
		case "admin.policy.current":
			args = []any{"1000", 2}
		case "admin.policy.scheduled":
			args = []any{"1000", "announced", "effective"}
		case "admin.policy.amount_invalid":
			args = []any{2, "Largest single trade", "90071992547409.93"}
		case "admin.policy.notice", "admin.policy.amount_hint", "admin.policy.grant_hint", "admin.policy.integer_invalid",
			"admin.network.confirm_invalid", "admin.network.category", "admin.network.closed_history_until",
			"admin.network.checksum", "admin.network.export_explanation", "admin.network.review_explanation":
			args = []any{"example"}
		case "admin.network.suspension", "admin.network.closing":
			args = []any{"example", "example"}
		case "admin.network.count":
			args = []any{1000}
		case "admin.network.proposals", "admin.network.nonzero":
			args = []any{1000, "1,000"}
		case "report.about":
			args = []any{"Looks like a scam", "listing", "Bike repair"}
		case "reversal.by":
			args = []any{"ledger reference 12345", "12.50", "credits"}
		case "queue.party_proposal":
			args = []any{"Ana", "Ben", "1.00", "credit"}
		case "common.unread", "common.new", "security.notices.unseen":
			args = []any{1000}
		case "overview.stale", "overview.invites":
			args = []any{2, 48}
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
		case "admin.member_action.confirm_title", "admin.member_action.done", "admin.members.action_for", "admin.invitations.row":
			args = []any{"example", "Zoë"}
		case "admin.members.caption", "admin.inactive.caption", "admin.inactive.no_rule", "admin.invitations.revoke_named":
			args = []any{"example"}
		case "erasure.member_title", "erasure.member_heading":
			args = []any{"example"}
		case "dispute.trade_sentence":
			args = []any{"Ana", "Ben", "90071992547409.93", "Hours"}
		case "dispute.outcome.upheld_pending", "dispute.outcome.upheld_posted", "dispute.outcome.partial_pending", "dispute.outcome.partial_posted":
			args = []any{"90071992547409.93", "Hours"}
		case "dispute.event_with_outcome", "dispute.form.trade":
			args = []any{"example", "example"}
		case "dispute.not_eligible_deadline", "dispute.form.deadline", "dispute.opened_at", "dispute.response_from",
			"dispute.request_reason", "dispute.rationale_value", "dispute.rereview_explanation", "dispute.row", "dispute.empty":
			args = []any{"example"}
		case "dispute.amount_hint":
			args = []any{2}
		case "dispute.decision.reversal_sentence":
			args = []any{"90071992547409.93", "Hours", "Ana", "Ben"}
		case "dispute.decision.amount_value", "dispute.rereview.upheld_posted", "dispute.rereview.partial_posted":
			args = []any{"90071992547409.93", "Hours"}
		case "dispute.decision.waiting", "dispute.decision.change_for", "dispute.decision.balance_now", "dispute.decision.balance_after",
			"dispute.decision.amount_max", "dispute.rereview.amount_max":
			args = []any{"example"}
		case "admin.adjust.queued", "admin.adjust.caption", "admin.adjust.related_sentence", "admin.adjust.expires_suffix",
			"admin.adjust.review_named", "admin.adjust.reject_named", "admin.adjust.cancel_named",
			"admin.reverse.reference_status", "admin.reverse.amount_max", "admin.reverse.insufficient_credit",
			"admin.reverse.balance_now", "admin.reverse.balance_after":
			args = []any{"example"}
		case "admin.adjust.amount_value", "admin.adjust.threshold", "admin.adjust.status_by", "admin.reverse.amount_value":
			args = []any{"90071992547409.93", "Hours"}
		case "admin.adjust.posted", "admin.adjust.row_amount":
			args = []any{"Zoë", "90071992547409.93", "Hours"}
		case "admin.reverse.trade_sentence", "admin.reverse.full_sentence", "admin.reverse.partial_sentence", "admin.reverse.posted":
			args = []any{"Ana", "Ben", "90071992547409.93", "Hours"}
		case "admin.reverse.amount_help":
			args = []any{2, "12.50"}
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
		case "trade.sentence.pay", "trade.sentence.receive":
			args = []any{"Zoë", "90071992547409.93", "Hours"}
		case "trade.sentence.other":
			args = []any{"Ana", "Ben", "90071992547409.93", "Hours"}
		case "trade.member_with_reference":
			args = []any{"Zoë", "1234-5678"}
		case "trade.amount_value", "trade.action.completed_balance":
			args = []any{"90071992547409.93", "Hours"}
		case "trade.propose.amount_help":
			args = []any{2, "12.50"}
		case "trade.propose.sent", "trade.review.too_much", "trade.review.reserved", "trade.review.awaiting",
			"trade.note_value", "trade.action.title", "trade.action.confirm", "trade.history.reason",
			"trade.history.reverses", "trade.history.adjusts":
			args = []any{"example"}
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
