# QA-001 manual sequence runs

Recorded execution: [100,000 sequences passed on 2026-11-16](QA001-2026-11-16.md), with exact revision, seed ranges, commands, durations and log hashes. This does not confirm a release threshold.

The harness runs at most 50 generated operations per seed and checks ledger invariants after every operation. Each seed gets a fresh database, clock and deterministic ID generator, so replay does not depend on earlier seeds or worker count. The ordinary test budget remains **250 sequences**, or **50 with the race detector**. Merge/release thresholds remain **UNCONFIRMED** in [THRESHOLDS.md](THRESHOLDS.md); this manual workflow does not approve or enforce a release gate.

Run a small batch locally first:

```sh
python3 scripts/run_qa001.py --sequences 50 --start-seed 100000 --shards 2 --timeout-seconds 300
```

The runner prints a new directory under `bin/`. It contains `metadata.json` (checkout revision, dirty state, Go version, exact command, seed range, worker count, elapsed time and exit status), `go-test.jsonl` (actual Go test output), and `failures/qa001-seed-N.json` for captured failures. Fixtures include the failing operation, prior steps, failure text, clock and runtime metadata. Helper panics and `t.Fatal` failures are captured too. Timeout or process termination may prevent a fixture; the test log identifies the last started seed in each worker. Do not interpret an interrupted run as a pass.

An explicit long run, when the available runtime is sufficient:

```sh
python3 scripts/run_qa001.py --sequences 100000 --start-seed 100000 --shards 4 --timeout-seconds 18000
```

This command is an opt-in workload; only a completed passing result counts as execution evidence, and running it does not approve the owner's release threshold. Benchmark the intended runner using a small batch before selecting its runtime budget. For separate batches, use adjacent non-overlapping ranges: for example, `--start-seed 100000 --sequences 10000`, then `--start-seed 110000 --sequences 10000`. The last included seed is `start + count - 1`.

Replay a failure at the recorded checkout revision. Prefer a clean checkout for shared evidence; if metadata records `dirty: true`, replay also requires those local source changes:

```sh
python3 scripts/run_qa001.py --seed 100017 --shards 1 --timeout-seconds 300
```

`--seed` excludes range options. `--shards` accepts 1–8 workers and is capped at the number of seeds. `--race` enables instrumentation; `--go /path/to/go` selects a toolchain. `--output /new/directory` changes the artifact location; an existing directory is refused so results cannot silently mix. Timeouts are 1–19,800 seconds, with a further 60-second hard process deadline. The runner clears inherited QA-001 range/replay variables so its explicit arguments control the run.

In GitHub Actions, dispatch **QA-001 manual invariant sequences** on the intended revision. Set a range or a single replay seed, workers, timeout and optional race instrumentation. The job runs only on manual dispatch and always attempts to upload the evidence directory. No PR, nightly, merge or release workflow automatically launches this long run.

Direct harness controls remain available:

```sh
CIRCULA_QA001_START_SEED=100000 CIRCULA_QA001_SEQUENCES=50 CIRCULA_QA001_SHARDS=2 \
  CIRCULA_QA001_FAILDIR=/tmp/qa001-failures \
  go test -count=1 -timeout=5m -run '^TestQA001InvariantSequences$' ./internal/service
```

`CIRCULA_QA001_SEED` replays one seed and cannot be combined with `START_SEED` or `SEQUENCES`. Counts must be 1–1,000,000 and ranges must fit a nonnegative signed 64-bit integer. Invalid settings fail rather than falling back to a shorter run. The runner is preferred because it records revision and toolchain metadata and preserves the test log.
