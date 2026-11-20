# Pinned lint qualification and two test repairs

Observed on 2026-11-19. The repository's configured `golangci-lint` v2.15.0
reported two findings on frozen source
`e0d27cc7cb73299219439919b260991b824d594a`. Both are resolved in
`e77559a1942c594ea3d9b6b6d8c5171c54b7aae5`, based on integration source
`a4dd964802d06324a5e4a465b2ec4cebed45abd5`. The repair changes only two test
files: six inserted lines and four removed lines.

## Before and after

| Qualification | Exact source | Result | Command wall time |
| --- | --- | --- | ---: |
| Full configured lint before repair | `e0d27cc7cb73299219439919b260991b824d594a` | Exit 1; two findings | 78.301 s |
| Focused normal store/web tests after repair | `e77559a1942c594ea3d9b6b6d8c5171c54b7aae5` | Exit 0; eight top-level tests and twelve subtests across two packages | 29.753 s |
| Full configured lint after repair | `e77559a1942c594ea3d9b6b6d8c5171c54b7aae5` | Exit 0; zero issues | 31.334 s |

All three invocations completed without a resource-watchdog termination. Their
source heads and trees stayed unchanged, and each checkout remained clean.
The after-repair tree is `69166d097603b050f5f5a73d7c81deb030279dd1`.

The original diagnostics are preserved in [before-lint.json](before-lint.json)
and [before-lint.log](before-lint.log). Locations refer to the frozen before
source; the later acknowledgment-loss regression shifts the storage file's line
numbers on the repair base.

| Linter | Before-source location | Diagnostic | Repair |
| --- | --- | --- | --- |
| `nilerr` | `internal/store/recovery_internal_test.go:83` | Error is not nil, but the branch returns a nil error. | Assign the deferred-fault row wrapper to `rows`, then use the existing successful-query return. |
| `staticcheck` | `internal/web/terms_rules_leave_catalog_test.go:148` | `QF1003`: use a tagged switch on `count`. | Use a tagged switch with the same zero, one and plural assertions. |

The storage fixture deliberately injects a read failure through
`storageFaultRows.Next` after one real row, while the underlying query succeeds.
The wrapper is still created only when the injected fault is nonnil. Query
errors, the raw rows returned without an injected fault, fault consumption,
cleanup, and the later stream error remain unchanged. The change makes the
query/row distinction explicit without suppressing the linter.

The catalogue change preserves all expected sentences, amounts, identifiers,
escaping checks, departure rules, blockers, and form/control assertions.
No production Go file, linter configuration, exclusion, or `nolint` directive
was changed.

## Executed scope

The same twelve configured checks were active before and after:
`bodyclose`, `depguard`, `errcheck`, `gofmt`, `gosec`, `govet`, `ineffassign`,
`misspell`, `nilerr`, `staticcheck`, `unconvert`, and `unused`.
Default test analysis stayed enabled. The existing repository exclusions and
layer-boundary rules were retained. Both logs show the same 106 findings
filtered by the configured `gosec` test exclusion and 24 findings filtered by
existing `nolint` directives; the two actionable findings disappear after the
test repairs.

The lint invocation was:

```sh
golangci-lint run --verbose --concurrency=2 --timeout=20m \
  --modules-download-mode=readonly \
  --max-issues-per-linter=0 --max-same-issues=0 \
  --output.json.path <receipt-path> ./...
```

The root module and its tests were analyzed using the unchanged `.golangci.yml`.
The added command options constrain resources, prevent module-file writes, and
retain all diagnostics rather than truncating repeated findings. They do not
disable checks or change their rules. The CI workflow's pinned action and
v2.15.0 setting are recorded in [source-readiness.json](source-readiness.json).
Its `verify: false` setting skips the action's remote schema fetch; the linter
still loads the actual repository configuration.

The focused normal selection covers the shared fault connector and the complete
terms/rules/leave catalogue test file:

- `TestCommitAcknowledgmentFailurePreservesPostingAndReplay`
- `TestFailedRollbackDiscardsConnection`
- `TestMigrationMetadataReadFailureStopsBeforeDDL`
- `TestMigrationWriteFailureRollsBackAndRecovers`
- `TestTermsPagesCatalogue`
- `TestRulesCatalogueRowsAndStates`
- `TestLeaveCataloguePreviewAndControls`
- `TestTermsRulesLeaveCatalogueEscapesMessages`

These tests ran with `-p=1 -count=1 -timeout=2m -json` and default vet. Their
full event stream is [focused-tests.jsonl](focused-tests.jsonl). This is focused
normal-mode qualification; it does not replace aggregate or race results.

## Tool identity and resource controls

The initial serialized source installation was interrupted when the executor
became unavailable. No terminal installation result was observed; the attempted
interrupt could not reconnect. The last checkpoint recorded 1,595 MiB
`MemAvailable`, above the 700 MiB watchdog threshold. The post-restart inspection
found the temporary binary and caches absent. This record does not assign a
cause to that executor failure.

The recovery used the official immutable v2.15.0 Linux/amd64 release archive,
avoiding another cold source build. Its 15,585,535 bytes matched both GitHub's
published asset digest and the official checksums file. The checksums file's
own digest also matched release metadata. Startup and build information identify
v2.15.0, Go 1.27.0, `CGO_ENABLED=0`, and clean upstream revision
`25c5369a93aacb8cccea0d46e1569ca4a194564d`. Exact URLs and digests are in
[tool-prebuilt.json](tool-prebuilt.json).

Runtime qualification used the project's Go 1.26.11 toolchain, the single shared
`/tmp/circula-go-cache-93` build cache and `/tmp/circula-go-modcache-93` module
cache, and isolated temporary/analysis directories under `/dev/shm`.
`GOMAXPROCS=2`, `GOMEMLIMIT=1GiB`, `GOGC=100`, and `GOFLAGS=-p=1` were applied.
The linter reported maximum memory of 1,215.5 MB before repair and 437.4 MB after
repair with the analysis cache warm. The frozen before checkout was never edited.

## Machine evidence

- Before invocation: [before-execution.json](before-execution.json),
  [before-lint.json](before-lint.json), [before-lint.log](before-lint.log).
- Focused tests: [focused-execution.json](focused-execution.json),
  [focused-tests.jsonl](focused-tests.jsonl).
- After invocation: [after-execution.json](after-execution.json),
  [after-lint.json](after-lint.json), [after-lint.log](after-lint.log).
- Interrupted installation: [source-install-checkpoint.json](source-install-checkpoint.json),
  [source-install.log](source-install.log),
  [source-install-interrupted.json](source-install-interrupted.json).

The raw files above were copied without rewriting their contents. Absolute
working/cache paths describe the execution environment at the recorded time;
source heads, file digests, argument vectors, exit codes and timestamps identify
the qualification independently of those temporary paths.
