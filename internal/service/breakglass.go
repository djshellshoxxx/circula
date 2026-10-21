package service

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Operator break-glass (SEC-013, DEC-014, TA-5).
//
// A break-glass session is the only authority under which platform staff may
// act on a network's rights. It is time boxed (hard maximum four hours), needs
// a documented reason, and is audited at platform scope and at network scope,
// where the network's administrators can read it (ListBreakGlass, the audit
// log). The operator methods take a System actor: they are reached only from
// the operator CLI, never from HTTP.

// OpenBreakGlassInput opens a session.
type OpenBreakGlassInput struct {
	Network  string        // slug or id
	Operator string        // operator label (name of the staff member)
	Reason   string        // documented reason, mandatory
	TTL      time.Duration // zero = default; clamped to domain.BreakGlassMaxTTL
}

// GrantBreakGlassInput gives a named existing member a temporary role.
type GrantBreakGlassInput struct {
	Network string // slug or id
	Login   string // login of the person; must already be an active member
	Reason  string
}

func requireSystem(actor domain.Actor) error {
	if !actor.System {
		return domain.Forbidden("operator action")
	}
	return nil
}

func cleanLabel(field, v string, max int) (string, error) {
	v = strings.TrimSpace(v)
	if c := utf8.RuneCountInString(v); c < 1 || c > max {
		return "", domain.Invalid(field, "required, at most 80 characters")
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return "", domain.Invalid(field, "must not contain control characters")
		}
	}
	return v, nil
}

// clampTTL applies the hard maximum: zero or negative selects the default,
// anything above the maximum is reduced to it.
func clampTTL(d time.Duration) time.Duration {
	if d <= 0 {
		return domain.BreakGlassDefaultTTL
	}
	if d > domain.BreakGlassMaxTTL {
		return domain.BreakGlassMaxTTL
	}
	return d
}

func resolveNetwork(tx store.Tx, ref string) (*domain.Network, error) {
	if n, err := tx.GetNetworkBySlug(ref); err == nil {
		return n, nil
	}
	n, err := tx.GetNetwork(ref)
	if err != nil {
		return nil, domain.NotFound("network")
	}
	return n, nil
}

// auditBreakGlass writes the event at platform scope and at the network's scope
// in the caller's unit of work. Network administrators read the second one.
func (s *Service) auditBreakGlass(tx store.Tx, actor domain.Actor, netID string, e auditEntry) error {
	e.Type = domain.ActorOperator
	if err := s.emit(tx, "", "", actor.RequestID, e); err != nil {
		return err
	}
	return s.emit(tx, netID, "", actor.RequestID, e)
}

// closeExpired closes every open session whose expiry has passed, auditing
// each. It is the lazy half of expiry; ExpireBreakGlass is the scheduled half.
func (s *Service) closeExpired(tx store.Tx, actor domain.Actor, limit int) (int, error) {
	now := s.now()
	exp, err := tx.ExpiredOpenBreakGlass(now, limit)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, b := range exp {
		closedAt := b.ExpiresAt
		ok, err := tx.CloseBreakGlass(b.ID, closedAt, domain.BreakGlassClosedExpired)
		if err != nil {
			return n, err
		}
		if !ok {
			continue
		}
		n++
		if err := s.auditBreakGlass(tx, actor, b.NetworkID, auditEntry{Action: domain.ActBreakGlassExpired,
			TargetType: "break_glass", TargetID: b.ID,
			Before: map[string]any{"status": "open"}, After: map[string]any{"status": "expired", "closed_at": closedAt.UnixMilli()}}); err != nil {
			return n, err
		}
	}
	return n, nil
}

// ExpireBreakGlass closes sessions that ran past their expiry. Call it from the
// job runner. Their temporary roles stopped applying at the expiry instant
// already; this makes the closure visible in the trail. System actor only.
func (s *Service) ExpireBreakGlass(ctx context.Context, actor domain.Actor) (int, error) {
	if err := requireSystem(actor); err != nil {
		return 0, err
	}
	total := 0
	for {
		var n int
		err := s.update(ctx, func(tx store.Tx) error {
			var err error
			n, err = s.closeExpired(tx, actor, 100)
			return err
		})
		if err != nil {
			return total, err
		}
		total += n
		if n < 100 {
			return total, nil
		}
	}
}

// OpenBreakGlass records a new session and audits it. The TTL is clamped to
// four hours. Operator CLI only (System actor).
func (s *Service) OpenBreakGlass(ctx context.Context, actor domain.Actor, in OpenBreakGlassInput) (*domain.BreakGlassSession, error) {
	if err := requireSystem(actor); err != nil {
		return nil, err
	}
	op, err := cleanLabel("operator", in.Operator, 80)
	if err != nil {
		return nil, err
	}
	if err := requireReason(in.Reason); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	var out *domain.BreakGlassSession
	err = s.update(ctx, func(tx store.Tx) error {
		net, err := resolveNetwork(tx, in.Network)
		if err != nil {
			return err
		}
		if _, err := s.closeExpired(tx, actor, 100); err != nil {
			return err
		}
		now := s.now()
		b := &domain.BreakGlassSession{ID: s.id(), NetworkID: net.ID, OperatorLabel: op, Reason: reason,
			OpenedAt: now, ExpiresAt: now.Add(clampTTL(in.TTL))}
		if err := tx.InsertBreakGlass(b); err != nil {
			return err
		}
		out = b
		return s.auditBreakGlass(tx, actor, net.ID, auditEntry{Action: domain.ActBreakGlassOpened, TargetType: "break_glass",
			TargetID: b.ID, Reason: reason,
			After: map[string]any{"operator": op, "expires_at": b.ExpiresAt.UnixMilli()}})
	})
	return out, err
}

// CloseBreakGlass ends a session early. Operator CLI only. Closing an already
// closed or expired session is refused.
func (s *Service) CloseBreakGlass(ctx context.Context, actor domain.Actor, sessionID, reason string) (*domain.BreakGlassSession, error) {
	if err := requireSystem(actor); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if err := boundedText("reason", reason, maxReasonLen); err != nil {
		return nil, err
	}
	var out *domain.BreakGlassSession
	err := s.update(ctx, func(tx store.Tx) error {
		if _, err := s.closeExpired(tx, actor, 100); err != nil {
			return err
		}
		b, err := tx.GetBreakGlass(sessionID)
		if err != nil {
			return domain.NotFound("break-glass session")
		}
		now := s.now()
		why := domain.BreakGlassClosedByOperator
		ok, err := tx.CloseBreakGlass(b.ID, now, why)
		if err != nil {
			return err
		}
		if !ok {
			return domain.InvalidState("break-glass session is already %s", b.Status(now))
		}
		if err := s.auditBreakGlass(tx, actor, b.NetworkID, auditEntry{Action: domain.ActBreakGlassClosed, TargetType: "break_glass",
			TargetID: b.ID, Reason: reason,
			Before: map[string]any{"status": "open"}, After: map[string]any{"status": "closed"}}); err != nil {
			return err
		}
		out, err = tx.GetBreakGlass(b.ID)
		return err
	})
	return out, err
}

// ListBreakGlassOperator lists sessions of one network (slug or id) for the
// operator CLI. System actor only.
func (s *Service) ListBreakGlassOperator(ctx context.Context, actor domain.Actor, network string, pg Page) ([]*domain.BreakGlassSession, error) {
	if err := requireSystem(actor); err != nil {
		return nil, err
	}
	var out []*domain.BreakGlassSession
	err := s.view(ctx, func(tx store.Tx) error {
		net, err := resolveNetwork(tx, network)
		if err != nil {
			return err
		}
		out, err = tx.ListBreakGlass(net.ID, pg.AfterID, clampPage(pg))
		return err
	})
	return out, err
}

// ListBreakGlass shows a network's break-glass history to its administrators
// (SEC-013 b). Read only; a session past its expiry is reported as expired
// even before the job has closed it.
func (s *Service) ListBreakGlass(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]*domain.BreakGlassSession, error) {
	var out []*domain.BreakGlassSession
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.actor.System {
			return domain.Forbidden("administrator role required")
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = tx.ListBreakGlass(netID, pg.AfterID, clampPage(pg))
		return err
	})
	return out, err
}

// requireBreakGlassTx fails unless the network has a live session. It returns the newest one.
func (s *Service) requireBreakGlassTx(tx store.Tx, netID string) (*domain.BreakGlassSession, error) {
	live, err := tx.LiveBreakGlass(netID, s.now())
	if err != nil {
		return nil, err
	}
	if len(live) == 0 {
		return nil, domain.Forbidden("operator access to this network requires an open break-glass session")
	}
	return live[0], nil
}

// RequireBreakGlass is the gate every operator action that reads or changes a
// network's data or rights must pass (SEC-013). It returns forbidden unless
// the network has an open, unexpired break-glass session. A session at or past
// its expiry never passes, whether or not the job has closed it yet.
func (s *Service) RequireBreakGlass(ctx context.Context, netID string) error {
	return s.view(ctx, func(tx store.Tx) error {
		_, err := s.requireBreakGlassTx(tx, netID)
		return err
	})
}

// GrantBreakGlassAdmin gives a named, active, existing member the admin role
// until the newest live break-glass session of the network ends. It is the only
// way the operator can obtain network rights. The grant is a row linked to the
// session, so it stops applying at the session's expiry without any further
// action (see applyBreakGlassRoles). System actor only.
func (s *Service) GrantBreakGlassAdmin(ctx context.Context, actor domain.Actor, in GrantBreakGlassInput) (*domain.BreakGlassSession, error) {
	if err := requireSystem(actor); err != nil {
		return nil, err
	}
	if err := requireReason(in.Reason); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	var out *domain.BreakGlassSession
	err := s.update(ctx, func(tx store.Tx) error {
		net, err := resolveNetwork(tx, in.Network)
		if err != nil {
			return err
		}
		if _, err := s.closeExpired(tx, actor, 100); err != nil {
			return err
		}
		b, err := s.requireBreakGlassTx(tx, net.ID)
		if err != nil {
			return err
		}
		person, _, err := tx.GetPersonByLogin(strings.TrimSpace(in.Login))
		if err != nil {
			return domain.NotFound("member")
		}
		m, err := tx.GetMemberByPerson(net.ID, person.ID)
		if err != nil {
			return domain.NotFound("member")
		}
		if m.Status != domain.MemberActive {
			return domain.InvalidState("only active members can receive roles")
		}
		if m.HasRole(domain.RoleAdmin) {
			return domain.InvalidState("member already holds admin")
		}
		live, err := tx.LiveBreakGlassRoles(net.ID, m.ID, s.now())
		if err != nil {
			return err
		}
		for _, r := range live {
			if r == domain.RoleAdmin {
				return domain.InvalidState("member already holds a break-glass admin grant")
			}
		}
		if err := tx.InsertBreakGlassGrant(&domain.BreakGlassGrant{ID: s.id(), SessionID: b.ID, NetworkID: net.ID,
			MemberID: m.ID, Role: domain.RoleAdmin, GrantedAt: s.now()}); err != nil {
			return err
		}
		out = b
		return s.auditBreakGlass(tx, actor, net.ID, auditEntry{Action: domain.ActBreakGlassRoleGranted, TargetType: "member",
			TargetID: m.ID, Reason: reason, Subjects: []string{m.ID},
			After: map[string]any{"role": string(domain.RoleAdmin), "session": b.ID, "expires_at": b.ExpiresAt.UnixMilli()}})
	})
	return out, err
}

// applyBreakGlassRoles adds the roles the member holds through live break-glass
// grants to the principal's member copy. It is the one place where role expiry
// is enforced: a grant whose session is closed or at/after its expiry adds
// nothing, on the very next call, with no job needed.
func (s *Service) applyBreakGlassRoles(tx store.Tx, m *domain.Member) error {
	roles, err := tx.LiveBreakGlassRoles(m.NetworkID, m.ID, s.now())
	if err != nil {
		return err
	}
	for _, r := range roles {
		if !m.HasRole(r) {
			m.Roles = append(m.Roles, r)
		}
	}
	return nil
}
