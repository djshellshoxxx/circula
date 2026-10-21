package api

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) adminReverse(c *reqCtx) error { // API-056
	var in struct {
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
	t, err := s.svc.ReverseTransaction(c.r.Context(), c.actor, c.netID, service.ReverseInput{TxID: c.r.PathValue("id"),
		Amount: amt, Reason: in.Reason, IdempotencyKey: c.idemKey})
	if err != nil {
		return s.txActionError(c, c.r.PathValue("id"), err)
	}
	c.w.Header().Set("Location", v1+"/transactions/"+t.ID)
	return ok(c, 201, toTx(t, c.member.ID))
}

func (s *server) adminAdjust(c *reqCtx) error { // API-057
	var in struct {
		MemberID    string `json:"member_id"`
		Direction   string `json:"direction"`
		AmountMinor string `json:"amount_minor"`
		Reason      string `json:"reason"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	amt, err := parseMinor("amount_minor", in.AmountMinor, false)
	if err != nil {
		return err
	}
	t, err := s.svc.CreateAdjustment(c.r.Context(), c.actor, c.netID, service.AdjustInput{MemberID: in.MemberID,
		Amount: amt, Direction: service.AdjustDirection(in.Direction), Reason: in.Reason, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/transactions/"+t.ID)
	return ok(c, 201, toTx(t, c.member.ID))
}

func (s *server) adminListMembers(c *reqCtx) error { // API-059
	if err := c.only("limit", "cursor", "status"); err != nil {
		return err
	}
	var sts []domain.MemberStatus
	if v := c.query.Get("status"); v != "" {
		switch st := domain.MemberStatus(v); st {
		case domain.MemberPending, domain.MemberActive, domain.MemberSuspended, domain.MemberDeparted:
			sts = []domain.MemberStatus{st}
		default:
			return invalidQuery("unknown status")
		}
	}
	p, err := c.page(c.query.Get("status"))
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.Member, error) {
		return s.svc.AdminListMembers(c.r.Context(), c.actor, c.netID, sts, pg)
	}, func(m *domain.Member) string { return m.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]adminMemberDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, toAdminMember(m))
	}
	return listOK(c, out, next)
}

// adminMember finds one member of any status (no single-member service read).
func (s *server) adminMember(c *reqCtx, id string) (*domain.Member, error) {
	after := ""
	for i := 0; i < 20; i++ {
		rows, err := s.svc.AdminListMembers(c.r.Context(), c.actor, c.netID, nil, service.Page{AfterID: after, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, m := range rows {
			if m.ID == id {
				return m, nil
			}
		}
		if len(rows) < 100 {
			break
		}
		after = rows[len(rows)-1].ID
	}
	return nil, notFoundErr()
}

func (s *server) memberAction(action string) handlerFunc { // API-059
	return func(c *reqCtx) error {
		var in struct {
			Reason string `json:"reason"`
		}
		if err := c.decodeOptional(&in); err != nil {
			return err
		}
		id, ctx := c.r.PathValue("id"), c.r.Context()
		if h := c.r.Header.Get("If-Match"); h != "" { // optional for these actions (see report)
			m, err := s.adminMember(c, id)
			if err != nil {
				return err
			}
			if err := c.requireIfMatch(memberETag(m)); err != nil {
				return err
			}
		}
		var m *domain.Member
		var err error
		switch action {
		case "approve":
			m, err = s.svc.DecideJoinRequest(ctx, c.actor, c.netID, id, true, in.Reason)
		case "suspend":
			err = s.svc.SuspendMember(ctx, c.actor, c.netID, id, in.Reason)
		case "reinstate":
			err = s.svc.RestoreMember(ctx, c.actor, c.netID, id, in.Reason)
		case "remove":
			m, err = s.svc.DepartMember(ctx, c.actor, c.netID, service.DepartInput{MemberID: id, Reason: in.Reason})
		}
		if err != nil {
			return err
		}
		if m == nil {
			if m, err = s.adminMember(c, id); err != nil {
				return err
			}
		}
		c.w.Header().Set("ETag", memberETag(m))
		return ok(c, 200, toAdminMember(m))
	}
}

func (s *server) roleAction(grant bool) handlerFunc { // extension: reviewer/admin roles
	return func(c *reqCtx) error {
		role := domain.Role(c.r.PathValue("role"))
		if role != domain.RoleAdmin && role != domain.RoleReviewer {
			return notFoundErr()
		}
		var err error
		if grant {
			err = s.svc.AssignRole(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"), role)
		} else {
			err = s.svc.RevokeRole(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"), role)
		}
		if err != nil {
			return err
		}
		m, err := s.adminMember(c, c.r.PathValue("id"))
		if err != nil {
			return err
		}
		c.w.Header().Set("ETag", memberETag(m))
		return ok(c, 200, toAdminMember(m))
	}
}
