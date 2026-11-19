# Standalone messaging contract

**HOLD — explicitly mounted module, no Circula routes enabled.**

This contract belongs to the independent messaging module. The root API, root database and root service are unchanged. Future integration must adopt this contract in core OpenAPI/service documents and prove host authentication and ledger regressions before enabling it.

## Service and authorization

`Open(path, authorizer)` explicitly opens a private regular SQLite file. New files use mode 0600; existing group/world-readable files are rejected. File URI/query parameters and in-memory databases are rejected by this constructor. `NewStore(db, authorizer)` borrows an explicitly configured host database pool. Host pools must enable foreign keys/secure deletion on every connection and serialize access consistently with the host store. Do not call `Close` on a borrowed pool. `Migrate(ctx)` explicitly applies the additive `001_messaging.sql` transaction. Importing the package installs nothing.

Every operation opens a transaction, reserves the SQLite writer before reading authorization state, then calls `Authorizer` within that transaction. The supplied callback must authenticate its trusted `Actor`, check that the network and feature remain enabled, and read fresh membership/roles/reference policy using that same transaction. It must not open a second pool connection while the store holds its single connection. The module has no API to post, accept, adjust or reverse any ledger transaction.

| Authorization action | Target | Required host decision |
|---|---|---|
| `member` | Member ID | Actor is active in active selected network; future messaging enabled. Creation/sending checks the other participant through the same callback. |
| `conversation` | Other member ID | Current network policy permits this actor to create/send to the other participant (for example, an approved counterparty rule); checked again on retries. Historical participant reads remain subject to their separate membership/retention policy. |
| `reference` | Local listing/transaction ID | Reference exists and is visible in the selected network; never interpret it as a command. |
| `retention-days` | Decimal requested days | Selected network permits that retention period; enforce its shorter configured maximum here. |
| `moderate` | Conversation ID | Current actor has explicit network moderation/legal-hold authority, separate from participant access. |
| `retention` | Empty | Current actor has explicit network retention-maintenance authority. |

Authorizer errors become `ErrDenied`, and HTTP errors never expose SQL, callback details or message text. Caller-supplied `active`, `actor`, `sender`, `network` or role fields cannot authorize a request. The module deliberately rejects a missing authorizer. Host callback behavior is still an integration qualification gate.

## Durable semantics

- Thread IDs and message IDs use 128 random bits. Thread sequence is allocated transactionally and immutable. Sending, sequence/activity updates and idempotency records commit together.
- Creation and sends require 8–128-character idempotency keys containing letters, numbers, `_ . : -`. Scope is network/actor/operation/key. Conflicting hashes fail; retries return the original result. A retry rechecks current membership, visibility and blocks first. A sender who hid a prior message receives empty body on replay. Closed-thread retries can return an already committed result, but never create a new message.
- Conversation and message lists use ascending ID and sequence cursors respectively, 100 results maximum. Snapshot authorization applies per page. Mark-read positions only advance and cannot exceed the latest sequence. Unread counts exclude the member's own and hidden messages.
- Enabling a block requires a current active target; unblocking a departed target remains allowed. Either block direction prevents creation/new sends. Historical participant reads remain available while membership and retention permit. Closing prevents new sends, preserving history. Hiding removes only the caller's existing visible history; new messages can make the conversation visible again. It does not remove the other participant's copy.
- Reports require a visible existing message from the other participant, retain the first report reason on duplicate submission, and never extend retention. Report reasons are bounded to 1024 UTF-8 bytes. Moderators may resolve/dismiss reports and independently place/release a thread legal hold. Reports do not automatically become a legal hold. A new hold cannot resurrect expired unheld content; releasing an expired hold is irreversible through this module.
- Initial safety limits: 4096 UTF-8 bytes/message, no NUL or control characters except tabs/newlines/carriage returns, 100 live open conversations for each participant, 60 new conversations/minute and 1000/day per initiator (including closed/purged tombstones in the window), 1000 directed block records/member, 60 sends/minute and 1000 sends/day per member/network, 20 reports/day, pages of at most 100, HTTP request bodies at most 64 KiB. Limits are checked transactionally. A host may apply stricter admission limits.
- Default inactivity retention is 30 days; creation accepts a shorter 1–30 days subject to host network policy. Authorized `ShortenRetention` applies a newly approved maximum to existing unpurged threads without extending any existing deadline or changing activity. Only a newly committed send extends last activity. At the exact deadline, ordinary reads, inbox, exports, reports, sends and idempotent replies deny access, including held content.
- Explicit `Purge` removes expired non-held entries, report reasons and idempotency request hashes/results in one transaction. It keeps closed purged thread tombstones and per-user read/hide metadata. Purged idempotency keys can eventually be reused for a new operation; durable duplicate suppression is guaranteed only while its retained conversation exists. Old conversation IDs never reopen.
- Only separately authorized moderator operations can access held expired content. Releasing hold makes it unavailable there too until purge deletes it. Moderator report pages return at most 100 reports, ordered by sequence and reporter; both values form the next-page cursor.

## HTTP adapter

`Handler(store, HTTPAuth)` returns an unmounted handler and opens no listener. All three host callbacks (`Resolve`, `VerifyCSRF`, `Token`) are mandatory. Resolve authenticates the request and resolves `X-Network` from authorized host context on **every** request. Cookie requests need session-bound CSRF verification; trusted bearer clients can be exempt only after the host authenticates a non-cookie credential. Do not accept the test-only headers in `http_test.go` in any deployment.

Host must mount on HTTPS, issue Secure/HttpOnly/SameSite cookies, enforce trusted origins, rate-limit authentication, and keep credential/session revocation current. No WebSocket, SSE or external delivery service is necessary: ordinary request/response works without JavaScript and SQLite unread state remains authoritative. Realtime was optional and is intentionally absent.

All responses, including errors, use `Cache-Control: no-store`, `Pragma: no-cache`, `nosniff` and restrictive CSP. HTML uses `html/template`, never trusted HTML, remote images, URL previews or attachments. POST errors preserve generic accessible alerts. API accepts bounded UTF-8 JSON objects, rejects unpaired UTF-16 escapes, duplicate/case-aliased keys, null/array roots, unknown fields/trailing documents and returns RFC 9457 problem documents for failures. IDs and network scope do not come from client actor fields.

| Path | Method | Input / result |
|---|---|---|
| `/api/v1/conversations` | GET | `after` ID cursor; conversation array including unread counts |
| `/api/v1/conversations` | POST | `{target, reference?, retention_days?}`; `Idempotency-Key`; conversation |
| `/api/v1/conversations/{id}/messages` | GET | `after` sequence cursor; message array |
| `/api/v1/conversations/{id}/search` | POST | `{query, after?}`; literal case-sensitive substring, 128 UTF-8 byte maximum, at most 100 currently visible matches |
| `/api/v1/conversations/{id}/messages` | POST | `{body}`; `Idempotency-Key`; committed message |
| `/api/v1/conversations/{id}/read` | POST | `{sequence}`; monotonic read marker |
| `/api/v1/conversations/{id}/hide` | POST | `{}`; hide caller's current history |
| `/api/v1/conversations/{id}/close` | POST | `{}`; prohibit new sends |
| `/api/v1/conversations/{id}/report` | POST | `{sequence, reason}`; report counterpart's visible message |
| `/api/v1/conversations/{id}/reports` | GET | Moderator only; `after` sequence and `reporter` cursor; at most 100 reports |
| `/api/v1/conversations/{id}/moderate` | POST | Moderator only; `{action, sequence?, reporter?}`, actions hold/release/resolve/dismiss |
| `/api/v1/conversations/{id}/held` | GET | Moderator only; `after` sequence cursor; active or legally held history |
| `/api/v1/conversations/{id}/export` | GET | `after` sequence; one JSON page, at most 100 visible messages |
| `/api/v1/block` | POST | `{target, enabled}`; caller-owned directed block |
| `/api/v1/retention` | POST | `{days}` (1–30); separately authorized shortening of existing network retention, never extension |
| `/api/v1/purge` | POST | `{}`; authorized network retention maintenance; count |

Message and export pages are exhausted when fewer than 100 entries return; use the last sequence for the next `after`. Conversation pages use the last ID. Report pages use the last sequence as `after` and last reporter as `reporter`. A 100-entry final page may require one additional empty request. Successful mutations with no entity result return `{ok:true}`. Errors: 400 invalid input, 403 unavailable/unauthorized, 409 idempotency conflict, 429 quota, 500 generic service failure. Unsupported verbs return 405.

## Retention, disable and restore

Run purge explicitly under a network retention principal. Schedule creation belongs to later opt-in integration; importing this package never creates a worker. Disabling removes handler/maintenance registration and revokes network feature authorization; no ledger migration or reversal is involved.

Use SQLite's backup API or `VACUUM INTO` to take a consistent snapshot. Restrict backup permissions to 0600 before transfer/restore; `VACUUM INTO` may create a less restrictive file under the caller's umask. Restore only while no process writes the file, restore the complete SQLite snapshot, reopen and rerun the idempotent migration. Do not copy only a live main DB while ignoring its WAL. The acceptance suite demonstrates snapshot restore of messages and own-view state. No destructive down migration is supplied: rollback means disabling code or restoring a verified pre-install backup, which intentionally discards later feature data.

`secure_delete` and row purge do not erase existing backups, OS snapshots or residual WAL/storage media. Operators must define protected backup retention and legal-hold access separately. Message bodies and reasons must be excluded from request/access logs, error telemetry, URLs and routine audit metadata. Exports deliberately contain plaintext and must be protected by their owner.

## Remaining qualification gates

The module's service/API/UI/CLI are implemented and locally exercised. Independent audit and module advisory scanning are recorded in AUDIT.md. Real browser accessibility/no-JS testing, manual screen-reader review, final API schema adoption, host fresh-authorization/CSRF adapters and full ledger/core regression remain required. No current test result grants approval to enable, merge, release or deploy this feature.

## Audited operator interfaces

Authorized moderators use `/conversations/{id}/moderation` for escaped evidence, report disposition and hold/release forms, with separate pagination for evidence and reports. Network retention maintainers use `/messaging-maintenance` for shortening and purge. Both require current transactional role checks and session-bound CSRF; ordinary members receive no evidence. No default administrator exists. CLI parity includes `reports`, `held`, `moderate`, `purge`, `retention` and a `-cursor` for conversation lists. CLI response download is bounded to 4 MiB and fails without emitting a truncated page on excess.

## Message search

`Search(ctx, actor, thread, query, after, limit)` uses SQLite's bound `instr` predicate. Wildcards and quotes are literal data. It accepts 1–128 UTF-8 bytes, rejects blank text and control characters, and returns ascending sequence pages of at most 100 messages. It rechecks current member authorization, tenant/participant boundaries, retention and the caller's hidden-through position in the same transaction. A legal hold does not make expired content searchable by members. No search changes last activity, unread counts or read positions.

The API and no-JavaScript search/continuation forms use POST, with the existing CSRF policy. Search text is never placed in query strings, redirects or pagination URLs. The CLI `search` command takes query text from stdin and `-after` provides the last matched sequence. Operators must continue excluding request bodies and response content from logs. Results are case-sensitive literal Unicode substrings; linguistic stemming or case folding is not promised.

## Current HTTP callback ordering

Raw queries are capped at 8192 bytes and malformed URL encoding/semicolon query syntax is rejected before host callbacks. Existing route-specific pagination remains supported. Ordinary POST forms are parsed under the 64 KiB cap before Resolve; valid cached PostForm is available to host CSRF checks. Resolve may run more than once per request and must be a read-only current-session decision. After VerifyCSRF or Token, the handler requires the same current actor and an uncancelled request before private reads/mutation. Search token acquisition precedes the existing transactional private search.

Host middleware must not consume malformed input and discard parse errors before invoking the handler. These bounded checks supplement transactional membership/content checks; they do not promise global atomicity across arbitrary external session stores.
