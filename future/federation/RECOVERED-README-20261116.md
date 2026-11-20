# FNET-209: Opt-in discovery federation

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Branch: `codex/future-federation-master-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`.
- Only `future/federation/` is added. No application imports, routes, migrations, root dependencies, feature flags, workflows, jobs or runtime configuration are changed.
- This nested Go module pins go-jose v4.1.9 and a test-only SQLite driver. It has no dependency on another future module. See THIRD_PARTY_NOTICES.md. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution or ledger write is present. SQLite and bounded HTTPS operations require explicit host construction and calls.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- An explicit caller-owned SQLite reference adapter provides standalone persistence and transactional auth/source callbacks. No production database is opened or registered. The real Circula service/store adapter and full core invariants remain unqualified.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. Runtime library APIs are reused through pinned dependencies; licenses/notices are retained.

## Implemented milestone

Experimental JWS/EdDSA v2 with strict typed public content, pinned peer keys, validity/revocation, durable transactional replay/projection/tombstones, scoped owner consent and withdrawal intents, bounded leased outbox/retries, SSRF-safe HTTPS transport and unregistered no-JS member/admin/CLI interfaces. The prototype v1 framing remains disconnected from ingress. See ADR-0001.md and ACCEPTANCE.md.

## Existing application seams inspected

`internal/service/listings.go; future separate peer-facing adapter`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

Current delta verification/review; live external-peer interoperability and lifetime-capacity/epoch operational qualification; actual Circula transaction/departure/suspension/delete adapters and ledger regression; production key custody/protocol approval; browser accessibility. Phase B settlement is excluded pending separate accounting approval.

## Standalone validation

```sh
cd future/federation
go test -race -cover ./...
go vet ./...
```

Tests use synthetic fixtures only. See `VALIDATION.md` for observed results. These tests qualify the stated core milestone, not the full roadmap feature, production behavior, browser accessibility, or Circula's release.

## Later integration, one feature at a time

1. Finish this module's remaining scope and acceptance tests. Audit its own code and record unresolved findings.
2. Qualify the then-current Circula core separately. Do not infer completion from this module's tests.
3. Rebase this branch on the qualified core; review the baseline diff and any API/schema drift. No promise of a conflict-free future merge is made.
4. Add a narrow adapter, reviewed API contracts, additive migrations (if needed), explicit per-network opt-in, CLI and accessible/no-JS UI in a separate integration change. Keep all other future features absent.
5. Recheck tenant/auth/CSRF/idempotency behavior, concurrency/restart, backup/restore, full ledger regression and relevant manual/browser acceptance.
6. Obtain the owner's explicit selection of this feature for integration. Only then merge that single integration change. Never auto-merge this hold branch or batch-enable the backlog.
7. Disable by removing routes/jobs and adapter registration; do not roll back ledger history. Any feature-data retention/migration procedure must be documented before integration.

No work on this branch changes Circula's core completion or release status.

Current master evidence: COMPLETION.md, SECURITY-AUDIT.md, INTEGRATION.md and FINAL-HANDOFF.md. Predecessor transport/quota/migration/restore audit results remain in AUDIT.md; they are not new current-source executions.
