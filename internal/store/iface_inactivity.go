package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// InactivityRepo supports the optional inactivity rule (GOV-012). It only
// reads activity and lists members; it never changes a membership.
type InactivityRepo interface {
	// RecordPersonActivity saves each person's newest session activity before
	// idle sessions are deleted.
	RecordPersonActivity() error
	// ListInactiveMembers lists active members whose newest activity is older
	// than cutoff, ordered by id, after afterID.
	ListInactiveMembers(netID string, cutoff time.Time, afterID string, limit int) ([]*domain.Member, error)
}
