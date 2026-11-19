# Amount fuzz tool receipt

Observed completed execution on 2026-11-19 at local source 5da8bc7, Go 1.26.11.
The original temporary log was lost in the later executor restart. This record
reproduces the completed tool output; it is not represented as the lost file.

Command: `go test -p 1 -run '^$' -fuzz '^FuzzParseAmount$' -parallel 1 -fuzztime 100000x -timeout 3m ./internal/domain`.

Environment: `GOMAXPROCS=1`, `GOMEMLIMIT=400MiB`.

```text
fuzz: elapsed: 0s, gathering baseline coverage: 0/21 completed
fuzz: elapsed: 0s, gathering baseline coverage: 21/21 completed, now fuzzing with 1 workers
fuzz: elapsed: 3s, execs: 65504 (21827/sec), new interesting: 30 (total: 51)
fuzz: elapsed: 4s, execs: 100000 (23507/sec), new interesting: 31 (total: 52)
PASS
ok  github.com/djshellshoxxx/circula/internal/domain 4.476s
```
