// Package web serves the server-rendered HTML UI (docs/specs/08-ux-accessibility.md).
// It translates HTTP forms into service calls and holds no business rules.
package web

import (
	"crypto/rand"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

//go:embed templates/*.html static/*
var assets embed.FS

// Options tunes the handler. The zero value is safe for production.
type Options struct {
	// SetupSecret, when non-empty, must be entered to create a network
	// (FR-002 bootstrap). Empty: any signed-in person may create a network.
	SetupSecret string
	// ForceSecure marks cookies Secure even when the request looks like plain
	// HTTP (use behind a TLS-terminating proxy that sets no forwarded header).
	ForceSecure bool
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
	login *limiter
	mux   *http.ServeMux
}

// NewHandler returns the UI handler. Mount it at /.
func NewHandler(svc *service.Service) http.Handler { return NewHandlerWith(svc, Options{}) }

// NewHandlerWith is NewHandler with options.
func NewHandlerWith(svc *service.Service, opts Options) http.Handler {
	s := &server{svc: svc, opts: opts, key: opts.CSRFKey, login: newLimiter(10, 60)}
	if len(s.key) == 0 {
		s.key = make([]byte, 32)
		if _, err := rand.Read(s.key); err != nil {
			panic("web: no randomness: " + err.Error())
		}
	}
	if s.opts.Clock == nil {
		s.opts.Clock = wallClock{}
	}
	s.pages = mustLoadTemplates()
	s.mux = http.NewServeMux()
	static, _ := fs.Sub(assets, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	s.routes()
	return s.wrap(s.mux)
}

func (s *server) routes() {
	s.authRoutes()
	s.homeRoutes()
	s.tradeRoutes()
	s.disputeRoutes()
	s.adminRoutes()
	s.mux.HandleFunc("/", s.notFoundOrRoot)
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }
