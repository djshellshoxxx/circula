package api_test

import (
	"net/http/httptest"
	"testing"
)

type listingResp struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (w *world) createListing(token, title string) (listingResp, string) {
	w.t.Helper()
	rec := w.post(token, "/listings", m{"kind": "offer", "title": title, "description": "d"}, w.key())
	wantStatus(w.t, rec, 201)
	return decode[listingResp](w.t, rec), rec.Header().Get("ETag")
}

func (w *world) getWith(token, path string, hdr map[string]string) *httptest.ResponseRecorder {
	return w.do(req{method: "GET", path: "/api/v1" + path, token: token, hdr: hdr})
}

func TestGetListingETagAndNotModified(t *testing.T) {
	w := newWorld(t)
	l, etag := w.createListing(w.aliceT, "Bike repair")
	if etag == "" {
		t.Fatal("create returned no ETag")
	}
	// another member can read an active listing; the ETag matches the create response
	rec := w.get(w.bobT, "/listings/"+l.ID)
	wantStatus(t, rec, 200)
	if rec.Header().Get("ETag") != etag || decode[listingResp](t, rec).Title != "Bike repair" {
		t.Fatalf("get = %s etag %q want %q", rec.Body.String(), rec.Header().Get("ETag"), etag)
	}
	// If-None-Match with the current ETag: 304 with no body, ETag repeated
	for _, inm := range []string{etag, "W/" + etag, `"nope", ` + etag, "*"} {
		rec = w.getWith(w.bobT, "/listings/"+l.ID, map[string]string{"If-None-Match": inm})
		wantStatus(t, rec, 304)
		if rec.Body.Len() != 0 || rec.Header().Get("ETag") != etag {
			t.Fatalf("304 for %q has body %q etag %q", inm, rec.Body.String(), rec.Header().Get("ETag"))
		}
	}
	// a stale ETag gets the full body
	wantStatus(t, w.getWith(w.bobT, "/listings/"+l.ID, map[string]string{"If-None-Match": `"stale"`}), 200)
	// after a change the old ETag no longer matches and the new one does
	rec = w.do(req{method: "PATCH", path: "/api/v1/listings/" + l.ID, token: w.aliceT, body: m{"title": "Bike and trike repair"},
		hdr: map[string]string{"If-Match": etag}})
	wantStatus(t, rec, 200)
	etag2 := rec.Header().Get("ETag")
	if etag2 == etag {
		t.Fatal("ETag did not change")
	}
	wantStatus(t, w.getWith(w.bobT, "/listings/"+l.ID, map[string]string{"If-None-Match": etag}), 200)
	wantStatus(t, w.getWith(w.bobT, "/listings/"+l.ID, map[string]string{"If-None-Match": etag2}), 304)
	// the owner still sees a paused listing; others get 404
	rec = w.do(req{method: "PATCH", path: "/api/v1/listings/" + l.ID, token: w.aliceT, body: m{"status": "paused"},
		hdr: map[string]string{"If-Match": etag2}})
	wantStatus(t, rec, 200)
	wantStatus(t, w.get(w.aliceT, "/listings/"+l.ID), 200)
	wantProblem(t, w.get(w.bobT, "/listings/"+l.ID), 404, "not-found")
	// negatives: unknown id, no token, query noise
	wantProblem(t, w.get(w.bobT, "/listings/does-not-exist"), 404, "not-found")
	wantProblem(t, w.get("", "/listings/"+l.ID), 401, "unauthenticated")
	wantProblem(t, w.get(w.aliceT, "/listings/"+l.ID+"?x=1"), 400, "invalid-query")
}

func TestAnnouncementsAPI(t *testing.T) {
	w := newWorld(t)
	type ann struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		EffectiveAt string `json:"effective_at"`
	}
	type annList struct {
		Data []ann `json:"data"`
	}
	rec := w.post(w.adminT, "/admin/announcements", m{"title": "Market day", "body": "Saturday"}, w.key())
	wantStatus(t, rec, 201)
	a := decode[ann](t, rec)
	if a.ID == "" || a.Title != "Market day" || rec.Header().Get("Location") != "/api/v1/admin/announcements/"+a.ID {
		t.Fatalf("create = %+v loc %s", a, rec.Header().Get("Location"))
	}
	// idempotent replay
	key := w.key()
	r1 := w.post(w.adminT, "/admin/announcements", m{"title": "Twice", "body": "b"}, key)
	r2 := w.post(w.adminT, "/admin/announcements", m{"title": "Twice", "body": "b"}, key)
	if r2.Header().Get("Idempotent-Replayed") != "true" || r1.Body.String() != r2.Body.String() {
		t.Fatal("create not idempotent")
	}
	// scheduled announcement: visible to the administrator only
	rec = w.post(w.adminT, "/admin/announcements", m{"title": "Soon", "body": "b", "effective_at": "2099-01-01T00:00:00Z"}, w.key())
	wantProblem(t, rec, 422, "validation-failed") // more than a year ahead
	// members read; the admin list matches here
	if got := decode[annList](t, w.get(w.aliceT, "/network/announcements")); len(got.Data) != 2 {
		t.Fatalf("member list = %+v", got)
	}
	if got := decode[annList](t, w.get(w.adminT, "/admin/announcements")); len(got.Data) != 2 {
		t.Fatalf("admin list = %+v", got)
	}
	// delete retracts; second delete 404
	wantStatus(t, w.do(req{method: "DELETE", path: "/api/v1/admin/announcements/" + a.ID, token: w.adminT}), 204)
	if got := decode[annList](t, w.get(w.aliceT, "/network/announcements")); len(got.Data) != 1 || got.Data[0].ID == a.ID {
		t.Fatalf("after delete = %+v", got)
	}
	wantProblem(t, w.do(req{method: "DELETE", path: "/api/v1/admin/announcements/" + a.ID, token: w.adminT}), 404, "not-found")
	// negatives
	wantProblem(t, w.post(w.aliceT, "/admin/announcements", m{"title": "x", "body": "y"}, w.key()), 403, "forbidden")
	wantProblem(t, w.get(w.aliceT, "/admin/announcements"), 403, "forbidden")
	wantProblem(t, w.do(req{method: "DELETE", path: "/api/v1/admin/announcements/" + a.ID, token: w.aliceT}), 403, "forbidden")
	wantProblem(t, w.post(w.adminT, "/admin/announcements", m{"title": "", "body": "y"}, w.key()), 422, "validation-failed")
	wantProblem(t, w.post(w.adminT, "/admin/announcements", m{"title": "t", "body": "y"}, ""), 400, "idempotency-key-required")
	wantProblem(t, w.post(w.adminT, "/admin/announcements", m{"title": "t", "body": "y", "effective_at": "yesterday"}, w.key()), 422, "validation-failed")
	wantProblem(t, w.get("", "/network/announcements"), 401, "unauthenticated")
}

func TestMeTimeZone(t *testing.T) {
	w := newWorld(t)
	type me struct {
		TimeZone *string `json:"time_zone"`
	}
	if got := decode[me](t, w.get(w.aliceT, "/me")); got.TimeZone != nil {
		t.Fatalf("default zone = %v", *got.TimeZone)
	}
	patch := func(tok string, body any) *httptest.ResponseRecorder {
		return w.do(req{method: "PATCH", path: "/api/v1/me", token: tok, body: body})
	}
	rec := patch(w.aliceT, m{"time_zone": "Asia/Kolkata"})
	wantStatus(t, rec, 200)
	if got := decode[me](t, w.get(w.aliceT, "/me")); got.TimeZone == nil || *got.TimeZone != "Asia/Kolkata" {
		t.Fatalf("zone = %v", got.TimeZone)
	}
	if got := decode[me](t, w.get(w.bobT, "/me")); got.TimeZone != nil {
		t.Fatal("zone leaked to another person")
	}
	// timestamps stay UTC in the API whatever the preference
	l, _ := w.createListing(w.aliceT, "Zone check")
	full := decode[struct {
		CreatedAt string `json:"created_at"`
	}](t, w.get(w.aliceT, "/listings/"+l.ID))
	if n := len(full.CreatedAt); n == 0 || full.CreatedAt[n-1] != 'Z' {
		t.Fatalf("created_at = %q", full.CreatedAt)
	}
	wantStatus(t, patch(w.aliceT, m{"time_zone": ""}), 200)
	if got := decode[me](t, w.get(w.aliceT, "/me")); got.TimeZone != nil {
		t.Fatal("cleared zone still set")
	}
	wantProblem(t, patch(w.aliceT, m{"time_zone": "Mars/Base"}), 422, "validation-failed")
	wantProblem(t, patch(w.aliceT, m{"time_zone": "Local"}), 422, "validation-failed")
	wantProblem(t, patch(w.aliceT, m{"time_zone": 5}), 422, "validation-failed")
	wantProblem(t, patch(w.aliceT, m{"time_zone": nil}), 422, "validation-failed")
	wantProblem(t, patch(w.aliceT, m{}), 422, "validation-failed")
	wantProblem(t, patch("", m{"time_zone": "UTC"}), 401, "unauthenticated")
}
