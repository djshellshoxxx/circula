# Areas upstream reuse matrix

| Component | Pinned revision/version | Actual reuse | Modification/license evidence |
|---|---|---|---|
| modernc.org/sqlite | v1.60.1, exact checksum in go.sum | Test-only real SQLite driver; production uses database/sql | No fork or copied source; retained licenses under third_party/licenses |
| Resolved test transitive modules | Exact25-module inventory in third_party/dependencies.json | Driver/tool dependency graph only | Actual license/notice files retained; checksums in go.sum |
| Leaflet / Orb research | References in UPSTREAM-RESEARCH.md | Not used; optional map/geometry scope not adopted | No dependency or source bundled |

Native area authorization, lifecycle, replay, scoped query and HTTP contract code remains Circula-specific. No third-party runtime library proves those boundaries. The current continuation adds no dependency. See THIRD_PARTY_NOTICES.md and third_party/README.md for actual reuse and notice inventory.
