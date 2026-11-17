package favorites

import (
	"errors"
	"fmt"
)

// Sentinel errors. Callers should compare with errors.Is. Refusals that could
// reveal whether another member exists, is hidden, blocked, departed or in
// another network all collapse to ErrUnavailable on purpose.
var (
	// ErrUnavailable means the target (or the acting owner) may not be used
	// right now. It is deliberately uniform: it covers unknown, hidden,
	// blocked, suspended, departed, self and cross-network targets.
	ErrUnavailable = errors.New("favorite unavailable")
	// ErrNotFound means the owner has no stored favorite or draft with that key.
	ErrNotFound = errors.New("favorite not found")
	// ErrExists means the favorite already exists with different data.
	ErrExists = errors.New("favorite already exists")
	// ErrLimit means a per-owner cap would be exceeded.
	ErrLimit = errors.New("limit reached")
	// ErrStale means the caller acted on an out-of-date list revision.
	ErrStale = errors.New("stale favorites revision")
	// ErrInvalidOrder means a reorder request was malformed: duplicate,
	// unknown or foreign identifiers, or an oversized list.
	ErrInvalidOrder = errors.New("invalid order")
	// ErrKeyReuse means a draft request key was reused for a different trade or target.
	ErrKeyReuse = errors.New("request key already used for a different draft")
	// ErrTradeUnavailable means the historical trade cannot be repeated.
	ErrTradeUnavailable = errors.New("trade unavailable")
	// ErrDuplicate is returned by stores when a primary key already exists.
	ErrDuplicate = errors.New("duplicate row")
	// ErrPolicy wraps failures of the injected authorization provider.
	// Callers must treat it as a refusal (fail closed).
	ErrPolicy = errors.New("authorization lookup failed")
	// ErrTradeNotFound is what a TradeSource returns for an unknown or
	// not-permitted trade.
	ErrTradeNotFound = errors.New("trade not found")
)

// InputError describes a validation failure of one input field.
type InputError struct{ Field, Reason string }

func (e *InputError) Error() string { return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason) }

func invalid(field, reason string) error { return &InputError{Field: field, Reason: reason} }
