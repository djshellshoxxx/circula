# FUX-105 milestone D handoff — 2026-11-16

**Standalone interfaces complete; verification partial; integration HOLD.**

- Local working branch: `work/requalify-messaging-20261116`.
- Exact implementation commit: `0cc6585f2e6c998f5208c6fe569ea4afbf9cc392`.
- Selected recovered baseline: `3b4fbe4497827072df15f75e8e518f89925d6435`, descended from published branch `future/fux-105-messaging-20261116` at `7f92251cdb0d6ed5d9559730045a021a173a278b`.
- Older outage implementation preserved separately: `1b5fe42883c29a2dcfd9e924f7d28facac9b541a`. Do not cherry-pick it over the newer implementation: architecture and interfaces differ.
- Worktree: `/workspace/scratch/3ef42c6ef5d8/feature-worktrees/messaging`.
- Completion: 87% weighted acceptance evidence; see COMPLETION.md. No full-feature or release qualification claim.

## Completed

Recovered and reproduced the newer audited standalone SQLite service, API, no-JS member/operator forms, CLI, retention/holds/reports, restart and concurrency behavior. Completed missing private-message search across the service, POST JSON API, no-JS forms and CLI. Added scoped search regression tests, OpenAPI contract, help, reuse matrix, migration/integration boundaries, security continuation and weighted completion documentation. No code outside `future/messaging/` changed.

The search uses existing SQLite `instr` with bound values; no new dependency or schema was introduced. Complete modernc driver v1.60.1 remains pinned to upstream `b122d0417c01508beb55158faedeb64a0c5bfd8a`; license and bundled notices are in THIRD_PARTY_NOTICES.md and third_party. Optional WebSocket candidates remain omitted by the selected architecture. UPSTREAM-RESEARCH.md preserves candidate and license evidence.

## Actual validation

- Exact newer baseline race suite reproduced: library 74.1%, CLI 77.7% statement coverage.
- New targeted search race suite passed.
- Final complete default-driver race suite passed: library 74.4%, CLI 78.4% statement coverage.
- `go vet -p 1 ./...`: passed without diagnostics.
- `go mod verify`: all modules verified.
- Strict JSON fuzz: 3,685 executions, 34 baseline cases, 15 new interesting cases; passed.
- Whitespace/diff checks and standalone OpenAPI JSON parse passed.

No cgo substitute was used. Tests use synthetic authorization, temporary private SQLite files and local TLS fixtures. No real member data or external messages were accessed.

## Review and unresolved gates

Inherited fresh independent audit at AUDIT.md has been preserved and its automated checks reproduced. New search delta has only implementation-team review; assign a fresh reviewer before marking its security gate passed. Review should focus on data visibility, retention/hold/hide, POST privacy/CSRF, bounded query and pagination behavior. No known confirmed search defect remains.

Actual browser creation is reported blocked by the coordinator's runtime attempts; this team did not rerun it. Real-browser/no-JS, WCAG and manual screen-reader acceptance remain unverified. Broad performance qualification remains open. Actual host authorization/CSRF/retention/audit adapters, opt-in installation and core/ledger regression remain future integration gates. Do not write a percentage of 100 merely because current Go tests pass.

Remote publication was not attempted by this team. The coordinator prohibited further remote writes after its automatic approval rejection; do not bypass that decision by using another transport. All commits are local.

## Resume

```sh
cd /workspace/scratch/3ef42c6ef5d8/feature-worktrees/messaging
git status --short
git log -3 --oneline
cd future/messaging
GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 /workspace/scratch/0afb7942ff6b/toolchain/go/bin/go test -p 1 -race -cover -timeout 90s ./...
GOMAXPROCS=1 /workspace/scratch/0afb7942ff6b/toolchain/go/bin/go vet -p 1 ./...
```

Fresh reviewer should read this handoff, the source at the exact implementation commit and its delta from the recovered baseline. Preserve the older checkpoint and current isolated branch. Integration, activation, release and deployment remain HOLD.
