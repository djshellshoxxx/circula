-- Owner bulk sign-out must visit the rows it changes, but a repeat request
-- must not rescan an ever-growing retained revoked history (API-036).
-- Keep the complete keyset index from 0020 for active-session inventories.
CREATE INDEX sessions_owner_unrevoked ON sessions(person_id, id) WHERE revoked_at IS NULL;
