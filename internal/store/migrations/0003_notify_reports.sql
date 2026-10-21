-- 0003_notify_reports: other-party dispute responses (FR-076), one re-review of
-- a dispute decision (FR-078), abuse reports (FR-024) and in-app notifications
-- (FR-038, UX-012). Additive only: no existing table is altered.

CREATE TABLE dispute_responses (
  id           TEXT PRIMARY KEY,
  network_id   TEXT NOT NULL,
  dispute_id   TEXT NOT NULL,
  responder_id TEXT NOT NULL,
  body         TEXT NOT NULL,
  submitted_at INTEGER NOT NULL,
  UNIQUE (network_id, dispute_id),                    -- one response per dispute
  FOREIGN KEY (network_id, dispute_id)   REFERENCES disputes(network_id, id),
  FOREIGN KEY (network_id, responder_id) REFERENCES members(network_id, id)
);

-- The re-review is a new record linked to the dispute; the original decision
-- on the disputes row is never edited. UNIQUE(dispute_id) makes "once only" a
-- database fact.
CREATE TABLE dispute_rereviews (
  id              TEXT PRIMARY KEY,
  network_id      TEXT NOT NULL,
  dispute_id      TEXT NOT NULL,
  requested_by    TEXT NOT NULL,
  reason          TEXT NOT NULL,
  requested_at    INTEGER NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('requested','decided')),
  reviewer_id     TEXT,
  outcome         TEXT CHECK (outcome IS NULL OR outcome IN ('confirmed','upheld','partially_upheld')),
  reversal_amount INTEGER NOT NULL DEFAULT 0 CHECK (reversal_amount >= 0),
  rationale       TEXT NOT NULL DEFAULT '',
  decided_at      INTEGER,
  UNIQUE (network_id, dispute_id),
  FOREIGN KEY (network_id, dispute_id)   REFERENCES disputes(network_id, id),
  FOREIGN KEY (network_id, requested_by) REFERENCES members(network_id, id)
);

CREATE TABLE reports (
  id            TEXT PRIMARY KEY,
  network_id    TEXT NOT NULL,
  reporter_id   TEXT NOT NULL,
  target_type   TEXT NOT NULL CHECK (target_type IN ('listing','member')),
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
-- No duplicate open report by one member about one target.
CREATE UNIQUE INDEX reports_open_dup ON reports(network_id, reporter_id, target_type, target_id)
  WHERE status = 'open';
CREATE INDEX reports_queue ON reports(network_id, status, id);
CREATE INDEX reports_rate ON reports(network_id, reporter_id, created_at);

CREATE TABLE notifications (
  id         TEXT PRIMARY KEY,
  network_id TEXT NOT NULL,
  member_id  TEXT NOT NULL,
  kind       TEXT NOT NULL,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL DEFAULT '',
  link       TEXT NOT NULL DEFAULT '',            -- path inside the network, e.g. /trades/<id>
  dedupe_key TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  read_at    INTEGER,
  FOREIGN KEY (network_id, member_id) REFERENCES members(network_id, id)
);
CREATE INDEX notif_inbox ON notifications(network_id, member_id, id);
CREATE INDEX notif_unread ON notifications(network_id, member_id) WHERE read_at IS NULL;
-- A non-empty dedupe key allows one notification per member and key.
CREATE UNIQUE INDEX notif_dedupe ON notifications(network_id, member_id, dedupe_key) WHERE dedupe_key <> '';
