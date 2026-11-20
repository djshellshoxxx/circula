package web

import (
	"errors"
	"regexp"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// fieldLabels turn service field names into words for error text.
var fieldLabels = map[string]string{
	"cursor":                  messages.Text("validation.label.cursor"),
	"login":                   messages.Text("validation.label.login"),
	"password":                messages.Text("validation.label.password"),
	"display_name":            messages.Text("validation.label.display_name"),
	"amount":                  messages.Text("validation.label.amount"),
	"counterparty_id":         messages.Text("validation.label.counterparty_id"),
	"direction":               messages.Text("validation.label.direction"),
	"title":                   messages.Text("validation.label.title"),
	"price":                   messages.Text("validation.label.price"),
	"member_id":               messages.Text("validation.label.member_id"),
	"related_transaction_id":  messages.Text("validation.label.related_transaction_id"),
	"reason":                  messages.Text("validation.label.reason"),
	"reason_code":             messages.Text("validation.label.reason_code"),
	"transaction_id":          messages.Text("validation.label.transaction_id"),
	"dispute_id":              messages.Text("validation.label.dispute_id"),
	"rationale":               messages.Text("validation.label.rationale"),
	"category":                messages.Text("validation.label.category"),
	"name":                    messages.Text("validation.label.name"),
	"slug":                    messages.Text("validation.label.slug"),
	"limit":                   messages.Text("validation.label.limit"),
	"type":                    messages.Text("validation.label.type"),
	"outcome":                 messages.Text("validation.label.outcome"),
	"ttl":                     messages.Text("validation.label.ttl"),
	"status":                  messages.Text("validation.label.status"),
	"currency_name":           messages.Text("validation.label.currency_name"),
	"default_credit_limit":    messages.Text("validation.label.default_credit_limit"),
	"max_tx_amount":           messages.Text("validation.label.max_tx_amount"),
	"proposal_expiry_days":    messages.Text("validation.label.proposal_expiry_days"),
	"dispute_window_days":     messages.Text("validation.label.dispute_window_days"),
	"join_mode":               messages.Text("validation.label.join_mode"),
	"departure_rule":          messages.Text("validation.label.departure_rule"),
	"signup_grant":            messages.Text("validation.label.signup_grant"),
	"limit_warning_days":      messages.Text("validation.label.limit_warning_days"),
	"response_period_days":    messages.Text("validation.label.response_period_days"),
	"re_review_window_days":   messages.Text("validation.label.re_review_window_days"),
	"limit_warning_percent":   messages.Text("validation.label.limit_warning_percent"),
	"inactivity_days":         messages.Text("validation.label.inactivity_days"),
	"waive_reason":            messages.Text("validation.label.waive_reason"),
	"notice_days":             messages.Text("validation.label.notice_days"),
	"notice_reason":           messages.Text("validation.label.notice_reason"),
	"terms_version":           messages.Text("validation.label.terms_version"),
	"body":                    messages.Text("validation.label.body"),
	"statement":               messages.Text("validation.label.statement"),
	"note":                    messages.Text("validation.label.note"),
	"expires_at":              messages.Text("validation.label.expires_at"),
	"confirm_open_grant_risk": messages.Text("validation.label.confirm_open_grant_risk"),
	// UX-021: every other field the service can name in a validation error.
	"description":            messages.Text("validation.label.description"),
	"response":               messages.Text("validation.label.response"),
	"time_zone":              messages.Text("validation.label.time_zone"),
	"version":                messages.Text("validation.label.version"),
	"effective_at":           messages.Text("validation.label.effective_at"),
	"supersedes_id":          messages.Text("validation.label.supersedes_id"),
	"listing_id":             messages.Text("validation.label.listing_id"),
	"listing":                messages.Text("validation.label.listing"),
	"target_type":            messages.Text("validation.label.target_type"),
	"target_id":              messages.Text("validation.label.target_id"),
	"role":                   messages.Text("validation.label.role"),
	"scopes":                 messages.Text("validation.label.scopes"),
	"code":                   messages.Text("validation.label.code"),
	"format":                 messages.Text("validation.label.format"),
	"scope":                  messages.Text("validation.label.scope"),
	"kind":                   messages.Text("validation.label.kind"),
	"decimals":               messages.Text("validation.label.decimals"),
	"dual_control_threshold": messages.Text("validation.label.dual_control_threshold"),
	"threshold":              messages.Text("validation.label.threshold"),
	"report":                 messages.Text("validation.label.report"),
	"page":                   messages.Text("validation.label.page"),
	"idempotency_key":        messages.Text("validation.label.idempotency_key"),
	"operator":               messages.Text("validation.label.operator"),
	"directory_visibility":   messages.Text("validation.label.directory_visibility"),
	"ids":                    messages.Text("validation.label.ids"),
}

func label(f string) string {
	if l, ok := fieldLabels[f]; ok {
		return l
	}
	return strings.ReplaceAll(f, "_", " ")
}

// choiceFixes are the corrections for fields filled from a fixed list (UX-021).
var choiceFixes = map[string]string{
	"counterparty_id":         messages.Text("validation.choice.counterparty_id"),
	"member_id":               messages.Text("validation.choice.member_id"),
	"related_transaction_id":  messages.Text("validation.choice.related_transaction_id"),
	"direction":               messages.Text("validation.choice.direction"),
	"category":                messages.Text("validation.choice.category"),
	"outcome":                 messages.Text("validation.choice.outcome"),
	"type":                    messages.Text("validation.choice.type"),
	"status":                  messages.Text("validation.choice.status"),
	"format":                  messages.Text("validation.choice.format"),
	"listing_id":              messages.Text("validation.choice.listing_id"),
	"terms_version":           messages.Text("validation.choice.terms_version"),
	"reason_code":             messages.Text("validation.choice.reason_code"),
	"confirm_open_grant_risk": messages.Text("validation.choice.confirm_open_grant_risk"),
}

// machineChoices matches a service detail that lists a choice field's stored values
// ("required, one of rule_breach, confirmed_fraud", "must be pay or request").
var machineChoices = regexp.MustCompile(`^(required, )?(must be )?(one of )?[a-z_]+((, | or )[a-z_]+)+$`)

// fieldDetail is the problem description shown for f. For a field filled from a list, a
// detail that only lists the stored values is put in plain words instead, so the member
// never sees values such as "rule_breach" (UX-020, UX-021).
func fieldDetail(f domain.FieldError) string {
	if _, choice := choiceFixes[f.Field]; choice && machineChoices.MatchString(f.Detail) {
		if strings.HasPrefix(f.Detail, "required") {
			return messages.Text("validation.detail.choice_required")
		}
		return messages.Text("validation.detail.choice_invalid")
	}
	return f.Detail
}

// fieldFix suggests how to correct one field error (UX-021: identify the field, describe the
// problem, suggest a correction). The service detail describes the problem; this says what to
// do about it, from the kind of problem first and then from the field. A field filled from a
// list (choiceFixes) always gets its own correction: "fill it in" does not fit a list.
func (c *rc) fieldFix(f domain.FieldError) string {
	if fix, ok := choiceFixes[f.Field]; ok {
		return fix
	}
	d := strings.ToLower(f.Detail)
	switch {
	case strings.HasPrefix(d, "required"):
		return messages.Text("validation.fix.required")
	case strings.Contains(d, "is not correct"):
		return messages.Text("validation.fix.incorrect")
	case strings.Contains(d, "too long"):
		return messages.Text("validation.fix.too_long")
	case strings.Contains(d, "control") && strings.Contains(d, "characters"):
		return messages.Text("validation.fix.control_characters")
	case strings.Contains(d, "choose another"):
		return messages.Text("validation.fix.different")
	case strings.Contains(d, "decimal place") && c != nil && c.Net != nil:
		return messages.Text("validation.fix.decimals", c.Net.Policy.Decimals)
	case strings.Contains(d, "not a number") && c != nil && c.Net != nil:
		return messages.Text("validation.fix.number", domain.FormatAmount(1250, c.Net.Policy.Decimals))
	case d == "too large":
		return messages.Text("validation.fix.smaller")
	}
	return messages.Text("validation.fix.check")
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
			fields = append(fields, fieldErr{Field: f.Field, Text: messages.Text("validation.field.message", label(f.Field), fieldDetail(f), c.fieldFix(f))})
		}
		if len(fields) == 0 { // no field to point at, for example an amount the accounts cannot record
			if strings.Contains(de.Msg, "overflows") {
				return messages.Text("validation.refusal.overflow"), nil, true
			}
			return messages.Text("validation.refusal.no_fields"), nil, true
		}
		return messages.Text("validation.refusal.fields"), fields, true
	case domain.CodeInsufficientCredit:
		if b, e := c.s.svc.GetBalance(c.ctx, c.actor, c.netID()); e == nil {
			return messages.Text("validation.refusal.insufficient_credit_available", plainAmt(b.Available, b.Decimals)), nil, true
		}
		return messages.Text("validation.refusal.insufficient_credit"), nil, true
	case domain.CodeAmountExceeds:
		return messages.Text("validation.refusal.amount_exceeds", plainAmt(c.Net.Policy.MaxTxAmount, c.Net.Policy.Decimals)), nil, true
	case domain.CodeCounterpartyInact:
		return messages.Text("validation.refusal.counterparty_inactive"), nil, true
	case domain.CodeKeyReused:
		return messages.Text("validation.refusal.key_reused"), nil, true
	case domain.CodeKeyRequired:
		return messages.Text("validation.refusal.key_required"), nil, true
	case domain.CodeInvalidState:
		if strings.Contains(de.Msg, "network is suspended") {
			return messages.Text("validation.refusal.suspended"), nil, true
		}
		return messages.Text("validation.refusal.invalid_state"), nil, true
	case domain.CodeConflict:
		return messages.Text("validation.refusal.conflict"), nil, true
	case domain.CodeBlocked:
		return messages.Text("validation.refusal.blocked", strings.Join(de.Blockers, "; ")), nil, true
	case domain.CodeRateLimited:
		return messages.Text("validation.refusal.rate_limited"), nil, true
	case domain.CodeApprovalRequired: // for example removing a member whose settlement is above the dual-control threshold
		return messages.Text("validation.refusal.approval_required"), nil, true
	}
	return "", nil, false
}

func plainAmt(v int64, dec int) string { return domain.FormatAmount(v, dec) }
