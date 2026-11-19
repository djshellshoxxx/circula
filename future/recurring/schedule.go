// Package recurring computes reminder windows, never proposals or ledger writes.
package recurring

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"
)

type Template struct {
	Network, ID, Owner, Target, Timezone string
	Hour, Minute                         int
	Active, MembersActive                bool
}
type Occurrence struct {
	Key       string
	Scheduled time.Time
}

// Frequency identifies a structured local-calendar recurrence. Arbitrary cron
// expressions, second-level intervals and host-local timezones are not accepted.
type Frequency string

const (
	Daily   Frequency = "daily"
	Weekly  Frequency = "weekly"
	Monthly Frequency = "monthly"
)

// Schedule selects the local dates on which a template may produce a reminder.
// Weekday is used only by Weekly and DayOfMonth only by Monthly; unused fields
// must be zero. A missing monthly date is skipped, never shifted into another
// month. These isolated-module calendar semantics do not approve core policy.
type Schedule struct {
	Frequency  Frequency
	Weekday    time.Weekday
	DayOfMonth int
}

// Due preserves the original daily-only API. It is equivalent to DueWithSchedule
// using Schedule{Frequency: Daily}.
func Due(t Template, network string, now time.Time) (Occurrence, bool, error) {
	return DueWithSchedule(t, Schedule{Frequency: Daily}, network, now)
}

// DueWithSchedule returns at most today's local-calendar occurrence. Nonexistent
// DST wall times are skipped; repeated wall times choose the first occurrence.
// No historical backlog is replayed, including missed weekly or monthly dates.
//
// The key deliberately excludes the rule, time and schedule revision: editing a
// template cannot issue another reminder for the same logical local date. The
// adapter must enforce UNIQUE(network, template, local-date) in the same
// transaction as notification. A returned occurrence is not a durable firing,
// permission to trade or evidence of current membership authorization.
func DueWithSchedule(t Template, s Schedule, network string, now time.Time) (Occurrence, bool, error) {
	if network == "" || t.Network != network || t.ID == "" || t.Owner == "" || t.Target == "" || t.Owner == t.Target || t.Timezone == "" || t.Timezone == "Local" || t.Hour < 0 || t.Hour > 23 || t.Minute < 0 || t.Minute > 59 {
		return Occurrence{}, false, errors.New("invalid template")
	}
	// encoding/json replaces invalid UTF-8. Reject it before key construction so
	// distinct identity byte strings cannot become the same logical firing key.
	for _, value := range []string{network, t.ID, t.Owner, t.Target, t.Timezone} {
		if !utf8.ValidString(value) {
			return Occurrence{}, false, errors.New("invalid template encoding")
		}
	}
	if err := s.validate(); err != nil {
		return Occurrence{}, false, err
	}
	loc, e := time.LoadLocation(t.Timezone)
	if e != nil {
		return Occurrence{}, false, e
	}
	if !t.Active || !t.MembersActive {
		return Occurrence{}, false, nil
	}
	local := now.In(loc)
	y, m, d := local.Date()
	if (s.Frequency == Weekly && local.Weekday() != s.Weekday) || (s.Frequency == Monthly && d != s.DayOfMonth) {
		return Occurrence{}, false, nil
	}
	chosen, found, err := localOccurrence(y, m, d, t.Hour, t.Minute, loc)
	if err != nil {
		return Occurrence{}, false, err
	}
	if !found || now.Before(chosen) {
		return Occurrence{}, false, nil
	}
	b, _ := json.Marshal([3]string{network, t.ID, local.Format("2006-01-02")})
	h := sha256.Sum256(b)
	return Occurrence{hex.EncodeToString(h[:]), chosen}, true, nil
}

func (s Schedule) validate() error {
	switch s.Frequency {
	case Daily:
		if s.Weekday == 0 && s.DayOfMonth == 0 {
			return nil
		}
	case Weekly:
		if s.Weekday >= time.Sunday && s.Weekday <= time.Saturday && s.DayOfMonth == 0 {
			return nil
		}
	case Monthly:
		if s.Weekday == 0 && s.DayOfMonth >= 1 && s.DayOfMonth <= 31 {
			return nil
		}
	}
	return errors.New("invalid schedule")
}

// localOccurrence examines zone intervals in a bounded 72-hour UTC window. A
// candidate is computed from each interval's exact second offset and accepted
// only if it round-trips to the requested wall time. The earliest valid candidate
// wins a fold. Walking backwards via interval starts also avoids stale end
// bounds in extrapolated timezone rules on the last day of a leap year. This
// avoids time.Date's unspecified fold choice and minute-scanning errors in
// historical IANA zones with offsets that are not whole minutes.
func localOccurrence(year int, month time.Month, day, hour, minute int, loc *time.Location) (time.Time, bool, error) {
	wall := time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
	anchor := time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
	windowStart := anchor.Add(-36 * time.Hour)
	windowEnd := anchor.Add(36 * time.Hour)
	var chosen time.Time
	found := false
	for cursor := windowEnd.In(loc); !cursor.Before(windowStart); {
		_, offset := cursor.Zone()
		start, _ := cursor.ZoneBounds()
		candidate := wall.Add(-time.Duration(offset) * time.Second)
		if !candidate.Before(windowStart) && !candidate.After(windowEnd) {
			local := candidate.In(loc)
			y, m, d := local.Date()
			if y == year && m == month && d == day && local.Hour() == hour && local.Minute() == minute && local.Second() == 0 && (!found || candidate.Before(chosen)) {
				chosen, found = candidate, true
			}
		}
		if start.IsZero() || start.Before(windowStart) {
			break
		}
		if start.After(cursor) {
			return time.Time{}, false, errors.New("invalid timezone interval")
		}
		cursor = start.Add(-time.Nanosecond)
	}
	return chosen, found, nil
}
