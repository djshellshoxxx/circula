package messages

import "testing"

// UX-047, FR-038, FR-024: complete inbox/report messages retain literal names,
// count formatting, report privacy, and the distinction between review and action.
func TestInboxReportMessages(t *testing.T) {
	const name = `Zoë 100% $& "><img src=x>`
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"inbox.unread_count", []any{0}, "0 unread."},
		{"inbox.unread_count", []any{1}, "1 unread."},
		{"inbox.unread_count", []any{1000}, "1,000 unread."},
		{"inbox.no_unread", nil, "No unread notifications."},
		{"inbox.all_read", nil, "All notifications are marked as read."},
		{"inbox.dispute_opened", []any{"Quality"}, "A dispute was opened on a trade with you (Quality)"},
		{"inbox.dispute_response", []any{"19 Nov 2026, 12:00 UTC"}, "Opened 19 Nov 2026, 12:00 UTC. You can add your response."},
		{"inbox.mark_read_named", []any{name}, "Mark as read: " + name},
		{"report.target_listing", []any{name}, "Listing: " + name},
		{"report.target_member", []any{name}, "Member: " + name},
		{"report.subject", []any{name}, "You are reporting: " + name + "."},
		{"report.about", []any{"Looks like a scam", "listing", name}, "Looks like a scam about listing " + name},
		{"report.reported_by", []any{name}, "Reported by " + name},
		{"report.mark_handled_named", []any{name}, "Mark handled: report from " + name},
		{"report.handling_note", []any{name}, "Note: " + name},
		{"report.duplicate", nil, "Not sent. You already have an open report about this. An administrator will look at it. Nothing changed."},
		{"report.throttled", nil, "Not sent. You have sent many reports today. Try again tomorrow. Nothing changed."},
		{"report.sent", nil, "Report sent. Administrators will review it. It does not change anything for the person you reported, and they are not told who reported."},
		{"report.handled", nil, "Report marked as handled. The reporter was told it was reviewed."},
		{"report.explanation", nil, "A report goes to the network administrators. It does not hide, suspend or change anything by itself, and the person you report is not told who reported."},
		{"report.queue_explanation", nil, "Reports from members about listings and members. Handling a report here only records that you looked at it; use the member and policy tools to act. The reporter is shown to administrators only."},
	} {
		t.Run(tc.key+"/"+tc.want, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReportAroundUsesKnownLabelsAndOneElementSlot(t *testing.T) {
	parts := Around("report.about", Text("report.category.scam"), Text("report.target_type.listing"))
	if parts.Before != "Looks like a scam about listing " || parts.After != "" {
		t.Fatalf("report sentence or element slot changed: %#v", parts)
	}
	for _, key := range []string{"inbox.dispute_response", "report.subject"} {
		parts = Around(key)
		if parts.Before+"example"+parts.After != Text(key, "example") {
			t.Fatalf("%s changed around its element slot", key)
		}
	}
	// The final slot must still occur exactly once when preceding labels exist.
	for _, labels := range [][]string{{}, {"Scam"}, {"Scam", "listing", "extra"}} {
		t.Run("wrong argument count", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing or mispositioned element slot was accepted")
				}
			}()
			Around("report.about", labels...)
		})
	}
}
