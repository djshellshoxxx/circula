package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

// UX-047: the sign-in, registration, recovery, password and account-security
// pages resolve their text from the catalogue, keep accessible names and the
// uniform sign-in failure wording, and escape user-controlled interpolations.
func renderAuthPage(t *testing.T, tpl string, p *page) string {
	t.Helper()
	var b bytes.Buffer
	if err := mustLoadTemplates()[tpl].ExecuteTemplate(&b, "base", p); err != nil {
		t.Fatalf("%s: %v", tpl, err)
	}
	return b.String()
}

func wantAll(t *testing.T, name, body string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("%s: missing %q", name, w)
		}
	}
	if strings.Contains(body, "%!") {
		t.Errorf("%s: catalogue interpolation corrupted", name)
	}
}

const evil = `"><script>alert(1)</script>`

func TestSignInPagesCatalogue(t *testing.T) {
	// Uniform failure: one summary and one field message for unknown login and wrong password.
	p := &page{Title: "Sign in", Next: evil, V: map[string]string{"login": evil},
		Summary: "The login or password is not correct. Check both and try again.",
		Errors:  []fieldErr{{"login", "Login or password: check both and try again."}}}
	body := renderAuthPage(t, "login", p)
	wantAll(t, "login", body, ">Login</label>", ">Password</label>", ">Sign in</button>", "New here?", "Create an account</a>",
		"To join a network you need an invitation link from its administrator, or the network&#39;s join page.",
		"Forgot your password?", "Recover your account</a> with one of your recovery codes.",
		"The login or password is not correct. Check both and try again.", "Login or password: check both and try again.",
		`<span class="sr">Error: </span>`)
	if strings.Contains(body, "<script>alert") {
		t.Error("login: user-controlled values were not escaped")
	}

	body = renderAuthPage(t, "login_mfa", &page{Title: "Enter your authentication code", D: evil, Next: evil,
		Errors: []fieldErr{{"code", "Code: not correct, or the sign-in timed out. Check the code and try again, or sign in again."}}})
	wantAll(t, "login_mfa", body, "Your password was correct. Enter the 6-digit code from your authenticator app, or one unused recovery code. This step expires in 5 minutes.",
		">Authentication or recovery code</label>", "For example 123456, or ABCDE-FGHIJ. Each code works once.", ">Continue</button>", ">Start again</a>",
		"Code: not correct, or the sign-in timed out.")
	if strings.Contains(body, "<script>alert") {
		t.Error("login_mfa: challenge or next not escaped")
	}

	body = renderAuthPage(t, "register", &page{Title: "Create an account", V: map[string]string{"login": evil}, Next: evil})
	wantAll(t, "register", body, "At least 3 characters, no spaces. An email address works well.",
		"At least 12 characters. A password manager is fine; pasting is allowed.", ">Confirm password</label>",
		">Create account</button>", "Already have an account?", ">Sign in</a>.")
	if strings.Contains(body, "<script>alert") {
		t.Error("register: user-controlled values were not escaped")
	}
}

func TestRecoveryAndPasswordPagesCatalogue(t *testing.T) {
	body := renderAuthPage(t, "recover", &page{Title: "Recover your account", V: map[string]string{"login": evil}})
	wantAll(t, "recover", body, "Lost your password? Enter your login, one of your recovery codes and a new password.",
		">Recovery code</label>", "Looks like ABCD-EFGH-IJKL-MNOP. Capital letters and dashes are optional.",
		">New password</label>", ">Confirm new password</label>", ">Set new password</button>", ">Back to sign in</a>")
	if strings.Contains(body, "<script>alert") {
		t.Error("recover: login not escaped")
	}

	for n, want := range map[int]string{0: "<strong>0</strong> unused codes.", 1: "<strong>1</strong> unused code.", 2: "<strong>2</strong> unused codes.", 1000: "<strong>1,000</strong> unused codes."} {
		body = renderAuthPage(t, "recovery_codes", &page{Title: "Recovery codes", D: &recoveryCodesData{Remaining: n}})
		wantAll(t, "recovery_codes", body, "Recovery codes let you set a new password if you forget it, without email. You have", want,
			"Making new codes replaces all the old ones. You need your current password.", ">Current password</label>", ">Make new recovery codes</button>")
	}
	body = renderAuthPage(t, "recovery_codes", &page{Title: "Recovery codes", D: &recoveryCodesData{Codes: []string{"AAAA-BBBB", evil}}})
	wantAll(t, "recovery_codes shown", body, "Your new recovery codes", "<strong>This is the only time they are shown.</strong>", ">I have saved them</a>", "<code>AAAA-BBBB</code>")
	if strings.Contains(body, "<script>alert") {
		t.Error("recovery codes not escaped")
	}

	body = renderAuthPage(t, "password", &page{Title: "Change your password",
		Errors: []fieldErr{{"password2", "Confirm new password: the two passwords do not match. Type the same password in both fields."}}})
	wantAll(t, "password", body, "Changing your password signs you out on every device. API tokens you created keep working until you revoke them.",
		">Current password</label>", ">New password</label>", ">Confirm new password</label>", ">Change password</button>",
		"Confirm new password: the two passwords do not match.")
}

func TestSecurityPagesCatalogue(t *testing.T) {
	notices := []noticeView{{Text: "Two-step sign-in was turned on.", Detail: evil, When: time.Unix(0, 0).UTC(), New: true}}
	for n, want := range map[int]string{0: "You have 0 unused recovery codes.", 1: "You have 1 unused recovery code.", 10: "You have 10 unused recovery codes."} {
		d := &securityData{MFA: &service.MFAStatus{Enabled: true, RecoveryRemaining: n}, Notices: notices, Unseen: 1}
		body := renderAuthPage(t, "security", &page{Title: "Account security", D: d})
		wantAll(t, "security on", body, "<h2>Two-step sign-in</h2>", "Two-step sign-in is <strong>on</strong>. "+want, "<h3>Replace recovery codes</h3>",
			">Code from your authenticator app</label>", "Older recovery codes stop working.", "<h3>Turn off</h3>", ">Authenticator code or recovery code</label>",
			">Turn off two-step sign-in</button>", "<h2>Password recovery codes</h2>", "Make or replace password recovery codes</a>",
			"<h2>Security notices</h2>", "1 new.", ">Mark all as read</button>", "<strong>New:</strong> Two-step sign-in was turned on.",
			"If you do not recognise a change, change your password and turn on two-step sign-in.")
		if strings.Contains(body, "<script>alert") {
			t.Error("security notice detail not escaped")
		}
	}
	body := renderAuthPage(t, "security", &page{Title: "Account security", D: &securityData{MFA: &service.MFAStatus{}}})
	wantAll(t, "security off", body, "Two-step sign-in is <strong>off</strong>. Turn it on to protect your account if your password is stolen. Administrators may be required to use it.",
		">Turn on two-step sign-in</a>", "No security notices yet.")
}

func TestMFAEnrolPagesCatalogue(t *testing.T) {
	body := renderAuthPage(t, "mfa_enrol", &page{Title: "Turn on two-step sign-in", D: &enrolData{Status: &service.MFAStatus{}}})
	wantAll(t, "mfa_enrol start", body, "Two-step sign-in asks for a code from an authenticator app each time you sign in, in addition to your password.",
		"For your security, enter your password to start.", ">Password</label>", ">Start setup</button>", ">Cancel</a>")
	body = renderAuthPage(t, "mfa_enrol", &page{Title: "Turn on two-step sign-in", D: &enrolData{Status: &service.MFAStatus{Enabled: true}}})
	wantAll(t, "mfa_enrol on", body, `Manage it on the <a href="/account/security">account security</a> page.`)
	body = renderAuthPage(t, "mfa_enrol", &page{Title: "Turn on two-step sign-in", D: &enrolData{Enrol: &service.MFAEnrolment{Secret: evil, URI: evil}}})
	wantAll(t, "mfa_enrol key", body, "Choose time-based, 6 digits, 30 seconds.", "Apps that import links can use this address instead:",
		"Enter the 6-digit code the app shows to confirm.", ">Code from your app</label>", ">Turn on two-step sign-in</button>",
		"expires after 15 minutes.")
	if strings.Contains(body, "<script>alert") {
		t.Error("setup key not escaped")
	}
	body = renderAuthPage(t, "mfa_recovery", &page{Title: "Save your recovery codes", D: []string{"AAAA-BBBB"}})
	wantAll(t, "mfa_recovery", body, "<strong>Save these codes now.</strong> They are shown only once and cannot be shown again.", "<code>AAAA-BBBB</code>", ">Back to account security</a>")
}
