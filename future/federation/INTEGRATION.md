# Federation integration and adapter contract

**Disabled; integration HOLD. No production registration or core integration is authorized.**

Construct peer Ingress separately from member/admin HTTP adapters. The host supplies configured local/remote network identities, caller-owned database, current transaction authorization/projection callbacks and explicit signing/delivery providers. No workers auto-start. No financial authority, balances or cross-network settlement exists.

## Current HTTP contract

Members accepts GET/POST with only optional single `after_peer`, `after_listing`, `limit` query keys. Query bytes are capped at 8192 before host/form callbacks; malformed encodings, duplicate/unknown fields and invalid identifiers/limits reject. Single empty cursors preserve first-page compatibility. Native forms post to the same URL, so valid pagination remains permitted on POST while all mutation fields come from PostForm. Admin URLs are query-free.

Forms require application/x-www-form-urlencoded, a 64KiB body cap, one value per allowed body key and no unknown fields. Oversized bodies return 413; malformed/ambiguous forms return 400. HostAuth validates current identity and cookie CSRF; query data cannot supply CSRF/action/peer/listing fields. CSRF token callbacks run before a fresh same-actor lookup and private transactional read. HostAuth must not allocate a new identity on each call.

## Deferred host work

Map authorization/read/write callbacks into the actual Circula service transaction rather than nesting an immediate transaction. Couple withdrawal to departure, suspension and source deletion atomically. Qualify existing ledger invariants, actual API/rate/CSRF/logging conventions and production key lifetimes. These are separate integration gates, not completed adapters.

## Disable, recovery and retention

Keep platform/network enablement off and omit routes/workers. Explicit candidate migration creates only discovery_* schema in a supplied database. No root migration is registered. Preserve replay/sequence/tombstone/request metadata through backup/restore; never erase safety metadata to recover capacity. Lifetime quotas deliberately stop new work. A peer-epoch/compaction upgrade needs separately reviewed protocol semantics.

See ACCEPTANCE.md and ADR-0001.md for operational contracts. Partitioned peers may retain earlier public projections until expiry; an in-flight dispatch cannot be recalled. Live external-peer interoperability, manual browser/accessibility and key-custody/protocol acceptance remain open.
