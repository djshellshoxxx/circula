package store

import "time"

// RecoveryRepo covers account recovery state (SEC-008): single-use recovery
// codes, the forced-password-change flag and bulk API-token revocation.
type RecoveryRepo interface {
	// ReplaceAccountRecoveryCodes deletes every code of the person and stores hashes.
	ReplaceAccountRecoveryCodes(personID string, ids, hashes []string, at time.Time) error
	// ConsumeAccountRecoveryCode marks the matching unused code used. It reports true
	// for exactly one of any number of concurrent callers.
	ConsumeAccountRecoveryCode(personID, hash string, at time.Time) (bool, error)
	// HasUnusedAccountRecoveryCode reports, without changing anything, whether the
	// person has an unused code with this hash (A3-04: checked before hashing).
	HasUnusedAccountRecoveryCode(personID, hash string) (bool, error)
	// CountAccountRecoveryCodes counts the person's unused codes.
	CountAccountRecoveryCodes(personID string) (int, error)
	DeleteAccountRecoveryCodes(personID string) error

	MustChangePassword(personID string) (bool, error)
	SetMustChangePassword(personID string, v bool) error
	// RevokeTokensOf revokes every unrevoked API token of the person.
	RevokeTokensOf(personID string, at time.Time) error
}
