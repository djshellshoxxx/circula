# FNET-203: Shareable, verifiable trade receipts

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 1 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members can obtain an accurate, printable proof of what Circula recorded, without revealing credentials or making unsupported payment guarantees.

**Current baseline:** Completed trade, audit event and immutable ledger entries already exist; optional formatted receipt artifact is new.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Expose receipt only for completed transactions to authorized counterparties or scoped administrators.
2. Receipt contains network ID/name, trade ID, payer/payee display identities, integer-formatted credit amount and unit, completion time, status, and immutable ledger sequence references.
3. Every receipt states 'Mutual-credit record; not bank/cash settlement'; no signature-based independent verification claim unless separately implemented.
4. Server-rendered HTML print version plus optional generated PDF; use explicit no-store/private headers.
5. After a reversal, original receipt remains immutable but a fresh status view links the compensating transaction; do not rewrite original.

## Proposed persistence and migration

- No mandatory new table; receipt projection is derived from immutable stored transaction and entry records.
- If signed receipts added later, separate key-management, signature version and verification requirements.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET /api/v1/transactions/{id}/receipt?format=html|pdf; use current authentication and network checks.
- No public/tokenized receipt links by default; exported files contain no session tokens.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Escape all text; pin/inspect fonts and license; avoid externally hosted PDF resources.
- Prevent other-network/member retrieval and inadvertent shared caches.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Receipt amount, timestamp, two entries and IDs exactly match authoritative ledger at time of creation.
2. Repeated downloads byte-stable where PDF tool supports deterministic output; no duplicate posting.
3. Unauthorized and cross-network receipt requests indistinguishable from other protected record requests.
4. PDF, printable HTML and accessible alternate text checked.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- go-pdf/fpdf — MIT Go PDF generator with no non-stdlib dependencies.
- GOV.UK Frontend — MIT print and accessible table patterns.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Blockchain notarization, bank-style legal settlement certification, public receipt lookup.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
