package histbal

import (
	"testing"
	"time"
)

func TestParseInstant(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ok := map[string]time.Time{
		"2026-03-01T12:00:00Z":          now,
		"2026-03-01T13:00:00+01:00":     now,
		"2026-03-01T07:00:00-05:00":     now,
		"2026-03-01T11:59:59.999Z":      now.Add(-time.Millisecond),
		"2026-03-01T11:59:59.5Z":        now.Add(-500 * time.Millisecond),
		"1999-12-31T23:59:59Z":          time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC),
		"2026-03-01T12:00:00.000+00:00": now,
	}
	for in, want := range ok {
		got, err := ParseInstant(in, now)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if !got.Equal(want) || got.Location() != time.UTC {
			t.Errorf("%q: got %v want %v UTC", in, got, want)
		}
	}
	bad := map[string]string{
		"":                          ReasonMissing,
		"2026-03-01":                ReasonFormat,
		"2026-03-01T12:00:00":       ReasonFormat,
		"2026-03-01 12:00:00Z":      ReasonFormat,
		"2026-03-01T12:00Z":         ReasonFormat,
		"2026-03-01T12:00:00+0100":  ReasonFormat,
		"2026-02-30T12:00:00Z":      ReasonFormat,
		"2026-03-01T25:00:00Z":      ReasonFormat,
		"yesterday":                 ReasonFormat,
		"2026-03-01T11:00:00.1234Z": ReasonPrecision,
		"2026-03-01T12:00:00.001Z":  ReasonFuture,
		"2026-03-01T12:30:00+00:29": ReasonFuture,
	}
	for in, reason := range bad {
		_, err := ParseInstant(in, now)
		ie, isIE := err.(*InputError)
		if !isIE || ie.Reason != reason || !IsInputError(err) {
			t.Errorf("%q: got %v, want reason %q", in, err, reason)
		}
	}
}

func TestFormatInstant(t *testing.T) {
	in := time.Date(2026, 3, 1, 13, 0, 0, 7e6, time.FixedZone("x", 3600))
	if got := FormatInstant(in); got != "2026-03-01T12:00:00.007Z" {
		t.Fatalf("got %s", got)
	}
}
