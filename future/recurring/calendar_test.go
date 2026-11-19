package recurring

import (
	"testing"
	"time"
)

func TestWeeklyCalendarBoundaries(t *testing.T) {
	x := validTemplate()
	rule := Schedule{Frequency: Weekly, Weekday: time.Monday}
	for _, tc := range []struct {
		now string
		due bool
	}{
		{"2023-12-31T23:59:59Z", false},
		{"2024-01-01T11:59:59.999999999Z", false},
		{"2024-01-01T12:00:00Z", true},
		{"2024-01-01T23:59:59.999999999Z", true},
		{"2024-01-02T12:00:00Z", false},
		{"2024-01-08T12:00:00Z", true},
	} {
		t.Run(tc.now, func(t *testing.T) {
			occurrence, due, err := DueWithSchedule(x, rule, x.Network, at(tc.now))
			if err != nil || due != tc.due {
				t.Fatalf("occurrence=%+v due=%v want=%v err=%v", occurrence, due, tc.due, err)
			}
		})
	}
	first, _, _ := DueWithSchedule(x, rule, x.Network, at("2024-01-01T12:00:00Z"))
	next, _, _ := DueWithSchedule(x, rule, x.Network, at("2024-01-08T12:00:00Z"))
	if first.Key == next.Key {
		t.Fatal("distinct weekly occurrences reused a logical key")
	}
}

func TestMonthlyCalendarBoundaries(t *testing.T) {
	x := validTemplate()
	for _, tc := range []struct {
		name string
		day  int
		now  string
		due  bool
	}{
		{"leap day", 29, "2024-02-29T12:00:00Z", true},
		{"century is not leap", 29, "1900-02-28T23:59:59Z", false},
		{"400-year leap", 29, "2000-02-29T12:00:00Z", true},
		{"missing February day", 31, "2024-02-29T23:59:59Z", false},
		{"no roll into March", 31, "2024-03-01T12:00:00Z", false},
		{"31st exists", 31, "2024-03-31T12:00:00Z", true},
		{"missing April day", 31, "2024-04-30T23:59:59Z", false},
		{"no catch-up in May", 31, "2024-05-01T12:00:00Z", false},
		{"next valid month", 31, "2024-05-31T12:00:00Z", true},
		{"before exact boundary", 1, "2024-06-01T11:59:59.999999999Z", false},
		{"exact boundary", 1, "2024-06-01T12:00:00Z", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			occurrence, due, err := DueWithSchedule(x, Schedule{Frequency: Monthly, DayOfMonth: tc.day}, x.Network, at(tc.now))
			if err != nil || due != tc.due {
				t.Fatalf("occurrence=%+v due=%v want=%v err=%v", occurrence, due, tc.due, err)
			}
		})
	}
}

func TestScheduleUsesLocalCalendarDate(t *testing.T) {
	x := validTemplate()
	x.Timezone = "Pacific/Kiritimati"
	x.Hour = 0
	now := at("2023-12-31T10:00:00Z") // Monday, January 1 in UTC+14.
	for _, rule := range []Schedule{{Frequency: Daily}, {Frequency: Weekly, Weekday: time.Monday}, {Frequency: Monthly, DayOfMonth: 1}} {
		occurrence, due, err := DueWithSchedule(x, rule, x.Network, now)
		if err != nil || !due || !occurrence.Scheduled.Equal(now) {
			t.Fatalf("local date ignored for %+v: occurrence=%+v due=%v err=%v", rule, occurrence, due, err)
		}
	}
	x.Timezone = "Pacific/Apia"
	x.Hour = 12
	if _, due, err := DueWithSchedule(x, Schedule{Frequency: Monthly, DayOfMonth: 30}, x.Network, at("2011-12-30T23:00:00Z")); err != nil || due {
		t.Fatalf("skipped local date rolled into December 31: due=%v err=%v", due, err)
	}
}

func TestScheduleDSTPolicies(t *testing.T) {
	for _, tc := range []struct {
		name, zone, now, scheduled string
		hour, minute, day          int
	}{
		{"one-hour gap", "America/New_York", "2024-03-10T18:00:00Z", "", 2, 30, 10},
		{"one-hour fold first", "America/New_York", "2024-11-03T05:45:00Z", "2024-11-03T05:30:00Z", 1, 30, 3},
		{"one-hour fold second", "America/New_York", "2024-11-03T06:45:00Z", "2024-11-03T05:30:00Z", 1, 30, 3},
		{"half-hour gap", "Australia/Lord_Howe", "2024-10-06T06:00:00Z", "", 2, 15, 6},
		{"half-hour fold first", "Australia/Lord_Howe", "2024-04-06T14:50:00Z", "2024-04-06T14:45:00Z", 1, 45, 7},
		{"half-hour fold second", "Australia/Lord_Howe", "2024-04-06T15:20:00Z", "2024-04-06T14:45:00Z", 1, 45, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := validTemplate()
			x.Timezone, x.Hour, x.Minute = tc.zone, tc.hour, tc.minute
			for _, rule := range []Schedule{{Frequency: Daily}, {Frequency: Weekly, Weekday: time.Sunday}, {Frequency: Monthly, DayOfMonth: tc.day}} {
				occurrence, due, err := DueWithSchedule(x, rule, x.Network, at(tc.now))
				if err != nil || due != (tc.scheduled != "") {
					t.Fatalf("rule=%+v occurrence=%+v due=%v err=%v", rule, occurrence, due, err)
				}
				if due && !occurrence.Scheduled.Equal(at(tc.scheduled)) {
					t.Fatalf("rule=%+v scheduled=%v want=%s", rule, occurrence.Scheduled, tc.scheduled)
				}
			}
		})
	}
}

func TestScheduleValidation(t *testing.T) {
	invalid := []Schedule{
		{}, {Frequency: "@every 1s"}, {Frequency: "0 12 * * *"},
		{Frequency: Daily, Weekday: time.Monday}, {Frequency: Daily, DayOfMonth: 1},
		{Frequency: Weekly, Weekday: -1}, {Frequency: Weekly, Weekday: 7},
		{Frequency: Weekly, Weekday: time.Monday, DayOfMonth: 1},
		{Frequency: Monthly}, {Frequency: Monthly, DayOfMonth: -1},
		{Frequency: Monthly, DayOfMonth: 32}, {Frequency: Monthly, DayOfMonth: 1, Weekday: time.Monday},
	}
	x := validTemplate()
	for _, rule := range invalid {
		if _, due, err := DueWithSchedule(x, rule, x.Network, at("2024-01-01T12:00:00Z")); err == nil || due {
			t.Fatalf("invalid schedule accepted: %+v due=%v err=%v", rule, due, err)
		}
	}
	// Validation still runs for inactive templates: invalid persisted data must
	// not silently appear valid until a later activation.
	x.Active = false
	if _, due, err := DueWithSchedule(x, Schedule{}, x.Network, time.Time{}); err == nil || due {
		t.Fatalf("inactive invalid rule accepted: due=%v err=%v", due, err)
	}
}

func TestLogicalKeyCompatibilityAndScope(t *testing.T) {
	x := validTemplate()
	now := at("2024-01-01T12:00:00Z")
	// This fixed output digest is not a credential. Split its golden value
	// across literals to avoid generic API-key false positives.
	const expectedDigest = "f6537855a3907885" +
		"98ea596e48565eee" +
		"8fb1c59a1860401f" +
		"8b12f362f5ad7c37"
	for _, rule := range []Schedule{{Frequency: Daily}, {Frequency: Weekly, Weekday: time.Monday}, {Frequency: Monthly, DayOfMonth: 1}} {
		occurrence, due, err := DueWithSchedule(x, rule, x.Network, now)
		if err != nil || !due || occurrence.Key != expectedDigest {
			t.Fatalf("rule edit changed daily key: %+v occurrence=%+v due=%v err=%v", rule, occurrence, due, err)
		}
	}
	x.Hour = 10
	occurrence, due, err := Due(x, x.Network, now)
	if err != nil || !due || occurrence.Key != expectedDigest {
		t.Fatalf("time edit changed same-date key: occurrence=%+v due=%v err=%v", occurrence, due, err)
	}
	x.Network = "other-network"
	occurrence, due, err = Due(x, x.Network, now)
	if err != nil || !due || occurrence.Key == expectedDigest {
		t.Fatal("network namespace not preserved", occurrence, due, err)
	}
	x = validTemplate()
	x.ID = "other-template"
	occurrence, due, err = Due(x, x.Network, now)
	if err != nil || !due || occurrence.Key == expectedDigest {
		t.Fatal("template namespace not preserved", occurrence, due, err)
	}
}

func TestScheduleMembershipAndActivation(t *testing.T) {
	for _, rule := range []Schedule{{Frequency: Daily}, {Frequency: Weekly, Weekday: time.Monday}, {Frequency: Monthly, DayOfMonth: 1}} {
		x := validTemplate()
		x.MembersActive = false
		if _, due, err := DueWithSchedule(x, rule, x.Network, at("2024-01-01T12:00:00Z")); err != nil || due {
			t.Fatalf("inactive member accepted: %+v due=%v err=%v", rule, due, err)
		}
		x.MembersActive, x.Active = true, false
		if _, due, err := DueWithSchedule(x, rule, x.Network, at("2024-01-01T12:00:00Z")); err != nil || due {
			t.Fatalf("paused template accepted: %+v due=%v err=%v", rule, due, err)
		}
		if _, due, err := DueWithSchedule(x, rule, "another-network", at("2024-01-01T12:00:00Z")); err == nil || due {
			t.Fatalf("tenant mismatch accepted: %+v due=%v err=%v", rule, due, err)
		}
	}
}

// Compare interval resolution against an independent exhaustive UTC-minute
// oracle for modern minute-offset zones. Historical second offsets have their
// own exact-instant regression. Include year boundaries beyond the explicit
// transition tables, where the timezone database uses extrapolated rules.
func TestCalendarResolutionAgainstMinuteOracle(t *testing.T) {
	zones := []string{"UTC", "America/New_York", "Australia/Lord_Howe", "Europe/Dublin", "Asia/Kathmandu", "Pacific/Kiritimati"}
	dates := []string{"2024-03-10T23:59:00Z", "2024-04-07T23:59:00Z", "2024-10-06T23:59:00Z", "2024-11-03T23:59:00Z", "2040-12-31T23:59:00Z", "2041-01-01T23:59:00Z"}
	for _, zone := range zones {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatal(err)
		}
		for _, date := range dates {
			now := at(date)
			y, m, d := now.In(loc).Date()
			anchor := time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
			first := make(map[int]time.Time)
			for probe := anchor.Add(-36 * time.Hour); !probe.After(anchor.Add(36 * time.Hour)); probe = probe.Add(time.Minute) {
				local := probe.In(loc)
				py, pm, pd := local.Date()
				if py != y || pm != m || pd != d || local.Second() != 0 {
					continue
				}
				key := local.Hour()*60 + local.Minute()
				if _, exists := first[key]; !exists {
					first[key] = probe
				}
			}
			for hour := 0; hour < 24; hour++ {
				for minute := 0; minute < 60; minute += 15 {
					x := validTemplate()
					x.Timezone, x.Hour, x.Minute = zone, hour, minute
					occurrence, due, err := Due(x, x.Network, now)
					want, exists := first[hour*60+minute]
					wantDue := exists && !now.Before(want)
					if err != nil || due != wantDue || (due && !occurrence.Scheduled.Equal(want)) {
						t.Fatalf("zone=%s date=%s wall=%02d:%02d occurrence=%+v due=%v want=%v wantDue=%v err=%v", zone, date, hour, minute, occurrence, due, want, wantDue, err)
					}
				}
			}
		}
	}

}

func FuzzScheduleInvariants(f *testing.F) {
	f.Add("template", uint8(0), int64(0), uint8(0), uint8(0), uint8(0))
	f.Add("template\xff", uint8(1), int64(1730616300), uint8(1), uint8(30), uint8(2))
	f.Add("é|☃", uint8(2), int64(-2208946161), uint8(12), uint8(0), uint8(5))
	zones := []string{"UTC", "America/New_York", "Australia/Lord_Howe", "Pacific/Kiritimati", "Pacific/Apia", "Europe/Paris"}
	f.Fuzz(func(t *testing.T, id string, kind uint8, seconds int64, hour, minute, zone uint8) {
		if len(id) > 1024 {
			return
		}
		// Keep generated dates in a broad reproducible range around the epoch.
		now := time.Unix(seconds%(200*366*24*3600), 0).UTC()
		x := validTemplate()
		x.ID, x.Timezone, x.Hour, x.Minute = id, zones[int(zone)%len(zones)], int(hour), int(minute)
		loc, err := time.LoadLocation(x.Timezone)
		if err != nil {
			t.Fatal(err)
		}
		local := now.In(loc)
		rules := []Schedule{{Frequency: Daily}, {Frequency: Weekly, Weekday: local.Weekday()}, {Frequency: Monthly, DayOfMonth: local.Day()}}
		rule := rules[int(kind)%len(rules)]
		occurrence, due, err := DueWithSchedule(x, rule, x.Network, now)
		if err != nil || !due {
			if due || occurrence != (Occurrence{}) {
				t.Fatal("invalid/absent schedule returned an occurrence", occurrence, due, err)
			}
			return
		}
		if occurrence.Scheduled.After(now) || len(occurrence.Key) != 64 {
			t.Fatal("future or invalid-key occurrence", occurrence, now)
		}
		scheduled := occurrence.Scheduled.In(loc)
		sy, sm, sd := scheduled.Date()
		ny, nm, nd := local.Date()
		if sy != ny || sm != nm || sd != nd || scheduled.Hour() != x.Hour || scheduled.Minute() != x.Minute || scheduled.Second() != 0 || scheduled.Nanosecond() != 0 {
			t.Fatal("occurrence is not the exact requested local minute", occurrence, local, x)
		}
		exact, exactDue, exactErr := DueWithSchedule(x, rule, x.Network, occurrence.Scheduled)
		if exactErr != nil || !exactDue || exact != occurrence {
			t.Fatal("exact boundary is not idempotent", occurrence, exact, exactDue, exactErr)
		}
		// Immediately before a midnight occurrence, the previous date may still
		// legitimately be due. It must never return this date's logical key.
		if prior, early, earlyErr := DueWithSchedule(x, rule, x.Network, occurrence.Scheduled.Add(-time.Nanosecond)); earlyErr != nil || (early && prior.Key == occurrence.Key) {
			t.Fatal("reminder occurs before its boundary", occurrence, early, earlyErr)
		}
	})
}
