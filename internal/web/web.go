// Package web serves the server-rendered HTML UI (docs/specs/08-ux-accessibility.md).
// It translates HTTP forms into service calls and holds no business rules.
package web

import (
	"crypto/rand"
	"embed"
	"io/fs"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

//go:embed templates/*.html static/*
var assets embed.FS

// Options tunes the handler. The zero value is safe for production.
type Options struct {
	// SetupSecret, when non-empty, must be entered to create a network
	// (FR-002 bootstrap) and stops working after the first network it
	// created in this process. Empty: any signed-in person may create a network.
	SetupSecret string
	// Edge carries the Secure-cookie, HSTS and trusted-proxy policy shared with
	// the JSON API. The zero value trusts no forwarding header.
	Edge httpx.Edge
	// PublicBaseURL, when set, is the origin used in invitation links and the
	// only Origin accepted for form posts (CSRF defence behind proxies that
	// rewrite Host).
	PublicBaseURL string
	// SessionIdle and SessionMax mirror the service's session lifetimes; they
	// only drive the expiry warning (UX-030). Zero disables the warning.
	SessionIdle, SessionMax time.Duration
	// CSRFKey is the HMAC key for form tokens; random per process when nil.
	CSRFKey []byte
	// Clock is used only to label invitations as expired; default wall clock.
	Clock domain.Clock
}

type server struct {
	svc   *service.Service
	opts  Options
	key   []byte
	pages map[string]*pageTemplate
	csrf  httpx.CSRF
	// Failure counters: per client address, per client+login, and per signed-in
	// person for password-change guesses. reg and setup count every attempt.
	ipFail, acctFail, pwFail, reg, setup *limiter
	setupUsed                            atomic.Bool
	mux                                  *http.ServeMux
}

// NewHandler returns the UI handler. Mount it at /.
func NewHandler(svc *service.Service) http.Handler { return NewHandlerWith(svc, Options{}) }

// NewHandlerWith is NewHandler with options.
func NewHandlerWith(svc *service.Service, opts Options) http.Handler {
	s := newServer(svc, opts)
	return s.wrap(s.mux)
}

func newServer(svc *service.Service, opts Options) *server {
	s := &server{svc: svc, opts: opts, key: opts.CSRFKey,
		ipFail: newLimiter(20, 60), acctFail: newLimiter(5, 60), pwFail: newLimiter(5, 600),
		reg: newLimiter(10, 60), setup: newLimiter(10, 600)}
	if len(s.key) == 0 {
		s.key = make([]byte, 32)
		if _, err := rand.Read(s.key); err != nil {
			panic("web: no randomness: " + err.Error())
		}
	}
	s.csrf = httpx.CSRF{Key: s.key}
	if s.opts.Clock == nil {
		s.opts.Clock = wallClock{}
	}
	s.pages = mustLoadTemplates()
	s.mux = http.NewServeMux()
	static, _ := fs.Sub(assets, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	s.routes()
	return s
}

func (s *server) routes() {
	s.authRoutes()
	s.accountRoutes()
	s.homeRoutes()
	s.tradeRoutes()
	s.disputeRoutes()
	s.adminRoutes()
	s.mux.HandleFunc("/", s.notFoundOrRoot)
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }
