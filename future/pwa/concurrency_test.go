package pwa

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// The bundle is immutable after New; hammer it concurrently under -race and
// check every response stays byte-identical and correctly cache-headered.
func TestConcurrentServeIsRaceFreeAndStable(t *testing.T) {
	b, err := New(testConfig)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{b.StylePath(), b.ClientPath(), "/app/sw.js", "/app/manifest.webmanifest", "/app/install", "/app/api/v1/balance"}
	want := map[string]string{}
	for _, p := range paths {
		w := httptest.NewRecorder()
		b.ServeHTTP(w, httptest.NewRequest("GET", "https://c.test"+p, nil))
		want[p] = w.Body.String()
	}
	var wg sync.WaitGroup
	errs := make(chan string, 400)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				p := paths[i%len(paths)]
				w := httptest.NewRecorder()
				b.ServeHTTP(w, httptest.NewRequest("GET", "https://c.test"+p, nil))
				if w.Body.String() != want[p] {
					errs <- "body drift " + p
				}
				cc := w.Header().Get("Cache-Control")
				if strings.Contains(p, "/static/") != strings.Contains(cc, "immutable") {
					errs <- "cache header " + p + " " + cc
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}

func TestHeadMatchesGetHeadersWithoutBody(t *testing.T) {
	b, err := New(testConfig)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{b.StylePath(), "/app/sw.js", "/app/manifest.webmanifest", "/app/install"} {
		g, h := httptest.NewRecorder(), httptest.NewRecorder()
		b.ServeHTTP(g, httptest.NewRequest("GET", "https://c.test"+p, nil))
		b.ServeHTTP(h, httptest.NewRequest("HEAD", "https://c.test"+p, nil))
		if h.Body.Len() != 0 || h.Code != 200 || h.Header().Get("Content-Type") != g.Header().Get("Content-Type") || h.Header().Get("Cache-Control") != g.Header().Get("Cache-Control") {
			t.Fatal(p)
		}
	}
}

func FuzzCacheable(f *testing.F) {
	for _, s := range []string{"https://c.test/assets/a.0123abcd.css", "https://c.test/assets/a.0123abcd.css?x", "https://c.test/assets/../a.0123abcd.css", "https://c.test/assets/a.0123abcd.css#"} {
		f.Add(s, "GET", "style")
	}
	assets := map[string]bool{"/assets/a.0123abcd.css": true}
	f.Fuzz(func(t *testing.T, raw, method, dest string) {
		if Cacheable("https://c.test", raw, method, dest, assets) {
			if method != "GET" || strings.ContainsAny(raw, "?#") || !strings.HasPrefix(raw, "https://c.test/assets/") || strings.Contains(raw, "..") {
				t.Fatalf("unsafe admission %q %q %q", raw, method, dest)
			}
		}
	})
}
