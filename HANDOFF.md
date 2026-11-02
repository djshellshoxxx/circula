# Circula navigation QA restart checkpoint

Repo: djshellshoxxx/circula; owned branch: codex/circula-noscript-qa.
Worktree: /workspace/circula-noscript-qa, original executor; one local worker.
Input receipt: f51295dd9bcce68467fcf928ff9d3de235db21ef.
Latest tested source: b231dc5d160d98614825bfc6f8099f3e409d5930.
Run 41472456920, attempt 1, completed: eight jobs passed; browser job
125369302163 failed with 324 passed / one failed (325 executed, 4.4 minutes).
All 60 new UX-002 navigation cases passed; the earlier 265-case baseline retains
its same one failure. Both unchanged 10-second fuzz commands passed; no reruns.
[Detailed coverage, commands, failures, artifacts and recovery](docs/qa/RESPONSIVE-CHECKS.md).

UX-002 measures screens FROM Home, not return-to-Home navigation. Twenty-nine
core/adjacent destinations at both viewports plus two own-trade dispute-form
paths use at most three visible rendered links, successful exact paths/queries
and visible headings. The ordinary matrix is GET-only. Two variants prepare
own 0.07 synthetic completed trades before measurement; seeded IDs stay unchanged.
No dispute/decision/reversal is posted. UX-002 remains partial for other signup/
join/auth/error/multi-network states, primary submissions and native/manual review.

Sole browser failure: original return-link assertion in zz-noscript.spec.js:108.
After Create an account, next remains encoded after URLSearchParams parsing.
The case is ACTIVE and byte-identical; subsequent account/sign-in/join steps are
unexecuted. No extra decode, skip/xfail, deleted assertion or runtime fix.
Passing direct entry and navigation paths do not certify the broken return link.
CI is not green/merge-ready. Runtime owner must assign finding/base/files and
unchanged regression before a fix, at least internal/web/templates/join_anon.html
and internal/web/render.go; owner verifies other next links and safe-path checks.
Generic continuation does not transfer ownership or establish another writer stopped.

Previous tested d75f31e76c0e1f67dcb744ff8899487b57edda6a: 264/1, eight other jobs
passed. Earlier test mistakes and corrections remain in Git/ledger. Last fully
green source: ca908a8b5743c7d674b4bac4b492c2a5c7c46698, 147 passes after one
unchanged fuzz retry. Its original deadline failure is unresolved; no reliability
fix claimed. Earlier invalid-meter assertion/source is also preserved.
Latest artifact: 13156910519, 116382 bytes; digest/retention in ledger, no bytes
fetched. Existing dependency/artifact HTTP403 boundaries remain binding; no retry.

Final receipt is documentation only with skip-ci. Resolve HEAD using
`git rev-parse HEAD`; CI belongs to the tested source above, not this receipt.
Local Node syntax/static registrations, hygiene/action pins, whitespace and limited
complete-outgoing-file credential/private-data/link/fence/conflict checks passed.
No local browser/full scanner result is claimed for this receipt. Normal source
push and remote readback succeeded; no unpushed source intentionally left.
Frozen responsive branch: 04e11989b811b4541a20b4e06e7a020beabb6af3.
PR17/application: 7643fb616208cc318939ded809e6cada66bb7415; PR18:
9c0bc500cb19f11f18694c76d134f4bfebf896e7; external integrated branch:
827e651fe151f49659cc20aefe85d4af67bdaa97. No published runtime handover found.
No runtime/CSS, baseline fixture/helper/test, dependency/workflow, OPS004 runbook,
role/grant/24-hour invariant, threshold, security/retention change. Departure.go/A1,
pagination, C09/A2-07 and audit/WIP reserved; no archived PoC restored/run.
PR creation PAUSED: automatic review rejected conditional creation without exact
approval; parent asked and no answer known. No create/retry/workaround, merge,
APPROVE review, deployment/release, credential/security change or purchase.
Pending approval for frozen older source does not approve this later source.
Next owner action is the bounded runtime handover above. Manual/screen-reader,
native and independent/professional review remain open; no full feature/release claim.
Preserve dirty/untracked/unpushed and external work; fetch explicit owned refs;
no main reset/force push, extra worker, paid call or environment replacement.
Usage/model switch unavailable; keep reserve. No weakened gates or repeated fuzz retries.
