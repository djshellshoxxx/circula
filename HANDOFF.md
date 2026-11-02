# Circula automated accessibility QA continuation

Repository: `djshellshoxxx/circula`; owned branch: `codex/circula-responsive-qa`.
Input: `9c0bc500cb19f11f18694c76d134f4bfebf896e7` (draft PR #18), retaining
application `7643fb616208cc318939ded809e6cada66bb7415` (draft PR #17).
Resolve current test HEAD with `git rev-parse HEAD` and remote readback.

Validated checkpoints:
- `5d2b2c219acbd0d6ba192c42372cc7a8a484ab7b`: push CI 41429404139 passed 9/9;
  browser job 125223913309 reports 127 passed, including 24 new viewport cases.
  [Coverage/evidence](docs/qa/RESPONSIVE-CHECKS.md).
- `2e323a18e4f3af5e080d58e941f8cc2f74f0842f`: push CI 41431266623 passed 9/9;
  browser job 125230148678 reports 131 passed, including four new text-preference
  form-flow cases. [Coverage/evidence](docs/qa/TEXT-PREFERENCES.md).

Next slice: [six Help-placement cases](e2e/tests/help-location.spec.js), comparing
three sections at two viewports across 12 representative pages. Browser results
are pending this new source's own CI; prior green results do not validate it.
[Scope, commands and limits](docs/qa/CONSISTENT-HELP.md). UX-033/034/036/049 stay
partial; no complete WCAG, all-page, native-device or manual-review claim.

One local worker; no subagent, external writer invocation or paid API call.
PR #17 integration and PR #12 audit/WIP, A1/A3 and C-09/A2-07 reservations remain.
No runtime handover was found; private external-session activity is unverified.
This reservation covers only new QA files, their notes, as-built notes and this
handoff. Runtime/CSS, old assertions, fixtures, dependencies, workflows, OPS-004
runbook, monetary values, grant lineage, 24-hour rules and security/retention are
unchanged. Preview tests stop before Send trade; no transaction is proposed.

Local checks passed: Node syntax, whitespace, hygiene/action-pin preservation,
29 local links, fences/conflicts and limited review of five outgoing files.
Record this new source's CI separately. No local browser/pinned-scanner
pass: Node modules/Chromium are absent, and earlier dependency/artifact HTTP 403
denials were not retried. Existing reviewed branch-push CI is the validation route.
Freeze the published candidate until CI finishes; preserve original failures and
assertions. Do not shrink coverage or alter reserved runtime/CSS to obtain green.

Draft creation returned no usable structured receipt and response storage failed;
readback showed no PR for this branch. No create retry/alternative route was used,
and no permission-denial reason was captured. Inspect the original action result
and current permissions before another attempt. Source push/readback succeeded;
expected draft target remains PR #18's branch. Never claim draft creation without
readback. Source checkpoints and this handoff are accessible on the owned branch.

Next: normally publish this candidate, verify SHA and observe existing CI; keep
all input histories/branches intact. Runtime A1-01/A1-06 needs an audit-owner
assignment for `internal/service/departure.go`, new `internal/service/departure_test.go`
and temporary `internal/service/a1_poc_test.go`, restored only with permission from
`docs/audit/2026-10-28/poc/A1/internal/service/a1_poc_test.go.txt`.
Original command after handover: `go test -tags=a1poc -run '^TestA1PoC_01' -count=1 ./internal/service`.
No A1 regression ran here. Keep A1-06 pagination separate unless handed over;
do not apply the saved patch wholesale. Cached/approved test access is required.
Hosted staging needs a separately authorised operator/target. Manual screen-reader,
independent/professional review, owner and release gates remain open. No merge,
APPROVE, closure, force push, deployment, purchase, credential/security change or
denial bypass is authorised here. Usage/model switching is unavailable; retain
a checkpointing reserve.
