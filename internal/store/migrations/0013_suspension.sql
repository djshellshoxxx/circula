-- 0013_suspension: network suspension by an administrator (GOV-023) and by the
-- platform operator (GOV-024). Additive and forward-only. The networks.status
-- check is untouched: a suspension is an overlay that pauses trading and
-- posting while status stays active, so reads and exports keep working. One row
-- per kind per network; lifting a suspension deletes the row, the audit log
-- keeps the history.
CREATE TABLE network_suspensions (
  network_id   TEXT NOT NULL REFERENCES networks(id),
  kind         TEXT NOT NULL CHECK (kind IN ('admin','platform')),
  category     TEXT NOT NULL DEFAULT '',
  reason       TEXT NOT NULL,
  suspended_by TEXT NOT NULL DEFAULT '',
  suspended_at INTEGER NOT NULL,
  PRIMARY KEY (network_id, kind)
);
