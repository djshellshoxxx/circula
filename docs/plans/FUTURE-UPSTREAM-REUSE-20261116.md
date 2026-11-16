# Upstream reuse research for 15 isolated features

All 15 research phases were performed by separate agents. Every linked report records inspected source/version pins, license evidence, alternatives, reuse boundaries and a requirement-level adaptation plan. Implementation and independent audit use fresh agents. These are research selections, not a claim that adoption or feature completion has passed.

| Feature and source report | Selected upstream components | Reuse boundary |
|---|---|---|
| [FUX-101 — Installable PWA](https://github.com/djshellshoxxx/Circula/blob/future/fux-101-pwa-20261116/future/pwa/UPSTREAM-RESEARCH.md) | Workbox 7.4.1 | Precache lifecycle and strategies; Circula supplies strict cache policy. |
| [FUX-102 — QR trading](https://github.com/djshellshoxxx/Circula/blob/future/fux-102-qr-20261116/future/qr/UPSTREAM-RESEARCH.md) | yeqown/go-qrcode; Nimiq qr-scanner 1.4.2 | QR encoder/scanner; atomic consent and claims remain native. |
| [FUX-103 — Marketplace discovery](https://github.com/djshellshoxxx/Circula/blob/future/fux-103-discovery-20261116/future/discovery/UPSTREAM-RESEARCH.md) | golang.org/x/image v0.48.0 | Thumbnail resampling; bounded decoding, privacy and storage remain native. |
| [FUX-104 — Email and push notifications](https://github.com/djshellshoxxx/Circula/blob/future/fux-104-notifications-20261116/future/notifications/UPSTREAM-RESEARCH.md) | go-mail v0.8.2; webpush-go pinned commit | SMTP and encrypted Web Push/VAPID; scoped outbox and egress policy remain native. |
| [FUX-105 — Private member messaging](https://github.com/djshellshoxxx/Circula/blob/future/fux-105-messaging-20261116/future/messaging/UPSTREAM-RESEARCH.md) | coder/websocket v1.8.15 | Optional bounded realtime hints; durable private messages remain native. |
| [FUX-106 — Guided onboarding and practice](https://github.com/djshellshoxxx/Circula/blob/future/fux-106-onboarding-20261116/future/onboarding/UPSTREAM-RESEARCH.md) | GOV.UK Frontend pinned 6.6.0 source | Accessible no-JS task-list and warning components. |
| [FNET-201 — Advisory trade matching](https://github.com/djshellshoxxx/Circula/blob/future/fnet-201-matching-20261116/future/matching/UPSTREAM-RESEARCH.md) | Gonum v0.17.0; x/text v0.43.0 | Bounded graph/SCC primitives and Unicode normalization. |
| [FNET-202 — Saved trading partners](https://github.com/djshellshoxxx/Circula/blob/future/fnet-202-favorites-20261116/future/favorites/UPSTREAM-RESEARCH.md) | linkding; GOV.UK Frontend pinned source | Owner-scoped relation/test patterns and accessible controls. |
| [FNET-203 — Optional digital trade receipts](https://github.com/djshellshoxxx/Circula/blob/future/fnet-203-receipts-20261116/future/receipts/UPSTREAM-RESEARCH.md) | gowebpki/jcs pinned commit; go-jose v4.1.9 | Canonicalization and separately optional JWS certificates. |
| [FNET-204 — Community activity feed](https://github.com/djshellshoxxx/Circula/blob/future/fnet-204-feed-20261116/future/feed/UPSTREAM-RESEARCH.md) | gorm-cursor-paginator v2.7.0 selected code | Composite keyset predicates/test patterns without importing GORM. |
| [FNET-205 — Credit-circulation analytics](https://github.com/djshellshoxxx/Circula/blob/future/fnet-205-analytics-20261116/future/analytics/UPSTREAM-RESEARCH.md) | Chart.js v4.6.1 | Optional local charts; exact integer accounting and privacy remain native. |
| [FNET-206 — Coarse geographic areas](https://github.com/djshellshoxxx/Circula/blob/future/fnet-206-areas-20261116/future/areas/UPSTREAM-RESEARCH.md) | Leaflet 1.9.4; optional Orb v0.14.0 | Local map/vector presentation; named areas need neither GPS nor a tile service. |
| [FNET-207 — Recurring trade templates](https://github.com/djshellshoxxx/Circula/blob/future/fnet-207-recurring-20261116/future/recurring/UPSTREAM-RESEARCH.md) | robfig/cron v3.0.1 | Pure parser/calendar iteration; durable reminders remain native. |
| [FNET-208 — Community governance](https://github.com/djshellshoxxx/Circula/blob/future/fnet-208-governance-20261116/future/governance/UPSTREAM-RESEARCH.md) | GOV.UK Frontend pinned 6.6.0 source | Accessible ballot forms; no copyleft governance-domain transplant. |
| [FNET-209 — Opt-in discovery federation](https://github.com/djshellshoxxx/Circula/blob/future/fnet-209-federation-20261116/future/federation/UPSTREAM-RESEARCH.md) | go-jose v4.1.9 | Signed discovery envelopes; trust, replay and transport remain native. |

Code will remain on the 15 independent future branches. Nothing is integrated or enabled in main. Preserve applicable licenses, transitive bundled notices and provenance. Root Circula's license is not changed by this work. Mature libraries supply reusable mechanisms, not Circula's ledger, authorization, consent or privacy policies.

See each feature's README, feature.json, VALIDATION.md and (when available) AUDIT.md for actual implementation and qualification status. Do not substitute this research table for those records.
