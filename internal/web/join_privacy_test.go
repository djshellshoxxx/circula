package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-10: the join endpoints must not become a directory of private networks
// for signed-in people who have no membership or invitation.
func TestInvitationOnlyJoinHidesNetworkFromNonmembers(t *testing.T) {
	w := newWorld(t)
	outsider := w.env.Person("outsider@example.org")
	b := w.browser(t)
	home := b.login("outsider@example.org")
	mainContent := func(p *pg) string {
		t.Helper()
		start, end := strings.Index(p.Body, "<main"), strings.Index(p.Body, "</main>")
		if start < 0 || end <= start {
			t.Fatal("response has no main content")
		}
		return p.Body[start:end]
	}
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			request := func(path string) *pg {
				if method == "POST" {
					return b.post(path, home, url.Values{"display_name": {"Outsider"}})
				}
				return b.get(path)
			}
			missing := request("/n/missing-network/join")
			private := request(w.base() + "/join")
			if private.Status != 404 || missing.Status != 404 {
				t.Fatalf("private join status %d, missing status %d; want both 404", private.Status, missing.Status)
			}
			if mainContent(private) != mainContent(missing) {
				t.Error("private join response differs from an unknown network")
			}
			for _, secret := range []string{w.net.Name, w.net.ID, w.net.Slug} {
				if strings.Contains(private.Body, secret) {
					t.Errorf("private join response revealed %q", secret)
				}
			}
		})
	}
	if memberships := tu.Must(w.env.Svc.MyMemberships(w.env.Ctx, outsider)); len(memberships) != 0 {
		t.Fatal("private join probe created a membership")
	}
	// Possession of an invitation continues to reveal its join form and allow
	// acceptance, after the same caller was refused the slug-based route.
	token, _ := tu.Must2(w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	form := b.get("/join/" + token)
	b.mustContain(form, "Accept your invitation")
	joined := b.post("/join/"+token, form, url.Values{"display_name": {"Invited member"}})
	if joined.Status != 200 || strings.TrimSuffix(joined.URL, "/") != w.base() {
		t.Fatalf("invitation acceptance: status %d, path %s", joined.Status, joined.URL)
	}
	b.mustContain(joined, "Available to spend")
	// Existing members still see their network and retain idempotent navigation.
	memberForm := b.get(w.base() + "/join")
	if memberForm.Status != 200 {
		t.Fatalf("member join page status %d", memberForm.Status)
	}
	returned := b.post(w.base()+"/join", memberForm, url.Values{"display_name": {"Invited member"}})
	if returned.Status != 200 || strings.TrimSuffix(returned.URL, "/") != w.base() {
		t.Fatalf("existing member navigation: status %d, path %s", returned.Status, returned.URL)
	}
}

func TestPublicJoinModesRemainAccessible(t *testing.T) {
	for _, mode := range []domain.JoinMode{domain.JoinOpen, domain.JoinApproval} {
		t.Run(string(mode), func(t *testing.T) {
			w := newWorld(t)
			net, _ := w.env.Network(w.admin, "public", func(p *domain.Policy) { p.JoinMode = mode })
			person := w.env.Person("joiner@example.org")
			b := w.browser(t)
			b.login("joiner@example.org")
			form := b.get("/n/public/join")
			if form.Status != 200 {
				t.Fatalf("public join page status %d", form.Status)
			}
			joined := b.post("/n/public/join", form, url.Values{"display_name": {"Joiner"}})
			if joined.Status != 200 || strings.TrimSuffix(joined.URL, "/") != "/n/public" {
				t.Fatalf("public join result: status %d, path %s", joined.Status, joined.URL)
			}
			memberships := tu.Must(w.env.Svc.MyMemberships(w.env.Ctx, person))
			want := domain.MemberActive
			if mode == domain.JoinApproval {
				want = domain.MemberPending
			}
			if len(memberships) != 1 || memberships[0].NetworkID != net.ID || memberships[0].Status != want {
				t.Fatalf("public join membership: %+v", memberships)
			}
		})
	}
}
