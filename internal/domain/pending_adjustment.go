package domain

import "time"

// PendingStatus is the state of a dual-control adjustment request (SEC-011).
type PendingStatus string

// Pending adjustment statuses. Only pending is open; every other status is final.
const (
	PendingOpen      PendingStatus = "pending"
	PendingApproved  PendingStatus = "approved"
	PendingRejected  PendingStatus = "rejected"
	PendingExpired   PendingStatus = "expired"
	PendingCancelled PendingStatus = "cancelled"
)

// PendingAdjustment is an administrator adjustment waiting for a second
// administrator. Nothing is posted until it is approved.
type PendingAdjustment struct {
	ID        string
	NetworkID string
	Proposer  string // member id
	Target    string // member id
	Amount    int64  // minor units, > 0
	Direction string // credit | debit
	Reason    string
	Status    PendingStatus
	CreatedAt time.Time
	ExpiresAt time.Time
	DecidedBy string // member id; "" while pending or when closed by the system
	DecidedAt *time.Time
	TxID      string // the posted adjustment, once approved
}
