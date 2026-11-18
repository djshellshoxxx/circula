# Audit 2026-10-28 Low and Info findings: triage and remediation, integrated on e67bad5

Integrated in a detached worktree at `e67bad5` (main after PR #69 and the handoff PR #70; `1ee699f` is an ancestor).
Triage, both fixes and both reviews ran at `1ee699f`. The two approved patches were applied here, in order, with `git apply --3way` and committed as `bf494d0` (A2-08) and `155caab` (A3-F10). This file was added in `8bd8e45`, the commit after them. A follow-up fixer then fixed the A3-F10 logging half (OPS-033) in `4bbc871`, on `da04d45`, and the commit after `4bbc871` updated this file. Nothing was pushed and GitHub was not used.
Toolchain: go1.26.11 (`go.mod` requires `go 1.26.11`; the auto toolchain selects it). Prefixes: S/=internal/service, C/=cmd/circula, A/=internal/api, W/=internal/web, T/=internal/store.
File and line references in the table are the triage's, at `1ee699f`. In `cmd/` and `internal/` only C/serve.go, C/config.go, C/serve_test.go and one new service test changed between `1ee699f` and `e67bad5` (PR #69).

Scope: every Low and Info finding of the three reports (A1: 8, A2: 21, A3: 20), plus A1-04-G2, which the triage split from A1-04. That is 50 findings: 2 fixed here, 41 already fixed on main, 4 owner decisions, 2 not worth fixing and 1 covered by an open PR. The parent session (the session that ran this remediation and started the triage, fixer, reviewer and integrator agents) overrode three triage verdicts (A3-F05, A3-F07, A3-F21); both verdicts are shown under [Overridden triage verdicts](#overridden-triage-verdicts).

| Finding | Severity | Disposition | Fix commit or reason |
|---|---|---|---|
| A1-04 | Low | already fixed | `845ad59` (#51): S/reversal.go:84-89 `reverse()` starts with `networkOpenErr`, and every dispute path reaches it through `reverseDispute`. `TestReverseRejectsAdministratorRequiredNetwork`; the PoC `TestA1PoC_04` (store import moved to internal/repository) passes at `1ee699f`. Since PR #69 (`91a26f2`) `TestResolveDisputeRefusesReversalWhenAdministratorRequired` also covers the entry point. The closing-state side effect is A1-04-G2 |
| A1-04-G2 | Info | owner decision | A1 spec gap G2: are dispute reversals exempt from the GOV-025 b posting freeze while the network is `closing`? As built, an upheld dispute is refused while closing (`invalid-transaction-state: network is closing`) and a dismissal works. No code change |
| A1-05 | Low | already fixed | S/transactions.go:381-390 passes `FrozenNetworks()` to `ListExpiredProposals`, which filters them in SQL before `LIMIT` (T/sqlite_tx.go:164-180). `TestExpiryFrozenNetworkDoesNotStarveActiveProposals`, `TestExpiryManyFrozenNetworks`. `TestA1PoC_05` still fails because of its fixture (net2 keeps the default 7-day expiry), not because of the finding |
| A1-06 | Low | already fixed | `2eacf77` (#54): S/departure.go:219-250 `cancelOpenProposals` pages until no proposal is left, in the same departure unit. `TestDepartureCancelsMoreThanFiveHundredProposals`; `TestA1PoC_06` passes |
| A1-07 | Low | already fixed | S/closure_schedule.go:170-204 `FinaliseClosures` retries closures that have no final export. `TestClosureFinalExportRetriesAfterFailure`; `TestA1PoC_07` passes. Residual outside the finding: one network whose export keeps failing delays later networks in the same run |
| A1-08 | Info | already fixed | S/dualcontrol_operator.go:42-88 appends at `LatestPolicyVersion()+1` and re-appends each live scheduled notice with only the threshold changed; the original rows stay. `TestOperatorDualControlPreservesScheduledPolicy`, `TestOperatorDualControlAfterCancelledPolicy`; `TestA1PoC_08` passes |
| A1-09 | Info | already fixed | S/dispute_reversal_audit.go:13-28 `reverseDispute` records `transaction.reversed` with the reversal as its target, in the same unit. `TestDisputeReversalAuditTargetsAndAtomicity`; `TestA1PoC_09` passes |
| A1-10 | Info | already fixed | S/dualcontrol.go:300-325 filters frozen networks in SQL before `LIMIT` and re-checks each row. Three `TestAdjustmentExpiry*` tests; `TestA1PoC_10` passes |
| A1-11 | Low | already fixed | `c34d947` ([REMEDIATION-2026-11-04.md](REMEDIATION-2026-11-04.md), A1-02/A1-11): the decimal scale is fixed at creation; `TestPolicyChanges`; FR-005 and GOV-004 rows corrected |
| A2-01 | Low | already fixed | A/canonical_fields.go:15 `checkCanonicalFields`, called from `strictUnmarshal` (A/decode.go:81); the ported PoC gets 400. `TestCanonicalRequestFields` |
| A2-02 | Low | already fixed | S/tokens.go:116 and :143 refuse person-level token operations for a network-scoped token; the PoC gets 403. `TestNetworkScopedTokenCannotManagePersonResources` |
| A2-03 | Low | already fixed | A/idem.go:177 re-checks scope, role and MFA before a replay (A/replay_auth.go:58); the PoC gets 403 and no `Idempotent-Replayed`. `TestReplayRechecksWriteScopeAndMembership`, `TestReplayRechecksAdministratorRole`. The A3-10 eviction question stays owner-pending |
| A2-04 | Low | already fixed | A/h_policy.go:102-103 `adminOnly` uses effective roles, scope and MFA, including a live break-glass grant (SEC-013 addendum decision 4). `TestAdapterAdminReadsUseLiveBreakGlass`, `TestWebAdminReadsAndNavigationUseLiveBreakGlass`. A3-F02 and DEC-014 are untouched |
| A2-05 | Low | already fixed | A/h_policy.go:103 `adminOnly` checks current MFA (SEC-007 addendum, RequireMFAForAdmins). `TestAdapterAdminReadsRequireCurrentMFA`, `TestWebAdminReadsAndNavigationRequireCurrentMFA` |
| A2-06 | Low | already fixed | W/h_auth.go:93 checks the error from `Logout`: the user gets a 503 page and keeps the cookie for a retry. `TestLogoutFailurePreservesSessionForRetry` |
| A2-07 | Low | already fixed | A/h_accounts.go:12-13 takes the ETag from `CreditLimitRevision`, and the expected revision is checked inside the write unit (:69-79). `TestCreditLimitETagTracksOverrideAndExpiry`, `TestCreditLimitConditionalWriteRejectsInterveningServiceEdit`, `TestPolicyConditionalWriteRejectsInterveningChange`. The C-09 regression is already on main (internal/contract/contract_test.go:478, `2cd1ec8`), so archive PR #13 needs no salvage. `If-Match: *` is report spec gap 2, not a defect |
| A2-08 | Low | **fixed here** | `bf494d0`: four rows in spec 07 section 7.2 and `TestProblemCodesHaveCatalogueRows`; see the A2-08 section below. The OpenAPI half was already fixed at `1ee699f` |
| A2-09 | Low | already fixed | A/h_gov.go:82-85 `policyLog` calls `c.only` and `c.page` (signed cursor pages of at most 100). `TestGovernancePolicyLogPagination`, `TestGovernanceChallengePagination`, `TestGovernancePaginationRejectsInvalidAndReboundCursors` |
| A2-10 | Low | already fixed | W/h_auth.go:352-360 answers the join page of an invitation-only network like an unknown slug. `TestInvitationOnlyJoinHidesNetworkFromNonmembers`, `TestPublicJoinModesRemainAccessible`. Report spec gap 5 (open and approval networks) is not decided |
| A2-11 | Low | already fixed | W/render.go:218 shows the sections a suspended or departed member can still read. `TestNavigationMatchesCurrentReadableSections` |
| A2-12 | Low | already fixed | W/h_disputes.go:345-352 and W/h_dualcontrol.go:207 add a review-and-confirm step (UX-018). `TestReReviewRequiresReviewedConfirmation`, `TestReReviewConfirmRechecksLiveStateAndAuthority`, `TestAdjustmentApprovalRequiresReviewedConfirmation`. The same change took a reading on report spec gap 3 |
| A2-13 | Info | already fixed | W/render.go:58-59 adds a separate `FrozenSince` field. `TestFrozenAndCheckTimesRenderAsElements` |
| A2-14 | Low | already fixed | `94728b1` (DEF-001): the custom `qs` template function is gone. `TestReturnLinksDecodeToTheOriginalPath` |
| A2-15 | Info | already fixed | W/middleware.go:62-64 sets no CSRF cookie on `/static/`, `/robots.txt` and `/favicon.ico`. `TestPublicAssetsDoNotSetCookies` |
| A2-16 | Info | already fixed | Identifiers are rejected before any store call (SEC-024). `TestPathIDsRejectedBeforeResourceLookup`, `TestIdentifierBoundariesAcrossHTTPAdapters` |
| A2-17 | Info | already fixed | W/h_home.go:40-41 and W/h_auth.go:239-240 send a person who must change their password to `/account/password`. `TestForcedPasswordGuardsHomeAndInvitation` |
| A2-18 | Info | already fixed | A/decode.go:79-86 gives a generic detail, and unknown query parameters get 400. `TestGetRepresentationsRejectUnknownQueryParameters`, `TestMalformedRawQueriesAreRejected`, `TestGetQueryErrorsMatchOpenAPI` |
| A2-19 | Info | already fixed | openapi.yaml now says `POST /api/v1/tokens` takes no `Idempotency-Key`, because token values are not kept in the replay cache (A3-09). The runtime is unchanged by design. Optional docs nit: an as-built note on the API-066 row |
| A2-20 | Info | not worth fixing | A caller or proxy `X-Request-Id` is accepted on purpose, so that the proxy, the access log and the user's reference match (SEC-020; 06:219 "request identifier from the edge"). The adapters were aligned in `79d22c0` (`TestRequestIDMatchesAcrossEdgeAndAdapters`). A separate server-only audit id would need an audit-schema change and edge changes; if the owner wants one, it becomes a spec change |
| A2-21 | Info | not worth fixing | S/auth.go:185-188 stores the SHA-256 of 256-bit random secrets (sessions, API tokens, invitations, MFA challenges). A keyed hash needs a mandatory server key (wiring in C/config.go and C/serve.go, or a mandatory `SecretKey`), a migration of every stored hash and a plan for losing the key. API-009 ("salted or keyed hash") stays unmet; recording it as a 07 section 12.3 deviation is the owner's call |
| A3-F03 | Low (verified Info) | already fixed | `6ac3132` (#52): S/principal.go:151-154 `requireDecider` refuses with `mfa-enrolment-required` for an administrator without MFA; plain reviewers stay exempt (DEC-007). Since PR #69 `TestDeciderEntryPointsRequireAdministratorMFA` also covers the entry points |
| A3-F05 | Low | owner decision (triage: fix; overridden) | PRIV-02; see [Overridden triage verdicts](#overridden-triage-verdicts). The recover-key throttle half is fixed on main (S/erasure.go:216-222); `challenges.reason` still survives erasure (T/sqlite_erasure.go:33-58). The stopped fixer's unreviewed patch is kept, not applied: [wip-fixes/challenge-reason-erasure.patch.txt](wip-fixes/challenge-reason-erasure.patch.txt) |
| A3-F06 | Low | already fixed | T/sqlite_person.go:66-76 compares characters on both sides (`substr(login, 1, length(?) + 1) = ?`). internal/store/throttle_family_test.go |
| A3-F09 | Low | already fixed | C/restore_security.go:13-24 invalidates the staging copy's credentials before the swap: T/sqlite_restore.go:16-50 revokes every session and API token, deletes MFA challenges, closes open break-glass sessions and fails if any remain. C/restore_credentials_test.go, S/restore_credentials_test.go |
| A3-F10 | Low | **fixed here** (retention and logging) | Retention: `155caab`. Logging the clock anomaly (OPS-033): `4bbc871`, `TestJobUntilNextBackupLogsFutureDatedBackups` and `TestJobBackupLoopLogsFutureDatedBackupEachEvaluation`. See the A3-F10 section below |
| A3-F11 | Low | already fixed | The A1-08 change (S/dualcontrol_operator.go:42-81). S/operator_policy_regression_test.go |
| A3-F13 | Low | already fixed | C/revoke_sessions.go:19-71 adds the audited `circula revoke-sessions`; docs/ops/configuration.md:125-136 documents it. Cosmetic: docs/specs/addenda/SEC-027-keys.md:13 could link to it |
| A3-F14 | Low | already fixed | scripts/check-hygiene.sh:36-41 runs scripts/check_migrations.py and its unit tests. It flags DROP TABLE, COLUMN and TRIGGER, and UPDATE, REPLACE INTO and upsert on protected tables, unless the statement is marked allow-destructive |
| A3-F15 | Low | already fixed | C/serve.go `jobRunner.verify` calls the read-only `verifyAllNetworks` (DEC-031, OPS-016). Optional cleanup: the unused `RecordScheduledConsistencyRun` branch (C/snapshot.go:150-154) |
| A3-F17 | Low | already fixed | C/util.go:48-73 `writeAtomicPrivateFile` (O_EXCL temporary file, mode 0600, sync, rename), used for manifests, JSON files and anchors |
| A3-F18 | Low | owner decision | A3 spec gap 3; docs/TODO.md:101 (member-limit notice timing). GOV-003 and the 10 section 2 table require notice for per-member overrides, while GOV-016 and GOV-019 read as immediate and then open to challenge. Choosing the timing is financial policy; S/membership.go:287-330 is unchanged |
| A3-F19 | Low | already fixed | `448b4c9`: bounded `auth-throttle-*` settings (C/config.go:59-61, 396-399); S/throttle.go:50-55 uses `AuthThrottleLock` |
| A3-F25 | Low | already fixed | `8b10d45` (#53): T/sqlite_erasure.go:95-118 selects notifications by member and by links to the member's records, not by name. S/notification_erasure_test.go |
| A3-F07 | Info | owner decision (triage: fix; overridden) | PRIV-03; see [Overridden triage verdicts](#overridden-triage-verdicts) |
| A3-F12 | Info | already fixed | T/migrate.go:101 refuses `v < 1`. internal/store/migrate_version_test.go |
| A3-F20 | Info | already fixed | scripts/scan-release.sh:52-67 lists and extracts `.zip` files; scripts/package-release.sh runs it through scripts/verify-release.sh |
| A3-F21 | Info | covered elsewhere (triage: fix; overridden) | Open PR #75; see [Overridden triage verdicts](#overridden-triage-verdicts) |
| A3-F22 | Info | already fixed | S/breakglass.go:312 looks the login up with `normalizeLogin`. S/breakglass_login_regression_test.go |
| A3-F23 | Info | already fixed | C/commands.go:517 calls `RegisterPersonAsOperator` with a system actor; the event records the operator actor (S/auth.go:58-64, 94-98) |
| A3-F24 | Info | already fixed | A/erasure_hook.go:21-35 drops every export artefact of the network that may hold the member's data (BR-04). A/export_lifecycle_test.go, A/export_spool_test.go |

Method. The triage re-checked each finding at `1ee699f` against the source, the specs, docs/DECISIONS.md and the auditors' PoCs (store imports moved to internal/repository where needed). A2 PoCs assert the defect, so for them a failure means the defect is gone. Each of the two fix groups had a fixer that reproduced the defect, attempted a refutation, wrote the fix and new regression tests, and proved fail-before by reverse-applying its own production or spec diff. An independent reviewer then repeated the checks in a fresh worktree and approved. Here the approved patches were applied in order with `git apply --3way` and committed one per group, and fail-before was proved again on this tree. PoC copies, shims and probes were deleted afterwards and are not committed.

---

## A2-08: the spec 07 error catalogue lacks four codes the API sends. FIXED (`bf494d0`)

- **Evidence at `1ee699f`** (triage and fixer).
  - The OpenAPI half was already fixed. The A2 PoC `TestA2_UndocumentedRuntimeStatuses` asserts the defect, and it now fails at its first assertion: `frozen write: POST /listings -> 423; openapi validation error: <nil>`. A copy with its direction converted passes for `POST /listings` 423 and `GET /network` 429. openapi.yaml has 52 shared 423 responses, and `TestCommonErrorResponsesDeclared` and `TestCommonErrorResponsesMatchOpenAPI` cover 413, 415, 423, 429 and 503.
  - Still present: section 7.2 (30 rows) had no row for `network-frozen` (423), `approval-required` (403), `password-change-required` (403) or `terms-acceptance-required` (409), and spec 07 did not mention them anywhere. A/respond.go `codeStatus` maps all four, and openapi.yaml lists them in its `ErrorCode` enum. The 12.2 error-catalogue bullet and G-2 ("No contract test validates live responses") were stale.
- **Refutation attempt failed.**
  - All 22 literal codes (32 `apiError` literals) in internal/api already had a 7.2 row with the same status. The only computed code is the `codeStatus` lookup in `fromErr`, and it produces the four missing codes.
  - `approval-required` is reachable. A temporary probe through the OpenAPI-validating harness sent `POST /admin/members/{id}/remove` with a `settle-to-zero` settlement of 5000 above a threshold of 1000. It got 403 `approval-required`, the response was valid against openapi.yaml, and the member stayed active. The legacy `CreateAdjustment` can also return the code but has no caller in api, web or cmd.
- **Owner gate?** No. API-074 allows adding error codes in a minor revision, and 7.2 already uses the "Added as built" pattern (report spec gap 8). The rows record shipped codes and statuses only. The `approval-required` row points to the open SEC-011 owner decision for other credit-position actions, and the `network-frozen` row leaves DEC-076 (scope of a verification failure) open.
- **Fix.** docs/specs/07-api-contract.md only; no production code changed.
  - Section 7.2: four rows in the existing "Added as built (API-074 ...)" style, dated 2026-11-18.
  - Section 10, API-077: the stale "No contract test yet validates live responses" now says that the internal/contract tests validate the responses they exercise. "Not yet released as a versioned artefact" stays, so API-077 is not claimed to be fully met.
  - Section 12.2: the error-catalogue bullet names the four codes and the new test.
  - Section 12.4, G-2: now "Partly closed". Live-response validation (API-077) is closed, citing `TestAPIMatchesOpenAPI`, `TestCommonErrorResponsesDeclared` and `TestCommonErrorResponsesMatchOpenAPI`. The API-078 previous-release fixtures still do not exist, so that half stays open.
- **Regression test** (new A/problem_catalogue_test.go, no build tag): `TestProblemCodesHaveCatalogueRows`. Every problem code internal/api can send must have a 7.2 row with the same status. The codes come from `codeStatus` and from an AST scan of every `apiError` literal in the package's non-test sources. A computed code outside `fromErr`, a positional literal or a non-literal status fails the test instead of being skipped. The test cites no spec ID, so docs/TRACEABILITY.md does not change.
- **Fail-before.**
  - Fixer and reviewer, at `1ee699f`: with the spec diff reverse-applied, the test fails with exactly the four missing codes; with it re-applied, it passes. The fixer also mutated the inputs. Changing the `network-frozen` row to 409 fails with `network-frozen is sent with 423 (codeStatus) but spec 07 section 7.2 says 409`. A temporary source with an unlisted literal (`teapot`, 418), a computed code and a positional literal gives three distinct failures.
  - Integrator, on this tree: `git diff e67bad5 bf494d0 -- docs/specs/07-api-contract.md | git apply -R`, then `go test -count=1 -run TestProblemCodesHaveCatalogueRows -v ./internal/api`. FAIL with `problem_catalogue_test.go:47: approval-required (403, codeStatus) can be sent but has no row in spec 07 section 7.2`, and the same line for `network-frozen` (423), `password-change-required` (403) and `terms-acceptance-required` (409). After `git checkout` of the spec it passes, and so do `TestCommonErrorResponsesDeclared` and `TestCommonErrorResponsesMatchOpenAPI`.
- **Review: approved.** Four minor points, not changed here:
  1. The AST scan matches only composite literals typed `apiError`, so a literal with the element type left out (`[]*apiError{{...}}`) would be skipped. No such literal exists today.
  2. Every non-literal `apiError` inside `fromErr` is exempt, not only the `codeStatus` lookup, so a future computed code there would not be checked.
  3. The `password-change-required` row: SEC-008 addendum decision 5 also lets signing out through. "Every write except changing the password and signing out" would be more exact.
  4. The `network-frozen` row: as built (C/serve.go `applyFreeze`), the freeze lifts at the next passing verification run after an operator repair or restore.

## A3-F10: a future-dated backup name pushes real backups out of retention. FIXED (retention `155caab`, logging `4bbc871`)

- **Evidence at `1ee699f`** (triage and fixer).
  - The cadence half was already fixed: C/serve.go `untilNextBackupAt` skips entries dated after now (`TestJobUntilNextBackupIgnoresFutureFiles`).
  - Still present: C/snapshot.go `listBackups` sorts by the time in the file name with no upper bound, and `pruneBackups` filled the keep-daily and keep-monthly slots in that order. So a future-dated file always took a daily slot and a month slot. No retention test covered future-dated files, and nothing logged the anomaly (the logging half, fixed later in `4bbc871`; see below).
  - Reproduction, with the new tests and a test-only shim that gives the old behaviour:
    - 50 years ahead with the default counts 30/12: the 30th newest real backup and the 12th real month were removed.
    - One hour ahead with counts 1/1: the newest real backup was removed, so no real backup was left.
    - A backup dated exactly now, counts 1/0: removed.
    - Several future-dated files: two of them were deleted, and so was the newest real backup.
    - Through `backupNow` with counts 2/1: one real backup was kept instead of two.
- **Refutation attempt failed.** At `bb5a36f` the verifier called the retention claim overstated, because backups stopped altogether. On main the cadence fix keeps backups running, so `pruneBackups` runs at every interval and a future-dated file holds one daily and one monthly slot for good. With keep-daily 1 (the allowed minimum, C/config.go:367), each new backup was deleted right after it was written. OPS-018 says expired backups are removed according to the configured count, and DEC-059 sets 30 daily backups plus the newest of each of 12 months.
- **Owner gate?** Not for retention. The fix only ever keeps more files and changes no count, period or approval rule; DEC-020 is untouched. The logging half was deferred because of file ownership, not because it needs a decision. Its fix (`4bbc871`) adds one warning and no setting, threshold or policy.
- **Fix** (C/snapshot.go, `pruneBackups` and the new `pruneBackupsAt` only).
  - `pruneBackups` keeps its signature and returns `pruneBackupsAt(dir, keepDaily, keepMonthly, time.Now())`. This follows the existing clock-injection pattern of `untilNextBackupAt(now)` and `pruneLogs(dir, age, now)`.
  - `pruneBackupsAt` keeps every entry with `at.After(now)` without counting it, then fills the daily and monthly slots from the other entries in the same order. The boundary is the strict `After(now)` of `untilNextBackupAt`, so a backup dated exactly now counts.
  - Real backups always sort after future-dated ones, so every real backup the old code kept is still kept. The fixer's differential test over 3000 random directories found no case where the new code removed a file the old code keeps, or removed a future-dated file. The directories mixed plain, encrypted and legacy names, with keep-daily 1-40 and keep-monthly 0-14.
  - `155caab` leaves C/serve.go unchanged; its only caller passes `time.Now()`. The functions that PR #72 changes are untouched.
  - docs/specs/11-deployment-operations.md: the OPS-018 row stays `built` and gets a dated note; the OPS-033 row stays `partial`. docs/TRACEABILITY.md was regenerated with `make traceability`: the OPS-018 row now cites `pruneBackupsAt` and both tests, and the header records `bf494d0`.
- **Regression tests.**
  - C/backup_test.go `TestPruneKeepsFutureDatedBackupsOutOfCounts`, with subtests `50_years_ahead,_default_counts`, `an_hour_ahead_in_the_current_month`, `backup_at_now_counts` and `several_future-dated_files`. They assert the exact set of removed files, check manifests and anchors, and check that the future file's bytes are unchanged.
  - C/jobs_regress_test.go `TestJobRetentionDoesNotCountFutureDatedBackup`, through the unchanged `backupNow` path in C/serve.go.
- **Fail-before.**
  - Fixer and reviewer, at `1ee699f`: they reverse-applied the snapshot.go production diff and added a test-only shim that maps `pruneBackupsAt` to the old `pruneBackups`. The old function never reads the clock, so this is exactly the old behaviour. All 4 subtests and the job test failed; with the diff re-applied they passed. The fixer saw no flake at `-count=20`, and the tests pass under `TZ=Pacific/Kiritimati` and `TZ=America/Los_Angeles`.
  - Integrator, on this tree, with the same reversal and shim:
    - `TestPruneKeepsFutureDatedBackupsOutOfCounts` fails in all 4 subtests. For example, `an_hour_ahead_in_the_current_month` gives `removed [circula-20261115T123000.123456789Z.db circula-20261116T122900.123456789Z.db], want [circula-20261115T123000.123456789Z.db]`. `50_years_ahead,_default_counts` reports `circula-20261018T113000.123456789Z.db: present=false, want true`, and the same for `circula-20251215T030000.000000000Z.db`.
    - `TestJobRetentionDoesNotCountFutureDatedBackup` fails at jobs_regress_test.go:506: it kept only the year-2076 file and the new backup, not yesterday's real backup.
    - With the shim deleted and snapshot.go restored, both tests pass. So do `TestPruneKeepsDailyAndNewestPerMonth`, `TestBackupNamesPreserveSubsecondTimeAndLegacyRetention`, `TestPruneAndListHandleEncryptedBackups`, `TestJobScheduledBackupAndRetention`, `TestJobUntilNextBackup` and `TestJobUntilNextBackupIgnoresFutureFiles`.
  - The retention part of the auditor PoC `TestA3Audit_FutureDatedBackupStopsSchedule` only logs, so the PoC passes either way. The fixer saw `real backups kept: 1` before the fix (keep-daily 2) and `real backups kept: 2` after.
- **The OPS-033 logging half: FIXED in `4bbc871`** (a follow-up on `da04d45`; the commit after it updated this file).
  - Why later: the retention fixer deferred it because C/serve.go belonged to PR #69. PR #69 has merged, and none of the open PRs #71-#75 changes C/serve.go or C/config.go.
  - Evidence at `da04d45`. OPS-033, "Clock error on host": time-based expiry and deadlines use the server clock and are re-evaluated on restart, and "the anomaly is logged". The detection in docs/ops/monitoring.md was "Logs and timestamps out of order". C/snapshot.go `pruneBackupsAt` kept a backup named after now without counting it, and C/serve.go `untilNextBackupAt` ignored it for scheduling, but nothing logged it.
  - Reproduction at `da04d45`, with the two new tests added: `TestJobUntilNextBackupLogsFutureDatedBackups` failed in its three future-dated subtests with `evaluation 1: 0 log entries, want 1 (one per evaluation):`, followed by an empty log. Its three subtests without such a file passed, and every returned wait matched. `TestJobBackupLoopLogsFutureDatedBackupEachEvaluation` saw the first scheduled backup (the file does not hold the schedule back), then failed with `timed out waiting for a warning at start-up and another after the first backup`.
  - **Fix** (C/serve.go, `untilNextBackupAt` only). The backup job runs it at start-up and on each backup cycle. After listing the backups it counts those with `at.After(now)`, the same strict boundary as scheduling and retention. If there is at least one, it writes one `Warn` through `j.logger`, the logx-wrapped serve logger: msg `backups named after the current time (host clock error or misplaced file): kept, but ignored by scheduling and retention (OPS-033)`, with `count`, `file` (`filepath.Base` of the newest such file, the first entry because `listBackups` is newest first), `name_time` and `now`, both in UTC. It writes one line per evaluation, never one per file. The wait loop below it is unchanged, so the returned durations are the same. No setting, threshold or policy was added.
  - Logger: the only production caller is `runServe`, which passes the `newLogger` logger and cannot pass nil, so the code has no nil check. The one test fixture without a logger, in `TestJobUntilNextBackupIgnoresFutureFiles`, now gets `discardLogger()`; its assertions are unchanged.
  - Docs. docs/ops/monitoring.md, row "Clock error on host": the detection cell names the warning, when it is logged and its fields. Spec 11, OPS-018 row: the note says that the backup job logs the anomaly and cites `TestJobUntilNextBackupLogsFutureDatedBackups`. No status changed; OPS-033 stays `partial`, because its other failure modes are unchanged. docs/TRACEABILITY.md was regenerated with `make traceability`: the OPS-033 row now cites `untilNextBackupAt` and both new tests, the OPS-018 row cites the unit test, and the header records `da04d45`.
  - **Regression tests** (C/jobs_regress_test.go, no build tag).
    - `TestJobUntilNextBackupLogsFutureDatedBackups`, six subtests, through the production `newLogger` in JSON. Each subtest evaluates twice with a `now` in UTC+14 and checks one new entry per evaluation, the exact level, msg and fields (and no other field), and the returned wait. With no backups, a recent backup or a backup dated exactly now, nothing is logged. A backup 50 years ahead, one 1 ns ahead, and three future-dated files (plain, encrypted and legacy names) beside an overdue backup each give one line per evaluation naming the newest file, with `now` in UTC.
    - `TestJobBackupLoopLogsFutureDatedBackupEachEvaluation`: the real `backups` loop, with a future-dated file, backs up at once and logs exactly two warnings, one at start-up and one after that backup.
  - **Fail-before**, done once, on the working tree that became `4bbc871` (the same code and tests): the serve.go diff reverse-applied with `git apply -R` (serve.go then matched `da04d45`), then `go test -count=1 -run 'TestJobUntilNextBackupLogsFutureDatedBackups|TestJobBackupLoopLogsFutureDatedBackupEachEvaluation' -v ./cmd/circula`. FAIL: `future_only`, `one_nanosecond_ahead` and `several_future-dated_files` with `jobs_regress_test.go:534: evaluation 1: 0 log entries, want 1 (one per evaluation):`, and the loop test at jobs_regress_test.go:575 with `timed out waiting for a warning at start-up and another after the first backup`. With the change re-applied both pass, and so does `TestJobUntilNextBackupIgnoresFutureFiles`.
  - **Owner gate?** No. OPS-033 already requires the anomaly to be logged.
  - **Review:** not yet reviewed independently.
- **Review of the retention fix (`155caab`): approved.** Three minor points, not changed here:
  1. The `pruneBackups` doc comment still says it keeps the newest keepDaily backups. The future-dated rule appears only in the inline comment.
  2. A future-dated file is now kept for as long as its name is in the future, even when there are more such files than keep-daily (the old code deleted the extras). This only keeps more files and decides no count or period. But OPS-018 ties retention to personal-data exposure, so the retention owner may want to confirm this case.
  3. docs/ops/backup-restore.md:86-87 says that only the names of pruned backups are logged, but `backupNow` logs only a count. This predates the change and belongs with the OPS-033 follow-up. Still open: `4bbc871` logs the clock anomaly only, and leaves the prune log line and backup-restore.md as they were.

## Overridden triage verdicts

The parent session overrode these three triage verdicts, because the main branch or an open PR already records the item. The A3-F05 fixer had started and was stopped. Both verdicts are shown.

| Finding | Triage verdict (`triage_disposition`, group) | Triage reason (`triage_reason`) | Final disposition | Defers to |
|---|---|---|---|---|
| A3-F05 | fix (mfa-throttle-erasure) | "SEC-018 acceptance (no name/email/phone left in personal-data fields after pseudonymisation) and docs/dev/erasure.md:51-53 (new member free-text columns must be added to erasure) define the correct behaviour. Redacting the text does not need a policy decision; whether a still-requested challenge should block erasure does, so leave that out." | owner decision. Main records it as PRIV-02, an open owner decision, and the recover-key throttle half is fixed on main. Not changed here; the stopped fixer's unreviewed patch is kept only as a reference for that decision | docs/security/security-review.md:90 (`PRIV-01, PRIV-02 \| Low \| Open, owner decision (SEC-014/018)`); docs/TODO.md:91; docs/security/personal-data-inventory.md, Findings item 2; the PR #69 record, docs/plans/PR-CONSOLIDATION-2026-11-17.md:166 ("PRIV-01/02 retention (SEC-014/018)") |
| A3-F07 | fix (backup-restore) | "docs/security/RAINBOW-BOOKS-SPEC.md:84-91 (RB-003, status Partial at :342) already recommends checkpointing the WAL after an erasure. Its verification step scans .db, -wal and -shm for the canary. A best-effort checkpoint is cheap and adds no policy. Info only because host access is trusted (TA-3)." | owner decision. Main records it as PRIV-03, "Open, design". Not changed here | docs/security/security-review.md:91 (`PRIV-03 / A3-F07 \| Info/Low \| Open, design`); the PR #69 record, docs/plans/PR-CONSOLIDATION-2026-11-17.md:166-167 ("WAL checkpoint design for PRIV-03") |
| A3-F21 | fix (docs-nits) | "The SEC-007 addendum's own amendment (line 15) already says the lock is fixed and points to A3-F21. Correcting the two leftover sentences to match the accepted, built behaviour adds no policy." | covered elsewhere. Open PR #75 reconciles the addenda with the fixed, configurable lock. Not duplicated here | PR #75, branch `integration/throttle-documentation-20261117` (head `83de536`): the SEC-007, SEC-008, SEC-013 and SEC-018 addenda, spec 09, docs/DECISIONS.md and docs/dev/recovery.md, with evidence in docs/audit/A3-F21-THROTTLE-DOCS-2026-11-17.md on that branch |

The triage evidence and fix sketches, at `1ee699f`, kept for the decisions. None of them was applied.
- **A3-F05.** S/erasure.go:216-222 deletes the throttle families for both `person.Login` and `recoverKey(person.Login)`. T/sqlite_erasure.go:33-58 `RedactDisputeStatements` clears disputes, dispute responses, re-review reasons and report notes, but not `challenges.reason` (migration 0014_closure_challenges.sql:22, written by the member). No other erasure path touches challenges. Sketch: add `UPDATE challenges SET reason = '' WHERE network_id = ? AND member_id = ? AND reason <> ''` to that loop, and test it with a canary. The stopped fixer's patch, a `git diff` against `1ee699f`, adds that statement to T/sqlite_erasure.go. It adds a new test, `TestErasureClearsMemberChallengeReasons`, instead of extending `TestAnonymiseMemberKeepsLedger`. It also updates the `RedactDisputeStatements` doc comment in internal/repository/breakglass_iface.go, docs/dev/erasure.md, docs/security/personal-data-inventory.md and the SEC-018 addendum. A check with a temporary index showed it applies cleanly at `1ee699f`, at `e67bad5` and on the PR #75 head; nothing was applied.
- **A3-F07.** T/sqlite.go:106-113 sets `journal_mode(WAL)` and `secure_delete(1)`. No `wal_checkpoint(TRUNCATE)` or `journal_size_limit` appears in internal/ or cmd/, and the erasure path does no checkpoint. Sketch: a best-effort `PRAGMA wal_checkpoint(TRUNCATE)` after the erasure transaction commits, logged if busy; a test that checks the -wal file for the canary without forcing the checkpoint itself; and an update to docs/ops/configuration.md:269-274.
- **A3-F21.** Already corrected in docs/dev/recovery.md:19 ("fixed, not doubling") and S/mfa_login.go:57. Still stale: docs/specs/addenda/SEC-007.md:14 ("5 failures, then 15 min doubling to 24 h") and docs/specs/addenda/SEC-008.md:25-26. The code (S/throttle.go:50-55) uses a fixed `auth-throttle-lock` (default 15 min, bounded to 15 min-24 h).

## Owner decisions and deferred items

| Item | Waits on | Note |
|---|---|---|
| A1-04-G2 | The owner's ruling on A1 spec gap G2 (GOV-025 b, FR-068) | Either confirm the refusal while `closing` and document it in GOV-025 and FR-068, or ask for a dispute-only exemption while closing. Until then a closure can finish only by dismissing or withdrawing a meritorious dispute |
| A3-F05 | PRIV-02 owner decision (SEC-014/018) | Empty `challenges.reason` on erasure, or document a retention exception. Separately: whether a challenge still waiting for a decision should block erasure. Reference patch: [wip-fixes/challenge-reason-erasure.patch.txt](wip-fixes/challenge-reason-erasure.patch.txt). It also edits the SEC-018 addendum, which PR #75 edits |
| A3-F07 | PRIV-03, WAL checkpoint design | docs/security/security-review.md:91 |
| A3-F18 | The owner's policy on notice timing for member-limit cuts (A3 spec gap 3; docs/TODO.md:101) | Financial policy. Open PR #73 changes `noticeAdminGranted` (S/membership.go:334-348), directly below `setCreditLimit` (:287-330), which an A3-F18 change would edit |
| A2-20, A2-21 | The owner, only if wanted | Not worth fixing as they stand. A2-20: server-only audit request ids would be a spec change. A2-21: whether to record API-009 (unkeyed hash) as a 07 section 12.3 deviation |
| A3-F21 | Open PR #75 | Not duplicated here |

The A3-F10 OPS-033 logging half, listed here as unblocked until the follow-up, is fixed in `4bbc871` (see the A3-F10 section). The A3-F10 review points stay under open item 8.

Not decided here and not touched: A3-F02 and anything that depends on DEC-014 (break-glass scope; see [REMEDIATION-2026-11-04.md](REMEDIATION-2026-11-04.md)); the A1 report spec gaps G1-G8; the owner-pending A3-04, A3-07, A3-10 and A3-11; the SEC-011 questions; DEC-020 retention; DEC-076, which the `network-frozen` row cites.

## Open consolidation PRs

No file or region that these PRs change was edited here. The adjacency was checked with `git merge-tree --write-tree` of `155caab` against each PR head, which writes no working tree. The same check of `4bbc871`, against the same local PR heads, gives the same result: #71, #72, #73 and #75 merge cleanly, and #74 conflicts only on the TRACEABILITY.md header line. None of the PR branches changes C/serve.go, C/jobs_regress_test.go, docs/ops/monitoring.md or spec 11.

| PR | Branch (head) | What it changes | Adjacency with these commits |
|---|---|---|---|
| #71 | `integration/store-untrusted-schema-20261117` (`1a02020`) | SQLite `trusted_schema` in the connection strings (T/sqlite.go, T/inspect.go), a test, an audit note | None; merges cleanly |
| #72 | `integration/backup-private-staging-20261117` (`9460895`) | NEW-01 private backup staging: in C/snapshot.go `verifyFile`, `verifyCopy`, `snapshot`, `snapshotWith`, `backupFileNameFor` and `sweepStaleTemp` (hunks at lines 202-464); C/util.go; new C/backup_work.go; C/commands.go; C/premigrate.go; tests; an audit note | Same file as `155caab`, different functions: the A3-F10 hunk is `pruneBackups`/`pruneBackupsAt` (C/snapshot.go:517-551). snapshot.go merges automatically and keeps both changes |
| #73 | `integration/admin-grant-notice-pagination-20261117` (`cb25b41`) | S/membership.go: `noticeAdminGranted` pages past 500 members; a test; an audit note | None for these commits; merges cleanly. See A3-F18 above |
| #74 | `integration/release-threshold-check-20261117` (`ec98cab`) | scripts/check_thresholds.py and its tests, scripts/release-gate.sh, .github/workflows/ci.yml, docs/TRACEABILITY.md (header line, QA-040 and DEC-023 rows), an audit note | docs/TRACEABILITY.md: both sides change the generated header line (`Input: git HEAD` is `bf494d0` here, `da04d45` since `4bbc871`, and `ef54238` there), so merge-tree reports a conflict on that line only. The rows do not overlap (OPS-018 here). Whichever branch merges second regenerates the file with `make traceability`, not by hand |
| #75 | `integration/throttle-documentation-20261117` (`83de536`) | docs/DECISIONS.md, docs/dev/recovery.md, spec 09, the SEC-007, SEC-008, SEC-013 and SEC-018 addenda, audit notes | No shared files; merges cleanly. Covers A3-F21 |

## New observations (not triaged)

The openapi-contract fixer found these at `1ee699f`, outside this triage. They are recorded for the next triage and were not changed.
1. C/maintenance.go answers API writes in maintenance mode with problem code `maintenance` (503). Section 7.2 lists maintenance under `unavailable`, and the OpenAPI `ErrorCode` enum has no `maintenance`. The body also lacks `instance` and `request_id` (API-029/030 and the OpenAPI `Problem` schema), so such a response would fail contract validation. docs/ops/upgrade.md:67 documents the `maintenance` code, so a fix must choose between changing the wire code and recording a deviation. `TestProblemCodesHaveCatalogueRows` scans only internal/api, so it does not cover this.
2. `mfa-required` is in the OpenAPI `ErrorCode` enum, but the API never sends it. Only the legacy `service.Login` returns it, no adapter calls `Login`, and `codeStatus` does not map it. It was not added to 7.2.
3. Three of the four new codes have no entry in the `titles` map, so their title falls back to `http.StatusText` (`Forbidden` or `Conflict`). That is still a fixed short summary under API-031; no change.

## Files touched

- `bf494d0` (A2-08): docs/specs/07-api-contract.md; new internal/api/problem_catalogue_test.go.
- `155caab` (A3-F10): cmd/circula/snapshot.go; cmd/circula/backup_test.go; cmd/circula/jobs_regress_test.go; docs/specs/11-deployment-operations.md (OPS-018 row); docs/TRACEABILITY.md (regenerated with `make traceability`).
- `8bd8e45` (this record): this file; docs/audit/2026-10-28/wip-fixes/challenge-reason-erasure.patch.txt, a byte copy of the stopped A3-F05 fixer's patch (sha256 `6642a7f2322548f35bbc2ad4d9a607141bf25bbed32fe073beb4772d968eaa24`); one row in docs/audit/2026-10-28/wip-fixes/README.md.
- The commit after `8bd8e45`: this file only, to add the validation results for `8bd8e45` under [Tests run](#tests-run) and correct the race-timeout note.
- `4bbc871` (A3-F10 logging half, OPS-033): cmd/circula/serve.go (`untilNextBackupAt` only); cmd/circula/jobs_regress_test.go (two new tests, and a logger for the `TestJobUntilNextBackupIgnoresFutureFiles` fixture); docs/ops/monitoring.md (the "Clock error on host" row); docs/specs/11-deployment-operations.md (the OPS-018 row note); docs/TRACEABILITY.md (regenerated with `make traceability`).
- The commit after `4bbc871`: this file only, to record the follow-up.

Only `4bbc871` touches cmd/circula/serve.go, and only `untilNextBackupAt` in it. Not touched by any of these commits: cmd/circula/config.go, migrations, e2e/, .github/, archive PR #13, every open PR branch, and what PR #72 owns (C/snapshot.go `snapshotWith`, `verifyFile`, `verifyCopy`, `backupFileNameFor` and `sweepStaleTemp`, C/util.go `copyFileSync`, and the new C/backup_work.go). No existing test was skipped or weakened, and no assertion was relaxed. The only change to an existing test is the logger that `4bbc871` gives the `TestJobUntilNextBackupIgnoresFutureFiles` fixture.

## Integration checks

- Resumed after a container restart. The previous integrator run had committed `bf494d0` and `155caab` and was stopped during this docs step. Checked before continuing:
  - HEAD is `155caab`, its parent `bf494d0`, and that commit's parent `e67bad5`.
  - The changed lines of every file match the approved patches (openapi-contract.patch sha256 `b8c673ea…`; backup-restore.patch sha256 `30720e81…`, as in the fixer's report). The one exception is the TRACEABILITY.md header line (`37958e9` to `bf494d0`), which `make traceability` wrote.
  - Both commit messages end with the two trailer lines.
  - The worktree had no uncommitted, untracked, ignored or stashed leftovers.
- `make traceability-check` passes. The generator reads the specs, docs/DECISIONS.md and the source outside docs/, so this file does not change docs/TRACEABILITY.md.
- The OPS-033 follow-up was made in a new detached worktree at `da04d45`; the main clone's working tree was not touched. The `4bbc871` message ends with the two trailer lines, and its validation is under [Tests run](#tests-run).

## Tests run

The first set ran on the tree of `155caab`, which has the same code as `8bd8e45`, the commit that added this file (it added only files under docs/audit/2026-10-28/). The integrator then repeated the full set on `8bd8e45`. The commit after `8bd8e45` adds those results here and changes nothing else.

| Command | On the `155caab` tree | On `8bd8e45` |
|---|---|---|
| Fail-before, both fixes reversed as described above | A2-08: FAIL with the 4 missing codes. A3-F10: FAIL in 4 subtests and the job test. Restored: PASS | not repeated |
| `gofmt -l .` | clean | clean |
| `go vet ./...` | ok | ok |
| `make traceability-check` | up to date | up to date |
| `scripts/check-hygiene.sh` | ok | ok |
| `scripts/check-action-pins.sh` | ok | ok |
| `GOTOOLCHAIN=go1.26.10 go run github.com/zricethezav/gitleaks/v8@v8.28.0 dir . --no-banner --redact` | no leaks found (9.11 MB scanned) | no leaks found (9.16 MB, including this file and the reference patch) |
| `go test -count=1 ./...` | every package ok | every package ok |
| `CGO_ENABLED=1 go test -race -count=1 ./internal/...` | internal/service hit Go's default 10-minute package timeout (600.04 s); the test running at that moment, `TestConfiguredAuthenticationThrottleLoginAndGlobal`, had run for 1 s. No test failed and no data race was reported. Every other package was ok (api 302.8 s, contract 29.1 s, store 118.8 s, web 227.3 s) | not repeated in this form; see the next two rows |
| `CGO_ENABLED=1 go test -race -count=1 ./internal/service`, alone, default timeout | not run | timed out the same way (600.03 s; `TestLoginCounterDecays` had run for 0 s). So the package's race run needs more than 10 minutes on this 4-CPU machine even with nothing running beside it. These commits change no file in internal/service or in the packages it uses, so the timeout is not caused by them |
| `CGO_ENABLED=1 go test -race -count=1 -timeout 20m ./internal/...`, the bound CI uses (.github/workflows/ci.yml) | not run | every package ok (api 306.8 s, contract 29.1 s, service 624.3 s, store 121.9 s, web 228.5 s); no data race |
| `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run` (go1.26.11, the version CI's setup-go takes from go.mod) | 0 issues. The same command with `GOTOOLCHAIN=go1.26.10` cannot load the packages: `go.mod requires go >= 1.26.11 (running go 1.26.10; GOTOOLCHAIN=go1.26.10)` | the same |
| `make build`, then in e2e/ `npm ci` and `npx playwright test` (Chromium at /opt/pw-browsers/chromium) | 584 passed (7.5 min, 1 worker) | 584 passed (7.7 min, 1 worker) |

Not run: the macOS and Windows CI jobs, govulncheck, the licence check and the release workflow.

The OPS-033 follow-up ran the set below on `4bbc871`. The commit after it changes only this file, so it has the same code and tests.

| Command | On `4bbc871` |
|---|---|
| Fail-before, with the serve.go change reversed as described in the A3-F10 section (once, on the tree that became `4bbc871`) | FAIL in 3 subtests and in the loop test. Re-applied: PASS |
| `gofmt -l .` | clean |
| `go vet ./...` | ok |
| `make traceability-check` | up to date |
| `scripts/check-hygiene.sh` | ok |
| `scripts/check-action-pins.sh` | ok |
| `GOTOOLCHAIN=go1.26.10 go run github.com/zricethezav/gitleaks/v8@v8.28.0 dir . --no-banner --redact` | no leaks found (9.17 MB scanned) |
| `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run` | 0 issues |
| `go test -count=1 ./...` | every package ok (cmd/circula 61.8 s, internal/service 67.9 s) |
| `CGO_ENABLED=1 go test -race -count=1 ./cmd/circula` | ok (286.8 s); no data race |

Not run on `4bbc871`: the race tests of internal/... (it changes no file there), e2e/ (it changes no UI or API code), and the jobs listed as not run above.

## Open items for the owner or integrator

1. PRIV-02 (A3-F05): empty `challenges.reason` on erasure or document a retention exception (SEC-014/018). The reference patch is in wip-fixes/.
2. PRIV-03 (A3-F07): WAL checkpoint design.
3. A1-04-G2: rule on spec gap G2 (dispute reversals while the network is `closing`).
4. A3-F18: notice timing for member-limit cuts.
5. PR #74 or this branch, whichever merges second: regenerate docs/TRACEABILITY.md with `make traceability`.
6. The new observations above (`maintenance` problem code; `mfa-required` in the enum).
7. An independent review of `4bbc871`, the A3-F10 logging half (OPS-033), which this list carried as unblocked until the follow-up fixed it.
8. Optional nits from the triage and the reviews: an as-built note on the API-066 row (A2-19); a link from SEC-027-keys.md:13 to `revoke-sessions` (A3-F13); the unused `RecordScheduledConsistencyRun` branch (A3-F15); the A1-07 residual (one failing export delays later networks in a run); the review points in the A2-08 and A3-F10 sections.
