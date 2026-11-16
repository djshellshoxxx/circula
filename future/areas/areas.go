// Package areas resolves administrator-defined area IDs, without coordinates,
// geolocation, geocoding, inferred locations or external network requests.
package areas

import (
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Area struct {
	Network, ID, Label string
	Visible            bool
}

func Validate(a Area) error {
	if a.Network == "" || a.ID == "" || len(a.ID) > 80 || strings.TrimSpace(a.Label) == "" || !utf8.ValidString(a.Label) || utf8.RuneCountInString(a.Label) > 100 {
		return errors.New("invalid area")
	}
	for _, r := range a.Label {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return errors.New("invalid label")
		}
	}
	return nil
}
func Visible(network string, rows []Area) ([]Area, error) {
	if network == "" || len(rows) > 1000 {
		return nil, errors.New("invalid area query")
	}
	out := []Area{}
	seen := map[string]bool{}
	for _, a := range rows {
		if a.Network != network || !a.Visible {
			continue
		}
		if e := Validate(a); e != nil {
			return nil, e
		}
		if seen[a.ID] {
			return nil, errors.New("duplicate area")
		}
		seen[a.ID] = true
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Label == out[j].Label {
			return out[i].ID < out[j].ID
		}
		return out[i].Label < out[j].Label
	})
	return out, nil
}

// Resolve returns nil for an intentionally unset area; GPS is never required.
func Resolve(network, id string, rows []Area) (*Area, error) {
	if network == "" {
		return nil, errors.New("missing network")
	}
	if id == "" {
		return nil, nil
	}
	all, e := Visible(network, rows)
	if e != nil {
		return nil, e
	}
	for _, a := range all {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, errors.New("area unavailable")
}
