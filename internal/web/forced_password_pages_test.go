package web

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-17: home must enforce SEC-008 itself. With exactly one membership its
// network redirect hid the missing check; zero or several memberships did not.
func TestForcedPasswordGuardsHomeAndInvitation(t *testing.T) {
	for _, memberships := range []int{0, 2} {
		t.Run(fmt.Sprintf("%d_memberships", memberships), func(t *testing.T) {
			w := newWorld(t)
			const login = "reset@example.org"
			actor := w.env.Person(login)
			for i := 0; i < memberships; i++ {
				w.env.Network(actor, fmt.Sprintf("reset-%d", i), nil)
			}
			token, _ := testutil.Must2(w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{}))
			invite := "/join/" + token
			temp := testutil.Must(w.env.Svc.OperatorResetPassword(w.env.Ctx, domain.SystemActor("fixture"), login, "lost password"))
			b := w.browser(t)
			// The invitation still explains sign-in to an anonymous visitor.
			b.mustContain(b.get(invite), "To accept, sign in or create an account first")
			next := "/"
			if memberships == 2 {
				next = invite
			}
			p := b.post("/login", b.get("/login"), url.Values{"login": {login}, "password": {temp}, "next": {next}})
			if p.URL != "/account/password" {
				t.Errorf("temporary-password login landed on %s, want /account/password", p.URL)
			}
			for _, path := range []string{"/", invite} {
				p := b.get(path)
				if p.Status != 200 || p.URL != "/account/password" {
					t.Errorf("GET %s: %d at %s; want password form", path, p.Status, p.URL)
				} else {
					b.mustContain(p, "Choose a new password to continue", "Current password")
				}
			}
			if got := len(testutil.Must(w.env.Svc.MyMemberships(w.env.Ctx, actor))); got != memberships {
				t.Fatalf("visiting invitation changed memberships: %d, want %d", got, memberships)
			}
			// Public recovery and sign-in remain reachable while the forced
			// change is pending, and the permitted password write still works.
			for _, path := range []string{"/login", "/recover", "/account/password"} {
				if p := b.get(path); p.Status != 200 || p.URL != path {
					t.Fatalf("GET %s: %d at %s", path, p.Status, p.URL)
				}
			}
			const fresh = "my own replacement passphrase"
			changed := b.post("/account/password", b.get("/account/password"), url.Values{
				"current_password": {temp}, "password": {fresh}, "password2": {fresh}})
			if changed.URL != "/login" {
				t.Fatalf("password rotation landed on %s", changed.URL)
			}
			p = b.post("/login", b.get("/login"), url.Values{"login": {login}, "password": {fresh}})
			if p.URL != "/" || p.Status != 200 {
				t.Fatalf("login after rotation: %d at %s", p.Status, p.URL)
			}
			b.mustContain(p, "Your networks")
			form := b.get(invite)
			b.mustContain(form, "Accept your invitation", "Your display name")
			joined := b.post(invite, form, url.Values{"display_name": {"Reset member"}})
			b.mustContain(joined, "You have joined Net town")
			if got := len(testutil.Must(w.env.Svc.MyMemberships(w.env.Ctx, actor))); got != memberships+1 {
				t.Fatalf("invitation was not redeemable after rotation: memberships=%d", got)
			}
		})
	}
}
