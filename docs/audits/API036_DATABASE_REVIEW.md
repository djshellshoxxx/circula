# API-036 database review and SQL measurements

Date: 2026-11-19. This report records database implementation evidence and an
independent review of the integrated pagination and qualification tests. It is
not a declaration that the full test suite or current-head CI has passed.

## Provenance and reproduction

[API036_SQL_MEASUREMENTS.json](API036_SQL_MEASUREMENTS.json) preserves the actual
probe output from local integration commit
`4cc0a1bcd2e5d204da9e06c6e09fc778e7cb55b5`, using Python SQLite **3.53.1**. The later
published candidate
[`6f8ead1951f2d4b72649909b2b4cc83000948afe`](https://github.com/djshellshoxxx/circula/commit/6f8ead1951f2d4b72649909b2b4cc83000948afe)
contains the same measured SQL. Measurements are attributed to the earlier
measured commit, not to the later candidate.

The probe requires Python 3 with SQLite support and Git, with no extra Python
packages. Run it from any working directory:

```sh
python3 /path/to/circula/docs/audits/api036-session-sql-probe.py
python3 /path/to/circula/docs/audits/api036-session-sql-probe.py /path/to/another/checkout
```

Use a clean checkout: the script reads working-tree SQL and records that
checkout's HEAD. It extracts the production SQL constants, applies all migration
SQL files to an in-memory database, uses synthetic identities, and prints JSON.
It does not open an application database. The committed measurement JSON is a
historical result; subsequent runs should be saved separately.

## Query bounds and indexes

The reviewed [repository queries](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/sqlite_person.go)
cap each inventory request at 100 retained rows. Ordering uses
`(created_at, public_id)` with a strict tuple boundary: descending for forward
pages and ascending when seeking the nearest previous page. The owner/page index
supports both directions without an unbounded sort. Direct confirmation and
revocation lookup use the unique public-selector index plus an owner predicate.

The [service](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/service/sessions_self.go)
inspects at most 20 batches of 100 retained rows per request, including expired
and revoked rows. Empty filtered pages carry progress cursors. If a linked newer
side disappears, the first empty reverse query refreshes from the newest page
within that same budget: at most 19 further queries and 1,900 fetched rows.
Forward exhaustion stays terminal and cannot wrap to the first page. The
disappearance, boundary-reachability, and exact-cap regressions are in
[session_page_internal_test.go](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/service/session_page_internal_test.go).

[Bulk revocation](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/sqlite_session_others.go)
returns an affected-row count without materializing owner inventory. Its work
scales with eligible unrevoked rows. The partial `sessions_owner_unrevoked` index
avoids scanning retained revoked history on repeated requests. Service-side
authentication revalidation, pending MFA challenge deletion, and minimal audit
insertion share the update transaction; SQL probe results do not substitute for
those service security tests.

These bounds concern session inventory and selective revocation. Existing
inactivity-maintenance aggregates are outside this claim.

## Measured SQL results

Each fixture contains the stated owner-history size, one retained current owner
session, and one foreign session. History has mixed expiry fields; 90% is already
revoked and 10% is unrevoked. This probes raw SQL, not service expiry decisions.

| Operation | 100 history rows | 10,000 history rows | 100,000 history rows |
| --- | ---: | ---: | ---: |
| First 25 rows, VM instructions | 316 | 316 | 316 |
| Deep forward 25 rows, VM instructions | 449 | 449 | 449 |
| Previous 25 rows, VM instructions | 443 | 443 | 443 |
| Direct lookup, VM instructions | 22 | 22 | 22 |
| Repeated bulk, zero changes, VM instructions | 21 | 21 | 21 |
| Foreign current selector, zero changes, VM instructions | 36 | 36 | 36 |
| Initial bulk rows changed | 10 | 1,000 | 10,000 |
| Initial bulk VM instructions | 391 | 34,051 | 340,051 |

All query-plan assertions passed: indexed searches, no full scan, and no temporary
sorting tree. The current and foreign sessions remained unrevoked; integrity and
foreign-key checks passed. The JSON also preserves measured elapsed times.

Constant observed VM counts do **not** establish constant wall time. B-tree seeks
have size-dependent work, and cache state, driver, runtime, and storage affect
latency. The per-instruction Python progress callback itself adds overhead.
Results may change with SQLite versions. This probe does not measure production
Go allocations or bound the cost of traversing an entire history.

## Migration and production-test review

[Migration 0020](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/migrations/0020_session_public_id.sql)
adds independent public selectors, uniqueness and identity protections, and the
owner/keyset index. Existing bearer hashes and lifetime fields are preserved.
[Migration 0021](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/migrations/0021_session_unrevoked_index.sql)
adds the partial bulk index without rewriting 0020. The
[migration runner](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/migrate.go)
uses transactional forward migrations and rejects a newer database schema.

The independent source review found no unresolved blocking issue in:

- [Migration rollback regression](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/session_migration_rollback_test.go): an induced index conflict after 0020 expansion/backfill must preserve version 19, credentials, timestamps, and revocation state; removing the conflict permits retry.
- [Production SQLite traversal and benchmark](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/store/session_performance_test.go): 10,000 retained rows, tied timestamps, deleted cursor boundary, owner isolation, and fixed 25-row first/deep pages at 100/10,000/100,000 rows with untimed fixtures.
- [Production service stress regression](https://github.com/djshellshoxxx/circula/blob/6f8ead1951f2d4b72649909b2b4cc83000948afe/internal/service/sessions_stress_test.go): actual SQLite query/fetched-row counters, traversal through 10,000 dead rows, and allocation comparison between 10,001 and 50,001 retained sessions.

These source-review statements are not pass claims for unexecuted tests. Earlier
resource-interrupted service runs are not counted as passes. Full Go, race,
browser, fuzz, and current-head cross-platform CI qualification are recorded by
the integration run separately. No Go compilation was performed to prepare this
evidence addition.
