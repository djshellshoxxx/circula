# A1 audit report: Circula ledger and money core

Auditor: A1 (read-only code auditor, one of six). Report date: 2026-10-28.

## 1. Scope, SHA and what was reviewed

- Repository: `djshellshoxxx/circula` (Go). Audited SHA: `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f` (Merge PR #11), detached worktree `/home/user/audit/circula-a1`.
- Scope A1 (ledger and money core):
  - `internal/domain`, all files.
  - `internal/store`: `sqlite*.go`, `iface*.go`, `migrations/0001`..`0015`, `migrate.go`, `sqlite_verify.go`, `inspect.go`.
  - `internal/histbal`.
  - `internal/service` money paths: `posting.go`, `accounts.go`, `histbal.go`, `transactions.go` (propose, accept, decline, cancel, expire), `reversal.go`, `adjustments.go`, `dualcontrol.go`, `dualcontrol_operator.go`, `disputes*.go` (money effects, re-review), `closure.go` and `closure_schedule.go` (final settlement, final export), `departure.go`, `networks.go` (policy, decimals), `membership.go` (credit limits), `verify.go`, `export*.go` (correctness only), `reads.go`, `tx_queries.go`, `idempotency.go`, `audit.go` (hash chain), `principal.go`, `freeze.go`, `join.go` (sign-up grant), `purge.go`.
  - The expiry job loop in `cmd/circula/serve.go` (406-417) was read to judge the impact of A1-05. The rest of `cmd` was not audited.
- Specs: 00-conventions (LED-1..LED-7), 03-domain-model, 04-functional-requirements (balances, trading, admin money actions, disputes, closure), and 06-data-ledger-audit. GOV-004/018/019/020/023/025 in 10-governance-operations were cross-checked where they constrain postings. DECISIONS DEC-031..DEC-079 were read where they touch the ledger. Addenda SEC-011 and SEC-011-owner-decision were also read.
- Prior audits checked: SECURITY-AUDIT.md, -2, -3, CODE-AUDIT.md, COMPLETENESS.md, RESUME-2026-10-24.md (including the 2026-10-28 section) and CONTINUATION.md. None of the findings below appears in them (grep for decimals/scheduled, serial/second dispute and starvation found nothing). Related known items are cross-referenced in each finding.
- Accepted rules not re-litigated: A3-01 (independent approver), and append-only entries and audit through SQLite triggers. Owner-pending items A3-04, A3-07, A3-10, A3-11, SA2-12 and the SEC-011 questions are referenced only where relevant.

## 2. Method

1. Read the specs (status tables and normative text) and the prior audits first, and built a requirement list for the scope.
2. Traced every posting path end to end. Every posting goes through `postTx`, whose callers are:
   - accept;
   - `reverse` (admin reversal, dispute resolve, retry, re-review);
   - `postAdjustment` (adjustment, dual-control approval, departure settlement);
   - `postSignupGrant`.
   For each path I checked the authorization, network-state gate, LED-5, idempotency, audit and state-machine guards in the same write unit.
3. Checked the DB triggers and constraints in `0001_init.sql`, `0002`, `0005` and `0015` against LED-1/2/7, DOM-010 and DOM-015.
4. Checked concurrency: every `Update` takes an in-process gate plus `BEGIN IMMEDIATE`. I looked for decisions split across write units (view then update), for numeric overflow (balance bound 2^62, policy bound 2^50) and for policy binding events (GOV-003 scheduled versions with `asOf`).
5. Wrote throwaway PoC tests in `internal/service/a1_poc_test.go`, behind build tag `a1poc` so the default suite is unaffected.
   - Each `TestA1PoC_NN*` asserts the behaviour the spec requires. A FAIL at this SHA confirms finding A1-NN.
   - Each `TestA1Check_*` is a passing check that supports an OK conformance verdict where the existing suite does not pin a boundary.
6. Ran the baseline suite, the race suite and vet for the in-scope packages (section 8).

## 3. Findings table

| ID | Sev | Confidence | Category | Spec IDs | file:line | Summary |
|---|---|---|---|---|---|---|
| A1-01 | Medium | CONFIRMED | correctness / spec | FR-068, GOV-025 b, GOV-023, GOV-020, DOM-024 | internal/service/departure.go:23-66, 129-158 | A settle-to-zero departure posts an adjustment even when the network is closed, closing, suspended or administrator-required. After closure this breaks the certified final export checksum. |
| A1-02 | Medium | CONFIRMED | correctness / regression of as-built guard | FR-005, GOV-004, DOM-031, DAT-049, DOM-030 | internal/service/networks.go:242, 258-291; internal/store/sqlite_network.go:27-32 | The decimals guard (`LedgerSeq > 0`) is evaluated only when a change is requested. A change scheduled with notice takes effect after postings exist, so every persisted amount is reinterpreted (×10^Δ) permanently. |
| A1-03 | Medium | CONFIRMED | state machine / dispute money effects | DOM-018, DOM-020, DOM-021, DOM-024, FR-068, FR-075, FR-078, DEC-052 | internal/service/disputes_response.go:274-281; disputes_decide.go:177-199; disputes.go:52-54; closure.go:13-23 | A re-review posts a reversal on a trade that another live dispute holds as `disputed`, and serial disputes are allowed. A `posting_pending` dispute can then never post or close. An open dispute on a reversed trade cannot be withdrawn and no longer blocks closure or departure. |
| A1-04 | Low | CONFIRMED | spec (network-state gate) | GOV-020 | internal/service/reversal.go:84-86; disputes_decide.go:107-115, 188 | `ResolveDispute` and `RetryDisputePosting` post reversals in an administrator-required network. `reverse()` checks only `Suspended()`. `DecideReReview` applies the full gate, so the code is inconsistent. |
| A1-05 | Low | CONFIRMED | liveness | FR-034, DOM-013, OPS-033 | internal/service/transactions.go:369-376; internal/store/sqlite_tx.go:156-159; cmd/circula/serve.go:406-417 | The expiry sweep takes the oldest N expired proposals across all networks and skips frozen ones. With 500 or more expired proposals in frozen networks, no other network's proposals are ever marked `expired`. |
| A1-06 | Low | CONFIRMED | correctness | DOM-024, FR-014 | internal/service/departure.go:101-102 | Departure cancels only the newest 500 open proposals. The rest stay `proposed` with a departed party and keep the counterparty's reservation until expiry. |
| A1-07 | Low | CONFIRMED | atomicity / closure | GOV-025 e, FR-068 | internal/service/closure_schedule.go:182-197, 295-312; internal/store/sqlite_closure.go:75-79 | The closure completes in one unit and the final export is written in later units. If the export fails once, the closure is never listed again, so no final checksum or reconciliation is ever recorded. |
| A1-08 | Info | CONFIRMED | policy versioning | SEC-011 (operator path, SA2-01b), GOV-003 | internal/service/dualcontrol_operator.go:40-41 | `OperatorSetDualControl` writes version effective+1, which collides with a scheduled version. The operator cannot change the threshold while any policy change is pending. |
| A1-09 | Info | CONFIRMED | audit linkage | DOM-035, FR-081, DAT-029 | internal/service/disputes_decide.go:152-171; disputes_response.go:295-298 | A reversal transaction posted by a dispute decision, retry or re-review has no audit event naming its id. Only the dispute is the target. |
| A1-10 | Info | CONFIRMED | freeze rule | OPS-033, SEC-011 D5 | internal/service/dualcontrol.go:282-307 | `ExpirePendingAdjustments` writes status changes and audit events into frozen networks. The proposal expiry sweep and closure finalisation skip frozen networks. |
| A1-11 | Low | CONFIRMED (code + test read) | conformance (status inflation) | FR-005, GOV-004 | docs/specs/04-functional-requirements.md:25; docs/specs/10-governance-operations.md:32; internal/service/admin_test.go:186-208 | FR-005 and GOV-004 are marked `built`, but creation-time immutability is unimplemented, as their own body notes say. `TestPolicyChanges` asserts the non-conforming behaviour: decimals 2 to 3 after creation succeeds. A1-02 shows even the interim guard can be bypassed. |

Counts:

| Severity | Count | Findings |
|---|---|---|
| Critical | 0 | |
| High | 0 | |
| Medium | 3 | A1-01, A1-02, A1-03 |
| Low | 5 | A1-04, A1-05, A1-06, A1-07, A1-11 |
| Info | 3 | A1-08, A1-09, A1-10 |

All 11 are CONFIRMED. Speculative items are in the appendix (section 7a).

## 4. Finding details

### A1-01 (Medium, CONFIRMED): settle-to-zero departure ignores the network state

- **Code.**
  - `DepartMember` (departure.go:23-99) opens with write scope.
  - It never calls `requireNetworkActive` or `networkOpenErr`.
  - `settleOnDeparture` (129-158) then calls `postAdjustment`, which calls `postTx`.
  - Compare `createAdjustment` (adjustments.go:90), `ReverseTransaction` (reversal.go:35) and `ApproveAdjustment` (dualcontrol.go:91). All of them refuse on a closed, suspended or administrator-required network.
  - `lifecycle` (closure_schedule.go:41-55) leaves `closed` untouched and derives `closing` and `AdminRequired`. Neither is consulted by departure.
- **Scenario.**
  1. A network with `departure_rule=settle-to-zero` is closed through ScheduleClosure and then FinaliseClosures. Its final export checksum is recorded.
  2. During the 90-day read-only period a still-active member calls DepartMember.
  3. Result: a completed `adjustment` posts, `ledger_seq` goes 1 to 2, and `VerifyClosureExport` returns `Match=false`.
  4. The same happens while the network is administrator-suspended (GOV-023: "MUST NOT accept new ... postings") and in the administrator-required state (GOV-020: "no value-moving action is allowed").
  5. By the same code path it also happens in the derived `closing` state; that variant is LIKELY and was not run separately.
  6. An administrator removal (`MemberID` set) below the dual-control threshold is equally ungated.
- **Evidence.** `go test -tags a1poc -count=1 -run 'TestA1PoC_01' -v ./internal/service/`
  ```
  control: CreateAdjustment on closed network -> invalid-transaction-state: network is closed
  DepartMember err=<nil> ledger_seq 1 -> 2, departed balance=0, closure export match=false
  control: CreateAdjustment while suspended -> invalid-transaction-state: network is suspended: ...
  DepartMember while suspended err=<nil> ledger_seq 1 -> 2
  DepartMember in admin-required network err=<nil> ledger_seq 1 -> 2
  ```
  Tests: `TestA1PoC_01_SettleToZeroPostsAfterClosure`, `TestA1PoC_01b_SettleToZeroPostsWhileSuspended`, `TestA1PoC_01c_SettleToZeroPostsWhenAdminRequired`.
- **Fix.** In `settleOnDeparture`, when a settlement posting is needed (`DepartureRule == settle-to-zero` and `Balance != 0`), call `p.requireNetworkActive()` first. Refuse with the network-state error, or let the departure proceed under `retain` semantics only if the spec owner decides so. The spec does not say which; see spec gap G3. Add regression tests for closed, closing, suspended and administrator-required networks.
- **KNOWN cross-ref.** Not the same as A3-07, which is owner-pending and concerns approval of voluntary settle-to-zero forgiveness in an active network. Not SA2-02 either, which is fixed: the threshold check for admin-initiated removal is at departure.go:143.

### A1-02 (Medium, CONFIRMED): a scheduled policy change bypasses the decimals lock

- **Code.**
  - `UpdatePolicy` (networks.go:242) refuses a decimals change only if `p.net.LedgerSeq > 0` at the time of the call.
  - With a notice period (`days > 0`, explicit `NoticeDays` or an adverse bundle), the new version is appended with a future `effective_at` (279-280).
  - `networkSelect` (sqlite_network.go:27-32) later picks it up as the effective policy with no re-check.
  - Once it is effective, the guard blocks any change back, because postings now exist.
- **Scenario.**
  1. A new network has decimals 2 and no posting yet.
  2. An admin submits a decimals change to 0 with `notice_days=14`. It passes validation.
  3. Members trade 25.00 (2500 minor units).
  4. On day 15 the network shows decimals 0 with `ledger_seq=1`. The payee's 2500 minor units now read as 2500 credits.
  5. Limits, `max_tx_amount`, prices, open proposals and exports are reinterpreted the same way. There is no way back.
- **Evidence.** `TestA1PoC_02_ScheduledDecimalChangeAfterPosting`
  ```
  control: immediate decimals change after posting -> validation-failed: decimals: cannot change after the first posting
  after notice: decimals=0 ledger_seq=1; payee balance 2500 minor units now reads 2500 credits (was 25.00)
  ```
- **Fix.** Implement the accepted DOM-031, DAT-049 and GOV-004 correction: refuse any decimals change after network creation, in both the scheduled and the immediate path. This removes the bypass. As an interim minimum, refuse a decimals change whenever a version would take effect later, or re-validate at activation. Update `TestPolicyChanges` (see A1-11).
- **KNOWN cross-ref.** The DOM-031 and DAT-049 "required unimplemented correction" is known: decimals can change while proposals, limits or prices exist. The scheduled-change bypass of the as-built "after first posting" guard is new.

### A1-03 (Medium, CONFIRMED): re-review reverses a trade held by another live dispute

- **Code.**
  - `OpenDispute` (disputes.go:52) only requires the trade to be `completed`. After a first dispute is dismissed, a second dispute on the same trade is accepted. The `dispute_live` unique index blocks concurrent disputes only, not serial ones.
  - `decideReReview` (disputes_response.go:274-281) calls `reverse()`. `reverse()` accepts `TxDisputed` (reversal.go:90), so the re-review posts against a trade that the second dispute holds.
  - DEC-052 states the intended rule: "A disputed trade is reversed only by the dispute decision ... avoids two reviewers acting on one trade." The admin path enforces it (reversal.go:48); the re-review path does not.
  - `RetryDisputePosting` (disputes_decide.go:188) re-posts the frozen `d.ReversalAmount`.
  - `ResolveDispute` refuses `posting_pending`.
  - `WithdrawDispute` (disputes.go:197) needs the trade to be `disputed`.
  - `closureBlockers` (closure.go:13-23) and `departBlockers` (departure.go:161-177) look only at transaction status.
- **Scenario 1: stuck `posting_pending`.**
  1. D1 is dismissed. The payee spends the credit. D2 is opened and upheld, but blocked by LED-5, so it goes to `posting_pending` with amount 1000.
  2. A re-review of D1 is partially upheld for 500 and posts.
  3. Even with a huge limit override, `RetryDisputePosting` fails forever ("must be between 1 and the remaining reversible amount 5.00"), and `ResolveDispute` is refused.
  4. The trade stays `disputed`, the payee's departure is blocked, and the network can never close.
- **Scenario 2: open dispute on a reversed trade.**
  1. D1 is dismissed and D2 is opened.
  2. The re-review of D1 is upheld, so the trade becomes `reversed` while D2 is still `open`.
  3. The opener's withdrawal fails with "transaction is reversed; close_dispute not allowed".
  4. `ScheduleClosure` succeeds even though a dispute is open (FR-068).
- **Evidence.** `TestA1PoC_03_PostingPendingStuckAfterReReview`, `TestA1PoC_03b_OpenDisputeOnReversedTradeDoesNotBlockClosure`
  ```
  RetryDisputePosting -> validation-failed: amount: must be between 1 and the remaining reversible amount 5.00
  ResolveDispute on posting_pending -> invalid-transaction-state: dispute is posting_pending; resolve not allowed
  payee departure -> blocked: departure is blocked; trade status disputed
  trade=reversed d2=open; opener withdraw -> invalid-transaction-state: ...; ScheduleClosure -> <nil>
  ```
- **Fix.**
  - Main fix: in `decideReReview`, refuse a posting outcome while `t.Status == TxDisputed`, mirroring DEC-052. A re-review can then post only when no other dispute holds the trade.
  - Defence in depth: in `RetryDisputePosting`, recompute the amount as `min(d.ReversalAmount, remaining)` for `upheld`, or refuse with a clear state. Add the live-dispute table to `closureBlockers` and `departBlockers`, not only the transaction status.
  - Whether a second dispute after a dismissal is allowed at all is spec gap G1.
- **KNOWN cross-ref.** None. DEC-040 (cumulative cap) and DEC-052 are related accepted decisions.

### A1-04 (Low, CONFIRMED): dispute reversals post in the administrator-required state

- **Code.**
  - `reverse()` checks only `net.Suspended()` (reversal.go:84-86).
  - `ResolveDispute` and `RetryDisputePosting` (disputes_decide.go:33-49, 177-199) never call `networkOpenErr`, so `AdminRequired` is ignored.
  - `DecideReReview` calls `requireNetworkActive` before posting (disputes_response.go:275).
- **Scenario.** The network has no active administrator. A member with only the reviewer role resolves an open dispute as upheld. A reversal posts (`ledger_seq` 1 to 2), while proposals are refused with "administrator required".
- **Evidence.** `TestA1PoC_04_ReversalPostsInAdminRequiredNetwork`: "ResolveDispute upheld -> status resolved, ledger_seq 1 -> 2".
- **Reachability.** The state is set up as in the existing `TestAdminRequiredReadOnly`, which uses raw SQL. I found no service path that leaves zero active administrators, because `lastAdminGuard` covers suspend, revoke and depart. Hence Low.
- **Fix.** Call `networkOpenErr`, or a dispute-specific variant that permits the `closing` state if the owner decides so (spec gap G2), before `reverse()` in both functions.

### A1-05 (Low, CONFIRMED): expiry sweep starved by a frozen network

- **Code.**
  - `ListExpiredProposals(now, limit)` orders by `expires_at, id` across all networks (sqlite_tx.go:156-159).
  - `ExpireProposals` skips frozen networks without counting them (transactions.go:374-376).
  - The job loops while `n == 500` (serve.go:406-417), so the first run that returns fewer stops it.
- **Scenario.** A frozen network holds at least 500 elapsed proposals. Every run returns those same 500, expires 0 and stops. Elapsed proposals in healthy networks never get `expired`, nor the audit event or status history FR-034 requires.
- **Money effect: none.** Reservations and acceptance already respect `expires_at` (store/sqlite_account.go:73-80; transactions.go:227).
- **Evidence.** `TestA1PoC_05_ExpiryStarvedByFrozenNetwork` uses limit 2 to show the mechanism: "run 0..2 expired 0; healthy network's elapsed proposal is still proposed".
- **Fix.** Filter frozen networks in SQL (`network_id NOT IN (...)`), or keyset-paginate past skipped rows. Do not let skipped rows count toward the page.

### A1-06 (Low, CONFIRMED): departure leaves proposals beyond 500 open

- **Code.** `cancelOpenProposals` reads one page with `Limit: 500` (departure.go:102) and does not loop. `finaliseOne` does loop (closure_schedule.go:215-239).
- **Scenario.** 501 open proposals involve member D: the 200 cap applies per proposer and the 5-request cap per pair, so this is reachable with several counterparties. D departs, and one proposal stays `proposed` with a departed party. It can no longer be accepted, but it keeps the counterparty's reservation until expiry. This contradicts DOM-024 and FR-014: "pending proposals are cancelled".
- **Evidence.** `TestA1PoC_06_DepartureLeavesProposalsBeyond500`: "after departure: 1 proposals still `proposed`; payer reserved 1".
- **Fix.** Loop until a page returns fewer than the limit, as `finaliseOne` does.

### A1-07 (Low, CONFIRMED): the final closure export is never retried

- **Code.**
  - `FinaliseClosures` calls `finaliseOne`, which commits status `closed` and `closed_at`.
  - It then calls `recordFinalExport`, which is a separate `Export` unit followed by a separate update (closure_schedule.go:186-196, 295-312).
  - `ListClosures(pending=true)` lists only `closed_at = 0` (sqlite_closure.go:75-79).
- **Scenario.** A crash, a busy database or any error between those steps leaves a closed network with an empty `final_sha256`. Later runs never retry. `VerifyClosureExport` answers "no completed closure with a final export" permanently, so GOV-025 e is unmet.
- **Evidence.** `TestA1PoC_07_FinalExportNeverRetried`, which injects a failure on the export's audit write:
  ```
  first run: finalised=0 err=injected: export audit failed status=closed
  second run: finalised=0 err=<nil>
  VerifyClosureExport -> invalid-transaction-state: the network has no completed closure with a final export
  ```
- **Fix.** Make `FinaliseClosures` also pick closures with `closed_at <> 0 AND final_sha256 = ''`, or record a "final export pending" marker.

### A1-08 (Info, CONFIRMED): operator dual-control change collides with a scheduled policy

- **Code.** `v := net.PolicyVersion + 1` (dualcontrol_operator.go:40). `net.PolicyVersion` is the version effective as of now, so a scheduled future version already holds that number.
- **Scenario.** An administrator schedules an adverse change (14-day notice). `circula dual-control` then fails with "conflict: duplicate record" until the change takes effect or is cancelled. Once the scheduled change does take effect, its JSON was built from the pre-operator policy. That cannot happen today only because of this conflict.
- **Evidence.** `TestA1PoC_08_OperatorDualControlBlockedByScheduledChange`: "OperatorSetDualControl while a change is scheduled -> conflict: duplicate record".
- **Fix.** Use `LatestPolicyVersion+1`, and either refuse explicitly while a change is pending or apply the threshold to the pending version as well.

### A1-09 (Info, CONFIRMED): dispute reversals are not audited under their own id

- **Code.** `finishDecision` and `decideReReview` audit only the dispute (disputes_decide.go:167-170; disputes_response.go:295-298). The admin path writes `transaction.reversed` with `TargetID: rev.ID` (reversal.go:63).
- **Impact.** An auditor or member filtering the audit log by the reversal transaction id finds nothing. LED-6 is met for the unit, but FR-081's target linkage for the reversal (a state change listed in DOM-035) is missing.
- **Evidence.** `TestA1PoC_09_DisputeReversalHasNoAuditEventOfItsOwn`: "0 audit events target it".
- **Fix.** Emit `transaction.reversed` for the reversal transaction in the same unit, or add the reversal id to the dispute event's `After` summary.

### A1-10 (Info, CONFIRMED): pending-adjustment expiry writes into frozen networks

- **Code.** dualcontrol.go:282-307 has no `IsFrozen` check. Compare transactions.go:374 and closure_schedule.go:183.
- **Evidence.** `TestA1PoC_10_PendingExpiryWritesToFrozenNetwork`: "expired 1 request(s), status=expired, audit rows 15 -> 16". This extends the audit chain of a network whose verification failed.
- **Fix.** Skip frozen networks, as `ExpireProposals` does (after fixing A1-05's paging).

### A1-11 (Low, CONFIRMED by reading code and tests): FR-005 and GOV-004 status inflation

- **Status tables.** 04 row FR-005 is "built: Decimals fixed after first posting" and 10 row GOV-004 is "built".
- **Normative text.** Creation-time immutability is required, and both bodies state the guard is unimplemented (FR-005 implementation note; GOV-004 "Current checkpoint gap").
- **Code.** networks.go:242 still allows any change while `LedgerSeq == 0`.
- **Test.** `internal/service/admin_test.go:186-195` (`TestPolicyChanges`) asserts that a post-creation change from 2 to 3 succeeds, which codifies the non-conformance. A1-02 adds that even the "after first posting" claim does not hold.
- **Fix.** Set both rows to "required unimplemented correction" (as DOM-031 and DAT-049 already are) until the A1-02 fix lands, and invert `TestPolicyChanges`.

## 5. Conformance table (scope A1)

Status values: OK / partial / not-implemented / untested / mismatch / deviation (accepted decision). Paths are relative to `internal/` unless they start with `docs/` or `cmd/`.

### Ledger invariants (LED-1..LED-7)

| Spec ID | Status | Code | Test |
|---|---|---|---|
| LED-1 | OK | service/posting.go:40-87; store/migrations/0001_init.sql:342-356; service/verify.go:65-71 | service/property_test.go `TestRandomisedLedgerInvariants`; service/ledger_prop_test.go; store/sqlite_test.go `TestTransactionIntegrityTriggers` |
| LED-2 | OK (cache row unguarded by design, KNOWN CODE-AUDIT LED-2) | 0001_init.sql:278-314 | store/sqlite_test.go `TestTriggersBlockUpdateDeleteOnAppendOnlyTables`; service/audit_test.go `TestAppendOnlyThroughRawAccess` |
| LED-3 | OK | AddBalance only at posting.go:75; store/sqlite_verify.go:11-28 | service/audit_test.go `TestVerifyDetectsBalanceCorruptionAndBalanceAt` |
| LED-4 | OK | service/idempotency.go:43-81; one `Update` per command | service/atomicity_test.go `TestAcceptFaultInjectionLeavesNoTrace`; service/idempotency_test.go; service/concurrency_test.go `TestIdempotentConcurrentPropose` |
| LED-5 | OK | service/posting.go:15-33, 57 (every posting, system account exempt) | `TestAcceptRechecksLimit`, `TestParallelAcceptsNeverBreachCreditLimit`, `TestQA004LimitBoundary`, `TestLimitReductionDoesNotPost`, `TestA1Check_BelowLimitRules` |
| LED-6 | OK (see A1-09) | service/audit.go:32-62 in-unit | service/atomicity_test.go `TestEveryCommandWritesAuditEvent`, `TestAuditFailureRollsBackStateChange` |
| LED-7 | OK | domain/money.go; 0001 CHECK amount>0 and <>0; 0015 bound | domain/domain_test.go `TestNoFloatsInMoneyPackages`, `TestAddChecked`; fuzz tests; store/sqlite_test.go `TestBalanceBoundTriggers` |

### Domain rules (03, DOM-xxx)

| Spec ID | Status | Code | Test |
|---|---|---|---|
| DOM-005 | OK | domain/money.go:16-47 | `TestParseAmount`, `FuzzParseAmountRoundTrip` |
| DOM-006 | OK | posting.go | `TestTradeScenarioBalances` |
| DOM-007 | OK | domain/types.go:137-142 | `TestEffectiveLimit`, `TestLimitOverrideExpires` |
| DOM-008 | OK | service/accounts.go:27-49 | `TestAvailableCreditTable` |
| DOM-010 | OK | domain/statemachine.go:44-70; trigger `tx_status_flow` | `TestTxStateMachineExhaustive` |
| DOM-011 | OK | transactions.go:210-219 | `TestConsentBothDirections` |
| DOM-012 | OK (DEC-035) | transactions.go:125-154 | `TestReservationAndRelease`, `TestUnsolicitedRequestsDoNotFreezeCredit` |
| DOM-013 | OK; exact boundary was untested in the suite | transactions.go:155, 227-229 | `TestExpiry`; `TestA1Check_ExpiryAcceptBoundary` (passes) |
| DOM-014 | OK | transactions.go:196-257 | `TestAcceptRechecksLimit`, `TestParallelDoubleAcceptSingleSuccess` |
| DOM-015 | OK | triggers 0001:342-356 | `TestTransactionIntegrityTriggers` |
| DOM-016 | OK (replay returns current object, see gap G6) | idempotency.go | `TestIdempotentConcurrentPropose`, `TestIdempotencyKeyReusedWithDifferentPayload` |
| DOM-017 | not-implemented (KNOWN required correction) | disputes.go:56 reads the current policy | `TestOpenDisputeWindowAndNoPostings` (as built); `TestA1Check_DisputeWindowBoundary` |
| DOM-018 | partial (A1-03) | disputes_decide.go | `TestResolveOutcomes`, `TestWithdrawDispute` |
| DOM-019 | OK | reversal.go:87-89 | `TestFullReversal`, `TestOpenDisputeRules` |
| DOM-020 | OK | reversal.go:93-96; CHECK `reversed_total <= amount` | `TestPartialReversalCumulativeCap`, `TestStressParallelReversalsRespectCap` |
| DOM-021 | partial (A1-03 dead end) | disputes_decide.go:107-115, 177-199 | `TestUpheldReversalBlockedThenOverride` |
| DOM-022 | OK | adjustments.go | `TestAdjustments` |
| DOM-023 | OK | triggers; no edit path | `TestTriggersBlockUpdateDeleteOnAppendOnlyTables` |
| DOM-024 | partial (A1-01, A1-06; A3-07 owner-pending) | departure.go | `TestDepartureRetain`, `TestDepartureSettleToZero`, `TestDepartureBlockersAndAdminOffboarding` |
| DOM-026 | OK | posting.go:16-18, 50-56 | `TestA1Check_BelowLimitRules`; property tests |
| DOM-027 | OK | adjustments.go:190-213; join.go:111-115, 319-323 | `TestSignupGrant` |
| DOM-030 | partial (DOM-017 known; A1-02; unnamed binding events, gap G5) | networks.go | `TestPolicyChanges`, `TestPolicyNoticePeriod` |
| DOM-031 | not-implemented (KNOWN) plus A1-02 bypass | networks.go:242 | `TestPolicyChanges` asserts the opposite |
| DOM-032 | OK | membership.go:250-315 | `TestLimitReductionDoesNotPost` |
| DOM-035 | partial (A1-09) | audit.go | `TestEveryCommandWritesAuditEvent` |
| DOM-036 | OK (DEC-052 deviation) | reversal.go:24-71 | `TestReversalAuthorization` |
| DOM-037 | OK | principal.go:162; reversal.go:45; dualcontrol.go:85 | `TestReversalAuthorization`, `TestDualControlTargetAdminCannotApprove` |
| DOM-038 | OK (DEC-072 open) | disputes.go:95-110 | `TestDisputeDecisionAuthorization` |
| DOM-039 | OK | principal.go:106; transactions.go:230-241 | authz_test.go; `TestOpenDisputeRules` |
| DOM-040 | OK | listings.go | `TestListingsDoNotTouchLedger` |

### Functional requirements (04, FR-xxx)

| Spec ID | Status | Code | Test |
|---|---|---|---|
| FR-005 | mismatch (A1-11, A1-02) | networks.go:242 | admin_test.go `TestPolicyChanges` (codifies the mismatch) |
| FR-030..FR-033 | OK | transactions.go | trades_test.go |
| FR-034 | OK (starvation A1-05; exact accept boundary untested in suite) | transactions.go:359-393 | `TestExpiry`; `TestA1Check_ExpiryAcceptBoundary` |
| FR-035 | OK | idempotency.go | idempotency_test.go |
| FR-036 | OK | gate plus `BEGIN IMMEDIATE` | concurrency_test.go, stress_test.go |
| FR-050 | OK (service) | accounts.go | `TestAvailableCreditTable` |
| FR-051 | OK (service) | reads.go:121-149 | `TestListEntriesOrderedByLedgerSeq` |
| FR-052 | OK | verify.go:41-91 (read-only) | `TestVerifyLedgerIsReadOnly` |
| FR-054 / DAT-050 | OK at service and store level (adapters are A2) | service/histbal.go; store/sqlite_tx.go:197-202 | service/histbal_test.go |
| FR-060, FR-062, FR-066 | OK | membership.go; adjustments.go; reversal.go | `TestLimitReductionDoesNotPost`, `TestAdjustments`, `TestReversalAuthorization` |
| FR-068 | mismatch (A1-01, A1-03, A1-07) | closure*.go, departure.go | closure_test.go |
| FR-070, FR-071 | OK as built (DOM-017 target known) | disputes.go | dispute_test.go |
| FR-073 | partial (A1-03) | disputes_decide.go | `TestResolveOutcomes` |
| FR-074 | OK (base path) | disputes_decide.go | `TestUpheldReversalBlockedThenOverride` |
| FR-075 | partial (A1-03: withdrawal impossible after a re-review reversal) | disputes.go:175-216 | `TestWithdrawDispute` |
| FR-077 | partial (DOM-017 KNOWN) | disputes.go:56 | `TestOpenDisputeWindowAndNoPostings` |
| FR-078 | partial (A1-03) | disputes_response.go | disputes_response_test.go |
| FR-090, FR-091 | OK (correctness) | export*.go; store/sqlite_export.go | `TestLedgerExportCSV`, `TestMemberExportIsScoped` |

### Data, ledger and audit (06, DAT-xxx)

| Spec ID | Status | Code | Test |
|---|---|---|---|
| DAT-007..DAT-011 | OK | 0001 triggers and CHECKs | store/sqlite_test.go |
| DAT-012 | deviation (DEC-040, KNOWN) | reversal.go | `TestPartialReversalCumulativeCap` |
| DAT-014..DAT-021 | OK | posting.go; sqlite_network.go:72-76; sqlite_verify.go | atomicity_test.go, audit_test.go |
| DAT-022, DAT-023 | OK | reversal.go; adjustments.go | reversal_test.go, admin_test.go |
| DAT-024 | partial (KNOWN) | membership.go override | `TestLimitOverrideExpires` |
| DAT-025..DAT-028 | OK (gap G6) | idempotency.go; purge.go | idempotency_test.go |
| DAT-029, DAT-030 | OK (unkeyed: A3-11 owner-pending; platform scope unverified: SVC-06 KNOWN) | audit.go; verify.go:153-191 | `TestAuditChainDetectsTampering`, `TestAuditTailTruncationDetected` |
| DAT-039 | OK | export.go:155-170 one View | `TestLedgerExportCSV` |
| DAT-049 | not-implemented (KNOWN) plus A1-02 | networks.go:242 | none |
| DAT-051 | not-implemented (KNOWN) | none | none |

### Governance cross-checks (10, GOV-xxx)

| Spec ID | Status | Code | Test |
|---|---|---|---|
| GOV-004 | mismatch (A1-11) | networks.go:242 | `TestPolicyChanges` |
| GOV-019 | OK | membership.go | `TestLimitReductionDoesNotPost` |
| GOV-020 | mismatch (A1-01, A1-04) | departure.go; reversal.go:84 | `TestAdminRequiredReadOnly` (propose only) |
| GOV-023 | mismatch (A1-01) | departure.go | freeze/suspend tests |
| GOV-025 | mismatch (A1-01, A1-07) | closure_schedule.go | closure_test.go |

Conformance summary (scope rows):

| Status | Count |
|---|---|
| OK | 49 (two of them with a boundary that was untested in the suite and is now covered by an A1 check test) |
| partial | 10 |
| not-implemented | 4, all KNOWN (DOM-017, DOM-031, DAT-049, DAT-051) |
| mismatch | 6 (FR-005, FR-068, GOV-004, GOV-020, GOV-023, GOV-025) |
| deviation | 1, accepted (DAT-012) |

## 6. Spec gaps (not bugs; no policy invented)

- **G1. Serial disputes on one trade.** FR-070 and DOM-017 do not say whether a second dispute may be opened after the first is decided. FR-078 limits re-review to one. The code allows serial disputes, which feeds A1-03.
- **G2. Dispute reversals during `closing`.** GOV-025 b freezes postings at the closure date. The closure then waits for disputes whose upheld outcome must post. The spec does not say whether dispute reversals are exempt from the freeze. The code allows them while `closing` but not on a suspended network.
- **G3. Postings to a departed member's closed account.** DOM-004 says account state "governs new transactions only". DOM-024 says disputes "remain resolvable". An admin reversal, or a re-review reversal of an older trade, can re-create a non-zero balance on an account that settle-to-zero had zeroed. The spec is silent on whether that is allowed and how it is settled. It also does not say whether a departure in a frozen network state should be refused or fall back to `retain` (A1-01).
- **G4. DOM-039 wording.** "A suspended member's balance does not change except by a posting that was already completed" is ambiguous. The code allows admin adjustments, dispute reversals and admin removal settlement on suspended members.
- **G5. Unnamed binding events.** DOM-030 requires each policy's binding event to be named before implementation. It is not named for `signup_grant` (bound at activation), `departure_rule` (at departure), `response_period_days` (at decision), `re_review_window_days` (at request) or `dual_control_threshold` (at request; approval does not re-check).
- **G6. Idempotent replay.** It returns the current state of the referenced object, for example a replayed propose shows `completed` after acceptance, rather than the original response. DAT-025 says "original stored response". The service stores only a reference.
- **G7. Elapsed but unswept proposal.** Decline or withdraw of such a proposal records `declined` or `cancelled` rather than `expired` (FR-033/FR-034 silent). Acceptance is correctly refused.
- **G8. Voluntary settle-to-zero authorship.** The adjustment is `created_by` the departing member, while DOM-024 says "administrator-created adjustment" and DOM-022 and DOM-037 bar self-adjustment. This overlaps the owner-pending A3-07 decision.

## 7. Not reviewed / coverage limits

- HTTP/API and web adapters (A2): amount parsing at the edge, the API replay cache, and the FR-054/API-083 and UX-061 adapters.
- MFA, recovery, break-glass, erasure and privacy, backup and restore, and the CLI (A3). Only one job-runner loop was read (A1-05). `RedactTransactionNotes` and the closure erasure were seen but not audited.
- No multi-process (serve plus CLI) concurrency test. No fuzzing beyond the seeds in `go test`. No load test. `scripts/read-export.py` was not run.
- Overview, limit-warning and inactivity reads were skimmed for money semantics only.
- The `closing` variant of A1-01 is LIKELY: same code, not run separately.

### 7a. Appendix: SPECULATIVE / defence-in-depth (not counted)

- `independent()` (disputes.go:100) checks only the newest 500 reversals of a trade. A reversal creator beyond 500 partial reversals escapes the conflict check.
- `CreateAdjustment` returns `(nil, nil)` on replay of a key first used with `CreateAdjustmentRequest` (shared op "adjustment.create", adjustments.go:46-50). There is no production caller.
- DB-level gaps (raw SQL only; `VerifyLedger` catches some):
  - entries can be inserted on a `proposed` transaction that is then declined, cancelled or expired;
  - `completed_at`, `ledger_seq`, `accepted_by` and `reversed_total` (decrease) are mutable after completion;
  - decided `disputes` rows are mutable.
- The `internal/histbal/histbal.go:4-8` package comment is stale: it says "service does not yet provide a timestamp-based read; no route is registered".
- `PendingIncoming` SUM could overflow only with thousands of near-2^50 reservations aimed at one payee. Not realistic.

### 7b. KNOWN items re-checked at this SHA

| Item | Status at this SHA |
|---|---|
| CODE-AUDIT SVC-01 (read-only verify) | fixed |
| CODE-AUDIT SVC-02 (`query_only` views) | fixed |
| CODE-AUDIT SVC-03 (accept re-checks the default flag) | fixed (transactions.go:238) |
| CODE-AUDIT DATA-01 (purge) | fixed |
| SA-08 (request squatting) | fixed (DEC-035) |
| SA2-02 (admin-initiated settlement threshold) | fixed |
| A3-03b (balance bound) | fixed (posting.go:50-56; migration 0015) |
| CODE-AUDIT LED-2 (unguarded cache row) | open by design |
| CODE-AUDIT SVC-06 (platform audit chain never verified) | not fixed |
| SA2-12, A3-07, A3-10, A3-11 | owner-pending; not re-reported |
| DOM-017/DAT-051, DOM-031/DAT-049 | still required unimplemented corrections |

## 8. Commands run

All commands ran in `/home/user/audit/circula-a1` with go1.26.10.

| # | Command | Result |
|---|---|---|
| 1 | `go test -count=1 ./internal/domain/... ./internal/store/... ./internal/histbal/... ./internal/service/...` (baseline) | PASS. domain 0.17 s, store 27.4 s, histbal 0.01 s, service 46.9 s |
| 2 | `CGO_ENABLED=1 go test -race -count=1` (same packages) | PASS. store 104 s, service 361 s; no race reported |
| 3 | `go vet ./internal/domain/... ./internal/store/... ./internal/histbal/... ./internal/service/...` | PASS |
| 4 | `go vet -tags a1poc ./internal/service/`; `gofmt -l internal/service/a1_poc_test.go` | PASS; clean after `gofmt -w` |
| 5 | `go test -tags a1poc -count=1 -run 'TestA1PoC\|TestA1Check' -v ./internal/service/` | 13 `TestA1PoC_*` FAIL, each confirming its finding (excerpts in section 4); 3 `TestA1Check_*` PASS. Full log: `/tmp/claude-0/-home-user/b9ccda38-9f8d-54d6-a99c-1f20e1924081/scratchpad/a1/poc-run.txt` |
| 6 | `go test -count=1 ./internal/service/` after adding the PoC file | PASS (29.7 s); the build tag keeps the PoCs out of the default suite |

The PoC file left in place as evidence is `/home/user/audit/circula-a1/internal/service/a1_poc_test.go` (build tag `a1poc`).
