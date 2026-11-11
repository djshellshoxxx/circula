# FUX-102: QR identifiers and one-time trade requests

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 1 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Static merchant identifiers and one-time requests can prefill a trade, while the existing proposal/acceptance mechanism remains the only posting authority.

**Current baseline:** Manual propose/accept already posts two ledger entries and an audit event; QR support does not exist as an approved settlement path.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Define versioned QR URLs on the trusted Circula origin, with opaque network-scoped member alias or 128-bit randomly generated one-time token. Never embed credentials, private keys, balance or exact location.
2. Static aliases resolve to recipient identity but contain no amount; one-time requests carry fixed amount in integer minor units, optional bounded note, expiry and revocation.
3. After scanning, render network, payer, payee, amount and estimated available balance on a confirmation screen; do not treat scan as acceptance or proof of payment.
4. Camera enhancement must be optional; allow clipboard/manual code input and downloadable printable high-contrast QR.
5. Single-use request claiming and transaction creation share one atomic operation, with a stable idempotency key bound to the intent.
6. A recipient cannot claim success until the transaction is completed after both sides consent; 'pending' and 'completed' are distinct.

## Proposed persistence and migration

- qr_member_alias(network_id,member_id,alias_hash,revoked_at,created_at) UNIQUE(network_id,alias_hash).
- qr_trade_request(id,network_id,recipient_id,token_hash,amount_minor,note,expires_at,revoked_at,claimed_tx_id) with unique token and claimed_tx_id.
- No persistence of raw camera image or raw bearer-like QR token in logs.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET /api/v1/me/qr, POST /api/v1/me/qr/rotate (idempotent).
- Proposed POST /api/v1/qr-requests, GET /api/v1/qr-requests/{token}, POST /api/v1/qr-requests/{id}/revoke; map intent to existing POST /api/v1/transactions and /accept.
- All additions must follow current RFC9457 problem responses, X-Network scoping, cookie CSRF and cursor policies.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- No open redirects or untrusted schemes; resolve URL/alias against configured trusted public origin and selected network.
- Expire and revoke atomically; reject duplicate claims and cross-tenant identifiers with the existing not-found policy.
- Bound QR size and scanner input; do not load remote scanner scripts or store token plaintext.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Static QR, two members and confirmed trade: exactly two balanced ledger entries and one completed transaction.
2. Concurrent scans of one-time QR yield at most one claimed transaction; retries return the prior result without duplicate posting.
3. Tampered host, amount, expired/revoked token, suspended user and cross-network scan all fail closed.
4. Mobile scan, print, manual-entry and keyboard paths; parser fuzzing and no secret logs.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- skip2/go-qrcode — MIT; Go QR PNG generation.
- nimiq/qr-scanner — MIT; local browser camera scanning and Web Worker; evaluate BarcodeDetector fallback.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Automatically accepting trades; QR as identity authentication; offline settlement.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
