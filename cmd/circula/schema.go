package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/djshellshoxxx/circula/internal/store"
)

// schemaState describes the on-disk schema relative to this binary. It is
// read without migrating, so commands can refuse or back up first.
type schemaState struct {
	Exists  bool // the database file exists
	Applied int  // highest applied migration version (0 if none)
	Known   int  // highest migration version embedded in this binary
}

// Pending returns the number of migrations this binary would apply.
func (s schemaState) Pending() int { return s.Known - s.Applied }

// inspectSchema reads schema_migrations from the file at path without
// running migrations. A missing file is reported as Exists false.
func inspectSchema(ctx context.Context, path string) (schemaState, error) {
	known := store.SchemaVersion()
	if known < 0 {
		return schemaState{}, errors.New("embedded migrations are invalid")
	}
	st := schemaState{Known: known}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) { //nolint:gosec // G703: operator-supplied database path
		return st, nil
	} else if err != nil {
		return st, err
	}
	st.Exists = true
	db, err := openRaw(path)
	if err != nil {
		return st, err
	}
	defer func() { _ = db.Close() }()
	var applied int
	err = db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&applied)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		applied = 0
		err = nil
	}
	if err != nil {
		return st, fmt.Errorf("read schema version of %s: %w", path, err)
	}
	st.Applied = applied
	return st, nil
}

// openRaw opens a SQLite file through the driver that the store package
// registers, without applying migrations. It is used only for read-only
// inspection and for online snapshots (VACUUM INTO).
func openRaw(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(2)
	return db, nil
}

// checkSchemaNotNewer refuses a database written by a newer binary.
func (s schemaState) checkSchemaNotNewer() error {
	if s.Exists && s.Applied > s.Known {
		return fmt.Errorf("database schema v%d is newer than this binary (v%d); use a newer circula", s.Applied, s.Known)
	}
	return nil
}
