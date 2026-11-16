// Package discovery filters a bounded, authorized listing snapshot.
package discovery

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type Listing struct {
	ID, Network, Title, Category, Area string
	Active, Visible                    bool
	Updated                            time.Time
}
type Query struct {
	Network, Text, Category, Area, AfterID string
	AfterTime                              time.Time
	Limit                                  int
}

// Search orders by updated descending then ID ascending. The caller must retain
// one database snapshot across pages and authenticate/bind its eventual cursor.
func Search(rows []Listing, q Query) ([]Listing, error) {
	if q.Network == "" || q.Limit < 1 || q.Limit > 100 || len(rows) > 10000 || len(q.Text) > 256 || (q.AfterID == "") != q.AfterTime.IsZero() {
		return nil, errors.New("invalid query")
	}
	result := []Listing{}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Network != q.Network || !r.Active || !r.Visible {
			continue
		}
		if r.ID == "" || r.Updated.IsZero() || seen[r.ID] {
			return nil, errors.New("invalid snapshot")
		}
		seen[r.ID] = true
		if q.Category != "" && r.Category != q.Category || q.Area != "" && r.Area != q.Area || !strings.Contains(strings.ToLower(r.Title), strings.ToLower(q.Text)) {
			continue
		}
		if q.AfterID != "" && (r.Updated.After(q.AfterTime) || r.Updated.Equal(q.AfterTime) && r.ID <= q.AfterID) {
			continue
		}
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Updated.Equal(result[j].Updated) {
			return result[i].ID < result[j].ID
		}
		return result[i].Updated.After(result[j].Updated)
	})
	if len(result) > q.Limit {
		result = result[:q.Limit]
	}
	return result, nil
}
