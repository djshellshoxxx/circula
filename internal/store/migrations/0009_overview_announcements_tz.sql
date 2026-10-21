-- 0009_overview_announcements_tz: announcements (API-064), the per-person
-- display time zone, and indexes behind the administrator overview counts
-- (FR-067). Additive and forward-only: no existing table is altered.

-- Display time zone per person (IANA name, validated by the service). Empty
-- means UTC. Kept apart from persons so identity rows stay untouched. No
-- foreign key: an erased person's row can go without ordering constraints.
CREATE TABLE person_prefs (
  person_id  TEXT PRIMARY KEY,
  time_zone  TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

-- Append-only announcements. A delete only stamps deleted_at/deleted_by.
CREATE TABLE announcements (
  id           TEXT PRIMARY KEY,
  network_id   TEXT NOT NULL REFERENCES networks(id),
  title        TEXT NOT NULL,
  body         TEXT NOT NULL,
  created_by   TEXT NOT NULL,
  published_at INTEGER NOT NULL,
  effective_at INTEGER NOT NULL,
  deleted_at   INTEGER,
  deleted_by   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX announcements_net ON announcements(network_id, effective_at, id);

CREATE TRIGGER announcements_no_delete BEFORE DELETE ON announcements
BEGIN SELECT RAISE(ABORT, 'announcements are append-only'); END;

CREATE TRIGGER announcements_retract_only BEFORE UPDATE ON announcements
WHEN OLD.deleted_at IS NOT NULL
  OR NEW.id <> OLD.id OR NEW.network_id <> OLD.network_id OR NEW.title <> OLD.title
  OR NEW.body <> OLD.body OR NEW.created_by <> OLD.created_by
  OR NEW.published_at <> OLD.published_at OR NEW.effective_at <> OLD.effective_at
BEGIN SELECT RAISE(ABORT, 'announcements can only be retracted'); END;

-- Overview counts: open proposals by age, disputes by status, expiring invitations, latest check.
CREATE INDEX tx_status_proposed ON transactions(network_id, status, proposed_at);
CREATE INDEX disputes_net_status ON disputes(network_id, status);
CREATE INDEX invitations_expiry ON invitations(network_id, expires_at);
CREATE INDEX consistency_runs_net ON consistency_runs(network_id, finished_at);
