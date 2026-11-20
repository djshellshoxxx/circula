# Migration, retention, backup and disable

Call `Migrate` explicitly under the host's deployment authority. This initial feature-owned schema is version 1 in `future_area_schema`; it neither reads nor changes SQLite's global `user_version`. It adds `future_areas`, `future_listing_areas`, `future_area_network_policy`, `future_area_retries`, `future_area_audit` and indexes. Existing core listings remain unclassified until an explicit owner operation. Schema creation and version checks are transactional and repeatable. An unexpected feature version fails rather than being guessed at.

Use the host's private database/file permissions, one-process-per-file guarantee, foreign-key enforcement on every connection and timeout policy. The module does not open a path or override host settings. Current policy callbacks must use the supplied transaction and existing principal/MFA checks. Actual root adapter compatibility is still unqualified.

A test closes the SQLite database cleanly, copies its complete database file, opens the backup, reruns migration and checks catalog/association revisions. For production, use the existing consistent SQLite backup method, or stop the writer and checkpoint/close before copying **all necessary database/WAL state**. Never copy a live database file alone. The module does not replace the core backup tooling.

Archival is the normal deletion path. It preserves internal references/audit while deactivating current selections. Restoration does not reactivate those selections. `CleanupListing` removes a departed/archived listing's current association after fresh ownership/departure policy checks. Cleanup is naturally idempotent by association deletion and audits only an actual removal. It does not consume or depend on retry quota, so quota exhaustion cannot prevent departure/privacy cleanup. Historical audit rows and actor-scoped retry tombstones remain private administrator data. Never prune retry records and then permit the same namespace to replay old keys. The bounded retry quota fails closed and requires a host-approved retention/retirement procedure; no background pruning job is installed.

If a rollout must be rolled back, remove registration/stop invoking these interfaces and retain its tables for backup/audit. Do not apply destructive down migrations or alter ledger history. Network sharing remains off until a fresh admin enables it; disabling immediately removes exports at the next fresh read. Rotate cursor keys to expire all outstanding 15-minute cursors; cursor secrets are host-owned and never written here.

Media remains outside this module. Before connecting listing photos, qualify an independent sanitizer that strips GPS/EXIF from decoded/reencoded images before storage/export. This is an integration gate, not a claim that area filtering itself sanitizes images.

## Audit safety operations

When retry quota is exhausted, use the admin **Stop sharing immediately** form or host CLI `unshare`; the service DisableSharing call is naturally idempotent and stores no retry record. Listing cleanup similarly remains available without retry allocation and atomically retires cached assignment replies into fingerprint tombstones. Never delete fingerprints merely to recover quota: host retention must preserve replay refusal or disable the feature during an approved data retirement. A stale assignment/sharing request can now return conflict/unavailable after lifecycle changes instead of returning a historical active result.

Owned schema definitions are checked at Migrate; unexpected/incompatible tables, views or indexes cause rollback rather than silently initializing around corrupt state. Foreign keys must be enabled on every pooled connection, and each mutation verifies that setting.
