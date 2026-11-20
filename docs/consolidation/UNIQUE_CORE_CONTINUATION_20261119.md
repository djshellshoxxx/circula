# Unique Core consolidation — 2026-11-19 Pacific

This is the current continuation of the owner's request to consolidate unique
Circula code and finish the approved Core features. It supersedes the active
status paragraphs in older consolidation records; their source-specific
evidence remains historical and is not rewritten as evidence for this source.

## Source and integration authority

- Repository: <https://github.com/djshellshoxxx/circula>.
- Main observed at `aa01563fcc891da60fefebb8335eca27855678e8`.
- Recovered Core baseline: PR #125,
  `22c767a90f69d286642ef96e6b3c174836637a3c`, tree
  `9c1ad082bfd7befb9914000c79d5974f1017cc27`.
- Owned continuation: `integration/unique-core-completion-20261120`,
  [PR #127](https://github.com/djshellshoxxx/circula/pull/127), targeting main.
- Published checkpoint: `e1438e2d5b3d7a5cfce5b20d272a448e99da1882`, tree
  `bb7ed87c3512348913424f2b422c7aa1d164bc90`, exactly matching local
  `18a33b9cf283e01e1bf594e6f5f1dbce8c771ab2`.
- Local integrated runtime through `df4d8fd336b1f9db41b0abdf05864d695be5d33e`
  additionally contains closure, platform-audit, report-privacy and bounded
  audit-mirror repairs, plus two persisted-identifier acceptance tests. Its
  combined qualification and publication are pending at this checkpoint.

Publication uses ordinary commits and expected-head branch updates through the
connected GitHub provider because this checkout has no Git push credentials.
Each published source tree is compared with the corresponding local Git tree.
The commit mapping is retained with the evidence. No source branch has been
deleted, force-pushed, or replaced. No main merge, release, or deployment is
claimed here.

## Unique-source recovery

The frozen inventory accounts for **145 actual remote branches**. The symbolic
`origin/HEAD` alias is excluded from that count. All source names and full SHAs,
patch comparisons, merge simulations, path differences, and dispositions are in
[the branch inventory](evidence/unique-core-20261119/branches.json) and
[the independent review](evidence/unique-core-20261119/branch-uniqueness-review.md).

| Disposition at the recovered baseline | Tips |
|---|---:|
| Ancestor of the Core candidate | 66 |
| Future module or supporting topic | 43 |
| Recorded policy or interface hold | 7 |
| Historical implementation recovered or superseded | 23 |
| Patch-equivalent history | 1 |
| Merge produces the identical source tree | 3 |
| Core runtime recovered; remaining documentation or alternate fixture | 2 |
| Total | 145 |

The inventory found no additional compatible Core runtime implementation
outside PR #125. The ledger-authorization, MFA-decider and dashboard-amount
branches each merge to the identical candidate tree. PR #104's Go verifier is
already identical; replacing the Python reader with that older branch would
remove newer compression safeguards. PR #111's alternate WAL fixture serves a
purpose already covered by the retained query-only witness. Their source refs
remain available.

A later refresh observed 148 branches: this continuation, a notes-only security
review at `896d69f472652ed54c051ec82efcb7f162d1d2c4`, and an isolated feed scanner
fixture repair were added. One future-feed tip moved to that repair. None of the
145 originally inventoried source tips otherwise moved. The feed change stays
on its future branch. The new review's confirmed Core findings informed the
repairs below; its notes are not themselves runtime changes.

## Completed implementation in this continuation

| Area | Resulting behavior | Focused evidence |
|---|---|---|
| Secret-scanner controls | Preserve the original evidence bytes and narrow scanner exceptions; check both raw escaped JSON and semantically identical decoded controls. | Ten real Gitleaks control runs pass. Raw before/after findings are 8/0; decoded before/after findings are 18/0. The directory scan passed before later edits; a final scan remains required. |
| SQLite member reads | A late row-iteration failure returns the error and no partial member list, while preserving rollback and connection reuse. | Four top-level race tests with ten subcases passed after the fix; four regression cases failed before it. |
| Python reference export reader | Match the established Go reader for repeated nested manifest objects and reused file-array slots, including nulls and shrink/regrow cases. | Full reader suite: 49 passing tests. Independent probes used the actual Go verifier. This is evidence for the covered cases, not a proof of universal parser equivalence. |
| Dispute text boundaries | Count Unicode code points consistently for statement, decision and re-review limits; browser forms no longer impose a conflicting UTF-16 maxlength. | Exact 2,000-code-point inputs pass, 2,001 fail; raw initial-rationale padding is counted as the service stores it. Focused HTTP/message race checks pass. The new browser scenario is discovered but still needs a browser run. |
| Trade previews and replay | Apply current maximum-amount and checked-balance rules to previews, compute request availability correctly, and let confirmed mutations reach service-owned idempotent replay. | Focused trade/dispute race checks pass. An independently found concurrent acceptance regression was fixed and a permanent exactly-one-posting regression passes. |
| Operator JSON files | Require one nonnull JSON object and end-of-file in configuration and audit-anchor input. | Six top-level tests and 26 subcases pass under race. Valid signatures, precision, whitespace, known-field rules, and read-only database behavior are preserved. |
| Account recovery authority | A network-local administrator cannot reset an identity with retained membership/history in another network; operator and eligible single-network recovery remain available. | Focused normal and race checks pass. The earlier resource-killed compile is retained as an incomplete attempt. |
| Administrator grant provenance | Decision approval requires a complete bounded chain to an explicit founder or operator root. Missing, cyclic or exhausted ancestry refuses safely. | Normal and race checks pass, including positive roots and existing waiting/break-glass behavior. No new quorum policy is chosen. |
| Conditional listing updates | Recheck the existing revision inside the owner write transaction; stale edits return HTTP 412 without overwriting later state. | Nine top-level tests and 16 subcases pass under race. Existing ETag bytes, wildcard behavior, ownership and archive finality remain covered. |
| Scheduled closure | Recheck the current announcement tuple, due date and consistency freeze inside finalization before any effects. | Replacement, cancellation, deadline and freeze controls pass under race, including a replacement that is already due when an old batch resumes. |
| Final closure checkpoint | New version 1 records use persisted completion time in UTC, stream through a discard sink, and save reconciliation totals from the hashed snapshot. Legacy hashes remain unchanged and explicitly unversioned. | Closure race tests pass across service/store/API/contract. Golden migration covers 10,000 transactions and 61 accounts from schema 1 to 22. OpenAPI checks cover 88 operations and 170 operation/status pairs. |
| Platform audit verification | Routine and operator checks include the platform chain without counting it as a financial network or freezing unrelated networks. Version 2 anchors cover both scopes in one snapshot; version 1 retains exact signed bytes and explicitly reports limited coverage. | Focused normal and predicate race checks pass. Snapshot/restore, migration, signature and strict-input compatibility selection passes; a larger CLI race attempt was killed and is not counted as a pass. |
| Report subject privacy | The subject and listing owner cannot read or handle their own report, correlated audit rows, cached handling responses or retained audit/full exports. Independent administrators and existing System authority retain complete access. | 22 top-level and 44 subtest passes under race across service/API/HTML, with pagination and role/snapshot changes. The all-conflicted operational procedure remains held. |
| Optional audit mirror | A single delivery worker uses a 256-record FIFO and a 16 KiB encoded-record cap; acknowledgement never waits for destination I/O. Observable drop counters and a two-second drain retain the documented best-effort contract. | Exact committed-source race checks pass for ordinary transactions, ordering, size limits, overflow, write errors, blocked Write/Close and concurrent shutdown; lint reports zero issues. |
| Persisted identifiers | Production default UUIDv7 and configured generator injection are checked through actual person registration and SQLite reads. | Two focused race tests pass; no generator, historical ID or session-selector change. |

The closure API deliberately exposes `final_export.version` and verification's
`checkpoint_version`. An unversioned or unsupported checkpoint returns an
explicit invalid-state response; elapsed time is no longer reported as a
content mismatch for new checkpoints. Covered data changes, including erasure,
can still legitimately change the digest. Future export changes must preserve
the version 1 representation or introduce a separately defined verifier.

The closure-focused receipts were produced on worker `f32c4525aef4f4af22f7033d31019ca2da932eeb`
(after `a7c555d570881235b99bcb196c6409150d90609c`), integrated unchanged as
`250bf57` and `62c2614`. Platform-audit receipts use worker
`e0cd553796bb71d5ea3c6d89a88a761ae3b176c7`, integrated as `2499e50`.

## Remaining feature-contract reconciliation

The reviewed fixes above are integrated. A final source reconciliation is
checking historical partial/not-built rows against current implementation,
accepted contracts and recorded holds. In particular, the report-privacy fix
does not implement SEC-025 transaction reports or related-dispute links; their
remaining contract must be reconciled before claiming complete abuse reporting.
No new report type, reviewer privilege or escalation policy was silently added.

Report-privacy worker `1c8a4777f9eee5e1076c7581228c2501ec38b0cb` is integrated as
`da1c032`; its focused verification record is retained. Identifier worker
`7fe8dc4d11a6e6b52c9d06a5498f3ac8a2c6cf71` is integrated as `ca01990`. Mirror worker
`04e856ab391c9861c513bfd6ef4f75a7ba3b3483` is integrated as `df4d8fd`. The mirror
uses the existing explicitly best-effort policy: queue overflow, oversize
records, write failure, expired drain and process crash can leave gaps. A write
in flight at drain timeout remains unconfirmed; one delivery worker may remain
until the destination returns or the process exits. The durable database audit
is unchanged. The bound applies to the mirror drain, not arbitrary application
logging or operating-system operations.

## Qualification and evidence boundaries

[The evidence index](evidence/unique-core-20261119/README.md) identifies the actual
source-specific logs and records. Independent review found and resolved both
the repeated-file-array compatibility gap and the concurrent acceptance replay
regression. That review is a bounded source review, not SEC-033 external review
or a GitHub approval.

The final combined normal/race, lint, vulnerability, hygiene, traceability,
helper, browser and native-platform results must identify the source actually
tested. Focused passes above do not establish a combined-suite pass. Prior
provider passes on PR #125 likewise qualify its older source only.

The local browser preflight uses the unchanged Playwright launch configuration
and aborts on `about:blank` with Chromium's process-singleton socket permission
failure. No product E2E assertion ran in that attempt. The provider browser gate
remains necessary. PR #127 carries `ci:cross-platform` so the next/current PR
source runs the native Core macOS and Windows jobs rather than counting a
cross-build as native execution.

The earlier scanner-only PR runs were superseded when the continuation advanced.
The newer provider checks are pending at this checkpoint. A queued, cancelled,
killed or interrupted attempt is never recorded as a pass. Main was observed
without required status enforcement; integration will wait for observed gates
rather than relying on an automatic-merge setting.

The generated inventory contains 531 requirement IDs. A bounded review
identified 15 built rows without automated test citations; these are mapped in
the evidence directory. Two meaningful persisted-person tests now establish
the production UUIDv7 default and injected-identifier seam (DAT-001/DAT-002);
the regenerated inventory now has 13 such rows. Other rows include
source-inspection, documentation, manual,
release, and owner decisions. Artificial tests or new citations do not satisfy
those acceptance criteria. Regenerate the committed matrix after the last
source change and record actual execution evidence separately.

## Preserved completion boundaries

- All 15 future modules remain isolated, disabled and on their existing HOLD.
  This continuation does not import their clean runtime diffs into Core.
- Existing SEC-026, GOV-021, reviewer-reset, quorum/independence, break-glass,
  retention and other recorded product decisions remain with their owners.
- Manual screen-reader qualification and independent external security review
  remain external gates. Automated accessibility and source review do not
  replace them.
- Reference hardware and release budgets, native private-storage ACL
  qualification, support/licence commitments, signing and other unresolved
  deployment decisions remain separately tracked. Physical device/power-loss
  qualification is not established by the retained SQLite fault regressions.
- No operational report-handling escalation is invented when every available
  administrator is conflicted. Such reports remain retained and open pending
  the existing independent-administrator/operator process decision.
- No release or deployment is part of this Core consolidation directive.

## Next integration actions

Finish the remaining feature-contract reconciliation; retain the independent
review; freeze a clean combined source; run its actual gates; publish the
tree with its source mapping and evidence; observe provider checks; and perform
an ordinary expected-head merge only after qualification. Preserve future and
held source branches. Refresh main and peer heads before any final integration,
and record the observed merge separately from this pending checkpoint.
