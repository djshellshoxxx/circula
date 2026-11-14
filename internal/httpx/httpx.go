// Package httpx holds the HTTP edge policy shared by the HTML and JSON
// adapters: one session cookie name, one CSRF scheme, trusted-proxy client
// address handling and the Secure/HSTS decision. Both adapters must use it so
// that a browser session set by the sign-in flow works for the API (API-008,
// API-014).
package httpx

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// Cookie names. When the request is secure (HTTPS) the sign-in flow sets and
// readers accept only the __Host- spelling (it needs Secure, Path=/ and no
// Domain, so a sibling subdomain cannot plant it, SA2-14); otherwise only the
// plain names are used.
const (
	SessionCookie       = "circula_session"
	SessionCookieSecure = "__Host-circula_session"
	CSRFCookie          = "cm_csrf"
	CSRFCookieSecure    = "__Host-cm_csrf"
)

// SessionCookieName returns the name the sign-in flow uses for secure.
func SessionCookieName(secure bool) string {
	if secure {
		return SessionCookieSecure
	}
	return SessionCookie
}

// CSRFCookieName returns the pre-login CSRF cookie name for secure.
func CSRFCookieName(secure bool) string {
	if secure {
		return CSRFCookieSecure
	}
	return CSRFCookie
}

// SessionToken returns the session token carried by the request, "" if none.
// Over HTTPS only the __Host- cookie counts; a plain circula_session is ignored.
func (e Edge) SessionToken(r *http.Request) string {
	if c, err := r.Cookie(SessionCookieName(e.Secure(r))); err == nil {
		return c.Value
	}
	return ""
}

// CSRFSecret returns the pre-login CSRF cookie value ("" if none).
func (e Edge) CSRFSecret(r *http.Request) string {
	if c, err := r.Cookie(CSRFCookieName(e.Secure(r))); err == nil {
		return c.Value
	}
	return ""
}

// CSRF is the single CSRF scheme: the token is an HMAC of a binding value
// (the session token when signed in, else the pre-login csrf cookie value),
// keyed with a process-wide key that the HTML and JSON adapters share.
type CSRF struct{ Key []byte }

// Token derives the token for a binding value.
func (c CSRF) Token(binding string) string {
	m := hmac.New(sha256.New, c.Key)
	m.Write([]byte("circula-csrf-v2\x00"))
	m.Write([]byte(binding))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// Valid reports whether got is the token for binding (constant time).
func (c CSRF) Valid(binding, got string) bool {
	if binding == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(c.Token(binding))) == 1
}

// Binding returns the value a request's CSRF token is bound to.
func (e Edge) Binding(r *http.Request) string {
	if s := e.SessionToken(r); s != "" {
		return s
	}
	return e.CSRFSecret(r)
}

// Edge describes how requests reach the process.
type Edge struct {
	// ForceSecure marks cookies Secure regardless of the request (setting cookie-secure).
	ForceSecure bool
	// PublicHTTPS is true when the public base URL is https; responses then carry HSTS.
	PublicHTTPS bool
	// TrustedProxies are the peers whose X-Forwarded-For and X-Forwarded-Proto
	// headers are believed. Empty: those headers are ignored.
	TrustedProxies []netip.Prefix
}

func remoteAddr(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return a.Unmap()
}

func (e Edge) trusted(a netip.Addr) bool {
	if !a.IsValid() {
		return false
	}
	for _, p := range e.TrustedProxies {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// ClientIP returns the client address. X-Forwarded-For is consulted only when
// the TCP peer is a trusted proxy, and then the right-most entry that is not
// itself a trusted proxy wins, so a client cannot choose its own address.
func (e Edge) ClientIP(r *http.Request) string {
	peer := remoteAddr(r)
	if !peer.IsValid() {
		return r.RemoteAddr
	}
	if !e.trusted(peer) {
		return peer.String()
	}
	var parts []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		parts = append(parts, strings.Split(v, ",")...)
	}
	for i := len(parts) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err != nil {
			return peer.String() // malformed chain: do not guess
		}
		a = a.Unmap()
		if !e.trusted(a) {
			return a.String()
		}
	}
	return peer.String()
}

// RateKey returns the address a per-client limiter should count: IPv4 as is
// (/32), IPv6 truncated to its /64 so that a client rotating addresses inside
// its allocation shares one budget (SA2-15). IPv4-mapped IPv6 is normalised to
// IPv4 first.
func RateKey(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	a = a.Unmap().WithZone("")
	if a.Is6() {
		p, err := a.Prefix(64)
		if err != nil {
			return ip
		}
		return p.String()
	}
	return a.String()
}

// RateKey is RateKey applied to the client address of r.
func (e Edge) RateKey(r *http.Request) string { return RateKey(e.ClientIP(r)) }

// Secure reports whether the request reached the user over HTTPS.
func (e Edge) Secure(r *http.Request) bool {
	if e.ForceSecure || r.TLS != nil {
		return true
	}
	if e.trusted(remoteAddr(r)) {
		v, _, _ := strings.Cut(r.Header.Get("X-Forwarded-Proto"), ",")
		return strings.EqualFold(strings.TrimSpace(v), "https")
	}
	return false
}

// HSTS sets Strict-Transport-Security when the deployment is HTTPS.
func (e Edge) HSTS(w http.ResponseWriter, r *http.Request) {
	if e.PublicHTTPS || r.TLS != nil || e.Secure(r) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
}

// IsBusy reports whether err is SQLite lock contention (SQLITE_BUSY or
// SQLITE_LOCKED): a retryable condition, not a server fault (OPS-011).
func IsBusy(err error) bool {
	if err == nil {
		return false
	}
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		switch coded.Code() & 0xff {
		case 5, 6:
			return true
		}
	}
	s := err.Error()
	return strings.Contains(s, "SQLITE_BUSY") || strings.Contains(s, "SQLITE_LOCKED") ||
		strings.Contains(s, "database is locked") || strings.Contains(s, "database table is locked")
}

// SafeNext returns next when it is a plain local path, else "". It rejects
// control characters (browsers strip TAB, CR and LF inside URLs, turning
// "/<TAB>/host" into "//host"), backslashes, a second leading slash and
// anything that parses with a scheme or host (SEC-024).
func SafeNext(next string) string {
	if next == "" || len(next) > 2048 || next[0] != '/' {
		return ""
	}
	for i := 0; i < len(next); i++ {
		c := next[i]
		if c < 0x20 || c == 0x7f || c == '\\' || c == ' ' {
			return ""
		}
	}
	if len(next) > 1 && next[1] == '/' {
		return ""
	}
	// Percent-encoded forms of the same characters are refused as well.
	dec, err := url.PathUnescape(next)
	if err != nil || strings.HasPrefix(dec, "//") {
		return ""
	}
	for i := 0; i < len(dec); i++ {
		if c := dec[i]; c < 0x20 || c == 0x7f || c == '\\' {
			return ""
		}
	}
	u, perr := url.Parse(next)
	if perr != nil || u.Scheme != "" || u.Host != "" || u.User != nil {
		return ""
	}
	return next
}
