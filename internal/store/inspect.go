package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// SchemaState describes the on-disk schema relative to this binary. It is
// read without migrating, so commands can refuse or back up first (OPS-018).
type SchemaState struct {
	Exists  bool // the database file exists
	Applied int  // highest applied migration version (0 if none)
	Known   int  // highest migration version embedded in this binary
}

// Pending returns the number of migrations this binary would apply.
func (s SchemaState) Pending() int { return s.Known - s.Applied }

// CheckNotNewer refuses a database written by a newer binary.
func (s SchemaState) CheckNotNewer() error {
	if s.Exists && s.Applied > s.Known {
		return fmt.Errorf("database schema v%d is newer than this binary (v%d); use a newer circula", s.Applied, s.Known)
	}
	return nil
}

// InspectSchema reads schema_migrations from the file at path without running
// migrations. A missing file is reported as Exists false.
func InspectSchema(ctx context.Context, path string) (SchemaState, error) {
	known := SchemaVersion()
	if known < 0 {
		return SchemaState{}, errors.New("embedded migrations are invalid")
	}
	st := SchemaState{Known: known}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return st, nil
	} else if err != nil {
		return st, err
	}
	st.Exists = true
	db, err := OpenRaw(path)
	if err != nil {
		return st, err
	}
	defer func() { _ = db.Close() }()
	applied, err := AppliedVersion(ctx, db)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		applied, err = 0, nil
	}
	if err != nil {
		return st, fmt.Errorf("read schema version of %s: %w", path, err)
	}
	st.Applied = applied
	return st, nil
}

// AppliedVersion returns the highest applied migration version of the open
// database (0 when none). It errors with "no such table" on a file that was
// never migrated.
func AppliedVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&v)
	return v, err
}

// OpenRaw opens a SQLite file through the registered driver without applying
// migrations. It is used only for read-only inspection and for online
// snapshots (VACUUM INTO).
func OpenRaw(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(2)
	return db, nil
}

// NetworkRef identifies a network.
type NetworkRef struct{ ID, Slug string }

// ListNetworks returns the networks ordered by id. A non-empty filter keeps
// only the network whose id or slug equals it.
func ListNetworks(ctx context.Context, db *sql.DB, filter string) ([]NetworkRef, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, slug FROM networks WHERE ?1 = '' OR id = ?1 OR slug = ?1 ORDER BY id`, filter)
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []NetworkRef
	for rows.Next() {
		var n NetworkRef
		if err := rows.Scan(&n.ID, &n.Slug); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// Pragmas are the connection settings the design depends on (OPS-011).
type Pragmas struct {
	JournalMode string
	ForeignKeys bool
}

// ReadPragmas reads the settings on one connection of db.
func ReadPragmas(ctx context.Context, db *sql.DB) (Pragmas, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return Pragmas{}, err
	}
	defer func() { _ = conn.Close() }()
	var p Pragmas
	if err := conn.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&p.JournalMode); err != nil {
		return Pragmas{}, err
	}
	var fk int
	if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk); err != nil {
		return Pragmas{}, err
	}
	p.ForeignKeys = fk == 1
	return p, nil
}

// CheckPragmas fails when the connection settings differ from the design.
func CheckPragmas(ctx context.Context, db *sql.DB) error {
	p, err := ReadPragmas(ctx, db)
	if err != nil {
		return err
	}
	if p.JournalMode != "wal" {
		return fmt.Errorf("journal_mode is %q; expected wal (OPS-011)", p.JournalMode)
	}
	if !p.ForeignKeys {
		return errors.New("foreign key enforcement is off; expected on (OPS-011)")
	}
	return nil
}

// VacuumInto writes a consistent snapshot of the database behind db to dest
// (which must not exist). It reads a snapshot and does not hold the write lock.
func VacuumInto(ctx context.Context, db *sql.DB, dest string) error {
	_, err := db.ExecContext(ctx, "VACUUM INTO ?", dest)
	return err
}

// IntegrityCheck runs PRAGMA integrity_check on the file at path and returns
// "ok" or the first reported problem.
func IntegrityCheck(ctx context.Context, path string) (string, error) {
	db, err := OpenRaw(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = db.Close() }()
	var res string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&res); err != nil {
		return "", fmt.Errorf("integrity check of %s: %w", path, err)
	}
	return res, nil
}
