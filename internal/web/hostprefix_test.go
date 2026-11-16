package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/httpx"
)

// SA2-14: on a secure deployment the CSRF cookie is __Host- prefixed, and
// plain (plantable) session and CSRF cookies are ignored.
func TestSecureDeploymentIgnoresPlainCookies(t *testing.T) {
	w := newWorldWith(t, func(o *Options) { o.Edge.ForceSecure, o.Edge.PublicHTTPS = true, true })
	b := w.browser(t)
	f := b.get("/login")
	var csrf string
	for _, c := range f.Header.Values("Set-Cookie") {
		if strings.HasPrefix(c, httpx.CSRFCookie+"=") {
			t.Errorf("plain csrf cookie set on a secure deployment: %s", c)
		}
		if strings.HasPrefix(c, httpx.CSRFCookieSecure+"=") {
			for _, want := range []string{"Secure", "Path=/", "HttpOnly"} {
				if !strings.Contains(c, want) {
					t.Errorf("__Host- csrf cookie lacks %s: %s", want, c)
				}
			}
			if strings.Contains(c, "Domain") {
				t.Errorf("__Host- cookie with Domain: %s", c)
			}
			csrf = strings.TrimPrefix(strings.SplitN(c, ";", 2)[0], httpx.CSRFCookieSecure+"=")
		}
	}
	if csrf == "" {
		t.Fatal("no __Host- csrf cookie")
	}
	token := httpx.CSRF{Key: []byte("0123456789abcdef0123456789abcdef")}.Token(csrf)
	post := func(cookie *http.Cookie) (status int, cookies []*http.Cookie) {
		req, _ := http.NewRequest("POST", b.url+"/login", strings.NewReader(url.Values{"login": {"alice@example.org"},
			"password": {testPW}, "csrf": {token}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(cookie)
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode, resp.Cookies()
	}
	// A plain cm_csrf (planted from a sibling subdomain) does not authorise a form.
	if st, _ := post(&http.Cookie{Name: httpx.CSRFCookie, Value: csrf}); st != 403 {
		t.Errorf("plain csrf cookie accepted on secure deployment: %d", st)
	}
	st, cookies := post(&http.Cookie{Name: httpx.CSRFCookieSecure, Value: csrf})
	if st != 303 {
		t.Fatalf("__Host- csrf login: %d", st)
	}
	var sess string
	for _, c := range cookies {
		if c.Name == httpx.SessionCookieSecure {
			sess = c.Value
		}
	}
	if sess == "" {
		t.Fatal("no session")
	}
	get := func(name string) int {
		req, _ := http.NewRequest("GET", b.url+"/account/password", nil)
		req.AddCookie(&http.Cookie{Name: name, Value: sess})
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if get(httpx.SessionCookie) != 303 {
		t.Error("plain circula_session accepted on a secure deployment")
	}
	if get(httpx.SessionCookieSecure) != 200 {
		t.Error("__Host- session rejected")
	}
}
