// Package notifications supplies delivery policy, with no network transport.
package notifications

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

const GenericBody = "You have new Circula activity. Sign in to view it."

type Preference struct {
	Enabled, Active      bool
	Channel, Timezone    string
	QuietStart, QuietEnd int
}

func (p Preference) Allowed(now time.Time) (bool, error) {
	if p.Channel != "email" && p.Channel != "push" || p.Timezone == "" || p.QuietStart < 0 || p.QuietStart > 1439 || p.QuietEnd < 0 || p.QuietEnd > 1439 {
		return false, errors.New("invalid preference")
	}
	loc, e := time.LoadLocation(p.Timezone)
	if e != nil {
		return false, e
	}
	if !p.Enabled || !p.Active {
		return false, nil
	}
	n := now.In(loc)
	m := n.Hour()*60 + n.Minute()
	s, e2 := p.QuietStart, p.QuietEnd
	quiet := s < e2 && m >= s && m < e2 || s > e2 && (m >= s || m < e2)
	return !quiet, nil
}
func DedupeKey(network, event, member, channel, kind string) (string, error) {
	if network == "" || event == "" || member == "" || kind == "" || (channel != "email" && channel != "push") {
		return "", errors.New("invalid event")
	}
	b, _ := json.Marshal([5]string{network, event, member, channel, kind})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Retry schedules at most six attempts; expiration always wins. Attempts counts
// completed failures. Delivery is at-least-once, never exactly-once SMTP.
func Retry(now, expires time.Time, attempts int) (time.Time, bool) {
	if attempts < 1 || attempts >= 6 {
		return time.Time{}, false
	}
	next := now.Add(time.Minute * time.Duration(1<<uint(attempts-1)))
	return next, next.Before(expires)
}
