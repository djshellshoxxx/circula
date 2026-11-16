package pwa

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBundleDefaultDisabled(t *testing.T) {
	b, e := New(Config{})
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", "https://c.test/app/sw.js", nil)
	w := httptest.NewRecorder()
	b.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
func TestConfigBoundaries(t *testing.T) {
	for _, c := range []Config{{Enabled: true, PublicOrigin: "http://evil.test", BasePath: "/"}, {Enabled: true, PublicOrigin: "https://c.test?", BasePath: "/"}, {Enabled: true, PublicOrigin: "https://c.test", BasePath: "//"}, {Enabled: true, PublicOrigin: "https://c.test", BasePath: "/app/", Find: "/app/../private"}, {Enabled: true, PublicOrigin: "https://c.test", BasePath: "/app/", Find: "https://evil.test/"}} {
		if _, e := New(c); e == nil {
			t.Fatal(c)
		}
	}
}
func TestBundleHeadersManifestAndScope(t *testing.T) {
	b, e := New(Config{Enabled: true, PublicOrigin: "https://c.test", BasePath: "/nested/app/"})
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{b.StylePath(), b.ClientPath(), "/nested/app/sw.js", "/nested/app/manifest.webmanifest", "/nested/app/install"} {
		w := httptest.NewRecorder()
		b.ServeHTTP(w, httptest.NewRequest("GET", "https://c.test"+p, nil))
		if w.Code != 200 {
			t.Fatal(p, w.Code)
		}
		if strings.Contains(p, "/static/") {
			if !strings.Contains(w.Header().Get("Cache-Control"), "public") {
				t.Fatal(p)
			}
		} else if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(p)
		}
		if strings.HasSuffix(p, "sw.js") {
			if w.Header().Get("Service-Worker-Allowed") != "/nested/app/" || strings.Contains(w.Body.String(), "__CIRCULA_CONFIG__") {
				t.Fatal("scope")
			}
		}
		if strings.HasSuffix(p, "webmanifest") {
			var m map[string]any
			if json.Unmarshal(w.Body.Bytes(), &m) != nil || m["scope"] != "/nested/app/" {
				t.Fatal(w.Body.String())
			}
		}
	}
	for _, u := range []string{"https://evil.test" + b.StylePath(), "https://c.test" + b.StylePath() + "?token=x", "https://c.test/nested/app/api/v1/balance"} {
		w := httptest.NewRecorder()
		b.ServeHTTP(w, httptest.NewRequest("GET", u, nil))
		if w.Code != 404 {
			t.Fatal(u, w.Code)
		}
	}
	w := httptest.NewRecorder()
	b.ServeHTTP(w, httptest.NewRequest("POST", "https://c.test/nested/app/sw.js", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}
