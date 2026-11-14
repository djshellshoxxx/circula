-- 0005_dualcontrol: pending administrator adjustments awaiting a second
-- administrator (SEC-011). Forward-only. The policy field dual_control_threshold
-- lives in the versioned policy JSON (absent = 0 = disabled); no column needed.

CREATE TABLE pending_adjustments (
  id                 TEXT PRIMARY KEY,
  network_id         TEXT NOT NULL REFERENCES networks(id),
  proposer_member_id TEXT NOT NULL,
  target_member_id   TEXT NOT NULL,
  amount             INTEGER NOT NULL CHECK (amount > 0),
  direction          TEXT NOT NULL CHECK (direction IN ('credit','debit')),
  reason             TEXT NOT NULL CHECK (length(reason) > 0),
  status             TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','approved','rejected','expired','cancelled')),
  created_at         INTEGER NOT NULL,
  expires_at         INTEGER NOT NULL,
  decided_by         TEXT,
  decided_at         INTEGER,
  tx_id              TEXT,
  FOREIGN KEY (network_id, proposer_member_id) REFERENCES members(network_id, id),
  FOREIGN KEY (network_id, target_member_id)   REFERENCES members(network_id, id),
  FOREIGN KEY (network_id, tx_id)              REFERENCES transactions(network_id, id),
  CHECK (expires_at > created_at),
  CHECK ((status = 'pending') = (decided_at IS NULL)),
  CHECK (tx_id IS NULL OR status = 'approved'),
  CHECK (status <> 'approved' OR (tx_id IS NOT NULL AND decided_by IS NOT NULL AND decided_by <> proposer_member_id))
);

CREATE INDEX pending_adjustments_net ON pending_adjustments(network_id, status, id);
CREATE INDEX pending_adjustments_expiry ON pending_adjustments(status, expires_at);

-- The request body is immutable and a decided request is final; rows are never deleted.
CREATE TRIGGER pending_adjustments_immutable BEFORE UPDATE ON pending_adjustments
WHEN NEW.id <> OLD.id OR NEW.network_id <> OLD.network_id OR NEW.proposer_member_id <> OLD.proposer_member_id
  OR NEW.target_member_id <> OLD.target_member_id OR NEW.amount <> OLD.amount OR NEW.direction <> OLD.direction
  OR NEW.reason <> OLD.reason OR NEW.created_at <> OLD.created_at OR NEW.expires_at <> OLD.expires_at
  OR OLD.status <> 'pending'
BEGIN SELECT RAISE(ABORT, 'pending_adjustments: immutable once decided'); END;

CREATE TRIGGER pending_adjustments_no_delete BEFORE DELETE ON pending_adjustments
BEGIN SELECT RAISE(ABORT, 'pending_adjustments are never deleted'); END;
