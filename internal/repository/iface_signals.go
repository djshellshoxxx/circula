package repository

import "time"

// ActivitySignal is one append-only velocity or reciprocal-pattern signal
// (SEC-026). It holds member ids and counts only, never personal text.
// MemberID is the proposer of a velocity signal; for a reciprocal signal the
// pair is ordered so that MemberID < OtherMemberID. PeriodStart is the fixed
// deduplication period (domain.SignalPeriodStart) and Window its length;
// Observed counts the events in the Window that ended at RaisedAt.
type ActivitySignal struct {
	ID, NetworkID, Kind     string
	MemberID, OtherMemberID string // OtherMemberID is "" for velocity
	PeriodStart             time.Time
	Window                  time.Duration
	Observed, Threshold     int
	RaisedAt                time.Time
}

// ActivitySignalFilter selects signals for a reviewer; newest first by id.
type ActivitySignalFilter struct {
	Kind string // "" = all kinds
	// Viewer is the reading member: signals naming them are left out, so a
	// signal is never shown to a member it concerns.
	Viewer  string
	AfterID string
	Limit   int
}

// SignalRepo stores activity signals and answers the counts they are based on.
type SignalRepo interface {
	// CountProposalsSince counts trade proposals created by memberID with
	// proposed_at after since (any later status).
	CountProposalsSince(netID, memberID string, since time.Time) (int, error)
	// CountCompletedTradesBetween counts the trades completed after since in
	// which payerAccountID paid payeeAccountID, and the reverse direction.
	CountCompletedTradesBetween(netID, accountA, accountB string, since time.Time) (aToB, bToA int, err error)
	// InsertActivitySignal stores s unless a signal with the same network,
	// kind, members and period exists. inserted is false when nothing was
	// written; the database enforces the uniqueness, so concurrent callers
	// cannot record two signals for one period.
	InsertActivitySignal(s ActivitySignal) (inserted bool, err error)
	ListActivitySignals(netID string, f ActivitySignalFilter) ([]*ActivitySignal, error)
	// Savepoint runs fn inside a nested savepoint of the current unit. When fn
	// fails, everything fn wrote is rolled back and the unit stays usable;
	// writes made before the savepoint are kept.
	Savepoint(fn func() error) error
}
