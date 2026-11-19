# Independent review: bind password proofs to protected writes

## Decision and exact scope

**Approved for integration at `d3fcc6831f81eea3344a03113e27532a07996bcf`; no required source or regression corrections identified.** Current-head CI and aggregate qualification remain separate merge gates. This is a narrow security repair and an additional finding beyond the historical 55-finding ledger.

The reviewed tree is `84ea230a25cf30a56f4f43b1a401abea6dc1b7fb`, directly descended from baseline `6785863252d2910553b1f4a26396f14b4405d6fa` (tree `f097b872f69b4b2c241da7499161ecbcf971c623`). The worktree was clean. The complete change consists of two production files, the new regression, and one audit note. All four working files matched their committed Git blobs. The JSON companion records their complete Git blob IDs, SHA-256 hashes and byte counts.

Method: Caveman focused adversarial review of source, transaction behavior, test validity and retained evidence. This reviewer made no source changes, ran no Go command and made no remote mutation.

## Security reasoning

| Protected operation | Guard and ordering | Existing error retained |
| --- | --- | --- |
| Begin or resume MFA enrolment | `internal/service/mfa.go:220`: re-read and compare inside the write before decrypting/returning an existing pending secret or creating a new factor | Password-field validation error |
| Confirm MFA enrolment | `internal/service/mfa.go:293`: compare before factor verification/consumption, MFA enablement, recovery-code issuance, session revocation or notices | Password-field validation error |
| Disable MFA with TOTP or recovery code | `internal/service/mfa.go:362`: compare before factor verification/consumption, factor deletion, recovery-code replacement or notices | Password-field validation error |
| Generate account recovery codes | `internal/service/recovery.go:124`: compare before persistent replacement, throttle clearing or audit insertion; errors return a nil code slice | Unauthenticated |

`checkCurrentPassword` now returns the exact encoded hash it verified. Both callers retain that value until the protected write. `DisableMFA` and `GenerateRecoveryCodes` already held their verified hash and now bind it to the write in the same way. Argon2 remains outside the write transaction.

The current SQLite implementation serializes writers and begins each update with `BEGIN IMMEDIATE` (`internal/store/sqlite.go:138`); the comparison and subsequent effects therefore share one write transaction. A mismatch returns before the sensitive work, and an error rolls back the unit (`sqlite.go:178`). Another password mutation cannot slip between the new comparison and that unit's effects. Freshly generated account-recovery plaintext may exist in local memory before the comparison, but the error path neither stores nor returns it.

Attempt reservations and throttle keys were not moved or changed: enrolment still uses the shared `recgen` family; confirmation retains its separate management-factor reservation; disabling retains the management-factor family; account-recovery generation retains `recgen`. The regression's state snapshot deliberately excludes these attempt counters. Existing focused tests exercise the throttle and sign-in separation rules.

## Regression validity and retained results

The new test obtains the affected person's actor through real `Login` and `Authenticate` calls. It then uses the existing Argon2 observer to commit a real self-service `ChangePassword` or administrator `AdminResetPassword` while the outer request is verifying its previously read hash. Its atomic one-shot flag prevents recursive observer execution during the legitimate change. The observer is disabled before subsequent assertions and the positive control.

Each change path covers six scenarios: fresh enrolment, resumption, confirmation, disabling with TOTP, disabling with a recovery code, and account-recovery generation. Setup completes before the observer is installed. The test asserts that the change occurred, the original session is revoked, the expected error category/password field is preserved, no secret or recovery material is returned, and the password/MFA/recovery/audit/notice state equals the snapshot taken after the legitimate change. For self-service changes, a new proof with the replacement password succeeds using the same factor; this rejects a false green caused by consuming the factor while reporting failure.

| Retained execution by implementation worker C | Observed terminal result | Service runtime |
| --- | --- | --- |
| Baseline production plus final new regression | All 12 subcases fail on intended stale-operation success; resumption discloses the pending secret; other scenarios also change protected state | `0.189s` |
| Fixed production, same regression | All 12 subcases pass | `0.179s` |
| Focused race selection | All 21 top-level tests pass: the new test plus 20 existing MFA/password/recovery tests; all 12 new subcases pass; no race diagnostic | `14.099s` |

Evidence: `audit-c/MFA-PASSWORD-PROOF-EVIDENCE.json` and its `before.log`, `after.log`, and `related-race.log` records. This reviewer independently verified the baseline/fixed tree and blob pins, the four current file bytes, all three log SHA-256 values, expected failure assertions, and terminal case counts. These are reviewed executions by C, not additional executions by this reviewer. No setup failure, ignored test or unconditional early pass explains the results.

The test models a request authenticated before session revocation. It does not claim a revoked session can authenticate again. Confirmation and disabling still require the valid factor supplied by the test; the defect concerns acceptance of a superseded password proof.

## Policy and scope limits

`AdminResetPassword` is byte-identical to baseline (method SHA-256 `da68577f82c8d9f918344679bcdaf61c95ff5a8f5e500aec45fd6dd2a00e8caf`). The supporting auth implementation, password hashing/test seam, and SQLite transaction implementation are also unchanged Git blobs. The diff adds no migration, ledger rule, tenant boundary, reset-authority/reviewer policy, governance rule or break-glass policy. **A3-F02 / DEC-014 remains unresolved.**

The audit note accurately limits its verification claims. Focused results do not establish whole-service, aggregate, browser, native, or current remote CI qualification. No additional mutation probe is needed for this review: the baseline failures already demonstrate sensitivity for every newly guarded operation, and the source order plus state and factor-reuse assertions cover the identified risk.
