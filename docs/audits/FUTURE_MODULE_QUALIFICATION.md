# Future module qualification

## Outcome and scope

All **15 isolated future modules** were inspected at pinned branch heads and tested with Go 1.26.11. Their final default race suites and `go vet` checks pass. The five modules with a test-only SQLite driver also have passing whole-module tagged race runs. Those are bounded implementation results, not feature-completion or core-release qualification: **all 15 remain disabled and HOLD, and zero production integrations are qualified by this audit**.

This follows the [107-ref branch inventory](BRANCH_CONSOLIDATION_MATRIX.md), whose starting main is `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Each module was exported from its exact source commit into an isolated snapshot; nested `go.mod` boundaries mean the root application's `go test ./...` does not execute these suites. The [machine-readable evidence](FUTURE_MODULE_QUALIFICATION.json) records source SHAs, commands, environments, outputs, timings, original manifests, failed attempts and repaired-source results. Source links below are immutable commit views; local repair SHAs are explicitly identified because publication can create a different commit with the same tree.

## Observed verification

| Gate | Result | Scope and limitation |
|---|---|---|
| Default `go test -race -count=1 -cover -timeout=90s ./...` | 15/15 final PASS | Run separately inside each nested module. Favorites' initial outer timeout was resolved by a full rerun. Default suites omit SQLite tests in five modules. |
| `go vet ./...` | 15/15 PASS | Original source snapshots; repaired notifications also passes. |
| `gofmt -l .` | 14/15 originally clean | `notifications/targets.go` emitted a filename despite gofmt exit 0; repaired on the isolated notifications branch. |
| Explicit SQLite-tagged race suite | 5/5 final PASS | Favorites, QR, discovery, notifications and onboarding. Standard command and compiler flags are below. |
| PWA `node --test src/*.test.mjs` | 22/22 PASS | Node v24.22.3; strict request/response caching and bounded asset verification. |
| PWA real-browser harness | BLOCKED | Playwright 1.62.1 with Chromium 156.0.8078.79 stopped before assertions because the sandbox denied its UNIX socket. No browser/accessibility pass is claimed. |
| Feed repair race + fuzz | PASS | 98.8% root-package coverage; 66,869 fuzz executions in 21.074 seconds; independently reviewed and race-tested. |
| Notifications repair, including SQLite | PASS | 71.0% root-package coverage with real test SQLite; independently reviewed. SMTP/Web Push transport packages were not exercised. |

The database command was executed with `GOTOOLCHAIN=local`, `CGO_ENABLED=1`, `GOMAXPROCS=2`, `GOGC=50` and `CGO_CFLAGS='-O0 -g0'`:

```sh
go test -race -p=1 -count=1 -tags=sqlite_cgo_test -cover -timeout=90s ./...
```

The lower C optimization flag addresses compiler memory pressure; Go race instrumentation remains enabled. An earlier default-optimization SQLite build was OOM-killed. The first serialized favorites/QR aggregate runs then exceeded a 240-second outer wall-clock cap during a broader runtime stall, after their database packages had passed. Their remaining web packages passed separately, followed by successful whole-module reruns in 14.944 and 13.287 seconds respectively. These failed/blocked attempts remain in the evidence; they are not relabelled as passes. The other original tagged aggregate runs completed in 32.797 seconds (discovery), 18.293 seconds (notifications) and 31.071 seconds (onboarding), including build time.

Coverage is package-specific and is not averaged across unrelated modules. Packages without tests report 0%; the statements below do not qualify real service delivery, browser behavior or production adapters.

## Pinned module and test matrix

All rows have final default race and vet passes. Every disposition is **continue isolated development; do not integrate or enable from these results alone**.

| Module | Source branch and immutable commit | Default root-package coverage | Explicit database result |
|---|---|---:|---|
| FNET-201 — matching | `future/fnet-201-matching-20261116` · [`12850290c6bc`](https://github.com/djshellshoxxx/circula/tree/12850290c6bca7720853c33689662d751aced160/future/matching) | 91.7% | Not applicable: no tagged SQLite suite in this source |
| FNET-202 — favorites | `future/fnet-202-favorites-20261116` · [`efc9d696eea2`](https://github.com/djshellshoxxx/circula/tree/efc9d696eea2dd3260ecb6c5091802ef31e20572/future/favorites) | 90.1% | PASS: SQL store 81.8%; HTTP 88.3%; CLI 87.3% |
| FNET-203 — receipts | `future/fnet-203-receipts-20261116` · [`bd2fbdbd3adf`](https://github.com/djshellshoxxx/circula/tree/bd2fbdbd3adf77b27fa12335ff2827359423aea3/future/receipts) | 92.9% | Not applicable: no tagged SQLite suite in this source |
| FNET-204 — feed | `future/fnet-204-feed-20261116` · [`5d3819c3b3af`](https://github.com/djshellshoxxx/circula/tree/5d3819c3b3af031b75eceb5251da0226a4ecee77/future/feed) | 82.1% original; 98.8% repaired | Not applicable: no tagged SQLite suite in this source |
| FNET-205 — analytics | `future/fnet-205-analytics-20261116` · [`0b2965101e1d`](https://github.com/djshellshoxxx/circula/tree/0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f/future/analytics) | 94.7% | Not applicable: no tagged SQLite suite in this source |
| FNET-206 — areas | `future/fnet-206-areas-20261116` · [`8b444b20139e`](https://github.com/djshellshoxxx/circula/tree/8b444b20139e3e687db494b3fe36bb3643148e5b/future/areas) | 74.3% | Not applicable: no tagged SQLite suite in this source |
| FNET-207 — recurring | `future/fnet-207-recurring-20261116` · [`0437e54897c6`](https://github.com/djshellshoxxx/circula/tree/0437e54897c6d9cfb3f2577807b82dd038493850/future/recurring) | 95.5% | Not applicable: no tagged SQLite suite in this source |
| FNET-208 — governance | `future/fnet-208-governance-20261116` · [`b4f511372ec2`](https://github.com/djshellshoxxx/circula/tree/b4f511372ec26dc63d005bb50d87c56e18fc1a88/future/governance) | 87.0% | Not applicable: no tagged SQLite suite in this source |
| FNET-209 — federation | `future/fnet-209-federation-20261116` · [`1b3c92e03007`](https://github.com/djshellshoxxx/circula/tree/1b3c92e03007e35e3f190e719a1ea78a807d4102/future/federation) | 81.0% | Not applicable: no tagged SQLite suite in this source |
| FUX-101 — pwa | `future/fux-101-pwa-20261116` · [`a6326b8725fe`](https://github.com/djshellshoxxx/circula/tree/a6326b8725fe6d7270997b4a284cbbc129022dd1/future/pwa) | 97.5% | Not applicable: no tagged SQLite suite in this source |
| FUX-102 — qr | `future/fux-102-qr-20261116` · [`9db8a323d7dc`](https://github.com/djshellshoxxx/circula/tree/9db8a323d7dcf430d1a58a4a4d10ca3628aeb401/future/qr) | 13.6% | PASS: root 84.5%; HTTP 84.1%; CLI 82.7% |
| FUX-103 — discovery | `future/fux-103-discovery-20261116` · [`34d0f889631e`](https://github.com/djshellshoxxx/circula/tree/34d0f889631e2d133149a89a276cf9c71fd261a9/future/discovery) | 26.8% | PASS: root 85.0%; SQL store 28.1%; HTTP 88.7% |
| FUX-104 — notifications | `future/fux-104-notifications-20261116` · [`3bdf8e92d84a`](https://github.com/djshellshoxxx/circula/tree/3bdf8e92d84af2c660c615bcf956cf76752e86ae/future/notifications) | 1.9% original; 7.2% repaired | PASS: original root 70.4%; repaired root 71.0%; transports 0% |
| FUX-105 — messaging | `future/fux-105-messaging-20261116` · [`7f92251cdb0d`](https://github.com/djshellshoxxx/circula/tree/7f92251cdb0d6ed5d9559730045a021a173a278b/future/messaging) | 100.0% | Not applicable: no tagged SQLite suite in this source |
| FUX-106 — onboarding | `future/fux-106-onboarding-20261116` · [`473cab2fb45a`](https://github.com/djshellshoxxx/circula/tree/473cab2fb45a7b67ea0001f9fdc7833ffda45eb9/future/onboarding) | 82.3% | PASS: root 90.1%, including durable store and isolation tests |

## Actual implementation and concrete completion work

Several old manifests still describe their initial pure-function milestones. Source inspection found substantially later implementations in favorites, QR, discovery, notifications and onboarding. For example, those branches now contain stores and HTTP handlers; onboarding includes an isolated practice server; notifications includes real transport adapters. Their old statements that persistence, transport or third-party dependencies do not exist must not be used as current evidence. The notifications repair updates its own README and manifest; this table records the actual scope of every source snapshot without silently rewriting historical validation claims.

| Module | Implemented in the inspected source | Remaining acceptance work before core integration |
|---|---|---|
| FNET-201 — matching | Category-based, bounded offer/request matching with bilateral exclusions; pure projection. | Persist interests/preferences; define ranking and per-member caps; authorize current source snapshots; add accessible suggestions and acceptance tests. |
| FNET-202 — favorites | Owner-scoped collections and removal, service, shared memory/SQL store contracts, draft/partner selection, CLI, JSON and server-rendered HTTP handlers. | Qualify a current core authorization/visibility adapter, per-network opt-in and durable migrations/restore; complete real-browser keyboard/accessibility and replay/revocation acceptance. |
| FNET-203 — receipts | Read-only TXT/JSON projections limited to explicit transaction parties; no signed certificate or ledger write. | Supply authoritative snapshot/redaction adapters and accessible retrieval; resolve receipt signing/verification policy before adding signatures. |
| FNET-204 — feed | Listing/announcement projection with permission callback and stable keyset order. PR67/80 consolidation adds expiry, key IDs, rotation, viewer/filter binding and HMAC cursor verification; new repairs reject conflicting source snapshots and unconfigured codecs. | Implement persistent snapshot/high-water semantics and source joins, mute/follow/preferences, moderation/deletion expiry and no-JS paging; qualify races and decide key custody/rotation/TTL. Preserve the no-trade-event scope. |
| FNET-205 — analytics | Bounded aggregate computation, checked integer gross-turnover arithmetic and suppression below a minimum cohort of five. | Define fixed reporting periods, reversal accounting and differencing protections; supply authorized snapshot/export adapters and accessible reporting. A small-cohort threshold alone is not a complete privacy policy. |
| FNET-206 — areas | Coarse service-area labels, validation and resolution; no GPS coordinate collection. | Add durable authorized area CRUD, listing associations and accessible selection; settle visibility/coarseness policy and qualify deletion/restore behavior. |
| FNET-207 — recurring | Daily reminders in an IANA timezone with DST handling, first occurrence for repeated local times and no catch-up posting. | Implement persistent schedules and atomic firing dedupe/restart tests; complete required weekly/monthly rules and reminder/draft UI. It must never automatically post ledger trades. |
| FNET-208 — governance | Pure advisory ballot eligibility/tally logic; the tally does not execute policy changes. | Persist eligibility snapshots and unique ballots under concurrency; define participation/closing rules; add audit/export and accessible voting without granting ledger or administrative authority. |
| FNET-209 — federation | Strict signed public-card payloads with Ed25519 domain separation, audience and expiry checks; no inter-network balance transfer. | Define consent/key trust/revocation and durable replay policy; qualify transport, key custody, current authorization and external schema interoperability. Preserve network-isolated ledgers. |
| FUX-101 — pwa | Standalone install/offline shell, bounded verified static assets and strict request/response cache policy, JS unit tests and a real-browser harness. | Complete real-browser install/update/offline/cache-clear acceptance and platform matrix; qualify route adapters and prove authenticated data/actions never enter offline cache. Browser execution here was blocked before assertions. |
| FUX-102 — qr | Durable token service with SQL tests, QR renderer/scanner assets, CLI, web handlers, OpenAPI and a browser-qualification harness. | Qualify real-camera and no-camera flows, browser permissions/accessibility, expiry/revocation races and production authentication/rate-limit/migration adapters; review vendored scanner and rendering dependencies. |
| FUX-103 — discovery | Taxonomy/media service, image sanitizer, signed cursor handling, SQL store, CLI and web handlers. | Qualify current core authorization/visibility and upload/storage adapters; complete real-browser accessibility, deployment resource limits, moderation/deletion/backup lifecycle and dependency checks. |
| FUX-104 — notifications | SQL preferences, encrypted targets, confirmation/unsubscribe tokens, transactional outbox, retry/quiet-hour dispatch, current-directory checks and SMTP/Web Push transports. New repairs make target AAD unambiguous and reject invalid UTF-8 identity normalization. | Qualify production source-transaction/current-session adapters, worker crash/restart, real transport/egress and credential lifecycle; complete bounce handling, operator CLI and accessible/browser preference flows. Transport packages had no executed tests. |
| FUX-105 — messaging | Participant-only access policy, bounded literal UTF-8 text and a 30-day purge-deadline primitive. | Implement durable messages and idempotency/current participant checks; define legal-hold/retention interaction and abuse controls; add accessible UI and race/backup/erasure acceptance. |
| FUX-106 — onboarding | Versioned durable checklist, memory and SQL store contracts, admin copy and aggregate metrics, server-rendered HTTP/CSRF, plus a separate-origin memory-only synthetic practice world with isolated IDs/cookies and bounded sessions. | Qualify production current-authorization/session and migration adapters; run real-browser keyboard/accessibility/practice-origin acceptance, configure distinct production/practice host and cookie settings, and qualify retention/restore. Standalone database tests include a production-database non-mutation fixture. |

## Completed isolated repairs

### Feed: consolidate PR #67 and PR #80 without losing history

The local consolidation preserves module source `5d3819c3b3af031b75eceb5251da0226a4ecee77`, PR #67 source `7a528c611bffdf92ef9f8cbddb98e8b088eb5036` and PR #80 source `0952c94a5dbab4044d7673ce9d51cec9063ab97f`. PR #80 already contains the earlier source heads. The combined cursor implementation now has expiry, signing-key IDs and rotation, viewer/filter/network binding, domain separation and strict token verification.

Independent review reproduced two additional defects. Conflicting duplicate source snapshots selected whichever copy arrived first; a visible/active copy could survive a hidden/inactive duplicate. The repair refuses contradictory owner, creation-time, visibility or active-state snapshots while retaining equivalent timestamp deduplication and network isolation. A zero-value exported cursor codec also panicked in `Seal`/`Open`, and a terminal page could bypass configuration validation. All entry points now fail closed for nil or zero-value codecs. The new regressions fail on the unchanged consolidated source and pass after repair.

Source repair commit: local `beb47d91a8151653b476cf7b0f4535b0e9c4d450`. A separate reviewer inspected the complete module and reran the race suite successfully (98.8%). The fuzz smoke executed 66,869 inputs without failure. The branch then merged published core audit candidate PR #90 at `9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c`; its refreshed race suite passes (1.862 seconds, 98.8%). The initial traceability check correctly reported new pagination citations; the checked-in generator reconciled API-034/API-035 and the repeat check passes. No requirement's completion status was changed by those source citations.

Current local branch head: `6aa06e97c9ac6d8a83f60cafad34401fa999051a`, on `audit/future-feed-cursors-20261119`. The proposed publication commit has original PR #80 first parent and published PR #90 second parent. Its tree is `2ffab7ac14c1617b233e5baac47dd022dc2fcd5a`, derived from the PR #90 tree plus the feed module, a dedicated `.github/workflows/future-feed.yml`, regenerated traceability, and reviewed monetary-guard follow-up `d627af48ff4042db73adac9a72f6208db268a924`. That follow-up permits exact `big.Int` aggregation while rejecting floating-point/rational money APIs; its focused race checks pass in 2.450 seconds. The workflow uses the core Go pin and validates the disabled/HOLD and standard-library module boundary, formatting, vet, race tests and cursor fuzzing. Existing core workflows remain intact. Remote workflow execution is a publication gate, and the future branch depends on PR #90's independent core qualification. This merge direction never places future code on main.

### Notifications: exact target and token identities

At original source `3bdf8e92d84af2c660c615bcf956cf76752e86ae`, accepted actor scopes `(network|extra, member)` and `(network, extra|member)` produced identical authenticated data for encrypted email/push targets. A second defect allowed JSON to replace invalid UTF-8 identity bytes with U+FFFD before dedupe hashing or signed-token creation. These are reproducible defects in the isolated module's accepted-input contract; this audit does not claim a deployed core exploit.

Local repair `b221e1bd1c4acc500be720d5668a4e011ac5d7e4` retains the exact `ntf/v1` binding for pipe-free tuples and uses a disjoint `ntf/v2` namespace with 64-bit byte-length prefixes when any field contains a separator. All sealing, opening and rotation paths use the same helper. There is no fallback to ambiguous legacy AAD. Actor/Event validation, the exported dedupe helper and every signed payload string reject invalid UTF-8 before encoding; ordinary valid Unicode is preserved.

Previously ambiguous legacy blobs now fail closed. A push re-registration reseals its target. Email recovery requires `RevokeEmailTarget`, then `AddEmailTarget` and address verification; simply adding the same address can take a fast path without resealing. The module's documentation explains this compatibility boundary and the token-key invalidation needed if an experimental deployment previously minted links from invalid identities. Safe legacy blobs and normally encoded tokens retain their existing format.

Both defect groups have fail-before regressions. Default race, vet and formatting pass; the explicit SQLite-tagged race suite passes at 71.0% root-package coverage. An independent reviewer approved the source and tests for isolated development. The branch now also merges published PR #90 at `9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c` and includes reviewed monetary-guard follow-up `d627af48ff4042db73adac9a72f6208db268a924`. The refreshed tagged race suite passes at the same coverage in 1.432 seconds. Current local head is `ab3f260f2067abb918688f66b11b183878597384`, with tree `865d6bc02d52958c41a8d4c90fb3cd2ced32bf41`; the publication parents preserve original notifications source `3bdf8e92d84af2c660c615bcf956cf76752e86ae` first and published PR #90 second.

Dedicated `.github/workflows/future-notifications.yml` uses the core Go pin, checks the disabled/HOLD and module boundaries, verifies checksums and formatting, and explicitly vets/race-tests with `sqlite_cgo_test`. It also configures the pinned dependency vulnerability check, whose provider outcome remains pending rather than locally claimed. Local YAML/shell, action-pin, module-boundary, checksum, tagged-vet and regenerated-traceability checks pass. Core qualification and the future acceptance work remain separate gates. No transport is enabled and no future module is placed on main.

## Preservation and next decisions

No original branch was deleted or force-pushed, and no original PR state was changed by this worker. Publication is a separate parent-controlled step; remote commit IDs and actual check outcomes must be recorded by the consolidation report rather than inferred from the local SHAs above.

The most immediately useful completed work is the feed consolidation and notifications identity repair. Remaining work follows the per-module table: implement missing persistence/adapter or interaction scope on each isolated branch, then qualify the relevant tagged database suite and real-browser/external-service acceptance. Existing tests are evidence for their named implementation boundaries. They do not select product policy, authorize a new deployment, turn a future feature on or replace the core release gates.
