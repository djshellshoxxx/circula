-- API-034..036 / SEC-003. Forward-only expansion for bounded owner session
-- pages and direct owner-scoped lookup. Public IDs are independent random
-- noncredential metadata; existing bearer hashes and lifetimes do not change.
-- The empty default exists only during a legacy-shape INSERT: the AFTER
-- trigger assigns its ID before that statement completes. New application
-- inserts supply the random ID directly and return it to the repository.
ALTER TABLE sessions ADD COLUMN public_id TEXT NOT NULL DEFAULT '' CHECK (
  public_id = '' OR (typeof(public_id) = 'text' AND length(public_id) = 64
    AND length(CAST(public_id AS BLOB)) = 64 AND public_id NOT GLOB '*[^0-9a-f]*')
);

UPDATE sessions SET public_id = lower(hex(randomblob(32)));

CREATE UNIQUE INDEX sessions_public_id ON sessions(public_id);
CREATE INDEX sessions_owner_page ON sessions(person_id, created_at DESC, public_id DESC);

-- The existing API-token inventory is capped at 100; seek its owner/order
-- directly as well, without sorting or scanning other owners (API-036).
CREATE INDEX api_tokens_person_order ON api_tokens(person_id, id);

CREATE TRIGGER sessions_public_id_default AFTER INSERT ON sessions
WHEN NEW.public_id = ''
BEGIN
  UPDATE sessions SET public_id = lower(hex(randomblob(32))) WHERE id = NEW.id;
END;

-- Preserve the complete owner/keyset identity through touches/revocations.
-- Only the AFTER INSERT assignment may replace the transient empty ID.
CREATE TRIGGER sessions_identity_immutable BEFORE UPDATE OF id, person_id, created_at, public_id ON sessions
WHEN NEW.id IS NOT OLD.id OR NEW.person_id IS NOT OLD.person_id
  OR NEW.created_at IS NOT OLD.created_at
  OR (OLD.public_id <> '' AND NEW.public_id IS NOT OLD.public_id)
BEGIN SELECT RAISE(ABORT, 'session identity is immutable'); END;

-- REPLACE can otherwise remove an existing identity without update triggers.
-- NULL primary keys are legal in legacy SQLite rowid tables, but would make
-- the legacy-shape INSERT assignment unable to find its row.
CREATE TRIGGER sessions_no_replace BEFORE INSERT ON sessions
WHEN NEW.id IS NULL OR EXISTS (SELECT 1 FROM sessions WHERE id = NEW.id)
  OR EXISTS (SELECT 1 FROM sessions WHERE public_id = NEW.public_id)
BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed: sessions identity'); END;
