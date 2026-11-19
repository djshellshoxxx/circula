# FNET-206: Coarse regions and privacy-safe local discovery

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Medium · Recommended phase: 2 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Show members trades in a selected area without exposing their precise home location.

**Current baseline:** Profiles and listings are private to network members, with no approved location tracking or exact-distance discovery.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Network admins define named areas (neighborhood, district or broad polygon), with no need for GPS.
2. A listing may choose zero or one coarse area and optionally an approximate public meetup point; never demand browser geolocation.
3. Distance estimates are rounded and optional; if privacy risk cannot be bounded, use only selected-area filtering instead.
4. Map mode is optional; list and keyboard view always available and equivalent.
5. Member exact location is never stored by default; location metadata stripped from listing images.
6. Federated networks do not share location attributes unless explicitly allowed per network.

## Proposed persistence and migration

- network_areas(id,network_id,label,coarse_geometry_or_code,visible,created_at).
- listing_area(listing_id,area_id,approx_public_point_optional,visibility) with strict area membership.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET /api/v1/network/areas and GET /api/v1/listings?area={id}; admin CRUD on area metadata.
- Map endpoint returns only permitted approximate/published area geometry, never member address.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Avoid use of hosted geocoding that discloses member search terms/IPs by default.
- Map tile provider policy and license independent of Leaflet; self-host tiles optional and usage-limited.
- Reject high-precision geometry from normal members.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Two listings same area and one different; filtering and map match, no cross-network leaks.
2. EXIF GPS removed and no exact coordinates in API/HTML/JSON.
3. Location permission denied/disabled still permits all core flows.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- Leaflet/Leaflet — BSD-2-Clause local map widget; does not supply maps/tiles.
- disintegration/imaging — MIT for metadata-safe listing thumbnails.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Realtime location, tracking, geofencing, exact-address routing.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
