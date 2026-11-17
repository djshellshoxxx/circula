package onboarding

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Key addresses one member of one network. Both fields come from the caller's
// freshly authorized Actor, never from request bodies.
type Key struct {
	NetworkID string
	MemberID  string
}

func validID(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			(i > 0 && (c == '.' || c == '_' || c == '-' || c == ':'))
		if !ok {
			return false
		}
	}
	return true
}

func (k Key) validate() error {
	if !validID(k.NetworkID) || !validID(k.MemberID) {
		return ErrInvalidIdentifier
	}
	return nil
}

// Store is the durable-state contract. The SQLite adapter (SQLStore) and the
// in-memory test double (MemStore) are both verified by the same contract
// suite. Implementations must be safe for concurrent use.
type Store interface {
	// LoadProgress returns the highest-version row for k. Loaded rows are
	// validated; a corrupt row yields ErrInvalidState.
	LoadProgress(ctx context.Context, k Key) (Progress, bool, error)
	// SaveProgress is a compare-and-set. expected is the Revision of the row
	// being replaced for next.Version, or 0 if no row exists at that version.
	// On success it returns the stored row (Revision incremented). A
	// mismatch returns ErrConflict. When the network has aggregate counting
	// enabled and next.Share is true, newly counts those steps in the same
	// transaction. UpdatedAt is set by the store from now.
	SaveProgress(ctx context.Context, k Key, expected int64, next Progress, newly []Step, now time.Time) (Progress, error)
	// DeleteMember removes every progress row of one member (account removal
	// or self-service reset). Aggregate counters are anonymous and kept.
	DeleteMember(ctx context.Context, k Key) error
	// PurgeStale removes progress rows not updated since before, across all
	// networks, and returns the number removed. This is the retention hook.
	PurgeStale(ctx context.Context, before time.Time) (int64, error)

	LoadCopy(ctx context.Context, networkID string, version int) (Copy, bool, error)
	// SaveCopy is a compare-and-set on Copy.Revision (0 = create).
	SaveCopy(ctx context.Context, c Copy, expected int64, now time.Time) (Copy, error)

	// SetMetricsEnabled turns anonymous counting on or off for one network.
	// Turning it off deletes that network's counters (data minimization).
	SetMetricsEnabled(ctx context.Context, networkID string, enabled bool, by string, now time.Time) error
	MetricsEnabled(ctx context.Context, networkID string) (bool, error)
	// MetricCounts returns raw completion counts per step for the network.
	MetricCounts(ctx context.Context, networkID string, version int) ([StepCount]int64, error)
}

// Actor is a trusted snapshot supplied by the caller's current-authorization
// layer on every request. Active must reflect the member's current standing.
type Actor struct {
	NetworkID string
	MemberID  string
	Active    bool
	Admin     bool
	Locale    string
}

func (a Actor) key() (Key, error) {
	k := Key{a.NetworkID, a.MemberID}
	if err := k.validate(); err != nil {
		return k, err
	}
	if !a.Active {
		return k, ErrForbidden
	}
	return k, nil
}

// Tracker applies checklist, copy and metrics operations for an Actor.
type Tracker struct {
	store   Store
	version int
	now     func() time.Time
	links   LinkPolicy
}

// TrackerConfig configures a Tracker. Store is required.
type TrackerConfig struct {
	Store   Store
	Version int              // defaults to CurriculumVersion
	Now     func() time.Time // defaults to time.Now
	Links   LinkPolicy       // trusted link policy for admin copy
}

// NewTracker validates cfg.
func NewTracker(cfg TrackerConfig) (*Tracker, error) {
	if cfg.Store == nil {
		return nil, errors.New("onboarding: store required")
	}
	if cfg.Version == 0 {
		cfg.Version = CurriculumVersion
	}
	if cfg.Version < 1 || cfg.Version > CurriculumVersion {
		return nil, fmt.Errorf("onboarding: unsupported curriculum version %d", cfg.Version)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if err := cfg.Links.validate(); err != nil {
		return nil, err
	}
	return &Tracker{cfg.Store, cfg.Version, cfg.Now, cfg.Links}, nil
}

// Version is the curriculum version this tracker serves.
func (t *Tracker) Version() int { return t.version }

const maxCASRetries = 16

func (t *Tracker) load(ctx context.Context, k Key) (Progress, error) {
	p, ok, err := t.store.LoadProgress(ctx, k)
	if err != nil {
		return Progress{}, err
	}
	if !ok || p.Version < t.version {
		// No row, or only an older curriculum: start this version fresh.
		return Progress{Version: t.version}, nil
	}
	if p.Version > t.version {
		return Progress{}, ErrStaleCurriculum
	}
	return p, nil
}

// Progress returns the actor's own progress (zero state if none).
func (t *Tracker) Progress(ctx context.Context, a Actor) (Progress, error) {
	k, err := a.key()
	if err != nil {
		return Progress{}, err
	}
	return t.load(ctx, k)
}

// Apply performs one idempotent transition on the actor's own progress.
// curriculum is the version the client's form was rendered from; a mismatch is
// ErrStaleCurriculum so an old page cannot overwrite a newer curriculum.
func (t *Tracker) Apply(ctx context.Context, a Actor, curriculum int, act Action) (Progress, error) {
	k, err := a.key()
	if err != nil {
		return Progress{}, err
	}
	if curriculum != t.version {
		return Progress{}, ErrStaleCurriculum
	}
	for i := 0; i < maxCASRetries; i++ {
		cur, err := t.load(ctx, k)
		if err != nil {
			return Progress{}, err
		}
		next, err := cur.Apply(act)
		if err != nil {
			return Progress{}, err
		}
		if next == cur { // idempotent no-op: nothing to write
			return cur, nil
		}
		saved, err := t.store.SaveProgress(ctx, k, cur.Revision, next, newlyCompleted(cur, next), t.now())
		if errors.Is(err, ErrConflict) {
			continue
		}
		return saved, err
	}
	return Progress{}, ErrConflict
}

// Forget deletes the actor's own progress.
func (t *Tracker) Forget(ctx context.Context, a Actor) error {
	k, err := a.key()
	if err != nil {
		return err
	}
	return t.store.DeleteMember(ctx, k)
}

// RemoveMember deletes a member's progress on behalf of a network admin.
func (t *Tracker) RemoveMember(ctx context.Context, admin Actor, memberID string) error {
	if _, err := admin.key(); err != nil {
		return err
	}
	if !admin.Admin {
		return ErrForbidden
	}
	k := Key{admin.NetworkID, memberID}
	if err := k.validate(); err != nil {
		return err
	}
	return t.store.DeleteMember(ctx, k)
}

// PurgeStale applies the retention policy: progress rows untouched for maxAge
// are deleted. It is meant to be called by an operator job; the package never
// schedules it.
func (t *Tracker) PurgeStale(ctx context.Context, maxAge time.Duration) (int64, error) {
	if maxAge < 24*time.Hour {
		return 0, errors.New("onboarding: retention window must be at least 24h")
	}
	return t.store.PurgeStale(ctx, t.now().Add(-maxAge))
}

// MinCohort is the smallest count that Aggregate reports. Smaller counts are
// suppressed. This alone does not prevent differencing across repeated reads;
// see AUDIT.md.
const MinCohort = 5

// StepAggregate is one anonymous counter.
type StepAggregate struct {
	Step       Step
	Completed  int64
	Suppressed bool
}

// SetMetrics lets a network admin turn anonymous aggregate counting on/off.
func (t *Tracker) SetMetrics(ctx context.Context, admin Actor, enabled bool) error {
	if _, err := admin.key(); err != nil {
		return err
	}
	if !admin.Admin {
		return ErrForbidden
	}
	return t.store.SetMetricsEnabled(ctx, admin.NetworkID, enabled, admin.MemberID, t.now())
}

// MetricsEnabled reports the (default false) network switch.
func (t *Tracker) MetricsEnabled(ctx context.Context, a Actor) (bool, error) {
	if _, err := a.key(); err != nil {
		return false, err
	}
	return t.store.MetricsEnabled(ctx, a.NetworkID)
}

// Aggregate returns anonymous completion counts for admins. It returns nil
// when counting is disabled. Counts below MinCohort are suppressed.
func (t *Tracker) Aggregate(ctx context.Context, admin Actor) ([]StepAggregate, error) {
	if _, err := admin.key(); err != nil {
		return nil, err
	}
	if !admin.Admin {
		return nil, ErrForbidden
	}
	on, err := t.store.MetricsEnabled(ctx, admin.NetworkID)
	if err != nil || !on {
		return nil, err
	}
	counts, err := t.store.MetricCounts(ctx, admin.NetworkID, t.version)
	if err != nil {
		return nil, err
	}
	out := make([]StepAggregate, 0, StepCount)
	for s := Terms; s < StepCount; s++ {
		a := StepAggregate{Step: s, Completed: counts[s]}
		if a.Completed < MinCohort {
			a.Completed, a.Suppressed = 0, true
		}
		out = append(out, a)
	}
	return out, nil
}
