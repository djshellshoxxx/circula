-- 0002_hardening: audit chain head (SA-09), purge and rate-limit indexes
-- (SA-12, SA-11). Forward-only.

-- The newest audit event per scope is mirrored here in the same unit of work
-- that appends it, so deleting the tail of audit_events is detectable: the
-- chain must end exactly at this head.
CREATE TABLE audit_heads (
  network_id TEXT PRIMARY KEY,           -- '' = platform scope
  seq        INTEGER NOT NULL,
  hash       TEXT NOT NULL
);

INSERT INTO audit_heads(network_id, seq, hash)
  SELECT e.network_id, e.seq, e.hash FROM audit_events e
  WHERE e.seq = (SELECT MAX(x.seq) FROM audit_events x WHERE x.network_id = e.network_id);

CREATE TRIGGER audit_heads_no_delete BEFORE DELETE ON audit_heads
BEGIN SELECT RAISE(ABORT, 'audit_heads are never deleted'); END;

CREATE TRIGGER audit_heads_only_advance BEFORE UPDATE ON audit_heads
WHEN NEW.network_id <> OLD.network_id OR NEW.seq <= OLD.seq
BEGIN SELECT RAISE(ABORT, 'audit head can only advance'); END;

-- Per-actor export quota lookups.
CREATE INDEX audit_actor_action ON audit_events(network_id, actor_id, action, occurred_at);

-- Retention purge scans.
CREATE INDEX sessions_expiry ON sessions(expires_at);
CREATE INDEX sessions_seen ON sessions(last_seen_at);
CREATE INDEX idem_expiry ON idempotency_keys(expires_at);
CREATE INDEX throttle_locked ON login_throttle(locked_until);
