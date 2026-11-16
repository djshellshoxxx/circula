package web

import (
	"fmt"
	"html"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// Fixture size: past the old 500-row admin cap, the 500/1000-member directory caps and the
// 500-row pending count.
const (
	townMembers = 620 // added on top of admin, alice and bob
	townPending = 520
)

var (
	dirEntryRe  = regexp.MustCompile(`<a href="/n/town/members/([^"]+)">([^<]+)</a>`)
	adminRowRe  = regexp.MustCompile(`<th scope="row" data-label="Name">[^<]*`)
	nextLinkRe  = regexp.MustCompile(`<a href="([^"]+)">Next page</a>`)
	prevLinkRe  = regexp.MustCompile(`<a href="([^"]+)">Previous page</a>`)
	foundLineRe = regexp.MustCompile(`<p>(\d+) members? found\.</p>`)
)

// newTownWorld builds a network with approval-mode joins, townMembers active members and
// townPending pending requests. alice is an ordinary active member; admin founded the network.
// It also returns the last member added, who sits past the old caps.
func newTownWorld(t *testing.T) (*world, domain.Actor, *domain.Member) {
	t.Helper()
	env := testutil.NewEnv(t)
	w := &world{env: env}
	w.admin = env.Person("admin@example.org")
	w.net, w.adminM = env.Network(w.admin, "town", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	w.alice, w.aliceM = env.Member(w.admin, w.net, "alice@example.org")
	w.bob, w.bobM = env.Member(w.admin, w.net, "bob@example.org")
	var lastA domain.Actor
	var lastM *domain.Member
	for i := 0; i < townMembers; i++ {
		lastA, lastM = env.Member(w.admin, w.net, townLogin(i))
	}
	for i := 0; i < townPending; i++ {
		login := fmt.Sprintf("p%04d@example.org", i)
		testutil.Must(env.Svc.JoinNetwork(env.Ctx, env.Person(login), w.net.ID, login))
	}
	opts := Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: env.Clock}
	w.srv = httptest.NewServer(NewHandlerWith(env.Svc, opts))
	t.Cleanup(w.srv.Close)
	return w, lastA, lastM
}

func townLogin(i int) string { return fmt.Sprintf("m%04d@example.org", i) }

// walk follows the link matched by re from first, collecting every page's body until no link
// is left. It returns the bodies in order.
func (b *browser) walk(first string, re *regexp.Regexp) []string {
	b.t.Helper()
	var bodies []string
	path := first
	for i := 0; i < 200; i++ {
		p := b.get(path)
		if p.Status != 200 {
			b.t.Fatalf("GET %s: status %d", path, p.Status)
		}
		bodies = append(bodies, p.Body)
		m := re.FindStringSubmatch(p.Body)
		if m == nil {
			return bodies
		}
		path = html.UnescapeString(m[1])
	}
	b.t.Fatal("paging does not terminate")
	return nil
}

// TestTownPaging shares one fixture (about 1,150 people) across the paging checks. The trade
// subtest runs last because it writes a ledger entry.
func TestTownPaging(t *testing.T) {
	w, lastA, lastM := newTownWorld(t)
	t.Run("directory pages reach every member", func(t *testing.T) { directoryPagesReachEveryMember(t, w) })
	t.Run("directory search reaches every match", func(t *testing.T) { directorySearchReachesEveryMatch(t, w) })
	t.Run("admin list and pending count", func(t *testing.T) { adminMemberListAndPendingCount(t, w) })
	t.Run("member past the caps resolves", func(t *testing.T) { memberPastCapStillResolves(t, w, lastA, lastM) })
}

// Nothing is hidden on the directory: every active member is reachable through Next links,
// the count is exact, and Previous returns the page before.
func directoryPagesReachEveryMember(t *testing.T, w *world) {
	b := w.browser(t)
	b.login("alice@example.org")
	bodies := b.walk(w.base()+"/directory", nextLinkRe)
	if len(bodies) < 2 {
		t.Fatalf("directory has %d page(s), want several", len(bodies))
	}
	want := townMembers + 3 // admin, alice, bob
	seen := map[string]bool{}
	for _, body := range bodies {
		for _, m := range dirEntryRe.FindAllStringSubmatch(body, -1) {
			if seen[m[1]] {
				t.Fatalf("member %s listed on two pages", m[2])
			}
			seen[m[1]] = true
		}
	}
	if len(seen) != want {
		t.Errorf("directory listed %d members, want %d", len(seen), want)
	}
	if m := foundLineRe.FindStringSubmatch(bodies[0]); m == nil || m[1] != fmt.Sprint(want) {
		t.Errorf("found line = %v, want %d", m, want)
	}
	// Previous from the second page returns the first page's rows.
	next := nextLinkRe.FindStringSubmatch(bodies[0])
	if next == nil {
		t.Fatal("first page has no Next link")
	}
	second := b.get(html.UnescapeString(next[1]))
	back := prevLinkRe.FindStringSubmatch(second.Body)
	if back == nil {
		t.Fatal("second page has no Previous link")
	}
	prev := b.get(html.UnescapeString(back[1]))
	if firstEntryID(t, prev.Body) != firstEntryID(t, bodies[0]) {
		t.Error("Previous did not return to the first page")
	}
}

func firstEntryID(t *testing.T, body string) string {
	t.Helper()
	m := dirEntryRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no directory entries on page")
	}
	return m[1]
}

// Searching the directory filters on the server, so matches past the old caps are found.
func directorySearchReachesEveryMatch(t *testing.T, w *world) {
	want := 0
	for i := 0; i < townMembers; i++ {
		if strings.Contains(townLogin(i), "m06") {
			want++
		}
	}
	b := w.browser(t)
	b.login("alice@example.org")
	bodies := b.walk(w.base()+"/directory?q=m06", nextLinkRe)
	got := 0
	for _, body := range bodies {
		got += len(dirEntryRe.FindAllString(body, -1))
	}
	if got != want {
		t.Errorf("search paged %d rows, want %d", got, want)
	}
	if m := foundLineRe.FindStringSubmatch(bodies[0]); m == nil || m[1] != fmt.Sprint(want) {
		t.Errorf("found line = %v, want %d", m, want)
	}
}

// The admin member list pages with Next and Previous, and the pending total is exact past 500.
func adminMemberListAndPendingCount(t *testing.T, w *world) {
	b := w.browser(t)
	b.login("admin@example.org")

	home := b.get(w.base() + "/admin")
	b.mustContain(home, fmt.Sprintf("%d membership requests are waiting.", townPending))

	pending := b.walk(w.base()+"/admin/members?status=pending", nextLinkRe)
	rows := 0
	for _, body := range pending {
		rows += len(adminRowRe.FindAllString(body, -1))
	}
	if rows != townPending {
		t.Errorf("pending list paged %d rows, want %d", rows, townPending)
	}
	b.mustContain(b.get(w.base()+"/admin/members?status=pending"), fmt.Sprintf("%d members match this filter.", townPending))

	all := b.walk(w.base()+"/admin/members?status=active", nextLinkRe)
	rows = 0
	for _, body := range all {
		rows += len(adminRowRe.FindAllString(body, -1))
	}
	if want := townMembers + 3; rows != want {
		t.Errorf("active list paged %d rows, want %d", rows, want)
	}

	// Previous from page two returns page one.
	first := b.get(w.base() + "/admin/members?status=pending")
	next := nextLinkRe.FindStringSubmatch(first.Body)
	if next == nil {
		t.Fatal("pending list has no Next link")
	}
	second := b.get(html.UnescapeString(next[1]))
	prev := prevLinkRe.FindStringSubmatch(second.Body)
	if prev == nil {
		t.Fatal("second page has no Previous link")
	}
	back := b.get(html.UnescapeString(prev[1]))
	if adminRowRe.FindString(back.Body) != adminRowRe.FindString(first.Body) {
		t.Error("Previous did not return to the first page")
	}
}

// A member past the old caps is still named where names are shown, and can be opened and
// traded with.
func memberPastCapStillResolves(t *testing.T, w *world, lastA domain.Actor, lastM *domain.Member) {
	last := townLogin(townMembers - 1)
	w.env.Trade(w.net.ID, w.alice, lastA, w.aliceM, lastM, 100)

	b := w.browser(t)
	b.login("alice@example.org")
	b.mustContain(b.get(w.base()+"/history"), last)
	b.mustContain(b.get(w.base()+"/trades/new"), last)
	b.mustContain(b.get(w.base()+"/members/"+lastM.ID), last)

	admin := w.browser(t)
	admin.login("admin@example.org")
	admin.mustContain(admin.get(w.base()+"/admin/adjust"), last)
}
