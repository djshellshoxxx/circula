# Owned development consolidation — 2026-10-30

This record covers `codex/circula-consolidate-20261030`, not a product-main merge.
The owner explicitly authorized a fresh checkout after both original checkout
paths were unavailable. Original dirty files, local-only recovery refs and local
worker reservations remain inaccessible and unverified. No paid infrastructure
was provisioned; no paid external model call was made.

## Inputs and pre-action records

Origin is `https://github.com/djshellshoxxx/circula.git` with fetch refspec
`+refs/heads/*:refs/remotes/origin/*`. The fresh main worktree is clean, has no
local-only commits and remains at `4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7`.
All 17 published branches were fetched and matched live metadata; the plan head
matches its published receipt rather than the older row in its own source table.
Eleven open PRs are unchanged. No existing remote branch was edited.

| Action recorded before execution | Current input | Incoming input | Result |
|---|---|---|---|
| Create owned application worktree | Product/audit `827e651fe151f49659cc20aefe85d4af67bdaa97` | None | Clean owned worktree |
| Ordinary `git merge --no-ff` | `827e651fe151f49659cc20aefe85d4af67bdaa97` | Receipt `f4bfe6bc3f81c94b859abeae902ae862ea6e143e` | `9aaa09724fd884c2e9afa49baa85f8d48b5fcd06`; identical tree to product |
| Ordinary `git merge --no-ff --no-commit` | `9aaa09724fd884c2e9afa49baa85f8d48b5fcd06` | Plan `38a13a68ff7a5c4de97aab96634bf5c55ecf05de` | Expected HANDOFF.md add/add conflict only |

Pre-action intent files were written outside the repository before each merge,
under the session's `consolidation-evidence/merge-intent-1.txt` and
`merge-intent-2.txt`. The table preserves their inputs, actions and outcomes.

## Conflict resolution and every changed path

- `HANDOFF.md`: short current record replacing conflicting historical next steps.
- `docs/audit/HANDOFF-product-827e651-2026-10-30.md`: preserved product handoff,
  original blob `dddfee155569f736b02e5de743bd8e2d99ef5118`.
- `docs/audit/HANDOFF-plan-38a13a6-2026-10-30.md`: preserved plan handoff,
  original blob `9d2bfd33e83169e9f514eed023b060a83490a0c9`.
- `docs/BRANCH-STATE-2026-10-30.md`, `docs/INTEGRATION-PLAN.md` and
  `prompts/continue-circula.md`: unchanged incoming plan documents.
- `docs/audit/CONSOLIDATION-2026-10-30.md`: this new receipt and content disposition.

Archived handoffs add provenance headers and rebase relative Markdown links with
`../../`; all original prose and commands remain. Reversing that relocation and
removing the header must reproduce each exact original blob. The originals also
remain reachable through parent history. No runtime source, test, audit PoC,
WIP patch, dependency, workflow, financial policy or threshold changes in this
merge. The root executable remains for a separately reviewed hygiene slice.

## Old PR content disposition

A read-only worker compared exact old heads with product
`827e651fe151f49659cc20aefe85d4af67bdaa97`. Unique commit counts use
`git rev-list --count PRODUCT..OLD_HEAD`, not missing-feature estimates.

| PR / pinned head | Unique commits | Disposition |
|---|---:|---|
| #1 `22e9202f19f55aa46be4b115edbe84178e9cee0b` | 31 | Preserve separate exploratory content. Current canon represents basic mutual-credit/ledger/self-hosting/export intent; the old README's no-application claim is obsolete. |
| #2 `01b6e5c7cfc5bd9d660b80aaafe2b35de3912904` | 13 | Preserve supplementary specs/research/tooling as references; selectively reconciled requirements already exist. Different IDs/defaults prohibit wholesale import. |
| #3 `d96f2e75634bff074d26965ae93c2fe00c6f7f24` | 1 | Preserve historical shells, explicitly not approved requirements or decisions. Current authored specs supersede the structure. |
| #5 `db7826495ec2e8bd3f5459afa5f619cdc8fe8def` | 1 | Preserve historically; no ready salvage identified. README/wiki no-software claims are obsolete; unresolved licence is already in current guidance. |

#1 retains three unique `docs/brainstorming/` documents: business-model options,
32 future ideas and exploratory crypto/decentralization design. Monetization is
outside current MVP scope and no cryptographic architecture is selected there.
Its 20 `wiki/` pages are unpublished draft framework content; `wiki/README.md`
requires version-specific verification before calling instructions verified.
Preserve for later bounded documentation/scope decisions, not automatic adoption.

#2 retains supplementary specs 14–19, `docs/notes/`, `docs/plans/`,
`docs/decisions/` and `scripts/check-specs.py` / `gen-traceability.py`. Current
`RESUME-2026-10-24.md` already records reconciliation of glossary/statuses, LED-5,
policy applicability, decimals, historical-balance contracts and notifications
(e.g. current DAT-049/050, FR-054, API-083 and FR-038). A critical conflict remains:
old POL-007/008 permits decimal-scale changes before the first journal entry;
current DAT-049 requires permanent immutability from creation, even at sequence
zero. Do not reintroduce the old rule. Old plans also recommend no holds and later
stack selection; current canon has reserved outgoing amounts and Go/SQLite.
Old DEC-004 means ledger model and DEC-015 licensing; current DEC-004 is licensing.
Its different FR grammar, eight recorded cross-document conflicts and research
claims tagged [K]/[U] need mapping/re-verification. Old ADR Accepted labels cannot
supply owner approval for current decisions, nor can its scripts be imported
unchanged against current IDs. Legal/owner gates remain open.

#3's unique cryptographic-foundations shell has only research headings and
placeholders, not approved backlog. #5's README and six-line wiki Home are
superseded by current README, CONTRIBUTING, guides, dev and ops documentation.
All four PRs stay intact/open. This is a content disposition, not exhaustive
requirement coverage or authorization to close them.

## Ownership and preserved work

The coordinator owns integration. One read-only worker compared old spec content;
no runtime finding was reserved. PR #12's merged state and automated reviews do
not hand over external remediation ownership. Saved departure/decimal/disputes
fixer output remains unapplied; disputes is incomplete. C-09/A2-07 stays coordinated
external work with archive PR #13 untouched. No broad audit was repeated.
Older code heads #4/#7/#8/#9 and the security branch are checked as product
ancestors. Missing historical `c134253` / `ee78a7f` remain unrecovered.

## Workflow side effects and validation boundary

The unchanged CI workflow triggers on pushes/PRs, with contents read permission.
Its nine jobs cover three OS tests, lint/security, E2E, contract/load, fuzz,
cross-build and reproducibility. It uploads build/failure artifacts. The
smoke-deploy script starts only a temporary loopback process and stops it; no
external environment is deployed. Cancellation is scoped by GitHub ref. Scorecard
pushes are main-only and release publishing triggers only for version tags.
The owned branch push creates no tag, release or deployment.

Go 1.26.10 is absent from PATH. No compiler/module/artifact download was attempted;
prior HTTP 403 denials remain unresolved. Local make check, E2E, API contract,
ledger runtime verification and pinned gitleaks scan are unexecuted. Available
static checks and new-source CI are recorded below as observed. Old green CI is
not validation of this consolidation.

## Checkpoint evidence

The following pre-commit records are historical; the publication receipt below
records the completed source checkpoint and still-pending aggregate CI.

Pre-commit static checks on the resolved merge index passed:
- `git diff --cached --check` (all seven changed Markdown paths).
- `scripts/check-hygiene.sh` and `scripts/check-action-pins.sh`.
- Relative-file links and conflict-marker checks on all seven complete documents.
- Exact original-handoff reconstruction after undoing only header/link relocation.
- Byte comparison of the three incoming plan files with their pinned source.
- Ancestor checks for all five older code-stack heads against the product input.
- Limited full-document credential/private-data pattern scan: no findings.
This limited scan does not substitute for CI's pinned gitleaks scan.

Independent read-only delta review by worker `spec_disposition` found no blockers:
seven Markdown paths, exact first-merge parents/tree, correct pending plan parent,
byte-preserved historical notes, unchanged incoming docs, valid links, no limited
credential findings and retained ownership/release boundaries. This is an AI
review of the consolidation delta, not a professional security review. Plan
ancestry is checked after finishing the merge; CI/publication remain separate.

## Publication transport adjustment

The completed local plan merge is `f9bbf13650a721c75f3dbe7e4b3cd9420052d738`,
with parents `9aaa09724fd884c2e9afa49baa85f8d48b5fcd06` and the pinned plan head.
All three inputs passed `git merge-base --is-ancestor INPUT HEAD`; its tree is
`f7c3981726f1187da19e5dfe950d9780b3751c7d`. The local preservation branch
`preservation/circula-local-consolidation-20261030` retains these exact merges.

The normal push failed with `fatal: could not read Username for 'https://github.com':
No such device or address`. Read-only remote verification confirms the owned
branch was not created. No credentials were changed or extracted, and the failed
push was not retried. The existing authenticated GitHub connector is used for
non-forced branch publication of equivalent merge trees and the same published
input histories. Connector-generated commit identities/timestamps yield different
commit SHAs; no claim of a successful Git push is made.

[Raw local merge objects](LOCAL-CONSOLIDATION-COMMITS-2026-10-30.json) preserve the
exact original ordinary merge headers/messages for reproducibility. Their tree
objects are uploaded and compared before equivalent remote merges are published.
This preserves local work rather than resetting or overwriting it. This extra
JSON provenance path is the only addition to the seven-path consolidation scope.

## Verified remote consolidation checkpoint

Receipt merge commit: `684d365502c154aff8d8c45caff6d249b940a51b`, with
the same product/receipt parents and product tree as the local ordinary merge.
Plan merge commit: `01a40dbb3d564651b446c4060187c2fd8508bf31`, parents
`684d365502c154aff8d8c45caff6d249b940a51b` and the pinned plan head; tree exactly
`f7c3981726f1187da19e5dfe950d9780b3751c7d`, matching the local resolved merge.

Published receipt checkpoint: `d9c8747280e42f1efeedfea48d0a9c9735bc62dd`, tree
`745a97cf70ce058ecc9e01ff32c3d24a2bc00913`, identical to the local publication
receipt commit. Fetched the branch back, compared full trees with `git diff
--exit-code`, verified all three input ancestors and read back its exact SHA with
`git ls-remote --heads`. Both checks succeeded. The local publication lineage is
retained under `preservation/circula-local-publish-20261030`; the active branch
tracks the published checkpoint. No force update or destructive reset was used.

[Draft PR #17](https://github.com/djshellshoxxx/circula/pull/17) targets product
`claude/awesome-darwin-y6eaiz@827e651fe151f49659cc20aefe85d4af67bdaa97`; it was
read back draft/open and mergeable. New-source CI was created for this exact
head: [push 41041037012](https://github.com/djshellshoxxx/circula/actions/runs/41041037012)
and [PR 41041084056](https://github.com/djshellshoxxx/circula/actions/runs/41041084056).
Both are queued at this receipt, not passed. Source consolidation and publication
are complete; required aggregate acceptance remains pending those results.

## Subsequent bounded README correction

After the hygiene commit, the coordinator reserved only README's build-prerequisite
sentence plus this receipt and the root handoff. README still required Go 1.24,
while `go.mod` requires 1.26.10. It now names the exact required version and links
to the module directive, with the no-C-compiler statement scoped to building the
service (race tests still require a C toolchain). This changes documentation only.
The version match, link target and whitespace are checked locally. No dependency,
Go code, runtime test or workflow changes, downloads or toolchain installation
are part of this correction. No implementation-mirroring test was added.

Historical traceability/QA/security tables were checked for provenance and retained
at their labelled source snapshots; stale historical rows were not reclassified
as present-day missing features or rewritten as new test evidence.

## Current continuation boundary

The hygiene and README source checkpoint `b0e9d5dc03016620b541cc073ec9ac11340de0a8`
was remotely read back with exact tree equality and all three input ancestors.
See the [hygiene receipt](BUILD-ARTIFACT-HYGIENE-2026-10-30.md) for source/fix SHAs,
independent tests and superseded-CI status. Required acceptance remains the actual
latest-head CI result, not the old consolidation runs (now cancelled).

No explicit external runtime-remediation handover was found in the retrieved
PR #12 discussion; it contains automated review feedback. Original executor
worker state is inaccessible. Further A1/A3/C-09 changes are ownership-blocked
until the external coordinator/owner reconciles a disjoint reservation. This is
not a claim that those candidates are all runtime defects or that their owners
are idle. The build-output and README scopes are complete and no competing audit
was started. Preserve all owner/manual/professional-release gates.

## Later plan/readiness reconciliation

During final remote verification, the plan branch advanced from the initial input
`38a13a68ff7a5c4de97aab96634bf5c55ecf05de` to
`24f417c944e4cae79147aa7b58985829b6553212`. Its sole additional commit records a
separate session's original-executor readiness and preserves this PR's ownership.
The published continuation prompt is byte-identical. Product, receipt, main and
archive heads were unchanged. This is new documentation evidence, not a transfer
of runtime findings or permission to duplicate completed work.

Before merging, `consolidation-evidence/merge-intent-3.txt` recorded the exact current
owned source `20bda306bdd180a919db1f567e2a745ebc24ce48`, the incoming plan SHA above
and ordinary `git merge --no-ff --no-commit` intent. Only HANDOFF.md conflicted.
The incoming handoff is retained as
[plan-24f417c history](HANDOFF-plan-24f417c-2026-10-30.md), with its original blob ID
and reversible relative-link relocation. The current root handoff incorporates
actor-specific access evidence and retains local limitations and ownership gates.

Every resulting path is documentation:
- `docs/BRANCH-STATE-2026-10-30.md`, `docs/INTEGRATION-PLAN.md` and
  `docs/audit/ORIGINAL-EXECUTOR-READINESS-2026-10-30.md` match the incoming source.
- `docs/audit/HANDOFF-plan-24f417c-2026-10-30.md` archives its original root handoff.
- `HANDOFF.md` is the short current record; this consolidation receipt explains
  the reconciliation and its validation boundary.

The readiness receipt reports a working original executor and Go compiler in its
own session, no local branch commits missing remote reachability, retained ignored
files, still-absent historical commits and missing SQLite/UUID modules. This
session has no new access to that executor; earlier actor-specific failures remain
historical evidence. Download denials and external runtime reservations remain
unresolved. ARC-018, DAT-040 and DAT-048 were qualified there as already implemented;
no duplicate implementation or whole-product conformance claim follows.

Source20bda30 PR run41044078436 had four successful jobs and five queued when read
before this merge. The pinned rescan was queued, not passed; previous source failures
remain in the CI diagnosis receipt. A new merge head requires its own CI. No local
Go or pinned-scanner rerun is claimed, and no denied download was retried.

The independent read-only delta review found no blockers in all six Markdown
paths: exact incoming documents, reversible source handoff, valid relative links,
actor-specific access evidence, preserved runtime reservations and no CI overclaim.
Local checks passed: staged `git diff --check`, `scripts/check-hygiene.sh`,
`scripts/check-action-pins.sh`, 17 relative links, no conflict markers, archived
handoff reconstruction and a limited credential-pattern review of all six files.
That limited review is not a pinned scanner result. The unchanged prompt matched
the incoming source byte-for-byte. All four input ancestors are required after
commit and remote readback; local merge history will remain under a preservation
ref if connector-generated metadata changes its published SHA.

A later live read of source20bda30 push run41044069883 found eight of nine jobs
successful, including lint/security job124023683341 and its pinned secret scan;
Ubuntu race testing remained active. This passing rescan verifies the prior prose
correction at its own source. The root handoff and
[CI diagnosis receipt](CI-SECRET-SCAN-2026-10-30.md) record that observation. The
latter adds one Markdown path to the six-path merge review, for seven total;
no workflow, scanner rule, runtime code or test changes result. New-head aggregate
acceptance remains separate.
