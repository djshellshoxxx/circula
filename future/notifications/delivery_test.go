package notifications

import (
	"testing"
	"time"
)

func TestQuietAndOptOut(t *testing.T) {
	p := Preference{true, true, "email", "America/Vancouver", 22 * 60, 8 * 60}
	for _, stamp := range []string{"2026-11-01T08:30:00Z", "2026-11-01T09:30:00Z"} {
		now, _ := time.Parse(time.RFC3339, stamp)
		ok, e := p.Allowed(now)
		if e != nil || ok {
			t.Fatal("DST quiet", ok, e)
		}
	}
	p.Enabled = false
	ok, e := p.Allowed(time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC))
	if ok || e != nil {
		t.Fatal("opt out")
	}
}
func TestDedupeAndRetry(t *testing.T) {
	a, _ := DedupeKey("n", "e", "m", "email", "new")
	b, _ := DedupeKey("x", "e", "m", "email", "new")
	if a == b {
		t.Fatal("scope collision")
	}
	now := time.Unix(0, 0)
	if _, ok := Retry(now, now.Add(time.Hour), 6); ok {
		t.Fatal("retry cap")
	}
	if _, ok := Retry(now, now.Add(time.Minute), 1); ok {
		t.Fatal("expiry")
	}
}
