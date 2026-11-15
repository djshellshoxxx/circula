-- 0004_recovery: account recovery without email (SEC-008). Forward-only.

-- Set when someone else (a network administrator or the operator) reset the
-- password; cleared by ChangePassword and by self-service recovery.
ALTER TABLE persons ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0
  CHECK (must_change_password IN (0,1));

-- Single-use recovery codes. Only the salted SHA-256 is stored. Regenerating
-- deletes the previous batch in the same unit of work. Separate from any MFA
-- recovery table.
CREATE TABLE account_recovery_codes (
  id         TEXT PRIMARY KEY,
  person_id  TEXT NOT NULL REFERENCES persons(id),
  code_hash  TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  used_at    INTEGER,
  UNIQUE (person_id, code_hash)
);
CREATE INDEX account_recovery_person ON account_recovery_codes(person_id);
