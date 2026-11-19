package messages

import "testing"

func TestSessionMessages(t *testing.T) {
	for key, want := range map[string]string{
		"sessions.title":               "Active sessions",
		"sessions.confirm_required":    "Not signed out. Confirm that you want to end this session. Nothing changed.",
		"sessions.current_consequence": "You will be signed out here and must sign in again to continue. Your other sessions and API tokens will keep working.",
		"sessions.other_consequence":   "This sign-in will stop working immediately. You will stay signed in here. Your other sessions and API tokens will keep working.",
		"sessions.bulk_title":          "Sign out other sessions?",
		"sessions.bulk_required":       "Not signed out. Confirm that you want to end your other sessions. Nothing changed.",
		"sessions.bulk_ended":          "Your other sessions have ended. You are still signed in here.",
		"sessions.pagination":          "Session list pages",
	} {
		if got := Text(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}
