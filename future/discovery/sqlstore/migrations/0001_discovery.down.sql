-- Rollback drops ONLY this module's tables. Core tables are untouched, and
-- sanitized image files on the blob store are NOT deleted by SQL: operators
-- must run the documented retention/cleanup before or after rollback.
DROP TABLE IF EXISTS discovery_idempotency;
DROP TABLE IF EXISTS discovery_audit;
DROP TABLE IF EXISTS listing_media;
DROP TABLE IF EXISTS listing_availability;
DROP TABLE IF EXISTS listing_tags;
DROP TABLE IF EXISTS listing_category;
DROP TABLE IF EXISTS taxonomy_tag;
DROP TABLE IF EXISTS coarse_area;
DROP TABLE IF EXISTS taxonomy_category;
DROP TABLE IF EXISTS discovery_listing_projection;
DROP TABLE IF EXISTS discovery_revision;
