# Future Core integration proposal — FNET-204

This document proposes future integration work only. No Core code, authentication, session handling, migrations, routes, or source-service behavior is changed by this branch. The feed remains disabled and on integration HOLD.

## Inspected baselines

- Starting feed candidate: PR #92 at `8638f3d30f27f4882c4c6424d4c8b248208b6f3d`.
- Observed `main` at discovery: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.
- Runtime module: `github.com/djshellshoxxx/circula/future/feed`, Go 1.25 minimum, standard library only.
- Qualification toolchain: Go 1.26.11, matching the starting repository pin.
- Separate integration harness: the existing Core SQLite driver, `modernc.org/sqlite v1.62.0`; this adds no dependency to the root application or runtime feed module.

These observations establish a structural comparison, not qualification of a moving Core candidate. API-036 remains owned by the primary process. No session branch is inspected for modification or used as a new merge parent. Recheck the final published Core candidate before implementing this proposal.

## Data-contract mapping

| Feed contract | Existing Core seam | Required integration decision/check |
|---|---|---|
| Authenticated network and viewer | `internal/service` request/actor boundaries | Obtain identifiers from existing authentication and network selection; never trust client-supplied identity claims |
| Listing reference | `internal/service/listings.go`, domain listing ID/network/member/creation/status | Map committed source IDs and immutable creation time; only public-to-network active sources are feed candidates |
| Announcement reference | `internal/service/announcements.go`, announcement ID/network/creator/published/effective/retracted state | Choose immutable feed event time; scheduling and supersession must be checked against current authoritative rows |
| Current listing permission | Existing `ListListings` service behavior | Active members and active owners only for public listings; owner-only paused/archived views do not become public feed visibility |
| Current announcement permission | Existing `ListAnnouncements` service behavior | Announcements currently allow active and suspended members; do not replace this with a blanket listing-membership rule |
| Categories and preference revision | Proposed feed preference adapter | Supported source kinds are listing/announcement; choose trusted revisions and mute/follow semantics before new routes |
| Commit synchronization | Existing repository/service transactions | Project references atomically from committed source changes or a durable, ordered replay mechanism; an asynchronous stale projection must not replace authoritative authorization |
| Module storage | Proposed `fnet204_sources` schema | Review additive Core migration, transaction ownership, backup/restore, tombstone retention, and connection settings separately |
| Cursor keys | Existing deployment secret-management boundary | Persist strong keys, define key IDs/overlap/cadence, set a GCM issuance-volume limit, and confirm TTL; do not modify session credentials to do this |

`SQLStore.Upsert` requires a nonempty `Source.Member`. The integration adapter must define an explicit mapping for any system-created source without a member; it must not invent a viewer or change Core authorization semantics.

## Authorization freshness

The callback to `PageFromStore` must check current authoritative source data and the authenticated viewer's permissions. Cached `Visible`/`Active` flags are a query accelerator only. If source state changes after the database batch is read, the authorization check must still reject deleted, hidden, expired, superseded, suspended-owner, or unauthorized sources. A transaction-scoped adapter should provide a coherent request view. The standalone SQL store demonstrates reference durability and bounded retrieval; it is not that Core service adapter.

The insertion fence excludes new committed source identities after traversal start. Existing-source visibility remains live. Grants or visibility changes behind the cursor do not cause rewind; a first-page refresh is required. Restore/rebuild operations that reset insertion sequences must invalidate existing v3 cursors by retiring their verification keys or changing a separately reviewed trusted binding revision.

## Integration gates still deferred

1. Select a stable Core consolidation/API-036 candidate and review interface/schema drift against its exact SHA.
2. Review the authoritative source adapter, atomic projection/replay, preference semantics, schema adoption, key custody, retention, and backup/restore behavior.
3. Add separately reviewed routes, API contracts, rate limits, CLI/help, and keyboard/no-JavaScript pagination. Clients must follow nonempty `Next` even on an empty page.
4. Recheck Core authentication, authorization, tenant isolation, CSRF, no-store responses, errors, ledger regression, and browser/accessibility behavior through those real entry points.
5. Obtain separate owner authorization for production integration. This isolated branch cannot activate the feed or publish a release.
