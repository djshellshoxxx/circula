package web

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// limiter is a small fixed-window per-address counter (SEC-006). It bounds
// memory by dropping stale windows when it grows.
type limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	m      map[string]*win
}

type win struct {
	start time.Time
	n     int
}

func newLimiter(max, seconds int) *limiter {
	return &limiter{max: max, window: time.Duration(seconds) * time.Second, m: map[string]*win{}}
}

// allow records one attempt and reports whether it is within the limit.
func (l *limiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.m) > 10000 {
		for k, w := range l.m {
			if now.Sub(w.start) > l.window {
				delete(l.m, k)
			}
		}
	}
	w := l.m[key]
	if w == nil || now.Sub(w.start) > l.window {
		w = &win{start: now}
		l.m[key] = w
	}
	w.n++
	return w.n <= l.max
}

func remoteIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}
