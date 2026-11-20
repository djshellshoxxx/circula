# Isolated migration plan — FUX-105

Integration is HOLD. The selected implementation continues the complete locally audited branch at `3b4fbe4497827072df15f75e8e518f89925d6435`, itself descended from published feature research `7f92251cdb0d6ed5d9559730045a021a173a278b`. The different outage implementation is safely checkpointed at `1b5fe42883c29a2dcfd9e924f7d28facac9b541a`; it is not silently merged into the selected implementation.

1. Keep modernc.org/sqlite v1.60.1 and verified module checksums in this nested module only. Preserve all included dependency notices.
2. Explicit `Migrate` installs the additive local schema; import does not connect to a database or register routes. Migration is transactional and idempotent.
3. Authoritative host authorization, network opt-in, origin/session/CSRF, retention policy and audit callbacks must be qualified in a future separately authorized integration. No core route or migration is registered now.
4. Restore uses a consistent SQLite backup, private file permissions, no active writer, and migration rerun. The existing backup/restore acceptance test covers current feature records and per-user visibility. Do not copy a live database without its WAL.
5. No destructive down migration exists. Disable removes future registrations and revokes feature authorization; core ledger history is unaffected.

The new search uses an existing SQLite operation and needs no new migration, index format or dependency. It applies existing authorization and visibility inside a single transaction.
