package service_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// sameNetworkWorld adds, to the standard world (admin, ana, ben), an active
// member cy who trades privately with ben, a suspended member, a pending
// member and a departed member. ben carries non-directory data (a default
// flag, a balance, a paused listing) that ana must not see.
type sameNetworkWorld struct {
	*world
	c                              domain.Actor
	cM, suspM, pendM, deptM        *domain.Member
	benCyTx                        *domain.Transaction
	benActive, benPaused, suspList *domain.Listing
}

func newSameNetworkWorld(t *testing.T) *sameNetworkWorld {
	t.Helper()
	w := &sameNetworkWorld{world: newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })}
	w.c, w.cM = w.Member(w.admin, w.net, "cy@example.org")
	w.benCyTx = w.Trade(w.net.ID, w.b, w.c, w.bM, w.cM, 4321)
	if err := w.Svc.SetDefaultFlag(w.Ctx, w.admin, w.net.ID, w.bM.ID, true, "test"); err != nil {
		t.Fatal(err)
	}
	listing := func(by domain.Actor, title string) *domain.Listing {
		return tu.Must(w.Svc.CreateListing(w.Ctx, by, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: title, IdempotencyKey: w.Key()}))
	}
	w.benActive, w.benPaused = listing(w.b, "Bread"), listing(w.b, "Paused jam")
	paused := domain.ListingPaused
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.b, w.net.ID, service.ListingUpdate{ListingID: w.benPaused.ID, Status: &paused}))

	var susp domain.Actor
	susp, w.suspM = w.Member(w.admin, w.net, "sue@example.org")
	w.suspList = listing(susp, "Suspended soap")
	if err := w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.suspM.ID, "test"); err != nil {
		t.Fatal(err)
	}
	_, w.pendM = w.joinPending("pat@example.org")
	dept, deptM := w.Member(w.admin, w.net, "dee@example.org")
	tu.Must(w.Svc.DepartMember(w.Ctx, dept, w.net.ID, service.DepartInput{}))
	w.deptM = deptM
	return w
}

// A4 / FR-020 / SEC-016 / API-043: an active member reading another member
// of the same network gets the directory fields only (id, network, display
// name, status), and only for active members. Suspended, pending and departed
// members are indistinguishable from a missing id.
func TestSameNetworkMemberReadExposesDirectoryFieldsOnly(t *testing.T) {
	w := newSameNetworkWorld(t)
	got := tu.Must(w.Svc.GetMember(w.Ctx, w.a, w.net.ID, w.bM.ID))
	want := &domain.Member{ID: w.bM.ID, NetworkID: w.net.ID, DisplayName: w.bM.DisplayName, Status: domain.MemberActive}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("member read of another member = %+v, want directory fields only %+v", got, want)
	}
	// The member's own record and the administrator's view are complete.
	if self := tu.Must(w.Svc.GetMember(w.Ctx, w.b, w.net.ID, w.bM.ID)); self.PersonID == "" || !self.DefaultFlag {
		t.Errorf("own record lacks fields: %+v", self)
	}
	if adm := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.bM.ID)); adm.PersonID == "" || !adm.DefaultFlag {
		t.Errorf("admin record lacks fields: %+v", adm)
	}

	_, missing := w.Svc.GetMember(w.Ctx, w.a, w.net.ID, "no-such-member")
	wantCode(t, missing, domain.ErrNotFound)
	for name, id := range map[string]string{"suspended": w.suspM.ID, "pending": w.pendM.ID, "departed": w.deptM.ID} {
		_, err := w.Svc.GetMember(w.Ctx, w.a, w.net.ID, id)
		wantCode(t, err, domain.ErrNotFound)
		if err != nil && err.Error() != missing.Error() {
			t.Errorf("%s member distinguishable from a missing one: %q vs %q", name, err, missing)
		}
	}
}

// The directory, its pages and its count list active members only, as
// DirectoryEntry values that carry no balance, limit, flag or login.
func TestSameNetworkDirectoryListsActiveMembersOnly(t *testing.T) {
	w := newSameNetworkWorld(t)
	if f := reflect.TypeFor[service.DirectoryEntry]().NumField(); f != 2 {
		t.Fatalf("DirectoryEntry has %d fields; review what the directory exposes (FR-020) and update this test", f)
	}
	wantIDs := []string{w.adminM.ID, w.aM.ID, w.bM.ID, w.cM.ID}
	slices.Sort(wantIDs)
	ids := func(es []service.DirectoryEntry) []string {
		var out []string
		for _, e := range es {
			out = append(out, e.MemberID)
		}
		slices.Sort(out)
		return out
	}
	list := tu.Must(w.Svc.ListDirectory(w.Ctx, w.a, w.net.ID, service.Page{}))
	if got := ids(list); !slices.Equal(got, wantIDs) {
		t.Errorf("directory = %v, want active members %v", got, wantIDs)
	}
	pg := tu.Must(w.Svc.ListDirectoryPage(w.Ctx, w.a, w.net.ID, service.MemberPageQuery{}))
	if got := ids(pg.Entries); !slices.Equal(got, wantIDs) {
		t.Errorf("directory page = %v, want %v", got, wantIDs)
	}
	if n := tu.Must(w.Svc.CountDirectory(w.Ctx, w.a, w.net.ID, "")); n != len(wantIDs) {
		t.Errorf("directory count = %d, want %d", n, len(wantIDs))
	}
}

// Balances, limits, history and the full member list of another member are
// not readable by a member: the member reads take no member id, and every
// admin read with one is refused.
func TestSameNetworkMemberCannotReadOthersBalanceOrHistory(t *testing.T) {
	w := newSameNetworkWorld(t)
	// Own reads are own: ana has no postings, ben's trade with cy is not hers.
	bv := tu.Must(w.Svc.GetBalance(w.Ctx, w.a, w.net.ID))
	if bv.Balance != 0 {
		t.Errorf("ana's balance view shows %d", bv.Balance)
	}
	if es := tu.Must(w.Svc.ListEntries(w.Ctx, w.a, w.net.ID, service.EntryQuery{})); len(es) != 0 {
		t.Errorf("ana's entries include %d rows of other members", len(es))
	}
	if txs := tu.Must(w.Svc.ListTransactions(w.Ctx, w.a, w.net.ID, nil, service.Page{})); len(txs) != 0 {
		t.Errorf("ana's history includes %d transactions of other members", len(txs))
	}
	_, err := w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, w.benCyTx.ID)
	wantCode(t, err, domain.ErrNotFound)

	refused := map[string]func() error{
		"AdminGetMemberBalance": func() error {
			_, err := w.Svc.AdminGetMemberBalance(w.Ctx, w.a, w.net.ID, w.bM.ID)
			return err
		},
		"PeekMemberBalance": func() error { _, err := w.Svc.PeekMemberBalance(w.Ctx, w.a, w.net.ID, w.bM.ID); return err },
		"AdminMemberPage": func() error {
			_, err := w.Svc.AdminMemberPage(w.Ctx, w.a, w.net.ID, nil, service.MemberPageQuery{})
			return err
		},
		"AdminListMembers": func() error {
			_, err := w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{})
			return err
		},
		"CountMembers":        func() error { _, err := w.Svc.CountMembers(w.Ctx, w.a, w.net.ID, nil); return err },
		"ListInactiveMembers": func() error { _, err := w.Svc.ListInactiveMembers(w.Ctx, w.a, w.net.ID, service.Page{}); return err },
	}
	for name, run := range refused {
		if err := run(); domain.CodeOf(err) != domain.CodeForbidden {
			t.Errorf("%s by a member about another member: got %v, want forbidden", name, err)
		}
	}
}

// Listings by another member: only that member's active listings, and none
// for a member who is not active (FR-023, FR-025).
func TestSameNetworkMemberListingsShowActiveOnly(t *testing.T) {
	w := newSameNetworkWorld(t)
	got := tu.Must(w.Svc.ListListings(w.Ctx, w.a, w.net.ID, service.ListingQuery{MemberID: w.bM.ID}))
	if len(got) != 1 || got[0].ID != w.benActive.ID {
		t.Errorf("ben's listings seen by ana = %v, want only the active one", got)
	}
	_, err := w.Svc.GetListing(w.Ctx, w.a, w.net.ID, w.benPaused.ID)
	wantCode(t, err, domain.ErrNotFound)
	if got := tu.Must(w.Svc.ListListings(w.Ctx, w.a, w.net.ID, service.ListingQuery{MemberID: w.suspM.ID})); len(got) != 0 {
		t.Errorf("suspended member's listings visible: %v", got)
	}
	_, err = w.Svc.GetListing(w.Ctx, w.a, w.net.ID, w.suspList.ID)
	wantCode(t, err, domain.ErrNotFound)
}
