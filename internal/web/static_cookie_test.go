package web

import (
	"net/http"
	"testing"
)

// A2-15: public cache entries must never carry a fresh browser CSRF secret.
func TestPublicAssetsDoNotSetCookies(t *testing.T) {
	w := newWorld(t)
	for _, path := range []string{"/static/style.css", "/static/app.js", "/robots.txt", "/favicon.ico"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			b := w.browser(t)
			r, err := http.NewRequest(method, b.url+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			p := b.do(r)
			if p.Status != 200 && p.Status != 204 {
				t.Fatalf("%s %s: %d", method, path, p.Status)
			}
			if got := p.Header.Values("Set-Cookie"); len(got) != 0 {
				t.Fatalf("%s %s cached response sets cookies: %v", method, path, got)
			}
		}
	}
	b := w.browser(t)
	login := b.get("/login")
	if len(login.Header.Values("Set-Cookie")) == 0 || login.csrf() == "" || login.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("login no longer issues a private CSRF-protected form")
	}
}
