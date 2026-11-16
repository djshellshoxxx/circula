# SQLite storage-failure qualification

L9 has real SQLite engine-capacity and Linux WAL-write failure regressions in addition to store-interface faults and process crashes. **Read, fsync, checkpoint and general device-failure qualification remain open.** `SQLITE_FULL` and `SQLITE_IOERR_WRITE` exercise different boundaries; neither regression exhausts the host filesystem.

## Covered failure

`internal/service/sqlite_full_test.go:TestSQLiteFullDuringPostingRollsBackAndRecovers` exercises acceptance, reversal and administrator adjustment against the production SQLite store and WAL mode.

The fixture pins the connection pool to one connection and sets `PRAGMA max_page_count` to the current size plus 64 pages. A test-only `AFTER INSERT` trigger on the final idempotency row allocates a blob larger than that limit in an empty fixture table. SQLite itself refuses the allocation with numeric result code `SQLITE_FULL` (13). No wrapper fabricates or substitutes an error. The fixture adds no production migration or schema change.

A read-only observation at `PutIdem` confirms that the transaction already has its two balanced entries, updated account balance cache, completed status and an appended audit event before the allocation failure. The test then checks:

- Two consecutive failed attempts leave a raw SQL checksum of every nonvolatile table unchanged, including transactions, status events, entries, cached balances, ledger sequence, audit chain/head, notifications and idempotency records.
- No post-commit audit sink event or successful-posting/replay metric escapes the rollback.
- Ledger and audit-chain verification succeeds after each failure.
- Removing the page limit and test trigger permits the same request and idempotency key to succeed on the same live connection. Replay returns the same transaction and makes no further database change.
- Exactly two entries and one ledger sequence are added. Closing and reopening the file preserves the recovered state, `PRAGMA integrity_check` reports `ok`, and ledger/audit verification still passes.

Run the focused check with:

```sh
go test ./internal/service -run '^TestSQLiteFullDuringPostingRollsBackAndRecovers$' -count=1
go test -race ./internal/service -run '^TestSQLiteFullDuringPostingRollsBackAndRecovers$' -count=1
```

Both focused commands passed on 2026-11-16. No full-suite result is claimed by this evidence note.

## Linux WAL commit write failure

`internal/service/sqlite_ioerr_linux_test.go:TestSQLiteWALWriteFailureRollsBackAndRecovers` uses the production driver and store to accept a trade against an existing ledger in a separate Linux subprocess. It reduces only that child's soft `RLIMIT_FSIZE`, leaving its hard limit and the parent process unchanged. Ignoring `SIGXFSZ` makes the actual write return `EFBIG` instead of terminating the child. The pinned driver's Unix write implementation maps that failure to extended result `SQLITE_IOERR_WRITE` (778).

Fixture setup, the WAL truncate checkpoint and all financial writes precede the limit. The fixture disables cache spilling and automatic checkpointing on its single connection. Immediately before returning from the service's write callback, it verifies the completed transaction, two balanced entries, changed balance caches, completion history, next audit event and idempotency record, while the WAL is still empty. It then sets a limit that permits a WAL header, one full frame and half the next frame. The production store executes `COMMIT`; no wrapper returns a fabricated storage error.

The regression requires the actual driver error to come from `store: commit`, restores the limit before returning to the service, checks unchanged live SQL state and no successful-posting/replay metric or audit-sink event, and verifies the ledger and audit chain. The child exits without closing the database, preserving the failed WAL for the parent. Before reopening, the parent verifies that the database bytes are unchanged and the WAL contains exactly one complete non-commit frame plus the incomplete next frame. This particular failure has a definite original-state outcome because no complete commit frame was written; the test does not generalize that outcome to errors after a commit marker or during fsync.

Reopening must reproduce the complete original SQL snapshot. Retrying with the same idempotency key posts once, replay adds nothing, and a second reopen must retain the complete committed snapshot with `integrity_check=ok` and passing ledger/audit verification. The fixture makes no production schema, dependency or code change.

Run on Linux:

```sh
go test ./internal/service -run '^TestSQLiteWALWriteFailureRollsBackAndRecovers$' -count=1
go test -race ./internal/service -run '^TestSQLiteWALWriteFailureRollsBackAndRecovers$' -count=1
```

Both focused commands passed locally on Linux on 2026-11-16 with Go 1.26.11: normal in 0.098s and race in 4.068s. Execution was serialized with the shared `flock`, `GOMAXPROCS=2`, `GOMEMLIMIT=600MiB`, `-p 1` and `-timeout=2m`. The full Linux race job also passed at commit `dc89212a8bf1e1207e5443ffc719670623de70e1` in [CI run 43914378342](https://github.com/djshellshoxxx/Circula/actions/runs/43914378342). These results qualify the specific WAL-write failure above, not L9 as a whole.

## Evidence boundaries

| Mechanism | Evidence | Boundary |
|---|---|---|
| Returned store-interface errors | `atomicity_test.go`, `ledger_beta_test.go` | Tests application rollback at individual posting operations; does not exercise SQLite failure handling. |
| Process termination at posting boundaries | `qa010_crash_test.go` | Covers QA-010 restart consistency, including before/after commit; does not emulate device I/O failure or power loss. |
| SQLite page-allocation exhaustion | `sqlite_full_test.go` | Exercises the real engine's `SQLITE_FULL` handling late in the posting unit and subsequent connection/file recovery. |
| Linux file-size-limit write failure during WAL commit | `sqlite_ioerr_linux_test.go` | Actual `EFBIG` produces `SQLITE_IOERR_WRITE` during a partial WAL-frame append. Covers this specific pre-commit-marker write failure and restart/retry recovery. |
| OS read/fsync/truncate and other write/device errors | Not covered by these tests | Still needs a writable VFS fault harness or an isolated filesystem/device fault environment. |
| Checkpoint I/O and uncertain commit outcomes | Not covered by these tests | No claim is made about checkpoint database writes, sync failures, errors after a complete commit frame or power-loss durability. |

## Driver source inspected

The implementation was checked against the locally installed primary source for the pinned `modernc.org/sqlite v1.60.1` dependency:

- `error.go` exposes the actual SQLite result through `Error.Code()` and distinguishes `SQLITE_FULL` from `SQLITE_IOERR` and its extended codes.
- `lib/sqlite.go`, `_sqlite3PagerMaxPageCount`, configures the pager limit. The generated pager acquisition code checks the requested page number against `mxPgno` and returns `SQLITE_FULL` when the limit is exceeded.
- `vfs/patches.go` documents the exported Go `fs.FS` adapter as **read-only**; it has no writable journal/WAL path suitable for this qualification.
- `lib/seh.go` and `seh_test.go` provide a process-global simulated WAL shared-memory fault facility and a platform-dependent truncated `-shm` test. Neither is a writable VFS read/write/fsync fault matrix. This test uses neither and does not relabel simulated faults as real device I/O errors.
- Generated shared Linux source `_seekAndWriteFd` calls the real positioned write and records its errno. `_unixWrite` retries a short write, then maps a negative result other than `ENOSPC` to `SQLITE_IOERR_WRITE`; `_walWriteOneFrame` writes the 24-byte frame header followed by the page bytes. These are in `sqlite_g_0000000000003e80.go`, `sqlite_g_000000000001ffff.go` and `sqlite_g_000000000001feab.go` respectively for Linux amd64.
- The Linux [getrlimit(2) manual](https://man7.org/linux/man-pages/man2/getrlimit.2.html) documents `RLIMIT_FSIZE`, `SIGXFSZ` and `EFBIG`. The [SQLite WAL format](https://www.sqlite.org/fileformat2.html#walformat) defines the 32-byte WAL header, 24-byte frame header and commit marker used to bound and inspect the failed append.

The remaining L9 storage-I/O qualification must cover the other real writable VFS operations and commit/checkpoint boundaries, retain the database and WAL for restart checks, and verify either the original or fully committed state where commit outcome is uncertain. These regressions do not close L9 as a whole.
