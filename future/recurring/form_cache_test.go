package recurring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormParseFailureCannotBeHiddenBySession(t *testing.T) {
	for _, tc := range []struct {
		name, suffix string
		want         int
	}{
		{"malformed discarded field", "&ignored=%zz", http.StatusBadRequest},
		{"oversized trailing field", "&ignored=" + strings.Repeat("x", 65536), http.StatusRequestEntityTooLarge},
		{"duplicate action", "&action=pause", http.StatusBadRequest},
		{"unknown field", "&ignored=value", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
			saved := create(t, s, a, "create")
			calls := 0
			h := Handler{s, func(r *http.Request, mutation bool) (Actor, string, error) {
				calls++
				// PostFormValue discards the first ParseForm error and caches partial fields.
				_ = r.PostFormValue("csrf")
				return a, "fixture", nil
			}}
			body := "action=pause&id=" + saved.ID + "&expected=1&key=pause&csrf=fixture" + tc.suffix
			r := httptest.NewRequest("POST", "/reminders", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			current, err := s.Get(context.Background(), a, saved.ID)
			if w.Code != tc.want || calls != 0 || err != nil || !current.Active || current.Version != saved.Version {
				t.Fatalf("invalid form reached session/mutation: status=%d want=%d session_calls=%d active=%t version=%d err=%v", w.Code, tc.want, calls, current.Active, current.Version, err)
			}
		})
	}
}

func TestSessionCanReadPrevalidatedForm(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	saved := create(t, s, a, "create")
	calls := 0
	h := Handler{s, func(r *http.Request, mutation bool) (Actor, string, error) {
		calls++
		if !mutation || r.PostFormValue("csrf") != "fixture" {
			t.Fatal("session lost valid CSRF form")
		}
		return a, "fixture", nil
	}}
	r := httptest.NewRequest("POST", "/reminders", strings.NewReader("action=pause&id="+saved.ID+"&expected=1&key=pause&csrf=fixture"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	current, err := s.Get(context.Background(), a, saved.ID)
	if w.Code != 200 || calls != 1 || err != nil || current.Active || current.Version != saved.Version+1 {
		t.Fatalf("valid form failed: status=%d calls=%d state=%+v err=%v", w.Code, calls, current, err)
	}
}
