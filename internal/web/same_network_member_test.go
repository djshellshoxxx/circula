package web

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A4 / FR-020 / SEC-016: a member's view of another member of the same
// network (profile and directory) shows directory data only: display name,
// active listings and the trade and report actions. It never shows the other
// member's login, balance or trades. A member who is not active has no
// profile: the page is the same not-found as for a missing id.
func TestSameNetworkMemberPageShowsDirectoryDataOnly(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx, svc := w.env.Ctx, w.env.Svc
	// rob's display name differs from his login, so a leaked login is visible.
	rob := w.env.Person("rob.login@example.org")
	tok, _ := testutil.Must2(svc.CreateInvitation(ctx, w.admin, w.net.ID, service.InviteInput{}))
	robM := testutil.Must(svc.RedeemInvitation(ctx, rob, tok, "Rob Baker"))
	carol, carolM := w.env.Member(w.admin, w.net, "carol@example.org")
	w.env.Trade(w.net.ID, rob, carol, robM, carolM, 4321) // 43.21: rob's balance, a trade alice is no party to
	if err := svc.SetDefaultFlag(ctx, w.admin, w.net.ID, robM.ID, true, "test"); err != nil {
		t.Fatal(err)
	}
	testutil.Must(svc.CreateListing(ctx, rob, w.net.ID, service.ListingInput{Type: "offer", Title: "Sourdough loaves", IdempotencyKey: "k-rob-1"}))
	sue, sueM := w.env.Member(w.admin, w.net, "sue@example.org")
	testutil.Must(svc.CreateListing(ctx, sue, w.net.ID, service.ListingInput{Type: "offer", Title: "Suspended soap", IdempotencyKey: "k-sue-1"}))
	if err := svc.SuspendMember(ctx, w.admin, w.net.ID, sueM.ID, "test"); err != nil {
		t.Fatal(err)
	}
	dee, deeM := w.env.Member(w.admin, w.net, "dee@example.org")
	testutil.Must(svc.DepartMember(ctx, dee, w.net.ID, service.DepartInput{}))

	a := w.browser(t)
	a.login("alice@example.org")
	p := a.get(b + "/members/" + robM.ID)
	if p.Status != 200 {
		t.Fatalf("profile of an active member: %d", p.Status)
	}
	a.mustContain(p, "Rob Baker", "Sourdough loaves")
	if !strings.Contains(p.Header.Get("X-Robots-Tag"), "noindex") {
		t.Error("member page lacks the noindex directive (SEC-016)")
	}
	for _, leak := range []string{"rob.login@example.org", "43.21", "4321", carolM.DisplayName, "Default"} {
		if strings.Contains(p.Body, leak) {
			t.Errorf("another member's profile shows %q", leak)
		}
	}

	missing := a.get(b + "/members/018f0000-0000-7000-8000-ffffffffffff")
	if missing.Status != 404 {
		t.Fatalf("missing member: %d", missing.Status)
	}
	for name, id := range map[string]string{"suspended": sueM.ID, "departed": deeM.ID} {
		q := a.get(b + "/members/" + id)
		if q.Status != 404 || title(q) != title(missing) {
			t.Errorf("%s member's page: %d %q, want the not-found page %q", name, q.Status, title(q), title(missing))
		}
		if strings.Contains(q.Body, "Suspended soap") {
			t.Errorf("%s member's listing shown", name)
		}
	}

	dir := a.get(b + "/directory")
	if dir.Status != 200 {
		t.Fatalf("directory: %d", dir.Status)
	}
	a.mustContain(dir, "Rob Baker", carolM.DisplayName)
	for _, hidden := range []string{sueM.DisplayName, deeM.DisplayName, "rob.login@example.org", "43.21"} {
		if strings.Contains(dir.Body, hidden) {
			t.Errorf("directory shows %q", hidden)
		}
	}
}
