// Package web provides no-JavaScript, server-rendered HTML pages and a JSON
// API over favorites.Service. It is an explicit constructor: nothing here
// registers routes, reads configuration or authenticates on its own. The host
// supplies an Authenticator that resolves the session to an Actor and CSRF
// token; the actor is never read from request bodies, query strings or
// headers other than the host's own session mechanism.
package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/djshellshoxxx/circula/future/favorites"
)

// Session is the authenticated caller as resolved by the host.
type Session struct {
	Actor favorites.Actor
	// CSRF is the per-session token expected in the csrf_token form field or
	// the X-CSRF-Token header on every state-changing request.
	CSRF string
}

// Authenticator resolves the request's session. Any error means unauthenticated.
type Authenticator interface {
	Authenticate(r *http.Request) (Session, error)
}

// AuthFunc adapts a function to Authenticator.
type AuthFunc func(r *http.Request) (Session, error)

func (f AuthFunc) Authenticate(r *http.Request) (Session, error) { return f(r) }

// ProposalFields names the host's proposal form fields and destination, so the
// draft page can post a prefilled, still-unpriced form to the host's normal
// confirmation flow. Nothing is posted by this module.
type ProposalFields struct {
	Action         string // required: URL of the host's proposal form handler
	Recipient      string // default "recipient"
	Amount         string // default "amount"
	Description    string // default "description"
	IdempotencyKey string // default "idempotency_key"
	CSRF           string // default "csrf_token"
}

// Config configures the handler.
type Config struct {
	Service *favorites.Service
	Auth    Authenticator
	// BasePath is the HTML mount path (default "/favorites").
	BasePath string
	// APIPath is the JSON mount path (default "/api/v1/me/favorites").
	APIPath string
	// DirectoryURL is the always-present fallback to directory search.
	DirectoryURL string
	Proposal     ProposalFields
}

// Handler serves both the HTML and JSON surfaces.
type Handler struct {
	svc          *favorites.Service
	auth         Authenticator
	base, api    string
	directoryURL string
	prop         ProposalFields
	mux          *http.ServeMux
	cssHash      string
}

// New validates cfg and builds the handler.
func New(cfg Config) (*Handler, error) {
	if cfg.Service == nil || cfg.Auth == nil {
		return nil, errors.New("web: Service and Auth are required")
	}
	h := &Handler{svc: cfg.Service, auth: cfg.Auth, base: cfg.BasePath, api: cfg.APIPath, directoryURL: cfg.DirectoryURL, prop: cfg.Proposal}
	if h.base == "" {
		h.base = "/favorites"
	}
	if h.api == "" {
		h.api = "/api/v1/me/favorites"
	}
	for _, p := range []string{h.base, h.api} {
		if !strings.HasPrefix(p, "/") || strings.HasSuffix(p, "/") || strings.ContainsAny(p, "{}?#") {
			return nil, fmt.Errorf("web: bad mount path %q", p)
		}
	}
	if h.directoryURL == "" {
		h.directoryURL = "/directory"
	}
	if h.prop.Action == "" {
		h.prop.Action = "/trades/new"
	}
	def := func(p *string, v string) {
		if *p == "" {
			*p = v
		}
	}
	def(&h.prop.Recipient, "recipient")
	def(&h.prop.Amount, "amount")
	def(&h.prop.Description, "description")
	def(&h.prop.IdempotencyKey, "idempotency_key")
	def(&h.prop.CSRF, "csrf_token")
	for _, u := range []string{h.directoryURL, h.prop.Action} {
		if !safeLocalURL(u) {
			return nil, fmt.Errorf("web: %q must be a local absolute path", u)
		}
	}
	sum := sha256.Sum256([]byte(css))
	h.cssHash = "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
	h.routes()
	return h, nil
}

func safeLocalURL(u string) bool {
	return strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//") && !strings.ContainsAny(u, "\\\r\n")
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *Handler) routes() {
	m := http.NewServeMux()
	b, a := h.base, h.api
	m.HandleFunc("GET "+b, h.page(false, h.htmlList))
	m.HandleFunc("POST "+b+"/add", h.page(true, h.htmlAdd))
	m.HandleFunc("POST "+b+"/repeat", h.page(true, h.htmlRepeat))
	m.HandleFunc("GET "+b+"/drafts/{key}", h.page(false, h.htmlDraft))
	m.HandleFunc("POST "+b+"/{id}/nickname", h.page(true, h.htmlNickname))
	m.HandleFunc("POST "+b+"/{id}/pin", h.page(true, h.htmlPin))
	m.HandleFunc("POST "+b+"/{id}/move", h.page(true, h.htmlMove))
	m.HandleFunc("POST "+b+"/{id}/remove", h.page(true, h.htmlRemove))
	m.HandleFunc("POST "+b+"/{id}/draft", h.page(true, h.htmlStartDraft))

	m.HandleFunc("GET "+a, h.jsonAPI(false, h.apiList))
	m.HandleFunc("POST "+a, h.jsonAPI(true, h.apiAdd))
	m.HandleFunc("PUT "+a+"/order", h.jsonAPI(true, h.apiReorder))
	m.HandleFunc("POST "+a+"/drafts", h.jsonAPI(true, h.apiCreateDraft))
	m.HandleFunc("GET "+a+"/drafts/{key}", h.jsonAPI(false, h.apiGetDraft))
	m.HandleFunc("PATCH "+a+"/{id}", h.jsonAPI(true, h.apiPatch))
	m.HandleFunc("DELETE "+a+"/{id}", h.jsonAPI(true, h.apiDelete))
	h.mux = m
}

// secureHeaders sets headers for every response: favorites are owner-private,
// so nothing may be cached, stored in the back/forward cache or sniffed.
func (h *Handler) secureHeaders(w http.ResponseWriter, html bool) {
	hd := w.Header()
	hd.Set("Cache-Control", "no-store, max-age=0")
	hd.Set("Pragma", "no-cache")
	hd.Set("Vary", "Cookie, X-Network")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Referrer-Policy", "no-referrer")
	hd.Set("X-Frame-Options", "DENY")
	if html {
		hd.Set("Content-Security-Policy", "default-src 'none'; style-src '"+h.cssHash+"'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	} else {
		hd.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	}
}

// sameOrigin rejects cross-site browser requests. Requests with neither
// Origin nor Sec-Fetch-Site (non-browser clients) rely on the CSRF token.
func sameOrigin(r *http.Request) bool {
	if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" && sfs != "same-origin" && sfs != "none" {
		return false
	}
	if o := r.Header.Get("Origin"); o != "" {
		u, err := url.Parse(o)
		if err != nil || u.Host == "" || !strings.EqualFold(u.Host, r.Host) {
			return false
		}
	}
	return true
}

func csrfOK(want, got string) bool {
	return want != "" && len(got) == len(want) && subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func randomKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("web: no entropy: " + err.Error())
	}
	return hex.EncodeToString(b)
}
