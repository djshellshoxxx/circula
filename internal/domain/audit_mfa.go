package domain

// Audit actions for second-factor management (SEC-007). None of these events
// carries a secret, code or hash.
const (
	ActMFAEnrolBegun       = "mfa.enrol_begun"
	ActMFAEnabled          = "mfa.enabled"
	ActMFADisabled         = "mfa.disabled"
	ActMFARecoveryRegen    = "mfa.recovery_codes_regenerated"
	ActMFARecoveryCodeUsed = "mfa.recovery_code_used"
)

// CodeMFARequired is returned by the legacy Login when the person has a second
// factor: no session is issued; callers must use LoginStep and CompleteMFA.
const CodeMFARequired Code = "mfa-required"
