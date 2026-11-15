# Circula security audit handoff

Date: 2026-11-15
Audit branch: `chatgpt/circula-security-audit-20261115`
Authoritative application baseline: PR #35 branch `claude/cool-ramanujan-n2peag`
PR #35 baseline commit: `c033cfa448fb89d1a5cc0ca87458b40d2709fa2e`
Audit handoff PR: #42 (draft; targets PR #35 integration branch)
Audit branch latest commit before this update: `7f41311fd93abaafea4c37e5753b243646c9f2d9`
Scope: security review only. No application code, financial rules, migrations, or production data were changed.

## Repository state

- Repository: `djshellshoxxx/circula` (Go, SQLite).
- PR #35 is the documented integration baseline. Its branch is 345 commits ahead of main and 0 behind at the checked comparison.
- PR #38: foundational specs. PR #40: QA-042 requirement-ID drift auditor. PR #41: future-only receipt/certificate specs; it explicitly says no current feature code is introduced.
- PR #35 handoff says QA-042 full traceability regeneration remains incomplete, alongside owner gates and SEC-033.
- PR #35 baseline commit `c033cfa` has a completed successful GitHub Actions run (run 43685188544). This is CI evidence, not proof of security.
- No historical branches were merged or changed during this audit.

## Completed: Priority 1 — ledger and financial integrity (source review only)

### Positive controls observed

- Posting is centralized in `internal/service/posting.go:postTx`: it reads both accounts, checks bounded arithmetic and payer capacity, obtains a ledger sequence, writes equal/opposite entries, updates cached balances, updates transaction state and writes a status event.
- SQLite writes use an in-process gate and `BEGIN IMMEDIATE` in `internal/store/sqlite.go:Update/run`.
- Service-level idempotency records are written in the same unit of work; the implementation hashes request payloads and rejects reuse with a different payload (`internal/service/idempotency.go`).
- `internal/service/atomicity_test.go:TestAcceptFaultInjectionLeavesNoTrace` injects failures across entry, balance, transaction, status-event, audit and idempotency writes and checks rollback.
- `internal/service/concurrency_test.go` contains concurrent acceptance and credit-limit tests.
- SQLite migrations define append-only triggers for entries and audit events; transaction posting invariants are also checked by `VerifyLedger`.
- These are implementation observations, not a claim that the ledger is secure. Tests were not executed in this environment.

### Historical A1 findings rechecked against PR #35

- A1-01 — departure settlement posting after closure/suspension/admin-required: **fix present in source**. `settleOnDeparture` now calls `p.requireNetworkActive()` before posting.
- A1-02 / A1-11 — scheduled decimal-scale changes reinterpret stored amounts: **fix present in source per remediation record**; inspect `UpdatePolicy` and regression tests again during a runtime-capable pass.
- A1-03 — re-review reverses a trade while another live dispute is open: **fix present in source**. `decideReReview` refuses reversal when `t.Status == disputed`.
- A1-04 — reversal can post while network is administrator-required: **CONFIRMED by source trace; runtime PoC not run**. `networkOpenErr` / `requireNetworkActive` rejects `AdminRequired`, but `internal/service/reversal.go:reverse` checks only `net.Suspended()`. Both `ResolveDispute` and `RetryDisputePosting` call `reverse` without a network-active gate. An administrator/reviewer can therefore resolve or retry a dispute and post a reversal while the network is read-only because it has no active administrator. Severity: Low under the existing A1 classification.
- A1-05 — expired-proposal sweep starvation: **CONFIRMED by source trace; runtime PoC not run**. `internal/store/sqlite_tx.go:ListExpiredProposals` selects the oldest expired proposals globally with a limit and no network filter. `internal/service/transactions.go:ExpireProposals` then skips frozen networks. If the first limited batch is continually occupied by frozen-network proposals, active-network proposals behind it can remain proposed indefinitely. Severity: Low.
- A1-06 — incomplete proposal cancellation on member departure: **CONFIRMED by source trace; runtime PoC not run**. `internal/service/departure.go:cancelOpenProposals` fetches only 500 proposed transactions and does not paginate. More than 500 open proposals involving a departing member leave excess proposals active, retaining reservations until expiry and referencing a departed party. Severity: Low.
- A1-07 — final closure export is not retried after a one-time export failure: **CONFIRMED by source trace; runtime PoC not run**. `FinaliseClosures` first calls `finaliseOne`, which sets `closed_at`; it then calls `recordFinalExport` in a separate unit. `ListClosures(true)` only returns rows with `closed_at = 0`. If export/recording fails after closure commits, the next sweep no longer selects that closure, so its final checksum/reconciliation may never be recorded. Severity: Low in the historical A1 report.
- A1-08 to A1-10 and remaining informational findings: not fully rechecked in this pass. Do not mark resolved.

### Tests and limitations

- Read PR #35 source patches for the posting path, SQLite transaction wrapper, transaction store, migrations, reversal/dispute/departure/closure paths, and the historical A1 report.
- Reviewed existing test source for atomicity, concurrent acceptance, departure settlement and reversal behavior.
- No shell/Go runtime was available through the connected GitHub interface. No `go test`, `go test -race`, fuzz, mutation, or PoC execution is claimed.
- Findings A1-04 through A1-07 are confirmed by code-path inspection but still need focused reproductions and regression tests before remediation is accepted.
- No destructive or production tests were run.

## Completed: Priority 2 — authentication, authorization and IDOR (source review only)

### Positive controls observed

- `service.openAny` resolves membership and token network scope server-side on every service call; callers do not supply roles.
- `requireAdmin` enforces active membership, the administrator role and configured MFA requirement for administrator operations using that gate.
- API bearer tokens are stored by hash, have expiry/revocation checks, and produce actors carrying only the stored scopes and optional network restriction.
- Token creation rejects scopes wider than the creator's and requires membership in a requested network.
- Session authentication checks revocation, absolute expiry and idle expiry. Password change revokes all sessions.
- Cookie-authenticated API unsafe methods use a session-bound CSRF token. Web forms use same-origin checks and CSRF tokens.
- `GetTransaction` applies object-level visibility checks; members cannot retrieve unrelated transactions. `GetMember` returns full details only to the subject or an administrator, and a reduced directory view to eligible active members.
- `TestAuthorizationMatrix` covers ten actor classes across service operations. `route_inventory_test.go` inventories API, web and ops routes and checks route authorization metadata. Tests were inspected but not executed.

### Confirmed findings

- **A3-F02 — Medium — break-glass administrators can make lasting changes. CONFIRMED by current source trace; historical A3 audit includes PoC evidence, but no current-head PoC was run.** `principal.requireLasting()` exists but is called only by a subset of write paths. Current code paths such as `CreateAdjustment`, `ApproveAdjustment`, `SetCreditLimit`, `AdminResetPassword`, `ReverseTransaction`, `DecideJoinRequest`, network suspension/resumption and other admin actions do not consistently call it. In addition, `requireIndependentApprover` returns success immediately for `p.bgAdmin`, bypassing the new-admin delay and appointment-chain independence check. A temporary break-glass admin can therefore create permanent balance changes, permanent credit-limit changes, reset another member's password, reverse a trade, or approve a pending adjustment immediately. The temporary role expires, but these effects do not. This contradicts the current code comment that break-glass access must not leave lasting changes. Remediation is blocked on an owner decision about the permitted break-glass scope; do not implement an allow-list or broaden powers without that decision.
- **A3-F03 — Low — configured MFA requirement is bypassed for decider operations. CONFIRMED by source trace; runtime test not run.** `principal.requireAdmin()` checks `mfaMissing`, but `principal.requireDecider()` does not. Admins without an enrolled second factor can reach decider-only paths such as dispute resolution (which can post a reversal) and challenge decisions. The intended policy must clarify whether an administrator using reviewer/decider powers must satisfy the administrator MFA requirement; the historical A3 audit treats this as a confirmed conformance weakness.
- **A3-F16 — previous throttle coupling appears fixed in current source.** Sign-in MFA attempts use `mfaKey`; session-side MFA management uses a distinct `mfaManageKey`. The current code comments describe the intended separation and lock decay. This is source-level verification only; the regression tests were not run.

### Owner decision required: A3-F02 / SEC-013 / DEC-014

The repository's SEC-013 addendum explicitly leaves open whether a break-glass administrator can use all administrator powers. The audit must not silently choose.

The owner must decide:
1. **Strict temporary access:** break-glass grants read/export access only, with no balance changes, adjustment approvals, credit-limit changes, credential resets, trade reversals, membership decisions, or network policy/state changes; or
2. **Explicitly bounded emergency powers:** name each permitted state-changing action, state its approval/notice requirements, explain how persistent effects are controlled after the 4-hour grant expires, and update the normative specification and tests.

Separately, SEC-013 still records owner/process gaps for a second approver, notice to affected administrators within 24 hours, and a published operator access policy. These are governance/verification gaps, not independently demonstrated code exploits in this pass.

## Completed: Priority 3 — cryptography and key management (source review only)

### Positive controls observed

- Password hashes use Argon2id with per-hash random salts and constant-time comparison. The configured default is 64 MiB, 3 iterations, 2 lanes.
- Session/API tokens use 32 random bytes and are stored as SHA-256 hashes; MFA challenges are also hashed and expire after five minutes.
- TOTP uses a 160-bit random seed, RFC 6238/HOTP HMAC-SHA1 interoperability, constant-time code comparison and a persisted last-accepted step to prevent replay.
- Recovery codes are random, single-use, stored as hashes bound to the person ID; tests cover reuse and concurrent replay.
- MFA seeds are AES-256-GCM sealed with the configured `MFASecret`, with the person ID as associated data. If no MFA secret is configured, the implementation stores a base32 seed and relies on database file permissions and backup encryption; this is an explicit configuration mode, not an implementation bypass.
- Backup encryption uses AES-256-GCM chunking, per-file random salt and nonce prefix, an incrementing chunk counter, and AAD binding the header, chunk index and final flag. Passphrase mode uses Argon2id; raw 32-byte base64 keys use HKDF-SHA256. Existing tests cover wrong keys, tampering, truncation, reordered/duplicated/appended chunks and nonce differentiation.
- The web signing key is 32 random bytes, stored with owner-only permissions on Unix; symlinked key files are rejected. Secret configuration files are permission-checked. Backup encryption and manifest-signing keys are required to differ.
- Backup restore checks the ciphertext checksum and manifest HMAC when a backup key is configured. Unsigned restore requires an explicit `-allow-unsigned` flag.

### Confirmed / suspected findings

- **CRYPTO-01 / A3-F17 — Low (conditional local attack) — predictable temporary files can follow attacker-planted symlinks. CONFIRMED by source trace; runtime PoC not run.** `cmd/circula/snapshot.go:writeManifest`, `cmd/circula/auditanchor.go:writeAnchorFile` and `cmd/circula/util.go:writeJSONFile` all use predictable `path + ".tmp"` and `os.WriteFile`, which follows an existing symlink. Manual `circula backup` accepts an operator-supplied destination and does not require a private output directory; `audit-anchor write` also accepts an arbitrary path. If an attacker can create a symlink at the corresponding temporary path in a directory the operator uses, the command can truncate/overwrite the symlink target with manifest, restore-record or anchor JSON. Impact depends on the operator process's file permissions and the chosen destination directory. Recommended fix: create an unpredictable temporary file with exclusive creation in the same directory, enforce mode 0600, sync and close it, then rename; reject unsafe destination directories where appropriate. Add symlink-preplant regression tests. Do not claim remote exploitability.
- **CRYPTO-02 — Low — backup manifest HMAC does not authenticate the ledger/integrity summary. CONFIRMED by source trace.** `manifestMAC` covers format, created time (to seconds), schema, byte count, SHA-256, and encrypted-container metadata, but omits `Integrity` and `Ledger` (`ledger_verification`) fields. An attacker who can alter only the manifest can change a recorded ledger result from failed/not-run to ok without invalidating the HMAC, while the backup file checksum still verifies. This does not alter the backed-up database; it can mislead an operator relying on the manifest as an attestation. Include the integrity and ledger summary fields in the authenticated payload, with a versioned MAC format.
- **Configuration risks, not confirmed code vulnerabilities:** backup encryption is optional unless `backup-require-encryption` is enabled or an encryption key is configured; the manual backup command warns when it writes plaintext. Audit-anchor authentication is also optional; without `audit-anchor-key`, the verifier explicitly reports the anchor as unauthenticated. Production deployments should make these choices explicit.
- No cryptographic primitive break, nonce reuse, token predictability, or TOTP replay flaw was identified in the sampled source. This is not an independent cryptographic review or a proof of safety.

### Tests and limitations

- Inspected password, MFA, token, backup-encryption, state-key and audit-anchor implementations and relevant test sources.
- Tests were not executed. No shell/Go runtime was available through the connected GitHub interface.
- No external cryptographic standards review, fuzzing, timing analysis, key-rotation drill or hardware-backed key-management test was performed.
- The symlink and manifest-MAC findings need focused reproductions and regression tests on a runtime-capable environment.

## Completed: Priority 4 — SQLite, migrations, backup/restore and recovery (source review only)

### Positive controls observed

- SQLite opens in WAL mode with foreign keys and `secure_delete=1`; the process applies a restrictive Unix umask. Database and WAL file permissions are tightened on startup.
- Writers are serialized in-process with a gate and across processes by `BEGIN IMMEDIATE`; read views run in a transaction with `query_only=ON`.
- Migrations are embedded, versioned contiguously, checked against recorded migration names, and each migration plus its schema_migrations row is applied in its own `BEGIN IMMEDIATE` transaction.
- Existing tests cover migration rollback after a mid-migration failure, upgrade from every prior schema version, idempotent reopen, foreign-key checks, concurrent writers, WAL-backed snapshots, damaged/truncated database detection, and refusing to overwrite an existing snapshot.
- Backup creation uses SQLite `VACUUM INTO`, then runs SQLite integrity and ledger/audit-chain checks on a throw-away copy before publishing the backup. Encrypted backups are decrypted again and compared with the verified plaintext before final rename.
- Restore verifies manifest policy, file hash, decryption, integrity, schema and ledger/audit chains before swapping. It takes a pre-restore backup and restores the old DB on ordinary returned errors.

### Confirmed findings

- **A3-F09 — Low — restore can resurrect credentials revoked after the backup. CONFIRMED by current source trace.** `cmdRestore` replaces the database with the historical snapshot and then only writes a restore record and `database.restored` audit events. It does not revoke restored sessions, API tokens or break-glass sessions. A session or API token revoked after the backup becomes valid again after restore. The documented post-restore password-change step revokes sessions but the CLI explicitly says API tokens are unchanged; token credentials therefore need separate revocation. Old password hashes may also be restored. Add a post-restore invalidation step or preserve a monotonic revocation ledger outside the restored database; require explicit reconciliation of API tokens and credentials.
- **RESTORE-01 — Medium (conditional crash recovery) — interrupted database swap has no startup recovery. CONFIRMED by source trace; process-kill test not run.** `swapDatabase` moves the current database and WAL files to the fixed `.pre-restore-swap` path, then renames the verified restore into place. Rollback only runs when the function returns an error. If the process or host stops after moving the old database aside but before the new database is installed, no startup code checks for `.pre-restore-swap`; normal startup may create a new empty database instead. The old database remains on disk but is not automatically restored or surfaced. Add a durable restore state marker and fail-closed startup recovery, with kill-point tests around each rename/check boundary.
- **BACKUP-01 — Low — timestamped backup names collide within one second. CONFIRMED by source trace; concurrency test not run.** `backupFileName` / `backupFileNameFor` encode time only to second precision. `snapshotWith` checks destination existence non-atomically and uses a shared `.partial` path (and, for encrypted output, a shared `.partial` ciphertext path). Two manual or scheduled/manual backups targeting the same directory within the same second can race; one can remove or fail against the other's partial file, and one backup may be lost. Use exclusive destination reservation or a unique suffix and unique temporary files; test simultaneous same-second backups.
- **A3-F10 — Low — a future-dated backup filename can suppress scheduled backups. CONFIRMED by source trace.** `listBackups` parses timestamps from filenames; `jobRunner.untilNextBackup` trusts the newest parsed timestamp and waits until that time plus the configured interval. A clock fault or future-dated file can therefore delay the next scheduled backup for years; the same file is retained as newest by pruning. Validate filename timestamps against signed manifest time and current time, and cap future skew.
- **Durability trade-off, not a confirmed defect:** the default SQLite synchronous mode is `NORMAL`. The repository documents that an OS crash or power loss can drop the newest committed WAL transactions (with their audit rows); `FULL` is an opt-in setting because the recorded benchmark showed roughly 2.7x posting latency. Deployments requiring the strongest commit durability should explicitly choose `db-synchronous=FULL`.
- **WAL erasure limitation for Priority 5:** `secure_delete=1` does not itself erase old values from WAL frames; the prior A3 review records an informational finding where a PASSIVE checkpoint left data in the WAL. Recheck checkpoint/erasure behavior in the privacy pass.
- The existing A3-F17 symlink issue also applies to `writeRestoreRecord` through `writeJSONFile`; it is documented above under CRYPTO-01.

### Migration and recovery gaps

- `migrateTo` creates the migration table before applying migrations; each actual migration and its version row are transactional. A failure rolls back the partial DDL and leaves the prior version usable. No confirmed partial-migration defect was found in source.
- The default NORMAL durability choice is documented and intentional; it remains an operator RPO decision.
- Restore is robust against ordinary returned errors and source-file TOCTOU, but not process termination during the swap. No restore kill-point test was found in the reviewed test inventory.
- Backup retention trusts file-name timestamps, not a signed monotonic sequence. The documented A3-F10 issue remains.
- Same-second collision and concurrent partial-file handling lack regression tests in the reviewed backup test file.

### Tests and limitations

- Inspected migration runner, SQLite open/transaction code, migration/upgrade/persistence tests, backup/restore CLI, snapshot writer, scheduled backup loop, restore swap, restore records and operations guide.
- Existing test sources were inspected but not run. No process-kill simulation, multi-process backup race, power-loss test, or Windows runtime test was performed.
- A3-F09 and RESTORE-01 need focused restore/revocation and crash-recovery reproductions on a runtime-capable environment.
- No application code or database was modified.

## Remaining audit areas

1. Priority 5 — privacy and data protection.
2. Priority 6 — HTTP/API/browser security.
3. Priority 7 — deployment, infrastructure and secrets.
4. Priority 8 — CI/CD and software supply chain.
5. Verify SEC-033, PR #38 requirements, PR #40 QA-042 implementation, historical branches and current CI.
6. Recheck A1-08 to A1-10 and all lower-severity ledger findings.
7. Reproduce all source-confirmed findings on a runtime-capable environment before accepting remediation.

## Exact next task

Continue with Priority 5. Trace personal data collection, storage, logs, exports, deletion/erasure, retention, backups, WAL/checkpoint behavior, account recovery and cross-user visibility. Pay particular attention to the previously reported erasure and WAL findings. Do not modify production behavior during the audit. Update this handoff after completing that area.
