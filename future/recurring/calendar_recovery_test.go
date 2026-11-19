package recurring

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestRecoveredCalendarIdentityEncoding(t *testing.T) {
	c := Calendar{Frequency: "daily", Zone: "UTC", Hour: 12}
	now := at("2024-01-01T12:00:00Z")
	methods := []struct {
		name string
		call func(string, string, time.Time) (Occurrence, bool, error)
	}{{"Window", c.Window}, {"Next", c.Next}}
	malformed := []string{"\xff", "\x80", "\xc0\xaf", "\xed\xa0\x80", "\xe2\x82", "\xf4\x90\x80\x80"}
	for _, method := range methods {
		for _, field := range []string{"network", "id"} {
			for i, invalid := range malformed {
				t.Run(fmt.Sprintf("%s/%s/%d", method.name, field, i), func(t *testing.T) {
					network, id := "network", "template"
					if field == "network" {
						network += invalid
					} else {
						id += invalid
					}
					occurrence, ok, err := method.call(network, id, now)
					if err == nil || ok || occurrence != (Occurrence{}) {
						t.Fatalf("malformed identity accepted: occurrence=%+v ok=%v err=%v", occurrence, ok, err)
					}
				})
			}
		}
		for _, valid := range []string{"literal-\uFFFD", "cafe\u0301", "réseau/北/🧭"} {
			t.Run(method.name+"/valid/"+valid, func(t *testing.T) {
				encoded, err := json.Marshal([3]string{valid, valid, "2024-01-01"})
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(encoded)
				occurrence, ok, err := method.call(valid, valid, now)
				if err != nil || !ok || !occurrence.Scheduled.Equal(now) || occurrence.Key != hex.EncodeToString(sum[:]) {
					t.Fatalf("valid identity/key changed: occurrence=%+v ok=%v err=%v", occurrence, ok, err)
				}
			})
		}
	}
}

func TestRecoveredCalendarHistoricalOffsetSeconds(t *testing.T) {
	// Paris mean time on this date was UTC+00:09:21. The expected instants
	// are literal wall times minus that offset, independent of cron/resolution.
	for _, minuteOfDay := range []int{0, 12 * 60, 12*60 + 15, 23*60 + 45} {
		for _, frequency := range []string{"daily", "weekly", "monthly"} {
			c := Calendar{Frequency: frequency, Zone: "Europe/Paris", Hour: minuteOfDay / 60, Minute: minuteOfDay % 60}
			if frequency == "weekly" {
				c.Weekday = int(time.Monday)
			}
			if frequency == "monthly" {
				c.MonthDay = 1
			}
			when := time.Date(1900, time.January, 1, c.Hour, c.Minute, 0, 0, time.UTC).Add(-561 * time.Second)
			t.Run(fmt.Sprintf("%s/%02d:%02d", frequency, c.Hour, c.Minute), func(t *testing.T) {
				occurrence, ok, err := c.Window("n", "id", when)
				if err != nil || !ok || !occurrence.Scheduled.Equal(when) {
					t.Fatalf("historical window: occurrence=%+v ok=%v err=%v want=%v", occurrence, ok, err, when)
				}
				if prior, early, err := c.Window("n", "id", when.Add(-time.Nanosecond)); err != nil || (early && prior.Key == occurrence.Key) {
					t.Fatalf("early historical window: occurrence=%+v ok=%v err=%v", prior, early, err)
				}
				for _, probe := range []time.Time{when.Add(-time.Minute), when} {
					next, ok, err := c.Next("n", "id", probe)
					if err != nil || !ok || !next.Scheduled.Equal(when) {
						t.Fatalf("historical next from %v: occurrence=%+v ok=%v err=%v want=%v", probe, next, ok, err, when)
					}
				}
				nextDay := 2
				if frequency == "weekly" {
					nextDay = 8
				}
				month := time.January
				if frequency == "monthly" {
					month, nextDay = time.February, 1
				}
				wantNext := time.Date(1900, month, nextDay, c.Hour, c.Minute, 0, 0, time.UTC).Add(-561 * time.Second)
				for _, probe := range []time.Time{when.Add(time.Nanosecond), when.Add(61 * time.Second)} {
					next, ok, err := c.Next("n", "id", probe)
					if err != nil || !ok || !next.Scheduled.Equal(wantNext) {
						t.Errorf("next historical date from %v: occurrence=%+v ok=%v err=%v want=%v", probe, next, ok, err, wantNext)
					}
				}
			})
		}
	}
}

func TestRecoveredCalendarWindowAgainstMinuteOracle(t *testing.T) {
	// Exhaustive UTC-minute enumeration is independent of both the retained
	// zone-interval resolver and the recovered cron iterator.
	zones := []string{"UTC", "America/New_York", "Australia/Lord_Howe", "Europe/Dublin", "Asia/Kathmandu", "Pacific/Kiritimati"}
	dates := []string{"2024-03-10T23:59:00Z", "2024-04-07T23:59:00Z", "2024-10-06T23:59:00Z", "2024-11-03T23:59:00Z", "2040-12-31T23:59:00Z", "2041-01-01T23:59:00Z"}
	for _, zone := range zones {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatal(err)
		}
		for _, date := range dates {
			now := at(date)
			local := now.In(loc)
			y, m, d := local.Date()
			anchor := time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
			first := make(map[int]time.Time)
			for probe := anchor.Add(-36 * time.Hour); !probe.After(anchor.Add(36 * time.Hour)); probe = probe.Add(time.Minute) {
				candidate := probe.In(loc)
				py, pm, pd := candidate.Date()
				if py != y || pm != m || pd != d || candidate.Second() != 0 {
					continue
				}
				key := candidate.Hour()*60 + candidate.Minute()
				if _, exists := first[key]; !exists {
					first[key] = probe
				}
			}
			for hour := 0; hour < 24; hour++ {
				for minute := 0; minute < 60; minute += 15 {
					for _, frequency := range []string{"daily", "weekly", "monthly"} {
						c := Calendar{Frequency: frequency, Zone: zone, Hour: hour, Minute: minute}
						if frequency == "weekly" {
							c.Weekday = int(local.Weekday())
						}
						if frequency == "monthly" {
							c.MonthDay = d
						}
						occurrence, due, err := c.Window("n", "id", now)
						want, exists := first[hour*60+minute]
						wantDue := exists && !now.Before(want)
						if err != nil || due != wantDue || (due && !occurrence.Scheduled.Equal(want)) {
							t.Fatalf("frequency=%s zone=%s date=%s wall=%02d:%02d occurrence=%+v due=%v want=%v wantDue=%v err=%v", frequency, zone, date, hour, minute, occurrence, due, want, wantDue, err)
						}
					}
				}
			}
		}
	}
}

func TestRecoveredCalendarRejectsZeroClock(t *testing.T) {
	c := Calendar{Frequency: "daily", Zone: "UTC"}
	// Unlike the legacy pure Due helper, the recovered service calendar has an
	// explicit invalid-clock contract. Composition must not silently broaden it.
	for _, method := range []func(string, string, time.Time) (Occurrence, bool, error){c.Window, c.Next} {
		occurrence, ok, err := method("n", "id", time.Time{})
		if err == nil || ok || occurrence != (Occurrence{}) {
			t.Fatalf("zero service clock accepted: occurrence=%+v ok=%v err=%v", occurrence, ok, err)
		}
	}
}

func TestRecoveredCalendarNextEligibleDate(t *testing.T) {
	for _, tc := range []struct {
		name string
		cal  Calendar
		now  string
		want string
	}{
		{"daily gap", Calendar{Frequency: "daily", Zone: "America/New_York", Hour: 2, Minute: 30}, "2024-03-10T05:00:00Z", "2024-03-11T06:30:00Z"},
		{"weekly gap", Calendar{Frequency: "weekly", Zone: "America/New_York", Hour: 2, Minute: 30, Weekday: int(time.Sunday)}, "2024-03-10T05:00:00Z", "2024-03-17T06:30:00Z"},
		{"monthly gap", Calendar{Frequency: "monthly", Zone: "America/New_York", Hour: 2, Minute: 30, MonthDay: 10}, "2024-03-10T05:00:00Z", "2024-04-10T06:30:00Z"},
		{"daily fold", Calendar{Frequency: "daily", Zone: "America/New_York", Hour: 1, Minute: 30}, "2024-11-03T05:45:00Z", "2024-11-04T06:30:00Z"},
		{"weekly fold", Calendar{Frequency: "weekly", Zone: "America/New_York", Hour: 1, Minute: 30, Weekday: int(time.Sunday)}, "2024-11-03T05:45:00Z", "2024-11-10T06:30:00Z"},
		{"monthly fold", Calendar{Frequency: "monthly", Zone: "America/New_York", Hour: 1, Minute: 30, MonthDay: 3}, "2024-11-03T05:45:00Z", "2024-12-03T06:30:00Z"},
		{"half-hour gap", Calendar{Frequency: "daily", Zone: "Australia/Lord_Howe", Hour: 2, Minute: 15}, "2024-10-05T15:00:00Z", "2024-10-06T15:15:00Z"},
		{"daily skipped date", Calendar{Frequency: "daily", Zone: "Pacific/Apia", Hour: 12}, "2011-12-29T23:00:00Z", "2011-12-30T22:00:00Z"},
		{"monthly skipped date", Calendar{Frequency: "monthly", Zone: "Pacific/Apia", Hour: 12, MonthDay: 30}, "2011-12-29T23:00:00Z", "2012-01-29T22:00:00Z"},
		{"missing month day", Calendar{Frequency: "monthly", Zone: "UTC", Hour: 12, MonthDay: 31}, "2024-04-01T00:00:00Z", "2024-05-31T12:00:00Z"},
		{"leap February", Calendar{Frequency: "monthly", Zone: "UTC", Hour: 12, MonthDay: 29}, "2024-02-01T00:00:00Z", "2024-02-29T12:00:00Z"},
		{"non-leap century", Calendar{Frequency: "monthly", Zone: "UTC", Hour: 12, MonthDay: 29}, "2100-02-01T00:00:00Z", "2100-03-29T12:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, ok, err := tc.cal.Next("n", "id", at(tc.now))
			if err != nil || !ok || !next.Scheduled.Equal(at(tc.want)) {
				t.Fatalf("next=%+v ok=%v err=%v want=%s", next, ok, err, tc.want)
			}
			window, due, err := tc.cal.Window("n", "id", next.Scheduled)
			if err != nil || !due || window != next {
				t.Fatalf("next is not its logical window: next=%+v window=%+v due=%v err=%v", next, window, due, err)
			}
		})
	}
}
