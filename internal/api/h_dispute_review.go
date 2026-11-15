package api

import (
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// disputeResponseDTO is the other party's response (FR-076).
type disputeResponseDTO struct {
	ID          string `json:"id"`
	ResponderID string `json:"responder_member_id"`
	Response    string `json:"response"`
	SubmittedAt string `json:"submitted_at"`
}

func toDisputeResponse(r *store.DisputeResponse) *disputeResponseDTO {
	if r == nil {
		return nil
	}
	return &disputeResponseDTO{ID: r.ID, ResponderID: r.ResponderID, Response: r.Body, SubmittedAt: ts(r.SubmittedAt)}
}

// reReviewDTO is the single re-review of a decision (FR-078), linked to the dispute.
type reReviewDTO struct {
	ID                  string  `json:"id"`
	RequestedBy         string  `json:"requested_by"`
	Reason              string  `json:"reason"`
	RequestedAt         string  `json:"requested_at"`
	Status              string  `json:"status"`
	ReviewerID          *string `json:"reviewer_id"`
	Outcome             *string `json:"outcome"`
	ReversalAmountMinor string  `json:"reversal_amount_minor"`
	DecisionReason      string  `json:"decision_reason"`
	DecidedAt           *string `json:"decided_at"`
}

func toReReview(r *store.ReReview) *reReviewDTO {
	if r == nil {
		return nil
	}
	return &reReviewDTO{ID: r.ID, RequestedBy: r.RequestedBy, Reason: r.Reason, RequestedAt: ts(r.RequestedAt), Status: r.Status,
		ReviewerID: strPtr(r.ReviewerID), Outcome: strPtr(r.Outcome), ReversalAmountMinor: minor(r.ReversalAmount),
		DecisionReason: r.Rationale, DecidedAt: tsPtr(r.DecidedAt)}
}

func (s *server) respondDispute(c *reqCtx) error { // FR-076
	var in struct {
		Response string `json:"response"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	r, err := s.svc.RespondToDispute(c.r.Context(), c.actor, c.netID, service.RespondInput{DisputeID: c.r.PathValue("id"),
		Response: in.Response, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	return ok(c, 201, toDisputeResponse(r))
}

func (s *server) requestReReview(c *reqCtx) error { // FR-078
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	r, err := s.svc.RequestReReview(c.r.Context(), c.actor, c.netID, service.ReReviewRequestInput{DisputeID: c.r.PathValue("id"),
		Reason: in.Reason, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	return ok(c, 201, toReReview(r))
}

func (s *server) decideReReview(c *reqCtx) error { // FR-078
	var in struct {
		Outcome     string  `json:"outcome"`
		Reason      string  `json:"reason"`
		AmountMinor *string `json:"amount_minor"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	var amt int64
	if in.AmountMinor != nil {
		var err error
		if amt, err = parseMinor("amount_minor", *in.AmountMinor, false); err != nil {
			return err
		}
	}
	r, err := s.svc.DecideReReview(c.r.Context(), c.actor, c.netID, service.ReReviewDecideInput{DisputeID: c.r.PathValue("id"),
		Outcome: in.Outcome, Amount: amt, Rationale: in.Reason, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	return ok(c, 200, toReReview(r))
}
