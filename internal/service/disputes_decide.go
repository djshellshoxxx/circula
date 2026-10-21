package service

import (
	"context"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ResolveInput records a reviewer decision (FR-073).
type ResolveInput struct {
	DisputeID string
	Outcome   domain.DisputeOutcome
	Amount    int64 // partially_upheld only: minor units to reverse
	Rationale string
	// WaiveResponse lets the reviewer decide before the response period ends and
	// without a response (FR-076); WaiveReason is then required and audited.
	WaiveResponse  bool
	WaiveReason    string
	IdempotencyKey string `json:"-"`
}

// ResolveDispute records dismissed / upheld / partially_upheld with a mandatory
// rationale and the decider's identity (DOM-018). Reversal and dispute state
// change commit in one unit. If the reversal would breach LED-5 on the debited
// account, nothing is posted, the dispute becomes `posting_pending`, the
// transaction stays `disputed` and the call succeeds with that state
// (DOM-021, FR-074). Resolve the block with SetCreditLimit (time-limited
// override) and RetryDisputePosting. The caller must be a reviewer or
// administrator who took no part in the transaction (DOM-038).
func (s *Service) ResolveDispute(ctx context.Context, actor domain.Actor, netID string, in ResolveInput) (*domain.Dispute, error) {
	var out *domain.Dispute
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireDecider(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "dispute.resolve", in.IdempotencyKey, in, func() (string, error) {
			return s.resolve(tx, p, in)
		}, func(id string) (*domain.Dispute, error) { return tx.GetDispute(netID, id) })
		return err
	})
	return out, err
}

func (s *Service) resolve(tx store.Tx, p *principal, in ResolveInput) (string, error) {
	netID := p.net.ID
	d, err := tx.GetDispute(netID, in.DisputeID)
	if err != nil {
		return "", domain.NotFound("dispute")
	}
	t, err := tx.GetTransaction(netID, d.TxID)
	if err != nil {
		return "", err
	}
	if err := independent(tx, netID, t, p.member.ID); err != nil {
		return "", err
	}
	if d.OpenedBy == p.member.ID { // implied by independence; explicit for clarity
		return "", domain.Forbidden("the opener cannot decide")
	}
	if strings.TrimSpace(in.Rationale) == "" {
		return "", domain.Invalid("rationale", "required")
	}
	if err := boundedText("rationale", in.Rationale, maxTextLen); err != nil {
		return "", err
	}
	waived, err := s.responseGate(tx, p, d, in)
	if err != nil {
		return "", err
	}
	remaining := t.Amount - t.ReversedTotal
	amount := int64(0)
	switch in.Outcome {
	case domain.OutcomeDismissed:
		if in.Amount != 0 {
			return "", domain.Invalid("amount", "not allowed when dismissing")
		}
	case domain.OutcomeUpheld:
		if in.Amount != 0 && in.Amount != remaining {
			return "", domain.Invalid("amount", "an upheld dispute reverses the full remaining amount; use partially_upheld")
		}
		amount = remaining
	case domain.OutcomePartiallyUpheld:
		if in.Amount <= 0 || in.Amount > remaining {
			return "", domain.Invalid("amount", "must be between 1 and the remaining reversible amount "+fmtAmt(p.net, remaining))
		}
		amount = in.Amount
	default:
		return "", domain.Invalid("outcome", "must be dismissed, upheld or partially_upheld")
	}
	if amount > 0 && remaining == 0 {
		return "", domain.InvalidState("nothing left to reverse")
	}
	if _, err := domain.NextDisputeStatus(d.Status, domain.DCmdResolve); err != nil {
		return "", err // also refuses posting_pending (use RetryDisputePosting) and closed disputes
	}
	now := s.now()
	d.Outcome, d.ReversalAmount, d.ReviewerID, d.Rationale, d.DecidedAt = in.Outcome, amount, p.member.ID, in.Rationale, &now

	cmd, action := domain.DCmdResolve, domain.ActDisputeResolved
	if amount > 0 {
		_, err := s.reverse(tx, p.net, t, amount, "dispute "+d.ID+": "+string(in.Outcome), p.member.ID, p.actor.RequestID)
		switch {
		case blockedByLimit(err):
			cmd, action = domain.DCmdBlock, domain.ActDisputeBlocked
		case err != nil:
			return "", err
		}
	}
	if cmd == domain.DCmdResolve && t.Status == domain.TxDisputed {
		if err := s.transition(tx, netID, t, domain.CmdCloseDispute, p.member.ID, p.actor.RequestID); err != nil {
			return "", err
		}
	}
	if d.Status, err = domain.NextDisputeStatus(d.Status, cmd); err != nil {
		return "", err
	}
	return d.ID, s.finishDecision(tx, p, d, t, action, in.IdempotencyKey, waived)
}

// responseGate enforces FR-076: a decision needs a submitted response, an
// elapsed response period (network policy), or a reviewer waiver with a reason.
// It returns the waiver reason actually relied on ("" when none was needed).
func (s *Service) responseGate(tx store.Tx, p *principal, d *domain.Dispute, in ResolveInput) (string, error) {
	resp, err := tx.GetDisputeResponse(d.NetworkID, d.ID)
	if err != nil {
		return "", err
	}
	closes := d.OpenedAt.Add(time.Duration(p.net.Policy.ResponsePeriodDays) * 24 * time.Hour)
	if resp != nil || !s.now().Before(closes) {
		return "", nil
	}
	if !in.WaiveResponse {
		return "", domain.InvalidState("the other party's response period ends on %s; wait for a response or waive it with a reason", closes.Format("2006-01-02"))
	}
	reason := strings.TrimSpace(in.WaiveReason)
	if reason == "" {
		return "", domain.Invalid("waive_reason", "required to waive the response")
	}
	if err := boundedText("waive_reason", reason, maxReasonLen); err != nil {
		return "", err
	}
	return reason, nil
}

func (s *Service) finishDecision(tx store.Tx, p *principal, d *domain.Dispute, t *domain.Transaction, action, key, waived string) error {
	if err := tx.UpdateDispute(d); err != nil {
		return err
	}
	if err := s.disputeEvent(tx, d, p.member.ID, string(d.Status)); err != nil {
		return err
	}
	if err := s.notifyOthers(tx, d.NetworkID, []string{t.PayerMemberID, t.PayeeMemberID}, p.member.ID, NotifDisputeDecided,
		"A dispute has a decision", "The dispute was decided: "+strings.ReplaceAll(string(d.Outcome), "_", " ")+".", "/disputes/"+d.ID); err != nil {
		return err
	}
	reason := "decision recorded; rationale held on dispute"
	if waived != "" {
		reason += "; response waived: " + waived
	}
	return s.audit(tx, p, auditEntry{Type: p.deciderType(), Action: action, TargetType: "dispute", TargetID: d.ID,
		Reason: reason, IdemRef: keyRef(key),
		Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
		After:    map[string]any{"status": d.Status, "outcome": string(d.Outcome), "amount": d.ReversalAmount, "transaction_status": t.Status}})
}

// RetryDisputePosting re-attempts the reversal of a `posting_pending` dispute
// (DOM-021). It fails with insufficient-available-credit, changing nothing,
// until the debited account can absorb the reversal (balance recovered or an
// audited limit override applied). The same independence rules apply.
func (s *Service) RetryDisputePosting(ctx context.Context, actor domain.Actor, netID, disputeID string) (*domain.Dispute, error) {
	var out *domain.Dispute
	err := s.update(ctx, func(tx store.Tx) error {
		p, d, t, err := s.decideSetup(tx, actor, netID, disputeID)
		if err != nil {
			return err
		}
		next, err := domain.NextDisputeStatus(d.Status, domain.DCmdRetryPost)
		if err != nil {
			return err
		}
		if _, err := s.reverse(tx, p.net, t, d.ReversalAmount, "dispute "+d.ID+": "+string(d.Outcome), p.member.ID, p.actor.RequestID); err != nil {
			return err
		}
		if t.Status == domain.TxDisputed {
			if err := s.transition(tx, netID, t, domain.CmdCloseDispute, p.member.ID, p.actor.RequestID); err != nil {
				return err
			}
		}
		d.Status = next
		out = d
		return s.finishDecision(tx, p, d, t, domain.ActDisputeResolved, "", "")
	})
	return out, err
}

// DisputeView is a dispute with its transaction and party display names, the
// only data a reviewer needs (PER-003).
type DisputeView struct {
	Dispute   *domain.Dispute
	Tx        *domain.Transaction
	PayerName string
	PayeeName string
	Events    []store.DisputeEvent
	Response  *store.DisputeResponse // the other party's response, nil until submitted (FR-076)
	ReReview  *store.ReReview        // the one re-review, nil until requested (FR-078)
	// Deadlines are the response deadline and decision target (GOV-014).
	Deadlines domain.DisputeDeadlines
}

func (s *Service) disputeView(tx store.Tx, net *domain.Network, d *domain.Dispute) (*DisputeView, error) {
	netID := net.ID
	t, err := tx.GetTransaction(netID, d.TxID)
	if err != nil {
		return nil, err
	}
	v := &DisputeView{Dispute: d, Tx: t}
	if m, err := tx.GetMember(netID, t.PayerMemberID); err == nil {
		v.PayerName = m.DisplayName
	}
	if m, err := tx.GetMember(netID, t.PayeeMemberID); err == nil {
		v.PayeeName = m.DisplayName
	}
	if v.Events, err = tx.ListDisputeEvents(netID, d.ID); err != nil {
		return nil, err
	}
	if v.Response, err = tx.GetDisputeResponse(netID, d.ID); err != nil {
		return nil, err
	}
	if v.ReReview, err = tx.GetReReview(netID, d.ID); err != nil {
		return nil, err
	}
	v.Deadlines = domain.DeadlinesOf(d, net.Policy.ResponsePeriodDays, v.Response != nil, s.now())
	return v, nil
}

// GetDispute returns a dispute to its parties, reviewers and administrators;
// everyone else gets not-found.
func (s *Service) GetDispute(ctx context.Context, actor domain.Actor, netID, disputeID string) (*DisputeView, error) {
	var out *DisputeView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		d, err := tx.GetDispute(netID, disputeID)
		if err != nil {
			return domain.NotFound("dispute")
		}
		v, err := s.disputeView(tx, p.net, d)
		if err != nil {
			return err
		}
		if !p.actor.System && !v.Tx.IsParty(p.memberID()) && p.requireDecider() != nil {
			return domain.NotFound("dispute")
		}
		out = v
		return nil
	})
	return out, err
}

// ListDisputes lists disputes: all for reviewers and administrators, own for members.
func (s *Service) ListDisputes(ctx context.Context, actor domain.Actor, netID string, statuses []domain.DisputeStatus, pg Page) ([]*domain.Dispute, error) {
	var out []*domain.Dispute
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		f := store.DisputeFilter{Statuses: statuses, AfterID: pg.AfterID, Limit: clampPage(pg)}
		if !p.actor.System && p.requireDecider() != nil {
			if p.member == nil || p.member.Status == domain.MemberPending {
				return domain.ErrForbidden
			}
			f.PartyID = p.member.ID
		}
		out, err = tx.ListDisputes(netID, f)
		return err
	})
	return out, err
}
