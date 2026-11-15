-- 0003_mfa: TOTP second factor (SEC-007), recovery codes, short-lived MFA
-- login challenges and per-person security notices (SEC-012). Forward-only.

-- One row per person who has begun enrolment. enabled=0 is a pending
-- (unconfirmed) enrolment. secret is the TOTP seed, sealed with AES-GCM when
-- the service has a SecretKey ("v1:" prefix) else base32 ("raw:" prefix); it is
-- never returned by any read path after confirmation. last_step is the newest
-- accepted RFC 6238 time step (replay guard).
CREATE TABLE person_mfa (
  person_id  TEXT PRIMARY KEY REFERENCES persons(id),
  secret     TEXT NOT NULL,
  enabled    INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
  last_step  INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  enabled_at INTEGER
);

-- Single-use recovery codes, stored as salted SHA-256 only.
CREATE TABLE recovery_codes (
  person_id TEXT NOT NULL REFERENCES persons(id),
  code_hash TEXT NOT NULL,
  used_at   INTEGER,
  PRIMARY KEY (person_id, code_hash)
);

-- Pending second step of a login: issued after the password verified, single
-- use, short lived. Only the SHA-256 of the bearer value is stored.
CREATE TABLE mfa_challenges (
  id_hash    TEXT PRIMARY KEY,
  person_id  TEXT NOT NULL REFERENCES persons(id),
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  used_at    INTEGER
);
CREATE INDEX mfa_challenges_person ON mfa_challenges(person_id);

CREATE TABLE security_notices (
  id         TEXT PRIMARY KEY,
  person_id  TEXT NOT NULL REFERENCES persons(id),
  kind       TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  seen_at    INTEGER,
  detail     TEXT NOT NULL DEFAULT ''
);
CREATE INDEX security_notices_person ON security_notices(person_id, created_at);
