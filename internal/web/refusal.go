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
	"reason": "Reason", "rationale": "Rationale", "category": "Category", "name": "Name", "slug": "Web address name",
	"limit": "Credit limit", "type": "Type", "outcome": "Outcome", "ttl": "Valid for", "status": "Status",
	"currency_name": "Credit unit name", "default_credit_limit": "Default credit limit", "max_tx_amount": "Largest trade",
	"proposal_expiry_days": "Proposal expiry (days)", "dispute_window_days": "Dispute window (days)",
	"join_mode": "Join mode", "departure_rule": "Departure rule", "signup_grant": "Sign-up grant",
	"limit_warning_days": "Limit warning (days)", "statement": "Statement", "note": "Note", "expires_at": "Expiry",
}

func label(f string) string {
	if l, ok := fieldLabels[f]; ok {
		return l
	}
	return strings.ReplaceAll(f, "_", " ")
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
			fields = append(fields, fieldErr{Field: f.Field, Text: fmt.Sprintf("%s: %s. Check the value and try again.", label(f.Field), f.Detail)})
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
		return "Not done. This item is no longer in a state that allows that. Reload to see its current state. Nothing changed.", nil, true
	case domain.CodeConflict:
		return "Not saved. That value is not available. Choose a different one. Nothing changed.", nil, true
	case domain.CodeBlocked:
		return "Not done. " + strings.Join(de.Blockers, "; ") + ". Nothing changed.", nil, true
	case domain.CodeRateLimited:
		return "Too many attempts. Wait a minute and try again. Nothing changed.", nil, true
	}
	return "", nil, false
}

func plainAmt(v int64, dec int) string { return domain.FormatAmount(v, dec) }
