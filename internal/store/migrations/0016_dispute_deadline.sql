-- 0016_dispute_deadline: DOM-017 / DAT-051. Forward-only.
--
-- The dispute deadline of a trade is fixed when the trade completes, from the
-- dispute-window policy in force at that instant. A later policy change must
-- neither shorten nor extend it. Earlier builds computed the deadline when a
-- dispute was opened, from the policy in force then.
--
-- Legacy backfill: for each completed trade, the policy in force at completion
-- is the highest version whose effective_at is at or before completed_at. This
-- is the same rule GetNetwork applies at a given time; a cancelled scheduled
-- change is superseded by a later version with an earlier effective_at, so it
-- never wins. A version stored without dispute_window_days was read with the
-- built-in default of 30 days (domain.DefaultPolicy), so 30 is used for it.
-- If no version was in force at completion, the deadline cannot be derived and
-- the migration stops (whole migration rolls back) for an owner mapping
-- instead of silently using the current policy.

ALTER TABLE transactions ADD COLUMN dispute_deadline_at INTEGER;
-- The network policy version the deadline was derived from (DAT-051).
ALTER TABLE transactions ADD COLUMN dispute_policy_version INTEGER;

UPDATE transactions
SET dispute_policy_version = (
    SELECT p.version FROM network_policies p
    WHERE p.network_id = transactions.network_id AND p.effective_at <= transactions.completed_at
    ORDER BY p.version DESC LIMIT 1)
WHERE kind = 'trade' AND completed_at IS NOT NULL;

UPDATE transactions
SET dispute_deadline_at = completed_at + 86400000 * (
    SELECT COALESCE(json_extract(p.policy_json, '$.dispute_window_days'), 30)
    FROM network_policies p
    WHERE p.network_id = transactions.network_id AND p.version = transactions.dispute_policy_version)
WHERE kind = 'trade' AND completed_at IS NOT NULL AND dispute_policy_version IS NOT NULL;

CREATE TEMP TABLE dispute_deadline_backfill_check (
  unmapped INTEGER NOT NULL,
  CONSTRAINT dispute_deadline_backfill_needs_owner_mapping CHECK (unmapped = 0)
);
INSERT INTO dispute_deadline_backfill_check(unmapped)
  SELECT COUNT(*) FROM transactions
  WHERE kind = 'trade' AND completed_at IS NOT NULL AND dispute_deadline_at IS NULL;
DROP TABLE dispute_deadline_backfill_check; -- allow-destructive: drops the temporary check table created above in this migration

-- Once set, the deadline and its policy version never change.
CREATE TRIGGER tx_dispute_deadline_immutable BEFORE UPDATE OF dispute_deadline_at, dispute_policy_version ON transactions
WHEN (OLD.dispute_deadline_at IS NOT NULL AND NEW.dispute_deadline_at IS NOT OLD.dispute_deadline_at)
  OR (OLD.dispute_policy_version IS NOT NULL AND NEW.dispute_policy_version IS NOT OLD.dispute_policy_version)
BEGIN SELECT RAISE(ABORT, 'dispute deadline is immutable'); END;

-- A trade cannot complete without its deadline; other kinds are never disputable.
CREATE TRIGGER tx_trade_completion_needs_deadline BEFORE UPDATE OF status ON transactions
WHEN NEW.kind = 'trade' AND OLD.status = 'proposed' AND NEW.status = 'completed'
 AND (NEW.dispute_deadline_at IS NULL OR NEW.dispute_policy_version IS NULL)
BEGIN SELECT RAISE(ABORT, 'a completed trade needs a dispute deadline'); END;
