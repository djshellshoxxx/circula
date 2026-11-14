package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Dual control for administrator adjustments (SEC-011, DEC-013). A request
// above Policy.DualControlThreshold is recorded by CreateAdjustmentRequest and
// posts only when a different administrator approves it. See docs/dev/dual-control.md.

// DefaultNewAdminApprovalDelay is the wait before a freshly granted
// administrator may decide a dual-control request (A3-01).
const DefaultNewAdminApprovalDelay = 24 * time.Hour

// maxGrantChain bounds the walk up the chain of who made whom administrator.
const maxGrantChain = 32

// requireIndependentApprover enforces A3-01 for a member deciding a request:
// the decider's administrator role must not descend (directly or through other
// grants) from the proposer, and must be older than NewAdminApprovalDelay. One
// person cannot then satisfy "two administrators" with a puppet account.
func (s *Service) requireIndependentApprover(tx store.Tx, p *principal, pa *domain.PendingAdjustment) error {
	if p.bgAdmin { // no grant row: the role is a temporary break-glass one
		return nil
	}
	cur, first := p.memberID(), true
	seen := map[string]bool{}
	for i := 0; i < maxGrantChain && cur != "" && !seen[cur]; i++ {
		seen[cur] = true
		by, at, ok, err := tx.LatestAdminGrant(pa.NetworkID, cur)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if by == cur { // the founder's own creation-time grant is the root of trust: no wait
			break
		}
		if first && s.cfg.NewAdminApprovalDelay > 0 && s.now().Before(at.Add(s.cfg.NewAdminApprovalDelay)) {
			return domain.Forbidden("a newly granted administrator cannot decide adjustment requests for %s", s.cfg.NewAdminApprovalDelay)
		}
		first = false
		if by == pa.Proposer && cur != pa.Proposer {
			return domain.Forbidden("an administrator appointed by the requester cannot decide this adjustment request")
		}
		cur = by
	}
	return nil
}

// ApproveAdjustment posts a pending adjustment on the second administrator's
// approval. The approver must be an active administrator who is neither the
// proposer (SEC-010) nor the target member (DOM-037). Everything is validated
// again now (network active, target still a member, proposer still an active
// administrator, the member's limit), because the world may have changed since
// the request was made. Re-approving by the same administrator returns the
// original transaction. A request past its expiry is closed with an audit
// event and posts nothing.
func (s *Service) ApproveAdjustment(ctx context.Context, actor domain.Actor, netID, id string) (*domain.Transaction, error) {
	var out *domain.Transaction
	expired := false
	err := s.update(ctx, func(tx store.Tx) error {
		p, pa, err := s.openPending(tx, actor, netID, id)
		if err != nil {
			return err
		}
		if pa.Status == domain.PendingApproved && pa.DecidedBy == p.memberID() { // idempotent replay
			out, err = tx.GetTransaction(netID, pa.TxID)
			return err
		}
		if err := requireOpen(pa); err != nil {
			return err
		}
		if expired, err = s.closeIfExpired(tx, p, pa); err != nil || expired {
			return err // an expired close commits: nothing is posted but the closure is audited
		}
		if p.memberID() == pa.Proposer {
			return domain.Forbidden("the administrator who requested an adjustment cannot approve it")
		}
		if err := p.notSelf(pa.Target); err != nil { // DOM-037
			return err
		}
		if err := s.requireIndependentApprover(tx, p, pa); err != nil { // A3-01
			return err
		}
		if err := p.requireNetworkActive(); err != nil {
			return err
		}
		proposer, err := tx.GetMember(netID, pa.Proposer)
		if err != nil || proposer.Status != domain.MemberActive || !proposer.HasRole(domain.RoleAdmin) {
			return domain.InvalidState("the administrator who requested this adjustment is no longer an active administrator")
		}
		dir := AdjustDirection(pa.Direction)
		target, err := s.validateAdjustTarget(tx, netID, pa.Target, pa.Amount, dir)
		if err != nil {
			return err
		}
		t, err := s.postAdjustment(tx, p.net, target, pa.Amount, dir, pa.Reason, p.memberID(), p.actor.RequestID)
		if err != nil {
			return err
		}
		won, err := tx.DecidePendingAdjustment(netID, pa.ID, domain.PendingApproved, p.memberID(), s.now(), t.ID)
		if err != nil {
			return err
		}
		if !won { // cannot happen inside one serialised write unit; rolls the posting back if it ever does
			return domain.InvalidState("adjustment request is no longer pending")
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAdjustmentApproved,
			TargetType: "pending_adjustment", TargetID: pa.ID, Reason: pa.Reason, Subjects: []string{pa.Target},
			Before: map[string]any{"status": "pending"},
			After:  map[string]any{"status": "approved", "proposer": pa.Proposer, "approver": p.memberID(), "transaction": t.ID}}); err != nil {
			return err
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAdjustment, TargetType: "transaction",
			TargetID: t.ID, Reason: pa.Reason, Subjects: []string{pa.Target},
			After: map[string]any{"member": pa.Target, "direction": pa.Direction, "amount": pa.Amount, "status": "completed",
				"request": pa.ID, "proposer": pa.Proposer, "approver": p.memberID()}}); err != nil {
			return err
		}
		out = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, errExpired()
	}
	return out, nil
}

// RejectAdjustment closes a pending request without posting. The rejecter must
// be an administrator other than the proposer and the target member.
func (s *Service) RejectAdjustment(ctx context.Context, actor domain.Actor, netID, id string) (*domain.PendingAdjustment, error) {
	return s.decideNoPost(ctx, actor, netID, id, domain.PendingRejected)
}

// CancelAdjustment withdraws a pending request. Only the proposer may.
func (s *Service) CancelAdjustment(ctx context.Context, actor domain.Actor, netID, id string) (*domain.PendingAdjustment, error) {
	return s.decideNoPost(ctx, actor, netID, id, domain.PendingCancelled)
}

func (s *Service) decideNoPost(ctx context.Context, actor domain.Actor, netID, id string, to domain.PendingStatus) (*domain.PendingAdjustment, error) {
	var out *domain.PendingAdjustment
	expired := false
	err := s.update(ctx, func(tx store.Tx) error {
		p, pa, err := s.openPending(tx, actor, netID, id)
		if err != nil {
			return err
		}
		if pa.Status == to && pa.DecidedBy == p.memberID() { // idempotent replay
			out = pa
			return nil
		}
		if err := requireOpen(pa); err != nil {
			return err
		}
		if expired, err = s.closeIfExpired(tx, p, pa); err != nil || expired {
			return err
		}
		act := domain.ActAdjustmentRejected
		if to == domain.PendingCancelled {
			act = domain.ActAdjustmentCancelled
			if p.memberID() != pa.Proposer {
				return domain.Forbidden("only the administrator who requested an adjustment can cancel it")
			}
		} else {
			if p.memberID() == pa.Proposer {
				return domain.Forbidden("the administrator who requested an adjustment cannot reject it; cancel it instead")
			}
			if err := p.notSelf(pa.Target); err != nil {
				return err
			}
			if err := s.requireIndependentApprover(tx, p, pa); err != nil { // A3-01
				return err
			}
		}
		won, err := tx.DecidePendingAdjustment(netID, pa.ID, to, p.memberID(), s.now(), "")
		if err != nil {
			return err
		}
		if !won {
			return domain.InvalidState("adjustment request is no longer pending")
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: act, TargetType: "pending_adjustment",
			TargetID: pa.ID, Reason: pa.Reason, Subjects: []string{pa.Target},
			Before: map[string]any{"status": "pending"},
			After:  map[string]any{"status": string(to), "proposer": pa.Proposer, "decided_by": p.memberID()}}); err != nil {
			return err
		}
		out, err = tx.GetPendingAdjustment(netID, pa.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, errExpired()
	}
	return out, nil
}

// GetPendingAdjustment returns one request of the network to an administrator.
// A request of another network is not found (API-017).
func (s *Service) GetPendingAdjustment(ctx context.Context, actor domain.Actor, netID, id string) (*domain.PendingAdjustment, error) {
	var out *domain.PendingAdjustment
	err := s.view(ctx, func(tx store.Tx) error {
		_, pa, err := s.openPending(tx, actor, netID, id)
		out = pa
		return err
	})
	return out, err
}

// ListPendingAdjustments lists requests, newest first. Empty statuses means all.
// Requests past their expiry still read as pending until the expiry job or a
// decision attempt closes them; they cannot be approved.
func (s *Service) ListPendingAdjustments(ctx context.Context, actor domain.Actor, netID string, statuses []domain.PendingStatus, pg Page) ([]*domain.PendingAdjustment, error) {
	var out []*domain.PendingAdjustment
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = tx.ListPendingAdjustments(netID, statuses, pg.AfterID, pg.Limit)
		return err
	})
	return out, err
}

// ExpirePendingAdjustments closes open requests past their expiry, with an
// audit event each and no posting (SEC-011). System actors only. Idempotent.
// Returns the number closed in this run (at most limit, default 100).
func (s *Service) ExpirePendingAdjustments(ctx context.Context, actor domain.Actor, limit int) (int, error) {
	if !actor.System {
		return 0, domain.Forbidden("system actor required")
	}
	if limit <= 0 {
		limit = 100
	}
	n := 0
	err := s.update(ctx, func(tx store.Tx) error {
		rows, err := tx.ListExpiredPendingAdjustments(s.now(), limit)
		if err != nil {
			return err
		}
		for _, pa := range rows {
			if err := s.expirePending(tx, pa, actor.RequestID); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// openPending resolves the actor as a member administrator of the network and
// loads the request; a request of another network is not found.
func (s *Service) openPending(tx store.Tx, actor domain.Actor, netID, id string) (*principal, *domain.PendingAdjustment, error) {
	p, err := s.open(tx, actor, netID, domain.ScopeWrite)
	if err != nil {
		return nil, nil, err
	}
	if err := p.requireAdmin(); err != nil {
		return nil, nil, err
	}
	if p.member == nil { // system actors cannot decide
		return nil, nil, domain.Forbidden("a member administrator is required")
	}
	pa, err := tx.GetPendingAdjustment(netID, id)
	if err != nil {
		return nil, nil, domain.NotFound("adjustment request")
	}
	return p, pa, nil
}

func requireOpen(pa *domain.PendingAdjustment) error {
	if pa.Status != domain.PendingOpen {
		return domain.InvalidState("adjustment request is %s", pa.Status)
	}
	return nil
}

func errExpired() error {
	return domain.InvalidState("adjustment request has expired; nothing was posted")
}

// closeIfExpired closes an open request past its expiry inside the caller's
// unit, which must then commit (not roll back) so the closure is kept.
func (s *Service) closeIfExpired(tx store.Tx, p *principal, pa *domain.PendingAdjustment) (bool, error) {
	if s.now().Before(pa.ExpiresAt) {
		return false, nil
	}
	return true, s.expirePending(tx, pa, p.actor.RequestID)
}

func (s *Service) expirePending(tx store.Tx, pa *domain.PendingAdjustment, requestID string) error {
	won, err := tx.DecidePendingAdjustment(pa.NetworkID, pa.ID, domain.PendingExpired, "", s.now(), "")
	if err != nil || !won {
		return err
	}
	return s.emit(tx, pa.NetworkID, "", requestID, auditEntry{Type: domain.ActorSystem, Action: domain.ActAdjustmentExpired,
		TargetType: "pending_adjustment", TargetID: pa.ID, Reason: pa.Reason, Subjects: []string{pa.Target},
		Before: map[string]any{"status": "pending"}, After: map[string]any{"status": "expired", "proposer": pa.Proposer}})
}
