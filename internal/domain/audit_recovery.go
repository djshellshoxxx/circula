package domain

// Audit actions of account recovery (SEC-008). None of them ever carries a
// code, a password or a temporary password.
const (
	ActPersonRecovered         = "person.recovered"                // self-service reset with a recovery code
	ActRecoveryCodesGenerated  = "person.recovery_codes_generated" // a new batch replaced the old one
	ActPersonPasswordReset     = "person.password_reset"           // platform scope: admin or operator reset (subject = person)
	ActMemberPasswordReset     = "member.password_reset"           // network scope: admin reset of a member
	CodePasswordChangeRequired = Code("password-change-required")  // forced change pending; see Actor.MustChangePassword
)

// ErrPasswordChangeRequired is returned by every write path other than
// ChangePassword and Logout while the actor's person must change the password.
// errors.Is(err, ErrPasswordChangeRequired) matches on the code only.
var ErrPasswordChangeRequired = &Error{Code: CodePasswordChangeRequired}

// PasswordChangeRequired builds the refusal with a message for humans.
func PasswordChangeRequired() *Error {
	return Errf(CodePasswordChangeRequired, "you must change your password before doing anything else")
}
