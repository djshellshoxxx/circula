# UX-047 existing currency-unit suffix continuation

Source checkpoint: 2026-11-23 Pacific. Active draft [#133](https://github.com/djshellshoxxx/circula/pull/133),
branch `codex/ux047-unit-suffix-20261123`, base `9cb4cc0250db487d6fd4760227ce906c4ea5f58f`.
Completed #130/#131/#132 and their integration evidence remain complete.

## Mechanism and bounded acceptance

`unitFor` still constructs the English suffix in Go. This is a confirmed
unfinished UX-047 catalogue seam, not a newly demonstrated wrong-English defect.
Only `return name + "s"` becomes `messages.Text("unit.plural", name)`; the new
entry is ordinary text, `other: "%[1]ss"`. Exact fixed-point one and negative-one
selection and case-insensitive already-s handling remain unchanged. No owner
policy decision, CLDR surrogate count, amount float, dependency, locale selection,
ledger/storage/mutation/authentication or privacy change is introduced.

| Existing test | Obligation |
| --- | --- |
| TestUnitForPlural | Precisions 0/1/2/4; positive/negative exact one, zero, fractional/large quantities, existing-s names, Unicode, percent/dollar and HTML-sensitive names |
| TestProposalMessageIsPlainText | New unit key must exist; name interpolation is literal and remains escaped by html/template |
| TestEveryCatalogueEntryFormats | The new entry receives its string argument without formatting diagnostics |
| Existing dashboard/marketplace/trade/dispute/admin render tests | All existing visible amount/unit and escaping behavior remains unchanged |

No new top-level test/requirement symbol is introduced; generated traceability
is not manually edited. Its exact-head freshness gate remains required. The
source/diff review must also confirm that the helper actually calls the new key.

## Actual proof and gates

Native regression-only commit `3d3e7e059e2a694709f99d7414322870db69814b`,
[run 45272197441](https://github.com/djshellshoxxx/circula/actions/runs/45272197441),
completed failing job 138034623179: focused message execution panicked exactly
with `unknown UI message: unit.plural` in the new existing-test subtest. The full
race recorder was skipped after that expected failure. Production files were
byte-identical to base in this red commit. Preserve its original log; no green
result is inferred from it. The red workflow's eventual overall state must be
reported separately from its completed failing job.

Independent browser investigation found no confirmed current browser-message
UX-047 defect; exact pinned production script/harness passed sixteen Node tests,
zero failures/skips (mock DOM only). Independent unit/source review approved
this narrow suffix externalization. These are assistant-worker reviews, not an
external security or manual accessibility assessment. No local Go compiler or
gofmt exists; native CI is execution/format authority.

Corrected current-head focused/full Go race, lint/vet/build, traceability,
browser, API/load/reproducibility/fuzz and applicable native checks remain gates.
Do not merge until all applicable final-head checks pass and source/diff is
independently reviewed; then verify actual main separately. Preserve the topic
branch and exact head/run/artifact results in the maintained #133 description.
This source checkpoint makes no subsequent passing-result claim.

UX-047 remains partial. Six prior explicit suite skips, bounded load/fuzz and
packaging/smoke limits, manual screen-reader/external security work, language,
policy/licence and remaining storage-fault qualification are not closed. Held
PRs #83/#85/#86/#94/#107 and all fifteen future modules remain untouched.

Fresh continuation: read live refs, #133's current source/checks and maintained
description, HANDOFF/TODO/specifications/decisions and this record. Finish this
same suffix slice while incomplete. If already fully qualified, do not repeat
it; release ownership for a fresh different UX-047 investigation. Never deploy,
release, select a licence, force-push, rewrite history or delete branches.
