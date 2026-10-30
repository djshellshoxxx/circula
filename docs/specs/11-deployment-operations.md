# 11 Deployment and Operations

Status: Draft v0.1 · Owner/role: Operations lead (platform operator role) with Product architect · Last updated: 2026-10-28 · Scope: supported deployment models, environments, single-binary packaging, configuration and secrets, embedded SQLite storage, backup and restore, migrations and upgrades, monitoring (health and metrics endpoints), scheduled jobs, support boundaries and failure recovery for the MVP · Dependencies: [00](00-conventions.md) (DEC-001, DEC-003), [05](05-architecture.md), [06](06-data-ledger-audit.md), [07](07-api-contract.md), [09](09-security-privacy.md), [10](10-governance-operations.md), [12](12-quality-test-plan.md), [13](13-assumptions-risks-decisions.md) · Open questions: section 12

Normative terms follow [00 section 1](00-conventions.md#1-normative-terms). Decisions DEC-001 (single binary, Go, embedded SQLite) and DEC-003 (release targets) are authoritative in [00 section 6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log) and are not restated here. Requirements use the prefix `OPS`.

This document is technology-specific only where [00](00-conventions.md) already commits the project to a technology (Go, SQLite, the listed platforms). Infrastructure vendors are not selected here.

---





## Historical implementation snapshot (not a current release attestation)

This table records a historical audit snapshot at `925f637` on `claude/circula-build`; it is not re-derived for the current checkpoint and must not be read as current implementation status or a release approval. Current checkpoint status and remaining gates are tracked in [RESUME-2026-10-24](../audit/RESUME-2026-10-24.md). **built**, **partial**, **not built**, and **deferred** below retain the meanings assigned in that historical snapshot. Detailed historical paths and test names are in [TRACEABILITY.md](../TRACEABILITY.md); owner decisions remain in [DECISIONS.md](../DECISIONS.md). No release gate is claimed here.

Summary: built 16, deferred 1, not built 2, partial 16.

| ID | Priority | Status | Note |
|---|---|---|---|
| OPS-001 | P0 | built | One artefact for both models |
| OPS-002 | P0 | built | Single process; embedded database; loopback default |
| OPS-003 | P0 | partial | Five targets built in CI with checksums; per-target smoke run not performed |
| OPS-004 | P1 | partial | Environments described in docs/ops/README.md; no staging refresh process |
| OPS-005 | P2 | deferred | Alternative store and multi-node deferred; no SQLite import outside store |
| OPS-006 | P0 | built | Flags, environment, file, defaults precedence |
| OPS-007 | P0 | built | Validation before serving; check-config does not create database |
| OPS-008 | P0 | built | Settings present with documented defaults |
| OPS-009 | P0 | partial | Secret files read; inline secret warning not separately tested |
| OPS-010 | P1 | partial | Backup key rotation documented; session key rotation not implemented |
| OPS-011 | P0 | built | WAL, foreign keys, busy timeout, startup pragma check; durability fixed (CMD-06) |
| OPS-012 | P0 | built | Short write units; busy returns retryable 503 |
| OPS-013 | P0 | built | healthz, readyz, metrics with allow-list and token |
| OPS-014 | P0 | partial | HTTP and job metrics built; posting attempt and idempotent-replay counters absent |
| OPS-015 | P0 | partial | Alert table in docs/ops/monitoring.md; alerts not configured or tested in staging |
| OPS-016 | P0 | built | Read-only verify with checksum test; CLI and admin entry points |
| OPS-017 | P0 | built | Online snapshot with manifest and checksum; load test not run |
| OPS-018 | P0 | built | Daily schedule with 30 daily and 12 monthly retention |
| OPS-019 | P0 | partial | Backups not encrypted; off-host copy is an operator procedure |
| OPS-020 | P0 | built | Restore verifies, refuses while serving, writes a local record and a database.restored audit event |
| OPS-021 | P0 | built | Drill procedure + checklist in docs/ops/backup-restore.md; scripts/restore-drill.sh records drills |
| OPS-022 | P0 | built | Versioned forward-only migrations; newer schema refused |
| OPS-023 | P0 | partial | Post-migration verification only; pre-migration verification and injected-failure test not present |
| OPS-024 | P0 | partial | Upgrade sequence documented; maintenance mode not built |
| OPS-025 | P1 | built | Explicit migrate mode default; auto mode takes backup |
| OPS-026 | P0 | partial | Release notes template exists; version set by build flag |
| OPS-027 | P0 | built | Expiry, purge, verify and backup jobs idempotent; last run recorded in metrics |
| OPS-028 | P0 | built | Expiry uses same service path; no entries |
| OPS-029 | P1 | not built | No mail and no stored notices |
| OPS-030 | P1 | partial | Severity levels in docs/ops/runbook.md; not linked from incident procedure |
| OPS-031 | P1 | partial | Support boundary documented; break-glass not built |
| OPS-032 | P0 | built | Structured JSON logs; log level needs restart |
| OPS-033 | P0 | partial | Failure table documented; per-network freeze not built |
| OPS-034 | P1 | partial | RPO and RTO recorded as DEC-016 defaults in docs/ops/rpo-rto.md; owner confirmation outstanding |
| OPS-035 | P1 | partial | Binary, checksum and module list per target; signing deferred; no secret scan in package |

## 1. Deployment models

| Model | Description | Who operates | MVP status |
|---|---|---|---|
| Self-hosted | Operator runs the same binary on their own host; one or more networks per instance | Operator of the instance | Required |
| Hosted | Platform operator runs the same binary for many networks | Platform operator | Required (DEC-001 optional hosted deployment) |
| Multi-node or clustered | Several application instances sharing one database | Operator | Not in MVP (see OPS-005 and section 11) |

**OPS-001** · P0 · Same artifact for both models
- Requirement: Self-hosted and hosted deployments MUST run the same release artifact for a given version. Hosted-only behaviour MUST be controlled by configuration, not by a separate build.
- Rationale: DEC-001 defines one binary. Divergent builds would create untested paths.
- Acceptance criteria:
  - A release build produces identical behaviour under self-hosted and hosted configurations in the end-to-end suite (test; [12](12-quality-test-plan.md)).
  - The release manifest lists one artifact per target platform and version (release check).

**OPS-002** · P0 · Single-process deployment
- Requirement: The MVP MUST run as a single application process with an embedded SQLite database file, a local file system for configuration and backups, and no required external service other than an optional outbound mail service (DEC-009). A reverse proxy for TLS termination is supported and recommended.
- Rationale: The embedded model keeps operations simple (DEC-001). It also constrains write throughput, which is an accepted risk (RSK in [13](13-assumptions-risks-decisions.md)).
- Acceptance criteria:
  - Start-up with only the binary and a configuration file succeeds on a clean host with no network service other than the database file path (install test on each platform, [12](12-quality-test-plan.md)).
  - Stopping the process and starting it again preserves all data (restart test).

**OPS-003** · P0 · Supported platforms
- Requirement: The release MUST provide builds for windows/amd64, linux/amd64, linux/arm64, darwin/amd64 and darwin/arm64, each with a SHA-256 checksum file (DEC-003).
- Rationale: Stated targets must be verifiable.
- Acceptance criteria:
  - Each target builds in CI from the same commit (build check).
  - Each target runs the smoke suite in [12](12-quality-test-plan.md) on a host of that platform, or the release notes state which targets were not run and why (release gate).
  - Checksums match recomputed values for each artifact (release check).

**OPS-004** · P1 · Environments
- Requirement: The project MUST define four environments: development (local), continuous integration (ephemeral), staging (hosted deployments only, mirrors production configuration), and production. Production data MUST NOT be copied to development or CI. Test data copied from production for staging MUST be anonymised under [09](09-security-privacy.md) SEC-014 and SEC-018 rules.
- Rationale: Environment parity catches configuration issues; data copies create privacy risk.
- Acceptance criteria:
  - The environment definitions are documented (documentation check).
  - A staging refresh produces a dataset in which the personal-data scan finds no unanonymised identifiers (test).

**OPS-005** · P2 · Horizontal scaling and alternative stores
- Requirement: Multi-node operation and an alternative database (for example PostgreSQL, as anticipated in DEC-001) are deferred. The storage layer MUST remain behind an interface that does not expose SQLite-specific behaviour to domain services, so that a later store can be added without changing domain rules ([05](05-architecture.md)).
- Rationale: Preserves the upgrade path without building it now.
- Acceptance criteria:
  - No package outside the storage adapter imports the SQLite driver (static check).

## 2. Configuration

**OPS-006** · P0 · Configuration sources and precedence
- Requirement: Configuration MUST be read from three sources: a configuration file, environment variables with a common prefix, and command-line flags. Precedence from highest to lowest MUST be: flags, environment variables, configuration file, built-in defaults (recommended; DEC-017). Every setting MUST have one documented name that is valid across all three sources, and a single documented default.
- Rationale: Operators need predictable overrides in containers, service managers and hand-edited files.
- Acceptance criteria:
  - A value set in all three sources resolves to the flag value (test).
  - Each documented setting is accepted by each source it is documented for (test).
  - The configuration reference in operator documentation matches the implemented settings (documentation check).

**OPS-007** · P0 · Validation at start-up
- Requirement: The application MUST validate all configuration before opening the database or listening on a network address. Invalid or missing required values MUST cause a non-zero exit with a message that names the setting, and MUST NOT print secret values. The product MUST provide a configuration check mode that validates without starting the service and reports the effective configuration with secrets redacted.
- Rationale: Fail-fast prevents partially configured production instances (T-11).
- Acceptance criteria:
  - An invalid value for each required setting causes exit with a non-zero code and a message naming the setting (test).
  - The configuration check output contains no secret canary value (test; [12](12-quality-test-plan.md)).
  - The configuration check does not modify the database file (test).

**OPS-008** · P0 · Required settings and defaults
- Requirement: The settings below MUST exist. Values marked "required" have no default. Others have the default shown.
  - Listen address and port (default: loopback address; the product MUST NOT default to listening on all interfaces without an explicit setting).
  - Public base address used for links in notices (required).
  - Database file path (required; the directory MUST be writable).
  - Backup directory and schedule (default: daily; recommended).
  - Session idle and absolute timeouts (defaults from DEC-005).
  - Authentication throttling limits (bounded by the minimum and maximum set in [09](09-security-privacy.md) SEC-006).
  - Maximum request body size (default: 1 MiB; recommended, to be confirmed by QA).
  - Request timeout (default: 30 s; recommended).
  - Log level and log format (default: info; structured).
  - Secure cookie and HTTPS mode (SEC-022).
  - Mail service settings (optional; DEC-009).
  - Metrics endpoint access mode (OPS-013).
- Rationale: A fixed, documented set keeps support predictable.
- Acceptance criteria:
  - Each setting appears in the configuration reference and in the configuration check output (documentation and test).
  - Starting with none of the required settings fails with a message naming each one (test).

**OPS-009** · P0 · Secrets handling
- Requirement: Secrets (per [09](09-security-privacy.md) SEC-027) MUST be provided by environment variable or by a file path referenced in configuration. A configuration file MUST NOT require secret values inline; if an inline value is present the product MUST warn at start-up. Secrets MUST NOT be written to the database except in encrypted form that is required for operation, and the key for it MUST NOT be stored alongside it.
- Rationale: Container and service environments commonly provide secrets as files; inline secrets leak into version control.
- Acceptance criteria:
  - A secret supplied as a file is read and used (test).
  - An inline secret in the configuration file produces a warning at start-up that does not print the value (test).
  - A scan of the database and backups finds no plaintext secret canary (test).

**OPS-010** · P1 · Secret rotation
- Requirement: The operator documentation MUST describe the rotation procedure for each secret. Rotating the session key MUST invalidate sessions and MUST NOT alter ledger data. Rotating the backup encryption key MUST keep existing backups restorable with the previous key until their retention expires.
- Rationale: Rotation is needed after suspected exposure and for routine hygiene.
- Acceptance criteria:
  - The rotation procedure exists for each secret (documentation check).
  - Rotation tests in [12](12-quality-test-plan.md) pass for session key and backup key (test).

## 3. Storage: embedded SQLite

**OPS-011** · P0 · Storage configuration
- Requirement: The database MUST use write-ahead logging, foreign-key enforcement, a busy timeout, and a documented durability setting that the operator can read in the configuration reference. Every connection MUST be opened with these settings. The database file MUST NOT be shared with another process other than the application's own backup routine.
- Rationale: These settings are required for consistent behaviour of concurrent posting (LED-4, LED-5) on an embedded single-writer database.
- Acceptance criteria:
  - Start-up verifies the journal mode and foreign-key setting and fails if they are not in effect (test).
  - A concurrent-write test with a busy database returns a retryable error rather than corrupting or silently dropping a posting (test; [12](12-quality-test-plan.md)).

**OPS-012** · P0 · Write concurrency and bounded transactions
- Requirement: Each posting MUST execute as one short write transaction that includes the balance check, both entries, the transaction status change and the audit event (LED-4, LED-5, LED-6). Long-running work such as exports and reports MUST NOT hold the write lock; they MUST read from a consistent snapshot. Posting requests that cannot acquire the write lock within the configured timeout MUST return a retryable error with no partial state.
- Rationale: Serialised postings protect the credit limit. Long write locks would block all members.
- Acceptance criteria:
  - An export running during concurrent postings completes and contains a consistent snapshot that reconciles (test; [12](12-quality-test-plan.md)).
  - A posting that times out waiting for the lock leaves no entries, status change or audit event (test).
  - No posting transaction in the code base performs network or file I/O inside the write transaction (static review check).

**OPS-013** · P0 · Metrics and health endpoints
- Requirement: The application MUST expose two operational endpoints. The health endpoint (path `/healthz`) MUST report whether the process is alive and whether the database is reachable with the expected schema version. It MUST return a coarse status and MUST NOT include member, network, balance or configuration secret data. Without an authorized operator credential it MUST return only the coarse status. The metrics endpoint (path `/metrics`) MUST expose operational metrics in a documented machine-readable format (recommended: a common text exposition format; DEC-018) and MUST be restricted to an allowed internal address range or an operator credential by default.
- Rationale: Operators need liveness and readiness signals and metrics without exposing member data (T-11).
- Acceptance criteria:
  - An unauthenticated request to the health endpoint from outside the allowed range returns the coarse status only (test).
  - Stopping database access makes the health endpoint report unhealthy (fault injection test).
  - The metrics endpoint is refused from an address outside the allowed range when no credential is given (test).
  - Metric label values contain no member identifiers, network names, balances or personal data (scan test; [12](12-quality-test-plan.md)).
  - Route labels use route templates, not raw paths (test).

**OPS-014** · P0 · Required operational metrics
- Requirement: The metrics endpoint MUST include at least: request count and latency by route template and status class; authentication failures and throttling events; posting attempts, successes and failures by reason class; idempotent replays; ledger verification result and time of last run; time of last successful backup; schema version; database file size and write-ahead log size; scheduled job run results; and process uptime.
- Rationale: These are the minimum signals to detect ledger, security and availability problems.
- Acceptance criteria:
  - Each listed metric is present in a test instance after the corresponding event (test).
  - A failed ledger verification sets a metric value that an alert rule can match (test).

**OPS-015** · P0 · Alerts
- Requirement: The operator documentation MUST include alert conditions for at least: ledger verification failure (critical); health endpoint unhealthy for more than a set period (critical); no successful backup within a set period (recommended default: 26 hours for a daily schedule); database or backup volume above a set percentage (recommended default: 80 percent); sustained authentication throttling above a set rate (warning); and migration failure (critical). Each alert MUST name its owner role and its first response step, which MUST refer to the incident procedure in [09](09-security-privacy.md) SEC-030.
- Rationale: Alerts without owners are not acted on.
- Acceptance criteria:
  - Each alert is listed with condition, severity, owner role and first response (documentation check).
  - A simulated ledger failure in staging triggers the critical alert (staging test).

**OPS-016** · P0 · Ledger verification command
- Requirement: The product MUST provide a verification operation that recomputes, for each network, the sum of entries per transaction, the sum of all balances, the cached balances against summed entries, and the credit-limit condition (LED-1, LED-3, LED-5). The LED-5 check MUST assess each debit against the effective credit limit applicable when that posting occurred: a debit that worsened an account beyond its then-effective limit is an integrity failure; an incoming credit that improves an already over-limit position is permitted. A current balance below the current limit, by itself, MUST NOT be reported as corruption when a prospective limit reduction left an inherited negative balance (GOV-019). The verification calculation MUST be read-only and runnable while the service is running against a consistent snapshot and offline against a restored copy. Scheduled and other automatic verification MUST write no database state; its outcome MUST be observable through operational metrics and logs (OPS-014, ARC-015, DEC-031). An explicit administrator-initiated check MUST additionally record its result in `consistency_runs` and as an audit event for an affected network. Neither path may change ledger or other business data.
- Rationale: Ledger verification is the primary detection control for integrity faults (T-04, T-14).
- Acceptance criteria:
  - On a seeded consistent dataset the operation reports success (test).
  - On a dataset with a seeded cached-balance mismatch the operation reports the network and account and exits non-zero (test).
  - A member's inherited below-limit balance after a prospective limit reduction, and a later incoming credit that improves it while it remains below that limit, do not fail verification; a debit that improperly worsened the position is detected (test; GOV-019, LED-5).
  - Read-only verification, including scheduled verification, leaves a checksum of the database unchanged (test).
  - A scheduled verification outcome is observable through metrics and logs without a database write; an explicit administrator-initiated check records a consistency run and audit event (test; DEC-031).

## 4. Backup and restore

**OPS-017** · P0 · Backup method and consistency
- Requirement: Backups MUST be produced by an online backup method that yields a consistent snapshot while the service is running, without stopping postings for longer than the write-lock timeout in OPS-012. Each backup MUST be accompanied by a manifest recording the time (UTC), the schema version, the application version, a checksum of the backup file, and the ledger verification result at the time of the backup. Copying the live database file without the method is not a supported backup.
- Rationale: A raw file copy during writes can be inconsistent.
- Acceptance criteria:
  - A backup taken during a concurrent posting load restores without integrity errors (test; [12](12-quality-test-plan.md)).
  - A backup manifest exists and its checksum matches the file (test).
  - The backup operation does not block postings longer than the timeout (performance test; threshold per DEC-023).

**OPS-018** · P0 · Backup schedule and retention
- Requirement: Backups MUST run on a schedule (default daily, configurable) and MUST also be taken before every upgrade or migration (OPS-024). Retention MUST be configurable. Recommended defaults: keep 30 daily backups and 12 monthly backups. Expired backups MUST be deleted and the deletion MUST be logged without personal data.
- Rationale: Retention limits both recovery loss and personal-data exposure (SEC-018, SEC-028).
- Acceptance criteria:
  - The scheduled backup runs and produces a manifest (test with injected clock).
  - A pre-upgrade backup exists before a migration proceeds (test).
  - Expired backups are removed according to the configured count (test).

**OPS-019** · P0 · Backup storage and protection
- Requirement: Backups MUST be encrypted at rest (SEC-028). The backup location MUST be writable by the application and readable only by the operator roles that need it. Hosted deployments MUST store at least one copy outside the primary host and outside the primary failure domain (recommended; DEC-016). Self-hosted operators SHOULD store a copy off the host.
- Rationale: A backup on the same disk as the database does not protect against disk loss (T-14, T-23).
- Acceptance criteria:
  - An encrypted backup cannot be opened without the key (test).
  - The hosted runbook names the off-host copy location and the access list (documentation check).

**OPS-020** · P0 · Restore procedure and verification
- Requirement: A restore MUST be performed into a stopped instance or a new instance, never into a running instance that holds newer data, unless the operator explicitly confirms the overwrite after a current backup is taken. After restore, the product MUST run the ledger verification (OPS-016) and MUST refuse to accept postings until verification passes or the operator explicitly records an override with a reason. The restore MUST record a restore audit event that states the backup time, the restore time and the result, and that identifies any window of data between the backup and the failure that may be missing.
- Rationale: Restoring over newer data loses records. Verification prevents posting on a corrupted copy (T-04).
- Acceptance criteria:
  - A restore over a running instance without the confirmation is refused (test).
  - A restored copy with a seeded inconsistency refuses postings until verification passes (test).
  - A restore marks the data window that may be missing in the restore audit event (test).
  - Restore time for the reference dataset is measured and recorded (performance test; DEC-016 for targets).

**OPS-021** · P0 · Restore drills
- Requirement: The operator MUST run a restore drill from a current backup into a separate instance before hosted launch, at least once per quarter for hosted deployments, and before each release that changes storage or migrations. Self-hosted operators are instructed to run a drill at least once per year. The drill result (date, backup used, restore duration, verification outcome) MUST be recorded.
- Rationale: Untested backups are not reliable recovery (RSK in [13](13-assumptions-risks-decisions.md)).
- Acceptance criteria:
  - A drill record exists for the release before the release gate passes (release gate, [12](12-quality-test-plan.md)).
  - A failed drill blocks the release gate until the cause is recorded and resolved (process check).

## 5. Migrations and upgrades

**OPS-022** · P0 · Versioned, forward-only migrations
- Requirement: The schema MUST be changed only by versioned migrations embedded in the binary, applied in order, each recorded with its version and time. Migrations MUST be forward-only: the product MUST NOT provide an automated downgrade path. A migration MUST NOT update or delete ledger entries or audit events, except for pseudonymisation under SEC-018 that is itself a versioned, reviewed migration.
- Rationale: Downgrades on a ledger are unsafe. Rollback uses backup restore (OPS-024). Entries are append-only (LED-2).
- Acceptance criteria:
  - Each migration has a version entry and runs once; a second run is a no-op (test).
  - Migration files contain no statement that updates or deletes entries or audit events, except for the reviewed pseudonymisation migration (static check, [12](12-quality-test-plan.md)).
  - Starting a binary with a schema version newer than it knows refuses to run (test).

**OPS-023** · P0 · Migration safety
- Requirement: Each migration MUST run in a transaction where the storage engine allows it. A failed migration MUST leave the database at the previous version. Before and after a migration that touches balances, entries or transaction records, the verification operation (OPS-016) MUST run, and the after-migration result MUST NOT show any invariant failure that was absent before the migration. Migration runs MUST be logged and produce a metric (OPS-014).
- Rationale: Migrations are a common source of silent data corruption.
- Acceptance criteria:
  - A migration with an injected failure midway leaves the database unchanged (test).
  - A migration that breaks a seeded invariant is refused before commit (test).
  - Migration tests run on fixtures for every released schema version (test; [12](12-quality-test-plan.md)).

**OPS-024** · P0 · Upgrade procedure
- Requirement: Every upgrade MUST follow this sequence: read the release notes for breaking changes; take a verified backup (OPS-018); stop postings (enter maintenance mode, which refuses writes and keeps read-only health); stop the process; replace the binary; start with the new version (which applies pending migrations, or runs them explicitly if the operator chose the explicit migration mode); run verification; resume postings. Rolling upgrades of a single-node embedded deployment are not supported. Rollback MUST be performed by restoring the pre-upgrade backup with the previous binary (OPS-020).
- Rationale: A fixed, verifiable sequence limits downtime and data risk.
- Acceptance criteria:
  - The upgrade test on a fixture instance follows the sequence and each step is logged (test).
  - Maintenance mode refuses postings with a clear message and keeps health and read access (test).
  - Rollback from a seeded failed upgrade restores the pre-upgrade state (test).

**OPS-025** · P1 · Migration mode
- Requirement: The operator MUST be able to choose between applying pending migrations automatically at start-up and applying them only through an explicit command. The default for self-hosted deployments is explicit (recommended; DEC-017). Hosted deployments MUST use the explicit mode.
- Rationale: Explicit migration gives operators control and an opportunity to take a backup first.
- Acceptance criteria:
  - In explicit mode, a start-up with pending migrations refuses to serve postings and reports the pending version (test).
  - In automatic mode, pending migrations run before serving and a backup is taken first (test).

**OPS-026** · P0 · Versioning and release notes
- Requirement: Releases MUST use a version number that identifies compatibility. Each release MUST include release notes listing breaking changes, schema changes, configuration changes, API changes (per [07](07-api-contract.md) compatibility rules), security fixes and required operator actions. A release that changes the schema MUST state whether a backup is required before upgrade.
- Rationale: Operators need to know when an upgrade is risky.
- Acceptance criteria:
  - Each release contains the listed sections or states "none" (release check).
  - The health endpoint reports the application version (test).

## 6. Scheduled work

**OPS-027** · P0 · Idempotent scheduled jobs
- Requirement: Scheduled jobs (proposal expiry, dispute deadlines, session cleanup, backup, verification, log retention and notice delivery) MUST be idempotent and MUST be safe to run twice or late. Each job MUST expose its last run and result through the appropriate operational metrics or logs. Jobs that maintain resumable work state MUST persist that state; scheduled ledger verification follows the read-only and reporting rule in OPS-016 and DEC-031. Jobs MUST use server time in UTC and MUST NOT depend on the local time zone of the host.
- Rationale: Restarts and clock changes are normal operating events.
- Acceptance criteria:
  - Running the expiry job twice expires each proposal once (test).
  - A job delayed by a simulated restart runs once on catch-up without duplicating effects (test with injected clock).
  - A time-zone change on the host does not change job timing (test).

**OPS-028** · P0 · Expiry does not bypass posting
- Requirement: Expiry of a proposal MUST change its status to expired and write an audit event, and MUST NOT post entries. A job that expires a proposal MUST use the same state-change path as user actions.
- Rationale: Background jobs are a common source of bypassed controls.
- Acceptance criteria:
  - An expired proposal has no entries (test).
  - The expiry audit event exists (test).

## 7. Notices and mail

**OPS-029** · P1 · Notification delivery
- Requirement: In-app notices MUST be stored and shown to the recipient. Outbound mail, if configured, MUST be retried with backoff for transient failures and MUST NOT include balances, transaction details or tokens in the message body beyond what the notice must state under [10](10-governance-operations.md). Mail failure MUST NOT block a ledger change; a failed notice MUST be recorded and retried or flagged for the administrator.
- Rationale: Notices are required for transparency (SEC-012, GOV-003), but a mail outage must not stop trade.
- Acceptance criteria:
  - A mail outage during trade acceptance does not prevent the acceptance (fault injection test).
  - A failed notice is retried and eventually flagged (test).
  - A mail body contains no token value (test).

## 8. Support and responsibilities

**OPS-030** · P1 · Support tiers and severity
- Requirement: The operator documentation MUST define severity levels for incidents and support requests (for example: critical for ledger or security incidents, high for loss of posting for a network, normal for other faults), response targets for each, and the channels used. Targets for the MVP are recommended and MUST NOT be presented as service-level commitments until the platform operator approves them (DEC-022).
- Rationale: Clear severity prevents delayed handling of ledger incidents.
- Acceptance criteria:
  - Severity levels and channels are documented (documentation check).
  - The incident procedure in [09](09-security-privacy.md) SEC-030 references the same levels (cross-document check).

**OPS-031** · P1 · Support access
- Requirement: Support actions MUST follow the boundary in [10](10-governance-operations.md) GOV-027. Any support action that reads or changes network data MUST be recorded and MUST go through the break-glass rule in [09](09-security-privacy.md) SEC-013 when it involves financial or personal data.
- Rationale: Support is a common route for unauthorized access (T-17).
- Acceptance criteria:
  - A support read of a network record without break-glass is refused (test).

GOV-027 support boundary: network administrators help members with network rules and trades. The platform operator helps with software, deployment and platform policy. A support request alone grants no access to network data. Any approved access to financial or personal data for support must use a recorded, time-limited break-glass process under SEC-013, including a documented reason, a second staff approver, minimum necessary access, an audit trail readable by network administrators and notice as required. In hosted mode the operator can technically read stored data; these policy, audit and notice controls do not provide cryptographic separation or protect data from a compromised host. SEC-013 implementation remains partial, and its specific technical controls are governed by DEC-014.

## 9. Monitoring of the application's own logs

**OPS-032** · P0 · Structured logs
- Requirement: Application logs MUST be structured, written to standard output or a configured file, and follow [09](09-security-privacy.md) SEC-020. The log level MUST be configurable without a restart where the platform supports it, or documented as requiring a restart.
- Rationale: Structured logs support alerting and forensics without leaking secrets.
- Acceptance criteria:
  - Log output parses as structured records in a test (test).
  - A log-level change is applied or the restart requirement is documented (test or documentation check).

## 10. Failure modes and recovery

**OPS-033** · P0 · Failure recovery requirements
- Requirement: The operator documentation MUST include, for each failure mode below, detection, immediate action, recovery action and the owner role. The product MUST behave as stated.
  - Process crash during a posting: the posting is either fully committed or not at all after restart (LED-4); no partial entries.
  - Power loss: the database recovers to the last committed transaction on restart; the durability setting in OPS-011 is documented.
  - Disk full: writes are refused with a clear error; reads continue; the disk alert in OPS-015 fires; no partial posting.
  - Database file corruption: postings are refused; the operator restores from a verified backup (OPS-020).
  - Ledger verification failure: postings for the affected network are frozen; the operation records the failure; nothing is auto-corrected; the incident procedure applies ([09](09-security-privacy.md) SEC-030).
  - Mail service outage: notices queue (OPS-029); trades continue.
  - Suspected secret exposure: rotate the secret (OPS-010) and invalidate affected sessions.
  - Clock error on host: time-based expiry and deadlines use the server clock and are re-evaluated on restart; the anomaly is logged.
- Acceptance criteria:
  - Each failure mode has a fault-injection test or a documented drill, referenced in [12](12-quality-test-plan.md) (traceability check).
  - The ledger verification failure freezes only the affected network (test).

**OPS-034** · P1 · Recovery objectives
- Requirement: The product owner MUST set a recovery point objective and a recovery time objective for hosted deployments and a recommended target for self-hosted deployments (DEC-016). Until set, backups MUST be at least daily and restore drills MUST record measured durations so that the objectives can be checked against evidence.
- Rationale: Objectives without measurements are guesses.
- Acceptance criteria:
  - The chosen objectives are recorded in [13](13-assumptions-risks-decisions.md) before hosted launch (documentation check).
  - Drill records include measured restore duration (drill check, OPS-021).

## 11. Release packaging and distribution

**OPS-035** · P1 · Packaging
- Requirement: Each release MUST provide, for each target in OPS-003: the binary, a SHA-256 checksum, a software bill of materials (SEC-029), and the release notes (OPS-026). Packaging MUST NOT include configuration with secret values or test fixtures containing personal data. Code signing and notarisation are deferred (DEC-027).
- Rationale: Verifiable artifacts reduce supply-chain risk (T-22).
- Acceptance criteria:
  - The release contents list matches the required set (release check).
  - A search of the release archives finds no secret canary or personal-data fixture (test).

## 12. Open questions

1. Recovery point and recovery time objectives (DEC-016).
2. Durability setting for the embedded database and the expected trade-off (write-up in [05](05-architecture.md)). Open: DEC-075 (the build fixes synchronous=NORMAL).
3. Default metrics exposition format (DEC-018) and the internal address range default. Resolved as built: DEC-058.
4. Configuration file format and precedence (DEC-017), including whether a migration mode is set in the file or only by flag. Resolved as built: DEC-057.
5. Mail service: required or optional for password recovery (DEC-009).
6. Support objectives and whether any are offered externally (DEC-022).
7. Whether hosted and self-hosted backups use different schedules (recommended: same default, configurable). Resolved as built: DEC-059 (same default).
8. Hosted storage isolation: one database file per network or one shared file with tenant scoping (DEC-008). This changes backup and restore granularity, and this document will be revised when DEC-008 is decided. Open: DEC-008. As built: one shared database with network-scoped rows; see DEC-008 status in 13.
9. Code signing and notarisation (DEC-027), deferred. Deferred (DEC-027).
