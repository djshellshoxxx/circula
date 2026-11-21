package web

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func requireAboutPage(t *testing.T, w *world, br *browser) {
	t.Helper()
	p := br.get(w.base() + "/about")
	if p.Status != 200 || p.URL != w.base()+"/about" {
		t.Fatalf("about returned %d at %s", p.Status, p.URL)
	}
	_, main, ok := strings.Cut(p.Body, `<main id="main"`)
	if !ok {
		t.Fatal("about page has no main content")
	}
	main, _, _ = strings.Cut(main, "</main>")
	if !strings.Contains(main, messages.Text("footer.credits")) {
		t.Fatal("approved acceptance statement is missing from the About content")
	}
	br.mustContain(p, `href="`+w.base()+`/policy">Rules</a>`, `href="`+w.base()+`/about">About this network</a>`)
	checkStructure(t, p.URL, p)
}

// VIS-002: both required placements use the approved statement. The new
// informational route inherits the existing network-read authority.
func TestNetworkAboutPageAndReadAuthority(t *testing.T) {
	w := newWorld(t)
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), messages.Text("footer.credits")) {
		t.Fatal("README lacks the approved acceptance statement")
	}
	anonymous := w.browser(t).get(w.base() + "/about")
	if anonymous.URL != "/login" {
		t.Fatalf("anonymous request reached %s", anonymous.URL)
	}
	w.env.Person("outside@example.org")
	outside := w.browser(t)
	outside.login("outside@example.org")
	for _, path := range []string{w.base() + "/about", "/n/missing-network/about"} {
		if p := outside.get(path); p.Status != 404 {
			t.Errorf("non-member about request %s returned %d", path, p.Status)
		}
	}
	br := w.browser(t)
	br.login("alice@example.org")
	requireAboutPage(t, w, br)
	if err := w.env.Svc.SuspendNetwork(w.env.Ctx, w.admin, w.net.ID, "Trading paused for maintenance"); err != nil {
		t.Fatal(err)
	}
	requireAboutPage(t, w, br)
	if err := w.env.Svc.ResumeNetwork(w.env.Ctx, w.admin, w.net.ID, "Maintenance finished"); err != nil {
		t.Fatal(err)
	}
	if err := w.env.Svc.SuspendMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, "Membership review"); err != nil {
		t.Fatal(err)
	}
	requireAboutPage(t, w, br)
	if err := w.env.Svc.RestoreMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, "Membership review finished"); err != nil {
		t.Fatal(err)
	}
	testutil.Must(w.env.Svc.DepartMember(w.env.Ctx, w.alice, w.net.ID, service.DepartInput{MemberID: w.aliceM.ID, Reason: "Member left"}))
	requireAboutPage(t, w, br)
	erased := testutil.Must(w.env.Svc.EraseMyData(w.env.Ctx, w.alice, w.net.ID, "Member requested erasure"))
	if !erased.PersonErased {
		t.Fatal("fixture did not erase the former member's only account")
	}
	if p := br.get(w.base() + "/about"); p.URL != "/login" {
		t.Fatalf("erased person retained about access: %d %s", p.Status, p.URL)
	}
}

func TestNetworkAboutDuringClosure(t *testing.T) {
	w := newWorld(t)
	testutil.Must(w.env.Svc.ScheduleClosure(w.env.Ctx, w.admin, w.net.ID, service.ClosureInput{Reason: "Network finishing"}))
	w.env.Clock.Advance(31 * 24 * time.Hour)
	if n := testutil.Must(w.env.Svc.FinaliseClosures(w.env.Ctx, domain.SystemActor("about-test"))); n != 1 {
		t.Fatalf("closed %d networks", n)
	}
	br := w.browser(t)
	br.login("alice@example.org")
	requireAboutPage(t, w, br)
	w.env.Clock.Advance(91 * 24 * time.Hour)
	if n := testutil.Must(w.env.Svc.EraseClosedNetworks(w.env.Ctx, domain.SystemActor("about-test"))); n != 1 {
		t.Fatalf("erased %d networks", n)
	}
	if p := br.get(w.base() + "/about"); p.URL != "/login" {
		t.Fatalf("erased network retained about access: %d %s", p.Status, p.URL)
	}
}
