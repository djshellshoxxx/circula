-- 0006_breakglass: SEC-013 operator break-glass trail. Forward-only.
--
-- A session is the only way an operator may touch a network's data or rights.
-- It is time boxed (hard maximum four hours, enforced here as well as in the
-- service) and recorded where the network's administrators can read it.

CREATE TABLE break_glass_sessions (
  id             TEXT PRIMARY KEY,
  network_id     TEXT NOT NULL REFERENCES networks(id),
  operator_label TEXT NOT NULL,
  reason         TEXT NOT NULL,
  opened_at      INTEGER NOT NULL,
  expires_at     INTEGER NOT NULL,
  closed_at      INTEGER,
  closed_reason  TEXT NOT NULL DEFAULT '',
  CHECK (expires_at > opened_at AND expires_at - opened_at <= 14400000),
  CHECK ((closed_at IS NULL AND closed_reason = '') OR (closed_at IS NOT NULL AND closed_reason <> ''))
);
CREATE INDEX break_glass_network ON break_glass_sessions(network_id, opened_at);
CREATE INDEX break_glass_open ON break_glass_sessions(expires_at) WHERE closed_at IS NULL;

CREATE TRIGGER break_glass_no_delete BEFORE DELETE ON break_glass_sessions
BEGIN SELECT RAISE(ABORT, 'break_glass_sessions are append-only'); END;

-- Only closing is allowed, once, and nothing else may change.
CREATE TRIGGER break_glass_close_only BEFORE UPDATE ON break_glass_sessions
WHEN NEW.id <> OLD.id OR NEW.network_id <> OLD.network_id OR NEW.operator_label <> OLD.operator_label
  OR NEW.reason <> OLD.reason OR NEW.opened_at <> OLD.opened_at OR NEW.expires_at <> OLD.expires_at
  OR OLD.closed_at IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'a break-glass session can only be closed, once'); END;

-- Temporary role grants tied to a session. A grant is in force only while its
-- session is open and unexpired; it is never revoked row by row.
CREATE TABLE break_glass_grants (
  id         TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES break_glass_sessions(id),
  network_id TEXT NOT NULL,
  member_id  TEXT NOT NULL,
  role       TEXT NOT NULL CHECK (role IN ('admin','reviewer')),
  granted_at INTEGER NOT NULL,
  FOREIGN KEY (network_id, member_id) REFERENCES members(network_id, id)
);
CREATE INDEX break_glass_grants_member ON break_glass_grants(network_id, member_id);
CREATE INDEX break_glass_grants_session ON break_glass_grants(session_id);

CREATE TRIGGER break_glass_grants_no_update BEFORE UPDATE ON break_glass_grants
BEGIN SELECT RAISE(ABORT, 'break_glass_grants are append-only'); END;
CREATE TRIGGER break_glass_grants_no_delete BEFORE DELETE ON break_glass_grants
BEGIN SELECT RAISE(ABORT, 'break_glass_grants are append-only'); END;
