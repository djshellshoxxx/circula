package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func edge(cidrs ...string) Edge {
	var e Edge
	for _, c := range cidrs {
		e.TrustedProxies = append(e.TrustedProxies, netip.MustParsePrefix(c))
	}
	return e
}

func req(remote string, hdr ...string) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = remote
	for i := 0; i+1 < len(hdr); i += 2 {
		r.Header.Add(hdr[i], hdr[i+1])
	}
	return r
}

func TestClientIP(t *testing.T) {
	proxy := edge("10.0.0.0/8", "192.168.1.1/32")
	cases := []struct {
		name string
		e    Edge
		r    *http.Request
		want string
	}{
		{"direct", proxy, req("203.0.113.9:1111"), "203.0.113.9"},
		{"untrusted peer, XFF ignored", proxy, req("203.0.113.9:1111", "X-Forwarded-For", "1.1.1.1"), "203.0.113.9"},
		{"no proxies configured", Edge{}, req("10.0.0.1:1", "X-Forwarded-For", "1.1.1.1"), "10.0.0.1"},
		{"trusted peer", proxy, req("10.0.0.1:1", "X-Forwarded-For", "198.51.100.4"), "198.51.100.4"},
		{"forged left entries ignored", proxy, req("10.0.0.1:1", "X-Forwarded-For", "6.6.6.6, 198.51.100.4"), "198.51.100.4"},
		{"chain of proxies", proxy, req("10.0.0.1:1", "X-Forwarded-For", "198.51.100.4, 10.0.0.7, 192.168.1.1"), "198.51.100.4"},
		{"several header lines", proxy, req("10.0.0.1:1", "X-Forwarded-For", "6.6.6.6", "X-Forwarded-For", "198.51.100.4"), "198.51.100.4"},
		{"all trusted", proxy, req("10.0.0.1:1", "X-Forwarded-For", "10.0.0.2"), "10.0.0.1"},
		{"no header", proxy, req("10.0.0.1:1"), "10.0.0.1"},
		{"garbage entry", proxy, req("10.0.0.1:1", "X-Forwarded-For", "not-an-ip"), "10.0.0.1"},
		{"ipv4-mapped peer", proxy, req("[::ffff:10.0.0.1]:1", "X-Forwarded-For", "198.51.100.4"), "198.51.100.4"},
		{"ipv6 client", proxy, req("10.0.0.1:1", "X-Forwarded-For", "2001:db8::1"), "2001:db8::1"},
	}
	for _, c := range cases {
		if got := c.e.ClientIP(c.r); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSecureAndHSTS(t *testing.T) {
	proxy := edge("10.0.0.0/8")
	if proxy.Secure(req("203.0.113.9:1", "X-Forwarded-Proto", "https")) {
		t.Error("X-Forwarded-Proto from an untrusted peer believed")
	}
	if !proxy.Secure(req("10.0.0.1:1", "X-Forwarded-Proto", "https")) {
		t.Error("trusted X-Forwarded-Proto ignored")
	}
	if proxy.Secure(req("10.0.0.1:1", "X-Forwarded-Proto", "http")) {
		t.Error("http treated as secure")
	}
	if !(Edge{ForceSecure: true}).Secure(req("1.1.1.1:1")) {
		t.Error("ForceSecure ignored")
	}
	rec := httptest.NewRecorder()
	Edge{}.HSTS(rec, req("1.1.1.1:1"))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS on plain http")
	}
	Edge{PublicHTTPS: true}.HSTS(rec, req("1.1.1.1:1"))
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Error("no HSTS for an https deployment")
	}
}

func TestCSRFScheme(t *testing.T) {
	c := CSRF{Key: []byte("k")}
	tok := c.Token("session-1")
	if !c.Valid("session-1", tok) || c.Valid("session-2", tok) || c.Valid("session-1", "") || c.Valid("", tok) {
		t.Error("token binding")
	}
	if (CSRF{Key: []byte("other")}).Valid("session-1", tok) {
		t.Error("token valid under another key")
	}
	r := req("1.1.1.1:1")
	r.AddCookie(&http.Cookie{Name: CSRFCookie, Value: "pre"})
	if Binding(r) != "pre" {
		t.Error("pre-login binding")
	}
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "s"})
	if Binding(r) != "s" || SessionToken(r) != "s" {
		t.Error("session binding")
	}
	r2 := req("1.1.1.1:1")
	r2.AddCookie(&http.Cookie{Name: SessionCookieSecure, Value: "hs"})
	if SessionToken(r2) != "hs" {
		t.Error("__Host- cookie not read")
	}
}

func TestSafeNext(t *testing.T) {
	for _, bad := range []string{"", "x", "//x", "/\\x", "/\tx", "/\vx", "/%09/x", "/%0a/x", "/ x", "http://x", "/\x00"} {
		if got := SafeNext(bad); got != "" {
			t.Errorf("SafeNext(%q) = %q", bad, got)
		}
	}
	for _, good := range []string{"/", "/a/b?c=d", "/n/town/trades/1"} {
		if got := SafeNext(good); got != good {
			t.Errorf("SafeNext(%q) = %q", good, got)
		}
	}
}

type coded struct{ c int }

func (e coded) Error() string { return "x" }
func (e coded) Code() int     { return e.c }

func TestIsBusy(t *testing.T) {
	for _, e := range []error{errors.New("a: database is locked (5) (SQLITE_BUSY)"), coded{5}, coded{262}, coded{6}, errors.New("SQLITE_LOCKED")} {
		if !IsBusy(e) {
			t.Errorf("%v not busy", e)
		}
	}
	for _, e := range []error{nil, errors.New("disk full"), coded{1}} {
		if IsBusy(e) {
			t.Errorf("%v busy", e)
		}
	}
}
