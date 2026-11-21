# Circula Core: post-merge continuation — 20 November 2026

This is the current continuation for the Core finalization lane. Earlier
checkpoints remain historical records. Read the live refs and jobs before any
action; this document is pinned to the observations it cites.

## Outcome and current boundary

[Core PR #127](https://github.com/djshellshoxxx/circula/pull/127) merged normally
at **18:28:43 America/Los_Angeles on 20 November 2026**, as actual main
`87651d36558a8e421e41fbd76c514045d88c74ed`. PR #129 had already merged into
that Core candidate. Do not repeat either integration or recreate their fixes.

The source `3ef43ac1e05360d2a1641742dc056afe8871879b`, tested synthetic merge
`6eb49aa2b2c7859fd2a8ceda207ddbb01eaa9d65`, and actual main share complete
tree `43e65ee91a90ff70b70cc69d0b517b8367a2585f`. The synthetic and actual
merge also share ordered parents: preceding main `aa01563fcc891da60fefebb8335eca27855678e8`,
then the qualified source. All nine pre-merge Core jobs and all three native
reader jobs succeeded. Four specialized independent reviews completed across
successive worker sets: CI checkout/results, historical preservation, scope and
remaining source boundaries, and Linux artifact/traceability interpretation.

At **18:37:47 America/Los_Angeles**, actual-main scorecard had completed
successfully; the nine Core and three native reader push jobs remained queued.
The [new evidence package](evidence/core-main-finalization-20261120/README.md)
preserves provider records, exact counts, original Linux evidence, source proofs,
scope boundaries, and a conditional reconciliation plan. A later observation
may supersede job statuses, never the recorded time or source of this snapshot.

The [18:43:25 provider checkpoint](evidence/core-main-finalization-20261120/provider-observation-20261120T184325.json)
records three successful, five running, and five queued actual-main jobs. There
were no failures. Historical PR closure remains conditional on completion.

## Required next action

Inspect existing actual-main runs [Core 44856110228](https://github.com/djshellshoxxx/circula/actions/runs/44856110228),
[native readers 44856110230](https://github.com/djshellshoxxx/circula/actions/runs/44856110230),
and [scorecard 44856110197](https://github.com/djshellshoxxx/circula/actions/runs/44856110197).
Require the complete applicable current-main composition to succeed. Do not
restart queued work, lower assertions, treat skipped/cancelled/missing jobs as
passes, or repeat long unchanged local suites while provider jobs wait. Read
complete logs and repair a verified failure if one occurs.

After successful post-merge qualification, refresh the four historical PRs and
their source refs, then apply the [reconciliation plan](evidence/core-main-finalization-20261120/reconciliation-plan.json):

| PR | Preserved source | Correct final disposition |
|---|---|---|
| #109 | `5db3efc084e6a807bb4702c09d3235490d42b71d` | Already incorporated by ancestry; close without another merge. |
| #112 | `7462138fb4f7110ea4f705c6f09742a2b5a253e4` | Already incorporated by ancestry; close without another merge. |
| #111 | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` | Six identical documents plus explicitly superseded WAL fixture; not an ancestor. |
| #104 | `bcdc569cbc483761bf2d0e2d7594636ac1713d7d` | Runtime supersession plus all 102 unique documents durably recovered; not whole-tree inclusion. |

All four preservation proofs passed independently. All original source branches
must remain. #90/#102/#125/#128 were already automatically marked merged after
#127; observe live states before mutating them. Preserve #128's original audit
and distinguish evidence inclusion from remediation of its findings.

Update PR descriptions and publish dated reconciliation receipts on the existing
`audit/core127-final-evidence-20261120` branch. Use expected-head updates and
verify the published Git tree. That evidence branch retains older runtime:
never merge it wholesale over actual main. Once post-merge qualification and
the four proven reconciliations are complete, the narrow finalization task can
complete. Separate Core-feature and future-module development continue in their
existing ownership lanes.

## Evidence interpretation

The pre-merge Linux artifact has **3,258 passing named tests/subtests, six named
skips, and zero failures**: 1,293 passing top-level names and 1,965 passing
subtests. Sixteen packages passed and two helper packages had no tests. The
3,282 count in its log totals terminal events, not passing tests. Its verified
source fingerprint matches current main, while its execution remains pre-merge.
The artifact review records all skip classifications and their evidence limits.

The thirteen built requirements without test citations are inherited citation
gaps with mixed inspection, shell, manual and deployment evidence. A missing
test citation alone does not establish a missing implementation or unmet
acceptance criterion. The eleven unresolved frozen legacy citations are historical code
references, unchanged from preceding main and the earlier audit. Two substantive
identifier regressions are implemented and passed. Do not modify frozen history,
add superficial tests, or restore removed stubs to manufacture complete coverage.

Scorecard execution success does not mean all supply-chain checks passed their
criteria. Its result still reports missing branch protection, no sampled GitHub
approval reviews, missing project license, and other process findings. The
existing owner/release decisions remain separate.

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
