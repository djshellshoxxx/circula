package messages

import "testing"

func TestMemberInvitationMessages(t *testing.T) {
	const name = `Zoë 100% $& "><img src=x>` + "\x00%!"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"admin.member_action.confirm_title", []any{"Suspend", name}, "Suspend: " + name + "?"},
		{"admin.member_action.done", []any{"Approve", name}, "Approve: done for " + name + "."},
		{"admin.members.action_for", []any{"Set credit limit", name}, "Set credit limit " + name},
		{"admin.members.caption", []any{name}, "Members of " + name},
		{"admin.inactive.caption", []any{name}, "Inactive members of " + name},
		{"admin.invitations.revoke_named", []any{"19 Nov 2026, 17:30 IST"}, "Revoke invitation expiring 19 Nov 2026, 17:30 IST"},
		{"admin.members.matches", []any{0}, "0 members match this filter."},
		{"admin.members.matches", []any{1}, "1 member matches this filter."},
		{"admin.members.matches", []any{1000}, "1,000 members match this filter."},
		{"admin.invitations.days_error", nil, "Valid for: enter a whole number of days from 1 to 90."},
		{"admin.invitations.invalid", nil, "Not created. Fix the problems below. Nothing changed."},
		{"admin.member_action.invalid", nil, "Not saved. Fix the problems below. Nothing changed."},
	} {
		if got := Text(tc.key, tc.args...); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.key, got, tc.want)
		}
	}
	for _, tc := range []struct {
		days int
		text string
	}{{1, "1 day"}, {60, "60 days"}, {1000, "1,000 days"}} {
		want := "Members with no sign-in and no trade for " + tc.text + ". They have been told in their inbox. They are still members: nothing is removed or suspended automatically. Decide on each one yourself."
		if got := Text("admin.inactive.explanation", tc.days); got != want {
			t.Errorf("inactivity: got %q, want %q", got, want)
		}
	}
}

// invitationRow supplies only these known catalogue labels to the one-slot
// formatter. Member names, statuses, roles and removal fallback values never
// enter Around; their templates use ordinary, contextually escaped strings.
func TestMemberInvitationMarkupSlots(t *testing.T) {
	for _, state := range []string{"unused", "used", "revoked", "expired"} {
		label := Text("admin.invitations.status." + state)
		parts := Around("admin.invitations.row", label)
		if parts.Before+"TIME"+parts.After != label+", expires TIME" {
			t.Fatalf("invitation timestamp slot changed: %#v", parts)
		}
	}
	parts := Around("admin.inactive.no_rule")
	if parts.Before+"LINK"+parts.After != "This network has no inactivity rule. You can set one in the LINK." {
		t.Fatalf("policy link slot changed: %#v", parts)
	}
}
