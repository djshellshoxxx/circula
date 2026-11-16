# A3 audit: Circula security features, membership lifecycle, operations and CLI

Auditor: A3 (one of six parallel read-only auditors). Date: 2026-10-28.
This is an AI-assisted code audit. It is not an independent professional security review.

## 1. Scope, SHA and what was reviewed

- Repository `djshellshoxxx/circula`, detached worktree `/home/user/audit/circula-a3`,
  SHA `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f`. Toolchain go1.26.10 (auto-selected from `go.mod`).
- Read in full: `cmd/circula/*.go` (serve, config, commands, snapshot, backupcrypt, premigrate, util, statekey,
  lock, maintenance, breakglass, dualcontrol, recover, reset_mfa, erase, suspend, unlock, auditanchor, auditmirror,
  log_retention, verify_export, ops_http, tls, metrics, jobs_*); `internal/service` files for auth, password,
  throttle, tokens, totp, mfa, mfa_login, notices, recovery, breakglass, principal, erasure, setup_state, networks
  (UpdatePolicy/notice), gov_terms, join, departure, membership (non-money parts), suspend, inactivity,
  announcements, notify, reports, challenges, closure_schedule, freeze, restore_audit, verify (recording),
  dualcontrol + dualcontrol_operator (approver rules and the operator CLI path), purge, idempotency;
  `internal/store` migrate, inspect, sqlite (open pragmas), sqlite_erasure, sqlite_mfa, sqlite_recovery,
  sqlite_breakglass, sqlite_gov, sqlite_network, sqlite_person (throttle), migrations 0001/0003/0004/0006/0008/
  0011-0014; `internal/api/erasure_hook.go`; `internal/web/session.go` (only to test the key-rotation claim);
  `deploy/*`, `scripts/*` (release-gate, restore-drill*, check-hygiene, check-action-pins, package-release),
  `.github/workflows/*`, `.github/dependabot.yml`, `.dockerignore`.
- Specs/addenda read: 00, 05 (ARC-007..025), 09, 10, 11, 12 (status table), addenda SEC-007, SEC-008, SEC-011,
  SEC-011-owner-decision, SEC-012, SEC-013, SEC-018, SEC-027-keys, OPS-019-SEC-028; `docs/dev/*.md`;
  `docs/ops/backup-restore.md`, `configuration.md`, `monitoring.md`; DECISIONS.md DEC-031 and register rows.
- Prior audits checked for KNOWN cross-references: SECURITY-AUDIT.md (SA-xx), SECURITY-AUDIT-2.md (SA2-xx),
  SECURITY-AUDIT-3.md (A3-01..A3-11, E-xx, R-01), CODE-AUDIT.md, COMPLETENESS.md, RESUME-2026-10-24.md (2026-10-28).
  Note on IDs: the prior audit 3 used `A3-01..A3-11`. Findings here are `A3-F01..A3-F25` to avoid a clash.
- Out of scope (other auditors): ledger/money math and settlement amounts (A1); HTTP route authz, CSRF and
  session-cookie handling in adapters (A2). Where a finding touches those layers it is noted as a boundary.

## 2. Method

1. Read the accepted rules and prior findings first; then traced each focus area end to end (CLI -> service ->
   store -> SQL/migration).
2. For every candidate issue I wrote a throwaway proof-of-concept test that asserts the behaviour the accepted
   spec/rule asks for, so a FAIL demonstrates the finding. PoC files (left in place as evidence):
   - `/home/user/audit/circula-a3/cmd/circula/a3audit_poc_test.go`
   - `/home/user/audit/circula-a3/internal/service/a3audit_poc_test.go`
   - `/home/user/audit/circula-a3/internal/store/a3audit_poc_test.go`
   - `/home/user/audit/circula-a3/internal/web/a3audit_poc_test.go`
   Run: `go test -count=1 -run 'TestA3Audit_' -v ./internal/service ./internal/store ./internal/web ./cmd/circula`
   (all PoCs FAIL by design; the product suite is run with `-skip 'TestA3Audit_'`).
3. Two non-Go demonstrations in a scratch copy: the migration static check (`check-hygiene.sh`) against a synthetic
   destructive migration, and the release archive name scan against a synthetic zip.
4. Conformance: for each in-scope requirement that a status table, addendum or COMPLETENESS.md marks built/implemented
   I checked implementation, behaviour against acceptance criteria, and a covering test (section 5).
5. Synthetic data only; no network access other than Go module downloads.

## 3. Findings table

Severity counts: Critical 0, High 0, Medium 3, Low 13, Info 7 (23 total).
Confidence counts: CONFIRMED 19, LIKELY 4, SPECULATIVE appendix only.

| ID | Sev | Confidence | Category | Spec IDs | file:line | Summary |
|---|---|---|---|---|---|---|
| A3-F01 | Medium | CONFIRMED | bug / conformance | OPS-023, OPS-018, OPS-025, OPS-019/SEC-028 | cmd/circula/premigrate.go:56 | With `backup-encryption-key` set, the pre-migration gate runs `integrity_check` on the `.db.enc` ciphertext, so `circula migrate` and `serve` (migrate-mode=auto) can never migrate an encrypted deployment |
| A3-F02 | Medium | CONFIRMED | authz / conformance | SEC-013, ARC-009, GOV-021, SA2-04/A3-02 rule | internal/service/principal.go:25; dualcontrol.go:27 | Break-glass "no lasting changes" rule covers only ~10 methods: a break-glass admin can post adjustments, approve dual-control requests at once (A3-01 exemption, no 24 h wait), raise limits, reset member passwords, reverse trades, decide disputes |
| A3-F16 | Medium | CONFIRMED | auth / DoS | SEC-006, SEC-007 (A3-02a class) | internal/service/mfa.go:49-60, 303 | Second-factor throttle is renewable by one failure at expiry and is shared with session-only `DisableMFA`: a stolen session keeps the owner out of MFA sign-in indefinitely |
| A3-F03 | Low | CONFIRMED | auth / conformance | SEC-007, DEC-007 | internal/service/principal.go:151-159 | `require-mfa-admins` is not applied to admin powers reached through `requireDecider`: an admin without a second factor resolves disputes (reversal posted), decides challenges |
| A3-F05 | Low | CONFIRMED | privacy / conformance | SEC-018 | internal/store/sqlite_erasure.go:32-59; internal/service/erasure.go:213 | Erasure leaves the member's challenge text (`challenges.reason`) and the login in `recover: <login>` throttle rows |
| A3-F06 | Low | CONFIRMED | bug | SEC-006, SEC-008, SEC-018 | internal/store/sqlite_person.go:66-68 | `DeleteThrottleFamily` mixes Go byte length with SQLite character `substr`: per-address rows of non-ASCII logins are never removed (`circula unlock`, resets, recovery, erasure) |
| A3-F09 | Low | CONFIRMED | ops security | OPS-020, SEC-003 | cmd/circula/commands.go:223-381 | Restore resurrects sessions and API tokens revoked after the backup (and, by trace, break-glass sessions closed after it); documented remedy (password change) leaves tokens valid |
| A3-F10 | Low | CONFIRMED | durability | OPS-018, OPS-027, OPS-033 | cmd/circula/serve.go:497-507; snapshot.go:459-526 | Schedule and retention trust the timestamp in backup file names: one future-dated backup (clock fault) stops scheduled backups for years and is always retained |
| A3-F11 | Low | CONFIRMED | governance / ops | SEC-011, GOV-003, SA2-01b | internal/service/dualcontrol_operator.go:40 | `circula dual-control` fails with "conflict: duplicate record" while a policy change waits out its notice (version clash) |
| A3-F13 | Low | CONFIRMED | secrets / docs | SEC-027, OPS-010, OPS-033 | docs/ops/configuration.md:104-106; internal/web/session.go:46-50 | Docs say rotating `circula-web.key` signs everyone out; sessions are unsigned DB tokens and survive; no mass session revocation exists |
| A3-F14 | Low | CONFIRMED | conformance | OPS-022, QA-037 | scripts/check-hygiene.sh:25-27 | Migration static check misses `UPDATE entries/audit_events` and `DROP TRIGGER` (append-only protection) |
| A3-F15 | Low | CONFIRMED | conformance | OPS-016, ARC-015, DEC-031 | cmd/circula/serve.go:439; internal/service/verify.go:103-136 | Hourly scheduled verification writes a `consistency_runs` row every run (and audit events on failure/recovery), contrary to the accepted read-only rule |
| A3-F17 | Low | CONFIRMED | file safety | SEC-028, OPS-017 | cmd/circula/snapshot.go:373-377; util.go:46-50; auditanchor.go:111-116 | Manifest/record/anchor writers use a fixed `<path>.tmp` with `os.WriteFile` (follows symlinks): file clobber when backing up into a shared directory |
| A3-F18 | Low | CONFIRMED | conformance | GOV-003, GOV-019 | internal/service/membership.go:275-317 | Per-member credit-limit cuts take effect at once although the policy table requires GOV-003 notice; sidesteps the 14-day notice on default-limit cuts |
| A3-F19 | Low | LIKELY | conformance | OPS-008, SEC-006 | cmd/circula/config.go:43-81; internal/service/throttle.go:22-27 | Authentication throttling limits are constants, not operator settings (OPS-008 lists them as required settings) |
| A3-F25 | Low | CONFIRMED | integrity / privacy | SEC-018 | internal/store/sqlite_erasure.go:95-102; internal/service/validate.go:18-21 | Erasure blanks every inbox body in the network that contains the display name as a substring; a 1-character name ("e") wipes other members' notifications via self-service erasure |
| A3-F07 | Info | CONFIRMED | privacy | SEC-018, RB-003 | internal/store/sqlite.go:107-109 | Erased values remain in `-wal` frames after the normal (PASSIVE) checkpoint; the RB-003 test only passes because it forces TRUNCATE |
| A3-F12 | Info | CONFIRMED | robustness | OPS-022 | internal/store/migrate.go:100-106 | A `schema_migrations` row with version <= 0 makes every open panic (`all[v-1]`) instead of failing cleanly |
| A3-F20 | Info | CONFIRMED | supply chain | OPS-035 | scripts/package-release.sh:62-68 | Forbidden-name scan runs `tar -tzf` on `.zip` archives, so the Windows archive is never inspected |
| A3-F21 | Info | CONFIRMED | docs | SEC-006, SEC-007, SEC-008 | docs/specs/addenda/SEC-007.md:14; SEC-008.md:25; docs/dev/recovery.md:20; internal/service/mfa_login.go:57 | Docs say "15 min doubling to 24 h"; code locks a fixed 15 min (`throttleMax`) |
| A3-F22 | Info | LIKELY | ops UX | SEC-013 | internal/service/breakglass.go:309 | `break-glass grant-admin -login` is not normalised (case), unlike every other operator command |
| A3-F23 | Info | LIKELY | audit | SEC-012, SEC-013 | cmd/circula/commands.go:563; internal/service/auth.go:79-80 | `admin create-operator` is audited as the new person registering themself; nothing marks it as an operator action |
| A3-F24 | Info | LIKELY | privacy (A2 boundary) | SEC-018 | internal/api/erasure_hook.go:25-34 | Erasure drops only the erased person's in-memory artefacts; another administrator's full export made before the erasure stays downloadable for up to 24 h |

## 4. Finding details

### A3-F01 Medium CONFIRMED: migrations impossible when backups are encrypted
- Where: `cmd/circula/premigrate.go:56` (`verifyPreMigrationBackup` runs `integrityCheck(ctx, path)` on the file it just
  wrote); `backupTo` (`cmd/circula/serve.go:318-338`) writes `circula-<t>.db.enc` whenever `backup-encryption-key` is set.
  Same path from `serve` with `migrate-mode=auto` (`serve.go:121` -> `preMigrationBackup` -> `preMigrationSafeguard`).
- Scenario: operator follows the recommended secure configuration (`backup-encryption-key-file`, which also turns on
  `backup-require-encryption=auto`), then upgrades: `circula migrate` takes the encrypted pre-migration backup, opens the
  ciphertext as SQLite, gets "file is not a database (26)" and refuses to migrate. Every schema upgrade is blocked. The
  only way out is to remove the key, which also switches the encryption requirement off and leaves a plaintext
  pre-migration backup holding password hashes and all member data, contrary to OPS-019/SEC-028 and
  `docs/ops/backup-restore.md:32` ("every backup (manual, scheduled, pre-migration, pre-restore) is written encrypted").
- Evidence: `TestA3Audit_MigrateWithEncryptionKey`, `TestA3Audit_PreMigrationSafeguardEncrypted`:
  `circula: pre-migration backup .../circula-20261028T065359Z.db.enc did not verify; migrations not applied: integrity check of ...db.enc: file is not a database (26)`.
  No existing test runs `migrate` with an encryption key (premigrate_test.go, upgrade_test.go).
- Fix: in `verifyPreMigrationBackup`, when `m.Encrypted`, decrypt to a private temp file (as `cmdRestore` does), check
  the plaintext digest against the signed `PlaintextSHA256`, run `integrity_check` on the temp copy, then securely remove
  it (or rely on the in-memory decrypt proof `snapshotWith` already performs plus the ciphertext re-hash). Add a
  migrate-with-encryption regression test.
- KNOWN: no.

### A3-F02 Medium CONFIRMED: break-glass administrators can still make lasting and value-moving changes
- Where: `requireLasting` (`internal/service/principal.go:25`) is called only by role/policy/terms changes, admin
  departure, suspend member, default flag, non-self anonymise, closure schedule/cancel. Not called by
  `createAdjustment` (adjustments.go:79), `ApproveAdjustment` (dualcontrol.go, plus `if p.bgAdmin { return nil }` at
  dualcontrol.go:27 that skips the A3-01 chain check and the 24 h wait), `setCreditLimit` (membership.go:275),
  `AdminResetPassword` (recovery.go:326), `ReverseTransaction` (reversal.go:24), dispute and challenge decisions via
  `requireDecider`, `DecideJoinRequest` (join.go:278, approval posts the sign-up grant), `RestoreMember`, `SuspendNetwork`/
  `ResumeNetwork` (suspend.go:52), invitations, announcements, `ActivateNetwork`.
- Scenario: the operator opens a 1-4 h session and grants admin to member X ("ticket 42: lost admin"). X, within the
  session: credits an accomplice with an adjustment below the threshold; approves the founder's pending 50,000.00
  request seconds after the grant (a System-granted admin would have to wait 24 h and other admins get no
  `admin.granted` notice for break-glass grants); sets a member's credit limit to the platform maximum with no expiry;
  resets a member's password and keeps the temporary password, which still signs in after the session expired;
  reverses a completed trade. All of these outlive the session, which contradicts the rule the code states
  ("Such access must not leave lasting changes behind", principal.go:18-19; SA2-04, A3-02b), ARC-009 ("Read-only access
  MUST be the default") and GOV-021 ("MUST NOT change balances as part of recovery").
- Evidence: `TestA3Audit_BreakGlassLastingChanges` (4 sub-tests) and `TestA3Audit_BreakGlassApprovesImmediately`:
  `break-glass admin posted a permanent adjustment ... (amount 5000)`, `set ben's credit limit to 1125899906842624 with no expiry`,
  `reset ben's password; the temporary password still signs in after the session expired`, `reversed completed trade ...`,
  `approved the founder's 50,000.00 request seconds after the grant ...; a System-granted admin must wait 24h0m0s`.
  Existing tests (`TestA3_02_...`, `TestBreakGlassCannotMakeLastingChanges`) cover only the narrowed set.
- Fix (implements the existing rule): call `p.requireLasting()` in every write path that leaves a persistent effect
  (ledger postings and approvals, limits, credentials, membership decisions, network suspension, invitations,
  announcements), or invert to an allow-list for break-glass admins (reads, exports, closing their own session).
  Remove the `bgAdmin` early return in `requireIndependentApprover` (refuse instead).
- KNOWN: partly. SEC-013 addendum "Not done" #4 (owner asked to decide whether to narrow), SA2-04 and A3-02b narrowed
  only part; SEC-011-owner-decision.md:66 notes break-glass admins are not covered by the A3-01 check. Value-moving,
  credential and approval paths are not listed anywhere as accepted.

### A3-F16 Medium CONFIRMED: stolen session can keep the owner out of MFA sign-in
- Where: `mfaReserve` (internal/service/mfa.go:49-60) counts per person and re-locks on every failure once the count is
  at 5 (no half-allowance after expiry, unlike `readThrottle` throttle.go:74-88 which carries the A3-02a fix).
  `DisableMFA` (mfa.go:299-311) reserves on that shared key before checking the password, and needs only a session.
  `CompleteMFA` (mfa_login.go:60-75) uses the same key. `reserveAttempt` (recovery.go:47-58, recgen key) has the same
  renewal pattern.
- Scenario: an attacker holds a stolen session of an MFA user. Five garbage `DisableMFA` calls lock the key; afterwards
  one call per 15 minutes keeps it locked. The owner, who knows the password and has the authenticator, gets
  `rate-limited` at every sign-in and cannot sign in to change the password (the step that would revoke the stolen
  session). Remedies exist only via an account recovery code, an administrator reset or the operator.
  SEC-006: "MUST NOT lock out the legitimate owner permanently".
- Evidence: `TestA3Audit_StolenSessionLocksOwnerOutOfMFA`: 3 rounds, each `owner with the right password and code is
  refused: rate-limited: too many attempts`.
- Fix: apply the `readThrottle` decay/half-allowance to `mfaReserve` and `reserveAttempt`; use a separate key for the
  management calls that only need a session (disable, regenerate, confirm) so they cannot lock `CompleteMFA`; verify
  the password in `DisableMFA` (on the recgen key) before touching the shared second-factor counter.
- KNOWN: the counter sharing is documented (SEC-007 addendum: "The same counter covers confirm, disable, regenerate and
  CompleteMFA. Never a permanent lock."); the renewable lock is the A3-02a pattern, fixed for login keys only. New.

### A3-F03 Low CONFIRMED: mandatory MFA for administrators bypassed through decider powers
- Where: `requireDecider` (principal.go:151-159) accepts the admin role without checking `p.mfaMissing`; only
  `requireAdmin` (principal.go:134-146) checks it. Callers: `ResolveDispute` (disputes_decide.go:33), dispute review,
  re-review decisions, `DecideChallenge` (challenges.go:162), reviewer audit scope (audit_read.go:65).
- Scenario: `require-mfa-admins=true`; an administrator has not enrolled (or the operator removed the factor). With only
  that admin's password, a user is refused admin-only operations but resolves a dispute "upheld", which posts a reversal,
  and decides challenges. SEC-007 acceptance: "An administrator with mandatory MFA cannot complete an administrative
  action without a valid second factor".
- Evidence: `TestA3Audit_MFAForAdminsBypassViaDecider`: control `CreateInvitation` refused with `mfa-enrolment-required`;
  then `administrator without the mandatory second factor resolved dispute ... (status resolved): reversal posted`.
- Fix: in `requireDecider`, when the caller's decider standing comes from the admin role and `p.mfaMissing`, refuse
  with `mfa-enrolment-required` (reviewers without the admin role stay exempt per DEC-007 default).
- Note: an attacker holding the password could also enrol their own factor first (TOFU), so the marginal gain is
  stealth; hence Low. KNOWN: no (addendum only exempts reviewers).

### A3-F05 Low CONFIRMED: erasure leftovers (challenge text, login in recovery throttle rows)
- Where: `RedactDisputeStatements` (sqlite_erasure.go:32-59) clears disputes, responses, re-review reasons and report
  notes but not `challenges.reason` (migration 0014, member-written). `erasePersonIfUnused` deletes only the
  `DeleteThrottleFamily(person.Login)` rows (erasure.go:213), not `recover: <login>` and its per-address rows.
- Scenario: an administrator removes a member; the member files a removal challenge ("... call me 555-0100") and then
  uses "erase my data"; the challenge text stays. A failed `/recover` attempt leaves `recover: ana@example.org ...` rows
  until the purge job (24 h after the lock ended). SEC-018 acceptance: no match for name/email/phone in personal-data
  fields after pseudonymisation; docs/dev/erasure.md asks new free-text columns to be added to erasure.
- Evidence: `TestA3Audit_ErasureLeftovers`: `member-written challenge text survives erasure in [challenges.reason]`,
  `erased login survives in [login_throttle.login]`.
- Fix: clear `challenges.reason` for `member_id = target` (and treat a `requested` challenge as a blocker or close it);
  delete the `recoverKey(login)` family too; extend `TestAnonymiseMemberKeepsLedger` with a challenge and a recovery
  attempt.
- KNOWN: no (SA2-05/A3-06 fixed other leftovers before migration 0014 existed).

### A3-F06 Low CONFIRMED: throttle cleanup misses non-ASCII logins
- Where: `DeleteThrottleFamily` (sqlite_person.go:66-68): `substr(login, 1, ?) = ?` with `len(key)+1` bytes. SQLite
  `substr` on TEXT counts characters, so for a login containing a multi-byte character the prefix never matches.
- Scenario: `józef01` is locked out from his address; the operator runs `circula unlock -login józef01` (the documented
  SA2-10 remedy); the per-address lock row survives and he is still refused. Same miss in admin/operator resets
  (`resetPerson`), successful `RecoverPassword`, `OperatorUnlockLogin`, erasure (login string kept in those rows).
- Evidence: `TestA3Audit_UnlockNonASCIILogin`: `after circula unlock (cleared 1 row(s)) the person is still locked out:
  rate-limited`; ASCII control passes.
- Fix: compare bytes (`substr(CAST(login AS BLOB),1,?) = CAST(? AS BLOB)`) or use `login >= key||' ' AND login < key||'!'`.
- KNOWN: no.

### A3-F09 Low CONFIRMED: restore brings back revoked sessions and tokens
- Where: `cmdRestore` (commands.go:223-381) swaps in the backup and writes `database.restored`; nothing revokes the
  sessions, API tokens or break-glass sessions contained in it.
- Scenario: a member logs out or revokes a leaked API token on Tuesday; the operator restores Monday's backup after an
  incident; the logged-out session and the leaked token work again (up to the 90-day absolute lifetime). By the same
  mechanism (trace, LIKELY) a break-glass session that was open at backup time and closed later is live again until its
  original expiry, granting admin without a new audit event. The documented remedy ("make everyone change their
  password; a password change revokes the person's sessions", backup-restore.md:150,158) does not revoke API tokens
  (auth.go:213).
- Evidence: `TestA3Audit_RestoreResurrectsRevokedSession`: `session revoked after the backup authenticates again after
  restore`.
- Fix: in the restore swap (on the verified copy, before it goes live) revoke all sessions and API tokens and close open
  break-glass sessions, recording it in the `database.restored` event; or make that the default with an explicit flag
  to keep them.
- KNOWN: partly. SA-07 proposed "revoke all on restore"; it was not implemented and is not tracked as open.

### A3-F10 Low CONFIRMED: future-dated backup name stops the schedule
- Where: `untilNextBackup` (serve.go:497-507) waits until newest-by-name + interval; `listBackups`/`pruneBackups`
  (snapshot.go:459-526) sort and retain by the name's timestamp.
- Scenario: NTP misconfiguration sets the clock to 2036 for one backup run; after correction the next scheduled backup
  is 10 years away, retention always keeps the bogus file and prunes real ones earlier. Nothing logs the anomaly
  (OPS-033 "clock error ... the anomaly is logged"); only the OPS-015 "no backup in 26 h" alert, if configured, notices.
- Evidence: `TestA3Audit_FutureDatedBackupStopsSchedule`: `next scheduled backup waits 87696h0m0s (about 3653 days)`;
  with keep-daily 2 only 1 real backup is kept.
- Fix: ignore (and log/metric) entries dated more than a tolerance after now when scheduling; base retention on
  manifest time bounded by now.
- KNOWN: no.

### A3-F11 Low CONFIRMED: operator dual-control command blocked during a notice period
- Where: `OperatorSetDualControl` appends `net.PolicyVersion + 1` (dualcontrol_operator.go:40) where `PolicyVersion` is
  the in-force version; `UpdatePolicy` stores a scheduled change as `latest+1` (networks.go:266) with a future
  `effective_at`.
- Scenario: an adverse policy change is waiting out its 14-day notice; the second administrator leaves and the operator
  must loosen/disable dual control (SA2-01b: only the operator can): the command fails with `conflict: duplicate
  record`. If versions did not collide, an operator version would silently supersede the scheduled one.
- Evidence: `TestA3Audit_OperatorDualControlBlockedByScheduledChange`.
- Fix: append at `LatestPolicyVersion()+1` and apply the threshold to the pending version as well (or refuse with a
  message to cancel the scheduled change), with a test.
- KNOWN: no.

### A3-F13 Low CONFIRMED: web key rotation does not sign anyone out
- Where: docs/ops/configuration.md:104-106 ("delete the file ... people sign in again"); addendum SEC-027-keys.md:13.
  The session cookie is the raw random token (web/session.go:46-50) checked only against the database
  (service/auth.go:184-205); the key signs CSRF tokens and the `cm_sstart` hint.
- Scenario: suspected secret exposure; the operator rotates `circula-web.key` as documented, believing every session
  ended; stolen session cookies keep working. There is no command to revoke all sessions or tokens. SEC-027/OPS-010:
  "Rotating a session key MUST invalidate existing sessions".
- Evidence: `TestA3Audit_WebKeyRotationKeepsSessions`: dashboard 200 with the old cookie on a handler built with a new key.
- Fix: add an audited `circula revoke-sessions [-tokens]` (or a session-epoch row checked by `Authenticate`) and correct
  configuration.md and the addendum.
- KNOWN: partly. SEC-027 status "session key rotation not implemented" and monitoring.md:79 say so; the configuration
  guide's claim is new and contradicts them.

### A3-F14 Low CONFIRMED: migration static check misses ledger/audit updates and trigger drops
- Where: scripts/check-hygiene.sh:25-27 flags only `DROP TABLE|COLUMN` and `DELETE FROM`.
- Scenario: a migration containing `DROP TRIGGER audit_no_update; UPDATE audit_events ...; UPDATE entries ...` passes.
  OPS-022 acceptance: "Migration files contain no statement that updates or deletes entries or audit events (static check)".
- Evidence: scratch copy with synthetic `0016_evil.sql` -> `hygiene checks: ok`, exit 0.
  (Runtime tests such as `TestUpgradeFromMigration0001` would still catch a dropped audit trigger.)
- Fix: also flag `UPDATE\s+(entries|audit_events|audit_subjects|...)` and `DROP\s+TRIGGER` without `allow-destructive`.
- KNOWN: QA-037 is "partial"; this specific gap is new.

### A3-F15 Low CONFIRMED: scheduled verification writes to the database
- Where: the hourly job calls `verifyNetworks(..., record=true)` (serve.go:439) -> `RecordScheduledConsistencyRun`
  (verify.go:103-136): a `consistency_runs` row every run per network, and a `ledger.verified` audit event whenever a run
  fails or the previous run had failed (verify.go:128-133).
- Mismatch: DEC-031 (accepted; DECISIONS.md:54-66: "Startup, hourly, migrate, restore and backup-verify callers use the
  read-only path"), OPS-016 ("Scheduled and other automatic verification MUST write no database state", acceptance
  "including scheduled verification, leaves a checksum of the database unchanged") and ARC-015. Both marked built.
- Evidence: the repository's own `TestScheduledVerifyRecordsWithoutFlooding` (cmd/circula/scheduled_verify_test.go:10-44)
  passes and asserts 4 rows for 4 runs.
- Fix (follows the accepted spec): `record=false` in `jobRunner.verify`; report through metrics/logs only; update the test.
- KNOWN: no (SVC-01 fixed the CLI/startup paths; the job was changed later).

### A3-F17 Low CONFIRMED: symlink-following temp files
- Where: `writeManifest` (snapshot.go:373-377), `writeJSONFile` (util.go:46-50), `writeAnchorFile` (auditanchor.go:111-116)
  write `<path>.tmp` via `os.WriteFile` (O_TRUNC, follows symlinks), then rename.
- Scenario: `circula backup /shared/backups/` (another local user can write there): that user pre-creates
  `circula-<next UTC second>.db.manifest.json.tmp -> /path/owned/by/operator`; the operator's run overwrites the target
  with manifest JSON. Scheduled backups are safe (private 0700 dir).
- Evidence: `TestA3Audit_ManifestTmpFollowsSymlink`: the victim file now holds `{"format": 1, ...`.
- Fix: `os.CreateTemp(dir, ".manifest-*")` (O_EXCL) then `os.Rename`.
- KNOWN: no (SA3 verified temp dirs/`O_EXCL` for the backup body only).

### A3-F18 Low CONFIRMED: per-member limit cuts skip the GOV-003 notice
- Where: `setCreditLimit` (membership.go:275-317) applies at once; notice logic exists only in `UpdatePolicy`.
- Spec: 10 section 2 policy table: "Per-member credit limit override | ... | Prospective; notice required (GOV-003)";
  GOV-003 covers "changes that reduce a member's available credit".
- Scenario: the 14-day notice on lowering the default limit (enforced) is sidestepped by setting each member's override
  to 0, effective immediately.
- Evidence: `TestA3Audit_PerMemberLimitCutSkipsNotice`: immediate default cut refused without reason; per-member cut
  takes effect, next payment refused `insufficient-available-credit`.
- Fix: schedule reductions with an `effective_at` after the notice (or require a notice reason shown in the log for an
  immediate cut). Spec tension noted in section 6 (GOV-016/GOV-019 assume immediate reductions).
- KNOWN: no.

### A3-F19 Low LIKELY: throttling limits not configurable
- `settingDefs` (config.go:43-81) has no throttle settings; limits are constants (throttle.go:22-27). OPS-008 requires
  "Authentication throttling limits (bounded by ... SEC-006)" as a setting; SEC-006 "Limits are configurable by the
  platform operator within bounds". OPS-008 is marked built. Fix: add bounded settings passed into `service.Config`.

### A3-F25 Low CONFIRMED: erasure over-redacts other members' notifications
- Where: `RedactNotificationNames` (sqlite_erasure.go:95-102) blanks any body in the network with `instr(body, name) > 0`;
  display names may be 1 character (validate.go:18-21).
- Scenario: a member joins as "e", leaves and uses self-service "erase my data": every notification body in the network
  containing "e" (other members' inbox entries) is blanked. Integrity of other members' data altered by a self-service
  action.
- Evidence: `TestA3Audit_ErasureOverRedactsOthersInbox`: `blanked 2 of 2 notification bodies that belong to other members`.
- Fix: record the subject member id on notifications that name a member (or store bodies as templates + ids) and
  redact by id; at minimum match whole names in the known body templates.
- KNOWN: no.

### Info findings
- A3-F07: `TestA3Audit_ErasureWALRemnant` (`erased display name is still readable in circula.db-wal`). The main file is
  clean after a PASSIVE checkpoint; the WAL keeps stale frames until overwritten. Fix: run `PRAGMA wal_checkpoint(TRUNCATE)`
  after erasure (best effort) and/or set `journal_size_limit`; document. Host access is trusted (TA-3), hence Info.
- A3-F12: `TestA3Audit_MigrationVersionZeroPanics` (`runtime error: index out of range [-1]`). Fix: refuse `v < 1`.
- A3-F20: `tar -tzf` on a zip exits 2 with no output, so `grep` never matches (scratch check). Fix: `unzip -Z1`.
- A3-F21: documentation drift; existing `TestMFAThrottle` shows sign-in works after 16 minutes (no doubling).
- A3-F22: `GrantBreakGlassAdmin` uses `strings.TrimSpace(in.Login)` without `normalizeLogin`; mixed-case input is "not found".
- A3-F23: operator-created persons are indistinguishable in the platform audit trail from self-registration.
- A3-F24: in-memory artefacts are keyed by creator; erasure of member M does not drop admin A's earlier full export.

## 5. Conformance table (in-scope requirements claimed built/implemented)

Claim sources: spec status tables (historical snapshot), addenda "Status: implemented", COMPLETENESS.md "built".

| Spec ID | Claimed | Verdict | Code | Test / evidence |
|---|---|---|---|---|
| SEC-007 MFA | addendum implemented; COMPLETENESS built | partial (A3-F03, A3-F16) | service/totp.go, mfa.go, mfa_login.go | mfa_test.go (TestTOTPRFC6238Vectors, TestMatchTOTPSkewAndReplay, TestRecoveryCodes, TestMFAThrottle, TestRequireMFAForAdmins); TOTP window +-1, replay guard, recovery single use + hashed OK |
| SEC-008 recovery | addendum implemented (deviations) | OK within recorded deviations; A3-F06 | service/recovery.go | recovery_test.go (TestRecoverReplay, TestRecoverCodeRace, TestRecoverRevokesEverything, TestAdminResetGuards) |
| SEC-010 role separation | 09 built | OK (self-approval, last admin, departed admin); operator self-grant not enforceable (spec gap) | membership.go, dualcontrol.go | membership_test.go, dualcontrol_test.go |
| SEC-011 operator CLI part | addendum implemented | OK, audited; A3-F11; A3-04 open | dualcontrol_operator.go, cmd dualcontrol.go | TestPolicyCannotLoosenDualControl |
| SEC-012 audit + notices | 09 built | OK | audit.go, notify.go, notices.go | atomicity_test TestEveryCommandWritesAuditEvent, admin_notice_test |
| SEC-013 operator access | addendum partial; COMPLETENESS built | partial (A3-F02; second approver, 24 h notice, CLI gate open) | breakglass.go, cmd breakglass.go | breakglass_test.go, TestA3_02_* |
| SEC-018 erasure | addendum implemented | partial (A3-F05, A3-F25, A3-F07, A3-F24) | erasure.go, sqlite_erasure.go | erasure_test.go TestAnonymiseMemberKeepsLedger |
| SEC-022 transport | 09 built | OK | config.go checkTransport, serve.go:228 | TestPlainHTTPRefusedUnlessExplicit |
| SEC-027 secrets | 09 partial; SEC-027-keys implemented | partial (A3-F13; rotation; key owner unchecked E-09) | config.go loadSecrets, statekey.go | TestSecretsLoadingAndRedaction, statekey_test.go |
| SEC-028 / OPS-019 backup security | addendum implemented; COMPLETENESS built (opt-in) | partial: encryption opt-in (plaintext default, warning); A3-F01; A3-F09 | backupcrypt.go, snapshot.go, commands.go | backupcrypt_test.go, TestEncryptedBackupAndRestoreIntegration |
| SEC-006 throttling | 09 partial | partial (A3-F16, A3-F06, A3-F19) | throttle.go, mfa.go | hardening_test TestLoginThrottleIsAtomicUnderConcurrency |
| ARC-009 break-glass | COMPLETENESS built; 05 not built | partial (read-only default not enforced: A3-F02) | breakglass.go | breakglass_test.go |
| ARC-011 single writer | 05 built | OK | cmd lock*.go | TestLockRefusesSecondHolder, TestOperatorCommandsTakeTheLock |
| ARC-012 idempotent jobs | 05 built | OK (A3-F10 scheduling caveat) | serve.go jobs | jobs_regress_test.go |
| ARC-015 invariant checks | 05 built | mismatch (A3-F15) | serve.go:439, verify.go | TestScheduledVerifyRecordsWithoutFlooding asserts writes |
| ARC-023 config/secrets | 05 built | OK | config.go | TestPrecedence; gitleaks run clean |
| GOV-002 policy record | 10 built | OK | networks.go UpdatePolicy, gov_terms.go | TestPolicyNoticePeriod |
| GOV-003 notice | 10 built | partial (A3-F18) | networks.go noticeDays, domain/terms.go Adverse | TestPolicyNoticePeriod, TestPolicyNoticeCancel |
| GOV-008 join modes | 10 built | OK (invitation 256-bit, hashed, single use, 7 d default, 90 d max) | join.go | TestJoinModes, TestInvitations |
| GOV-009 terms | 10 built | OK | gov_terms.go, join.go | TestTermsGovernance |
| GOV-012 inactivity | 10 built | OK | inactivity.go | TestInactivityRuleFlagsAndNeverRemoves |
| GOV-016 challenges | 10 built | OK for decisions; erasure gap A3-F05 | challenges.go | TestChallengeRemovalAndLimit |
| GOV-020 admin continuity | 10 built | OK (break-glass admin does not lift admin-required; see GOV-021 gap) | closure_schedule.go lifecycle | TestAdminRequiredReadOnly |
| GOV-023 / GOV-024 suspension | 10 built | OK | suspend.go, cmd suspend.go | TestAdminSuspensionRefusesPostingsKeepsReads, TestPlatformSuspension |
| GOV-025 closure | 10 built | OK (notice >= 30 d, requireLasting); erasure afterwards inherits A3-F05 | closure_schedule.go | closure_test.go |
| OPS-006 / OPS-007 config | 11 built | OK | config.go | TestPrecedence, TestInvalidValuesNameEachSetting, TestCheckConfigDoesNotCreateDatabase |
| OPS-008 settings | 11 built | partial (A3-F19; mail settings absent, DEC-009 open) | config.go | TestRequiredSettingsAreNamed |
| OPS-011 storage | 11 built | OK (WAL, FK, busy, synchronous setting, secure_delete) | store/sqlite.go, inspect.go | TestSynchronousOption, CheckPragmas |
| OPS-013 health/metrics | 11 built | OK | cmd ops_http.go | TestHealthzReportsCoarseStatusOnly, TestMetricsGate |
| OPS-016 verify | 11 built | mismatch (A3-F15) | verify.go, serve.go | as above |
| OPS-017 backup | 11 built | OK | snapshot.go | TestSnapshotWritesVerifiedManifestAndRestores |
| OPS-018 schedule/retention | 11 built | partial (A3-F01 pre-migration under encryption; A3-F10) | serve.go, snapshot.go | TestJobScheduledBackupAndRetention |
| OPS-020 restore | 11 built | OK vs acceptance (TOCTOU A3-01b fixed, re-hash/plaintext digest); A3-F09 hardening | commands.go cmdRestore | TestRestoreSignedSwapDuringPreRestoreBackup, TestRestoreRefusesBackupFailingEmbeddedChecks |
| OPS-021 drills | 11 built | partial (records only for empty databases, docs/qa/evidence) | scripts/restore-drill*.sh | drill-20261021T*.json |
| OPS-022 migrations | 11 built | partial (A3-F14, A3-F12) | store/migrate.go, check-hygiene.sh | TestMigrationsIdempotentAndVersioned |
| OPS-023 migration safety | COMPLETENESS built; 11 partial | mismatch (A3-F01; invariant check runs after commit, not "refused before commit") | premigrate.go, commands.go cmdMigrate | TestPreMigrationSafeguardVerifiesBackup (plaintext only) |
| OPS-024 maintenance/upgrade | COMPLETENESS built | OK | maintenance.go | maintenance_test.go |
| OPS-025 migrate mode | 11 built | partial (auto mode fails under encryption, A3-F01) | serve.go:116-125 | TestServeRefusesPendingMigrationsInExplicitMode |
| OPS-027 jobs | 11 built | OK (A3-F10 caveat) | serve.go | TestJobRunStartsAllJobsAndStops |
| OPS-032 logs | 11 built | OK (route templates, no personal data found in log calls) | serve.go, ops_http.go | — |
| OPS-035 packaging | 11 partial | partial (A3-F20; no signing, DEC-027) | package-release.sh | — |
| SEC-029 / CI hygiene | 09 partial | OK for pins/permissions (all actions on 40-hex SHAs, least privilege, release cache off, tag ancestry) | .github/workflows, check-action-pins.sh | check-action-pins.sh PASS |

## 6. Spec gaps (not bugs)

1. SEC-010 "operator staff roles MUST NOT grant themselves a network role": the product has no operator identity, and
   `break-glass grant-admin` accepts any login, including the operator's own member account.
2. GOV-021: no path to appoint a permanent administrator when none remains (break-glass admins cannot `AssignRole`,
   and there is no operator CLI for a System grant). Owner decision; GOV-021 is not claimed built.
3. GOV-003 versus GOV-016/GOV-019: the policy table requires notice for per-member overrides, while GOV-016/019 read as
   if reductions are immediate and then challengeable (A3-F18).
4. SEC-007/DEC-007: whether an administrator acting in the reviewer role is an "administrative action" for mandatory
   MFA (A3-F03); DEC-007 itself is open and `require-mfa-admins` defaults to false.
5. OPS-020 is silent on credentials contained in a restored backup (A3-F09).
6. SEC-013(d)/ARC-009: the set of actions allowed under break-glass is not enumerated; the code's own rule is "no
   lasting changes" (A3-F02).
7. OPS-033 clock errors: no requirement covers backup naming/scheduling under clock faults (A3-F10).
8. SEC-018 "deferral recorded" and retention clock remain owner decisions (addendum items 3 and 5).

## 7. Not reviewed / coverage limits

- Not reviewed: HTTP routing/authz/CSRF/cookies (A2), ledger and settlement math (A1); web/API MFA and recovery page
  flows except as service callers; e2e/browser behaviour; Windows ACLs; real TLS or reverse proxy; load.
- `govulncheck` could not run: `vuln.go.dev` returns 403 through the proxy (same as prior audits). Dependency
  vulnerabilities NOT assessed. Note: with `GOTOOLCHAIN=local` the local toolchain is go1.24.7 and govulncheck cannot load
  the module; it needs `GOTOOLCHAIN=go1.26.10`.
- `golangci-lint` not run. `scripts/restore-drill*.sh`, `smoke-deploy.sh`, `release-gate.sh` not executed.
- KNOWN items re-checked and still open at this SHA (not counted as findings): A3-04/A3-04b (operator CLI commands do
  not call `RequireBreakGlass`; only `GrantBreakGlassAdmin` does), A3-07, A3-10, A3-11, SA2-12 (SEC-011 scope),
  SA2-21 (no second break-glass approver, no 24 h notice), E-02 (restore opens unsigned DB without `trusted_schema=OFF`),
  E-06 (no release signing), E-09 (secret/key file owner not checked), E-10 (`break-glass list` prints reasons raw;
  `requireReason` allows control characters), R-01 (50-bit MFA recovery codes under unkeyed SHA-256; seal key =
  SHA-256 of `mfa-secret`), SA2-11 rollback residual (documented in backup-restore.md), tracked ELF `circula` at repo root
  (built from 7f1e55cc93fc, stale; not in `.dockerignore`; not shipped by release scripts), Docker base images by tag.
- KNOWN items verified fixed at this SHA: A3-01b (restore re-hash / plaintext digest), A3-02a for IP-keyed login and
  recovery rows (not for MFA/recgen, see A3-F16), A3-03a, A3-04a, A3-05a, A3-05b, A3-06, SA2-01b, SA2-03, SA2-04 (partial,
  A3-F02), SA2-05, SA2-11, SA2-13 (resets and recovery delete challenges), SA2-16, SA2-19, SA2-20, SA-06, SA-07, SA-12,
  SA-14, CMD-01, CMD-02.

### Appendix: SPECULATIVE
- `ChangePassword` does not delete pending MFA challenges (5-minute window for a phished password + real-time TOTP relay).
- `deploy/compose.yaml` defaults `trusted-proxies` to `172.16.0.0/12`: any local process or container on a Docker
  network can spoof `X-Forwarded-For` (A2 boundary; remote clients behind the proxy cannot).
- `cancelOpenProposals` caps at 500 per departure (A1 boundary).
- `verifyPassword` accepts stored Argon2 memory up to 2 GiB (DB-writer DoS only).
- Manifest HMAC does not cover `ledger_verification`, `integrity_check`, `reason`, `app_version` (re-verified on restore).

## 8. Commands run

| Command | Result |
|---|---|
| `go version` (worktree) | go1.26.10 linux/amd64 (auto toolchain) |
| `go test ./internal/... ./cmd/...` (baseline, before PoCs) | PASS, all packages |
| `CGO_ENABLED=1 go test -race -count=1 -skip 'TestA3Audit_' ./internal/... ./cmd/...` | PASS, all packages (api 357 s, service 512 s, store 117 s, web 290 s, cmd 302 s); no race reported |
| `go test -count=1 -run 'TestA3Audit_' -v ./internal/service ./internal/store ./internal/web ./cmd/circula` | FAIL by design: all 17 PoC tests (plus 4 sub-tests) fail, reproducing the CONFIRMED findings |
| `go vet ./...` | PASS |
| `gofmt -l .` | clean |
| `scripts/check-hygiene.sh` | PASS |
| `scripts/check-action-pins.sh` | PASS |
| `scripts/check-hygiene.sh` in scratch copy + synthetic `0016_evil.sql` | PASS (demonstrates A3-F14) |
| synthetic zip + `tar -tzf` scan | scan misses zip content, tar exit 2 (A3-F20) |
| `go run github.com/zricethezav/gitleaks/v8@v8.28.0 dir . --no-banner --redact` | PASS, "no leaks found" (3.8 MB) |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` | BLOCKED (vuln.go.dev 403; toolchain note in section 7) |
| `go version -m circula` | tracked binary built from 7f1e55cc93fc with go1.26.10, CGO on |
