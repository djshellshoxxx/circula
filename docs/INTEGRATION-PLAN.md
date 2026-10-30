# Circula integration receipts and proposed main plan

Updated 2026-10-30. Repository: `djshellshoxxx/circula`.
This document records completed development merges and proposes later work. It
does not authorize product integration to `main`, a release, or closing old PRs.

## Completed owner-approved development merges

Before each action, the source/target SHAs, CI evidence, review scope and intended
normal merge were recorded in the coordinator's ignored local evidence directory.
The owner approved the recommendations with “i approve recommendations, go ahead”.
The PRs were marked ready, merged normally with their exact expected head SHA,
and read back as merged; the target refs were verified remotely. No formal GitHub
APPROVE review was submitted, source branch was overwritten, or workflow changed.

| PR | Exact source | Target before merge | Verified merge result |
|---|---|---|---|
| [#10](https://github.com/djshellshoxxx/circula/pull/10) | `claude/awesome-darwin-y6eaiz@ae512e51d4f7a611119a5567225b862969770207` | `codex/circula-resume-20261024@30d18a1626427fa6aa14f70c8df05f3e807efab5` | `f4bfe6bc3f81c94b859abeae902ae862ea6e143e` |
| [#12](https://github.com/djshellshoxxx/circula/pull/12) | `claude/sharp-archimedes-tu76rh@fe111799ffabb2427d6d94feb32a037fa3ed2af8` | `claude/awesome-darwin-y6eaiz@ae512e51d4f7a611119a5567225b862969770207` | `827e651fe151f49659cc20aefe85d4af67bdaa97` |

PR #10 contains historical balances, independent-approver UI/fixtures, support
guidance and the request-ID assertion correction. Its merge tree equals source
`ae512e5...`. PR #12 adds 18 Markdown/JSON/text audit paths only; its merge tree
differs from `ae512e5...` only under `docs/audit/2026-10-28/`. No runtime fix was
applied. The three WIP patch packets remain unapplied and unreviewed as labelled;
the disputes packet is incomplete. Audit verification, coverage and consolidation
remain unfinished despite the evidence PR being merged. Merging it does not
satisfy the independent professional security-review gate.

## Verification receipts and limits

| Exact source | Existing CI evidence | Observed result |
|---|---|---|
| `ae512e51d4f7a611119a5567225b862969770207` | [push 40820305135](https://github.com/djshellshoxxx/circula/actions/runs/40820305135), [PR 40820312600](https://github.com/djshellshoxxx/circula/actions/runs/40820312600) | 9/9 jobs successful in each run |
| `fe111799ffabb2427d6d94feb32a037fa3ed2af8` | [push 40811138499](https://github.com/djshellshoxxx/circula/actions/runs/40811138499), [PR 40811136958](https://github.com/djshellshoxxx/circula/actions/runs/40811136958) | 9/9 jobs successful in each run |
| `f4bfe6bc3f81c94b859abeae902ae862ea6e143e` | [post-merge 41032607314](https://github.com/djshellshoxxx/circula/actions/runs/41032607314) | Completed success; all 9/9 jobs passed |
| `827e651fe151f49659cc20aefe85d4af67bdaa97` | [post-merge 41032669029](https://github.com/djshellshoxxx/circula/actions/runs/41032669029) | Completed success; all 9/9 jobs passed |

The nine-job suites include OS tests/race tests where configured, E2E with axe
and keyboard coverage, formatting/vet/lint/vulnerability/secret checks,
cross-build, reproducibility, contract/load and fuzz. No new manual rerun or CI
dispatch was requested. Source/target ancestry and conflict simulations passed.
Targeted review covered the historical read, own-account access, fixed snapshot,
timestamp cutoff, UI escaping, retained independent-approver/24-hour checks and
the test-only request-ID correction. This is not a whole-codebase re-audit.

`git diff --check` passes for PR #10 and the prompt-only PR #15. For PR #12 it
reports 137 whitespace diagnostics entirely inside the three preserved raw
unified-diff `.patch.txt` packets; the other audit documents pass. Context lines
in those packets include a diff marker before original indentation. No artifact
was rewritten, test weakened or gate disabled to suppress these diagnostics.
Those packets must not be applied without source-pinned review and regressions.

Full application tests were not rerun in this executor. Previous dependency
downloads and artifact-byte retrieval were denied with HTTP 403; those denials
remain unresolved here and were not retried or routed around. Other sessions'
local results and the recorded remote CI are separate evidence, not proof that
this executor's access changed. The product requires Go 1.26.10; the ignored
local official compiler and ephemeral caches are not part of the source backup.

## Preserved work and branch disposition

- Current product-plus-audit tip: `claude/awesome-darwin-y6eaiz@827e651fe151f49659cc20aefe85d4af67bdaa97`.
- Development target after #10: `codex/circula-resume-20261024@f4bfe6bc3f81c94b859abeae902ae862ea6e143e`.
  This merge receipt is a separate descendant of `ae512e5...`; preserve its
  ancestry when constructing a later consolidated candidate.
- External audit source `claude/sharp-archimedes-tu76rh@fe111799ffabb2427d6d94feb32a037fa3ed2af8`
  was not edited. Its ownership/WIP must be reconciled before remediation;
  inactivity or PR closure is not an ownership transfer.
- [PR #14](https://github.com/djshellshoxxx/circula/pull/14) was already merged;
  `codex/circula-support-boundaries-20261028@837d51b0d5374822b28f1b3a9206f9e2d23bd60c`
  remains preserved and is contained in the current product tip.
- [PR #13](https://github.com/djshellshoxxx/circula/pull/13) stays draft/unmerged at
  `checkpoint/circula-handoff-20261025@3d0080a9c43680f69f81217af3b42374e7160bf8`;
  frozen source is `59f7af9bcb39162c94756560178f71e439aef474`. Its C-09 regression
  has genuine unchanged-ETag/409-instead-of-412 CI failures plus QF1003 lint.
  It overlaps A2-07 and conflicts with the product tip in `HANDOFF.md` and
  `e2e/tests/security.spec.js`. Do not merge it wholesale. Preserve local retired
  `91851e0560c4c7a656968aa56a0f14ecec511d99`, reachable from that remote archive.
- PRs #4/#7/#8/#9 are older stack heads already contained in the product tip.
  #9 still has failing CI at its old source, while later fixes are in the tip.
  They were neither merged individually nor closed during this continuation.
- PRs #1/#2/#3/#5 are competing early specification/README branches, each with
  unique commits and conflicts against the product tip. Keep them intact. Compare
  content and salvage accepted unique requirements before proposing closure;
  scope-expanding crypto/tokenization ideas are not automatically accepted.
- [PR #15](https://github.com/djshellshoxxx/circula/pull/15) remains intact at
  `claude/vibrant-brahmagupta-3uofnj@33b200e00c088c86ffb0d0d8176f1575376a9b49`.
  This separate documentation successor descends from that exact commit, corrects
  stale CI/axe/ownership guidance and adds this plan. It preserves the author's
  branch and original PR; no competing product implementation was started.
- `main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` still contains prompts only.
  Missing historical `c134253`/`ee78a7f` remain unrecovered, not recreated.

## Proposed route to main — not executed

1. Reconcile live refs, external remediation ownership and exact post-merge CI.
   Finish audit verification/coverage/consolidation and reproduce/disprove the six
   reported Medium candidates before treating interim labels as final. Reserve
   disjoint work explicitly. Preserve ledger, monetary thresholds, independent
   founder/grant-chain rules and the 24-hour approval delay.
2. Fix accepted owned findings in small reviewed commits with preserved
   failing-before evidence. Reconcile C-09/A2-07 without merging the archive.
   Remove the tracked 21,703,878-byte `circula` build artifact in a normal hygiene
   commit and ignore future build output; do not rewrite history or force push.
   Revalidate stale gap tables against code; automated axe CI already exists.
3. Create a new consolidation branch from the current product-plus-audit tip.
   Incorporate the separate `f4bfe6b...` development-merge ancestry with an ordinary
   merge, preserving unique commits on both sides. Review the resulting tree,
   expected changes and all fresh source/target SHAs; never reset either source.
4. Run required checks against the actual candidate: `make check`, relevant E2E
   and API-contract coverage, the pinned secret scan, hygiene and workflow-pin
   checks. Resolve cancelled/pending/failed checks without weakening gates.
   Keep local prerequisite failures distinct from actual test failures.
5. Open one reviewed candidate PR from that consolidated tip to current `main`,
   with exact refs, findings fixed/deferred, actual checks and retained blockers.
   Request the owner's concrete approval before the proposed product-main merge.
   Preserve history with a normal merge; no stale intermediate PR substitutes
   for reviewing the complete candidate. This plan does not perform that action.
6. Only after a verified product-main merge, propose closure of redundant stack
   PRs and older conflicting documentation PRs with a salvage/disposition list.
   Do not delete branches or discard original audit/PoC/WIP evidence.

Manual screen-reader records, independent professional security review, licence,
unconfirmed release thresholds and other explicit owner decisions remain separate
release/pilot gates. Assistant reviews or green CI do not resolve them. No tag,
deployment, release, purchase, credential/security-setting change or external
message is authorized by this plan.

## Documentation checkpoint

This separate branch descends from PR #15's original source and contains only
three Markdown paths: the corrected prompt, this plan and `HANDOFF.md`.
Whitespace and all three relative-file links passed; the limited scan of the
complete documents found no credential/private-data patterns. There is no
application or workflow in this prompt-only tree, so no application test pass or
new full local pinned-scanner pass is claimed for its document head. Product CI
above belongs to the exact merge SHAs, not this separate documentation commit.
The original branch is unchanged and the corrected successor is published as
[draft PR #16](https://github.com/djshellshoxxx/circula/pull/16). No product-main
merge or old-PR closure is performed by publishing this proposal.
