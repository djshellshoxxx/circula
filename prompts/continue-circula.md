# Circula: consolidate preserved work and continue development

Use this as the next coding agent's task for `djshellshoxxx/circula` only. Continue
accepted, unowned work autonomously after consolidation; stop for a genuine
access, ownership, security, capacity or required-approval blocker. Create concrete
checkpoints and draft PRs rather than ending after an inventory or plan.

## 1. Reconcile the live checkout

Read applicable repository/environment instructions. Verify origin, status,
worktrees, local-only commits and current remote refs. Preserve every dirty,
untracked and unique item; reuse the existing executor. If it is unavailable,
report the exact failure instead of provisioning a replacement. Report
actual access and workers. Use one integration owner and disjoint worker scopes
only when useful; reuse source-pinned packets and avoid duplicate audits.

This packet was refreshed on 2026-10-30 from 17 remote branches and 11 open PRs.
Read its current `HANDOFF.md`, `docs/BRANCH-STATE-2026-10-30.md`, then
`docs/INTEGRATION-PLAN.md` on `codex/circula-integration-plan-20261030`. They are
absent from the product tip until consolidated. Fetch explicit refs because this
checkout's default fetch configuration may cover only main:

```sh
git fetch --no-tags origin \
  refs/heads/claude/awesome-darwin-y6eaiz:refs/remotes/origin/claude/awesome-darwin-y6eaiz \
  refs/heads/codex/circula-resume-20261024:refs/remotes/origin/codex/circula-resume-20261024 \
  refs/heads/codex/circula-integration-plan-20261030:refs/remotes/origin/codex/circula-integration-plan-20261030
git show origin/codex/circula-integration-plan-20261030:HANDOFF.md
git show origin/codex/circula-integration-plan-20261030:docs/BRANCH-STATE-2026-10-30.md
```

Known application inputs, to be rechecked before use:

- Product plus interim audit: `claude/awesome-darwin-y6eaiz@827e651fe151f49659cc20aefe85d4af67bdaa97`.
- Separate development-merge receipt: `codex/circula-resume-20261024@f4bfe6bc3f81c94b859abeae902ae862ea6e143e`.
- Current prompt/plan: the verified live SHA of `codex/circula-integration-plan-20261030`, draft PR #16. Record this SHA before moving to an application worktree.

`main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` contains prompts only. Start
application work from the verified product tip. PRs #10/#12/#14 are already
merged; PR #12's merge preserves evidence and applies no audit fix. Both current
application merge SHAs passed 9/9 post-merge CI jobs, linked in the plan. Those
results do not validate a later consolidation or changed source.

## 2. Consolidate on a new owned development branch

Create a new branch/worktree from the verified product tip; preserve existing
branches and worktrees. Before each local merge record exact inputs and intended
action. Use ordinary history-preserving merges on this owned branch to include
both the development receipt and the current prompt/plan branch.

The pinned development receipt merges cleanly and changes no product tree. The
pinned prompt/plan merge has an expected add/add conflict in `HANDOFF.md`. Preserve
both original handoffs as source-labelled records under `docs/audit/`, repair
links if relocated, and write one short current root handoff pointing to them.
Resolve the conflict using the latest verified state; retain all underlying
source, audit, test and documentation commits. Verify each recorded input SHA is
an ancestor of HEAD and review every resulting path before implementation.

Keep archive PR #13 intact: its six unique commits include the failing C-09
regression and overlap audit A2-07. Salvage that test only through coordinated
source-pinned work, preserving assertions and original failure evidence. Older
code PRs #4/#7/#8/#9 are contained in the product tip. Older spec/README PRs
#1/#2/#3/#5 have unique conflicting content; compare and document a disposition
rather than merging wholesale or closing them. Missing historical c134253 and
ee78a7f remain unrecovered.

Consolidation is complete when the three verified input heads are ancestors,
original notes are preserved, the resulting diff is explained, required checks
are recorded for the new source, and an exact remote checkpoint is verified.
This is development-branch consolidation, not a product-main merge.

## 3. Continue small accepted development slices

Reconcile live external audit/remediation ownership before reserving any finding.
A merged or idle PR does not transfer ownership. Read the merged audit's
`interim-verification-circula.md`, its JSON evidence and WIP README only for the
selected subsystem. Verification, coverage and consolidation remain unfinished;
there are six reported Medium candidates, incomplete extra-lens checks and a
refuted skeptic result for A3-F02. Reproduce and try to disprove a candidate at the
current source before claiming a runtime defect. WIP patches are unapplied,
unreviewed references; the disputes patch is incomplete.

Suggested first independent slice, if unowned: remove the tracked 21,703,878-byte
root `circula` build artifact in a normal commit, ignore future build output and
add a narrowly justified hygiene check. Preserve its original commit; no history
rewrite. Then work through accepted owned Medium candidates A1-01, A1-02/A1-11,
A1-03, A3-F01, A3-F02 and A3-F16, using the detailed references in the branch-state
record. Take one bounded slice at a time; continue another ready independent
slice if a particular finding is owned or blocked.

For each runtime fix, preserve the original regression and failing-before output,
implement the smallest spec-conforming correction, run focused tests then relevant
aggregate checks, and obtain independent delta review. A1/A3 PoCs fail before a
fix; A2 PoCs assert defect presence and pass before a fix, so explicitly convert
the regression direction. Record finding IDs, source/fix SHAs, actual results and
deferred work. Update affected specs/traceability without inventing owner decisions.

Historical balances, independent-founder/24-hour approval behavior, GOV-027
support guidance, the request-ID assertion correction and automated axe/keyboard
CI already exist. Verify stale gap tables against current code before calling
anything missing. Prefer ready, unowned accepted work over duplicate implementation.

## 4. Validate and checkpoint every small milestone

Use Go 1.26.10 and the application's documented commands in CONTRIBUTING.md,
Makefile and docs/qa. Preserve `make check`, relevant `make e2e`/`make contract`,
ledger verification, hygiene/action pins and the pinned secret scan. Keep tests,
financial thresholds, append-only/equal-and-opposite ledger rules, grant lineage
and the 24-hour approval delay intact. Record commands, exact source and observed
passes, failures and unexecuted prerequisites separately.

This executor has unresolved dependency-download and artifact-byte HTTP 403
denials. Preserve them; do not retry denied operations or use alternate routes
without materially new relevant authorization. Another session's working downloads
or old green CI do not establish local access. Use an existing authorized CI route
only after checking workflow side effects; never claim unrun local tests passed.

Review outgoing source/documents for secrets and private data, then checkpoint
source plus handoff with a normal push to the owned branch and read back the
remote SHA. Open/update a draft development PR against the current product tip.
Maintain a short HANDOFF.md with done/remaining work, exact refs, ownership, tests,
blockers and the next bounded task; keep detailed receipts linked under docs/audit.
Continue ready work rather than waiting for a timer. Keep a reserve for checkpointing;
paid model use must follow the owner's authorized broker and recorded budget.
No new paid infrastructure or service purchase is authorized by this prompt.

## 5. Preserve owner and release boundaries

Merges into existing external/main targets, formal GitHub APPROVE submissions,
PR closure, tags, releases, deployments, purchases and credential/security-setting
changes require the applicable concrete owner authorization. Existing approvals
for completed #10/#12 merges are not approval for future product-main integration.
Preserve branches and review gates; no force push, destructive reset or discarded
unique work. A prompt-only documentation branch does not establish application CI.

Licence, DEC-072/074, UX-047 locale scope, confirmation of built defaults, pilot
jurisdictions, unconfirmed release thresholds, manual screen-reader evidence and
independent professional security review remain owner/release gates. AI review
and green CI do not satisfy them. Financial-policy changes and deferred crypto,
tokenization/redemption scope are not personal implementation choices.

When genuinely blocked, report the exact action, target, reason and whether it
was denied, pending or uncertain; preserve the checkpoint and continue independent
allowed work. Finish each session with a verified handoff and the next actionable
slice, not an unsupported claim that the entire product is complete.
