# FUX-101: Installable mobile-first PWA

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 1 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members install Circula on mobile and reopen it as an app without creating a second transaction client or losing no-JavaScript usability.

**Current baseline:** Circula already renders responsive server-side HTML and has a JSON API; no installable/offline web-app acceptance is claimed.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Provide same-origin web manifest, stable app scope/start URL, install help, icons and mobile home navigation for Find, Trade, Balance and Inbox.
2. Build a service worker that precaches only fingerprinted public assets; authenticated HTML, API responses, tokens, receipts, account details and exports MUST use no-store and never enter browser caches.
3. Offline mode explicitly displays 'unavailable' and refuses create/accept/decline/reverse actions; there is no background replay, queued trade, or claimed offline balance.
4. Support screen-reader labels, safe areas, reduced motion, 200–400% zoom, 320px minimum reflow and plain HTML online workflows.
5. Provide a visible sign-out flow that scrubs in-memory private views and does not expose stale content when an installed app is reopened or devices are shared.

## Proposed persistence and migration

- No new authoritative ledger or user data table; any saved presentation preferences must be per-member and exclude sensitive information.
- Service worker version, cache names and manifest version are build artefacts; updates must purge obsolete caches.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Reuse existing /api/v1 network, session, listings, transactions, notifications and balances. Introduce no parallel wallet or posting API.
- Static assets served by Go http handlers with explicit cache headers; app routes remain compatible with reverse proxy/base path.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- HTTPS required outside loopback; service-worker scope is same-origin and tightly bounded.
- No client-side storage of auth responses, private inbox, offline proposals or credentials.
- Real network status comes from successful server communication, not browser navigator.onLine alone.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Install/reopen on supported Android Chromium and manual iOS Safari Add to Home Screen; login, browse, propose and accept one synthetic trade.
2. Disconnect immediately before submit: no posting, no deferred replay; reconnect requires explicit new action.
3. Switch account, sign out, go Back and offline reload: prior member's private data never appears.
4. Cross-browser keyboard/screen-reader/mobile accessibility and CSP/asset integrity tests.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- GoogleChrome/workbox — MIT; reuse only install/caching mechanics with explicit no-store allowlist.
- alphagov/govuk-frontend — MIT; adapt accessible patterns to Go templates, not an entire frontend rebuild.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Native Android/iOS apps; offline trade authority; caching sensitive balances.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
