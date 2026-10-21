package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

// opsServer wires the HTTP edge: request policy (ID, body limit, deadline),
// observability, the operational endpoints and the mounted API and UI.
type opsServer struct {
	cfg     *Config
	db      *sql.DB
	logger  *slog.Logger
	metrics *metrics
	ready   atomic.Bool // set once serving starts, cleared on shutdown
}

// routeOf maps a request path to a route template label. Only fixed templates
// are returned, so metric labels cannot carry identifiers from the path.
func routeOf(p string) string {
	switch {
	case strings.HasPrefix(p, "/api/"):
		return "/api/"
	case p == "/healthz", p == "/readyz", p == "/metrics":
		return p
	default:
		return "/"
	}
}

// routes assembles the handler tree. /api/ and / receive the API and UI with
// the request deadline; the operational endpoints do not.
func (o *opsServer) routes(api, web http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", o.bounded(api))
	mux.Handle("/", o.bounded(web))
	mux.HandleFunc("/healthz", o.healthz)
	mux.HandleFunc("/readyz", o.readyz)
	mux.HandleFunc("/metrics", o.metricsAccess(o.serveMetrics))
	return o.observe(mux)
}

// observe is the outermost middleware: panic recovery, request ID, body size
// limit, metrics and a structured access log. The log records the route
// template, never the raw path, query, headers or body.
func (o *opsServer) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := r.Header.Get("X-Request-Id")
		if !validRequestID(reqID) {
			reqID = newRequestID()
			r.Header.Set("X-Request-Id", reqID)
		}
		w.Header().Set("X-Request-Id", reqID)
		route := routeOf(r.URL.Path)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		r.Body = http.MaxBytesReader(rec, r.Body, o.cfg.MaxBodyBytes)
		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler {
					panic(p)
				}
				o.logger.Error("panic recovered", "request_id", reqID, "route", route, "stack", string(debug.Stack()))
				if !rec.wrote {
					http.Error(rec, "internal error", http.StatusInternalServerError)
				}
			}
			d := time.Since(start)
			o.metrics.observe(route, rec.status, d)
			level := slog.LevelInfo
			if route == "/healthz" || route == "/readyz" || route == "/metrics" {
				level = slog.LevelDebug
			}
			o.logger.Log(r.Context(), level, "request", "request_id", reqID, "method", r.Method,
				"route", route, "status", rec.status, "duration_ms", d.Milliseconds())
		}()
		next.ServeHTTP(rec, r)
	})
}

// bounded applies the request deadline to the context and the response write
// deadline. Streaming responses longer than request-timeout are cut off.
func (o *opsServer) bounded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), o.cfg.RequestTimeout)
		defer cancel()
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(o.cfg.RequestTimeout))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// healthz is the coarse liveness and database check (OPS-013). It reports the
// database reachability and whether the schema matches this binary. It never
// includes member, network, balance or configuration data.
func (o *opsServer) healthz(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r) {
		return
	}
	status, code := "ok", http.StatusOK
	if err := o.probeDB(r.Context()); err != nil {
		status, code = "unavailable", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{"status": status, "version": version})
}

// readyz reports whether the instance accepts traffic: started, not shutting
// down, database reachable and schema current.
func (o *opsServer) readyz(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r) {
		return
	}
	status, code := "ready", http.StatusOK
	if !o.ready.Load() || o.probeDB(r.Context()) != nil {
		status, code = "not-ready", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{"status": status})
}

// probeDB pings the database and checks the schema version.
func (o *opsServer) probeDB(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := o.db.PingContext(ctx); err != nil {
		return err
	}
	cur, err := currentSchemaVersion(ctx, o.db)
	if err != nil {
		return err
	}
	if cur != store.SchemaVersion() {
		return errSchemaMismatch
	}
	return nil
}

// metricsAccess admits only clients from the configured internal ranges (OPS-013).
// Requests carrying forwarding headers are refused, because a reverse proxy
// that forwards /metrics would otherwise appear to come from loopback.
func (o *opsServer) metricsAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("Forwarded") != "" || !o.internal(r.RemoteAddr) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (o *opsServer) internal(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, p := range o.cfg.MetricsAllow {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (o *opsServer) serveMetrics(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r) {
		return
	}
	now := time.Now().UTC()
	cur, _ := currentSchemaVersion(r.Context(), o.db)
	dyn := dynamicMetrics{
		version:    version,
		schemaCur:  cur,
		schemaPend: store.SchemaVersion() - cur,
		dbBytes:    fileSize(o.cfg.Database),
		walBytes:   fileSize(o.cfg.Database + "-wal"),
		now:        now,
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	o.metrics.write(w, dyn)
}

// currentSchemaVersion reads the highest applied migration version.
func currentSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&v)
	return v, err
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func allowMethod(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// validRequestID accepts a caller-supplied ID only when it is short and printable.
func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "req-unavailable"
	}
	return hex.EncodeToString(b)
}

// statusRecorder captures the status code and passes Flush and Unwrap through.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wrote = true
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
