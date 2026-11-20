# API-036 implementation and qualification record

Date: 2026-11-19 Pacific. Scope: bounded self-service session inventory,
individual revocation, and revocation of other sessions. The implementation has
been independently reviewed. Other repository activity incorporated
[PR #101](https://github.com/djshellshoxxx/circula/pull/101) into the
[PR #90](https://github.com/djshellshoxxx/circula/pull/90) integration branch at
**03:39:56 Pacific on November 19**. This turn observed that integration; it did
not initiate the merge. The combined Core tree includes additional changes and
has not been qualified by the earlier API-036 test receipts. Qualification of
that combined candidate exposed a WAL fault-fixture failure. A reviewed
12-line test-only repair is published in draft
[PR #111](https://github.com/djshellshoxxx/circula/pull/111). The repaired Core
tree has passed vet, the complete non-race suite, and the focused integrated
session/contract/WAL race checks. A subsequent full store race attempt has no
confirmed completion after the workspace interruption. The later concurrent audit-only
base update is preserved in the reconciled branch. Full current-candidate qualification and
consolidation into main remain pending. No deployment or release is part of
this work.

## Revisions and preservation

| Checkpoint | Exact revision |
| --- | --- |
| Starting main | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` |
| Starting PR #90 | `244e1356201e6477eab4694d82947f2058f3b0d8` |
| Concurrent bounded inventory correction preserved | `1e2f369a541963540d93c3b7f455b052827839c7` |
| Concurrent HTML cursor-link repair preserved; follow-up base | `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` |
| Initially reviewed local implementation | `143de8dba59a80002e9152a86576e277849e6738` |
| Initially published implementation | `6f8ead1951f2d4b72649909b2b4cc83000948afe` |
| Identical initial local and published implementation tree | `a56e533e88275c05498c39db2c97c5f15b2af2fa` |
| Repaired local full-suite candidate | `c75a344c48e28ab679e2e5706dbc21c158454bff` |
| Published equivalent of repaired candidate | `baedf4b6670f6bf39ec593978ce38780c5640215` |
| Identical repaired local and published tree | `1c1f1fe64cd063b41c75fe3fd34ddedf51f7c86f` |
| Reviewed insertion-regression source commit | `aecdde102e43bc11344f7d6f1dcfece42c2a5a34` |
| Tested local API-036 checkpoint, including refreshed traceability | `9cc1fb2f23071354ecdde12d3c6d05693e713eb7` |
| Published API-036 code-and-test checkpoint | `f7ac6d22a1bd1718f6f4e3a840da7a28c83d0cb0` |
| Identical tested local and published API-036 tree | `4829b19db18835e7358c6987a2c3ea154e818e5f` |
| Historical GitHub synthetic merge candidate for PR #101 | `845661a361088596321cd87113cd484f32655e2e` |
| Observed concurrent PR #90 integration head | `992478594e5902f57a39f3772739bee6ab495d24` |
| Combined Core tree, not qualified by earlier API-036 receipts | `3725d720ce7fc41f148b560e5f06c8f9bf5ab221` |
| Reviewed local Core WAL-fixture repair | `9edf59a626c0332a122bb618865673d8b1911a4d` |
| Published identical-tree repair, PR #111 | `ab7d64c5a31653e3a2a14ddb88940e59b4d02892` |
| Identical local and published repaired Core tree | `f1fa093683ca3ec2c39bdb46fcd78e9443622e96` |
| Subsequent audit-only PR #90 base update, parent `99247859...` | `839e86935767288e594914ad711486616a604eca` |
| Audit-updated integration base tree | `9d23b7b81d60b07066444232dee65b0fda071b0d` |
| PR #111 synthetic merge candidate at inspection | `51383fe6e95928b8010593d5e09e276c736224d4` |
| PR #111 synthetic merge tree, including concurrent audit updates | `ee976c3b2dd2cf248b4fd5d016e79b6d9876c3ac` |
| Current main, unchanged at the integration observation | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` |

The historical PR #101 synthetic merge candidate has tree
`4829b19db18835e7358c6987a2c3ea154e818e5f`, with parents `f25d10b6...` and
`f7ac6d22...`. Its contents exactly match the reviewed published candidate.
Between local `c75a344...` and `9cc1fb2...`, only the new insertion regression
and generated traceability changed; production code is identical. Results below
are attributed to the revision actually tested, rather than reassigned to a
later SHA.

GitHub publication used a new branch and non-force updates. Reviewed trees were
compared exactly before publication. Local implementation commits provide
provenance; they are not assertions that those SHAs are remotely retrievable.
PR #101 now records the incorporated implementation. Draft PR #111 on
`codex/api036-qualification-20261119` carries the reviewed WAL fixture repair;
these qualification reports accompany it while preserving concurrent audit
updates from its integration-branch base. PR #90 remains the Core consolidation
PR.

The observed Core commit is an octopus integration containing `f25d10b6...`,
`0a73ee42...`, `f7ac6d22...`, `f29ebb01...`, `c3c991aa...`, and `e1ea0852...`.
It includes additional authentication, export and token changes. Its tree
`3725d720...` is different from the tested API-036 tree `4829b19d...`.
The API-036 receipts therefore remain evidence for their recorded sources;
they do not establish a full pass for `99247859...`. Actual qualification of
that combined source passed vet but failed the full non-race suite on a WAL
fixture assertion. The reviewed repair at `9edf59a...` / published `ab7d64c5...`
changes only twelve lines in the fixture. Its own full non-race and vet passes
are recorded below; no earlier API-036 result is reassigned to it.

The integration branch subsequently advanced to audit-only commit
`839e8693...`. GitHub's PR #111 synthetic merge `51383fe6...` has that base and
published repair `ab7d64c5...` as parents. Its comparison against the repaired
tree contains exactly eight `docs/audits/` files: six reports and two evidence
JSON files, with no production or test difference. The full merge tree is still
different, so exact candidate attribution and CI remain necessary. Those
concurrent audit updates are retained by the reviewed normal merge
`e90433f856a7b64d3c24ad67d59c1b1ac40d9369`, whose tree matches the synthetic
merge tree `ee976c3b2dd2cf248b4fd5d016e79b6d9876c3ac`. The report additions
change only audit/evidence files relative to that reconciliation.

| Branch examined or integrated | Disposition |
| --- | --- |
| `main`, `integration/core-consolidation-20261119` | Verified baseline; no direct main write. |
| `integration/comprehensive-audit-20261119` | Initial baseline and concurrent repairs preserved; other activity integrated Core at `99247859...`, then advanced its audit records to `839e8693...`. Preserve both when incorporating the fixture/evidence follow-up. |
| `audit/self-sessions-20261119` | Original `0bca598` session work already represented by the later baseline; not reapplied. |
| `audit/bounded-sessions-20261119` | Preserved existing `853a388` database/service migration and `04413b1` adapter repair. |
| `audit/bounded-session-adapters-20261119` | `8461711` has the same tree as `04413b1`; not duplicated. |
| `audit/bounded-token-list-20261119` | Concurrent `274d907` token correction preserved through PR #90. |
| `codex/api036-database-20261119` | Reviewed reverse query/index and bulk service commits `f92b5c5` and `9f948a0` integrated; subsequent migration-fixture repair retained. |
| `codex/api036-ui-20261119` | API/HTML/docs `74271eb` and disappearing-page repair `4f55560` integrated; subsequent help-fixture repair retained. |
| `codex/api036-security-20261119`, `codex/api036-cursor-local-20261119` | Canonical cursor repair, fuzz targets and allocation regression integrated. |
| `codex/api036-cursor-hardening-20261119` | Draft PR #96 superseded by #101 and closed; branch preserved. |
| `codex/api036-qa-inserts-20261119` | Independently reviewed insertion-between-pages regression integrated, with refreshed traceability. |
| `codex/api036-completion-20261119` | API-036 checkpoint `f7ac6d22...` incorporated into PR #90 through observed concurrent activity; branch preserved. |
| `codex/api036-qualification-20261119` | Published repair `ab7d64c5...` in draft PR #111; the qualification report commit preserves the reconciled audit-only base update. |

Parallel modifications used isolated worktrees. The final QA change used an
isolated clone with separate Git metadata after worktree creation encountered a
full workspace filesystem. No concurrent source was overwritten. All fifteen
future modules remain outside this follow-up, isolated, disabled and on HOLD.
Ledger, monetary and governance implementations, dependency versions and
authentication architecture were not changed by this focused follow-up.
These scope and review statements describe the API-036 change set. They do not
prejudge the additional changes in the combined Core candidate.

## Files changed

The published `f7ac6d22...` checkpoint changes **58 files, with 2,644 additions
and 90 deletions**, relative to `f25d10b6...`. These totals describe the API-036
checkpoint, not the broader combined Core delta. Later report-only updates must
also be counted separately. The complete API-036 change list is reviewable in the
[PR #101 file comparison](https://github.com/djshellshoxxx/circula/pull/101/files).

The changed files cover repository interfaces, SQLite keyset/revocation queries,
migration 0021, service paging and bulk revocation, API routes and cursors,
server-rendered session controls, localization, OpenAPI/authorization fixtures,
contract/store/service/web regressions, the Playwright lifecycle, CI fuzz
campaigns, traceability, help and operational/audit documentation. Migration
0020 and the earlier bounded-query and HTML cursor fixes were preserved from
the baseline rather than rewritten by the follow-up.

The subsequent Core fixture repair changes one additional file,
`internal/service/sqlite_ioerr_linux_test.go`, with twelve insertions relative
to `99247859...`. It changes neither production code nor an existing assertion.
These repair and concurrent audit changes are separate from the 58-file
API-036 checkpoint count.

After reconciliation, PR #111 changes seven files relative to integration base
`839e8693...`: the WAL fixture above, this completion record,
`docs/audits/API036_QUALIFICATION.json`, and the focused API-036 continuations in
`CONSOLIDATION_FINAL_REPORT.md`, `FEATURE_COMPLETION_STATUS.md`,
`REMAINING_BLOCKERS.md` and `SECURITY_REVIEW.md`. The structured record preserves
the complete 58-file feature list, seventeen command receipts, twenty-eight
log receipt entries with availability, digests and result excerpts, separate fuzz/benchmark evidence,
and the unsuccessful or interrupted attempts. Unavailable earlier raw logs are
identified as unavailable rather than reconstructed as original receipts.

## Completed behavior and defects repaired

- Database inventory uses owner-scoped tuple seeks on immutable
  `(created_at, public_id)`, with `LIMIT` capped at 100. The service inspects at
  most twenty batches of 100 retained rows per request, including expired and
  revoked rows. It never loads an owner's entire retained history to paginate.
- Independent 256-bit public selectors, unique and immutable in migration
  0020, support direct owner-scoped confirmation and revocation. Existing bearer
  hashes and session lifetime fields are preserved. Forward migration 0021 adds
  the partial unrevoked-owner index used by counted bulk revocation.
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
- Pagination remains live between requests. A deterministic SQLite regression
  commits newer and tied-time owner/foreign rows after the first page's `View`
  closes. It verifies every original owner selector exactly once, strict tuple
  descent, page limits, explicit exhaustion, inclusion of a new row below the
  boundary, and first-page visibility of new rows above it. It demonstrates the
  intended creation interleaving without claiming a frozen multi-request
  snapshot. Existing regressions cover touched, revoked and deleted boundaries.

Two unrelated test-fixture assumptions exposed by full qualification were also
repaired without weakening assertions: generic migration rollback no longer
assumes every migration creates multiple objects, and standalone help rendering
uses the production localization function. Their measured red/green evidence is
recorded below.

Combined-Core qualification identified one further fixture compatibility issue:
safe production connection discard could trigger SQLite's last-connection WAL
cleanup before the test inspected its partial-write artifact. The repair adds a
separate real read transaction after the initial WAL truncation. It preserves
the evidence while leaving production `rollbackOrDiscard`, the writer pool,
and every existing fault, state and retry assertion unchanged.

## Independent security and integration review

Separate database, security and API/UI workers reviewed the integrated source
and each other's relevant fixes. An independent integration review verified the
initial exact published tree and 46-file implementation diff. Later reviews
covered the fixture repairs, source/test preservation and current 58-file
candidate. Final QA identified a direct insertion-interleaving coverage gap;
the new regression closed that gap and received a separate database review
before integration. No unresolved blocking source finding was identified by
these reviews. This finding does not imply that uncompleted execution gates
have passed.

Review covered live authentication and role lifetimes, ownership, CSRF and
confirmation, fixation and replay, selector uniqueness, cursor boundaries,
revocation races, rollback, confidential auditing, error equivalence and query
exhaustion. Responses, URLs and audit records contain no bearer tokens or stored
credential hashes. Audit payloads retain only person targeting and current/count
disposition. This engineering review is separate from the project's external
SEC-033 release assessment.

The insertion test review verified transaction boundaries, owner isolation on
both sides of the cursor, timestamp precedence over selector order, equal-time
ordering, no offset-style repetition, and refresh behavior. Source review was
followed by actual execution with the race detector on the tested API-036 tree
`4829b19d...`, and later in the repaired-Core focused race selection.

Database and security reviewers independently approved the WAL fixture repair,
including exact reviewed blob `d1d19fb96357b5eabab43d926e89dffbbcca23e9`.
They traced the cleanup behavior in the pinned modernc SQLite 1.62.0 source
and confirmed that a separate reader prevents last-connection checkpoint/delete
without occupying the sole production writer connection. Production
`rollbackOrDiscard` remains untouched. A real `store: commit:`
`SQLITE_IOERR_WRITE` remains mandatory; a `BUSY` result cannot satisfy the test.
The detailed unchanged assertions and red/green evidence follow.

## Measured performance

The actual SQLite/service stress regression traversed 10,000 dead retained rows
to the current session in six requests. A capped request at both **10,001 and
50,001 retained rows** made **20 inventory queries, fetched 2,000 rows and
allocated 30,865 objects**. The measurement uses the actual production driver
through a counting repository wrapper.

The API-036 checkpoint targeted service race run repeated this production SQLite
stress test successfully. At both 10,001 and 50,001 retained rows it again used
20 queries and fetched 2,000 rows per capped request, with **30,925 allocations
per request at each size under race instrumentation**. Its complete 10,001-row
traversal took six requests. The earlier non-race count of 30,865 and the later
race count of 30,925 are separate measurements; neither is substituted for the
other. Both show bounded per-request work and equal allocation counts across
the tested history sizes in their respective runs.

The repaired-Core focused race run at `9edf59a...` / tree `f1fa0936...`
independently repeated the same stress regression: 10,001 and 50,001 retained
rows each used 20 queries, 2,000 fetched rows and 30,925 allocations per capped
request. The 10,001-row traversal again took six requests.

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
used 1,048,800 bytes / 5 allocations. These measurements exclude input buffers
already created by the caller.

[The database review](API036_DATABASE_REVIEW.md),
[raw SQL measurements](API036_SQL_MEASUREMENTS.json) and
[portable probe](api036-session-sql-probe.py) preserve the independent SQLite
3.53.1 evidence from local `4cc0a1bcd2e5d204da9e06c6e09fc778e7cb55b5`.
All 21 forward migration SQL files were applied by that probe. The production
SQLite regression separately exercised migration failure, rollback and retry.

| SQL operation | 100 history rows | 10,000 history rows | 100,000 history rows |
| --- | ---: | ---: | ---: |
| First 25 rows, VM instructions | 316 | 316 | 316 |
| Deep forward 25 rows, VM instructions | 449 | 449 | 449 |
| Previous 25 rows, VM instructions | 443 | 443 | 443 |
| Direct lookup, VM instructions | 22 | 22 | 22 |
| Repeated bulk, zero changes, VM instructions | 21 | 21 | 21 |
| Foreign current selector, zero changes, VM instructions | 36 | 36 | 36 |
| Initial bulk rows changed | 10 | 1,000 | 10,000 |
| Initial bulk VM instructions | 391 | 34,051 | 340,051 |

Query-plan assertions verified indexed searches without a full scan or a
temporary sorting tree; integrity and foreign-key checks passed. Constant
observed VM counts do not establish constant wall time. B-tree seeks still
have size-dependent storage work, and initial bulk writes scale with the rows
changed. Neither whole-history traversal nor mass mutation is claimed to be
constant cost. SQL probes supplement actual Go driver/service tests; they do
not replace them.

## Execution receipts

Each result below applies to its explicitly named source revision. The first
subsections cover actual combined-Core and repaired-Core qualification. Later
API-036 checkpoint receipts remain historical evidence for their original
trees and do not qualify additional merged changes.

The durable command, revision and result summary is
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). Source log filenames
mentioned below identify the original execution receipts; they do not imply
that the agent's adjacent workspace evidence directory is part of the GitHub
repository.

### Combined Core `99247859...`: vet passed, full suite found a fixture failure

Commands at head `992478594e5902f57a39f3772739bee6ab495d24`, tree
`3725d720ce7fc41f148b560e5f06c8f9bf5ab221`:

```sh
go vet ./...
go test -count=1 -timeout=20m ./...
```

The environment was `CGO_ENABLED=1`, `GOMAXPROCS=2`, `GOMEMLIMIT=2GiB`,
`GOGC=50`, and `GOFLAGS=-p=1`. **Vet passed, exit 0, in 23.322s. The full
non-race command failed, exit 1, after 248.175s.** The service package failed
after 85.277s because `TestSQLiteWALWriteFailureRollsBackAndRecovers` reported:

```text
failed WAL size=0, want partial second frame at 6224
```

This was an actual assertion failure, distinct from the earlier killed or
timed-out attempts. Every other tested package passed, including `cmd/circula`
(59.088s), API (22.394s), contract (3.877s), store (11.566s), and web (32.034s).
Source receipts: `core-check-results.json` and `core-test-all.log`, summarized
in [API036_QUALIFICATION.json](API036_QUALIFICATION.json).

### WAL fixture repair: preserved fault and recovery assertions

Independent diagnosis found that production `rollbackOrDiscard` can discard a
failed writer when rollback fails. If it is SQLite's last connection, normal
close processing can checkpoint/delete the WAL; subsequent fixture reads can
then observe an empty recreated file. The production safety behavior must be
retained.

Local repair `9edf59a626c0332a122bb618865673d8b1911a4d` adds twelve lines to
the fixture. After its successful `TRUNCATE` checkpoint, the child opens a
separate raw reader and starts a real read transaction. A schema-version query
establishes and verifies the snapshot. The separate pool prevents the failed
writer's discard from becoming last-connection cleanup while allowing the
production writer to stage, fail at COMMIT and be discarded normally.

All previous assertions remain unchanged: fully staged posting, balances,
history, audit and idempotency; restored `RLIMIT_FSIZE`; actual COMMIT
`SQLITE_IOERR_WRITE`; exactly 6,224 WAL bytes for this fixture; valid WAL magic
and page size; a non-commit first frame and partial second frame; byte-identical
database; unchanged live and restarted state; no uncommitted sink or metric
publication; exactly-once retry/replay; reopen, integrity and ledger checks.
The repair does not accept `BUSY`, relax the artifact-size assertion, or alter
production `rollbackOrDiscard`.

The isolated WAL regression's package run reproduced the original failure in
**0.103s** and passed after the repair in **0.172s**. Source receipts:
`wal-fixture-red-992.log` and `wal-fixture-green-992.log`. Database and security
reviews approved the exact fixture change before publication. Published commit
`ab7d64c5a31653e3a2a14ddb88940e59b4d02892` has parent `99247859...` and tree
`f1fa093683ca3ec2c39bdb46fcd78e9443622e96`, identical to local `9edf59a...`.

### Repaired Core: full non-race suite and vet passed

Both commands ran at local `9edf59a626c0332a122bb618865673d8b1911a4d`, tree
`f1fa093683ca3ec2c39bdb46fcd78e9443622e96`, identical to published `ab7d64c5...`:

```sh
go vet ./...
go test -count=1 -timeout=20m ./...
```

The recorded environment remained `CGO_ENABLED=1`, `GOMAXPROCS=2`,
`GOMEMLIMIT=2GiB`, `GOGC=50`, and `GOFLAGS=-p=1`. **Vet passed, exit 0, in
1.571s. The full non-race suite passed, exit 0, in 251.654s**, completing at
04:40:54 Pacific. Every tested package passed, including:

| Package | Measured package time |
| --- | ---: |
| `cmd/circula` | 55.620s |
| `internal/api` | 20.332s |
| `internal/contract` | 5.210s |
| `internal/service` | 96.499s |
| `internal/store` | 13.046s |
| `internal/web` | 36.149s |

Source receipts: `core-repaired-vet.log`, `core-repaired-test-all.log`, and
`core-repaired-check-results.json`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). This is actual full-suite
evidence for the repaired combined source, not an inference from the earlier
API-036 checkpoint.

### Repaired Core focused session, contract and WAL race checks: passed

The following command ran at the same repaired Core revision and environment:

```sh
go test -race -count=1 -timeout=20m \
  -run 'Session|TestAPIMatchesOpenAPI|TestFailingMigrationRollsBackAndReopensCleanly|TestSQLiteWALWriteFailureRollsBackAndRecovers' \
  -v ./internal/service ./internal/store ./internal/api ./internal/web ./internal/contract
```

**Passed, exit 0, in 218.628s**, completing at 04:44:32 Pacific. Recorded package
times were:

| Package | Measured package time |
| --- | ---: |
| `internal/service` | 87.744s |
| `internal/store` | 21.341s |
| `internal/api` | 21.550s |
| `internal/web` | 19.662s |
| `internal/contract` | 8.323s |

The OpenAPI contract exercised 88 operations and 170 distinct operation/status
pairs. The actual 10,001/50,001-row SQLite/service stress test retained the
20-query/2,000-row cap and reported 30,925 allocations at both sizes. The real
WAL write-fault regression passed under race instrumentation in 3.53s. Source
receipts: `core-repaired-session-race.log` and
`core-repaired-check-results.json`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json).

This is a completed focused race selection, not a full `go test -race ./...`
pass. A separate full store race attempt with `GOMAXPROCS=1` and
`GOMEMLIMIT=512MiB` was interrupted by loss of the workspace execution
connection around 04:46 Pacific. After access recovered, its log contained
launch metadata only, the original runner transport was unavailable, and no
completion record or exit code was available. Its outcome is **unknown /
interrupted, not passed**. The subsequent static step in that interrupted pipeline
had no execution receipt.

After recovery, reconciliation preserved the concurrent audit-only base update.
A separate final static command ran on local merge
`e90433f856a7b64d3c24ad67d59c1b1ac40d9369`, tree
`ee976c3b2dd2cf248b4fd5d016e79b6d9876c3ac`, with the pending audit/evidence
updates. All runtime and test files were unchanged from tested `9edf59a...`:

```sh
make -j1 fmt-check actions-check hygiene traceability-check
```

It **passed, exit 0, in 27.279s**. Formatting, workflow pins/permissions,
repository hygiene (four tests) and generated traceability were all valid.
The exact source scope and output are preserved in the structured receipt's
`qualification-docs-static` entry. This command excludes the full race, lint
and vulnerability gates; it is not a `make check` pass.
No termination cause or completed result is inferred from the lost connection.
The audit-only integration-base advance and exact PR #111 merge candidate
still require correct attribution and current-head provider qualification.

### API-036 checkpoint full non-race suite and vet: passed

Both commands ran at local `9cc1fb2f23071354ecdde12d3c6d05693e713eb7`, tree
`4829b19db18835e7358c6987a2c3ea154e818e5f`, identical to published `f7ac6d22...`.
The recorded environment was `CGO_ENABLED=1`, `GOMAXPROCS=2`,
`GOMEMLIMIT=2GiB`, `GOGC=50`, and `GOFLAGS=-p=1`.

```sh
go vet ./...
go test -count=1 -timeout=20m ./...
```

**Vet passed, exit 0, in 45.310s. The complete non-race suite passed, exit 0,
in 273.463s.** Every tested package passed in that full-suite command, including:

| Package | Measured package time |
| --- | ---: |
| `cmd/circula` | 66.806s |
| `internal/api` | 23.471s |
| `internal/contract` | 4.262s |
| `internal/service` | 85.507s |
| `internal/store` | 10.954s |
| `internal/web` | 31.178s |

Source receipts: `vet-final.log`, `go-test-all-final.log`, and
`final-check-results.json`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). These API-036 checkpoint
passes do not rewrite the earlier failed attempts recorded below.

### Focused implementation and fixture verification

The focused production store/service/API/web/contract command passed on the
preceding integrated source `4cc0a1b...`: package times 1.480s, 6.596s, 18.800s,
1.975s and 2.362s. The contract exercised 88 operations and 170 distinct
operation/status pairs. The final reverse-only correction's ten pager cases
passed in 0.070s before integration. These are historical results on their
recorded sources, not current-head full-suite claims.

The first complete `go test -count=1 ./...` run found two fixture compatibility
failures: the generic migration rollback test assumed at least two objects per
migration, and the standalone help renderer did not register the production
localization function. These remain recorded failures. The rollback repair
injects failure in an AFTER INSERT trigger on the target migration record, after
the migration SQL ran, and checks the specific injected error before verifying
complete rollback, unchanged data, reopen and retry. Actual red/green package
results were 0.132s / 0.082s. The help fixture registers actual `messages.Text`
and also asserts localized session help; both support-boundary tests passed in
0.011s. No production SQL was padded and no original support-boundary assertion
was removed.

### Earlier repaired full non-race attempt: aggregate failed

Command: `go test -count=1 ./...` at local `c75a344...`, using
`GOMAXPROCS=2`, `GOMEMLIMIT=1GiB`, `GOGC=30`, and `GOFLAGS=-p=1`.
The command exited **1 after 249.294s**. `cmd/circula` reported
`signal: killed` after 60.693s. No assertion failure was reported for that
package; the cause of the process termination has not been established.

Every other tested package passed, including:

| Package | Measured package time |
| --- | ---: |
| `internal/api` | 25.121s |
| `internal/contract` | 4.420s |
| `internal/service` | 89.375s |
| `internal/store` | 10.922s |
| `internal/web` | 39.964s |

These package results verify the repaired migration/help fixtures and other
non-CLI tests in that run. They do not convert its aggregate exit 1 into a pass.

### Earlier `make check`: stopped at vet

Command: `make check` at local `c75a344...`, with the same environment as the
non-race run. It exited **2 after 65.292s**. During `go vet ./...`, the compiler
for `modernc.org/sqlite/lib` and vet process for `modernc.org/libc` reported
`signal: killed`.

This attempt did not complete vet and **never reached the race, lint,
vulnerability, action-pin, hygiene or traceability targets**. The cause of the
killed processes has not been established. Later successful static targets,
listed below, are separate receipts and do not establish a full `make check`
pass.

### Explicit full race run: aggregate failed

Command: `go test -race -count=1 -timeout=20m ./...` at local
`c75a344c48e28ab679e2e5706dbc21c158454bff`. The command completed with **exit 1**.

| Package that did not pass | Observed failure | Package time |
| --- | --- | ---: |
| `internal/service` | Package exhausted its 20-minute timeout | 1,200.067s |
| `internal/store` | Process reported `signal: killed`; cause not established | 164.853s |

At the service timeout, unchanged `TestSignupGrantCap` had been active for
approximately two seconds. The captured stack was executing SQLite schema
initialization, and independent database diagnosis identified migration 0001.
The package's cumulative timeout expired; the log does not show a 20-minute
hang in that individual test. No test assertion failure or race detector warning
was recorded, but the timeout and killed process still make the aggregate run a
failure. The cause of the store termination has not been established.

Every other tested package passed in that race run:

| Package | Measured package time |
| --- | ---: |
| `cmd/circula` | 563.958s |
| `internal/api` | 680.568s |
| `internal/contract` | 52.543s |
| `internal/domain` | 2.439s |
| `internal/histbal` | 1.013s |
| `internal/httpx` | 1.018s |
| `internal/logx` | 1.130s |
| `internal/repository` | 1.186s |
| `internal/web` | 415.693s |
| `internal/web/messages` | 1.036s |
| `scripts/sbom` | 1.088s |
| `scripts/traceability` | 2.394s |

Source receipt: `go-test-race-all.log`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). The tested API-036 checkpoint
has the same production code but adds the insertion regression and generated
traceability. Separate API-036 checkpoint session race results follow; they do not
substitute for a successful complete race suite.

### API-036 checkpoint committed-insertion regression: passed with race detection

Source: local `9cc1fb2f23071354ecdde12d3c6d05693e713eb7`, tree
`4829b19db18835e7358c6987a2c3ea154e818e5f`, identical to published `f7ac6d22...`.

```sh
go test -p 1 -race -count=1 \
  -run '^TestSessionKeysetContinuationAfterCommittedInserts$' \
  -timeout 3m -v ./internal/store
```

**PASS, exit 0**: test 1.21s, package 2.270s, command elapsed 15.932s.
Source receipt: `qa-insertion-race.log`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). This is a real SQLite integration
test with the race detector; no mock store or production replacement was used.

### API-036 checkpoint session service, SQLite and repository race checks: passed

These complete selections ran against local `9cc1fb2...` / tree `4829b19d...`
with actual production code and SQLite. They isolate the session qualification
from the unrelated tests that consumed the full service package's timeout.

```sh
go test -p 1 -race -count=1 -timeout=20m -run Session -v ./internal/service
go test -p 1 -race -count=1 -timeout=20m \
  -run 'Session|TestFailingMigrationRollsBackAndReopensCleanly' -v ./internal/store
go test -p 1 -race -count=1 -timeout=20m \
  -run '^TestRepositoryContract(SQLite|Wrapped)$/.*Session' -v ./internal/store
```

| Selection | Package time | Command elapsed | Exit |
| --- | ---: | ---: | ---: |
| Service tests matching `Session` | 90.947s | 93.712s | 0 |
| SQLite session tests and generic migration rollback regression | 18.639s | 22.492s | 0 |
| Session repository contracts, direct SQLite and wrapped store | 10.224s | 12.159s | 0 |

All three selections passed. The service run includes authority revalidation,
individual/bulk revocation concurrency, current-session revocation races,
pending MFA challenge invalidation, audit/challenge rollback, lifetime checks,
bounded navigation, and real 10,001/50,001-session stress. The SQLite run includes
10,000-row traversal, indexed queries, migration rollback/retry, selector
identity protections and the committed-insertion regression, which passed again
in 1.32s. Both repository adapters passed ownership, pagination and selective
revocation contracts.

Source receipts: `session-service-race-final.log`,
`session-service-race-final-result.json`, `session-store-race-final.log`,
`session-repository-race-final.log`, and `session-store-race-final-results.json`,
summarized in [API036_QUALIFICATION.json](API036_QUALIFICATION.json).
These scoped passes do not turn the earlier full race run into a pass.

### Actual mutation fuzz campaigns: passed

All three campaigns passed, with **Go reporting 100,000 executions per campaign**.
The retained fuzz logs did not record an exact checkout SHA. All production files
and the actual fuzz targets are identical between `c75a344...`, `9cc1fb2...` and
published `f7ac6d22...`; these receipts establish that code's fuzz result without
reassigning a whole-checkout qualification to a later revision.
The counts below are Go's reported execution totals. They used pinned Go 1.26.11 and
`GOMAXPROCS=1`, `GOMEMLIMIT=512MiB`, `GOGC=50`.

The actual invocation prefix for each target was:

```sh
go test -p 1 -run '^$' -fuzz '^TARGET$' -fuzztime 100000x -timeout 3m
```

`TARGET` was the exact target in the table below. The prefix was followed by
the sorted paths of every production `.go` file in that package, excluding
`*_test.go`, then the actual fuzz test file. The API invocation included 37
production files and `internal/api/cursor_fuzz_test.go`; each service invocation
included 73 production files and
`internal/service/session_position_security_test.go`. Explicit file selection
causes Go to label the package `command-line-arguments`. No stubs, copied
algorithm or production edit was used.

| Target | Executions | Go package time | Whole command elapsed | Exit |
| --- | ---: | ---: | ---: | ---: |
| `FuzzOpenCursor` | 100,000 | 28.992s | 168.559s | 0 |
| `FuzzSessionPosition` | 100,000 | 6.159s | 21.202s | 0 |
| `FuzzSessionSelector` | 100,000 | 6.370s | 8.695s | 0 |

Source receipts: `fuzz-production-api-cursor.log`,
`fuzz-production-session-position.log`, `fuzz-production-session-selector.log`,
and `fuzz-production-service-results.json`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json).

These are completed mutation campaigns against actual production package
sources. Their scope excludes unrelated SQLite integration test fixtures; they
are not full-package integration runs. Seed-only execution and earlier
interrupted campaigns were not counted toward these results. The configured
CI fuzz job still must complete on the integration candidate.

### Lint and vulnerability scan: execution incomplete

The initial API-036 checkpoint lint invocation was
`golangci-lint run --concurrency 1 --timeout 10m`, using the same environment as
the final non-race/vet commands. It was killed with recorded exit **-9 after
40.087s**, before emitting a finding. That attempt is not a lint pass. Its source
receipts are `lint-final.log` and `final-check-results.json`; the termination
cause has not been established.

The retry retained the same linters, `--concurrency 1` and `--timeout 10m`, with
`GOMAXPROCS=1`, `GOMEMLIMIT=512MiB`, `GOGC=20`, `CGO_ENABLED=1`, and
`GOFLAGS=-p=1`. It was also killed, with recorded **exit -9 after 42.229s**,
before emitting a finding. Neither lint attempt completed; neither establishes
that lint passed. No termination cause is asserted.

The subsequent vulnerability command used the retry's environment:

```sh
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

It exited **1 after 69.170s**, after module-download messages followed by
`signal: killed`. There is no completed vulnerability scan and no claim of zero
vulnerabilities. Current-head provider execution is still needed for both lint
and vulnerability qualification.

Source receipts: `lint-limited-final.log`, `vuln-final.log`, and
`remaining-check-results.json`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json).

### API-036 checkpoint static checks: passed

At local `9cc1fb2...` / tree `4829b19d...`, the following targets passed:

```sh
make -j1 fmt-check actions-check hygiene traceability-check
```

The final invocation **passed, exit 0, in 5.202s**. The action pin/permission
check passed, hygiene's four tests passed, and traceability reported
`docs/TRACEABILITY.md` up to date. Source receipts: `static-final.log` and
`remaining-check-results.json`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). An earlier API-036 checkpoint
pass of the same targets is also retained in `static-qualification-final.log`.
These targets do not include vet, golangci-lint, govulncheck, or the full test
suite. The API-036 qualification commands recorded here have finished; review
and relevant checks for the subsequently observed combined Core candidate are
separate work.

### Browser execution: blocked before assertions

The local full Playwright attempt failed in global setup before test assertions.
Chromium aborted at `process_singleton_posix.cc` because `socket()` returned
`Operation not permitted`. Existing launch configuration was preserved.
Source receipt: `playwright-all.log`, summarized in
[API036_QUALIFICATION.json](API036_QUALIFICATION.json). No browser assertion is reported as
passed; the configured provider Playwright/axe/keyboard job remains required.

## Hosted CI and consolidation gates

### Observed integration, repaired follow-up and current candidate

GitHub reports PR #101 closed and merged at **03:39:56 Pacific** through other
repository activity. This turn did not initiate that merge. PR #90 advanced to
`992478594e5902f57a39f3772739bee6ab495d24`, tree
`3725d720ce7fc41f148b560e5f06c8f9bf5ab221`; main remains `ead63b5c...`.
The new head incorporates API-036 together with additional authentication,
export and token changes. The historical synthetic merge tree for #101 and its
local test receipts did not qualify that different combined tree. Subsequent
actual Core testing and the reviewed WAL fixture repair are recorded above.

PR #90 then advanced to audit-only base `839e86935767288e594914ad711486616a604eca`,
tree `9d23b7b81d60b07066444232dee65b0fda071b0d`, with parent `99247859...`.
Draft [PR #111](https://github.com/djshellshoxxx/circula/pull/111) is open and
unmerged, publishing the repair at `ab7d64c5...` on
`codex/api036-qualification-20261119` against that integration branch. Its
`ci:cross-platform` label has been added; actual current-head native jobs and
their outcomes still require verification.

The observed PR #111 synthetic merge is `51383fe6...`, tree `ee976c3b...`.
Compared with the tested repair tree, it adds only the eight concurrent
`docs/audits/` files, with no production or test differences. This is source
attribution evidence, not a CI pass. Qualification-report publication must
preserve that audit delta and recheck the resulting exact candidate.

At the latest provider snapshot, **04:46 Pacific**:

| Candidate | Workflow | Observed result |
| --- | --- | --- |
| Earlier Core `99247859...` | [44516848989](https://github.com/djshellshoxxx/circula/actions/runs/44516848989) | All nine jobs cancelled; its earlier queued snapshot remains historical. |
| Current PR #90 base `839e8693...` | [44523175368](https://github.com/djshellshoxxx/circula/actions/runs/44523175368) | All nine jobs queued; native Linux, macOS and Windows instantiated. |
| Initial PR #111 head `ab7d64c5...` | [44526199270](https://github.com/djshellshoxxx/circula/actions/runs/44526199270) | Seven jobs queued, plus a skipped cross-platform placeholder. |

The PR #111 label is present, but its initial run does not provide native
Windows/macOS qualification. The next real report/source commit must instantiate
those jobs and obtain their actual results. No completed Core CI pass is
recorded here. Cancelled, queued and skipped jobs do not qualify a candidate;
the draft follow-up requires current-head provider qualification after its
evidence updates and concurrent audit changes are reconciled.

### Historical API-036 CI checkpoint

At the recorded check around 04:09 Pacific on November 19,
[run 44513297231](https://github.com/djshellshoxxx/circula/actions/runs/44513297231)
was correctly anchored to published head `f7ac6d22...`, with tree `4829b19d...`.
All nine jobs were **queued**, with null conclusions. No hosted failure or pass
was available at that snapshot.

| Job | Job ID | Observed result |
| --- | ---: | --- |
| Linux, Ubuntu 24.04 | 135522745044 | Queued |
| Windows 2022 | 135522744888 | Queued |
| macOS 15 Intel | 135522744926 | Queued |
| API contract and load smoke | 135522745035 | Queued |
| Vet, formatting, lint, vulnerability checks | 135522744870 | Queued |
| Fuzz smoke | 135522744662 | Queued |
| Playwright, axe, keyboard | 135522744325 | Queued |
| Release-target builds | 135522744810 | Queued |
| Reproducible builds | 135522744762 | Queued |

In that API-036 run, Windows and macOS are instantiated native jobs, not skipped placeholders. The
`ci:cross-platform` label and synchronization enabled them. The workflow runs
complete native Linux and macOS race suites, the native Windows test suite,
and the session cursor/position/selector mutation campaigns. Building release
targets in CI does not deploy or publish a release.

Before the subsequently observed merge, the independent integration review at
approximately 03:23 Pacific recorded
PR #101 as open, draft, unmerged and conflict-mergeable. PR #90
was also open, draft and unmerged; its head remained `f25d10b6...`, targeting
unchanged main `ead63b5c...`. No submitted GitHub reviews were returned for either
PR, and no review threads were returned for #101. Independent engineering
reviews above are recorded separately from GitHub review submissions. This
earlier snapshot does not describe PR #101's current merged status.

Visible branch metadata for main and the integration branch reported
`protected: false`, status-check enforcement off and no required contexts;
the repository rulesets collection was empty. Detailed protection reads for
both branches returned HTTP 403, `Resource not accessible by integration`, so
administrative settings could not be independently inspected. No protection
setting was changed. The user's requested verification and ordinary PR review
requirements remain applicable regardless of the visible enforcement metadata.

### Remaining gates at this checkpoint

1. Preserve the completed repaired-Core receipts: `9edf59a...` / published
   `ab7d64c5...` passed full non-race, vet and focused integrated race checks.
   The later full store race attempt has no confirmed completion after the
   workspace interruption; do not count it as passed. The separate final static
   command passed in 27.279s on the reconciled source. Preserve the original
   `99247859...` assertion failure and its red/green fixture proof.
2. Obtain successful complete race and `make check` qualification, including
   completed lint and vulnerability scans. The earlier full race and check
   attempts failed, both lint attempts were killed, and the vulnerability scan
   did not complete. Preserve these receipts alongside the scoped passes.
3. Obtain successful current-head Core provider checks, including native
   platforms, full-package mutation fuzz, API/load qualification and browser
   assertions. Core `839e8693...` has nine queued jobs; the initial PR #111 run
   has seven queued jobs and a skipped platform placeholder. The final PR #111
   commit must instantiate native Windows/macOS jobs. The cancelled
   `99247859...` run and historical API-036 checks cannot substitute for it.
4. Integrate the fixture and qualification evidence through draft PR #111, which
   preserves the audit-only base update `839e8693...`. Finish ordinary review and
   readiness requirements before incorporating that follow-up and merging #90
   into main through normal PR procedures. Reconcile any later base change and
   qualify the exact resulting candidate. This turn has not initiated a PR merge into Core or main;
   main remains unchanged at the recorded observation.

[PR #101](https://github.com/djshellshoxxx/circula/pull/101) provides implementation
history; [PR #90](https://github.com/djshellshoxxx/circula/pull/90) identifies the
Core consolidation PR. [PR #111](https://github.com/djshellshoxxx/circula/pull/111)
is the open draft fixture/evidence follow-up carrying this report and the
structured qualification receipts. The evidence does
not establish the user's complete Definition of Done. No force push, branch
deletion, branch-protection bypass, deployment or release is authorized by this
record.
