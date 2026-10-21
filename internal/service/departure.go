package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// DepartInput ends a membership, voluntarily or administratively.
type DepartInput struct {
	MemberID string // empty = the caller departs voluntarily
	Reason   string // mandatory when an administrator removes someone else
}

// DepartMember ends a membership (DOM-024, FR-014, FR-015). It is refused with
// a blocker list while the member is party to a `disputed` transaction or is the
// last active administrator. Pending proposals involving the member are
// auto-cancelled and audited. Per the network's departure rule the balance is
// retained on the closed account (default) or moved to the system account by
// an audited settlement adjustment. Roles are revoked; entries are retained
// and history stays exportable by the departed member.
func (s *Service) DepartMember(ctx context.Context, actor domain.Actor, netID string, in DepartInput) (*domain.Member, error) {
	var out *domain.Member
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.member == nil {
			return domain.Forbidden("a member is required")
		}
		target := p.member
		adminAction := in.MemberID != "" && in.MemberID != p.member.ID
		if adminAction {
			if err := p.requireAdmin(); err != nil {
				return err
			}
			if err := requireReason(in.Reason); err != nil {
				return err
			}
			if target, err = tx.GetMember(netID, in.MemberID); err != nil {
				return domain.NotFound("member")
			}
		} else if err := boundedText("reason", in.Reason, maxReasonLen); err != nil {
			return err
		}
		if target.Status != domain.MemberActive && target.Status != domain.MemberSuspended {
			return domain.InvalidState("member is %s", target.Status)
		}
		var blockers []string
		disputed, err := tx.ListTransactions(netID, store.TxFilter{MemberID: target.ID, Statuses: []domain.TxStatus{domain.TxDisputed}, Limit: 500})
		if err != nil {
			return err
		}
		for _, t := range disputed {
			blockers = append(blockers, "disputed transaction "+t.ID)
		}
		if err := lastAdminGuard(tx, netID, target); err != nil {
			if domain.CodeOf(err) != domain.CodeBlocked {
				return err
			}
			blockers = append(blockers, "last active administrator")
		}
		if len(blockers) > 0 {
			return &domain.Error{Code: domain.CodeBlocked, Msg: "departure is blocked", Blockers: blockers}
		}
		if err := s.cancelOpenProposals(tx, p, target); err != nil {
			return err
		}
		if err := s.settleOnDeparture(tx, p, target, adminAction); err != nil {
			return err
		}
		for _, r := range target.Roles {
			if _, err := tx.RevokeRole(netID, target.ID, r, s.now()); err != nil {
				return err
			}
		}
		now := s.now()
		if err := tx.SetMemberStatus(netID, target.ID, domain.MemberDeparted, &now); err != nil {
			return err
		}
		at := domain.ActorMember
		if adminAction {
			at = domain.ActorAdmin
		}
		if err := s.audit(tx, p, auditEntry{Type: at, Action: domain.ActMemberDeparted, TargetType: "member", TargetID: target.ID,
			Reason: in.Reason, Subjects: []string{target.ID}, Before: map[string]any{"status": target.Status},
			After: map[string]any{"status": domain.MemberDeparted, "rule": string(p.net.Policy.DepartureRule)}}); err != nil {
			return err
		}
		out, err = tx.GetMember(netID, target.ID)
		return err
	})
	return out, err
}

func (s *Service) cancelOpenProposals(tx store.Tx, p *principal, target *domain.Member) error {
	props, err := tx.ListTransactions(p.net.ID, store.TxFilter{MemberID: target.ID, Statuses: []domain.TxStatus{domain.TxProposed}, Limit: 500})
	if err != nil {
		return err
	}
	for _, t := range props {
		if err := s.transition(tx, p.net.ID, t, domain.CmdCancel, "", p.actor.RequestID); err != nil {
			return err
		}
		if err := s.emit(tx, p.net.ID, "", p.actor.RequestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActTxCancelled,
			TargetType: "transaction", TargetID: t.ID, Reason: "party departed",
			Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
			Before:   map[string]any{"status": "proposed"}, After: map[string]any{"status": "cancelled"}}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) settleOnDeparture(tx store.Tx, p *principal, target *domain.Member, adminAction bool) error {
	if p.net.Policy.DepartureRule != domain.DepartSettleToZero {
		return nil
	}
	acc, err := tx.GetAccountByMember(p.net.ID, target.ID)
	if err != nil || acc.Balance == 0 {
		return err
	}
	dir, amt := AdjustCredit, -acc.Balance
	if acc.Balance > 0 {
		dir, amt = AdjustDebit, acc.Balance
	}
	t, err := s.postAdjustment(tx, p.net, target, amt, dir, "departure settlement", p.memberID(), p.actor.RequestID)
	if err != nil {
		return err
	}
	at := domain.ActorMember
	if adminAction {
		at = domain.ActorAdmin
	}
	return s.audit(tx, p, auditEntry{Type: at, Action: domain.ActAdjustment, TargetType: "transaction", TargetID: t.ID,
		Reason: "departure settlement", Subjects: []string{target.ID},
		After: map[string]any{"member": target.ID, "direction": string(dir), "amount": amt, "status": "completed"}})
}
