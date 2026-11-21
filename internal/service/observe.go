package service

import (
	"context"
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Observer receives operational events for metrics (OPS-014). Callbacks run
// after the unit of work that caused them has committed, on the calling
// goroutine, and must be fast and must not call back into the Service.
type Observer interface {
	// Posted reports n ledger postings committed by one write unit.
	Posted(n int)
	// Replayed reports n idempotent retries answered from a stored record by
	// one write unit (no new state was created).
	Replayed(n int)
}

// PostingObserver is an optional extension for attempts and failure classes
// (OPS-014, ARC-022). Callbacks run after the write unit finishes, including a
// refused or rolled-back unit, and carry no transaction or member data. A
// failure records the observed outcome, not proof that an uncertain COMMIT
// wrote nothing. The original Observer remains source-compatible.
type PostingObserver interface {
	PostingAttempted(n int)
	PostingFailed(n int, reason string)
}

var postingFailureClasses = [...]string{
	"credit_limit", "validation", "state", "authorization", "rate_limited",
	"unavailable", "cancelled", "timeout", "internal",
}

// PostingFailureClasses returns the bounded vocabulary used for metric labels.
func PostingFailureClasses() []string {
	return append([]string(nil), postingFailureClasses[:]...)
}

// NormalizePostingFailureClass prevents even an unexpected observer input
// from becoming a free-text metric label.
func NormalizePostingFailureClass(reason string) string {
	for _, known := range postingFailureClasses {
		if reason == known {
			return known
		}
	}
	return "internal"
}

func postingFailureClass(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}
	switch domain.CodeOf(err) {
	case domain.CodeInsufficientCredit:
		return "credit_limit"
	case domain.CodeValidation, domain.CodeAmountExceeds, domain.CodeNotFound:
		return "validation"
	case domain.CodeInvalidState, domain.CodeConflict, domain.CodeBlocked, domain.CodeCounterpartyInact:
		return "state"
	case domain.CodeForbidden, domain.CodeUnauthenticated:
		return "authorization"
	case domain.CodeRateLimited:
		return "rate_limited"
	case domain.CodeUnavailable:
		return "unavailable"
	default:
		return "internal"
	}
}

// SetObserver installs o (nil removes it). Call before serving starts.
func (s *Service) SetObserver(o Observer) { s.obs = o }

// unitStats counts events inside one write unit. Units are serialised by the
// store's write gate, so the fields are only touched by one unit at a time.
type unitStats struct {
	posts, replays int
	pendingPosts   int
	postFailures   map[string]int
}

func (s *Service) recordPostingAttempt(err error) {
	if _, ok := s.obs.(PostingObserver); !ok {
		return
	}
	if err == nil {
		s.unit.pendingPosts++
		return
	}
	if s.unit.postFailures == nil {
		s.unit.postFailures = make(map[string]int)
	}
	s.unit.postFailures[postingFailureClass(err)]++
}

func (s *Service) observePostingUnit(stats unitStats, unitErr error) {
	o, ok := s.obs.(PostingObserver)
	if !ok {
		return
	}
	attempts := stats.pendingPosts
	for _, n := range stats.postFailures {
		attempts += n
	}
	if attempts == 0 {
		return
	}
	// A posting that reached the end of postTx is successful only when its
	// enclosing unit commits. Later audit/idempotency/COMMIT errors remain
	// failures, even though the posting function itself returned nil.
	if unitErr != nil && stats.pendingPosts > 0 {
		if stats.postFailures == nil {
			stats.postFailures = make(map[string]int)
		}
		stats.postFailures[postingFailureClass(unitErr)] += stats.pendingPosts
	}
	o.PostingAttempted(attempts)
	for _, reason := range postingFailureClasses {
		if n := stats.postFailures[reason]; n > 0 {
			o.PostingFailed(n, reason)
		}
	}
}
