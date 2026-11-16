# FUX-105 messaging: upstream research and adaptation handoff

Date: 2026-11-16. Phase: research only. **HOLD: not integrated, not feature-complete.**

## Recommendation

Reuse **coder/websocket v1.8.15** as a pinned optional transport dependency and adapt its bounded subscription/write-deadline pattern. Keep durable messaging, network authorization, participant access, idempotency, moderation and retention as Circula-native code. A transport's public-chat sample is not a private-message security model. Use existing **modernc.org/sqlite v1.60.1** for durable standalone acceptance tests and future host adapters; do not introduce a second SQLite driver into Circula.

The upstream library supplies WebSocket framing, protocol validation, cancellation and closing. Its chat example contributes concrete queue/backpressure mechanics. It does **not** supply any of this feature's authoritative business rules. Reusing more of that anonymous broadcast example would remove required privacy controls. No upstream source was copied or dependency added during this phase.

## Verified GitHub candidates

Repository discovery used GitHub repository search. Source and license inspection used shallow clones from each official repository; release dates came from GitHub's releases API. These observations are a maintenance signal, not an independent security certification.

| Candidate | Exact inspected source / release | License actually read | Maintenance evidence and disposition |
|---|---|---|---|
| [coder/websocket](https://github.com/coder/websocket) | Selected **v1.8.15**, commit **9c8faadccd1b679e811a79ce506f8a10237251ad** | [LICENSE.txt](https://github.com/coder/websocket/blob/9c8faadccd1b679e811a79ce506f8a10237251ad/LICENSE.txt): ISC-style permission, copyright 2025 Coder | [Release](https://github.com/coder/websocket/releases/tag/v1.8.15) published 2026-06-15. Inspected master `ac2177d22d9117ec38f06358fac56e7d70e98c15`, dated 2026-11-14. Selected release requires Go 1.23 and has no module dependencies. Upstream README reports Autobahn protocol tests; use our own auth/retention tests. **Select.** |
| [centrifugal/centrifuge](https://github.com/centrifugal/centrifuge) | Inspected master **99674433635f3c9ee96d2f7b36ecc25bf0c91e1b**; release v0.40.1 resolves to **79c893e8692d5caf29e4c1e5983191e6e5ee3530** | [LICENSE at inspected source](https://github.com/centrifugal/centrifuge/blob/99674433635f3c9ee96d2f7b36ecc25bf0c91e1b/LICENSE): MIT | Release published 2026-10-20; inspected master dated 2026-11-16. README identifies use by Centrifugo/Grafana Live, but warns of pre-v1 API changes. Rich channel/session recovery, brokers, Redis, metrics and SDK surface exceeds this single-process module. Its ephemeral stream history is not authoritative retained private messaging. **Do not add for v1.** |
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | Inspected master **4cddc794daa0340d50de741dae025421bbdbba0c**; release v1.14.53 resolves to **93c7e0a2ec6762a38ab371755d49a2b3f4a12e67** | [LICENSE at inspected source](https://github.com/mattn/go-sqlite3/blob/4cddc794daa0340d50de741dae025421bbdbba0c/LICENSE): MIT | Release published 2026-11-05; inspected master dated 2026-11-16. Mature database/sql SQLite driver, but README explicitly requires cgo. Existing Circula uses pure-Go modernc SQLite. **Reject replacement/second driver**, retain native SQL patterns. |

**Correction to the historical spec:** the copied spec labels coder/websocket “MIT”; the selected release's actual license is ISC-style, not MIT. Preserve that exact license text and copyright if vendoring or adapting example code. Do not propagate the historical label into notices. A future release must include all required transitive notices. No project-wide product license is chosen by this research.

### Existing storage dependency verified

`go mod download -json modernc.org/sqlite@v1.60.1` resolved upstream GitLab commit **b122d0417c01508beb55158faedeb64a0c5bfd8a**, module sum `h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw=` and go.mod sum `h1:1dIoEagfDE72QytD5scH1lxARtaUgKgHC/NuApA27r0=`. Its downloaded LICENSE is BSD-3-Clause-style (2017 The Sqlite Authors). [Source repository](https://gitlab.com/cznic/sqlite/-/tree/b122d0417c01508beb55158faedeb64a0c5bfd8a), [module documentation](https://pkg.go.dev/modernc.org/sqlite@v1.60.1).

This version requires Go >=1.26.0. Initial local Go1.25.4 correctly rejected it; the coordinating agent subsequently verified Go1.26.11 can be downloaded/used. Use `GOTOOLCHAIN=go1.26.11 /tmp/circula-toolchain/go/bin/go ...`, matching core, instead of silently downgrading the driver or rewriting the root go.mod.

## Specific reusable source and API boundary

All selected links below are immutable source links, not floating master examples:

- [`accept.go`](https://github.com/coder/websocket/blob/9c8faadccd1b679e811a79ce506f8a10237251ad/accept.go): `websocket.Accept`, `AcceptOptions`. Authorize the session and selected network **before** upgrade. Keep origin verification enabled; do not set `InsecureSkipVerify` or wildcard `OriginPatterns`. Add explicit configured same-origin enforcement because a Host header is not a trusted deployment configuration. Do not enable compression by default.
- [`read.go`](https://github.com/coder/websocket/blob/9c8faadccd1b679e811a79ce506f8a10237251ad/read.go): `Conn.CloseRead` for hint-only server-to-client connections; `SetReadLimit` for any explicitly supported inbound traffic. Send actual messages through normal authenticated idempotent HTTP, not a second WebSocket write path.
- [`write.go`](https://github.com/coder/websocket/blob/9c8faadccd1b679e811a79ce506f8a10237251ad/write.go) and [`close.go`](https://github.com/coder/websocket/blob/9c8faadccd1b679e811a79ce506f8a10237251ad/close.go): bounded writes with contexts; close slow clients; always release connection resources.
- [`internal/examples/chat/chat.go`](https://github.com/coder/websocket/blob/9c8faadccd1b679e811a79ce506f8a10237251ad/internal/examples/chat/chat.go): `subscriber`, `subscribe`, `writeTimeout`, `addSubscriber`, `deleteSubscriber`. Adapt the bounded per-subscriber queue and timeout mechanics with retained ISC notice and a provenance entry. **Do not copy** its global public subscriber collection, unauthenticated publish endpoint, in-memory source of truth, broadcast message body, or global rate-limiter placement under the subscriber mutex. Index subscriptions by server-authorized `(network,member)`; hints carry no body, participant names or trade references. Commit first, signal second. Dropped hints must never lose messages.
- [Versioned API docs](https://pkg.go.dev/github.com/coder/websocket@v1.8.15): implementation agent should check exact signatures against this version, pin in the nested module only, and commit go.sum.

Real-time is optional in the spec. A functional no-JS request/response interface plus database-backed unread count is required even when the optional WebSocket adapter is absent or fails.

## Current Circula seams inspected (read-only)

- `internal/service/principal.go`: `Service.open`, `openAny`, `principal.requireActive`, `requireNetworkActive`. These must remain the authoritative host decisions; an untrusted `active: true` request field never grants access.
- `internal/service/membership.go`: current member status and departure policy.
- `internal/store/iface.go`: `Store.Update`/`View` unit of work and network-scoped repository interfaces.
- `internal/store/sqlite.go`: serialized `BEGIN IMMEDIATE` writes; snapshot `View`; WAL, foreign keys and `secure_delete(1)`. Reuse the host transaction boundary rather than opening a second independent write transaction during an authorized operation.
- `internal/service/notify.go`: existing notifications may receive minimal post-commit inbox hints later; no dependency on the unfinished future notifications feature.
- Root `go.mod` pins modernc.org/sqlite v1.60.1; standalone messaging currently has no external dependency and Go1.25.0.

Do not modify these core files now. Develop an explicit adapter interface and standalone contract harness under `future/messaging/`. A later single-feature integration change can implement the host adapter and adopted API/schema contracts.

## Full-spec implementation plan for the fresh phase-2 agent

### 1. Durable schema and transactional service

Create additive SQL in the module, applied only by an explicit standalone installer/test harness (no automatic migration on import):

- Threads keyed by `(network_id,id)` with distinct participant IDs, optional validated listing/transaction reference, closed timestamp, last activity timestamp, next sequence, retention days (1–30).
- Entries keyed by `(network_id,thread_id,sequence)` and unique stable message ID. Store bounded UTF-8 literal text and immutable sender/time. Do not place message text in logs or audit metadata.
- Idempotency record keyed by `(network_id,actor_id,operation,key)` with request hash and resulting message ID; reject a reused key with different content. Insert message, increment sequence, update last activity and idempotency result in one `BEGIN IMMEDIATE` transaction. Check auth/block/quota state in that same unit.
- Read positions and per-participant hidden-through sequence, unique per `(network,thread,member)`. Advance read positions monotonically and never past latest visible sequence. Hiding one participant's view must not delete the other participant's copy.
- Network-scoped directed block relations; either direction rejects new conversation/sends. Explicitly distinguish read-history policy from send policy rather than relying only on today's single `Thread.Blocked` boolean.
- Reports bound to a message the reporter may actually read; bounded reason, dedupe/idempotency, moderation state, an authorized reviewer read surface, and legal-hold references. Reporting is not a license to enumerate arbitrary threads.

Use injected clock/ID generator for deterministic tests. Include `Close` behavior and canceled-context propagation. Do not implement JSON-file persistence as a substitute for the required SQLite concurrency/restart semantics.

### 2. Authorization and service contracts

Expose operations for create/list thread, send/list messages, advance read marker, hide own history, block/unblock, report, close, purge, and authorized moderation. All require explicit trusted actor/network context via a host-authorizer interface. The standalone test adapter must model membership changes, suspension, blocks and role-based moderation. Reject missing authorizer by construction. Never authorize by caller-supplied participant names or body fields.

Authorize both participants and the selected network when creating a thread; validate optional listing/transaction references within that network through a read-only callback. No call to proposal, acceptance, posting, balance adjustment or reversal methods is permitted. Fresh membership is required on reads as well as sends. Recheck current auth before sending any hint to a long-lived connection; disconnect revoked sessions/members and bound maximum connection lifetime.

### 3. Retention, deletion, limits and recovery

Default expiry is 30 days **after thread last activity**, with network policy only shortening it. A new accepted message extends that inactivity deadline; merely reading, reconnecting, reporting or marking unread must not extend it. Preserve legal-held/disputed material under a separate restricted policy; do not present it in ordinary expired inbox/export paths. Define how metadata/idempotency tombstones survive payload purging without retaining plaintext.

Set explicit maximum message size (existing 4096-byte limit), active threads/member, sends/time window, request body size (host 64 KiB), page size and report limits. Quotas must be transactional if they affect acceptance. Queue limits in the transport cannot replace durable send quotas. Apply expiry/read-visibility filters on every path, including exports/moderator paths according to separate permissions. Document physical deletion limits involving SQLite WAL and existing backups; application row deletion does not prove that historical backups were erased.

### 4. No-JS UI, optional transport and contract artifacts

Implement reusable `html/template` views and an explicitly mounted handler factory in the module/harness. Include thread list, escaped message history, send form, unread status, block/report/hide actions, accessible validation errors and pagination. Never cast message text to trusted HTML; no link previews, remote images or attachments. Show “Messages can be read by the server operator; this is not end-to-end encrypted.” Provide CSRF and auth adapter requirements, no-store headers and normal HTTP form responses.

Add a standalone OpenAPI fragment/API contract and CLI service client/harness as appropriate without editing the root routes or advertised API. Optional WebSocket implementation uses the selected dependency and adapted bounded queues, sends unread-change hints only, and remains dispensable. Real-time failure must not affect durable message commit. No deployment configuration, background worker or listener starts automatically.

### 5. Attribution and compatibility

If example snippets are copied or modified, add `THIRD_PARTY_NOTICES.md`, the complete upstream ISC license in `third_party/licenses/`, and a provenance table mapping local file to upstream commit/path and modifications. If only the module dependency is used, still document the exact pin and packaging notice obligations. No GPL/AGPL project code is selected. Do not choose a new product license. Keep all new dependencies in this nested module and note the minimum toolchain change explicitly.

## Acceptance evidence required before claiming completion

| Area | Required tests/evidence |
|---|---|
| Tenant and participant isolation | Wrong network, nonparticipant, spoofed sender, missing auth callback, suspended/departed member, foreign linked listing/trade, unauthorized moderation all fail for read/write/export/hints. |
| Durability/idempotency | Many concurrent sends allocate unique monotonic sequences; same-key identical retries return one result; conflicting retries fail; injected errors roll back sequence/content/idem together; close/reopen SQLite preserves order/read markers/blocks/reports. |
| Privacy and rendering | Script/HTML bodies render escaped; no body in logs/URLs/transport hints; no-store on private pages/errors; one participant's hide leaves the other copy; absent JavaScript leaves all essential actions usable. |
| Retention | Boundary expiry at 30 days or shorter configured policy, extension only on accepted send, legal holds, hidden-through state, purge/restart, export filtering, no plaintext in idempotency payload or ordinary audit. |
| Abuse and quotas | Both block directions, unblock, duplicate report, invalid report target, reporter access, rate/page/body limits, concurrent quota races, control/invalid UTF-8 cases. |
| WebSocket adapter | Same-origin and missing/hostile Origin handling, unauthorized upgrade, read limits, slow subscribers, canceled context, disconnect cleanup, session/membership revocation, queue pressure and post-commit-only hints. |
| Deployment/core independence | Standalone race tests/vet; no `internal/` imports and no writes outside module; root remains byte-identical; no future-feature dependency; installer is explicit; no `init` side effects or ledger methods. |
| Final qualification | Fresh audit agent reviews every requirement; run parser/service fuzzing, dependency vulnerability checks, browser/no-JS accessibility and manual screen-reader acceptance; verify backup/restore semantics and eventual host contract/ledger regression. |

A phase-2 agent may finish a standalone module without completing the eventual core integration. It must state that distinction and leave integration HOLD unchanged. Do not mark browser/manual, restore, host-auth or independent-audit gates passed without actually observing them.

## Research limitations

No dependency vulnerability scan or upstream test suite was run in this research-only phase. Activity/release observations do not establish absence of vulnerabilities. Modernc source license and module metadata were inspected from the downloaded official module archive; no new SQLite implementation was copied. Complete feature readiness depends on the Circula-native service and integration tests above; none of the three candidate repositories is a drop-in replacement for those requirements.
