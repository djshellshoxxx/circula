# FUX-106: Guided onboarding and practice network

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 1 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

A new member can understand balances, locate a first offer and complete a practice trade without risking real community credit.

**Current baseline:** Network join, member terms, listings and trading are implemented but an integrated tutorial/sandbox is not established.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Welcome checklist: join/accept terms, learn credit signs/limits, add an offer/request, find a trading partner, propose and accept an exchange.
2. Use accessible server-rendered progressive steps with skip/resume; no blocking overlay or forced tour for returning users.
3. Create a strictly isolated synthetic practice network with canned members and scripted events; practice data must never be mixed into real network lists or exports.
4. Expose a preview of balance before and after a proposal, explain credit limit and reservation, and clearly label every practice screen.
5. Allow community admins to configure onboarding copy and links to governance/terms without changing core rules.
6. Collect only aggregate opted-in completion metrics; default off, never infer identity or creditworthiness from progress.

## Proposed persistence and migration

- optional member_onboarding_progress(network_id,member_id,step_version,completed_mask,updated_at) and practice fixtures kept outside production database by default.
- No real-money or live ledger entries created by tutorial.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Use existing account, listings, transaction read endpoints; optional GET/PATCH /api/v1/me/onboarding-progress.
- Practice mode should run in isolated in-memory or temporary database with a separate origin/network explicit marker and no production imports.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Strong demo/real visual separation and zero ability to convert demo trades into real postings.
- Tutorial steps work with keyboard/screen reader; respect reduced-motion and skip buttons.
- Demo and real credentials must never be interchangeable.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. First-time account can reach a real listing and propose in no more than five guided actions.
2. Practice balances change with fixtures but production ledger and admin audit remain byte-identical.
3. Browser test covers skip/resume, account switch, reduced motion, no-JS and translations.
4. Users correctly identify a negative balance and that a proposal is not completed payment.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- alphagov/govuk-frontend — MIT reusable form/error-summary/accessibility patterns.
- GoogleChrome/workbox — MIT only for optional app-shell/offline help; not needed for guided onboarding logic.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Automated account approvals; fake credit awards; importing sandbox state into production.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
