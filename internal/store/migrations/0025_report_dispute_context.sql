-- SEC-025: a new member/listing report may name the exact dispute that it
-- concerns. Existing reports gain no inferred relation. SQLite requires a
-- rebuild to add this composite foreign key; SQL NULL preserves context-free
-- reports, while a non-null reference must belong to the same network.
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
  dispute_id    TEXT CHECK (dispute_id IS NULL OR target_type IN ('listing','member')),
  FOREIGN KEY (network_id, reporter_id) REFERENCES members(network_id, id),
  FOREIGN KEY (network_id, dispute_id) REFERENCES disputes(network_id, id)
);

INSERT INTO reports_next(id,network_id,reporter_id,target_type,target_id,category,note,status,created_at,
  handled_by,handled_at,handling_note,dispute_id)
  SELECT id,network_id,reporter_id,target_type,target_id,category,note,status,created_at,
    handled_by,handled_at,handling_note,NULL FROM reports ORDER BY rowid;
DROP TABLE reports; -- allow-destructive: every original column is copied above; transactional rebuild adds only a nullable case reference and rollback preserves the original table
ALTER TABLE reports_next RENAME TO reports;

CREATE UNIQUE INDEX reports_open_dup ON reports(network_id, reporter_id, target_type, target_id)
  WHERE status = 'open';
CREATE INDEX reports_queue ON reports(network_id, status, id);
CREATE INDEX reports_rate ON reports(network_id, reporter_id, created_at);
CREATE INDEX reports_dispute ON reports(network_id, dispute_id, id) WHERE dispute_id IS NOT NULL;

-- A report's chosen context is part of its original statement. Handling must
-- never attach, remove or redirect it, including from/to the legacy NULL state.
CREATE TRIGGER reports_dispute_immutable BEFORE UPDATE OF dispute_id ON reports
WHEN NEW.dispute_id IS NOT OLD.dispute_id
BEGIN
  SELECT RAISE(ABORT, 'report dispute context is immutable');
END;
