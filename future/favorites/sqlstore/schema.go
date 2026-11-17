package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SchemaVersion is the newest module-local schema version.
const SchemaVersion = 1

// All DDL is additive and uses module-owned table names. Foreign keys to the
// host application's members table are intentionally absent here: this module
// cannot reference root tables. The integration migration must add them (or
// call Service.MemberDeparted / PurgeMemberTx from the departure
// transaction); see README "Persistence and retention".
const upV1 = `
CREATE TABLE member_favorites (
  network_id       TEXT NOT NULL CHECK (length(network_id) BETWEEN 1 AND 128),
  owner_member_id  TEXT NOT NULL CHECK (length(owner_member_id) BETWEEN 1 AND 128),
  target_member_id TEXT NOT NULL CHECK (length(target_member_id) BETWEEN 1 AND 128),
  nickname         TEXT NOT NULL DEFAULT '' CHECK (length(nickname) <= 80),
  pinned           INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
  position         INTEGER NOT NULL CHECK (position >= 0),
  created_at       TEXT NOT NULL,
  updated_at       TEXT NOT NULL,
  PRIMARY KEY (network_id, owner_member_id, target_member_id),
  CHECK (owner_member_id <> target_member_id)
) WITHOUT ROWID;
CREATE INDEX member_favorites_by_target ON member_favorites (network_id, target_member_id);
CREATE TABLE favorites_owner_state (
  network_id      TEXT NOT NULL,
  owner_member_id TEXT NOT NULL,
  revision        INTEGER NOT NULL CHECK (revision >= 0),
  PRIMARY KEY (network_id, owner_member_id)
) WITHOUT ROWID;
CREATE TABLE favorite_repeat_intents (
  network_id       TEXT NOT NULL,
  owner_member_id  TEXT NOT NULL,
  request_key      TEXT NOT NULL CHECK (length(request_key) BETWEEN 1 AND 64),
  trade_id         TEXT NOT NULL DEFAULT '',
  target_member_id TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',
  intent           TEXT NOT NULL UNIQUE,
  created_at       TEXT NOT NULL,
  expires_at       TEXT NOT NULL,
  PRIMARY KEY (network_id, owner_member_id, request_key)
) WITHOUT ROWID;
CREATE INDEX favorite_repeat_intents_by_target ON favorite_repeat_intents (network_id, target_member_id);
CREATE INDEX favorite_repeat_intents_by_expiry ON favorite_repeat_intents (expires_at);
`

// downV1 is the documented rollback. It is destructive: it deletes all
// favorites, revisions and drafts. Back up the database file first.
const downV1 = `
DROP TABLE IF EXISTS favorite_repeat_intents;
DROP TABLE IF EXISTS favorites_owner_state;
DROP TABLE IF EXISTS member_favorites;
`

const migrationsTable = `CREATE TABLE IF NOT EXISTS favorites_schema_migrations (
  version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`

// ErrNewerSchema means the database was migrated by a newer version of this module.
var ErrNewerSchema = errors.New("sqlstore: database schema is newer than this module")

// Migrate applies pending module-local migrations in one transaction. It is
// idempotent and forward-only; it refuses a database that is ahead of this build.
func Migrate(ctx context.Context, db *sql.DB) error {
	return withImmediate(ctx, db, func(x DBTX) error {
		if _, err := x.ExecContext(ctx, migrationsTable); err != nil {
			return err
		}
		v, err := currentVersion(ctx, x)
		if err != nil {
			return err
		}
		if v > SchemaVersion {
			return ErrNewerSchema
		}
		if v < 1 {
			if _, err := x.ExecContext(ctx, upV1); err != nil {
				return fmt.Errorf("sqlstore: migration 1: %w", err)
			}
			if _, err := x.ExecContext(ctx, `INSERT INTO favorites_schema_migrations(version, applied_at) VALUES (1, ?)`, fmtTime(time.Now())); err != nil {
				return err
			}
		}
		return nil
	})
}

// Rollback removes every module table and the migration record. Destructive.
func Rollback(ctx context.Context, db *sql.DB) error {
	return withImmediate(ctx, db, func(x DBTX) error {
		if _, err := x.ExecContext(ctx, downV1); err != nil {
			return err
		}
		_, err := x.ExecContext(ctx, `DROP TABLE IF EXISTS favorites_schema_migrations`)
		return err
	})
}

func currentVersion(ctx context.Context, x DBTX) (int, error) {
	var v sql.NullInt64
	if err := x.QueryRowContext(ctx, `SELECT MAX(version) FROM favorites_schema_migrations`).Scan(&v); err != nil {
		return 0, err
	}
	return int(v.Int64), nil
}

// Version reports the applied schema version (0 if not migrated).
func Version(ctx context.Context, db *sql.DB) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='favorites_schema_migrations'`).Scan(&n)
	if err != nil || n == 0 {
		return 0, err
	}
	return currentVersion(ctx, db)
}

const timeLayout = "2006-01-02T15:04:05.000000Z"

func fmtTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) { return time.Parse(timeLayout, s) }
