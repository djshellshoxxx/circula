-- FUX-102 module-owned additive schema (forward-only; not a root migration).
-- Times are INTEGER unix milliseconds (UTC). Hashes are SHA-256 BLOBs; raw codes are never stored.

CREATE TABLE qr_member_alias (
  network_id  TEXT    NOT NULL,
  member_id   TEXT    NOT NULL,
  generation  INTEGER NOT NULL CHECK (generation >= 1),
  alias_hash  BLOB    NOT NULL CHECK (length(alias_hash) = 32),
  revoked_at  INTEGER,
  created_at  INTEGER NOT NULL,
  PRIMARY KEY (network_id, member_id, generation),
  UNIQUE (network_id, alias_hash)
) STRICT;
-- at most one live alias per member; revoked rows are kept so old QR codes stay revoked
CREATE UNIQUE INDEX qr_member_alias_live ON qr_member_alias (network_id, member_id) WHERE revoked_at IS NULL;

CREATE TABLE qr_trade_request (
  id           TEXT    PRIMARY KEY,
  network_id   TEXT    NOT NULL,
  recipient_id TEXT    NOT NULL,
  token_hash   BLOB    NOT NULL UNIQUE CHECK (length(token_hash) = 32),
  amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
  note         TEXT    NOT NULL DEFAULT '',
  created_at   INTEGER NOT NULL,
  expires_at   INTEGER NOT NULL CHECK (expires_at > created_at),
  revoked_at   INTEGER,
  claimed_tx_id TEXT   UNIQUE,
  claimed_by   TEXT,
  claimed_at   INTEGER,
  claim_fingerprint BLOB,
  CHECK ((claimed_tx_id IS NULL) = (claimed_by IS NULL)),
  CHECK ((claimed_tx_id IS NULL) = (claimed_at IS NULL)),
  CHECK ((claimed_tx_id IS NULL) = (claim_fingerprint IS NULL)),
  CHECK (claimed_tx_id IS NULL OR revoked_at IS NULL),
  CHECK (claimed_by IS NULL OR claimed_by <> recipient_id)
) STRICT;
CREATE INDEX qr_trade_request_owner ON qr_trade_request (network_id, recipient_id, created_at DESC, id);

-- Stable idempotency records: one row per (network, member, scope, key); the
-- fingerprint binds the key to the exact intent.
CREATE TABLE qr_idempotency (
  network_id  TEXT    NOT NULL,
  member_id   TEXT    NOT NULL,
  scope       TEXT    NOT NULL CHECK (scope IN ('alias.rotate','request.create','claim.request','claim.alias')),
  idem_key    TEXT    NOT NULL,
  fingerprint BLOB    NOT NULL CHECK (length(fingerprint) = 32),
  result      TEXT    NOT NULL,
  created_at  INTEGER NOT NULL,
  PRIMARY KEY (network_id, member_id, scope, idem_key)
) STRICT;
CREATE INDEX qr_idempotency_age ON qr_idempotency (created_at);
