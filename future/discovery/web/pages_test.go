package web_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/internal/testkit"
	"github.com/djshellshoxxx/circula/future/discovery/web"
)

var (
	reImg     = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	reAlt     = regexp.MustCompile(`(?i)\balt="`)
	reControl = regexp.MustCompile(`(?is)<(input|select|textarea)\b([^>]*)>`)
	reID      = regexp.MustCompile(`\bid="([^"]+)"`)
	reLabel   = regexp.MustCompile(`<label[^>]*\bfor="([^"]+)"`)
	reOnAttr  = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
)

// a11yChecks applies static, automatable structure checks (not a WCAG audit).
func a11yChecks(t *testing.T, name, html string) {
	t.Helper()
	for _, must := range []string{`<html lang="en">`, `name="viewport"`, `<title>`, `class="skip" href="#main"`, `<main id="main"`, `<h1>`} {
		if !strings.Contains(html, must) {
			t.Errorf("%s: missing %q", name, must)
		}
	}
	if strings.Count(html, "<h1>") != 1 {
		t.Errorf("%s: want exactly one h1", name)
	}
	for _, img := range reImg.FindAllString(html, -1) {
		if !reAlt.MatchString(img) {
			t.Errorf("%s: image without alt: %s", name, img)
		}
	}
	labels := map[string]bool{}
	for _, m := range reLabel.FindAllStringSubmatch(html, -1) {
		labels[m[1]] = true
	}
	for _, m := range reControl.FindAllStringSubmatch(html, -1) {
		attrs := m[2]
		if strings.Contains(attrs, `type="hidden"`) || strings.Contains(attrs, `type="radio"`) {
			continue
		}
		id := reID.FindStringSubmatch(attrs)
		if id == nil || !labels[id[1]] {
			t.Errorf("%s: control without label: <%s %s>", name, m[1], attrs)
		}
	}
	if strings.Contains(strings.ToLower(html), "<script") || strings.Contains(strings.ToLower(html), "javascript:") || reOnAttr.MatchString(html) || strings.Contains(html, "style=\"") && false {
		t.Errorf("%s: contains script content", name)
	}
	for _, m := range regexp.MustCompile(`<form\b[^>]*method="post"[^>]*>(.*?)</form>`).FindAllStringSubmatch(strings.ReplaceAll(html, "\n", " "), -1) {
		if !strings.Contains(m[1], `name="csrf"`) {
			t.Errorf("%s: POST form without csrf field", name)
		}
	}
}

func checkHTMLHeaders(t *testing.T, name string, h map[string][]string) {
	t.Helper()
	get := func(k string) string { return strings.Join(h[k], ",") }
	if ct := get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("%s: content-type %q", name, ct)
	}
	csp := get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'none'") || strings.Contains(csp, "unsafe") || strings.Contains(csp, "script-src") {
		t.Errorf("%s: csp %q", name, csp)
	}
	if get("Cache-Control") != "no-store" || get("X-Content-Type-Options") != "nosniff" || get("X-Frame-Options") != "DENY" {
		t.Errorf("%s: headers %v", name, h)
	}
}

func TestSearchPageNoJSAccessible(t *testing.T) {
	f := newFixture(t, "/market")
	goods := f.classes(t)
	f.listing("n1", "L1", "alice", `<script>alert("xss")</script> Bike`)
	f.svc.UpdateDiscovery(ctx, actors["alice"], "L1", dInput(goods.ID, "red"))
	w := f.get("bob", "/listings")
	wantStatus(t, w, 200)
	html := w.Body.String()
	checkHTMLHeaders(t, "search", w.Header())
	a11yChecks(t, "search", html)
	if strings.Contains(html, "<script>alert") || !strings.Contains(html, "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt; Bike") {
		t.Fatal("title not escaped:", html)
	}
	for _, must := range []string{`role="search"`, `<nav aria-label="Pagination"`, `href="/market/listings/L1"`, `href="/market/static/discovery.css"`, `aria-live="polite"`} {
		if !strings.Contains(html, must) {
			t.Errorf("missing %s", must)
		}
	}
	// empty state is explanatory, not blank
	w = f.get("bob", "/listings?q=zzzzzz")
	if !strings.Contains(w.Body.String(), "Nothing matches these filters yet") || !strings.Contains(w.Body.String(), "No listings found.") {
		t.Fatal("empty state missing")
	}
	a11yChecks(t, "search-empty", w.Body.String())
	// invalid filters: friendly 400, no stack/SQL
	w = f.get("bob", "/listings?limit=zzz&token=abc")
	wantStatus(t, w, 400)
	if !strings.Contains(w.Body.String(), `role="alert"`) || strings.Contains(strings.ToLower(w.Body.String()), "sql") {
		t.Fatal(w.Body.String())
	}
	// unauthenticated and cross-tenant
	wantStatus(t, f.get("", "/listings"), 401)
	wantStatus(t, f.get("carol", "/listings/L1"), 404)
	// static CSS
	css := f.get("", "/static/discovery.css")
	wantStatus(t, css, 200)
	if css.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(css.Body.String(), ":focus-visible") {
		t.Fatal("css headers/content")
	}
}

func TestHTMLPaginationFollowsLinks(t *testing.T) {
	f := newFixture(t, "")
	for i := 0; i < 23; i++ {
		f.listing("n1", fmt.Sprintf("P%02d", i), "alice", fmt.Sprintf("Pager %02d", i))
	}
	seen := map[string]bool{}
	next := "/listings?limit=10&q=pager"
	reNext := regexp.MustCompile(`<a class="btn" href="([^"]+)" rel="next">`)
	reItem := regexp.MustCompile(`href="/listings/(P\d\d)"`)
	pages := 0
	for next != "" {
		w := f.get("bob", next)
		wantStatus(t, w, 200)
		a11yChecks(t, "page", w.Body.String())
		for _, m := range reItem.FindAllStringSubmatch(w.Body.String(), -1) {
			if seen[m[1]] {
				t.Fatal("duplicate", m[1])
			}
			seen[m[1]] = true
		}
		nx := reNext.FindStringSubmatch(w.Body.String())
		next = ""
		if nx != nil {
			next = strings.ReplaceAll(nx[1], "&amp;", "&")
			if pages > 0 && !strings.Contains(w.Body.String(), "Back to first page") {
				t.Fatal("no way back to first page")
			}
		}
		pages++
	}
	if len(seen) != 23 || pages != 3 {
		t.Fatalf("%d items in %d pages", len(seen), pages)
	}
	// expired / tampered cursor link → friendly page, not an error dump
	w := f.get("bob", "/listings?limit=10&q=pager&cursor=abc.def")
	wantStatus(t, w, 400)
	if !strings.Contains(w.Body.String(), "expired") {
		t.Fatal(w.Body.String())
	}
}

func TestDetailGalleryAndEditFlowWithoutJS(t *testing.T) {
	f := newFixture(t, "")
	goods := f.classes(t)
	area, _ := f.svc.CreateArea(ctx, actors["adm"], "North side", nil)
	f.listing("n1", "L1", "alice", "Bike")
	// 1. edit page
	w := f.get("alice", "/listings/L1/edit")
	wantStatus(t, w, 200)
	a11yChecks(t, "edit", w.Body.String())
	checkHTMLHeaders(t, "edit", w.Header())
	wantStatus(t, f.get("bob", "/listings/L1/edit"), 403)
	wantStatus(t, f.get("carol", "/listings/L1/edit"), 404)
	// 2. CSRF protected save
	form := url.Values{"category": {goods.ID}, "type": {"goods"}, "condition": {"good"}, "area": {area.ID}, "availability": {"limited"},
		"tags": {"red, Road bike"}, "expires_at": {"2026-12-20"}, "version": {"0"}}
	bad := f.form("/listings/L1/edit", "alice", cloneWith(form, "csrf", "nope"))
	wantStatus(t, bad, 403)
	w = f.form("/listings/L1/edit", "alice", form)
	wantStatus(t, w, 303)
	if loc := w.Header().Get("Location"); loc != "/listings/L1/edit?notice=saved" {
		t.Fatal(loc)
	}
	w = f.get("alice", "/listings/L1/edit?notice=saved")
	if !strings.Contains(w.Body.String(), "Details saved.") || !strings.Contains(w.Body.String(), `role="status"`) {
		t.Fatal("no confirmation")
	}
	// arbitrary notice text is not reflected
	if w := f.get("alice", "/listings/L1/edit?notice=<b>evil</b>"); strings.Contains(w.Body.String(), "evil") {
		t.Fatal("notice reflected")
	}
	// 3. validation errors re-render the form with summary + field association
	form.Set("version", "1")
	form.Set("condition", "pristine")
	w = f.form("/listings/L1/edit", "alice", form)
	wantStatus(t, w, 422)
	html := w.Body.String()
	for _, must := range []string{`role="alert"`, `aria-invalid="true"`, `href="#f-condition"`, `id="f-condition"`, `value="red, Road bike"`} {
		if !strings.Contains(html, must) {
			t.Errorf("error render missing %s", must)
		}
	}
	a11yChecks(t, "edit-error", html)
	// stale version → 409 with guidance
	form.Set("condition", "good")
	form.Set("version", "0")
	w = f.form("/listings/L1/edit", "alice", form)
	wantStatus(t, w, 409)
	if !strings.Contains(w.Body.String(), "changed since you opened the form") {
		t.Fatal("conflict message")
	}
	// bad date
	form.Set("version", "1")
	form.Set("expires_at", "next week")
	wantStatus(t, f.form("/listings/L1/edit", "alice", form), 422)
	// 4. photo upload with multipart form (no JS)
	src := testkit.WithGPSEXIF(testkit.JPEG(t, 700, 400, 3), 1)
	w = f.upload("/listings/L1/photos", "alice", true, "Blue bike leaning on a fence", src, false)
	wantStatus(t, w, 303)
	wantStatus(t, f.upload("/listings/L1/photos", "alice", false, "x", src, false), 403) // csrf
	w = f.upload("/listings/L1/photos", "alice", true, "evil", []byte(`<svg onload=alert(1)>`), false)
	wantStatus(t, w, 422)
	if !strings.Contains(w.Body.String(), "not an acceptable JPEG or PNG image") || !strings.Contains(w.Body.String(), `role="alert"`) {
		t.Fatal("upload rejection message:", w.Body.String())
	}
	// 5. detail gallery: alt text, figure/figcaption, full-size link, keyboard reachable (plain links)
	w = f.get("bob", "/listings/L1")
	wantStatus(t, w, 200)
	html = w.Body.String()
	a11yChecks(t, "detail", html)
	for _, must := range []string{`alt="Blue bike leaning on a fence"`, `<figure>`, `<figcaption>Photo 1 of 1.`, `loading="lazy"`, `href="/api/v1/listings/L1/media/`, `North side`, `Goods`} {
		if !strings.Contains(html, must) {
			t.Errorf("detail missing %s", must)
		}
	}
	if strings.Contains(html, ">road-bike<") || strings.Contains(html, ">red<") {
		t.Fatal("pending tags visible to other member")
	}
	// search result shows the cover with alt text
	w = f.get("bob", "/listings")
	if !strings.Contains(w.Body.String(), `alt="Blue bike leaning on a fence"`) {
		t.Fatal("no cover in result card")
	}
	// 6. remove via form (owner) / forbidden (other member) / renew
	mid := regexp.MustCompile(`/media/(med_[0-9a-f]+)`).FindStringSubmatch(html)[1]
	wantStatus(t, f.form("/listings/L1/photos/"+mid+"/remove", "bob", url.Values{}), 403)
	w = f.get("alice", "/listings/L1/edit")
	if !strings.Contains(w.Body.String(), "Remove photo 1") {
		t.Fatal("no remove button")
	}
	wantStatus(t, f.form("/listings/L1/photos/"+mid+"/remove", "alice", url.Values{}), 303)
	wantStatus(t, f.get("bob", "/api/v1/listings/L1/media/"+mid), 404)
	v, _ := f.svc.GetListing(ctx, actors["alice"], "L1")
	w = f.form("/listings/L1/renew", "alice", url.Values{"version": {fmt.Sprint(v.Version)}})
	wantStatus(t, w, 303)
	wantStatus(t, f.form("/listings/L1/renew", "alice", url.Values{"version": {fmt.Sprint(v.Version)}}), 409)
}

func cloneWith(v url.Values, k, val string) url.Values {
	c := url.Values{}
	for kk, vv := range v {
		c[kk] = append([]string(nil), vv...)
	}
	c.Set(k, val)
	return c
}

func TestAdminPages(t *testing.T) {
	f := newFixture(t, "")
	wantStatus(t, f.get("bob", "/admin/taxonomy"), 403)
	w := f.get("adm", "/admin/taxonomy")
	wantStatus(t, w, 200)
	a11yChecks(t, "admin", w.Body.String())
	for _, u := range []string{"/admin/categories", "/admin/areas", "/admin/tags/x"} {
		wantStatus(t, f.form(u, "bob", url.Values{"name": {"Tools"}}), 403)
	}
	wantStatus(t, f.form("/admin/categories", "adm", url.Values{"name": {"Tools"}, "kind": {"goods"}}), 303)
	w = f.form("/admin/categories", "adm", url.Values{"name": {"tools"}, "kind": {"goods"}})
	wantStatus(t, w, 409)
	if !strings.Contains(w.Body.String(), "already exists") || !strings.Contains(w.Body.String(), `role="alert"`) {
		t.Fatal(w.Body.String())
	}
	wantStatus(t, f.form("/admin/categories", "adm", url.Values{"seed": {"1"}}), 303)
	cats, _ := f.svc.ListCategories(ctx, actors["adm"])
	if len(cats) != len(d.DefaultCategories)+1 {
		t.Fatal(len(cats))
	}
	var toolsID string
	for _, c := range cats {
		if c.Name == "Tools" {
			toolsID = c.ID
		}
	}
	wantStatus(t, f.form("/admin/categories/"+toolsID, "adm", url.Values{"op": {"rename"}, "name": {"Equipment"}}), 303)
	wantStatus(t, f.form("/admin/categories/"+toolsID, "adm", url.Values{"op": {"disable"}}), 303)
	wantStatus(t, f.form("/admin/categories/"+toolsID, "adm", url.Values{"op": {"explode"}}), 400)
	// area with coordinates rounded to ~11 km cells
	wantStatus(t, f.form("/admin/areas", "adm", url.Values{"name": {"Downtown"}, "lat": {"49.28273"}, "lon": {"-123.12074"}}), 303)
	as, _ := f.svc.ListAreas(ctx, actors["adm"])
	if len(as) != 1 || as[0].Cell == nil || as[0].Cell.Lat10 != 493 || as[0].Cell.Lon10 != -1231 {
		t.Fatalf("%+v", as)
	}
	w = f.get("adm", "/admin/taxonomy")
	if strings.Contains(w.Body.String(), "49.28") {
		t.Fatal("precise coordinate retained")
	}
	wantStatus(t, f.form("/admin/areas", "adm", url.Values{"name": {"Bad"}, "lat": {"abc"}, "lon": {"1"}}), 422)
	wantStatus(t, f.form("/admin/areas", "adm", url.Values{"name": {"Bad2"}, "lat": {"99"}, "lon": {"1"}}), 422)
	wantStatus(t, f.form("/admin/areas/"+as[0].ID, "adm", url.Values{"op": {"disable"}}), 303)
	// tag moderation via form
	f.listing("n1", "L1", "alice", "Bike")
	f.svc.UpdateDiscovery(ctx, actors["alice"], "L1", dInput("", "fixie"))
	w = f.get("adm", "/admin/taxonomy")
	if !strings.Contains(w.Body.String(), "Approve fixie") {
		t.Fatal("tag queue missing")
	}
	wantStatus(t, f.form("/admin/tags/fixie", "adm", url.Values{"status": {"approved"}}), 303)
	wantStatus(t, f.form("/admin/tags/fixie", "adm", url.Values{"status": {"weird"}}), 422)
	tags, _ := f.svc.ListTags(ctx, actors["adm"], d.TagApproved)
	if len(tags) != 1 {
		t.Fatal(tags)
	}
	// admin cannot cross networks
	wantStatus(t, f.form("/admin/categories/"+toolsID, "carol", url.Values{"op": {"enable"}}), 403)
	// admin all-listings view
	wantStatus(t, f.get("adm", "/listings?scope=all"), 200)
	wantStatus(t, f.get("bob", "/listings?scope=all"), 403)
	wantStatus(t, f.get("alice", "/listings?scope=mine"), 200)
}

func TestConstructionValidation(t *testing.T) {
	f := newFixture(t, "")
	auth := web.AuthFunc(func(r *http.Request) (d.Actor, error) { return d.Actor{}, fmt.Errorf("x") })
	for name, c := range map[string]web.Config{
		"no service":   {Auth: auth},
		"no auth":      {Service: f.svc},
		"slash suffix": {Service: f.svc, Auth: auth, BasePath: "/x/"},
		"no leading":   {Service: f.svc, Auth: auth, BasePath: "x"},
	} {
		if _, err := web.New(c); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// nothing is registered globally: the default mux stays empty
	if h, pat := http.DefaultServeMux.Handler(httptest.NewRequest("GET", "/listings", nil)); pat != "" {
		t.Fatalf("package registered on DefaultServeMux: %v", h)
	}
}
