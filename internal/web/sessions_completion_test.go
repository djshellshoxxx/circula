package web

import (
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestAccountSessionsBulkRevocationRequiresConfirmationAndKeepsCurrent(t *testing.T) {
	w := newWorld(t)
	current, other, foreign := w.browser(t), w.browser(t), w.browser(t)
	current.login("alice@example.org")
	other.login("alice@example.org")
	foreign.login("bob@example.org")
	currentRaw, _, _ := browserSession(t, current)
	otherRaw, _, _ := browserSession(t, other)
	foreignRaw, _, _ := browserSession(t, foreign)
	const path = "/account/sessions/revoke-others"
	confirmation := current.get(path)
	if confirmation.Status != http.StatusOK {
		t.Fatalf("confirmation: %d", confirmation.Status)
	}
	current.mustContain(confirmation, "Sign out other sessions?", "You will stay signed in here", `for="f-confirm"`)
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); err != nil {
		t.Fatalf("GET ended another session: %v", err)
	}
	if p := current.post(path, confirmation, nil); p.Status != http.StatusUnprocessableEntity {
		t.Fatalf("missing confirmation: %d", p.Status)
	}
	if p := current.post(path, nil, url.Values{"confirm": {"yes"}, "csrf": {"invalid"}}); p.Status != http.StatusForbidden {
		t.Fatalf("invalid CSRF: %d", p.Status)
	}
	for i := 0; i < 2; i++ {
		if p := current.post(path, confirmation, url.Values{"confirm": {"yes"}}); p.Status != http.StatusOK || p.URL != "/account/sessions" {
			t.Fatalf("bulk revocation %d: %d at %s", i, p.Status, p.URL)
		}
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("other session survived: %v", err)
	}
	for _, raw := range []string{currentRaw, foreignRaw} {
		if _, err := w.env.Svc.Authenticate(w.env.Ctx, raw); err != nil {
			t.Fatalf("current or foreign session ended: %v", err)
		}
	}
	if p := other.get("/account/sessions"); p.URL != "/login" {
		t.Fatalf("revoked browser still has access: %s", p.URL)
	}
}

func TestAccountSessionsPreviousPageRoundTrip(t *testing.T) {
	w := newWorld(t)
	current := w.browser(t)
	current.login("alice@example.org")
	for i := 0; i < 29; i++ {
		w.env.Clock.Advance(time.Millisecond)
		if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", testPW); err != nil {
			t.Fatal(err)
		}
	}
	selectors := regexp.MustCompile(`/account/sessions/([0-9a-f]{64})/revoke`)
	link := func(body, direction string) string {
		t.Helper()
		match := regexp.MustCompile(`<a href="(/account/sessions\?` + direction + `=[^"]+)"`).FindStringSubmatch(body)
		if match == nil {
			t.Fatalf("no %s navigation link", direction)
		}
		return html.UnescapeString(match[1])
	}
	first := current.get("/account/sessions")
	second := current.get(link(first.Body, "after"))
	if second.Status != http.StatusOK || len(selectors.FindAllString(second.Body, -1)) != 5 {
		t.Fatalf("second page: %d", second.Status)
	}
	previous := current.get(link(second.Body, "before"))
	if previous.Status != http.StatusOK || strings.Join(selectors.FindAllString(previous.Body, -1), "|") != strings.Join(selectors.FindAllString(first.Body, -1), "|") {
		t.Fatal("Previous did not restore the first page")
	}
	if p := current.get("/account/sessions?after=invalid&before=invalid"); p.Status != http.StatusUnprocessableEntity {
		t.Fatalf("conflicting directions accepted: %d", p.Status)
	}
}

func TestAccountSessionsRejectMalformedPaginationQuery(t *testing.T) {
	w := newWorld(t)
	current := w.browser(t)
	current.login("alice@example.org")
	for _, query := range []string{"after=%GG", "before=%GG", "after=&after=", "before=&before=", "limit=1000"} {
		if p := current.get("/account/sessions?" + query); p.Status != http.StatusUnprocessableEntity {
			t.Errorf("query %q: %d, want 422", query, p.Status)
		}
	}
}
