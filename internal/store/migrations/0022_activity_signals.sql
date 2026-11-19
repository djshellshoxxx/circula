-- 0022 / SEC-026: basic velocity and reciprocal-pattern signals for a network's
-- administrators and dispute reviewers. Additive only: no existing table is
-- altered and no existing row changes. A signal never blocks a trade.
--
-- Rows are append-only records of who (member ids), what (kind), when (fixed
-- period and raise time) and how many (observed count against the threshold
-- in force). They hold no personal free text, so member erasure keeps them
-- unchanged, like audit_subjects and challenges (SEC-018).
CREATE TABLE activity_signals (
  id              TEXT PRIMARY KEY,
  network_id      TEXT NOT NULL REFERENCES networks(id),
  kind            TEXT NOT NULL CHECK (kind IN ('velocity','reciprocal')),
  member_id       TEXT NOT NULL,
  other_member_id TEXT,
  period_start    INTEGER NOT NULL,
  window_ms       INTEGER NOT NULL CHECK (window_ms > 0),
  observed        INTEGER NOT NULL,
  threshold       INTEGER NOT NULL CHECK (threshold > 0),
  raised_at       INTEGER NOT NULL,
  CHECK (observed > threshold),
  CHECK ((kind = 'velocity') = (other_member_id IS NULL)),
  CHECK (other_member_id IS NULL OR member_id < other_member_id),
  CHECK (period_start <= raised_at AND raised_at < period_start + window_ms),
  FOREIGN KEY (network_id, member_id)       REFERENCES members(network_id, id),
  FOREIGN KEY (network_id, other_member_id) REFERENCES members(network_id, id)
);

-- At most one signal per network, kind, member (or ordered pair) and period.
-- The index makes this a database fact, so two concurrent writers cannot
-- record two signals for one period; the writer inserts with ON CONFLICT DO
-- NOTHING inside its own unit of work.
CREATE UNIQUE INDEX activity_signals_once ON activity_signals(
  network_id, kind, member_id, COALESCE(other_member_id, ''), period_start);
CREATE INDEX activity_signals_queue ON activity_signals(network_id, id);

-- Velocity and reciprocal counts read the network's trades by proposer and
-- by account pair within a time window.
CREATE INDEX tx_velocity ON transactions(network_id, created_by, proposed_at) WHERE kind = 'trade';
CREATE INDEX tx_pair_completed ON transactions(payer_account_id, payee_account_id, completed_at)
  WHERE kind = 'trade' AND completed_at IS NOT NULL;

-- Append-only guards. REPLACE performs an implicit delete that does not fire
-- delete triggers unless recursive_triggers is enabled, so an existing id is
-- refused and an existing (network, kind, members, period) turns any insert,
-- REPLACE included, into a no-op that leaves the recorded signal untouched.
CREATE TRIGGER activity_signals_no_replace BEFORE INSERT ON activity_signals
WHEN EXISTS (SELECT 1 FROM activity_signals WHERE id = NEW.id)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: activity_signals.id (append-only signal)'); END;
CREATE TRIGGER activity_signals_once_only BEFORE INSERT ON activity_signals
WHEN EXISTS (SELECT 1 FROM activity_signals s WHERE s.network_id = NEW.network_id AND s.kind = NEW.kind
  AND s.member_id = NEW.member_id AND COALESCE(s.other_member_id, '') = COALESCE(NEW.other_member_id, '')
  AND s.period_start = NEW.period_start)
BEGIN SELECT RAISE(IGNORE); END;
CREATE TRIGGER activity_signals_no_update BEFORE UPDATE ON activity_signals
BEGIN SELECT RAISE(ABORT, 'activity signals are append-only'); END;
CREATE TRIGGER activity_signals_no_delete BEFORE DELETE ON activity_signals
BEGIN SELECT RAISE(ABORT, 'activity signals are append-only'); END;
