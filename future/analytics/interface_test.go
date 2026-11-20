package analytics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReadOnlyInterface(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	source := fixtureSource{snapshotFixture(now), true}
	s := DashboardService{Source: source, Now: func() time.Time { return now }, Zone: time.UTC, K: 5}
	h := Handler{s, func(r *http.Request) (Actor, error) {
		if r.Header.Get("Authorization") != "fixture" {
			return Actor{}, errors.New("denied")
		}
		return Actor{"n", "a"}, nil
	}}
	for _, c := range []struct {
		method, path, auth, network string
		status                      int
	}{{"GET", "/?format=json", "fixture", "", 200}, {"GET", "/?view=own", "fixture", "", 200}, {"GET", "/?format=csv", "fixture", "", 200}, {"GET", "/?format=html", "fixture", "", 200}, {"GET", "/?from=2026-01-01", "fixture", "", 400}, {"GET", "/?period=week", "fixture", "", 403}, {"GET", "/", "fixture", "other", 403}, {"POST", "/", "fixture", "", 405}, {"GET", "/", "", "", 403}} {
		r := httptest.NewRequest(c.method, c.path, nil)
		r.Header.Set("Authorization", c.auth)
		r.Header.Set("X-Network", c.network)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.status || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%+v => %d %s", c, w.Code, w.Body.String())
		}
	}
	var b strings.Builder
	if e := RunCLI(context.Background(), s, Actor{"n", "a"}, []string{"admin", "month", "1"}, &b); e != nil || !strings.Contains(b.String(), "50,0,50") {
		t.Fatal(b.String(), e)
	}
	if _, e := s.Admin(context.Background(), Actor{"n", "a"}, "week", 1); e == nil {
		t.Fatal("overlapping cohort granularity")
	}
}

func TestQueryLimitPrecedesHostCallbacks(t *testing.T) {
	called := 0
	h := Handler{Auth: func(r *http.Request) (Actor, error) {
		called++
		// A host may legitimately parse a small query for its own policy.
		_, _ = r.URL.Query()["count"]
		return Actor{"n", "a"}, nil
	}}
	for _, query := range []string{strings.Repeat("x", 2049), strings.Repeat("count=1&", 512), "bad=%zz&" + strings.Repeat("x", 2048)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/?"+query, nil))
		if w.Code != 400 || called != 0 {
			t.Fatalf("oversized query reached host: status=%d calls=%d", w.Code, called)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/?"+strings.Repeat("x", 2048), nil))
	if w.Code != 400 || called != 1 {
		t.Fatalf("exact boundary bypassed normal validation: status=%d calls=%d", w.Code, called)
	}
}
