# Circula member-flow QA restart checkpoint

Repo: `djshellshoxxx/circula`; branch: `codex/circula-noscript-qa`.
Worktree: `/workspace/circula-noscript-qa`, original executor; one worker only.
Input receipt: `5337c443d3b4f5d839e4d9b6f5f0854011cef53b`.
Earlier responsive branch stays frozen at `04e11989b811b4541a20b4e06e7a020beabb6af3`.
PR18: `9c0bc500cb19f11f18694c76d134f4bfebf896e7`; application:
`7643fb616208cc318939ded809e6cada66bb7415`. External integrated branch:
`827e651fe151f49659cc20aefe85d4af67bdaa97`. No published runtime handover found.

Latest executed source: `d93a97b314e4ced620a05a0eeae5c59cb4ad920e`, run41447665095:
eight jobs passed on attempt1 (both unchanged10s fuzz commands included).
Browser job125284693199: 223passed/4failed. Two root-route failures were a test
assumption (single membership intentionally redirects). One balance failure mixed
innerText/textContent. These new tests are corrected with strict destination and
same-representation equality. Original failures remain in Git and the ledger.
The invitation-return failure is a runtime candidate: after Create an account,
`next` remains encoded after URLSearchParams decoding. The exact original case
stays ACTIVE, byte-identical; no extra decoding, skip/xfail or runtime fix is added.
Subsequent registration/login/join in that case were unexecuted.
[Coverage, exact failures, artifacts, ownership and commands](docs/qa/RESPONSIVE-CHECKS.md).
Last fully green source remains `ca908a8b5743c7d674b4bac4b492c2a5c7c46698`,147passes
with one unchanged failed-fuzz-job retry. Original deadline failure is unresolved;
no rerun to obtain green here. The earlier invalid-meter assertion is also retained.

Current follow-up adds 16 text-preference variants using fresh synthetic accounts,
invitations/trades and safe seeded preview/edit; plain registration plus explicit
invitation join does NOT validate the broken return link. Adds one no-script
validation/correction preview and18 GET-only keyboard/no-script focus probes.
Static counts:110 viewport,20 preference,5 no-script,18 focus; expected262 total,
not an executed/pass count. Current new-source runtime/aggregate CI is PENDING.
The known failure prevents green/merge-ready claims even if independent tests pass.

Local Node syntax, unique static registrations, whitespace, hygiene/action pins,
unchanged baseline assertions/fixture/helper/workflow scope checks pass. Inspect
complete outgoing files/private-data/link scan before normal checkpoint publication.
Existing push CI only; no tag/dispatch. No local browser/full scanner claim.
Local dependencies absent; dependency/artifact HTTP403 boundaries remain binding;
no retry/alternate route. Failure artifact metadata is in ledger, no bytes fetched.

No runtime/CSS, baseline test, helper/fixture, dependency/workflow, OPS004 runbook,
role/grant/24h invariant, threshold, security/retention change. Departure.go/A1,
pagination,C09/A2-07,audit/WIP reserved; no archived PoC restored/run.
Smallest owner action: hand over invitation return-path finding/base/files and
original regression (at least join_anon.html/render.go; owner verifies other next
links) before a reviewed fix preserving safe local-path validation. No transfer
is inferred from generic continuation. Park only dependent work meanwhile.

PR CREATION PAUSED: automatic review rejected conditional creation without exact
approval; parent asked, no answer known. No create/retry/workaround, merge,
APPROVE review, deploy, release, credential/security change or purchase is attempted.
Pending approval for frozen04e1198 is not approval for this later source.

Next: inspect exact-source CI, preserve all failures; correct only demonstrated
test mistakes within ownership. No weakening budgets/assertions or repeated fuzz
retry. Manual/native/screen-reader and independent/professional review stay open.
Resolve HEAD with `git rev-parse HEAD`; later docs receipts are not tested source.
Preserve all dirty/untracked/unpushed work and external owners. Explicitly fetch
owned refs if default fetch is main-only; no reset/force push. No extra workers,
paid calls or environment replacement. Usage/model switching unavailable; reserve.
