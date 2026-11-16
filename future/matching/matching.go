// Package matching produces advisory category matches from public listing data.
package matching

import (
	"errors"
	"sort"
)

type Listing struct {
	ID, Network, Member, Kind, Category string
	Active, Visible, OptedIn            bool
}
type Pair struct{ Offer, Request, Reason string }

// Suggest bounds work to 500 listings. Exclusion is called in both directions for
// blocks/dismissals. All membership and preference fields must come from the DB.
// There is no posting API and no balance/message input.
func Suggest(network string, rows []Listing, limit int, exclude func(string, string) bool) ([]Pair, error) {
	if network == "" || len(rows) > 500 || limit < 1 || limit > 100 || exclude == nil {
		return nil, errors.New("invalid matching request")
	}
	eligible := []Listing{}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Network != network || !r.Active || !r.Visible || !r.OptedIn || r.Category == "" {
			continue
		}
		if r.ID == "" || r.Member == "" || seen[r.ID] || (r.Kind != "offer" && r.Kind != "request") {
			return nil, errors.New("invalid snapshot")
		}
		seen[r.ID] = true
		eligible = append(eligible, r)
	}
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].ID < eligible[j].ID })
	out := []Pair{}
	for _, a := range eligible {
		if a.Kind != "offer" {
			continue
		}
		for _, b := range eligible {
			if b.Kind != "request" || a.Member == b.Member || a.Category != b.Category || exclude(a.ID, b.ID) || exclude(b.ID, a.ID) {
				continue
			}
			out = append(out, Pair{a.ID, b.ID, "same_category"})
			if len(out) == limit {
				return out, nil
			}
		}
	}
	return out, nil
}
