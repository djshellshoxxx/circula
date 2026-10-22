package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// MFARepo covers second-factor state, recovery codes, login challenges and
// security notices (migration 0003).
type MFARepo interface {
	// GetMFA returns the person's record or ErrNotFound.
	GetMFA(personID string) (*domain.MFARecord, error)
	// PutMFA inserts or replaces the record.
	PutMFA(r *domain.MFARecord) error
	DeleteMFA(personID string) error
	// AdvanceMFAStep sets last_step=step only if step is greater than the
	// stored value; reports whether it advanced (replay guard, atomic).
	AdvanceMFAStep(personID string, step int64) (bool, error)

	// ReplaceRecoveryCodes deletes all codes of the person and stores hashes.
	ReplaceRecoveryCodes(personID string, hashes []string) error
	// ConsumeRecoveryCode marks an unused code used; reports whether it did.
	ConsumeRecoveryCode(personID, hash string, at time.Time) (bool, error)
	// CountRecoveryCodes counts unused codes.
	CountRecoveryCodes(personID string) (int, error)

	InsertMFAChallenge(c *domain.MFAChallenge) error
	GetMFAChallenge(idHash string) (*domain.MFAChallenge, error)
	// UseMFAChallenge marks the challenge used if it is unused; reports success.
	UseMFAChallenge(idHash string, at time.Time) (bool, error)
	// DeleteDeadMFAChallenges removes the person's challenges that expired by cutoff or were used.
	// DeleteMFAChallengesOf removes every challenge of the person, used or not (SA2-13).
	DeleteMFAChallengesOf(personID string) error
	DeleteDeadMFAChallenges(personID string, cutoff time.Time) error

	InsertNotice(n *domain.SecurityNotice) error
	// ListNotices returns newest first. unseenOnly restricts to seen_at IS NULL.
	ListNotices(personID string, unseenOnly bool, limit int) ([]*domain.SecurityNotice, error)
	CountUnseenNotices(personID string) (int, error)
	// AckNotices marks every unseen notice of the person seen; returns the count.
	AckNotices(personID string, at time.Time) (int64, error)
	// DeleteNoticesOf removes every notice of the person (erasure, SEC-018).
	DeleteNoticesOf(personID string) error
}
