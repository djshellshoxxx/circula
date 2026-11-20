package federation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestMasterQueriesAndFormsBeforeHost(t *testing.T) {
	for _, admin := range []bool{false, true} {
		calls := 0
		auth := func(r *http.Request, _ bool) (Actor, error) {
			calls++
			_ = r.ParseForm()
			return Actor{"local", "admin"}, nil
		}
		var handler http.Handler = Members{Auth: auth}
		if admin {
			handler = Admin{Auth: auth}
		}
		for _, method := range []string{"GET", "POST"} {
			for _, query := range []string{"limit=1&limit=2", "bad=%zz", "action=enable", "after_peer=" + strings.Repeat("a", 8192), "limit=0", "after_listing=%00"} {
				r := httptest.NewRequest(method, "/?"+query, strings.NewReader("action=enable"))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				if w.Code != 400 || calls != 0 || r.Form != nil {
					t.Fatalf("admin=%t %s parsed invalid query: code=%d calls=%d", admin, method, w.Code, calls)
				}
			}
		}
		r := httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("a", 65537)))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 413 || calls != 0 {
			t.Fatalf("admin=%t excessive body reached host: %d %d", admin, w.Code, calls)
		}
		for _, body := range []string{"action=enable&action=disable", "action=enable&unknown=true", "action=%zz", "action=enable&discarded=%zz"} {
			r := httptest.NewRequest("POST", "/", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 400 || calls != 0 {
				t.Fatalf("admin=%t ambiguous body reached host: %d %d", admin, w.Code, calls)
			}
		}
	}
}

func TestMasterCallbackReauthorizesBeforePrivateRead(t *testing.T) {
	for _, admin := range []bool{false, true} {
		for _, change := range []string{"revoke-session", "replace-session", "revoke-member", "cancel", "unchanged"} {
			t.Run(map[bool]string{true: "admin", false: "member"}[admin]+"/"+change, func(t *testing.T) {
				s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
				pub, key, _ := ed25519.GenerateKey(rand.Reader)
				fixturePeer(t, s, "local", "remote", "key", pub)
				event := fixtureEvent("event", "listing.publish", 1)
				event.Content.Title = "Private remote item"
				wire, err := SignDiscovery(event, "key", key)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = s.Receive(context.Background(), "local", "remote", wire); err != nil {
					t.Fatal(err)
				}
				actor := Actor{"local", "owner"}
				if admin {
					actor.Member = "admin"
				}
				current := actor
				revoked := false
				memberRevoked := false
				original := s.Authorize
				s.Authorize = func(ctx context.Context, c *sql.Conn, a Actor, adm bool) error {
					if memberRevoked {
						return ErrInvalid
					}
					return original(ctx, c, a, adm)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				auth := func(*http.Request, bool) (Actor, error) {
					if revoked {
						return Actor{}, ErrInvalid
					}
					return current, nil
				}
				token := func(*http.Request) string {
					switch change {
					case "revoke-session":
						revoked = true
					case "replace-session":
						current.Member = "replacement"
					case "revoke-member":
						memberRevoked = true
					case "cancel":
						cancel()
					}
					return "token"
				}
				var handler http.Handler = Members{Store: s, Auth: auth, CSRFToken: token}
				if admin {
					handler = Admin{Store: s, Auth: auth, CSRFToken: token}
				}
				r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				want := 403
				if change == "unchanged" {
					want = 200
				}
				if w.Code != want {
					t.Fatalf("status=%d want=%d body=%s", w.Code, want, w.Body.String())
				}
				if want != 200 && (strings.Contains(w.Body.String(), "Private remote item") || strings.Contains(w.Body.String(), "https://example.com")) {
					t.Fatal("private state exposed after token callback")
				}
			})
		}
	}
}

func TestMasterPaginatedNativeMemberSubmission(t *testing.T) {
	s := fixtureStore(t, filepath.Join(t.TempDir(), "db"))
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	fixturePeer(t, s, "local", "remote", "key", pub)
	h := Members{Store: s, Auth: func(r *http.Request, mutation bool) (Actor, error) {
		if mutation && r.Form.Get("csrf") != "token" {
			return Actor{}, ErrInvalid
		}
		return Actor{"local", "owner"}, nil
	}}
	for _, query := range []string{"?after_peer=&after_listing=", "?after_peer=remote&after_listing=listing&limit=1"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/"+query, nil))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		r := httptest.NewRequest("POST", "/"+query, strings.NewReader("csrf=token&action=consent&remote=remote&listing=listing"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
