# FNET-207 combined recovery qualification — 2026-11-19

**Local combined-source qualification passed. The feature remains disabled and integration HOLD.** The results below apply to the exact nested-module source recorded here. Native provider and Core checks at the new published head remain separate gates.

| Identity | Value |
|---|---|
| Tested local commit | `e79cd7a3d8bc1830aeb5ee255c044ddfef72f278` |
| Tested whole tree | `707f6fb5db85808a86a7fae87c716b6ebecfec66` |
| Tested module tree | `3c0580c334255092bba0fc2f4d0ea144f021e816` |
| Toolchain | Go 1.26.11, Linux/amd64 |
| SQLite driver | Actual pinned `modernc.org/sqlite v1.60.1`, default build tags |
| Source stability | All tracked qualification fingerprints remained unchanged during execution |

## Commands and observed results

The environment was `GOTOOLCHAIN=local`, `GOMAXPROCS=1`, `GOGC=10`, `GOMEMLIMIT=256MiB`, `CGO_ENABLED=1`, `GOWORK=off`, and `GOFLAGS=-mod=readonly -p=1`. The full race command used `-ldflags=-w` for the constrained shared environment. Every command ran serially with a separate temporary directory; module sources and dependency pins were unchanged. The JSON companion records the actual executable, environment, commands, per-command outcomes, test names and SHA-256 fingerprints.

| Check | Observed result | Wall time including build |
|---|---|---|
| toolchain | PASS | 0.008s |
| isolation | PASS | 0.217s |
| format | PASS | 0.033s |
| race | PASS; 48 top-level tests, zero skips, 80.0% statement coverage; package 6.281s | 11.363s |
| vet | PASS | 1.068s |
| checksums | PASS | 2.259s |
| fuzz-schedule | PASS; `FuzzScheduleInvariants`; exactly 100,000 executions; package 5.193s | 10.008s |
| fuzz-calendar | PASS; `FuzzCalendar`; exactly 100,000 executions; package 5.022s | 10.341s |
| dependencies | PASS | 9.910s |

Exact runtime commands, using the recorded Go toolchain:

```sh
go test -p 1 -race -count=1 -cover -json -ldflags=-w -timeout=2m ./...
go vet -p 1 ./...
go mod verify
go test -p 1 -run '^$' -fuzz '^FuzzScheduleInvariants$' -fuzztime=100000x -parallel=1 -timeout=150s schedule.go schedule_test.go schedule_regression_test.go due_with_schedule_test.go
go test -p 1 -run '^$' -fuzz '^FuzzCalendar$' -fuzztime=100000x -parallel=1 -timeout=150s calendar.go schedule.go calendar_fuzz_test.go
```

The full race suite has a two-minute Go test deadline and a five-minute outer deadline. Each fuzz target has a 150-second Go deadline, a 180-second outer deadline, and an exact 100,000-execution budget. The fuzz commands compile the actual production files named in each invocation; complete real-driver race and vet qualification ran separately over `./...`.

Dependency isolation and disabled/HOLD checks ran the exact dedicated-workflow guard. Its `go list -buildvcs=false -deps -test` query avoids incidental managed-worktree VCS stamping; separate Git commands record the tested commit and module tree. The first recurring attempt stopped at that metadata lookup, before the full suite, and is preserved as a failed attempt. It is not counted as a pass. Formatting, exact requirements/indirectness, no replacements/exclusions, package boundaries and module checksums all passed.

## Qualified scope and independent review

The complete suite covers the retained daily/weekly/monthly helper and recovered Calendar APIs; SQLite CRUD, authorization, atomic firings/private reminders, dedupe, restart/concurrency, lifecycle, HTTP/native forms and CLI fixtures. Calendar checks include the retained 3,456-case minute oracle, the separate 10,368-case all-cadence oracle, historical second-offset and gap/fold/date-boundary controls. The reproduced chronology regressions cover strict retention cutoffs, equality, one-nanosecond ordering, foreign-network preservation and tombstones. Production JSON regressions preserve decoder simple-fold aliases while rejecting duplicate names.

The JSON companion lists the independent Calendar/workflow/archive-preservation review at `048c7156133bef0b6a7d394c4e7814a975f1d8b2`, the complete service and chronology/parser review at `63bb886d8951f79e3e2d2633c64096dcbc4240e8`, and root's separate VCS metadata-adjustment review. These scopes collectively cover the preserved implementation and corrective deltas; provider and integration gates remain separate.

After these tests, the dedicated workflow adds the owned PR source branch as a second explicit target for the new stacked recovery draft. That single routing line leaves all test/guard scripts, action pins, permissions and deadlines unchanged. Documentation and descriptive manifest status follow the tested source; all Go files, tests, `go.mod` and `go.sum` retain their tested bytes. No test rerun is inferred from these documentary successors.

## Remaining gates

At this receipt's original source, external PR #110 still owned the pending accepted-main Core baseline and fixed-public-digest test correction. That handoff subsequently merged through #97 into the actual recurring target. The November 20 composition preserves the accepted legacy test under `due_with_schedule_test.go`, retains the recovered `calendar_test.go`, and records separate current qualification in `CONSOLIDATION-20261120.md`. The results above remain historical rather than a newly executed composed-head pass. Actual current-Core authorization/capacity/inbox adapters, host scheduler orchestration and network opt-in, Core invariant regression, real browser/manual accessibility and operational/release qualification remain open. No automatic trade posting is introduced.

The dedicated workflow requires native Linux/macOS race tests and native Windows ordinary tests at the final published head; Windows does not supply race coverage. The provider has not qualified this new recovery tree yet. Historical prototype, archived baseline and prior native results retain their original scope. Coverage percentages measure exercised statements and do not establish feature completion, host integration, policy approval or release readiness.
