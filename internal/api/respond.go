package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

// apiError is the transport form of every failure (RFC 9457, API-029..032).
type apiError struct {
	status        int
	code          string
	detail        string
	errors        []fieldProblem
	blockers      []string
	currentStatus string
	retryAfter    string
}

func (e *apiError) Error() string { return e.code + ": " + e.detail }

type fieldProblem struct {
	Pointer string `json:"pointer"`
	Code    string `json:"code"`
	Detail  string `json:"detail"`
}

type problemBody struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance"`
	Code          string         `json:"code"`
	RequestID     string         `json:"request_id"`
	Errors        []fieldProblem `json:"errors,omitempty"`
	Blockers      []string       `json:"blockers,omitempty"`
	CurrentStatus string         `json:"current_status,omitempty"`
}

var titles = map[string]string{
	"invalid-body": "Invalid request body", "invalid-query": "Invalid query parameter",
	"invalid-cursor": "Invalid cursor", "credentials-in-query": "Credentials in URL",
	"idempotency-key-required": "Idempotency-Key required", "validation-failed": "Validation failed",
	"insufficient-available-credit": "Insufficient available credit", "amount-exceeds-policy": "Amount exceeds policy",
	"counterparty-not-active": "Counterparty not active", "idempotency-key-reused": "Idempotency key reused",
	"unauthenticated": "Authentication required", "forbidden": "Forbidden", "csrf-invalid": "CSRF token invalid",
	"not-found": "Not found", "invalid-transaction-state": "Invalid state for this action",
	"idempotency-request-in-progress": "Request in progress", "precondition-failed": "Precondition failed",
	"precondition-required": "Precondition required", "payload-too-large": "Payload too large",
	"unsupported-media-type": "Unsupported media type", "rate-limited": "Rate limit exceeded",
	"internal-error": "Internal error", "unavailable": "Service unavailable", "conflict": "Conflict",
	"blocked": "Action blocked", "method-not-allowed": "Method not allowed", "not-implemented": "Not implemented",
	"network-required": "Network required", "invalid-header": "Invalid request header", "member-has-open-balance": "Member has open balance",
}

var (
	errInternal = &apiError{status: 500, code: "internal-error", detail: "an unexpected error occurred"}
	errTooLarge = &apiError{status: 413, code: "payload-too-large", detail: "request body exceeds the size limit"}
	errUnauth   = &apiError{status: 401, code: "unauthenticated", detail: "valid credentials are required"}
)

func invalidBody(detail string) *apiError {
	return &apiError{status: 400, code: "invalid-body", detail: detail}
}

func invalidQuery(detail string) *apiError {
	return &apiError{status: 400, code: "invalid-query", detail: detail}
}

func notFoundErr() *apiError {
	return &apiError{status: 404, code: "not-found", detail: "resource not found"}
}

func notImplemented(what string) *apiError {
	return &apiError{status: 501, code: "not-implemented", detail: what + " is not supported by the service layer yet"}
}

func validation(field, code, detail string) *apiError {
	return &apiError{status: 422, code: "validation-failed", detail: "one or more fields are invalid",
		errors: []fieldProblem{{Pointer: pointerFor(field), Code: code, Detail: detail}}}
}

// fieldNames maps service field names to the API's field names.
var fieldNames = map[string]string{
	"amount": "amount_minor", "counterparty_id": "counterparty_member_id", "note": "description",
	"price": "price_minor", "type": "kind", "rationale": "reason", "default_credit_limit": "default_credit_limit_minor",
	"max_tx_amount": "max_tx_amount_minor", "signup_grant": "signup_grant_minor", "limit": "credit_limit_minor",
	"tx_id": "transaction_id",
}

func pointerFor(field string) string {
	if n, ok := fieldNames[field]; ok {
		field = n
	}
	return "/" + field
}

var codeStatus = map[domain.Code]int{
	domain.CodeValidation: 422, domain.CodeInsufficientCredit: 422, domain.CodeAmountExceeds: 422,
	domain.CodeCounterpartyInact: 422, domain.CodeKeyReused: 422, domain.CodeKeyRequired: 400,
	domain.CodeUnauthenticated: 401, domain.CodeForbidden: 403, domain.CodeNotFound: 404,
	domain.CodeInvalidState: 409, domain.CodeConflict: 409, domain.CodeBlocked: 409, domain.CodeRateLimited: 429, domain.CodeUnavailable: 503,
	domain.CodePasswordChangeRequired: 403,
}

// fromErr converts any error into an apiError. Unknown errors become opaque
// internal errors (API-032); the caller logs the cause.
func fromErr(err error) (*apiError, bool) {
	var ae *apiError
	if errors.As(err, &ae) {
		return ae, true
	}
	if httpx.IsBusy(err) { // lock contention is retryable, not a fault (OPS-011)
		return &apiError{status: 503, code: "unavailable", detail: "the service is busy; retry shortly", retryAfter: "5"}, true
	}
	var de *domain.Error
	if errors.As(err, &de) {
		st, ok := codeStatus[de.Code]
		if !ok {
			return errInternal, false
		}
		out := &apiError{status: st, code: string(de.Code), detail: de.Msg, blockers: de.Blockers}
		if out.detail == "" {
			out.detail = titles[out.code]
		}
		for _, f := range de.Fields {
			out.errors = append(out.errors, fieldProblem{Pointer: pointerFor(f.Field), Code: "invalid", Detail: f.Detail})
		}
		if out.status == 429 {
			out.retryAfter = "60"
		}
		if out.status == 503 {
			out.retryAfter = "5" // retryable (CMD-05)
		}
		return out, true
	}
	return errInternal, false
}

func writeProblem(w http.ResponseWriter, r *http.Request, rid string, e *apiError) {
	title := titles[e.code]
	if title == "" {
		title = http.StatusText(e.status)
	}
	if e.retryAfter != "" {
		w.Header().Set("Retry-After", e.retryAfter)
	}
	if e.status == 401 {
		w.Header().Set("WWW-Authenticate", `Bearer realm="circula"`)
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(e.status)
	_ = json.NewEncoder(w).Encode(problemBody{Type: "urn:circula:problem:" + e.code, Title: title, Status: e.status,
		Detail: e.detail, Instance: service.RedactSecrets(r.URL.Path), Code: e.code, RequestID: rid, Errors: e.errors,
		Blockers: e.blockers, CurrentStatus: e.currentStatus})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
