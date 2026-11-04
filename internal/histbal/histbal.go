// Package histbal holds the adapter-facing contract for a member's balance at
// an exact past instant (FR-054, DAT-050, API-083, UX-061).
//
// The service does not yet provide a timestamp-based read: its BalanceAt takes
// a ledger_seq prefix, which FR-054 forbids as a substitute. Until the
// coordinator settles that read (one fixed snapshot, posted_at <= as_of), the
// adapters depend only on Reader and are tested against fakes. Nothing here
// touches the ledger, the store or the service; no route is registered.
package histbal

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Result is the historical projection. It deliberately carries no credit
// limit, reserved or available amount: those are current values (FR-054).
type Result struct {
	AsOf              time.Time // normalized UTC, millisecond precision
	BalanceMinor      int64     // signed sum of the caller's entries with posted_at <= AsOf
	Decimals          int       // immutable network scale
	LedgerSeq         int64     // snapshot high-water sequence, not a cursor
	HasPostingsByAsOf bool      // false exactly when no entry was included
}

// Reader is the pending authenticated read. Implementations MUST resolve the
// caller's own account from actor (never another member's) and MUST apply
// the timestamp predicate inside one read snapshot.
type Reader interface {
	HistoricalBalance(ctx context.Context, actor domain.Actor, netID string, asOf time.Time) (Result, error)
}

// InputError explains why an instant was refused. Adapters map it to 400
// invalid-query (API) or a timestamp field error (HTML).
type InputError struct {
	Reason string
}

func (e *InputError) Error() string { return "invalid instant: " + e.Reason }

// Reasons, worded for the person correcting the value.
const (
	ReasonMissing   = "enter a date and time"
	ReasonFormat    = "use a full RFC 3339 date-time with Z or a numeric offset, for example 2026-03-01T12:00:00Z"
	ReasonPrecision = "use at most millisecond precision (3 fractional digits)"
	ReasonFuture    = "choose an instant that is not in the future"
)

// IsInputError reports whether err is an *InputError.
func IsInputError(err error) bool {
	var ie *InputError
	return errors.As(err, &ie)
}

var (
	shape     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)
	fracShape = regexp.MustCompile(`\.(\d+)`)
)

// ParseInstant accepts an RFC 3339 date-time with an explicit Z or numeric
// offset and at most millisecond precision, refuses instants after now, and
// returns the instant in UTC. It never infers a time zone or a time of day.
func ParseInstant(raw string, now time.Time) (time.Time, error) {
	if raw == "" {
		return time.Time{}, &InputError{ReasonMissing}
	}
	if !shape.MatchString(raw) {
		return time.Time{}, &InputError{ReasonFormat}
	}
	if m := fracShape.FindStringSubmatch(raw); m != nil && len(m[1]) > 3 {
		return time.Time{}, &InputError{ReasonPrecision}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, &InputError{ReasonFormat}
	}
	t = t.UTC()
	if t.After(now) {
		return time.Time{}, &InputError{ReasonFuture}
	}
	return t, nil
}

// FormatInstant renders an instant as UTC with Z and millisecond precision
// (API-005), e.g. 2026-03-01T12:00:00.000Z.
func FormatInstant(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}
