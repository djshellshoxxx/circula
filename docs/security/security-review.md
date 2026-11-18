# Security review: beta consolidation

Date: 2026-11-11. Revision: branch `claude/circula-beta-completion-1npjw6`. The final commit SHA is in
[../releases/beta-readiness-report.md](../releases/beta-readiness-report.md).

This is an AI-assisted targeted review. It is not an independent professional audit, and it does not replace SEC-033
(the independent security review owner gate). It builds on the earlier reviews and does not repeat them:
[SECURITY-AUDIT.md](../audit/SECURITY-AUDIT.md), [SECURITY-AUDIT-2.md](../audit/SECURITY-AUDIT-2.md),
[SECURITY-AUDIT-3.md](../audit/SECURITY-AUDIT-3.md), the 2026-10-28 code audit in [../audit/2026-10-28/](../audit/2026-10-28/README.md)
with its [remediation](../audit/2026-10-28/REMEDIATION-2026-11-04.md), and the ChatGPT source-pinned audit ledger
[../audits/CHATGPT_RANDOM_AUDIT_LEDGER.md](../audits/CHATGPT_RANDOM_AUDIT_LEDGER.md).

## Assets, trust boundaries, adversaries

- **Assets:** ledger integrity (balances, entries, the history of each transaction), members' credit capacity,
  dispute rights, personal data, credentials and sessions, backups.
- **Boundaries:** browser ↔ HTML handlers (cookie session plus CSRF); API client ↔ JSON API (scoped bearer tokens plus
  idempotency keys); operator ↔ CLI (local file access, no network auth); process ↔ SQLite file; backup file ↔
  restore.
- **Adversaries:** an outside attacker without an account; a member acting against other members (IDOR, replay,
  racing the credit limit); a single administrator acting alone (dual control, break-glass, policy changes against
  members); someone holding a stolen cookie or token; anyone with read access to a backup.

## Findings from this review

| ID | Severity | Area | Affected code | Preconditions | Status | Remediation | Verification |
|---|---|---|---|---|---|---|---|
| BR-01 | Medium | Governance / financial fairness | `service/disputes.go` `OpenDispute` (before `e127db8`) | An administrator changes `dispute_window_days` | **Fixed** | The deadline is fixed at completion and stored immutably (DOM-017/DAT-051, migration 0016, triggers) | Before the fix, an administrator could shorten the window and strip members of the right to dispute trades already completed, or lengthen it and revive expired disputes. `TestDisputeDeadlineFixedAtCompletion` fails against the old logic and passes now. Backfill tests in `internal/store` |
| BR-02 | Low | API concurrency control | `api/h_policy.go` `policyETag` (before `addc229`) | Two administrators edit policy at once | **Fixed** | The validator includes the pending policy version | The stale `If-Match` reached the service; only the one-scheduled-change rule stopped a lost update (409 instead of 412). `TestPendingCurrencyNamePolicyPatchChangesETag`; contract walk updated |
| BR-03 | Low | Release supply chain | `.github/workflows/release.yml` | A tag is pushed | **Fixed** | The release workflow runs `check-release-notes.sh` and `verify-release.sh` (the scan included) | YAML parses; `check-action-pins.sh` ok; not yet exercised on a real tag |
| BR-04 | Low | Privacy | `api/erasure_hook.go`, `api/jobs.go` `dropErased` (before branch `wk/export`) | An admin made a `full` export, then a member is erased | **Fixed** | Member erasure drops every admin-scope (`full`, `ledger`, `members`, `audit`) export artefact of the network and closes its spool file; an admin-scope export still running across the erasure is not stored (503, retry). The person's own exports and every affected network member archive are dropped too, since counterpart names/notes are personal data. Active downloads are revoked; unrelated networks remain isolated | `TestErasureDropsNetworkExports` (`internal/api/export_spool_test.go`; erases a member through `EraseMyData`) fails without the fix (all four admin-scope artefacts survive; the running export is stored) and passes now; `TestA3_06_ErasureDropsReplayCacheAndExports` still passes |
| BR-05 | Low | Enumeration | sign-in and recovery; `service/auth.go` `LoginStep`, `service/password.go` | Timing measurement | **Fixed** (residual noted) | Unknown, malformed and wrong-password sign-ins already ran one Argon2id evaluation (dummy hash). The gap: an account whose stored hash cannot be evaluated (the erased marker, a damaged row) skipped Argon2id, so it answered faster. `verifyLoginPassword` now runs the dummy hash for it too. Recovery refusals (unknown login, wrong, used or malformed code) run no Argon2id on any path (A3-04) | Deterministic: an Argon2id observer seam counts evaluations and their parameters, and a store wrapper records the statements in order. `TestLoginRefusalsDoEqualWork` (fails without the fix) and `TestRecoveryRefusalsDoEqualWork`. Statistical: `CIRCULA_TIMING_TEST=1 go test ./internal/service -run TestLoginTimingStatistical` (median difference 2.9% with default parameters in this container; provisional 10% limit, threshold pending DEC-023 / QA-040). Residual: a hash stored with older Argon2id parameters costs what those parameters cost; no rehash on sign-in yet |
| BR-06 | Medium | Resource exhaustion | export spool (`api/jobs.go`, `api/export_spool.go`, `api/h_export.go`), branch `wk/export` | Parallel large exports | **Fixed** | (a) A running export holds its slot (5 per person, 50 in total, new cap of 4 running) and reserves every archive chunk against the 64 MiB cap before writing it; over the cap it stops with 503 and frees its file and reservation. (b) New setting `export-spool-dir` / `CIRCULA_EXPORT_SPOOL_DIR`, default `<state-dir>/export-spool`, created `0700`; `serve` refuses to start when it is unusable; at run time an unusable directory answers 503 and logs the path. The in-memory fallback is removed. (c) The manifest field `reconciliation` is documented in `docs/api/export-schema.md`; the openapi `createExport` text is updated (the manifest is not an API schema) | `TestParallelExportWritesStayWithinCap` (disk use measured from the spool files) and `TestParallelLargeExportsStayBounded` (8 parallel 8.8 MiB exports under a 13.1 MiB cap: peak held 13.1 MiB, heap growth 5.1 MiB) fail without the reservation; `TestRunningExportsHoldTheirSlot`, `TestExportRefusedLeavesNoSpool`, `TestExportRefusedWithoutSpoolDir`, `TestExportSpoolPreparedOrStartRefused`, `TestExportSpoolDirSetting`, `TestExportManifestFieldsDocumented`; contract test passes |

## Checks run against the high-risk classes

| Class | Result | Evidence |
|---|---|---|
| Authentication, session fixation, logout | Pass | `TestLoginDoesNotAdoptPlantedSessionCookie`, `TestLogoutInvalidatesReplayedCookie` (new); `TestLoginSessionLifecycle` |
| Cookies (HttpOnly, SameSite, Secure, `__Host-`), HSTS | Pass | `TestHeadersAndCookies`, `TestSecureCookiesAndHSTS`, `cookie_policy_adapter_test.go` (merged from PR #28) |
| CSRF | Pass | `TestCSRFRejections`, `TestCookieSessionNeedsCSRF` |
| XSS | Pass (automated) | `html/template` everywhere; the only `template.HTML` conversion formats times (`render.go:138`), so no user text reaches the output unescaped |
| SQL injection | Pass | Parameterized queries throughout `internal/store`; table names in tests only; fuzzed API bodies (`FuzzAPICommandBody`) |
| Authorization, IDOR | Pass | `TestAuthorizationMatrix` (10 actor classes × operations); non-party gets 404 in service, web and API |
| Race on the credit limit | Pass | `BEGIN IMMEDIATE` plus process write gate; `TestParallelAcceptsNeverBreachCreditLimit` under `-race` |
| Replay, duplicate submission | Pass | Idempotency keys with body hash; replay of reversals and adjustments posts nothing (new test) |
| Partial failure | Pass | Fault injection at 8 write points each for accept, reversal and adjustment |
| Tampering | Detected | Append-only triggers; `VerifyLedger` reports a tampered entry when triggers are bypassed (new test); hash-chained audit |
| Privileged operations | Pass, one owner gate | Dual control (SEC-011), break-glass time limits, audited adjustments; A3-F02 (break-glass scope) waits on DEC-014 |
| Secrets in logs, configuration | Pass (existing) | Secrets only from env or a 0600 file; `cmd_secrets.go`; `TestConfig*` |
| HTTP security headers | Pass | `TestHeadersAndCookies` |
| Resource limits | Pass (existing) | Rate limits per address and /64; body size limits; `network-join-limit` |
| Dependencies | **Not run here** | `govulncheck` needs `vuln.go.dev`, which this environment's proxy refuses (HTTP 403). CI runs it on every push (`ci.yml`). Modules are pinned in `go.sum` and verified with `go mod verify` by `package-release.sh` |
| Static analysis | Pass | `go vet ./...`; `golangci-lint` v2.15.0 (the CI pin): 0 issues, after two fixes in integrated test code |

## Earlier findings still open

- 2026-10-28 audit: 6 Medium findings. Five are remediated with regression tests (A1-01, A1-02/A1-11, A1-03,
  A3-F01, A3-F16). A3-F02 (break-glass scope) waits on owner decision DEC-014.
- The 31 Low and 18 Info findings of that audit were not part of the 2026-11-04 remediation. Their status is not
  re-verified here.
- Owner questions listed in `HANDOFF.md`: GOV-011, DOM-031, SEC-032 and SEC-006.

## Consolidation re-check (2026-11-18)

AI-assisted re-check during the PR consolidation of `main@b53f757`
([record](../plans/PR-CONSOLIDATION-2026-11-17.md)). It is not an independent review and does not
satisfy SEC-033. Scope: the findings in the draft PR #42 audit notes (archived as
[docs/audits/CHATGPT_SECURITY_AUDIT_HANDOFF_2026-11-15.md](../audits/CHATGPT_SECURITY_AUDIT_HANDOFF_2026-11-15.md)),
and the squash-merged fix PRs #45, #51, #52, #53 and #54.

- Fix PRs: each fix is on `main` and its regression test passes. Removing each fix on a scratch copy turned its
  test red. #53's query was later replaced by #56 (`TestNotificationErasureUsesResourceRelations`).
- #51 and #52 had helper-level tests only. Service entry-point regressions are added in
  `internal/service/ledger_auth_integration_regression_test.go`
  (`TestResolveDisputeRefusesReversalWhenAdministratorRequired`, `TestDeciderEntryPointsRequireAdministratorMFA`).
  Both fail when the fix is removed.
- PR #42 findings fixed on `main`, each with a named regression test: A1-01 to A1-11, A3-F03, A3-F09, A3-F10,
  A3-F16, A3-F24 (BR-04), A3-F25, CRYPTO-01 (A3-F17), CRYPTO-02, RESTORE-01 and BACKUP-01 (21 IDs). The PR #42 notes
  left A1-08 to A1-10 unchecked; their tests on `main` are `TestOperatorDualControlPreservesScheduledPolicy` and
  `TestOperatorDualControlAfterCancelledPolicy` (A1-08), `TestDisputeReversalAuditTargetsAndAtomicity` (A1-09) and
  `TestAdjustmentExpirySkipsFrozenNetworksWithoutStarvation` (A1-10). Minor residuals:
  CRYPTO-01 (see NEW-01), A3-F09 (passwords, recovery codes and revoked invitations keep their backup state after
  a restore), BACKUP-01. The remaining PR #42 findings are in the table below.
- Open, and recently fixed, on `main` (6 PR #42 IDs; E-02, NEW-01, AUD4-D01 and AUD4-D02 are fixed):

| ID | Severity | State | Note |
|---|---|---|---|
| A3-F02 | Medium | Open, owner decision (DEC-014, SEC-013 addendum) | A break-glass administrator can still make lasting changes: `CreateAdjustment` and `SetCreditLimit` do not call `principal.requireLasting`, and `dualcontrol.go:requireIndependentApprover` accepts a break-glass administrator (`bgAdmin`). Reproduced with a throwaway probe |
| PRIV-01, PRIV-02 | Low | Open, owner decision (SEC-014/018) | `person_activity.last_seen_at`, `person_prefs.time_zone` and `challenges.reason` survive erasure |
| PRIV-03 / A3-F07 | Info/Low | Open, design | After a delete and a PASSIVE checkpoint the deleted text can remain in the WAL file. `TestSecureDeleteLeavesNoCanary` passes because it forces a TRUNCATE checkpoint |
| PRIV-04, PRIV-05 | Policy/process | Open | Retention policy (DEC-020); free-text audit reasons |
| E-02 (from `docs/audit/SECURITY-AUDIT-3.md`, not PR #42) | Low | Fixed (PR #71); path escaping completed by AUD4-D01 | Every normal and raw connection opens with `trusted_schema(0)` (`TestSQLiteDistrustsSchemaOnEveryConnection`, `TestSQLiteRefusesSchemaFunctionCalls`). No schema fingerprint: built-in-only triggers in a database restored with `-allow-unsigned` still run |
| NEW-01 (new in this re-check) | Low | Fixed (PR #72); cleanup race closed by AUD4-D02 | Snapshot, ciphertext and verification copies are staged in a leased private `os.MkdirTemp` directory and `copyFileSync` uses `O_EXCL` (`cmd/circula/backup_private_staging_test.go`, `backup_work_lease_test.go`) |
| AUD4-D01 (2026-11-18 audit) | Low | Fixed, regression-tested | Paths were placed in the SQLite `file:` URI unescaped; a `backup-dir` or `circula backup` directory containing `?`, `#` or `%` opened a different file and could drop `trusted_schema(0)`/`foreign_keys(1)`. `internal/store/uri.go` escapes them. See [AUD4-D01-D02](../audit/consolidation-completion-2026-11/AUD4-D01-D02-BACKUP-PATHS.md) |
| AUD4-D02 (2026-11-18 audit) | Low | Fixed, regression-tested | Backup cleanup checked entries by name and then wrote through them; in a destination writable by other users a planted or swapped entry could make cleanup zero or delete data elsewhere. Cleanup now re-checks identity on the open handle, never follows symlinks or zeroes multi-link files, sweeps only private directories, and staging refuses shared non-sticky directories |

Older open items are unchanged and remain owner decisions: SA2-12 (SEC-011), A3-04b, A3-07, A3-10/11, A3-F18, SA2-21.
A3-F20 and A3-F21 were not re-verified.

## Not covered

Manual penetration testing, a timing side-channel analysis, a review of the Windows-specific code paths, container
image hardening (Docker is not available here), and an independent review (SEC-033).
