# UX-047 rules-log quantity continuation

Checkpoint: 2026-11-23 Pacific. Status: candidate; runtime qualification pending.

- Qualified starting main: `fff81fc7f64cb4858c09d4a3a6f1ff0a55a1a337` (PR #130 already complete).
- Branch / PR: `codex/ux047-rule-quantities-20261123`, [#131](https://github.com/djshellshoxxx/circula/pull/131).
- Regression-only source: `15664ad3341428b24d1ccf8c9078b9f2f12fb9cc`.
- Corrected product source: `037f10a7d8729ab7e3add4c06e724e30b8e8cd77`; tree `2dba0fba87d01f5244da84592d8845b4ae6177dc`.
- The documentation continuation follows that product source. Its exact PR head must pass CI before merge; these source identifiers are not passing-result claims.

## Observed mechanism and bounded correction

The current rules log passes numeric policy values through `fmtAny`, whose
`strconv` conversion bypassed the English catalogue. Allowed inactivity
values 1000 and 3650 therefore appeared as raw digits in complete change
sentences. `rules.quantity` now formats integer quantities through the same
catalogue as other counts. The three visible monetary policy keys still use
exact fixed-point domain formatting before this fallback; version identifiers
follow a separate path. No float conversion, new dependency, locale selection,
policy setting or mutation is introduced. The catalogue grows from 1,458 to
1,459 entries.

The five product files are `internal/web/h_gov.go`,
`internal/web/messages/en.json`, `internal/web/messages/messages_test.go`,
`internal/web/messages/terms_rules_leave_test.go` and
`internal/web/terms_rules_leave_catalog_test.go`.

## Acceptance and traceability

This is another bounded part of UX-047, not whole-UI localization acceptance.

| Existing cited test | Added evidence obligation |
| --- | --- |
| `TestRulesCatalogueRowsAndStates` | int/int64 quantities at 0, 1, 1000 and 3650; real rule rows/template; previous/new and initial values; ungrouped version 1000; numeric-looking text unchanged; all three monetary-key bypasses unchanged |
| `TestTermsRulesLeaveMessages` | complete English integer formatting including exact int64 9007199254740993, without floating point |
| `TestEveryCatalogueEntryFormats` | new quantity entry receives an integer argument and produces no formatting diagnostic |

The existing UX-047 symbols and requirement citations are retained. The generated
`docs/TRACEABILITY.md` inventory requires no new symbol citation; this note adds
exact-source qualification context without manually editing generated output.
The CI traceability freshness check remains required. Existing amount/Unicode/
HTML-escaping, Boolean and version tests remain in the same test selection.

The initial regression-only CI attempt at `6bd3dc3a41b9eb7c3eabef180928a2da38b1ad6f`
was superseded to normalize an unintended terminal blank line. The current
regression-only run is [45230855699](https://github.com/djshellshoxxx/circula/actions/runs/45230855699).
The green fixture retains the failure assertions and additionally covers all
monetary-key bypasses and numeric-looking text; redundant int casts were removed
following independent review. The original assertions' behavior is unchanged.

## Verification so far and remaining gates

- Fresh live main/ref, open Core PR metadata and actual-main CI were read before editing. Main still matches the completed #130 source.
- Independent number/type investigation, diff review and acceptance/security review found no remaining blocker in the bounded correction. They are source reviews, not independent external security qualification.
- Local `node --test scripts/test_app_messages.cjs`: 16 passed, zero failures/skips, on unchanged browser code. JSON validation and diff whitespace checks passed.
- No local Go compiler is installed. No local Go test, vet, race, build, browser or screen-reader result is claimed. Exact-head CI results and regression failure logs are still pending at this documentation checkpoint.
- Merge remains blocked until applicable checks pass at the final current PR head, its diff is verified and the integration record is published. Preserve all topic/evidence branches; never force-push.

Policy-log reads continue removing the private dual-control threshold. This
presentation-only patch does not change ledger amounts, SQLite operations,
authorization, tenant selection, audit, CSRF, balances or stored policy values.

SEC-026 visibility/threshold/retention and GOV-021 authority decisions remain
held. Held policy PRs stay untouched. Translation scope, manual accessibility,
independent security review, licensing and release/operations qualification are
not closed by this slice. All fifteen future modules remain out of scope.

## Fresh continuation prompt

Read live main, #131 and its current head/checks plus this note, HANDOFF.md and
docs/TODO.md. Continue only the active UX-047 rules-log quantity slice until its
regression evidence, exact-head CI and integration handoff are complete. Do not
repeat #130 setup precision, inspect future modules, touch held policy PRs or
claim whole UX-047 complete. After this slice is qualified, release ownership
and let a fresh run select a different bounded current-source UX-047 investigation.
