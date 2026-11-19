package areas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestMalformedFormRejectedBeforeHostParseCache(t *testing.T) {
	for _, suffix := range []string{"&ignored=%zz", "&area=%zz", "&ignored=x;y"} {
		t.Run(suffix, func(t *testing.T) {
			s := newService(t, filepath.Join(t.TempDir(), "areas.db"), &fixturePolicy{})
			createArea(t, s, "north", "North")
			calls := 0
			web := Web{Service: s, Auth: func(r *http.Request, _ bool) (Identity, error) {
				calls++
				// CSRF integrations commonly use PostFormValue, which discards ParseForm errors.
				_ = r.PostFormValue("csrf")
				return owner, nil
			}}
			body := "csrf=token&listing=own-a&area=north&revision=0&key=cache-test" + suffix
			r := httptest.NewRequest("POST", "/assign", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			out := httptest.NewRecorder()
			web.ServeHTTP(out, r)
			current, err := s.Current(context.Background(), owner, "own-a")
			if out.Code != 400 || calls != 0 || err != nil || current.Revision != 0 || current.Active {
				t.Fatalf("invalid form reached host/mutation: status=%d auth=%d current=%+v err=%v", out.Code, calls, current, err)
			}
		})
	}
}

func TestValidFormHostParseCacheRemainsCompatible(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "areas.db"), &fixturePolicy{})
	createArea(t, s, "north", "North")
	calls := 0
	web := Web{Service: s, Auth: func(r *http.Request, _ bool) (Identity, error) {
		calls++
		if got := r.PostFormValue("csrf"); got != "token" {
			t.Fatalf("host token=%q", got)
		}
		return owner, nil
	}}
	r := httptest.NewRequest("POST", "/assign", strings.NewReader("csrf=token&listing=own-a&area=north&revision=0&key=valid-cache"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	out := httptest.NewRecorder()
	web.ServeHTTP(out, r)
	current, err := s.Current(context.Background(), owner, "own-a")
	if out.Code != 200 || calls != 1 || err != nil || current.Revision != 1 || !current.Active || current.AreaID != "north" {
		t.Fatalf("valid form failed: status=%d auth=%d current=%+v err=%v", out.Code, calls, current, err)
	}
}
