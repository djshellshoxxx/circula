package service_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Scale for the paging tests: well past the old 500 and 1000 caps and the 2000-member scan window.
const (
	pagingActive  = 640 // active members, admin included
	pagingPending = 520 // pending join requests, past the old 500 cap
)

type pagingWorld struct {
	*tu.Env
	admin  domain.Actor
	net    *domain.Network
	active []string // member ids of every active member, ascending
	names  map[string]string
}

func newPagingWorld(t *testing.T) *pagingWorld {
	t.Helper()
	e := tu.NewEnv(t)
	w := &pagingWorld{Env: e, names: map[string]string{}}
	w.admin = e.Person("admin@example.org")
	var adminM *domain.Member
	w.net, adminM = e.Network(w.admin, "town", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	w.add(adminM.ID, adminM.DisplayName)
	for i := 1; i < pagingActive; i++ {
		login := fmt.Sprintf("m%04d@example.org", i)
		_, m := e.Member(w.admin, w.net, login)
		w.add(m.ID, m.DisplayName)
	}
	sort.Strings(w.active)
	for i := 0; i < pagingPending; i++ {
		login := fmt.Sprintf("p%04d@example.org", i)
		a := e.Person(login)
		tu.Must(e.Svc.JoinNetwork(e.Ctx, a, w.net.ID, login))
	}
	return w
}

func (w *pagingWorld) add(id, name string) {
	w.active = append(w.active, id)
	w.names[id] = name
}

// walkDirectory follows Next links from the first page and returns every page in order.
func (w *pagingWorld) walkDirectory(t *testing.T, name string, limit int) [][]service.DirectoryEntry {
	t.Helper()
	var pages [][]service.DirectoryEntry
	q := service.MemberPageQuery{Name: name, Limit: limit}
	for i := 0; ; i++ {
		if i > pagingActive {
			t.Fatal("paging does not terminate")
		}
		pg := tu.Must(w.Svc.ListDirectoryPage(w.Ctx, w.admin, w.net.ID, q))
		if (i == 0) == pg.HasPrev {
			t.Errorf("page %d: HasPrev = %v", i, pg.HasPrev)
		}
		if len(pg.Entries) > limit {
			t.Fatalf("page %d has %d rows, limit %d", i, len(pg.Entries), limit)
		}
		pages = append(pages, pg.Entries)
		if !pg.HasNext {
			return pages
		}
		q = service.MemberPageQuery{Name: name, Limit: limit, AfterID: pg.Entries[len(pg.Entries)-1].MemberID}
	}
}

func ids(es []service.DirectoryEntry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.MemberID
	}
	return out
}

// TestMemberPaging shares one fixture (about 1,160 members) across the paging checks.
func TestMemberPaging(t *testing.T) {
	w := newPagingWorld(t)
	t.Run("directory reaches every member", func(t *testing.T) { directoryReachesEveryMember(t, w) })
	t.Run("name filter covers all matches", func(t *testing.T) { nameFilterCoversAllMatches(t, w) })
	t.Run("pending count is not capped", func(t *testing.T) { pendingCountIsNotCapped(t, w) })
	t.Run("rejects both cursors and non-members", func(t *testing.T) { rejectsBothCursorsAndNonAdmins(t, w) })
}

// Nothing is hidden: forward paging visits every active member exactly once, the count agrees,
// and paging backward from each page returns the page before it.
func directoryReachesEveryMember(t *testing.T, w *pagingWorld) {
	pages := w.walkDirectory(t, "", 100)
	var all []string
	for _, p := range pages {
		all = append(all, ids(p)...)
	}
	if len(all) != pagingActive {
		t.Fatalf("paged %d members, want %d", len(all), pagingActive)
	}
	if !sort.StringsAreSorted(all) {
		t.Error("pages are not in id order")
	}
	seen := map[string]bool{}
	for _, id := range all {
		if seen[id] {
			t.Fatalf("member %s on two pages", id)
		}
		seen[id] = true
	}
	n := tu.Must(w.Svc.CountDirectory(w.Ctx, w.admin, w.net.ID, ""))
	if n != pagingActive {
		t.Errorf("CountDirectory = %d, want %d", n, pagingActive)
	}
	// Backward: the page before page i is exactly page i-1, and HasNext is set.
	for i := 1; i < len(pages); i++ {
		first := pages[i][0].MemberID
		back := tu.Must(w.Svc.ListDirectoryPage(w.Ctx, w.admin, w.net.ID, service.MemberPageQuery{BeforeID: first, Limit: 100}))
		if strings.Join(ids(back.Entries), ",") != strings.Join(ids(pages[i-1]), ",") {
			t.Fatalf("backward from page %d did not return page %d", i, i-1)
		}
		if !back.HasNext {
			t.Errorf("backward page %d: HasNext = false", i-1)
		}
		if back.HasPrev != (i-1 > 0) {
			t.Errorf("backward page %d: HasPrev = %v", i-1, back.HasPrev)
		}
	}
}

// Name search is a server-side filter: count and pages agree and no matching member is lost.
func nameFilterCoversAllMatches(t *testing.T, w *pagingWorld) {
	want := 0
	for _, id := range w.active {
		if strings.Contains(w.names[id], "m06") {
			want++
		}
	}
	if want < 30 {
		t.Fatalf("fixture has %d matches, want a multi-page set", want)
	}
	for _, q := range []string{"m06", "M06"} { // case-insensitive
		if n := tu.Must(w.Svc.CountDirectory(w.Ctx, w.admin, w.net.ID, q)); n != want {
			t.Errorf("CountDirectory(%q) = %d, want %d", q, n, want)
		}
		got := 0
		for _, p := range w.walkDirectory(t, q, 25) {
			got += len(p)
		}
		if got != want {
			t.Errorf("paged %q: %d rows, want %d", q, got, want)
		}
	}
}

func pendingCountIsNotCapped(t *testing.T, w *pagingWorld) {
	n := tu.Must(w.Svc.CountMembers(w.Ctx, w.admin, w.net.ID, []domain.MemberStatus{domain.MemberPending}))
	if n != pagingPending {
		t.Fatalf("pending count = %d, want %d", n, pagingPending)
	}
	total := 0
	after := ""
	for {
		pg := tu.Must(w.Svc.AdminMemberPage(w.Ctx, w.admin, w.net.ID, []domain.MemberStatus{domain.MemberPending},
			service.MemberPageQuery{AfterID: after, Limit: 100}))
		total += len(pg.Members)
		for _, m := range pg.Members {
			if m.Status != domain.MemberPending {
				t.Fatalf("member %s status %s in pending list", m.ID, m.Status)
			}
		}
		if !pg.HasNext {
			break
		}
		after = pg.Members[len(pg.Members)-1].ID
	}
	if total != pagingPending {
		t.Errorf("admin paging saw %d pending, want %d", total, pagingPending)
	}
	all := tu.Must(w.Svc.CountMembers(w.Ctx, w.admin, w.net.ID, nil))
	if all != pagingActive+pagingPending {
		t.Errorf("total count = %d, want %d", all, pagingActive+pagingPending)
	}
}

func rejectsBothCursorsAndNonAdmins(t *testing.T, w *pagingWorld) {
	_, err := w.Svc.ListDirectoryPage(w.Ctx, w.admin, w.net.ID, service.MemberPageQuery{AfterID: "a", BeforeID: "b"})
	wantCode(t, err, domain.ErrValidation)
	plain := w.Person("nobody@example.org")
	_, err = w.Svc.CountMembers(w.Ctx, plain, w.net.ID, nil)
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.AdminMemberPage(w.Ctx, plain, w.net.ID, nil, service.MemberPageQuery{})
	wantCode(t, err, domain.ErrNotFound)
}
