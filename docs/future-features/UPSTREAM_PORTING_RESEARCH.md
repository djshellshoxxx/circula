# Upstream porting and reuse research for Circula

Date: 2026-11-10 (America/Vancouver). Research notes, **not code imports**.
Baseline: Circula draft beta integration PR #31, `claude/circula-beta-completion-1npjw6`, commit `bffcede8c7dca0d702e1a5cd5b5e8b5d363c7c3b`.
Related feature specifications: [README](README.md). Related ideas: [adoption backlog](ADOPTION_IDEAS.md).

## Executive conclusion

Circula is a single Go server, using SQLite, server-rendered HTML with progressive enhancement, an authenticated versioned JSON API, service-layer authorization, an append-only balanced ledger, network/tenant isolation and explicit consent before credit transfer. The best reuse targets are **narrow, permissively licensed modules** for QR, email, Web Push, PDF, image processing, charting, maps and PWA behavior. Avoid grafting another application's identity, database, payment or commerce engine onto Circula. For matching, governance, business credit limits and federation, only infrastructural algorithms or message formats are reusable; the actual mutual-credit rules must remain original and independently tested.

**Assessment categories**
- **A (library integration):** directly import pinned Go package or vendor self-contained browser JS, then build native Circula adapters and tests.
- **B (selective adaptation):** copy selected compatible source only after per-file license/dependency review, or reimplement a well-defined API/algorithm in native Go with independent tests.
- **C (reference only):** use user-flow/architecture inspiration; no source copying without separate license/architecture decision.
- **N (native):** Circula already has most of the required primitives; adding a dependency would make the product less maintainable.

All software licenses below are as stated by the upstream repositories at the time of research. Recheck **exact files, licenses, tags, dependencies, security advisories and full immutable source SHA** before importing. No upstream commit is yet pinned for incorporation; this document records candidates, not a supply-chain approval. Circula's own product LICENSE remains owner-undecided.

## Candidates by module and code paths to investigate

| Source | Upstream | License (reported) | Reusable capability | Fit | Scope and cautions |
|---|---|---|---|---|---|
| GoogleChrome/workbox | https://github.com/GoogleChrome/workbox | MIT | Service worker precaching, updates | A | Use only for immutable public assets; **do not cache HTML/API/account/inbox** |
| skip2/go-qrcode | https://github.com/skip2/go-qrcode | MIT | QR encoding in Go | A | Inspect `encoder.go`, `qrcode/`, `reedsolomon/`; check age/security/maintenance |
| nimiq/qr-scanner | https://github.com/nimiq/qr-scanner | MIT | Local camera scanning Web Worker and BarcodeDetector fallback | A | Bundle scanner + worker locally; test mobile permission and parser malicious data |
| disintegration/imaging | https://github.com/disintegration/imaging | MIT | Image sizing, transformation | A | Set bounding checks before decode; external file decoder behavior must be reviewed |
| blevesearch/bleve | https://github.com/blevesearch/bleve | Apache-2.0 | Advanced indexed full-text/geospatial search | B | Evaluate SQLite FTS/current query first; second index adds synchronization complexity |
| wneessen/go-mail | https://github.com/wneessen/go-mail | MIT | RFC-compatible message composition and SMTP | A | No mandatory mail provider; sender and credentials in existing secret config |
| SherClockHolmes/webpush-go | https://github.com/SherClockHolmes/webpush-go | MIT | VAPID Web Push encryption/transport | A | Endpoint SSRF checks, encrypted subscription state, opt-in preferences |
| gotify/server | https://github.com/gotify/server | MIT (software); logo CC BY 4.0 | Self-hosted notification transport | B/C | Can be separately integrated by API; avoid requiring another service in default Go binary |
| coder/websocket | https://github.com/coder/websocket | MIT (verify exact LICENSE on adoption) | Go WebSocket transport | A | Transport only; adds **zero** message privacy/encryption semantics |
| centrifugal/centrifuge | https://github.com/centrifugal/centrifuge | MIT | Realtime channels, reconnect, WebSocket/SSE | B | Consider only if base net/http/SSE inadequate; extra protocol and browser SDK |
| alphagov/govuk-frontend | https://github.com/alphagov/govuk-frontend | MIT for code (documentation separate OGL) | Accessible forms, error summaries, step-by-step patterns | B | Adapt small template/CSS/JS patterns; avoid wholesale Nunjucks/Node runtime |
| go-pdf/fpdf | https://github.com/go-pdf/fpdf | MIT | Generate PDFs in Go | A | File generator, not proof-of-payment; font assets may have separate licenses |
| chartjs/Chart.js | https://github.com/chartjs/Chart.js | MIT | Analytics charts and legends | A | Bundle locally; provide equivalent server-rendered accessible data table |
| gonum/gonum | https://github.com/gonum/gonum | Gonum BSD-style; third-party notices | Graph/numeric algorithms | B | Matching *candidates* only; apply permissions/consent; verify included files |
| Leaflet/Leaflet | https://github.com/Leaflet/Leaflet | BSD-2-Clause | Optional zoomable area map | A | Map tiles/data licensed separately; do not leak exact home location |
| robfig/cron | https://github.com/robfig/cron | MIT | Recurrence parsing, scheduling | A | Reminders and draft generation only; never auto-post money-equivalent entries |
| cloudevents/sdk-go | https://github.com/cloudevents/sdk-go | Apache-2.0 | Versioned signed-event envelope building block | B | Does not provide cryptographic identity, approvals, settlement or credit accounting |
| go-ap/activitypub | https://github.com/go-ap/activitypub | MIT | ActivityPub/ActivityStreams vocabulary for listings | B | Federation of **public-to-peer listings only**, not ledger entries |
| medusajs/medusa | https://github.com/medusajs/medusa | MIT core; enterprise source separate | Catalog/storefront/merchant UX patterns | C | TypeScript/Node ecommerce engine is not a direct transplant into SQLite/Go credit ledger |
| loomio/loomio | https://github.com/loomio/loomio | AGPL-3.0 | Decision-making UX/workflows | C | Reference only pending licensing and separate implementation review |
| decidim/decidim | https://github.com/decidim/decidim | AGPL-3.0 | Governance proposals, deliberation, voting | C | Ruby on Rails/AGPL, reference only |
| simplex-chat/simplex-chat | https://github.com/simplex-chat/simplex-chat | AGPL-3.0 | Private messenger architecture | C | Cannot drop in E2EE as a transport. Identity, key management, threat modeling and license separate |
| djshellshoxxx/quietvote | https://github.com/djshellshoxxx/quietvote | Owner project, own license/assurance not release-qualified | Optional community ballot engine | C | Future API integration only; no claims of audited anonymity or binding tallies |

Where a candidate has a non-MIT notice, re-evaluate the actual file set rather than relying solely on this table. Some repos contain generated files, bundled third-party code, logos or assets under different terms.

## Mapping to the 15 requested feature specifications

| Feature | Best candidate | Recommended reuse | What remains native in Circula |
|---|---|---|---|
| FUX-101 Installable PWA | Workbox, GOV.UK Frontend | A/B: cache and accessible mobile controls | Auth/no-store policy, logout, route selection, offline prohibition |
| FUX-102 QR trading | go-qrcode + qr-scanner | A: encoding/decoding | QR payload v1, token database, consent/replay/tenant and trade posting |
| FUX-103 Marketplace discovery | imaging, optionally Bleve; Medusa inspiration | A/B/C | Listings, network policy, moderation, private geographic areas |
| FUX-104 Email/push | go-mail + webpush-go; optionally Gotify | A | Durable event outbox, preferences, secret management, delivery state |
| FUX-105 Member messaging | coder/websocket or Centrifuge | A/B (transport only) | Messaging ACL, retention, spam, threading and independently evaluated encryption |
| FUX-106 Onboarding | GOV.UK Frontend | B: components and flow patterns | Real vs demo separation, tutorial state and consent |
| FNET-201 Trade matching | Gonum, optionally Bleve | B: graph/search algorithms | Matching policy, explanation, opt-out, privacy and no autonomous trading |
| FNET-202 Favorites | Existing Circula directory/forms | N | Private favorite table, quick-repeat confirmation, independent intent IDs |
| FNET-203 Receipts | go-pdf/fpdf | A | Authoritative receipt projection, access control, reversal links |
| FNET-204 Activity feed | Existing notifications/announcements; Gotify optional | N/B | Permission-filtered event aggregation, cursor and moderation |
| FNET-205 Credit analytics | Chart.js, Gonum optional | A/B | Ledger-derived aggregate measures and small-population suppression |
| FNET-206 Geographic areas | Leaflet | A | Admin area definitions, coarse resolution, privacy/permission checks |
| FNET-207 Recurring templates | robfig/cron | A: schedules | Durable dedupe, reminders, explicit new two-party consent |
| FNET-208 Governance | Loomio/Decidim ideas; QuietVote later | C | Scoped eligibility, tamper-evident tallies, and separate admin policy change |
| FNET-209 Federation | CloudEvents sdk-go; go-ap/activitypub for announcements | B | Trust, cryptographic peer identity, authorization and any future bridge accounting |

### Critical incompatibilities

1. **No upstream mutual-credit posting engine should replace `service/postTx`.** Any new trade path must call existing Circula services and pass its two-entry balance, credit-limit, idempotency, audit and tenant-isolation tests.
2. **PWA and push services must not cache sensitive data.** Workbox is not a substitute for a privacy/cache policy.
3. **WebSocket transport is not E2EE.** Do not claim private-from-operator messaging by installing coder/websocket, Centrifuge or Gotify.
4. **Medusa payment/order engines are not Circula trade ledgers.** Only catalog, filtering, storefront or UI patterns are relevant.
5. **AGPL source is not a drop-in module for an unlicensed Circula.** Loomio, Decidim and SimpleX are design references; copying their source needs a specific GPL-family compatibility and distribution analysis.
6. **ActivityPub and CloudEvents cannot transfer mutual credit.** Federation initially shares selected offers, not account balances or settlement authority.
7. **License notices/third-party assets are separate.** Do not silently copy logos, map tiles, fonts, datasets, or screenshots.

## Mapping to eight additional attractiveness ideas

| Adoption opportunity | Suggested sources | Reuse | Notes |
|---|---|---|---|
| Merchant directory | Circula member/listing code; Medusa catalog UX | N/C | Add merchant roles, category, opt-in public hours/area; no separate payment subsystem |
| Spending opportunities dashboard | Circula listing search + matching + Chart.js | N/A | Search offers by budget/area/category; do not promise redemption or exchange rate |
| Business storefronts | Medusa open-core merchant/catalog UX; imaging | C/A | Implement small Go-native shop pages backed by Circula listings, no checkout replacement |
| Trade coordinator tools | Gonum + existing admin directory | B/N | Coordinators only get consented, policy-scoped data; no unilateral ledger changes |
| Privacy-first profiles | Circula membership/auth and permissions | N | Design first; never assume another social app's data disclosure defaults are safe |
| Community branding | GOV.UK Frontend design patterns | B | Network-specific safe CSS tokens, uploads and accessible labels; no remote CSS |
| Referrals/invitations | Existing Circula single-use invitation code | N | Extend roles, referral provenance and abuse caps; no need to port account system |
| Vouchers/gift credits | Existing system-account grants and ledger | N | Credits issuance must be explicitly configured, fully balanced, capped and audited; not a coupon library |

### Recommended first five integration spikes

These deliver the most reusability while minimizing disruption to Circula's current beta.

1. **QR:** validate `go-qrcode` + `qr-scanner` (pinned versions, local bundles). Implement a test-only non-authoritative read/preview before adding durable QR request tokens; test copy/scan/print and replay.
2. **Receipts:** `go-pdf/fpdf` proof of concept from an immutable synthetic completed trade. Compare fields with the ledger and PDF bytes; handle font/notice license separately.
3. **Email/push:** `go-mail` and `webpush-go` send mock messages only. First implement durable local outbox, idempotent event mapping and per-member preferences; then opt-in transports.
4. **Listing photos:** `imaging` to transform hostile test images after size/decode bounds. Verify EXIF/GPS removal, ACL, limits, no remote URL fetch and backup behavior.
5. **PWA:** `Workbox` shell precache; prove authenticated responses cannot enter Cache Storage using automated browser checks before enabling install UI.

After those: improve existing listing search without Bleve first; add charts and coarse maps only with accessible table/list fallbacks; add matching and messaging only after privacy/identity policies are approved. Treat governance and federation as separate long-horizon projects.

## Port procedure required for *every* upstream

- [ ] Confirm file-level license/notice, trademark/assets restrictions and dependency licenses; log license approval and attribution in a third-party inventory before copying.
- [ ] Capture immutable full upstream Git commit SHA, tag and exact file paths; don't install unpinned `latest` or load runtime CDN scripts.
- [ ] Inspect active maintenance, security advisories, issue backlog and transitive dependencies; evaluate whether Circula's current `go.mod`/Go 1.26 and CSP allow the import.
- [ ] Choose dependency use over source copying for small Go packages; vendor JS with SRI/hash manifest and local hosting as required by production CSP.
- [ ] Write an adapter interface and acceptance tests first. Keep Circula service layer as the only authority for ledger, membership and domain rules.
- [ ] Add a test that fails against the old behavior and passes the integrated code; test invalid inputs, authorization, network isolation, idempotency, concurrency, restart and migration rollback.
- [ ] Verify implementation on Linux, macOS and Windows where applicable; test no-JavaScript fallback and accessibility.
- [ ] Record SBOM/digest/license provenance and results of source, dependency and secret scans in the PR.
- [ ] Don't automatically merge into `main`, change live deployment, or enable future features until owner scope/acceptance is approved.

## Source provenance and license inspection entrypoints

- Workbox: https://github.com/GoogleChrome/workbox ; https://github.com/GoogleChrome/workbox/blob/main/LICENSE
- go-qrcode: https://github.com/skip2/go-qrcode ; inspect repo LICENSE
- qr-scanner: https://github.com/nimiq/qr-scanner/blob/master/LICENSE
- imaging: https://github.com/disintegration/imaging
- Bleve: https://github.com/blevesearch/bleve
- go-mail: https://github.com/wneessen/go-mail
- webpush-go: https://github.com/SherClockHolmes/webpush-go
- Gotify: https://github.com/gotify/server/blob/master/LICENSE
- Coder WS: https://github.com/coder/websocket
- Centrifuge: https://github.com/centrifugal/centrifuge
- GOV.UK: https://github.com/alphagov/govuk-frontend/blob/main/LICENSE.txt
- PDF: https://github.com/go-pdf/fpdf/blob/main/LICENSE
- Chart.js: https://github.com/chartjs/Chart.js/blob/master/LICENSE.md
- Gonum: https://github.com/gonum/gonum/blob/master/LICENSE
- Leaflet: https://github.com/Leaflet/Leaflet/blob/main/LICENSE
- cron: https://github.com/robfig/cron/blob/master/LICENSE
- CloudEvents: https://github.com/cloudevents/sdk-go/blob/main/LICENSE
- ActivityPub types: https://github.com/go-ap/activitypub
- Medusa: https://github.com/medusajs/medusa/blob/develop/LICENSE
- Loomio: https://github.com/loomio/loomio
- Decidim: https://github.com/decidim/decidim
- SimpleX: https://github.com/simplex-chat/simplex-chat

The links are upstream review starting points. Do not use their current moving default branch as a reproducible source for an implementation.
