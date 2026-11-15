package web

// Beta qualification (docs/plans/circula-completion-matrix.md, A1 and A7): the
// session cookie is minted by the server at sign-in and is invalidated on the
// server at sign-out. These checks run over real HTTP, not only at service level.

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/httpx"
)

func sessionValue(t *testing.T, b *browser) string {
	t.Helper()
	u, _ := url.Parse(b.url)
	for _, c := range b.c.Jar.Cookies(u) {
		if c.Name == httpx.SessionCookie {
			return c.Value
		}
	}
	return ""
}

// replay sends only the given session cookie value, as an attacker holding a
// stolen or planted value would, and returns the final path reached.
func replay(t *testing.T, w *world, value string) string {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(w.srv.URL)
	jar.SetCookies(u, []*http.Cookie{{Name: httpx.SessionCookie, Value: value, Path: "/"}})
	b := &browser{t: t, w: w, url: w.srv.URL, c: &http.Client{Jar: jar}}
	return b.get(w.base()).URL
}

// A7: after sign-out, the old cookie value no longer authenticates anyone.
func TestLogoutInvalidatesReplayedCookie(t *testing.T) {
	w := newWorld(t)
	alice := w.browser(t)
	alice.login("alice@example.org")
	stolen := sessionValue(t, alice)
	if stolen == "" {
		t.Fatal("no session cookie after sign-in")
	}
	if got := replay(t, w, stolen); got == "/login" {
		t.Fatalf("control: a live session cookie was sent to %s", got)
	}
	hp := alice.get(w.base())
	alice.mustContain(alice.post("/logout", hp, nil), "You are signed out")
	if got := replay(t, w, stolen); got != "/login" {
		t.Fatalf("replayed cookie after sign-out reached %s, want /login", got)
	}
}

// A1: a session value planted before sign-in is never adopted. Sign-in issues a
// fresh value, the planted one stays unauthenticated, and signing in again
// rotates the value.
func TestLoginDoesNotAdoptPlantedSessionCookie(t *testing.T) {
	w := newWorld(t)
	victim := w.browser(t)
	u, _ := url.Parse(w.srv.URL)
	const planted = "attacker-chosen-session-value-0123456789abcdef"
	victim.c.Jar.SetCookies(u, []*http.Cookie{{Name: httpx.SessionCookie, Value: planted, Path: "/"}})

	victim.login("alice@example.org")
	first := sessionValue(t, victim)
	if first == "" || first == planted {
		t.Fatalf("sign-in kept the planted session value (%q)", first)
	}
	if got := replay(t, w, planted); got != "/login" {
		t.Fatalf("planted value authenticated: reached %s", got)
	}

	victim.login("alice@example.org")
	if second := sessionValue(t, victim); second == "" || second == first {
		t.Fatalf("signing in again did not rotate the session value")
	}
}
