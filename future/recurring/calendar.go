package recurring

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/robfig/cron/v3"
)

// Calendar accepts structured, at-most-daily recurrences. Arbitrary cron strings,
// TZ overrides, seconds and @every descriptors are never accepted.
type Calendar struct {
	Frequency    string
	Zone         string
	Hour, Minute int
	Weekday      int
	MonthDay     int
}

func (c Calendar) parse() (cron.Schedule, error) {
	if len(c.Zone) > 128 || c.Zone == "" || c.Zone == "Local" || strings.ContainsAny(c.Zone, " \t\r\n=") || c.Hour < 0 || c.Hour > 23 || c.Minute < 0 || c.Minute > 59 {
		return nil, errors.New("invalid calendar")
	}
	if _, err := time.LoadLocation(c.Zone); err != nil {
		return nil, err
	}
	dom, dow := "*", "*"
	switch c.Frequency {
	case "daily":
		if c.Weekday != 0 || c.MonthDay != 0 {
			return nil, errors.New("unused calendar fields")
		}
	case "weekly":
		if c.Weekday < 0 || c.Weekday > 6 || c.MonthDay != 0 {
			return nil, errors.New("invalid weekday")
		}
		dow = fmt.Sprint(c.Weekday)
	case "monthly":
		if c.MonthDay < 1 || c.MonthDay > 31 || c.Weekday != 0 {
			return nil, errors.New("invalid month day")
		}
		dom = fmt.Sprint(c.MonthDay)
	default:
		return nil, errors.New("unsupported frequency")
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return parser.Parse(fmt.Sprintf("CRON_TZ=%s %d %d %s * %s", c.Zone, c.Minute, c.Hour, dom, dow))
}

// Window only considers today's local date. A missed spring wall time and a
// nonexistent monthly day are skipped. The first physical time wins in a fold.
// No downtime backfill is generated; this is a reminder, never a payment.
func (c Calendar) Window(network, id string, now time.Time) (Occurrence, bool, error) {
	_, err := c.parse()
	if err != nil || network == "" || id == "" || !utf8.ValidString(network) || !utf8.ValidString(id) || now.IsZero() {
		if err == nil {
			err = errors.New("invalid occurrence context")
		}
		return Occurrence{}, false, err
	}
	loc, _ := time.LoadLocation(c.Zone)
	local := now.In(loc)
	y, m, d := local.Date()
	if (c.Frequency == "weekly" && int(local.Weekday()) != c.Weekday) || (c.Frequency == "monthly" && d != c.MonthDay) {
		return Occurrence{}, false, nil
	}
	// Resolve the exact local minute, including historical offsets containing
	// seconds. The bounded interval resolver is shared with the retained pure
	// helper; cron still validates the structured rule and finds future dates.
	chosen, found, err := localOccurrence(y, m, d, c.Hour, c.Minute, loc)
	if err != nil {
		return Occurrence{}, false, err
	}
	if !found || now.Before(chosen) {
		return Occurrence{}, false, nil
	}
	b, _ := json.Marshal([3]string{network, id, local.Format("2006-01-02")})
	sum := sha256.Sum256(b)
	return Occurrence{hex.EncodeToString(sum[:]), chosen}, true, nil
}

// Next supplies the next logical occurrence, including an occurrence exactly
// at now. A fold's second physical wall time never becomes a separate skip
// target after the first occurrence has passed.
func (c Calendar) Next(network, id string, now time.Time) (Occurrence, bool, error) {
	_, err := c.parse()
	if err != nil || network == "" || id == "" || !utf8.ValidString(network) || !utf8.ValidString(id) || now.IsZero() {
		if err == nil {
			err = errors.New("invalid next context")
		}
		return Occurrence{}, false, err
	}
	// Use cron only to enumerate eligible Gregorian dates at fixed UTC noon.
	// Its local minute truncation can skip entire dates when an IANA offset
	// contains seconds. Resolve each date's exact wall time independently.
	loc, _ := time.LoadLocation(c.Zone)
	y, m, d := now.In(loc).Date()
	dateCalendar := c
	dateCalendar.Zone, dateCalendar.Hour, dateCalendar.Minute = "UTC", 12, 0
	schedule, err := dateCalendar.parse()
	if err != nil {
		return Occurrence{}, false, err
	}
	anchor := time.Date(y, m, d, 12, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
	for i := 0; i < 5; i++ {
		next := schedule.Next(anchor)
		if next.IsZero() {
			return Occurrence{}, false, nil
		}
		y, m, d := next.Date()
		chosen, found, err := localOccurrence(y, m, d, c.Hour, c.Minute, loc)
		if err != nil {
			return Occurrence{}, false, err
		}
		if found && !chosen.Before(now) {
			first, due, err := c.Window(network, id, chosen)
			if err != nil || due {
				return first, due, err
			}
		}
		anchor = next
	}
	return Occurrence{}, false, errors.New("next logical occurrence exceeds calendar bound")
}
