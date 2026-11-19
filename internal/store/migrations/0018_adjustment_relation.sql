-- Optional adjustment context (UX-017). Original reversal accounting remains
-- exclusive to original_tx_id. Existing transaction and audit rows are unchanged.
ALTER TABLE transactions ADD COLUMN related_tx_id TEXT REFERENCES transactions(id)
  CHECK (related_tx_id IS NULL OR (kind = 'adjustment' AND related_tx_id <> id));
ALTER TABLE pending_adjustments ADD COLUMN related_tx_id TEXT REFERENCES transactions(id);
CREATE INDEX tx_related ON transactions(network_id, related_tx_id) WHERE related_tx_id IS NOT NULL;

CREATE TRIGGER tx_relation_valid BEFORE INSERT ON transactions
WHEN NEW.related_tx_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM transactions original JOIN accounts adjusted
    ON adjusted.id IN (NEW.payer_account_id, NEW.payee_account_id)
  WHERE original.id = NEW.related_tx_id AND original.network_id = NEW.network_id
    AND original.ledger_seq > 0 AND original.completed_at IS NOT NULL
    AND original.status IN ('completed','disputed','reversed') AND adjusted.member_id IS NOT NULL
    AND adjusted.id IN (original.payer_account_id, original.payee_account_id)
    AND NOT EXISTS (
      SELECT 1 FROM accounts exposed
      WHERE exposed.id IN (NEW.payer_account_id, NEW.payee_account_id)
        AND exposed.member_id IS NOT NULL
        AND exposed.id NOT IN (original.payer_account_id, original.payee_account_id)
    )
)
BEGIN SELECT RAISE(ABORT, 'adjustment relation requires a posted transaction involving the member'); END;

CREATE TRIGGER tx_relation_immutable BEFORE UPDATE OF related_tx_id ON transactions
WHEN NEW.related_tx_id IS NOT OLD.related_tx_id
BEGIN SELECT RAISE(ABORT, 'transaction relation is immutable'); END;

CREATE TRIGGER pending_relation_valid BEFORE INSERT ON pending_adjustments
WHEN NEW.related_tx_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM transactions original JOIN accounts adjusted
    ON adjusted.id IN (original.payer_account_id, original.payee_account_id)
  WHERE original.id = NEW.related_tx_id AND original.network_id = NEW.network_id
    AND original.ledger_seq > 0 AND original.completed_at IS NOT NULL
    AND original.status IN ('completed','disputed','reversed') AND adjusted.member_id = NEW.target_member_id
)
BEGIN SELECT RAISE(ABORT, 'adjustment relation requires a posted transaction involving the member'); END;

CREATE TRIGGER pending_relation_immutable BEFORE UPDATE OF related_tx_id ON pending_adjustments
WHEN NEW.related_tx_id IS NOT OLD.related_tx_id
BEGIN SELECT RAISE(ABORT, 'pending adjustment relation is immutable'); END;

-- REPLACE can bypass DELETE triggers when recursive_triggers is disabled.
-- Reject existing IDs before insertion so relation immutability also holds
-- under conflict replacement. Keep the ordinary duplicate-key error category.
CREATE TRIGGER tx_no_replace BEFORE INSERT ON transactions
WHEN EXISTS (SELECT 1 FROM transactions WHERE id = NEW.id)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: transactions.id (immutable transaction)'); END;

CREATE TRIGGER pending_no_replace BEFORE INSERT ON pending_adjustments
WHEN EXISTS (SELECT 1 FROM pending_adjustments WHERE id = NEW.id)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: pending_adjustments.id (immutable request)'); END;
