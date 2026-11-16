# Audit 2026-10-28 Medium candidates: re-verification at ba57512

Executed in an isolated worktree of `claude/happy-noether-444cum` at `ba57512`.
It was created at `main` 4b422ad and moved to `ba57512` (`claude/happy-noether-444cum`) with `git reset --hard` before any work. Nothing was committed or pushed.
Fix applied on `claude/happy-noether-444cum` in the commit that adds this file.
Toolchain: go1.26.10. Prefixes: S/=internal/service, C/=cmd/circula.

| Finding | Verdict | Fix in patch |
|---|---|---|
| A1-01 | CONFIRMED | yes. S/departure.go `settleOnDeparture` gate |
| A1-02 / A1-11 | CONFIRMED | yes. S/networks.go `UpdatePolicy` refuses any decimals change; `TestPolicyChanges` inverted; FR-005/GOV-004 rows corrected |
| A1-03 | CONFIRMED | yes. S/disputes_response.go `decideReReview` refuses a posting outcome while the trade is `disputed` |
| A3-F01 | CONFIRMED | yes. C/premigrate.go `verifyPreMigrationBackup` decrypts and then checks |
| A3-F02 | OWNER-GATE | no code change |
| A3-F16 | CONFIRMED | yes. S/mfa.go separate session-side throttle key and decay. Amends the SEC-007 addendum throttle sentence |

Method. The PoC files were copied into place without `.txt`: S/a1_poc_test.go (build tag `a1poc`), S/a3audit_poc_test.go and C/a3audit_poc_test.go. The focused tests were run at ba57512 before any change. A refutation was attempted against the current code, specs and DECISIONS. Each fix is guarded by renamed regression tests, which assert the corrected behaviour. The PoC copies were deleted afterwards and are not in the patch. As a fail-before check, the production-code diff was reverse-applied, and all 16 behaviour-changing regression tests then failed. `TestNetworkCreationStoresScaleAtomically` passes before and after, because creation was already atomic. The diff was then re-applied.

---

## A1-01: settle-to-zero departure posts in a closed, closing, suspended or admin-required network. CONFIRMED

- **Evidence at ba57512** (`go test -tags a1poc -count=1 -run '^TestA1PoC_01...$' ./internal/service`):
  - `DepartMember err=<nil> ledger_seq 1 -> 2, departed balance=0, closure export match=false` (closed)
  - `DepartMember while suspended err=<nil> ledger_seq 1 -> 2`
  - `DepartMember in admin-required network err=<nil> ledger_seq 1 -> 2`
  - Control: `CreateAdjustment on closed network -> invalid-transaction-state: network is closed`.
- **Refutation attempt failed.**
  - `DepartMember` / `settleOnDeparture` never call `requireNetworkActive` or `networkOpenErr`. `postAdjustment` -> `postTx` has no state gate.
  - `lifecycle` derives `closing` and `AdminRequired`, but departure does not consult them.
  - Specs: GOV-023 "MUST NOT accept ... postings"; GOV-025 b freeze plus the acceptance "after closure the network refuses postings"; FR-068 "adjustments ... refused"; GOV-020 "no value-moving action". DEC-049 says nothing on network state.
- **Owner gate?** No.
  - Spec gap G3 (refuse vs fall back to `retain`) only chooses between two options, and neither may post.
  - Refusing keeps the network's configured departure rule (DOM-024, DEC-049). Falling back to `retain` would silently override that rule, so it would need an owner decision.
  - A zero-balance or `retain` departure is unaffected.
  - The end-of-retention path (`EraseClosedNetworks`) already departs members without settlement.
- **Fix.** In `settleOnDeparture`, after `acc.Balance == 0` returns early, call `p.requireNetworkActive()`. The whole departure unit rolls back, including the proposal cancellations. No migration. The doc comment of `DepartMember` was updated.
- **WIP `departure.patch.txt`.** Its A1-01 part is the same change and its tests were reviewed and adopted. Its A1-06 part (paging past 500 open proposals) was left out: it is a Low finding outside this task.
- **Regression tests** (new S/departure_test.go): `TestDepartureSettlementRefusedAfterClosure` (was PoC_01, plus admin removal and a zero-balance departure; the final export still matches), `TestDepartureSettlementFrozenFromClosureDate` (closing; 1 ms before the date still posts), `TestDepartureSettlementRefusedWhileSuspended` (was PoC_01b; proposals stay open on refusal; posts after resume), `TestDepartureSettlementRefusedWhenAdminRequired` (was PoC_01c).
- After the fix, the original PoCs 01/01b/01c PASS: `DepartMember err=invalid-transaction-state: network is closed ledger_seq 1 -> 1 ... closure export match=true`.

## A1-02 / A1-11: a scheduled decimals change bypasses the lock; FR-005/GOV-004 status was inflated. CONFIRMED

- **Evidence at ba57512** (`TestA1PoC_02_ScheduledDecimalChangeAfterPosting`):
  - `after notice: decimals=0 ledger_seq=1; payee balance 2500 minor units now reads 2500 credits (was 25.00)`
  - `decimal scale changed from 2 to 0 after 1 posting(s): persisted amounts reinterpreted x100`
- **A1-11.** S/admin_test.go `TestPolicyChanges` asserted that a change from 2 to 3 after creation succeeds, while the FR-005 row said `built`.
- **Refutation attempt failed.**
  - networks.go guarded with `Decimals != cur && LedgerSeq > 0` only at request time. `networkSelect` picks up the scheduled version later with no re-check.
  - DOM-031, FR-005, GOV-004 and DAT-049 all require the scale to be immutable from network creation, "including before the first posting". This is an accepted spec, so no owner gate.
- **Fix.** In S/networks.go `UpdatePolicy`, refuse any `in.Policy.Decimals != p.net.Policy.Decimals` with `validation-failed decimals: cannot change after the network is created`. This covers immediate and scheduled changes and every network status. No migration.
  - The `/setup` web form and the API pass the scale on create (`CreateNetwork`, one unit), and the admin policy form re-sends the current scale, so neither flow breaks.
- **Tests.**
  - `TestPolicyChanges` inverted: refused before any posting; still refused after it; one audit event.
  - New S/decimals_test.go:
    - `TestScheduledDecimalChangeCannotReinterpretPostings` (was PoC_02; asserts the scheduled change is refused, and after the posting and 15 days the scale is 2, the balance reads 25.00, stored scales `[2]`)
    - Adopted after review from WIP `decimals.patch.txt`: `TestDecimalsFixedFromCreation`, `TestDecimalsLockedWhateverIsStored`, `TestNetworkCreationStoresScaleAtomically`, `TestDecimalsLockUnderConcurrentWrites`. The concurrency test ran `-count=10` with no flake.
- **WIP usable?** Partly.
  - Its service guard and tests were adopted.
  - **Not adopted:** its store change (`networkSelect` pins `Decimals` to the version-1 scale) and `TestStoredScaleChangeNeverApplies`. That pin would reinterpret every legacy network that, under the old rule, legally changed scale immediately before its first posting and then posted under the new scale. For example, the old `TestPolicyChanges` flow 2->3 would be read back at 2 (x10).
  - **Residual:** a scale change that an earlier build stored with a future `effective_at` still applies when it comes into force. No released deployments are known. Before upgrading, check with:
    `SELECT p.network_id, p.version FROM network_policies p WHERE json_extract(p.policy_json,'$.decimals') <> (SELECT json_extract(c.policy_json,'$.decimals') FROM network_policies c WHERE c.network_id=p.network_id AND c.version=1);`
    Neutralising such rows is an owner or operator call.
- **Docs (A1-11).**
  - 04 FR-005 row and 10 GOV-004 row text corrected. Dated implementation notes were added and the 2026-10-24 notes kept.
  - TRACEABILITY rows for DOM-031 and FR-005 updated.
  - The DOM-031 and DAT-049 status rows and their summary counts ("required unimplemented correction") are left for the integration owner, because of the residual above.
- After the fix, the original PoC_02 panics at its setup line 148 `tu.Must(w.setPolicy(...decimals 0, notice 14))` with `validation-failed: decimals: cannot change after the network is created`. The PoC assumed scheduling succeeds; the regression test asserts the refusal instead.

## A1-03: a re-review reverses a trade held by another live dispute. CONFIRMED

- **Evidence at ba57512.**
  - PoC_03:
    - `RetryDisputePosting -> validation-failed: amount: must be between 1 and the remaining reversible amount 5.00`
    - `ResolveDispute on posting_pending -> invalid-transaction-state: dispute is posting_pending; resolve not allowed`
    - `payee departure -> blocked: departure is blocked; trade status disputed`
  - PoC_03b: `trade=reversed d2=open; opener withdraw -> invalid-transaction-state: transaction is reversed; close_dispute not allowed; ScheduleClosure -> <nil>` then `FR-068: closure scheduled while dispute ... is still open`.
- **Refutation attempt failed.**
  - Serial disputes are accepted (`OpenDispute` needs only `completed`; the `dispute_live` index blocks concurrent disputes only).
  - `decideReReview` calls `reverse()`, which accepts `TxDisputed`.
  - The `reverse()` callers are: `ResolveDispute` and `RetryDisputePosting` (both act on the live dispute itself), `ReverseTransaction` (`completed` only, per DEC-052), and the re-review. The re-review is the only path that reverses under another dispute.
- **Owner gate?** No.
  - DEC-052 (accepted): "A `disputed` trade is reversed only by the dispute decision", to avoid two reviewers acting on one trade.
  - The fix applies that rule and moves no money. Spec gap G1 (should serial disputes be allowed at all?) stays an owner question and is not needed for the fix.
- **Fix.** In S/disputes_response.go `decideReReview`, inside `amount > 0`, refuse with `invalid-transaction-state` when `t.Status == domain.TxDisputed`.
  - The re-review stays `requested`.
  - `confirmed` is still allowed.
  - Once the later dispute is decided or withdrawn, the re-review can proceed.
  - No migration.
- **Not done (optional defence in depth):** `RetryDisputePosting` using `min(ReversalAmount, remaining)`, and adding a live-dispute check to `closureBlockers` and `departBlockers`. After the fix nothing produces the bad state, but existing bad rows (pre-fix data) would not be repaired.
- **WIP `disputes.patch.txt`:** not usable for A1-03. It covers A1-04 (`reverse()` AdminRequired gate) and A1-09 (`transaction.reversed` audit) only, and it is incomplete.
- **Tests** (new S/dispute_rereview_test.go):
  - `TestReReviewWaitsWhileAnotherDisputeIsPostingPending` (was PoC_03): a partial or upheld re-review is refused with nothing changed; the retry succeeds after a limit raise; departure is unblocked; the re-review can then only confirm; the ledger verifies.
  - `TestReReviewCannotReverseUnderAnOpenDispute` (was PoC_03b): the re-review is refused; closure stays blocked; withdrawal works; the re-review then reverses; closure schedules.
- After the fix, the original PoC_03/03b fail at their setup step `re-review decision: invalid-transaction-state: another dispute on this trade is still undecided ...`. That is intended; the regression tests carry the corrected flow.

## A3-F01: the pre-migration gate integrity-checks the ciphertext, so encrypted deployments never migrate. CONFIRMED

- **Evidence at ba57512.** `TestA3Audit_MigrateWithEncryptionKey` and `TestA3Audit_PreMigrationSafeguardEncrypted` both fail with:
  `circula: pre-migration backup .../circula-20261104T072932Z.db.enc did not verify; migrations not applied: integrity check of ....db.enc: file is not a database (26)`
- **Refutation attempt failed.**
  - `backupTo` writes `.db.enc` whenever `backup-encryption-key` is set.
  - `verifyPreMigrationBackup` ran `integrityCheck(ctx, path)` on that file.
  - `serve` with `migrate-mode=auto` uses the same gate.
  - OPS-019/SEC-028 and docs/ops/backup-restore.md:32 require encrypted pre-migration backups. OPS-023 requires the verified gate.
- **Fix.** In C/premigrate.go `verifyPreMigrationBackup`, when `m.Encrypted`:
  1. Make a private `os.MkdirTemp(dir, tempDirPrefix)` (0700) with `defer secureRemoveAll`.
  2. `decryptFile` into it.
  3. Require the decrypted sum and size to equal `m.PlaintextSHA256` / `m.PlaintextBytes`.
  4. Run `integrityCheck` on that plaintext.
  This mirrors `cmdRestore`. The ciphertext hash, manifest HMAC and schema checks are unchanged. No migration. The WIP patch has nothing for A3-F01.
- **Tests** (new C/premigrate_encrypted_test.go):
  - `TestMigrateWithEncryptionKey` (was the PoC, plus: no pending migrations, exactly 1 `.db.enc`, no plaintext or work dir left).
  - `TestPreMigrationSafeguardEncrypted` (was the PoC, plus: a wrong key is refused with "decrypt", a manifest plaintext-checksum mismatch is refused, no plaintext left after either).

## A3-F02: break-glass admin can make lasting and value-moving changes. OWNER-GATE

- **Evidence at ba57512.** The behaviour is real. `TestA3Audit_BreakGlassLastingChanges` fails in all 4 subtests:
  - `break-glass admin posted a permanent adjustment ... (amount 5000)`
  - `set ben's credit limit to 1125899906842624 with no expiry`
  - `reset ben's password; the temporary password still signs in after the session expired`
  - `reversed completed trade ...`
  `TestA3Audit_BreakGlassApprovesImmediately` fails with `approved the founder's 50,000.00 request seconds after the grant ...; a System-granted admin must wait 24h0m0s`.
- **Why it is owner-gated, not a defect fix:**
  - docs/specs/addenda/SEC-013.md Decision 7 gives the full admin role. "Not done, owner decisions" #4 says: "A break-glass administrator can use all administrator powers ... Decide whether to narrow this."
  - DEC-014 is `open` (docs/specs/13-assumptions-risks-decisions.md:34,75,237; docs/DECISIONS.md:830).
  - The immediate-approval part is accepted: docs/specs/addenda/SEC-011-owner-decision.md:66 says "Break-glass administrators are not covered by this check" (A3-01).
  - SEC-011-owner-decision.md:9-13 leaves dual control for reversals, limits and disputes to the owner.
  - SEC-008 (09:197) permits a password reset under break-glass.
  - GOV-021 constrains the operator, is `not built`, and its scope is governance open question 9.
- **Agree with the skeptic** that no accepted rule is broken. **Agree with the impact lens** that it is a real, value-moving exposure.
- **Owner decision needed:** a break-glass allow-list vs calling `requireLasting` on value, credential and approval paths (`createAdjustment`, `ApproveAdjustment`, `setCreditLimit`, `AdminResetPassword`, `ReverseTransaction`, `requireDecider` callers, `DecideJoinRequest`, `Suspend/ResumeNetwork`, invitations, announcements).
- **Doc-only nit (Info, not changed):** the `bgAdmin` field comment in S/principal.go:18-19 ("must not leave lasting changes behind") overstates what is implemented; `requireLasting`'s own comment says "role changes, policy changes".

## A3-F16: a stolen session keeps the owner out of MFA sign-in. CONFIRMED

- **Evidence at ba57512.** `TestA3Audit_StolenSessionLocksOwnerOutOfMFA` fails in rounds 0, 1 and 2 with: `owner with the right password and code is refused: rate-limited: too many attempts; try again later`.
- **Refutation attempt failed.**
  - `DisableMFA` reserves the shared `"\x1fmfa:"+pid` counter before checking the password.
  - `RegenerateRecoveryCodes` needs only a session plus a code.
  - `mfaReserve` re-locks on every failure at 5 or more (there is no `readThrottle` half allowance, unlike the A3-02a login fix).
  - `CompleteMFA` uses the same key. One bad call per 15 min re-arms the lock indefinitely.
  - SEC-006 MUST: "MUST NOT lock out the legitimate owner permanently". The SEC-007 addendum's own intent is "Never a permanent lock".
- **Owner gate?** Not a financial or product decision. The fix does amend one engineering line of the SEC-007 addendum ("The same counter covers confirm, disable, regenerate and CompleteMFA"). That amendment is recorded in the addendum for reviewer sign-off.
  - Half allowance alone is not enough: with 5 attempts and 3 restored, 2 bad calls per window still re-lock. So a separate key is required.
- **Fix** (S/mfa.go, S/mfa_login.go). No migration; it uses existing `login_throttle` rows.
  - `CompleteMFA` counts on `mfaKey` only.
  - The session-side calls (`ConfirmMFAEnrolment`, `DisableMFA`, `RegenerateRecoveryCodes`) use `reserveMFAManage`: refused while `mfaKey` **or** `mfaManageKey` (`mfaKey+" manage"`) is locked, counted on `mfaManageKey` only. So a sign-in lock still covers them and `TestMFAThrottle` is unchanged.
  - Both keys go through `reserveFactorAttempt`, which uses `readThrottle` (24 h decay and half allowance after a lock, as A3-02).
  - A success clears both keys (`DeleteThrottleFamily(mfaKey)`). `OperatorUnlockLogin` and `ResetMFA` already use `DeleteThrottleFamily(mfaKey)`, which now also covers the manage key. Person ids are UUIDs, so the A3-F06 byte/char issue does not apply.
- **Not done:** the `recgen` key (`reserveAttempt`) has the same renewal pattern, but it only gates session-side password checks, not sign-in. The "doubling to 24 h" docs mismatch is A3-F21.
- **WIP:** none for A3-F16.
- **Docs.** The SEC-007 addendum throttle bullet was amended (dated). docs/dev/mfa.md throttle-key sentence updated.
- **Tests** (new S/mfa_throttle_test.go):
  - `TestStolenSessionCannotLockOwnerOutOfMFA` (was the PoC, hardened: 8 bad disable and 8 bad regenerate calls every window; the session-side calls are throttled; the owner signs in for all 3 rounds).
  - `TestMFASignInLockNotRenewableByOneFailure`.
  - `TestMFASessionThrottleBoundAndUnlock` (5 bad regenerate calls lock regenerate and disable even with the right code; sign-in is unaffected; the sign-in success and `circula unlock` clear it).

---

## Files touched

Production:
- `cmd/circula/premigrate.go`
- `internal/service/departure.go`
- `internal/service/networks.go`
- `internal/service/disputes_response.go`
- `internal/service/mfa.go`
- `internal/service/mfa_login.go`

Tests:
- `internal/service/admin_test.go` (`TestPolicyChanges` inverted)
- New: `internal/service/departure_test.go`, `internal/service/decimals_test.go`, `internal/service/dispute_rereview_test.go`, `internal/service/mfa_throttle_test.go`, `cmd/circula/premigrate_encrypted_test.go`

Docs:
- `docs/specs/04-functional-requirements.md` (FR-005 row and note)
- `docs/specs/10-governance-operations.md` (GOV-004 row and note)
- `docs/TRACEABILITY.md` (DOM-031 and FR-005 rows)
- `docs/specs/addenda/SEC-007.md` (throttle amendment)
- `docs/dev/mfa.md`

Not touched: e2e/, .github/, internal/web templates, migrations.

## Tests run

| Command | Result |
|---|---|
| Baseline PoCs at ba57512: `go test -tags a1poc -count=1 -run '^(TestA1PoC_01_...\|01b\|01c\|02\|03\|03b)$' -v ./internal/service/` | 6 FAIL (confirms A1-01, A1-02, A1-03; output lines above) |
| Baseline PoCs: `go test -count=1 -run '^(TestA3Audit_MigrateWithEncryptionKey\|TestA3Audit_PreMigrationSafeguardEncrypted)$' ./cmd/circula/` | 2 FAIL (A3-F01) |
| Baseline PoCs: `go test -count=1 -run '^(TestA3Audit_BreakGlassLastingChanges\|TestA3Audit_BreakGlassApprovesImmediately\|TestA3Audit_StolenSessionLocksOwnerOutOfMFA)$' ./internal/service/` | 3 FAIL (A3-F02 4/4 subtests; A3-F16 3/3 rounds) |
| Post-fix original PoCs | A1 01/01b/01c PASS. A3-F01 both PASS. A3-F16 PASS. A1 02/03/03b stop at a setup step the fix refuses (intended). A3-F02 unchanged (owner-gated). |
| Fail-before: production diff reverse-applied, then the new regression tests run | 14 service tests and 2 cmd tests FAIL. `TestNetworkCreationStoresScaleAtomically` passes before and after (coverage test). Diff then re-applied. |
| Focused regression tests after the fix (service and cmd) | all PASS |
| `go test -count=10 -run '^(TestDecimalsLockUnderConcurrentWrites\|TestNetworkCreationStoresScaleAtomically)$' ./internal/service/` | PASS (no flake) |
| `gofmt -l ./cmd ./internal` | clean |
| `go vet ./...` | PASS |
| `go test -count=1 ./internal/service ./internal/store` | ok service 32.3 s, ok store 14.6 s |
| `go test -count=1 ./cmd/circula ./internal/web ./internal/api ./internal/domain` | ok cmd/circula 42.3 s, web 19.5 s, api 14.9 s, domain 0.08 s |

Not run: the race suite, e2e (Playwright), CI.

## Open items for the owner or integrator

1. A3-F02: whether to narrow break-glass powers (SEC-013 addendum Not done #4, DEC-014).
2. A1-02 residual: what to do with any scale change already stored with a future effective time by an earlier build (detection query above). Then the DOM-031 and DAT-049 status rows.
3. A1-01 / G3: refusal was chosen. A fallback to `retain` would need an owner decision.
4. A1-03 / G1: whether a second dispute after a dismissal is allowed at all. Also the optional defence in depth (retry `min`, live-dispute blockers).
5. A3-F16: reviewer sign-off on the amended SEC-007 addendum throttle sentence.
