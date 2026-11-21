-- SEC-025: permit reports about ordinary two-member transactions. Preserve
-- every existing report value and the existing queue, duplicate and rate
-- indexes. SQLite cannot widen a CHECK constraint in place. No audit or
-- financial table is changed by this migration.
CREATE TABLE reports_next (
  id            TEXT PRIMARY KEY,
  network_id    TEXT NOT NULL,
  reporter_id   TEXT NOT NULL,
  target_type   TEXT NOT NULL CHECK (target_type IN ('listing','member','transaction')),
  target_id     TEXT NOT NULL,
  category      TEXT NOT NULL,
  note          TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL CHECK (status IN ('open','handled')),
  created_at    INTEGER NOT NULL,
  handled_by    TEXT,
  handled_at    INTEGER,
  handling_note TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (network_id, reporter_id) REFERENCES members(network_id, id)
);

INSERT INTO reports_next(id,network_id,reporter_id,target_type,target_id,category,note,status,created_at,
  handled_by,handled_at,handling_note)
  SELECT id,network_id,reporter_id,target_type,target_id,category,note,status,created_at,
    handled_by,handled_at,handling_note FROM reports ORDER BY rowid;
DROP TABLE reports; -- allow-destructive: all twelve columns are copied above; transactional rebuild widens only target_type and rollback preserves the original table
ALTER TABLE reports_next RENAME TO reports;

CREATE UNIQUE INDEX reports_open_dup ON reports(network_id, reporter_id, target_type, target_id)
  WHERE status = 'open';
CREATE INDEX reports_queue ON reports(network_id, status, id);
CREATE INDEX reports_rate ON reports(network_id, reporter_id, created_at);
