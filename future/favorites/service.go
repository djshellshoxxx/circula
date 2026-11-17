package favorites

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// Defaults and hard bounds.
const (
	DefaultMaxFavorites = 200
	HardMaxFavorites    = 1000
	DefaultMaxDrafts    = 100
	DefaultDraftTTL     = 24 * time.Hour
	DefaultPageLimit    = 50
	MaxPageLimit        = 200
)

// Config wires a Service. Store, Policy and Enabled are required. Trades may be
// nil, in which case trade-based quick repeat is unavailable.
type Config struct {
	Store  Store
	Policy Policy
	// Enabled is the explicit per-network/per-owner opt-in gate. There is
	// deliberately no default: the host must decide, and a false or failing
	// answer disables every actor operation (ErrDisabled or ErrPolicy).
	Enabled func(ctx context.Context, a Actor) (bool, error)
	Trades  TradeSource
	Clock   func() time.Time
	// MaxFavorites caps stored favorites per owner and network (default 200, max 1000).
	MaxFavorites int
	// MaxDrafts caps unexpired quick-repeat drafts per owner (default 100).
	MaxDrafts int
	// DraftTTL is how long a RequestKey keeps returning the same draft (default 24h).
	DraftTTL time.Duration
}

// Service implements owner-private saved trading partners. It has no
// capability to post, accept or change money, balances, limits, reputation,
// notifications or any other member's data: it only returns Draft values.
type Service struct {
	store                   Store
	policy                  Policy
	enabled                 func(context.Context, Actor) (bool, error)
	trades                  TradeSource
	now                     func() time.Time
	maxFavorites, maxDrafts int
	draftTTL                time.Duration
}

// NewService validates cfg and returns a Service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Store == nil || cfg.Policy == nil || cfg.Enabled == nil {
		return nil, errors.New("favorites: Store, Policy and Enabled are required")
	}
	s := &Service{store: cfg.Store, policy: cfg.Policy, enabled: cfg.Enabled, trades: cfg.Trades, now: cfg.Clock,
		maxFavorites: cfg.MaxFavorites, maxDrafts: cfg.MaxDrafts, draftTTL: cfg.DraftTTL}
	if s.now == nil {
		s.now = time.Now
	}
	if s.maxFavorites <= 0 {
		s.maxFavorites = DefaultMaxFavorites
	}
	if s.maxFavorites > HardMaxFavorites {
		return nil, fmt.Errorf("favorites: MaxFavorites exceeds %d", HardMaxFavorites)
	}
	if s.maxDrafts <= 0 {
		s.maxDrafts = DefaultMaxDrafts
	}
	if s.draftTTL <= 0 {
		s.draftTTL = DefaultDraftTTL
	}
	return s, nil
}

func (s *Service) utcNow() time.Time { return s.now().UTC().Truncate(time.Microsecond) }

// View is the owner-facing projection of a favorite.
type View struct {
	Target      string
	DisplayName string
	Nickname    string
	Pinned      bool
	Order       int
	CreatedAt   time.Time
}

// Page is one page of visible favorites. Revision must be echoed to Reorder.
type Page struct {
	Items    []View
	Next     string
	Revision int64
}

// ListOptions bounds a List call. Cursor is opaque.
type ListOptions struct {
	Limit  int
	Cursor string
}

// AddRequest creates a favorite for the authenticated owner.
type AddRequest struct {
	Target   string
	Nickname string
	Pinned   bool
}

// ReorderRequest places Targets first, in the given order. Targets must be
// unique stored favorites of the owner; unlisted favorites follow in their
// existing order. Revision must equal the revision returned by List.
type ReorderRequest struct {
	Revision int64
	Targets  []string
}

// Enabled reports whether the feature is switched on for a. A failing gate
// is a policy failure (fail closed).
func (s *Service) Enabled(ctx context.Context, a Actor) (bool, error) {
	if !validID(a.Network) || !validID(a.Member) {
		return false, nil
	}
	ok, err := s.enabled(ctx, a)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrPolicy, err)
	}
	return ok, nil
}

// checkActor validates the identity and applies the opt-in gate.
func (s *Service) checkActor(ctx context.Context, a Actor) error {
	if !validID(a.Network) || !validID(a.Member) {
		return ErrUnavailable
	}
	ok, err := s.Enabled(ctx, a)
	if err != nil {
		return err
	}
	if !ok {
		return ErrDisabled
	}
	return nil
}

// lookup resolves ids through the Policy and verifies each snapshot really
// belongs to the requested identity and network. Missing ids are absent.
func (s *Service) lookup(ctx context.Context, a Actor, ids []string) (map[string]Member, error) {
	if len(ids) == 0 {
		return map[string]Member{}, nil
	}
	m, err := s.policy.Members(ctx, a, ids)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPolicy, err)
	}
	out := make(map[string]Member, len(m))
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	for k, v := range m {
		if !want[k] {
			continue
		}
		if v.ID != k || v.Network != a.Network {
			// A snapshot for another identity or network is an adapter fault; refuse.
			return nil, fmt.Errorf("%w: inconsistent snapshot", ErrPolicy)
		}
		out[k] = v
	}
	return out, nil
}

// owner returns the fresh snapshot of the acting member or ErrUnavailable.
func (s *Service) owner(ctx context.Context, a Actor, extra ...string) (Member, map[string]Member, error) {
	if err := s.checkActor(ctx, a); err != nil {
		return Member{}, nil, err
	}
	ids := append([]string{a.Member}, extra...)
	m, err := s.lookup(ctx, a, ids)
	if err != nil {
		return Member{}, nil, err
	}
	o, ok := m[a.Member]
	if !ok || !o.Active || o.Blocked {
		return Member{}, nil, ErrUnavailable
	}
	return o, m, nil
}

// Add saves target as a favorite of the authenticated owner. Adding an
// identical existing favorite is an idempotent success (created=false); one
// with different data is ErrExists. Self, cross-network, inactive, hidden and
// blocked targets are refused with ErrUnavailable.
func (s *Service) Add(ctx context.Context, a Actor, req AddRequest) (View, bool, error) {
	nick, err := CleanNickname(req.Nickname)
	if err != nil {
		return View{}, false, err
	}
	if !validID(req.Target) {
		return View{}, false, ErrUnavailable
	}
	owner, m, err := s.owner(ctx, a, req.Target)
	if err != nil {
		return View{}, false, err
	}
	target, ok := m[req.Target]
	if !ok {
		return View{}, false, ErrUnavailable
	}
	fav, err := Save(owner, target, nick)
	if err != nil {
		return View{}, false, err
	}
	if err := s.pruneIfFull(ctx, a); err != nil {
		return View{}, false, err
	}
	var out View
	created := false
	err = s.store.Update(ctx, func(tx Tx) error {
		if ex, ok, err := tx.GetFavorite(ctx, a.Network, a.Member, fav.Target); err != nil {
			return err
		} else if ok {
			if ex.Nickname != fav.Nickname || ex.Pinned != req.Pinned {
				return ErrExists
			}
			out = toView(ex, target)
			return nil
		}
		entries, err := tx.ListFavorites(ctx, a.Network, a.Member, s.maxFavorites+1)
		if err != nil {
			return err
		}
		if len(entries) >= s.maxFavorites {
			return ErrLimit
		}
		next := 1
		for _, e := range entries {
			if e.Order >= next {
				next = e.Order + 1
			}
		}
		now := s.utcNow()
		e := Entry{Favorite: fav, Order: next, Pinned: req.Pinned, CreatedAt: now, UpdatedAt: now}
		if err := tx.InsertFavorite(ctx, e); err != nil {
			return err
		}
		if _, err := tx.BumpRevision(ctx, a.Network, a.Member); err != nil {
			return err
		}
		out, created = toView(e, target), true
		return nil
	})
	if err != nil {
		return View{}, false, err
	}
	return out, created, nil
}

// pruneIfFull deletes stored favorites whose target no longer exists in the
// network at all, so obsolete rows the owner cannot see never block adding.
func (s *Service) pruneIfFull(ctx context.Context, a Actor) error {
	var entries []Entry
	if err := s.store.View(ctx, func(tx Tx) (err error) {
		entries, err = tx.ListFavorites(ctx, a.Network, a.Member, s.maxFavorites+1)
		return
	}); err != nil {
		return err
	}
	if len(entries) < s.maxFavorites {
		return nil
	}
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.Target
	}
	m, err := s.lookup(ctx, a, ids)
	if err != nil {
		return err
	}
	return s.store.Update(ctx, func(tx Tx) error {
		changed := false
		for _, e := range entries {
			if _, ok := m[e.Target]; ok {
				continue
			}
			if ok, err := tx.DeleteFavorite(ctx, a.Network, a.Member, e.Target); err != nil {
				return err
			} else if ok {
				changed = true
			}
		}
		if changed {
			_, err := tx.BumpRevision(ctx, a.Network, a.Member)
			return err
		}
		return nil
	})
}

func toView(e Entry, t Member) View {
	return View{Target: e.Target, DisplayName: t.DisplayName, Nickname: e.Nickname, Pinned: e.Pinned, Order: e.Order, CreatedAt: e.CreatedAt}
}

type cursor struct {
	P bool   `json:"p"`
	O int    `json:"o"`
	T string `json:"t"`
}

func encodeCursor(v View) string {
	b, _ := json.Marshal(cursor{v.Pinned, v.Order, v.Target})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(c string) (cursor, error) {
	var cur cursor
	if len(c) > 512 {
		return cur, invalid("cursor", "malformed")
	}
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil || json.Unmarshal(b, &cur) != nil || !validID(cur.T) {
		return cur, invalid("cursor", "malformed")
	}
	return cur, nil
}

// after reports whether v sorts strictly after the cursor position.
func (c cursor) before(v View) bool {
	if v.Pinned != c.P {
		return !v.Pinned && c.P
	}
	if v.Order != c.O {
		return v.Order > c.O
	}
	return v.Target > c.T
}

// List returns the owner's currently usable favorites, pinned first. Targets
// that are departed, hidden, blocked, suspended or in another network are
// omitted immediately because every call re-reads the Policy. A policy failure
// fails the whole call.
func (s *Service) List(ctx context.Context, a Actor, opt ListOptions) (Page, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	var cur *cursor
	if opt.Cursor != "" {
		c, err := decodeCursor(opt.Cursor)
		if err != nil {
			return Page{}, err
		}
		cur = &c
	}
	if err := s.checkActor(ctx, a); err != nil {
		return Page{}, err
	}
	var entries []Entry
	var rev int64
	if err := s.store.View(ctx, func(tx Tx) (err error) {
		if rev, err = tx.Revision(ctx, a.Network, a.Member); err != nil {
			return err
		}
		entries, err = tx.ListFavorites(ctx, a.Network, a.Member, s.maxFavorites+1)
		return err
	}); err != nil {
		return Page{}, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.Target)
	}
	owner, m, err := s.owner(ctx, a, ids...)
	if err != nil {
		return Page{}, err
	}
	vis := VisibleFavorites(owner, entries, m)
	items := make([]View, 0, limit)
	page := Page{Revision: rev}
	for _, e := range vis {
		v := toView(e, m[e.Target])
		if cur != nil && !cur.before(v) {
			continue
		}
		if len(items) == limit {
			page.Next = encodeCursor(items[len(items)-1])
			break
		}
		items = append(items, v)
	}
	page.Items = items
	return page, nil
}

// SetNickname changes the private nickname of an existing favorite. It
// succeeds idempotently and also works for a favorite whose target is no
// longer available.
func (s *Service) SetNickname(ctx context.Context, a Actor, target, nickname string) error {
	nick, err := CleanNickname(nickname)
	if err != nil {
		return err
	}
	return s.mutate(ctx, a, target, func(e *Entry) bool {
		if e.Nickname == nick {
			return false
		}
		e.Nickname = nick
		return true
	})
}

// SetPinned pins or unpins a favorite. Idempotent.
func (s *Service) SetPinned(ctx context.Context, a Actor, target string, pinned bool) error {
	return s.mutate(ctx, a, target, func(e *Entry) bool {
		if e.Pinned == pinned {
			return false
		}
		e.Pinned = pinned
		return true
	})
}

func (s *Service) mutate(ctx context.Context, a Actor, target string, f func(*Entry) bool) error {
	if !validID(target) {
		return ErrNotFound
	}
	if _, _, err := s.owner(ctx, a); err != nil {
		return err
	}
	return s.store.Update(ctx, func(tx Tx) error {
		e, ok, err := tx.GetFavorite(ctx, a.Network, a.Member, target)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if !f(&e) {
			return nil
		}
		e.UpdatedAt = s.utcNow()
		if ok, err := tx.UpdateFavorite(ctx, e); err != nil {
			return err
		} else if !ok {
			return ErrNotFound
		}
		_, err = tx.BumpRevision(ctx, a.Network, a.Member)
		return err
	})
}

// Remove deletes the owner's favorite for target. It works even when the
// target is blocked, hidden or departed. It reports whether a row was removed.
func (s *Service) Remove(ctx context.Context, a Actor, target string) (bool, error) {
	if !validID(target) {
		return false, nil
	}
	if _, _, err := s.owner(ctx, a); err != nil {
		return false, err
	}
	removed := false
	err := s.store.Update(ctx, func(tx Tx) error {
		ok, err := tx.DeleteFavorite(ctx, a.Network, a.Member, target)
		if err != nil || !ok {
			return err
		}
		removed = true
		_, err = tx.BumpRevision(ctx, a.Network, a.Member)
		return err
	})
	return removed, err
}

// Reorder atomically places req.Targets first in the given order. A stale
// revision, a duplicate, an unknown or foreign target, or an oversized list
// changes nothing and returns ErrStale or ErrInvalidOrder.
func (s *Service) Reorder(ctx context.Context, a Actor, req ReorderRequest) (int64, error) {
	if len(req.Targets) > s.maxFavorites {
		return 0, ErrInvalidOrder
	}
	if _, _, err := s.owner(ctx, a); err != nil {
		return 0, err
	}
	var rev int64
	err := s.store.Update(ctx, func(tx Tx) error {
		cur, err := tx.Revision(ctx, a.Network, a.Member)
		if err != nil {
			return err
		}
		if cur != req.Revision {
			return ErrStale
		}
		entries, err := tx.ListFavorites(ctx, a.Network, a.Member, s.maxFavorites+1)
		if err != nil {
			return err
		}
		byTarget := make(map[string]Entry, len(entries))
		for _, e := range entries {
			byTarget[e.Target] = e
		}
		seen := make(map[string]bool, len(req.Targets))
		ordered := make([]string, 0, len(entries))
		for _, t := range req.Targets {
			if _, ok := byTarget[t]; !ok || seen[t] {
				return ErrInvalidOrder
			}
			seen[t] = true
			ordered = append(ordered, t)
		}
		rest := make([]Entry, 0, len(entries))
		for _, e := range entries {
			if !seen[e.Target] {
				rest = append(rest, e)
			}
		}
		sort.SliceStable(rest, func(i, j int) bool {
			if rest[i].Order != rest[j].Order {
				return rest[i].Order < rest[j].Order
			}
			return rest[i].Target < rest[j].Target
		})
		for _, e := range rest {
			ordered = append(ordered, e.Target)
		}
		now := s.utcNow()
		for i, t := range ordered {
			e := byTarget[t]
			if e.Order == i+1 {
				continue
			}
			e.Order, e.UpdatedAt = i+1, now
			if ok, err := tx.UpdateFavorite(ctx, e); err != nil {
				return err
			} else if !ok {
				return ErrNotFound
			}
		}
		rev, err = tx.BumpRevision(ctx, a.Network, a.Member)
		return err
	})
	if err != nil {
		return 0, err
	}
	return rev, nil
}

// MemberDeparted removes every favorite and draft owned by or pointing at
// member in network. It is a system command for the host's departure
// transaction, not an actor operation. Access is already denied immediately by
// List because the Policy is read on every call; this only deletes rows.
func (s *Service) MemberDeparted(ctx context.Context, network, member string) (PurgeResult, error) {
	if !validID(network) || !validID(member) {
		return PurgeResult{}, ErrUnavailable
	}
	var res PurgeResult
	err := s.store.Update(ctx, func(tx Tx) (err error) {
		res, err = tx.PurgeMember(ctx, network, member)
		return err
	})
	return res, err
}

// PurgeExpiredDrafts deletes drafts past their TTL (housekeeping).
func (s *Service) PurgeExpiredDrafts(ctx context.Context) (int64, error) {
	var n int64
	err := s.store.Update(ctx, func(tx Tx) (err error) {
		n, err = tx.PurgeExpiredIntents(ctx, s.utcNow())
		return err
	})
	return n, err
}
