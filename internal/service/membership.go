package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// DirectoryEntry is what ordinary members may see of one another (FR-020,
// SEC-016): an id and a display name, nothing else.
type DirectoryEntry struct {
	MemberID    string
	DisplayName string
}

// ListDirectory lists active members to active members and administrators.
// Suspended, pending and departed members cannot browse the directory.
func (s *Service) ListDirectory(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]DirectoryEntry, error) {
	var out []DirectoryEntry
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		ms, err := tx.ListMembers(netID, store.MemberFilter{Statuses: []domain.MemberStatus{domain.MemberActive}, AfterID: pg.AfterID, Limit: clampPage(pg)})
		for _, m := range ms {
			out = append(out, DirectoryEntry{MemberID: m.ID, DisplayName: m.DisplayName})
		}
		return err
	})
	return out, err
}

// AdminListMembers lists members of any status with roles and flags (admin).
func (s *Service) AdminListMembers(ctx context.Context, actor domain.Actor, netID string, statuses []domain.MemberStatus, pg Page) ([]*domain.Member, error) {
	var out []*domain.Member
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = tx.ListMembers(netID, store.MemberFilter{Statuses: statuses, AfterID: pg.AfterID, Limit: clampPage(pg)})
		return err
	})
	return out, err
}

// adminOnMember loads the target and applies the shared guards: admin caller,
// target exists in this network, administrators never act on themselves (DOM-037).
func (s *Service) adminOnMember(tx store.Tx, actor domain.Actor, netID, memberID string, selfOK bool) (*principal, *domain.Member, error) {
	p, err := s.open(tx, actor, netID, domain.ScopeWrite)
	if err != nil {
		return nil, nil, err
	}
	if err := p.requireAdmin(); err != nil {
		return nil, nil, err
	}
	if !selfOK {
		if err := p.notSelf(memberID); err != nil {
			return nil, nil, err
		}
	}
	m, err := tx.GetMember(netID, memberID)
	if err != nil {
		return nil, nil, domain.NotFound("member")
	}
	return p, m, nil
}

// lastAdminGuard refuses removal of the only active administrator (FR-069, SEC-010).
func lastAdminGuard(tx store.Tx, netID string, target *domain.Member) error {
	if !target.HasRole(domain.RoleAdmin) || target.Status != domain.MemberActive {
		return nil
	}
	n, err := tx.CountActiveAdmins(netID)
	if err != nil {
		return err
	}
	if n <= 1 {
		return &domain.Error{Code: domain.CodeBlocked, Msg: "the network must keep at least one active administrator",
			Blockers: []string{"last active administrator"}}
	}
	return nil
}

// SuspendMember freezes a member (-> suspended) with a mandatory reason.
// Balances and history are untouched (FR-061, DOM-004, DOM-039).
func (s *Service) SuspendMember(ctx context.Context, actor domain.Actor, netID, memberID, reason string) error {
	return s.setMemberStatus(ctx, actor, netID, memberID, reason, domain.MemberSuspended)
}

// RestoreMember returns a suspended member to active.
func (s *Service) RestoreMember(ctx context.Context, actor domain.Actor, netID, memberID, reason string) error {
	return s.setMemberStatus(ctx, actor, netID, memberID, reason, domain.MemberActive)
}

func (s *Service) setMemberStatus(ctx context.Context, actor domain.Actor, netID, memberID, reason string, to domain.MemberStatus) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, m, err := s.adminOnMember(tx, actor, netID, memberID, false)
		if err != nil {
			return err
		}
		if to == domain.MemberSuspended {
			if err := p.requireLasting(); err != nil { // A3-02
				return err
			}
		}
		if err := requireReason(reason); err != nil {
			return err
		}
		if err := domain.CheckMemberTransition(m.Status, to); err != nil {
			return err
		}
		action := domain.ActMemberRestored
		if to == domain.MemberSuspended {
			action = domain.ActMemberSuspended
			if err := lastAdminGuard(tx, netID, m); err != nil {
				return err
			}
		}
		if err := tx.SetMemberStatus(netID, m.ID, to, nil); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: action, TargetType: "member", TargetID: m.ID,
			Reason: reason, Subjects: []string{m.ID}, Before: map[string]any{"status": m.Status}, After: map[string]any{"status": to}})
	})
}

// AssignRole grants admin or reviewer to an active member (FR-065, FR-069).
// Administrators cannot grant roles to themselves (SEC-010).
func (s *Service) AssignRole(ctx context.Context, actor domain.Actor, netID, memberID string, role domain.Role) error {
	return s.changeRole(ctx, actor, netID, memberID, role, true)
}

// RevokeRole removes a role. Removing the last active administrator is refused.
func (s *Service) RevokeRole(ctx context.Context, actor domain.Actor, netID, memberID string, role domain.Role) error {
	return s.changeRole(ctx, actor, netID, memberID, role, false)
}

func (s *Service) changeRole(ctx context.Context, actor domain.Actor, netID, memberID string, role domain.Role, grant bool) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, m, err := s.adminOnMember(tx, actor, netID, memberID, !grant)
		if err != nil {
			return err
		}
		if err := p.requireLasting(); err != nil {
			return err
		}
		if role != domain.RoleAdmin && role != domain.RoleReviewer {
			return domain.Invalid("role", "must be admin or reviewer")
		}
		now, action := s.now(), domain.ActRoleRevoked
		if grant {
			action = domain.ActRoleGranted
			if m.Status != domain.MemberActive {
				return domain.InvalidState("only active members can receive roles")
			}
			if m.HasRole(role) {
				return domain.InvalidState("member already holds %s", role)
			}
			if err := tx.GrantRole(s.id(), netID, m.ID, role, p.memberID(), now); err != nil {
				return err
			}
			if role == domain.RoleAdmin {
				if err := s.noticeAdminGranted(tx, netID, m, p); err != nil { // A3-01
					return err
				}
			}
		} else {
			if !m.HasRole(role) {
				return domain.InvalidState("member does not hold %s", role)
			}
			if role == domain.RoleAdmin {
				if err := lastAdminGuard(tx, netID, m); err != nil {
					return err
				}
			}
			if _, err := tx.RevokeRole(netID, m.ID, role, now); err != nil {
				return err
			}
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: action, TargetType: "member", TargetID: m.ID,
			Subjects: []string{m.ID}, After: map[string]any{"role": string(role)}})
	})
}

// SetDefaultFlag sets or clears the governance "default" marker (DOM-025,
// FR-064). It never posts entries or changes balances.
func (s *Service) SetDefaultFlag(ctx context.Context, actor domain.Actor, netID, memberID string, flag bool, reason string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, m, err := s.adminOnMember(tx, actor, netID, memberID, false)
		if err != nil {
			return err
		}
		if err := p.requireLasting(); err != nil { // A3-02
			return err
		}
		if err := requireReason(reason); err != nil {
			return err
		}
		if m.DefaultFlag == flag {
			return domain.InvalidState("default flag already %v", flag)
		}
		if err := tx.SetDefaultFlag(netID, m.ID, flag); err != nil {
			return err
		}
		action := domain.ActDefaultFlagCleared
		if flag {
			action = domain.ActDefaultFlagSet
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: action, TargetType: "member", TargetID: m.ID,
			Reason: reason, Subjects: []string{m.ID}, Before: map[string]any{"default_flag": m.DefaultFlag}, After: map[string]any{"default_flag": flag}})
	})
}

// SetLimitInput sets or removes a per-member credit-limit override.
type SetLimitInput struct {
	MemberID  string
	Limit     *int64     // nil removes the override (default applies again)
	ExpiresAt *time.Time // optional: time-limited override (DOM-021, OQ-DOM-2)
	Reason    string
}

// SetCreditLimit changes a member's override. It posts nothing: lowering a
// limit below the current negative balance leaves the balance as is and later
// debits are refused (DOM-032). Administrators cannot change their own limit.
func (s *Service) SetCreditLimit(ctx context.Context, actor domain.Actor, netID string, in SetLimitInput) error {
	return s.update(ctx, func(tx store.Tx) error {
		_, _, err := s.setCreditLimit(tx, actor, netID, in)
		return err
	})
}

// SetCreditLimitView is SetCreditLimit that also returns the member's position
// after the change, read in the same unit. Adapters use it to answer a limit
// PUT (and compute the new ETag) with ONE write and ONE audit event
// (limit.changed), instead of reading the balance through the audited admin
// read before and after (SVC-05).
func (s *Service) SetCreditLimitView(ctx context.Context, actor domain.Actor, netID string, in SetLimitInput) (*BalanceView, error) {
	var out *BalanceView
	err := s.update(ctx, func(tx store.Tx) error {
		p, m, err := s.setCreditLimit(tx, actor, netID, in)
		if err != nil {
			return err
		}
		out, err = s.balanceView(tx, p.net, m)
		return err
	})
	return out, err
}

func (s *Service) setCreditLimit(tx store.Tx, actor domain.Actor, netID string, in SetLimitInput) (*principal, *domain.Member, error) {
	p, m, err := s.adminOnMember(tx, actor, netID, in.MemberID, false)
	if err != nil {
		return nil, nil, err
	}
	if err := requireReason(in.Reason); err != nil {
		return nil, nil, err
	}
	if in.Limit != nil && (*in.Limit < 0 || *in.Limit > domain.MaxPolicyAmount) {
		return nil, nil, domain.Invalid("limit", "must be >= 0 and within the platform maximum")
	}
	if in.Limit == nil && in.ExpiresAt != nil {
		return nil, nil, domain.Invalid("expires_at", "only valid together with a limit")
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return nil, nil, domain.Invalid("expires_at", "must be in the future")
	}
	if m.Status != domain.MemberActive && m.Status != domain.MemberSuspended {
		return nil, nil, domain.InvalidState("member is %s", m.Status)
	}
	acc, err := tx.GetAccountByMember(netID, m.ID)
	if err != nil {
		return nil, nil, err
	}
	now := s.now()
	before := acc.EffectiveLimit(p.net.Policy.DefaultCreditLimit, now)
	if err := tx.SetLimitOverride(netID, acc.ID, in.Limit, in.ExpiresAt); err != nil {
		return nil, nil, err
	}
	after := p.net.Policy.DefaultCreditLimit
	if in.Limit != nil {
		after = *in.Limit
	}
	return p, m, s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActLimitChanged, TargetType: "account", TargetID: acc.ID,
		Reason: in.Reason, Subjects: []string{m.ID}, Before: map[string]any{"effective_limit": before},
		After: map[string]any{"effective_limit": after, "override": in.Limit != nil, "expires": in.ExpiresAt != nil}})
}

// noticeAdminGranted tells every other active administrator of the network
// that a new one was appointed, so a rogue appointment is visible (A3-01).
func (s *Service) noticeAdminGranted(tx store.Tx, netID string, grantee *domain.Member, p *principal) error {
	ms, err := tx.ListMembers(netID, store.MemberFilter{Statuses: []domain.MemberStatus{domain.MemberActive}, Limit: 500})
	if err != nil {
		return err
	}
	for _, o := range ms {
		if o.ID == grantee.ID || !o.HasRole(domain.RoleAdmin) {
			continue
		}
		if err := s.notify(tx, o.PersonID, domain.NoticeAdminGranted, p.net.Name+": a new administrator was appointed"); err != nil {
			return err
		}
	}
	return nil
}
