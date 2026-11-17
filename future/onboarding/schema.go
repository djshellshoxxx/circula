package onboarding

// SchemaSQL is the additive, forward-only SQLite schema owned by this module.
// It creates only tables prefixed member_onboarding_ / network_onboarding_ and
// never alters or reads any other table. It is NOT registered with any
// application migration runner; a future integration copies it into the core's
// migration system after review.
//
// Indexes: the primary keys serve all lookups; the updated_at index serves the
// retention purge. Retention: rows live until the member is removed
// (DeleteMember) or an operator runs PurgeStale. Anonymous counters hold no
// member data and are deleted when counting is switched off. Backup: these
// tables are ordinary rows in the host database file, so they are covered by
// that file's backup/restore; none holds a secret.
const SchemaSQL = `
CREATE TABLE IF NOT EXISTS onboarding_schema_migrations (
  version    INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS member_onboarding_progress (
  network_id      TEXT    NOT NULL CHECK (length(network_id) BETWEEN 1 AND 64),
  member_id       TEXT    NOT NULL CHECK (length(member_id) BETWEEN 1 AND 64),
  step_version    INTEGER NOT NULL CHECK (step_version >= 1),
  completed_mask  INTEGER NOT NULL DEFAULT 0 CHECK (completed_mask BETWEEN 0 AND 31),
  skipped_mask    INTEGER NOT NULL DEFAULT 0 CHECK (skipped_mask BETWEEN 0 AND 31 AND (completed_mask & skipped_mask) = 0),
  dismissed       INTEGER NOT NULL DEFAULT 0 CHECK (dismissed IN (0, 1)),
  share_aggregate INTEGER NOT NULL DEFAULT 0 CHECK (share_aggregate IN (0, 1)),
  revision        INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
  updated_at      TEXT    NOT NULL,
  PRIMARY KEY (network_id, member_id, step_version)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS member_onboarding_progress_updated_at ON member_onboarding_progress (updated_at);
CREATE TABLE IF NOT EXISTS network_onboarding_copy (
  network_id     TEXT    NOT NULL CHECK (length(network_id) BETWEEN 1 AND 64),
  step_version   INTEGER NOT NULL CHECK (step_version >= 1),
  intro_text     TEXT    NOT NULL DEFAULT '' CHECK (length(intro_text) <= 4000),
  governance_url TEXT    NOT NULL DEFAULT '' CHECK (length(governance_url) <= 512),
  terms_url      TEXT    NOT NULL DEFAULT '' CHECK (length(terms_url) <= 512),
  revision       INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
  updated_by     TEXT    NOT NULL,
  updated_at     TEXT    NOT NULL,
  PRIMARY KEY (network_id, step_version)
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS network_onboarding_metrics (
  network_id TEXT    PRIMARY KEY CHECK (length(network_id) BETWEEN 1 AND 64),
  enabled    INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
  updated_by TEXT    NOT NULL,
  updated_at TEXT    NOT NULL
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS network_onboarding_counts (
  network_id   TEXT    NOT NULL CHECK (length(network_id) BETWEEN 1 AND 64),
  step_version INTEGER NOT NULL CHECK (step_version >= 1),
  step         INTEGER NOT NULL CHECK (step BETWEEN 0 AND 4),
  completions  INTEGER NOT NULL DEFAULT 0 CHECK (completions >= 0),
  PRIMARY KEY (network_id, step_version, step)
) WITHOUT ROWID;
`

// RollbackSQL drops every table SchemaSQL created. It deletes feature data
// only; ledger and every other table are untouched. Disabling the feature does
// not require running it.
const RollbackSQL = `
DROP TABLE IF EXISTS network_onboarding_counts;
DROP TABLE IF EXISTS network_onboarding_metrics;
DROP TABLE IF EXISTS network_onboarding_copy;
DROP INDEX IF EXISTS member_onboarding_progress_updated_at;
DROP TABLE IF EXISTS member_onboarding_progress;
DROP TABLE IF EXISTS onboarding_schema_migrations;
`
