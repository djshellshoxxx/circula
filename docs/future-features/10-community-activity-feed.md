# FNET-204: Community activity feed

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Medium · Recommended phase: 2 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members see newly available goods, recent requests, community announcements and scheduled events in one network-scoped feed.

**Current baseline:** Network announcements and a private notification inbox already exist.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Unify public-to-network announcements and listing events with a consistent newest-first feed; no disclosure of balances or private trades.
2. Member-controlled feed categories, mute/follow and optional digest, with privacy-preserving defaults.
3. Feed event derived from committed listing/announcement event, not duplicate stored copies; enable decay/expiry and moderation.
4. Private notifications remain private and separate; no 'X paid Y' posts unless both parties expressly authorize future feature.
5. Infinite/next-page keyboard-accessible pagination and readable timestamps; no required JavaScript.

## Proposed persistence and migration

- Optional feed_preferences(network_id,member_id,categories,updated_at).
- If materialized, feed_items(network_id,source_kind,source_id,event_kind,created_at) unique source event; content derived by permission-checked join.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET /api/v1/feed?limit=&cursor=&type=; GET/PATCH /api/v1/me/feed-preferences.
- Use existing announcements and listing APIs, with consistent HMAC cursor paging.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Filtering at read time must re-check latest listing visibility and membership; deletion immediately disappears.
- Rate-limit feed writes, never include private profile data or exact geolocation.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Only current-network active listing/announcements appear; suspended and deleted listing disappears.
2. Insert/update ordering and cursors are stable under concurrent writes.
3. One source event cannot create duplicate feed entries; no trade/ledger data leak.
4. Accessibility with no-JS.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- gotify/server — MIT real-time push model as a reference or optional external transport; not suitable as feed database.
- Existing Circula in-app notifications and announcements should be reused directly.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Public social media; behavioral ads; transaction surveillance feed.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
