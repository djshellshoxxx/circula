# Database durability (SA2-20)

`store.Options.Synchronous` sets `PRAGMA synchronous` for every pooled connection (`NORMAL` default, `FULL`). The
operator setting is `db-synchronous` (`cmd/circula/config.go`, passed through `storeOptions`).

WAL mode with `NORMAL` syncs the WAL only at checkpoint, so an OS crash or power loss can drop the newest commits. A
posting and its audit row are one transaction, so they are lost together: the database stays consistent and verifies.
`FULL` syncs the WAL on every commit.

Benchmark: `go test ./internal/store -run NONE -bench PostingSynchronous -benchtime 1000x` (one propose, two entries and a
completion per transaction). Recorded run, 4 vCPU sandbox VM: `NORMAL` 0.70 to 0.86 ms, `FULL` 1.89 to 2.41 ms per
posting, about 2.7x. Above the 2x bar for changing the default, so `NORMAL` stays the default and `FULL` is an opt-in
setting, documented in `docs/ops/configuration.md`. Re-run on production-like storage before deciding for a deployment.

Tests: `TestSynchronousOption` (store) and `TestDBSynchronousSetting` (cmd).
