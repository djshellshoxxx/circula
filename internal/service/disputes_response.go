package service

import (
	"context"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// ReReviewWindowDays is how long after a decision a party may ask for the one
// re-review (FR-078 "within a stated period"). It is a platform constant until
// the product owner makes it a network policy.
const ReReviewWindowDays = 14

// Re-review outcomes beyond the dispute outcomes upheld / partially_upheld.
const ReReviewConfirmed = "confirmed"

// RespondInput is the other party's answer to a dispute (FR-076).
type RespondInput struct {
	DisputeID      string
	Response       string
	IdempotencyKey string `json:"-"`
}

// RespondToDispute lets the party who did not open a dispute submit one
// bounded response while the dispute is open or under review. The response is
// visible to the opener, the reviewer and administrators through GetDispute.
// Suspended parties may respond: answering adds no obligation (DOM-039). Only
// one response is accepted. A decision does not wait for a response (OQ-FR-2
// is open: no response period is configured yet).
func (s *Service) RespondToDispute(ctx context.Context, actor domain.Actor, netID string, in RespondInput) (*store.DisputeResponse, error) {
	var out *store.DisputeResponse
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberSuspended) {
			return domain.Forbidden("membership cannot perform this action")
		}
		out, err = idempotent(s, tx, netID, actor, "dispute.respond", in.IdempotencyKey, in, func() (string, error) {
			d, err := tx.GetDispute(netID, in.DisputeID)
			if err != nil {
				return "", domain.NotFound("dispute")
			}
			t, err := tx.GetTransaction(netID, d.TxID)
			if err != nil || !t.IsParty(p.member.ID) {
				return "", domain.NotFound("dispute")
			}
			if d.OpenedBy == p.member.ID {
				return "", domain.Forbidden("only the other party can respond")
			}
			if d.Status != domain.DisputeOpen && d.Status != domain.DisputeUnderReview {
				return "", domain.InvalidState("dispute is %s; a response can only be added before a decision", d.Status)
			}
			body := strings.TrimSpace(in.Response)
			if body == "" {
				return "", domain.Invalid("response", "required")
			}
			if err := boundedText("response", body, maxTextLen); err != nil {
				return "", err
			}
			if err := tx.InsertDisputeResponse(store.DisputeResponse{ID: s.id(), NetworkID: netID, DisputeID: d.ID,
				ResponderID: p.member.ID, Body: body, SubmittedAt: s.now()}); err != nil {
				if domain.CodeOf(err) == domain.CodeConflict {
					return "", domain.InvalidState("a response has already been submitted")
				}
				return "", err
			}
			if err := s.disputeEvent(tx, d, p.member.ID, "responded"); err != nil {
				return "", err
			}
			link := "/disputes/" + d.ID
			if err := s.notifyOthers(tx, netID, []string{d.OpenedBy, d.ReviewerID}, p.member.ID, NotifDisputeResponse,
				"A response was added to a dispute", p.member.DisplayName+" responded to the dispute.", link); err != nil {
				return "", err
			}
			return d.ID, s.audit(tx, p, auditEntry{Action: domain.ActDisputeResponded, TargetType: "dispute", TargetID: d.ID,
				IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
				After: map[string]any{"status": d.Status}})
		}, func(id string) (*store.DisputeResponse, error) {
			r, err := tx.GetDisputeResponse(netID, id)
			if err == nil && r == nil {
				err = domain.NotFound("response")
			}
			return r, err
		})
		return err
	})
	return out, err
}

// ReReviewRequestInput asks for the one re-review of a decided dispute (FR-078).
type ReReviewRequestInput struct {
	DisputeID      string
	Reason         string
	IdempotencyKey string `json:"-"`
}

// RequestReReview lets either party to a decided dispute ask, once, within
// ReReviewWindowDays of the decision, for a different reviewer to look again.
// The request is stored as its own record; the original decision is untouched.
// A second request on the same dispute is refused even if the first was
// decided (the database also enforces it).
func (s *Service) RequestReReview(ctx context.Context, actor domain.Actor, netID string, in ReReviewRequestInput) (*store.ReReview, error) {
	var out *store.ReReview
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "dispute.rereview", in.IdempotencyKey, in, func() (string, error) {
			if err := p.requireNetworkActive(); err != nil {
				return "", err
			}
			d, err := tx.GetDispute(netID, in.DisputeID)
			if err != nil {
				return "", domain.NotFound("dispute")
			}
			t, err := tx.GetTransaction(netID, d.TxID)
			if err != nil || !t.IsParty(p.member.ID) {
				return "", domain.NotFound("dispute")
			}
			if err := requireReason(in.Reason); err != nil {
				return "", domain.Invalid("reason", "required, up to 500 characters")
			}
			if d.Status != domain.DisputeResolved || d.DecidedAt == nil {
				return "", domain.InvalidState("dispute is %s; only a decided dispute can be re-reviewed", d.Status)
			}
			closes := d.DecidedAt.Add(ReReviewWindowDays * 24 * time.Hour)
			if s.now().After(closes) {
				return "", domain.InvalidState("the re-review period closed on %s", closes.Format("2006-01-02"))
			}
			if prev, err := tx.GetReReview(netID, d.ID); err != nil {
				return "", err
			} else if prev != nil {
				return "", domain.InvalidState("this dispute has already had its one re-review request")
			}
			r := store.ReReview{ID: s.id(), NetworkID: netID, DisputeID: d.ID, RequestedBy: p.member.ID,
				Reason: strings.TrimSpace(in.Reason), RequestedAt: s.now(), Status: store.ReReviewRequested}
			if err := tx.InsertReReview(r); err != nil {
				if domain.CodeOf(err) == domain.CodeConflict {
					return "", domain.InvalidState("this dispute has already had its one re-review request")
				}
				return "", err
			}
			if err := s.disputeEvent(tx, d, p.member.ID, "rereview_requested"); err != nil {
				return "", err
			}
			link := "/disputes/" + d.ID
			deciders, err := deciderIDs(tx, netID)
			if err != nil {
				return "", err
			}
			var eligible []string
			for _, id := range deciders {
				if id != d.ReviewerID && !t.IsParty(id) {
					eligible = append(eligible, id)
				}
			}
			if err := s.notifyOthers(tx, netID, append(eligible, t.PayerMemberID, t.PayeeMemberID), p.member.ID,
				NotifReReviewAsked, "A dispute decision is awaiting re-review", "One re-review was requested.", link); err != nil {
				return "", err
			}
			return d.ID, s.audit(tx, p, auditEntry{Action: domain.ActDisputeReReviewAsked, TargetType: "dispute", TargetID: d.ID,
				IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
				After: map[string]any{"status": "requested"}})
		}, func(id string) (*store.ReReview, error) { return loadReReview(tx, netID, id) })
		return err
	})
	return out, err
}

func loadReReview(tx store.Tx, netID, disputeID string) (*store.ReReview, error) {
	r, err := tx.GetReReview(netID, disputeID)
	if err == nil && r == nil {
		err = domain.NotFound("re-review")
	}
	return r, err
}

// ReReviewDecideInput records the re-review decision (FR-078).
type ReReviewDecideInput struct {
	DisputeID string
	// Outcome is "confirmed" (the original decision stands), "upheld" or
	// "partially_upheld" (a further reversal is posted, at most what is still
	// reversible on the trade).
	Outcome        string
	Amount         int64 // partially_upheld only
	Rationale      string
	IdempotencyKey string `json:"-"`
}

// DecideReReview records the decision on a requested re-review. The decider
// must be a reviewer or administrator who is not the original reviewer and who
// took no part in the trade (DOM-038). The original dispute decision is never
// edited: "confirmed" posts nothing, while "upheld" / "partially_upheld" post
// a new reversal of the remaining reversible amount, refused with
// insufficient-available-credit (nothing changes) if LED-5 blocks it. An
// earlier full reversal cannot be undone, so only "confirmed" remains.
func (s *Service) DecideReReview(ctx context.Context, actor domain.Actor, netID string, in ReReviewDecideInput) (*store.ReReview, error) {
	var out *store.ReReview
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireDecider(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "dispute.rereview.decide", in.IdempotencyKey, in, func() (string, error) {
			return s.decideReReview(tx, p, in)
		}, func(id string) (*store.ReReview, error) { return loadReReview(tx, netID, id) })
		return err
	})
	return out, err
}

func (s *Service) decideReReview(tx store.Tx, p *principal, in ReReviewDecideInput) (string, error) {
	netID := p.net.ID
	d, err := tx.GetDispute(netID, in.DisputeID)
	if err != nil {
		return "", domain.NotFound("dispute")
	}
	rr, err := tx.GetReReview(netID, d.ID)
	if err != nil {
		return "", err
	}
	if rr == nil {
		return "", domain.InvalidState("no re-review was requested for this dispute")
	}
	if rr.Status != store.ReReviewRequested {
		return "", domain.InvalidState("this re-review has already been decided")
	}
	t, err := tx.GetTransaction(netID, d.TxID)
	if err != nil {
		return "", err
	}
	if err := independent(tx, netID, t, p.member.ID); err != nil {
		return "", err
	}
	if p.member.ID == d.ReviewerID {
		return "", domain.Forbidden("the re-review needs a different reviewer than the original decision")
	}
	if strings.TrimSpace(in.Rationale) == "" {
		return "", domain.Invalid("rationale", "required")
	}
	if err := boundedText("rationale", in.Rationale, maxTextLen); err != nil {
		return "", err
	}
	remaining := t.Amount - t.ReversedTotal
	amount := int64(0)
	switch in.Outcome {
	case ReReviewConfirmed:
		if in.Amount != 0 {
			return "", domain.Invalid("amount", "not allowed when confirming")
		}
	case string(domain.OutcomeUpheld):
		if in.Amount != 0 && in.Amount != remaining {
			return "", domain.Invalid("amount", "an upheld re-review reverses the full remaining amount; use partially_upheld")
		}
		amount = remaining
	case string(domain.OutcomePartiallyUpheld):
		if in.Amount <= 0 || in.Amount > remaining {
			return "", domain.Invalid("amount", "must be between 1 and the remaining reversible amount "+fmtAmt(p.net, remaining))
		}
		amount = in.Amount
	default:
		return "", domain.Invalid("outcome", "must be confirmed, upheld or partially_upheld")
	}
	if in.Outcome != ReReviewConfirmed && amount <= 0 {
		return "", domain.InvalidState("nothing left to reverse; the only available outcome is confirmed")
	}
	if amount > 0 {
		if err := p.requireNetworkActive(); err != nil {
			return "", err
		}
		if _, err := s.reverse(tx, p.net, t, amount, "re-review of dispute "+d.ID, p.member.ID, p.actor.RequestID); err != nil {
			return "", err
		}
	}
	now := s.now()
	rr.Status, rr.ReviewerID, rr.Outcome, rr.ReversalAmount, rr.Rationale, rr.DecidedAt =
		store.ReReviewDecided, p.member.ID, in.Outcome, amount, strings.TrimSpace(in.Rationale), &now
	if err := tx.UpdateReReview(*rr); err != nil {
		return "", err
	}
	if err := s.disputeEvent(tx, d, p.member.ID, "rereview_"+in.Outcome); err != nil {
		return "", err
	}
	if err := s.notifyOthers(tx, netID, []string{t.PayerMemberID, t.PayeeMemberID, rr.RequestedBy}, "", NotifReReviewDecided,
		"The re-review of a dispute is decided", "The re-review is decided: "+strings.ReplaceAll(in.Outcome, "_", " ")+".", "/disputes/"+d.ID); err != nil {
		return "", err
	}
	return d.ID, s.audit(tx, p, auditEntry{Type: p.deciderType(), Action: domain.ActDisputeReReviewDecide, TargetType: "dispute", TargetID: d.ID,
		Reason: "decision recorded; rationale held on re-review", IdemRef: keyRef(in.IdempotencyKey),
		Subjects: []string{t.PayerMemberID, t.PayeeMemberID},
		After:    map[string]any{"status": "decided", "outcome": in.Outcome, "amount": amount, "transaction_status": t.Status}})
}
