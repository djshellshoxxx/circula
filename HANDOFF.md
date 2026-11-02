# Circula core UX QA restart checkpoint

Repo `djshellshoxxx/circula`; owned branch `codex/circula-noscript-qa`.
Worktree `/workspace/circula-noscript-qa`, original executor; one active worker.
Frozen earlier responsive branch: `04e11989b811b4541a20b4e06e7a020beabb6af3`.
PR18: `9c0bc500cb19f11f18694c76d134f4bfebf896e7`; application:
`7643fb616208cc318939ded809e6cada66bb7415`. External integrated branch readback:
`827e651fe151f49659cc20aefe85d4af67bdaa97`. No published runtime handover found.

Last executed source `6a4aa6190dcb001fad4ba6aa5bb5e9fd366ddc02`:
run41449741715 attempt1 completed; eight jobs passed, browser failed261passed/1failed.
Passing groups:110 viewport,20 text-preference,18 no-script focus,4/5 no-script
journeys/validation. No jobs retried; both unchanged10s fuzz commands passed.
The sole failure is the original invitation return-link assertion at line108:
`next` is still encoded after URLSearchParams decoding. Case remains ACTIVE, byte-identical; no extra decode/skip/xfail or runtime fix. Its later account/sign-in/
join steps are unexecuted. Runtime owner must assign finding/base/files/regression.
[Full coverage, source/command/failure/artifact ledger](docs/qa/RESPONSIVE-CHECKS.md).

Earlier d93a97b ran223passed/4failed; the root-path assumption and mixed
innerText/textContent comparison were demonstrated test mistakes and corrected.
6a4aa61 validates those corrections. Original failures remain in Git/ledger.
Last fully green source is `ca908a8b5743c7d674b4bac4b492c2a5c7c46698`,147passes
with one unchanged fuzz retry. Its original deadline failure remains unresolved;
no new reliability fix or retry to obtain green is claimed. Earlier invalid-meter
assertion/source is also preserved. Failure artifact metadata only; no bytes fetched.

Current follow-up adds a separate key-only/no-script plain-registration/sign-in/
explicit-invitation/join/balance/empty-history journey, plus two404/Help layout
probes. They preserve the known broken return-link case and do not claim its fix.
Expected next suite265:112 viewport/error,20 preference,6 no-script,18 focus cases
plus unchanged baseline. This is static enumeration, not executed/pass evidence.
Current new-source runtime/aggregate CI is PENDING; inspect exact source before
any pass claim. Known failure means no green/merge-ready claim even if others pass.

Node syntax, unique case counts, whitespace, hygiene/action pins and limited
complete-outgoing-file credential/private-data/link/fence/conflict review precede
publication. Existing branch-push CI only; no tag/dispatch. No local browser/full
scanner claim. Local dependency/artifact HTTP403 boundaries remain binding;
no retry, alternate route or environment replacement. Final docs may use skip-ci;
resolve HEAD with `git rev-parse HEAD` and distinguish it from tested source.

No runtime/CSS, baseline test, helper/fixture, dependency/workflow, OPS004 runbook,
role/grant/24h invariant, threshold or security/retention change. Departure.go/A1,
pagination,C09/A2-07,audit/WIP reserved; no archived PoC restored/run.
Smallest owner action: hand over return-path finding/base/files and original test,
at least join_anon.html/render.go (owner verifies other next links), then reviewed
fix preserving safe local-path validation. Generic continuation is not handover.

PR CREATION PAUSED: automatic review rejected conditional creation without exact
approval; parent asked, no answer known. No create/retry/workaround, merge,
APPROVE review, deploy/release, credential/security change or purchase attempted.
Pending approval for frozen04e1198 does not approve this later source.

Next: inspect exact-source CI, preserve all failures; fix only demonstrated test
mistakes in owned scope. No weakened assertions/budgets or repeated fuzz retries.
Native/manual/screen-reader and independent/professional gates remain open;
other data permutations, admin mutations, loading and full security review are not
established. Preserve dirty/untracked/unpushed and external work; explicit ref fetch
if needed, no main reset/force push. No extra workers/paid calls. Usage/model switch
unavailable; keep reserve.
