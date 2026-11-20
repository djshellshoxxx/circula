# AUTH-04 password-work admission — execution receipt

Date: 2026-11-20 UTC. Base: `21933a3c650690db5a8e1e44cc236d8d1b5cdb24`.
Local implementation branch: `fix/core-password-admission-20261120`.

The change implements the aggregate admission invariant identified in AUTH-04
of the read-only security audit at commit
`896d69f472652ed54c051ec82efcb7f162d1d2c4`. The implementation and compatibility
boundaries are documented in
[password-work admission](../../../dev/password-work-admission.md).

## Result

One immutable process default admits two password-work operations with no queued
waiters. All production Argon2 evaluations use an admitted ticket: service
startup, registration, sign-in/dummy verification, password change,
recovery/reset, password-protected MFA operations, and backup passphrase
derivation. The ordinary application does not inject local service budgets.

Admission precedes credential repository reads and durable guessing reservations.
Full capacity returns `unavailable`; the existing HTML and JSON mappings produce
503 with `Retry-After`. A single ticket covers sequential old-password proof and
replacement hashing. Cancellation retains capacity until synchronous work has
returned, then prevents credential issuance or the protected change. Existing
guessing reservations are not refunded after admitted cancellation.

The code retains the existing PHC parameters/range, dummy-work equivalence,
stale-proof checks, recovery-code atomicity, session authority and backup format.
The bound counts operations and is not a 128 MiB RSS promise. Minimum-hardware
sizing (DEC-006), byte-weighted admission and new historical-PHC migration policy
remain outside this milestone. Existing context-free startup and backup stream
constructors keep their synchronous behavior. Adapter authentication, CSRF and
independent request rate limits retain their original ordering.

## Executed validation

All Go commands used Go 1.26.11, `GOTOOLCHAIN=local`, `GOMAXPROCS=1`, `-p 1`,
`GOGC=20`, and `GOMEMLIMIT=512MiB`, with the session's pinned caches. Exact command
arguments, environment, package terminal events, source-file hashes and log
hashes are in [manifest.json](manifest.json).

Counts below include successful executions repeated across runs. The event
column includes named subtests as well as their enclosing tests.

| Receipt | Top-level passes | Passed test/subtest events | Failed tests | Skipped tests |
|---|---:|---:|---:|---:|
| [Focused service race](service-focused.jsonl) | 20 | 50 | 0 | 0 |
| [Backup and HTTP adapter race](adapters-backup-race.jsonl) | 11 | 11 | 0 | 0 |
| [Final admission and stale-proof race](admission-final-race.jsonl) | 17 | 44 | 0 | 0 |
| [Credential regressions and bounded property fixtures](credentials-fixtures.jsonl) | 96 | 219 | 0 | 1 |
| [Final limiter race after lint correction](limiter-final-race.jsonl) | 8 | 8 | 0 | 0 |

Across those runs, **116 distinct top-level tests and 239 distinct test/subtest
names passed**, with **zero failed tests** and **one skipped test**. The skip is
the existing `TestLoginTimingStatistical`, which requires the
`CIRCULA_TIMING_TEST` opt-in; no statistical timing result is claimed. The
property run selected four QA-001 sequences across four shards and also checked
isolated replay. The existing concurrent registration and durable guessing/code
consumption tests retained their original assertions with explicit fixture
capacity.

The first focused run preceded the explicit two-operation reentrant test scopes
and the additional cancellation-before-replacement regression. The final
admission race receipt reran those changes. All listed race executions completed
successfully without race reports. The final limiter race followed the test-only
staticcheck correction described below.

[Benchmark smoke](benchmark-smoke.jsonl) executed all three Argon2 benchmark
variants once each with `GOMAXPROCS=1`. This validates the adapted benchmark
ticket wiring; it is not a hardware sizing, throughput, exhaustion or release
benchmark. [smallchecks-execution.json](smallchecks-execution.json) records the
final limiter and benchmark subprocess exit codes and UTC timestamps.

### Static checks and resolved setup issues

- `gofmt -l` reported no differences across all 22 changed/new Go source files.
- Both unstaged and staged `git diff --check` completed successfully.
- The final changed-code `golangci-lint` 2.15.0 run returned exit 0 and
  [0 issues](lint.log). It covered the passwordwork, service, command, API and
  web packages with `--new-from-rev=HEAD`; `HEAD` was the base commit above.
- The [first lint invocation](lint-env-failure.log) failed before analysis
  because the linter's child process selected an unrelated `go` executable.
  Adding the pinned Go directory to `PATH` repaired the environment.
- The next invocation found [one SA4000 test assertion warning](lint-source-before.log)
  on comparing two identical `Process()` call expressions. Comparing two named
  return values preserves the shared-identity assertion. Lint and the eight
  limiter race tests then passed.

## Integration limits

This receipt qualifies the isolated implementation and records every failed
lint attempt as well as its successful retry. It does not claim complete
repository CI, a new fuzz campaign, a dependency vulnerability scan or a full
repository test run. The consolidation owner will requalify the combined branch
and attach an independent review to the implementation commit. No dependency
versions, database schema or future modules changed. This worker performed no
remote mutation, deployment or release.
