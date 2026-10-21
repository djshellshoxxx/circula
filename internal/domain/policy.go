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
}

// Platform bounds for open questions OQ-DOM-5/6.
const (
	MaxProposalExpiryDays = 365
	MaxDisputeWindowDays  = 3650
	MaxLimitWarningDays   = 3650
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
	if p.DirectoryVisibility != "members-only" {
		add("directory_visibility", "only members-only is supported")
	}
	if len(fe) > 0 {
		return &Error{Code: CodeValidation, Msg: "invalid policy", Fields: fe}
	}
	return nil
}
