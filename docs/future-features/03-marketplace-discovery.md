# FUX-103: Enhanced marketplace and discovery

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: High · Recommended phase: 1 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members locate relevant goods and services with less scrolling, while exact location and profile information remain private.

**Current baseline:** Listings already support offers/requests, title/description search, visibility, pagination, create/edit/pause/archive and report flow.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Add controlled category taxonomy with network-admin extension, service/goods type, moderated tags and optional condition/availability.
2. Allow up to six listing photos under configurable per-file and per-listing limits, with server-generated thumbnails and removal of EXIF, GPS and unneeded metadata.
3. Add optional approximate area and distance filter; listings default to no location, and maps must never expose residential coordinates.
4. Offer relevance and recent-update sorting; include predictable filters, empty states, saved view URLs that do not embed secrets and accessible pagination.
5. Permit user-configurable availability/expiration and stale listing renewal while preserving original trade records.
6. Add result privacy: only members of that network can retrieve active listings; suspended/archived entries excluded except to owner/admin.

## Proposed persistence and migration

- Additive listing_category, listing_tags, listing_media (owner,sha256,mime,dimensions,storage_path,retention,status), listing_availability and coarse_area.
- Migration must preserve every existing listing; old unclassified listings remain searchable.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Extend GET /api/v1/listings with optional category, availability, coarse_area, sort, cursor fields; preserve existing response/ETag behavior.
- Proposed POST /api/v1/listings/{id}/media and DELETE /api/v1/listings/{id}/media/{mediaId}, POST /api/v1/listings/{id}/renew.
- All writes pass existing authorizations and Go service layer; never edit ledger outside transaction service.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Decode images with bounded dimensions/memory and recompress; reject SVG and dangerous polyglots.
- Never fetch arbitrary remote image URLs from server (SSRF); upload authorization, rate and quota limits; encrypted backup retention policy reviewed.
- Never infer user home location from metadata; hide coordinates from images and APIs.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Search preserves network isolation and pagination stability under inserts/updates.
2. Malicious ZIP/SVG/huge image, decompression bomb, EXIF geolocation and unauthorized delete are rejected or stripped.
3. Archived images hidden and GC'd after policy retention; existing listing search and transaction reference tests remain green.
4. Photos appear in responsive, keyboard-accessible gallery with alt text and no JS fallback.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- disintegration/imaging — MIT Go image resize; evaluate maintenance/security posture.
- blevesearch/bleve — Apache-2.0 Go search if SQLite query/FTS5 proves inadequate; avoid two unsynchronized indexes initially.
- medusajs/medusa — MIT core but Node/TypeScript commerce platform; patterns only, not transplant of payment engine.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Full e-commerce carts, custodial inventory or external payment processing.
- Public indexing of members or exact home address searches.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
