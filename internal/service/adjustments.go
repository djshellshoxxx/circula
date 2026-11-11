package service

import (
	"context"
	"strings"

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

// AdjustResult is the outcome of CreateAdjustmentRequest: exactly one field is
// set. Tx when the adjustment was posted at once, Pending when it is waiting
// for a second administrator (SEC-011).
type AdjustResult struct {
	Tx      *domain.Transaction
	Pending *domain.PendingAdjustment
}

// CreateAdjustment posts one entry on the member and the equal opposite entry
// on the system account, completed on creation (DOM-022, FR-062). Reason is
// mandatory; an administrator cannot adjust their own account (DOM-037); a
// debit that would breach the member's effective limit is refused (LED-5).
//
// When the amount is above the network's dual-control threshold (SEC-011) this
// call fails safe with domain.CodeApprovalRequired and changes nothing; use
// CreateAdjustmentRequest, which records a pending request instead.
func (s *Service) CreateAdjustment(ctx context.Context, actor domain.Actor, netID string, in AdjustInput) (*domain.Transaction, error) {
	r, err := s.createAdjustment(ctx, actor, netID, in, false)
	if err != nil {
		return nil, err
	}
	return r.Tx, nil
}

// CreateAdjustmentRequest is CreateAdjustment with dual control (SEC-011): at or
// below the threshold (or when it is disabled) the adjustment posts at once;
// above it a pending request and an adjustment.pending audit event are recorded
// and nothing is posted until another administrator approves it.
func (s *Service) CreateAdjustmentRequest(ctx context.Context, actor domain.Actor, netID string, in AdjustInput) (*AdjustResult, error) {
	return s.createAdjustment(ctx, actor, netID, in, true)
}

// validateAdjustTarget checks the rules shared by proposal and approval.
func (s *Service) validateAdjustTarget(tx store.Tx, netID, memberID string, amount int64, dir AdjustDirection) (*domain.Member, error) {
	if amount <= 0 || amount > domain.MaxPolicyAmount {
		return nil, domain.Invalid("amount", "must be a positive amount within the platform maximum")
	}
	if dir != AdjustCredit && dir != AdjustDebit {
		return nil, domain.Invalid("direction", "must be credit or debit")
	}
	target, err := tx.GetMember(netID, memberID)
	if err != nil {
		return nil, domain.NotFound("member")
	}
	if target.Status != domain.MemberActive && target.Status != domain.MemberSuspended {
		return nil, domain.InvalidState("member is %s", target.Status)
	}
	return target, nil
}

func (s *Service) createAdjustment(ctx context.Context, actor domain.Actor, netID string, in AdjustInput, allowPending bool) (*AdjustResult, error) {
	var out *AdjustResult
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
			target, err := s.validateAdjustTarget(tx, netID, in.MemberID, in.Amount, in.Direction)
			if err != nil {
				return "", err
			}
			if th := p.net.Policy.DualControlThreshold; th > 0 && in.Amount > th {
				if !allowPending {
					return "", domain.Errf(domain.CodeApprovalRequired, "amounts above the dual-control threshold need a second administrator")
				}
				return s.proposeAdjustment(tx, p, target, in)
			}
			t, err := s.postAdjustment(tx, p.net, target, in.Amount, in.Direction, in.Reason, p.memberID(), p.actor.RequestID)
			if err != nil {
				return "", err
			}
			return "t:" + t.ID, s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAdjustment, TargetType: "transaction",
				TargetID: t.ID, Reason: in.Reason, IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{target.ID},
				After: map[string]any{"member": target.ID, "direction": string(in.Direction), "amount": in.Amount, "status": "completed"}})
		}, func(ref string) (*AdjustResult, error) {
			if id, ok := strings.CutPrefix(ref, "p:"); ok {
				pa, err := tx.GetPendingAdjustment(netID, id)
				return &AdjustResult{Pending: pa}, err
			}
			t, err := tx.GetTransaction(netID, strings.TrimPrefix(ref, "t:"))
			return &AdjustResult{Tx: t}, err
		})
		return err
	})
	return out, err
}

// proposeAdjustment records the pending request (SEC-011). With fewer than two
// active administrators no approver can exist, so it refuses rather than park
// a request nobody can decide.
func (s *Service) proposeAdjustment(tx store.Tx, p *principal, target *domain.Member, in AdjustInput) (string, error) {
	if p.member == nil {
		return "", domain.Forbidden("a member administrator is required to propose an adjustment")
	}
	n, err := tx.CountActiveAdmins(p.net.ID)
	if err != nil {
		return "", err
	}
	if n < 2 {
		return "", domain.InvalidState("dual control is on but the network has no second administrator to approve")
	}
	now := s.now()
	pa := &domain.PendingAdjustment{ID: s.id(), NetworkID: p.net.ID, Proposer: p.memberID(), Target: target.ID,
		Amount: in.Amount, Direction: string(in.Direction), Reason: in.Reason, Status: domain.PendingOpen,
		CreatedAt: now, ExpiresAt: now.Add(s.cfg.PendingAdjustmentTTL)}
	if err := tx.InsertPendingAdjustment(pa); err != nil {
		return "", err
	}
	return "p:" + pa.ID, s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAdjustmentPending,
		TargetType: "pending_adjustment", TargetID: pa.ID, Reason: in.Reason, IdemRef: keyRef(in.IdempotencyKey),
		Subjects: []string{target.ID},
		After: map[string]any{"member": target.ID, "direction": in.Direction, "amount": in.Amount, "status": "pending",
			"proposer": pa.Proposer, "expires_at": pa.ExpiresAt.UnixMilli()}})
}

// postAdjustment is shared by CreateAdjustment and departure settlement.
func (s *Service) postAdjustment(tx store.Tx, net *domain.Network, target *domain.Member, amount int64,
	dir AdjustDirection, reason, createdBy, requestID string) (*domain.Transaction, error) {

	if err := s.adminNotice(tx, net.ID, target.ID, NotifAdminAdjust, "An administrator adjusted your balance",
		"An administrator posted an adjustment to your account. See your history for the amount.", "/history"); err != nil {
		return nil, err
	}
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
