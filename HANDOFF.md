# Circula accessibility QA restart checkpoint

Repository: `djshellshoxxx/circula`; owned branch: `codex/circula-responsive-qa`.
Worktree: `/workspace/circula-responsive-qa` in the original executor.
Application remains `7643fb616208cc318939ded809e6cada66bb7415` (draft PR #17).
Input `9c0bc500cb19f11f18694c76d134f4bfebf896e7` and draft PR #18 are preserved.

Frozen test source: `32a8314ebd28094dd6d2500c454ed809f6b16b3c`.
Push run 41432973449: completed successfully, all 9 jobs passed at that source.
Browser job 125235861360: 137 passed, including 34 new cases (24 viewport,
4 text-preference and 6 Help-placement). Older sources 5d2b2c2 and 2e323a1 each
passed all 9 CI jobs; their exact evidence remains in the coverage records.
Resolve a later document-only HEAD with `git rev-parse HEAD`; do not attribute
source CI to a later handoff-only commit. No local browser or full scanner pass.

Coverage/commands/limits: [viewport](docs/qa/RESPONSIVE-CHECKS.md),
[text preferences](docs/qa/TEXT-PREFERENCES.md), [Help](docs/qa/CONSISTENT-HELP.md).
UX-033/034/036/049 remain partial: representative screens/flows do not establish
all-state/all-page, native-device, manual reading or complete WCAG acceptance.
Runtime/CSS, old assertions, fixtures, dependencies, workflows, OPS-004 runbook,
monetary values, grant lineage, 24-hour rules and security/retention are unchanged.
Preview tests stop before Send trade; no transaction is proposed.

One local worker; no subagent, external writer invocation or paid API call.
PR #17 integration and PR #12 audit/WIP, A1/A3 and C-09/A2-07 reservations remain.
No runtime handover was found; private external-session activity is unverified.
Final receipt checks passed: whitespace, 29 local links, fences/conflicts and
limited review of four complete documents. Prior dependency/artifact HTTP 403 denials remain
binding and were not retried. Local Node modules/Chromium are absent; existing
reviewed CI supplied browser evidence. Usage/model switching is unavailable;
retain a checkpointing reserve rather than guessing a usage percentage.

Draft creation returned no usable structured receipt, then response storage failed.
Repeated readback found no PR for this branch. No create retry/alternative route
was used and no permission-denial reason was captured. Review the original action
result/current permissions before another PR-create attempt. Source push/readback
succeeded; expected draft target remains PR #18's branch. Never claim draft creation
without readback. Keep this tested checkpoint unmerged pending appropriate review.

This later receipt uses `[skip ci]` only for documentation; source/tests are
unchanged from the validated commit. No passing CI is claimed for the later head.
Next: review this checkpoint and resolve the PR-create result before another
attempt. Normally publish only this owned handoff and verify SHA. Runtime needs an owner
assignment for `internal/service/departure.go`, new `internal/service/departure_test.go`
and temporary `internal/service/a1_poc_test.go`, restored only with permission from
`docs/audit/2026-10-28/poc/A1/internal/service/a1_poc_test.go.txt`.
After handover: `go test -tags=a1poc -run '^TestA1PoC_01' -count=1 ./internal/service`.
No A1 regression ran here. Keep A1-06 pagination separate unless handed over;
do not apply the saved patch wholesale. Cached/approved test access is required.
Broader QA follow-up needs an accurate screen/role inventory and additional flows;
manual/native and independent review remain open. Hosted staging needs a separately
authorised operator/target. Owner/licence/default/threshold and release gates remain.
No merge, APPROVE, closure, force push, deployment, purchase, credential/security
change or denial bypass is authorised by this QA task.
