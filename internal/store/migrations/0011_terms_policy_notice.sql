-- 0011_terms_policy_notice: terms versions and acceptances (GOV-009) and the
-- notice-period columns of policy versions (GOV-003). Additive and forward-only.

-- A policy version may take effect later than it was announced (notice period).
-- announced_at = 0 on rows written before this migration (effective at once).
ALTER TABLE network_policies ADD COLUMN announced_at  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE network_policies ADD COLUMN notice_days   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE network_policies ADD COLUMN notice_reason TEXT NOT NULL DEFAULT '';

-- Append-only onboarding terms; the highest version is the current one.
CREATE TABLE network_terms (
  network_id   TEXT NOT NULL REFERENCES networks(id),
  version      INTEGER NOT NULL,
  body         TEXT NOT NULL,
  published_by TEXT NOT NULL,
  published_at INTEGER NOT NULL,
  PRIMARY KEY (network_id, version)
);
CREATE TRIGGER network_terms_no_update BEFORE UPDATE ON network_terms
BEGIN SELECT RAISE(ABORT, 'network_terms are append-only'); END;
CREATE TRIGGER network_terms_no_delete BEFORE DELETE ON network_terms
BEGIN SELECT RAISE(ABORT, 'network_terms are append-only'); END;

-- Which terms version a member accepted, and when. Append-only.
CREATE TABLE terms_acceptances (
  network_id  TEXT NOT NULL REFERENCES networks(id),
  member_id   TEXT NOT NULL REFERENCES members(id),
  version     INTEGER NOT NULL,
  accepted_at INTEGER NOT NULL,
  PRIMARY KEY (network_id, member_id, version)
);
CREATE TRIGGER terms_acceptances_no_update BEFORE UPDATE ON terms_acceptances
BEGIN SELECT RAISE(ABORT, 'terms_acceptances are append-only'); END;
CREATE TRIGGER terms_acceptances_no_delete BEFORE DELETE ON terms_acceptances
BEGIN SELECT RAISE(ABORT, 'terms_acceptances are append-only'); END;
