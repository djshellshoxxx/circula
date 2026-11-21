# Argon2id benchmark (SEC-001, DEC-006, DEC-065)

**This is not the owner's minimum-hardware benchmark.** DEC-006 asks for the parameters to be chosen by a benchmark on the
minimum supported hardware. The owner has not named that hardware yet, so DEC-006 stays open and the release gate in
SEC-001 is not passed by this page. It records what the parameters in use cost on the build machine below, so the owner
has a reference point and a repeatable procedure.

## Parameters in use

`service.DefaultArgon2Params()` (`internal/service/password.go`), not overridden by `circula serve`:

| Parameter | Value |
|---|---|
| Algorithm | Argon2id, version 19 (`golang.org/x/crypto/argon2`) |
| Memory | 65536 KiB (64 MiB) |
| Time (passes) | 3 |
| Parallelism (lanes) | 2 |
| Key length | 32 bytes |
| Salt length | 16 bytes, random per password |

Each hash is stored as a PHC string (`$argon2id$v=19$m=65536,t=3,p=2$...`), so verification uses the parameters recorded
with the hash and the defaults can change later without a mass reset. `TestDefaultArgon2ParamsMatchBenchmarkDoc` fails when
the defaults change without this page.

## How to run

```sh
go test ./internal/service -run NONE -bench Argon2id -benchtime 20x -count 3 -benchmem
```

- `BenchmarkArgon2idHash`: one hash (registration, password change, reset).
- `BenchmarkArgon2idVerify`: one verification, the cost of every sign-in attempt. Unknown logins verify against a dummy
  hash of the same cost (SEC-005), so failed and successful attempts cost the same.
- `BenchmarkArgon2idVerifyParallel`: raw verifications on every CPU at once (`GOMAXPROCS` goroutines), using an isolated
  test admission scope with one ticket per worker. Its ns/op is wall time divided by completed verifications. It measures
  the cryptographic primitive, not application sign-in throughput or the production admission limit.

The application now admits at most two password-work operations per process, with no queued waiters. A password change
keeps its one admission across verification and replacement hashing. See [password-work admission](password-work-admission.md)
for the shared backup/startup coverage, cancellation behavior and historical-PHC compatibility. This is an operation-count
bound; hashes with historical parameters can require more memory than the current 64 MiB default.

## Measured on the build machine (2026-11-04)

Machine: KVM virtual machine, `lscpu` model name `Intel(R) Xeon(R) Processor @ 2.10GHz` (family 6, model 207), 4 vCPUs
(1 thread per core), 16 GiB RAM, Linux 6.18 x86_64; `go1.26.10 linux/amd64`, `GOMAXPROCS=4`. Shared cloud host, so the
spread between runs is large.

| Benchmark | Run 1 | Run 2 | Run 3 | Memory per op |
|---|---|---|---|---|
| `Argon2idHash-4` | 164 ms | 133 ms | 124 ms | 64.0 MiB |
| `Argon2idVerify-4` | 120 ms | 187 ms | 131 ms | 64.0 MiB |
| `Argon2idVerifyParallel-4` | 223 ms | 216 ms | 198 ms | 67.2 MiB |

Reading:

- One sign-in costs about 120 to 190 ms of CPU and 64 MiB of memory here, within the common 0.1 to 1 s guidance for
  interactive logins.
- Parallel verification in this historical primitive benchmark was slower per operation than serial (about 200 ms
  against 120 ms): each verification already used two lanes, so four at once oversubscribed the four vCPUs and competed
  for memory bandwidth. The four-worker benchmark completed roughly five verifications per second with about
  `GOMAXPROCS x 64 MiB` of simultaneous hashing memory (256 MiB here). These pre-admission measurements do not establish
  the current application's throughput or memory ceiling. The process admission bound complements the unchanged
  per-login throttle and sign-in request limits (SEC-006, `rate-signin-*` settings).
- On smaller hardware (for example a single-board computer with 1 to 2 GiB RAM) the memory term matters more than the
  time term. Choosing that hardware, and the parameters for it, is the owner's DEC-006 decision; record the result in
  docs/DECISIONS.md (DEC-065) once made.
