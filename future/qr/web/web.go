// Package web is the unregistered HTTP surface of the QR module: a JSON API
// that follows Circula's conventions (RFC 9457 problems, CSRF for cookie
// sessions, no-store, 64 KiB bodies, Idempotency-Key) and server-rendered,
// no-JavaScript HTML pages with an optional camera enhancement script.
//
// NewHandler returns an http.Handler. Nothing here registers routes with any
// application; the host mounts the handler and supplies the Authenticator.
package web

import (
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	qr "github.com/djshellshoxxx/circula/future/qr"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// ErrUnauthenticated is returned by an Authenticator when there is no valid session.
var ErrUnauthenticated = errors.New("unauthenticated")

// Session is the host's authenticated caller for one request.
type Session struct {
	Principal qr.Principal // network from X-Network selection, member from the session or token
	// CookieAuth must be true for cookie sessions: unsafe methods then require the CSRF token.
	CookieAuth bool
	// CSRF is the session's expected token (compared constant-time to X-CSRF-Token or the csrf form field).
	CSRF string
	// DisplayName and NetworkName are used in page text (optional).
	DisplayName string
	NetworkName string
}

// Authenticator resolves the caller. It owns authentication, X-Network
// scoping, session expiry and suspension checks that precede this module.
type Authenticator interface {
	Authenticate(r *http.Request) (Session, error)
}

// Config configures NewHandler.
type Config struct {
	Service *qr.Service
	Auth    Authenticator
	// LoginURL is where unauthenticated HTML requests are sent (303). It is a
	// fixed host-supplied path; no request data is copied into it. Empty = 401 page.
	LoginURL string
	// FormatAmount renders minor units for display, given the network id. Default: "1,234 units".
	FormatAmount func(networkID string, minor int64) string
	// OnError receives unexpected (5xx) errors with the path already redacted.
	OnError func(redactedPath string, err error)
	// MaxBody is the request body limit (default 64 KiB).
	MaxBody int64
	Now     func() time.Time
}

type handler struct {
	cfg   Config
	svc   *qr.Service
	pages map[string]*template.Template
	mux   *http.ServeMux
}

// NewHandler builds the handler.
func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.Service == nil || cfg.Auth == nil {
		return nil, errors.New("web: Service and Auth are required")
	}
	if cfg.MaxBody <= 0 {
		cfg.MaxBody = 64 << 10
	}
	if cfg.FormatAmount == nil {
		cfg.FormatAmount = func(_ string, v int64) string { return groupDigits(v) + " units" }
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	h := &handler{cfg: cfg, svc: cfg.Service, mux: http.NewServeMux()}
	if err := h.loadTemplates(); err != nil {
		return nil, err
	}
	h.routes()
	return h, nil
}

func durationSeconds(n int64) time.Duration { return time.Duration(n) * time.Second }

func groupDigits(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func (h *handler) routes() {
	m := h.mux
	// JSON API (proposed contract; see openapi.yaml)
	m.HandleFunc("GET /api/v1/me/qr", h.api(h.apiMyQR))
	m.HandleFunc("GET /api/v1/me/qr.png", h.api(h.apiMyQRPNG))
	m.HandleFunc("POST /api/v1/me/qr/rotate", h.api(h.apiRotate))
	m.HandleFunc("POST /api/v1/qr-requests", h.api(h.apiCreateRequest))
	m.HandleFunc("GET /api/v1/qr-requests", h.api(h.apiListRequests))
	m.HandleFunc("GET /api/v1/qr-requests/{ref}", h.api(h.apiGetRequest))
	m.HandleFunc("POST /api/v1/qr-requests/{id}/revoke", h.api(h.apiRevoke))
	m.HandleFunc("GET /api/v1/qr-aliases/{code}", h.api(h.apiResolveAlias))
	m.HandleFunc("POST /api/v1/qr-resolve", h.api(h.apiResolve))
	m.HandleFunc("POST /api/v1/qr-claims", h.api(h.apiClaim))
	// HTML (no JavaScript required)
	m.HandleFunc("GET /qr", h.html(h.pageHome))
	m.HandleFunc("GET /qr/me.png", h.html(h.pageMyPNG))
	m.HandleFunc("GET /qr/print", h.html(h.pagePrint))
	m.HandleFunc("GET /qr/scan", h.html(h.pageScan))
	m.HandleFunc("POST /qr/manual", h.html(h.pageManual))
	m.HandleFunc("GET /qr/v1/a/{code}", h.html(h.pageAlias))
	m.HandleFunc("GET /qr/v1/{code}", h.html(h.pageRequest))
	m.HandleFunc("POST /qr/confirm", h.html(h.pageConfirm))
	m.HandleFunc("POST /qr/rotate", h.html(h.pageRotate))
	m.HandleFunc("POST /qr/requests", h.html(h.pageCreate))
	m.HandleFunc("GET /qr/requests/{id}", h.html(h.pageStatus))
	m.HandleFunc("POST /qr/requests/{id}/revoke", h.html(h.pageRevoke))
	m.HandleFunc("GET /qr/static/", h.static)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *handler) fail(r *http.Request, err error) {
	if h.cfg.OnError != nil {
		h.cfg.OnError(qr.RedactPath(r.URL.Path), err)
	}
}

// secureHeaders applies the privacy headers to every response.
func secureHeaders(w http.ResponseWriter) {
	hd := w.Header()
	hd.Set("Cache-Control", "no-store")
	hd.Set("Pragma", "no-cache")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Referrer-Policy", "no-referrer")
	hd.Set("Cross-Origin-Resource-Policy", "same-origin")
}

// ---- errors ------------------------------------------------------------

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Code   string `json:"code"`
}

type mapped struct {
	status int
	code   string
	title  string
	detail string
}

var generic = map[string]string{
	"not_found":          "No matching QR code or request was found in this network.",
	"forbidden":          "You are not allowed to do this.",
	"expired":            "This request has expired.",
	"revoked":            "This code has been revoked.",
	"claimed":            "This request has already been claimed.",
	"idempotency":        "This idempotency key was already used for a different request.",
	"rate_limited":       "Too many active codes. Revoke or wait for one to expire.",
	"insufficient":       "Your available credit does not cover this amount.",
	"counterparty":       "The other member cannot trade right now.",
	"unauthenticated":    "Sign in to continue.",
	"csrf":               "The form expired. Reload the page and try again.",
	"internal":           "Something went wrong. Nothing was changed by this request.",
	"payload_too_large":  "The request body is too large.",
	"unsupported_media":  "Send application/json.",
	"method_not_allowed": "Method not allowed.",
}

func mapError(err error) mapped {
	m := func(status int, code, title string) mapped { return mapped{status, code, title, generic[code]} }
	switch {
	case errors.Is(err, qr.ErrInvalid):
		d := strings.TrimPrefix(err.Error(), qr.ErrInvalid.Error())
		d = strings.TrimPrefix(d, ": ")
		if d == "" || len(d) > 200 || strings.Contains(d, "\n") {
			d = "The request is not valid."
		}
		return mapped{http.StatusBadRequest, "invalid_request", "Invalid request", d}
	case errors.Is(err, qr.ErrNotFound):
		return m(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, qr.ErrForbidden):
		return m(http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, qr.ErrExpired):
		return m(http.StatusGone, "expired", "Request expired")
	case errors.Is(err, qr.ErrRevoked):
		return m(http.StatusGone, "revoked", "Code revoked")
	case errors.Is(err, qr.ErrClaimed):
		return m(http.StatusConflict, "claimed", "Already claimed")
	case errors.Is(err, qr.ErrIdempotencyConflict):
		return m(http.StatusUnprocessableEntity, "idempotency", "Idempotency key reuse")
	case errors.Is(err, qr.ErrRateLimited):
		return m(http.StatusTooManyRequests, "rate_limited", "Too many requests")
	case errors.Is(err, qr.ErrInsufficientCredit):
		return m(http.StatusUnprocessableEntity, "insufficient", "Insufficient available credit")
	case errors.Is(err, qr.ErrCounterpartyInactive):
		return m(http.StatusConflict, "counterparty", "Counterparty unavailable")
	case errors.Is(err, ErrUnauthenticated):
		return m(http.StatusUnauthorized, "unauthenticated", "Unauthenticated")
	case errors.Is(err, errCSRF):
		return m(http.StatusForbidden, "csrf", "CSRF check failed")
	case errors.Is(err, errTooLarge):
		return m(http.StatusRequestEntityTooLarge, "payload_too_large", "Request too large")
	case errors.Is(err, errMedia):
		return m(http.StatusUnsupportedMediaType, "unsupported_media", "Unsupported media type")
	}
	return m(http.StatusInternalServerError, "internal", "Internal error")
}

var (
	errCSRF     = errors.New("csrf")
	errTooLarge = errors.New("too large")
	errMedia    = errors.New("media type")
)

func (h *handler) problem(w http.ResponseWriter, r *http.Request, err error) {
	m := mapError(err)
	if m.status >= 500 {
		h.fail(r, err)
	}
	secureHeaders(w)
	w.Header().Set("Content-Type", "application/problem+json")
	if m.status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Session realm="circula"`)
	}
	w.WriteHeader(m.status)
	_ = json.NewEncoder(w).Encode(problem{Type: "https://circula.invalid/problems/qr/" + strings.ReplaceAll(m.code, "_", "-"), Title: m.title, Status: m.status, Detail: m.detail, Code: m.code})
}

// ---- API plumbing ------------------------------------------------------

type apiFn func(w http.ResponseWriter, r *http.Request, s Session) error

func (h *handler) api(fn apiFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := h.cfg.Auth.Authenticate(r)
		if err != nil {
			if !errors.Is(err, ErrUnauthenticated) {
				h.fail(r, err)
				err = ErrUnauthenticated
			}
			h.problem(w, r, err)
			return
		}
		if isUnsafe(r.Method) && s.CookieAuth {
			if !csrfOK(s.CSRF, r.Header.Get("X-CSRF-Token")) {
				h.problem(w, r, errCSRF)
				return
			}
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBody)
		}
		if err := fn(w, r, s); err != nil {
			h.problem(w, r, err)
		}
	}
}

func isUnsafe(m string) bool {
	return m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions
}

func csrfOK(want, got string) bool {
	return want != "" && len(want) == len(got) && subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func decodeJSON(r *http.Request, dst any) error {
	ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || ct != "application/json" {
		return errMedia
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return errTooLarge
		}
		return fmt.Errorf("%w: body must be a JSON object with known fields", qr.ErrInvalid)
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("%w: unexpected data after JSON object", qr.ErrInvalid)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	secureHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// ---- JSON views ----------------------------------------------------------

type aliasJSON struct {
	Generation  int       `json:"generation"`
	URL         string    `json:"url"`
	CompactCode string    `json:"compact_code"`
	CreatedAt   time.Time `json:"created_at"`
	Revoked     bool      `json:"revoked,omitempty"`
}

func aliasView(a qr.AliasInfo) aliasJSON {
	return aliasJSON{a.Generation, a.URL, a.Compact, a.CreatedAt, a.Revoked}
}

type requestJSON struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	AmountMinor   int64      `json:"amount_minor"`
	Note          string     `json:"note"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	Claimed       bool       `json:"claimed"`
	TransactionID string     `json:"transaction_id,omitempty"`
	// Completed is true only once the counterparty accepted and the ledger posted.
	Completed bool `json:"completed"`
}

func requestView(v qr.RequestView) requestJSON {
	return requestJSON{ID: v.ID, Status: string(v.Status), AmountMinor: v.AmountMinor, Note: v.Note, CreatedAt: v.CreatedAt, ExpiresAt: v.ExpiresAt,
		RevokedAt: v.RevokedAt, Claimed: v.Claimed, TransactionID: v.TransactionID, Completed: v.Status == qr.StatusCompleted}
}

type previewJSON struct {
	Kind            string    `json:"kind"`
	Network         string    `json:"network"`
	Payer           string    `json:"payer"`
	Payee           string    `json:"payee"`
	AmountMinor     int64     `json:"amount_minor,omitempty"`
	Note            string    `json:"note,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitzero"`
	RequestID       string    `json:"request_id,omitempty"`
	AvailableCredit int64     `json:"estimated_available_credit_minor"`
	Sufficient      *bool     `json:"sufficient,omitempty"`
	Own             bool      `json:"own"`
	Claimed         bool      `json:"claimed,omitempty"`
	ClaimedByYou    bool      `json:"claimed_by_you,omitempty"`
	TransactionID   string    `json:"transaction_id,omitempty"`
	Status          string    `json:"status,omitempty"`
	// Notice repeats the contract: scanning is not acceptance or payment.
	Notice string `json:"notice"`
}

const scanNotice = "Scanning only shows details. Nothing is paid until you confirm and the other member accepts."

func previewView(p qr.Preview) previewJSON {
	v := previewJSON{Kind: string(p.Kind), Network: p.NetworkName, Payer: p.PayerName, Payee: p.PayeeName, AmountMinor: p.AmountMinor, Note: p.Note,
		ExpiresAt: p.ExpiresAt, RequestID: p.RequestID, AvailableCredit: p.AvailableCredit, Own: p.Own, Claimed: p.Claimed, ClaimedByYou: p.ClaimedByYou,
		TransactionID: p.TransactionID, Status: string(p.Status), Notice: scanNotice}
	if p.Kind == qr.KindRequest {
		s := p.Sufficient
		v.Sufficient = &s
	}
	return v
}

type claimJSON struct {
	TransactionID string `json:"transaction_id"`
	// Status is "pending" until the counterparty accepts; only "completed" means posted.
	Status    string `json:"status"`
	RequestID string `json:"request_id,omitempty"`
	Replayed  bool   `json:"replayed"`
}

func claimView(c qr.ClaimResult) claimJSON {
	st := "pending"
	switch c.State {
	case qr.TxCompleted:
		st = "completed"
	case qr.TxClosed:
		st = "closed"
	}
	return claimJSON{c.TransactionID, st, c.RequestID, c.Replayed}
}

// ---- API handlers ----------------------------------------------------

func (h *handler) apiMyQR(w http.ResponseWriter, r *http.Request, s Session) error {
	a, err := h.svc.MyAlias(r.Context(), s.Principal)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"alias": aliasView(a), "png_path": "/api/v1/me/qr.png"})
}

func (h *handler) apiMyQRPNG(w http.ResponseWriter, r *http.Request, s Session) error {
	b, _, err := h.svc.RenderAlias(r.Context(), s.Principal)
	if err != nil {
		return err
	}
	return h.sendPNG(w, b, "circula-member-qr.png")
}

func (h *handler) sendPNG(w http.ResponseWriter, b []byte, name string) error {
	secureHeaders(w)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	_, err := w.Write(b)
	return err
}

func idemKey(r *http.Request) (string, error) {
	k := r.Header.Get("Idempotency-Key")
	if k == "" {
		return "", fmt.Errorf("%w: Idempotency-Key header is required", qr.ErrInvalid)
	}
	return k, nil
}

func (h *handler) apiRotate(w http.ResponseWriter, r *http.Request, s Session) error {
	k, err := idemKey(r)
	if err != nil {
		return err
	}
	a, err := h.svc.RotateAlias(r.Context(), s.Principal, k)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"alias": aliasView(a), "png_path": "/api/v1/me/qr.png"})
}

type createBody struct {
	AmountMinor      int64  `json:"amount_minor"`
	Note             string `json:"note"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

func (h *handler) apiCreateRequest(w http.ResponseWriter, r *http.Request, s Session) error {
	k, err := idemKey(r)
	if err != nil {
		return err
	}
	var b createBody
	if err := decodeJSON(r, &b); err != nil {
		return err
	}
	if b.ExpiresInSeconds < 0 || b.ExpiresInSeconds > int64(30*24*time.Hour/time.Second) {
		return fmt.Errorf("%w: expires_in_seconds out of range", qr.ErrInvalid)
	}
	c, err := h.svc.CreateRequest(r.Context(), s.Principal, k, qr.CreateRequestInput{AmountMinor: b.AmountMinor, Note: b.Note, TTL: time.Duration(b.ExpiresInSeconds) * time.Second})
	if err != nil {
		return err
	}
	out := map[string]any{"request": requestView(c.View)}
	status := http.StatusOK
	if !c.Replayed {
		status = http.StatusCreated
		out["token"], out["url"], out["compact_code"] = c.Token, c.URL, c.Compact
		if png, err := h.svc.RenderURL(c.URL); err == nil {
			out["png_base64"] = base64.StdEncoding.EncodeToString(png)
		}
	} else {
		out["token_available"] = false // raw tokens are never stored: revoke and create a new request if lost
	}
	return writeJSON(w, status, out)
}

func (h *handler) apiListRequests(w http.ResponseWriter, r *http.Request, s Session) error {
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return fmt.Errorf("%w: limit must be 1-100", qr.ErrInvalid)
		}
		limit = n
	}
	items, next, err := h.svc.ListRequests(r.Context(), s.Principal, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		return err
	}
	out := make([]requestJSON, 0, len(items))
	for _, it := range items {
		out = append(out, requestView(it))
	}
	return writeJSON(w, http.StatusOK, map[string]any{"items": out, "next_cursor": next})
}

func (h *handler) apiGetRequest(w http.ResponseWriter, r *http.Request, s Session) error {
	ref := r.PathValue("ref")
	if strings.HasPrefix(ref, "qrr_") {
		v, err := h.svc.GetRequest(r.Context(), s.Principal, ref)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, requestView(v))
	}
	p, err := h.svc.PreviewRequest(r.Context(), s.Principal, ref)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, previewView(p))
}

func (h *handler) apiRevoke(w http.ResponseWriter, r *http.Request, s Session) error {
	v, err := h.svc.RevokeRequest(r.Context(), s.Principal, r.PathValue("id"))
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, requestView(v))
}

func (h *handler) apiResolveAlias(w http.ResponseWriter, r *http.Request, s Session) error {
	p, err := h.svc.ResolveAlias(r.Context(), s.Principal, r.PathValue("code"))
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, previewView(p))
}

type codeBody struct {
	Code        string `json:"code"`
	AmountMinor int64  `json:"amount_minor"`
	Note        string `json:"note"`
}

// preview resolves any scanned or pasted code without putting it in a URL.
func (h *handler) preview(r *http.Request, s Session, code string) (qr.Preview, qr.Parsed, error) {
	parsed, err := qr.ParseManual(h.svc.Origin(), code)
	if err != nil {
		return qr.Preview{}, parsed, fmt.Errorf("%w: not a Circula QR code for this site", qr.ErrInvalid)
	}
	var p qr.Preview
	if parsed.Kind == qr.KindAlias {
		p, err = h.svc.ResolveAlias(r.Context(), s.Principal, parsed.Code)
	} else {
		p, err = h.svc.PreviewRequest(r.Context(), s.Principal, parsed.Code)
	}
	return p, parsed, err
}

func (h *handler) apiResolve(w http.ResponseWriter, r *http.Request, s Session) error {
	var b codeBody
	if err := decodeJSON(r, &b); err != nil {
		return err
	}
	p, _, err := h.preview(r, s, b.Code)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, previewView(p))
}

func (h *handler) apiClaim(w http.ResponseWriter, r *http.Request, s Session) error {
	k, err := idemKey(r)
	if err != nil {
		return err
	}
	var b codeBody
	if err := decodeJSON(r, &b); err != nil {
		return err
	}
	res, err := h.claim(r, s, b.Code, b.AmountMinor, b.Note, k)
	if err != nil {
		return err
	}
	st := http.StatusCreated
	if res.Replayed {
		st = http.StatusOK
	}
	return writeJSON(w, st, claimView(res))
}

func (h *handler) claim(r *http.Request, s Session, code string, amount int64, note, key string) (qr.ClaimResult, error) {
	parsed, err := qr.ParseManual(h.svc.Origin(), code)
	if err != nil {
		return qr.ClaimResult{}, fmt.Errorf("%w: not a Circula QR code for this site", qr.ErrInvalid)
	}
	if parsed.Kind == qr.KindAlias {
		return h.svc.ProposeToAlias(r.Context(), s.Principal, parsed.Code, amount, note, key)
	}
	if amount != 0 || note != "" {
		// the request fixes amount and note; client values are never used
		return qr.ClaimResult{}, fmt.Errorf("%w: a one-time request fixes its own amount and note", qr.ErrInvalid)
	}
	return h.svc.ClaimRequest(r.Context(), s.Principal, parsed.Code, key)
}

// ---- static assets -----------------------------------------------------

func (h *handler) static(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/qr/")
	if name == "" || path.Clean(name) != name || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(staticFS, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch path.Ext(name) {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	default:
		http.NotFound(w, r) // LICENSE files and anything else are not served as assets
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=3600") // static, secret-free assets
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	_, _ = w.Write(b)
}
