// Package sqlstore is the SQLite Repository for the discovery module. It
// works with any database/sql driver for SQLite; the module itself imports no
// driver outside the sqlite_cgo_test build tag (tests/CLI only).
//
// Operational requirements: one process per database file, WAL journal mode,
// a busy timeout, and write transactions serialized through this Store.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version  int
	up, down string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	byV := map[int]*migration{}
	for _, e := range entries {
		name := e.Name()
		parts := strings.SplitN(name, "_", 2)
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("bad migration name %q", name)
		}
		b, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		m := byV[v]
		if m == nil {
			m = &migration{version: v}
			byV[v] = m
		}
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			m.up = string(b)
		case strings.HasSuffix(name, ".down.sql"):
			m.down = string(b)
		}
	}
	var out []migration
	for _, m := range byV {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// DSN builds an SQLite connection string with WAL, a busy timeout and full
// synchronous mode. Parameter names follow mattn/go-sqlite3; a host using
// another driver must set equivalent pragmas itself.
func DSN(path string) string {
	q := url.Values{}
	q.Set("_busy_timeout", "10000")
	q.Set("_journal_mode", "WAL")
	q.Set("_synchronous", "FULL")
	q.Set("_foreign_keys", "on")
	return "file:" + path + "?" + q.Encode()
}

const migTable = `CREATE TABLE IF NOT EXISTS discovery_migrations (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL)`

// Migrate applies pending module migrations (forward-only, each in one
// transaction). It is idempotent and safe to call on every start.
func Migrate(ctx context.Context, db *sql.DB) error {
	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, migTable); err != nil {
		return err
	}
	for _, m := range ms {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM discovery_migrations WHERE version=?`, m.version).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.up); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", m.version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO discovery_migrations (version, applied_at) VALUES (?, ?)`, m.version, time.Now().Unix()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Rollback reverses the newest applied migration (module tables only).
func Rollback(ctx context.Context, db *sql.DB) error {
	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, migTable); err != nil {
		return err
	}
	var v sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT max(version) FROM discovery_migrations`).Scan(&v); err != nil {
		return err
	}
	if !v.Valid {
		return nil
	}
	for i := len(ms) - 1; i >= 0; i-- {
		if int64(ms[i].version) != v.Int64 {
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, ms[i].down); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM discovery_migrations WHERE version=?`, v.Int64); err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	return fmt.Errorf("no down migration for version %d", v.Int64)
}
