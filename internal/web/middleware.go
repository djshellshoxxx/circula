package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
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

// wrap adds request id, security headers, panic recovery, body limit and CSRF.
func (s *server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := randHex(8)
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cache-Control", "no-store")
		h.Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), keyReqID, id)
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("web: panic req=%s: %v", id, rec)
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
func sameOrigin(r *http.Request) bool {
	if sf := r.Header.Get("Sec-Fetch-Site"); sf == "cross-site" {
		return false
	}
	if o := r.Header.Get("Origin"); o != "" && o != "null" {
		u, err := url.Parse(o)
		if err != nil || u.Host != r.Host {
			return false
		}
	} else if o == "null" {
		return false
	}
	return true
}
