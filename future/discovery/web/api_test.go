package web_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/discovery/internal/testkit"
)

func TestAuthTenantAndCSRFGates(t *testing.T) {
	f := newFixture(t, "")
	f.listing("n1", "L1", "alice", "Bike")
	// unauthenticated and inactive
	wantProblem(t, f.get("", "/api/v1/listings"), 401)
	wantProblem(t, f.get("nobody", "/api/v1/listings"), 401)
	wantProblem(t, f.get("ghost", "/api/v1/listings"), 403)
	// X-Network must match the session's network
	w := f.do(req{method: "GET", path: "/api/v1/listings", user: "bob", headers: map[string]string{"X-Network": "n2"}})
	wantProblem(t, w, 403)
	wantStatus(t, f.do(req{method: "GET", path: "/api/v1/listings", user: "bob", headers: map[string]string{"X-Network": "n1"}}), 200)
	// security headers on every response
	w = f.get("bob", "/api/v1/listings")
	for k, want := range map[string]string{"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff"} {
		if w.Header().Get(k) != want {
			t.Errorf("%s = %q", k, w.Header().Get(k))
		}
	}
	// CSRF: missing, wrong, cross-origin, right
	body := map[string]any{"version": 0}
	wantProblem(t, f.jsonReq("POST", "/api/v1/listings/L1/renew", "alice", "", body, nil), 403)
	wantProblem(t, f.jsonReq("POST", "/api/v1/listings/L1/renew", "alice", "csrf-bob", body, nil), 403)
	p := wantProblem(t, f.jsonReq("POST", "/api/v1/listings/L1/renew", "alice", "csrf-alice", body, map[string]string{"Origin": "https://evil.example"}), 403)
	_ = p
	wantStatus(t, f.jsonReq("POST", "/api/v1/listings/L1/renew", "alice", "csrf-alice", body, map[string]string{"Origin": "http://example.com"}), 200)
}

func TestSearchJSONContract(t *testing.T) {
	f := newFixture(t, "")
	goods := f.classes(t)
	f.listing("n1", "L1", "alice", "Red bicycle")
	f.listing("n1", "L2", "bob", "Garden hose")
	f.listing("n2", "X1", "carol", "Red bicycle")
	f.svc.UpdateDiscovery(ctx, actors["alice"], "L1", dInput(goods.ID))
	w := f.get("bob", "/api/v1/listings?q=bicycle&category="+goods.ID)
	wantStatus(t, w, 200)
	var out struct {
		Items []struct {
			ID, Title string
			Mine      bool
			Tags      []struct{ Name string }
			Cover     any
		}
		NextCursor string `json:"next_cursor"`
		SavedView  string `json:"saved_view"`
	}
	decode(t, w, &out)
	if len(out.Items) != 1 || out.Items[0].ID != "L1" || out.Items[0].Mine {
		t.Fatalf("%+v", out)
	}
	if strings.Contains(out.SavedView, "cursor") || !strings.Contains(out.SavedView, "q=bicycle") {
		t.Fatal(out.SavedView)
	}
	// ETag / conditional GET
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no etag")
	}
	w2 := f.do(req{method: "GET", path: "/api/v1/listings?q=bicycle&category=" + goods.ID, user: "bob", headers: map[string]string{"If-None-Match": etag}})
	wantStatus(t, w2, 304)
	// network isolation: n2 member never sees n1 rows
	var out2 struct{ Items []struct{ ID string } }
	decode(t, f.get("carol", "/api/v1/listings"), &out2)
	if len(out2.Items) != 1 || out2.Items[0].ID != "X1" {
		t.Fatalf("%+v", out2)
	}
	// bad inputs
	for _, q := range []string{"bogus=1", "q=a&q=b", "limit=9999", "sort=foo", "token=abc", "cursor=garbage", "scope=all"} {
		w := f.get("bob", "/api/v1/listings?"+q)
		if w.Code < 400 || w.Code >= 500 {
			t.Errorf("%s -> %d", q, w.Code)
		}
		if w.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s: not problem json", q)
		}
	}
	wantProblem(t, f.get("bob", "/api/v1/listings?scope=all"), 403)
	// pagination over the API
	for i := 0; i < 12; i++ {
		f.listing("n1", fmt.Sprintf("P%02d", i), "alice", "Pager item")
	}
	seen := map[string]bool{}
	next := "/api/v1/listings?q=pager&limit=5"
	for pages := 0; next != "" && pages < 10; pages++ {
		var pg struct {
			Items      []struct{ ID string }
			NextCursor string `json:"next_cursor"`
			SavedView  string `json:"saved_view"`
		}
		decode(t, f.get("bob", next), &pg)
		for _, it := range pg.Items {
			if seen[it.ID] {
				t.Fatal("duplicate", it.ID)
			}
			seen[it.ID] = true
		}
		next = ""
		if pg.NextCursor != "" {
			next = "/api/v1/listings?" + pg.SavedView + "&cursor=" + pg.NextCursor
		}
	}
	if len(seen) != 12 {
		t.Fatal(len(seen))
	}
}

func TestListingDetailAndDiscoveryUpdateAPI(t *testing.T) {
	f := newFixture(t, "")
	goods := f.classes(t)
	f.listing("n1", "L1", "alice", "Bike")
	w := f.get("alice", "/api/v1/listings/L1")
	wantStatus(t, w, 200)
	var v struct {
		Version      int64
		Unclassified bool
	}
	decode(t, w, &v)
	if v.Version != 0 || !v.Unclassified {
		t.Fatal(v)
	}
	put := map[string]any{"category_id": goods.ID, "type": "goods", "condition": "good", "availability": "available",
		"expires_at": f.clock.Now().Add(72 * time.Hour).Format(time.RFC3339), "tags": []string{"Red", "fast"}, "version": 0}
	wantStatus(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "alice", "csrf-alice", put, nil), 200)
	// stale version → 409
	p := wantProblem(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "alice", "csrf-alice", put, nil), 409)
	_ = p
	// non-owner → 403; foreign → 404
	put["version"] = 1
	wantProblem(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "bob", "csrf-bob", put, nil), 403)
	wantProblem(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "carol", "csrf-carol", put, nil), 404)
	// validation → 422 with field
	put["condition"] = "pristine"
	pr := wantProblem(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "alice", "csrf-alice", put, nil), 422)
	if pr.Field != "condition" {
		t.Fatal(pr)
	}
	// strict JSON: unknown field, bad types, trailing data, wrong content type, huge body
	wantProblem(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "alice", "csrf-alice", map[string]any{"owner_id": "bob", "version": 1}, nil), 400)
	wantProblem(t, f.jsonReq("PUT", "/api/v1/listings/L1/discovery", "alice", "csrf-alice", map[string]any{"expires_at": "tomorrow", "version": 1}, nil), 422)
	w = f.do(req{method: "PUT", path: "/api/v1/listings/L1/discovery", user: "alice", body: strings.NewReader("{}"), headers: map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "text/plain"}})
	wantProblem(t, w, 415)
	w = f.do(req{method: "PUT", path: "/api/v1/listings/L1/discovery", user: "alice", body: strings.NewReader(`{"version":1} {}`), headers: map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "application/json"}})
	wantProblem(t, w, 400)
	big := `{"tags":["` + strings.Repeat("a", 70<<10) + `"],"version":1}`
	w = f.do(req{method: "PUT", path: "/api/v1/listings/L1/discovery", user: "alice", body: strings.NewReader(big), headers: map[string]string{"X-CSRF-Token": "csrf-alice", "Content-Type": "application/json"}})
	wantProblem(t, w, 413)
	// pending tags hidden from other members, shown with status to the owner
	var own, other struct {
		Tags []struct{ Name, Status string }
	}
	decode(t, f.get("alice", "/api/v1/listings/L1"), &own)
	decode(t, f.get("bob", "/api/v1/listings/L1"), &other)
	if len(own.Tags) != 2 || own.Tags[0].Status != "pending" || len(other.Tags) != 0 {
		t.Fatal(own, other)
	}
	// renew
	w = f.jsonReq("POST", "/api/v1/listings/L1/renew", "alice", "csrf-alice", map[string]any{"version": 1}, nil)
	wantStatus(t, w, 200)
	wantProblem(t, f.jsonReq("POST", "/api/v1/listings/L1/renew", "alice", "csrf-alice", map[string]any{"version": 1}, nil), 409)
}

func TestMediaAPIFlow(t *testing.T) {
	f := newFixture(t, "")
	f.listing("n1", "L1", "alice", "Bike")
	src := testkit.WithGPSEXIF(testkit.JPEG(t, 800, 500, 1), 1)
	// multipart; lying filename/content-type are ignored
	w := f.upload("/api/v1/listings/L1/media", "alice", false, "A red bike", src, true)
	wantStatus(t, w, 201)
	var m struct {
		ID, URL, MIME string
		ThumbURL      string `json:"thumb_url"`
		Width         int
	}
	decode(t, w, &m)
	if m.MIME != "image/jpeg" || m.Width != 500 || !strings.HasPrefix(w.Header().Get("Location"), "/api/v1/listings/L1/media/") {
		t.Fatalf("%+v %v", m, w.Header())
	}
	// served with hardening headers and without GPS
	for _, u := range []string{m.URL, m.ThumbURL} {
		g := f.get("bob", u)
		wantStatus(t, g, 200)
		h := g.Header()
		if h.Get("Content-Type") != "image/jpeg" || h.Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(h.Get("Cache-Control"), "no-store") ||
			h.Get("Content-Disposition") != "inline" || !strings.Contains(h.Get("Content-Security-Policy"), "sandbox") {
			t.Fatalf("headers %v", h)
		}
		if bytes.Contains(g.Body.Bytes(), []byte("GPS")) || bytes.Contains(g.Body.Bytes(), []byte("Exif")) {
			t.Fatal("metadata served")
		}
	}
	// raw image upload; alt via query
	w = f.do(req{method: "POST", path: "/api/v1/listings/L1/media?alt=Second+photo", user: "alice", body: bytes.NewReader(testkit.PNG(t, 40, 40)),
		headers: map[string]string{"Content-Type": "image/png", "X-CSRF-Token": "csrf-alice"}})
	wantStatus(t, w, 201)
	// hostile uploads
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "alice", false, "x", []byte(`<svg onload=alert(1)>`), true), 422)
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "alice", false, "x", append([]byte("PK\x03\x04"), make([]byte, 64)...), true), 422)
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "alice", false, "x", make([]byte, 250<<10), true), 413)
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "alice", false, "", src, true), 422)
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "bob", false, "x", src, true), 403)
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "carol", false, "x", src, true), 404)
	wantProblem(t, f.upload("/api/v1/listings/L1/media", "alice", false, "x", src, false), 403) // no CSRF
	wantProblem(t, f.do(req{method: "POST", path: "/api/v1/listings/L1/media", user: "alice", body: strings.NewReader("x"), headers: map[string]string{"Content-Type": "application/zip", "X-CSRF-Token": "csrf-alice"}}), 415)
	wantProblem(t, f.get("bob", "/api/v1/listings/L1/media/"+m.ID+"?variant=huge"), 422)
	// body above the media exception is rejected (413) before storage
	wantProblem(t, f.do(req{method: "POST", path: "/api/v1/listings/L1/media?alt=x", user: "alice", body: bytes.NewReader(make([]byte, 400<<10)),
		headers: map[string]string{"Content-Type": "image/jpeg", "X-CSRF-Token": "csrf-alice"}}), 413)
	// cross-network and unauthorized access
	wantProblem(t, f.get("carol", m.URL), 404)
	wantProblem(t, f.do(req{method: "DELETE", path: m.URL, user: "bob", headers: map[string]string{"X-CSRF-Token": "csrf-bob"}}), 403)
	wantProblem(t, f.do(req{method: "DELETE", path: m.URL, user: "carol", headers: map[string]string{"X-CSRF-Token": "csrf-carol"}}), 404)
	wantProblem(t, f.do(req{method: "DELETE", path: m.URL, user: "alice"}), 403) // CSRF
	wantStatus(t, f.do(req{method: "DELETE", path: m.URL, user: "alice", headers: map[string]string{"X-CSRF-Token": "csrf-alice"}}), 204)
	wantProblem(t, f.get("bob", m.URL), 404)
}

func TestMediaAPISeventhPhotoConcurrent(t *testing.T) {
	f := newFixture(t, "")
	f.listing("n1", "L1", "alice", "Bike")
	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[int]int{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := f.upload("/api/v1/listings/L1/media", "alice", false, "photo", testkit.JPEG(t, 60, 60, 200+i), true)
			mu.Lock()
			codes[w.Code]++
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	if codes[201] != 6 || codes[409] != 4 {
		t.Fatalf("%v", codes)
	}
}

func TestIdempotencyKeys(t *testing.T) {
	f := newFixture(t, "")
	f.listing("n1", "L1", "alice", "Bike")
	img := testkit.JPEG(t, 60, 60, 1)
	// multipart boundaries are random, so use the raw-image form for byte-identical retries
	raw := func(key string, data []byte) int {
		w := f.do(req{method: "POST", path: "/api/v1/listings/L1/media?alt=a", user: "alice", body: bytes.NewReader(data),
			headers: map[string]string{"Content-Type": "image/jpeg", "X-CSRF-Token": "csrf-alice", "Idempotency-Key": key}})
		return w.Code
	}
	if c := raw("key-1", img); c != 201 {
		t.Fatal(c)
	}
	w := f.do(req{method: "POST", path: "/api/v1/listings/L1/media?alt=a", user: "alice", body: bytes.NewReader(img),
		headers: map[string]string{"Content-Type": "image/jpeg", "X-CSRF-Token": "csrf-alice", "Idempotency-Key": "key-1"}})
	wantStatus(t, w, 201)
	if w.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("not replayed")
	}
	var v struct{ Media []struct{} }
	decode(t, f.get("alice", "/api/v1/listings/L1"), &v)
	if len(v.Media) != 1 {
		t.Fatalf("retry created %d photos", len(v.Media))
	}
	// same key, different payload → 422
	if c := raw("key-1", testkit.JPEG(t, 60, 60, 2)); c != 422 {
		t.Fatal(c)
	}
	// deterministic client errors are replayed, not re-executed
	if c := raw("key-2", []byte("not an image")); c != 422 {
		t.Fatal(c)
	}
	// the same key from another actor/network is independent
	w = f.do(req{method: "POST", path: "/api/v1/listings/L1/media?alt=a", user: "bob", body: bytes.NewReader(img),
		headers: map[string]string{"Content-Type": "image/jpeg", "X-CSRF-Token": "csrf-bob", "Idempotency-Key": "key-1"}})
	wantProblem(t, w, 403)
	// bad keys
	for _, k := range []string{strings.Repeat("k", 129), "bad key", "a/b"} {
		if c := raw(k, img); c != 400 {
			t.Errorf("key %q -> %d", k, c)
		}
	}
	// concurrent duplicates execute once
	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[int]int{}
	img2 := testkit.JPEG(t, 61, 61, 9)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := raw("race-key", img2)
			mu.Lock()
			codes[c]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	decode(t, f.get("alice", "/api/v1/listings/L1"), &v)
	if len(v.Media) != 2 {
		t.Fatalf("concurrent duplicates created %d photos total (codes %v)", len(v.Media), codes)
	}
}

func TestTaxonomyAPI(t *testing.T) {
	f := newFixture(t, "")
	wantProblem(t, f.jsonReq("POST", "/api/v1/categories", "bob", "csrf-bob", map[string]any{"name": "Tools", "kind": "goods"}, nil), 403)
	w := f.jsonReq("POST", "/api/v1/categories", "adm", "csrf-adm", map[string]any{"name": "Tools", "kind": "goods"}, nil)
	wantStatus(t, w, 201)
	var c struct{ ID string }
	decode(t, w, &c)
	wantProblem(t, f.jsonReq("POST", "/api/v1/categories", "adm", "csrf-adm", map[string]any{"name": "tools", "kind": "goods"}, nil), 409)
	wantProblem(t, f.jsonReq("POST", "/api/v1/categories", "adm", "csrf-adm", map[string]any{"name": "<b>x</b>", "kind": "goods"}, nil), 422)
	wantStatus(t, f.jsonReq("PATCH", "/api/v1/categories/"+c.ID, "adm", "csrf-adm", map[string]any{"enabled": false}, nil), 204)
	var list struct{ Items []struct{ ID string } }
	decode(t, f.get("bob", "/api/v1/categories"), &list)
	if len(list.Items) != 0 {
		t.Fatal("disabled category listed to member")
	}
	decode(t, f.get("adm", "/api/v1/categories"), &list)
	if len(list.Items) != 1 {
		t.Fatal(list)
	}
	wantProblem(t, f.jsonReq("PATCH", "/api/v1/categories/"+c.ID, "carol", "csrf-carol", map[string]any{"enabled": true}, nil), 403)
	// areas never reveal centroids to members
	wantStatus(t, f.jsonReq("POST", "/api/v1/areas", "adm", "csrf-adm", map[string]any{"name": "North", "coarse_cell": map[string]any{"lat10": 492, "lon10": -1231}}, nil), 201)
	if w := f.get("bob", "/api/v1/areas"); strings.Contains(w.Body.String(), "492") || strings.Contains(w.Body.String(), "lat10") {
		t.Fatal("centroid shown to member:", w.Body.String())
	}
	if w := f.get("adm", "/api/v1/areas"); !strings.Contains(w.Body.String(), "lat10") {
		t.Fatal("admin should see coarse cell")
	}
	// tag moderation
	f.listing("n1", "L1", "alice", "Bike")
	f.svc.UpdateDiscovery(ctx, actors["alice"], "L1", dInput("", "Fixie"))
	wantProblem(t, f.jsonReq("POST", "/api/v1/tags/fixie/moderation", "bob", "csrf-bob", map[string]any{"status": "approved"}, nil), 403)
	wantStatus(t, f.jsonReq("POST", "/api/v1/tags/fixie/moderation", "adm", "csrf-adm", map[string]any{"status": "approved"}, nil), 204)
	wantProblem(t, f.jsonReq("POST", "/api/v1/tags/fixie/moderation", "adm", "csrf-adm", map[string]any{"status": "bogus"}, nil), 422)
	var tags struct {
		Items []struct{ Name, Status string }
	}
	decode(t, f.get("bob", "/api/v1/tags"), &tags)
	if len(tags.Items) != 1 || tags.Items[0].Status != "approved" {
		t.Fatal(tags)
	}
}
