# Concurrent qualification — exact-tree evidence index

Reviewed 2026-11-18T09:14:28.264867+00:00. No tests were run to create this index.

**All 14 test-bearing Go packages have passing race results across unchanged-tree invocations. No single aggregate invocation passed.** Two helper packages have no test files. The privileged foreign-owned-parent regression remains skipped and is not represented as executed coverage.

- Qualification commit: `a8a677e4bd27eee64cc27ee71d693d2f5b8e1f92`.
- Tree: `e8b07dfc8a3875eb3b25dc92adb0de5a63fcbfbe`.
- At review, `circula-integration` had that exact HEAD/tree and an empty porcelain status. Git independently confirms the reviewed #79, #81 and compatible #82 heads are ancestors. The recorded Go toolchain is 1.26.11.
- This tree includes #79 `dd3e82e27ddb3d1cbf03460e6a5fc879f16691db`, #81 `09ef8a5aa25f11ee4d96f043e7d686e97598b7e3`, and compatible #82 `5c39abdb89faa505533e51d3878b3b119957ff32`. It does not claim qualification of subsequent #84, #85 or #86 work.

## Package results

The durations below are Go's **package** runtimes. They are not whole-invocation wall times. The qualifying result comes from the indicated source log; failed earlier results remain recorded separately.

| Package | Result | Package seconds | Qualifying source log |
|---|---|---:|---|
| `cmd/circula` | PASS | 181.148 | `aggregate-concurrent-auth-cli-workspace-cache-race.log` |
| `internal/api` | PASS | 188.648 | `aggregate-concurrent-auth-race.log` |
| `internal/contract` | PASS | 18.463 | `aggregate-concurrent-auth-race.log` |
| `internal/domain` | PASS | 1.418 | `aggregate-concurrent-auth-race.log` |
| `internal/histbal` | PASS | 1.009 | `aggregate-concurrent-auth-race.log` |
| `internal/httpx` | PASS | 1.011 | `aggregate-concurrent-auth-race.log` |
| `internal/logx` | PASS | 1.030 | `aggregate-concurrent-auth-race.log` |
| `internal/repository` | PASS | 1.092 | `aggregate-concurrent-auth-race.log` |
| `internal/service` | PASS | 433.795 | `aggregate-concurrent-auth-race.log` |
| `internal/store` | PASS | 74.275 | `aggregate-concurrent-auth-race.log` |
| `internal/web` | PASS | 146.700 | `aggregate-concurrent-auth-isolated-race.log` |
| `internal/web/messages` | PASS | 1.062 | `aggregate-concurrent-auth-race.log` |
| `scripts/sbom` | PASS | 1.081 | `aggregate-concurrent-auth-race.log` |
| `scripts/traceability` | PASS | 2.078 | `aggregate-concurrent-auth-race.log` |
| `internal/storetest` | No test files | — | `aggregate-concurrent-auth-race.log` |
| `internal/testutil` | No test files | — | `aggregate-concurrent-auth-race.log` |

## Invocation outcomes — failures preserved

1. **Primary aggregate: FAILED, exit 1.** `GOMAXPROCS=2 CGO_ENABLED=1 go test -race -count=1 -p 2 -timeout=25m ./...` produced twelve package passes and two no-test helpers. CLI failed with `signal: killed` at 2.704 s; web failed with the same diagnostic at 87.904 s. Source: `aggregate-concurrent-auth-race.log`. The logged `FAIL` is not converted to an aggregate pass.
2. **Isolated CLI/web retry: FAILED.** `GOMAXPROCS=2 CGO_ENABLED=1 go test -race -count=1 -p 1 -timeout=25m ./cmd/circula ./internal/web` passed web in 146.700 s, but CLI was killed at 61.597 s; the log ends `FAIL`. Its numeric process exit code was not retained in the supplied source record. Source: `aggregate-concurrent-auth-isolated-race.log`.
3. **Bounded CLI diagnostic: FAILED, exit 1.** Full unchanged CLI suite with `GOMAXPROCS=2 GOGC=20 GOMEMLIMIT=512MiB CGO_ENABLED=1`, `-v -race -count=1 -p 1 -timeout=25m`. It was killed during `TestServeStartsWithFailingNetwork`; package runtime 73.639 s, wrapper wall time 74.893 s, peak child RSS 676264 KiB. Sources: `aggregate-concurrent-auth-cli-bounded-race.log` and its companion JSON.
4. **Workspace-cache CLI retry: PASSED, exit 0.** Full unchanged CLI suite with `GOMAXPROCS=2 GOGC=50 GOMEMLIMIT=512MiB CGO_ENABLED=1`, `-v -race -count=1 -p 1 -timeout=25m`; `GOTOOLCHAIN=local` and the preserved workspace `GOCACHE`. Package runtime 181.148 s; wrapper wall time 183.030 s; peak child RSS 643488 KiB. The receipt spans 09:00:49.224801–09:03:52.255174 UTC. Sources: `aggregate-concurrent-auth-cli-workspace-cache-race.log` and its companion JSON. No tests or race instrumentation were removed.

An earlier candidate `a04ff2323f84eeb4a46e3c462ad312dd6596a72a`, tree `1e2923efdff1adcf60c2718fe2a6a56edb3c1ed9`, also failed its aggregate invocation: CLI was killed at 2.900 s and service at 4.915 s; twelve other packages passed. Its `aggregate-concurrent-race.log` evidence is preserved but **none of those results is needed or counted for the current tree**.

## Cache diagnosis and causal limits

A separate PR84-focused command failed before tests with a concrete `no space left on device` error while writing `/home/agent/.cache/go-build`. The saved disk report shows the home tmpfs at 100%, with zero available space. The cache had been preserved in the workspace; after only the old generated home cache was cleared, the home tmpfs had 3.3G available and the workspace cache remained 3.3G. The successful final CLI receipt explicitly selects that workspace cache.

This evidence establishes that **the separate cache setup failure was ENOSPC**. It does not establish why earlier CLI, web or service processes reported `signal: killed`. Their causes remain unknown; no OOM diagnosis or retrospective ENOSPC attribution is asserted. See the four cache evidence records in the source manifest below.

## Skips and PR81 regression coverage

`TestBackupStagingRefusesForeignOwnedParent` explicitly skipped with “needs root on Unix to create a directory owned by another user.” The review environment is UID 1000; privileged ownership rejection is therefore an unexecuted qualification limit. Both this exact-tree CLI log and the provider's terminal record show the skip.

The provider record `audit-f/final-ci/pr81-new-regression-terminal-results.json` identifies all nine new PR81 tests: **eight pass and one skips**, with no missing terminal result. That provider execution is separately pinned to reviewed PR head `09ef8a5` and checkout `f2dd3bd`. The six other new CLI regressions also have direct PASS entries in this exact-tree final CLI log; the two new store regressions have provider PASS entries, and the exact-tree complete store package passes in the primary invocation. Provider evidence is not relabeled as an execution at this local aggregate commit.

`TestBackupWorkLeaseChild` and `TestRestoreSwapCrashChild` also skip in the direct top-level CLI enumeration because they are subprocess helper entry points. They are distinct from the privileged ownership skip. All three direct skips are preserved in the machine-readable index.

## Provenance and practical scope

The bounded and successful CLI metadata contain explicit commit/tree, command, environment, timing and exit-code receipts. The primary and isolated plain logs do not embed commit IDs; their exact-tree association is taken from the coordinator's retained `concurrent_qualification` record and checked against the current Git tree and pinned CLI receipts. The JSON index preserves that coordinator record, its source-file hash at review, all package terminal lines and the qualified source-head ancestry. The coordinator record still described the final CLI attempt as running when captured; the completed, separately pinned CLI receipt supplies the final result used here. A currently clean worktree is an observation, not a reconstructed per-invocation pre/post snapshot.

This index establishes package-complete local race evidence across several invocations on the unchanged recorded tree. It does not establish one successful aggregate command, privileged ownership coverage, native-platform CI completion, browser qualification, or readiness of later concurrent changes. Existing CI and release-limit records retain their separate roles.

## Source hashes

Paths are relative to the review workspace and identify the source artifacts copied into the durable evidence package by the coordinator. The companion `CONCURRENT-QUALIFICATION.json` records the same hashes, byte counts, exact commands and structured results.

| Source record | Bytes | SHA-256 |
|---|---:|---|
| `aggregate-concurrent-auth-race.log` | 1120 | `c92230a091cb5b38247e5cfa5f70c0dc058bdd15499b0c78878c5885709acc99` |
| `aggregate-concurrent-auth-isolated-race.log` | 138 | `b293da2184594fc7efee21add034e62812c5f5f4b77f07e86fda1d9f18403032` |
| `aggregate-concurrent-auth-cli-bounded-race.log` | 16022 | `86f963de9ad1d89e8922d5c887ff00797359ac3397f119006e2d8633c82245e5` |
| `aggregate-concurrent-auth-cli-bounded-race.json` | 720 | `530ca0197f11e3d26a8d97966f01a8ddb26e4f7eefc2a08757b72ffa4927a577` |
| `aggregate-concurrent-auth-cli-workspace-cache-race.log` | 31770 | `b1ea5b242bd430de3132aeed605f3d43f502144999dcc43e705c6344b198cb5b` |
| `aggregate-concurrent-auth-cli-workspace-cache-race.json` | 860 | `73239b06e361294ba9834a225905985cfe44c58a7344de38b3d632524e590bc9` |
| `aggregate-concurrent-race.log` | 1042 | `e44a2970e1786c3167ed361c3178df8e312b2dd18593025e38dd1688b03cd24c` |
| `audit-f/final-ci/pr81-new-regression-terminal-results.json` | 2575 | `ccb410769b8b101c98e1c4980a41542611560165e809bb591e31804c612c5fde` |
| `audit-f/final-ci/pr81-09ef8a5-linux-race-summary.json` | 3937 | `4b62fbb090eb72eccca5247441e00c4c038c998a6eee496bdf89771333888a38` |
| `audit-c/PR81-09ef8a5-COMPATIBILITY-REVIEW.md` | 4223 | `c024a96b7da138b0f6b00383ef43d696d7466a68a204c19e93ba01257bb179d7` |
| `audit-c/PR84-retention-cache-setup-failure.log` | 184 | `e90347fe730180f1a9448efe9acf88757915de6438e2fc3c0b217986fe35ad69` |
| `audit-c/PR84-disk-before-cache-clean.txt` | 336 | `356459a1243746df1ffbbff70b3d857dcf4c72dc1fec798d5f5f6e3c0f2072b3` |
| `audit-c/PR84-disk-after-cache-clean.txt` | 336 | `22abe7f04f3a849023c10c998a9a184586f64fa4499d5b5fe4c57c5a66342486` |
| `audit-c/PR84-old-cache-clean.log` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `audit-c/PR84-regression-snapshots.json` | 2844 | `fadc1a2aa738e63474d5064aea0fb32402fd9579831dfa540b267d54e13e8dc7` |
