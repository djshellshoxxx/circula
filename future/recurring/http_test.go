package recurring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPNoJSAndBoundary(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	r0 := create(t, s, a, "create")
	session := func(r *http.Request, mutate bool) (Actor, string, error) {
		if r.Header.Get("Authorization") != "fixture" {
			return Actor{}, "", ErrDenied
		}
		if mutate && r.Header.Get("X-CSRF") != "fixture" {
			return Actor{}, "", ErrDenied
		}
		return a, "fixture", nil
	}
	h := Handler{s, session}
	r := httptest.NewRequest("GET", "/reminders", nil)
	r.Header.Set("Accept", "text/html")
	r.Header.Set("Authorization", "fixture")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "<form") || !strings.Contains(w.Body.String(), "Skip next occurrence") || strings.Contains(w.Body.String(), "<script>") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cache")
	}
	for _, test := range []struct {
		method, body, auth, csrf, network, content string
		status                                     int
	}{{"GET", "", "", "", "", "", 403}, {"GET", "", "fixture", "", "other", "", 403}, {"POST", `{"Action":"pause","ID":"` + r0.ID + `","Expected":1}`, "fixture", "", "", "application/json", 403}, {"POST", `{"Action":"pause","ID":"` + r0.ID + `","Expected":1,"unknown":1}`, "fixture", "fixture", "", "application/json", 400}, {"POST", `{"Action":"pause","ID":"` + r0.ID + `","Expected":1}`, "fixture", "fixture", "", "application/json", 200}, {"DELETE", "", "fixture", "", "", "", 405}} {
		r = httptest.NewRequest(test.method, "/", strings.NewReader(test.body))
		r.Header.Set("Authorization", test.auth)
		r.Header.Set("X-CSRF", test.csrf)
		r.Header.Set("X-Network", test.network)
		r.Header.Set("Idempotency-Key", "pause")
		r.Header.Set("Content-Type", test.content)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Fatal(test, w.Code, w.Body.String())
		}
	}
	form := url.Values{"action": {"save"}, "key": {"form-create"}, "target": {"b"}, "note": {"<img src=x onerror=alert(1)>"}, "frequency": {"daily"}, "zone": {"UTC"}, "hour": {"10"}, "minute": {"0"}, "weekday": {"0"}, "monthday": {"0"}, "active": {"on"}, "csrf": {"fixture"}}
	r = httptest.NewRequest("POST", "/reminders", strings.NewReader(form.Encode()))
	r.Header.Set("Authorization", "fixture")
	r.Header.Set("X-CSRF", "fixture")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "<img") {
		t.Fatal(w.Code, w.Body.String())
	}
	r = httptest.NewRequest("POST", "/", strings.NewReader(`{"Action":"save","Record":{"Note":"`+strings.Repeat("x", 65537)+`"}}`))
	r.Header.Set("Authorization", "fixture")
	r.Header.Set("X-CSRF", "fixture")
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatal(w.Code)
	}
}
func TestCLIAndFreshDraftDenial(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	r := record()
	payload, _ := json.Marshal(r)
	var output bytes.Buffer
	if err := RunCLI(ctx, s, a, []string{"save", "cli-create", "0"}, bytes.NewReader(payload), &output); err != nil {
		t.Fatal(err)
	}
	var saved Record
	if err := json.Unmarshal(output.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := RunCLI(ctx, s, a, []string{"draft", saved.ID, "cli-draft"}, nil, &output); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := RunCLI(ctx, s, a, []string{"preview", saved.ID}, nil, &output); err != nil || !strings.Contains(output.String(), "no automatic trade") {
		t.Fatal(output.String(), err)
	}
	if _, err := s.DB.Exec(`UPDATE fixture_members SET active=0 WHERE network='n' AND member='b'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDraft(ctx, a, saved.ID, "cli-draft"); !errors.Is(err, ErrDenied) {
		t.Fatal("replay bypassed revoked target", err)
	}
}

func TestStrictJSONAndInactiveTargetCancellation(t *testing.T) {
	for _, raw := range []string{`{"Action":"save","Action":"draft"}`, `{"Record":{"Target":"b","Target":"c"}}`, `{"unknown":1}`, `{"Action":"save"} {}`} {
		var command command
		if err := decodeJSON([]byte(raw), &command); err == nil {
			t.Fatal("accepted ambiguous JSON", raw)
		}
	}
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	r := create(t, s, a, "create")
	if _, err := s.DB.Exec(`UPDATE fixture_members SET active=0 WHERE network='n' AND member='b'`); err != nil {
		t.Fatal(err)
	}
	cancelled, err := s.Change(context.Background(), a, r.ID, "cancel", r.Version, "cancel")
	if err != nil || !cancelled.Cancelled {
		t.Fatal(cancelled, err)
	}
}

func TestAuditFormsRejectExternalActions(t *testing.T) {
	for _, path := range []string{"//example.invalid/private", "/\\example.invalid/path", "https://example.invalid", "/path?redirect=bad"} {
		var b bytes.Buffer
		if err := Render(&b, Page{Path: path, CSRF: "private-token"}); err == nil || b.Len() != 0 {
			t.Fatal("external form rendered", path, err, b.String())
		}
	}
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	h := Handler{s, func(*http.Request, bool) (Actor, string, error) { return a, "private-token", nil }}
	r := httptest.NewRequest("GET", "http://site.invalid//example.invalid/private", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || strings.Contains(w.Body.String(), "private-token") {
		t.Fatal(w.Code, w.Body.String())
	}
}
