-- 0014_closure_challenges: scheduled network closure (GOV-025, DEC-025),
-- challenges to administrator decisions (GOV-016) and dedupe records for the
-- dispute deadline reminders (GOV-014). Additive and forward-only. The
-- networks.status check is untouched: "closing" is derived from the closure
-- row and the clock, so the freeze at the closure date needs no job to run.
CREATE TABLE network_closures (
  network_id         TEXT PRIMARY KEY REFERENCES networks(id),
  reason             TEXT NOT NULL,
  scheduled_by       TEXT NOT NULL DEFAULT '',
  scheduled_at       INTEGER NOT NULL,
  closing_at         INTEGER NOT NULL,
  notice_days        INTEGER NOT NULL,
  read_only_days     INTEGER NOT NULL,
  closed_at          INTEGER NOT NULL DEFAULT 0,
  final_ledger_seq   INTEGER NOT NULL DEFAULT 0,
  final_sha256       TEXT NOT NULL DEFAULT '',
  final_balance_total INTEGER NOT NULL DEFAULT 0,
  final_entries_sum  INTEGER NOT NULL DEFAULT 0,
  erased_at          INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE challenges (
  id          TEXT PRIMARY KEY,
  network_id  TEXT NOT NULL REFERENCES networks(id),
  member_id   TEXT NOT NULL,
  kind        TEXT NOT NULL CHECK (kind IN ('removal','credit_limit','listing_removal')),
  subject_id  TEXT NOT NULL DEFAULT '',
  reason      TEXT NOT NULL,
  involved_id TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL CHECK (status IN ('requested','decided')),
  outcome     TEXT NOT NULL DEFAULT '' CHECK (outcome IN ('','upheld','rejected')),
  decided_by  TEXT NOT NULL DEFAULT '',
  rationale   TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL,
  decided_at  INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY (network_id, member_id) REFERENCES members(network_id, id)
);
CREATE INDEX challenges_by_network ON challenges(network_id, created_at);

CREATE TABLE dispute_deadline_notices (
  network_id  TEXT NOT NULL,
  dispute_id  TEXT NOT NULL,
  kind        TEXT NOT NULL CHECK (kind IN ('response','decision')),
  notified_at INTEGER NOT NULL,
  PRIMARY KEY (network_id, dispute_id, kind),
  FOREIGN KEY (network_id, dispute_id) REFERENCES disputes(network_id, id)
);
