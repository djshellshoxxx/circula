# Independent source review: COMMIT acknowledgment loss

## Findings and status

**No blocking source defect identified.** The injection reaches the requested
successful-COMMIT boundary, the connection checks distinguish discard from
reuse, and the financial-state and replay assertions are meaningful for this
driver-boundary regression. This conclusion is a source review, not an
execution pass.

Reviewed commit: `08003c900e3afdf8c369c1ab8fd0c826422326b7`.
Base: `0492d798ac2a92c382ded051ddbcafbf0854f403`.
Review date: 2026-11-19. The change contains three test files and its evidence
note; production code is unchanged. No Go build, test, race or SQLite process
was started for this review. The integrating parent owns runtime qualification.

## Injection ordering and error identity

In `storageFaultConn.ExecContext`
([connector source](../../../internal/store/recovery_internal_test.go)), the
pre-COMMIT mode returns its one-shot statement error before invoking the real
driver. The acknowledgment-loss mode instead calls the wrapped SQLite
`driver.ExecerContext.ExecContext` first. Only `commit && err == nil` increments
the successful-commit counter and consumes `commitAcknowledgmentErr`. A failed
underlying COMMIT therefore cannot be mislabeled as acknowledgment loss.

The exported test helper arms an ordinary `errors.New` value, not
`driver.ErrBadConn`. The regression requires `errors.Is` identity, the production
`store: commit:` wrapper and a nonretryable result. It also requires exactly one
COMMIT attempt and the appropriate pre- or post-COMMIT counter delta. An error
before the callback completes, a swallowed error or an extra retry would fail
these checks. The fault is armed after fixture preparation and service creation;
the next COMMIT belongs to the real `AcceptTransaction` write unit.

## Cleanup and live connection assertions

The production `SQLite.run` / `rollbackOrDiscard` path
([transaction source](../../../internal/store/sqlite.go)) attempts ROLLBACK
after the reported COMMIT error and marks the connection bad when that cleanup
fails. The new after-COMMIT mode does not inject a rollback error: the rollback
is sent to real SQLite, where the transaction has already ended. Its observed
error and the wrapped driver's `Close` call are counted separately.

The test snapshots these counters immediately after the failed command. It
then reads state and starts a fresh, nonmutating `BEGIN IMMEDIATE` through the
same live pool, before any explicit reopen. Exactly one replacement connection
is required in the after-COMMIT branch; none is allowed in the before-COMMIT
branch. Initial and replacement connections receive the production default
pragmas, with WAL, foreign keys, trusted schema, secure deletion, synchronous
mode and writable state checked. The fixture's one-connection pool makes these
open/close deltas unambiguous. A reopen cannot conceal the cleanup failure under
test because these checks precede it.

## Financial outcome, partial effects and durable replay

The regression uses the real service, store and SQLite database
([test source](../../../internal/store/commit_ack_test.go)). A previously
committed trade establishes nonempty balances and ledger history before the
second proposal is accepted.

| Boundary | Required assertions |
|---|---|
| Error before real COMMIT | No successful COMMIT, one successful real rollback, no discarded connection, unchanged stable-table snapshot and unchanged posting projection. |
| Error after successful real COMMIT | One successful COMMIT, one failed real rollback, discarded/replaced connection, two new balanced entries, exact debit/credit, completed transaction, one ledger-sequence advance, and matching audit/status/notification/idempotency deltas. |

`readCommitFaultPosting` reads raw SQL independently of service return values.
It checks the exact transaction, both member balances, transaction and network
ledger sequences, entry sum/count and the idempotency record scoped by network,
actor, operation and key. The result reference must name the accepted proposal.
`VerifyLedger` additionally checks cached balances, posting shape, totals and
audit-chain integrity; `PRAGMA integrity_check` checks SQLite integrity.

The existing [snapshot helper](../../../internal/testutil/boundary.go) hashes
the stable application tables in one SQL statement. It includes ledger, audit,
notification, idempotency and other application rows, excluding its three
documented request-activity tables. Thus the before-COMMIT control checks more
than balances and cannot accept leaked financial, notification or idempotency
rows. The connection-health transaction must also leave that snapshot intact.

The database is closed and reopened before the first retry, and the entire
stable-table snapshot must match the state observed after the error. The
after-COMMIT branch then requires the same-key retry to return the original
completed transaction without changing any stable table. The before-COMMIT
branch must instead post exactly once. Both branches require unchanged state
on another replay and on a replay after a second reopen. This makes persisted
idempotency material to the result; a response-only cache or a later repair
cannot satisfy the reopen-before-retry comparison.

## Qualification boundaries

The successful branch checks an explicit financial projection plus complete
stable-table equality across reopen/replay; it is not an exhaustive expected
row-by-row diff of every older record across the new posting. It uses a serial
one-connection fault fixture, not a concurrent pool stress test.

These assertions qualify logical persistence through ordinary close/reopen
after a driver acknowledgment error. They do not simulate power loss, physical
I/O/fsync faults, ambiguous on-disk commit markers, migration acknowledgment
loss or external audit-sink delivery. They do not close the broader L9 gate.
The original [qualification note](commit-acknowledgment-loss.md) states those
boundaries and the required focused execution selection. Source whitespace
validation (`git diff --check` against the reviewed base) passed; compilation
and execution remain explicitly outside this review.
