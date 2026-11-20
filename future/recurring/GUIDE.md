# Recurring reminders: standalone guide

This module is disabled by default and is not registered in Circula. It has no payment, proposal, autoaccept or ledger API. Core API/CLI routing and eventual hosting remain a separate integration change.

## Member flow

1. An active member opens the host-mounted reminders page. With JavaScript disabled, create a template using a counterparty, optional private note/default amount, structured daily/weekly/monthly schedule and IANA zone.
2. No amount means an empty amount prefill. Saving a template grants no authority to transfer credit. At most twenty non-cancelled templates are allowed per owner/network.
3. Edit uses the displayed revision. A stale form returns a conflict rather than overwriting another edit. Pause stops reminders; resume rechecks the counterparty. Cancel is permanent. Pause/cancel/skip still require current owner membership but do not require an active counterparty.
4. Skip next occurrence stores a logical-window tombstone, so restart cannot recreate it. It skips the next scheduled time at or after the action; a reminder already generated remains historical.
5. A reminder contains only an owner-private reference to the template, never copied note/amount/target data. Open the template and deliberately create a draft for review. A fresh action key creates a distinct random intent; retrying the same key returns the same prefill. The draft still needs the existing trade proposal and counterparty acceptance flow.
6. Local time gaps are skipped; folds use the first occurrence; a missing monthly day skips that month. Daily/weekly/monthly are the only supported recurrences. No seconds, arbitrary cron, descriptors or untrusted TZ overrides are accepted. An additional 20-hour guard prevents a rapid timezone edit from generating a second reminder under a different local-date key while allowing ordinary 23-hour DST days. Downtime creates at most today's local occurrence, never an unbounded backlog.

## Embedding service and CLI

Supply an explicit `database/sql` SQLite connection, injected UTC clock and transaction-aware `Authorize` callback. The callback must check current network opt-in, owner membership, target membership and policy on the supplied transaction. On `draft`, check current terms and capacity. Do not consume authorization booleans from JSON. Use `ErrDenied` only for policy/membership denial; operational failures must remain errors, so they do not silently suspend a template.

Construct `Service{DB: db, Auth: policy, Now: clock, Enabled: true}` explicitly. `Migrate` is additive and transactional. The host must set a bounded SQLite busy timeout and foreign_keys on **every** connection. No driver is imported by the library. SQLite v1.60.1 is pinned only for synthetic standalone tests.

`RunCLI(ctx, service, currentActor, args, stdin, stdout)` shares that service. It is an embedding API, not an installed root Circula command. Argument examples:

- `list [cursor]`
- `save <action-key> <expected-version>` with a bounded `Record` JSON on stdin. Omit server-owned Network, Owner, Version, Created, Updated and Cancelled fields. ID is empty on creation, otherwise supply ID and expected revision.
- `pause|resume|skip|cancel <template-id> <version> <action-key>`
- `draft <template-id> <action-key>`
- `preview <template-id>`
- `inbox`

`Handler` uses injected current-session authentication and cookie CSRF verification. Its form path is the current mounted path. All responses use no-store, no sniffing and a restrictive CSP; JSON errors are RFC 9457. Body size is at most 64 KiB. `X-Network`, when present, must match the authenticated network. JSON mutations require `Idempotency-Key`; native forms carry a fresh hidden action key and host CSRF token. No alternate login or credentials are stored.

## Scheduler and atomic inbox delivery

An explicitly started host job calls `RunDue(ctx, network, cursor, limit)` with 1–100 rows. Persist/continue `Scan.Next` after each successful call; when Complete, wrap to empty on the next scan. Every examined row advances the cursor, even if not yet due, revoked or already fired. New rows before a cursor are considered on the next wrap. Keep scheduler frequency and network scan fairness bounded in the host.

One SQLite write transaction obtains a writer lock, reads the current record and authorization, inserts the unique `(network, template, window)` firing, inserts the private reminder outbox and writes audit. Concurrent workers cannot create duplicate occurrences. An optional `Remind(ctx, tx, reminder)` callback may insert the core in-app inbox row **using the same transaction**; callback failure rolls everything back. It must not call a remote service or write a ledger. Without that callback, only the durable module inbox/outbox exists; external delivery is not claimed exactly once.

## Operations, retention and restore

- Apply the module migration explicitly and back up before future migration revisions. It creates only `future_*` tables/indexes, not core schema or ledger entries. Failed migration rolls back its schema changes.
- Use a database-consistent SQLite online backup/VACUUM snapshot, or stop **all writers**, close all connections and copy the complete SQLite database. Never copy an active WAL database file alone. Tests exercise a closed-database restore and integrity check.
- `Cleanup` is an explicit trusted maintenance API, with a minimum 90-day retention. It removes older reminders/audit, clears private response bodies from old idempotency records and scrubs notes/amounts from old cancelled templates. It retains opaque operation fingerprints and logical firing tombstones; retired retry keys fail conflict rather than creating new actions. The explicit trusted `Depart(network, member)` lifecycle hook cancels all owner/target templates and removes owned reminder/private retry payloads without pagination. A full owner/network erasure/archive policy still needs core integration review because minimal metadata and tombstones intentionally remain.
- Disable by stopping the host scheduler and removing handler/CLI registration. No ledger rollback is involved. Keep tombstones and module data until the approved retention/backup procedure runs. This branch does not change root configuration or start jobs.

Independent candidate audit completed (see AUDIT.md); actual core transaction/inbox adapters, host scheduler recovery/fairness, root API/CLI contracts, full-ledger regression, browser/keyboard/screen-reader qualification and production operational approval remain gates. Unit coverage does not qualify these gates.

## Audited persistence and retry boundaries

Migration accepts only the known reference schema and rolls back incompatible pre-existing shapes. Safety writes check that a row persisted; ignored mutations and absent serialization rows fail atomically. Read DTOs must match indexed owner/network/state/version fields. Cached drafts are reauthorized against their own original counterparty/default amount after template edits; target departure retires matching historic cached response bodies as well as scrubbing live records. Retired retry keys remain conflicts, never new actions. Form mounting paths must be local absolute paths (or empty for explicit self-target); protocol-relative, backslash and query/fragment/control paths are rejected. CLI preview uses a direct private lookup instead of a first-page scan.

Calendar Next selects the next logical window: a fold's second physical time is not another occurrence or another skip target. The actual fold regression uses New York in November 2026; Vancouver's November 2026 daily fixture does not prove a fold with current tzdata.

## Strict pagination and mutation URLs

GET accepts only single `after` and `limit` fields, with an 8192-byte transport bound before session callbacks. Empty after remains a valid first page; nonempty limit must be 1–100. POST URLs must have no nonempty query. Native forms explicitly post to the current local path without pagination fields. Invalid pagination is rejected before any mutation, avoiding a reported request failure after a successful state change.
