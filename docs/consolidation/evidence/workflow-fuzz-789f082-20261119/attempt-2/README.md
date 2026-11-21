# Completed workflow fuzz budgets: preserved retry

All five requested iteration gates passed on source
`789f082a337eac18f9628976846536e6230f5975`, tree
`8e2a6cb1fd3b83c151ec7cdc5dddedd06d5d54cf`. The amount-parser pass remains
the original successful execution; it was not repeated. The first API build
failure is retained independently in [attempt 1](../README.md).

| Gate | Actual executions | Result | Go package time | Command wall time |
| --- | ---: | --- | ---: | ---: |
| `FuzzParseAmount` | 1,000,000 | PASS, attempt 1 | 57.093 s | 118.045 s |
| `FuzzStrictUnmarshal` | 100,000 | PASS, attempt 2 | 50.502 s | 200.754 s |
| `FuzzOpenCursor` | 100,000 | PASS, attempt 2 | 48.249 s | 49.777 s |
| `FuzzSessionPosition` | 100,000 | PASS, attempt 2 | 38.179 s | 59.778 s |
| `FuzzSessionSelector` | 100,000 | PASS, attempt 2 | 36.285 s | 38.599 s |

All four retry commands exited zero, printed `PASS`, and reported exactly
100,000 executions. Their checkout matched the source above and was clean
before and after each command. No failing corpus, production/test edit,
reduced budget, skipped assertion, or changed timeout was introduced.

## Retry conditions and commands

The first attempt's `modernc.org/libc` compiler process was killed before API
fuzzing started. After the executor restart, the parent restored one shared
cache and warmed the production build. The retry waited for the coordinated
lint repair to release its build slot and then ran these four gates
sequentially with **Go 1.26.11, linux/amd64**:

```text
GOCACHE=/tmp/circula-go-cache-93
GOMODCACHE=/tmp/circula-go-modcache-93
TMPDIR=<a private per-gate directory under /dev/shm>
GOMAXPROCS=2
GOMEMLIMIT=750MiB
GOGC=100
```

```sh
go test -p=1 -count=1 -run '^$' -fuzz '^FuzzStrictUnmarshal$' -fuzztime 100000x -parallel=1 -timeout 3m ./internal/api
go test -p=1 -count=1 -run '^$' -fuzz '^FuzzOpenCursor$' -fuzztime 100000x -parallel=1 -timeout 3m ./internal/api
go test -p=1 -count=1 -run '^$' -fuzz '^FuzzSessionPosition$' -fuzztime 100000x -parallel=1 -timeout 3m ./internal/service
go test -p=1 -count=1 -run '^$' -fuzz '^FuzzSessionSelector$' -fuzztime 100000x -parallel=1 -timeout 3m ./internal/service
```

The strict-JSON command's 200.754-second wall duration includes instrumented
compilation. Its test-package runtime was 50.502 seconds; the unchanged Go
test timeout remained three minutes. The retry began at 14:10:21 UTC and its
last command finished at 14:16:31 UTC on 2026-11-19. No other agent's cache was
deleted, and the build slot was released after the last command exited.

## Raw evidence and source boundaries

Each JSON receipt records the complete executable path and argument list,
actual environment settings and private temporary directory, UTC timestamps,
resource observations, exact source/package/workflow identities, before/after
clean-source checks, exit code, reported execution count, and raw-log SHA-256.
The four logs were reread, hash-checked, copied without alteration, and
byte-compared with their persistent originals before this receipt was committed.

| Gate | Receipt | Raw output |
| --- | --- | --- |
| Strict JSON | [FuzzStrictUnmarshal.json](FuzzStrictUnmarshal.json) | [FuzzStrictUnmarshal.log](FuzzStrictUnmarshal.log) |
| Signed cursor | [FuzzOpenCursor.json](FuzzOpenCursor.json) | [FuzzOpenCursor.log](FuzzOpenCursor.log) |
| Session position | [FuzzSessionPosition.json](FuzzSessionPosition.json) | [FuzzSessionPosition.log](FuzzSessionPosition.log) |
| Session selector | [FuzzSessionSelector.json](FuzzSessionSelector.json) | [FuzzSessionSelector.log](FuzzSessionSelector.log) |

The workflow blob defining these iteration budgets is
`0dd0a10dd3ec0243e816f0935da73686adc8fa84`. The retry's API package tree is
`e470b1346fce57777526ffeb79b0c2356f761179`, and its service package tree is
`a18b0b2cbf6e50b07df4ec92fa08a916eadf46de`.

These results qualify the five requested iteration gates on the recorded
source. They do not claim a complete provider workflow run, a new full-suite
or race pass, native cross-platform execution, or browser qualification.
