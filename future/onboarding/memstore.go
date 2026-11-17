package onboarding

import (
	"context"
	"sync"
	"time"
)

// MemStore is the in-memory Store test double. It is not durable and exists to
// run the shared contract suite and to power demos that must not touch disk.
type MemStore struct {
	mu       sync.Mutex
	progress map[Key]map[int]Progress
	copies   map[copyKey]Copy
	metrics  map[string]bool
	counts   map[countKey]int64
}

type copyKey struct {
	net string
	ver int
}
type countKey struct {
	net  string
	ver  int
	step Step
}

// NewMemStore returns an empty MemStore.
func NewMemStore() *MemStore {
	return &MemStore{
		progress: map[Key]map[int]Progress{},
		copies:   map[copyKey]Copy{},
		metrics:  map[string]bool{},
		counts:   map[countKey]int64{},
	}
}

func (m *MemStore) LoadProgress(ctx context.Context, k Key) (Progress, bool, error) {
	if err := ctx.Err(); err != nil {
		return Progress{}, false, err
	}
	if err := k.validate(); err != nil {
		return Progress{}, false, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	best, found := Progress{}, false
	for v, p := range m.progress[k] {
		if !found || v > best.Version {
			best, found = p, true
		}
	}
	if found {
		if err := best.Validate(); err != nil {
			return Progress{}, false, err
		}
	}
	return best, found, nil
}

func (m *MemStore) SaveProgress(ctx context.Context, k Key, expected int64, next Progress, newly []Step, now time.Time) (Progress, error) {
	if err := ctx.Err(); err != nil {
		return Progress{}, err
	}
	if err := k.validate(); err != nil {
		return Progress{}, err
	}
	if err := next.Validate(); err != nil {
		return Progress{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.progress[k][next.Version]
	if (ok && cur.Revision != expected) || (!ok && expected != 0) {
		return Progress{}, ErrConflict
	}
	next.Revision = expected + 1
	next.UpdatedAt = now.UTC()
	if m.progress[k] == nil {
		m.progress[k] = map[int]Progress{}
	}
	m.progress[k][next.Version] = next
	if next.Share && m.metrics[k.NetworkID] {
		for _, s := range newly {
			if s < StepCount {
				m.counts[countKey{k.NetworkID, next.Version, s}]++
			}
		}
	}
	return next, nil
}

func (m *MemStore) DeleteMember(ctx context.Context, k Key) error {
	if err := k.validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.progress, k)
	return nil
}

func (m *MemStore) PurgeStale(ctx context.Context, before time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for k, vs := range m.progress {
		for v, p := range vs {
			if p.UpdatedAt.Before(before) {
				delete(vs, v)
				n++
			}
		}
		if len(vs) == 0 {
			delete(m.progress, k)
		}
	}
	return n, nil
}

func (m *MemStore) LoadCopy(ctx context.Context, network string, version int) (Copy, bool, error) {
	if !validID(network) {
		return Copy{}, false, ErrInvalidIdentifier
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.copies[copyKey{network, version}]
	return c, ok, nil
}

func (m *MemStore) SaveCopy(ctx context.Context, c Copy, expected int64, now time.Time) (Copy, error) {
	if !validID(c.NetworkID) {
		return Copy{}, ErrInvalidIdentifier
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := copyKey{c.NetworkID, c.Version}
	cur, ok := m.copies[key]
	if (ok && cur.Revision != expected) || (!ok && expected != 0) {
		return Copy{}, ErrConflict
	}
	c.Revision = expected + 1
	c.UpdatedAt = now.UTC()
	m.copies[key] = c
	return c, nil
}

func (m *MemStore) SetMetricsEnabled(ctx context.Context, network string, enabled bool, by string, now time.Time) error {
	if !validID(network) {
		return ErrInvalidIdentifier
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[network] = enabled
	if !enabled {
		for k := range m.counts {
			if k.net == network {
				delete(m.counts, k)
			}
		}
	}
	return nil
}

func (m *MemStore) MetricsEnabled(ctx context.Context, network string) (bool, error) {
	if !validID(network) {
		return false, ErrInvalidIdentifier
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.metrics[network], nil
}

func (m *MemStore) MetricCounts(ctx context.Context, network string, version int) ([StepCount]int64, error) {
	var out [StepCount]int64
	if !validID(network) {
		return out, ErrInvalidIdentifier
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for s := Terms; s < StepCount; s++ {
		out[s] = m.counts[countKey{network, version, s}]
	}
	return out, nil
}
