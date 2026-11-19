# Circula future usability and network improvements

**Proposal only.** These 15 separate specifications are for planning after the current beta candidate; none states that its feature is shipped. Baseline: PR #31 head `bffcede8c7dc`. Date: 2026-11-10 (America/Vancouver).

The files below preserve the original November 10 proposals. For the later FNET-203 trade receipt design, see the [optional receipt and certificate specification](../specs/future/FNET-203-optional-trade-receipts-certificates.md) and its [engineering companion](../specs/future/FNET-203-engineering.md).

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

## Module branches (status 2026-11-18)

Each module is developed only on its own `future/*` branch, under `future/<module>/` with its own
`go.mod`. No branch touches root `go.mod`, `internal/`, `cmd/` or migrations, and `main` contains no
`future/` directory. Every `feature.json` reads `"enabled": false`, `"integration": "HOLD"`. These
branches are not merged into `main`; they fork from `main@2eacf77` and need a rebase before any
integration. Statuses come from each branch's `feature.json`, not from a release qualification.

| Module | Branch | Tip | Status |
|---|---|---|---|
| FUX-101 | `future/fux-101-pwa-20261116` | `a6326b87` | isolated implementation tested, Chromium passed, platform qualification pending |
| FUX-102 | `future/fux-102-qr-20261116` | `9db8a323` | initial core tested, feature incomplete |
| FUX-103 | `future/fux-103-discovery-20261116` | `34d0f889` | initial core tested, feature incomplete |
| FUX-104 | `future/fux-104-notifications-20261116` | `3bdf8e92` | initial core tested, feature incomplete |
| FUX-105 | `future/fux-105-messaging-20261116` | `7f92251c` | initial core tested, feature incomplete |
| FUX-106 | `future/fux-106-onboarding-20261116` | `473cab2f` | initial core tested, feature incomplete |
| FNET-201 | `future/fnet-201-matching-20261116` | `12850290` | initial core tested, feature incomplete |
| FNET-202 | `future/fnet-202-favorites-20261116` | `efc9d696` | initial core tested, feature incomplete (includes merged PR #66) |
| FNET-203 | `future/fnet-203-receipts-20261116` | `bd2fbdbd` | initial core tested, feature incomplete |
| FNET-204 | `future/fnet-204-feed-20261116` | `5d3819c3` | initial core tested, feature incomplete; draft PR #67 (signed feed cursors) open against it |
| FNET-205 | `future/fnet-205-analytics-20261116` | `0b296510` | initial core tested, feature incomplete |
| FNET-206 | `future/fnet-206-areas-20261116` | `8b444b20` | initial core tested, feature incomplete |
| FNET-207 | `future/fnet-207-recurring-20261116` | `0437e548` | initial core tested, feature incomplete |
| FNET-208 | `future/fnet-208-governance-20261116` | `b4f51137` | initial core tested, feature incomplete |
| FNET-209 | `future/fnet-209-federation-20261116` | `1b3c92e0` | initial core tested, feature incomplete |

The module catalogue on `future/module-catalog-20261116` (`dfa4779d`) predates the later module
commits; its per-branch SHAs and commit counts are stale. Review notes for PR #67 are in
[the PR consolidation record](../plans/PR-CONSOLIDATION-2026-11-17.md).

## Implementation governance

1. Complete and qualify the existing beta before changing ledger or security-critical behavior; no feature here authorizes a main merge, release or deployment.
2. Prefer native Go/service/SQLite extensions and local bundled JS modules. Preserve server-rendered HTML, no-JS core flows, existing error and idempotency semantics.
3. Assign each feature to a separate branch or small PR, with one behavior/test boundary per change. An integration PR may combine only independently qualified, compatible contributions.
4. Reuse third-party *libraries* with verifiable licenses; port reference patterns across languages only through original code and tests. Do not copy AGPL/GPL code while the project's license is undecided.
5. For each feature, document existing relevant code first, pin dependency SHAs/versions, enumerate new authorization decisions, and record measurable test evidence. No new runtime service is mandatory by default.
