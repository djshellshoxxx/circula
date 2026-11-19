# API-036 implementation and qualification record

Date: 2026-11-19 Pacific. Scope: bounded self-service session inventory,
individual revocation, and revocation of other sessions. This record separates
implemented behavior from execution and integration gates. The current-head
checks and merge receipt are on [PR #101](https://github.com/djshellshoxxx/circula/pull/101),
which targets the [PR #90](https://github.com/djshellshoxxx/circula/pull/90)
integration branch. No deployment or release is part of this work.

## Revisions and preservation

| Checkpoint | Exact revision |
| --- | --- |
| Starting main | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` |
| Starting PR #90 | `244e1356201e6477eab4694d82947f2058f3b0d8` |
| Concurrent bounded inventory correction preserved | `1e2f369a541963540d93c3b7f455b052827839c7` |
| Concurrent HTML cursor-link repair preserved | `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` |
| Reviewed local implementation | `143de8dba59a80002e9152a86576e277849e6738` |
| Published implementation | `6f8ead1951f2d4b72649909b2b4cc83000948afe` |
| Identical local and published implementation tree | `a56e533e88275c05498c39db2c97c5f15b2af2fa` |

GitHub publication used a new branch and non-force updates. The reviewed tree
was compared exactly before publication. Local implementation commits are
provenance, not assertions that those SHAs are remotely retrievable. Subsequent
test/evidence commits and the authoritative current head remain in PR #101.

| Branch examined or integrated | Disposition |
| --- | --- |
| `main`, `integration/core-consolidation-20261119` | Verified baseline; no direct main write. |
| `integration/comprehensive-audit-20261119` | PR #90 baseline and concurrent changes reconciled, including token LIMIT/index and HTML cursor repair. |
| `audit/self-sessions-20261119` | Original `0bca598` session work already represented by the later baseline; not reapplied. |
| `audit/bounded-sessions-20261119` | Preserved existing `853a388` database/service migration and `04413b1` adapter repair. |
| `audit/bounded-session-adapters-20261119` | `8461711` has the same tree as `04413b1`; not duplicated. |
| `audit/bounded-token-list-20261119` | Concurrent `274d907` token correction preserved through PR #90. |
| `codex/api036-database-20261119` | Reviewed reverse query/index and bulk service commits `f92b5c5` and `9f948a0` integrated. |
| `codex/api036-ui-20261119` | API/HTML/docs `74271eb` and disappearing-page repair `4f55560` integrated. |
| `codex/api036-security-20261119`, `codex/api036-cursor-local-20261119` | Canonical cursor repair, fuzz targets and allocation regression integrated. |
| `codex/api036-cursor-hardening-20261119` | Draft PR #96 superseded by #101 and closed; branch preserved. |
| `codex/api036-completion-20261119` | Complete focused candidate; merge only after current-head checks and required approvals. |

All fifteen future modules remain outside this diff, isolated, disabled and on
HOLD. Ledger, monetary and governance implementations, dependency versions and
authentication architecture were not changed by this focused follow-up.

## Completed behavior and defects repaired

- Database inventory uses owner-scoped tuple seeks on immutable
  `(created_at, public_id)`, with `LIMIT` capped at 100. The service inspects at
  most twenty batches of 100 retained rows per request, including expired and
  revoked rows. It never loads an owner's entire retained history to paginate.
- Independent 256-bit public selectors, unique and immutable in migration
  0020, support direct owner-scoped confirmation and revocation. Existing bearer
  hashes and session lifetime fields are preserved. Migration 0021 adds the
  partial unrevoked-owner index used by counted bulk revocation.
- HTML offers Previous, Next and First links. Empty filtered windows retain
  progress cursors. If the newer side of a previous link disappears, a bounded
  refresh includes the surviving boundary; the empty query consumes the same
  twenty-query budget. Forward exhaustion remains terminal, including exact
  2,000-row boundaries.
- The shared dynamic pager previously produced `#ZgotmplZ` for a timestamp-colon
  cursor, causing a repeated first page. Literal session-specific query prefixes
  preserve normal contextual escaping. Actual rendered-link tests verify exact
  decoded cursors, no fragments and no escaping failsafe in either direction.
- Cookie-authenticated API and HTML flows list current/other live sessions,
  expose creation/activity/effective-expiry metadata, revoke one session, and
  revoke other sessions while keeping the current one. Bulk revocation clears
  pending owner MFA challenges and preserves foreign sessions and API tokens.
- Both mutations revalidate stored authority inside their write transaction.
  Revocation, pending challenge removal and minimal audit insertion commit or
  roll back together. Subsequent authentication rejects a revoked credential.
  A fresh password login that commits afterward may create a new session; the
  operation does not introduce an authentication epoch or change login policy.
- Signed cursors reject input above 4,096 bytes before decoding/MAC work and
  require canonical payload/signature encoding. Filter-size budgeting prevents
  issuing cursors that cannot be decoded. Encoding aliases never bypassed the
  HMAC or owner scope; their acceptance and resource costs were repaired.
- Raw positions reject input above 85 bytes before integer parsing. The prior
  oversized numeric-error path allocated approximately the input's full size.
  HTML query parsing now rejects malformed escaping and duplicate/unknown keys;
  signed malformed session positions receive the API's generic invalid-cursor
  response.
- OpenAPI, route authorization and contract fixtures, localization, help,
  operator guidance, browser inventory and no-JavaScript lifecycle coverage
  reflect the implemented endpoints and navigation.

## Independent security and integration review

Separate database, security and API/UI workers reviewed the integrated source
and each other's relevant fixes. A later independent review verified the exact
published tree, 46-file source diff and preserved scope. No unresolved blocking
source finding was identified at the implementation checkpoint.

Review covered live authentication and role lifetimes, ownership, CSRF and
confirmation, fixation and replay, selector uniqueness, cursor boundaries,
revocation races, rollback, confidential auditing, error equivalence and query
exhaustion. Responses, URLs and audit records contain no bearer tokens or stored
credential hashes. Audit payloads retain only person targeting and current/count
disposition. This engineering review is separate from the project's external
SEC-033 release assessment.

## Measured performance

The actual SQLite/service stress regression traversed 10,000 dead retained rows
to the current session in six requests. A capped request at both **10,001 and
50,001 retained rows** made **20 inventory queries, fetched 2,000 rows and
allocated 30,865 objects**. The measurement uses the actual production driver
through a counting repository wrapper.

Go 1.26.11, Linux/amd64, `GOMAXPROCS=2`; each benchmark below used 100 timed
iterations after untimed fixture construction. Shared-host latency is a local
measurement, not a service-level or reference-hardware performance guarantee.

| Retained rows | Page, 25 rows | ns/op | Bytes/op | Allocations/op |
| ---: | --- | ---: | ---: | ---: |
| 100 | First | 261,311 | 12,507 | 352 |
| 100 | Deep | 922,698 | 12,712 | 356 |
| 10,000 | First | 510,251 | 12,840 | 377 |
| 10,000 | Deep | 490,322 | 13,064 | 382 |
| 100,000 | First | 265,151 | 12,840 | 377 |
| 100,000 | Deep | 545,955 | 13,064 | 382 |

Rejecting a 1 MiB malformed raw position used 176 bytes / 3 allocations; rejecting
a 1 MiB API cursor used 128 bytes / 1 allocation. The earlier raw parser probe
used 1,048,800 bytes / 5 allocations. These measurements and their limits are
distinct from input buffers already created by the caller.

[The database review](API036_DATABASE_REVIEW.md),
[raw SQL measurements](API036_SQL_MEASUREMENTS.json) and
[portable probe](api036-session-sql-probe.py) preserve the independent SQLite
3.53.1 evidence. First/deep/previous fixed-size pages used 316/449/443 VM
instructions at 100, 10,000 and 100,000 history rows. Direct lookup used 22;
repeat bulk revocation used 21. B-tree seeks still have size-dependent storage
work. Initial bulk writes scale with the rows changed; neither whole-history
traversal nor mass mutation is claimed to be constant cost.

## Execution and merge gates

The focused production store/service/API/web/contract command passed on the
preceding integrated source `4cc0a1b`: package times 1.480s, 6.596s, 18.800s,
1.975s and 2.362s. The contract exercised 88 operations and 170 distinct
operation/status pairs. The final reverse-only correction's ten pager cases
passed in 0.070s before integration.

The first complete `go test -count=1 ./...` run on the published implementation
found two fixture compatibility failures: the generic migration rollback test
assumed at least two objects per migration, and the standalone help renderer
did not register the production localization function. These are recorded as
failures, not discarded passes. Their repairs preserve rollback/reopen and
rendered-content assertions. The rollback fixture now fails in an AFTER INSERT
trigger on the target migration record, after the migration SQL ran, and checks
the specific injected error before verifying complete rollback and retry. Its
actual red/green package results were 0.132s / 0.082s. The help fixture registers
the actual `messages.Text` function and also asserts localized session help;
both support-boundary tests passed in 0.011s. No production SQL was padded and no
original support-boundary assertion was removed.

The local full Playwright attempt failed in global setup before test assertions:
Chromium aborted at `process_singleton_posix.cc` because `socket()` returned
`Operation not permitted`. The existing launch configuration was preserved.
This is blocked browser execution, not a browser test pass; the configured
provider browser job remains required.

Earlier commands interrupted by workspace resource failures or terminated
during compilation are not passes. SQL probes, source-file tests and fuzz seed
execution do not substitute for full driver/HTTP/browser or mutation campaigns.
Current-head full/race/vet/check, browser, mutation fuzz and native provider
qualification remain the merge gates. The `ci:cross-platform` label is set on
PR #101; a subsequent synchronize event must instantiate Windows and macOS jobs.

Follow [PR #101](https://github.com/djshellshoxxx/circula/pull/101) for the exact
qualification head and checks, then PR #90 for Core consolidation. Do not merge
based on a historical head, a skipped platform placeholder or source review
alone. No force push, branch deletion or branch-protection bypass is authorized
by this record.
