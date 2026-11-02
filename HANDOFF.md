# Circula member-flow QA restart checkpoint

Repository: `djshellshoxxx/circula`; owned branch: `codex/circula-noscript-qa`.
Worktree: `/workspace/circula-noscript-qa`, original executor; one local worker.
Input `04e11989b811b4541a20b4e06e7a020beabb6af3` remains frozen on the responsive-QA
branch for pending exact PR approval. PR #18 remains at
`9c0bc500cb19f11f18694c76d134f4bfebf896e7`; application remains at
`7643fb616208cc318939ded809e6cada66bb7415`. No environment was replaced.

Last tested source: `ca908a8b5743c7d674b4bac4b492c2a5c7c46698`.
Run 41442176782 concluded success on attempt 2 after one unchanged failed-fuzz-job
rerun. Browser job 125266472066 reports 147 passed: 10 new cases beyond the
prior 34 (two no-script keyboard journeys and eight party-state viewport cases).
Attempt 1 passed eight jobs; FuzzParseAmount stopped at 10.10s with context deadline
exceeded and no reported input. Retry job 125272062062 passed both unchanged 10s
fuzz commands. Eight earlier successes were carried forward, not freshly rerun.
The original deadline failure remains a reliability observation, not a fixed defect.

Earlier source `de156e4e098991c9e393bce932f73c689c100d09` passed 9/9 with 139 browser
cases. Initial 0666f4636ed70b21d3586e9f3e696f60e3907b34 failed a new invalid meter
assertion; UX-011 makes that meter optional. It was corrected to UX-009 required
fields, with the failure/source/artifact metadata preserved and no runtime change.
[Reconciled inventory, commands, failures and exact evidence](docs/qa/RESPONSIVE-CHECKS.md).

The no-script cases complete sign-in/propose/accept/balance/history and propose/
decline/history via keys, on their own new synthetic trade IDs. The viewport
matrix now has 32 cases for 16 screen/role entries, including Bob's pending detail/
accept/decline and Carol's own-proposal cancellation, using GET only for the seed.
Runtime/CSS, baseline assertions, fixtures, dependencies, workflows, OPS-004 runbook,
credentials/security settings, roles, grant lineage, 24-hour rules, thresholds and
retention are unchanged. Departure.go/A1, pagination, C-09/A2-07 and audit/WIP
reservations remain intact. No archived PoC was restored or run.

Local source checks passed. Final receipt checks passed: whitespace, 28 local
links, fences/conflicts and limited review of three complete documents.
No local browser or full pinned-scanner run is claimed; existing reviewed CI
provided validation. Local dependency/artifact HTTP 403 boundaries remain binding
and were not retried; failure artifact bytes were not retrieved.

This later receipt changes documentation only and uses [skip ci]. Resolve its
head with `git rev-parse HEAD`; do not attribute tested-source CI to a later head.
Normal source pushes and remote readbacks succeeded. Before resuming, inspect
status/refs, preserve later authors' work and fetch explicit refs if default fetch
covers only main. Never reset to the prompt-only main to recreate application work.

PR CREATION PAUSED: automatic approval review rejected the parent's conditional
PR-creation step because clear exact approval was absent. The parent requested
approval; no answer is known. This continuation made no PR-create attempt or
workaround, merge or APPROVE review. Do not reinterpret pending approval for the
frozen earlier branch as authority for a different source or action.

Remaining QA: no-script registration/invitation/dispute flows, wider screen/state
coverage, native/manual focus/reading checks and independent/professional review.
Runtime changes need a bounded owner handover naming finding/base/files/regression.
Hosted staging needs its own authorised operator/target. Owner/licence/default/
threshold and release gates remain open. No extra/review workers or paid API calls;
external private writer state is unverified. Usage/model switching is unavailable;
keep a reserve. No deployment, purchase, credential/security change, force push,
destructive reset, source loss, denied download or approval bypass is authorised.
