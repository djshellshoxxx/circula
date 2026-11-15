package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// This file is the storage boundary of the command (CMD-03, ARC-002). All
// SQLite-specific inspection lives in internal/store; these are the names the
// commands use. Besides serve.go (composition root) and snapshot.go (backup
// tooling), it is the only file here allowed to import internal/store
// (depguard rule "cmd" in .golangci.yml). Everything else the commands read or
// change goes through internal/service.

// schemaState describes the on-disk schema relative to this binary.
type schemaState = store.SchemaState

// Synchronous levels accepted by the db-synchronous setting (SA2-20).
const (
	syncNormal = store.SyncNormal
	syncFull   = store.SyncFull
)

// inspectSchema reads the schema state of the file at path without migrating.
func inspectSchema(ctx context.Context, path string) (schemaState, error) {
	return store.InspectSchema(ctx, path)
}

// openRaw opens a SQLite file without applying migrations (inspection and
// online snapshots only).
func openRaw(path string) (*sql.DB, error) { return store.OpenRaw(path) }

// binarySchemaVersion is the highest schema version this binary knows.
func binarySchemaVersion() int { return store.SchemaVersion() }

// appliedSchemaVersion reads the schema version recorded in db.
func appliedSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	return store.AppliedVersion(ctx, db)
}

// storeOptions maps the configuration to store.Options (SA2-20).
func storeOptions(cfg *Config) store.Options {
	return store.Options{Synchronous: cfg.Synchronous}
}

// openStore opens the database at path, applying pending migrations (the
// storage migration tool, ARC-002), with the configured options.
func openStore(ctx context.Context, path string, cfg *Config) (*store.SQLite, error) {
	return store.OpenWith(ctx, path, storeOptions(cfg))
}

// openDatabase opens the configured database after checking that it exists
// and has no pending migrations. Commands that must not migrate use it.
func openDatabase(ctx context.Context, cfg *Config) (*store.SQLite, *service.Service, error) {
	state, err := inspectSchema(ctx, cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	if !state.Exists {
		return nil, nil, fmt.Errorf("database %s does not exist", cfg.Database)
	}
	if err := state.CheckNotNewer(); err != nil {
		return nil, nil, err
	}
	if state.Pending() > 0 {
		return nil, nil, fmt.Errorf("database has %d pending migration(s); run `circula migrate` first", state.Pending())
	}
	st, err := openStore(ctx, cfg.Database, cfg)
	if err != nil {
		return nil, nil, err
	}
	svc, err := newService(st, cfg)
	if err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	return st, svc, nil
}
