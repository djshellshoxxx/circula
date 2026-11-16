package feed

import (
	"testing"
	"time"
)

func TestFeedPrivacyAndDedupe(t *testing.T) {
	now := time.Unix(1, 0)
	a := Source{"n", "listing", "1", "m", now, true, true}
	rows := []Source{a, a, {"n", "transaction", "tx", "m", now, true, true}, {"x", "listing", "x", "m", now, true, true}, {"n", "announcement", "2", "m", now, true, true}}
	out, e := Page("n", rows, Cursor{}, 1, func(Source) bool { return true })
	if e != nil || len(out) != 1 || out[0].Kind != "announcement" {
		t.Fatal(out, e)
	}
	out, e = Page("n", rows, Cursor{now, "announcement", "2"}, 10, func(Source) bool { return true })
	if e != nil || len(out) != 1 || out[0].ID != "1" {
		t.Fatal(out, e)
	}
	out, e = Page("n", rows, Cursor{}, 10, func(Source) bool { return false })
	if e != nil || len(out) != 0 {
		t.Fatal("revoked", out, e)
	}
}
