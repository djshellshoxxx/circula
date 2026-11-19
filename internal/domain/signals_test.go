package domain

import (
	"testing"
	"time"
)

// SEC-026: signals require explicit activation; provisional thresholds stay bounded.
func TestSEC026DefaultPolicyRequiresActivation(t *testing.T) {
	p := DefaultPolicy()
	if p.ActivitySignalsEnabled {
		t.Fatal("activity signals must stay off until explicitly enabled")
	}
	want := map[string][2]int{
		"velocity_signal_proposals":      {p.VelocitySignalProposals, DefaultVelocitySignalProposals},
		"velocity_signal_window_minutes": {p.VelocitySignalWindowMinutes, DefaultVelocitySignalWindowMinutes},
		"reciprocal_signal_trades":       {p.ReciprocalSignalTrades, DefaultReciprocalSignalTrades},
		"reciprocal_signal_window_days":  {p.ReciprocalSignalWindowDays, DefaultReciprocalSignalWindowDays},
	}
	for k, v := range want {
		if v[0] != v[1] {
			t.Errorf("%s = %d, want %d", k, v[0], v[1])
		}
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

// SEC-026 / GOV-005: each signal threshold accepts its boundary values and
// refuses the value just outside, naming the field. A disabled policy keeps
// its thresholds valid, so switching signals back on cannot meet nonsense.
func TestSEC026PolicyBounds(t *testing.T) {
	type bound struct {
		field    string
		set      func(*Policy, int)
		min, max int
	}
	bounds := []bound{
		{"velocity_signal_proposals", func(p *Policy, v int) { p.VelocitySignalProposals = v }, MinSignalThreshold, MaxSignalThreshold},
		{"velocity_signal_window_minutes", func(p *Policy, v int) { p.VelocitySignalWindowMinutes = v }, MinVelocityWindowMinutes, MaxVelocityWindowMinutes},
		{"reciprocal_signal_trades", func(p *Policy, v int) { p.ReciprocalSignalTrades = v }, MinSignalThreshold, MaxSignalThreshold},
		{"reciprocal_signal_window_days", func(p *Policy, v int) { p.ReciprocalSignalWindowDays = v }, MinReciprocalWindowDays, MaxReciprocalWindowDays},
	}
	for _, b := range bounds {
		for _, enabled := range []bool{true, false} {
			for _, v := range []int{b.min, b.max} {
				p := DefaultPolicy()
				p.ActivitySignalsEnabled = enabled
				b.set(&p, v)
				if err := p.Validate(); err != nil {
					t.Errorf("%s=%d (enabled %v) refused: %v", b.field, v, enabled, err)
				}
			}
			for _, v := range []int{b.min - 1, b.max + 1, 0, -1} {
				if v >= b.min && v <= b.max {
					continue
				}
				p := DefaultPolicy()
				p.ActivitySignalsEnabled = enabled
				b.set(&p, v)
				err := p.Validate()
				var de *Error
				if !asError(err, &de) || de.Code != CodeValidation || len(de.Fields) != 1 || de.Fields[0].Field != b.field {
					t.Errorf("%s=%d (enabled %v): got %v, want one validation error at %s", b.field, v, enabled, err, b.field)
				}
			}
		}
	}
}

func asError(err error, out **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*out = e
	}
	return ok
}

// SEC-026: signal periods are fixed, UTC-epoch-aligned intervals of the
// window length. An instant exactly on a boundary starts the next period.
func TestSEC026SignalPeriodStart(t *testing.T) {
	hour := time.Hour
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		at   time.Time
		want time.Time
	}{
		{base, base},
		{base.Add(time.Millisecond), base},
		{base.Add(hour - time.Millisecond), base},
		{base.Add(hour), base.Add(hour)},
		{base.Add(-time.Millisecond), base.Add(-hour)},
	}
	for _, c := range cases {
		if got := SignalPeriodStart(c.at, hour); !got.Equal(c.want) {
			t.Errorf("SignalPeriodStart(%s) = %s, want %s", c.at, got, c.want)
		}
	}
	// A 30-day period is aligned to the Unix epoch, not to calendar months.
	month := 30 * 24 * time.Hour
	got := SignalPeriodStart(base, month)
	if got.After(base) || base.Sub(got) >= month || got.UnixMilli()%month.Milliseconds() != 0 {
		t.Errorf("30-day period start %s for %s", got, base)
	}
}

// SEC-026: a reciprocal pattern needs completed trades in both directions;
// the count is the total of both directions.
func TestSEC026ReciprocalCount(t *testing.T) {
	cases := []struct {
		ab, ba, want int
		ok           bool
	}{
		{0, 0, 0, false},
		{12, 0, 12, false},
		{0, 12, 12, false},
		{1, 1, 2, true},
		{6, 5, 11, true},
	}
	for _, c := range cases {
		got, ok := ReciprocalCount(c.ab, c.ba)
		if got != c.want || ok != c.ok {
			t.Errorf("ReciprocalCount(%d,%d) = %d,%v want %d,%v", c.ab, c.ba, got, ok, c.want, c.ok)
		}
	}
}

// SEC-026: the evaluation windows follow the policy in force.
func TestSEC026SignalWindows(t *testing.T) {
	p := DefaultPolicy()
	p.VelocitySignalWindowMinutes, p.ReciprocalSignalWindowDays = 90, 3
	if got := p.VelocitySignalWindow(); got != 90*time.Minute {
		t.Errorf("velocity window %s", got)
	}
	if got := p.ReciprocalSignalWindow(); got != 72*time.Hour {
		t.Errorf("reciprocal window %s", got)
	}
}
