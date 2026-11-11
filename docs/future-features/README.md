# Circula future usability and network improvements

**Proposal only.** These 15 separate specifications are for planning after the current beta candidate; none states that its feature is shipped. Baseline: PR #31 head `bffcede8c7dc`. Date: 2026-11-10 (America/Vancouver).

## Six member-facing usability improvements

- [FUX-101 — Installable mobile-first PWA](01-installable-pwa.md) · phase 1 · High
- [FUX-102 — QR identifiers and one-time trade requests](02-qr-trading.md) · phase 1 · High
- [FUX-103 — Enhanced marketplace and discovery](03-marketplace-discovery.md) · phase 1 · High
- [FUX-104 — Opt-in email and web push notifications](04-email-push-notifications.md) · phase 2 · High
- [FUX-105 — Private member-to-member messaging](05-private-member-messaging.md) · phase 2 · Medium
- [FUX-106 — Guided onboarding and practice network](06-onboarding-practice.md) · phase 1 · High

## Nine network and community improvements

- [FNET-201 — Advisory trade matching](07-smart-trade-matching.md) · phase 2 · High
- [FNET-202 — Saved trading partners and quick repeat](08-saved-trading-partners.md) · phase 1 · High
- [FNET-203 — Shareable, verifiable trade receipts](09-digital-trade-receipts.md) · phase 1 · High
- [FNET-204 — Community activity feed](10-community-activity-feed.md) · phase 2 · Medium
- [FNET-205 — Aggregate credit-circulation analytics](11-credit-circulation-analytics.md) · phase 3 · Medium
- [FNET-206 — Coarse regions and privacy-safe local discovery](12-coarse-geographic-areas.md) · phase 2 · Medium
- [FNET-207 — Recurring trade templates and reminders](13-recurring-trade-templates.md) · phase 2 · Medium
- [FNET-208 — Community governance and member polls](14-community-governance.md) · phase 3 · Medium
- [FNET-209 — Opt-in cross-network federation](15-opt-in-network-federation.md) · phase 4 · Future

## Research and separate attractiveness backlog

- [Upstream porting research: candidates, license and reuse gates](UPSTREAM_PORTING_RESEARCH.md)
- [Additional adoption-focused proposals](ADOPTION_IDEAS.md)

## Implementation governance

1. Complete and qualify the existing beta before changing ledger or security-critical behavior; no feature here authorizes a main merge, release or deployment.
2. Prefer native Go/service/SQLite extensions and local bundled JS modules. Preserve server-rendered HTML, no-JS core flows, existing error and idempotency semantics.
3. Assign each feature to a separate branch or small PR, with one behavior/test boundary per change. An integration PR may combine only independently qualified, compatible contributions.
4. Reuse third-party *libraries* with verifiable licenses; port reference patterns across languages only through original code and tests. Do not copy AGPL/GPL code while the project's license is undecided.
5. For each feature, document existing relevant code first, pin dependency SHAs/versions, enumerate new authorization decisions, and record measurable test evidence. No new runtime service is mandatory by default.
