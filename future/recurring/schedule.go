// Package recurring computes reminder windows, never proposals or ledger writes.
package recurring

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
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

// Due supports a daily local-time schedule, with max one reminder for today's
// window. Nonexistent DST wall times are skipped; repeated wall times choose the
// first occurrence. No historical backlog is replayed. The adapter must enforce
// UNIQUE(network, template, local-date) in the same transaction as notification.
func Due(t Template, network string, now time.Time) (Occurrence, bool, error) {
	if network == "" || t.Network != network || t.ID == "" || t.Owner == "" || t.Target == "" || t.Owner == t.Target || t.Timezone == "" || t.Hour < 0 || t.Hour > 23 || t.Minute < 0 || t.Minute > 59 {
		return Occurrence{}, false, errors.New("invalid template")
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
	// Scan a bounded UTC window instead of relying on time.Date's ambiguous DST
	// resolution. This also handles half-hour and historic non-hour transitions.
	anchor := time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
	var chosen time.Time
	for x := anchor.Add(-36 * time.Hour); !x.After(anchor.Add(36 * time.Hour)); x = x.Add(time.Minute) {
		v := x.In(loc)
		yy, mm, dd := v.Date()
		if yy == y && mm == m && dd == d && v.Hour() == t.Hour && v.Minute() == t.Minute {
			chosen = x
			break
		}
	}
	if chosen.IsZero() || now.Before(chosen) {
		return Occurrence{}, false, nil
	}
	b, _ := json.Marshal([3]string{network, t.ID, local.Format("2006-01-02")})
	h := sha256.Sum256(b)
	return Occurrence{hex.EncodeToString(h[:]), chosen}, true, nil
}
