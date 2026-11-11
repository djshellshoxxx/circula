package web

import (
	"strings"
	"sync"
	"time"
)

// limiter is a small fixed-window per-key counter (SEC-006). Callers either
// count every attempt with allow, or count only failures with fail and ask
// blocked beforehand, so that successful sign-ins never use up a budget.
// Stale windows are swept at most once per window.
type limiter struct {
	mu        sync.Mutex
	max       int
	window    time.Duration
	m         map[string]*win
	lastSweep time.Time
	now       func() time.Time
}

type win struct {
	start time.Time
	n     int
}

func newLimiter(max, seconds int) *limiter {
	return &limiter{max: max, window: time.Duration(seconds) * time.Second, m: map[string]*win{}, now: time.Now}
}

func (l *limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	l.lastSweep = now
	for k, w := range l.m {
		if now.Sub(w.start) > l.window {
			delete(l.m, k)
		}
	}
}

// allow records one attempt and reports whether it is within the limit.
func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.bump(key) <= l.max
}

func (l *limiter) bump(key string) int {
	now := l.now()
	l.sweep(now)
	w := l.m[key]
	if w == nil || now.Sub(w.start) > l.window {
		w = &win{start: now}
		l.m[key] = w
	}
	w.n++
	return w.n
}

// fail records one failure for key.
func (l *limiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.bump(key)
}

// blocked reports whether key has used its budget, without counting.
func (l *limiter) blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.m[key]
	return w != nil && l.now().Sub(w.start) <= l.window && w.n >= l.max
}

// reset forgets key (after a success).
func (l *limiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.m, key)
}

// acctKey scopes a login attempt to client address and login name, so one
// client guessing at one account does not consume the budget of other clients.
func acctKey(ip, login string) string {
	login = strings.ToLower(strings.TrimSpace(login))
	if len(login) > 128 {
		login = login[:128]
	}
	return ip + "|" + login
}
