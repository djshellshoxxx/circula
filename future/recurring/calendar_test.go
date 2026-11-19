package recurring

import (
	"testing"
	"time"
)

func TestCalendarBoundaries(t *testing.T) {
	tests := []struct {
		cal       Calendar
		now       string
		due       bool
		scheduled string
	}{{Calendar{Frequency: "daily", Zone: "America/Vancouver", Hour: 1, Minute: 30}, "2026-11-01T09:45:00Z", true, "2026-11-01T08:30:00Z"}, {Calendar{Frequency: "daily", Zone: "America/Vancouver", Hour: 2, Minute: 30}, "2026-03-08T20:00:00Z", false, ""}, {Calendar{Frequency: "monthly", Zone: "UTC", Hour: 10, MonthDay: 31}, "2026-02-28T20:00:00Z", false, ""}, {Calendar{Frequency: "monthly", Zone: "UTC", Hour: 10, MonthDay: 29}, "2028-02-29T10:00:00Z", true, "2028-02-29T10:00:00Z"}, {Calendar{Frequency: "weekly", Zone: "UTC", Hour: 10, Weekday: 1}, "2026-11-16T10:00:00Z", true, "2026-11-16T10:00:00Z"}, {Calendar{Frequency: "daily", Zone: "Australia/Lord_Howe", Hour: 2, Minute: 15}, "2026-10-04T06:00:00Z", false, ""}}
	for _, test := range tests {
		window, ok, err := test.cal.Window("n", "id", at(test.now))
		if err != nil || ok != test.due {
			t.Fatal(test, window, ok, err)
		}
		if ok && !window.Scheduled.Equal(at(test.scheduled)) {
			t.Fatal(test, window)
		}
	}
	if _, _, err := (Calendar{Frequency: "@every 1s", Zone: "UTC"}).Window("n", "id", time.Now()); err == nil {
		t.Fatal("descriptor")
	}
	if _, _, err := (Calendar{Frequency: "daily", Zone: "UTC CRON_TZ=other"}).Window("n", "id", time.Now()); err == nil {
		t.Fatal("timezone override")
	}
}

func TestAuditNextSkipsSecondPhysicalFold(t *testing.T) {
	c := Calendar{Frequency: "daily", Zone: "America/New_York", Hour: 1, Minute: 30}
	next, ok, err := c.Next("n", "template", at("2026-11-01T05:45:00Z"))
	if err != nil || !ok || !next.Scheduled.Equal(at("2026-11-02T06:30:00Z")) {
		t.Fatal("second fold was treated as another logical occurrence", next, ok, err)
	}
	exact, ok, err := c.Next("n", "template", at("2026-11-01T05:30:00Z"))
	if err != nil || !ok || !exact.Scheduled.Equal(at("2026-11-01T05:30:00Z")) {
		t.Fatal(exact, ok, err)
	}
	if _, _, err = c.Next("n", "template", time.Time{}); err == nil {
		t.Fatal("zero clock")
	}
}
