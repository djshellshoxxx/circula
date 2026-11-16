# SQLite storage-failure qualification

L9 now has a real SQLite engine-capacity failure test in addition to store-interface faults and process crashes. **OS-level write, read and fsync failures remain unqualified.** `SQLITE_FULL` is not `SQLITE_IOERR`, and the test does not exhaust the host filesystem.

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

## Evidence boundaries

| Mechanism | Evidence | Boundary |
|---|---|---|
| Returned store-interface errors | `atomicity_test.go`, `ledger_beta_test.go` | Tests application rollback at individual posting operations; does not exercise SQLite failure handling. |
| Process termination at posting boundaries | `qa010_crash_test.go` | Covers QA-010 restart consistency, including before/after commit; does not emulate device I/O failure or power loss. |
| SQLite page-allocation exhaustion | `sqlite_full_test.go` | Exercises the real engine's `SQLITE_FULL` handling late in the posting unit and subsequent connection/file recovery. |
| OS read/write/fsync/truncate errors | Not covered by these tests | Still needs a writable VFS fault harness or an isolated filesystem/device fault environment. |
| Failure during WAL commit or checkpoint I/O | Not covered by the page-cap test | The allocation fails while executing the final INSERT, before `COMMIT`; no claim is made about failed WAL writes, syncs, checkpoint writes or power-loss durability. |

## Driver source inspected

The implementation was checked against the locally installed primary source for the pinned `modernc.org/sqlite v1.60.1` dependency:

- `error.go` exposes the actual SQLite result through `Error.Code()` and distinguishes `SQLITE_FULL` from `SQLITE_IOERR` and its extended codes.
- `lib/sqlite.go`, `_sqlite3PagerMaxPageCount`, configures the pager limit. The generated pager acquisition code checks the requested page number against `mxPgno` and returns `SQLITE_FULL` when the limit is exceeded.
- `vfs/patches.go` documents the exported Go `fs.FS` adapter as **read-only**; it has no writable journal/WAL path suitable for this qualification.
- `lib/seh.go` and `seh_test.go` provide a process-global simulated WAL shared-memory fault facility and a platform-dependent truncated `-shm` test. Neither is a writable VFS read/write/fsync fault matrix. This test uses neither and does not relabel simulated faults as real device I/O errors.

The remaining L9 storage-I/O qualification must inject errors into real writable VFS operations at the ledger and commit/checkpoint boundaries, retain the database and WAL for restart checks, and verify either the original or fully committed state where commit outcome is uncertain. The current evidence does not close that boundary.
