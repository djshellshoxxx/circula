package api

import (
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) openDispute(c *reqCtx) error { // API-054
	var in struct {
		TransactionID string `json:"transaction_id"`
		Category      string `json:"category"`
		Statement     string `json:"statement"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	d, err := s.svc.OpenDispute(c.r.Context(), c.actor, c.netID, service.OpenDisputeInput{TxID: in.TransactionID,
		Category: in.Category, Statement: in.Statement, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/disputes/"+d.ID)
	return ok(c, 201, toDispute(d))
}

func (s *server) listDisputes(c *reqCtx) error { // API-054
	if err := c.only("limit", "cursor", "status"); err != nil {
		return err
	}
	var sts []domain.DisputeStatus
	if raw := c.query.Get("status"); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			found := false
			for _, k := range domain.AllDisputeStatuses {
				if string(k) == p {
					sts, found = append(sts, k), true
				}
			}
			if !found {
				return invalidQuery("unknown status " + p)
			}
		}
	}
	p, err := c.page(c.query.Get("status"))
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.Dispute, error) {
		return s.svc.ListDisputes(c.r.Context(), c.actor, c.netID, sts, pg)
	}, func(d *domain.Dispute) string { return d.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]disputeDTO, 0, len(rows))
	for _, d := range rows {
		out = append(out, toDispute(d))
	}
	return listOK(c, out, next)
}

type disputeEventDTO struct {
	Action     string `json:"action"`
	Outcome    string `json:"outcome,omitempty"`
	ActorID    string `json:"actor_id"`
	OccurredAt string `json:"occurred_at"`
}

type disputeDetail struct {
	disputeDTO
	Transaction txDTO               `json:"transaction"`
	PayerName   string              `json:"payer_display_name"`
	PayeeName   string              `json:"payee_display_name"`
	Events      []disputeEventDTO   `json:"events"`
	Response    *disputeResponseDTO `json:"response"`  // FR-076
	ReReview    *reReviewDTO        `json:"re_review"` // FR-078
	// GOV-014: deadlines visible to parties. A missed response deadline decides nothing.
	ResponseDeadline string `json:"response_deadline"`
	DecisionTarget   string `json:"decision_target"`
	ResponseOverdue  bool   `json:"response_overdue"`
	DecisionOverdue  bool   `json:"decision_overdue"`
}

func (s *server) getDispute(c *reqCtx) error { // API-054
	v, err := s.svc.GetDispute(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	out := disputeDetail{disputeDTO: toDispute(v.Dispute), Transaction: toTx(v.Tx, c.member.ID),
		PayerName: v.PayerName, PayeeName: v.PayeeName, Events: []disputeEventDTO{},
		Response: toDisputeResponse(v.Response), ReReview: toReReview(v.ReReview),
		ResponseDeadline: ts(v.Deadlines.ResponseDeadline), DecisionTarget: ts(v.Deadlines.DecisionTarget),
		ResponseOverdue: v.Deadlines.ResponseOverdue, DecisionOverdue: v.Deadlines.DecisionOverdue}
	for _, e := range v.Events {
		out.Events = append(out.Events, disputeEventDTO{Action: e.Action, Outcome: e.Outcome, ActorID: e.ActorID, OccurredAt: ts(e.OccurredAt)})
	}
	return ok(c, 200, out)
}

// outcomes accepts the contract vocabulary (API-055) and the service's own.
var outcomes = map[string]domain.DisputeOutcome{
	"no_action": domain.OutcomeDismissed, "dismissed": domain.OutcomeDismissed,
	"reversal": domain.OutcomeUpheld, "upheld": domain.OutcomeUpheld,
	"partially_upheld": domain.OutcomePartiallyUpheld,
}

func (s *server) resolveDispute(c *reqCtx) error { // API-055
	var in struct {
		Outcome       string  `json:"outcome"`
		Reason        string  `json:"reason"`
		AmountMinor   *string `json:"amount_minor"`
		WaiveResponse bool    `json:"waive_response"`
		WaiveReason   string  `json:"waive_reason"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	oc, found := outcomes[in.Outcome]
	if !found {
		if in.Outcome == "adjustment" {
			return notImplemented("resolving a dispute with an adjustment outcome")
		}
		return validation("outcome", "invalid-enum", "must be no_action, reversal or partially_upheld")
	}
	var amt int64
	if in.AmountMinor != nil {
		var err error
		if amt, err = parseMinor("amount_minor", *in.AmountMinor, false); err != nil {
			return err
		}
	}
	d, err := s.svc.ResolveDispute(c.r.Context(), c.actor, c.netID, service.ResolveInput{DisputeID: c.r.PathValue("id"),
		Outcome: oc, Amount: amt, Rationale: in.Reason, WaiveResponse: in.WaiveResponse, WaiveReason: in.WaiveReason, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	return ok(c, 200, toDispute(d))
}

func (s *server) reviewDispute(c *reqCtx) error {
	d, err := s.svc.StartDisputeReview(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, toDispute(d))
}

func (s *server) withdrawDispute(c *reqCtx) error {
	d, err := s.svc.WithdrawDispute(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, toDispute(d))
}

func (s *server) retryDispute(c *reqCtx) error {
	d, err := s.svc.RetryDisputePosting(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, toDispute(d))
}
