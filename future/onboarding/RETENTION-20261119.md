# Onboarding retention comparison repair — 2026-11-19

This repair starts from the current isolated onboarding branch at
`473cab2fb45a7b67ea0001f9fdc7833ffda45eb9`. It changes the existing SQL
adapter's retention comparison and adds regression tests. The module remains
disabled and on integration **HOLD**. No recovered archival persistence,
curriculum, metrics policy, API or storage schema is substituted.

## Reproduced behavior

The Store contract makes `PurgeStale` an operator retention hook across all
networks. The memory implementation removes a row only when its update time is
strictly before the cutoff. The SQL adapter writes UTC RFC3339Nano text and
previously compared that variable-width text directly with `updated_at < ?`.

That ordering is incorrect within a second. Through the actual current
`SaveProgress`, `PurgeStale` and `LoadProgress` methods, the unchanged SQL
adapter deleted a row at `12:00:00.000000001Z` with a cutoff of
`12:00:00Z`, even though the row was newer. It also retained a row at
`12:00:00Z` with a cutoff one nanosecond later. Different fractional widths
produce the same problem: `.1Z` and `.100000001Z` compare in the wrong
chronological order. A wrong deletion and a wrong retention can leave the total
deletion count unchanged, so the regression checks row membership as well as
the count.

The existing equality behavior and global retention scope are preserved. This
finding concerns retention ordering; it does not establish an authorization
or tenant-access bypass.

## Repair and compatibility

Only `SQLStore.PurgeStale` changes in onboarding production. For the comparison, the
stored UTC fraction is stripped of its trailing `Z` and padded to exactly
nine decimal digits; whole seconds receive nine zeros. The cutoff is formatted
to that same width. Strict `<` then preserves nanosecond ordering and equality
without rounding through SQLite date/time functions.

The query also keeps a raw `updated_at` upper bound for the entire cutoff
second, so the existing retention index can serve the range predicate. The
bound's final `~` sorts after both canonical suffixes, `.` and `Z`; the
exact padded comparison still determines which rows are removed.

The guarantee covers canonical UTC RFC3339Nano values written by the existing
`ts` helper. This comparison does not validate or repair arbitrary malformed
database strings. Existing timestamp parsing and corrupt-row errors are
unchanged. Stored bytes, schema, indexes, policy duration, APIs, dependencies,
network scope, curriculum revisions and existing tests remain unchanged.

## Regression coverage

The same new Store contract runs against the memory implementation and the
existing real-SQLite test factory. It includes twelve cutoff fixtures plus a
curriculum-version case for each backend:

- Whole seconds, one nanosecond, fractional widths, and submillisecond values.
- Strict equality and values one nanosecond before and after the cutoff.
- Midnight, year, leap-day and pre-epoch boundaries.
- Non-UTC caller locations converted by the existing writer and cutoff API.
- The same member IDs in two networks and independently stored curriculum versions.
- Exact removed counts, surviving progress fields and repeated-purge idempotence.
- Retained network copy, metric opt-in and anonymous aggregate counters.

The expected outcome comes from `time.Time.Before`, independently of the
SQL representation. Every fixture is written through the actual Store API.

## Local qualification

Runtime and test source are frozen at
`d2b6555d3c6407d654f6dc9845c220d792fb2050`, whole tree
`81cee292659dc7fc4c1c26e39e73c9fa77a1806e`. This qualification note is
added after that execution; it changes no runtime, test or dependency bytes.

The tests use the module's existing test-only
`github.com/mattn/go-sqlite3 v1.14.53` dependency and
`sqlite_cgo_test` build tag. No driver or version was substituted. Execution
used Go `1.26.11` on Linux, `GOMAXPROCS=1`, `GOGC=10`,
`GOMEMLIMIT=256MiB`, `CGO_ENABLED=1`, `CGO_CFLAGS='-O0 -g0'`,
`GOTOOLCHAIN=local`, `GOWORK=off` and
`GOFLAGS='-mod=readonly -p=1'`.

The selected regression command was:

```sh
go test -json -p 1 -ldflags=-w -tags sqlite_cgo_test -count=1 -timeout=3m \
  -run '^Test(Mem|SQL)PurgeStaleChronology$' ./...
```

Before the production repair, all thirteen memory scenarios passed and eleven
of thirteen SQL scenarios failed. After the repair, the exact same test files
passed all twenty-six scenarios with no skips. The green run took 6.248 seconds
of wall time, including build/link work; package execution took 0.081 seconds.

Full current-module qualification also passed:

```sh
go test -json -race -p 1 -ldflags=-w -tags sqlite_cgo_test -count=1 -timeout=3m ./...
go vet -tags sqlite_cgo_test ./...
go mod verify
gofmt -l sqlstore.go retention_contract_test.go retention_cgo_test.go
git diff --check
```

The full race run passed sixty top-level tests and ninety-nine test events,
with zero skips. It took 60.970 seconds wall time, including build/link work;
package execution took 14.955 seconds. Tagged vet passed in 14.064 seconds.
Dependency verification passed; changed Go files were formatted and the diff
had no whitespace errors.

Raw commands, source/test/dependency hashes, output logs and result summaries
are retained under `evidence/onboarding-retention/` in the consolidation
workspace, specifically `red-20261119T142101Z`,
`green-20261119T142341Z` and `full-20261119T142428Z`.

## Dedicated native qualification

The new `future-onboarding-retention.yml` workflow targets this isolated
module on Ubuntu 24.04, macOS 15 Intel and Windows Server 2022. It retains the
repository's immutable checkout/setup-go action pins, a read-only token,
disabled/HOLD and dependency-boundary checks, and a ten-minute job deadline.
It changes no existing shared workflow.

All three runners require `CGO_ENABLED=1`, an available `gcc` command and the
same pinned `mattn/go-sqlite3 v1.14.53` test driver. The workflow records the
actual compiler and Go versions and fails if the compiler is absent. It does
not invent a compiler path, install an unrelated toolchain or substitute a
driver. Linux and macOS run the race detector; Windows runs the ordinary
native suite. Every platform runs tagged vet and the full tagged test suite;
the JSON result guard requires both real-SQLite Store contracts, the new
retention regression and zero skipped tests. No native provider pass is
claimed by the local Linux execution above.

## Completed-main baseline restoration

The onboarding target contained the same two inherited Core baseline defects
already fixed on accepted main `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.
A separate commit restores only the following exact completed-main blobs:

| Path | Restored blob | Introducing accepted-main commit |
| --- | --- | --- |
| `cmd/circula/auditanchor.go` | `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` | `dc97f7d5b0ae64306a8e2b1e57ec83f233d3d09d` |
| `internal/service/departure_pagination_regression_test.go` | `3b117ecfe2f01e7e2363b9d201503cde6e0f5595` | `7457b618057a8b1757ec643806e9afbfc5ba907e` |

The complete diffs are the missing `os` import and a test-fixture-only proposal
cap of 1000 so the existing 501-proposal departure scenario can be constructed.
The production proposal cap is unchanged. No unfinished active-worker source,
other Core changes or shared CI files are copied. This restoration changes no
onboarding runtime, tests or dependencies, and the local module suite does not
constitute a fresh whole-Core qualification.

These are local Linux results for the current isolated module and its pinned
test driver. They do not qualify a different Core SQLite driver, native macOS
or Windows execution, provider checks on a subsequently published commit,
production integration, feature activation or the wider onboarding acceptance
criteria. The original module's policy, host authorization, practice isolation,
browser/accessibility and integration gates remain.
