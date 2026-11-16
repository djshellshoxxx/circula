# FUX-104 upstream research and implementation handoff

Research date: 2026-11-16. Phase 1 only. **HOLD: no integration, activation, release, or feature-completion claim.** Current Circula baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`. The next phase must use a fresh implementation agent. No upstream implementation has been copied by this research commit.

## Decision

Reuse SMTP composition/transport from **wneessen/go-mail**, and Web Push encryption/VAPID from **SherClockHolmes/webpush-go**, as pinned module dependencies behind narrow Circula interfaces. Do not transplant an entire notification server. Write Circula's scoped outbox/preferences/subscription services around its existing SQLite transaction boundary; use Watermill's SQLite implementation as a reviewed design/test reference, not a mandatory runtime or an assurance of production maturity.

This maximizes reusable protocol code while retaining the application-specific authorization, consent, privacy, idempotency and event atomicity locally. No candidate supplies all of FUX-104 safely by itself.

## Candidate evidence and pins

Repository search was performed through the connected GitHub repository search, followed by cloning official repositories, reading `go.mod`, source, tests and license files, and resolving tags/HEAD with Git. Pins below are actual observed commit IDs, not inferred versions. Maintenance observations are evidence of activity, not proof of security.

| Component | Exact inspected pin | License and maintenance | Disposition |
|---|---|---|---|
| [wneessen/go-mail](https://github.com/wneessen/go-mail) | Latest observed release `v0.8.2`, commit `c9de0036dfb34e1ed2f79ea1c1fccb3c61db501a` (2026-11-08). Main inspected at `e3e1ecc11f26d7c79c1de5d61aecb1debe80a56a` (2026-11-14). Compatibility candidate `v0.7.3` = `52312c1ecd6735ecdbda674fe6ac5d7d561939e8` (2026-05-12). | Root source MIT; bundled `smtp/` has Go Authors BSD-3-Clause; assets/docs can have other licenses. Existing versioned release history and recent fixes. Latest release `go.mod` requires Go 1.26.10 and declares a Go 1.27.3 toolchain; v0.7.3 requires Go 1.25.0. | Selected SMTP library. Prefer latest release once available toolchain is qualified. v0.7.3 is only a compatibility fallback, not implicitly approved: review intervening fixes and advisories before using it. Do not silently downgrade to pass Go 1.25 tests. |
| [SherClockHolmes/webpush-go](https://github.com/SherClockHolmes/webpush-go) | `76cc00f5ac2d6e5333eabe149f52f6b6d80247a0` (2026-04-21); release v1.4.0 = `f5c3e9f7b642a8dd66cb844050520526f721971d` (2025-01-02) also inspected. | MIT, copyright Ethan Holmes. Compact Go Web Push implementation with versioned release history and tests. Inspected HEAD adds auth scheme selection and declares jwt/v5 v5.2.2 and x/crypto v0.31.0. | Select exact HEAD commit for current API, resolve/store its Go pseudo-version and go.sum during implementation. Review transitive advisories; do not copy its default unconstrained HTTP transport behavior. |
| [ThreeDotsLabs/watermill-sqlite](https://github.com/ThreeDotsLabs/watermill-sqlite) | `843d10d38a356dde0c794325c7d215b589e443c2` (2026-10-28). | MIT, copyright Dmitry Kotik. README explicitly calls it beta/not widely production-tested. ModernC submodule Go 1.23.0, Watermill v1.5.1, SQLite v1.36.1. Has transactional/concurrent publishing tests. | Reference only. It is not sufficiently evidenced as a mature drop-in outbox; consumer-group schema and dependencies would broaden Circula unnecessarily. No dependency needed. |
| [ThreeDotsLabs/watermill-sql](https://github.com/ThreeDotsLabs/watermill-sql) | `fe6475af52c9348dbe6f14eb540d3c3513683c72` (2026-10-28). | MIT; v4 module uses MySQL/PostgreSQL drivers and Watermill v1.5.2. | Rejected for direct integration: not the project's SQLite storage model. No code migration planned. |

### Exact source and license references

- go-mail [v0.8.2 go.mod](https://github.com/wneessen/go-mail/blob/c9de0036dfb34e1ed2f79ea1c1fccb3c61db501a/go.mod), [MIT license](https://github.com/wneessen/go-mail/blob/c9de0036dfb34e1ed2f79ea1c1fccb3c61db501a/LICENSE), [SMTP BSD license](https://github.com/wneessen/go-mail/blob/c9de0036dfb34e1ed2f79ea1c1fccb3c61db501a/smtp/LICENSE), [client API](https://github.com/wneessen/go-mail/blob/c9de0036dfb34e1ed2f79ea1c1fccb3c61db501a/client.go), [message API](https://github.com/wneessen/go-mail/blob/c9de0036dfb34e1ed2f79ea1c1fccb3c61db501a/msg.go).
- go-mail [v0.7.3 go.mod](https://github.com/wneessen/go-mail/blob/52312c1ecd6735ecdbda674fe6ac5d7d561939e8/go.mod). Source tree includes CC-BY-ND licensed material in addition to code licenses: **do not copy logos/docs/assets indiscriminately**.
- webpush [API implementation](https://github.com/SherClockHolmes/webpush-go/blob/76cc00f5ac2d6e5333eabe149f52f6b6d80247a0/webpush.go), [VAPID implementation](https://github.com/SherClockHolmes/webpush-go/blob/76cc00f5ac2d6e5333eabe149f52f6b6d80247a0/vapid.go), [license](https://github.com/SherClockHolmes/webpush-go/blob/76cc00f5ac2d6e5333eabe149f52f6b6d80247a0/LICENSE), [module](https://github.com/SherClockHolmes/webpush-go/blob/76cc00f5ac2d6e5333eabe149f52f6b6d80247a0/go.mod), [official usage](https://github.com/SherClockHolmes/webpush-go/blob/76cc00f5ac2d6e5333eabe149f52f6b6d80247a0/README.md).
- Watermill SQLite [README maturity caveat](https://github.com/ThreeDotsLabs/watermill-sqlite/blob/843d10d38a356dde0c794325c7d215b589e443c2/README.md), [license](https://github.com/ThreeDotsLabs/watermill-sqlite/blob/843d10d38a356dde0c794325c7d215b589e443c2/LICENSE), [publisher](https://github.com/ThreeDotsLabs/watermill-sqlite/blob/843d10d38a356dde0c794325c7d215b589e443c2/wmsqlitemodernc/publisher.go), [transaction tests](https://github.com/ThreeDotsLabs/watermill-sqlite/blob/843d10d38a356dde0c794325c7d215b589e443c2/wmsqlitemodernc/transaction_test.go).

## Specific reuse boundary

1. **SMTP adapter:** `mail.NewMsg`, validated From/To fields, fixed subject, `SetBodyString`, `NewClient`, `WithTLSPolicy`, `WithTimeout`, `WithSMTPAuth`, `WithUsername`, `WithPassword`, `DialAndSendWithContext`. Require TLS with normal certificate/hostname verification. Do not use opportunistic downgrade or debug logging. Sender/SMTP host is operator configuration, never a member-provided arbitrary SMTP endpoint. Always send `GenericBody`, not a ledger payload. Validate authentication mode against chosen release docs. Make a new immutable message per send to avoid message reuse/reset mistakes.
2. **Web Push adapter:** `Subscription`, `Keys`, `Options`, `SendNotificationWithContext`, and injected `Options.HTTPClient`. Upstream handles encryption and VAPID; Circula handles subscription ownership, endpoint policy, deadlines, response classification, opt-out and secret lifecycle. Close response bodies and read a bounded error response without logging provider tokens. Configure auth scheme explicitly if required by tested browser/push provider, rather than guessing from an untrusted URL.
3. **Outbox:** study accepting an existing transaction handle and transactional tests in Watermill's `publisher.go`/`transaction_test.go`. Implement a small `EnqueueTx`/claim/ack/retry repository for Circula instead of importing the publisher. In particular, do **not** copy Watermill trace logging of query arguments/payloads, dynamic topic table machinery, or consumer-group offset tables. Ordinary upstream comments are not a substitute for verifying SQLite behavior.
4. **License bookkeeping:** phase 2 must add exact dependency pins/go.sum and third-party notices within this module, including applicable transitive/license notices. If any source is actually copied, record original commit/path, unmodified original license, local modifications and copied-file hashes. No GPL/AGPL copying is proposed. Circula's product license remains undecided; do not invent or change it.

## Existing Circula seams inspected

- `internal/service/notify.go`: `notify(tx store.Tx, ...)` inserts the in-app notification inside the same caller unit of work as the originating event. `notifyOthers` deduplicates recipients. Notification kinds already use fixed strings and no member-written text.
- `internal/store/iface_notify.go`: `InsertNotification` returns `(inserted bool, error)` and deduplicates known keys. External outbox insertion must use the authoritative committed source identity and be atomic with the source event, not a later poll that could miss events.
- `internal/store/sqlite.go`: `SQLite.Update(ctx, fn)` supplies the transaction boundary. Production wiring is deferred; standalone module tests can exercise an adapter using its own test SQLite database and explicit transaction.
- New interfaces must accept trusted network/member/session status from a service adapter. Do not treat fields posted by the client as membership evidence.
- Schema spec's example `UNIQUE(event_id,member_id,channel)` conflicts with the stated kind-aware dedupe rule and tenant independence. Use/explicitly document `UNIQUE(network_id,event_id,member_id,channel,kind)` unless a reviewed source-ID invariant proves an equivalent key. Existing `DedupeKey` already includes all five.

## Phase 2 full-spec implementation plan

All implementation remains under `future/notifications/`, with no core imports or automatic registration. Separate files/packages for policy, persistence, dispatcher, SMTP, Web Push, secrets, endpoint validation, HTTP/HTML adapters, CLI, and fixtures.

| Requirement | Implementation | Required evidence |
|---|---|---|
| Per-network/member/event/channel consent | Durable preference CRUD defaults off, kind allowlist, quiet hours/timezone validation; audit callback; current membership lookup | Ownership/tenant denial, inactive/pending members, unknown kinds, DST fold/gap, invalid zone, equal quiet endpoints documented |
| Optional transport | Inject `Sender` interface with separate SMTP/Web Push implementations; zero configured transports remain valid | Fixed bodies/headers, fake SMTP TLS/auth errors, cancellation, Web Push cryptographic success/failure, browser/provider compatibility |
| Atomic durable outbox | Explicit migration, enqueue on passed transaction, unique five-field identity, ready/in-flight/retry/sent/cancelled/expired states, bounded leases and attempt count | Source rollback leaves zero rows, duplicate event returns original row, two workers cannot own same lease, restart before send/after send/before ack; bounded at-least-once delivery documented |
| Private delivery | Minimal generic body only; opaque IDs in metrics; encrypted endpoint/key material with network/member/subscription binding as associated data, secret file/env references | DB inspection contains no plaintext subscription or keys; tamper/wrong-tenant decryption fails; logs exclude endpoints, passwords, email addresses and trade fields |
| Unsubscribe/revocation/bounce | Owner-scoped revocation, expiry, logout-associated subscription revocation, authenticated signed unsubscribe token; permanent SMTP errors and push 404/410 disable targets | Recheck subscription/preference/session immediately before dispatch; race test opt-out versus claimed delivery; document already-in-flight provider delivery cannot be recalled |
| Retry/diagnostics | Clock-injected bounded dispatcher, backoff, capped Retry-After parsing, no retries past expiry, redacted status/test CLI and metrics | Provider outages never call ledger hooks; attempt cap, poison-row progression, no starvation, health/status do not expose keys |
| APIs/UI | Standalone adapter with injected actor/network resolver, CSRF/auth and limits contracts; no-JS preference form, progressive push subscription JS and service-worker integration contract | 64 KiB limits, RFC9457 behavior, no-store, idempotency, network switch/logout, keyboard/accessibility and browser permission flows |

### SSRF boundary is Circula's responsibility

Only HTTPS subscription endpoints with an explicitly approved provider-host policy; reject userinfo/fragments, unusual ports and malformed names. Disable redirects and environment proxy inheritance. Resolve and validate every actual dial target, reject loopback/private/link-local/multicast/reserved addresses including IPv4-mapped IPv6, and bind the connection to the checked address while preserving TLS hostname verification. A one-time string or DNS check is inadequate against rebinding. Bound resolution/dial/request duration, payload and response bytes. Test redirects, mixed DNS answers, encoded hosts, DNS rebinding, proxy environment, IPv6, cancellation and provider error handling through injected resolver/dialer fixtures; no real user subscriptions needed.

### Durable send limitations

A provider may accept a message before the process crashes and before the outbox acknowledges it. Thus one logical outbox event with bounded **at-least-once** delivery is the achievable contract; do not advertise exactly-once SMTP/Web Push. Ledger posting must never be retried by a notification retry. An opt-out committed before the dispatch authorization gate prevents a new send; a message already accepted by a provider cannot be recalled. Distinguish this from the spec's intent that revocation stops subsequent sends.

### Completion gates remaining after automated module tests

Full source/advisory/license audit of selected dependency graph; pinned toolchain build; real browser Web Push matrix and permission/logout behavior; no-JS/accessibility/manual screen-reader checks; production service/store adapter atomicity and current-auth tests; operational secret rotation, backup/restore and retention qualification; Circula core release qualification and individual integration approval. A fake transport or in-memory outbox does not satisfy the complete feature.

## Fresh-agent handoff

Implement the isolated SQLite repository and transport adapters first, then endpoint/auth adapters and acceptance harness. Follow the pinned official source APIs; do not blindly copy README examples (notably unconstrained default HTTP client). Update README, feature manifest, validation and requirement coverage honestly. Maintain HOLD throughout. No new runtime service is mandatory. Do not merge or publish a release.
