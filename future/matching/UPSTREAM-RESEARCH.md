# FNET-201 upstream research and implementation handoff

Date: 2026-11-16. Phase: research only. Integration: **HOLD**. No source, dependency, production route or migration was added by this phase. A fresh implementation agent should consume this handoff; a separate fresh audit agent should subsequently review its changes.

## Decision

Reuse **Gonum v0.17.0 graph primitives and strongly connected components**, plus **golang.org/x/text v0.43.0 Unicode normalization/case folding**. These mature libraries provide reusable algorithmic code without importing another application's identity, privacy, database or payment model. Implement Circula's small, bounded, explainable matching policy around them. No mature upstream examined supplies the complete Circula feature; claiming otherwise would conceal substantial original policy and adapter work.

Do not transplant a full recommendation service or introduce a search server. Keep the module independent of FUX-103 discovery and FNET-206 areas: accept optional public category/coarse-area strings through a snapshot interface, with no module imports. No balance, exact location, message, dispute, private profile or behavioral tracking input is permitted.

## Evidence and pinned candidates

GitHub repository search was performed for Gonum graph and Bleve search. Official repositories were then cloned/read; exact release refs, source files and licenses were inspected. The Go module proxy's verified module metadata was used for x/text. Activity observations are point-in-time evidence, not security certification or a promise of future support.

| Candidate | Exact pin and observed maintenance | License actually inspected | Decision |
|---|---|---|---|
| [gonum/gonum](https://github.com/gonum/gonum) | Release `v0.17.0`, commit `fc402bc485e3a92f8d4f1f0ee5a49e2edf232ed2`, dated 2025-12-30. Repository HEAD `bda3a37c92f9e957b722e54d48976dbac393bec4`, dated 2026-11-10. Release `go.mod` requires Go 1.24.0. | [BSD-3-Clause license at selected commit](https://github.com/gonum/gonum/blob/fc402bc485e3a92f8d4f1f0ee5a49e2edf232ed2/LICENSE), plus BSD-style file headers in graph source. | **Selected** for graph representation and SCC pruning. Stable release rather than moving HEAD. |
| [golang/text](https://github.com/golang/text) | `golang.org/x/text v0.43.0`, commit `ac40b0f06c159afb5bfca4d1860f8b4b60dbc160`, release metadata timestamp 2026-10-09. This exact version already exists in Circula root `go.mod`; it requires Go 1.26.0. | [BSD-3-Clause license](https://github.com/golang/text/blob/ac40b0f06c159afb5bfca4d1860f8b4b60dbc160/LICENSE). Module downloaded successfully with Go 1.26.11. | **Selected** for Unicode matching; pin explicitly in isolated module. |
| [blevesearch/bleve](https://github.com/blevesearch/bleve) | `v2.6.3`, commit `d6b0c82cd5c1bf27a0b74931d956becbdf4e038e`, dated 2026-11-04; tag and inspected HEAD resolve to same commit. Go 1.25.0. | [Apache-2.0 license](https://github.com/blevesearch/bleve/blob/d6b0c82cd5c1bf27a0b74931d956becbdf4e038e/LICENSE). Root contains LICENSE, no root NOTICE observed. Transitive/file notices require their own review if adopted. | **Deferred**, not copied. Useful search primitives, but a second index creates synchronization, deletion and tenant-isolation burdens. The current 500-listing bounded scan does not justify it. |

No GPL/AGPL source is selected. Libraries should remain pinned module dependencies, avoiding a fork that loses upstream fixes. Any vendoring/copying must retain original copyright/license files, mark local modifications, and record exact paths/commit. Binary distributions must carry required third-party notices. This research does not assign a product license to Circula, approve release, or override its integration hold.

## Concrete code/API reuse

### Gonum

- [`graph/simple/directed.go`](https://github.com/gonum/gonum/blob/fc402bc485e3a92f8d4f1f0ee5a49e2edf232ed2/graph/simple/directed.go): `simple.NewDirectedGraph`, `NewEdge`, `SetEdge`. Assign stable numeric node IDs from lexicographically sorted network-local member IDs; do not persist global identity mappings.
- [`graph/topo/tarjan.go`](https://github.com/gonum/gonum/blob/fc402bc485e3a92f8d4f1f0ee5a49e2edf232ed2/graph/topo/tarjan.go): `topo.TarjanSCC` identifies connected candidate groups. Sort results explicitly; graph/map iteration order is not a stable presentation contract.
- [`graph/topo/johnson_cycles.go`](https://github.com/gonum/gonum/blob/fc402bc485e3a92f8d4f1f0ee5a49e2edf232ed2/graph/topo/johnson_cycles.go) was inspected as a **non-adoption** reference. `DirectedCyclesIn` enumerates elementary cycles without a Circula length-3/output/work bound. Do not call it on an untrusted dense graph and truncate afterward; the expensive work has already happened.
- Keep original Circula code for bounded length-3 traversal, canonical cycle rotation, per-member caps, consent policy and explanation generation. Reuse algorithms, not upstream domain assumptions.

### x/text

- [`cases/cases.go`](https://github.com/golang/text/blob/ac40b0f06c159afb5bfca4d1860f8b4b60dbc160/cases/cases.go): `cases.Fold().String`. Its documented case folding does **not** normalize text; use a normalization step, not lowercase-only matching.
- [`unicode/norm/normalize.go`](https://github.com/golang/text/blob/ac40b0f06c159afb5bfca4d1860f8b4b60dbc160/unicode/norm/normalize.go): `norm.NFKD`, `norm.NFC` and `Form.String`; [`runes`](https://github.com/golang/text/tree/ac40b0f06c159afb5bfca4d1860f8b4b60dbc160/runes) and [`transform`](https://github.com/golang/text/tree/ac40b0f06c159afb5bfca4d1860f8b4b60dbc160/transform) allow compatibility with the existing application folding pipeline.
- Treat folded terms as search evidence only, never identity keys or authorization identifiers. Bound original input bytes/runes and number/length of unique tokens before scoring. Language-specific stemming and semantic/AI scoring are out of this first deterministic model.
- `v0.43.0` module checksum observed: `h1:1QivrlhwAsnMOcqgOdl9my7s9OyZl8O5/UGUTJVXoWc=`. Let `go mod tidy` generate/check the complete dependency graph; do not hand-author sums.

### Bleve (not selected now)

[`query.go`](https://github.com/blevesearch/bleve/blob/d6b0c82cd5c1bf27a0b74931d956becbdf4e038e/query.go) supplies `NewMatchQuery` and `NewConjunctionQuery` if future profiling justifies indexed candidate generation. Authorization/visibility must still be rechecked after search; a filtered query is not an authorization boundary. Do not copy sample applications or add a persistent index in this phase.

## Existing Circula seams verified

- `internal/service/listings.go:ListListings`: opens an authenticated service read transaction, requires an active member, and applies active-listing/active-owner filtering for other members. Querying the actor's own member ID can return their paused/archived listings; a matching adapter must independently exclude them. Never assume every service listing result is eligible for matching.
- `internal/store/sqlite_listing.go:ListListings`: network parameter is explicit, owner filtering checks current membership, and text search uses `circula_fold`. Keyset paging is by listing ID. Fetch a bounded coherent snapshot, not unbounded pages assembled while state changes.
- `internal/domain/fold.go:FoldName`: existing pipeline is NFKD, removal of combining marks, Unicode folding, NFC, plus a dotless-i compatibility rule. Keep matching token normalization consistent through a narrow injected normalizer or duplicated documented adapter implementation in this independent module; do not import `internal/domain` from a separate module.
- `internal/domain/types.go:Listing`: current listing model has title/description, type, owner, status and timestamps; **it has no category or area fields**. Standalone matching can accept optional category/area metadata, but the future adapter must map from adopted schemas explicitly. Missing category must not silently make every historical listing invisible if normalized terms provide adequate evidence.
- The inspected baseline contains no established per-member block-list service suitable for this feature. Define an injected exclusion/policy interface and fail closed if it cannot be evaluated. Do not create a dependency on the unfinished messaging module to obtain block decisions.

## Full-spec implementation plan for the fresh agent

| Requirement | Implementation boundary and reuse | Acceptance evidence needed |
|---|---|---|
| Deterministic opt-in pair matching | Extend trusted snapshots with public title/description terms, optional category/coarse area, listing revision and separate offer/request opt-in flags. Use x/text folding, a fixed documented integer scoring formula and named evidence codes. Reject invalid input and suppress zero-evidence pairs. Sort by score then stable IDs. | Same results under input permutation; Unicode composed/decomposed/case tests; missing-category term match; no match for mere same-network membership; bounded text/token sizes. |
| Explainability and advisory-only behavior | Return reasons such as `same_category`, `shared_terms`, and only consented `same_coarse_area`. Supply plain-language suggestion labels and links to existing review flow. No price/balance ranking or transaction mutation interface. | Assertions on exact reason codes, escaped HTML, neutral no-JS output; no posting service in module dependency graph. |
| Separate opt-outs, exclusions, caps and dismissals | Define narrow authoritative `SnapshotProvider`/`PolicyStore` interfaces. Persist preferences and owner-private dismissals through an explicit `database/sql` adapter or service-store port; tenant/member keys mandatory. Default opt-in false. Inject clock. Expired dismissals do not suppress indefinitely; preference changes rechecked at read time. | Cross-tenant/member denials; offer-only/request-only opt-out; two-direction member block checks; per-listing exclusion; owner-only dismissal visibility; expiry boundary; restart durability and concurrent writes. |
| Optional 3-way cycles | Convert approved offer-to-request pairs into member-directed edges. Use Gonum for graph/SCCs. Store listing-pair evidence separately from the simple member graph so parallel listing alternatives are not lost. Enumerate only length three with three distinct members; sort and canonicalize rotations, retain meaningful direction. Enforce node/edge/operation/output/time budgets before and during work; context cancellation. | Obvious 3-member fixture; deduplicate rotations; reject self, 2- and 4-cycles; dense graph/cancellation tests; every constituent edge remains authorized and opted-in; no automatic settlement. |
| Privacy/input minimization | Types contain only authorized public listing fields and coarse area consent. No balance, account risk, messages, disputes or exact coordinates. Return suggestions relevant to the requesting active member, and avoid disclosing why another user was excluded. Recheck permissions on read even if cached. | Reflection/schema tests for prohibited fields where meaningful; hidden/departed/archived/suspended/cross-network cases; cached item revocation/deletion tests. |
| Rescoring without behavioral tracking | Prefer recomputing a bounded in-memory snapshot. If cached later, key by network, actor, listing/policy revision and expiry, with explicit size limits. Persist only opt-in preferences and explicit dismissals; no click tracking. | Approved listing edit changes score; deletion/opt-out invalidates immediately; empty/cache-failure path remains independent of trade service. |

### Storage/API/UI deliverables still necessary

1. Keep all implementation, module dependencies, SQL schema, handlers, templates and tests under `future/matching/` on this branch. No registration or migration in root.
2. Ship additive module-owned preference/dismissal schema with network/owner scoped unique keys, expiry indexes, migration versioning, retention policy, restart tests and backup/restore instructions. Inject DB/transaction and identity adapters; do not open the production DB automatically.
3. Provide explicit optional handlers/renderers for matching list, preferences and dismiss action. Authentication, X-Network, 64-KiB body cap, CSRF, idempotency, no-store, RFC9457 errors and HMAC cursor binding must be adapter contracts verified in tests. Prefer no-JS server-rendered forms; CLI calls the same service layer. Do not create a second login/session system.
4. Document proposed OpenAPI separately within the module. Updating root API docs/routes and exposing the actual existing Circula auth/store adapter are later single-feature integration work, still on HOLD.
5. Use Go 1.26.11 (available via `GOTOOLCHAIN=go1.26.11 /tmp/circula-toolchain/go/bin/go`), aligning with root and x/text requirements. Record dependency/notice provenance and run `go mod verify`, tests/race/vet, adversarial/fuzz inputs and dependency advisory scanning if available. No claim that this research checked all upstream advisories.
6. Provide a requirement-by-requirement completion matrix and remaining manual accessibility, actual Circula-adapter, full-ledger-regression and release gates. Unit-test success is not full-feature completion, and full-feature completion is not permission to merge.

## Fresh review phase checklist

The next reviewer must specifically inspect policy freshness, preference defaults, private dismissal ownership, candidate-cap starvation, integer-score overflow, Unicode resource limits, canonical cycle IDs, parallel-edge treatment, actual work bounds (not merely output truncation), signed cursor scope, SQL isolation/retention and no-ledger authority. Confirm copied code/dependency notices against the exact pin. Test malformed/large inputs, cancellation, cross-network IDs, expired dismissals and revision changes. Keep all unfinished items explicit rather than marking the feature complete by narrowing the spec.

## Research disposition

Suitable reusable upstreams were identified and license/API evidence read. No donor was claimed to supply Circula's authorization, tenant privacy, trade consent, persistence or complete UI. Those are necessary original adaptation tasks. This handoff authorizes no runtime activation or main merge.
