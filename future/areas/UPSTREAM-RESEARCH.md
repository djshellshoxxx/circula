# FNET-206 upstream research and implementation handoff

Research date: 2026-11-16. Phase: **research only**. Integration remains **HOLD**. No runtime dependency, third-party code, route, schema, or main-application change is introduced by this document.

## Decision

Use **Leaflet 1.9.4** as an optional, locally bundled map renderer, with no tile layer, geocoder, GPS request, CDN or remote asset enabled by default. Reuse the library rather than copying a whole location-sharing application. The primary capability remains named-area administration and area-filtered listings using server-rendered forms. The feature must work with JavaScript and geolocation both disabled.

For an explicitly adopted coarse-geometry extension, **paulmach/orb v0.14.0** is the best-fit Go component for typed Polygon/GeoJSON data. Do not make geometry necessary to create or select an area. **golang/geo** is a credible alternative for spherical geometry, but its spatial indexing/cell machinery is unnecessary for the initial named-area use case.

No mature upstream component inspected supplies Circula's network authorization, administrator policy, listing ownership, metadata stripping, or retention rules. Those must remain Circula-specific service/store logic. Importing more code would not satisfy these requirements.

## Candidates and evidence

GitHub repository search was used to locate all three repositories. The official repositories were cloned and their source, exact commit metadata, package/version declarations and license files inspected. Maturity judgments below are engineering judgments, not security certifications. A release age and a recent project commit are different facts.

| Candidate | Exact pin and maintenance evidence | License inspected | Disposition |
|---|---|---|---|
| [Leaflet/Leaflet](https://github.com/Leaflet/Leaflet) | Stable `v1.9.4`, peeled source commit `d15112c9e8ac339f0f74f563959d0423d291308d`, 2023-05-18. Annotated tag object is `91900f7b177e6a926dfc3a75c86dc11fd4f2cf34`; do not confuse it with the source commit. Current main inspected at `ee96166ccfcf941cd91fc99fcb3f38734816deaa`, 2026-11-16, package `2.0.0-beta.4`. | BSD-2-Clause; retain copyright, conditions and disclaimer. | Selected stable rendering API. The stable release is old; current development is active but beta. Do not silently upgrade to 2.x APIs. Dependency advisory/browser qualification remains required. |
| [paulmach/orb](https://github.com/paulmach/orb) | `v0.14.0` and HEAD resolve to `b85ffaf98ee23643f268efd131b4af2fc656f861`, 2026-11-01. Changelog includes fixes to malformed geometry decoding and bounds. Go declaration 1.18. | MIT in `LICENSE.md`; retain notice for copied/substantial bundled portions. | Selected conditional geometry helper. Use only typed geometry/GeoJSON packages actually needed. Do not adopt MongoDB, MVT or protobuf functionality merely because it is in the module graph. |
| [golang/geo](https://github.com/golang/geo) | Inspected commit `5e8d6967632ac525710fe8003a87ca2614bca61e`, 2026-11-06. Go declaration 1.25.0; includes tests and CodeQL/CI configuration. | Apache-2.0 in `LICENSE`; preserve required notices if copied; assess any NOTICE file in the adopted revision. | Not selected for this milestone. Useful if future design needs spherical polygon/cell operations, but would add sophistication without solving privacy/authorization. |

Pinned official sources:

- [Leaflet package/version](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/package.json), [license](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/LICENSE), [stable release](https://github.com/Leaflet/Leaflet/releases/tag/v1.9.4).
- [Orb changelog](https://github.com/paulmach/orb/blob/b85ffaf98ee23643f268efd131b4af2fc656f861/CHANGELOG.md), [license](https://github.com/paulmach/orb/blob/b85ffaf98ee23643f268efd131b4af2fc656f861/LICENSE.md), [module dependencies](https://github.com/paulmach/orb/blob/b85ffaf98ee23643f268efd131b4af2fc656f861/go.mod), [README/API examples](https://github.com/paulmach/orb/blob/b85ffaf98ee23643f268efd131b4af2fc656f861/README.md).
- [Go Geo scope](https://github.com/golang/geo/blob/5e8d6967632ac525710fe8003a87ca2614bca61e/README.md), [license](https://github.com/golang/geo/blob/5e8d6967632ac525710fe8003a87ca2614bca61e/LICENSE), [module](https://github.com/golang/geo/blob/5e8d6967632ac525710fe8003a87ca2614bca61e/go.mod).

This feature need not copy GPL/AGPL code. No product-wide license is chosen here. Retaining upstream notices for a permitted component does not license unrelated Circula code.

## Exact reusable APIs/files and adaptations

### Leaflet

- [`src/layer/GeoJSON.js`](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/src/layer/GeoJSON.js): `L.geoJSON` converts an already-authorized coarse area projection into vectors. Use explicit styling and a strict Polygon/MultiPolygon-only wrapper; do not render member Point features.
- [`src/layer/vector/Polygon.js`](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/src/layer/vector/Polygon.js), [`src/map/handler/Map.Keyboard.js`](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/src/map/handler/Map.Keyboard.js), and [`dist/leaflet.css`](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/dist/leaflet.css): bundled polygon presentation and keyboard navigation. Keep equivalent links/list controls outside the map.
- [`spec/suites/layer/GeoJSONSpec.js`](https://github.com/Leaflet/Leaflet/blob/d15112c9e8ac339f0f74f563959d0423d291308d/spec/suites/layer/GeoJSONSpec.js), `spec/suites/layer/vector/PolygonSpec.js`, and `spec/suites/map/handler/Map.KeyboardSpec.js`: adapt relevant fixture patterns with attribution if code is copied.
- Bundle built assets from a verified pinned package/build with a lockfile, SHA-256 artifact hashes and local license notice. Do not fetch code or map tiles in the end user's browser. A bundled library is actual code reuse even when its source is not forked.

**Two source-level traps:** `GeoJSON`'s `filter` only applies when new data is added; changing it does not revoke already rendered layers. On permission/visibility refresh, clear old layers and replace the authorized projection. The upstream popup example returns a description string, which is inappropriate for user-supplied labels: create DOM nodes and assign `textContent`, not HTML strings. Do not call `map.locate` or add a tile provider just because examples do.

### Orb, only for approved coarse geometry

- `orb.Polygon`, `orb.MultiPolygon`, `Geometry.Bound`, and `geojson.Feature`/`FeatureCollection` represent controlled geometry and serialize its projection.
- [`planar/area.go`](https://github.com/paulmach/orb/blob/b85ffaf98ee23643f268efd131b4af2fc656f861/planar/area.go) and [`planar/contains.go`](https://github.com/paulmach/orb/blob/b85ffaf98ee23643f268efd131b4af2fc656f861/planar/contains.go) provide planar operations, **not** square-meter area or a privacy guarantee for longitude/latitude. Never use planar degree area as a physical-area privacy threshold.
- Input bounds, geometry-type allowlist, vertex limits, finite/range checks, antimeridian policy, administrator authority and public-coarseness policy must precede geometric processing. Arbitrary member-supplied geometry is prohibited. A label can itself expose an address; coordinates being absent is not proof of anonymity.
- Prefer a pinned Go dependency rather than copying parsing code and losing upstream bug fixes. If geometry is deferred, do not add the dependency gratuitously.

## Existing Circula seams inspected

- `internal/service/listings.go`: `ListListings` runs inside `s.view`, opens the network principal, requires active membership, and uses `ActiveOwnersOnly` for another member's listings. Add any future area filter inside that same authorization and query path; do not query all listings then filter in HTML.
- `internal/store/iface.go`: `ListingFilter`/`ListingRepo` are the repository boundary; `internal/store/sqlite_listing.go` implements tenant-scoped queries. Future association lookups must use network plus listing ID and preserve existing paging order.
- `internal/domain/types.go`: `Listing` currently has no area field. Keep the future association additive and nullable; preserve all historical listings.
- `internal/service/principal.go`: `requireAdmin` enforces active admin status and configured MFA. Do not replace it with a boolean supplied by an HTTP request.
- Core persistence uses `modernc.org/sqlite`; don't introduce a second SQLite driver. The standalone module may implement a narrow store interface with a test SQLite adapter while main remains untouched.

## Full-spec adaptation sequence for the fresh implementation agent

1. **Durable catalog and association model.** Separate `AreaService`, repository interfaces, and renderer. Define network-scoped area IDs, bounded normalized labels, visible/archived state, timestamps and revision. Add a unique listing association per `(network_id, listing_id)`, with composite relationship constraints to area network. Unset is an explicit valid value. Define idempotent admin writes and member association updates with current authority checks in the same transaction.
2. **Lifecycle.** Prefer archival over deleting referenced areas. Hidden/archived areas immediately disappear from ordinary catalog/filter results. Preserve historical internal associations for audit while preventing stale labels/geometry from leaking. Decide whether restoring an area restores listing visibility; document/test instead of implicit behavior. Listing departure/archive cleanup cannot mutate ledger or historical trade records.
3. **Primary interface.** Server-rendered select/list/filter forms with an unset choice, clear labels/errors, keyboard navigation, and no geolocation request. Narrow optional handler/CLI adapters belong inside this module until later explicit core integration. No new global feature flags or core routes now. Tenant/member/admin context must be supplied by an authenticated adapter, never trusted from body JSON.
4. **Privacy projection/export.** Export only allowed named-area fields for the current network. No member coordinates, residential address fields, inferred location or precise distance. Federation export defaults to omitting all area attributes even if another module is absent; provide an explicit per-network sharing policy boundary, not a dependency on the unfinished federation feature.
5. **Optional map.** Only if coarse geometry scope is adopted, bundle the pinned Leaflet renderer behind explicit use. Feed it server-authorized polygons and render the same area IDs as the list. No raster tile layer is needed for the local vector-only preview. Geometry unavailability must preserve list filtering. Public meetup points and distance estimates may remain absent because the spec explicitly makes them optional; document that scope, never claim they are implemented.
6. **Media boundary.** Area service must not accept/upload images or read EXIF. If future listing photos are connected, require an image-sanitization adapter contract and a test fixture proving GPS metadata removed before storage. Do not import the unfinished discovery module. Until that integration exists, mark cross-feature photo acceptance unqualified rather than claiming areas itself strips metadata.
7. **Documentation and acceptance.** Add an implementation-status matrix per functional requirement, migration/backup/retention guidance, dependency notices and exact test output. Maintain HOLD until fresh reviewer audits the module, core is separately complete, and owner selects individual integration.

## Tests required before calling the module complete

- Persist/reopen catalog and associations; migrate a DB with pre-existing unclassified listings; verify backup/restore and forward migration behavior. Do not use a down migration that destroys user data as a rollback story.
- Admin/member/nonmember/suspended/MFA-required matrix for CRUD and assignment; foreign network area and foreign listing references denied. Verify checks use fresh trusted state, not a captured authorization flag.
- Two concurrent assignment updates, duplicate idempotency keys, archived/deleted area during assignment, stale revisions and restart after commit. One listing must never acquire two area associations.
- Two listings in area A, one in B, one unset; every list/filter/map result uses identical visible IDs. Insert/update pagination remains stable under the stated snapshot semantics.
- Label control/bidi/invalid UTF-8/HTML payloads; unknown JSON fields and coordinate injection; network/ID limits; bounded catalog queries. Existing `Validate` does not reject Unicode line/paragraph separator characters—review along with ID and network validation rather than assuming it is production-ready.
- No-JS, keyboard, zoom/reflow and manual screen-reader checks. Optional map cannot become a keyboard trap or the only way to select an area.
- Browser deny-geolocation scenario and network interception proving no GPS, geocoder, tile, CDN or analytics request. Map disabled or failed script still supports all essential flows.
- If geometry adopted: NaN/infinity, huge vertex count, invalid rings, Point/GeometryCollection rejection, high-precision member submissions, antimeridian boundaries, and escaped labels. No mathematical threshold alone should be labelled anonymity.
- Export/federation tests prove location omission by default; listing image adapter fixture proves no GPS EXIF. Record these as integration gates if the core adapter is deliberately deferred.
- `go test -race ./...`, `go vet ./...`, dependency vulnerability scan, fuzz input parsers, plus browser tests for adopted JS. Race tests alone do not establish transactional correctness.

## Handoff status

Research is complete; feature implementation is not. No third-party files were copied during this phase. The next agent should implement on this worktree, retain `integration: HOLD`, and commit only `future/areas/`. Never merge/main-enable a map or area feature merely because its standalone unit tests pass.
