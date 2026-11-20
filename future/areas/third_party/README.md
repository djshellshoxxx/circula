# Dependencies and scope decision

Production source uses the Go standard library only. Tests reuse the exact `modernc.org/sqlite v1.60.1` module as the existing core-compatible SQLite driver. Exact module/transitive versions and checksums are in go.mod/go.sum. `dependencies.json` records the resolved 25-module graph and retained actual upstream LICENSE/COPYING/NOTICE files. No upstream source is forked or modified.

UPSTREAM-RESEARCH.md selected Leaflet as an optional renderer and Orb only for an adopted geometry extension. This implementation adopts the specification's named-area/list-filter path; maps/geometry/points/distances remain absent, so no unused map dependency is bundled. Adding a map later requires explicit scope/privacy/advisory/browser review against those pinned sources. No product license is assigned to Circula by these notices.
