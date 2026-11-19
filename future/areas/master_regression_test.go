package areas

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestMasterStrictQueriesBeforeHost(t *testing.T) {
	calls := 0
	web := Web{Auth: func(r *http.Request, _ bool) (Identity, error) { calls++; _ = r.ParseForm(); return owner, nil }}
	for _, path := range []string{"/catalog?listing=a&listing=b", "/catalog?listing=a&unknown=b", "/catalog?listing=%zz", "/catalog?listing=" + strings.Repeat("a", 8193), "/assignment?area=a", "/filter?area=a&area=b", "/filter?cursor=a&cursor=b", "/admin?listing=a"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		web.ServeHTTP(w, r)
		if w.Code != 400 || calls != 0 || r.Form != nil {
			t.Fatalf("late rejection %s: code=%d calls=%d", path, w.Code, calls)
		}
	}
	for _, path := range []string{"/assign?listing=own-a", "/admin?id=a", "/sharing?enabled=true", "/disable-sharing?csrf=token"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", path, strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		web.ServeHTTP(w, r)
		if w.Code != 400 || calls != 0 {
			t.Fatalf("mutation query reached host: %s code=%d calls=%d", path, w.Code, calls)
		}
	}
}

func TestMasterFormCallbacksRequireFreshSession(t *testing.T) {
	for _, path := range []string{"/catalog", "/catalog?listing=own-a", "/admin"} {
		for _, change := range []string{"session-revoke", "session-replace", "member-revoke", "cancel", "unchanged"} {
			t.Run(path+"/"+change, func(t *testing.T) {
				p := &fixturePolicy{}
				s := newService(t, filepath.Join(t.TempDir(), "areas.db"), p)
				createArea(t, s, "north", "Private broad area")
				_, err := s.Assign(context.Background(), owner, "assign", Assign{"own-a", "north", 0})
				if err != nil {
					t.Fatal(err)
				}
				actor := owner
				if path == "/admin" {
					actor = admin
				}
				current := actor
				revoked := false
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				web := Web{Service: s, Auth: func(*http.Request, bool) (Identity, error) {
					if revoked {
						return Identity{}, errors.New("revoked")
					}
					return current, nil
				}, CSRFToken: func(*http.Request) string {
					switch change {
					case "session-revoke":
						revoked = true
					case "session-replace":
						current = Identity{"n", "other"}
					case "member-revoke":
						p.mu.Lock()
						p.revoked = true
						p.mu.Unlock()
					case "cancel":
						cancel()
					}
					return "token"
				}}
				r := httptest.NewRequest("GET", path, nil).WithContext(ctx)
				r.Header.Set("Accept", "text/html")
				w := httptest.NewRecorder()
				web.ServeHTTP(w, r)
				want := 403
				if change == "unchanged" {
					want = 200
				}
				if w.Code != want {
					t.Fatalf("code=%d want=%d body=%s", w.Code, want, w.Body.String())
				}
				if change != "unchanged" && strings.Contains(w.Body.String(), "Private broad area") {
					t.Fatal("private catalog exposed after callback")
				}
			})
		}
	}
}

func TestMasterSingleEmptyCursorAndListingRemainValid(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "areas.db"), &fixturePolicy{})
	createArea(t, s, "a", "North")
	web := Web{Service: s, Auth: func(*http.Request, bool) (Identity, error) { return owner, nil }, CSRFToken: func(*http.Request) string { return "token" }, CursorKey: []byte(strings.Repeat("k", 32))}
	for _, path := range []string{"/filter?area=a&cursor=", "/catalog?listing="} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		web.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("compatibility %s status=%d %s", path, w.Code, w.Body.String())
		}
	}
}
