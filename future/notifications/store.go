package notifications

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed sql/*.sql
var sqlFS embed.FS

// DBTX is the subset of *sql.DB, *sql.Tx and *sql.Conn the module needs. The
// host adapter passes its own open transaction to EnqueueTx so outbox rows
// commit or roll back together with the source event.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store runs the module's SQLite work on a caller-supplied *sql.DB. No SQLite
// driver is linked by the module: the host registers one (tests use the
// test-only cgo build tag, see VALIDATION.md). Writes go through Update, which
// holds one BEGIN IMMEDIATE transaction for the whole callback; an in-process
// gate serialises writers (Circula's one-process-per-file model) and
// busy_timeout covers short overlaps with the host's own connections.
type Store struct {
	db          *sql.DB
	gate        chan struct{}
	BusyTimeout time.Duration
}

// NewStore wraps db without opening, migrating or configuring it.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db, gate: make(chan struct{}, 1), BusyTimeout: 5 * time.Second}
}

// DB exposes the underlying handle for host-owned operations (backup etc.).
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) conn(ctx context.Context) (*sql.Conn, error) {
	c, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := c.ExecContext(ctx, "PRAGMA busy_timeout="+strconv.Itoa(int(s.BusyTimeout/time.Millisecond))); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

// Update runs fn in one write transaction; an error, panic or cancelled
// context rolls everything back.
func (s *Store) Update(ctx context.Context, fn func(ctx context.Context, q DBTX) error) error {
	select {
	case s.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.gate }()
	c, err := s.conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := c.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	done := false
	defer func() {
		if !done {
			rb, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = c.ExecContext(rb, "ROLLBACK")
		}
	}()
	if err := fn(ctx, c); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := c.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("notifications: commit: %w", err)
	}
	done = true
	return nil
}

// View runs fn against a read snapshot.
func (s *Store) View(ctx context.Context, fn func(ctx context.Context, q DBTX) error) error {
	c, err := s.conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := c.ExecContext(ctx, "BEGIN"); err != nil {
		return err
	}
	defer func() {
		rb, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = c.ExecContext(rb, "ROLLBACK")
	}()
	return fn(ctx, c)
}

type migration struct {
	version  int
	name     string
	up, down string
}

func loadMigrations() ([]migration, error) {
	ents, err := sqlFS.ReadDir("sql")
	if err != nil {
		return nil, err
	}
	byVer := map[int]*migration{}
	for _, e := range ents {
		n := e.Name()
		ver, rest, ok := strings.Cut(n, "_")
		v, err := strconv.Atoi(ver)
		if !ok || err != nil {
			continue
		}
		body, err := sqlFS.ReadFile("sql/" + n)
		if err != nil {
			return nil, err
		}
		m := byVer[v]
		if m == nil {
			m = &migration{version: v}
			byVer[v] = m
		}
		switch {
		case strings.HasSuffix(n, ".up.sql"):
			m.up, m.name = string(body), strings.TrimSuffix(rest, ".up.sql")
		case strings.HasSuffix(n, ".down.sql"):
			m.down = string(body)
		}
	}
	out := make([]migration, 0, len(byVer))
	for _, m := range byVer {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

const ledgerDDL = `CREATE TABLE IF NOT EXISTS notification_schema_migrations (
  version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`

// SchemaVersion is the newest migration this build knows.
func SchemaVersion() int {
	ms, _ := loadMigrations()
	if len(ms) == 0 {
		return 0
	}
	return ms[len(ms)-1].version
}

// Migrate applies pending additive forward migrations. Idempotent.
func (s *Store) Migrate(ctx context.Context) error {
	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	return s.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := q.ExecContext(ctx, ledgerDDL); err != nil {
			return err
		}
		for _, m := range ms {
			var n int
			if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM notification_schema_migrations WHERE version=?", m.version).Scan(&n); err != nil {
				return err
			}
			if n > 0 {
				continue
			}
			if m.up == "" {
				return fmt.Errorf("notifications: migration %d has no up script", m.version)
			}
			if _, err := q.ExecContext(ctx, m.up); err != nil {
				return fmt.Errorf("notifications: migration %d: %w", m.version, err)
			}
			if _, err := q.ExecContext(ctx, "INSERT INTO notification_schema_migrations(version,name,applied_at) VALUES(?,?,?)", m.version, m.name, time.Now().Unix()); err != nil {
				return err
			}
		}
		return nil
	})
}

// Rollback reverts migrations newer than toVersion (0 drops every module
// table). Feature data is discarded; back up first.
func (s *Store) Rollback(ctx context.Context, toVersion int) error {
	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	return s.Update(ctx, func(ctx context.Context, q DBTX) error {
		for i := len(ms) - 1; i >= 0; i-- {
			m := ms[i]
			if m.version <= toVersion {
				break
			}
			var n int
			if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM notification_schema_migrations WHERE version=?", m.version).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				continue
			}
			if m.down == "" {
				return fmt.Errorf("notifications: migration %d has no down script", m.version)
			}
			if _, err := q.ExecContext(ctx, m.down); err != nil {
				return fmt.Errorf("notifications: rollback %d: %w", m.version, err)
			}
			if _, err := q.ExecContext(ctx, "DELETE FROM notification_schema_migrations WHERE version=?", m.version); err != nil {
				return err
			}
		}
		return nil
	})
}

var errRows = errors.New("notifications: unexpected row count")
