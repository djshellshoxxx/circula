package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// SessionInfo is the owner's active-session view. ID is an independent,
// persisted public selector: neither the bearer credential nor its stored hash
// leaves the service. Expiry reflects the current role-specific idle and
// absolute lifetimes, just as Authenticate does.
type SessionInfo struct {
	ID         string
	Current    bool
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// SessionPage contains a bounded active-session page. Next is a public keyset
// position, not a credential. A nonempty Next must be followed even when Rows
// is empty: the scan may have reached its work cap in retained dead rows.
type SessionPage struct {
	Rows []SessionInfo
	Next string
}

func sessionPosition(created time.Time, id string) string {
	return strconv.FormatInt(created.UnixMilli(), 10) + ":" + id
}

func parseSessionPosition(pos string) (*store.SessionPosition, error) {
	if pos == "" {
		return nil, nil
	}
	stamp, id, ok := strings.Cut(pos, ":")
	ms, err := strconv.ParseInt(stamp, 10, 64)
	if !ok || err != nil || strconv.FormatInt(ms, 10) != stamp || !validSessionSelector(id) {
		return nil, domain.Invalid("cursor", "invalid session page position")
	}
	return &store.SessionPosition{CreatedAt: time.UnixMilli(ms), PublicID: id}, nil
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
// first with a public-ID tie-breaker. Limit defaults to 25 and must be 1..100.
// Each request seeks through at most 20 batches of 100 retained rows, including
// expired/revoked rows (API-034..036). No other person's ID is accepted as input.
func (s *Service) ListOwnSessions(ctx context.Context, actor domain.Actor, pg Page) (SessionPage, error) {
	out := SessionPage{Rows: []SessionInfo{}}
	err := s.view(ctx, func(tx store.Tx) error {
		idle, maxAge, err := s.ownSessionGate(tx, actor, domain.ScopeRead)
		if err != nil {
			return err
		}
		before, err := parseSessionPosition(pg.AfterID)
		if err != nil {
			return err
		}
		limit := pg.Limit
		if limit == 0 {
			limit = 25
		}
		if limit < 1 || limit > 100 {
			return domain.Invalid("limit", "must be between 1 and 100")
		}
		const batch, maxRounds = 100, 20
		now := s.now()
		for round := 0; round < maxRounds; round++ {
			rows, err := tx.ListSessions(actor.PersonID, before, batch)
			if err != nil {
				return err
			}
			for _, sess := range rows {
				expires := sessionExpiry(sess, idle, maxAge)
				if sess.RevokedAt != nil || !now.Before(expires) {
					continue
				}
				out.Rows = append(out.Rows, sessionInfo(sess, actor, expires))
				if len(out.Rows) > limit {
					// The extra live row belongs to the next page. Resume
					// from the last returned row, not the fetched batch end.
					out.Rows = out.Rows[:limit]
					last := out.Rows[len(out.Rows)-1]
					out.Next = sessionPosition(last.CreatedAt, last.ID)
					return nil
				}
			}
			if len(rows) < batch {
				return nil
			}
			last := rows[len(rows)-1]
			before = &store.SessionPosition{CreatedAt: last.CreatedAt, PublicID: last.PublicID}
		}
		// The bounded scan stopped before proving exhaustion. Advancing over
		// the scanned dead rows allows even an empty page to make progress.
		out.Next = sessionPosition(before.CreatedAt, before.PublicID)
		return nil
	})
	if err != nil {
		return SessionPage{}, err
	}
	return out, nil
}

func sessionInfo(sess *domain.Session, actor domain.Actor, expires time.Time) SessionInfo {
	return SessionInfo{ID: sess.PublicID, Current: sess.IDHash == actor.SessionID,
		CreatedAt: sess.CreatedAt, LastSeenAt: sess.LastSeenAt, ExpiresAt: expires}
}

// GetOwnSession performs one indexed owner-scoped lookup for the HTML
// confirmation page. Dead and foreign sessions are both not found.
func (s *Service) GetOwnSession(ctx context.Context, actor domain.Actor, id string) (*SessionInfo, error) {
	var out *SessionInfo
	err := s.view(ctx, func(tx store.Tx) error {
		idle, maxAge, err := s.ownSessionGate(tx, actor, domain.ScopeRead)
		if err != nil {
			return err
		}
		if !validSessionSelector(id) {
			return domain.NotFound("session")
		}
		sess, err := tx.GetOwnedSession(actor.PersonID, id)
		if err != nil {
			return err
		}
		expires := sessionExpiry(sess, idle, maxAge)
		if sess.RevokedAt != nil || !s.now().Before(expires) {
			return domain.NotFound("session")
		}
		info := sessionInfo(sess, actor, expires)
		out = &info
		return nil
	})
	return out, err
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
		sess, err := tx.GetOwnedSession(actor.PersonID, id)
		if err != nil {
			return err
		}
		changed, err := tx.RevokeOwnedSession(actor.PersonID, sess.IDHash, s.now())
		if err != nil || !changed {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActSessionRevoked,
			TargetType: "person", TargetID: actor.PersonID,
			After: map[string]any{"current": sess.IDHash == actor.SessionID}})
	})
}
