package domain

import "time"

// MFARecord is the stored second-factor state of a person. Secret is sealed
// text (see service sealing) and must never leave the service layer.
type MFARecord struct {
	PersonID  string
	Secret    string
	Enabled   bool
	LastStep  int64
	CreatedAt time.Time
	EnabledAt *time.Time
}

// MFAChallenge is the stored form of a pending MFA login step.
type MFAChallenge struct {
	IDHash    string
	PersonID  string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// Security notice kinds (SEC-012).
const (
	NoticePasswordChanged  = "password.changed"
	NoticeMFAEnabled       = "mfa.enabled"
	NoticeMFADisabled      = "mfa.disabled"
	NoticeRecoveryRegen    = "mfa.recovery_codes_regenerated"
	NoticeRecoveryCodeUsed = "mfa.recovery_code_used"
	NoticeTokenCreated     = "token.created"
	NoticeTokenRevoked     = "token.revoked"
	NoticeAdminGranted     = "admin.granted"
)

// SecurityNotice tells a person about a security-relevant change to their
// own account. Detail never holds secrets.
type SecurityNotice struct {
	ID        string
	PersonID  string
	Kind      string
	CreatedAt time.Time
	SeenAt    *time.Time
	Detail    string
}
