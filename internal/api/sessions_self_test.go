package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
)

type ownSessionView struct {
	ID         string `json:"id"`
	Current    bool   `json:"current"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	ExpiresAt  string `json:"expires_at"`
}

type ownSessionPage struct {
	Data       []ownSessionView `json:"data"`
	NextCursor *string          `json:"next_cursor"`
}

func apiSession(t *testing.T, w *world, login string) (string, domain.Actor) {
	t.Helper()
	raw, _, err := w.env.Svc.Login(w.env.Ctx, login, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	a, err := w.env.Svc.Authenticate(w.env.Ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, a
}

func sessionRequest(w *world, raw, method, path string, csrf bool) *httptest.ResponseRecorder {
	headers := map[string]string{"Cookie": (&http.Cookie{Name: httpx.SessionCookie, Value: raw}).String()}
	if csrf {
		headers["X-CSRF-Token"] = csrfFor(raw)
	}
	return w.do(req{method: method, path: "/api/v1" + path, hdr: headers})
}

func TestOwnSessionsAPIRequiresCookieAndOwner(t *testing.T) {
	w := newWorld(t)
	raw, actor := apiSession(t, w, "alice")
	secondRaw, second := apiSession(t, w, "alice")
	foreignRaw, foreign := apiSession(t, w, "bob")
	const path = "/me/sessions"
	if r := w.get(w.aliceT, path); r.Code != http.StatusForbidden {
		t.Fatalf("API token listed sessions: %d", r.Code)
	}
	if r := w.get("", path); r.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous listing: %d", r.Code)
	}
	r := sessionRequest(w, raw, "GET", path, false)
	if r.Code != http.StatusOK {
		t.Fatalf("list: %d %s", r.Code, r.Body.String())
	}
	p := decode[ownSessionPage](t, r)
	if len(p.Data) != 2 || p.NextCursor != nil {
		t.Fatalf("own page: %+v", p)
	}
	for _, secret := range []string{raw, secondRaw, foreignRaw, actor.SessionID, second.SessionID, foreign.SessionID} {
		if strings.Contains(r.Body.String(), secret) {
			t.Fatal("API exposed a credential or stored credential hash")
		}
	}
	var currentID, otherID string
	for _, row := range p.Data {
		if row.Current {
			currentID = row.ID
		} else {
			otherID = row.ID
		}
		for _, stamp := range []string{row.CreatedAt, row.LastSeenAt, row.ExpiresAt} {
			if _, err := time.Parse(time.RFC3339Nano, stamp); err != nil {
				t.Fatal(err)
			}
		}
	}
	if currentID == "" || otherID == "" {
		t.Fatal("missing current/other markers")
	}
	if r := sessionRequest(w, raw, "DELETE", path+"/"+otherID, false); r.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF: %d", r.Code)
	}
	if r := w.do(req{method: "DELETE", path: "/api/v1" + path + "/" + otherID, token: w.aliceT}); r.Code != http.StatusForbidden {
		t.Fatalf("API token revoked a session: %d", r.Code)
	}
	foreignPage := decode[ownSessionPage](t, sessionRequest(w, foreignRaw, "GET", path, false))
	for _, id := range []string{foreignPage.Data[0].ID, actor.SessionID, "malformed"} {
		if r := sessionRequest(w, raw, "DELETE", path+"/"+id, true); r.Code != http.StatusNotFound {
			t.Fatalf("foreign/invalid selector: %d", r.Code)
		}
	}
	for i := 0; i < 2; i++ {
		if r := sessionRequest(w, raw, "DELETE", path+"/"+otherID, true); r.Code != http.StatusNoContent {
			t.Fatalf("revoke other %d: %d %s", i, r.Code, r.Body.String())
		}
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, secondRaw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("revoked credential survived: %v", err)
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, foreignRaw); err != nil {
		t.Fatalf("foreign session changed: %v", err)
	}
	if r := sessionRequest(w, raw, "DELETE", path+"/"+currentID, true); r.Code != http.StatusNoContent {
		t.Fatalf("revoke current: %d", r.Code)
	}
	if r := sessionRequest(w, raw, "GET", path, false); r.Code != http.StatusUnauthorized {
		t.Fatalf("current credential survived: %d", r.Code)
	}
}

func TestOwnSessionsAPIPaginationSurvivesBoundaryRevocation(t *testing.T) {
	w := newWorld(t)
	raw, _ := apiSession(t, w, "alice")
	for i := 0; i < 4; i++ {
		w.env.Clock.Advance(time.Millisecond)
		apiSession(t, w, "alice")
	}
	foreignRaw, _ := apiSession(t, w, "bob")
	path := "/me/sessions?limit=2"
	first := decode[ownSessionPage](t, sessionRequest(w, raw, "GET", path, false))
	if len(first.Data) != 2 || first.NextCursor == nil {
		t.Fatalf("first page: %+v", first)
	}
	seen := map[string]bool{}
	for _, row := range first.Data {
		seen[row.ID] = true
	}
	boundary := first.Data[1].ID
	if r := sessionRequest(w, raw, "DELETE", "/me/sessions/"+boundary, true); r.Code != http.StatusNoContent {
		t.Fatalf("boundary revoke: %d", r.Code)
	}
	if r := sessionRequest(w, foreignRaw, "GET", path+"&cursor="+*first.NextCursor, false); r.Code != http.StatusBadRequest {
		t.Fatalf("foreign cursor: %d", r.Code)
	}
	for next := first.NextCursor; next != nil; {
		r := sessionRequest(w, raw, "GET", path+"&cursor="+*next, false)
		if r.Code != http.StatusOK {
			t.Fatalf("continuation: %d %s", r.Code, r.Body.String())
		}
		p := decode[ownSessionPage](t, r)
		for _, row := range p.Data {
			if seen[row.ID] {
				t.Fatal("pagination repeated a session")
			}
			seen[row.ID] = true
		}
		next = p.NextCursor
	}
	if len(seen) != 5 {
		t.Fatalf("pagination lost sessions after boundary revocation: %d", len(seen))
	}
}
