# FNET-204: Community activity feed

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Original branch: `future/fnet-204-feed-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`. The consolidation branch `audit/future-feed-cursors-20261119` preserves PR #67 (`7a528c61`) and PR #80 (`0952c94a`), then repairs conflicting source snapshots and unconfigured cursor codecs.
- The consolidation also merges core audit candidate PR #90 at `9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c`. Core qualification remains a dependency; this refresh does not establish that the candidate passed its release gates.
- It includes the reviewed monetary-guard follow-up `d627af48ff4042db73adac9a72f6208db268a924`, which allows exact `big.Int` aggregation while still rejecting floating-point and rational money arithmetic. This is a core regression-test dependency, not a feed behavior change.
- The final refresh merges published core candidate `4140bc135479077f6297d53ed2e9dfcd85c056d6`, including the completed setup, catalogue and account-session work. The feed's runtime source is unchanged by that merge. Its dedicated fuzz gate uses 100,000 executions with an independent three-minute timeout; provider execution remains a separate qualification result.
- Runtime changes are confined to `future/feed/`. The dedicated `.github/workflows/future-feed.yml` validates this disabled nested module on relevant pushes and pull requests. Application imports, routes, migrations, root dependencies, feature flags, core workflows and runtime configuration remain unchanged.
- This nested Go module has no third-party dependencies and no dependency on another future module. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution, DB connection, external service or ledger write is present.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- No production persistence is provided. Pure preflight checks do not guarantee atomicity, idempotency, authorization freshness or restart durability; future adapters must supply these within existing service/store transactions.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied.

## Implemented milestone

Listing/announcement-only projection, read-time permission callback, source deduplication and stable keyset ordering, and signed, viewer-bound pagination cursors wired into `Page()`.

## Signed cursor contract

`Page(codec, Query{Network, Viewer, Filter, Limit, Cursor}, rows, allowed)` takes the previous page's `Result.Next` as an opaque token and returns the next one. There is no unsigned path: the raw keyset position (`Cursor`) is internal, and `Page` fails closed without a codec or authorization callback. The design follows `internal/api/cursor.go` (API-034, API-035) without importing or changing it.

- **Format (v2):** `<key id>.<base64url payload>.<base64url HMAC-SHA256>`. Strict base64url only; any other character, padding, line break or non-canonical encoding is refused. Tokens are at most 2,048 bytes.
- **Binding:** the MAC covers network, viewer (actor) and a canonical filter digest (SHA-256 over the sorted, de-duplicated kinds plus `Filter.PreferenceRevision`). A cursor from member A fails for member B, from network X for network Y, and from filter X for filter Y. Filters that select the same items (`nil` and both kinds; reordered or repeated kinds) share a digest. `Limit` is not bound.
- **Readable metadata:** network, viewer and filter are **not** in the token; they enter only the MAC input. The token reveals the key ID, the issue time and the position (kind, ID and time of the last item on the page the viewer was just shown). Hiding the position would need encryption or server-side state, and it tells the holder nothing they were not shown.
- **Domain separation:** the MAC input is length-prefixed `label, key id, network, viewer, filter digest, payload` with label `circula/future/feed-cursor/v2`. Core cursors MAC raw JSON, so a feed MAC can never be a core MAC even under one shared secret, and field boundaries cannot be shifted.
- **Expiry:** the payload carries the issue time (`iat`, Unix seconds) from an injected clock. A cursor opens while `iat - 1 min <= now < iat + TTL`. TTL is `CodecOptions.TTL` (1 s to 24 h); zero selects `DefaultCursorTTL` = **1 hour, provisional, owner to confirm**. The verifier's TTL applies to cursors already issued. The 1-minute allowance tolerates a verifier clock behind the signer's. The row time (`t`) is only the position.
- **Key ID and rotation:** `KeySet` signs with `SigningKey()` and verifies with `VerificationKey(id)`, so a rotated-out key keeps verifying while it is in the set. `NewKeyRing` is an immutable in-memory implementation (keys at least 32 bytes, IDs `[A-Za-z0-9_-]{1,32}`, keys copied). The key ID is MAC input, so relabelling to an aliased key fails. An unknown key ID still costs one MAC under a placeholder key.
- **Uniform refusal:** `Open` returns exactly `ErrInvalidCursor` for every failure (malformed, tampered, unknown key, expired, wrong binding), with a zero `Cursor`. The MAC is compared with `hmac.Equal` before the payload is parsed. A nil or zero-value codec refuses operations without panicking; construct a usable codec with `NewCursorCodec`. `Page` returns `ErrInvalidQuery` for a malformed query and `ErrInvalidSource` for an adapter row that cannot be paged.
- **Duplicate snapshots:** identical `(kind, ID)` references are de-duplicated. Conflicting creation time, owner, visibility or active state fails with `ErrInvalidSource`, regardless of input order. This prevents a stale visible duplicate from winning over a hidden or inactive snapshot. Rows from other networks do not affect the selected network. The caller must still supply a current, authorized snapshot from an appropriate transaction; this validation does not add persistence.

## Owner decisions (not made here)

These are documented, not decided. The module supplies mechanisms only.

1. **Signing-key custody and persistence:** where keys live (secret store, file, environment), how they survive restarts and who can read them. A future adapter implements `KeySet`.
2. **Rotation cadence and retirement:** how often the signing key changes and how long a retired key stays verifiable. The mechanism requires at least one TTL of overlap.
3. **Cursor TTL value:** `DefaultCursorTTL` = 1 h is provisional. An expired cursor means the reader restarts at the first page. The 1-minute clock allowance is a fixed constant and could be confirmed with it.
4. **Mute/follow/category semantics:** what `Filter.PreferenceRevision` covers, its retention, and moderation. Changing it invalidates outstanding cursors by design.
5. **Persistent source joins:** the core schema, adapter and high-water snapshot anchor for durable feed items. A module-local SQL draft is possible; a core schema or adapter change needs owner approval.

## Existing application seams inspected

`internal/service/announcements.go; internal/service/listings.go`. A future adapter must map these through the existing service authorization boundary. No direct import or code change is made to these seams. APIs here are provisional until that adapter is qualified.

## Remaining feature work

Database snapshot and source joins (with a high-water mark bound into the cursor); preferences/mute/follow; committed source IDs; feed-item expiry/moderation; no-JS paging, browser/CLI harness and deletion/revocation race tests against a real store. No trade events. Signed cursors are done (see above); key custody, rotation cadence and TTL await the owner.

## Standalone validation

```sh
cd future/feed
gofmt -l .                       # must print nothing
go vet ./...
go test -count=1 -cover ./...
CGO_ENABLED=1 go test -race -count=1 ./...
go test -run='^$' -fuzz=FuzzOpenCursor -fuzztime=60s
```

Tests use synthetic fixtures only. See `VALIDATION.md` for observed results. These tests qualify the stated core milestone, not the full roadmap feature, production behavior, browser accessibility, or Circula's release.

## Later integration, one feature at a time

1. Finish this module's remaining scope and acceptance tests. Audit its own code and record unresolved findings.
2. Qualify the then-current Circula core separately. Do not infer completion from this module's tests.
3. Rebase this branch on the qualified core; review the baseline diff and any API/schema drift. No promise of a conflict-free future merge is made.
4. Add a narrow adapter, reviewed API contracts, additive migrations (if needed), explicit per-network opt-in, CLI and accessible/no-JS UI in a separate integration change. Keep all other future features absent.
5. Recheck tenant/auth/CSRF/idempotency behavior, concurrency/restart, backup/restore, full ledger regression and relevant manual/browser acceptance.
6. Obtain the owner's explicit selection of this feature for integration. Only then merge that single integration change. Never auto-merge this hold branch or batch-enable the backlog.
7. Disable by removing routes/jobs and adapter registration; do not roll back ledger history. Any feature-data retention/migration procedure must be documented before integration.

No work on this branch changes Circula's core completion or release status.
