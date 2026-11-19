# Circula continuation evidence — 2026-11-19

This bundle preserves actual local execution logs and independently recounts their Go JSON events. It does **not** qualify the final consolidation head, replace provider CI, or enable any future module. No Go command, build, fuzz target, or module test was rerun while assembling it.

## Recorded outcomes

- **9 of 15 future modules completed their race command successfully.** Two commands timed out with exit `124`; four have no race attempt receipt.
- **11 format checks and 11 vet checks completed successfully.** FUX-103 started formatting, but no completion/exit receipt was recorded. Its empty log is not a pass.
- Completed future race runs contain **42 named pass events, 0 failures, and 0 skips**. FNET-202 separately emitted **72 named passes and 1 skip before timeout**. Those partial events do not qualify that module.
- All 15 captured future manifests remain `HOLD` and disabled. No production integration is qualified.

## Core execution scopes

| Run | Named pass / fail / skip events | Scope |
|---|---:|---|
| [core-session-bounded-service.jsonl](core/core-session-bounded-service.jsonl) | 3 / 0 / 0 | All production internal/service Go source files plus session_page_internal_test.go; no SQLite driver or external integration tests. |
| [core-job-identifiers.jsonl](core/core-job-identifiers.jsonl) | 1 / 0 / 0 | Isolated UUIDv7 job-identifier helper regression; does not build the entire API or execute API routes. |
| [core-job-identifiers-full-api-source.jsonl](core/core-job-identifiers-full-api-source.jsonl) | 1 / 0 / 0 | All production internal/api Go source files plus the UUIDv7 helper regression; no SQLite-backed handler/integration tests. |
| [security-token-authority-after.jsonl](core/security-token-authority-after.jsonl) | 19 / 0 / 0 | All production internal/service Go source files plus token_authority_internal_test.go; driver-neutral mutation probes. Does not execute the SQLite token-reset integration tests. |

The token authorization count is **one parent test plus 18 subcases**, with a successful package result and no race warning. Its source is `fe8324fca1e106493eb3e583d7c35ac732db2824`. The two job-identifier logs execute the same helper test under different source scopes; do not sum them as independent feature coverage. Other intermediate checkout hashes were not recorded in the logs and are left unknown in the manifest.

[core-lightweight-checks.json](core/core-lightweight-checks.json) retains successful exit receipts for diff, formatting, workflow pins, hygiene, export-reader tests, QA-runner tests, and threshold tests. Their exact source hash and invocation arguments were not preserved. The Python unittest outputs report 4 hygiene, 10 export, 4 QA-runner, and 8 threshold tests.

[core-inventory-sqlite-smoke.json](core/core-inventory-sqlite-smoke.json) records Python SQLite 3.53.1 SQL checks: migrations through 0020, 20,002 session rows, 40,000 token rows, indexed bounded query plans, integrity `ok`, and zero foreign-key violations. This is SQL-level evidence, not a test of the Go/modernc driver.

The coordinator observed an exit-zero build of production packages at intermediate `655c164375a626ec41673bb88310dc45a9a68a66`, before later UUID/security changes. That build excluded modernc/CLI. Its empty log cannot independently attest success, so the manifest labels it a coordinator observation. Empty SQLite-compilation and amount-fuzz logs have no successful completion receipt and are **not passes**.

The coordinator also reported successful traceability generation/checking at `c1190ceb99630be4a09721cc800e0699fa10c5be`: 531 IDs, 19 built rows without a test citation, zero unknown IDs, zero duplicates, and legacy reference report 895/906. This separately labelled observation was not rerun for this bundle.

## Future-module results

All commands, source commit hashes, branch names, timing, exits and independent event counts are recorded in [manifest.json](manifest.json), with original receipts in [future/current-results.json](future/current-results.json) and [future/sources.json](future/sources.json).

| Module | Source commit | Race outcome | Named pass / fail / skip events |
|---|---|---|---:|
| FNET-201 | `12850290c6bca7720853c33689662d751aced160` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-202 | `efc9d696eea2dd3260ecb6c5091802ef31e20572` | Timed out, exit 124 | 72 / 0 / 1 |
| FNET-203 | `bd2fbdbd3adf77b27fa12335ff2827359423aea3` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-204 | `5d3819c3b3af031b75eceb5251da0226a4ecee77` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-205 | `0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-206 | `8b444b20139e3e687db494b3fe36bb3643148e5b` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-207 | `0437e54897c6d9cfb3f2577807b82dd038493850` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-208 | `b4f511372ec26dc63d005bb50d87c56e18fc1a88` | Completed, exit 0 | 1 / 0 / 0 |
| FNET-209 | `1b3c92e03007e35e3f190e719a1ea78a807d4102` | Completed, exit 0 | 1 / 0 / 0 |
| FUX-101 | `a6326b8725fe6d7270997b4a284cbbc129022dd1` | Completed, exit 0 | 34 / 0 / 0 |
| FUX-102 | `9db8a323d7dcf430d1a58a4a4d10ca3628aeb401` | Timed out, exit 124 | 0 / 0 / 0 |
| FUX-103 | `34d0f889631e2d133149a89a276cf9c71fd261a9` | Not attempted / no race receipt | — |
| FUX-104 | `3bdf8e92d84af2c660c615bcf956cf76752e86ae` | Not attempted / no race receipt | — |
| FUX-105 | `7f92251cdb0d6ed5d9559730045a021a173a278b` | Not attempted / no race receipt | — |
| FUX-106 | `473cab2fb45a7b67ea0001f9fdc7833ffda45eb9` | Not attempted / no race receipt | — |

FNET-202 emitted successful package/test events before the runner killed the overall command at 180 seconds. No failure event is required for a timed-out run to remain incomplete. FUX-102 timed out without emitting any Go JSON test event. FUX-103 has only a started formatting log; FUX-104, FUX-105 and FUX-106 have no check receipts in this continuation.

## Integrity and interpretation

- Raw log bytes are preserved. SHA-256 and byte length are listed in the manifest for every copied artifact.
- [SHA256SUMS](SHA256SUMS) also covers this README and the manifest; it excludes itself.
- Go event counts include named parent/subtest events and are separate from package counts. They are not assertion counts or a claim that the complete Go suite executed.
- Source manifests describe isolated module work; they do not prove integration readiness. Default Go invocations do not imply execution of separately tagged SQLite paths, browser tests, or Node suites.
- Storage recovery tests and SQLite-backed token-reset integration tests require actual provider execution. Driver-boundary fault tests do not qualify physical disk, fsync, power-loss, or ambiguous post-commit acknowledgment failures.
- Blank output with an explicit exit-zero receipt can be a successful formatting/vet check. Blank output without such a receipt is only an incomplete artifact.
