# Recovery status — 2026-11-19

**Disabled; integration HOLD. Independent recovery-delta review and Linux real-driver race/vet/checksum checks passed. Native and provider qualification remains pending.**

This branch restores verified archived `areas` work from `999dcbf12ffcf2bdcd1764393c5f470fe3c6fbc3`. [RECOVERY-20261119.md](RECOVERY-20261119.md) records the exact source, new parser regressions, current evidence and remaining gates. The earlier README, validation/audit claims and `feature.json` completion percentage below are preserved historical evidence; they are not current-head or cross-platform qualification.

The recovery's scope is `future/areas/`, its new `.github/workflows/future-areas.yml`, and two explicitly authorized exact restorations from accepted main: `cmd/circula/auditanchor.go` and `internal/service/departure_pagination_regression_test.go`. No other Core runtime, dependency, migration, shared workflow or feature path changes.

The current master directive authorizes reviewed isolated publication and normal gated merges into this module target. Any older publication prohibition in the preserved manifest or historical documents describes the earlier authorization state; it is not a current publication blocker. HOLD continues to prohibit Core activation.

The actual production parser and original fuzz target also completed 100,000 fuzz executions. This parser-only result is separate from the passing complete real-driver race suite.

---

# FNET-206: Coarse geographic areas

**HOLD — isolated standalone implementation; predecessor audit completed; current HTTP delta tested and independently reviewed. Not approved for integration or release.**

This module implements administrator-defined **named areas**, optional listing assignment, privacy-scoped filtering, and explicit network sharing policy. It stores no coordinates, residential addresses, precise distances or automatic location tracking. Explicit private assignment retry replies are retired on listing cleanup while their replay fingerprints remain. It requests no GPS permissions and uses no geocoder, tiles, CDN or tracker. A human-readable label can still disclose a location; administrators must choose appropriately broad names.

## Isolation and scope

Only `future/areas/` changes. No core route, migration, dependency, feature registration, job or ledger operation is added. The package imports no other future feature or Circula internal package. Its SQLite schema installs only when the host explicitly calls `Service.Migrate` on an explicitly supplied connection. The test-only pinned `modernc.org/sqlite v1.60.1` driver exercises persistence; production code accepts `database/sql` and opens no database.

The owner authorized isolated future-feature development, while deferring integration until the feature and core are qualified. Historical implementation prohibitions in copied specifications are superseded for this isolated scope only. HOLD remains.

The adopted initial surface uses named areas and selected-area filtering. Geometry, maps, distance estimates and public meetup points are optional in the specification and deliberately absent. Leaflet/Orb research remains available for a separately adopted geometry extension; this implementation does not copy unused map code or claim a map milestone. No image upload is accepted: a later listing-media adapter must prove EXIF/GPS sanitization separately, without importing the unfinished discovery module.

## Implemented behavior

- Admin/MFA-aware trusted policy seam, checked inside every mutation transaction; current membership, ownership and listing visibility checked inside reads.
- Network-scoped area creation/editing/hiding/archival with optimistic revisions, durable actor-scoped idempotency fingerprints and audit rows. Area quota: 1,000 historical catalog records/network.
- One nullable area per network/listing, ownership checks, current-selection lookup, explicit unset and cleanup. Cross-network foreign keys and service checks preserve scope.
- Archiving immediately deactivates associations and advances their revisions. Restoring an area **does not** restore those selections. Hiding removes labels/results immediately; restoring visibility can expose still-active selections. This distinction is intentional.
- Freshly authorized listing filtering with encrypted actor/network/area-bound cursors. Cursors expire after 15 minutes; each request scans at most 1,000 references and returns at most 100 visible IDs. Hidden listing IDs never appear in cursor plaintext.
- Durable per-network federation sharing defaults off. Explicit admin changes use revisions/idempotency. Export contains only visible named taxonomy, never members or listing associations.
- Escaped no-JavaScript member selection/filter forms and admin catalog/sharing forms, no-store/CSP headers, bounded strict JSON and form requests, host-auth/CSRF adapters and shared CLI functions.
- Reopen, closed-database backup/restore, transaction rollback injection, concurrent stale updates, current-auth/MFA/tenant/owner denial and privacy regressions.

## Host contracts

Construct `Service{DB, Policy, Now}` explicitly. `Policy.Check(ctx, tx, identity, action, object)` must read the host's current principal/network/membership/MFA/listing authority **through the supplied transaction**, not trust client booleans or cached session roles. Actions: `read`, `admin`, `assign`, `cleanup`, `listing`. `assign` and `cleanup` require listing ownership or narrowly delegated departure authority; `listing` requires current visibility. Policy callbacks must not perform side effects. Host transaction/store compatibility is not yet qualified.

Use one process per SQLite file, the existing host connection/transaction model, foreign keys on **every** connection and an appropriate busy timeout. The standalone tests serialize an explicit connection pool; production must retain the core's concurrency contract. No service method silently starts a second transaction from a host callback. For future in-transaction core adapters, map these operations into the established unit of work rather than nesting independent writes.

For `Web`, inject host authentication with current session checks and CSRF verification on mutations; provide a CSRF token renderer and a 32-byte cursor secret from the host secret store. Never accept identity, admin privilege, sharing authority or cursor keys from request JSON. Mount with an explicit path-strip adapter; no route is automatically registered.

Each fresh write needs a distinct request key; retries reuse the same key and exact payload. Changed payloads conflict. Retries still check current policy before replaying a result. Retry records are not expired automatically, preventing forgotten-key replay. Assignment/sharing retries require current matching object state; archived, changed or retired replies do not return stale active selections. Listing cleanup replaces cached assignment payloads with fingerprint tombstones. A 10,000-record/member/network quota bounds storage; retention/compaction must preserve replay tombstones or disable the module during an approved data retirement. Owner-authorized listing cleanup is naturally idempotent and bypasses the retry quota, so departure/privacy cleanup remains possible. The no-allocation `DisableSharing` safety operation also remains available after quota exhaustion, through admin form/HTTP/CLI (`unshare`). This operational retention policy remains a host integration gate.

Paging is a **fresh keyset view**, not an immutable historical listing snapshot. Concurrent inserts/deletions can change later pages; every page rechecks visibility and bounds work. Rotate a cursor key by expiring outstanding cursors, then loading a new 32-byte key. No secret is persisted by this module.

## Local contract and CLI

See `CONTRACT.md`, `OPERATIONS.md`, `ACCEPTANCE.md` and `VALIDATION.md`. `RunPagedCLI` supplies the same authenticated service and opaque listing paging as HTTP; no second login/database path exists. `RunCLI` is the unpaged compatibility seam, whose bulk `filter` deliberately refuses over 1,000 references.

## Qualification and integration gates

Independent fresh-agent audit and fixes are recorded in AUDIT.md. Actual Circula principal/store adapter, root API/CLI/help registration, full ledger regression, media EXIF acceptance, network opt-in, deployment cookie/CSRF review, browser keyboard/zoom/screen-reader checks, and release qualification remain pending. The standalone tests are not proof that those gates pass.

Keep this branch independent. Qualify core separately, then obtain the owner's selection before integrating **one** feature. No merge, auto-merge, deployment or release follows from passing module tests. Disable by removing host registration; preserve feature/audit history and ledger data according to approved retention policy.

## Master continuation

Current branch: `codex/future-areas-master-20261116`, recovered from `a5f28203a5aa389f81f4e9aad1ab97d612bef74d`. Current strict query/callback correction is documented in SECURITY-AUDIT.md and INTEGRATION.md. COMPLETION.md records standard weighted criteria; VALIDATION.md records the current delta separately from inherited tests.
