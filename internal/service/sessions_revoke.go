package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// RevokeSessionsInput selects the sessions `circula revoke-sessions` ends.
type RevokeSessionsInput struct {
	// Login, when non-empty, limits the revocation to that person's sessions.
	// Empty ends every session of every person.
	Login string
	// Reason is mandatory and recorded in the audit event.
	Reason string
	// WebKeyRotated records in the audit event that the caller replaces the
	// web signing key in the same operation (circula revoke-sessions
	// -rotate-web-key). The service does not touch the key itself.
	WebKeyRotated bool
}

// RevokeAllSessions is the operator's emergency sign-out (SEC-027, OPS-010):
// it revokes every live session, or every live session of in.Login, and
// records one platform audit event (sessions.revoked, platform operator) with
// the count and the reason. actor must be domain.SystemActor (the CLI). It
// writes nothing else: no ledger entry, transaction, balance or policy is
// touched, and API tokens stay valid (they are separate credentials). Sessions
// are opaque server-side rows, so revoking them here, not replacing the web
// signing key, is what makes old session cookies stop working. An unknown
// login is NotFound. The result is the number of sessions revoked.
func (s *Service) RevokeAllSessions(ctx context.Context, actor domain.Actor, in RevokeSessionsInput) (int64, error) {
	if !actor.System {
		return 0, domain.ErrForbidden
	}
	reason, err := checkResetReason(in.Reason)
	if err != nil {
		return 0, err
	}
	login := normalizeLogin(in.Login)
	var n int64
	err = s.update(ctx, func(tx store.Tx) error {
		personID, target, scope := "", "sessions", "all"
		if login != "" {
			p, _, err := tx.GetPersonByLogin(login)
			if err != nil {
				return domain.NotFound("person")
			}
			personID, target, scope = p.ID, "person", "person"
		}
		c, err := tx.RevokeLiveSessions(personID, s.now())
		if err != nil {
			return err
		}
		n = c
		// A pending second-factor challenge is a 5-minute bearer value that
		// turns into a session; the other revocation paths drop them too
		// (SA2-13), so an emergency sign-out must not leave them usable.
		if personID != "" {
			err = tx.DeleteMFAChallengesOf(personID)
		} else {
			err = tx.DeleteAllMFAChallenges()
		}
		if err != nil {
			return err
		}
		return s.auditPlatform(tx, actor, auditEntry{Type: domain.ActorOperator, Action: domain.ActSessionsRevoked,
			TargetType: target, TargetID: personID, Reason: reason,
			After: map[string]any{"revoked": c, "scope": scope, "web_key_rotated": in.WebKeyRotated}})
	})
	return n, err
}

// roleLifetime is the role-specific session lifetime v, never longer than the
// member lifetime def; zero (unset) means def (SEC-003).
func roleLifetime(v, def time.Duration) time.Duration {
	if v <= 0 || v > def {
		return def
	}
	return v
}

// roleLifetimesSet reports whether any role-specific lifetime is shorter than
// the member one, i.e. whether Authenticate must look at the person's roles.
func (s *Service) roleLifetimesSet() bool {
	c := s.cfg
	return c.AdminSessionIdle < c.SessionIdle || c.AdminSessionMax < c.SessionMax ||
		c.ReviewerSessionIdle < c.SessionIdle || c.ReviewerSessionMax < c.SessionMax
}

// sessionLifetimes returns the idle and absolute session lifetimes for
// personID (SEC-003, DEC-005): the member values, shortened to the
// administrator or reviewer values when the person holds that role in any
// network where they are not departed, including a live break-glass grant.
// Without role-specific values it does no lookup.
func (s *Service) sessionLifetimes(tx store.Tx, personID string) (idle, maxAge time.Duration, err error) {
	idle, maxAge = s.cfg.SessionIdle, s.cfg.SessionMax
	if !s.roleLifetimesSet() {
		return idle, maxAge, nil
	}
	ms, err := tx.ListMembershipsOfPerson(personID)
	if err != nil {
		return 0, 0, err
	}
	for _, m := range ms {
		if m.Status == domain.MemberDeparted {
			continue
		}
		// Roles granted through break-glass pass admin checks like assigned
		// ones (applyBreakGlassRoles), so they shorten the lifetime too; an
		// error fails closed.
		if err := s.applyBreakGlassRoles(tx, m); err != nil {
			return 0, 0, err
		}
		if m.HasRole(domain.RoleAdmin) {
			idle, maxAge = min(idle, s.cfg.AdminSessionIdle), min(maxAge, s.cfg.AdminSessionMax)
		}
		if m.HasRole(domain.RoleReviewer) {
			idle, maxAge = min(idle, s.cfg.ReviewerSessionIdle), min(maxAge, s.cfg.ReviewerSessionMax)
		}
	}
	return idle, maxAge, nil
}
