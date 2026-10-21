package main

import (
	"context"
	"database/sql"

	"github.com/djshellshoxxx/circula/internal/store"
)

// All SQLite-specific inspection lives in internal/store (CMD-03); these are
// the names the commands use.

// schemaState describes the on-disk schema relative to this binary.
type schemaState = store.SchemaState

// inspectSchema reads the schema state of the file at path without migrating.
func inspectSchema(ctx context.Context, path string) (schemaState, error) {
	return store.InspectSchema(ctx, path)
}

// openRaw opens a SQLite file without applying migrations (inspection and
// online snapshots only).
func openRaw(path string) (*sql.DB, error) { return store.OpenRaw(path) }
