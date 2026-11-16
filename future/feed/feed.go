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
type Cursor struct {
	Created  time.Time
	Kind, ID string
}

// Page rechecks current authorization for every source, even those before the
// cursor. No event body is stored here; adapters must re-read current source data.
func Page(network string, rows []Source, after Cursor, limit int, allowed func(Source) bool) ([]Source, error) {
	if network == "" || limit < 1 || limit > 100 || len(rows) > 10000 || allowed == nil {
		return nil, errors.New("invalid feed query")
	}
	if after.ID == "" && (!after.Created.IsZero() || after.Kind != "") || after.ID != "" && (after.Created.IsZero() || (after.Kind != "listing" && after.Kind != "announcement")) {
		return nil, errors.New("invalid cursor")
	}
	out := []Source{}
	seen := map[[2]string]bool{}
	for _, s := range rows {
		if s.Network != network || !s.Visible || !s.Active || (s.Kind != "listing" && s.Kind != "announcement") || !allowed(s) {
			continue
		}
		if s.ID == "" || s.Created.IsZero() {
			return nil, errors.New("invalid source")
		}
		key := [2]string{s.Kind, s.ID}
		if seen[key] {
			continue
		}
		seen[key] = true
		if after.ID != "" && (s.Created.After(after.Created) || s.Created.Equal(after.Created) && (s.Kind < after.Kind || s.Kind == after.Kind && s.ID <= after.ID)) {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if !a.Created.Equal(b.Created) {
			return a.Created.After(b.Created)
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
