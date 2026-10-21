package service

import (
	"context"
	"strings"
	"time"
	_ "time/tzdata" // the zone database is embedded so validation does not depend on the host

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ValidTimeZone reports whether name is an IANA zone such as "Europe/Berlin" or
// "America/Argentina/Buenos_Aires". The empty string is valid and means UTC.
// "Local" and file-system style names are refused.
func ValidTimeZone(name string) bool {
	if name == "" {
		return true
	}
	if len(name) > 64 || name == "Local" {
		return false
	}
	for _, r := range name {
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("/_+-", r)
		if !letter {
			return false
		}
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return false
	}
	// Case-insensitive file systems (macOS) would accept "europe/berlin" from disk:
	// require IANA capitalisation (each segment starts with an upper-case letter).
	for _, seg := range strings.Split(name, "/") {
		if seg == "" || seg[0] < 'A' || seg[0] > 'Z' {
			return false
		}
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

// Location returns the zone for name, falling back to UTC for "" or an unknown name.
func Location(name string) *time.Location {
	if name == "" || !ValidTimeZone(name) {
		return time.UTC
	}
	if l, err := time.LoadLocation(name); err == nil {
		return l
	}
	return time.UTC
}

func prefs(tx store.Tx) (store.PrefsRepo, error) {
	r, ok := tx.(store.PrefsRepo)
	if !ok {
		return nil, domain.Errf(domain.CodeInternal, "store does not support preferences")
	}
	return r, nil
}

// GetTimeZone returns the caller's display time zone ("" means UTC). The API always sends UTC;
// this only drives how pages show times (the preference belongs to the person, so it applies
// in every network they belong to).
func (s *Service) GetTimeZone(ctx context.Context, actor domain.Actor) (string, error) {
	if actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return "", domain.ErrForbidden
	}
	var z string
	err := s.view(ctx, func(tx store.Tx) error {
		r, err := prefs(tx)
		if err != nil {
			return err
		}
		z, err = r.TimeZone(actor.PersonID)
		return err
	})
	return z, err
}

// SetTimeZone stores the caller's display time zone. name must be an IANA zone or "" to go back
// to UTC. The change is audited at platform scope.
func (s *Service) SetTimeZone(ctx context.Context, actor domain.Actor, name string) error {
	if actor.PersonID == "" || !actor.Can(domain.ScopeWrite) {
		return domain.ErrForbidden
	}
	name = strings.TrimSpace(name)
	if !ValidTimeZone(name) {
		return domain.Invalid("time_zone", "must be an IANA time zone such as Europe/Berlin, or empty for UTC")
	}
	return s.update(ctx, func(tx store.Tx) error {
		r, err := prefs(tx)
		if err != nil {
			return err
		}
		before, err := r.TimeZone(actor.PersonID)
		if err != nil {
			return err
		}
		if before == name {
			return nil
		}
		if err := r.SetTimeZone(actor.PersonID, name, s.now()); err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Type: domain.ActorPerson, Action: domain.ActTimeZoneChanged,
			TargetType: "person", TargetID: actor.PersonID, Before: map[string]any{"time_zone": before}, After: map[string]any{"time_zone": name}})
	})
}
