-- 0007_erasure: SEC-018 erasure / anonymisation record. Forward-only.
--
-- The personal data itself is changed by the service (display name, listing
-- text, memos, dispute statements, person login and credentials). This table
-- only records THAT it happened, so the action is idempotent and visible,
-- and holds no personal data.

CREATE TABLE member_erasures (
  member_id      TEXT PRIMARY KEY,
  network_id     TEXT NOT NULL,
  person_id      TEXT NOT NULL,
  erased_at      INTEGER NOT NULL,
  person_erased  INTEGER NOT NULL CHECK (person_erased IN (0,1)),
  FOREIGN KEY (network_id, member_id) REFERENCES members(network_id, id)
);

CREATE TRIGGER member_erasures_no_update BEFORE UPDATE ON member_erasures
BEGIN SELECT RAISE(ABORT, 'member_erasures are append-only'); END;
CREATE TRIGGER member_erasures_no_delete BEFORE DELETE ON member_erasures
BEGIN SELECT RAISE(ABORT, 'member_erasures are append-only'); END;
