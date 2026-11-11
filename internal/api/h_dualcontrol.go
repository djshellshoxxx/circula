package api

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// pendingDTO is a dual-control adjustment request (SEC-011).
type pendingDTO struct {
	ID                string  `json:"id"`
	MemberID          string  `json:"member_id"` // the adjusted member
	ProposerMemberID  string  `json:"proposer_member_id"`
	Direction         string  `json:"direction"`
	AmountMinor       string  `json:"amount_minor"`
	Reason            string  `json:"reason"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	ExpiresAt         string  `json:"expires_at"`
	DecidedByMemberID *string `json:"decided_by_member_id"`
	DecidedAt         *string `json:"decided_at"`
	TransactionID     *string `json:"transaction_id"`
}

func toPending(p *domain.PendingAdjustment) pendingDTO {
	d := pendingDTO{ID: p.ID, MemberID: p.Target, ProposerMemberID: p.Proposer, Direction: p.Direction,
		AmountMinor: minor(p.Amount), Reason: p.Reason, Status: string(p.Status), CreatedAt: ts(p.CreatedAt),
		ExpiresAt: ts(p.ExpiresAt), DecidedAt: tsPtr(p.DecidedAt)}
	if p.DecidedBy != "" {
		d.DecidedByMemberID = &p.DecidedBy
	}
	if p.TxID != "" {
		d.TransactionID = &p.TxID
	}
	return d
}

// dualControlRoutes registers the SEC-011 endpoints (extension to API-057).
func (s *server) dualControlRoutes(R func(string, handlerFunc, ...func(*route))) {
	R("GET "+v1+"/admin/adjustments", s.listPendingAdjustments)
	R("GET "+v1+"/admin/adjustments/{id}", s.getPendingAdjustment)
	R("POST "+v1+"/admin/adjustments/{id}/approve", s.approveAdjustment)
	R("POST "+v1+"/admin/adjustments/{id}/reject", s.rejectAdjustment)
	R("POST "+v1+"/admin/adjustments/{id}/cancel", s.cancelAdjustment)
}

func (s *server) listPendingAdjustments(c *reqCtx) error {
	if err := c.only("limit", "cursor", "status"); err != nil {
		return err
	}
	var sts []domain.PendingStatus
	if v := c.query.Get("status"); v != "" {
		switch st := domain.PendingStatus(v); st {
		case domain.PendingOpen, domain.PendingApproved, domain.PendingRejected, domain.PendingExpired, domain.PendingCancelled:
			sts = []domain.PendingStatus{st}
		default:
			return invalidQuery("unknown status")
		}
	}
	p, err := c.page(c.query.Get("status"))
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.PendingAdjustment, error) {
		return s.svc.ListPendingAdjustments(c.r.Context(), c.actor, c.netID, sts, pg)
	}, func(x *domain.PendingAdjustment) string { return x.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]pendingDTO, 0, len(rows))
	for _, x := range rows {
		out = append(out, toPending(x))
	}
	return listOK(c, out, next)
}

func (s *server) getPendingAdjustment(c *reqCtx) error {
	x, err := s.svc.GetPendingAdjustment(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, toPending(x))
}

// approveAdjustment posts the adjustment and returns the transaction. A repeat
// by the same approver returns the same transaction.
func (s *server) approveAdjustment(c *reqCtx) error {
	t, err := s.svc.ApproveAdjustment(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/transactions/"+t.ID)
	return ok(c, 200, toTx(t, c.member.ID))
}

func (s *server) rejectAdjustment(c *reqCtx) error {
	x, err := s.svc.RejectAdjustment(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, toPending(x))
}

func (s *server) cancelAdjustment(c *reqCtx) error {
	x, err := s.svc.CancelAdjustment(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	return ok(c, 200, toPending(x))
}
