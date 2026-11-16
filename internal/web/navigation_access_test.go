package web

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// A2-11 / UX-001: readable sections remain discoverable after membership
// changes, while directory, listing and privileged permissions stay enforced.
func TestNavigationMatchesCurrentReadableSections(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	base := w.base()
	nav := func(t *testing.T, p *pg) string {
		t.Helper()
		_, rest, ok := strings.Cut(p.Body, `<nav aria-label="Main">`)
		if !ok {
			t.Fatal("main navigation missing")
		}
		content, _, ok := strings.Cut(rest, "</nav>")
		if !ok {
			t.Fatal("main navigation not closed")
		}
		return content
	}
	for _, step := range []struct {
		name    string
		active  bool
		prepare func() error
	}{
		{name: "active", active: true},
		{name: "suspended", prepare: func() error {
			return w.env.Svc.SuspendMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, "review")
		}},
		{name: "restored", active: true, prepare: func() error {
			return w.env.Svc.RestoreMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, "review complete")
		}},
		{name: "departed", prepare: func() error {
			_, err := w.env.Svc.DepartMember(w.env.Ctx, w.alice, w.net.ID, service.DepartInput{MemberID: w.aliceM.ID, Reason: "left"})
			return err
		}},
	} {
		t.Run(step.name, func(t *testing.T) {
			if step.prepare != nil {
				if err := step.prepare(); err != nil {
					t.Fatal(err)
				}
			}
			home := b.get(base)
			if home.Status != 200 {
				t.Fatalf("home status = %d", home.Status)
			}
			main := nav(t, home)
			for _, section := range []string{"/trades", "/history", "/disputes", "/policy", "/directory", "/listings", "/review", "/admin"} {
				want := 200
				if section == "/review" || section == "/admin" || (!step.active && (section == "/directory" || section == "/listings")) {
					want = 403
				}
				href := `href="` + base + section + `"`
				if shown := strings.Contains(main, href); shown != (want == 200) {
					t.Errorf("%s navigation visible = %v, expected readable = %v", section, shown, want == 200)
				}
				p := b.get(base + section)
				if p.Status != want {
					t.Errorf("%s status = %d, want %d", section, p.Status, want)
				}
				if want == 200 && !strings.Contains(nav(t, p), href+` aria-current="page"`) {
					t.Errorf("%s lacks current navigation marker", section)
				}
			}
		})
	}
}
