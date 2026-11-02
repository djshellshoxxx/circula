# Circula no-script QA restart handoff

Repository: `djshellshoxxx/circula`; sole local worker on
`codex/circula-noscript-qa`, worktree `/workspace/circula-noscript-qa`.
Input: `04e11989b811b4541a20b4e06e7a020beabb6af3`. Keep the responsive-QA branch
frozen there for its pending exact PR approval. Draft PR #18 remains at
`9c0bc500cb19f11f18694c76d134f4bfebf896e7`; application is unchanged at
`7643fb616208cc318939ded809e6cada66bb7415`. No environment was replaced.

Earlier tested source `32a8314ebd28094dd6d2500c454ed809f6b16b3c` passed 9/9 in
run 41432973449, with 137 browser tests including 34 new UX cases. Those results
are historical evidence, not validation of this new slice. Current test source
must be resolved with `git rev-parse HEAD` after publication and remote readback.

Selected QA-034 slice: [two no-script keyboard journeys](e2e/tests/zz-noscript.spec.js),
using existing kbd.js helpers and ordinary-member fixtures in explicit
javaScriptEnabled:false contexts. The tests propose two new synthetic trades,
accept one and decline the other, then check balance/history and no posting for
the declined proposal. Seeded trade IDs and all baseline assertions are intact.
[Reconciled screen/role/flow inventory and coverage](docs/qa/RESPONSIVE-CHECKS.md).
First source `0666f4636ed70b21d3586e9f3e696f60e3907b34` failed run 41439165546:
8 jobs passed; E2E had 138 passed/1 failed at an invalid new Bob meter assertion.
UX-011 makes that debt meter optional. The corrected test checks UX-009's required
fields; no runtime fix, script enablement or wait extension. Original failure and
artifact metadata 13139080390 are preserved in the inventory record; bytes unfetched.
The corrected source still needs its own CI. Do not report the failed source green.
QA-034 remains partial: registration, invitation acceptance, dispute raise,
all states and manual focus/reading checks are outside these two cases.

Initial slice: Node syntax, whitespace, hygiene/action pins, 36 local links, fences/conflicts
and limited review of four complete outgoing files passed. Record actual new-source
CI before claiming validation. Local browser dependencies remain unavailable;
prior dependency/artifact HTTP 403 denials are binding and were not retried.
Correction checks passed: syntax/whitespace/hygiene, 5 links and limited scan of 3 files.
Only the existing reviewed branch-push CI route may supply browser validation.
Freeze its source while CI runs, preserve failures/assertions and diagnose actual
causes; do not enable site scripting or weaken a test to obtain green checks.

PR CREATION PAUSED: automatic approval review rejected the parent's conditional
PR-creation step because clear exact approval was absent. The parent requested
that approval; no answer is known. This task attempts no PR creation, workaround,
merge or APPROVE review. Ordinary owned source checkpoints remain permitted.
The earlier unconfirmed creation result is historical; it does not lift this gate.

Only this new test, existing QA inventory, QA-034 as-built note and this handoff
are reserved. Runtime/CSS, departure.go/A1, pagination, archived PoCs, fixtures,
workflows, dependencies, credentials/security settings, roles, grant lineage,
24-hour rules, monetary thresholds and retention remain unchanged. Security tests
retain their separate secfixture instance. No reserved PoC is restored or run.
One local worker, no extra/review workers or paid API calls; external private
writer state is unverified. Usage/model switching is unavailable; keep a reserve.

Next: finish scoped checks, normally checkpoint this owned child branch and
verify its SHA; observe exact-source CI without a PR or manual dispatch. If it
fails, preserve the exact run/job/assertion and fix only an allowed test defect.
A runtime/CSS defect needs an explicit bounded owner handover before edits.
Manual/native, independent/professional, owner/licence/default/threshold and
release gates remain open. Hosted staging still needs its own authorised target.
No deployment, purchase, credential/security change, force push, destructive
reset, denial bypass or reinterpretation of pending PR approval is authorised.
