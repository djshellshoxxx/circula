# Recurring recovery consolidation — 2026-11-20

**Disabled; integration HOLD.** This local composition preserves completed recurring work on the actual accepted future target. It does not register a Core feature, select production policy, publish a branch, merge a remote PR or deploy a release.

## Source and destination

| Role | Exact identity |
| --- | --- |
| Recovery PR | [#118](https://github.com/djshellshoxxx/circula/pull/118), source `master/recover-recurring-20261119` |
| Recovery source | `1e0e7e2f12a77d6d8969d1593ecdb6309bb13c48` |
| Actual target | `future/fnet-207-recurring-20261116` |
| Accepted target commit | `94deb0a20df07a3b38c8a905254645d6cd3019f2` |
| Merge base | `73a911a9bec387381633c031d8a4083c9fc724ad` |
| Local composition branch | `recover/fnet207-recurring-consolidation-20261120` |
| Tested composition commit | `1e0838e04310bb2a19a585d69e40426b3c50fb71` |
| Tested composition tree | `d34913b1dc28aa735bc180d3b152b3394144ddb2` |

PR #118 still pointed to the earlier `master/fnet207-recurring-remote-20261119` topic at discovery. [PR #110](https://github.com/djshellshoxxx/circula/pull/110) has since merged into that topic at `9bf260d63f454868e27d2d85a3d9fb5dce4beeae`, and [PR #97](https://github.com/djshellshoxxx/circula/pull/97) merged it into the actual future target at `94deb0a20df07a3b38c8a905254645d6cd3019f2`. A live provider comparison confirmed that the target branch still matched that exact commit with zero ahead/behind commits and zero differing files.

The local branch begins at the accepted target and uses a normal two-parent merge of the recovery source. Both histories remain ancestors. No source branch is reset, rebased, force-pushed or deleted. Publication and any PR retargeting are separate coordinator actions after current qualification.

## Exact conflict resolution

The read-only merge simulation and actual local merge both identified only one conflict: `future/recurring/calendar_test.go`.

The target's sole recurring-module change since the merge base was #110's accepted public-digest fixture correction. The recovery had already retained the older helper tests under `due_with_schedule_test.go` and installed a distinct archived `calendar_test.go` for the recovered Calendar API. The resolution preserves both complete suites:

| Composed path | Exact preserved source | SHA-256 |
| --- | --- | --- |
| `calendar_test.go` | Recovery source's archived Calendar suite | `4f08eb425996bd9d2ef3e0cf3aa5c05cc062660857c629cb0ec44e78e79aa674` |
| `due_with_schedule_test.go` | Accepted target's complete `calendar_test.go` helper suite | `1a6bda9cca2b8b0533c4c2bdc2c7557c324969bb30d64cfce520c296a6b4c7be` |

The exact fixed digest value and every golden/scoping comparison remain intact. Only the already accepted literal spelling and variable name follow the renamed file. No scanner rule, exclusion, assertion or production calendar behavior changes.

`schedule.go`, `schedule_test.go` and `schedule_regression_test.go` are identical across the merge base, recovery and accepted target. Every other recovered Go file, dependency file and manifest remains byte-identical to the recovery source. No arbitrary winner was selected between two production implementations.

Independent read-only review by `/root/password_admission/recurring_source_review` inspected the complete recovered production surface, module overlap/history, host contracts and historical receipts. It found no module compatibility or product-decision blocker for this isolated preservation and independently verified both test-file hashes above.

## Accepted Core and CI work retained

The accepted target already contains the missing audit-anchor import, the 501-proposal departure fixture correction, resource-related notification redaction, the operator-path G703 comments, the opt-in native PR matrix and fixed-execution Core fuzz budgets. These are retained exactly; this composition does not replace Core with a newer main snapshot.

The target's `erasure_notification_regression_test.go` is retained. The recovery contributes its already accepted deterministic `erasure_test.go` blob from `fc31dc9a6b8c102fe612f47bc543e94e8114df0f` and `notification_erasure_test.go` blob from `28326597340a4f0efad33f37feecc380db0de66f`. Both retain meaningful independent privacy and credential assertions.

Compared with the accepted target, the only paths outside `future/recurring/` are the dedicated recurring workflow and those two accepted Core test files. Root dependencies, shared CI, production Core, other future modules and all feature registrations remain unchanged. The dedicated recovery workflow already accepts the actual future target, so retargeting needs no weakened or bypassed checks.

## Verified original failures

The original recovery Core [run 44555387657](https://github.com/djshellshoxxx/circula/actions/runs/44555387657) had two successful applicable jobs, five failures and one conditional native placeholder skip. All five failed job logs were read:

| Failed job | ID | Actual diagnostic |
| --- | --- | --- |
| Ubuntu race | 135670118059 | Audit-anchor missing `os` import; departure fixture exceeds the ordinary proposal cap. |
| Secret/lint gate | 135670117906 | One generic-key finding at the public digest fixture in `due_with_schedule_test.go`. |
| Reproducibility | 135670117290 | Audit-anchor missing `os` import. |
| Browser | 135670118048 | Application build fails on the same missing import. |
| Release-target build | 135670118140 | Same missing import. |

The recovered module [run 44555387672](https://github.com/djshellshoxxx/circula/actions/runs/44555387672) passed all three native jobs at the original recovery head: macOS `135669955288`, Windows `135669955885`, Ubuntu `135669956076`. These are prior-source results, not a composed-head pass.

The exact source/target provider records, failed-log hashes and relevant diagnostic excerpts are in `consolidation-evidence-20261120/`. No pre-fix reproduction or adversarial traffic was run for this composition.

## Actual local qualification

The source-composition commit above was reviewed before execution; its original static receipts remain unchanged as a historical checkpoint. The subsequent commands below actually executed on that exact committed source between 17:13:46 and 17:20:22 UTC on November 20. Every command exited zero, and all 34 recorded module/Core/workflow source fingerprints remained unchanged before and after each command. This follow-up changes only documentation and qualification evidence.

| Executed check | Actual result | Wall time |
| --- | --- | --- |
| Exact dedicated-workflow isolation guard | Disabled/HOLD, exact dependency graph, no Core or foreign-feature imports | 6.198 s |
| Full real-driver module race suite with coverage | 48 ordinary Test functions plus 2 Fuzz seed entrypoints; 160 passing test/subtest terminal events; 0 failed, 0 skipped; 80.0% statement coverage | 110.446 s, including compilation |
| Module `go vet -p 1 ./...` | Passed | 77.032 s |
| Module `go mod verify` | All modules verified | 3.135 s |
| `FuzzScheduleInvariants` | Exactly 100,000 executions; passed | 7.660 s |
| `FuzzCalendar` | Exactly 100,000 executions; passed | 5.184 s |
| Focused Core race qualification | 4 ordinary tests, 6 passing test/subtest terminal events; 0 failed, 0 skipped | 171.493 s, including compilation |

The module race package reported 5.805 s of package execution (5.806 s in its terminal JSON event). The selected Core service package reported 35.320 s. The same Core command compiled `cmd/circula`, confirming the inherited missing-import failure is repaired, but its test filter selected no command-package tests; that package is not described as a full test-suite pass.

These runs used Go 1.26.11 on Linux/amd64, the actual pinned SQLite drivers, default build tags, `CGO_ENABLED=1`, `GOMAXPROCS=1`, `GOGC=20`, `GOMEMLIMIT=256MiB`, `GOTOOLCHAIN=local`, `GOWORK=off` and `GOFLAGS='-mod=readonly -p=1'`. The race commands' `-ldflags=-w` setting omits DWARF only. No resource retry, driver replacement, assertion removal or race suppression was used.

From `future/recurring/`, the exact test and fuzz commands were:

```sh
go test -p 1 -race -count=1 -cover -json -ldflags=-w -timeout=2m ./...
go test -p 1 -run '^$' -fuzz '^FuzzScheduleInvariants$' -fuzztime=100000x -parallel=1 -timeout=150s schedule.go schedule_test.go schedule_regression_test.go due_with_schedule_test.go
go test -p 1 -run '^$' -fuzz '^FuzzCalendar$' -fuzztime=100000x -parallel=1 -timeout=150s calendar.go schedule.go calendar_fuzz_test.go
```

The fuzz campaigns target the calendar implementations; they do not claim fuzz instrumentation of the SQLite service. The complete module's real-driver race suite covers the separate service fixtures. From the repository root, the focused accepted-Core fixture command was:

```sh
go test -p 1 -race -json -count=1 -ldflags=-w -timeout=5m -run '^(TestDepartureCancelsMoreThanFiveHundredProposals|TestAnonymiseMemberKeepsLedger|TestNotificationErasureUsesResourceRelations|TestErasureRedactsCounterpartyNotificationsWithoutCollateral)$' ./cmd/circula ./internal/service
```

The complete command arrays, working directories, timestamps, source hashes, exit statuses and raw outputs are retained in `consolidation-evidence-20261120/qualification-module.json`, `qualification-core.json` and their referenced logs. The ordinary-test count of 48 is consistent with the historical receipt; counting both named Fuzz seed entrypoints gives 50 top-level module entries. No failed/skipped terminal test event occurred in either race log.

The repository scanner used the unchanged Gitleaks v8.28.0 configuration. The initial source scan passed; the qualification follow-up includes a final scan receipt covering the assembled documentation and test evidence. This does not claim a full Core lint, browser, release-build, reproducibility, native-platform or provider workflow run at the local composition head. Those remain required after publication.

The provider readback at 17:22:47 UTC still showed original PR #118 source `1e0e7e2f12a77d6d8969d1593ecdb6309bb13c48` and an identical actual target at `94deb0a20df07a3b38c8a905254645d6cd3019f2`. It remains an open draft against the older stacked topic and reports conflicts there. `provider-final.json` records this observed state; no local result is substituted for the future published-head checks.

## Historical evidence and whitespace

The full staged whitespace check reports one newly recovered, byte-identical upstream notice with a blank final line: `third_party/modernc_org_memory/LICENSE-MMAP-GO:25`. Its original bytes are preserved. The check excluding only that immutable notice succeeds; no authored source whitespace issue remains. This is documented rather than misreported as a completely clean original full-diff check.

The historical November 19 qualification contains 25 fingerprints. All 23 Go/test/dependency hashes match the original recovery source; descriptive manifest metadata and workflow routing were the two documented later changes. The historical full real-driver suite recorded 48 top-level passes, no skips and 80.0% coverage, with 100,000 executions for each calendar fuzz target. The composed helper fixture now intentionally matches the accepted target, so that historical receipt is not asserted to be an unchanged whole-composition test run.

## Boundaries that remain open

The manifest stays exactly `enabled: false`, `integration: HOLD`, with an empty future-feature dependency list. Recovered cron/SQLite pins remain within the nested module. No automatic route, scheduler, migration or database registration is added.

The completed standalone implementation retains both calendar APIs, private SQLite CRUD/revisions/idempotency, atomic unique firings/reminders, the same-transaction inbox seam, pause/skip/cancel/edit behavior, authorized manual drafts, no-JS HTTP/CLI and lifecycle/restore fixtures. Its inherited quotas, twenty-hour refiring guard, retention minimum and tombstones remain isolated candidate behavior.

Actual Core transaction/current-auth/terms/capacity adapters, network opt-in, inbox integration, scheduler continuation/restart/fairness, owner/network archival/erasure policy, browser/accessibility, operational backup and full ledger qualification remain separate work. There is no need to invent those decisions to preserve this branch.

Before any remote merge, the coordinator must refresh source/target pins, route the PR to the actual accepted future target, require all applicable exact-head Core and three-platform module checks, and apply normal review/permission requirements. Native Core qualification uses the accepted `ci:cross-platform` opt-in; skipped jobs are not passes.
