package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type ctxKey int

const (
	keyReqID ctxKey = iota
	keyCSRF
)

const csp = "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self'; " +
	"form-action 'self'; base-uri 'none'; frame-ancestors 'none'"

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("web: no randomness: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// reqIDPattern is the X-Request-Id an upstream layer (the serve edge, a
// proxy) may set and the UI keeps, so its log lines and error references match
// the access log (SEC-020). The JSON API accepts the same form.
var reqIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// wrap adds request id, security headers, panic recovery, body limit and CSRF.
func (s *server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !reqIDPattern.MatchString(id) {
			id = randHex(8)
		}
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cache-Control", "no-store")
		h.Set("X-Robots-Tag", "noindex, nofollow, noarchive") // FR-025
		h.Set("X-Request-Id", id)
		s.opts.Edge.HSTS(w, r)
		ctx := context.WithValue(r.Context(), keyReqID, id)
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("web panic", "request_id", id, "panic", rec)
				s.renderError(w, r.WithContext(ctx), http.StatusInternalServerError)
			}
		}()
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		ctx = s.ensureCSRF(w, r.WithContext(ctx))
		r = r.WithContext(ctx)
		if strings.HasPrefix(r.URL.Path, "/static/") {
			h.Set("Cache-Control", "public, max-age=3600")
		}
		if r.Method == http.MethodPost && !s.csrfOK(r) {
			s.renderError(w, r, errCSRF)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func reqID(r *http.Request) string {
	v, _ := r.Context().Value(keyReqID).(string)
	return v
}

// sameOrigin rejects cross-site browser requests when the browser says so.
// With a configured public base URL the Origin must be exactly that origin;
// otherwise it must match the Host header.
func (s *server) sameOrigin(r *http.Request) bool {
	if sf := r.Header.Get("Sec-Fetch-Site"); sf == "cross-site" {
		return false
	}
	o := r.Header.Get("Origin")
	if o == "null" {
		return false
	}
	if o == "" {
		return true
	}
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	if s.opts.PublicBaseURL != "" {
		pub, err := url.Parse(s.opts.PublicBaseURL)
		return err == nil && strings.EqualFold(u.Scheme, pub.Scheme) && strings.EqualFold(u.Host, pub.Host)
	}
	return u.Host == r.Host
}
