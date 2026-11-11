# FNET-201: Advisory trade matching

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 2 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Find likely offer/request pairs and optional multi-member trading chains without exposing private member information or making automated transfers.

**Current baseline:** Existing listings can be searched by type and text. No production matching service or ranking assurance exists.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Start with deterministic opt-in matching of active offers to active requests inside the same network using category, normalized terms and approximate geography if permitted.
2. Return explainable reason codes and a clear 'suggestion' label; no obligation to accept and no automated action on balances.
3. Allow separate seller/buyer opt-out and listing-level exclusion, block lists, per-member suggestion caps and dismissals.
4. Optional cycle suggestions (A wants B, B wants C, C wants A) are advisory only; each subsequent trade still needs payer/payee acceptance.
5. Use only network-visible listings, not messages, private disputes, balances or exact location; suppress matches when insufficient evidence.
6. Avoid persistent behavioral tracking by default; support rescoring on approved public listing edits.

## Proposed persistence and migration

- match_preferences(network_id,member_id,enabled,updated_at) and match_dismissals(network_id,member_id,listing_pair_hash,expires_at).
- No derived matching graph needs to contain user secrets; cached candidates expire/recompute.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET /api/v1/matches?cursor=...; POST /api/v1/matches/{id}/dismiss and POST /api/v1/me/match-preferences.
- No posting endpoints; suggestion click opens existing listing/trade review only.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Never match across network boundaries or reveal hidden/deleted listings.
- Strict CPU/time caps on candidate graphs; cycle length capped at three initially.
- Don't infer credit-worthiness or rank people based on negative balances.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Fixtures with obvious pair and 3-way cycle yield deterministic, explainable suggestions.
2. Cross-network, blocked, archived or opted-out listings are never surfaced.
3. An empty match cache or matcher crash never interrupts trade proposal or posting.
4. No automatic acceptance, ledger mutation or hidden sensitive input.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- blevesearch/bleve — Apache-2.0 for search candidate generation, only if existing SQLite queries insufficient.
- gonum/gonum — BSD-style Go graph algorithms, reusable for bounded matching/cycle discovery.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- AI pricing, automatic barter settlement, personal credit scores.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
