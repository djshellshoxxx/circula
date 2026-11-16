# FNET-207: Recurring trade templates and reminders

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Medium · Recommended phase: 2 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Frequent exchanges can be prepared on a schedule, but every actual trade still receives fresh consent.

**Current baseline:** Current trades require active proposal and counterparty acceptance; there is no approved recurring posting bypass.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. A member may save a template (counterparty, note, optional default amount, recurrence expression, reminder time, active state).
2. At each due interval create an in-app reminder or draft proposal only; no autopost or autoaccept.
3. Each generated proposal must receive a distinct idempotency key/intent ID and respect current membership, credit limits and terms.
4. Allow pause, skip, cancel and edit with audit; DST/calendar behavior specified for named IANA timezone.
5. Bound backfill after downtime: max one notification per schedule window, no unbounded catch-up posting.

## Proposed persistence and migration

- trade_templates(id,network_id,owner_id,target_id,note,amount_minor_optional,frequency,timezone,active,created_at).
- trade_template_firings(template_id,scheduled_for,reminder_id,created_at) UNIQUE(template_id,scheduled_for).

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed CRUD /api/v1/me/trade-templates and POST /api/v1/me/trade-templates/{id}/create-draft (Idempotency-Key).
- No new ledger endpoint; proposal/accept must use current transaction routes.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Maximum schedule frequency and per-member count; input validation for timezone, leap days and duplicates.
- Member suspensions and deleted counterparts suspend triggers; no hidden background financial authority.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Restart at boundary/timezone DST yields one reminder per planned occurrence.
2. Two workers racing generate no duplicate intent; separate repeats never reuse key.
3. Disabled/expired/suspended member does not create proposals; no automatic posting ever.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- robfig/cron — MIT Go schedule parsing and timing, but persistence/dedupe is Circula-specific.
- Existing in-app notification scheduler and outbox can provide reminders.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Subscriptions automatically debiting credit, escrow, recurring acceptance delegation.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
