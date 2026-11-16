# FNET-204 upstream research and implementation handoff

Research phase: 2026-11-16. **Research only; feature remains incomplete and integration HOLD remains in force.** A fresh agent performs adaptation, followed by another fresh audit agent. This report does not authorize enabling routes or merging into Circula.

## Decision

Adapt the narrowly scoped **mixed-direction keyset pagination implementation and tests from `pilagod/gorm-cursor-paginator` v2.7.0**, retaining its MIT notice. Rewrite the database adapter around `database/sql`/Circula's existing SQLite store; do not import GORM, its database drivers, reflection layer, or its unsigned cursor wire format.

Use Circula's existing HMAC cursor contract as the production adapter contract. Build the feed-specific authorized source projection, private preferences, retention and HTML using original code because neither upstream feed/message application supplies Circula's authorization rules. Reuse is intentionally concentrated on the reusable pagination mechanism rather than copying a foreign application's private-message storage/authentication.

No upstream source has been copied during this phase. The implementation agent must create an attribution file when adaptation begins.

## Candidates checked against official repositories

| Candidate | Exact pin examined | License inspected | Maintenance evidence and suitability |
|---|---|---|---|
| [pilagod/gorm-cursor-paginator](https://github.com/pilagod/gorm-cursor-paginator) | v2.7.0 annotated tag object `6bbfb84543ecc7abcbe892816eac7b1fcadd6c45`, resolved commit `d077284390d037a0a50fd6f7f3da470858a0b68b` | [MIT](https://github.com/pilagod/gorm-cursor-paginator/blob/d077284390d037a0a50fd6f7f3da470858a0b68b/LICENSE), copyright Cyan Ho | HEAD/release commit dated 2025-05-23. Longstanding project with extensive composite-key, direction, limit, join and codec tests. Mature algorithm; no recent maintenance after that date was observed. **Selected for a small attributed source port**, not an ORM dependency. |
| [gotify/server](https://github.com/gotify/server) | HEAD `01086b95a8b69ee87383f6e8bcae601d8c33384b`, 2026-11-09 | [MIT code license](https://github.com/gotify/server/blob/01086b95a8b69ee87383f6e8bcae601d8c33384b/LICENSE); logo has separate CC BY 4.0 terms | Active maintenance observed. Go message history, ownership joins and paging tests are useful references. Its stored private message payloads, Gin/GORM application and token identities are **not** a Circula community-feed model. No wholesale port. No logo/assets copied. |
| [gorilla/securecookie](https://github.com/gorilla/securecookie) | v1.1.2 / `eae3c1840ec4adda88a4af683ad0f60bb690e7c2`, 2023-10-18 | [BSD-3-Clause-style text](https://github.com/gorilla/securecookie/blob/eae3c1840ec4adda88a4af683ad0f60bb690e7c2/LICENSE), Gorilla Authors | Mature signed serialization, size/age limits and key rotation helpers. No later HEAD change observed. **Not selected**: Circula already has route/network/caller/filter-bound HMAC cursors, and securecookie's timestamped cookie wire format would be a separate incompatible format. Its default Gob serializer is also unnecessary here. |

Evidence method: GitHub repository searches via the connected GitHub tool, shallow clones of the official repositories, `git log -1`, full root license inspection, source/test inspection, and `git ls-remote --tags` to resolve release tags. Repository age/activity is evidence of maturity, not a security certification. No advisory scan or full upstream test execution was performed in this research phase.

## Exact code and tests to adapt

### Selected paginator source

Pinned [paginator/paginator.go](https://github.com/pilagod/gorm-cursor-paginator/blob/d077284390d037a0a50fd6f7f3da470858a0b68b/paginator/paginator.go):

- `buildCursorSQLQuery`: lexicographic disjunction over ordered cursor fields.
- `buildCursorSQLQueryArgs`: repeats the prefix arguments for each disjunct.
- `getCmpOperator`: chooses comparison direction from paging direction/order.
- `appendPagingQuery`: `limit + 1` lookahead and application of ordering/predicate.
- The `canOptimizePagingQuery` constraint is important: tuple comparison is valid only if all fields use the same direction. The feed prototype sorts time descending but kind/ID ascending; a naive tuple `<` comparison is wrong.

Adapt to a fixed, reviewed list of feed columns, with parameterized **values**. Never let client text become a column identifier or SQL expression. Do not retain upstream reflection/GORM dependencies merely to save a few lines. The current mixed-direction predicate should be equivalent to:

`event_time < ? OR (event_time = ? AND source_kind > ?) OR (event_time = ? AND source_kind = ? AND event_id > ?)`

Use a unique committed event ID as the final tie-breaker. Retain a bounded forward-only API unless a concrete requirement justifies backward paging. Restrict page limits to 1..100 and use lookahead to avoid a false continuation on exact page boundaries.

Pinned [paginator/paginator_paginate_test.go](https://github.com/pilagod/gorm-cursor-paginator/blob/d077284390d037a0a50fd6f7f3da470858a0b68b/paginator/paginator_paginate_test.go): adapt fixtures/assertions from `TestPaginateNoMore`, `TestPaginateSpecialCharacter`, `TestPaginateMultipleKeys`, `TestPaginateMultipleKeysTupleCmpFallback`, `TestPaginateLimit`, `TestPaginateOrderByKey` and `TestPaginateJoinQuery`. Replace their ORM fixtures with the isolated SQLite adapter and add Circula-specific tenant/revocation cases. Keep the original MIT license text in `third_party/paginator-LICENSE.txt` and document which source/tests were adapted, exact SHA and local filenames in `THIRD_PARTY.md`.

### Reference-only Gotify files

- [database/message.go](https://github.com/gotify/server/blob/01086b95a8b69ee87383f6e8bcae601d8c33384b/database/message.go), `GetMessagesByUserSince`: scope in the SQL ownership join before ordering/limit, rather than loading all private rows and filtering afterward.
- [database/message_test.go](https://github.com/gotify/server/blob/01086b95a8b69ee87383f6e8bcae601d8c33384b/database/message_test.go), `TestGetMessagesSince`: separate user/application fixtures and page boundaries.
- [model/paging.go](https://github.com/gotify/server/blob/01086b95a8b69ee87383f6e8bcae601d8c33384b/model/paging.go): explicit next-page representation.

Do not copy Gotify message bodies, application/user tokens, websocket handlers, external push behavior or deletion semantics into Circula's feed. If any nontrivial code/test is actually ported from these files later, retain its MIT notice separately; this report currently uses them only as design references.

## Existing Circula seams and compatibility findings

Inspected at core baseline `2eacf7718f5bb3da3be5a288024af879076a9597`:

- `internal/service/listings.go::ListListings` opens an authenticated read view and requires an active member. Its query sets active listing/owner filters when browsing others; an owner's archived or paused listings must **not** become feed-visible just because that directory path permits owner inspection.
- `internal/service/announcements.go::ListAnnouncements/listAnnouncements` deliberately permits active **and suspended** members to view in-force announcements. A feed can require active membership for its combined listing view, but must not silently change this existing announcement access policy. Suspended members keep their existing announcement route. Make the feed's policy explicit.
- `internal/store/sqlite_f2.go::ListAnnouncements` suppresses retracted, not-yet-effective and superseded announcements. The feed must apply the same current predicates; simply checking `Visible` is insufficient.
- `internal/api/cursor.go` binds HMAC-signed position to route, network, caller and filters. The eventual feed adapter must include category/mute/follow/preference revision and a snapshot high-water mark in that binding where they affect results. Do not expose a raw unsigned feed cursor in production or reuse a notification cursor.
- `internal/api/more_test.go` and `cursor_fuzz_test.go` already exercise tampering, caller/filter reuse and malformed cursor input. Reproduce those guarantees for feed paging.
- Current `future/feed/feed.go::Page` is a bounded in-memory prototype, not a repository. Its dedupe key `(kind, source ID)` conflates repeated updates of one listing with one event. Introduce a unique **committed source-event ID** before materializing update events. Duplicate deliveries of that same event dedupe; distinct permitted update events may remain distinct under the declared feed policy.
- The current model omits announcement effective time, supersession, expiry, moderation, follow/mute preferences and digest state. Full feature completion cannot be inferred from its current test.

## Full-spec isolated implementation plan

Keep every file under `future/feed/`; add no main route, feature flag, root migration or cross-future-module import.

1. **Service and ports.** Define explicit authenticated member/network context, a transactional `Repository` port, and an authoritative `SourceResolver`/read-view port. Return only network-visible listing/announcement projections with escaped title/summary, canonical same-origin link and timestamp. No transaction, balance, private inbox or precise-location fields are accepted in the event union.
2. **Durable projection references.** SQLite tables under a feature namespace: source-event reference (network, event ID, source kind, source ID, event time, expiry/moderation state), private preferences, mute/follow relationships and optional digest checkpoint. Store no duplicated title/body. Unique `(network, event ID)` dedupes committed events. Insert references in the originating transaction or via a durable, replay-safe event handoff; never publish uncommitted sources. A standalone adapter can expose explicit transaction attachment, but cannot pretend it has already been wired to core posting.
3. **Read-time visibility.** Query current membership and current source state in a bounded database view. Exclude paused/archived/deleted listings, inactive owners, future/retracted/superseded announcements and expired/moderated references. Recheck immediately on each page and digest read. Snapshot pagination may anchor insertion history; it must never resurrect revoked content. Do not cache private feed HTML across accounts.
4. **Pagination port.** Adapt selected paginator code with fixed columns, stable immutable event order and page-size lookahead. Anchor the first page to a high-water event sequence; bind that plus member/network/filter/preference version to cursor HMAC. An update creates a new source event instead of moving a previous event's ordering key. Serialize/verify cursors through an injected codec compatible with the production adapter. Fail closed when codec or authorization port is absent.
5. **Preferences.** Owner-only network-scoped categories (listing offers, listing requests, announcements), bounded mute/follow target sets, explicit semantics for follow filtering, digest off by default. A muted target takes precedence over follow. Validate current-network visible targets, avoid self/foreign IDs and never reveal another member's relationships. Persist updates atomically with revision/idempotency and retain no behavioral analytics.
6. **Digest independence.** Optional digest produces a freshly authorized bounded projection, with durable checkpoint/idempotency, and calls an injected delivery port only when explicitly configured/opted in. It must not depend on unfinished FUX-104 or silently send email. Turning it off or removing membership stops future dispatch. Do not retain copied private source text in the checkpoint.
7. **No-JS presentation/API adapters.** Build Go `html/template` rendering with a heading, semantic list, source kind labels, absolute timestamp plus readable display, ordinary next-page links, empty state and standard HTML preference form. Inject CSRF/auth/request-scope validation at the adapter boundary; request context is not client-controlled JSON. Use no-store, bounded inputs, current RFC9457 errors and the existing API envelope. No JS is needed for the basic feed or preference updates. Optional enhancement must not replace accessible links.
8. **Operations and removal.** Add explicit standalone schema/migration apply function with no `init` or automatic core migration. Document index query plans, reference expiry cleanup, limits/rate caps, restore/restart tests and retention. Disabling drops adapter registration/worker scheduling, not core data or ledger history.

Use the root-compatible SQLite driver version if a concrete standalone adapter is added (root pins `modernc.org/sqlite v1.60.1`; verify it and its Go requirement before use). A fresh implementation agent must update module manifests, checksums and notices, rather than importing any global service by accident.

## Acceptance matrix for the next agents

| Requirement/boundary | Evidence required before calling the isolated feature complete |
|---|---|
| Unified feed | Mixed listing offer/request and effective announcement fixtures; newest-first deterministic order; no private trade/inbox fields in JSON/HTML/schema. |
| Current permission | Cross-member/network reads denied; suspension/removal, owner inactivity, listing pause/archive/delete, announcement future/effective/retract/supersede all tested after prior page creation. |
| Committed events/dedupe | Rollback creates no reference; replay creates one reference; distinct source events remain distinguishable; concurrent inserts/restart preserve dedupe and order. |
| Preferences | Owner-only scope, bounded categories/targets, mute overrides follow, revision-bound cursors, cross-network target rejection, account switching and opt-out. |
| Pagination | Empty/exact/partial page, equal timestamps, mixed order, insert-between-pages, deletion-between-pages, unique tie-break, no starvation after hidden candidates, bounded query work. |
| Cursors | Tamper/truncate/oversize/unknown version, other route/member/network/filter/preference revision/key, expiry if adopted, malformed-input fuzz test; constant-time HMAC. |
| Digest | Off by default; no delivery dependency required; authorized current projection only; retries/dedupe, revocation/opt-out race and restart checkpoint; no automatic external delivery. |
| HTML/API | No-JS next links/preferences, keyboard focus and screen-reader labels, escaped hostile titles/links, no-store, CSRF and same-origin link checks, RFC9457/API envelope compatibility. |
| Storage and audit | Migration/restore/restart; index plan and quotas; race tests, full module tests/vet, dependency advisory/license scan, independent fresh-agent review. |

Human screen-reader checks and eventual core adapter/full-ledger regression remain integration qualification gates, not tests that can be claimed from a source port. Record each pending item accurately in README/feature.json/VALIDATION.md and retain HOLD until the owner's later single-feature integration selection.
