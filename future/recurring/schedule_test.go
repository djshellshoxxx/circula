package recurring

import (
	"testing"
	"time"
)

func at(s string) time.Time {
	t, e := time.Parse(time.RFC3339, s)
	if e != nil {
		panic(e)
	}
	return t
}
func TestDSTAndBoundedCatchup(t *testing.T) {
	x := Template{"n", "t", "a", "b", "America/Vancouver", 1, 30, true, true}
	a, ok, e := Due(x, "n", at("2026-11-01T08:45:00Z"))
	if e != nil || !ok || !a.Scheduled.Equal(at("2026-11-01T08:30:00Z")) {
		t.Fatal(a, ok, e)
	}
	b, ok, e := Due(x, "n", at("2026-11-01T09:45:00Z"))
	if e != nil || !ok || a.Key != b.Key {
		t.Fatal("duplicate DST window", a, b, e)
	}
	x.Hour = 2
	a, ok, e = Due(x, "n", at("2026-03-08T20:00:00Z"))
	if e != nil || ok {
		t.Fatal("nonexistent time", a, e)
	}
	x.Active = false
	if _, ok, e = Due(x, "n", at("2026-03-09T20:00:00Z")); e != nil || ok {
		t.Fatal("disabled")
	}
	if _, _, e = Due(x, "other", time.Now()); e == nil {
		t.Fatal("tenant")
	}
}
