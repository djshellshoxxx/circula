# Dispute presentation catalogue and unit correction

Source base: `0fb5304a2caeb375d1791bf583744ce2a81cc3ba`. This bounded UX-047
continuation adds 117 English messages for the dispute list, reviewer queue,
opening/review form, detail page and their existing actions. All 859 catalogue
values in the base remain unchanged; this isolated candidate has 976 entries.

`h_disputes.go` now uses complete messages for category/status/outcome labels,
deadline refusals and action notices. Known timeline event values have explicit
catalogue labels; unknown category/status/event values retain their exact text
and remain ordinary escaped data. Unknown values are not used as catalogue
keys or interpreted as format strings. An empty event value no longer depends
on indexing its first byte.

The detail template previously appended `s` to every network unit. Actual
template regressions on the baseline fail for one `credit` and custom `Hours`.
The handler now supplies the existing `unitFor` result, and the template formats
the whole trade sentence with exact amount strings. There is no financial
arithmetic or stored-data change. The opener's name and opening timestamp remain
separate escaped/template-owned values, with a complete `Opened …` sentence;
untrusted names never enter the sentinel-based element formatter.

The actual base/dispute templates preserve timestamp elements, routes, literal
IDs, CSRF/idempotency values, submitted fields, category/outcome values,
field-error associations, required/maxlength attributes, review/back controls,
queue filters and continuation links. Conditional response, re-review request,
and decision controls keep their existing conditions. Service calls and
authorization/governance rules are unchanged.

## Observed verification

- **Fail-before:** a standalone temporary module used the exact baseline
  messages implementation/catalogue and five actual template files, plus the
  new regression file. The command returned **1**; the singular/custom-unit
  detail cases failed. Opening-form and list/queue controls passed. Source Git
  blobs and the regression SHA-256 are recorded in the receipt.
- **Pass-after:** `GOMAXPROCS=1 GOMEMLIMIT=400MiB go test -p=1 -race
  ./internal/web/messages -count=1` returned **0**, package time **1.224s**.
  This includes all existing catalogue tests, exact/hostile message values,
  actual template escaping and form controls, and all new conditional detail
  branches. It does not compile SQLite or exercise handler authorization.
- `gofmt` and `git diff --check` are clean. Parsed catalogue comparison confirms
  exactly 117 added keys and zero changed baseline values.

[Before output](dispute-catalogue-before.log),
[after output](dispute-catalogue-after.log), and the
[source/command receipt](dispute-catalogue-receipt.json) retain the evidence.

## Remaining boundary

The new `internal/web/dispute_catalog_test.go` enum/outcome regressions are
included for the combined web suite, but were not executed in this lightweight
pass. Full web/handler and combined-candidate qualification remain required.

Dedicated decision/re-review handlers and confirmation templates remain a
separate catalogue slice: `h_dispute_decide.go`, `h_dispute_review.go`,
`h_rereview_decide.go`, `dispute_decide_review.html`, and
`dispute_rereview_confirm.html`. Existing inline domain-error detail text and
the wider locale/date scope also remain. This change does not declare all
UX-047 complete, select translation languages, alter policy holds, or enable
any future module. Regenerate shared traceability after concurrent integration.
