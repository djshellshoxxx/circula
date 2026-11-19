package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

type sessionPageClock struct{ at time.Time }

func (c sessionPageClock) Now() time.Time { return c.at }

// This driver-neutral probe records the rows the service asks to inspect.
// Storage ordering, owner isolation and seek plans have separate SQLite and
// repository contract regressions; no driver is needed to test the work cap.
type sessionPageProbe struct {
	store.Store
	store.Tx
	rows                    []*domain.Session
	current                 *domain.Session
	calls, fetched, lookups int
	firstBefore             *store.SessionPosition
}

func (p *sessionPageProbe) View(_ context.Context, fn func(store.Tx) error) error {
	return fn(p)
}

func (p *sessionPageProbe) GetSession(id string) (*domain.Session, error) {
	if id == p.current.IDHash {
		return p.current, nil
	}
	return nil, domain.NotFound("session")
}

func (*sessionPageProbe) MustChangePassword(string) (bool, error) { return false, nil }

func (p *sessionPageProbe) GetOwnedSession(owner, id string) (*domain.Session, error) {
	p.lookups++
	for _, row := range p.rows {
		if row.PersonID == owner && row.PublicID == id {
			return row, nil
		}
	}
	return nil, domain.NotFound("session")
}

func (p *sessionPageProbe) ListSessions(owner string, before *store.SessionPosition, limit int) ([]*domain.Session, error) {
	if owner != p.current.PersonID || limit != 100 {
		return nil, fmt.Errorf("unscoped or unbounded session query: owner=%s limit=%d", owner, limit)
	}
	if p.calls == 0 {
		p.firstBefore = before
	}
	p.calls++
	start := 0
	if before != nil {
		start = sort.Search(len(p.rows), func(i int) bool {
			row := p.rows[i]
			return row.CreatedAt.Before(before.CreatedAt) || row.CreatedAt.Equal(before.CreatedAt) && row.PublicID < before.PublicID
		})
	}
	end := min(start+limit, len(p.rows))
	p.fetched += end - start
	return p.rows[start:end], nil
}

func sessionPageFixture(n int, dead bool) (*Service, domain.Actor, *sessionPageProbe) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	current := &domain.Session{IDHash: "private-current-hash", PublicID: strings.Repeat("f", 64), PersonID: "owner",
		CreatedAt: now.Add(-time.Hour), LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	p := &sessionPageProbe{current: current}
	for i := 0; i < n; i++ {
		expires := now.Add(time.Hour)
		if dead {
			expires = now // exact stored expiry boundary
		}
		p.rows = append(p.rows, &domain.Session{IDHash: fmt.Sprintf("private-%d", i), PublicID: fmt.Sprintf("%064x", n-i),
			PersonID: "owner", CreatedAt: now.Add(-time.Minute), LastSeenAt: now, ExpiresAt: expires})
	}
	p.rows = append(p.rows, current)
	s := &Service{st: p, cfg: Config{Clock: sessionPageClock{now}, SessionIdle: 24 * time.Hour, SessionMax: 24 * time.Hour,
		AdminSessionIdle: 24 * time.Hour, AdminSessionMax: 24 * time.Hour,
		ReviewerSessionIdle: 24 * time.Hour, ReviewerSessionMax: 24 * time.Hour}}
	a := domain.Actor{PersonID: "owner", SessionID: current.IDHash, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	return s, a, p
}

func TestOwnSessionWorkCapReturnsUsableEmptyPage(t *testing.T) {
	s, actor, p := sessionPageFixture(2000, true)
	page, err := s.ListOwnSessions(context.Background(), actor, Page{})
	if err != nil || len(page.Rows) != 0 || page.Next == "" || p.calls != 20 || p.fetched != 2000 {
		t.Fatalf("empty work-cap page: %+v, err=%v calls=%d inspected=%d", page, err, p.calls, p.fetched)
	}
	want := p.rows[1999]
	if page.Next != sessionPosition(want.CreatedAt, want.PublicID) {
		t.Fatal("continuation did not advance past the last inspected dead row")
	}
	p.calls, p.fetched = 0, 0
	next, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: page.Next})
	if err != nil || len(next.Rows) != 1 || !next.Rows[0].Current || next.Next != "" || p.calls != 1 || p.fetched != 1 {
		t.Fatalf("empty-page continuation: %+v err=%v calls=%d inspected=%d", next, err, p.calls, p.fetched)
	}
	if p.firstBefore == nil || p.firstBefore.PublicID != want.PublicID {
		t.Fatal("continuation restarted the owner scan")
	}
	p.calls = 0
	row, err := s.GetOwnSession(context.Background(), actor, p.current.PublicID)
	if err != nil || !row.Current || p.lookups != 1 || p.calls != 0 {
		t.Fatalf("single target lookup scanned the inventory: %+v %v calls=%d lookups=%d", row, err, p.calls, p.lookups)
	}
	if _, err := s.GetOwnSession(context.Background(), actor, want.PublicID); domain.CodeOf(err) != domain.CodeNotFound {
		t.Fatalf("expired confirmation target: %v", err)
	}
}

func TestOwnSessionLookaheadKeepsEqualTimeRows(t *testing.T) {
	s, actor, p := sessionPageFixture(205, false)
	var after string
	count := 0
	for round := 0; round < 100; round++ {
		page, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: after, Limit: 7})
		if err != nil || len(page.Rows) > 7 {
			t.Fatalf("bounded page: %+v %v", page, err)
		}
		for _, row := range page.Rows {
			if count >= len(p.rows) || row.ID != p.rows[count].PublicID {
				t.Fatalf("equal-time lookahead skipped or repeated row %d: %+v", count, row)
			}
			if strings.Contains(page.Next, p.rows[count].IDHash) {
				t.Fatal("page continuation exposed a private credential hash")
			}
			count++
		}
		if page.Next == "" {
			break
		}
		if page.Next == after {
			t.Fatal("page did not advance")
		}
		after = page.Next
	}
	if count != len(p.rows) {
		t.Fatalf("traversed %d rows, want %d", count, len(p.rows))
	}
}

func TestOwnSessionPageRejectsMalformedBoundsBeforeListing(t *testing.T) {
	s, actor, p := sessionPageFixture(1, false)
	for _, pg := range []Page{
		{Limit: -1}, {Limit: 101}, {AfterID: "nonsense"},
		{AfterID: "1:" + strings.Repeat("G", 64)}, {AfterID: "01:" + strings.Repeat("a", 64)},
		{AfterID: "9223372036854775808:" + strings.Repeat("a", 64)},
		{AfterID: "1:" + strings.Repeat("a", 64) + ":extra"},
	} {
		if _, err := s.ListOwnSessions(context.Background(), actor, pg); domain.CodeOf(err) != domain.CodeValidation {
			t.Errorf("%+v: %v", pg, err)
		}
	}
	if p.calls != 0 {
		t.Fatalf("invalid boundaries reached listing: %d calls", p.calls)
	}
}
