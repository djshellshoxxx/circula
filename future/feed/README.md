# FNET-204: isolated feed pagination and cursor security

**Disabled. Integration HOLD. No Core routes, imports, migrations, or registration.**

This nested Go module implements listing/announcement reference projection, bounded SQLite retrieval, and authenticated pagination. The current completion effort covers the feed pagination/security task. The broader community-feed roadmap in `SPEC-SNAPSHOT.md` still requires separately authorized Core integration, product decisions, preferences, and UI work.

## Use the correct retrieval path

| API | Purpose | Resource bound | Cursor |
|---|---|---|---|
| `Page(codec, query, rows, allowed)` | Compatibility helper over an already-loaded trusted snapshot | At most 10,000 supplied rows; scans and sorts that bounded snapshot | Existing signed v2 positions |
| `PageFromStore(ctx, codec, query, store, allowed)` | Ordinary pagination over a persisted history | At most 1,000 candidate rows per request, at most 101 per SQL query, at most 100 returned references | Encrypted/authenticated v3 traversal state |
| `SQLStore.QueryBatch(ctx, request)` | Storage adapter primitive; not a user-facing authorization API | Indexed keyset seek plus SQL `LIMIT`, 1–101 rows | Decoded internal position only |

`PageFromStore` is the large-history path. Do not load an entire database history and pass it to `Page`. Neither API authenticates a caller. Network/viewer identifiers and source snapshots must come from trusted adapters. `allowed` must recheck current membership, source visibility, deletion, suspension, scheduling, and applicable preferences. A callback error discards the entire page and returns a uniform error.

### Bounded traversal contract

1. The first request captures the network's maximum committed insertion sequence.
2. Results use a total order: creation time descending, then kind and source ID ascending.
3. Source identity, insertion sequence, and creation time are immutable. Updates can change visibility/active state and owner metadata. Deletes retain a tombstone; stale creates cannot resurrect a deleted identity.
4. Every continuation carries the original insertion fence and original issue time. New events, including backdated inserts, stay outside that traversal.
5. Current visibility and permissions still apply. Removed/revoked events disappear. A permission grant or visibility change can affect an existing event only if its position has not already been traversed; refresh to reconsider earlier positions. This is an insertion-fenced traversal with live revocation, not a historical permissions snapshot.
6. SQL limits candidate reads **before** checking the insertion fence or per-viewer callback. Otherwise a flood of newer/backdated rows or sparse permissions could force an unbounded scan.
7. A request that reaches its 1,000-candidate budget can return fewer than `Limit` items, including zero, with a nonempty `Next`. Follow `Next` until it is empty. An extra empty terminal request is possible at an exact scan-budget boundary. Cursors always advance, and a lookahead row is never consumed prematurely.
8. Cursor expiry requires restarting from the first page. Continuations do not extend the traversal lifetime.

No private trade, balance, profile, or event-body content is stored or emitted by this package. Returned `Source` values are references; a future adapter must retrieve permitted content through existing service authorization.

## Cursor contracts

### Existing signed v2 cursors

`Seal`/`Open` and `Page` preserve the previously emitted v2 wire format. A golden fixture verifies compatibility. The format is `<key-id>.<base64url JSON>.<base64url HMAC-SHA256>`. The MAC binds a domain label, key ID, network, viewer, canonical category digest, preference revision, and payload using length-prefixed fields. The visible payload contains only the previously returned item position and issue time. Canonical JSON and strict base64url are required. Noncanonical payloads were never emitted by the codec and are now refused.

### Encrypted v3 traversal cursors

`SealSnapshot`/`OpenSnapshot` and `PageFromStore` use `3.<key-id>.<base64url nonce+ciphertext+tag>`. AES-256-GCM authenticates and encrypts the scan position, insertion fence, and original issue time. A separate HMAC-derived encryption key and associated-data label separate this protocol from v2 and from Core cursors. Associated data binds key ID, network, viewer, canonical categories, and preference revision.

The encrypted plaintext is always 1,024 bytes: a two-byte JSON length, canonical JSON, and zero padding. The encrypted body is always 1,052 bytes. This conceals hidden frontier kind/ID lengths and insertion-counter magnitude. Only the version and key ID remain visible. Empty continuation counts can still reveal coarse activity volume; cursor encryption does not provide traffic-analysis or count privacy. Nonces use the cryptographic random source. Retire an encryption key before 2^32 total v3 cursor issuances across all instances sharing that key and key ID; choose a lower operational limit. Retain verification keys for the chosen TTL plus clock allowance after their last issuance. Production custody, issuance accounting, and rotation cadence remain integration decisions.

A v2 token cannot be upgraded into an insertion fence and is refused by `OpenSnapshot`. Existing v2 consumers continue using `Page`; switching to the database API starts a new traversal. v3 tokens are refused by `Open`.

### Shared validation and rotation

- Tokens are limited to 2,048 bytes, with strict unpadded base64url and bounded field sizes.
- Key IDs are 1–32 URL-safe characters; keys are at least 32 bytes. `KeyRing` copies keys both on construction and on retrieval.
- Configurable TTL is one second through 24 hours; zero selects the provisional one-hour default. A one-minute future-clock allowance applies. v3 preserves subsecond issue time and the original traversal lifetime.
- Keep the old verification key during the overlap period; sign new continuations with the current key. Removing a key invalidates its tokens. Reusing a key ID for different key material invalidates old tokens under that ID.
- Unknown, malformed, forged, cross-context, expired, and retired-key tokens all return `ErrInvalidCursor` and zero state. Unconfigured/typed-nil codecs fail closed.
- Changing `Filter.PreferenceRevision` invalidates outstanding tokens. Its value must come from a trusted preference adapter, not an untrusted claim.

## Isolated SQLite store

`NewSQLStore` wraps a caller-owned `*sql.DB`; it opens no connection and performs no schema work. `InitSchema` explicitly creates only `fnet204_sources` and its indexes. This is a module-local schema, not a Core migration.

The schema stores unique `(network, kind, source_id)` references and immutable insertion sequences. Negated Unix nanoseconds make every index key ascend, allowing tuple seeks through arbitrarily large timestamp ties. Separate partial indexes serve combined-kind and single-kind queries; a network/sequence index supplies the insertion fence. Reads use parameterized statements and SQL limits. Tombstones persist across restart. Source creation time changes and resurrection attempts are rejected. Database diagnostics are not returned to readers.

The runtime module uses only Go's standard library (`database/sql` does not register a driver). The separate `integration/` test module pins the same SQLite driver as the inspected Core baseline. Production connection ownership, busy timeouts, transaction synchronization with authoritative sources, tombstone retention, backup/restore, schema adoption, and current authorization joins require the separate integration proposal.

## Validation

Run from `future/feed` with the repository's pinned Go toolchain and a race-capable C compiler:

```sh
go test -count=1 -cover ./...
go test -race -count=1 ./...
go vet ./...
go test -run '^$' -fuzz '^FuzzOpenCursor$' -fuzztime=100000x -timeout=3m -parallel=2
go test -run '^$' -fuzz '^FuzzOpenSnapshotCursor$' -fuzztime=100000x -timeout=3m -parallel=2
go test -run '^$' -fuzz '^FuzzPagePermutation$' -fuzztime=10000x -timeout=3m -parallel=2
cd integration
go mod download
go mod verify
go vet ./...
go test -race -count=1 -timeout=8m -v ./...
go test -run '^$' -bench . -benchmem -benchtime=100x -count=1 ./...
```

The dedicated `.github/workflows/future-feed.yml` runs both nested modules. Root `go test ./...` does not cover either one. Synthetic fixtures cover 10,000/100,000 histories, timestamp ties, deletion/restart, concurrent readers/writers, real query plans, and full cursor traversal. See `COMPLETION-2026-11-19.md` for candidate-specific results and `TRACEABILITY.md` for the acceptance mapping. `VALIDATION.md` preserves historical results with their original attribution.

## History and integration boundary

Starting candidate: PR #92, `8638f3d30f27f4882c4c6424d4c8b248208b6f3d`. This already preserves the original feed branch and PRs #67/#80, together with historical Core audit merges. New work stays in `future/feed/` plus its dedicated qualification workflow. Original branches are preserved, and no primary API-036 branch is merged, rebased, rewritten, or modified.

The module stays `enabled:false` and `integration:HOLD`. See `INTEGRATION-PROPOSAL.md` for the inspected interfaces and outstanding Core dependencies. Key custody, rotation cadence/volume limits, final TTL, preference semantics, source joins, browser/CLI/no-JS UI, and release approval are not inferred from isolated module tests.
