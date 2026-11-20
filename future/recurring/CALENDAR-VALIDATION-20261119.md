> Historical validation of the pre-recovery calendar-only candidate. These results do not qualify the combined SQLite service. See RECOVERY-PROVENANCE.md and VALIDATION.md for current scope.

# Validation record — 2026-11-19

Source baseline: `0437e54897c6d9cfb3f2577807b82dd038493850`, branch `future/fnet-207-recurring-20261116`. The calendar completion changes are confined to `future/recurring/`. A follow-up adds only the dedicated `.github/workflows/future-recurring.yml` qualification workflow. The module remains disabled and on integration HOLD; no root application file, shared migration, existing CI workflow or other future module was modified.

Toolchain: `go version go1.26.11 linux/amd64`, standard library only. Commands ran inside the nested module with `GOTOOLCHAIN=local` and `GOMAXPROCS=2`.

## Executed verification

| Command | Observed result |
|---|---|
| `go test -race -count=1 -cover -timeout=30s ./...` | PASS, 4.684 seconds, 94.4% statement coverage |
| `go vet ./...` | PASS, exit 0, no diagnostics |
| `go test -run '^$' -fuzz '^FuzzScheduleInvariants$' -fuzztime=100000x -parallel=2 -timeout=90s ./...` | PASS, 100,000 executions, 54.418 seconds |
| `gofmt -l .` | PASS, no paths emitted |
| `git diff --check` | PASS |
| Parse `feature.json` and check disabled/HOLD/no future dependencies | PASS |

The measured elapsed times include this execution environment and are not throughput or latency guarantees. Fuzzing used an execution budget and hard timeout; no blocked, killed or timed-out attempt is included as a pass.

### Resumed current-source verification and provider qualification

The seven-file calendar implementation was committed locally as `4b26004065ff669143a985dcd0b3dab99e7099f3`, tree `6aa59de50d2f75f8182b8fa936eb48e78b0bc391`. A subsequent executor outage interrupted a repeat verification pipeline before it produced results; that attempt is not a pass. After access resumed, the actual commit and files were recovered unchanged. No source reconstruction was used.

At that exact recovered commit, `go test -race -count=1 -cover -timeout=30s ./...` passed again in 1.483 seconds at 94.4% coverage, followed by a successful `go vet ./...`. The dedicated workflow and these documentation notes are the only follow-up changes. Its YAML, shell blocks and embedded Python compile checks pass, as does the repository action-pin/permission checker.

The dedicated workflow retains `contents: read`, pinned existing checkout/setup-go actions, no persisted checkout credentials, a 10-minute job limit and a 100,000-execution fuzz budget with both Go and outer process deadlines. It checks the disabled/HOLD and standard-library-only boundary explicitly. All provider checks and independent review must be evaluated against the final published commit; no remote success is inferred from the local source results above.

## Regressions and calendar acceptance

The initial source passed its existing race suite at 95.5% coverage. Four new regressions then failed against that unchanged production source: invalid UTF-8 identities were accepted before JSON key construction, the host-dependent `Local` timezone was accepted, the exact scheduled instant was missed for historical second-offset timezones, and a valid zero instant was confused with an absent occurrence. All four pass after repair.

| Requirement or defect | Evidence |
|---|---|
| Weekly local weekday and exact boundary; no missed-week replay | `TestWeeklyCalendarBoundaries` |
| Monthly date, missing-day skip, leap-year/century behavior | `TestMonthlyCalendarBoundaries` |
| Local calendar date across UTC date changes and skipped whole dates | `TestScheduleUsesLocalCalendarDate` |
| One-hour and half-hour DST gaps/folds, all recurrence types | `TestScheduleDSTPolicies` and existing `TestDSTAndBoundedCatchup` |
| Reject unknown/empty rules, overfrequent expression syntax and contradictory selectors | `TestScheduleValidation` |
| Preserve the original daily key; same-date rule/time edits; separate tenant/template keys | `TestLogicalKeyCompatibilityAndScope` |
| Inactive membership/template snapshots and tenant mismatch | `TestScheduleMembershipAndActivation` |
| Invalid UTF-8 must not be normalized into an existing logical identity | `TestDueRejectsInvalidIdentityEncoding` |
| Explicit named zone; exact second-offset boundary; valid zero instant | `TestDueRejectsHostLocalTimezone`, `TestDueHistoricalOffsetSeconds`, `TestDueAtZeroInstant` |
| Independent exhaustive UTC-minute comparison | `TestCalendarResolutionAgainstMinuteOracle`: 3,456 comparisons across six zones and DST/year-boundary dates |
| Output belongs to the exact requested local minute; never future; stable key and exact boundary | `FuzzScheduleInvariants`: 100,000 executions on the final resolver |

The independent minute oracle found an intermediate resolver defect at the leap-year boundary in extrapolated timezone rules. That attempt failed and is not a verification pass. The final resolver walks interval starts backwards, validates each candidate by its wall-time round trip and selects the earliest valid instant; the oracle now passes. The historical second-offset regression supplies an exact known instant independently of the modern minute oracle.

## Scope and remaining qualification

No third-party source or dependency was added. The original `Template` layout, `Due` signature and valid daily key encoding remain unchanged. `DueWithSchedule` adds explicit daily/weekly/monthly rules. Previously accepted invalid UTF-8 identities and `Local` zones now fail validation; there is no silent normalization or host-dependent fallback.

These are tests of pure calendar computation using trusted adapter snapshots. They do not establish durable firing uniqueness, fresh production authorization, quota/retention policy, atomic reminder writes, crash recovery, browser accessibility, CLI usability, source transaction integration or release readiness. Persistent service work and those acceptance gates remain open. The user must still select and approve future Core integration; `enabled: false` and `integration: HOLD` are preserved.

An independent reviewer inspected the complete seven-file diff at `4b26004065ff669143a985dcd0b3dab99e7099f3` against `0437e54897c6d9cfb3f2577807b82dd038493850`, plus the new dedicated workflow, and found no blocking source or workflow issue. The review covered daily API/key compatibility, interval resolution, DST/calendar semantics, identity validation, oracle independence and workflow permissions/time bounds. It does not substitute for provider execution at the published head. All observed local execution is Linux-only; no Windows or macOS module pass is inferred.

## Historical validation record — 2026-11-16

Toolchain: Go 1.25.4 linux/amd64, standard library only. The root application requires Go 1.26.11; it was not built or tested here. Its files remain unchanged.

```
$ go test -race -cover ./...
ok  	github.com/djshellshoxxx/circula/future/recurring	1.022s	coverage: 95.5% of statements
$ go vet ./...
PASS (exit 0, no diagnostics)
```

Manual source review checked: no app/internal imports, no cross-feature imports, no init/registration or background jobs, no external dependencies, no network transport or production DB writes, and no ledger posting API. All feature inputs requiring trusted current state are documented as adapter responsibilities.

The suite tests this initial core milestone only. Coverage is not proof of security or feature completion. Durable stores, fresh authorization adapters, full acceptance tests, browser/CLI experience and independent security review remain unfinished as detailed in README.md. No claim of race-safe DB persistence, production qualification or release readiness is made.
