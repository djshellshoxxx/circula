-- FUX-103 discovery: additive, forward-only, module-local migration.
-- NOT placed in Circula's core migration directory; applied only by this
-- module's Migrate in standalone development. Core tables are never altered.

CREATE TABLE discovery_revision (
  id  INTEGER PRIMARY KEY CHECK (id = 1),
  seq INTEGER NOT NULL
);
INSERT INTO discovery_revision (id, seq) VALUES (1, 0);

-- Adapter-synced read model of core listings. In an integration this becomes
-- a view/join over core `listings` + member state; here it is the fixture.
CREATE TABLE discovery_listing_projection (
  id           TEXT NOT NULL,
  network_id   TEXT NOT NULL,
  owner_id     TEXT NOT NULL,
  kind         TEXT NOT NULL CHECK (kind IN ('offer','request')),
  title        TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  title_fold   TEXT NOT NULL,
  desc_fold    TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN ('active','paused','archived','suspended')),
  owner_active INTEGER NOT NULL DEFAULT 1 CHECK (owner_active IN (0,1)),
  created_at   INTEGER NOT NULL,
  updated_at   INTEGER NOT NULL,
  archived_at  INTEGER,
  seq          INTEGER NOT NULL,
  PRIMARY KEY (network_id, id)
);
CREATE INDEX discovery_proj_feed ON discovery_listing_projection (network_id, status, updated_at DESC, id);
CREATE INDEX discovery_proj_owner ON discovery_listing_projection (network_id, owner_id);

CREATE TABLE taxonomy_category (
  network_id TEXT NOT NULL,
  id         TEXT NOT NULL,
  name       TEXT NOT NULL,
  name_fold  TEXT NOT NULL,
  kind       TEXT NOT NULL CHECK (kind IN ('goods','service','any')),
  enabled    INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
  created_at INTEGER NOT NULL,
  PRIMARY KEY (network_id, id),
  UNIQUE (network_id, name_fold)
);

CREATE TABLE coarse_area (
  network_id TEXT NOT NULL,
  id         TEXT NOT NULL,
  name       TEXT NOT NULL,
  name_fold  TEXT NOT NULL,
  enabled    INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
  lat10      INTEGER,
  lon10      INTEGER,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (network_id, id),
  UNIQUE (network_id, name_fold),
  CHECK ((lat10 IS NULL) = (lon10 IS NULL))
);

CREATE TABLE taxonomy_tag (
  network_id   TEXT NOT NULL,
  name         TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN ('pending','approved','rejected')),
  moderated_by TEXT NOT NULL DEFAULT '',
  updated_at   INTEGER NOT NULL,
  PRIMARY KEY (network_id, name)
);

-- Listing classification: category, goods/service type, condition, coarse area.
-- No coordinates of any kind are stored per listing.
CREATE TABLE listing_category (
  network_id   TEXT NOT NULL,
  listing_id   TEXT NOT NULL,
  category_id  TEXT NOT NULL DEFAULT '',
  listing_type TEXT NOT NULL DEFAULT '' CHECK (listing_type IN ('','goods','service')),
  condition    TEXT NOT NULL DEFAULT '' CHECK (condition IN ('','new','like_new','good','fair','for_parts')),
  area_id      TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (network_id, listing_id)
);
CREATE INDEX listing_category_cat ON listing_category (network_id, category_id);
CREATE INDEX listing_category_area ON listing_category (network_id, area_id);

CREATE TABLE listing_tags (
  network_id TEXT NOT NULL,
  listing_id TEXT NOT NULL,
  tag        TEXT NOT NULL,
  PRIMARY KEY (network_id, listing_id, tag)
);
CREATE INDEX listing_tags_tag ON listing_tags (network_id, tag);

CREATE TABLE listing_availability (
  network_id     TEXT NOT NULL,
  listing_id     TEXT NOT NULL,
  availability   TEXT NOT NULL DEFAULT '' CHECK (availability IN ('','available','limited','unavailable')),
  available_from INTEGER,
  expires_at     INTEGER,
  renewed_at     INTEGER,
  renew_count    INTEGER NOT NULL DEFAULT 0,
  version        INTEGER NOT NULL DEFAULT 1,
  updated_at     INTEGER NOT NULL,
  PRIMARY KEY (network_id, listing_id)
);
CREATE INDEX listing_availability_exp ON listing_availability (network_id, expires_at);

CREATE TABLE listing_media (
  id              TEXT PRIMARY KEY,
  network_id      TEXT NOT NULL,
  listing_id      TEXT NOT NULL,
  owner_id        TEXT NOT NULL,
  sha256          TEXT NOT NULL,
  mime            TEXT NOT NULL CHECK (mime IN ('image/jpeg','image/png')),
  alt             TEXT NOT NULL,
  width           INTEGER NOT NULL,
  height          INTEGER NOT NULL,
  thumb_w         INTEGER NOT NULL,
  thumb_h         INTEGER NOT NULL,
  bytes           INTEGER NOT NULL,
  thumb_bytes     INTEGER NOT NULL,
  storage_path    TEXT NOT NULL,
  thumb_path      TEXT NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('pending','active','removed','deleting')),
  position        INTEGER NOT NULL,
  created_at      INTEGER NOT NULL,
  retention_until INTEGER
);
CREATE UNIQUE INDEX listing_media_dedupe ON listing_media (network_id, listing_id, sha256) WHERE status IN ('pending','active');
CREATE INDEX listing_media_listing ON listing_media (network_id, listing_id, status, position);
CREATE INDEX listing_media_gc ON listing_media (status, retention_until);

CREATE TABLE discovery_audit (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  network_id TEXT NOT NULL,
  actor      TEXT NOT NULL,
  action     TEXT NOT NULL,
  target     TEXT NOT NULL,
  detail     TEXT NOT NULL DEFAULT '',
  at         INTEGER NOT NULL
);
CREATE INDEX discovery_audit_net ON discovery_audit (network_id, id DESC);

CREATE TABLE discovery_idempotency (
  network_id   TEXT NOT NULL,
  actor        TEXT NOT NULL,
  idem_key     TEXT NOT NULL,
  req_hash     TEXT NOT NULL,
  state        TEXT NOT NULL CHECK (state IN ('inflight','done')),
  status       INTEGER NOT NULL DEFAULT 0,
  content_type TEXT NOT NULL DEFAULT '',
  body         BLOB,
  created_at   INTEGER NOT NULL,
  PRIMARY KEY (network_id, actor, idem_key)
);
