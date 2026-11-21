-- GOV-025 e: version the deterministic final ledger checkpoint separately
-- from the downloadable export schema. Preserve every historical checksum
-- and reconciliation value: version 0 explicitly identifies legacy hashes,
-- whose original generated_at was not saved and cannot be reconstructed.
ALTER TABLE network_closures ADD COLUMN final_export_version INTEGER NOT NULL DEFAULT 0
  CHECK (final_export_version >= 0);
