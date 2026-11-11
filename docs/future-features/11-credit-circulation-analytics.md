# FNET-205: Aggregate credit-circulation analytics

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Medium · Recommended phase: 3 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Network coordinators can see overall exchange volume and liquidity patterns without exposing individual private balances.

**Current baseline:** Ledger verification and admin overview/metrics exist; circulation-oriented dashboards are not established.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Compute volumes by week/month, unique active traders, exchange count, dormant offers, and aggregate positive/negative distributions.
2. Provide member-view of their own trend and admin-only anonymized aggregate view with export.
3. Use ledger postings and completed transactions as source truth; distinguish flow (turnover) from stock (outstanding balances).
4. Suppress small cohorts and high-risk slices (recommended threshold at least five accounts) and make raw personal-data export separately permissioned.
5. Chart accessibility includes descriptive tables; chart JS optional; no analytics tracker beacon.
6. Support read-only snapshots generated under database isolation and explicit time zone/period labels.

## Proposed persistence and migration

- No mandatory new table; materialized aggregate cache may be keyed by network_id, period, revision and invalidated on posting.
- Never cache unscoped raw member balances in a shared location.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET /api/v1/admin/analytics/circulation?from=&to=&period= and GET /api/v1/me/trend.
- No new write path to ledger.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Avoid charts that reveal trades from small populations; permission checks on every aggregate.
- No member-level reputation or scoring derived from circulating balances.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Test fixture turnover and number of posted trades against independently summed ledger entries.
2. Cross-network isolation, k-anonymity suppression, timezone edges, reversal accounting.
3. No public chart endpoint or private export from ordinary members.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- chartjs/Chart.js — MIT client charts; bundle locally, no CDN.
- gonum/gonum — BSD-style numeric functions if advanced statistics later justified.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Personal credit scoring or automatic sanctions.
- Substituting analytics cache for verified ledger.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
