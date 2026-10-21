package service

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// principal is the authorised view of a call: the network and the acting
// member with roles, resolved server-side on every request (SEC-009). The
// caller never supplies roles. System actors have no member.
type principal struct {
	actor  domain.Actor
	net    *domain.Network
	member *domain.Member // nil for System actors
	// mfaMissing: Config.RequireMFAForAdmins is on, the member is an
	// administrator and has no second factor (enforced in requireAdmin).
	mfaMissing bool
	// bgAdmin: the administrator role comes only from a live break-glass grant
	// (SEC-013). Such access must not leave lasting changes behind (SA2-04).
	bgAdmin bool
}

// requireLasting refuses actions whose effect outlives a break-glass session
// (role changes, policy changes) when the caller is admin only through one.
func (p *principal) requireLasting() error {
	if p.bgAdmin {
		return domain.Forbidden("break-glass access cannot make lasting changes")
	}
	return nil
}

// open resolves actor + network. Anyone who is not a member of the network,
// or whose token is scoped to another network, gets the same not-found as for
// a non-existent network (API-017, FR-006).
//
// A write-scope open on a frozen network fails with network-frozen (CMD-07),
// after membership is established so outsiders learn nothing.
func (s *Service) open(tx store.Tx, actor domain.Actor, netID string, scope domain.Scope) (*principal, error) {
	if actor.MustChangePassword && scope == domain.ScopeWrite {
		return nil, domain.PasswordChangeRequired() // SEC-008
	}
	p, err := s.openAny(tx, actor, netID, scope)
	if err != nil {
		return nil, err
	}
	if scope == domain.ScopeWrite {
		if err := s.checkNotFrozen(netID); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// openAny is open without the frozen-network check; only the consistency-run
// record uses it, so a frozen network's verification stays auditable.
func (s *Service) openAny(tx store.Tx, actor domain.Actor, netID string, scope domain.Scope) (*principal, error) {
	if !actor.Can(scope) {
		return nil, domain.Forbidden("token lacks %s scope", scope)
	}
	if actor.NetworkScope != "" && actor.NetworkScope != netID {
		return nil, domain.NotFound("network")
	}
	net, err := tx.GetNetwork(netID)
	if err != nil {
		return nil, domain.NotFound("network")
	}
	p := &principal{actor: actor, net: net}
	if actor.System {
		return p, nil
	}
	if actor.PersonID == "" {
		return nil, domain.ErrUnauthenticated
	}
	m, err := tx.GetMemberByPerson(netID, actor.PersonID)
	if err != nil {
		return nil, domain.NotFound("network")
	}
	p.member = m
	hadAdmin := m.HasRole(domain.RoleAdmin)
	if err := s.applyBreakGlassRoles(tx, m); err != nil { // SEC-013: temporary roles end with their session
		return nil, err
	}
	p.bgAdmin = !hadAdmin && m.HasRole(domain.RoleAdmin)
	if s.cfg.RequireMFAForAdmins && m.HasRole(domain.RoleAdmin) {
		on, err := s.hasMFA(tx, actor.PersonID)
		if err != nil {
			return nil, err
		}
		p.mfaMissing = !on
	}
	return p, nil
}

func (p *principal) memberID() string {
	if p.member == nil {
		return ""
	}
	return p.member.ID
}

// requireActive: the member is active (DOM-039: suspended/pending/departed
// members cannot add obligations).
func (p *principal) requireActive() error {
	if p.member == nil || p.member.Status != domain.MemberActive {
		return domain.Forbidden("membership is not active")
	}
	return nil
}

// requireNetworkActive: ledger-affecting operations need an active network (FR-004, FR-068).
func (p *principal) requireNetworkActive() error {
	return networkOpenErr(p.net)
}

// networkOpenErr refuses trading and membership changes unless the network is
// active and not suspended (GOV-023, GOV-024).
func networkOpenErr(n *domain.Network) error {
	if n.Status != domain.NetworkActive {
		return domain.InvalidState("network is %s", n.Status)
	}
	if n.Suspended() {
		return domain.InvalidState("network is suspended: trading and posting are paused; balances, history and exports stay available")
	}
	return nil
}

// requireAdmin: active member holding the admin role (or a System actor).
func (p *principal) requireAdmin() error {
	if p.actor.System {
		return nil
	}
	if err := p.requireActive(); err != nil {
		return err
	}
	if !p.member.HasRole(domain.RoleAdmin) {
		return domain.Forbidden("administrator role required")
	}
	if p.mfaMissing {
		return domain.Forbidden("mfa-enrolment-required")
	}
	return nil
}

// requireDecider: admin or reviewer.
func (p *principal) requireDecider() error {
	if err := p.requireActive(); err != nil {
		return err
	}
	if !p.member.HasRole(domain.RoleAdmin) && !p.member.HasRole(domain.RoleReviewer) {
		return domain.Forbidden("reviewer or administrator role required")
	}
	return nil
}

// notSelf enforces DOM-037 for administrator actions on their own member.
func (p *principal) notSelf(memberID string) error {
	if p.member != nil && p.member.ID == memberID {
		return domain.Forbidden("administrators cannot perform this action on themselves")
	}
	return nil
}

// actorType labels audit events with the role the actor used.
func (p *principal) actorType(role domain.ActorType) domain.ActorType {
	if p.actor.System {
		return domain.ActorSystem
	}
	return role
}
