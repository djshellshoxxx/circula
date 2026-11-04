package service

import (
	"context"
	"errors"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// DepartInput ends a membership, voluntarily or administratively.
type DepartInput struct {
	MemberID string // empty = the caller departs voluntarily
	// ReasonCode is mandatory when an administrator removes someone else: one
	// of domain.RemovalReasons (GOV-011). Ignored for a voluntary departure.
	ReasonCode string
	// Reason is optional free text: the departing member's own words, or the
	// administrator's detail on a removal. It is kept in the audit log only.
	Reason string
	// MatterTxID optionally names the transaction a removal is about. It
	// must be one the removed member is a party to (GOV-011).
	MatterTxID string
}

// DepartMember ends a membership (DOM-024, FR-014, FR-015). It is refused with
// a blocker list while the member is party to a `disputed` transaction or is the
// last active administrator. Pending proposals involving the member are
// auto-cancelled and audited. Per the network's departure rule the balance is
// retained on the closed account (default) or moved to the system account by
// an audited settlement adjustment; that adjustment is a posting, so a
// departure that needs one is refused unless the network accepts postings.
// Roles are revoked; entries are retained and history stays exportable by the
// departed member.
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
			if err := p.requireLasting(); err != nil { // A3-02
				return err
			}
			if domain.RemovalReasonLabel(in.ReasonCode) == "" { // GOV-011
				return domain.Invalid("reason_code", "required, one of "+strings.Join(domain.RemovalReasons, ", "))
			}
		}
		if err := boundedText("reason", in.Reason, maxReasonLen); err != nil {
			return err
		}
		if adminAction {
			if target, err = tx.GetMember(netID, in.MemberID); err != nil {
				return domain.NotFound("member")
			}
		}
		if target.Status != domain.MemberActive && target.Status != domain.MemberSuspended {
			return domain.InvalidState("member is %s", target.Status)
		}
		if adminAction {
			if err := removalIndependent(tx, netID, p.member.ID, target.ID, in); err != nil {
				return err
			}
		}
		blockers, err := departBlockers(tx, netID, target)
		if err != nil {
			return err
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
		at, reason := domain.ActorMember, in.Reason
		after := map[string]any{"status": domain.MemberDeparted, "rule": string(p.net.Policy.DepartureRule)}
		if adminAction {
			at = domain.ActorAdmin
			if err := s.adminNotice(tx, netID, target.ID, NotifAdminRemoved, "Your membership was ended",
				"An administrator ended your membership. Your history stays available to you.", "/"); err != nil {
				return err
			}
			// GOV-011: the member is told, with the listed reason (fixed text, so
			// they can challenge it under GOV-016); the administrator's detail
			// stays in the audit log.
			label := domain.RemovalReasonLabel(in.ReasonCode)
			if err := s.notify(tx, netID, target.ID, NotifMemberRemoved, "Your membership was ended",
				"An administrator ended your membership. Recorded reason: "+label+
					". You keep read-only access to your own history and can export it. You can challenge the removal.", "/history", ""); err != nil {
				return err
			}
			reason = label
			if d := strings.TrimSpace(in.Reason); d != "" {
				reason += ": " + d
			}
			after["reason_code"] = in.ReasonCode
			if in.MatterTxID != "" {
				after["matter_transaction"] = in.MatterTxID
			}
		}
		if err := s.audit(tx, p, auditEntry{Type: at, Action: domain.ActMemberDeparted, TargetType: "member", TargetID: target.ID,
			Reason: reason, Subjects: []string{target.ID}, Before: map[string]any{"status": target.Status},
			After: after}); err != nil {
			return err
		}
		out, err = tx.GetMember(netID, target.ID)
		return err
	})
	return out, err
}

// errRemovalConflict refuses a removal for fraud or breach decided by an
// administrator who is the counterparty in the matter (GOV-011, SEC-010).
var errRemovalConflict = domain.Forbidden("conflict of interest: you took part in the matter of this removal; another administrator must decide it")

// IsRemovalConflict reports whether err is the refusal of a removal by a
// conflicted administrator (GOV-011), so adapters can explain it.
func IsRemovalConflict(err error) bool {
	var de *domain.Error
	return errors.As(err, &de) && de == errRemovalConflict
}

// removalIndependent checks the matter of an administrator removal (GOV-011,
// SEC-010). A named matter must be a transaction the removed member is a
// party to. A removal for fraud or breach is refused when the deciding
// administrator took part (DOM-038: party, proposer, accepter or reverser)
// in the matter: the named transaction, or any trade of the member that is
// still an open case, under an undecided dispute or a requested re-review.
func removalIndependent(tx store.Tx, netID, adminID, targetID string, in DepartInput) error {
	var matters []*domain.Transaction
	if in.MatterTxID != "" {
		t, err := tx.GetTransaction(netID, in.MatterTxID)
		if err != nil || !t.IsParty(targetID) {
			return domain.Invalid("transaction_id", "must be a transaction the member is a party to")
		}
		matters = append(matters, t)
	}
	if !domain.RemovalNeedsIndependence(in.ReasonCode) {
		return nil
	}
	open, err := openCases(tx, netID, targetID)
	if err != nil {
		return err
	}
	for _, t := range append(matters, open...) {
		if err := independent(tx, netID, t, adminID); err != nil {
			if domain.CodeOf(err) == domain.CodeForbidden {
				return errRemovalConflict
			}
			return err
		}
	}
	return nil
}

// openCases lists the member's trades whose dispute is undecided (open, under
// review or awaiting its posting) or whose decision awaits a requested
// re-review.
func openCases(tx store.Tx, netID, memberID string) ([]*domain.Transaction, error) {
	var out []*domain.Transaction
	after := ""
	for {
		ds, err := tx.ListDisputes(netID, store.DisputeFilter{PartyID: memberID, AfterID: after, Limit: 200})
		if err != nil {
			return nil, err
		}
		for _, d := range ds {
			after = d.ID
			live := d.Status == domain.DisputeOpen || d.Status == domain.DisputeUnderReview || d.Status == domain.DisputePostingPending
			if !live && d.Status == domain.DisputeResolved {
				rr, err := tx.GetReReview(netID, d.ID)
				if err != nil {
					return nil, err
				}
				live = rr != nil && rr.Status == store.ReReviewRequested
			}
			if !live {
				continue
			}
			t, err := tx.GetTransaction(netID, d.TxID)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		}
		if len(ds) < 200 {
			return out, nil
		}
	}
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
		// GOV-010: the other party is told their open proposal ended.
		other := t.PayerMemberID
		if other == target.ID {
			other = t.PayeeMemberID
		}
		if err := s.notify(tx, p.net.ID, other, NotifMemberLeft, "A trade was cancelled",
			"A member left the network, so your open trade proposal with them was cancelled.", "/trades/"+t.ID, ""); err != nil {
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
	// The settlement is a posting, so it gets the same network gate as any
	// adjustment: nothing posts from the closure date on (GOV-025 b, FR-068),
	// while the network is suspended (GOV-023) or while it has no active
	// administrator (GOV-020). The whole departure is refused and its unit
	// rolls back; completing it under `retain` instead would override the
	// network's departure rule (A1-01).
	if err := p.requireNetworkActive(); err != nil {
		return err
	}
	dir, amt := AdjustCredit, -acc.Balance
	if acc.Balance > 0 {
		dir, amt = AdjustDebit, acc.Balance
	}
	// SA2-02: an administrator removing someone else must not move more than the
	// dual-control threshold alone; settle it through a pending adjustment first.
	if t := p.net.Policy.DualControlThreshold; adminAction && t > 0 && amt > t {
		return &domain.Error{Code: domain.CodeApprovalRequired,
			Msg: "the departure settlement exceeds the dual-control threshold; adjust the balance through an approved adjustment first"}
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

// departBlockers lists what stops target from leaving (FR-014, FR-015).
func departBlockers(tx store.Tx, netID string, target *domain.Member) ([]string, error) {
	var blockers []string
	disputed, err := tx.ListTransactions(netID, store.TxFilter{MemberID: target.ID, Statuses: []domain.TxStatus{domain.TxDisputed}, Limit: 500})
	if err != nil {
		return nil, err
	}
	for _, t := range disputed {
		blockers = append(blockers, "disputed transaction "+t.ID)
	}
	if err := lastAdminGuard(tx, netID, target); err != nil {
		if domain.CodeOf(err) != domain.CodeBlocked {
			return nil, err
		}
		blockers = append(blockers, "last active administrator")
	}
	return blockers, nil
}

// DepartPreview is what a member sees before confirming that they leave.
type DepartPreview struct {
	Blockers      []string
	Rule          domain.DepartureRule
	Balance       int64
	Decimals      int
	OpenProposals int
}

// PreviewDeparture shows the caller what leaving would do, without changing
// anything (FR-014): blockers, the departure rule, the balance it applies to
// and how many open proposals would be cancelled.
func (s *Service) PreviewDeparture(ctx context.Context, actor domain.Actor, netID string) (*DepartPreview, error) {
	var out *DepartPreview
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberSuspended) {
			return domain.InvalidState("only a current member can leave")
		}
		out = &DepartPreview{Rule: p.net.Policy.DepartureRule, Decimals: p.net.Policy.Decimals}
		if out.Blockers, err = departBlockers(tx, netID, p.member); err != nil {
			return err
		}
		if acc, err := tx.GetAccountByMember(netID, p.member.ID); err == nil {
			out.Balance = acc.Balance
		}
		props, err := tx.ListTransactions(netID, store.TxFilter{MemberID: p.member.ID, Statuses: []domain.TxStatus{domain.TxProposed}, Limit: 500})
		out.OpenProposals = len(props)
		return err
	})
	return out, err
}
