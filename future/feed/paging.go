package feed

import (
	"context"
	"errors"
	"reflect"
	"unicode/utf8"
)

const (
	// MaxPageSize is the largest number of visible references in a page.
	MaxPageSize = 100
	// MaxStoreBatch bounds each database query and its result buffer.
	MaxStoreBatch = MaxPageSize + 1
	// MaxPageCandidates bounds candidate reads and authorization calls per request.
	MaxPageCandidates = 1000
)

var (
	// ErrStore hides database details. Context cancellation remains observable.
	ErrStore = errors.New("feed: store unavailable")
	// ErrAuthorization hides adapter details and discards the complete page.
	ErrAuthorization = errors.New("feed: authorization unavailable")
)

// StoredSource is a unique source identity with its immutable insertion
// sequence. Updates must preserve Sequence and Source.Created. Deletion must
// never permit an identity to be reinserted with a new sequence.
type StoredSource struct {
	Source   Source
	Sequence int64
}

// StoreQuery selects a bounded, ordered batch of active, visible references.
// Kinds is canonical; After is an exclusive keyset position; Limit is 1 to
// MaxStoreBatch. Apply LIMIT in the database using indexed ordering/filtering.
// Do not filter by a traversal's high-water sequence before LIMIT: later
// inserts must be filtered after the bounded candidate read instead.
type StoreQuery struct {
	Network string
	Kinds   []string
	After   *Cursor
	Limit   int
}

// Store is the isolated storage boundary. QueryBatch returns unique identities
// in strict feed order, in the requested network and kinds. HighWater is a
// nonnegative, monotonic maximum insertion sequence for the network. Source
// identity, Created and Sequence are immutable. Calls must honor cancellation.
//
// SQLStore supplies these properties for its module-owned SQLite schema. A Core
// adapter must additionally synchronize references with committed source
// changes and authorize current source data in an appropriate read transaction.
// Implementations must be safe for concurrent use.
type Store interface {
	HighWater(context.Context, string) (int64, error)
	QueryBatch(context.Context, StoreQuery) ([]StoredSource, error)
}

// PageFromStore reads a bounded page from a potentially large history. Cursors
// fence out inserts after the first request, including backdated inserts.
// Existing sources retain immutable order. Visibility and authorization are
// rechecked at read time. This is an insertion-fenced, live-revocation traversal,
// not a historical snapshot of permissions or content.
//
// The trusted allowed adapter must check the authenticated viewer's current
// network membership and permission for each source, including deletion,
// suspension, scheduling and current preferences. It must not trust client
// claims or just cached Source flags. Errors discard the complete page. This
// function does not authenticate callers or replace a source authorization API.
//
// Each request reads at most MaxPageCandidates rows in bounded queries. When
// that budget is exhausted, Items may be empty or shorter than Limit while
// Next remains nonempty. Consumers must follow Next until empty. Encryption
// protects a private scan frontier and sequence. Legacy v2 cursors remain valid
// for Page only; they have no insertion fence and must restart with this API.
func PageFromStore(ctx context.Context, codec *CursorCodec, q Query, store Store, allowed func(context.Context, Source) (bool, error)) (Result, error) {
	b := Binding{Network: q.Network, Viewer: q.Viewer, Filter: q.Filter}
	kinds, _, err := b.canonical()
	if ctx == nil || !codec.ready() || nilStore(store) || allowed == nil || err != nil || q.Limit < 1 || q.Limit > MaxPageSize {
		return Result{}, ErrInvalidQuery
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	var state SnapshotCursor
	var after *Cursor
	if q.Cursor != "" {
		state, err = codec.OpenSnapshot(b, q.Cursor)
		if err != nil {
			return Result{}, ErrInvalidCursor
		}
		position := state.Position
		after = &position
	} else {
		state.IssuedAt = codec.now()
		state.HighWater, err = store.HighWater(ctx, q.Network)
		if err != nil {
			return Result{}, readError(ctx, err)
		}
		if err = ctx.Err(); err != nil {
			return Result{}, err
		}
		if state.HighWater < 0 {
			return Result{}, ErrInvalidSource
		}
	}
	result := Result{Items: make([]Source, 0, q.Limit)}
	if state.HighWater == 0 {
		return result, nil
	}
	seen := make(map[[2]string]struct{})
	sequences := make(map[int64]struct{})
	remaining := MaxPageCandidates
	for remaining > 0 {
		if err = ctx.Err(); err != nil {
			return Result{}, err
		}
		limit := min(MaxStoreBatch, remaining)
		batch, readErr := store.QueryBatch(ctx, StoreQuery{Network: q.Network, Kinds: append([]string(nil), kinds...), After: copyPosition(after), Limit: limit})
		if readErr != nil {
			return Result{}, readError(ctx, readErr)
		}
		if err = ctx.Err(); err != nil {
			return Result{}, err
		}
		if len(batch) > limit {
			return Result{}, ErrInvalidSource
		}
		// Validate the whole bounded batch before exposing any of it.
		previous := after
		for _, row := range batch {
			s := row.Source
			p := positionOf(s)
			identity := [2]string{s.Kind, s.ID}
			if s.Network != q.Network || !hasKind(kinds, s.Kind) || !validPosition(p) || row.Sequence <= 0 ||
				len(s.Member) > maxBindingLen || !utf8.ValidString(s.Member) || (previous != nil && !precedes(*previous, p)) {
				return Result{}, ErrInvalidSource
			}
			if _, exists := seen[identity]; exists {
				return Result{}, ErrInvalidSource
			}
			if _, exists := sequences[row.Sequence]; exists {
				return Result{}, ErrInvalidSource
			}
			seen[identity] = struct{}{}
			sequences[row.Sequence] = struct{}{}
			previous = &p
		}
		remaining -= len(batch)
		for _, row := range batch {
			if err = ctx.Err(); err != nil {
				return Result{}, err
			}
			s := row.Source
			p := positionOf(s)
			if row.Sequence <= state.HighWater && s.Visible && s.Active {
				ok, authErr := allowed(ctx, s)
				if authErr != nil {
					if err = ctx.Err(); err != nil {
						return Result{}, err
					}
					return Result{}, ErrAuthorization
				}
				if err = ctx.Err(); err != nil {
					return Result{}, err
				}
				if ok {
					if len(result.Items) == q.Limit {
						// Do not consume the lookahead row or skip it on Next.
						state.Position = *after
						return finishStoredPage(codec, b, state, result)
					}
					result.Items = append(result.Items, s)
				}
			}
			after = &p
		}
		if len(batch) < limit {
			return result, nil
		}
	}
	// Exhaustion bounds work even if every source is denied or a writer adds
	// backdated rows continuously. The encrypted frontier always advances.
	state.Position = *after
	return finishStoredPage(codec, b, state, result)
}

func finishStoredPage(codec *CursorCodec, b Binding, state SnapshotCursor, result Result) (Result, error) {
	token, err := codec.SealSnapshot(b, state)
	if err != nil {
		return Result{}, err
	}
	result.Next = token
	return result, nil
}

func copyPosition(p *Cursor) *Cursor {
	if p == nil {
		return nil
	}
	copy := *p
	return &copy
}

func nilStore(s Store) bool {
	if s == nil {
		return true
	}
	v := reflect.ValueOf(s)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func readError(ctx context.Context, err error) error {
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return ErrStore
}
