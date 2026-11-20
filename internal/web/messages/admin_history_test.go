package messages

import "testing"

func TestAdminHistoryMessages(t *testing.T) {
	const unsafe = `Zoë 100% $& "><img src=x>` + "\x00%!"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"admin.announcements.published", nil, "Announcement published."},
		{"admin.announcements.deleted", nil, "Announcement deleted. Members no longer see it; the audit log keeps the record."},
		{"admin.announcements.invalid", nil, "Not published. Fix the problems below. Nothing changed."},
		{"admin.announcements.days_invalid", nil, "Start in (days): enter a whole number from 0 to 365, or leave it empty to publish now."},
		{"admin.announcements.starts", []any{unsafe, "TIME"}, unsafe + " · starts TIME"},
		{"admin.announcements.since", []any{unsafe, "TIME"}, unsafe + " · since TIME"},
		{"admin.announcements.delete_named", []any{unsafe}, "Delete announcement " + unsafe},
		{"overview.frozen", []any{"TIME"}, "Frozen since TIME. Members can read balances and history, but every change is refused (status 423) until the ledger check passes again."},
		{"overview.frozen_reason", []any{unsafe}, "Reason: " + unsafe},
		{"overview.counted", []any{"TIME"}, "Counted from the ledger and the dispute records when this page loaded, TIME."},
		{"overview.manage_members", []any{"Members"}, "Members: approve, suspend, restore, remove, roles and credit limits"},
		{"queue.explanation", []any{0, "TIME"}, "Open proposals waiting at least 0 hours for an answer, as of TIME. Expired proposals are excluded."},
		{"queue.explanation", []any{1, "TIME"}, "Open proposals waiting at least 1 hour for an answer, as of TIME. Expired proposals are excluded."},
		{"queue.explanation", []any{48, "TIME"}, "Open proposals waiting at least 48 hours for an answer, as of TIME. Expired proposals are excluded."},
		{"queue.explanation", []any{1000, "TIME"}, "Open proposals waiting at least 1,000 hours for an answer, as of TIME. Expired proposals are excluded."},
		{"queue.empty", []any{"LINK"}, "No proposals are waiting this long. LINK."},
		{"admin.break_glass.closed_with_reason", []any{"TIME", unsafe}, "TIME (" + unsafe + ")"},
		{"admin.break_glass.status_open", nil, "open"},
		{"admin.break_glass.status_closed", nil, "closed"},
		{"admin.break_glass.status_expired", nil, "expired"},
		{"admin.break_glass.not_closed", nil, "Not closed"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if _, ok := keys[tc.key]; !ok {
				t.Fatalf("message is not in the catalogue: %s", tc.key)
			}
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAdminHistoryMarkupSlots(t *testing.T) {
	for _, key := range []string{"overview.frozen", "overview.counted", "overview.manage_members", "overview.manage_support", "queue.empty"} {
		parts := Around(key)
		if parts.Before+"ELEMENT"+parts.After != Text(key, "ELEMENT") {
			t.Fatalf("link or timestamp slot changed for %q", key)
		}
	}
	for _, key := range []string{"admin.announcements.starts", "admin.announcements.since", "admin.break_glass.closed_with_reason"} {
		parts := AroundTwo(key)
		if parts.Before+"FIRST"+parts.Between+"SECOND"+parts.After != Text(key, "FIRST", "SECOND") {
			t.Fatalf("data/timestamp slots changed for %q", key)
		}
	}
	for _, tc := range []struct {
		hours int
		want  string
	}{{0, "0 hours"}, {1, "1 hour"}, {48, "48 hours"}, {1000, "1,000 hours"}} {
		parts := AroundCount("queue.explanation", tc.hours)
		if parts.Before != "Open proposals waiting at least "+tc.want+" for an answer, as of " || parts.After != ". Expired proposals are excluded." {
			t.Fatalf("queue duration or timestamp slot changed: %#v", parts)
		}
	}
}
