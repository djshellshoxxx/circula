# Fifteen future modules: technical completion and remaining work

Assessment date: 2026-11-19. Scope: the fourteen immutable module snapshots recorded in [MODULE_QUALIFICATION.json](MODULE_QUALIFICATION.json), the reviewed recurring candidate, and the separately identified notification transport and federation signing repairs. This is a source/specification assessment, not an authorization to integrate, enable or release a feature.

## Recovery correction — qualify the backlog before reimplementation

After this published-snapshot assessment was written, the referenced continuation record yielded the SHA-verified Circula-Future-Recovery.zip v2 archive, containing 48 Git refs, 164 evidence files and substantial later unpublished implementations for all fifteen modules. In particular, it preserves recurring a755732a02dfe0eaa884691d59a6f748f96039e7 and federation dc59d9b8349cebefb35278bd37ae5e4f207dd833. See [ARCHIVE_RECOVERY.md](ARCHIVE_RECOVERY.md) and its JSON companion for the verified inventory and containment comparisons.

**All “unimplemented,” “remaining engineering” and “independently actionable” source-absence statements below apply only to the specific published/tested snapshots assessed here. They are not claims that the functionality is absent from all saved work.** Archive containment, source reconciliation, current ownership and fresh qualification must precede any reimplementation. Preserve useful recovered services, persistence, interfaces and tests, together with the independently reviewed 2026-11-19 repairs; do not overwrite either line of work or merge unfinished archived commits merely because historical logs report passes.

The exact tests and narrow repair findings recorded below remain valid for their identified source trees. Archived evidence is historical until the recovered candidate, its current dependencies and any composed fixes are independently reviewed and rerun. This report's original module sections are retained as a published-snapshot assessment while read-only recovery reconciliation proceeds; the archive does not itself establish full feature completion, current CI qualification or permission to integrate.

## Findings that determine the disposition

**None of the fifteen modules has evidence satisfying its full intended definition of done.** That does not mean all unfinished work is blocked by policy. Several small modules still lack substantial, independently implementable services, persistence and tests. Other modules already contain broad standalone implementations and mainly need accurate documentation, additional acceptance evidence and a qualified Core adapter.

All fifteen audited manifests retain enabled=false, integration=HOLD and an empty depends_on_future_features list. A future feature therefore does not need another unfinished future feature merely to continue isolated engineering. Membership, block policy, notifications, listing snapshots and transaction access can be injected through narrow interfaces with synthetic fixtures. Production registration, shared migrations and host authorization remain separate integration work.

The assessment uses three distinct categories:

| Category | Meaning | Examples |
| --- | --- | --- |
| Unimplemented, independently actionable engineering | The module's specification/research already gives enough direction to build and test an isolated candidate. HOLD does not prohibit this work. | Recurring firing persistence and race tests; advisory ballot persistence; ordered private messages; matching preferences; named-area CRUD; federation replay/queue state. |
| Concrete design or policy decision | A particular behavior needs an explicit decision or candidate ADR; the dependency applies to that behavior, not automatically to the whole feature. | Analytics repeated-release privacy; production certificate key trust; binding governance rules; legal-hold deletion; federation settlement. |
| Integration, ownership or external qualification gate | Work needs the eventual stable Core contract, another worker's handoff, real devices/providers, independent approval or deployment configuration. | API-036 and FNET-204 active ownership; same-transaction proposal adapter; real mobile install/Web Push tests; owner-selected activation. |

The largest reporting risk is stale status text. Favorites, QR, discovery and onboarding still have README/manifest passages describing only their original small primitives. Their source and executable tests contain materially more functionality. This report credits that functionality without treating a standalone fixture as a production adapter or marking unperformed browser/manual tests passed.

“Initial → assessed final” below means the scope at the start of this consolidation pass and the changes made or verified during it. Existing expanded module implementations were not written by this master pass. Their earlier minimal milestones are mentioned only where necessary to explain stale documentation.

## Exact tested source and evidence limits

Each source link pins a complete immutable commit. The fourteen-snapshot receipt records commands, exit codes, elapsed time, output and manifest data. Those checks used Go 1.26.11 on Linux/amd64 with GOTOOLCHAIN=local, CGO_ENABLED=1, GOMAXPROCS=1, GOGC=50, GOMEMLIMIT=512MiB and GOFLAGS=-p=1. The ordinary race command was:

    go test -race -p=1 -count=1 -cover -timeout=90s ./...

All fourteen also passed go vet, go mod verify and Go formatting. Five modules additionally passed the explicit SQLite fixture build:

    go test -race -p=1 -count=1 -tags=sqlite_cgo_test -cover -timeout=90s ./...
    go vet -tags=sqlite_cgo_test ./...

This tagged qualification is significant: the default QR, discovery and notification runs do not execute their full database/service suites. A default green run alone would materially overstate their coverage. The QR checksum command exited successfully and reported all modules verified, but its retained output also contains cache-write “no space left on device” warnings; those warnings are not erased from the receipt.

| Module | Exact qualified source | Observed automated scope |
| --- | --- | --- |
| FNET-201 matching | [12850290c6bca7720853c33689662d751aced160](https://github.com/djshellshoxxx/circula/tree/12850290c6bca7720853c33689662d751aced160/future/matching) | Linux race/vet; 91.7% root-package statements. |
| FNET-202 favorites | [efc9d696eea2dd3260ecb6c5091802ef31e20572](https://github.com/djshellshoxxx/circula/tree/efc9d696eea2dd3260ecb6c5091802ef31e20572/future/favorites) | Default and SQLite-tagged race/vet; root 90.1%, SQL store 81.8%, web 88.3%. |
| FNET-203 receipts | [bd2fbdbd3adf77b27fa12335ff2827359423aea3](https://github.com/djshellshoxxx/circula/tree/bd2fbdbd3adf77b27fa12335ff2827359423aea3/future/receipts) | Linux race/vet; 92.9% of the small receipt package. |
| FNET-204 feed | [8638f3d30f27f4882c4c6424d4c8b248208b6f3d](https://github.com/djshellshoxxx/circula/tree/8638f3d30f27f4882c4c6424d4c8b248208b6f3d/future/feed) | Linux race/vet; 98.8% of this frozen snapshot. The newer externally owned PR #98 is not covered by this receipt. |
| FNET-205 analytics | [0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f](https://github.com/djshellshoxxx/circula/tree/0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f/future/analytics) | Linux race/vet; 94.7% of the existing aggregate helper. |
| FNET-206 areas | [8b444b20139e3e687db494b3fe36bb3643148e5b](https://github.com/djshellshoxxx/circula/tree/8b444b20139e3e687db494b3fe36bb3643148e5b/future/areas) | Linux race/vet; 74.3% of validation/projection helpers. |
| FNET-207 recurring | [73a911a9bec387381633c031d8a4083c9fc724ad](https://github.com/djshellshoxxx/circula/tree/73a911a9bec387381633c031d8a4083c9fc724ad/future/recurring) | Published tree equals tested local cdcd2cb38b3af0275e5ceb58242384cfa916be45. Linux race/vet 94.4%; 3,456 independent timezone-oracle comparisons; 100,000 fuzz executions. Current publication has a separately observed Windows ordinary-test pass; Linux/macOS provider jobs were still pending at the last dedicated receipt. |
| FNET-208 governance | [b4f511372ec26dc63d005bb50d87c56e18fc1a88](https://github.com/djshellshoxxx/circula/tree/b4f511372ec26dc63d005bb50d87c56e18fc1a88/future/governance) | Linux race/vet; 87.0% of ballot/tally primitives. |
| FNET-209 federation | [059b0e4508a6c4f8e29f6a4cccd06f92b14ea115](https://github.com/djshellshoxxx/circula/tree/059b0e4508a6c4f8e29f6a4cccd06f92b14ea115/future/federation) | New signing repair: exact tree equals local db21ea0a0008309d6e8d116ccb00eb77ac6873d9. Linux race/vet 87.5%; 100,000 string-fidelity fuzz executions. Earlier fourteen-module receipt tests original 1b3c92e03007e35e3f190e719a1ea78a807d4102 at 81.0%. New provider qualification remains separate. |
| FUX-101 PWA | [f25d037c3cc53fbf1ebbf687831300be9bdb749f](https://github.com/djshellshoxxx/circula/tree/f25d037c3cc53fbf1ebbf687831300be9bdb749f/future/pwa) | Linux Go race/vet 97.5% root coverage; 22 Node tests. This receipt does not claim a browser run or qualify the newer complete PR #95 tree. |
| FUX-102 QR | [9db8a323d7dcf430d1a58a4a4d10ca3628aeb401](https://github.com/djshellshoxxx/circula/tree/9db8a323d7dcf430d1a58a4a4d10ca3628aeb401/future/qr) | Default root 13.6%; SQLite-tagged root 84.5%, web 84.1%; both race/vet suites pass. |
| FUX-103 discovery | [34d0f889631e2d133149a89a276cf9c71fd261a9](https://github.com/djshellshoxxx/circula/tree/34d0f889631e2d133149a89a276cf9c71fd261a9/future/discovery) | Default root 26.8%; SQLite-tagged root 85.0%, image sanitizer 87.7%, web 88.5%, SQL store 28.1%; both race/vet suites pass. |
| FUX-104 notifications | [13669485b641475fce526633f47055ef31997098](https://github.com/djshellshoxxx/circula/tree/13669485b641475fce526633f47055ef31997098/future/notifications) | Transport repair tree equals local 4a59bd1e740c2fcf5877de2a90594439cb9cd570. Adapter race/vet: Web Push 85.3%, SMTP 84.2%; full SQLite-tagged module race/vet pass; 10,000 Retry-After fuzz executions. Earlier snapshot 9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae had 71.0% tagged root coverage and no transport-package test execution. |
| FUX-105 messaging | [7f92251cdb0d6ed5d9559730045a021a173a278b](https://github.com/djshellshoxxx/circula/tree/7f92251cdb0d6ed5d9559730045a021a173a278b/future/messaging) | Linux race/vet; 100% of a small authorization/text/retention helper, not 100% of the messaging feature. |
| FUX-106 onboarding | [473cab2fb45a7b67ea0001f9fdc7833ffda45eb9](https://github.com/djshellshoxxx/circula/tree/473cab2fb45a7b67ea0001f9fdc7833ffda45eb9/future/onboarding) | Default and SQLite-tagged race/vet; 82.3% and 90.1% root coverage respectively. |

Coverage is a description of executed source statements, not a completeness score. Root go test ./... does not traverse these nested modules. Fuzz seed execution during ordinary tests is not equivalent to a dedicated fuzz campaign.

## Module assessments

### FNET-201 — Matching

**Initial → assessed final:** unchanged bounded advisory category matcher, independently requalified. Suggest filters network-visible, active, opted-in listings, caps input at 500 and output at 100, excludes same-member pairs, calls bilateral exclusion and returns deterministic same_category reason codes. It has no trade-posting capability. Source/spec basis: [matching module](https://github.com/djshellshoxxx/circula/tree/12850290c6bca7720853c33689662d751aced160/future/matching).

**Independently actionable work:** implement separate offer/request opt-outs, listing exclusions, owner-private durable dismissals with expiry, per-member caps, normalized term scoring and evidence thresholds, bounded rescoring on public listing changes, and a no-JS explanation/dismissal surface with synthetic adapters. The research already proposes bounded graph primitives and Unicode normalization. Optional three-way cycles must remain advisory, bounded and clearly labeled; their optional status does not excuse missing required pair/preferences behavior. Add cancellation, candidate starvation, concurrent preference changes, restart and cache-failure isolation tests.

**Concrete dependencies:** the production snapshot must obtain current membership, visibility, preferences and bilateral blocks from Core. No existing block-list service is assumed; use an injected policy interface, not a dependency on unfinished messaging. Approximate geography needs the selected privacy rule only if adopted. The module has not implemented the wider feature; no broad policy prohibition prevents the basic service/store work.

### FNET-202 — Favorites and quick repeat

**Initial → assessed final:** the expanded standalone implementation already existed and is now requalified, including its SQLite build. Source contains owner-private add/remove/rename/pin/reorder, revision checks, fresh policy lookups, bounded lists, departure cleanup, memory/SQL stores, durable draft intents, CLI and JSON/server-rendered HTML adapters. Quick repeats preserve blank amounts and generate distinct deliberate intents; retrying the same request key reuses its stored draft. Tests cover same-key races, concurrent reorder/add limits, reopened databases, hostile text, CSRF, account/network switching and disabled routes. [Source and tests](https://github.com/djshellshoxxx/circula/tree/efc9d696eea2dd3260ecb6c5091802ef31e20572/future/favorites).

**Independently actionable work:** reconcile README/manifest/validation with actual source and prepare a requirement-to-test matrix. Qualify the real browser/no-JS picker and repeat confirmation, keyboard and assistive-technology behavior; inspect untested adapter edges and operational cleanup/restore procedures. Do not list all persistence, reorder or intent idempotency as missing: they are already implemented in the isolated module.

**Concrete dependencies:** a host Authenticator, current Policy and authorized TradeSource must replace fixtures; departure cleanup must join the host transaction or deny access immediately before deferred cleanup. The draft form must feed the existing proposal/confirmation flow. Migration registration and exact Core auth/session compatibility await a separately reviewed adapter, respecting active API-036 ownership. Optional favorite state never grants credit or creates a public social graph.

### FNET-203 — Receipts and optional certificates

**Initial → assessed final:** unchanged read-only primitive requalified. It requires explicit receipt choice and active same-network parties for a completed trade, validates identities/text, and produces deterministic TXT/JSON with bounded data. It does not load authoritative ledger facts, render an authenticated download route or issue certificates. [Receipt source and newer specification snapshots](https://github.com/djshellshoxxx/circula/tree/bd2fbdbd3adf77b27fa12335ff2827359423aea3/future/receipts).

**Independently actionable work:** define and test the minimal read-only trade/ledger snapshot interface, exact amount/sequence/reversal semantics, user-initiated no-JS downloads, no-store headers, redacted errors and consent flows. The newer receipt specification explicitly makes TXT/JSON the baseline and HTML/PDF optional; the older general snapshot must not be used to declare PDF a mandatory blocker. A separately labeled experimental certificate ADR and injected signer/verifier fixture can be developed under the research plan without generating production keys. Amount handling needs exact bounds before JCS signing; do not silently round arbitrary int64 amounts.

**Concrete dependencies:** ordinary history permission is broader than this primitive's active-party rule, so actual receipt access for administrators, reviewers, departed or suspended members needs an explicit contract. Production certificate format/key trust, retirement/revocation and independent security approval remain real gates. They do not block completing a basic unsigned, non-retained receipt path. No receipt request may participate in or delay ledger posting.

### FNET-204 — Activity feed

**Initial → assessed final in this audit:** snapshot 8638f3d contains public-to-network listing/announcement projection, permission callbacks, duplicate handling, stable ordering and authenticated, expiring, viewer/filter/network-bound cursors. That snapshot is requalified; no feed code was edited by this workstream. [Frozen source](https://github.com/djshellshoxxx/circula/tree/8638f3d30f27f4882c4c6424d4c8b248208b6f3d/future/feed).

**Newer work must not be erased or appropriated:** the separately owned [PR #98](https://github.com/djshellshoxxx/circula/pull/98), observed at a8aecfe8782df4782af256bbcecdf5fd1469f783, reports a bounded SQLite reference store, private padded v3 cursors, insertion fences/tombstones and extensive SQLite/property/fuzz qualification. Those claims have their own source/evidence; this fourteen-module receipt does not execute that head. Do not duplicate its work, overwrite its paths, retarget it or treat an old helper limitation as the current author's failure.

**Remaining engineering and concrete dependencies:** production source projection and current visibility joins, preferences/mute/follow, moderation/deletion lifecycle, accessible paging and route/CLI acceptance still need explicit coverage. Atomic projection, key custody/rotation volume limits, final TTL and Core authorization costs need an adapter/operational decision. Active-worker ownership is a real coordination blocker for this agent; it is not evidence that all feed development is policy-blocked. No feed may disclose private trades or balances.

### FNET-205 — Analytics

**Initial → assessed final:** unchanged aggregate helper requalified. It counts completed-trade gross turnover and distinct traders within a half-open interval, rejects malformed/duplicate identities and overflow, and fully suppresses small cohorts below a caller threshold of at least five. It does not supply reversal-aware ledger reporting, stock distributions, dormant listings or a member/admin reporting product. [Source and research plan](https://github.com/djshellshoxxx/circula/tree/0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f/future/analytics).

**Independently actionable work:** add a bounded immutable ledger/listing snapshot contract; independently verify balanced posting pairs and reversal links; separate turnover from outstanding-balance stock; implement fixed labeled weekly/monthly periods in a configured IANA zone, member-only own trends and accessible tables/export fixtures. Add overflow, timezone, cross-network, reversal, small/complementary-bin and serialized-output tests. Optional charts do not replace these accounting and privacy requirements.

**Concrete dependencies:** the production snapshot must use Core's consistent database view and fresh scoped authorization. Repeated-release, adjacent-period differencing, approved slice dimensions and cohort policy need an explicit privacy design before disclosure; threshold-five alone is not an anonymity proof. The missing accounting/service implementation is still actionable with synthetic private fixtures. No analytics cache becomes a ledger authority, credit score or sanctions mechanism.

### FNET-206 — Coarse areas

**Initial → assessed final:** unchanged named-area validation, visible sorting and same-network resolution primitives requalified. They store no coordinates and do not implement administrative persistence or listing assignment. [Source/specification](https://github.com/djshellshoxxx/circula/tree/8b444b20139e3e687db494b3fe36bb3643148e5b/future/areas).

**Independently actionable work:** implement module-owned named-area CRUD, unique tenant-scoped identities, admin checks, listing zero-or-one assignment, deterministic disabled/deleted-area handling, accessible list/filter UI, export projection and restart/concurrency tests. Document backup/retention and correct current status text. This useful label-only path does not require GPS, polygons, a map or a new cross-feature dependency.

**Concrete dependencies:** host authorization and listing ownership must be current at assignment; shared schema/route integration needs a later adapter. Geometry, if chosen, requires explicit coarseness, bounds, antimeridian and disclosure rules. Image EXIF stripping belongs to an injected media boundary and needs actual integration evidence; this module must not claim that its label validator strips photos. Federation sharing defaults to omitting location until separately allowed. None of those optional decisions blocks named-area engineering.

### FNET-207 — Recurring reminders

**Initial → assessed final:** daily-only pure due calculation at 0437e548 became structured daily/weekly/monthly calculation with validated selectors, named-zone deterministic behavior, missing-month-day skip, DST gap skip and earliest-fold selection. Exact historical second offsets and the valid zero instant are handled; invalid UTF-8 identities and host-dependent Local are refused. Original Template layout, daily API and daily key bytes are preserved. [PR #97 and source evidence](https://github.com/djshellshoxxx/circula/pull/97).

The tests include 3,456 independent minute-oracle comparisons over six zones and calendar dates, explicit all-cadence gap/fold cases, leap/century/missing-day cases, membership/tenant boundaries and same-date key stability. A 100,000-execution property fuzz run passed. Native workflow qualification remains distinct from local evidence: Windows ordinary tests passed; Windows race is deliberately not claimed.

**Independently actionable work:** durable owner-scoped templates, template edits/pause/skip/cancel and audit, a unique logical-window firing row, atomic reminder/outbox insertion, bounded RunDue scans with continuation/fairness, same-operation membership rechecks, concurrent worker, fault injection, restart and retention/replay tests. These are specified in the research and remain unimplemented. The calendar milestone must not be called a complete recurring service.

**Concrete dependencies:** a production reminder uses the existing Core inbox through a shared transaction or durable outbox interface, not an assumed dependency on future SMTP/push. Template revision/timezone-edit policy, final quotas and retention need explicit documented candidate decisions and production review. The eventual member-confirmed draft must use the ordinary proposal service with fresh limits/terms and a distinct intent. Scheduled code must never auto-propose, auto-accept or post trades. Integration and activation remain HOLD.

### FNET-208 — Advisory governance

**Initial → assessed final:** unchanged preflight and tally primitives requalified. They check scope, current activity, snapshot eligibility, voting window, choice, duplicate existing ballots and withdrawal, then count yes/no/abstain. The in-memory duplicate check is not an atomic ballot store. [Source/research](https://github.com/djshellshoxxx/circula/tree/b4f511372ec26dc63d005bb50d87c56e18fc1a88/future/governance).

**Independently actionable work:** durable immutable eligibility snapshots, unique ballots within a transaction, proposal CRUD and moderation, withdrawal/process-dispute records, deterministic close results and separate certification, privacy-limited exports, no-JS forms, current authorization adapters and concurrent/restart tests. A read-only link to an already-existing policy version can be modeled without giving a tally operation policy-write power.

**Concrete dependencies:** quorum, binding success criteria, conflict rules and production ballot disclosure/retention require explicit decisions for the corresponding behavior. The existing plan deliberately permits advisory counts without inventing a binding majority. Therefore basic advisory persistence and UI are not blocked by monetary governance decisions. Applying a policy remains the existing separately authorized admin workflow with its own MFA, notices and audit. QuietVote is optional and is not a dependency.

### FNET-209 — Discovery federation

**Initial → assessed final:** the small draft Ed25519 envelope primitive now rejects malformed UTF-8 in all five signed strings before marshaling. Valid Unicode, literal U+FFFD, decomposed text, escaping, domain separation, keys and Verify behavior are unchanged. [PR #105](https://github.com/djshellshoxxx/circula/pull/105) contains exactly two module files and one new dedicated native qualification workflow. Local commit db21ea0a0008309d6e8d116ccb00eb77ac6873d9 and published commit 059b0e4508a6c4f8e29f6a4cccd06f92b14ea115 have whole tree 4d9f608041e65a2a1a297f17614438790240bc04.

**Reproduced finding, accurately bounded:** the original Sign succeeded on thirty malformed-field cases after JSON replaced their bytes. Verify rejected every resulting envelope through its existing exact re-encoding check. The defect is signer input/fidelity validation and false success; an accepted-identity, signature-verification or replay bypass was not reproduced. The regression retains that distinction. Committed-head race passed in 1.071s with 87.5% coverage; vet passed; 100,000 bounded fidelity fuzz executions passed in 5.682s. A separate reviewer found no blocking source/workflow finding. Remote provider tests are a separate pending gate.

**Independently actionable work:** finish a versioned discovery-only DTO/experimental wire ADR, injected peer trust and owner consent, bounded durable inbox/outbox, atomic event-hash replay/equivocation storage, per-listing revision/tombstone ordering, expiry/GC rules, SSRF-safe transport fixtures, retries/rate limits and partition/restart tests. Current helpers contain no listing content, durable replay state or operational transport.

**Concrete dependencies:** production peer approval, trust/key lifecycle, shared listing projection and egress configuration need reviewed contracts. The published research proposes a JWS adaptation; this narrow UTF-8 repair does not claim to complete or adopt it. Cross-network settlement is a separate excluded Phase B requiring an approved accounting/protocol model, never a prerequisite for isolated discovery-only queue tests.

### FUX-101 — PWA

**Initial → assessed final:** the existing broad isolated implementation was requalified at the frozen snapshot. It includes pinned local Workbox assets, manifest/icons/install help, strict public-asset request/response/integrity checks, no private caching or background replay, explicit opt-in Go handlers, offline-unavailable navigation, private-DOM lifecycle scrub, cross-tab sign-out locking and a loopback synthetic demo. [Source and completion matrix](https://github.com/djshellshoxxx/circula/tree/f25d037c3cc53fbf1ebbf687831300be9bdb749f/future/pwa).

**Evidence distinction:** the fourteen-module run executed Go and 22 Node tests. The separately owned [PR #95](https://github.com/djshellshoxxx/circula/pull/95) records a Chromium provider pass on its earlier candidate and later Core/action-registration refreshes; the observed latest head was f153bf5b18f933308479eea7fe75c0f60d33ca38. The old provider pass and module blob preservation do not automatically qualify that complete newer repository tree.

**Independently actionable acceptance:** current-head Chromium/Firefox/Safari and failure/lifecycle assertions, real BFCache restores, install/resume, no-JS/keyboard/zoom and automated accessibility checks can continue against the synthetic harness. Actual Android/iOS installation, OS task-switcher snapshots and manual screen-reader qualification require suitable devices/tools and observations. Lack of such observations is a qualification gap, not a product-policy decision.

**Concrete dependencies:** production adapter scope, base-path/CSP, sign-out/session events and Core tenant/CSRF/idempotency/ledger regression await stable host integration and respect active ownership. Authoritative offline trades, private caches and queued replay remain excluded. Additional user-data persistence is not required for this design.

### FUX-102 — QR trade requests

**Initial → assessed final:** the existing expanded source was requalified, including explicit SQLite tests. It already has opaque hashed requests, static aliases/rotation, expiry/revocation, durable idempotency, one-write-unit claim/propose service, printable QR encoding, manual input/scanner assets, CLI and no-JS/JSON HTTP adapters. Tests exercise concurrent scans and revoke races, rollback after proposal failure, restart/backup restore, no raw tokens at rest, schema constraints, independent PNG decoding and synthetic balanced-ledger completion. [Source and fixtures](https://github.com/djshellshoxxx/circula/tree/9db8a323d7dcf430d1a58a4a4d10ca3628aeb401/future/qr).

**Independently actionable work:** correct stale README/manifest claims that all aliases/rendering/persistence are missing; complete current camera/manual/print browser and accessibility evidence, scanner CSP qualification, source/license/dependency notices and operational secret/restore guidance. Existing same-transaction service tests must be credited rather than reimplemented.

**Concrete dependencies:** the Core interface's Propose receives the already-open DB transaction and must execute the real current membership, terms, credit/limits and rate checks there. A host public method that opens another transaction does not satisfy this contract. The synthetic fixture's two balanced entries do not prove the production adapter. Host authentication/CSRF, migrations, alias-secret custody and eventual per-network opt-in require reviewed integration. Scanning only prepares/confirms a proposal; it never constitutes acceptance or proof of payment.

### FUX-103 — Discovery and media

**Initial → assessed final:** the expanded source was already present and is now requalified with SQLite. It contains controlled taxonomy, tag moderation, coarse optional areas, availability/expiry/renewal, normalized relevance/recent search, signed bound cursors, media quotas and retention, JPEG/PNG sanitization and thumbnails, SQL/blob adapters, CLI and no-JS/JSON pages. Tests cover concurrent seventh-photo rejection, EXIF/GPS removal, malformed/oversized images, crash between metadata/blob writes, restart, tampered blobs, cursor rotation, pagination under writes and tenant authorization. [Source and tests](https://github.com/djshellshoxxx/circula/tree/34d0f889631e2d133149a89a276cf9c71fd261a9/future/discovery).

**Independently actionable work:** reconcile old prototype status text, audit low-covered repository branches, complete actual browser/gallery/alt-text/accessibility and resource-bound qualification, and document blob/database backup/restore and quota/GC operations. Taxonomy, sanitizer, renewal and media persistence are not absent merely because their original manifest says “filters.”

**Concrete dependencies:** the production listing read model must track current Core listing and owner activity; stale copied visibility is not authorization. Core search folding/order behavior and the new cursor contract need explicit compatibility tests. Final media retention, moderation and optional coarse-distance disclosure are policy choices for those surfaces. Host migration/auth/media route adoption and current ledger reference regression remain unqualified. Exact home coordinates and public member indexing remain excluded.

### FUX-104 — Notifications

**Initial → assessed final:** the initial snapshot already contained SQL preferences, encrypted scoped targets, confirmation/unsubscribe tokens, transactional deduplicated outbox, quiet-hours/retry dispatch, fresh directory checks, key rotation and SMTP/Web Push adapters. Earlier identity/AAD hardening was merged into its isolated target. The later four-file [PR #103](https://github.com/djshellshoxxx/circula/pull/103) fixed a successful Web Push response returned as a typed-nil error, pre-cap Retry-After overflow, and implicit SMTP TLS with an injected TCP dialer. It added actual synthetic TLS/SMTP/HTTP, Web Push encryption/decryption, egress, cancellation, bounds and redaction tests. Remote head 13669485b641475fce526633f47055ef31997098 equals the fully reviewed local transport tree 198d2e6d9e40922080eb91f3ab73d011808feea9.

**Independently actionable work:** accessible/no-JS preference and subscription management, operator CLI, crash/restart and lease-recovery qualification, and richer operational bounce/inspection/restore evidence. Dispatch already handles hard/soft bounce and gone-target outcomes in fixtures; production bounce ingestion/provider behavior still needs qualification. Do not describe all bounce handling or all transports as absent. Current synthetic transport coverage is no longer zero.

**Concrete dependencies:** the real outbox enqueue must share the authoritative source-event transaction; delivery is bounded at-least-once, not an exactly-once external send guarantee. Production membership/session revocation, browser permission/logout, SMTP credentials, VAPID/encryption key custody, approved egress hosts and actual providers require host/operational qualification. These dependencies do not block local CLI, UI or crash-state tests. Provider outages must never undo or repeat ledger entries.

### FUX-105 — Messaging

**Initial → assessed final:** unchanged small policy module requalified. It authorizes active same-network thread participants, rejects blocked/closed states, bounds valid UTF-8 literal text and calculates an inactivity deadline of at most thirty days. It has no persistent threads/messages, delivery service or UI. The reported 100% coverage applies only to those helpers. [Source/specification/research](https://github.com/djshellshoxxx/circula/tree/7f92251cdb0d6ed5d9559730045a021a173a278b/future/messaging).

**Independently actionable work:** durable tenant-scoped threads/messages, monotonic per-thread sequence, idempotent sends, unread positions, current bilateral blocks, report/moderation state, bounded quotas and escaped no-JS inbox/forms. Add concurrent sends, restart, foreign participant, rate, deletion visibility and retention-boundary tests. Realtime WebSocket/SSE is an optional transport; SQLite remains authoritative and ordinary HTTP can be completed first.

**Concrete dependencies:** production participant eligibility, referenced listing/trade authority and account-state freshness need Core adapters. History access after blocking and legal-hold/dispute preservation need explicit policy; a holding policy for deletion does not prevent implementing ordered persistence and private access. V1 must disclose operator-readable server-side text; end-to-end encryption is a separate assurance milestone. Messages never authorize trades.

### FUX-106 — Onboarding and practice

**Initial → assessed final:** an expanded isolated implementation already existed and now passes default/SQLite qualification. It has versioned idempotent progress, separate completed/skipped state, memory/SQL stores with optimistic updates, bounded admin copy/links, default-off aggregate metrics, server-rendered/JSON checklist routes and translations. Its separate-origin synthetic PracticeServer accepts no production database or service handle and has separate credentials, bounded sessions, idempotent practice trades, limit/balance examples and a fixture test leaving the host database byte-identical. [Source and tests](https://github.com/djshellshoxxx/circula/tree/473cab2fb45a7b67ea0001f9fdc7833ffda45eb9/future/onboarding).

**Independently actionable work:** update the stale minimal milestone documentation, complete actual browser no-JS/keyboard/reduced-motion/account-switch/translation acceptance and user comprehension checks, and qualify backup/restore/deletion and curriculum upgrades. HTTP tests cover five guided actions and accessible markup; they are not a completed human usability or screen-reader study.

**Concrete dependencies:** the production tracker must obtain current authenticated Actor/Facts; tutorial progress cannot prove accepted terms or a real trade. Real practice-origin/cookie deployment must preserve credential separation. Optional aggregate metrics need a minimum-cohort/repeated-release privacy decision before exposure; the default-off checklist does not depend on that decision. Core session/migration adoption remains separate and cannot overlap API-036 ownership.

## What is required before declaring any module done

A complete disposition needs requirement-to-code/test evidence covering the real implemented scope, accurate module status documents, current dependency/license/advisory evidence where dependencies exist, negative tenant/authorization tests, appropriate concurrency/restart/fault/restore coverage, actual CLI/UI/API contracts, and the required browser/accessibility observations. Review of a pure helper or green default tests cannot substitute for missing feature components.

After standalone completion, the selected module still needs a narrowly reviewed adapter against the then-qualified Core, correct shared-transaction behavior, explicit network opt-in, current-auth/CSRF/idempotency/full-ledger regression, operational rollback/disable/retention procedures and the owner's individual integration/activation decision. Those gates remain distinct from continuing isolated engineering.

The implementation gaps described above belong to the dated published snapshots inspected for this report. The subsequently verified recovery archive contains durable recurrence, matching preferences/scoring, named-area service/storage, advisory poll storage, private message persistence, receipt interfaces and discovery-only federation state/transport. See `ARCHIVE_FEATURE_STATUS.md` for that recovered source, its remaining implementation and policy gates, and the disposition of each module. Current engineering work must qualify and reconcile the recovered implementations while preserving later repairs and tests. Active external feed and Core/auth work must reach a verified handoff before overlapping integration.

No production rollout, cross-network settlement, automatic trade posting, branch overwrite or merge into main follows from this report.
