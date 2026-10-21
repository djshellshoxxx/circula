package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	_ "modernc.org/sqlite" // pure-Go driver, no cgo (DEC-001)
)

// SQLite is the embedded default Store (DEC-001): WAL journal, foreign keys
// on, and a process-wide write gate on top of BEGIN IMMEDIATE so postings are
// serialised (LED-5, ARC-006, ARC-011).
type SQLite struct {
	db   *sql.DB
	gate chan struct{}
}

var _ Store = (*SQLite)(nil)

// Open opens (creating if needed) the database file at path, applies pending
// forward-only migrations and returns the store. Use ":memory:"-free paths:
// an in-memory database cannot be shared across pooled connections.
func Open(ctx context.Context, path string) (*SQLite, error) {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(10000)")
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
	if err := s.migrate(ctx); err != nil {
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
	return s.run(ctx, "BEGIN IMMEDIATE", fn)
}

// View runs fn in a deferred transaction, which under WAL is a consistent
// snapshot for every read it performs. Writes through it fail.
func (s *SQLite) View(ctx context.Context, fn func(Tx) error) error {
	return s.run(ctx, "BEGIN", fn)
}

func (s *SQLite) run(ctx context.Context, begin string, fn func(Tx) error) (err error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: conn: %w", err)
	}
	defer func() { _ = conn.Close() }()
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
