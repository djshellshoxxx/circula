# FNET-209 milestone M — strict HTTP boundaries

Branch: `codex/future-federation-master-20261116`. Worktree: `/workspace/scratch/4f14005536ba/work-federation`. Recovered predecessor: `e09f1598c32b11922982517b7f64a4b5a0f0b5c9`. Exact tested source: `a833a8636724ac437d9680f9689f28a7257c7419`. This handoff is a documentation/evidence successor; no runtime or dependency change follows the tested source.

## Implemented and reviewed

Bound member pagination before host parsing; reject query ambiguity. Admin routes accept no query. Parse bounded forms strictly before HostAuth and use body-only mutation fields. Preserve valid same-URL paginated member submissions and the oversized-body 413 response. Generate form tokens before fresh same-actor authentication and private transactional reads. Regressions cover authorization revocation/replacement/cancellation and malformed ignored values.

Coordinator independently reviewed the runtime correction, final response-contract preservation and new regressions. SECURITY-AUDIT.md distinguishes predecessor whole-module review from this bounded delta review. No new unresolved defect was confirmed in this bounded inspection. No global atomicity across arbitrary host session providers is claimed.

## Actual verification

Focused tests PASS (0.026s); full uncached race PASS (15.312s, 74.2%); vet PASS; module verification PASS. Raw logs and checksums are in recovery-evidence/. New code is limited to interface.go and members.go; tests are master_interface_test.go. Remaining changes are current documentation, manifest and evidence.

## Reused components

No new upstream component. go-jose v4.1.9, commit `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`, supplies standard JWS EdDSA APIs under Apache-2.0. modernc SQLite v1.60.1 supplies test-only database/sql fixtures. UPSTREAM-REUSE-MATRIX.md and THIRD_PARTY_NOTICES.md retain exact URLs, graph pins and 33 license files for 26 modules. No borrowed code is modified and no root dependency is added.

## Remaining acceptance and next team

Weighted criteria: 81.25/100 in COMPLETION.md, not feature completion. Independent live-peer interoperability, real browser/manual accessibility, integrated rate-limit deployment and production key custody/protocol decisions remain open. Lifetime quotas preserve replay safety but need an approved operational epoch/compaction design before indefinite operation; do not erase replay/high-water metadata to recover capacity. Actual Circula transaction/departure/suspension/deletion/auth adapters remain separately deferred integration work.

Next fresh team: protocol/operations reviewer for an explicit safe lifetime-capacity strategy, plus independent browser/interoperability tester when an executable environment or peer exists. Preserve signed discovery-only semantics. No settlement or balances. Do not restart historical implementations or treat earlier scans as fresh.

## Resume commands and boundary

Use `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go` from this worktree's future/federation directory. Set `GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache`; run `go test -p 1 -race -cover -count=1 -timeout 90s ./...`, `go vet -p 1 ./...`, and `go mod verify` only after a relevant delta or unresolved concern. Coordinate the shared compiler slot.

All feature paths remain future/federation, disabled and integration HOLD. No push, main merge, activation, root routes/migrations/dependencies, deployment or release. GitHub publication still requires explicit named-destination authorization and successful review.
