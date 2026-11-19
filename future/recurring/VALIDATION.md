# Recovered composition validation — 2026-11-19

Dependency isolation lists packages with `go list -buildvcs=false -deps -test`. This disables only Go's incidental VCS stamping for the dependency query, after the managed worktree produced a VCS-status error despite successful direct Git reads. The dedicated workflow separately records the exact Git head and module tree; the dependency allowlist, disabled/HOLD assertions and actual test/vet flags are unchanged. The initial failed recurring isolation attempt is preserved in the coordinator's evidence and is not a test pass.

**Current combined local qualification: PASS at `e79cd7a3d8bc1830aeb5ee255c044ddfef72f278`.** Full race testing passed 48 top-level tests with zero skips and 80.0% statement coverage. Vet, checksums, formatting, exact dependency/HOLD checks and both 100,000-execution fuzz targets passed. See `RECOVERY-QUALIFICATION-20261119.md` and its JSON companion for exact commands, fingerprints, independent review scopes and remaining provider/handoff gates. `RECOVERY-PROVENANCE.md` records the preserved interfaces and archive identities.

All changes are confined to this nested module and its dedicated `.github/workflows/future-recurring.yml`. The feature remains disabled, integration HOLD, with no root application, root dependency, shared workflow, migration registry or other feature changes.

The complete local suite used the actual pinned `modernc.org/sqlite v1.60.1` driver and default build tags. Independent Calendar and service/chronology/parser reviews are recorded separately. Native Windows ordinary tests and Linux/macOS race tests must still run against the new published head; no historical provider result transfers to a changed tree.

## Executed calendar reconciliation probes

These checks ran with Go 1.26.11 linux/amd64, `GOTOOLCHAIN=local`, `GOMAXPROCS=1`, `GOGC=20`, `GOMEMLIMIT=450MiB`, `GOWORK=off` and `GOFLAGS=-mod=readonly`. They explicitly compile calendar files only; they do not qualify SQL, HTTP, CLI or the whole combined module.

1. With the archived `calendar.go` unchanged, `go test -p 1 -run '^TestRecoveredCalendar' -count=1 -timeout=45s schedule.go calendar.go schedule_test.go calendar_recovery_test.go` failed as expected. All 24 malformed identity cases were accepted by direct `Window` / `Next` calls before JSON key hashing. The service already rejects those malformed identities at its own boundary; no service authorization bypass was reproduced. Historical Paris UTC+00:09:21 also exposed a missing midnight window and skipped still-upcoming minutes. The 10,368-case independent modern minute oracle passed.
2. Independent source review identified that a current-date-only preflight would not protect the following date. The added after-target-plus-61-seconds regression failed for all 12 historical daily/weekly/monthly cases against that intermediate correction. The intermediate attempt is not a qualified fix.
3. The final adapter enumerates eligible dates at fixed UTC noon through the pinned cron parser/iterator, then uses the retained exact timezone-interval resolver to select each local minute. It keeps a five-candidate bound, current-date-only windows, first-fold selection, skipped gaps/dates, and explicit zero-clock rejection. Malformed UTF-8 identities now fail before key construction; valid literal U+FFFD and Unicode keys retain exact JSON bytes.

The complete combined calendar check passed:

```sh
go test -p 1 -race -count=1 -timeout=45s \
  schedule.go calendar.go schedule_test.go schedule_regression_test.go \
  due_with_schedule_test.go calendar_test.go calendar_fuzz_test.go \
  calendar_recovery_test.go
```

Observed result: **PASS, 3.343 seconds** (4.866 seconds including the warm build). This includes all retained prior calendar regressions, both fuzz seed corpora, the original 3,456-case independent oracle, a separate 10,368-case oracle across all three service cadences, 24 malformed-identity negatives, valid Unicode controls, 12 historical second-offset cases, and 12 explicit next-date gap/fold/half-hour/dateline/month-end/leap/century cases. That focused command executed fuzz seeds only. Both full 100,000-execution budgets subsequently passed on the combined source recorded above.

`gofmt -l`, authored-source whitespace checks, the action-pin/permission checker, and workflow YAML/embedded Python/bash syntax checks passed. The complete recovered diff reports one trailing-blank-line warning in the byte-preserved upstream `third_party/modernc_org_memory/LICENSE-MMAP-GO`; its original license bytes are retained, not silently reformatted. The subsequent combined local race, vet, checksum and fuzz results are recorded above; final publication review and exact-head native provider checks remain separate.

## Chronological cleanup, inbox and JSON follow-up

Independent root review found variable-width RFC3339Nano text comparisons in Cleanup and incomplete Unicode alias equivalence in duplicate-field rejection. The related recent-inbox ordering used the same timestamp representation. Each behavior was reproduced before its correction, using the actual `modernc.org/sqlite v1.60.1` driver and the production parser/service.

- The retention regression uses five whole/fractional/nanosecond/year-boundary cutoffs and seven surrounding instants per cutoff, including equality and exactly one nanosecond before/after. It checks reminder/audit deletion, operation-response retirement, unchanged foreign-network rows, and preservation of firing and operation tombstones. Raw TEXT comparison deleted newer rows and retained older rows; the operation `julianday` comparison also retained three one-nanosecond-old cases.
- The corrected comparisons give the module's canonical UTC timestamps exactly nine fractional digits in SQL, removing the terminal `Z` before padding and retaining strict `<` comparison. The cutoff has the same fixed format. This guarantee applies to canonical UTC timestamps written by the module; the expression does not validate or repair arbitrary malformed database timestamps. Retention policy, storage representation and tombstone rules are unchanged.
- The Inbox regression creates reminders through public Save/RunDue calls. Raw descending text order returned an older whole-second row ahead of a newer fractional row, and a shorter fractional spelling ahead of a later one-nanosecond value. The corrected ORDER BY uses the same exact timestamp order while retaining the ID tie-break, limits, owner/network scope and current authorization.
- Direct `decodeJSON` regressions accepted Version/Verſion duplicates before the repair, including reverse order, escaped long-s and nested objects. The comparison key now uses Unicode simple-fold equivalence, matching the decoder. Valid single aliases and unknown-field rejection are retained; Straße and STRASSE remain distinct because no multi-rune expansion is used. This is an ambiguous-input strictness repair, not a reproduced authorization bypass.

The unchanged-source retention/parser reproduction failed as expected in 0.767 seconds (5.378 seconds including build). The separate unchanged-Inbox-order reproduction failed as expected in 0.584 seconds (4.965 seconds including build). Those failures are retained in the consolidation evidence and are not counted as passes.

```sh
go test -p 1 -race \
  -run '^TestRecovered(CleanupChronologicalBoundaries|InboxChronologicalOrder|JSONSimpleFoldDuplicates|JSONSingleAliasesAndUnknownFields)$' \
  -count=1 -timeout=60s ./...
```

Final focused result: **PASS, 1.812 seconds** (6.258 seconds including build), Go 1.26.11 linux/amd64, `GOMAXPROCS=1`, `GOGC=10`, `GOMEMLIMIT=256MiB`, `CGO_ENABLED=1`, `GOFLAGS=-mod=readonly -p=1`, `GOTOOLCHAIN=local` and `GOWORK=off`. The full module is compiled with the real driver, but the command intentionally selects only these four follow-up tests. It is not a full combined-module test result. Root independently reviewed the complete follow-up at `63bb886d8951f79e3e2d2633c64096dcbc4240e8`; the subsequent full combined race/vet/fuzz/checksum pass is recorded above. Provider and external handoff gates remain.

## Existing provider failure and external handoff

The preceding PR #97 secret scan flagged a fixed public SHA-256 occurrence test vector under the local name `dailyKey`; it is not a credential. External PR #110 owns the naming/literal correction together with two already-accepted Core baseline restorations. This recovery candidate preserves the currently reviewed legacy test bytes while that exact-head handoff is qualified. No scanner rule, allowlist or competing digest repair is introduced here. Inherited Core compile/fixture failures are separately coordinated; neither historical provider results nor those external fixes qualify this new composition automatically.

`CALENDAR-VALIDATION-20261119.md` preserves the November 19 calendar-only repair evidence. The section below preserves the recovered November 16 evidence and its earlier follow-ups verbatim. Those results and completion scores are historical, not new passes.

---

# Standalone validation — 2026-11-16

Toolchain: Go 1.26.11 linux/amd64, explicitly located at `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`. `/usr/bin/go` is unrelated and was not used for validation.

## Reproduced checks

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...`: **PASS**, 74.1% statement coverage.
- `go vet ./...`: **PASS**, no diagnostics.
- `go mod verify`: **PASS**, all modules verified.
- Parser-only `go test -p 1 -run '^$' -fuzz FuzzCalendar -fuzztime 3s -parallel 1 calendar.go schedule.go calendar_fuzz_test.go`: **PASS**, 89,883 executions. This command deliberately excludes SQL instrumentation; the full SQL tests pass separately under race detection.
- `govulncheck ./...` v1.1.4 with Go 1.26.11: **PASS**, `No vulnerabilities found.` This is point-in-time dependency/reachable-code evidence, not a security certification.
- Exact resolved graph notice inventory: 26 module roots, no missing root license/notice files. Go module checksums pin the graph. Runtime parser is imported unmodified; no upstream source transplant or scheduler runner is present.

An initial parser-only fuzz invocation mistakenly included HTTP tests without their fixtures and failed to compile. The corrected parser-only invocation above passed; the failure is not counted as fuzz evidence. Early dependency setup briefly modified root go.mod/go.sum due to a working-directory mistake; both were restored exactly to HEAD before any commit. Final diff must contain only `future/recurring/`.

## Requirement-to-test evidence

| Requirement | Concrete reproduced evidence |
|---|---|
| Private CRUD, revisions, retry conflicts | `TestCRUDIdempotencyAuthAndDraft`: stable create retry, conflicting payload, foreign network/owner denial, pause/resume/cancel, stale edit/revision conflict |
| Fresh policy and capacity | Same test revokes current owner; `TestCLIAndFreshDraftDenial` revokes a target before retry; `TestInputAndDisabled` rejects current draft capacity and default-disabled service |
| Daily/weekly/monthly and named zones | `TestCalendarBoundaries`: exact boundary, fold-first, spring gap, monthly 31 skip, leap day, weekly weekday, Lord Howe half-hour gap, descriptor/TZ injection rejection; original daily DST suite retained |
| Atomic firing and current inbox adapter | `TestFiringsAtomicRestartConcurrencyAndSkip`: injected same-transaction inbox failure leaves zero firings, two independent connections race to exactly one local reminder |
| Restart and skip/edit dedupe | Same test reopens DB, retains skip tombstone and prevents edited-hour duplicate; `TestTimezoneEditGuardAndDeparture` changes local date by timezone but cannot refire rapidly |
| Bounded backfill and no starvation | `TestQuotaBoundedScanAndSuspension`: twenty records scanned in batches of three; `TestDowntimeBackfillIsOneCurrentWindow`: five years of downtime creates one current occurrence and no retry duplicate |
| Suspended/departed members | Quota test disables target and suspends all twenty templates without firing; departure test cancels related templates and scrubs note/amount; inactive target does not prevent current owner cancellation |
| Private manual draft only | Stable retry intent and distinct deliberate repeat intent; cancelled template denied; no proposal/accept/ledger interface exists in source |
| HTTP/no-JS/CLI | `TestHTTPNoJSAndBoundary`: real template render, native form save, escaping, CSRF/auth/network mismatch, unknown fields, 64 KiB bound, unsupported method; CLI save/draft/preview shares service |
| Strict request parsing | `TestStrictJSONAndInactiveTargetCancellation`: duplicate nested/root keys, unknown fields and trailing JSON rejected; UTF-8/schedule/amount invalid cases rejected in service tests |
| Migration/retention/backup | `TestBackupRestoreRetentionAndMigrationRollback`: failed migration rolls back partial schema; private response retirement does not allow replay; firing tombstones survive; closed-file restore is reopened and integrity checked |

## Authority and qualification limits

The library owns only isolated feature data and imports no driver automatically. Every mutation uses an explicit SQLite writer transaction. The core `Authorize` and optional `Remind` adapters are **contracts tested with synthetic fixtures**, not actual Circula service wiring. A same-transaction callback's exactly-once local insertion does not claim exactly-once remote delivery. No remote delivery or external scheduler is implemented here.

No root file, ledger posting endpoint, migration registry, route, CLI registration, release workflow, job or configuration is changed. No production DB, member record, key or message was used. The nested module is excluded from root `go test ./...`; test it explicitly.

Fresh independent audit completed with fixes and adversarial regressions; see AUDIT.md. Actual core transactional policy/inbox compatibility and ledger regression, host network opt-in/scheduler recovery/fairness, real browser/device/keyboard/WCAG/screen-reader tests, operational backup/lifecycle erasure and production license/release approval are not qualified by these results. No full-feature or release-completion claim. Integration remains **HOLD**.

## Fresh independent audit rerun

At the post-fix audited tree, Go 1.26.11 nested-module validation passed:

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...`: **PASS**, 76.8% statements.
- `GOMAXPROCS=1 go vet -p 1 ./...`: **PASS**.
- `go mod verify`: **PASS**, all modules verified.
- Audit rerun `govulncheck ./...` v1.1.4: **PASS**, `No vulnerabilities found.` Runtime/reachable-code scan; not test-inclusive or a security certification.
- Expanded parser-only `go test -p 1 -run '^$' -fuzz FuzzCalendar -fuzztime 3s -parallel 1 calendar.go schedule.go calendar_fuzz_test.go`: **PASS**, 60,023 executions. The corpus now exercises Window and Next, including valid daily/weekly/monthly seeds and a no-backward-Next assertion.

Added `TestAudit*` regressions reproduce and prevent cached-draft stale authorization, historical/private retry leakage at departure, silent ignored writes, missing lock state, incompatible schema and partial-migration persistence, denormalized/cached scope corruption, protocol-relative form actions, case-alias/unpaired-surrogate JSON, fold duplicate skip targets and CLI preview truncation behind 120 cancelled records. Existing two-connection worker, atomic inbox failure, restart, bounded scan, retention/restore, state, HTTP and CLI suites still pass.

The first new fold test incorrectly assumed Vancouver still falls back in November 2026; current tzdata does not. It was corrected to New York's actual November 2026 fold and passes. This test correction is not counted as a code failure in the final result. No real browser, screen-reader, production network opt-in/scheduler or actual core/ledger qualification is inferred. Integration remains HOLD.

## Master continuation: current exact source

Runtime/test source `688c22ac2bb01fa66fbbb5512643c085a699d9ca`, branch `codex/future-recurring-master-20261116`. Go 1.26.11 linux/amd64 at `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`; environment `GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache`. All checks ran from this nested module.

- `go test -p 1 -run '^TestMaster' -count=1 -timeout 90s ./...`: PASS, 0.012s.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS, 2.447s, 78.1% statements.
- `go vet -p 1 ./...`: PASS, exit0, no diagnostics.
- `go mod verify`: PASS, all modules verified.
- `go test -p 1 -run '^$' -fuzz FuzzCalendar -fuzztime 2s -parallel 1 calendar.go schedule.go calendar_fuzz_test.go`: PASS,55,117 executions. Calendar-only fuzz excludes SQLite instrumentation; full SQL suite ran with race separately.
- JSON manifest/HOLD parse, OpenAPI YAML parse and `git diff --check`: PASS. All33 retained third-party notice files are present; dependencies unchanged.

Raw logs and SHA256 manifest are under `master-evidence/`. Coordinator independently reviewed the small query correction and regressions; no issue found. The focused tests prove invalid POST queries do not reach session parsing or change revision/activity, and query-free native POST still works from paginated GET pages. Existing9-finding adversarial, lifecycle, restart/concurrency and calendar tests pass in the full suite.

No current advisory scan was rerun for this unchanged dependency graph; earlier point-in-time govulncheck evidence remains historical. No browser/manual screen-reader, actual host scheduler/core transaction adapter, production backup/erasure or release qualification is inferred. No root files or production ledger behavior changed.

## Fresh form-cache follow-up — 2026-11-16

Exact runtime/test source `b4b8aec884c53a78a8586e0d94fad95e9416dc53`. Same Go 1.26.11 constrained compiler/cache settings as above.

- Red reproduction on predecessor 053ffac plus the new test: `go test -p 1 -run '^TestFormParseFailureCannotBeHiddenBySession$' -count=1 -timeout 90s ./...` failed as expected. Malformed discarded field reached Session and paused the template (HTTP 200, active=false, revision 2). Oversized input returned 400 instead of 413 after Session; unknown/duplicate fields reached Session before rejection.
- Focused fixed source `go test -p 1 -run '^(TestFormParseFailureCannotBeHiddenBySession|TestSessionCanReadPrevalidatedForm)$' -count=1 -timeout 90s ./...`: PASS 0.013s.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS 2.611s, 78.6% statements.
- `go vet -p 1 ./...`: PASS, no diagnostics.
- `go mod verify`: PASS, all modules verified.

Raw red/focused/race/vet/checksum output is retained in form-audit-evidence with SHA256SUMS. No new browser, fuzz or advisory result is claimed for this follow-up; calendar/runtime dependencies are unchanged.
