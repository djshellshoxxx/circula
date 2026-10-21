package api

import (
	"encoding/json"
	"strconv"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

type policyDTO struct {
	CurrencyName              string `json:"currency_name"`
	Decimals                  int    `json:"decimals"`
	DefaultCreditLimitMinor   string `json:"default_credit_limit_minor"`
	MaxTxAmountMinor          string `json:"max_tx_amount_minor"`
	ProposalExpiryDays        int    `json:"proposal_expiry_days"`
	DisputeWindowDays         int    `json:"dispute_window_days"`
	JoinMode                  string `json:"join_mode"`
	DepartureRule             string `json:"departure_rule"`
	SignupGrantMinor          string `json:"signup_grant_minor"`
	LimitWarningDays          int    `json:"limit_warning_days"`
	DefaultFlagBlocksPayer    bool   `json:"default_flag_blocks_payer"`
	DirectoryVisibility       string `json:"directory_visibility"`
	DualControlThresholdMinor string `json:"dual_control_threshold_minor"`
	ResponsePeriodDays        int    `json:"response_period_days"`
	ReReviewWindowDays        int    `json:"re_review_window_days"`
	LimitWarningPercent       int    `json:"limit_warning_percent"`
}

func toPolicy(p domain.Policy) policyDTO {
	return policyDTO{CurrencyName: p.CurrencyName, Decimals: p.Decimals, DefaultCreditLimitMinor: minor(p.DefaultCreditLimit),
		MaxTxAmountMinor: minor(p.MaxTxAmount), ProposalExpiryDays: p.ProposalExpiryDays, DisputeWindowDays: p.DisputeWindowDays,
		JoinMode: string(p.JoinMode), DepartureRule: string(p.DepartureRule), SignupGrantMinor: minor(p.SignupGrant),
		LimitWarningDays: p.LimitWarningDays, DefaultFlagBlocksPayer: p.DefaultFlagBlocksPayer, DirectoryVisibility: p.DirectoryVisibility,
		DualControlThresholdMinor: minor(p.DualControlThreshold),
		ResponsePeriodDays:        p.ResponsePeriodDays, ReReviewWindowDays: p.ReReviewWindowDays, LimitWarningPercent: p.LimitWarningPercent}
}

func (d policyDTO) domain() (domain.Policy, error) {
	var p domain.Policy
	var err error
	if p.DefaultCreditLimit, err = parseMinor("default_credit_limit_minor", d.DefaultCreditLimitMinor, true); err != nil {
		return p, err
	}
	if p.MaxTxAmount, err = parseMinor("max_tx_amount_minor", d.MaxTxAmountMinor, false); err != nil {
		return p, err
	}
	if p.SignupGrant, err = parseMinor("signup_grant_minor", d.SignupGrantMinor, true); err != nil {
		return p, err
	}
	if p.DualControlThreshold, err = parseMinor("dual_control_threshold_minor", d.DualControlThresholdMinor, true); err != nil {
		return p, err
	}
	p.CurrencyName, p.Decimals, p.ProposalExpiryDays = d.CurrencyName, d.Decimals, d.ProposalExpiryDays
	p.DisputeWindowDays, p.JoinMode, p.DepartureRule = d.DisputeWindowDays, domain.JoinMode(d.JoinMode), domain.DepartureRule(d.DepartureRule)
	p.LimitWarningDays, p.DefaultFlagBlocksPayer, p.DirectoryVisibility = d.LimitWarningDays, d.DefaultFlagBlocksPayer, d.DirectoryVisibility
	p.ResponsePeriodDays, p.ReReviewWindowDays, p.LimitWarningPercent = d.ResponsePeriodDays, d.ReReviewWindowDays, d.LimitWarningPercent
	return p, nil
}

func policyETag(n *domain.Network) string {
	return etagOf("policy", n.ID, strconv.Itoa(n.PolicyVersion))
}

type policyResp struct {
	Version     int       `json:"version"`
	EffectiveAt *string   `json:"effective_at"`
	Policy      policyDTO `json:"policy"`
}

// adminOnly is a cheap pre-check for admin-only reads whose service method
// would otherwise return a reduced view; the service remains authoritative.
func (c *reqCtx) adminOnly() error {
	if c.member == nil || c.member.Status != domain.MemberActive || !c.member.HasRole(domain.RoleAdmin) {
		return domain.Forbidden("administrator role required")
	}
	return nil
}

func (s *server) getPolicy(c *reqCtx) error { // API-058
	if err := c.adminOnly(); err != nil {
		return err
	}
	n, err := s.svc.GetNetwork(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	c.w.Header().Set("ETag", policyETag(n))
	return ok(c, 200, policyResp{Version: n.PolicyVersion, Policy: toPolicy(n.Policy)})
}

func (s *server) patchPolicy(c *reqCtx) error { // API-058
	if err := c.adminOnly(); err != nil {
		return err
	}
	b, err := c.body()
	if err != nil {
		return err
	}
	patch := map[string]json.RawMessage{}
	if len(b) == 0 {
		return invalidBody("a JSON request body is required")
	}
	if err := strictUnmarshal(b, &patch); err != nil {
		return err
	}
	var reason string
	if r, has := patch["reason"]; has {
		if json.Unmarshal(r, &reason) != nil {
			return validation("reason", "invalid-type", "must be a string")
		}
		delete(patch, "reason")
	}
	s.casMu.Lock()
	defer s.casMu.Unlock()
	n, err := s.svc.GetNetwork(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	cur := policyETag(n)
	if err := c.requireIfMatch(cur); err != nil {
		return err
	}
	base, _ := json.Marshal(toPolicy(n.Policy))
	merged := map[string]json.RawMessage{}
	_ = json.Unmarshal(base, &merged)
	for k, v := range patch { // RFC 7396 over a flat object
		if _, known := merged[k]; !known {
			return invalidBody("unknown policy field " + strconv.Quote(k))
		}
		if isNull(v) {
			return validation(k, "required", "policy fields cannot be removed")
		}
		merged[k] = v
	}
	mb, _ := json.Marshal(merged)
	var dto policyDTO
	if err := strictUnmarshal(mb, &dto); err != nil {
		return err
	}
	pol, err := dto.domain()
	if err != nil {
		return err
	}
	key := c.r.Header.Get("Idempotency-Key")
	if key == "" || !validKey(key) { // idempotent by ETag (API-058)
		key = "policy:" + cur + ":" + etagOf(string(b))
	}
	u, err := s.svc.UpdatePolicy(c.r.Context(), c.actor, c.netID, service.PolicyChange{Policy: pol, Reason: reason, IdempotencyKey: key})
	if err != nil {
		return err
	}
	eff := ts(s.opt.Now())
	c.w.Header().Set("ETag", policyETag(u))
	return ok(c, 200, policyResp{Version: u.PolicyVersion, EffectiveAt: &eff, Policy: toPolicy(u.Policy)})
}
