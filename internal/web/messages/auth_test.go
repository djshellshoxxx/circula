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
		{"auth.codes.count", 0, "0"},
		{"auth.codes.count", 1, "1"},
		{"auth.codes.count", 1000, "1,000"},
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
		"auth.mismatch.confirm_new":   "Confirm new password: the two passwords do not match. Type the same password in both fields.",
		"security.notice.other":       "A security-related change was made to your account.",
		"auth.flash.password_changed": "Password changed. You were signed out everywhere. Sign in with the new password.",
	} {
		if got := Text(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestAuthenticationCompleteSentences(t *testing.T) {
	for _, tc := range []struct{ key, label, want string }{
		{"auth.login.create_prompt", "Create an account", "New here? Create an account. To join a network you need an invitation link from its administrator, or the network's join page."},
		{"auth.login.recovery_prompt", "Recover your account", "Forgot your password? Recover your account with one of your recovery codes."},
		{"auth.enrol.already_on", "account security", "Two-step sign-in is already on. Manage it on the account security page."},
		{"auth.enrol.uri_note", "app-address", "Apps that import links can use this address instead: app-address"},
		{"auth.register.have_account", "Sign in", "Already have an account? Sign in."},
		{"security.mfa.state", "on", "Two-step sign-in is on."},
		{"security.mfa.state", "off", "Two-step sign-in is off."},
	} {
		parts := Around(tc.key)
		if got := parts.Before + tc.label + parts.After; got != tc.want {
			t.Errorf("%s = %q, want %q", tc.key, got, tc.want)
		}
	}
	for _, tc := range []struct {
		count int
		want  string
	}{
		{0, "You have 0 unused codes."},
		{1, "You have 1 unused code."},
		{2, "You have 2 unused codes."},
		{1000, "You have 1,000 unused codes."},
	} {
		parts := AroundCount("auth.codes.remaining", tc.count)
		got := parts.Before + Text("auth.codes.count", tc.count) + parts.After
		want := "Recovery codes let you set a new password if you forget it, without email. " + tc.want
		if got != want {
			t.Errorf("count %d = %q, want %q", tc.count, got, want)
		}
	}
}
