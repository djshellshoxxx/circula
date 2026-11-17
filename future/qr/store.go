package qr

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

// Store runs the module's SQLite work on a caller-supplied *sql.DB. The module
// links no SQLite driver: the host (or a test build tag) registers one. Every
// write goes through Update, which holds one BEGIN IMMEDIATE transaction for
// the whole callback, so QR rows and the adapter's proposal rows commit or roll
// back together and a crash can never leave a half-applied claim.
//
// Operational assumptions follow Circula's one-process-per-file model: an
// in-process gate serialises writers and busy_timeout covers short overlaps.
type Store struct {
	db   *sql.DB
	gate chan struct{}
	// BusyTimeout is applied per acquired connection (default 5s).
	BusyTimeout time.Duration
}

// NewStore wraps db. It does not open, migrate or configure the database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db, gate: make(chan struct{}, 1), BusyTimeout: 5 * time.Second}
}

func (s *Store) acquire(ctx context.Context) error {
	select {
	case s.gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Store) release() { <-s.gate }

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

// Update runs fn inside one write transaction. If fn returns an error or
// panics the transaction is rolled back and nothing is persisted.
func (s *Store) Update(ctx context.Context, fn func(ctx context.Context, q DBTX) error) (err error) {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
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
	if err := ctx.Err(); err != nil { // cancelled before commit: roll everything back
		return err
	}
	if _, err := c.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("qr: commit: %w", err)
	}
	done = true
	return nil
}

// View runs fn against a consistent read snapshot.
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
		parts := strings.SplitN(n, "_", 2)
		if len(parts) != 2 {
			continue
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
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
			m.up, m.name = string(body), strings.TrimSuffix(parts[1], ".up.sql")
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

const ledgerDDL = `CREATE TABLE IF NOT EXISTS qr_schema_migrations (
  version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL) STRICT`

// Migrate applies pending additive migrations in order, each in its own
// transaction. It is idempotent and safe to run at every start.
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
			if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM qr_schema_migrations WHERE version=?", m.version).Scan(&n); err != nil {
				return err
			}
			if n > 0 {
				continue
			}
			if m.up == "" {
				return fmt.Errorf("qr: migration %d has no up script", m.version)
			}
			if _, err := q.ExecContext(ctx, m.up); err != nil {
				return fmt.Errorf("qr: migration %d: %w", m.version, err)
			}
			if _, err := q.ExecContext(ctx, "INSERT INTO qr_schema_migrations(version,name,applied_at) VALUES(?,?,?)", m.version, m.name, time.Now().UnixMilli()); err != nil {
				return err
			}
		}
		return nil
	})
}

// Rollback reverts applied migrations newer than toVersion (0 removes all
// module tables). It discards QR feature data only; the host ledger, proposals
// and audit trail are untouched. Operators must back up first.
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
			if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM qr_schema_migrations WHERE version=?", m.version).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				continue
			}
			if m.down == "" {
				return fmt.Errorf("qr: migration %d has no down script", m.version)
			}
			if _, err := q.ExecContext(ctx, m.down); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, "DELETE FROM qr_schema_migrations WHERE version=?", m.version); err != nil {
				return err
			}
		}
		return nil
	})
}

// Backup writes a transactionally consistent copy of the whole database file
// to dest with VACUUM INTO. dest must not exist. Restore is a file copy while
// the process is stopped; there is no feature-specific restore step because
// tokens are stored only as hashes and aliases re-derive from the server secret.
func (s *Store) Backup(ctx context.Context, dest string) error {
	if dest == "" || strings.ContainsRune(dest, 0) {
		return errors.New("qr: invalid backup path")
	}
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	c, err := s.conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	_, err = c.ExecContext(ctx, "VACUUM INTO ?", dest)
	return err
}

// Retention controls Purge.
type Retention struct {
	// Requests: unclaimed requests expired or revoked longer ago than this are deleted.
	Requests time.Duration
	// Idempotency: idempotency rows older than this are deleted. Claimed
	// requests keep their claim fingerprint, so intent replays keep working.
	Idempotency time.Duration
}

// DefaultRetention is 30 days for both. Claimed requests and rotated aliases are
// retained (they are small and keep old codes permanently revoked).
var DefaultRetention = Retention{Requests: 30 * 24 * time.Hour, Idempotency: 30 * 24 * time.Hour}

// Purge deletes expired data per Retention and returns the rows removed.
func (s *Store) Purge(ctx context.Context, now time.Time, r Retention) (requests, idem int64, err error) {
	if r.Requests <= 0 || r.Idempotency <= 0 {
		return 0, 0, errors.New("qr: retention must be positive")
	}
	err = s.Update(ctx, func(ctx context.Context, q DBTX) error {
		res, err := q.ExecContext(ctx, `DELETE FROM qr_trade_request WHERE claimed_tx_id IS NULL AND
			(expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?))`,
			now.Add(-r.Requests).UnixMilli(), now.Add(-r.Requests).UnixMilli())
		if err != nil {
			return err
		}
		requests, _ = res.RowsAffected()
		res, err = q.ExecContext(ctx, `DELETE FROM qr_idempotency WHERE created_at < ?`, now.Add(-r.Idempotency).UnixMilli())
		if err != nil {
			return err
		}
		idem, _ = res.RowsAffected()
		return nil
	})
	return
}
