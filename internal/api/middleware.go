package api

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ridPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// credentialParams are query names that must never carry credentials (API-012).
var credentialParams = []string{"token", "access_token", "api_key", "apikey", "session", "session_id",
	"sessionid", "sid", "authorization", "password", "secret", "csrf", "csrf_token"}

// edge wraps next with request id, security headers, panic recovery, IP rate
// limiting, credentials-in-query rejection and the body size limit.
func (s *server) edge(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if !ridPattern.MatchString(rid) {
			rid = newRequestID()
		}
		h := w.Header()
		h.Set("X-Request-Id", rid)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Cache-Control", "no-store") // API-039; safe default for every response
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.Error("api panic", "request_id", rid, "panic", v)
				writeProblem(w, r, rid, errInternal)
			}
		}()
		if s.opt.IPPerMinute > 0 {
			if ok, retry := s.rl.allow("ip:"+clientIP(r), s.opt.IPPerMinute); !ok {
				writeProblem(w, r, rid, rateLimited(retry))
				return
			}
		}
		if hasCredentialQuery(r) {
			writeProblem(w, r, rid, &apiError{status: 400, code: "credentials-in-query",
				detail: "credentials must not be sent in the URL"})
			return
		}
		if r.ContentLength > s.opt.MaxBodyBytes {
			writeProblem(w, r, rid, errTooLarge)
			return
		}
		r.Header.Set("X-Request-Id", rid) // normalised value for handlers
		next.ServeHTTP(w, r)
	})
}

func hasCredentialQuery(r *http.Request) bool {
	for k, vs := range r.URL.Query() {
		lk := strings.ToLower(k)
		for _, c := range credentialParams {
			if lk == c {
				return true
			}
		}
		for _, v := range vs {
			if strings.HasPrefix(v, "cir_") || strings.HasPrefix(v, "inv_") {
				return true
			}
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limiter is a fixed-window counter per key (API-038). In memory, per process.
type limiter struct {
	now func() time.Time
	mu  sync.Mutex
	m   map[string]*window
}

type window struct {
	start time.Time
	n     int
}

func newLimiter(now func() time.Time) *limiter { return &limiter{now: now, m: map[string]*window{}} }

func (l *limiter) allow(key string, perMinute int) (bool, int) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.m) > 50000 {
		for k, w := range l.m {
			if now.Sub(w.start) >= time.Minute {
				delete(l.m, k)
			}
		}
	}
	w := l.m[key]
	if w == nil || now.Sub(w.start) >= time.Minute {
		l.m[key] = &window{start: now, n: 1}
		return true, 0
	}
	w.n++
	if w.n > perMinute {
		retry := int(time.Minute.Seconds() - now.Sub(w.start).Seconds())
		if retry < 1 {
			retry = 1
		}
		return false, retry
	}
	return true, 0
}

func rateLimited(retry int) *apiError {
	return &apiError{status: 429, code: "rate-limited", detail: "rate limit exceeded", retryAfter: strconv.Itoa(retry)}
}
