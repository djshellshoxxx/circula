package api

import (
	"encoding/json"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

func limitETag(b *service.BalanceView) string {
	return etagOf("limit", b.MemberID, minor(b.CreditLimit))
}

// adminGetAccount shows a member's balance view and the ETag for the
// credit-limit PUT. The service audits this read (account.viewed).
func (s *server) adminGetAccount(c *reqCtx) error {
	b, err := s.svc.AdminGetMemberBalance(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	c.w.Header().Set("ETag", limitETag(b))
	return ok(c, 200, toAccount(b))
}

// putCreditLimit implements API-060. {id} is the member id; null removes the
// override so the policy default applies again.
func (s *server) putCreditLimit(c *reqCtx) error {
	var in struct {
		CreditLimitMinor json.RawMessage `json:"credit_limit_minor"`
		Reason           string          `json:"reason"`
		ExpiresAt        *string         `json:"expires_at"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	if in.CreditLimitMinor == nil {
		return invalidBody("credit_limit_minor is required (a string, or null to remove the override)")
	}
	var lim *int64
	if !isNull(in.CreditLimitMinor) {
		str, err := rawString("credit_limit_minor", in.CreditLimitMinor)
		if err != nil {
			return err
		}
		v, err := parseMinor("credit_limit_minor", str, true)
		if err != nil {
			return err
		}
		lim = &v
	}
	var exp *time.Time
	if in.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
		if err != nil {
			return validation("expires_at", "invalid-timestamp", "must be an RFC 3339 timestamp")
		}
		exp = &t
	}
	id := c.r.PathValue("id")
	s.casMu.Lock()
	defer s.casMu.Unlock()
	cur, err := s.svc.AdminGetMemberBalance(c.r.Context(), c.actor, c.netID, id)
	if err != nil {
		return err
	}
	if err := c.requireIfMatch(limitETag(cur)); err != nil {
		return err
	}
	if err := s.svc.SetCreditLimit(c.r.Context(), c.actor, c.netID, service.SetLimitInput{MemberID: id, Limit: lim,
		ExpiresAt: exp, Reason: in.Reason}); err != nil {
		return err
	}
	b, err := s.svc.AdminGetMemberBalance(c.r.Context(), c.actor, c.netID, id)
	if err != nil {
		return err
	}
	c.w.Header().Set("ETag", limitETag(b))
	return ok(c, 200, toAccount(b))
}
