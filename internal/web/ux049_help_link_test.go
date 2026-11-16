package web

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// helpFooter pins the shared Help link to one place: the first link of the second
// paragraph of the page footer, after the main landmark.
var helpFooter = regexp.MustCompile(`</main>\s*<footer class="foot">\s*<p>[^<]*</p>\s*<p><a href="/help">Help: credits, balances and statuses</a>`)

func assertHelpLocation(t *testing.T, where, body string) {
	t.Helper()
	if !helpFooter.MatchString(body) {
		t.Errorf("%s: the Help link is not in its footer location", where)
	}
	foot := body[strings.LastIndex(body, `<footer class="foot">`)+1:]
	if n := strings.Count(foot, `href="/help"`); n != 1 {
		t.Errorf("%s: %d Help links in the footer, want 1", where, n)
	}
	if i := strings.Index(body, "<main"); i >= 0 && strings.Contains(body[:i], `href="/help`) {
		t.Errorf("%s: a Help link moved into the header", where)
	}
}

// UX-049 (WCAG 3.2.6): every page template renders through the base layout, whose footer
// carries the Help link; no page replaces the layout or the footer.
func TestEveryPageTemplateHasHelpLink(t *testing.T) {
	pages := mustLoadTemplates()
	if len(pages) < 50 {
		t.Fatalf("only %d page templates loaded", len(pages))
	}
	names := make([]string, 0, len(pages))
	for n := range pages {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		tmpl, err := pages[name].Clone()
		if err != nil {
			t.Fatal(err)
		}
		// Only the screen content is replaced; whatever else the page file defines stays.
		stub := `{{define "content"}}<p>content of ` + name + `</p>{{end}}`
		if _, err := tmpl.Parse(stub); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, p := range []*page{
			{Title: name},
			{Title: name, Authed: true, Slug: "town", NetName: "Town", ReqID: "req-1", Nav: []navItem{{Label: "Home", Href: "/n/town"}}},
		} {
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, "base", p); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !strings.Contains(buf.String(), "<p>content of "+name+"</p>") {
				t.Errorf("%s: the page does not render its content inside the base layout", name)
			}
			assertHelpLocation(t, name, buf.String())
		}
	}
}

var getRoute = regexp.MustCompile(`HandleFunc\("GET (/[^"]*)"`)

// UX-049: every GET route, for an anonymous visitor, a member and an administrator, answers
// with a page whose Help link sits in the same footer location, error pages included.
func TestEveryRouteHasHelpLinkInFooter(t *testing.T) {
	w := newWorld(t)
	ctx := w.env.Ctx
	in := testutil.Must(w.env.Svc.ProposeTransaction(ctx, w.bob, w.net.ID, service.ProposeInput{CounterpartyID: w.aliceM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.env.Key()}))

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range getRoute.FindAllStringSubmatch(string(src), -1) {
			routes[m[1]] = true
		}
	}
	if len(routes) < 40 {
		t.Fatalf("found only %d GET routes", len(routes))
	}
	fill := strings.NewReplacer("{slug}", w.net.Slug, "{$}", "", "{token}", "no-such-token")
	path := func(r string) string {
		switch {
		case strings.HasPrefix(r, "/n/{slug}/trades/{id}"):
			r = strings.Replace(strings.Replace(r, "{id}", in.ID, 1), "{action}", "accept", 1)
		case strings.HasPrefix(r, "/n/{slug}/admin/members/{id}"):
			r = strings.Replace(strings.Replace(r, "{id}", w.bobM.ID, 1), "{action}", "suspend", 1)
		case strings.Contains(r, "{id}"):
			r = strings.Replace(r, "{id}", w.bobM.ID, 1)
		}
		return fill.Replace(r)
	}

	anon := w.browser(t)
	member := w.browser(t)
	member.login("alice@example.org")
	admin := w.browser(t)
	admin.login("admin@example.org")
	viewers := map[string]*browser{"anonymous": anon, "member": member, "admin": admin}

	sorted := make([]string, 0, len(routes))
	for r := range routes {
		sorted = append(sorted, r)
	}
	sort.Strings(sorted)
	checked := 0
	for _, r := range sorted {
		if r == "/favicon.ico" || r == "/robots.txt" {
			continue
		}
		u := path(r)
		if strings.Contains(u, "{") {
			t.Errorf("route %s: unfilled placeholder in %s", r, u)
			continue
		}
		html := false
		for who, b := range viewers {
			p := b.get(u)
			if !strings.HasPrefix(p.Header.Get("Content-Type"), "text/html") {
				continue
			}
			html = true
			checked++
			if p.Status >= 500 {
				t.Errorf("%s GET %s: status %d", who, u, p.Status)
			}
			assertHelpLocation(t, who+" GET "+u+" -> "+p.URL+" ("+itoa(p.Status)+")", p.Body)
		}
		if !html {
			t.Errorf("route %s (%s) answered no viewer with a page", r, u)
		}
	}
	// Error pages are pages of the same site: a missing page and a refused one keep the link.
	for _, u := range []string{"/no/such/page", "/n/no-such-network", "/n/" + w.net.Slug + "/trades/" + url.PathEscape("no-such-id")} {
		p := member.get(u)
		assertHelpLocation(t, "member GET "+u+" ("+itoa(p.Status)+")", p.Body)
		checked++
	}
	t.Logf("checked %d pages over %d routes", checked, len(sorted))
}
