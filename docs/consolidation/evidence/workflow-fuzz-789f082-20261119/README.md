# Workflow fuzz gates: preserved partial attempt

**Completion update:** the remaining four gates passed at their original
budgets in [attempt 2](attempt-2/README.md). The sections below retain the
first attempt, including its separate API compiler failure.

Tested source: `789f082a337eac18f9628976846536e6230f5975`.
Tree: `8e2a6cb1fd3b83c151ec7cdc5dddedd06d5d54cf`.
The isolated checkout matched that exact commit and was clean before and after
each attempted command. No production/test source or generated corpus was
committed by this qualification attempt.

## Actual outcomes

| Workflow gate | Requested budget | Observed outcome |
|---|---:|---|
| `FuzzParseAmount`, `internal/domain` | 1,000,000 executions | **PASS**, exactly 1,000,000 executions; 57.093s package time, 118.045s total including build. |
| `FuzzStrictUnmarshal`, `internal/api` | 100,000 executions | **BUILD FAILED**, exit 1 after 149.625s. The `modernc.org/libc` compiler reported `signal: killed`; fuzzing did not start. |
| `FuzzOpenCursor`, `internal/api` | 100,000 executions | Not started; held after the build failure. |
| `FuzzSessionPosition`, `internal/service` | 100,000 executions | Not started; held after the build failure. |
| `FuzzSessionSelector`, `internal/service` | 100,000 executions | Not started; held after the build failure. |

The API build result is not a fuzz assertion failure or a target pass. No
reduced execution budget, substituted target or weakened timeout was used.
The parent paused further Go work after this attempt. An executor restart
subsequently emptied the temporary caches; the persistent logs were reread and
their recorded SHA-256 values matched before this receipt was committed.

## Commands and environment

Toolchain: **Go 1.26.11, linux/amd64**. Commands ran sequentially with one build
package and one fuzz worker. The shared cache and private temporary directory
below describe the completed attempt, not the replacement environment after
the restart:

```text
GOCACHE=/dev/shm/circula-runtime-cache-93
GOMODCACHE=/tmp/circula-go-modcache-93
TMPDIR=/dev/shm/circula-fuzz-js-93-yf17muos
GOMAXPROCS=1
GOMEMLIMIT=250MiB
GOGC=40
```

```sh
go test -p=1 -count=1 -run '^$' -fuzz '^FuzzParseAmount$' \
  -fuzztime 1000000x -parallel=1 -timeout 3m ./internal/domain
go test -p=1 -count=1 -run '^$' -fuzz '^FuzzStrictUnmarshal$' \
  -fuzztime 100000x -parallel=1 -timeout 3m ./internal/api
```

[attempt-1.json](attempt-1.json) preserves full executable paths and arguments,
UTC timestamps, resource observations, source blobs/SHA-256 values and log
checksums. The raw outputs are [FuzzParseAmount.log](FuzzParseAmount.log) and
[FuzzStrictUnmarshal.log](FuzzStrictUnmarshal.log). The amount-parser pass is
retained independently of the later build failure. The other four gates still
require successful completion at their unchanged workflow budgets.
