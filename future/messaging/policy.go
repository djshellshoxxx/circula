// Package messaging validates messages before a transactional persistence adapter.
// Messages are server-readable text; this package makes no encryption claim.
package messaging

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrDenied = errors.New("message unavailable")

type Access struct {
	Network, Member string
	Active          bool
}
type Thread struct {
	Network, A, B   string
	Closed, Blocked bool
}

func (t Thread) Authorize(a Access) error {
	if a.Network == "" || a.Network != t.Network || !a.Active || t.A == "" || t.B == "" || t.A == t.B || a.Member == "" || (a.Member != t.A && a.Member != t.B) || t.Closed || t.Blocked {
		return ErrDenied
	}
	return nil
}

// Prepare returns literal text. Renderers MUST escape it, never mark it trusted HTML.
func Prepare(t Thread, a Access, body string) (string, error) {
	if e := t.Authorize(a); e != nil {
		return "", e
	}
	if !utf8.ValidString(body) || len(body) > 4096 || strings.TrimSpace(body) == "" || strings.ContainsRune(body, 0) {
		return "", errors.New("invalid message")
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return "", errors.New("invalid message")
		}
	}
	return body, nil
}

// PurgeAfter is a thread-level inactivity deadline, not a per-message timer.
// Legal hold must be checked separately by the persistence adapter.
func PurgeAfter(lastActivity time.Time, days int) (time.Time, error) {
	if lastActivity.IsZero() || days < 1 || days > 30 {
		return time.Time{}, errors.New("invalid retention")
	}
	return lastActivity.Add(time.Duration(days) * 24 * time.Hour), nil
}
