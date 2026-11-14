package web

import (
	"html"
	"net/url"
	"regexp"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// Return links carry the local path to come back to in their "next" query
// parameter (SEC-024). html/template already query-escapes a value placed in
// an href query, so a second escape in the template made the parameter decode
// to "%2Fjoin%2F..." instead of "/join/...", and the invitation was lost after
// registration (zz-noscript.spec.js, QA-034 return path).

var returnLinkRe = regexp.MustCompile(`<a[^>]*href="(/(?:login|register)\?[^"]*)"[^>]*>([^<]+)</a>`)

// nextOf returns the decoded next parameter of the first link whose text is label.
func nextOf(t *testing.T, body, label string) string {
	t.Helper()
	for _, m := range returnLinkRe.FindAllStringSubmatch(body, -1) {
		if m[2] != label {
			continue
		}
		u, err := url.Parse(html.UnescapeString(m[1]))
		if err != nil {
			t.Fatalf("link %q: %v", m[1], err)
		}
		return u.Query().Get("next")
	}
	t.Fatalf("no login/register link labelled %q", label)
	return ""
}

func TestReturnLinksDecodeToTheOriginalPath(t *testing.T) {
	w := newWorld(t)
	tok, _, err := w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	want := "/join/" + tok
	b := w.browser(t)

	anon := b.get(want)
	for _, label := range []string{"Sign in", "Create an account"} {
		if got := nextOf(t, anon.Body, label); got != want {
			t.Errorf("invitation page %q link: next=%q, want %q", label, got, want)
		}
	}
	reg := b.get("/register?next=" + url.QueryEscape(want))
	if got := nextOf(t, reg.Body, "Sign in"); got != want {
		t.Errorf("register page sign-in link: next=%q, want %q", got, want)
	}
	lf := b.get("/login?next=" + url.QueryEscape(want))
	if got := nextOf(t, lf.Body, "Create an account"); got != want {
		t.Errorf("login page register link: next=%q, want %q", got, want)
	}
	// A path with its own query survives intact as one parameter.
	q := "/n/town/trades?status=pending&page=2"
	lq := b.get("/login?next=" + url.QueryEscape(q))
	if got := nextOf(t, lq.Body, "Create an account"); got != q {
		t.Errorf("login page register link with query: next=%q, want %q", got, q)
	}
}
