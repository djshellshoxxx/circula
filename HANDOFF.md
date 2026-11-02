# Circula core UX QA restart checkpoint

Repo: `djshellshoxxx/circula`; owned branch: `codex/circula-noscript-qa`.
Worktree: `/workspace/circula-noscript-qa`, original executor; one active worker.
Latest tested source: `d75f31e76c0e1f67dcb744ff8899487b57edda6a`.
Run 41451135622, attempt 1, completed: eight jobs passed; browser job
125296339997 failed with 264 passed / one failed (265 executed, 5.4 minutes).
Both unchanged 10-second fuzz commands passed. No job reruns requested here.

Passing groups: 112 viewport/error, 20 text-preference, 18 no-script focus and
five of six no-script journey/validation cases, plus unchanged baseline tests.
118 cases were added beyond the input's 147; 117 pass and one remains failing.
Plain key-only/no-script registration, explicit invitation join, balance and
empty history pass, as do fresh-trade/dispute/response, native validation recovery
and 404/Help recovery. No original core assertion, runtime or policy is weakened.
[Complete source, command, coverage, failure and artifact ledger](docs/qa/RESPONSIVE-CHECKS.md).

Sole browser failure: original return-link assertion in zz-noscript.spec.js:108.
After Create an account, next is still encoded after URLSearchParams parsing.
It remains ACTIVE and byte-identical, reproduced across d93a97b, 6a4aa61 and
this source. Later account/sign-in/join steps in that case are unexecuted.
No extra decoding, skip/xfail, deleted assertion or runtime fix. Direct entry
paths passing do not certify the broken return link. CI is not green/merge-ready.

Earlier d93a97b ran 223 passed / four failed: two root-path assumptions and one
innerText/textContent mismatch were demonstrated test mistakes. Their corrections
passed at 6a4aa61 (261/1) and this source. Original failures remain in Git/ledger.
Last fully green source: ca908a8b5743c7d674b4bac4b492c2a5c7c46698, 147 passes
with one unchanged fuzz retry. Its original deadline failure remains unresolved;
no reliability fix claimed. Earlier invalid-meter assertion/source also preserved.
Latest failure artifact: 13145300826, 116458 bytes; metadata/digest in ledger.
No artifact bytes fetched or dependency/artifact HTTP403 denial retried.

Final receipt changes documentation only and uses skip-ci. Resolve its head with
`git rev-parse HEAD`; the tested source above remains distinct. Local checks:
Node syntax/static registration, whitespace, hygiene/action pins and limited
complete-outgoing-file credential/private-data/link/fence/conflict review.
No local browser or full scanner result is claimed for this documentation head.
Normal source pushes/readbacks succeeded; no unpushed source intentionally left.
Frozen earlier branch: codex/circula-responsive-qa at 04e11989b811b4541a20b4e06e7a020beabb6af3.
PR17 is draft/open at 7643fb616208cc318939ded809e6cada66bb7415; PR18 draft/open
at 9c0bc500cb19f11f18694c76d134f4bfebf896e7. External integrated branch remains
827e651fe151f49659cc20aefe85d4af67bdaa97. PR12 audit is merged, not runtime handover.
No runtime/CSS, baseline helper/fixture, dependency/workflow, OPS004 runbook,
role/grant/24-hour invariant, threshold or security/retention change. Departure.go/
A1, pagination, C09/A2-07 and audit/WIP remain reserved; no archived PoC run.
Smallest next owner action: explicitly hand over the return-path finding at this
base, bounded files (at least internal/web/templates/join_anon.html and
internal/web/render.go; owner checks other next links) and the unchanged original
regression; then make a reviewed fix preserving safe local-path validation.
Generic continuation is not that handover. Broader permutations/admin/security/
lifecycle/loading/multi-network and native/manual/screen-reader/independent/
professional review remain unestablished. No full feature/release-completion claim.
PR creation PAUSED: automatic review rejected conditional creation without exact
approval; parent asked and no answer is known. No create/retry/workaround, merge,
APPROVE review, deploy/release, credential/security change or purchase attempted.
Pending approval for the frozen older branch does not approve this later source.
Preserve dirty/untracked/unpushed and external work; fetch explicit owned refs as
needed, no main reset/force push. No extra workers/paid calls/environment replacement.
Usage/model switching unavailable; keep reserve. Do not weaken gates or repeat fuzz retries.
