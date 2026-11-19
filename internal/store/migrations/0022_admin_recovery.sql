-- GOV-021: recovery when no administrator is available. Forward-only and
-- self-contained: it creates one table and touches no existing row.
--
-- A recorded, active member of a network with no active administrator asks to
-- be appointed. Two different platform operators must approve; the second
-- approval applies the request as an ordinary role_assignments row. A request
-- is never deleted, its subject never changes, each step happens once, and the
-- ledger (entries, accounts, transactions) is not referenced at all.

CREATE TABLE admin_recovery_requests (
  id                  TEXT PRIMARY KEY,
  network_id          TEXT NOT NULL REFERENCES networks(id),
  requester_member_id TEXT NOT NULL,
  status              TEXT NOT NULL DEFAULT 'requested'
                      CHECK (status IN ('requested','approved','applied','refused','expired','superseded')),
  requested_at        INTEGER NOT NULL,
  expires_at          INTEGER NOT NULL,
  first_operator      TEXT NOT NULL DEFAULT '',
  first_reason        TEXT NOT NULL DEFAULT '',
  first_approved_at   INTEGER,
  closed_by           TEXT NOT NULL DEFAULT '',
  closed_reason       TEXT NOT NULL DEFAULT '',
  closed_at           INTEGER,
  role_assignment_id  TEXT REFERENCES role_assignments(id),
  FOREIGN KEY (network_id, requester_member_id) REFERENCES members(network_id, id),
  -- At most seven days open (GOV-021, provisional; the service uses the same bound).
  CHECK (expires_at > requested_at AND expires_at - requested_at <= 604800000),
  CHECK (CASE status
    WHEN 'requested' THEN first_operator = '' AND first_reason = '' AND first_approved_at IS NULL
      AND closed_by = '' AND closed_reason = '' AND closed_at IS NULL AND role_assignment_id IS NULL
    WHEN 'approved' THEN first_operator <> '' AND first_reason <> '' AND first_approved_at IS NOT NULL
      AND closed_by = '' AND closed_reason = '' AND closed_at IS NULL AND role_assignment_id IS NULL
    -- Applied needs both approvals, by two different operator labels, and the appointment.
    WHEN 'applied' THEN first_operator <> '' AND first_reason <> '' AND first_approved_at IS NOT NULL
      AND closed_by <> '' AND closed_reason <> '' AND closed_at IS NOT NULL AND role_assignment_id IS NOT NULL
      AND lower(trim(closed_by)) <> lower(trim(first_operator))
    WHEN 'refused' THEN closed_by <> '' AND closed_reason <> '' AND closed_at IS NOT NULL AND role_assignment_id IS NULL
    ELSE closed_by = '' AND closed_reason <> '' AND closed_at IS NOT NULL AND role_assignment_id IS NULL
  END)
);
CREATE INDEX admin_recovery_network ON admin_recovery_requests(network_id, id);
CREATE INDEX admin_recovery_expiry ON admin_recovery_requests(expires_at) WHERE status IN ('requested','approved');
-- One open request per member and network; one request per appointment.
CREATE UNIQUE INDEX admin_recovery_open_member ON admin_recovery_requests(network_id, requester_member_id)
  WHERE status IN ('requested','approved');
CREATE UNIQUE INDEX admin_recovery_role ON admin_recovery_requests(role_assignment_id) WHERE role_assignment_id IS NOT NULL;

CREATE TRIGGER admin_recovery_no_delete BEFORE DELETE ON admin_recovery_requests
BEGIN SELECT RAISE(ABORT, 'admin_recovery_requests are append-only'); END;

-- REPLACE can bypass DELETE triggers; refuse an existing id before insertion.
CREATE TRIGGER admin_recovery_no_replace BEFORE INSERT ON admin_recovery_requests
WHEN EXISTS (SELECT 1 FROM admin_recovery_requests WHERE id = NEW.id)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: admin_recovery_requests.id (immutable request)'); END;

CREATE TRIGGER admin_recovery_insert_requested BEFORE INSERT ON admin_recovery_requests
WHEN NEW.status <> 'requested'
BEGIN SELECT RAISE(ABORT, 'an administrator recovery request starts as requested'); END;

-- Subject and times never change; requested -> approved -> applied, or either
-- open state -> refused/expired/superseded; final states are final; the first
-- approval is written once, on the step that records it.
CREATE TRIGGER admin_recovery_transition BEFORE UPDATE ON admin_recovery_requests
WHEN NEW.id <> OLD.id OR NEW.network_id <> OLD.network_id OR NEW.requester_member_id <> OLD.requester_member_id
  OR NEW.requested_at <> OLD.requested_at OR NEW.expires_at <> OLD.expires_at
  OR NOT ((OLD.status = 'requested' AND NEW.status IN ('approved','refused','expired','superseded'))
       OR (OLD.status = 'approved' AND NEW.status IN ('applied','refused','expired','superseded')))
  OR (NOT (OLD.status = 'requested' AND NEW.status = 'approved')
      AND (NEW.first_operator <> OLD.first_operator OR NEW.first_reason <> OLD.first_reason
           OR NEW.first_approved_at IS NOT OLD.first_approved_at))
BEGIN SELECT RAISE(ABORT, 'an administrator recovery request changes state once per step and is otherwise immutable'); END;

-- The appointment is the requester's own live admin role in the same network.
CREATE TRIGGER admin_recovery_applied_role BEFORE UPDATE ON admin_recovery_requests
WHEN NEW.status = 'applied' AND NOT EXISTS (
  SELECT 1 FROM role_assignments r WHERE r.id = NEW.role_assignment_id AND r.network_id = NEW.network_id
    AND r.member_id = NEW.requester_member_id AND r.role = 'admin' AND r.revoked_at IS NULL)
BEGIN SELECT RAISE(ABORT, 'an applied recovery must name the requester''s admin role assignment'); END;
