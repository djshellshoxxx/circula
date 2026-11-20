package messages

import "testing"

// UX-047 authentication continuation: counted sentences select CLDR plural
// forms, and fixed refusal text is returned verbatim.
func TestAuthenticationMessages(t *testing.T) {
	for _, c := range []struct {
		key  string
		n    int
		want string
	}{
		{"security.mfa.recovery_left", 0, "You have 0 unused recovery codes."},
		{"security.mfa.recovery_left", 1, "You have 1 unused recovery code."},
		{"security.mfa.recovery_left", 10, "You have 10 unused recovery codes."},
		{"security.mfa.recovery_left", 1000, "You have 1,000 unused recovery codes."},
		{"auth.codes.unused", 0, "unused codes"},
		{"auth.codes.unused", 1, "unused code"},
		{"auth.codes.unused", 2, "unused codes"},
		{"security.notices.unseen", 1, "1 new."},
		{"security.notices.unseen", 1000, "1,000 new."},
	} {
		if got := Text(c.key, c.n); got != c.want {
			t.Errorf("%s(%d) = %q, want %q", c.key, c.n, got, c.want)
		}
	}
	for key, want := range map[string]string{
		"auth.login.failed":           "The login or password is not correct. Check both and try again.",
		"auth.login.failed_field":     "Login or password: check both and try again.",
		"auth.throttle.minute":        "Too many attempts. Wait a minute and try again. Nothing changed.",
		"auth.throttle.few_minutes":   "Too many attempts. Wait a few minutes and try again. Nothing changed.",
		"auth.field.code_wrong":       "Code: not correct. Use the current 6-digit code from your authenticator app (or an unused recovery code where allowed). Codes work once.",
		"auth.login.join_info":        "To join a network you need an invitation link from its administrator, or the network's join page.",
		"auth.mismatch.confirm_new":   "Confirm new password: the two passwords do not match. Type the same password in both fields.",
		"security.notice.other":       "A security-related change was made to your account.",
		"auth.flash.password_changed": "Password changed. You were signed out everywhere. Sign in with the new password.",
	} {
		if got := Text(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}
