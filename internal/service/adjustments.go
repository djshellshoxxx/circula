package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// AdjustDirection says which way an adjustment moves the member's balance.
type AdjustDirection string

// Adjustment directions.
const (
	AdjustCredit AdjustDirection = "credit" // member balance goes up; system account down
	AdjustDebit  AdjustDirection = "debit"  // member balance goes down; system account up
)

// AdjustInput is an administrator correction against the system account.
type AdjustInput struct {
	MemberID       string
	Amount         int64 // minor units, > 0
	Direction      AdjustDirection
	Reason         string
	IdempotencyKey string `json:"-"`
}

// CreateAdjustment posts one entry on the member and the equal opposite entry
// on the system account, completed on creation (DOM-022, FR-062). Reason is
// mandatory; an administrator cannot adjust their own account (DOM-037); a
// debit that would breach the member's effective limit is refused (LED-5).
func (s *Service) CreateAdjustment(ctx context.Context, actor domain.Actor, netID string, in AdjustInput) (*domain.Transaction, error) {
	var out *domain.Transaction
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "adjustment.create", in.IdempotencyKey, in, func() (string, error) {
			if err := p.requireNetworkActive(); err != nil {
				return "", err
			}
			if err := p.notSelf(in.MemberID); err != nil {
				return "", err
			}
			if err := requireReason(in.Reason); err != nil {
				return "", err
			}
			if in.Amount <= 0 || in.Amount > domain.MaxPolicyAmount {
				return "", domain.Invalid("amount", "must be a positive amount within the platform maximum")
			}
			if in.Direction != AdjustCredit && in.Direction != AdjustDebit {
				return "", domain.Invalid("direction", "must be credit or debit")
			}
			target, err := tx.GetMember(netID, in.MemberID)
			if err != nil {
				return "", domain.NotFound("member")
			}
			if target.Status != domain.MemberActive && target.Status != domain.MemberSuspended {
				return "", domain.InvalidState("member is %s", target.Status)
			}
			t, err := s.postAdjustment(tx, p.net, target, in.Amount, in.Direction, in.Reason, p.memberID(), p.actor.RequestID)
			if err != nil {
				return "", err
			}
			return t.ID, s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAdjustment, TargetType: "transaction",
				TargetID: t.ID, Reason: in.Reason, IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{target.ID},
				After: map[string]any{"member": target.ID, "direction": string(in.Direction), "amount": in.Amount, "status": "completed"}})
		}, func(id string) (*domain.Transaction, error) { return tx.GetTransaction(netID, id) })
		return err
	})
	return out, err
}

// postAdjustment is shared by CreateAdjustment and departure settlement.
func (s *Service) postAdjustment(tx store.Tx, net *domain.Network, target *domain.Member, amount int64,
	dir AdjustDirection, reason, createdBy, requestID string) (*domain.Transaction, error) {

	sys, err := tx.GetSystemAccount(net.ID)
	if err != nil {
		return nil, err
	}
	acc, err := tx.GetAccountByMember(net.ID, target.ID)
	if err != nil {
		return nil, err
	}
	payer, payee := sys.ID, acc.ID
	if dir == AdjustDebit {
		payer, payee = acc.ID, sys.ID
	}
	t, err := s.newInternalTx(tx, net, domain.KindAdjustment, payer, payee, amount, reason, createdBy, "")
	if err != nil {
		return nil, err
	}
	if err := s.postTx(tx, net, t, createdBy, requestID); err != nil {
		return nil, err
	}
	return t, nil
}

// postSignupGrant posts the network's sign-up grant to a newly active member
// (DOM-027). It is a no-op when the grant is disabled. Grants are system
// actions; the audit event names the system as actor.
func (s *Service) postSignupGrant(tx store.Tx, net *domain.Network, m *domain.Member, requestID string) error {
	amt := net.Policy.SignupGrant
	if amt <= 0 {
		return nil
	}
	sys, err := tx.GetSystemAccount(net.ID)
	if err != nil {
		return err
	}
	acc, err := tx.GetAccountByMember(net.ID, m.ID)
	if err != nil {
		return err
	}
	t, err := s.newInternalTx(tx, net, domain.KindGrant, sys.ID, acc.ID, amt, "sign-up grant", "", "")
	if err != nil {
		return err
	}
	if err := s.postTx(tx, net, t, "", requestID); err != nil {
		return err
	}
	return s.emit(tx, net.ID, "", requestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActGrant,
		TargetType: "transaction", TargetID: t.ID, Reason: "sign-up grant", Subjects: []string{m.ID},
		After: map[string]any{"member": m.ID, "amount": amt, "status": "completed"}})
}
