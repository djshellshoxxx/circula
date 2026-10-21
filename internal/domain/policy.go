package domain

import "fmt"

// JoinMode determines how a person becomes a member (DOM-003).
type JoinMode string

// Join modes.
const (
	JoinInvite   JoinMode = "invite"
	JoinApproval JoinMode = "approval"
	JoinOpen     JoinMode = "open"
)

// DepartureRule says what happens to a departing member's balance (DOM-024).
type DepartureRule string

// Departure rules.
const (
	DepartRetain       DepartureRule = "retain"
	DepartSettleToZero DepartureRule = "settle-to-zero"
)

// Policy is the set of configurable network rules (DOM-029). It never holds
// an invariant (DOM-028). Monetary values are minor units.
type Policy struct {
	CurrencyName           string        `json:"currency_name"`
	Decimals               int           `json:"decimals"`
	DefaultCreditLimit     int64         `json:"default_credit_limit"`
	MaxTxAmount            int64         `json:"max_tx_amount"`
	ProposalExpiryDays     int           `json:"proposal_expiry_days"`
	DisputeWindowDays      int           `json:"dispute_window_days"`
	JoinMode               JoinMode      `json:"join_mode"`
	DepartureRule          DepartureRule `json:"departure_rule"`
	SignupGrant            int64         `json:"signup_grant"` // 0 = disabled
	LimitWarningDays       int           `json:"limit_warning_days"`
	DefaultFlagBlocksPayer bool          `json:"default_flag_blocks_payer"`
	DirectoryVisibility    string        `json:"directory_visibility"`
	// DualControlThreshold: administrator adjustments above this amount need a
	// second administrator (SEC-011). 0 = disabled (the default).
	DualControlThreshold int64 `json:"dual_control_threshold,omitempty"`
	// ResponsePeriodDays: days the other party has to answer a dispute before a
	// decision may be recorded without a response (FR-076, OQ-FR-2; DEC-012
	// recommends 7). 0 = no waiting period.
	ResponsePeriodDays int `json:"response_period_days"`
	// ReReviewWindowDays: days after a decision in which one re-review may be
	// requested (FR-078).
	ReReviewWindowDays int `json:"re_review_window_days"`
	// LimitWarningPercent: share of the credit limit in use at which a member is
	// warned (UX-012, OQ-UX-02).
	LimitWarningPercent int `json:"limit_warning_percent"`
}

// Platform bounds for open questions OQ-DOM-5/6.
const (
	MaxProposalExpiryDays = 365
	MaxDisputeWindowDays  = 3650
	MaxLimitWarningDays   = 3650
	MaxResponsePeriodDays = 90
	MaxReReviewWindowDays = 365
	// MaxPolicyAmount keeps minor-unit magnitudes far from int64 overflow.
	MaxPolicyAmount int64 = 1 << 50
)

// DefaultPolicy returns the defaults from docs/specs/00-conventions.md s5.
func DefaultPolicy() Policy {
	return Policy{
		CurrencyName:        "credit",
		Decimals:            2,
		DefaultCreditLimit:  0,
		MaxTxAmount:         100_000_000,
		ProposalExpiryDays:  7,
		DisputeWindowDays:   30,
		ResponsePeriodDays:  7,
		ReReviewWindowDays:  14,
		LimitWarningPercent: 80,
		JoinMode:            JoinInvite,
		DepartureRule:       DepartRetain,
		DirectoryVisibility: "members-only",
	}
}

// Validate checks every bound and returns a validation error naming each
// offending field.
func (p Policy) Validate() error {
	var fe []FieldError
	add := func(f, d string) { fe = append(fe, FieldError{f, d}) }
	if p.CurrencyName == "" || len(p.CurrencyName) > 40 {
		add("currency_name", "must be 1 to 40 characters")
	}
	if p.Decimals < 0 || p.Decimals > MaxDecimals {
		add("decimals", fmt.Sprintf("must be 0 to %d", MaxDecimals))
	}
	if p.DefaultCreditLimit < 0 || p.DefaultCreditLimit > MaxPolicyAmount {
		add("default_credit_limit", "must be >= 0 and within platform maximum")
	}
	if p.MaxTxAmount < 1 || p.MaxTxAmount > MaxPolicyAmount {
		add("max_tx_amount", "must be >= 1 minor unit and within platform maximum")
	}
	if p.ProposalExpiryDays < 1 || p.ProposalExpiryDays > MaxProposalExpiryDays {
		add("proposal_expiry_days", fmt.Sprintf("must be 1 to %d", MaxProposalExpiryDays))
	}
	if p.DisputeWindowDays < 1 || p.DisputeWindowDays > MaxDisputeWindowDays {
		add("dispute_window_days", fmt.Sprintf("must be 1 to %d", MaxDisputeWindowDays))
	}
	switch p.JoinMode {
	case JoinInvite, JoinApproval, JoinOpen:
	default:
		add("join_mode", "must be invite, approval or open")
	}
	switch p.DepartureRule {
	case DepartRetain, DepartSettleToZero:
	default:
		add("departure_rule", "must be retain or settle-to-zero")
	}
	if p.SignupGrant < 0 || p.SignupGrant > MaxPolicyAmount {
		add("signup_grant", "must be 0 (disabled) or a positive amount")
	}
	if p.LimitWarningDays < 0 || p.LimitWarningDays > MaxLimitWarningDays {
		add("limit_warning_days", fmt.Sprintf("must be 0 (disabled) to %d", MaxLimitWarningDays))
	}
	if p.DualControlThreshold < 0 || p.DualControlThreshold > MaxPolicyAmount {
		add("dual_control_threshold", "must be 0 (disabled) or a positive amount within platform maximum")
	}
	if p.ResponsePeriodDays < 0 || p.ResponsePeriodDays > MaxResponsePeriodDays {
		add("response_period_days", fmt.Sprintf("must be 0 (no waiting period) to %d", MaxResponsePeriodDays))
	}
	if p.ReReviewWindowDays < 1 || p.ReReviewWindowDays > MaxReReviewWindowDays {
		add("re_review_window_days", fmt.Sprintf("must be 1 to %d", MaxReReviewWindowDays))
	}
	if p.LimitWarningPercent < 1 || p.LimitWarningPercent > 100 {
		add("limit_warning_percent", "must be 1 to 100")
	}
	if p.DirectoryVisibility != "members-only" {
		add("directory_visibility", "only members-only is supported")
	}
	if len(fe) > 0 {
		return &Error{Code: CodeValidation, Msg: "invalid policy", Fields: fe}
	}
	return nil
}
