# FNET-209: Opt-in discovery federation

**Disabled; integration HOLD. This standalone candidate is not approved for Core integration, activation or release.**

This independent module contains the preserved v1 envelope helpers and the recovered experimental v2 discovery service. It has no balance, ledger-posting or cross-network settlement authority. The owner authorized isolated future-feature development; selecting and enabling this feature in Core remains a separate decision.

## Implemented standalone behavior

- Legacy domain-separated Ed25519 v1 envelopes retain their existing API and the malformed-string signing repair. The v1 framing remains separate from v2 ingress.
- Experimental v2 uses compact JWS/EdDSA with explicit public-content projections, configured bilateral peer trust, validity and revocation checks, signing-material lifecycle, and owner consent.
- SQLite persistence provides transactional replay records, revision ordering, projections, tombstones, bounded outbox entries, leases and retry state. Withdrawal intents survive owner departure or unavailable signing material.
- Explicit maintenance, departure and source-deletion hooks preserve replay and sequence metadata. Lifetime quotas stop new work rather than silently evicting that metadata.
- Explicit HTTPS delivery uses configured endpoints, bounded resources and validated addresses. Network I/O takes place outside the SQLite writer transaction. Independent-peer interoperability is still an acceptance gate.
- Unregistered member/admin HTTP adapters provide native forms and a CLI seam. Current authorization, CSRF, source visibility and transactional host callbacks remain required.

See [INTEGRATION.md](INTEGRATION.md), [ADR-0001.md](ADR-0001.md) and [ACCEPTANCE.md](ACCEPTANCE.md) for the adopted contracts and limits. These implemented seams do not supply the actual Circula host adapter.

## Isolation and dependencies

Module implementation stays under `future/federation/`. The dedicated [future-federation workflow](../../.github/workflows/future-federation.yml) performs standalone qualification; it does not activate the feature. The recovered branch also retains the separately accepted Core baseline repairs identified in [the composition receipt](COMPOSITION-20261120.md). Those repairs introduce no federation imports, routes, worker registration or root migration.

This nested Go module is excluded from the root module's ordinary `go test ./...` and build traversal. It has no dependency on another future module. Its pinned dependencies include `github.com/go-jose/go-jose/v4` v4.1.9 and the standalone SQLite fixture driver `modernc.org/sqlite` v1.60.1, with the recorded indirect dependencies. See [go.mod](go.mod), [go.sum](go.sum) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Importing the module registers no route, opens no database and starts no worker. The host must explicitly supply a database, configured network identities, current transaction authorization/projection callbacks, and signing/delivery providers. The candidate migration installs only feature-owned `discovery_*` schema in that supplied database. No root migration is registered. Client fields cannot substitute for authenticated host authority.

Keep platform/network enablement off and omit route and worker construction. A later approved adapter must share the existing Core transaction for authority, source changes and departure/deletion handling, rather than nesting a separate immediate transaction. Disablement must preserve replay, sequence and tombstone metadata; see [INTEGRATION.md](INTEGRATION.md).

## Source history and qualification records

[RECOVERY-20261119.md](RECOVERY-20261119.md) records the verified archive, the signing repair and the recovered implementation. [SIGNING-VALIDATION-20261119.md](SIGNING-VALIDATION-20261119.md) preserves the precise v1 repair boundary. The archive's original README remains at [RECOVERED-README-20261116.md](RECOVERED-README-20261116.md).

[COMPOSITION-20261120.md](COMPOSITION-20261120.md) records the normal merge preserving PR #106's standalone-workflow acknowledgement, reconciliation of this README, and the intended direct route to `future/fnet-209-federation-20261116`. Earlier ownership, pending-provider and completion fields in [feature.json](feature.json) and the original receipts are retained historical records. The composition receipt records the newer local handoff without altering those originals; it does not claim a remote merge or fresh provider pass.

The dedicated workflow records the checked source and toolchain, verifies disabled/HOLD and dependency boundaries, and checks checksums, formatting and vet. Linux and macOS run the full race suite; Windows runs the full ordinary suite. Linux also runs the three bounded fuzz targets. The workflow's original conditions and assertions are preserved.

The earlier full real-driver race and bounded fuzz observations are documented in [RECOVERY-QUALIFICATION-20261119.md](RECOVERY-QUALIFICATION-20261119.md) with source identities and raw evidence. They remain historical observations. The final composed publication must pass its own applicable Core jobs and native module jobs. Conditional Core native jobs that do not execute must be reported as skipped.

## Remaining integration and acceptance gates

1. Implement and qualify actual transaction-bound Core membership, source, departure and deletion adapters; current session/CSRF and rate/logging conventions; and the full Core invariant regression suite.
2. Settle final discovery-protocol and bilateral-peer policy, production custody/distribution/rotation, integrated rate policy, and the operational lifetime-capacity strategy. Any epoch or compaction change requires reviewed protocol semantics; clearing safety metadata is not a recovery procedure.
3. Qualify live independent-peer protocol/TLS interoperability, real browser/keyboard/zoom/screen-reader behavior, and reviewed deployment, backup/restore and operating procedures. Partitioned peers can retain previously published projections until expiry, and an in-flight dispatch cannot be recalled.
4. Obtain separate selection of this feature for a narrowly scoped Core integration with explicit per-network opt-in. Preserve the independent development paths of other future modules.

Cross-network settlement is excluded from discovery-only completion and requires a separate accounting design. Nothing in the standalone implementation or its historical test results establishes Core release readiness.
