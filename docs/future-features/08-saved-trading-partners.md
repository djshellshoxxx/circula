# FNET-202: Saved trading partners and quick repeat

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 1 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members reach a frequent trading partner and prefilled proposal without searching the directory every time.

**Current baseline:** Member directory and historical trade counterparties exist; no verified favorites/quick-repeat interface.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Opt-in favorites list scoped to network, with private nickname and explicit pin/unpin.
2. Quick-repeat copies recipient and optional description from a historical trade but defaults to an editable blank amount, with full new confirmation.
3. Block/suspended/departed rules override favorites; never suggest members absent from the current network or whose visibility denies access.
4. Display favorites in Find and Trade with keyboard/mobile affordances; fallback to directory search.
5. Favorite state has no effect on credit limits, ledger or reputation.

## Proposed persistence and migration

- member_favorites(network_id,owner_member_id,target_member_id,nickname,created_at), UNIQUE(owner,target); cascade/soft-delete on departure policy.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET/POST /api/v1/me/favorites and DELETE /api/v1/me/favorites/{memberId}.
- Reuse POST /api/v1/transactions with a newly generated stable idempotency key per distinct proposal.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Only owner sees their list; favorite relation is never exposed to target or operator metrics by default.
- Cross-tenant and self-favorite requests refused; quick-repeat never reuses the original idempotency key.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Favorite add/remove/reorder and nickname privacy; suspended partner cannot be paid.
2. Two repeats of one old trade remain two deliberate, independently consented proposals.
3. Account switching and network switching never leak favorites.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- No standalone matching project worth porting; extend the existing member directory and transaction form.
- alphagov/govuk-frontend — MIT for accessible links/buttons only.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Automatic repeat transfers, member contact import or public social graphs.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
