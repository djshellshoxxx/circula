# Independent analytics audit — 2026-11-16

Reviewer phase: fresh independent audit after implementation. Reviewed the full isolated module, historical specification, upstream handoff, HTTP/CLI/render/export seams and all tests. Findings below were fixed in this branch. Integration stays **HOLD**, feature disabled; no core file, route, schema, ledger writer or deployment changed.

## Findings and fixes

| Finding | Fix | Regression evidence |
|---|---|---|
| A dominant member could split turnover into small transactions and evade the per-transfer 80% threshold. Stock and dormant-offer dominance were unchecked. | Accumulate contributions per ordinary member, independently for gross and reversal volume. Suppress a band/metric when one contributor exceeds 80%, with an exact overflow-safe integer threshold. | `TestAuditDominanceCannotBeSplitAndStockComplement`, `TestAuditDominanceThresholdDoesNotRoundUp` |
| Publishing current stock alongside a suppressed latest flow could reconstruct hidden changes by comparing earlier releases. | Complementary suppression hides stock when the most recent completed flow is suppressed. Fixed per-network release granularity and cutoff remain mandatory. | `TestAuditDominanceCannotBeSplitAndStockComplement`, existing suppression tests |
| Raw source revision could be a global database sequence, revealing unrelated or hidden activity. | Return a SHA-256 fingerprint of the already privacy-filtered public release instead of the source revision. | `TestAuditRevisionOnlyUsesReleasedData` |
| Stock and offer rows had no explicit tenant binding. Invalid/duplicate or invisible offers could distort metrics. | Require network-scoped balance/offer rows, unique offer IDs, known account references, visible active offers and valid cutoff timestamps. System-account classification is explicit. | `TestAuditScopedOffersAndBalancesAndCancellation`, `TestAuditOrdinarySystemAccountAndNoPostingDistinction` |
| Full-only reversal validation rejected valid partial reversals; grants/adjustments and own historical balances were missing. | Validate cumulative partial reversals against originals; distinguish immutable event kinds; exclude grants/adjustments and system trades from ordinary trade turnover. Own trends include all posting kinds, opening/closing balances and no-posting state. | `TestAuditPartialReversalsAdjustmentsAndOwnOpening`, `TestAuditLaterPartialReversalPeriod` |
| Monetary JSON numbers could silently round above 2^53 in common clients. | All monetary output DTOs use exact decimal strings in JSON; server HTML/CSV retain exact integer minor units, scale and zone. | `TestAuditExactIntegerOutputAndNoJSOwn` |
| A source callback could cancel the request but still produce output. Repeated clock reads could cross a calendar boundary. | Check cancellation around the source read and aggregation; capture one clock instant per operation. | `TestAuditScopedOffersAndBalancesAndCancellation` |
| Own trend lacked no-JS HTML; stock/dormant data were absent from the HTML view. Duplicate query fields and invalid formats were loosely handled. | Add complete descriptive own/stock/dormant tables, exact own CSV and full aggregate JSON CLI; reject duplicate/unknown/malformed/oversized query shapes and unsupported formats. | `TestAuditExactIntegerOutputAndNoJSOwn`, `TestReadOnlyInterface` |
| CSV metadata needed exact unit context and safe textual cells. | Include decimal scale and time zone, encode through the standard CSV writer, neutralize formula-shaped textual zone labels, propagate buffered writer errors. | `TestAuditCSVMetadataFormulaAndWriterFailure` |

## Requirement-to-evidence matrix

| Requirement | Implemented evidence and boundary |
|---|---|
| Read-only ledger truth | Two postings per immutable event, tenant scope, account references, amount/pair/reversal validation. Invalid snapshots fail without partial output. The real core producer remains an external integration gate. |
| Gross/reversal/net and ordinary exchange count | Original ordinary trades counted once; partial/later reversals at their own posting time; independent fixture oracle. Separate reversal count is exposed in JSON. Grants/adjustments do not enter turnover. |
| Stock versus flow | Authoritative ledger-derived balances at the exact closed cutoff, not summed period trades. Ordinary/system account distinction. Source must supply the historical stock correctly. |
| Own member trend | Authenticated actor only; exact incoming/outgoing/opening/closing, all posting kinds and no-posting distinction. Unknown own account rejected. No arbitrary member/account query accepted. |
| Cohorts and high-risk slices | k >= 5, separate actual contributors for original/reversal flows and stock bins, per-member dominance, complementary stock suppression, private no-store outputs. This is conservative suppression, not differential privacy. |
| Fixed periods | Completed month or Monday-start week, one host-configured aggregate granularity/network, 1–12 closed periods, named zone in outputs, fixed cutoff. No caller-selected date ranges or fine-grained cohorts. |
| Dormant offers | Explicitly scoped active visible ordinary-member offers, configurable 1–365 calendar-day inactivity threshold (default 30), contributor/dominance suppression. No inference of demand. |
| No-JS and export parity | Descriptive server tables for admin and own views; full privacy-filtered aggregate JSON, flow CSV, own JSON/CSV. CLI `admin-json` includes stock/dormant metrics. No chart JS, CDN or beacon. |
| Precision and injection | Exact int64 JSON strings, checked integer sums, order-independent snapshot balance verification, escaped HTML, generated-date CSV labels and formula-safe textual metadata. |
| Auth/tenant/privacy | Host session callback plus fresh scoped Source authorization for every request/read. Wrong-network data denied. No raw personal export from admin analytics. |
| Concurrency/restart | Stateless repeatable release calculation and concurrent immutable-source readers pass race tests. No local cache/schema exists to migrate or restore; actual database transaction/restore tests belong to the eventual core adapter. |
| Bounded behavior | <= 100,000 events/accounts/offers, <= 200,000 postings, bounded identifiers/revision/query/count, O(periods × events) calculation; no graph, network call or background worker. |

## Actual validation

Go 1.26.11 linux/amd64:

- `GOMAXPROCS=1 go test -race -cover -timeout 90s ./...`: PASS, 85.9% statements.
- `GOMAXPROCS=1 go vet ./...`: PASS.
- `go mod verify`: PASS, no external module dependencies.
- `GOMAXPROCS=1 go test -run='^$' -fuzz=FuzzClosedPeriods -fuzztime=5s -parallel=1`: PASS, 52,138 executions.
- Runtime `govulncheck v1.1.4 ./...`: no vulnerabilities found.

## Explicit remaining gates

Actual Circula authentication/service/store adaptation, immutable historical snapshot generation, stable published releases across restart/corrections, production API/CLI wiring and full ledger regression remain unregistered and unqualified. The host must authorize membership/admin privilege inside the same read-only snapshot and must never filter originals using their current reversed/disputed state. No synthetic test proves those production contracts.

Repeated/adjacent-period aggregates can still disclose information with external knowledge; suppression and a fixed granularity are not a formal anonymization proof. The integration must adopt a per-network release/query-budget and correction policy, preserve one immutable release per cutoff, and obtain privacy review before rollout. Do not let callers vary the host configuration as a query parameter.

Manual browser/WCAG 2.2 AA/screen-reader assessment, operational snapshot/restore qualification and owner-selected single-feature integration are pending. Optional Chart.js is deliberately not adopted; no new dependency or upstream source is copied. No claim of full feature, production privacy or core completion is made.
