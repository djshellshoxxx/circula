package api

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/service"
)

// Administrator recovery (GOV-021): the member side only. The operator side
// is the CLI `circula admin-recovery`; there is no HTTP route for it (ARC-008,
// SEC-013 decision 4).

type adminRecoveryDTO struct {
	ID                string  `json:"id"`
	RequesterMemberID string  `json:"requester_member_id"`
	Status            string  `json:"status"`
	RequestedAt       string  `json:"requested_at"`
	ExpiresAt         string  `json:"expires_at"`
	FirstApprovedAt   *string `json:"first_approved_at"`
	ClosedAt          *string `json:"closed_at"`
	Mine              bool    `json:"mine"`
}

func toAdminRecovery(v service.AdminRecoveryView) adminRecoveryDTO {
	return adminRecoveryDTO{ID: v.ID, RequesterMemberID: v.RequesterID, Status: string(v.Status), RequestedAt: ts(v.RequestedAt),
		ExpiresAt: ts(v.ExpiresAt), FirstApprovedAt: tsPtr(v.FirstApprovedAt), ClosedAt: tsPtr(v.ClosedAt), Mine: v.Mine}
}

// requestAdminRecovery: the caller asks to be appointed administrator of a
// network that has none. Session authentication only (route flag
// cookieOnly): a stolen API token must not be able to start an appointment.
func (s *server) requestAdminRecovery(c *reqCtx) error {
	var in struct{}
	if err := c.decodeOptional(&in); err != nil {
		return err
	}
	r, err := s.svc.RequestAdminRecovery(c.r.Context(), c.actor, c.netID, service.AdminRecoveryInput{IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/admin-recovery/requests")
	return ok(c, http.StatusCreated, adminRecoveryDTO{ID: r.ID, RequesterMemberID: r.RequesterID, Status: string(r.Status),
		RequestedAt: ts(r.RequestedAt), ExpiresAt: ts(r.ExpiresAt), FirstApprovedAt: tsPtr(r.FirstApprovedAt), ClosedAt: tsPtr(r.ClosedAt), Mine: true})
}

func (s *server) listAdminRecovery(c *reqCtx) error {
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page("")
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]service.AdminRecoveryView, error) {
		return s.svc.ListAdminRecoveryRequests(c.r.Context(), c.actor, c.netID, pg)
	}, func(v service.AdminRecoveryView) string { return v.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]adminRecoveryDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAdminRecovery(r))
	}
	return listOK(c, out, next)
}

// listAdminRecoveryEvents returns the recovery audit events of the network to
// any of its members, reduced as in the members' own audit view.
func (s *server) listAdminRecoveryEvents(c *reqCtx) error {
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page("")
	if err != nil {
		return err
	}
	q := service.AuditQuery{Limit: p.limit + 1}
	if p.after != "" {
		n, ok := atoiNonNeg(p.after)
		if !ok {
			return &apiError{status: 400, code: "invalid-cursor", detail: "cursor is not valid for this request"}
		}
		q.AfterSeq = int64(n)
	}
	rows, err := s.svc.ListAdminRecoveryEvents(c.r.Context(), c.actor, c.netID, q)
	if err != nil {
		return err
	}
	var next *string
	if len(rows) > p.limit {
		rows = rows[:p.limit]
		n := s.sealCursor(itoa(int(rows[len(rows)-1].Seq)), p.filter)
		next = &n
	}
	out := make([]auditDTO, 0, len(rows))
	for _, e := range rows {
		out = append(out, toAudit(e))
	}
	return listOK(c, out, next)
}
