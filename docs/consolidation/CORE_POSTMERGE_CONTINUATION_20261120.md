# Circula Core: finalization complete — 20 November 2026

**The Core consolidation finalization is complete.** The actual merged main
passed all thirteen applicable jobs, and the four independently proven historical
PRs have been reconciled. This is the current handoff for that completed lane;
separate Core-feature and future-module development continues under its existing
owners. Earlier checkpoints remain historical records at their original commits.

## Qualified main

[Core PR #127](https://github.com/djshellshoxxx/circula/pull/127) merged normally
at **18:28:43 America/Los_Angeles on 20 November 2026**, as actual main
`87651d36558a8e421e41fbd76c514045d88c74ed`. PR #129 had already merged into
that source. Both integrations are complete.

The complete tree is `43e65ee91a90ff70b70cc69d0b517b8367a2585f`. Its ordered
parents are preceding main `aa01563fcc891da60fefebb8335eca27855678e8`, then
qualified Core source `3ef43ac1e05360d2a1641742dc056afe8871879b`. The earlier
synthetic merge `6eb49aa2b2c7859fd2a8ceda207ddbb01eaa9d65` has the same
ordered parents and tree. All twelve pre-merge jobs passed; the following are
separate successful executions on actual main.

| Actual-main workflow | Completed result |
|---|---|
| [Core 44856110228](https://github.com/djshellshoxxx/circula/actions/runs/44856110228) | 9/9 jobs successful. |
| [Native readers 44856110230](https://github.com/djshellshoxxx/circula/actions/runs/44856110230) | 3/3 successful, 49 tests per platform. |
| [Scorecard 44856110197](https://github.com/djshellshoxxx/circula/actions/runs/44856110197) | 1/1 execution successful; findings remain visible. |

The final macOS race job completed at **19:22:54 Pacific**. Independent provider
observations at **19:23:28** and **19:23:46** confirmed all thirteen completed
successful jobs, all three successful push runs, and unchanged main/tree/parents.
Every full saved job log identifies the exact actual-main checkout. The
[completed evidence package](evidence/core-main-qualified-20261120/README.md)
contains original artifacts, full logs, provider observations, independent reviews,
source fingerprints, closure receipts and verified archive manifests.

## Historical PR reconciliation completed

| PR | Final disposition | Preserved source |
|---|---|---|
| #109 | Closed as already incorporated through #127; an ancestor, with zero source-only commits. | `5db3efc084e6a807bb4702c09d3235490d42b71d` |
| #112 | Closed as already incorporated through #127; an ancestor, with zero source-only commits. | `7462138fb4f7110ea4f705c6f09742a2b5a253e4` |
| #111 | Closed with six identical documents and explicit alternate WAL-fixture supersession; not an ancestor. | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` |
| #104 | Closed after runtime supersession and separate recovery of all 102 unique documents; not whole-tree inclusion. | `bcdc569cbc483761bf2d0e2d7594636ac1713d7d` |

These closures completed at **19:26:42 Pacific**. Every PR and source branch was
refreshed before and after its update; all source heads and original description
remainders were preserved. No additional runtime merge was performed. #127's
status description was updated and verified at **19:27:14**. The final open list
contains only policy drafts #83/#85/#86/#94/#107.

The [reconciliation receipt](evidence/core-main-qualified-20261120/reconciliation-result.json)
retains exact times, classifications, states and body/source-preservation checks.
#90/#102/#125/#128 were already marked merged after #127. The earlier #80/#87,
#105/#106 and future-target reconciliations also remain complete. Preserve their
original evidence and audits rather than recreating or repeating those operations.

## Finalization task retirement

The narrow task's engineering and historical-reconciliation conditions are met.
After the completed evidence package is published and its tree and blobs are
verified, pause only **Finalize Circula qualified merges**
(`Automation_e7f2412412bc81919afc68682f3c0580`) with a final completion handoff.
Do not alter the separate **Finish Circula features** or **Continue Circula future
modules** tasks. Do not start a duplicate feature lane from this continuation.

Publish on the existing `audit/core127-final-evidence-20261120` branch using a
normal expected-head fast-forward. That branch retains older runtime and must
never be merged wholesale over actual main. Future work must first inspect the
current live heads and its existing owner because this is a dated completion record.

## Evidence interpretation

The independently downloaded **actual-main** Linux artifact records **3,258 passing
named tests/subtests, six named skips and zero failures**: 1,293 top-level passes
and 1,965 passing subtests. Sixteen packages passed; two helper packages had no
tests. The 3,282 terminal-event total includes package events and skips, and is
not the count of passing tests. The archive records exact main, a clean checkout,
exit zero and a source fingerprint independently reproduced from 2,023 inputs.
Browser execution separately passed **597** tests. The full qualification review
retains all six named-skip classifications and their evidence limits.

The thirteen built requirements without test citations are inherited citation
gaps with mixed inspection, shell, manual and deployment evidence. A missing
citation alone does not establish missing implementation or unmet acceptance.
The eleven unresolved frozen legacy citations are historical code references,
unchanged from preceding main and the earlier audit. Both substantive persisted-ID
regressions are implemented and passed on actual main. Preserve frozen history;
do not add superficial tests or restore removed stubs to manufacture coverage.

Scorecard execution success does not mean every supply-chain criterion passed.
Its retained 5.9 score across eighteen criteria includes missing branch protection,
no sampled GitHub approval reviews, missing project license and other process
findings. Existing owner/release decisions remain separate. Worker review is not
a formal GitHub approval or an independent professional security audit.

## Remaining ordinary engineering

The latest security decision table is not an exhaustive list of all unfinished
engineering. The [fresh scope report](evidence/core-main-finalization-20261120/scope-review/report.md)
reconciles the older backlog with actual source:

- Current English catalogue has 1,458 entries. Broad flow externalization and
  the later full-sentence/count fixes are already included. UX-047 still retains
  broader completion/locale boundaries; do not rebuild the old 525-entry stage.
- Late driver-read errors, successful-COMMIT acknowledgment loss, and persisted
  UUID generator tests are implemented. Their coverage is narrower than real
  operating-system or writable-VFS storage failures.
- Physical SQLite read/fsync/truncate and checkpoint sync/truncate qualification,
  physical uncertain commits and migration acknowledgment-loss behavior remain
  accepted harness/environment work. Take one bounded failure boundary at a
  time and preserve committed-state/restart/retry/idempotency assertions. Do not
  recast callback errors or an EFBIG WAL write as all-device-failure coverage.
- Representative pointer cancellation, reduced-motion behavior, packaged-artifact
  configuration parity and native smoke qualification remain legitimate evidence
  work where appropriate execution environments are available. They do not follow
  automatically from the existing keyboard tests or cross-compilation.

Continue one accepted Core feature at a time under the existing feature owner,
with fresh implementation and review handoffs. This finalization continuation
does not duplicate that separately owned work or launch a second feature lane.

## Preserved decisions and independent modules

OPS-04 reader resource budgets/large-export compatibility, OPS-03 Core Windows
private-file policy, LED-R02 financial refusal disclosure, GOV-021 recovery,
AUD4-F02 reviewer-reset authority, SEC-026 visibility/retention, report escalation,
CLI alternatives, API-010's D-07 contract deviation, and reference-hardware
acceptance remain explicit decisions or scoped follow-ups. A compatible streaming
reader improvement alone would not resolve the budget policy.

All seven policy/interface branches remain outside main; policy PRs
#83/#85/#86/#94/#107 remain open/draft. All fifteen actual future-module manifests
were independently parsed and remain `enabled: false` and `integration: "HOLD"`.
No future runtime paths or imports entered Core. Preserve independent future
development, host/device/peer acceptance, and activation decisions.

No release, deployment, branch deletion, force push, history rewrite, new policy,
or claim of whole-product completion is part of this continuation.
