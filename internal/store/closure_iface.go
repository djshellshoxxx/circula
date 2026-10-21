package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Challenge is a member's challenge to an administrator decision (GOV-016).
type Challenge struct {
	ID, NetworkID, MemberID string
	Kind                    string // removal | credit_limit | listing_removal
	SubjectID               string // listing id for listing_removal
	Reason                  string
	InvolvedID              string // member who took the original decision, "" if unknown
	Status                  string // requested | decided
	Outcome                 string // upheld | rejected
	DecidedBy, Rationale    string
	CreatedAt, DecidedAt    time.Time
}

// ClosureRepo covers scheduled closures (GOV-025), challenges (GOV-016) and
// the dedupe records of dispute deadline reminders (GOV-014).
type ClosureRepo interface {
	// GetClosure returns the network's closure, or nil.
	GetClosure(netID string) (*domain.Closure, error)
	// PutClosure inserts a closure; false when the network already has one.
	PutClosure(c domain.Closure) (bool, error)
	// DeleteOpenClosure removes a closure that has not completed; false if none.
	DeleteOpenClosure(netID string) (bool, error)
	SetClosureDone(netID string, at time.Time) error
	SetClosureFinal(netID string, ledgerSeq int64, sha string, balanceTotal, entriesSum int64) error
	SetClosureErased(netID string, at time.Time) error
	// ListClosures returns closures: pending (not completed) or completed and not yet erased.
	ListClosures(pending bool) ([]domain.Closure, error)

	InsertChallenge(c Challenge) error
	GetChallenge(netID, id string) (*Challenge, error)
	DecideChallenge(c Challenge) error
	// ListChallenges lists newest first; memberID "" lists all.
	ListChallenges(netID, memberID, status string, limit int) ([]Challenge, error)

	// RecordDeadlineNotice returns true the first time (dispute, kind) is recorded.
	RecordDeadlineNotice(netID, disputeID, kind string, at time.Time) (bool, error)
}
