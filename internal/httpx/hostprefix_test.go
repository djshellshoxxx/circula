package httpx

import (
	"net/http"
	"testing"
)

// SA2-14: over HTTPS only the __Host- names are read; plain ones are ignored.
func TestHostPrefixOnlyWhenSecure(t *testing.T) {
	sec, plain := Edge{ForceSecure: true}, Edge{}
	r := req("1.1.1.1:1")
	r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "planted"})
	r.AddCookie(&http.Cookie{Name: CSRFCookie, Value: "planted"})
	if sec.SessionToken(r) != "" || sec.CSRFSecret(r) != "" || sec.Binding(r) != "" {
		t.Error("secure edge read a plain cookie")
	}
	if plain.SessionToken(r) != "planted" || plain.CSRFSecret(r) != "planted" {
		t.Error("plain edge ignored plain cookies")
	}
	r2 := req("1.1.1.1:1")
	r2.AddCookie(&http.Cookie{Name: SessionCookieSecure, Value: "hs"})
	r2.AddCookie(&http.Cookie{Name: CSRFCookieSecure, Value: "hc"})
	if plain.SessionToken(r2) != "" || plain.CSRFSecret(r2) != "" {
		t.Error("plain edge read a __Host- cookie")
	}
	if sec.SessionToken(r2) != "hs" || sec.CSRFSecret(r2) != "hc" || sec.Binding(r2) != "hs" {
		t.Error("secure edge did not read __Host- cookies")
	}
	if CSRFCookieName(true) != "__Host-cm_csrf" || CSRFCookieName(false) != "cm_csrf" {
		t.Error("csrf cookie names")
	}
}
