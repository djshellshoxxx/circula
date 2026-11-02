# Circula responsive QA continuation

Repository: `djshellshoxxx/circula`.
Owned branch: `codex/circula-responsive-qa`.
Input checkpoint: `9c0bc500cb19f11f18694c76d134f4bfebf896e7` (draft PR #18),
preserving application `7643fb616208cc318939ded809e6cada66bb7415` (draft PR #17).
Use `git rev-parse HEAD` and remote readback for the current test checkpoint.
Prior staging handoff remains at `git show 9c0bc500cb19f11f18694c76d134f4bfebf896e7:HANDOFF.md`.

Ready slice: [24 responsive browser cases](e2e/tests/responsive.spec.js) for 12
representative pages at 320×640 portrait and 640×320 landscape. They check the
intended route, heading/main visibility, page-level overflow and table-scroll
region labelling/focusability. These cases navigate only; fixtures and every
existing assertion are unchanged. UX-033/UX-036 remain partial.
[Coverage, commands and actual evidence](docs/qa/RESPONSIVE-CHECKS.md).

Local checks passed: Node syntax, whitespace, repository hygiene/action pins,
27 local links and complete outgoing limited credential/private-data patterns
across four files. Browser execution is pending exact-source CI, not passed.
No local application/browser or pinned full-scanner pass is claimed. Node modules
and Chromium are absent here; earlier module/artifact HTTP 403 denials remain
binding and were not retried. Existing reviewed CI is the validation route.

Freeze the published test candidate while CI runs. Record exact source/run/job,
actual results and original failures before claiming this QA slice validated.
Do not weaken assertions, shrink coverage or edit reserved runtime/CSS if a new
case fails. Keep routine CI receipts separate until candidate checks finish.
No workflow, runtime, dependency, ledger, monetary value, threshold, credential,
security setting, policy or 24-hour/grant-lineage fixture changed.

Ownership: this coordinator reserves only the new responsive test, its QA notes,
two as-built notes and this branch's handoff. Existing PR #17 integration and
PR #12 audit/WIP, A1/A3 and C-09/A2-07 reservations remain intact. No runtime
handover was found; no external private-session activity state is inferred.
One local worker; no coding/review subagents or paid API calls. Model switching
and reliable usage percentage are unavailable; retain a checkpointing reserve.

Next: normally publish this owned branch and open a draft into the preserved
PR #18 branch after checking live refs and outgoing content. Verify remote SHA,
draft/base/head and exact-source CI. Preserve the runbook and all older refs.
Before runtime remediation, obtain a bounded audit-owner assignment naming the
finding, base, permitted files and original regression. Hosted staging still
needs a separately authorised operator/target. Manual screen-reader, independent
professional review, owner decisions and release gates remain open. No APPROVE,
merge, PR closure, force push, release/deployment, purchase, credential/security
change or denial bypass is authorised by this QA task.
