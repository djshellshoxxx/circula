package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/web"
)

var errSchemaMismatch = errors.New("database schema does not match this binary")

const (
	expiryInterval      = 5 * time.Minute
	verifyInterval      = time.Hour
	purgeInterval       = time.Hour
	backupRetryInterval = 15 * time.Minute
	shutdownGracePeriod = 30 * time.Second
)

// newService builds the service layer with the configured session, password
// and idempotency settings.
func newService(st store.Store, cfg *Config) (*service.Service, error) {
	return service.New(st, service.Config{
		SessionIdle:    cfg.SessionIdle,
		SessionMax:     cfg.SessionMax,
		IdempotencyTTL: cfg.IdempotencyTTL,
		MinPasswordLen: cfg.MinPasswordLength,

		RequireMFAForAdmins: cfg.RequireMFAAdmins,
		SecretKey:           cfg.MFASecret,
	})
}

func newLogger(cfg *Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "text" {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}

// cmdServe runs the HTTP service and its scheduled jobs.
func cmdServe(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("serve", stderr)
	checkOnly := fs.Bool("check-config", false, "validate the configuration, print effective settings and exit (no database access)")
	sf := bindSettingFlags(fs)
	if code, done := parseFlags(fs, args); done {
		return code
	}
	cfg, vals, err := sf.Resolve(os.LookupEnv, true)
	if err != nil {
		fmt.Fprintf(stderr, "circula: invalid configuration:\n%v\n", err)
		return exitUsage
	}
	if *checkOnly {
		writeEffective(stdout, vals)
		writeSecretStatus(stdout, cfg)
		return exitOK
	}
	logger := newLogger(cfg, stderr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runServe(ctx, cfg, logger); err != nil {
		if errors.Is(err, ErrLocked) {
			fmt.Fprintf(stderr, "circula: cannot start: %v (%s). Only one serve process may use a database file (ARC-011); stop the other process first.\n",
				err, lockPath(cfg.Database))
			return exitFailure
		}
		fmt.Fprintf(stderr, "circula: serve: %v\n", err)
		return exitFailure
	}
	return exitOK
}

// runServe performs the start-up sequence and blocks until ctx is cancelled
// or the listener fails. Order: lock; schema check and pre-migration backup;
// open and migrate; pragma check; startup ledger verification; listen.
func runServe(ctx context.Context, cfg *Config, logger *slog.Logger) error {
	if err := ensurePrivateDir(cfg.BackupDir); err != nil && cfg.BackupInterval > 0 {
		return fmt.Errorf("backup directory %s: %w", cfg.BackupDir, err)
	}
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	m := newMetrics(time.Now().UTC())
	state, err := inspectSchema(ctx, cfg.Database)
	if err != nil {
		return err
	}
	if err := state.checkSchemaNotNewer(); err != nil {
		return err
	}
	if state.Pending() > 0 && state.Exists {
		if cfg.MigrateMode == "explicit" {
			return fmt.Errorf("database has %d pending migration(s) (schema v%d, binary v%d); "+
				"run `circula migrate` to back up and migrate, then start again", state.Pending(), state.Applied, state.Known)
		}
		if err := preMigrationBackup(ctx, cfg, logger, m, state); err != nil {
			m.migrationRan("refused")
			return err
		}
	}

	if tightened, err := secureDatabase(cfg.Database); err != nil {
		return err
	} else if tightened {
		logger.Warn("database file was readable by group or others; permissions restricted to the owner", "file", cfg.Database)
	}
	st, err := store.Open(ctx, cfg.Database)
	if err != nil {
		m.migrationRan("failed")
		return err
	}
	defer func() { _ = st.Close() }()
	if state.Pending() > 0 {
		m.migrationRan("applied")
		logger.Info("migrations applied", "from", state.Applied, "to", state.Known)
	}
	if err := checkPragmas(ctx, st.DB()); err != nil {
		return err
	}
	svc, err := newService(st, cfg)
	if err != nil {
		return err
	}

	checks, err := verifyAllNetworks(ctx, svc, st.DB(), "startup-verify", "")
	if err != nil {
		return fmt.Errorf("startup ledger verification: %w", err)
	}
	sum := summarise(checks)
	m.ledgerResult(sum.Failing == 0, sum.Failing, time.Now().UTC())
	if sum.Failing > 0 {
		logger.Error("ledger verification failed at start-up; serving refused", "failing_networks", sum.Failing)
		return fmt.Errorf("ledger verification failed for %d network(s); serving refused. Run `circula verify-ledger` "+
			"and restore a verified backup if the fault is confirmed (OPS-016, OPS-020)", sum.Failing)
	}

	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Listen, err)
	}
	ops := &opsServer{cfg: cfg, db: st.DB(), logger: logger, metrics: m}
	apiH, webH, err := newHandlers(svc, cfg, logger)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           ops.routes(apiH, webH),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.RequestTimeout,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 16,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	logger.Info("circula serving", "version", version, "listen", cfg.Listen,
		"public_base_url", cfg.PublicBaseURL, "schema_version", store.SchemaVersion())
	if !isLoopback(cfg.Listen) {
		logger.Warn("listening on a non-loopback address; put TLS termination and /metrics filtering in front of it")
		if cfg.MetricsToken == "" {
			logger.Warn("no metrics-token configured; /metrics relies on the source address check only (SA-15)")
		}
	}
	if cfg.SetupSecret == "" {
		logger.Warn("no setup-secret configured: any signed-in person can create a network (FR-002); set setup-secret-file or " + envPrefix + "SETUP_SECRET")
	}
	if !cfg.CookieSecure {
		logger.Warn("cookie-secure is false: session cookies are not marked Secure (SEC-022)")
	}

	jobCtx, cancelJobs := context.WithCancel(context.Background())
	var jobs sync.WaitGroup
	jobs.Add(1)
	go func() {
		defer jobs.Done()
		(&jobRunner{cfg: cfg, svc: svc, st: st, logger: logger, metrics: m}).run(jobCtx)
	}()

	ops.ready.Store(true)
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	}
	ops.ready.Store(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown: %w", err)
	}
	cancelJobs()
	jobs.Wait()
	logger.Info("circula stopped")
	return runErr
}

// newHandlers builds the JSON API and the HTML UI with one shared edge policy
// and CSRF key, so a browser session works for both (API-01) and the
// operator's settings reach the adapters (WEB-01): setup secret, Secure
// cookies, trusted proxies, public base URL and session lifetimes.
func newHandlers(svc *service.Service, cfg *Config, logger *slog.Logger) (apiH, webH http.Handler, err error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, nil, fmt.Errorf("csrf key: %w", err)
	}
	edge := httpx.Edge{
		ForceSecure:    cfg.CookieSecure,
		PublicHTTPS:    strings.HasPrefix(cfg.PublicBaseURL, "https://"),
		TrustedProxies: cfg.TrustedProxies,
	}
	apiH = api.NewHandlerWithOptions(svc, api.Options{Logger: logger, CSRFKey: key, Edge: edge})
	webH = web.NewHandlerWith(svc, web.Options{
		SetupSecret: cfg.SetupSecret, Edge: edge, PublicBaseURL: cfg.PublicBaseURL, CSRFKey: key,
		SessionIdle: cfg.SessionIdle, SessionMax: cfg.SessionMax,
	})
	return apiH, webH, nil
}

// preMigrationBackup takes a verified-by-construction backup before any
// migration of an existing database (OPS-018, OPS-023). Migration is refused
// if the backup fails.
func preMigrationBackup(ctx context.Context, cfg *Config, logger *slog.Logger, m *metrics, state schemaState) error {
	if state.Applied == 0 {
		return nil // nothing recorded yet; there is no data to protect
	}
	logger.Info("taking pre-migration backup", "from", state.Applied, "to", state.Known)
	mf, path, err := backupTo(ctx, cfg, cfg.BackupDir, "pre-migration", nil)
	if err != nil {
		return fmt.Errorf("pre-migration backup failed; migrations not applied: %w", err)
	}
	m.backupSucceeded(mf.CreatedAt)
	logger.Info("pre-migration backup written", "file", filepath.Base(path))
	return nil
}

// backupTo writes a snapshot of the database into dir and returns its manifest
// and path. A nil db means the database is opened for the duration of the call.
func backupTo(ctx context.Context, cfg *Config, dir, reason string, db *sql.DB) (*Manifest, string, error) {
	if err := ensurePrivateDir(dir); err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	dest := filepath.Join(dir, backupFileName(now))
	if db == nil {
		raw, err := openRaw(cfg.Database)
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = raw.Close() }()
		db = raw
	}
	ledger := func(ctx context.Context, path string) (LedgerSummary, error) {
		return verifyFile(ctx, path, cfg)
	}
	mf, err := snapshot(ctx, db, dest, reason, now, ledger, cfg.BackupKey)
	return mf, dest, err
}

// checkPragmas confirms the connection settings the design depends on (OPS-011).
func checkPragmas(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	var mode string
	if err := conn.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&mode); err != nil {
		return err
	}
	if mode != "wal" {
		return fmt.Errorf("journal_mode is %q; expected wal (OPS-011)", mode)
	}
	var fk int
	if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk); err != nil {
		return err
	}
	if fk != 1 {
		return errors.New("foreign key enforcement is off; expected on (OPS-011)")
	}
	return nil
}

// jobRunner runs the scheduled jobs: proposal expiry (OPS-027, OPS-028),
// periodic ledger verification (OPS-016) and backups (OPS-018).
type jobRunner struct {
	cfg     *Config
	svc     *service.Service
	st      *store.SQLite
	logger  *slog.Logger
	metrics *metrics
}

func (j *jobRunner) run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); j.every(ctx, "expiry", expiryInterval, j.expire) }()
	go func() { defer wg.Done(); j.every(ctx, "verify", verifyInterval, j.verify) }()
	go func() { defer wg.Done(); j.every(ctx, "purge", purgeInterval, j.purge) }()
	go func() { defer wg.Done(); j.backups(ctx) }()
	wg.Wait()
}

// every runs fn at start and then on each interval until ctx ends.
func (j *jobRunner) every(ctx context.Context, name string, interval time.Duration, fn func(context.Context) error) {
	for {
		j.attempt(ctx, name, fn)
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func (j *jobRunner) attempt(ctx context.Context, name string, fn func(context.Context) error) {
	if err := fn(ctx); err != nil {
		if ctx.Err() != nil {
			return
		}
		j.metrics.jobRan(name, "failed")
		j.logger.Error("scheduled job failed", "job", name, "error", err)
		return
	}
	j.metrics.jobRan(name, "ok")
}

func (j *jobRunner) expire(ctx context.Context) error {
	total := 0
	for {
		n, err := j.svc.ExpireProposals(ctx, domain.SystemActor("job-expiry"), 500)
		if err != nil {
			return err
		}
		total += n
		if n < 500 {
			break
		}
	}
	if total > 0 {
		j.logger.Info("proposals expired", "count", total)
	}
	return nil
}

// purge deletes expired sessions, throttle rows and idempotency records (SA-12).
func (j *jobRunner) purge(ctx context.Context) error {
	rep, err := j.svc.Purge(ctx)
	if err != nil {
		return err
	}
	j.logger.Debug("expired rows purged", "report", fmt.Sprintf("%+v", rep))
	return nil
}

func (j *jobRunner) verify(ctx context.Context) error {
	checks, err := verifyAllNetworks(ctx, j.svc, j.st.DB(), "job-verify", "")
	if err != nil {
		return err
	}
	sum := summarise(checks)
	j.metrics.ledgerResult(sum.Failing == 0, sum.Failing, time.Now().UTC())
	if sum.Failing > 0 {
		j.logger.Error("ledger verification failed", "failing_networks", sum.Failing)
	}
	return nil
}

// backups takes a backup when the newest one is older than backup-interval,
// then prunes according to retention. Retries use a shorter wait.
func (j *jobRunner) backups(ctx context.Context) {
	if j.cfg.BackupInterval <= 0 {
		return
	}
	for {
		wait := j.untilNextBackup()
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if err := j.backupNow(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			j.metrics.jobRan("backup", "failed")
			j.logger.Error("scheduled backup failed", "error", err)
			wait = backupRetryInterval
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			continue
		}
	}
}

// untilNextBackup returns how long to wait before the next scheduled backup.
func (j *jobRunner) untilNextBackup() time.Duration {
	entries, err := listBackups(j.cfg.BackupDir)
	if err != nil || len(entries) == 0 {
		return 0
	}
	due := entries[0].at.Add(j.cfg.BackupInterval)
	if d := time.Until(due); d > 0 {
		return d
	}
	return 0
}

func (j *jobRunner) backupNow(ctx context.Context) error {
	mf, path, err := backupTo(ctx, j.cfg, j.cfg.BackupDir, "scheduled", nil)
	if err != nil {
		return err
	}
	j.metrics.backupSucceeded(mf.CreatedAt)
	j.metrics.jobRan("backup", "ok")
	if mf.Ledger.Status == "failed" {
		j.metrics.ledgerResult(false, mf.Ledger.Failing, time.Now().UTC())
		j.logger.Error("scheduled backup recorded a ledger verification failure", "file", filepath.Base(path))
	}
	j.logger.Info("backup written", "file", filepath.Base(path), "bytes", mf.Bytes, "ledger", mf.Ledger.Status)
	removed, err := pruneBackups(j.cfg.BackupDir, j.cfg.BackupRetainDaily, j.cfg.BackupRetainMonthly)
	if err != nil {
		return fmt.Errorf("prune backups: %w", err)
	}
	if len(removed) > 0 {
		j.logger.Info("expired backups removed", "count", len(removed))
	}
	return nil
}

func isLoopback(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// newFlagSet returns a flag set that reports errors to stderr.
func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("circula "+name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

// parseFlags parses args. done is true when the caller must return code.
func parseFlags(fs *flag.FlagSet, args []string) (code int, done bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK, true
		}
		return exitUsage, true
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(fs.Output(), "circula: unexpected arguments: %v\n", fs.Args())
		return exitUsage, true
	}
	return 0, false
}
