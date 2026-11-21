# Dispute decision and re-review confirmation catalogue

Source base: `4c988eeebddc6494257d5ef8b071402048665d22`. This bounded UX-047
continuation adds 40 English messages for the existing decision/re-review
handlers and their two confirmation pages. The base's 982 catalogue values
remain unchanged; this isolated candidate has 1022 entries.

The scope is `h_dispute_decide.go`, `h_dispute_review.go`,
`h_rereview_decide.go`, `dispute_decide_review.html`, and
`dispute_rereview_confirm.html`. The catalogue holds complete outcome,
validation, reversal, pending-state and confirmation explanations, plus the
labels and actions on those screens. Dynamic member names, rationale text,
amount strings and units remain ordinary template-escaped values. The unknown
completed re-review outcome keeps the existing generic explanation.

The two refusal explanations preserve their different existing behavior: an
initial decision can be recorded while its reversal waits, whereas a re-review
that cannot post its reversal records nothing. The amount calculations,
outcome values, authorization conditions, response-waiver fields, CSRF and
idempotency controls, status checks and service calls are unchanged. The
`reReviewProof` function, including its signed values and scope, is unchanged.
Full balance rows retain the existing `ShowBalances` condition and the
handler's administrator-only account reads.

Completed re-review prose previously appended `s` to every supplied unit. Its
caller now uses the existing `unitFor` helper before formatting the whole
sentence. This follows the same unit selection already used on the review
screen and retains exact `domain.FormatAmount` strings. The new web-package
regression covers one `credit`, plural `credits`, custom `Hours`, the exact
amount `90071992547409.93`, confirmed outcomes and the unknown-outcome fallback.
That regression is included for combined-suite execution; it was not run in
this lightweight verification.

## Observed verification

- **Fail-before:** the new message/template regression ran on the exact base
  before implementation changes. The command exited **1**; 13 new message
  cases failed because their catalogue keys were absent. The confirmation
  template controls already passed. This receipt establishes missing catalogue
  coverage, rather than an executed baseline failure of the unit helper.
- **Pass-after:** `GOMAXPROCS=1 GOMEMLIMIT=400MiB go test -p=1 -race
  ./internal/web/messages -count=1` exited **0**, package time **1.159s**.
  It covers every catalogue entry and renders both actual confirmation
  templates with the actual base partials. The new cases cover reversal and
  no-reversal branches, conditional balance rows, hostile member/rationale
  text, large exact amounts, signed changes, route/ID values, review proof,
  response-waiver values and confirmation/back controls.
- A direct comparison confirms unchanged form/input/button opening tags and
  CSRF/idempotency partial calls in both templates, and an unchanged
  `reReviewProof` body. `gofmt` and `git diff --check` are clean. Parsed catalogue
  comparison confirms exactly 40 added keys and zero changed base values.

[Before output](dispute-decision-before.log),
[after output](dispute-decision-after.log), and the
[source/command receipt](dispute-decision-receipt.json) retain the evidence.

## Remaining qualification

The messages/template pass does not execute handler authorization, service
transactions or SQLite. Run `TestReReviewCatalogueKeepsAmountAndUnit` and the
existing web/handler tests as part of the combined candidate's qualification,
then refresh shared traceability. This batch retains the current governance
holds and future-module boundaries and leaves UX-047's wider completion claim
unchanged.
