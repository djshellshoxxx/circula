// Package feed projects allowed source references, never private trade content.
package feed

import (
	"errors"
	"sort"
	"time"
)

type Source struct {
	Network, Kind, ID, Member string
	Created                   time.Time
	Visible, Active           bool
}

// Cursor is a decoded feed position: the last item of the previous page. It
// crosses the API boundary only as a token sealed by CursorCodec.
type Cursor struct {
	Created  time.Time
	Kind, ID string
}

// Query is one feed page request. Network, Viewer and Filter shape the result
// and are bound into every cursor that Page issues; a cursor is refused under
// any other combination of them.
type Query struct {
	Network string
	Viewer  string
	Filter  Filter
	Limit   int    // 1 to 100
	Cursor  string // Next of the previous page; empty for the first page
}

// Result is one feed page.
type Result struct {
	Items []Source
	Next  string // signed cursor for the following page; empty on the last page
}

var (
	// ErrInvalidQuery reports a malformed query or a missing codec or
	// authorization callback.
	ErrInvalidQuery = errors.New("feed: invalid feed query")
	// ErrInvalidSource reports an adapter row that cannot be shown or paged.
	ErrInvalidSource = errors.New("feed: invalid source")
)

const (
	maxPageLimit = MaxPageSize
	maxRows      = 10000
)

// Page returns one page of the viewer's feed, newest first. A non-empty
// q.Cursor must be a token that codec issued for the same network, viewer and
// filter, within its TTL; otherwise Page returns ErrInvalidCursor. Page fails
// closed when codec or allowed is nil.
//
// Page rechecks current authorization for every source, even those before the
// cursor. Identical source snapshots are de-duplicated; contradictory snapshots
// of the same source fail closed, including stale visible/active copies. No event
// body is stored here; adapters must re-read current source data.
//
// This legacy snapshot helper examines at most 10,000 already-loaded rows. Use
// PageFromStore for ordinary persisted feeds and histories of arbitrary size;
// it applies indexed database limits, an insertion fence and a scan budget.
func Page(codec *CursorCodec, q Query, rows []Source, allowed func(Source) bool) (Result, error) {
	b := Binding{Network: q.Network, Viewer: q.Viewer, Filter: q.Filter}
	kinds, _, err := b.canonical()
	if !codec.ready() || allowed == nil || err != nil || q.Limit < 1 || q.Limit > maxPageLimit || len(rows) > maxRows {
		return Result{}, ErrInvalidQuery
	}
	var after *Cursor
	if q.Cursor != "" {
		pos, err := codec.Open(b, q.Cursor)
		if err != nil {
			return Result{}, ErrInvalidCursor
		}
		after = &pos
	}
	items, err := selectPage(q.Network, kinds, rows, after, q.Limit+1, allowed)
	if err != nil {
		return Result{}, err
	}
	res := Result{Items: items}
	if len(items) > q.Limit {
		res.Items = items[:q.Limit]
		if res.Next, err = codec.Seal(b, positionOf(res.Items[q.Limit-1])); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

// selectPage returns up to limit allowed, de-duplicated sources of the given
// kinds that follow after (all of them when after is nil), in feed order.
func selectPage(network string, kinds []string, rows []Source, after *Cursor, limit int, allowed func(Source) bool) ([]Source, error) {
	out := []Source{}
	seen := map[[2]string]bool{}
	snapshots := map[[2]string]Source{}
	for _, s := range rows {
		if s.Network != network || !hasKind(kinds, s.Kind) {
			continue
		}
		key := [2]string{s.Kind, s.ID}
		if previous, exists := snapshots[key]; exists &&
			(!previous.Created.Equal(s.Created) || previous.Member != s.Member ||
				previous.Visible != s.Visible || previous.Active != s.Active) {
			return nil, ErrInvalidSource
		}
		snapshots[key] = s
		if !s.Visible || !s.Active || !allowed(s) {
			continue
		}
		if !validPosition(positionOf(s)) {
			return nil, ErrInvalidSource
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		if after != nil && !precedes(*after, positionOf(s)) {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return precedes(positionOf(out[i]), positionOf(out[j])) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func positionOf(s Source) Cursor { return Cursor{Created: s.Created, Kind: s.Kind, ID: s.ID} }

// precedes reports whether a comes before b in feed order: newest first, then
// kind, then ID. (Kind, ID) is unique after de-duplication, so the order is total.
func precedes(a, b Cursor) bool {
	if !a.Created.Equal(b.Created) {
		return a.Created.After(b.Created)
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.ID < b.ID
}

func hasKind(kinds []string, k string) bool {
	for _, x := range kinds {
		if x == k {
			return true
		}
	}
	return false
}
