-- FUX-104 additive migration. Times are unix seconds (UTC). Nothing here
-- references or alters ledger, proposal or member tables; network_id/member_id
-- are opaque text keys supplied by the host adapter.

CREATE TABLE notification_preferences (
  network_id  TEXT    NOT NULL,
  member_id   TEXT    NOT NULL,
  event_type  TEXT    NOT NULL CHECK (event_type IN ('proposal_received','proposal_resolved','dispute','proposal_expiring','account_security')),
  channel     TEXT    NOT NULL CHECK (channel IN ('email','push')),
  enabled     INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
  quiet_start INTEGER CHECK (quiet_start BETWEEN 0 AND 1439),
  quiet_end   INTEGER CHECK (quiet_end   BETWEEN 0 AND 1439),
  tz          TEXT    NOT NULL DEFAULT 'UTC',
  updated_at  INTEGER NOT NULL,
  PRIMARY KEY (network_id, member_id, event_type, channel),
  CHECK ((quiet_start IS NULL) = (quiet_end IS NULL))
) WITHOUT ROWID;

-- One current e-mail address per member and network. The address is sealed
-- (AES-256-GCM, AAD = network|member|id); address_hash is a keyed HMAC.
CREATE TABLE notification_email_targets (
  id           TEXT PRIMARY KEY,
  network_id   TEXT NOT NULL,
  member_id    TEXT NOT NULL,
  address_hash TEXT NOT NULL,
  sealed       BLOB NOT NULL,
  hint         TEXT NOT NULL DEFAULT '',   -- masked form (j***@example.com) shown only to the owner
  created_at   INTEGER NOT NULL,
  verified_at  INTEGER,
  revoked_at   INTEGER,
  bounced_at   INTEGER,
  soft_bounces INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX notification_email_active ON notification_email_targets (network_id, member_id) WHERE revoked_at IS NULL;

CREATE TABLE push_subscriptions (
  id            TEXT PRIMARY KEY,
  network_id    TEXT NOT NULL,
  member_id     TEXT NOT NULL,
  endpoint_hash TEXT NOT NULL,
  sealed        BLOB NOT NULL,
  provider      TEXT NOT NULL DEFAULT '',   -- endpoint host only, for the owner's list
  session_id    TEXT,
  created_at    INTEGER NOT NULL,
  revoked_at    INTEGER,
  revoke_reason TEXT,
  UNIQUE (network_id, member_id, endpoint_hash)
);
CREATE INDEX push_subscriptions_member ON push_subscriptions (network_id, member_id) WHERE revoked_at IS NULL;
CREATE INDEX push_subscriptions_session ON push_subscriptions (network_id, member_id, session_id) WHERE revoked_at IS NULL;

CREATE TABLE notification_delivery (
  id                  TEXT PRIMARY KEY,
  network_id          TEXT NOT NULL,
  member_id           TEXT NOT NULL,
  event_id            TEXT NOT NULL,
  dedupe_key          TEXT NOT NULL UNIQUE,   -- DedupeKey(network,event,member,channel,kind): opaque, safe to log
  channel             TEXT NOT NULL CHECK (channel IN ('email','push')),
  kind                TEXT NOT NULL,
  status              TEXT NOT NULL CHECK (status IN ('ready','in_flight','sent','cancelled','expired','failed')),
  attempts            INTEGER NOT NULL DEFAULT 0,
  claims              INTEGER NOT NULL DEFAULT 0,
  next_attempt_at     INTEGER NOT NULL,
  expiry              INTEGER NOT NULL,
  provider_message_id TEXT,
  target_id           TEXT,
  lease_owner         TEXT,
  lease_until         INTEGER,
  last_error          TEXT,
  created_at          INTEGER NOT NULL,
  updated_at          INTEGER NOT NULL,
  -- Tenant-, recipient-, channel- and kind-aware identity. The planning
  -- document's UNIQUE(event_id,member_id,channel) would collapse two kinds for
  -- one event and trust event ids to be globally unique across networks.
  UNIQUE (network_id, event_id, member_id, channel, kind)
);
CREATE INDEX notification_delivery_due ON notification_delivery (status, next_attempt_at);
CREATE INDEX notification_delivery_member ON notification_delivery (network_id, member_id, status);

-- Per-target outcome of a fan-out so a retry never repeats a target that
-- already accepted the message.
CREATE TABLE notification_delivery_target (
  delivery_id TEXT NOT NULL REFERENCES notification_delivery(id) ON DELETE CASCADE,
  target_id   TEXT NOT NULL,
  status      TEXT NOT NULL CHECK (status IN ('sent','gone','failed')),
  attempts    INTEGER NOT NULL DEFAULT 0,
  updated_at  INTEGER NOT NULL,
  PRIMARY KEY (delivery_id, target_id)
) WITHOUT ROWID;

-- Append-only preference/target audit. Detail holds fixed vocabulary only:
-- never addresses, endpoints, keys or trade data.
CREATE TABLE notification_audit (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  network_id TEXT NOT NULL,
  member_id  TEXT NOT NULL,
  actor      TEXT NOT NULL,
  action     TEXT NOT NULL,
  subject    TEXT NOT NULL DEFAULT '',
  detail     TEXT NOT NULL DEFAULT '',
  at         INTEGER NOT NULL
);
CREATE INDEX notification_audit_member ON notification_audit (network_id, member_id, id);
CREATE TRIGGER notification_audit_no_update BEFORE UPDATE ON notification_audit
BEGIN SELECT RAISE(ABORT, 'notification_audit is append-only'); END;
CREATE TRIGGER notification_audit_no_delete BEFORE DELETE ON notification_audit
BEGIN SELECT RAISE(ABORT, 'notification_audit is append-only'); END;

CREATE TABLE notification_idempotency (
  network_id   TEXT NOT NULL,
  member_id    TEXT NOT NULL,
  key          TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  status       INTEGER NOT NULL,
  response     BLOB NOT NULL,
  created_at   INTEGER NOT NULL,
  PRIMARY KEY (network_id, member_id, key)
) WITHOUT ROWID;
