// Package api serves the versioned JSON API (/api/v1) defined in
// docs/specs/07-api-contract.md. It is a thin adapter: every operation calls
// *service.Service (ARC-002/003) and never touches the store. Business rules
// live in the service; this package only authenticates, parses, maps errors and
// shapes representations.
package api

import (
	"crypto/rand"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

// Options tunes the handler. The zero value is production-ready.
type Options struct {
	Logger *slog.Logger // default: slog.Default()
	Now    func() time.Time
	// MaxBodyBytes caps request bodies (API-037). Default 64 KiB.
	MaxBodyBytes int64
	// PrincipalPerMinute and IPPerMinute are fixed-window rate limits
	// (API-038). 0 selects the default (600 / 1200); negative disables.
	PrincipalPerMinute int
	IPPerMinute        int
	// CursorKey signs pagination cursors (API-035). Default: random per
	// process, so cursors do not survive a restart.
	CursorKey []byte
	// CSRFKey keys the CSRF tokens. It must equal the key the HTML adapter
	// uses (web.Options.CSRFKey) so that one browser session works for both;
	// default: random per process.
	CSRFKey []byte
	// Edge carries the trusted-proxy, Secure and HSTS policy shared with the
	// HTML adapter. The zero value ignores forwarding headers.
	Edge httpx.Edge
}

type server struct {
	svc    *service.Service
	opt    Options
	log    *slog.Logger
	mux    *http.ServeMux
	idem   *idemStore
	rl     *limiter
	jobs   *jobStore
	casMu  sync.Mutex // serialises check-then-write for If-Match updates
	cursor []byte
	csrf   httpx.CSRF
	pwFail *limiter // current-password guesses per person
}

// NewHandler returns the API handler. Mount it at /api/.
func NewHandler(svc *service.Service) http.Handler { return NewHandlerWithOptions(svc, Options{}) }

// NewHandlerWithOptions is NewHandler with explicit options.
func NewHandlerWithOptions(svc *service.Service, opt Options) http.Handler {
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.MaxBodyBytes <= 0 {
		opt.MaxBodyBytes = 64 << 10
	}
	if opt.PrincipalPerMinute == 0 {
		opt.PrincipalPerMinute = 600
	}
	if opt.IPPerMinute == 0 {
		opt.IPPerMinute = 1200
	}
	if len(opt.CursorKey) == 0 {
		opt.CursorKey = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, opt.CursorKey); err != nil {
			panic("api: no entropy: " + err.Error())
		}
	}
	if len(opt.CSRFKey) == 0 {
		opt.CSRFKey = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, opt.CSRFKey); err != nil {
			panic("api: no entropy: " + err.Error())
		}
	}
	s := &server{svc: svc, opt: opt, log: opt.Logger, mux: http.NewServeMux(),
		idem: newIdemStore(24*time.Hour, 10000), rl: newLimiter(opt.Now), jobs: newJobStore(opt.Now),
		cursor: opt.CursorKey, csrf: httpx.CSRF{Key: opt.CSRFKey}, pwFail: newLimiter(opt.Now)}
	s.routes()
	s.mux.HandleFunc("/", s.notFound)
	setErasureHook(s)
	return s
}

// ServeHTTP applies the edge middleware, then dispatches.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Tolerate a mount that strips "/api".
	if strings.HasPrefix(r.URL.Path, "/v1/") {
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/api" + r.URL.Path
		r2.URL.RawPath = ""
		r = r2
	}
	s.edge(s.mux).ServeHTTP(w, r)
}

// notFound answers unmatched routes with problem+json (404, or 405 + Allow).
func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	var allow []string
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		if m == r.Method {
			continue
		}
		r2 := r.Clone(r.Context())
		r2.Method = m
		if _, pat := s.mux.Handler(r2); pat != "" && pat != "/" {
			allow = append(allow, m)
		}
	}
	rid := w.Header().Get("X-Request-Id")
	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		writeProblem(w, r, rid, &apiError{status: 405, code: "method-not-allowed", detail: "method not allowed for this path"})
		return
	}
	writeProblem(w, r, rid, &apiError{status: 404, code: "not-found", detail: "no such resource"})
}
