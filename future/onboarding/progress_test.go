package onboarding

import (
	"errors"
	"testing"
)

func TestApplyIdempotentAndCommutative(t *testing.T) {
	base := Progress{Version: 1}
	acts := []Action{{ActComplete, Credit}, {ActSkip, Partner}, {ActComplete, Terms}, {ActShare, 0}, {ActDismiss, 0}}
	apply := func(order []int) Progress {
		p := base
		for _, i := range order {
			for rep := 0; rep < 3; rep++ { // repeating must change nothing
				var err error
				if p, err = p.Apply(acts[i]); err != nil {
					t.Fatal(err)
				}
			}
		}
		return p
	}
	a, b := apply([]int{0, 1, 2, 3, 4}), apply([]int{4, 3, 2, 1, 0})
	if a != b {
		t.Fatalf("not commutative: %+v vs %+v", a, b)
	}
	if !a.IsCompleted(Credit) || !a.IsSkipped(Partner) || !a.Dismissed || !a.Share {
		t.Fatalf("%+v", a)
	}
}

func TestSkipResumeRestore(t *testing.T) {
	p := Progress{Version: 1}
	p, _ = p.Apply(Action{ActSkip, Terms})
	if s, ok := p.Next(); !ok || s != Credit {
		t.Fatalf("skipped step must be passed over, got %v %v", s, ok)
	}
	p, _ = p.Apply(Action{ActResume, Terms})
	if s, _ := p.Next(); s != Terms {
		t.Fatalf("resume should bring step back, got %v", s)
	}
	// Completing clears a skip; skipping a completed step is a no-op.
	p, _ = p.Apply(Action{ActSkip, Credit})
	p, _ = p.Apply(Action{ActComplete, Credit})
	p, _ = p.Apply(Action{ActSkip, Credit})
	if !p.IsCompleted(Credit) || p.IsSkipped(Credit) {
		t.Fatalf("%+v", p)
	}
	p, _ = p.Apply(Action{ActDismiss, 0})
	if _, ok := p.Next(); ok {
		t.Fatal("dismissed checklist offers nothing")
	}
	p, _ = p.Apply(Action{ActSkip, Listing})
	p, _ = p.Apply(Action{ActRestore, 0})
	if p.Dismissed || p.Skipped != 0 || !p.IsCompleted(Credit) {
		t.Fatalf("restore keeps completions only: %+v", p)
	}
}

func TestApplyRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		p    Progress
		a    Action
		want error
	}{
		{"step range", Progress{Version: 1}, Action{ActComplete, StepCount}, ErrInvalidStep},
		{"huge step", Progress{Version: 1}, Action{ActSkip, 200}, ErrInvalidStep},
		{"kind zero", Progress{Version: 1}, Action{}, ErrInvalidAction},
		{"kind high", Progress{Version: 1}, Action{Kind: 99}, ErrInvalidAction},
		{"version 0", Progress{}, Action{ActComplete, Terms}, ErrInvalidState},
		{"mask high", Progress{Version: 1, Completed: 0xE0}, Action{ActComplete, Terms}, ErrInvalidState},
		{"skip mask high", Progress{Version: 1, Skipped: 0x20}, Action{ActComplete, Terms}, ErrInvalidState},
		{"overlap", Progress{Version: 1, Completed: 1, Skipped: 1}, Action{ActComplete, Credit}, ErrInvalidState},
		{"neg revision", Progress{Version: 1, Revision: -1}, Action{ActComplete, Credit}, ErrInvalidState},
	}
	for _, c := range cases {
		if _, err := c.p.Apply(c.a); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v want %v", c.name, err, c.want)
		}
	}
}

func TestParseStepAndKeys(t *testing.T) {
	for s := Terms; s < StepCount; s++ {
		got, err := ParseStep(s.Key())
		if err != nil || got != s {
			t.Fatalf("%v %v", got, err)
		}
	}
	for _, bad := range []string{"", "TERMS", "terms ", "5", "../terms"} {
		if _, err := ParseStep(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	if StepCount.Key() != "" {
		t.Fatal("out of range key")
	}
}

func TestDoneAndNext(t *testing.T) {
	p := Progress{Version: 1}
	for s := Terms; s < StepCount; s++ {
		n, ok := p.Next()
		if !ok || n != s {
			t.Fatalf("want %v got %v", s, n)
		}
		p, _ = p.Complete(s)
	}
	if !p.Done() {
		t.Fatal("not done")
	}
	if _, ok := p.Next(); ok {
		t.Fatal("next after done")
	}
}

func FuzzApplyKeepsStateValid(f *testing.F) {
	f.Add(uint8(0), uint8(0), uint8(1), uint8(0))
	f.Fuzz(func(t *testing.T, done, skip, kind, step uint8) {
		p := Progress{Version: 1, Completed: done, Skipped: skip}
		next, err := p.Apply(Action{ActionKind(kind), Step(step)})
		if err != nil {
			return
		}
		if verr := next.Validate(); verr != nil {
			t.Fatalf("Apply produced invalid state %+v from %+v: %v", next, p, verr)
		}
		again, err := next.Apply(Action{ActionKind(kind), Step(step)})
		if err != nil || again != next {
			t.Fatalf("not idempotent: %+v -> %+v (%v)", next, again, err)
		}
	})
}
