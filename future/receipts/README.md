# FNET-203: Optional digital trade receipts

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

## Recovery checkpoint — 2026-11-19

This isolated candidate restores saved head `91529c5a3a4268ecf3a919e099ab9aca779aab2d` onto its published ancestor `bd2fbdbd3adf77b27fa12335ff2827359423aea3`. All original files and tests are retained. A reviewed follow-up repairs numeric rounding in the plain offline decoder: original amount, revision and precision tokens must equal their validated integer values exactly. Thirteen original Go files and both pinned module files remain byte-identical; `canonical.go` changes and one new regression file is added. New recovery documentation and a dedicated native qualification workflow distinguish this candidate from the older milestone. Two inherited Core files are restored to their exact already-merged main versions so the combined checks can build and exercise the existing departure regression. See [RECOVERY-20261119.md](RECOVERY-20261119.md).

The implementation, completion percentage and `tested_source` records below are historical source provenance. They are not a current-head release, external interoperability or production-key approval. The current master directive authorizes reviewed isolated repository publication; the integration HOLD remains.


The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Current branch: `codex/future-receipts-master-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`.
- Recovered feature code remains in `future/receipts/`, with its new dedicated `.github/workflows/future-receipts.yml`. The candidate also restores the accepted-main audit-anchor `os` import and departure-test fixture cap. No future-feature imports, routes, migrations, root dependencies, feature flags, background jobs or runtime configuration are added.
- This nested Go module has the pinned JCS and go-jose dependencies documented in THIRD-PARTY.md and no dependency on another future module. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution, DB connection, external service or ledger write is present.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- No artifact persistence is permitted or provided. The read-only service and adapters require a fresh authorized core snapshot in one existing read transaction; actual core adapter conformance remains a later qualification gate.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied; pinned JCS and go-jose are reused as module dependencies with their licenses.

## Implemented milestone

Versioned authorized read-only projection with deterministic TXT/JSON/print HTML, exact integer bounds, two entry references and immutable status supplements. Added strict JCS offline parser/canonical digest, no-JS opt-in HTTP adapter, shared authenticated CLI seam, host auth/CSRF/rate-limit hooks, tenant/consent/concurrency/recreation and adversarial tests. Added an independently enabled/consented standard JWS EdDSA candidate issuer using an opaque server signing handle, strict offline verifier, immutable local public trust-set loader, key lifecycle/rotation/revocation checks and separate no-JS certificate HTTP/CLI seams. See IMPLEMENTATION.md for the acceptance matrix, CANDIDATE-ADR-001.md for the reviewed experimental profile and CERTIFICATE-OPERATIONS.md for the key/trust runbook.

## Existing application seams inspected

`internal/service/transactions.go; internal/service/export.go`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

External whole-verifier interoperability; actual Circula read-transaction/auth/protected-key adapters; production cryptographic/operations and trust-distribution approval; manual browser/accessibility/Tor and full core regression gates. No production key or final cryptographic approval has been adopted.

## Standalone validation

```sh
cd future/receipts
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

Current master criteria and remaining gates: COMPLETION.md. Query validation and ordinary form compatibility: INTEGRATION.md. Current runtime passed fresh tests and independent delta review; external qualification remains open.
