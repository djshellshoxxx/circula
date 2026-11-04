package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// PendingAdjustmentRepo stores dual-control adjustment requests (SEC-011).
type PendingAdjustmentRepo interface {
	InsertPendingAdjustment(p *domain.PendingAdjustment) error
	// GetPendingAdjustment returns not-found for a missing id or another network's id.
	GetPendingAdjustment(netID, id string) (*domain.PendingAdjustment, error)
	// ListPendingAdjustments lists a network's requests newest first by id, keyset after afterID
	// (the smaller ids). Empty statuses means all.
	ListPendingAdjustments(netID string, statuses []domain.PendingStatus, afterID string, limit int) ([]*domain.PendingAdjustment, error)
	// DecidePendingAdjustment moves a pending request to a final status. It reports false when the
	// request was no longer pending (a concurrent decision won), so exactly one decision applies.
	DecidePendingAdjustment(netID, id string, to domain.PendingStatus, by string, at time.Time, txID string) (bool, error)
	// ListExpiredPendingAdjustments returns up to limit open requests whose expiry is at or before now.
	ListExpiredPendingAdjustments(now time.Time, limit int) ([]*domain.PendingAdjustment, error)
}
