# QA gap analysis: spec 12 versus the test suite

Date: 2026-10-21. Base: `f8278a0`. Method: requirement text in `docs/specs/12-quality-test-plan.md`
compared with test function names and bodies under `internal/` and `cmd/`. "Covered" means a named test
asserts the acceptance criteria as written; "Partial" means some criteria are asserted; "Gap" means no test.
Matching was done by reading test names and, for the ones added or cited below, their bodies. It was not a
line-by-line audit of all 319 tests, so a "Covered" row names the evidence to check.

Added by this QA pass (all new files):

| File | Purpose |
|---|---|
| `internal/domain/money_fuzz_test.go` | `FuzzParseAmount`, `FuzzFormatAmount`, `FuzzAddChecked` |
| `internal/service/mfa_fuzz_test.go` | `FuzzMatchTOTP`, `FuzzRecoveryCodeNormalise`, `FuzzOpenSecret`, window and seal tests |
| `internal/service/export_fuzz_test.go` | `FuzzCSVCell`, number and nil cases |
| `internal/service/ledger_prop_test.go` | `TestPropTwoNetworkLedger` (two networks, cross-network misuse) |
| `internal/service/stress_test.go` | `TestStress*` (parallel reversals, disputes, approvals, 100 disjoint trades, mixed two-network workload, register/redeem races) |
| `internal/api/stress_test.go` | `TestStressAPIParallelTradesNo5xx` (no 500/503 through HTTP under contention) |
| `internal/store/migrate_upgrade_prop_test.go` | `TestUpgradeFromEverySchemaVersion` (QA-027) |

Fuzz runs (40 s each, Go 1.26.10, no failures, no new corpus files needed): `FuzzParseAmount` 4.3M execs,
`FuzzFormatAmount`, `FuzzAddChecked`, `FuzzMatchTOTP`, `FuzzRecoveryCodeNormalise`, `FuzzOpenSecret`, `FuzzCSVCell`.
Re-run with `go test ./internal/<pkg> -run '^$' -fuzz '^FuzzName$' -fuzztime 60s`. The seed corpora run on every
plain `go test`.

## Ledger invariants (section 2)

| Req | Status | Evidence / note |
|---|---|---|
| QA-001 property tests | Partial | `TestRandomisedLedgerInvariants` (6 seeds x 400 ops, one network), `TestPropTwoNetworkLedger` (new, two networks). Run size is far below 1,000 x 50 per change and 100,000 on release; no seed-fixture file is saved on failure (the seed is in the subtest name). Operations "create network", "add member", "change policy" and "depart" are not in the generator. |
| QA-002 cache agreement | Covered | `checkInvariants` after every op in both property tests; `TestVerifyDetectsBalanceCorruptionAndBalanceAt` is the negative test. |
| QA-003 equal and opposite | Covered | `checkInvariants` (two entries, sum zero); `TestTransactionIntegrityTriggers` refuses single-sided entries. |
| QA-004 limit boundary | Covered | `TestAvailableCreditTable`, `TestAcceptRechecksLimit`, `TestLimitReductionDoesNotPost`, `TestReversalBlockedByLimit`, `TestParallelAcceptsNeverBreachCreditLimit`. Limits of 0, 1 and large are in the table test. |
| QA-005 integer money | Covered | `TestNoFloatsInMoneyPackages`, `TestMigrationsAvoidFloatColumns`, `TestParseAmount`, `TestAddChecked`, new `FuzzParseAmount` (no panic, no wrap, no sign, roundtrip). |
| QA-006 audit atomicity | Covered | `TestAcceptFaultInjectionLeavesNoTrace`, `TestAuditFailureRollsBackStateChange`, `TestEveryCommandWritesAuditEvent`. |

## Concurrency, retries, idempotency (section 3)

| Req | Status | Evidence / note |
|---|---|---|
| QA-007 one account | Partial | `TestParallelAcceptsNeverBreachCreditLimit` (25 racers, exactly 7 fit), `TestStressParallelReversalsRespectCap` (new). The "100 random interleavings per CI configuration" criterion is not met (one interleaving per run; the scheduler varies it). |
| QA-008 disjoint accounts | Covered | `TestStressDisjointTradesNoDeadlock` (new): 100 pairs, 120 s ceiling (under 1 s on the dev box), sum zero. |
| QA-009 idempotent retries | Covered | `TestIdempotentConcurrentPropose`, `TestIdempotentRetryReturnsOriginal`, `TestIdempotencyKeyReusedWithDifferentPayload`, `TestIdempotencyRecordExpires`, `TestFullTradeFlow` (replay header, 422 on reuse), new API same-key race. |
| QA-010 crash and restart | Gap | Only `TestAcceptFaultInjectionLeavesNoTrace` (in-process fault, no process kill or reopen). Cannot be done black-box here; a reopen-after-fault variant is possible. |
| QA-011 lock contention | Partial | `TestBusyDatabaseIsRetryable`, `TestBusyDatabaseIs503`, `TestBusyMapsTo503`. New stress tests assert busy/unavailable never surfaces from in-process contention. Cross-process timeout under a held write lock is only unit-tested via the error mapping. |

## Authorization, disputes, governance (section 4)

| Req | Status | Evidence / note |
|---|---|---|
| QA-012 authz matrix | Partial | `TestAuthorizationMatrix` (service), `TestAuthorisation`, `TestCrossNetworkIdsAreNotFound`. The criterion "route inventory equals the table; a route missing from the table fails the build" is not asserted for HTTP routes. |
| QA-013 dispute lifecycle | Covered | `TestDisputeStateMachineExhaustive`, `TestOpenDisputeRules`, `TestResolveOutcomes`, `TestWithdrawDispute`, `TestStressParallelDisputeResolve` (new). DST and year-boundary deadlines are not tested explicitly. |
| QA-014 non-destructive | Covered | `TestFullReversal` (entries DeepEqual), `TestPartialReversalCumulativeCap`, dispute tests, property tests. |
| QA-015 governance | Partial | `TestPolicyBounds`, `TestPolicyChanges`, `TestRejoinSemantics`, `TestAdministratorSuccession`, `TestSuspension`. No traceability check that each GOV rule has a test. |
| QA-016 dual control | Covered | `TestDualControl*` (self-approval, expiry, last admin, concurrent approve) plus `TestStressConcurrentApprovals` (new, 12 requests x 5 approvers). |

## Scenarios (section 5)

| Req | Status | Evidence / note |
|---|---|---|
| QA-017 invite join | Covered | `TestInvitations`, `TestJoinByInvitationLink`. |
| QA-018 trade | Covered | `TestTradeScenarioBalances`, `TestLoginTradeFlow`, `TestFullTradeFlow`. |
| QA-019 decline, expiry | Covered | `TestDeclineCancelExitRules`, `TestExpiry`. |
| QA-020 limit boundary | Covered | `TestAvailableCreditTable`. |
| QA-021 dispute and reversal | Covered | `TestResolveOutcomes`, `TestUpheldReversalBlockedThenOverride`. |
| QA-022 policy change | Covered | `TestPolicyChanges`, `TestLimitReductionDoesNotPost`, `TestSetCreditLimitViewAuditsOnce`. |
| QA-023 departure | Covered | `TestDepartureRetain`, `TestDepartureSettleToZero`, `TestDepartureBlockersAndAdminOffboarding`. |
| QA-024 backup and restore | Covered | `TestSnapshotWritesVerifiedManifestAndRestores`, `TestEncryptedBackupAndRestoreIntegration`, `TestBackupEqualsSource` (cmd). |
| QA-025 export and re-import | Gap | Export is tested (`TestFullExportJSON`, `TestLedgerExportCSV`); no import path exists in the code base, so no re-import test is possible. Spec/feature gap, not a test gap. |
| QA-026 upgrade with rollback | Partial | `TestMigrationsIdempotentAndVersioned` (newer schema refused), `TestServeRefusesPendingMigrationsInExplicitMode`, `TestSwapDatabaseRollsBack`. A forced mid-migration failure leaving the prior schema is not tested (the runner has no injection point; the embedded migration set is fixed). |

## Migrations, backup (section 6)

| Req | Status | Evidence / note |
|---|---|---|
| QA-027 every released schema | Covered (new) | `TestUpgradeFromEverySchemaVersion`: each of v1..v(head-1) with a ledger row, upgrade, integrity and foreign-key check, second open no-op. No committed binary fixtures; older schemas are rebuilt from the migration files. The >10,000-entry timing criterion is not covered. |
| QA-028 backup under load | Gap | Snapshot/restore tests run on a quiet database. A backup during concurrent postings (cmd package) is not tested. |
| QA-029 restore drill record | n/a | Release process evidence, not a test. |

## Export (section 7)

| Req | Status | Evidence / note |
|---|---|---|
| QA-030 reconciliation | Partial | `TestLedgerExportCSV` reconciles; no export taken during concurrent postings. |
| QA-031 isolation, secrecy | Covered | `TestMemberExportIsScoped`, `TestExportCapPerPerson`, `TestExportQuotaPerActor`. |
| QA-032 spreadsheet-safe | Covered (extended) | `TestCSVFormulaNeutralisation`, new `FuzzCSVCell` (any input: no leading `= + @ TAB CR`, `-` only for plain numbers, only an apostrophe added, csv roundtrip). |

## Accessibility, gates (sections 8 to 10)

| Req | Status | Evidence / note |
|---|---|---|
| QA-033 automated a11y | Partial | `TestPageStructure`, `TestFormLabelsAndAutocomplete`, `TestValidationErrorSummary` (hand-written structural checks, no axe-class checker). |
| QA-034 keyboard, no script | Partial | Server-rendered flows are tested with plain HTTP (works without scripting by construction); no browser test. |
| QA-035 manual review | n/a | Release evidence. |
| QA-036 direction/status presentation | Covered | `TestPageStructure`, balance `Direction` assertions in API tests. |
| QA-037..039 gates | Partial | `Makefile` / CI run vet, race tests, lint. No integration-gate long run (100,000 sequences) exists. |
| QA-040 thresholds | Gap | No coverage threshold is enforced. |
| QA-041 determinism | Covered | Fixed seeds, injected `Clock` and `IDs` in `testutil`. |
| QA-042 traceability matrix | Gap | No machine-readable requirement-to-test map. This document is the first manual one. |
| QA-043, QA-044 | n/a | Process. |

## Not done and why

- Long-run property sizes (QA-001) were not raised; the per-change run time would grow past minutes under `-race`.
  Raising them behind an environment variable (for the integration gate) is the recommended follow-up.
- Cross-process SQLITE_BUSY (a second process holding the write lock) is not exercised.
- No bugs were found by the new tests, so no production code was changed. See the report for the findings list.
