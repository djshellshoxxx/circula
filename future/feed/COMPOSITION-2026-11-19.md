# Isolated feed composition on the settled core

This candidate composes reviewed feed development with an exact core checkpoint. The feed remains `enabled: false` and `integration: HOLD`. It adds no core import, route, source adapter, credential change, schema migration, or activation switch.

## Preserved sources

| Source | Exact commit | Scope |
|---|---|---|
| Current isolated feed PR #92 | `0fed0480d3b129351c96cc0f62bd19479acceab7` | Original feed history and prior core refreshes |
| Reviewed feed completion PR #98 | `a8aecfe8782df4782af256bbcecdf5fd1469f783` | Bounded stored pagination, authenticated cursors, regression/integration tests, workflow and recorded evidence |
| Settled core PR #90 | `992478594e5902f57a39f3772739bee6ab495d24` | Exact tree `3725d720ce7fc41f148b560e5f06c8f9bf5ab221`; core migrations 0001–0021 |

The local work uses native merges. PR #98 merged into the current PR #92 source without a conflict. The subsequent core merge conflicted only in generated `docs/TRACEABILITY.md`: the base-head label and overlapping pagination evidence rows. Regeneration from the complete composed source resolves both sections without dropping either source's citations. Publication preserves all three remote source commits as parents, with PR #92's current head first for a normal, nonforce update.

All 34 existing PR #98 feed-module and dedicated-workflow files are byte-identical in this composition, including runtime code, tests, manifests, dependency lockfile, evidence, timeout settings and disabled/HOLD metadata. The five security-reviewed cursor implementation/test files also match the independently reviewed `949d32907f8cc2fab30ce237d95bb361023e84ba` source exactly. The only additional module file is this composition note.

Of the settled core's 1,470 leaf files, all 1,469 other than generated traceability are unchanged. In particular, the root module manifests, all 21 core migrations, session/token authority, bounded session reads and bulk revocation, export authorization, UUID job IDs, and SQLite transaction/migration/verifier repairs remain the published core's bytes. The feed's SQLite harness remains a separate nested test module; it does not add a dependency or table to the root application.

## Fresh composition checks

- Feed Go source formatting passes without edits.
- `go list -mod=readonly -m all` reports only `github.com/djshellshoxxx/circula/future/feed` for the runtime module.
- The unchanged dedicated workflow parses correctly, retains pinned actions and read-only repository permissions, and retains its reviewed 25-minute job / 12-minute SQLite race limits.
- The manifest still requires disabled/HOLD isolation.
- Root traceability generation and freshness pass: 531 traceability IDs, 15 built rows without test citations, no unknown or duplicate IDs, and 895 of 906 legacy citations resolving. The independent module's existing mapping remains unchanged.
- The composition diff against either feed parent is whitespace-clean. An unfiltered diff against core reports two existing Markdown hard line breaks in the frozen `SPEC-SNAPSHOT.md` (lines 3–4); its blob is identical in both feed parents and is preserved. The rest of the feature delta passes the whitespace check.

These are new source/static/composition checks. No module race/fuzz suite or cold SQLite compilation was repeated for this merge, because the reviewed runtime, test and dependency blobs are unchanged. No current composed-head runtime pass is claimed.

## Revision-specific qualification

| Evidence | Revision and observed scope | Meaning for this candidate |
|---|---|---|
| PR #92 dedicated provider run `44507142488`, job `135501587428` | Exact `0fed0480`; completed successfully on 2026-11-19. Race regression test 1.226 s, 98.8% coverage, and 100,000 signed-cursor fuzz executions | Historical qualification of the smaller starting module; it does not qualify the later stored-pagination/v3/SQLite additions |
| PR #92 core run `44507142398` | At the recorded 2026-11-19 11:44:13 UTC snapshot, core fuzz had passed and the broader run remained incomplete | Historical core-baseline receipt only |
| PR #98 exact-source evidence | The checked-in `COMPLETION-2026-11-19.md` and `evidence/` record author-observed module, property/fuzz and full real-SQLite ordinary/race/benchmark results, with runtime/dependency hashes matching reviewed `949d3290`; later `a8aecfe8` changes no runtime, test or dependency blob | Source equivalence is independently checked; those runs remain attributed to their original source/evidence, not a new run by this composition worker |
| Settled core run `44516848989` | Exact `99247859`; nine configured jobs were queued at the same recorded snapshot | Core qualification remains owned by the primary integration process |
| Newly composed publication | Not yet assigned a provider run when this note was written | Require exact published-head core and expanded dedicated module CI before merging the isolated candidate |

The earlier security review approved cursor framing, authenticated context binding, expiry, canonical encodings, key copying and fail-closed behavior at the identical reviewed source. The separate module review approved bounded queries, paging semantics, test coverage and isolation. Neither approval authorizes production feed integration.

## Remaining integration boundary

The held work in `INTEGRATION-PROPOSAL.md` remains necessary: an authoritative transaction-scoped source/permission adapter and atomic projection; separately reviewed schema, retention and restore invalidation; production key custody, rotation/issuance limits and TTL; preference/mute/follow policy; and separately reviewed routes, rate limits, CLI and browser/no-JavaScript behavior. This composition identifies the core baseline; it does not complete or waive those gates. No original source branch is rewritten, no release is published, and no historical data is changed by this refresh.
