# Shared errors, help, member notices and historical balance

Source base: `40277310094ac16b34969fc3b002b2182983d8b5`, the frozen 1087-entry
presentation source with the three known literal-error vet corrections. This
independent UX-047 continuation adds **64 entries**, producing **1151** in its
isolated candidate. All 1087 base values remain unchanged.

| Message group | Added entries | Scope |
|---|---:|---|
| `error.*` | 22 | Page-level titles/explanations, plain HTTP fallbacks, reference and home action |
| `help.*` | 24 | Credits, balances, statuses and existing support boundaries |
| `histbal.*` | 12 | Historical lookup form/result and web display of known input reasons |
| `home.*` | 1 | Recent-activity load failure notice |
| `notice.*` | 5 | Terms redirect flash, pending membership, credit-limit and frozen notices |

The dashboard title and error-page sign-in action reuse existing catalogue
labels. Session-help messages remain unchanged.

## Source and behavior boundaries

`render.go` changes only copy within `render`, `renderError` and `fail`.
`mustLoadTemplates`, its function map and the existing message helpers are
byte-identical to the base. HTTP statuses, retry headers, generic-error privacy,
request-reference handling, buffered-render fallbacks and authentication/terms
redirects retain their existing behavior. The recovery middleware and
`http.ErrAbortHandler` handling are unchanged.

The handler scope is the help title in `h_auth.go`; dashboard title/activity
notice and the submitted-empty historical reason in `h_home.go`;
`h_notify.go`'s credit-limit notice; and the pending title in `session.go`.
Template changes cover the non-session help content, pending membership, the
base frozen notice and the three error-page labels. Complete sentences retain
their `<strong>`, `<code>` and `<time>` elements in templates. Untrusted names,
references and reason text remain ordinary escaped values.

`histbal_view.go` maps the four known shared reason constants to web messages
and returns unknown reasons unchanged as plain text. Both parser and reader
refusals use that mapping; the History handler's submitted blank uses it too.
`internal/histbal/histbal.go`, service behavior and API text are untouched.
Raw submitted text, explicit offset requirements, millisecond precision,
future-bound refusal, normalized UTC output, exact signed amounts and account
identifiers remain unchanged. Historical results retain their separation from
current available/reserved/credit-limit values.

The historical fragment's standalone template loader registers the existing
`msg` and `msgAround` functions. The main loader already provides them. The
small support-boundaries test loader also gains `msgAround` to parse the
preserved bold help labels. This is the same fixture extension coordinated
with the parallel administrator-home batch.

## Observed verification

**Before:** on the exact base with only the new message/template regression
file added, the focused messages command exited **1** in **0.027s**. Fourteen
new message cases failed because their keys were absent. Actual help, pending,
error, frozen-notice and historical-template controls already passed. This is
evidence of missing catalogue coverage, without a claim of a baseline HTTP
runtime defect.

**After, complete messages package:**

```text
GOMAXPROCS=1 GOMEMLIMIT=250MiB GOGC=40 go test -p=1 -race ./internal/web/messages -count=1
```

Exit **0**, package time **1.306s**. This includes every catalogue formatting
case and the actual help/pending/error/base templates. The historical fragment
is read from its actual Go string constant and rendered both alone and inside
the actual History template. Tests preserve form/action/field associations,
conditional error/result visibility, escaped unknown reasons and hostile text,
exact large signed amounts, raw offset input, UTC timestamp elements, support
disclosures and the existing session-help links.

**After, database-free HTTP and production loaders:** Go's named-file test
mode used the exact **33 production web files** plus four selected test files,
with default vet, normal instrumentation, `-p=1`, `-count=1` and `-timeout=60s`.
It exited **0** with **35 passing test/subtest events across 13 top-level
tests**, no failures, **2.424s wall time**, under `GOMEMLIMIT=250MiB`, `GOGC=40`
and `GOMAXPROCS=1`. The receipt gives the complete command/file list and every
source SHA-256. No production functions were extracted or replaced by stubs;
the dependency list excludes `modernc.org`, `internal/store` and
`internal/testutil`.

Those tests execute the real public renderer, refusal mapper and recovery
wrapper; check statuses, Retry-After, private-cause suppression, request
correlation, authentication/password/terms redirects and signed flash text;
exercise missing/broken-template plaintext fallbacks; and require the abort
sentinel to escape without an appended error page. Historical tests execute
the real parser, view builder, standalone loader and main History loader,
including unknown reader reasons and exact values beyond JavaScript's safe
integer range. Existing historical-form and support-boundary tests also run.

An earlier execution of the same named-file selection passed all 35 events in
9.117s under the previously supplied 400MiB setting. It completed before the
tighter setting arrived; both actual attempts are retained in the receipt.
No additional compilation followed the final recorded checks.

Direct comparison confirms unchanged production template-function
registration, middleware, shared historical contract, existing message helpers,
`h_disputes.go`, the session-help block and all action/input/button/link/time
opening tags in the touched templates. `gofmt` and `git diff --check` are clean.

[Before output](shared-pages-before.log),
[messages output](shared-pages-after.log),
[HTTP/loader JSON test output](shared-pages-http-loader.log), and the
[exact source/command receipt](shared-pages-receipt.json) retain the evidence.

## Integration boundary

The named-file selection excludes the rest of the web test suite and all
database-backed tests. Complete combined web/service/SQLite/browser and race
qualification remains a parent-candidate gate. This batch makes no locale or
policy decision and leaves the parallel function-map helpers and remaining
dispute validation details with their assigned owners. Refresh shared
traceability after integrating the concurrent catalogue slices.
