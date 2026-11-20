-- OPS-016 / LED-5: preserve the actual inputs of each new debit's credit
-- decision. Existing debit identities are explicitly unverified: historical
-- override expiry was not retained and must never be inferred or backfilled.
CREATE TABLE posting_credit_checks (
  entry_id             TEXT PRIMARY KEY REFERENCES entries(id),
  network_id           TEXT NOT NULL REFERENCES networks(id),
  legacy_unverified    INTEGER NOT NULL CHECK (legacy_unverified IN (0,1)),
  checked_at           INTEGER,
  policy_version       INTEGER,
  balance_before       INTEGER,
  is_system            INTEGER CHECK (is_system IS NULL OR is_system IN (0,1)),
  limit_override       INTEGER CHECK (limit_override IS NULL OR limit_override >= 0),
  limit_override_until INTEGER,
  effective_limit      INTEGER CHECK (effective_limit IS NULL OR effective_limit >= 0),
  FOREIGN KEY (network_id, policy_version) REFERENCES network_policies(network_id, version),
  CHECK (
    (legacy_unverified = 1 AND checked_at IS NULL AND policy_version IS NULL
      AND balance_before IS NULL AND is_system IS NULL AND limit_override IS NULL
      AND limit_override_until IS NULL AND effective_limit IS NULL)
    OR
    (legacy_unverified = 0 AND checked_at IS NOT NULL AND policy_version IS NOT NULL
      AND balance_before IS NOT NULL AND is_system IS NOT NULL AND effective_limit IS NOT NULL
      AND (limit_override_until IS NULL OR limit_override IS NOT NULL))
  )
);
CREATE INDEX posting_credit_checks_network ON posting_credit_checks(network_id, entry_id);

-- This exact immutable set is the adoption boundary. Neither posting clocks
-- nor ledger sequence values decide whether a later entry is legacy.
INSERT INTO posting_credit_checks(entry_id, network_id, legacy_unverified)
  SELECT id, network_id, 1 FROM entries WHERE amount < 0;

-- Installed after the one-time legacy marking. No runtime writer can add
-- another legacy identity or replace any original marker/witness.
CREATE TRIGGER posting_credit_checks_new_witness BEFORE INSERT ON posting_credit_checks
WHEN NEW.legacy_unverified <> 0
  OR EXISTS (SELECT 1 FROM posting_credit_checks WHERE entry_id = NEW.entry_id)
BEGIN SELECT RAISE(ABORT, 'posting credit checks are immutable; new legacy markers are forbidden'); END;

CREATE TRIGGER posting_credit_checks_entry BEFORE INSERT ON posting_credit_checks
WHEN NOT EXISTS (
  SELECT 1 FROM entries e JOIN accounts a ON a.id = e.account_id
  WHERE e.id = NEW.entry_id AND e.network_id = NEW.network_id AND e.amount < 0
    AND e.posted_at = NEW.checked_at AND a.is_system = NEW.is_system
)
BEGIN SELECT RAISE(ABORT, 'posting credit check must describe its negative entry'); END;

CREATE TRIGGER posting_credit_checks_no_update BEFORE UPDATE ON posting_credit_checks
BEGIN SELECT RAISE(ABORT, 'posting credit checks are append-only'); END;
CREATE TRIGGER posting_credit_checks_no_delete BEFORE DELETE ON posting_credit_checks
BEGIN SELECT RAISE(ABORT, 'posting credit checks are append-only'); END;

-- Additive NULL means an older saved run did not record coverage. No old run,
-- ledger row or audit event is rewritten from today's knowledge.
ALTER TABLE consistency_runs ADD COLUMN historical_limit_coverage TEXT
  CHECK (historical_limit_coverage IS NULL OR json_valid(historical_limit_coverage));
