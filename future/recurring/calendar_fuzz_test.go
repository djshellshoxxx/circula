package recurring

import (
	"testing"
	"time"
)

func FuzzCalendar(f *testing.F) {
	f.Add("daily", "UTC", 10, 0, 0, 0)
	f.Add("weekly", "America/New_York", 1, 30, 0, 0)
	f.Add("monthly", "UTC", 10, 0, 0, 31)
	f.Fuzz(func(t *testing.T, frequency, zone string, hour, minute, weekday, monthday int) {
		if len(zone) > 256 || len(frequency) > 256 {
			return
		}
		calendar := Calendar{frequency, zone, hour, minute, weekday, monthday}
		now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
		_, _, _ = calendar.Window("n", "id", now)
		next, ok, err := calendar.Next("n", "id", now)
		if err == nil && ok && next.Scheduled.Before(now) {
			t.Fatal("next occurrence moved backward", next)
		}
	})
}
