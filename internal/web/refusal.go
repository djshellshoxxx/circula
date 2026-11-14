package web

import (
	"errors"
	"fmt"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// fieldLabels turn service field names into words for error text.
var fieldLabels = map[string]string{
	"login": "Login", "password": "Password", "display_name": "Display name", "amount": "Amount",
	"counterparty_id": "Counterparty", "direction": "Direction", "title": "Title", "price": "Price",
	"reason": "Reason", "reason_code": "Reason for removal", "transaction_id": "Trade reference", "rationale": "Rationale", "category": "Category", "name": "Name", "slug": "Web address name",
	"limit": "Credit limit", "type": "Type", "outcome": "Outcome", "ttl": "Valid for", "status": "Status",
	"currency_name": "Credit unit name", "default_credit_limit": "Default credit limit", "max_tx_amount": "Largest trade",
	"proposal_expiry_days": "Proposal expiry (days)", "dispute_window_days": "Dispute window (days)",
	"join_mode": "Join mode", "departure_rule": "Departure rule", "signup_grant": "Sign-up grant",
	"limit_warning_days": "Limit warning (days)", "response_period_days": "Dispute response period", "re_review_window_days": "Re-review window", "limit_warning_percent": "Limit warning percent", "inactivity_days": "Inactivity flag", "waive_reason": "Reason for not waiting", "notice_days": "Notice (days)", "notice_reason": "Reason for shorter notice", "terms_version": "Terms", "body": "Terms text", "statement": "Statement", "note": "Note", "expires_at": "Expiry",
	"confirm_open_grant_risk": "Open joining with a sign-up grant",
	// UX-021: every other field the service can name in a validation error.
	"description": "Description", "response": "Response", "time_zone": "Time zone", "version": "Terms version",
	"effective_at": "Takes effect", "supersedes_id": "Replaces announcement", "listing_id": "Listing", "listing": "Listings",
	"target_type": "Report about", "target_id": "Reported item", "role": "Role", "scopes": "Token permissions",
	"code": "Code", "format": "File format", "scope": "Export type", "kind": "Challenge type", "decimals": "Decimal places",
	"dual_control_threshold": "Dual-control threshold", "threshold": "Dual-control threshold", "report": "Ledger report",
	"page": "Page", "idempotency_key": "Form key", "operator": "Operator", "directory_visibility": "Directory visibility",
	"ids": "Notices",
}

func label(f string) string {
	if l, ok := fieldLabels[f]; ok {
		return l
	}
	return strings.ReplaceAll(f, "_", " ")
}

// choiceFixes are the corrections for fields filled from a fixed list (UX-021).
var choiceFixes = map[string]string{
	"counterparty_id": "Choose another active member from the list.",
	"direction":       "Choose whether you pay or are paid.",
	"category":        "Choose one of the listed categories.",
	"outcome":         "Choose one of the listed outcomes.",
	"type":            "Choose one of the listed types.",
	"status":          "Choose one of the listed statuses.",
	"format":          "Choose CSV or JSON.",
	"listing_id":      "Start again from the other member's listing, or propose without one.",
	"terms_version":   "Read the terms shown, tick the box to accept them and try again.",
}

// fieldFix suggests how to correct one field error (UX-021: identify the field, describe the
// problem, suggest a correction). The service detail describes the problem; this says what to
// do about it, from the kind of problem first and then from the field.
func (c *rc) fieldFix(f domain.FieldError) string {
	d := strings.ToLower(f.Detail)
	switch {
	case strings.HasPrefix(d, "required"):
		return "Fill it in and try again."
	case strings.Contains(d, "is not correct"):
		return "Type it again carefully and try again."
	case strings.Contains(d, "too long"):
		return "Shorten it and try again."
	case strings.Contains(d, "control") && strings.Contains(d, "characters"):
		return "Remove hidden or special characters and try again."
	case strings.Contains(d, "choose another"):
		return "Choose a different one and try again."
	case strings.Contains(d, "decimal places") && c != nil && c.Net != nil:
		return fmt.Sprintf("Use at most %d decimal places and try again.", c.Net.Policy.Decimals)
	case strings.Contains(d, "not a number") && c != nil && c.Net != nil:
		return "Enter digits only, for example " + domain.FormatAmount(1250, c.Net.Policy.Decimals) + "."
	case d == "too large":
		return "Enter a smaller amount and try again."
	}
	if fix, ok := choiceFixes[f.Field]; ok {
		return fix
	}
	return "Check the value and try again."
}

// refuse turns a service error into summary text and field errors.
// ok is false when the error is not a form-level refusal (use fail instead).
func (c *rc) refuse(err error) (summary string, fields []fieldErr, ok bool) {
	var de *domain.Error
	if !errors.As(err, &de) {
		return "", nil, false
	}
	switch de.Code {
	case domain.CodeValidation:
		for _, f := range de.Fields {
			fields = append(fields, fieldErr{Field: f.Field, Text: fmt.Sprintf("%s: %s. %s", label(f.Field), f.Detail, c.fieldFix(f))})
		}
		if len(fields) == 0 { // no field to point at, for example an amount the accounts cannot record
			if strings.Contains(de.Msg, "overflows") {
				return "Not done. The amount is too large to record on these accounts. Try a smaller amount. Nothing changed.", nil, true
			}
			return "Not saved. Something entered was not accepted. Check the form and try again. Nothing changed.", nil, true
		}
		return "Not saved. Fix the problems below. Nothing changed.", fields, true
	case domain.CodeInsufficientCredit:
		extra := ""
		if b, e := c.s.svc.GetBalance(c.ctx, c.actor, c.netID()); e == nil {
			extra = " Available to spend is " + plainAmt(b.Available, b.Decimals) + "."
		}
		return "Not sent. This trade would take your available credit below zero." + extra + " Try a smaller amount. Nothing changed.", nil, true
	case domain.CodeAmountExceeds:
		return "Not sent. The amount is above the largest single trade this network allows (" +
			plainAmt(c.Net.Policy.MaxTxAmount, c.Net.Policy.Decimals) + "). Nothing changed.", nil, true
	case domain.CodeCounterpartyInact:
		return "Not sent. The other member is not active in this network, so they cannot take part in a trade. Nothing changed.", nil, true
	case domain.CodeKeyReused:
		return "Not sent. This form was already used with different details. Reload the page and start again. Nothing changed.", nil, true
	case domain.CodeKeyRequired:
		return "Not sent. The form is missing its safety key. Reload the page and try again. Nothing changed.", nil, true
	case domain.CodeInvalidState:
		if strings.Contains(de.Msg, "network is suspended") {
			return "Not done. Trading in this network is paused (suspended). You can still read balances and history and download your data. Nothing changed.", nil, true
		}
		return "Not done. This item is no longer in a state that allows that. Reload to see its current state. Nothing changed.", nil, true
	case domain.CodeConflict:
		return "Not saved. That value is not available. Choose a different one. Nothing changed.", nil, true
	case domain.CodeBlocked:
		return "Not done. " + strings.Join(de.Blockers, "; ") + ". Nothing changed.", nil, true
	case domain.CodeRateLimited:
		return "Too many attempts. Wait a minute and try again. Nothing changed.", nil, true
	case domain.CodeApprovalRequired: // for example removing a member whose settlement is above the dual-control threshold
		return "Not done. The amount is above the dual-control threshold, so one administrator cannot move it alone. " +
			"Settle it first with an adjustment that a second administrator approves (Adjustments), then try again. Nothing changed.", nil, true
	}
	return "", nil, false
}

func plainAmt(v int64, dec int) string { return domain.FormatAmount(v, dec) }
