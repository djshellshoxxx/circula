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
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

func browserSession(t *testing.T, b *browser) (string, domain.Actor, string) {
	t.Helper()
	u, err := url.Parse(b.url)
	if err != nil {
		t.Fatal(err)
	}
	for _, cookie := range b.c.Jar.Cookies(u) {
		if cookie.Name != httpx.SessionCookie {
			continue
		}
		a, err := b.w.env.Svc.Authenticate(b.w.env.Ctx, cookie.Value)
		if err != nil {
			t.Fatal(err)
		}
		for after := ""; ; {
			page, err := b.w.env.Svc.ListOwnSessions(b.w.env.Ctx, a, service.Page{AfterID: after})
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range page.Rows {
				if row.Current {
					return cookie.Value, a, row.ID
				}
			}
			if page.Next == "" {
				break
			}
			if page.Next == after {
				t.Fatal("session cursor did not advance")
			}
			after = page.Next
		}
	}
	t.Fatal("browser has no current session")
	return "", domain.Actor{}, ""
}

// API-036: the no-JS inventory follows bounded pages, and the confirmation
// lookup must still reach an owned session outside the first page.
func TestAccountSessionsPaginateAndConfirmOlderSessions(t *testing.T) {
	w := newWorld(t)
	current, other := w.browser(t), w.browser(t)
	current.login("alice@example.org")
	_, _, currentID := browserSession(t, current)
	w.env.Clock.Advance(time.Millisecond)
	other.login("alice@example.org")
	otherRaw, _, otherID := browserSession(t, other)
	for i := 0; i < 28; i++ {
		w.env.Clock.Advance(time.Millisecond)
		if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", testPW); err != nil {
			t.Fatal(err)
		}
	}
	rows := regexp.MustCompile(`/account/sessions/([0-9a-f]{64})/revoke`)
	nextLink := regexp.MustCompile(`<a href="(/account/sessions\?after=[^"]+)">Next page</a>`)
	first := current.get("/account/sessions")
	if first.Status != http.StatusOK {
		t.Fatalf("first page: %d", first.Status)
	}
	firstRows := rows.FindAllStringSubmatch(first.Body, -1)
	if len(firstRows) != 25 || strings.Contains(first.Body, currentID) || strings.Contains(first.Body, otherID) {
		t.Fatalf("first page did not contain exactly the 25 newer sessions: %d", len(firstRows))
	}
	oldPath := "/account/sessions/" + otherID + "/revoke"
	oldConfirm := current.get(oldPath)
	if oldConfirm.Status != http.StatusOK {
		t.Fatalf("confirmation outside first page: %d", oldConfirm.Status)
	}
	current.mustContain(oldConfirm, "You will stay signed in here", "Keep session")
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); err != nil {
		t.Fatalf("confirmation ended the older session: %v", err)
	}
	boundaryPath := "/account/sessions/" + firstRows[len(firstRows)-1][1] + "/revoke"
	if p := current.post(boundaryPath, current.get(boundaryPath), url.Values{"confirm": {"yes"}}); p.Status != http.StatusOK {
		t.Fatalf("boundary revocation: %d", p.Status)
	}
	seen := map[string]bool{}
	for _, row := range firstRows {
		seen[row[1]] = true
	}
	page := first
	for round := 0; ; round++ {
		link := nextLink.FindStringSubmatch(page.Body)
		if link == nil {
			break
		}
		if round >= 4 {
			t.Fatal("session pagination did not finish")
		}
		page = current.get(html.UnescapeString(link[1]))
		if page.Status != http.StatusOK {
			t.Fatalf("continuation page: %d", page.Status)
		}
		for _, row := range rows.FindAllStringSubmatch(page.Body, -1) {
			if seen[row[1]] {
				t.Fatal("session pagination repeated a row")
			}
			seen[row[1]] = true
		}
	}
	if len(seen) != 30 || !seen[currentID] || !seen[otherID] {
		t.Fatalf("pagination lost older sessions after boundary revocation: %d", len(seen))
	}
	current.mustContain(page, "This session")
	if p := current.post(oldPath, oldConfirm, url.Values{"confirm": {"yes"}}); p.Status != http.StatusOK {
		t.Fatalf("older-session revocation: %d", p.Status)
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("older session survived: %v", err)
	}
	if p := current.get("/account/sessions?after=malformed"); p.Status != http.StatusUnprocessableEntity {
		t.Fatalf("malformed session position: %d", p.Status)
	}
}

func TestAccountSessionsRequireOwnerCSRFAndConfirmation(t *testing.T) {
	w := newWorld(t)
	a, other, foreign := w.browser(t), w.browser(t), w.browser(t)
	a.login("alice@example.org")
	other.login("alice@example.org")
	foreign.login("bob@example.org")
	raw, actor, ownID := browserSession(t, a)
	otherRaw, otherActor, otherID := browserSession(t, other)
	foreignRaw, foreignActor, foreignID := browserSession(t, foreign)
	a.mustContain(a.get("/account/security"), "Manage active sessions", "/account/sessions")
	list := a.get("/account/sessions")
	if list.Status != http.StatusOK {
		t.Fatalf("list: %d", list.Status)
	}
	a.mustContain(list, "Active sessions", "This session", "Other session", "Last active", "Ends by")
	for _, secret := range []string{raw, otherRaw, foreignRaw, actor.SessionID, otherActor.SessionID, foreignActor.SessionID, foreignID} {
		if strings.Contains(list.Body, secret) {
			t.Fatal("HTML exposed session credentials or a foreign selector")
		}
	}
	path := "/account/sessions/" + otherID + "/revoke"
	confirm := a.get(path)
	a.mustContain(confirm, "End this session?", "You will stay signed in here", `for="f-confirm"`, "Keep session")
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); err != nil {
		t.Fatal("GET confirmation revoked a session")
	}
	if p := a.post(path, confirm, nil); p.Status != http.StatusUnprocessableEntity {
		t.Fatalf("missing confirmation: %d", p.Status)
	}
	if p := a.post(path, nil, url.Values{"confirm": {"yes"}, "csrf": {"invalid"}}); p.Status != http.StatusForbidden {
		t.Fatalf("invalid CSRF: %d", p.Status)
	}
	foreignPath := "/account/sessions/" + foreignID + "/revoke"
	if p := a.get(foreignPath); p.Status != http.StatusNotFound {
		t.Fatalf("foreign confirmation: %d", p.Status)
	}
	if p := a.post(foreignPath, confirm, url.Values{"confirm": {"yes"}}); p.Status != http.StatusNotFound {
		t.Fatalf("foreign revocation: %d", p.Status)
	}
	if p := a.post(path, confirm, url.Values{"confirm": {"yes"}}); p.Status != http.StatusOK || p.URL != "/account/sessions" {
		t.Fatalf("revoke other: %d at %s", p.Status, p.URL)
	} else {
		a.mustContain(p, "The selected session has ended", "You are still signed in here")
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("other session survived: %v", err)
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, foreignRaw); err != nil {
		t.Fatalf("foreign session changed: %v", err)
	}
	selfPath := "/account/sessions/" + ownID + "/revoke"
	selfConfirm := a.get(selfPath)
	a.mustContain(selfConfirm, "You will be signed out here")
	if p := a.post(selfPath, selfConfirm, url.Values{"confirm": {"yes"}}); p.Status != http.StatusOK || p.URL != "/login" {
		t.Fatalf("revoke current: %d at %s", p.Status, p.URL)
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, raw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("current session survived: %v", err)
	}
	if p := a.get("/account/sessions"); p.URL != "/login" {
		t.Fatalf("ended browser re-entered session list at %s", p.URL)
	}
}

func TestAccountSessionsOmitExpiredSignIns(t *testing.T) {
	w := newWorldCfg(t, func(c *service.Config) {
		c.SessionIdle = service.DefaultAuthThrottleLock
	}, nil)
	old := w.browser(t)
	old.login("alice@example.org")
	_, _, oldID := browserSession(t, old)
	w.env.Clock.Advance(service.DefaultAuthThrottleLock)
	current := w.browser(t)
	current.login("alice@example.org")
	list := current.get("/account/sessions")
	current.mustContain(list, "This session")
	if strings.Contains(list.Body, oldID) || strings.Contains(list.Body, "Other session") {
		t.Fatal("expired sign-in appears active")
	}
}
