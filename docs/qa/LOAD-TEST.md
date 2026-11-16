# Load test

Date: 2026-10-21. Test: `TestLoadConcurrentTrades` in `internal/service/loadtest_test.go`. It is skipped unless
`CIRCULA_LOAD=1`, so `go test ./...` and CI do not run it.

```
CIRCULA_LOAD=1 CIRCULA_LOAD_WORKERS=16 CIRCULA_LOAD_MEMBERS=50 CIRCULA_LOAD_TRADES=250 \
  go test ./internal/service -run LoadConcurrentTrades -v -count=1
```

## What is measured

The service layer is driven directly (no HTTP, no TLS, no JSON), on a real SQLite file in WAL mode with the default
5 s busy timeout. `W` goroutines each run `T` trades between random pairs of `M` members. A trade is a propose by the
payer (amount 0.01 to 3.00, credit limit 100.00) followed by an accept by the payee. Each is one committed write
transaction, so one trade is two write operations. Latency is wall time of the service call as the goroutine sees it,
including time queued behind the single in-process writer gate. Errors that are `unavailable` (SQLITE_BUSY past the
timeout) would be retried with the same idempotency key, up to 20 times, and counted. Refusals for insufficient credit
are counted separately and are a business outcome, not a fault.

After each run the test checks: no unexpected error; no member balance below `-credit limit`; member balances sum to
exactly 0; `VerifyLedger` reports OK (ledger sums, audit hash chain); samples equal completed trades. All runs below
passed all checks.

## Results

All numbers are from one run each, on the hardware below.

| Workers | Members | Trades attempted | Completed | Insufficient credit | Elapsed | Trades/s | Write ops/s |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 50 | 300 | 300 | 0 | 0.83 s | 361.7 | 723.4 |
| 4 | 50 | 1200 | 1200 | 0 | 4.07 s | 294.7 | 589.5 |
| 16 | 50 | 4000 | 4000 | 0 | 15.75 s | 253.9 | 507.8 |
| 64 | 50 | 3840 | 3840 | 0 | 14.36 s | 267.4 | 534.7 |
| 64 | 3 | 3840 | 3830 | 10 | 13.21 s | 290.0 | 580.1 |
| 128 | 50 | 3840 | 3840 | 0 | 15.06 s | 254.9 | 509.9 |

Latency per call (propose and accept shown separately; trade is the sum for one trade):

| Workers / members | Op | p50 | p95 | p99 | max |
|---|---|---:|---:|---:|---:|
| 1 / 50 | propose | 0.84 ms | 1.68 ms | 10.5 ms | 12.0 ms |
| 1 / 50 | accept | 1.08 ms | 3.17 ms | 12.2 ms | 13.3 ms |
| 1 / 50 | trade | 1.98 ms | 11.3 ms | 13.5 ms | 14.0 ms |
| 4 / 50 | propose | 5.06 ms | 17.7 ms | 22.6 ms | 29.9 ms |
| 4 / 50 | accept | 5.12 ms | 18.2 ms | 24.4 ms | 31.5 ms |
| 4 / 50 | trade | 10.9 ms | 26.2 ms | 33.4 ms | 37.4 ms |
| 16 / 50 | propose | 30.5 ms | 47.3 ms | 54.8 ms | 66.5 ms |
| 16 / 50 | accept | 32.6 ms | 47.9 ms | 54.7 ms | 63.3 ms |
| 16 / 50 | trade | 62.7 ms | 81.2 ms | 88.9 ms | 114 ms |
| 64 / 50 | propose | 118 ms | 150 ms | 185 ms | 202 ms |
| 64 / 50 | accept | 117 ms | 149 ms | 181 ms | 210 ms |
| 64 / 50 | trade | 240 ms | 289 ms | 325 ms | 340 ms |
| 64 / 3 | propose | 109 ms | 140 ms | 158 ms | 186 ms |
| 64 / 3 | accept | 108 ms | 138 ms | 151 ms | 186 ms |
| 64 / 3 | trade | 219 ms | 265 ms | 307 ms | 322 ms |
| 128 / 50 | propose | 245 ms | 314 ms | 343 ms | 364 ms |
| 128 / 50 | accept | 247 ms | 320 ms | 345 ms | 364 ms |
| 128 / 50 | trade | 488 ms | 618 ms | 639 ms | 653 ms |

Errors and retries in every run: `unavailable` errors 0, retries 0, gave up 0, unexpected errors 0.

## Reading the numbers

* Throughput is roughly flat at 250 to 360 trades/s (about 500 to 720 committed writes/s) regardless of concurrency,
  and latency grows linearly with the number of workers (about 1.9 ms per queued writer per call at 64 and 128 workers). This is
  the behaviour of a single serialised writer (ARC-011): extra goroutines queue rather than add throughput. With one
  worker the per-call latency is about 1 ms at p50.
* Throughput with 1 worker (361.7 trades/s) was higher than with 4 or more (253.9 to 294.7). The cause was not
  investigated here; no claim is made about it.
* No `unavailable` errors occurred. Writers share one process, so the in-process gate serialises them before SQLite can
  return SQLITE_BUSY. The retry path of the test is therefore exercised only if a second process writes to the same
  file, which this test does not do.
* Contention on a small member set (3 members) did not reduce throughput; the 10 refusals are payers reaching their
  credit limit.

## Hardware and caveats

* Machine: 4 vCPU Intel Xeon @ 2.10 GHz (virtual machine), 16 GB RAM, ext4 on a virtual block device, Linux 6.18, Go
  1.26.10, pure-Go SQLite driver (modernc.org/sqlite), `CGO_ENABLED` default.
* One run per row; no warm-up, no repetition, no confidence intervals. Differences of a few percent between rows are
  not meaningful.
* The test uses a deliberately cheap password hash and a fixed clock (test helpers), which do not affect the write path
  being measured but mean login cost is not represented. HTTP, TLS, the web UI, the rate limiters, session
  authentication and the reverse proxy are not included; end-to-end numbers will be lower.
* The workload is writes only. Read latency and mixed read/write behaviour were not measured.
* The database file is small (a few thousand transactions). Behaviour with a large ledger was not measured.
* Argon2 parameters, backup runs and the periodic jobs were not running during the test.
