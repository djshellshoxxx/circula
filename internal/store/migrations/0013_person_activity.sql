-- 0013_person_activity: the newest time a person was seen signed in (GOV-012).
-- Additive and forward-only. Sessions are deleted when they idle out, so the
-- inactivity job copies the newest last_seen_at of each person here before they
-- go. A network that defines an inactivity rule flags a member whose newest
-- activity (joining, this value, or a trade) is older than the rule. Nothing here
-- ever changes a membership.
CREATE TABLE person_activity (
  person_id    TEXT PRIMARY KEY,
  last_seen_at INTEGER NOT NULL
);
