# FUX-103 upstream research and adaptation handoff

Phase: **research complete; implementation and qualification pending**. Date: 2026-11-16. This report does not lift the feature's HOLD on integration, activation, merge, or release.

## Decision

Use the official **`golang.org/x/image/draw` at `v0.48.0`** as a pinned dependency for high-quality bounded thumbnail resampling, with standard-library JPEG/PNG decoding and clean re-encoding. Keep taxonomy, authorization, quotas, retention, pagination, and availability as Circula-owned code behind small repository/service interfaces. Do not transplant another marketplace's identity, listing ownership, payment, or persistence model.

The Go Authors' repository has mature interpolation code, tests, active maintenance, and a directly compatible Go API. Reusing this library executes its upstream code without creating a fork of generated resampling internals. It is more appropriate than maximizing copied lines. This is a suitable reusable component, **not an existing complete FUX-103 implementation**; no evaluated upstream supplies all of Circula's feature and security semantics.

The nested module currently declares Go 1.25.0. Selected x/image v0.48.0 requires Go 1.26.0, so the implementation agent should explicitly raise only this nested module's Go directive. The coordinator verified Go 1.26.11 can be downloaded by the existing toolchain; the core already requires it. Do not downgrade to stale packages merely to retain the earlier standalone toolchain. Pin module/checksums and record the actual toolchain/test output.

## Search and primary-source evidence

Used GitHub repository searches for `imaging user:disintegration`, `image user:golang`, and `bleve user:blevesearch`, then shallow-cloned all three repositories to inspect real source, license text, go.mod, current HEAD dates, and tags. The results showed all three repositories non-archived. Popularity counts are not treated as security evidence.

| Candidate | Exact revision inspected | Maintenance evidence observed | License read | Disposition |
|---|---|---|---|---|
| [golang/image](https://github.com/golang/image) | `v0.48.0`, `884d13f747188b6b8581f58ddb403e7a98874f3c` | HEAD tagged v0.48.0; commit dated 2026-11-09; maintained by Go project; upstream test and generated resampling implementations present | BSD-3-Clause-style [LICENSE](https://github.com/golang/image/blob/884d13f747188b6b8581f58ddb403e7a98874f3c/LICENSE), copyright 2009 The Go Authors | **Selected:** import draw package, not the repository's unnecessary decoders |
| [disintegration/imaging](https://github.com/disintegration/imaging) | `d40f48ce0f098c53ab1fcd6e0e402da682262da5` | Current HEAD commit dated 2020-12-18; non-archived but no recent HEAD activity; go.mod pins a 2019 x/image revision | MIT [LICENSE](https://github.com/disintegration/imaging/blob/d40f48ce0f098c53ab1fcd6e0e402da682262da5/LICENSE), copyright 2012 Grigory Dryapak | Not selected: attractive Fit/Thumbnail/auto-orientation API, but stale dependency baseline and unnecessary broad decoder registration |
| [blevesearch/bleve](https://github.com/blevesearch/bleve) | `v2.6.3`, `d6b0c82cd5c1bf27a0b74931d956becbdf4e038e` | HEAD tagged v2.6.3; commit dated 2026-11-04; Go 1.25.0 and substantial dependency graph | Apache-2.0 [LICENSE](https://github.com/blevesearch/bleve/blob/d6b0c82cd5c1bf27a0b74931d956becbdf4e038e/LICENSE); no root NOTICE file found in inspected checkout | Reserve for a measured future need; adding a second persistent index now creates authorization freshness and index consistency work |

### Specific APIs and source to reuse

- [x/image/draw/scale.go](https://github.com/golang/image/blob/884d13f747188b6b8581f58ddb403e7a98874f3c/draw/scale.go): `draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)` or `draw.ApproxBiLinear.Scale` where measured cost warrants. Aspect-fit dimensions and all resource caps remain Circula responsibility. Source comments document concurrent-safe Scaler implementations; they do not make input decoding or adapter state automatically safe.
- [x/image/draw/example_test.go](https://github.com/golang/image/blob/884d13f747188b6b8581f58ddb403e7a98874f3c/draw/example_test.go): official interpolation usage examples; implement original Circula fixture tests rather than copying its image corpus without per-file review.
- [x/image README security model](https://github.com/golang/image/blob/884d13f747188b6b8581f58ddb403e7a98874f3c/README.md#security-considerations): inspect `DecodeConfig` before `Decode`; upstream intentionally does not treat arbitrarily large dimensions as a decoder security bug. Circula must impose its own pixel, byte, concurrency and output budgets.
- [imaging/resize.go](https://github.com/disintegration/imaging/blob/d40f48ce0f098c53ab1fcd6e0e402da682262da5/resize.go) supplies `Resize`, `Fit`, and `Thumbnail`; [imaging/io.go](https://github.com/disintegration/imaging/blob/d40f48ce0f098c53ab1fcd6e0e402da682262da5/io.go) supplies `Decode` and `AutoOrientation`. Inspecting it showed imported GIF/BMP/TIFF decoders and `image.Decode` directly, without application pixel budgets. Do not use its convenience Decode as the upload boundary.
- [Bleve README](https://github.com/blevesearch/bleve/blob/d6b0c82cd5c1bf27a0b74931d956becbdf4e038e/README.md) shows `NewIndexMapping`, index creation, document indexing and querying. These are useful search capabilities but not network authorization. Existing SQLite search avoids an asynchronously stale external index for the first implementation.

## Licensing and provenance handoff

No upstream code or dependency has been added in this research phase. Before the implementation commit:

1. Add the exact x/image module version and go.sum entries in `future/discovery/` only; record the full upstream commit and packages used in THIRD_PARTY_NOTICES.md.
2. Preserve the complete BSD license text with the module's distribution notices. Source/binary redistribution must retain/reproduce the notice, conditions, and disclaimer; do not imply endorsement by Google or contributors.
3. If vendoring instead of importing, vendor only through the module tooling, preserve file headers/license, and record any modifications. Do not manually transplant generated `draw/impl.go` without its supporting package files.
4. No MIT imaging or Apache Bleve code is selected for copying. If the choice changes, retain the relevant full license and applicable attribution/NOTICE and re-review transitive dependencies.
5. Circula still has no approved project-wide product license. This report does not assign or change one. Preserve dependency notices independently and record distribution/license disposition before release.
6. Run current dependency/advisory checks as part of implementation; this research did not establish that any package is vulnerability-free.

## Existing Circula behavior and seams inspected

Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597` (unchanged).

- `internal/service/listings.go`: CreateListing and UpdateListing already validate text/price and use the service's transaction/idempotency/audit boundary. ListListings uses `s.open(... ScopeRead)`, `requireActive`, and `ActiveOwnersOnly`; owner views can see their paused/archived listings. Preserve those distinctions rather than replacing everything with the standalone Boolean `Visible` field.
- `internal/store/iface.go`: ListingFilter and ListingRepo expose listing persistence. Extend through an adapter and separate additive feature schema; do not introduce direct SQL from the web layer.
- `internal/store/sqlite_listing.go`: current search uses parameterized network filtering, active-owner membership filtering, `circula_fold` over title/description, and ID-descending pagination. The standalone future module currently searches only title using `strings.ToLower`, with updated-time/ID ordering. That is an intentional **compatibility gap requiring tests and an adapter decision**, not a drop-in replacement.
- `internal/api/h_listings.go`: current query allowlist and cursor binding include limit/cursor/kind/q/member_id; query length limit is 200 bytes. Add filters only through reviewed contract updates, bind all new filter/sort state into cursors, retain actor/network binding, and preserve existing ETag behavior.
- `internal/web/h_home.go`: current server-rendered listings/new/edit routes and forms provide the no-JS host seam. New gallery/filter templates should be independently rendered/tested without registering routes in the live app.
- `internal/store/migrations/0001_init.sql`: existing listing table must remain intact. The future module's migrations are opt-in artefacts only, never placed into the core's automatic migration directory in this development phase.

## Implementation plan covering the complete feature

| Spec requirement | Isolated module work | Tests required before calling it complete |
|---|---|---|
| Categories, goods/services, moderated tags, condition | Typed taxonomy and tag policy; network-scoped admin service operations; separate repository interface; preserve unclassified legacy listings | Duplicate names/IDs, invalid text, category disabled while listings exist, cross-network category assignment, non-admin mutation, tag moderation; legacy rows still searchable |
| Up to six photos, budgets, thumbnails, metadata removal | Media service with atomic quota reservation and owner authorization; permit JPEG/PNG initially; read bounded bytes, sniff/DecodeConfig, reject pixel/dimension overages before full decode, draw resample, re-encode into a fresh image file; never serve original bytes | Oversized body and dimensions, truncation, malformed image, unsupported SVG/GIF/TIFF, malicious metadata/EXIF/GPS, polyglot input, concurrent seventh upload, unauthorized delete/read, bounded output and metadata absence |
| Optional coarse location | Accept network-owned area ID via adapter; area unset by default; no coordinates or external geocoding; do not depend on future areas module | No GPS fields in DTOs, foreign/deleted area rejection, no geolocation permission needed. Distance/map remains optional and must not be simulated with exact locations |
| Relevance/recent sorting, filters, saved URLs, pagination | Preserve folded title+description behavior; deterministic relevance score with stable ties; bounded saved query serializer with no private/token fields; HMAC cursor bound to all filters, actor/network and snapshot revision | Equal timestamps/ranks, Unicode/accent matching, inserts/updates between pages, tampered/expired/cross-actor cursor, no hidden rows, query caps, stable no-duplicate pagination |
| Availability, expiration and renewal | Explicit clock; validated availability window; expiry filtering; renewal operation does not mutate trade history; audit and version checks through repository transaction interface | Boundary timestamps, stale concurrent renewals, suspended/departed owner, inactive network, historical transaction references unchanged |
| Privacy and lifecycle | Membership/owner/admin authorization evaluated at each read and mutation; archived media immediately unservable; retention-aware deletion job accepting injected clock and repository, never autostarting | Isolation matrix, owner/admin exceptions, revoke during read, crash between metadata write/file write, reference-safe GC, retention boundaries, restore consistency |
| API, CLI, accessible UI and help | Standalone adapters/demo harness with explicit construction; contract fragments and no-JS templates in module; CLI inspector/self-test if needed; no root route registration | RFC9457 responses, CSRF/idempotency, body-limit distinction for approved media exception, keyboard/alt text/320px/zoom, no-JS gallery/empty states, API compatibility |

### Media design details that must not be delegated to the image library

- Proposed initial configurable limits should be explicit in a Limits type, not hidden constants: maximum input bytes, width, height, pixels, output pixels, output bytes, six photos per listing, total listing bytes, concurrent decodes. Tests should use small limits to exercise boundaries. Select production defaults only with resource-budget evidence.
- Check pixel products without integer overflow. Decode from already bounded memory or bounded temporary storage; allocation limits must apply before image allocation.
- Decode/re-encode strips metadata only if the original container bytes are never copied. Hash the normalized output, set explicit MIME and `nosniff`, choose safe generated paths, and return authenticated media responses with private/no-store behavior according to adopted policy.
- Decoder success does not prove absence of polyglot content. Either reject trailing/container-invalid data with strict parsers or accept only the sanitized re-encoded raster while explicitly testing that active payload is absent and original bytes cannot be retrieved. Do not claim universal polyglot detection.
- EXIF orientation is a presentation concern separate from metadata stripping. If initial JPEG handling strips without applying orientation, record and fix that gap before full acceptance; do not silently claim smartphone orientation correctness.
- Do not derive filesystem paths from user filenames, fetch remote image URLs, or reuse uploader-provided content types.

## Phase-2 execution order and completion evidence

1. Pin dependency and notices, add an original bounded raster sanitizer and thumbnail tests/fuzz target.
2. Expand pure listing/taxonomy/availability/filter/cursor contracts with backward-compatibility tests and explicit input trust boundaries.
3. Add transactional storage adapter and migration artefacts scoped only to this module; prove restart, concurrency, quota and GC behavior with a real temporary database/filesystem.
4. Add explicit-construction API/HTML/CLI adapter harness, no-JS templates, and API/auth/CSRF contract tests. Never register with the main app.
5. Run unit/race/vet, targeted fuzzing and dependency checks, then hand off to a **fresh** reviewer agent. Record exact commands/toolchain/results and unresolved requirements; failing/manual gates remain pending.
6. Full core ledger regression, live integration contract migration, human accessibility and browser testing cannot be represented as passed just because standalone tests pass. They remain separate later acceptance gates, and integration stays on HOLD.

No implementation or feature-completion claim is made by this report.
