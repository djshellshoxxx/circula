# FUX-105 milestone E — independent search delta audit

2026-11-16. Standalone verification milestone complete; overall qualification partial. Feature disabled, integration HOLD, no remote writes.

## Exact recovery and ownership

- Branch: `work/requalify-messaging-20261116`.
- Worktree: `/workspace/scratch/3ef42c6ef5d8/feature-worktrees/messaging`.
- Recovery HEAD: `d4498fc7a6bce9c99d27a9f717149e755e07f56b`.
- Runtime search implementation: `0cc6585f2e6c998f5208c6fe569ea4afbf9cc392`.
- Prior independently audited baseline: `3b4fbe4497827072df15f75e8e518f89925d6435`.
- Current tested source: `da972087fbfa28e6bf71b3310038f70f105d0055`.
- This handoff is a following documentation-only commit. No runtime code changed in milestone E; only files under `future/messaging` changed.

The recovery worker was independent of the original search implementation team. It read the master directive and Caveman instructions, audited the source delta and reused the existing default modernc SQLite implementation and upstream notices. No paid service or external coding worker was used.

## Work completed

Preserved and committed the previously dirty `testing.TB` fixture change and cancellation/benchmark tests. Tightened the benchmark to assert 100 exact hit results and zero miss results, and seeded the matching thread sequence. Added a boundary regression proving that search one second before expiry does not extend content visibility.

Independent inspection covered SQL bindings, current transaction authorization, network/participant/hidden/expiry/purge checks, HTTP method/CSRF/JSON handling, escaped no-JS forms, CLI stdin/transport, and their contracts/tests. No confirmed new content-disclosure or authorization defect was found. SECURITY-AUDIT.md documents the bounded review; AUDIT.md preserves the prior broader audit.

## Observed validation

Go 1.26.11 at `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`, `GOGC=20 GOMAXPROCS=1`, `-p 1`, default build cache. The full uncached race suite passed with library 74.4% and CLI 78.4% statement coverage, including recovered cancellation and new retention regression. Vet exited zero without diagnostics; module checksum verification passed. Targeted search race tests also passed before the new deadline regression. See VALIDATION.md for exact commands and outputs.

The diagnostic benchmark used 10,000 synthetic 3,900-byte messages and three iterations per query: hit 633,055 ns/op, 448,101 B/op, 953 allocs/op; miss 87,842,174 ns/op, 6,813 B/op, 150 allocs/op. Both expected result counts passed. These measurements do not prove a production latency objective.

`git diff --check` and JSON parsing of the standalone OpenAPI/feature descriptors passed. No new fuzz or vulnerability scan ran; earlier evidence is retained with its original scope. No browser setup was rerun because the coordinator has an existing runtime launch blocker.

## Remaining gates and next work

Weighted evidence rises from 87% to 90% solely for the independent search review criterion. Real-browser/no-JS, automated WCAG and manual screen-reader acceptance are unverified. Broad performance qualification remains open: a miss scans the remaining conversation while the transaction holds the existing SQLite writer reservation; the response cap does not bound query work. A successor should define and execute realistic retained-history/concurrency workloads and cancellation during an active scan before closing this gate.

Real host session/CSRF/membership/network-retention/moderation adapters, platform retention/backups and full core/ledger regression remain deferred integration gates. This milestone grants no merge, route registration, core migration, activation, deployment or release permission.

Remote publication remains blocked by the coordinator's automatic approval rejection. Do not push or use an alternate write transport. Preserve the current worktree and older recovery source named in milestone D.

Future compiler work must be coordinated with the shared queue and use `GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache`, `GOGC=20 GOMAXPROCS=1` and `-p 1`; this milestone used the default cache before that shared-cache instruction arrived. The validation slot was released after the benchmark completed.
