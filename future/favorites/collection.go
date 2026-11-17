package favorites

import (
	"sort"
	"time"
)

// Entry is an owner-scoped saved trading partner with presentation order.
// Persistence, authorization freshness, and atomic writes remain adapter responsibilities.
type Entry struct {
	Favorite
	Order  int
	Pinned bool
	// CreatedAt and UpdatedAt are set by the Service (UTC).
	CreatedAt, UpdatedAt time.Time
}

// VisibleFavorites returns a copy of entries that remain visible to an active owner.
// Duplicate targets are suppressed, and the result is ordered pinned-first.
func VisibleFavorites(owner Member, entries []Entry, targets map[string]Member) []Entry {
	if owner.ID == "" || owner.Network == "" || !owner.Active || owner.Blocked {
		return nil
	}
	out := make([]Entry, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Owner != owner.ID || e.Network != owner.Network || seen[e.Target] {
			continue
		}
		target, ok := targets[e.Target]
		if !ok {
			continue
		}
		if _, err := Save(owner, target, e.Nickname); err != nil {
			continue
		}
		seen[e.Target] = true
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].Target < out[j].Target
	})
	return out
}

// RemoveFavorite produces a copy without the owner's matching favorite.
// Persistence adapters must authenticate the owner and enforce an atomic delete.
func RemoveFavorite(owner Member, entries []Entry, target string) ([]Entry, bool) {
	out := append([]Entry(nil), entries...)
	if owner.ID == "" || owner.Network == "" || !owner.Active || owner.Blocked || target == "" {
		return out, false
	}
	for i, e := range out {
		if e.Network == owner.Network && e.Owner == owner.ID && e.Target == target {
			return append(out[:i:i], out[i+1:]...), true
		}
	}
	return out, false
}
