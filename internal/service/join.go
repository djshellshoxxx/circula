package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const inviteTokenPrefix = "inv_"

// InviteInput creates a single-use invitation.
type InviteInput struct {
	TTL time.Duration // default 7 days, max 90 days
}

// CreateInvitation returns the one-time token (shown once; only its hash is
// stored) and the invitation record (FR-010, FR-063).
func (s *Service) CreateInvitation(ctx context.Context, actor domain.Actor, netID string, in InviteInput) (string, *domain.Invitation, error) {
	ttl := in.TTL
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}
	if ttl < time.Minute || ttl > 90*24*time.Hour {
		return "", nil, domain.Invalid("ttl", "must be between 1 minute and 90 days")
	}
	raw, err := randToken(32)
	if err != nil {
		return "", nil, err
	}
	token := inviteTokenPrefix + raw
	var inv *domain.Invitation
	err = s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if p.net.Status == domain.NetworkClosed {
			return domain.InvalidState("network is closed")
		}
		now := s.now()
		inv = &domain.Invitation{ID: s.id(), NetworkID: netID, CreatedBy: p.memberID(), CreatedAt: now, ExpiresAt: now.Add(ttl)}
		if err := tx.InsertInvitation(inv, hashToken(token)); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActInviteCreated, TargetType: "invitation", TargetID: inv.ID})
	})
	if err != nil {
		return "", nil, err
	}
	return token, inv, nil
}

// RevokeInvitation voids an unused invitation (FR-063).
func (s *Service) RevokeInvitation(ctx context.Context, actor domain.Actor, netID, invID string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		ok, err := tx.RevokeInvitation(netID, invID, s.now())
		if err != nil {
			return err
		}
		if !ok {
			return domain.NotFound("invitation")
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActInviteRevoked, TargetType: "invitation", TargetID: invID})
	})
}

// ListInvitations lists the network's invitations (admin).
func (s *Service) ListInvitations(ctx context.Context, actor domain.Actor, netID string) ([]*domain.Invitation, error) {
	var out []*domain.Invitation
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = tx.ListInvitations(netID)
		return err
	})
	return out, err
}

// createMember inserts a member with its account and, when active, the
// sign-up grant. The caller audits.
func (s *Service) createMember(tx store.Tx, net *domain.Network, personID, name string, status domain.MemberStatus, method, requestID string) (*domain.Member, error) {
	if err := s.checkNotFrozen(net.ID); err != nil {
		return nil, err
	}
	m := &domain.Member{ID: s.id(), NetworkID: net.ID, PersonID: personID, DisplayName: name,
		Status: status, JoinMethod: method, JoinedAt: s.now()}
	if err := tx.InsertMember(m); err != nil {
		return nil, err
	}
	m.AccountID = s.id()
	if err := tx.InsertAccount(&domain.Account{ID: m.AccountID, NetworkID: net.ID, MemberID: m.ID}); err != nil {
		return nil, err
	}
	if status == domain.MemberActive {
		if err := s.postSignupGrant(tx, net, m, requestID); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func joinableActor(actor domain.Actor) error {
	if actor.MustChangePassword {
		return domain.PasswordChangeRequired() // SEC-008
	}
	if actor.PersonID == "" || !actor.Can(domain.ScopeWrite) || actor.NetworkScope != "" {
		return domain.Forbidden("a full-scope person is required to join a network")
	}
	return nil
}

// existingMembership is the outcome of a join attempt by a person who already
// has a membership (DOM-002, FR-017, SVC-04). A pending or active membership is
// returned unchanged (idempotent join). A suspended or departed one is NOT a
// success: DOM-003 has no transition out of `departed` (denied, withdrawn,
// left and removed members alike) and a suspended member is under
// administrative control, so the join is refused with forbidden and nothing is
// changed. Re-application after denial or removal is therefore not supported
// (recorded decision; reinstating a departed person would need a new DOM-003
// transition).
func existingMembership(ex *domain.Member) (*domain.Member, error) {
	switch ex.Status {
	case domain.MemberPending, domain.MemberActive:
		return ex, nil
	case domain.MemberSuspended:
		return nil, domain.Forbidden("your membership in this network is suspended; contact an administrator")
	default:
		return nil, domain.Forbidden("your membership in this network has ended and cannot be re-opened by joining again")
	}
}

// RedeemInvitation joins the caller to the invitation's network as an active
// member. Unknown, expired, used and revoked tokens are indistinguishable
// (SEC-005). A person who already has a pending or active membership gets it
// back unchanged and the token is not consumed (DOM-002, FR-017); a suspended
// or departed one is refused (see existingMembership).
func (s *Service) RedeemInvitation(ctx context.Context, actor domain.Actor, token, displayName string) (*domain.Member, error) {
	return s.RedeemInvitationTerms(ctx, actor, token, displayName, 0)
}

// RedeemInvitationTerms is RedeemInvitation for a network that publishes
// terms: termsVersion is the version the person accepted, which must be the
// current one (GOV-009). It is ignored when the network has no terms.
func (s *Service) RedeemInvitationTerms(ctx context.Context, actor domain.Actor, token, displayName string, termsVersion int) (*domain.Member, error) {
	if err := joinableActor(actor); err != nil {
		return nil, err
	}
	name, err := cleanName(displayName)
	if err != nil {
		return nil, err
	}
	var out *domain.Member
	err = s.update(ctx, func(tx store.Tx) error {
		inv, err := tx.GetInvitationByHash(hashToken(token))
		now := s.now()
		if err != nil {
			return domain.NotFound("invitation")
		}
		// A retry by the person who already redeemed it, or a redemption by an
		// existing member, returns that membership unchanged (DOM-002, FR-017).
		if ex, err := tx.GetMemberByPerson(inv.NetworkID, actor.PersonID); err == nil && (inv.UsedAt == nil || inv.UsedBy == ex.ID) {
			out, err = existingMembership(ex)
			return err
		}
		if inv.UsedAt != nil || inv.RevokedAt != nil || !now.Before(inv.ExpiresAt) {
			return domain.NotFound("invitation")
		}
		net, err := tx.GetNetwork(inv.NetworkID)
		if err != nil {
			return domain.NotFound("invitation")
		}
		if err := networkOpenErr(net); err != nil {
			return domain.InvalidState("%s; not accepting members", err.Error())
		}
		record, err := s.joinTerms(tx, net.ID, termsVersion)
		if err != nil {
			return err
		}
		m, err := s.createMember(tx, net, actor.PersonID, name, domain.MemberActive, "invite", actor.RequestID)
		if err != nil {
			return err
		}
		if err := record(m.ID); err != nil {
			return err
		}
		if err := tx.MarkInvitationUsed(inv.ID, m.ID, now); err != nil {
			return err
		}
		out = m
		return s.emit(tx, net.ID, m.ID, actor.RequestID, auditEntry{Type: domain.ActorMember, Action: domain.ActInviteRedeemed,
			TargetType: "member", TargetID: m.ID, Subjects: []string{m.ID}, After: map[string]any{"status": m.Status, "invitation": inv.ID}})
	})
	return out, err
}

// JoinNetwork joins an approval-mode network (-> pending) or open-mode network
// (-> active). Invite-mode networks refuse it. An existing pending or active
// membership is returned unchanged (FR-017); a suspended or departed one is
// refused (see existingMembership).
func (s *Service) JoinNetwork(ctx context.Context, actor domain.Actor, netID, displayName string) (*domain.Member, error) {
	return s.JoinNetworkTerms(ctx, actor, netID, displayName, 0)
}

// JoinNetworkTerms is JoinNetwork with the terms version the person accepted
// (GOV-009); see RedeemInvitationTerms.
func (s *Service) JoinNetworkTerms(ctx context.Context, actor domain.Actor, netID, displayName string, termsVersion int) (*domain.Member, error) {
	if err := joinableActor(actor); err != nil {
		return nil, err
	}
	name, err := cleanName(displayName)
	if err != nil {
		return nil, err
	}
	var out *domain.Member
	err = s.update(ctx, func(tx store.Tx) error {
		net, err := tx.GetNetwork(netID)
		if err != nil {
			return domain.NotFound("network")
		}
		if ex, err := tx.GetMemberByPerson(netID, actor.PersonID); err == nil {
			out, err = existingMembership(ex)
			return err
		}
		if err := networkOpenErr(net); err != nil {
			return domain.InvalidState("%s; not accepting members", err.Error())
		}
		status, action := domain.MemberPending, domain.ActMemberRequested
		switch net.Policy.JoinMode {
		case domain.JoinOpen:
			status, action = domain.MemberActive, domain.ActMemberJoined
		case domain.JoinApproval:
		default:
			return domain.Forbidden("this network is invitation-only")
		}
		record, err := s.joinTerms(tx, net.ID, termsVersion)
		if err != nil {
			return err
		}
		m, err := s.createMember(tx, net, actor.PersonID, name, status, string(net.Policy.JoinMode), actor.RequestID)
		if err != nil {
			return err
		}
		if err := record(m.ID); err != nil {
			return err
		}
		out = m
		return s.emit(tx, netID, m.ID, actor.RequestID, auditEntry{Type: domain.ActorMember, Action: action,
			TargetType: "member", TargetID: m.ID, Subjects: []string{m.ID}, After: map[string]any{"status": status}})
	})
	return out, err
}

// DecideJoinRequest approves (-> active) or denies (-> departed) a pending
// member (FR-011). The denial reason, if any, is stored in the audit event.
func (s *Service) DecideJoinRequest(ctx context.Context, actor domain.Actor, netID, memberID string, approve bool, reason string) (*domain.Member, error) {
	var out *domain.Member
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := p.notSelf(memberID); err != nil {
			return err
		}
		if err := boundedText("reason", reason, maxReasonLen); err != nil {
			return err
		}
		m, err := tx.GetMember(netID, memberID)
		if err != nil {
			return domain.NotFound("member")
		}
		to, action := domain.MemberDeparted, domain.ActMemberDenied
		if approve {
			to, action = domain.MemberActive, domain.ActMemberApproved
			if err := p.requireNetworkActive(); err != nil {
				return err
			}
		}
		if m.Status != domain.MemberPending {
			return domain.InvalidState("member is %s, not pending", m.Status)
		}
		if err := domain.CheckMemberTransition(m.Status, to); err != nil {
			return err
		}
		var dep *time.Time
		if !approve {
			t := s.now()
			dep = &t
		}
		if err := tx.SetMemberStatus(netID, m.ID, to, dep); err != nil {
			return err
		}
		if approve {
			if err := s.postSignupGrant(tx, p.net, m, p.actor.RequestID); err != nil {
				return err
			}
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: action, TargetType: "member", TargetID: m.ID,
			Reason: reason, Subjects: []string{m.ID}, Before: map[string]any{"status": m.Status}, After: map[string]any{"status": to}}); err != nil {
			return err
		}
		out, err = tx.GetMember(netID, memberID)
		return err
	})
	return out, err
}
