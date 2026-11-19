# Standalone implementation validation — 2026-11-16

Go 1.26.11 linux/amd64. `go test -race -cover ./...` passes, 80.3% statement coverage. `go vet ./...` passes.

Tests verify independently constructed two-posting trades, flow versus stock, own-only trends, reversal privacy suppression, ordinary-member admin denial, invalid ledger rejection, calendar boundaries, escaped HTML and aggregate-only CSV. No dependency is required for optional charts: an original descriptive HTML table is the no-JavaScript presentation. This deliberately defers optional Chart.js reuse; no upstream code was copied.

Unregistered read-only HTTP/CLI seams include denial/body-slice/method/format/tenant tests and need independent adversarial review. Actual source must authenticate current membership/admin privilege and return one consistent read-only DB snapshot. No root files, production routes, ledger writer or background work are added. No database schema/cache is introduced, so there is no module migration to roll back. Full-ledger regression, DB restart/backup restore and platform accessibility require later adapter qualification.

Admin release granularity is fixed per network (month by default); overlapping week/month requests are rejected. Source snapshots must exactly match the requested completed-period cutoff, including historical stock. Independent privacy audit remains necessary for repeated releases, dominant cohorts and stock/flow differencing. Small-cohort and dominant-transfer suppression alone is not a proof of anonymization. Integration remains HOLD; no feature-completion or release claim.

Runtime govulncheckv1.1.4 on the committed standalone package: no vulnerabilities found. This does not replace independent privacy review or core adapter qualification.

## Fresh independent audit

See AUDIT.md for findings fixed, requirement-to-test matrix and residual privacy/integration gates. Final audited race/cover suite passes85.9%, vet and module verification pass. Closed-period fuzz passes52,138 executions. Runtime govulncheckv1.1.4 reports no vulnerabilities. Tests include split contributor dominance, stock complementary suppression, source revision privacy, explicit stock/offer network isolation, partial/later reversals, grant/adjustment/system classification, own historical opening/closing, exact monetary JSON strings, no-JS views, CSV metadata/writer failures, cancellation and concurrent stateless reads. No production core, database restore or manual browser qualification is implied.

## Master-scope continuation — 2026-11-16

Tested implementation: `5c239f0494ff9ee69c4e84f985ec15d638140bbd`, recovered from `243f7fc181a14cecb124c4b9733b86c010d53993`. Toolchain: Go 1.26.11 linux/amd64. Environment: `GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache`. Commands ran inside the nested module with `-p 1`.

| Executed check | Observed result |
|---|---|
| `go test -p 1 -run 'TestQueryLimitPrecedesHostCallbacks\|TestPerformanceFixtureAccounting' -count=1 ./...` | PASS, 0.005s |
| `go test -p 1 -race -cover -count=1 -timeout 90s ./...` | PASS, 1.033s, 86.4% statements |
| `go vet -p 1 ./...` | Completed with no diagnostics |
| `go mod verify` | All modules verified; no external module dependencies |
| `go test -p 1 -run '^$' -bench BenchmarkAdminVerifiedSnapshot -benchtime=1x -count=1 -timeout 90s ./...` | PASS; both 12-period fixtures checked exact turnover, counts, stock and dormant values |

Diagnostic benchmark on AMD EPYC 9V74, one iteration per fixture:

| Snapshot events / postings | Time/op | Bytes allocated/op | Allocations/op |
|---|---:|---:|---:|
| 10,000 / 20,000 | 43,317,595 ns | 9,216,096 | 30,749 |
| 100,000 / 200,000 | 497,751,914 ns | 80,868,632 | 302,735 |

These numbers include snapshot validation, twelve calendar periods, privacy projection and fingerprinting. Fixture construction is outside the timer. They are diagnostic measurements, not an adopted latency/memory budget, sustained load test or real database performance result. The maximum fixture reaches the existing event/posting cap; all events are ordinary trades in the most recent period across five equally represented pairs. Existing tests separately cover reversals, suppression and exact large integer semantics.

Raw outputs and checksums are in `master-evidence/`. The coordinator independently reviewed the new guard and benchmark source and found no issue. No browser, screen-reader, external privacy assessment, source restore or current advisory scan ran in this milestone. Previous audit/fuzz/scan records remain tied to their recorded source. Immutable release persistence/query-budget/correction policy remains incomplete. Integration HOLD remains.
