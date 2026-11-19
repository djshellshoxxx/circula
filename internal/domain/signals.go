package domain

import "time"

// Activity signals (SEC-026) surface unusual activity to a network's
// administrators and dispute reviewers. A signal is a review prompt, never a
// finding: it does not block, delay, change or reverse any trade, and the
// members it names are not told. Advanced fraud analytics is deferred.
//
// Two kinds exist:
//   - velocity: one member created more than VelocitySignalProposals trade
//     proposals in the VelocitySignalWindow before the latest one;
//   - reciprocal: two members completed more than ReciprocalSignalTrades trades
//     with each other, with at least one in each direction, in the
//     ReciprocalSignalWindow before the latest completion.
//
// Counts slide with the triggering event; deduplication uses fixed periods of
// the window length aligned to the Unix epoch (SignalPeriodStart), so at most
// one signal per network, kind, member (or pair) and period is recorded.

// Signal kinds.
const (
	SignalVelocity   = "velocity"
	SignalReciprocal = "reciprocal"
)

// Provisional defaults and bounds (SEC-026). 12-quality-test-plan.md sets no
// recommended defaults and 09 open question 10 is undecided: these values are
// provisional, for the owner to confirm. They are conservative in the sense of
// rarely raising a signal for ordinary use.
const (
	DefaultVelocitySignalProposals     = 20 // more than 20 proposals ...
	DefaultVelocitySignalWindowMinutes = 60 // ... in 60 minutes
	DefaultReciprocalSignalTrades      = 10 // more than 10 completed trades, both directions ...
	DefaultReciprocalSignalWindowDays  = 30 // ... in 30 days

	MinSignalThreshold       = 1
	MaxSignalThreshold       = 1000
	MinVelocityWindowMinutes = 1
	MaxVelocityWindowMinutes = 7 * 24 * 60 // one week
	MinReciprocalWindowDays  = 1
	MaxReciprocalWindowDays  = 365
)

// VelocitySignalWindow is the velocity evaluation window.
func (p Policy) VelocitySignalWindow() time.Duration {
	return time.Duration(p.VelocitySignalWindowMinutes) * time.Minute
}

// ReciprocalSignalWindow is the reciprocal-pattern evaluation window.
func (p Policy) ReciprocalSignalWindow() time.Duration {
	return time.Duration(p.ReciprocalSignalWindowDays) * 24 * time.Hour
}

// SignalPeriodStart returns the start of the fixed period of length window
// that contains at: periods are aligned to the Unix epoch in UTC
// milliseconds, and an instant on a boundary belongs to the period it starts.
func SignalPeriodStart(at time.Time, window time.Duration) time.Time {
	w := window.Milliseconds()
	if w <= 0 {
		return at.UTC()
	}
	ms := at.UnixMilli()
	start := ms - ms%w
	if ms%w < 0 { // instants before the epoch round down, not toward zero
		start -= w
	}
	return time.UnixMilli(start).UTC()
}

// ReciprocalCount combines the completed trades of a pair in each direction.
// ok is false unless both directions occur: one-way trading is not a
// reciprocal pattern, however frequent.
func ReciprocalCount(aToB, bToA int) (total int, ok bool) {
	return aToB + bToA, aToB > 0 && bToA > 0
}
