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
	// Version 0 is an unversioned legacy checkpoint. Version 1 fixes the JSON
	// generated_at to ClosedAt, so unchanged data can be verified later.
	FinalExportVersion int
	FinalLedgerSeq     int64
	FinalSHA256        string
	FinalBalanceTotal  int64
	FinalEntriesSum    int64

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
	// ResponseDeadline ends the response period (FR-076): before it, a
	// decision needs a response or a reviewer waiver.
	ResponseDeadline time.Time
	DecisionTarget   time.Time
	// ReReviewCloses is the last instant a re-review may be requested
	// (FR-078); zero while the dispute is undecided.
	ReReviewCloses time.Time
	// ResponseOverdue: no response and the deadline passed. The case stays
	// open for reviewer assessment; silence decides nothing.
	ResponseOverdue bool
	// DecisionOverdue: undecided and the decision target passed.
	DecisionOverdue bool
}

// DeadlinesOf computes deadlines for d. atOpen must be the network policy in
// force when d was opened: dispute timelines change prospectively (GOV-001,
// DEC-012), so the response period and re-review window of a dispute are those
// of that version, never of a later one, shorter or longer. Both are exact
// multiples of 24 hours (QA-013). responded says a response exists.
func DeadlinesOf(d *Dispute, atOpen Policy, responded bool, now time.Time) DisputeDeadlines {
	out := DisputeDeadlines{ResponseDeadline: d.OpenedAt.Add(daysOf(atOpen.ResponsePeriodDays))}
	out.DecisionTarget = out.ResponseDeadline.AddDate(0, 0, DecisionTargetDays)
	out.ReReviewCloses = ReReviewCloses(d, atOpen)
	live := d.Status == DisputeOpen || d.Status == DisputeUnderReview || d.Status == DisputePostingPending
	out.ResponseOverdue = live && !responded && now.After(out.ResponseDeadline)
	out.DecisionOverdue = live && now.After(out.DecisionTarget)
	return out
}

// ReReviewCloses is the end, inclusive, of d's re-review window (FR-078):
// atOpen's ReReviewWindowDays after the decision, atOpen being the policy in
// force when d was opened (see DeadlinesOf). Zero while d is undecided.
func ReReviewCloses(d *Dispute, atOpen Policy) time.Time {
	if d.DecidedAt == nil {
		return time.Time{}
	}
	return d.DecidedAt.Add(daysOf(atOpen.ReReviewWindowDays))
}

func daysOf(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }
