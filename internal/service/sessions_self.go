package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// SessionInfo is the owner's active-session view. ID is a domain-separated,
// one-way selector: neither the bearer credential nor its stored hash leaves
// the service. Expiry reflects the current role-specific idle and absolute
// lifetimes, just as Authenticate does.
type SessionInfo struct {
	ID         string
	Current    bool
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

func sessionSelector(idHash string) string {
	sum := sha256.Sum256([]byte("circula/session-selector/v1\x00" + idHash))
	return hex.EncodeToString(sum[:])
}

func validSessionSelector(id string) bool {
	if len(id) != 64 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func sessionExpiry(sess *domain.Session, idle, maxAge time.Duration) time.Time {
	expires := sess.ExpiresAt
	if t := sess.CreatedAt.Add(maxAge); t.Before(expires) {
		expires = t
	}
	if t := sess.LastSeenAt.Add(idle); t.Before(expires) {
		expires = t
	}
	return expires
}

// ownSessionGate is checked inside the same read/write unit as the action.
// A retained actor cannot list or revoke after its own session was ended.
// API tokens and operators do not gain session-management authority.
func (s *Service) ownSessionGate(tx store.Tx, actor domain.Actor, scope domain.Scope) (time.Duration, time.Duration, error) {
	if actor.System || actor.PersonID == "" || actor.SessionID == "" || actor.NetworkScope != "" {
		return 0, 0, domain.Forbidden("session authentication is required")
	}
	if scope == domain.ScopeWrite && actor.MustChangePassword {
		return 0, 0, domain.PasswordChangeRequired()
	}
	if !actor.Can(scope) {
		return 0, 0, domain.Forbidden("session scope required")
	}
	sess, err := tx.GetSession(actor.SessionID)
	if err != nil {
		return 0, 0, authnErr(err)
	}
	if sess.PersonID != actor.PersonID {
		return 0, 0, domain.ErrUnauthenticated
	}
	idle, maxAge, err := s.sessionLifetimes(tx, actor.PersonID)
	if err != nil {
		return 0, 0, err
	}
	if sess.RevokedAt != nil || !s.now().Before(sessionExpiry(sess, idle, maxAge)) {
		return 0, 0, domain.ErrUnauthenticated
	}
	mustChange, err := tx.MustChangePassword(actor.PersonID)
	if err != nil {
		return 0, 0, err
	}
	if scope == domain.ScopeWrite && mustChange {
		return 0, 0, domain.PasswordChangeRequired()
	}
	return idle, maxAge, nil
}

// ListOwnSessions lists active sessions of the authenticated person, newest
// first. No network or other person's identifier is accepted as input.
func (s *Service) ListOwnSessions(ctx context.Context, actor domain.Actor) ([]SessionInfo, error) {
	var out []SessionInfo
	err := s.view(ctx, func(tx store.Tx) error {
		idle, maxAge, err := s.ownSessionGate(tx, actor, domain.ScopeRead)
		if err != nil {
			return err
		}
		rows, err := tx.ListSessions(actor.PersonID)
		if err != nil {
			return err
		}
		now := s.now()
		for _, sess := range rows {
			expires := sessionExpiry(sess, idle, maxAge)
			if sess.RevokedAt != nil || !now.Before(expires) {
				continue
			}
			out = append(out, SessionInfo{ID: sessionSelector(sess.IDHash), Current: sess.IDHash == actor.SessionID,
				CreatedAt: sess.CreatedAt, LastSeenAt: sess.LastSeenAt, ExpiresAt: expires})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Use only the public selector as a tie-breaker, so pagination need not
	// reveal a credential hash when several logins share a millisecond.
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

// RevokeOwnSession ends only the selected session of the caller. Unknown and
// foreign selectors are indistinguishable. Repeating an already completed
// revocation has no additional effect or audit event while the row is retained.
func (s *Service) RevokeOwnSession(ctx context.Context, actor domain.Actor, id string) error {
	return s.update(ctx, func(tx store.Tx) error {
		if _, _, err := s.ownSessionGate(tx, actor, domain.ScopeWrite); err != nil {
			return err
		}
		if !validSessionSelector(id) {
			return domain.NotFound("session")
		}
		rows, err := tx.ListSessions(actor.PersonID)
		if err != nil {
			return err
		}
		for _, sess := range rows {
			if sessionSelector(sess.IDHash) != id {
				continue
			}
			changed, err := tx.RevokeOwnedSession(actor.PersonID, sess.IDHash, s.now())
			if err != nil || !changed {
				return err
			}
			return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActSessionRevoked,
				TargetType: "person", TargetID: actor.PersonID,
				After: map[string]any{"current": sess.IDHash == actor.SessionID}})
		}
		return domain.NotFound("session")
	})
}
