package store

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var ms []migration
	for _, e := range entries {
		name := e.Name()
		parts := strings.SplitN(strings.TrimSuffix(name, ".sql"), "_", 2)
		v, err := strconv.Atoi(parts[0])
		if err != nil || len(parts) != 2 {
			return nil, fmt.Errorf("store: bad migration name %q", name)
		}
		b, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		ms = append(ms, migration{v, parts[1], string(b)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	for i, m := range ms {
		if m.version != i+1 {
			return nil, fmt.Errorf("store: migration versions must be contiguous, got %d at position %d", m.version, i+1)
		}
	}
	return ms, nil
}

// SchemaVersion returns the highest migration version known to this binary.
func SchemaVersion() int {
	ms, err := loadMigrations()
	if err != nil {
		return -1
	}
	return len(ms)
}

// migrate applies pending migrations in order, each inside its own
// BEGIN IMMEDIATE transaction (DDL is transactional in SQLite), so a failed
// migration leaves no partial schema (DAT-042). It refuses to run against a
// database newer than the binary or one whose recorded names differ.
func (s *SQLite) migrate(ctx context.Context) error { return s.migrateTo(ctx, 0) }

// migrateTo is migrate limited to versions up to limit (0 = all). Upgrade
// tests use it to build a database at an earlier schema (DAT-047).
func (s *SQLite) migrateTo(ctx context.Context, limit int) error {
	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	all := ms
	if limit > 0 && limit < len(ms) {
		ms = ms[:limit]
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("store: migrations table: %w", err)
	}
	applied := map[int]string{}
	rows, err := conn.QueryContext(ctx, `SELECT version, name FROM schema_migrations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v int
		var n string
		if err := rows.Scan(&v, &n); err != nil {
			_ = rows.Close()
			return err
		}
		applied[v] = n
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for v, n := range applied {
		if v > len(all) {
			return fmt.Errorf("store: database schema v%d is newer than this binary (v%d)", v, len(all))
		}
		if all[v-1].name != n {
			return fmt.Errorf("store: migration %d recorded as %q but binary has %q", v, n, all[v-1].name)
		}
	}
	for _, m := range ms {
		if _, ok := applied[m.version]; ok {
			continue
		}
		if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
			return err
		}
		_, err := conn.ExecContext(ctx, m.sql)
		if err == nil {
			_, err = conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,applied_at) VALUES (?,?,?)`,
				m.version, m.name, time.Now().UnixMilli())
		}
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			return fmt.Errorf("store: migration %04d_%s: %w", m.version, m.name, err)
		}
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			return err
		}
	}
	return nil
}
