# COMMIT acknowledgment-loss regression

Source base: `0492d798ac2a92c382ded051ddbcafbf0854f403`.
Prepared on 2026-11-19. This change contains test code and this note only;
production transactions, service behavior and SQLite configuration are unchanged.

## Boundary and fixture

`TestCommitAcknowledgmentFailurePreservesPostingAndReplay` uses the existing
`testutil.NewFixture`, production service/store, WAL mode, and `testutil.Snapshot`
raw-SQL checksums. The fixture starts with an already committed trade and a second
proposal. It checks two distinct outcomes of accepting the second proposal:

| Injected driver outcome | Required state |
| --- | --- |
| Ordinary error before forwarding `COMMIT` | The original error reaches the caller, real rollback succeeds, and the entire nonvolatile database snapshot remains unchanged. |
| Real `COMMIT` succeeds, then an ordinary acknowledgment error is returned | The original error reaches the caller, real rollback fails because no transaction remains, the connection is discarded, and the complete committed posting is retained. |

The new one-shot mode extends `storageFaultConnector` in
`internal/store/recovery_internal_test.go`. It reports the acknowledgment error
only after the underlying driver has returned success for the real `COMMIT`.
The injected error is not `driver.ErrBadConn` and must not become a retryable
`unavailable` response. Counters independently record attempted/successful
commits, both fault modes, actual rollback attempts/errors, and connection
opens/closes.

`internal/store/commit_ack_export_test.go` exposes the connector to the external
test package only. This permits reuse of the existing service fixture and SQL
snapshot utility without an import cycle; no hook enters a production binary.
Its connection pragmas match production defaults and are checked on both the
initial connection and its replacement.

## Assertions

- Preserve the original nonretryable error and demonstrate exactly which COMMIT
  boundary was reached.
- Check discarded-versus-retained connections and a healthy, non-mutating
  `BEGIN IMMEDIATE` on the live pool before reopening could hide cleanup errors.
- Check balanced entries, exact account balances, transaction status, network and
  transaction ledger sequences, audit/status/notification counts, and the scoped
  idempotency record/result reference.
- Verify the ledger, audit hash chain and SQLite integrity.
- Reopen before retry and compare every nonvolatile table with the observed
  post-error state. The reused snapshot helper excludes only its documented
  request-activity tables: sessions, API tokens and person activity.
- Retry with the same key, then replay again and after a second reopen. The
  already committed case must not change the snapshot; the pre-COMMIT case must
  post exactly once. Both retain the earlier committed trade.

## Preparation checkpoint

`gofmt` completed and `git diff --check` passed. **The new regression has not been
compiled or run at this checkpoint.** Root is serializing the SQLite race build
and aggregate qualification; no execution pass or production defect is claimed.

Once that compile hold is lifted, the focused selection should include the
existing tests that share the modified connector:

```sh
GOMAXPROCS=1 GOMEMLIMIT=250MiB GOGC=40 \
  /workspace/scratch/beb70d57deba/toolchain/go/bin/go test -p=1 -race \
  -count=1 -timeout=2m \
  -run '^(TestCommitAcknowledgmentFailurePreservesPostingAndReplay|TestFailedRollbackDiscardsConnection|TestMigrationMetadataReadFailureStopsBeforeDDL|TestMigrationWriteFailureRollsBackAndRecovers)$' \
  ./internal/store
```

This needs the ordinary store test binary, including the SQLite dependency and
the external `store_test` package. A file-list invocation omitting the test-only
export helper is insufficient. Default vet remains enabled.

## Focused race qualification on 2026-11-19

The immutable test source `08003c900e3afdf8c369c1ab8fd0c826422326b7` was
subsequently exported with `git archive` to a standalone directory and tested
without modifying that source. The focused selection above, with `-v`,
**passed with race detection and default vet enabled**: four top-level tests and
eight subtests, including both the before-COMMIT control and the successful-COMMIT
acknowledgment-loss case. The existing rollback-discard and migration fault
regressions that share the connector also passed.

The command exited zero. The store package reported 14.008 seconds; total command
wall time, including preparation by the Go tool, was 215.825 seconds. It ran from
13:19:02Z to 13:22:38Z with `GOMAXPROCS=1`, `GOMEMLIMIT=250MiB`, `GOGC=40`, and
`GOMODCACHE=/tmp/circula-go-modcache-93`. `GOCACHE` was unset and the effective
cache was `/home/agent/.cache/go-build`; this run did not use the other worker's
separate cache under `/tmp`.

The [machine receipt](commit-acknowledgment-loss-08003c9-result.json) records the
exact source, argument vector, environment, times and exit status. The
[complete focused test output](commit-acknowledgment-loss-08003c9-test.log) records
the passing cases. This is a focused store qualification, not a complete-suite
result or a claim that the entire storage fault backlog is closed.

## Qualification limit

This is application/driver-boundary fault injection. It does not emulate an OS
read/write/fsync/truncate failure, an uncertain physical commit marker, a device
fault or power loss, and does not close L9 as a whole. It qualifies database audit
state, not external audit-sink delivery. Migration acknowledgment loss and the
existing physical fault backlog remain separate from this posting regression.
