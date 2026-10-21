package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	sqlite "modernc.org/sqlite" // pure-Go driver, no cgo (DEC-001)
)

// SQLite is the embedded default Store (DEC-001): WAL journal, foreign keys
// on, and a process-wide write gate on top of BEGIN IMMEDIATE so postings are
// serialised (LED-5, ARC-006, ARC-011).
type SQLite struct {
	db   *sql.DB
	gate chan struct{}
}

var _ Store = (*SQLite)(nil)

// DefaultBusyTimeout is how long a statement waits for another process's lock
// before failing with the retryable domain code "unavailable". In-process
// writers never reach SQLite's lock concurrently (the gate serialises them),
// so the wait only matters against a second process (operator CLI, backup) and
// is kept short so a request fails fast instead of holding a worker.
const DefaultBusyTimeout = 5 * time.Second

// Options tunes Open.
type Options struct {
	// BusyTimeout overrides DefaultBusyTimeout (zero = default).
	BusyTimeout time.Duration
}

// Open opens (creating if needed) the database file at path, applies pending
// forward-only migrations and returns the store. Use ":memory:"-free paths:
// an in-memory database cannot be shared across pooled connections.
func Open(ctx context.Context, path string) (*SQLite, error) {
	return OpenWith(ctx, path, Options{})
}

// OpenWith is Open with explicit options.
func OpenWith(ctx context.Context, path string, opt Options) (*SQLite, error) {
	return openAt(ctx, path, opt, 0)
}

// OpenAtVersion opens (creating if needed) the database but migrates it only
// up to schema version v. It exists so upgrade tests can build a database as
// an earlier release left it (DAT-047); production code uses Open.
func OpenAtVersion(ctx context.Context, path string, v int) (*SQLite, error) {
	if v < 1 || v > SchemaVersion() {
		return nil, fmt.Errorf("store: schema version %d is not between 1 and %d", v, SchemaVersion())
	}
	return openAt(ctx, path, Options{}, v)
}

func openAt(ctx context.Context, path string, opt Options, limit int) (*SQLite, error) {
	busy := opt.BusyTimeout
	if busy <= 0 {
		busy = DefaultBusyTimeout
	}
	q := url.Values{}
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busy.Milliseconds()))
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "foreign_keys(1)")
	dsn := "file:" + path + "?" + q.Encode()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetConnMaxLifetime(0)
	s := &SQLite{db: db, gate: make(chan struct{}, 1)}
	if err := s.migrateTo(ctx, limit); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database.
func (s *SQLite) Close() error { return s.db.Close() }

// DB exposes the pool for tests that attempt raw access (DAT-011).
func (s *SQLite) DB() *sql.DB { return s.db }

// Update runs fn in one BEGIN IMMEDIATE transaction. Writers are serialised
// in-process by a channel gate (honours ctx) and across processes by SQLite.
func (s *SQLite) Update(ctx context.Context, fn func(Tx) error) error {
	select {
	case s.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.gate }()
	return s.run(ctx, "BEGIN IMMEDIATE", false, fn)
}

// View runs fn in a deferred transaction, which under WAL is a consistent
// snapshot for every read it performs. The connection runs with
// PRAGMA query_only=ON for the duration, so any write through it fails with
// "attempt to write a readonly database" and cannot bypass the write gate
// (SVC-02).
func (s *SQLite) View(ctx context.Context, fn func(Tx) error) error {
	return s.run(ctx, "BEGIN", true, fn)
}

// busyErr is the retryable error for lock contention.
func busyErr(cause error) error {
	return &domain.Error{Code: domain.CodeUnavailable, Msg: "database busy; retry", Cause: cause}
}

// isBusy reports SQLITE_BUSY / SQLITE_LOCKED (including extended codes).
func isBusy(err error) bool {
	if err == nil {
		return false
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		c := se.Code() & 0xff
		return c == 5 || c == 6
	}
	m := err.Error()
	return strings.Contains(m, "SQLITE_BUSY") || strings.Contains(m, "SQLITE_LOCKED") || strings.Contains(m, "database is locked")
}

func (s *SQLite) run(ctx context.Context, begin string, readOnly bool, fn func(Tx) error) (err error) {
	// Contention surfaces as the retryable domain error, wherever it arose.
	defer func() {
		var de *domain.Error
		if err != nil && !errors.As(err, &de) && isBusy(err) {
			err = busyErr(err)
		}
	}()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: conn: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if readOnly {
		if _, err = conn.ExecContext(ctx, "PRAGMA query_only = ON"); err != nil {
			return fmt.Errorf("store: query_only: %w", err)
		}
		defer func() {
			// Restore the pooled connection; if that fails, discard it so a
			// read-only connection can never be handed to a writer.
			if _, rerr := conn.ExecContext(context.Background(), "PRAGMA query_only = OFF"); rerr != nil {
				_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			}
		}()
	}
	if _, err = conn.ExecContext(ctx, begin); err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	done := false
	defer func() {
		if done {
			return
		}
		// Roll back on error or panic, even if ctx is already cancelled.
		_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		if r := recover(); r != nil {
			panic(r)
		}
	}()
	if err = fn(&txImpl{ctx: ctx, q: conn}); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	done = true
	return nil
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txImpl struct {
	ctx context.Context
	q   querier
}

var _ Tx = (*txImpl)(nil)

func (t *txImpl) exec(query string, args ...any) (sql.Result, error) {
	res, err := t.q.ExecContext(t.ctx, query, args...)
	return res, mapErr(err)
}

func (t *txImpl) query(query string, args ...any) (*sql.Rows, error) {
	rows, err := t.q.QueryContext(t.ctx, query, args...)
	return rows, mapErr(err)
}

func (t *txImpl) row(query string, args ...any) *sql.Row {
	return t.q.QueryRowContext(t.ctx, query, args...)
}

// mapErr converts driver errors into domain errors where callers branch on them.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NotFound("record")
	}
	if isBusy(err) {
		return busyErr(err)
	}
	m := err.Error()
	if strings.Contains(m, "UNIQUE constraint failed") || strings.Contains(m, "PRIMARY KEY") {
		return domain.Errf(domain.CodeConflict, "duplicate record")
	}
	return err
}

// notFound turns a missing row into a typed not-found for `what`.
func notFound(err error, what string) error {
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		return domain.NotFound(what)
	}
	return mapErr(err)
}

// ---- time and nullable helpers (UTC milliseconds, DAT-017) ----

func ms(t time.Time) int64 { return t.UTC().UnixMilli() }

func fromMS(v int64) time.Time { return time.UnixMilli(v).UTC() }

func msPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return ms(*t)
}

func ptrTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := fromMS(v.Int64)
	return &t
}

func ptrInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// inClause builds "(?,?,?)" and args for a string-like slice.
func inClause[T ~string](vals []T) (string, []any) {
	ph := make([]string, len(vals))
	args := make([]any, len(vals))
	for i, v := range vals {
		ph[i] = "?"
		args[i] = string(v)
	}
	return "(" + strings.Join(ph, ",") + ")", args
}

func limitOf(n int) int {
	if n <= 0 || n > 500 {
		return 100
	}
	return n
}
