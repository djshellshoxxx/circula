package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// A3-01 end to end: attacker on IPv6 /64 A locks the victim on IPv6 /64 B.
func TestA3_01_WebIPv6CrossPrefixNoLockout(t *testing.T) {
	w := newWorld(t)
	h := NewHandlerWith(w.env.Svc, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: w.env.Clock})
	post := func(remote, login, pw string) int {
		g := httptest.NewRequest("GET", "/login", nil)
		g.RemoteAddr = remote
		rg := httptest.NewRecorder()
		h.ServeHTTP(rg, g)
		var ck *http.Cookie
		for _, c := range rg.Result().Cookies() {
			ck = c
		}
		tok := csrfRe.FindStringSubmatch(rg.Body.String())[1]
		p := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"login": {login}, "password": {pw}, "csrf": {tok}}.Encode()))
		p.RemoteAddr = remote
		p.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		p.AddCookie(ck)
		rp := httptest.NewRecorder()
		h.ServeHTTP(rp, p)
		return rp.Code
	}
	for i := 0; i < 5; i++ {
		post("[2001:db8:aaaa::1]:5555", "alice@example.org", "wrong password!!")
	}
	if c := post("[2001:db8:bbbb::1]:5555", "alice@example.org", testPW); c != 303 {
		t.Fatalf("victim from other /64 got %d instead of 303", c)
	}
}
