-- GOV-001 / FR-076 / FR-078 / AUD4-F03. Forward-only.
-- Bind dispute timelines to one immutable policy version. New disputes bind
-- the version in the same write that opens them; wall time is never used to
-- reconstruct that binding on later reads.
--
-- Legacy backfill preserves the existing timestamp-based interpretation:
-- highest version with effective_at <= opened_at, including scheduled-policy
-- withdrawal semantics. If none exists, stop for an owner mapping. Historical
-- clock ambiguity cannot recover a missing original binding; this migration
-- freezes the prior interpretation rather than claiming to reconstruct it.
-- No ledger, audit, or existing dispute outcome is changed.

ALTER TABLE disputes ADD COLUMN policy_version INTEGER CHECK (policy_version > 0);

UPDATE disputes SET policy_version = (
  SELECT p.version FROM network_policies p
  WHERE p.network_id = disputes.network_id AND p.effective_at <= disputes.opened_at
  ORDER BY p.version DESC LIMIT 1
);

CREATE TEMP TABLE dispute_policy_backfill_check (
  unmapped INTEGER NOT NULL,
  CONSTRAINT dispute_policy_backfill_needs_owner_mapping CHECK (unmapped = 0)
);
INSERT INTO dispute_policy_backfill_check(unmapped)
  SELECT COUNT(*) FROM disputes WHERE policy_version IS NULL;
DROP TABLE dispute_policy_backfill_check; -- allow-destructive: drops the temporary check table created above in this migration

CREATE TRIGGER dispute_policy_valid BEFORE INSERT ON disputes
WHEN NEW.policy_version IS NULL OR NOT EXISTS (
  SELECT 1 FROM network_policies p
  WHERE p.network_id = NEW.network_id AND p.version = NEW.policy_version
    AND p.effective_at <= NEW.opened_at
)
BEGIN SELECT RAISE(ABORT, 'a dispute needs a policy version in force at opening'); END;

CREATE TRIGGER dispute_policy_immutable BEFORE UPDATE OF policy_version, opened_at, network_id ON disputes
WHEN NEW.policy_version IS NOT OLD.policy_version
  OR NEW.opened_at IS NOT OLD.opened_at OR NEW.network_id IS NOT OLD.network_id
BEGIN SELECT RAISE(ABORT, 'dispute policy binding is immutable'); END;

-- Conflict replacement can otherwise bypass update/delete triggers.
CREATE TRIGGER dispute_no_replace BEFORE INSERT ON disputes
WHEN EXISTS (SELECT 1 FROM disputes WHERE id = NEW.id)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: disputes.id (immutable dispute policy binding)'); END;

CREATE TRIGGER policies_no_replace BEFORE INSERT ON network_policies
WHEN EXISTS (SELECT 1 FROM network_policies WHERE network_id = NEW.network_id AND version = NEW.version)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: network_policies.network_id, network_policies.version (immutable policy version)'); END;
