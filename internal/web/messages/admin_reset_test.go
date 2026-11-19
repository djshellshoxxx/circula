package messages

import "testing"

// UX-047: administrator reset messages keep complete sentences, literal
// percent signs and Unicode names without treating member data as a format.
func TestAdministratorResetMessages(t *testing.T) {
	name := "Zoë 100% <strong>Sam</strong>"
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"auth.admin_reset.title", []any{name}, "Reset password: " + name + "?"},
		{"auth.admin_reset.subject", []any{name}, "Reset password: " + name},
		{"auth.admin_reset.done.subject", []any{name}, "Temporary password for " + name},
		{"auth.admin_reset.done.explain", []any{name}, "Give it to " + name + " in person or by a channel you trust. They must choose a new password when they sign in. All their sessions and API tokens were ended."},
		{"auth.admin_reset.forbidden", nil, "Not done. This password cannot be reset here. Use Change password for your own account, or ask the platform operator for help. Nothing changed."},
		{"auth.admin_reset.rate_limited", nil, "Not done. You have reset too many passwords in the last hour. Try again later. Nothing changed."},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
