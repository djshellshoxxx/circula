package onboarding

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func newTracker(t *testing.T, st Store) *Tracker {
	t.Helper()
	tr, err := NewTracker(TrackerConfig{Store: st, Now: func() time.Time { return t0 },
		Links: LinkPolicy{PathPrefixes: []string{"/governance/", "/terms"}, Hosts: []string{"example.org"}}})
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

var (
	ctx    = context.Background()
	member = Actor{NetworkID: "net-a", MemberID: "m1", Active: true}
	admin  = Actor{NetworkID: "net-a", MemberID: "adm", Active: true, Admin: true}
)

func TestTrackerFlow(t *testing.T) {
	tr := newTracker(t, NewMemStore())
	p, err := tr.Progress(ctx, member)
	if err != nil || p.Version != 1 || p.Completed != 0 {
		t.Fatalf("%+v %v", p, err)
	}
	p, err = tr.Apply(ctx, member, 1, Action{ActComplete, Terms})
	if err != nil || p.Revision != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	again, err := tr.Apply(ctx, member, 1, Action{ActComplete, Terms})
	if err != nil || again.Revision != 1 {
		t.Fatalf("idempotent replay must not bump revision: %+v %v", again, err)
	}
	// account switch: another member of the same network and the same member
	// id in another network have independent state.
	other, _ := tr.Progress(ctx, Actor{NetworkID: "net-a", MemberID: "m2", Active: true})
	cross, _ := tr.Progress(ctx, Actor{NetworkID: "net-b", MemberID: "m1", Active: true})
	if other.Completed != 0 || cross.Completed != 0 {
		t.Fatal("progress leaked between accounts")
	}
}

func TestTrackerAuthorization(t *testing.T) {
	tr := newTracker(t, NewMemStore())
	suspended := member
	suspended.Active = false
	bad := member
	bad.MemberID = ""
	for name, a := range map[string]Actor{"suspended": suspended, "empty id": bad, "zero": {}} {
		if _, err := tr.Progress(ctx, a); err == nil {
			t.Errorf("%s: read allowed", name)
		}
		if _, err := tr.Apply(ctx, a, 1, Action{ActComplete, Terms}); err == nil {
			t.Errorf("%s: write allowed", name)
		}
		if _, err := tr.Copy(ctx, a); err == nil {
			t.Errorf("%s: copy read allowed", name)
		}
	}
	if _, err := tr.SaveCopy(ctx, member, Copy{Intro: "x"}, 0); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin saved copy: %v", err)
	}
	if err := tr.SetMetrics(ctx, member, true); !errors.Is(err, ErrForbidden) {
		t.Fatal("non-admin set metrics")
	}
	if _, err := tr.Aggregate(ctx, member); !errors.Is(err, ErrForbidden) {
		t.Fatal("non-admin read aggregate")
	}
	if err := tr.RemoveMember(ctx, member, "m2"); !errors.Is(err, ErrForbidden) {
		t.Fatal("non-admin remove")
	}
	susAdmin := admin
	susAdmin.Active = false
	if _, err := tr.SaveCopy(ctx, susAdmin, Copy{Intro: "x"}, 0); err == nil {
		t.Fatal("suspended admin saved copy")
	}
}

func TestTrackerStaleCurriculum(t *testing.T) {
	st := NewMemStore()
	tr := newTracker(t, st)
	if _, err := tr.Apply(ctx, member, 2, Action{ActComplete, Terms}); !errors.Is(err, ErrStaleCurriculum) {
		t.Fatalf("form from other curriculum accepted: %v", err)
	}
	// A newer curriculum already stored must never be overwritten by this build.
	st.SaveProgress(ctx, Key{"net-a", "m1"}, 0, Progress{Version: 2, Completed: 1}, nil, t0)
	if _, err := tr.Progress(ctx, member); !errors.Is(err, ErrStaleCurriculum) {
		t.Fatal(err)
	}
	if _, err := tr.Apply(ctx, member, 1, Action{ActComplete, Credit}); !errors.Is(err, ErrStaleCurriculum) {
		t.Fatal(err)
	}
	got, _, _ := st.LoadProgress(ctx, Key{"net-a", "m1"})
	if got.Version != 2 || got.Completed != 1 {
		t.Fatalf("newer curriculum row modified: %+v", got)
	}
}

func TestTrackerCorruptLoadedStateRejected(t *testing.T) {
	st := NewMemStore()
	st.progress[Key{"net-a", "m1"}] = map[int]Progress{1: {Version: 1, Completed: 0xFF}}
	tr := newTracker(t, st)
	if _, err := tr.Progress(ctx, member); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("%v", err)
	}
	if _, err := tr.Apply(ctx, member, 1, Action{ActComplete, Terms}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("%v", err)
	}
}

func TestTrackerConcurrentTabsNoLostSteps(t *testing.T) {
	tr := newTracker(t, NewMemStore())
	var wg sync.WaitGroup
	for rep := 0; rep < 5; rep++ {
		for s := Terms; s < StepCount; s++ {
			wg.Add(1)
			go func(s Step) {
				defer wg.Done()
				if _, err := tr.Apply(ctx, member, 1, Action{ActComplete, s}); err != nil {
					t.Error(err)
				}
			}(s)
		}
	}
	wg.Wait()
	if p, _ := tr.Progress(ctx, member); !p.Done() {
		t.Fatalf("%+v", p)
	}
}

// conflictStore forces permanent CAS conflicts to prove the retry loop ends.
type conflictStore struct{ *MemStore }

func (conflictStore) SaveProgress(context.Context, Key, int64, Progress, []Step, time.Time) (Progress, error) {
	return Progress{}, ErrConflict
}

func TestTrackerRetryBounded(t *testing.T) {
	tr := newTracker(t, conflictStore{NewMemStore()})
	if _, err := tr.Apply(ctx, member, 1, Action{ActComplete, Terms}); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestForgetRemoveAndPurge(t *testing.T) {
	now := t0
	tr, _ := NewTracker(TrackerConfig{Store: NewMemStore(), Now: func() time.Time { return now }})
	m2 := Actor{NetworkID: "net-a", MemberID: "m2", Active: true}
	tr.Apply(ctx, member, 1, Action{ActComplete, Terms})
	tr.Apply(ctx, m2, 1, Action{ActComplete, Terms})
	if err := tr.Forget(ctx, member); err != nil {
		t.Fatal(err)
	}
	if p, _ := tr.Progress(ctx, member); p.Completed != 0 {
		t.Fatal("forget failed")
	}
	// An admin of another network cannot remove net-a members.
	foreign := Actor{NetworkID: "net-b", MemberID: "adm", Active: true, Admin: true}
	tr.RemoveMember(ctx, foreign, "m2")
	if p, _ := tr.Progress(ctx, m2); p.Completed == 0 {
		t.Fatal("cross-network removal")
	}
	if err := tr.RemoveMember(ctx, admin, "m2"); err != nil {
		t.Fatal(err)
	}
	if p, _ := tr.Progress(ctx, m2); p.Completed != 0 {
		t.Fatal("admin removal failed")
	}
	if _, err := tr.PurgeStale(ctx, time.Hour); err == nil {
		t.Fatal("tiny retention window accepted")
	}
	tr.Apply(ctx, member, 1, Action{ActComplete, Terms})
	now = now.Add(100 * 24 * time.Hour)
	n, err := tr.PurgeStale(ctx, 90*24*time.Hour)
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
}

func TestMetricsDefaultOffAndAggregateAnonymous(t *testing.T) {
	st := NewMemStore()
	tr := newTracker(t, st)
	for i := 0; i < 7; i++ {
		a := Actor{NetworkID: "net-a", MemberID: "u" + string(rune('a'+i)), Active: true}
		tr.Apply(ctx, a, 1, Action{ActShare, 0})
		tr.Apply(ctx, a, 1, Action{ActComplete, Terms})
	}
	if agg, err := tr.Aggregate(ctx, admin); err != nil || agg != nil {
		t.Fatalf("aggregate visible while off: %v %v", agg, err)
	}
	if c, _ := st.MetricCounts(ctx, "net-a", 1); c != [StepCount]int64{} {
		t.Fatalf("events recorded while default-off: %v", c)
	}
	tr.SetMetrics(ctx, admin, true)
	for i := 0; i < 5; i++ {
		a := Actor{NetworkID: "net-a", MemberID: "v" + string(rune('a'+i)), Active: true}
		tr.Apply(ctx, a, 1, Action{ActShare, 0})
		tr.Apply(ctx, a, 1, Action{ActComplete, Credit})
	}
	// four non-sharing members are never counted
	for i := 0; i < 4; i++ {
		a := Actor{NetworkID: "net-a", MemberID: "w" + string(rune('a'+i)), Active: true}
		tr.Apply(ctx, a, 1, Action{ActComplete, Credit})
	}
	agg, _ := tr.Aggregate(ctx, admin)
	if agg[Credit].Completed != 5 || agg[Credit].Suppressed || !agg[Terms].Suppressed || agg[Terms].Completed != 0 {
		t.Fatalf("%+v", agg)
	}
}
