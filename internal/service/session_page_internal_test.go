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
	if before != nil && before.Reverse {
		end := sort.Search(len(p.rows), func(i int) bool {
			row := p.rows[i]
			return row.CreatedAt.Before(before.CreatedAt) || row.CreatedAt.Equal(before.CreatedAt) && row.PublicID <= before.PublicID
		})
		start := max(0, end-limit)
		out := make([]*domain.Session, 0, end-start)
		for i := end - 1; i >= start; i-- {
			out = append(out, p.rows[i])
		}
		p.fetched += len(out)
		return out, nil
	}
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

func TestOwnSessionPreviousRoundTrip(t *testing.T) {
	s, actor, p := sessionPageFixture(60, false)
	first, err := s.ListOwnSessions(context.Background(), actor, Page{Limit: 7})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: first.Next, Limit: 7})
	if err != nil || second.Previous == "" {
		t.Fatalf("second page missing previous: %+v %v", second, err)
	}
	p.calls, p.fetched = 0, 0
	previous, err := s.ListOwnSessions(context.Background(), actor, Page{BeforeID: second.Previous, Limit: 7})
	if err != nil || len(previous.Rows) != len(first.Rows) || previous.Previous != "" || p.calls > 20 || p.fetched > 2000 {
		t.Fatalf("previous page: %+v %v calls=%d rows=%d", previous, err, p.calls, p.fetched)
	}
	for i := range first.Rows {
		if first.Rows[i].ID != previous.Rows[i].ID {
			t.Fatal("previous page skipped or repeated an equal-time row")
		}
	}
	back, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: previous.Next, Limit: 7})
	if err != nil || len(back.Rows) != len(second.Rows) {
		t.Fatalf("next after previous: %+v %v", back, err)
	}
	for i := range second.Rows {
		if second.Rows[i].ID != back.Rows[i].ID {
			t.Fatal("next after previous changed the second page")
		}
	}
}

func TestOwnSessionPreviousCrossesEmptyBoundedWindow(t *testing.T) {
	s, actor, p := sessionPageFixture(4000, true)
	// Keep the newest row live, with the current session below 3,999 dead rows.
	p.rows[0].ExpiresAt = p.current.ExpiresAt
	anchor := sessionPosition(p.current.CreatedAt, p.current.PublicID)
	page, err := s.ListOwnSessions(context.Background(), actor, Page{BeforeID: anchor})
	if err != nil || len(page.Rows) != 0 || page.Previous == "" || page.Next == "" || p.calls != 20 || p.fetched != 2000 {
		t.Fatalf("reverse work-cap page: %+v %v calls=%d rows=%d", page, err, p.calls, p.fetched)
	}
	// The opposite cursor must include the original live boundary again.
	back, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: page.Next})
	if err != nil || len(back.Rows) != 1 || !back.Rows[0].Current {
		t.Fatalf("empty reverse page skipped its original boundary: %+v %v", back, err)
	}
	p.calls, p.fetched = 0, 0
	newer, err := s.ListOwnSessions(context.Background(), actor, Page{BeforeID: page.Previous})
	if err != nil || len(newer.Rows) != 1 || newer.Rows[0].ID != p.rows[0].PublicID || p.calls > 20 || p.fetched > 2000 {
		t.Fatalf("reverse continuation: %+v %v calls=%d rows=%d", newer, err, p.calls, p.fetched)
	}
	if _, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: anchor, BeforeID: anchor}); domain.CodeOf(err) != domain.CodeValidation {
		t.Fatalf("ambiguous page accepted: %v", err)
	}
}

func TestOwnSessionEmptyForwardPageCanReturnToLiveBoundary(t *testing.T) {
	s, actor, p := sessionPageFixture(4001, true)
	p.rows[0].ExpiresAt = p.current.ExpiresAt
	anchor := sessionPosition(p.rows[0].CreatedAt, p.rows[0].PublicID)
	page, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: anchor})
	if err != nil || len(page.Rows) != 0 || page.Previous == "" || page.Next == "" || p.calls != 20 || p.fetched != 2000 {
		t.Fatalf("forward work-cap page: %+v %v calls=%d rows=%d", page, err, p.calls, p.fetched)
	}
	back, err := s.ListOwnSessions(context.Background(), actor, Page{BeforeID: page.Previous})
	if err != nil || len(back.Rows) != 1 || back.Rows[0].ID != p.rows[0].PublicID {
		t.Fatalf("empty forward page skipped its original boundary: %+v %v", back, err)
	}
}

func TestOwnSessionDisappearedNewerSideRefreshesAtLiveBoundary(t *testing.T) {
	s, actor, p := sessionPageFixture(60, false)
	boundary := p.rows[7]
	anchor := sessionPosition(boundary.CreatedAt, boundary.PublicID)
	// A previous-page link was issued before every newer retained row was
	// removed. Its initiating boundary still exists and remains active.
	p.rows = p.rows[7:]
	page, err := s.ListOwnSessions(context.Background(), actor, Page{BeforeID: anchor, Limit: 7})
	if err != nil || len(page.Rows) != 7 || page.Rows[0].ID != boundary.PublicID || page.Previous != "" {
		t.Fatalf("disappeared newer side did not refresh at the live boundary: %+v %v", page, err)
	}
	if p.calls != 2 || p.fetched > 2000 {
		t.Fatalf("refresh work: calls=%d fetched=%d", p.calls, p.fetched)
	}
}

func TestOwnSessionDisappearedOlderSideTerminatesWithoutWrapping(t *testing.T) {
	s, actor, p := sessionPageFixture(60, false)
	// Keep authentication on a surviving newer row while older history is
	// removed. Forward API traversal must end rather than wrap to earlier rows.
	p.current = p.rows[0]
	actor.SessionID = p.current.IDHash
	boundary := p.rows[39]
	anchor := sessionPosition(boundary.CreatedAt, boundary.PublicID)
	p.rows = p.rows[:40]
	page, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: anchor, Limit: 7})
	if err != nil || len(page.Rows) != 0 || page.Next != "" || page.Previous != anchor {
		t.Fatalf("disappeared older side wrapped instead of ending: %+v %v", page, err)
	}
	if p.calls != 1 || p.fetched != 0 {
		t.Fatalf("empty terminal page work: calls=%d fetched=%d", p.calls, p.fetched)
	}
	// The live-view grouping may shift, but Previous followed by Next still
	// reaches the surviving initiating boundary.
	previous, err := s.ListOwnSessions(context.Background(), actor, Page{BeforeID: page.Previous, Limit: 7})
	if err != nil || len(previous.Rows) != 7 || previous.Next == "" {
		t.Fatalf("previous from terminal page: %+v %v", previous, err)
	}
	back, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: previous.Next, Limit: 7})
	if err != nil || len(back.Rows) != 1 || back.Rows[0].ID != boundary.PublicID || back.Next != "" {
		t.Fatalf("previous/next skipped surviving boundary: %+v %v", back, err)
	}
}

func TestOwnSessionEmptyReverseSideRefreshSharesScanBudget(t *testing.T) {
	s, actor, p := sessionPageFixture(3000, true)
	pg := Page{BeforeID: sessionPosition(p.rows[0].CreatedAt, p.rows[0].PublicID)}
	page, err := s.ListOwnSessions(context.Background(), actor, pg)
	if err != nil || len(page.Rows) != 0 || page.Previous != "" || page.Next == "" || p.calls != 20 || p.fetched != 1900 {
		t.Fatalf("refresh exceeded or failed to use the shared work budget: %+v %v calls=%d fetched=%d", page, err, p.calls, p.fetched)
	}
	want := p.rows[1899]
	if page.Next != sessionPosition(want.CreatedAt, want.PublicID) {
		t.Fatal("refreshed continuation did not follow the last inspected row")
	}
}

func TestOwnSessionExactWorkCapExhaustionTerminates(t *testing.T) {
	s, actor, p := sessionPageFixture(2000, true)
	expires := p.current.ExpiresAt
	p.current = p.rows[0]
	p.current.ExpiresAt = expires
	actor.SessionID = p.current.IDHash
	p.rows = p.rows[:2000]
	page, err := s.ListOwnSessions(context.Background(), actor, Page{})
	if err != nil || len(page.Rows) != 1 || page.Next == "" || p.calls != 20 || p.fetched != 2000 {
		t.Fatalf("exact-cap first page: %+v %v calls=%d fetched=%d", page, err, p.calls, p.fetched)
	}
	p.calls, p.fetched = 0, 0
	last, err := s.ListOwnSessions(context.Background(), actor, Page{AfterID: page.Next})
	if err != nil || len(last.Rows) != 0 || last.Next != "" || p.calls != 1 || p.fetched != 0 {
		t.Fatalf("exact-cap continuation wrapped instead of terminating: %+v %v calls=%d fetched=%d", last, err, p.calls, p.fetched)
	}
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
