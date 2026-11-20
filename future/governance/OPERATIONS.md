# Operations and deferred integration

This module is not activated in Circula. Enabling requires a separately selected integration change after core qualification. There is no root schema, route, job, flag or release edit here.

## Schema and transactions

Migrate performs additive CREATEs, indexes and schema-version initialization in one SQL transaction. Version 1 is the sole supported module schema. Unsupported versions fail without leaving a partially-created schema. There is no destructive down migration: disable the feature, preserve its data, and restore a verified whole-file backup if recovery requires it. All tables use a future_gov_ prefix. Network/owner/proposal compound keys, frozen eligibility and ballot uniqueness are enforced at the DB layer; foreign keys must be enabled on each connection.

Opening snapshots eligibility in the same transaction as its state change. Ballot insertion, command idempotency and audit commit together. A failing audit write rolls the ballot back. Concurrent writers can receive SQLite busy/constraint failures; callers must not report success on failure and can retry the same intent. Close computes one persisted outcome transactionally; certification and linking preserve its counts. Read operations use one DB transaction and current membership decision.

## Backup, restore and disable

Use the existing application's SQLite online backup/VACUUM mechanism or stop all writers, checkpoint WAL and copy a consistent database with protected permissions. Never copy only a live main file while ignoring WAL. The tests use SQLite VACUUM INTO and reopen the resulting snapshot to verify ballot/eligibility survival. Actual core-store backup/restore qualification is a later integration gate. Protect backups as sensitive ballot data; an encrypted storage/backup policy belongs to the host deployment.

Disable by removing route/CLI registration and setting host network opt-in false. No automatic startup or poll scheduler exists. Retain posted proposals, frozen eligibility, ballot identities, outcomes, intent records and audit for the community's published governance record lifetime; **do not** expire ballot/intent uniqueness records while a proposal remains accessible, because expiry could permit duplicate action replay. Hidden comment content is erased immediately from the visible content row; residual data in protected backups follows the host backup-retention policy. There is no automatic bulk purge. A production retention/deletion policy and legally required record handling must be reviewed before activation, rather than silently inventing a destructive retention period here.

## Remaining gates

Independent source/security review and acceptance reproduction; actual Circula transaction/auth/MFA and policy-history adapter; API contract/root regression qualification; automated and manual browser/accessibility checks; translations; deployment private-file/backup permissions and agreed retention policy; owner-selected single-feature rollout. Unit/race tests do not complete these gates. No merge, tag, deployment or publication is authorized by this module's successful tests.

## Audited limits and failure handling

See CONTRACT.md for retained-history and intent limits. Monitor storage before physical exhaustion; logical normal-command saturation preserves authorized withdrawal/moderation/review safety operations. The reference adapter validates table/index/constraint compatibility on migration and requires a persisted version-row lock before authority decisions. A failed or ignored write rolls back associated ballots, outcomes, audit and retry records. Reject/restore corrupt payloads rather than serving partially decoded state. Backup/restore and the retained-history policy still need qualification in the actual core deployment.
