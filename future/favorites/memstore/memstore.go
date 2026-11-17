// Package memstore is an in-memory favorites.Store with real transaction
// semantics (serialized writers, atomic commit, full rollback on error). It is
// a reference implementation and test fixture; it is NOT durable and must not
// be used to claim restart persistence.
package memstore

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
)

type fkey struct{ network, owner, target string }
type okey struct{ network, owner string }
type ikey struct{ network, owner, key string }

type state struct {
	favs    map[fkey]favorites.Entry
	rev     map[okey]int64
	intents map[ikey]favorites.Intent
}

func (s *state) clone() *state {
	c := &state{favs: make(map[fkey]favorites.Entry, len(s.favs)), rev: make(map[okey]int64, len(s.rev)), intents: make(map[ikey]favorites.Intent, len(s.intents))}
	for k, v := range s.favs {
		c.favs[k] = v
	}
	for k, v := range s.rev {
		c.rev[k] = v
	}
	for k, v := range s.intents {
		c.intents[k] = v
	}
	return c
}

// Store implements favorites.Store.
type Store struct {
	w   sync.Mutex // serializes writers
	mu  sync.RWMutex
	cur *state
}

// New returns an empty Store.
func New() *Store {
	return &Store{cur: &state{favs: map[fkey]favorites.Entry{}, rev: map[okey]int64{}, intents: map[ikey]favorites.Intent{}}}
}

var errReadOnly = errors.New("memstore: write in read-only transaction")

func (s *Store) snapshot() *state {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

func (s *Store) View(ctx context.Context, fn func(favorites.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn(&tx{st: s.snapshot(), ro: true})
}

func (s *Store) Update(ctx context.Context, fn func(favorites.Tx) error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.w.Lock()
	defer s.w.Unlock()
	work := s.snapshot().clone()
	if err := fn(&tx{st: work}); err != nil {
		return err
	}
	s.mu.Lock()
	s.cur = work
	s.mu.Unlock()
	return nil
}

type tx struct {
	st *state
	ro bool
}

func (t *tx) Revision(_ context.Context, network, owner string) (int64, error) {
	return t.st.rev[okey{network, owner}], nil
}

func (t *tx) BumpRevision(_ context.Context, network, owner string) (int64, error) {
	if t.ro {
		return 0, errReadOnly
	}
	k := okey{network, owner}
	t.st.rev[k]++
	return t.st.rev[k], nil
}

func (t *tx) ListFavorites(_ context.Context, network, owner string, limit int) ([]favorites.Entry, error) {
	var out []favorites.Entry
	for k, e := range t.st.favs {
		if k.network == network && k.owner == owner {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].Target < out[j].Target
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (t *tx) GetFavorite(_ context.Context, network, owner, target string) (favorites.Entry, bool, error) {
	e, ok := t.st.favs[fkey{network, owner, target}]
	return e, ok, nil
}

func (t *tx) InsertFavorite(_ context.Context, e favorites.Entry) error {
	if t.ro {
		return errReadOnly
	}
	k := fkey{e.Network, e.Owner, e.Target}
	if _, ok := t.st.favs[k]; ok {
		return favorites.ErrDuplicate
	}
	if e.Owner == e.Target {
		return errors.New("memstore: self favorite")
	}
	t.st.favs[k] = e
	return nil
}

func (t *tx) UpdateFavorite(_ context.Context, e favorites.Entry) (bool, error) {
	if t.ro {
		return false, errReadOnly
	}
	k := fkey{e.Network, e.Owner, e.Target}
	old, ok := t.st.favs[k]
	if !ok {
		return false, nil
	}
	old.Nickname, old.Pinned, old.Order, old.UpdatedAt = e.Nickname, e.Pinned, e.Order, e.UpdatedAt
	t.st.favs[k] = old
	return true, nil
}

func (t *tx) DeleteFavorite(_ context.Context, network, owner, target string) (bool, error) {
	if t.ro {
		return false, errReadOnly
	}
	k := fkey{network, owner, target}
	_, ok := t.st.favs[k]
	delete(t.st.favs, k)
	return ok, nil
}

func (t *tx) GetIntent(_ context.Context, network, owner, key string) (favorites.Intent, bool, error) {
	in, ok := t.st.intents[ikey{network, owner, key}]
	return in, ok, nil
}

func (t *tx) InsertIntent(_ context.Context, in favorites.Intent) error {
	if t.ro {
		return errReadOnly
	}
	k := ikey{in.Network, in.Owner, in.Key}
	if _, ok := t.st.intents[k]; ok {
		return favorites.ErrDuplicate
	}
	for _, o := range t.st.intents {
		if o.Intent == in.Intent {
			return favorites.ErrDuplicate
		}
	}
	t.st.intents[k] = in
	return nil
}

func (t *tx) DeleteIntent(_ context.Context, network, owner, key string) error {
	if t.ro {
		return errReadOnly
	}
	delete(t.st.intents, ikey{network, owner, key})
	return nil
}

func (t *tx) CountIntents(_ context.Context, network, owner string, now time.Time) (int, error) {
	n := 0
	for k, in := range t.st.intents {
		if k.network == network && k.owner == owner && in.ExpiresAt.After(now) {
			n++
		}
	}
	return n, nil
}

func (t *tx) PurgeMember(_ context.Context, network, member string) (favorites.PurgeResult, error) {
	var r favorites.PurgeResult
	if t.ro {
		return r, errReadOnly
	}
	affected := map[string]bool{}
	for k := range t.st.favs {
		if k.network == network && (k.owner == member || k.target == member) {
			delete(t.st.favs, k)
			r.Favorites++
			if k.owner != member {
				affected[k.owner] = true
			}
		}
	}
	// Owners who lost a favorite to the purge see a changed list revision.
	for o := range affected {
		t.st.rev[okey{network, o}]++
	}
	for k, in := range t.st.intents {
		if k.network == network && (k.owner == member || in.Target == member) {
			delete(t.st.intents, k)
			r.Intents++
		}
	}
	if _, ok := t.st.rev[okey{network, member}]; ok {
		delete(t.st.rev, okey{network, member})
		r.States++
	}
	return r, nil
}

func (t *tx) PurgeExpiredIntents(_ context.Context, cutoff time.Time) (int64, error) {
	if t.ro {
		return 0, errReadOnly
	}
	var n int64
	for k, in := range t.st.intents {
		if !in.ExpiresAt.After(cutoff) {
			delete(t.st.intents, k)
			n++
		}
	}
	return n, nil
}
