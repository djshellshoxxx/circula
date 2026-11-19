package recurring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestMasterQueryRejectedBeforeSessionAndMutation(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	saved := create(t, s, a, "create")
	calls := 0
	h := Handler{s, func(r *http.Request, mutation bool) (Actor, string, error) {
		calls++
		_ = r.ParseForm()
		return a, "fixture", nil
	}}
	for _, method := range []string{"GET", "POST"} {
		queries := []string{"limit=bad", "limit=0", "limit=101", "limit=1&limit=2", "after=&after=", "unknown=1", "after=%zz", "after=%00", "after=" + strings.Repeat("a", 8192)}
		if method == "POST" {
			queries = append(queries, "limit=20", "after=")
		}
		for _, query := range queries {
			body := "action=pause&id=" + saved.ID + "&expected=1&key=pause&csrf=fixture"
			r := httptest.NewRequest(method, "/reminders?"+query, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 || calls != 0 || r.Form != nil {
				t.Fatalf("%s query reached session/mutation: status=%d calls=%d", method, w.Code, calls)
			}
		}
	}
	current, err := s.Get(context.Background(), a, saved.ID)
	if err != nil || !current.Active || current.Version != saved.Version {
		t.Fatalf("invalid POST changed state: %+v %v", current, err)
	}
}

func TestMasterEmptyFirstPageCursorAndNativePost(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	saved := create(t, s, a, "create")
	h := Handler{s, func(*http.Request, bool) (Actor, string, error) { return a, "fixture", nil }}
	for _, query := range []string{"", "?after=", "?after=&limit=1", "?limit=100"} {
		r := httptest.NewRequest("GET", "/reminders"+query, nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "action=\"/reminders\"") {
			t.Fatalf("valid first page failed: %d %s", w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/reminders", strings.NewReader("action=pause&id="+saved.ID+"&expected=1&key=pause&csrf=fixture"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	current, err := s.Get(context.Background(), a, saved.ID)
	if w.Code != 200 || err != nil || current.Active || current.Version != saved.Version+1 {
		t.Fatalf("native POST failed: %d %+v %v", w.Code, current, err)
	}
}
