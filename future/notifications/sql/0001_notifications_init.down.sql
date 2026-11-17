-- Rollback for 0001. Discards notification feature data only (queued
-- deliveries, preferences, sealed targets, audit); the ledger, proposals and
-- in-app notifications are untouched. Take a backup first.
DROP TABLE IF EXISTS notification_idempotency;
DROP TRIGGER IF EXISTS notification_audit_no_delete;
DROP TRIGGER IF EXISTS notification_audit_no_update;
DROP TABLE IF EXISTS notification_audit;
DROP TABLE IF EXISTS notification_delivery_target;
DROP TABLE IF EXISTS notification_delivery;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS notification_email_targets;
DROP TABLE IF EXISTS notification_preferences;
