// Package web is an explicitly constructed HTTP adapter for the discovery
// service: a JSON API with RFC 9457 errors and server-rendered, no-JavaScript
// HTML. Nothing here registers itself or is mounted anywhere; a host must call
// New and choose to route to the returned handler.
//
// The host supplies an Authenticator. Every request is re-authenticated, so
// membership, suspension and role changes take effect immediately. Unsafe
// methods require the actor's per-session CSRF token and a same-origin Origin
// (when present).
package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

//go:embed templates/*.html static/*.css
var assets embed.FS

// Authenticator maps a request to the trusted, current actor. Returning an
// error yields 401. It must never derive identity from client-supplied JSON.
type Authenticator interface {
	Authenticate(r *http.Request) (d.Actor, error)
}

// AuthFunc adapts a function to Authenticator.
type AuthFunc func(r *http.Request) (d.Actor, error)

// Authenticate implements Authenticator.
func (f AuthFunc) Authenticate(r *http.Request) (d.Actor, error) { return f(r) }

// Config configures the adapter.
type Config struct {
	Service *d.Service
	Auth    Authenticator
	// BasePath is the URL prefix the host mounts the handler under (used for
	// generated links only; the host strips it). Empty or like "/market".
	BasePath string
	// MaxMediaBody bounds the reviewed media-upload exception to the 64 KiB
	// API body policy. Default 6 MiB.
	MaxMediaBody int64
	// Logf receives internal errors (never client data). Default log.Printf.
	Logf func(format string, args ...any)
}

// Server is the HTTP adapter.
type Server struct {
	cfg  Config
	svc  *d.Service
	tpl  *template.Template
	mux  *http.ServeMux
	base string
}

const apiBodyLimit = 64 << 10

// New builds the handler. It returns an error for incomplete configuration.
func New(cfg Config) (http.Handler, error) {
	if cfg.Service == nil || cfg.Auth == nil {
		return nil, errors.New("web: Service and Auth are required")
	}
	if cfg.BasePath != "" && (!strings.HasPrefix(cfg.BasePath, "/") || strings.HasSuffix(cfg.BasePath, "/")) {
		return nil, errors.New("web: BasePath must start with / and not end with /")
	}
	if cfg.MaxMediaBody <= 0 {
		cfg.MaxMediaBody = 6 << 20
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	s := &Server{cfg: cfg, svc: cfg.Service, mux: http.NewServeMux(), base: cfg.BasePath}
	tpl, err := template.New("").Funcs(s.funcs()).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	s.tpl = tpl
	s.routes()
	return s.mux, nil
}

func (s *Server) routes() {
	m := s.mux
	// JSON API (proposed contract; see openapi-fragment.yaml)
	m.HandleFunc("GET /api/v1/listings", s.api(s.apiSearch))
	m.HandleFunc("GET /api/v1/listings/{id}", s.api(s.apiGetListing))
	m.HandleFunc("PUT /api/v1/listings/{id}/discovery", s.api(s.apiSaveDiscovery))
	m.HandleFunc("POST /api/v1/listings/{id}/renew", s.api(s.apiRenew))
	m.HandleFunc("POST /api/v1/listings/{id}/media", s.api(s.apiAddMedia))
	m.HandleFunc("GET /api/v1/listings/{id}/media/{mediaId}", s.api(s.apiGetMedia))
	m.HandleFunc("DELETE /api/v1/listings/{id}/media/{mediaId}", s.api(s.apiDeleteMedia))
	m.HandleFunc("GET /api/v1/categories", s.api(s.apiListCategories))
	m.HandleFunc("POST /api/v1/categories", s.api(s.apiCreateCategory))
	m.HandleFunc("PATCH /api/v1/categories/{cid}", s.api(s.apiPatchCategory))
	m.HandleFunc("GET /api/v1/areas", s.api(s.apiListAreas))
	m.HandleFunc("POST /api/v1/areas", s.api(s.apiCreateArea))
	m.HandleFunc("PATCH /api/v1/areas/{aid}", s.api(s.apiPatchArea))
	m.HandleFunc("GET /api/v1/tags", s.api(s.apiListTags))
	m.HandleFunc("POST /api/v1/tags/{tag}/moderation", s.api(s.apiModerateTag))
	// Server-rendered, no-JS pages
	m.HandleFunc("GET /listings", s.page(s.pageSearch))
	m.HandleFunc("GET /listings/{id}", s.page(s.pageDetail))
	m.HandleFunc("GET /listings/{id}/edit", s.page(s.pageEdit))
	m.HandleFunc("POST /listings/{id}/edit", s.page(s.postEdit))
	m.HandleFunc("POST /listings/{id}/renew", s.page(s.postRenew))
	m.HandleFunc("POST /listings/{id}/photos", s.page(s.postPhoto))
	m.HandleFunc("POST /listings/{id}/photos/{mediaId}/remove", s.page(s.postRemovePhoto))
	m.HandleFunc("GET /admin/taxonomy", s.page(s.pageAdmin))
	m.HandleFunc("POST /admin/categories", s.page(s.postAdminCategory))
	m.HandleFunc("POST /admin/categories/{cid}", s.page(s.postAdminCategoryUpdate))
	m.HandleFunc("POST /admin/areas", s.page(s.postAdminArea))
	m.HandleFunc("POST /admin/areas/{aid}", s.page(s.postAdminAreaUpdate))
	m.HandleFunc("POST /admin/tags/{tag}", s.page(s.postAdminTag))
	m.HandleFunc("GET /static/discovery.css", s.static)
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	b, err := assets.ReadFile("static/discovery.css")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(b)
}

type handlerFn func(w http.ResponseWriter, r *http.Request, a d.Actor) error

func secure(w http.ResponseWriter, html bool) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	if html {
		h.Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	} else {
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; sandbox")
	}
}

func unsafeMethod(m string) bool {
	return m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions
}

// gate authenticates, scopes and CSRF-checks a request.
func (s *Server) gate(w http.ResponseWriter, r *http.Request, html bool) (d.Actor, error) {
	secure(w, html)
	a, err := s.cfg.Auth.Authenticate(r)
	if err != nil || a.ID == "" || a.Network == "" {
		return a, errUnauthenticated
	}
	if !a.Active {
		return a, d.ErrForbidden
	}
	if n := r.Header.Get("X-Network"); n != "" && n != a.Network {
		return a, d.ErrForbidden
	}
	if unsafeMethod(r.Method) {
		if o := r.Header.Get("Origin"); o != "" {
			u, perr := url.Parse(o)
			if perr != nil || u.Host != r.Host {
				return a, d.ErrForbidden
			}
		}
		tok := r.Header.Get("X-CSRF-Token")
		if tok == "" && html {
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxMediaBody)
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					return a, badRequest("malformed or oversized form")
				}
			} else {
				r.Body = http.MaxBytesReader(w, r.Body, apiBodyLimit)
				if err := r.ParseForm(); err != nil {
					return a, badRequest("malformed or oversized form")
				}
			}
			tok = r.PostFormValue("csrf")
		}
		if a.CSRF == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(a.CSRF)) != 1 {
			return a, errCSRF
		}
	}
	return a, nil
}

var (
	errUnauthenticated = errors.New("unauthenticated")
	errCSRF            = errors.New("csrf")
)

type httpError struct {
	status int
	detail string
}

func (e *httpError) Error() string { return e.detail }

func badRequest(detail string) error { return &httpError{http.StatusBadRequest, detail} }

// --- RFC 9457 problems ---

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Field  string `json:"field,omitempty"`
}

func (s *Server) classify(err error) (status int, p problem) {
	var de *d.Error
	var he *httpError
	var mbe *http.MaxBytesError
	field, detail := "", ""
	if errors.As(err, &de) {
		field, detail = de.Field, de.Detail
	}
	switch {
	case errors.Is(err, errUnauthenticated):
		status, p = 401, problem{Type: "urn:circula:problem:unauthenticated", Title: "Authentication required"}
	case errors.Is(err, errCSRF):
		status, p = 403, problem{Type: "urn:circula:problem:csrf", Title: "Missing or invalid CSRF token"}
	case errors.As(err, &he):
		status, p = he.status, problem{Type: "urn:circula:problem:bad-request", Title: "Bad request", Detail: he.detail}
	case errors.As(err, &mbe):
		status, p = 413, problem{Type: "urn:circula:problem:too-large", Title: "Request body too large"}
	case errors.Is(err, d.ErrForbidden):
		status, p = 403, problem{Type: "urn:circula:problem:forbidden", Title: "Forbidden"}
	case errors.Is(err, d.ErrNotFound):
		status, p = 404, problem{Type: "urn:circula:problem:not-found", Title: "Not found"}
	case errors.Is(err, d.ErrBadCursor):
		status, p = 400, problem{Type: "urn:circula:problem:bad-cursor", Title: "Invalid or expired cursor", Detail: "Start again from the first page."}
	case errors.Is(err, d.ErrConflict):
		status, p = 409, problem{Type: "urn:circula:problem:conflict", Title: "Conflict", Detail: detail, Field: field}
	case errors.Is(err, d.ErrDuplicate):
		status, p = 409, problem{Type: "urn:circula:problem:duplicate", Title: "Already exists", Detail: detail, Field: field}
	case errors.Is(err, d.ErrQuota):
		if field == "file" && strings.Contains(detail, "too large") {
			status = 413
		} else {
			status = 409
		}
		p = problem{Type: "urn:circula:problem:quota", Title: "Limit reached", Detail: detail, Field: field}
	case errors.Is(err, d.ErrRateLimited):
		status, p = 429, problem{Type: "urn:circula:problem:rate-limited", Title: "Too many requests"}
	case errors.Is(err, d.ErrRejected):
		status, p = 422, problem{Type: "urn:circula:problem:rejected", Title: "Rejected by policy", Detail: detail, Field: field}
	case errors.Is(err, d.ErrInvalid):
		status, p = 422, problem{Type: "urn:circula:problem:validation", Title: "Validation failed", Detail: detail, Field: field}
	default:
		s.cfg.Logf("discovery/web: internal error: %v", err)
		status, p = 500, problem{Type: "urn:circula:problem:internal", Title: "Internal error"}
	}
	p.Status = status
	return status, p
}

func (s *Server) writeProblem(w http.ResponseWriter, err error) {
	status, p := s.classify(err)
	if status == 429 {
		w.Header().Set("Retry-After", "60")
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(p)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// api wraps JSON endpoints: auth, CSRF, idempotency, error mapping.
func (s *Server) api(fn handlerFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := s.gate(w, r, false)
		if err != nil {
			s.writeProblem(w, err)
			return
		}
		if key := r.Header.Get("Idempotency-Key"); key != "" && unsafeMethod(r.Method) {
			s.idempotent(w, r, a, key, fn)
			return
		}
		if err := fn(w, r, a); err != nil {
			s.writeProblem(w, err)
		}
	}
}

type recorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
	over   bool
}

func (rc *recorder) WriteHeader(c int) { rc.status = c; rc.ResponseWriter.WriteHeader(c) }
func (rc *recorder) Write(b []byte) (int, error) {
	if rc.status == 0 {
		rc.status = 200
	}
	if rc.buf.Len()+len(b) > apiBodyLimit {
		rc.over = true
	} else {
		rc.buf.Write(b)
	}
	return rc.ResponseWriter.Write(b)
}

func (s *Server) idempotent(w http.ResponseWriter, r *http.Request, a d.Actor, key string, fn handlerFn) {
	if !validIdemKey(key) {
		s.writeProblem(w, badRequest("Idempotency-Key must be 1-128 characters of [A-Za-z0-9._:-]"))
		return
	}
	limit := int64(apiBodyLimit)
	if strings.HasSuffix(r.URL.Path, "/media") {
		limit = s.cfg.MaxMediaBody
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		s.writeProblem(w, err)
		return
	}
	sum := sha256.New()
	fmt.Fprintf(sum, "%s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)
	sum.Write([]byte(r.Header.Get("Content-Type") + "\n"))
	sum.Write(body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	res, err := s.svc.Repo().IdemBegin(r.Context(), a.Network, a.ID, key, hex.EncodeToString(sum.Sum(nil)), s.svc.Config().Now())
	if err != nil {
		s.writeProblem(w, err)
		return
	}
	switch res.State {
	case "replay":
		w.Header().Set("Idempotency-Replayed", "true")
		w.Header().Set("Content-Type", res.ContentType)
		w.WriteHeader(res.Status)
		w.Write(res.Body)
		return
	case "mismatch":
		s.writeProblem(w, &d.Error{Kind: d.ErrInvalid, Field: "Idempotency-Key", Detail: "key was already used with a different request"})
		return
	case "inflight":
		s.writeProblem(w, &d.Error{Kind: d.ErrConflict, Field: "Idempotency-Key", Detail: "a request with this key is still being processed"})
		return
	}
	rec := &recorder{ResponseWriter: w}
	herr := fn(rec, r, a)
	if herr != nil {
		status, _ := s.classify(herr)
		if status >= 500 || status == 429 {
			_ = s.svc.Repo().IdemAbort(context.WithoutCancel(r.Context()), a.Network, a.ID, key)
		}
		// Record deterministic client errors so replays are identical.
		rec2 := &recorder{ResponseWriter: w}
		s.writeProblem(rec2, herr)
		if status < 500 && status != 429 && !rec2.over {
			_ = s.svc.Repo().IdemFinish(context.WithoutCancel(r.Context()), a.Network, a.ID, key, rec2.status, "application/problem+json", rec2.buf.Bytes())
		}
		return
	}
	if rec.over || rec.status >= 500 {
		_ = s.svc.Repo().IdemAbort(context.WithoutCancel(r.Context()), a.Network, a.ID, key)
		return
	}
	ct := w.Header().Get("Content-Type")
	_ = s.svc.Repo().IdemFinish(context.WithoutCancel(r.Context()), a.Network, a.ID, key, rec.status, ct, rec.buf.Bytes())
}

func validIdemKey(k string) bool {
	if len(k) == 0 || len(k) > 128 {
		return false
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}
