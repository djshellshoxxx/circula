package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

var disputeCategories = map[string]bool{"not_delivered": true, "quality": true, "wrong_amount": true, "unauthorised": true, "other": true}

// OpenDisputeInput contests a completed trade (FR-070).
type OpenDisputeInput struct {
	TxID           string
	Category       string // not_delivered | quality | wrong_amount | unauthorised | other
	Statement      string
	IdempotencyKey string `json:"-"`
}

// OpenDispute lets a party to a completed trade contest it within the network's
// dispute window, measured from completion (DOM-017). It sets the transaction
// to `disputed` and creates no entries (FR-071). Suspended members cannot open
// disputes (DOM-039). Only trades can be disputed.
func (s *Service) OpenDispute(ctx context.Context, actor domain.Actor, netID string, in OpenDisputeInput) (*domain.Dispute, error) {
	var out *domain.Dispute
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "dispute.open", in.IdempotencyKey, in, func() (string, error) {
			if err := p.requireNetworkActive(); err != nil {
				return "", err
			}
			if !disputeCategories[in.Category] {
				return "", domain.Invalid("category", "must be one of not_delivered, quality, wrong_amount, unauthorised, other")
			}
			if err := boundedText("statement", in.Statement, maxTextLen); err != nil {
				return "", err
			}
			t, err := loadParty(tx, p, in.TxID)
			if err != nil {
				return "", err
			}
			if t.Kind != domain.KindTrade {
				return "", domain.InvalidState("only trades can be disputed")
			}
			if _, err := domain.NextTxStatus(t.Status, domain.CmdOpenDispute); err != nil {
				return "", err
			}
			now := s.now()
			closes := t.CompletedAt.Add(time.Duration(p.net.Policy.DisputeWindowDays) * 24 * time.Hour)
			if now.After(closes) {
				return "", domain.InvalidState("the dispute window closed on %s", closes.Format("2006-01-02"))
			}
			d := &domain.Dispute{ID: s.id(), NetworkID: netID, TxID: t.ID, OpenedBy: p.member.ID, Category: in.Category,
				Statement: in.Statement, Status: domain.DisputeOpen, OpenedAt: now}
			if err := tx.InsertDispute(d); err != nil {
				if domain.CodeOf(err) == domain.CodeConflict {
					return "", domain.InvalidState("a dispute is already open on this transaction")
				}
				return "", err
			}
			if err := s.transition(tx, netID, t, domain.CmdOpenDispute, p.member.ID, p.actor.RequestID); err != nil {
				return "", err
			}
			if err := s.disputeEvent(tx, d, p.member.ID, "opened"); err != nil {
				return "", err
			}
			return d.ID, s.audit(tx, p, auditEntry{Action: domain.ActDisputeOpened, TargetType: "dispute", TargetID: d.ID,
				IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
				After: map[string]any{"transaction": t.ID, "category": in.Category, "status": "open"}})
		}, func(id string) (*domain.Dispute, error) { return tx.GetDispute(netID, id) })
		return err
	})
	return out, err
}

func (s *Service) disputeEvent(tx store.Tx, d *domain.Dispute, actorID, action string) error {
	return tx.InsertDisputeEvent(store.DisputeEvent{ID: s.id(), NetworkID: d.NetworkID, DisputeID: d.ID,
		ActorID: actorID, Action: action, Outcome: string(d.Outcome), OccurredAt: s.now()})
}

// independent enforces reviewer independence (DOM-038, FR-072, DOM-037): the
// decider must not be a party, nor have proposed, accepted or reversed the
// transaction. The refusal is a conflict-of-interest forbidden error.
func independent(tx store.Tx, netID string, t *domain.Transaction, memberID string) error {
	conflict := domain.Forbidden("conflict of interest: you took part in this transaction")
	if t.IsParty(memberID) || t.CreatedBy == memberID || t.AcceptedBy == memberID {
		return conflict
	}
	revs, err := tx.ListTransactions(netID, store.TxFilter{OriginalTxID: t.ID, Limit: 500})
	if err != nil {
		return err
	}
	for _, r := range revs {
		if r.CreatedBy == memberID {
			return conflict
		}
	}
	return nil
}

// decideSetup loads dispute and transaction and applies the decider guards.
func (s *Service) decideSetup(tx store.Tx, actor domain.Actor, netID, disputeID string) (*principal, *domain.Dispute, *domain.Transaction, error) {
	p, err := s.open(tx, actor, netID, domain.ScopeWrite)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := p.requireDecider(); err != nil {
		return nil, nil, nil, err
	}
	d, err := tx.GetDispute(netID, disputeID)
	if err != nil {
		return nil, nil, nil, domain.NotFound("dispute")
	}
	t, err := tx.GetTransaction(netID, d.TxID)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := independent(tx, netID, t, p.member.ID); err != nil {
		return nil, nil, nil, err
	}
	return p, d, t, nil
}

func (p *principal) deciderType() domain.ActorType {
	if p.member != nil && p.member.HasRole(domain.RoleReviewer) {
		return domain.ActorReviewer
	}
	return p.actorType(domain.ActorAdmin)
}

// StartDisputeReview takes an open dispute into review (open -> under_review).
func (s *Service) StartDisputeReview(ctx context.Context, actor domain.Actor, netID, disputeID string) (*domain.Dispute, error) {
	var out *domain.Dispute
	err := s.update(ctx, func(tx store.Tx) error {
		p, d, t, err := s.decideSetup(tx, actor, netID, disputeID)
		if err != nil {
			return err
		}
		next, err := domain.NextDisputeStatus(d.Status, domain.DCmdStartReview)
		if err != nil {
			return err
		}
		d.Status, d.ReviewerID = next, p.member.ID
		if err := tx.UpdateDispute(d); err != nil {
			return err
		}
		if err := s.disputeEvent(tx, d, p.member.ID, "under_review"); err != nil {
			return err
		}
		out = d
		return s.audit(tx, p, auditEntry{Type: p.deciderType(), Action: domain.ActDisputeReview, TargetType: "dispute", TargetID: d.ID,
			Subjects: []string{t.PayerMemberID, t.PayeeMemberID}, After: map[string]any{"status": next}})
	})
	return out, err
}

// WithdrawDispute lets the opener withdraw before a decision (FR-075); the
// transaction returns to `completed` and no entries are created. Allowed for
// suspended openers, as an exit that adds no obligation.
func (s *Service) WithdrawDispute(ctx context.Context, actor domain.Actor, netID, disputeID string) (*domain.Dispute, error) {
	var out *domain.Dispute
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberSuspended) {
			return domain.Forbidden("membership cannot perform this action")
		}
		d, err := tx.GetDispute(netID, disputeID)
		if err != nil || d.OpenedBy != p.member.ID {
			return domain.NotFound("dispute")
		}
		next, err := domain.NextDisputeStatus(d.Status, domain.DCmdWithdraw)
		if err != nil {
			return err
		}
		t, err := tx.GetTransaction(netID, d.TxID)
		if err != nil {
			return err
		}
		if err := s.transition(tx, netID, t, domain.CmdCloseDispute, p.member.ID, p.actor.RequestID); err != nil {
			return err
		}
		d.Status = next
		if err := tx.UpdateDispute(d); err != nil {
			return err
		}
		if err := s.disputeEvent(tx, d, p.member.ID, "withdrawn"); err != nil {
			return err
		}
		out = d
		return s.audit(tx, p, auditEntry{Action: domain.ActDisputeWithdrawn, TargetType: "dispute", TargetID: d.ID,
			Subjects: []string{t.PayerMemberID, t.PayeeMemberID}, After: map[string]any{"status": next}})
	})
	return out, err
}
