package web

import (
	"net/url"
	"strings"
	"testing"
)

// FR-012: join attempts are rate limited and answer retry-later.
func TestJoinRateLimited(t *testing.T) {
	w := newWorld(t)
	w.env.Person("zed@example.org")
	z := w.browser(t)
	z.login("zed@example.org")
	f := z.get("/n/town/join")
	var last *pg
	for i := 0; i < 12; i++ {
		last = z.post("/n/town/join", f, url.Values{"display_name": {""}})
	}
	if last.Status != 429 || last.Header.Get("Retry-After") == "" || !strings.Contains(last.Body, "Too many join attempts") {
		t.Fatalf("status %d retry-after %q", last.Status, last.Header.Get("Retry-After"))
	}
}

// FR-025: every page asks crawlers not to index, and robots.txt disallows all.
func TestNoIndex(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	for _, path := range []string{"/login", "/help", "/no/such/page"} {
		p := b.get(path)
		if !strings.Contains(p.Header.Get("X-Robots-Tag"), "noindex") {
			t.Errorf("%s: no X-Robots-Tag", path)
		}
		if !strings.Contains(p.Body, `name="robots" content="noindex`) {
			t.Errorf("%s: no robots meta", path)
		}
	}
	r := b.get("/robots.txt")
	if r.Status != 200 || !strings.Contains(r.Body, "Disallow: /") {
		t.Errorf("robots.txt: %d %q", r.Status, r.Body)
	}
	// A crawler without a session gets no member data.
	if p := b.get("/n/town/listings"); strings.Contains(p.Body, "alice") {
		t.Error("anonymous listing leak")
	}
}
