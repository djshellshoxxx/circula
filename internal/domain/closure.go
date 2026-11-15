package domain

import "time"

// Closure periods (DEC-025, DEC-020, GOV-025).
const (
	// MinClosureNoticeDays is the shortest notice a closure may give; there is
	// no override, unlike policy notices.
	MinClosureNoticeDays = 30
	// MaxClosureNoticeDays bounds the announcement horizon.
	MaxClosureNoticeDays = 365
	// ClosureReadOnlyDays is how long read and export access continue after closure.
	ClosureReadOnlyDays = 90
)

// NetworkClosing is a derived status: a closure date has passed but the
// closure has not finished (open disputes remain). Trading is already frozen.
// It is never stored.
const NetworkClosing NetworkStatus = "closing"

// Closure is a scheduled or completed network closure (GOV-025).
type Closure struct {
	NetworkID    string
	Reason       string
	ScheduledBy  string // member id
	ScheduledAt  time.Time
	ClosingAt    time.Time
	NoticeDays   int
	ReadOnlyDays int
	ClosedAt     time.Time // zero until the closure completes

	// Final export (GOV-025 e): the checksum covers the ledger package manifest.
	FinalLedgerSeq    int64
	FinalSHA256       string
	FinalBalanceTotal int64
	FinalEntriesSum   int64

	ErasedAt time.Time // zero until the post-period erasure ran
}

// Done reports whether the closure has completed.
func (c *Closure) Done() bool { return !c.ClosedAt.IsZero() }

// Due reports whether the closure date has been reached.
func (c *Closure) Due(now time.Time) bool { return !now.Before(c.ClosingAt) }

// ReadOnlyUntil is the end of the read-only window after completion.
func (c *Closure) ReadOnlyUntil() time.Time {
	return c.ClosedAt.AddDate(0, 0, c.ReadOnlyDays)
}

// Dispute deadlines (GOV-014, DEC-012).
const (
	// DecisionTargetDays is the decision target after the response deadline.
	DecisionTargetDays = 14
)

// DisputeDeadlines are the dates a dispute's parties can see.
type DisputeDeadlines struct {
	ResponseDeadline time.Time
	DecisionTarget   time.Time
	// ResponseOverdue: no response and the deadline passed. The case stays
	// open for reviewer assessment; silence decides nothing.
	ResponseOverdue bool
	// DecisionOverdue: undecided and the decision target passed.
	DecisionOverdue bool
}

// DeadlinesOf computes deadlines for d. responded says a response exists.
func DeadlinesOf(d *Dispute, responseDays int, responded bool, now time.Time) DisputeDeadlines {
	out := DisputeDeadlines{ResponseDeadline: d.OpenedAt.AddDate(0, 0, responseDays)}
	out.DecisionTarget = out.ResponseDeadline.AddDate(0, 0, DecisionTargetDays)
	live := d.Status == DisputeOpen || d.Status == DisputeUnderReview || d.Status == DisputePostingPending
	out.ResponseOverdue = live && !responded && now.After(out.ResponseDeadline)
	out.DecisionOverdue = live && now.After(out.DecisionTarget)
	return out
}
