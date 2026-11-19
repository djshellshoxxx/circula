# Branch and PR refresh after draft PR #112

This additive refresh records **130 remote branches and 112 pull requests**.
Branch and PR enumeration ran from **2026-11-19 13:06:24 through 13:09:16 UTC**;
the provider checks and reviews below were read at their individually recorded
times through **13:11:19 UTC**. Main remains
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.

The meaningful new Core source is **PR #104's Python manifest-validation
follow-up**. Its additive recovery has been assigned separately, with review
and qualification still required. PR #111 contributes six preserved audit
documents and an alternate WAL test witness; the frozen Core candidate already
has the required witness behavior. PR #110 and the new PR #108 revision restore
two Core baseline files that are already byte-identical in the candidate.

The [machine-readable evidence](evidence/branch-refresh-post112-20261119.json)
contains all current branch heads and dispositions, all PR states and targets,
immutable changed-path/blob comparisons, merge states for all 28 open PRs,
263 check-run records, submitted reviews, selected provider failure output,
and document-preservation hashes. This report was prepared on isolated branch
`docs/branch-refresh-post112-20261119` from frozen local source
`e0d27cc7cb73299219439919b260991b824d594a`.

## Preserved inventories and base documents

The [114-branch inventory](BRANCH_INVENTORY.md),
[120-branch checkpoint](BRANCH_REFRESH_20261119.md), and
[128-branch/109-PR snapshot](BRANCH_REFRESH_POST109_20261119.md) remain
byte-for-byte unchanged, including their three JSON evidence files. The new
evidence verifies all six SHA-256 values against the report base. In particular:

| Preserved record | SHA-256 |
|---|---|
| Initial 114-branch JSON | `8989490d42896e49d05a105ee22dc76a7a4b7c76db47bee327aa8e8717f8b967` |
| Prior 128-branch JSON | `3278831fbb0d67206c8790dce4fc0bf376f6eb7b5436e0efb5ea72a813571c92` |

The original requirements and history remain linked at the unchanged main
revision. The [specification index](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/README.md)
defines the authoritative reading order; the
[conventions and invariants](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/00-conventions.md),
[functional requirements](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/04-functional-requirements.md),
[ledger and audit specification](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/06-data-ledger-audit.md),
[security specification](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/09-security-privacy.md),
and [quality plan](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/12-quality-test-plan.md)
remain the base documents. Their historical build-status text is not evidence
for a later candidate.

The [decision log](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/DECISIONS.md),
[completion matrix](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/plans/circula-completion-matrix.md),
[original branch ledger](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/plans/BRANCH-CONSOLIDATION-2026-11-17.md),
[original PR ledger](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/plans/PR-CONSOLIDATION-2026-11-17.md),
and [historical specification archive](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/archive/specs-historical/README.md)
preserve their existing roles. Archived competing drafts remain provenance;
this refresh does not promote them into replacement requirements.

## Counts and collection boundary

| Category | Prior snapshot | Current snapshot | Delta |
|---|---:|---:|---|
| Remote branches | 128 | 130 | Two new, three moved, 125 unchanged, none deleted |
| Pull requests | 109 | 112 | New #110, #111 and #112 |
| Open PRs | 25 | 28 | Three newly opened |
| Merged PRs | 47 | 47 | No additional merge |
| Closed without merge | 37 | 37 | No additional closure |
| Known future modules | 15 | 15 | No new module identity or activation |
| Advertised remote tags | Zero in initial inventory | Zero | Independently read again through Git |

`git fetch --all --tags` completed successfully. REST branch pages contained
**100 + 30** entries; all-state PR pages, ordered by creation, contained
**100 + 12**. Branch names are unique and PR numbers 1–112 are present.
The true fetched remote heads were checked against those records, excluding
symbolic `origin/HEAD`. `git ls-remote --tags origin` returned empty output and
exit zero. The connector rejected the REST tags endpoint, so the tag count
comes from that Git advertisement. Standalone issues, releases and deployments
were not re-enumerated.

REST pages and CI records are separate observations. No state is asserted
beyond its recorded timestamp, and unchanged branch dispositions inherit their
earlier source-review provenance rather than a new review of every branch.

## Current Core composition

| Reference | Exact head | Disposition |
|---|---|---|
| Main | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` | Unchanged |
| [#90](https://github.com/djshellshoxxx/circula/pull/90) | `839e86935767288e594914ad711486616a604eca` | Unchanged Core integration target; draft, open |
| [#109](https://github.com/djshellshoxxx/circula/pull/109) | `5db3efc084e6a807bb4702c09d3235490d42b71d` | Advanced from `afb0da2`; draft, open against #90 |
| [#111](https://github.com/djshellshoxxx/circula/pull/111) | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` | New qualification/evidence draft against #90 |
| [#112](https://github.com/djshellshoxxx/circula/pull/112) | `08d4b0078e8c308ae7f353849783913f4302882a` | New Core UI draft against #109 |
| Frozen local source | `e0d27cc7cb73299219439919b260991b824d594a` | Exact full-tree match to published #112 |

Published #112 and local `e0d27cc` both have tree
**`9ae8efd0306da7de8984e239254aa6b01996f02e`**. This establishes source equality,
not shared CI results or commit ancestry. The subsequent documentation-only
report commit will have a different tree.

The #109 revision changes 93 paths from its initial publication: it preserves
the late #90 audit documents, adds the completed dispute-confirmation and
terms/rules/leave presentation work, repairs printf-style constructor calls,
and records source-specific evidence. Catalogue entries increase from 982 to
1087. The published #112 increment changes 84 paths and reaches 1431 entries,
covering the remaining bounded administration/shared presentation work and
decimal-remediation correction. The existing Python compressed-stream check,
WAL observation fixture and their regressions remain in the frozen source.

## New and moved branch dispositions

| Branch / PR | Previous → current head | Source disposition |
|---|---|---|
| `integration/circula-master-consolidation-20261119`, [#104](https://github.com/djshellshoxxx/circula/pull/104) | `c3c991aa3c12` → `707a4363a869` | Thirteen-path manifest-validation follow-up; approved for separately owned additive Core recovery, pending qualification |
| `master/recover-analytics-20261119`, [#108](https://github.com/djshellshoxxx/circula/pull/108) | `29a0312c5c4d` → `38b9bcd0e049` | Six paths; two already-present Core baseline restorations and four analytics documentation/evidence updates |
| `feature/core-catalogue-continuation-20261119`, #109 | `afb0da286098` → `5db3efc084e6` | Owned Core continuation described above |
| `codex/api036-qualification-20261119`, #111 | New → `45356ca10e1e` | One alternate WAL witness and six preserved audit documents; no missing production implementation |
| `integration/core-ui-completion-20261119`, #112 | New → `08d4b0078e8c` | Exact publication of the frozen local continuation tree |
| `fix/fnet207-ci-proven-20261119`, [#110](https://github.com/djshellshoxxx/circula/pull/110) | Unchanged `aeb70db0255d` | Existing inventoried branch now has a non-draft PR against isolated recurring #97 |

### PR #104: preserve both independent export-reader improvements

The new commit `707a4363a86922d152ed2d203c9ea184fe4b323f` is a direct child of
the previously inventoried `c3c991aa`. Its reader change validates known
manifest-field types and all duplicate occurrences, resolves supported case
aliases and null behavior, validates timestamps, and rejects nonstandard JSON
constants. Its tests exercise those boundaries and valid controls. These
changes are absent from frozen `e0d27cc`; the original verifier contribution's
presence in Core does not imply this new follow-up is present.

The exact path disposition is:

| Paths | Required treatment |
|---|---|
| `scripts/read-export.py`, `scripts/test_read_export.py` | Apply the new manifest work additively, retaining frozen Core's decompressor-completion guard and all three `test_zip_compression_*` regressions |
| `.github/workflows/export-reader.yml` | Review and recover the dedicated Linux/macOS/Windows Python workflow with the combined tests |
| Ten files under `docs/audits/master-consolidation-20261119/` | Preserve with their original source-qualified evidence in that separate recovery |

Those ten files are `CORE_PROVIDER_CHECKPOINT.json`, `CORE_QUALIFICATION.json`,
`EXECUTION_CHECKPOINT.md`, `MANIFEST_PARITY_EVIDENCE.json`,
`MANIFEST_PARITY_QUALIFICATION.md`, `pending-traceability.patch`, and
`verification/python-native-local.json`, `verification/python-native-local.log`,
`verification/python-native-workflow-review.json`, and
`verification/python-native-workflow-review.md`.
The machine-readable path records contain every exact blob.

PR #104's current reader/test composition predates our stream-completion fix.
Replacing either complete file would remove that independently verified
protection. The JavaScript catalogue worker owns additive recovery from
`e0d27cc`, including meaningful baseline/after checks. This report copies none
of those thirteen paths and claims no combined-reader or native-workflow pass.

### PRs #108 and #110: no missing Core baseline repair

Both heads restore `cmd/circula/auditanchor.go` and
`internal/service/departure_pagination_regression_test.go`. Their blobs are
respectively `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` and
`3b117ecfe2f01e7e2363b9d201503cde6e0f5595`, exactly matching unchanged main and
frozen Core. #110's remaining change splits a fixed recurring-test digest into
literal parts without changing the digest or assertions; its branch/head
already appeared in the 128-branch inventory before the PR existed.

#108's other four paths update its README, recovery note and two evidence
records. Its production and test Go files, native workflow and nested
dependency file are unchanged. Analytics metadata remains `enabled:false`,
`integration:HOLD`, with no future-feature dependencies. Its historical
completion percentage and tested-source metadata are retained as attributed
module evidence, not a Core release or feature-adoption decision.

## Six PR #111 documents preserved exactly

All six documents below are copied byte-for-byte from
**`45356ca10e1ecd6e6441e0171910dd918b36f5d9`**. Five replace earlier copies at
the same paths; their earlier blobs remain in Git history. The qualification
JSON is new. The evidence records each source blob, SHA-256, byte count, prior
blob/hash and immutable source URL.

| Document | Change from frozen source | Original source |
|---|---|---|
| [API036_COMPLETION.md](../audits/API036_COMPLETION.md) | Updated completion and source-specific qualification account | [Immutable #111 copy](https://github.com/djshellshoxxx/circula/blob/45356ca10e1ecd6e6441e0171910dd918b36f5d9/docs/audits/API036_COMPLETION.md) |
| [API036_QUALIFICATION.json](../audits/API036_QUALIFICATION.json) | Added structured receipts and limitations | [Immutable #111 copy](https://github.com/djshellshoxxx/circula/blob/45356ca10e1ecd6e6441e0171910dd918b36f5d9/docs/audits/API036_QUALIFICATION.json) |
| [CONSOLIDATION_FINAL_REPORT.md](../audits/CONSOLIDATION_FINAL_REPORT.md) | Added API-036 combined-source qualification section | [Immutable #111 copy](https://github.com/djshellshoxxx/circula/blob/45356ca10e1ecd6e6441e0171910dd918b36f5d9/docs/audits/CONSOLIDATION_FINAL_REPORT.md) |
| [FEATURE_COMPLETION_STATUS.md](../audits/FEATURE_COMPLETION_STATUS.md) | Added qualification status and limits | [Immutable #111 copy](https://github.com/djshellshoxxx/circula/blob/45356ca10e1ecd6e6441e0171910dd918b36f5d9/docs/audits/FEATURE_COMPLETION_STATUS.md) |
| [REMAINING_BLOCKERS.md](../audits/REMAINING_BLOCKERS.md) | Added the combined-source follow-up boundary | [Immutable #111 copy](https://github.com/djshellshoxxx/circula/blob/45356ca10e1ecd6e6441e0171910dd918b36f5d9/docs/audits/REMAINING_BLOCKERS.md) |
| [SECURITY_REVIEW.md](../audits/SECURITY_REVIEW.md) | Added WAL-fixture/security qualification account | [Immutable #111 copy](https://github.com/djshellshoxxx/circula/blob/45356ca10e1ecd6e6441e0171910dd918b36f5d9/docs/audits/SECURITY_REVIEW.md) |

The PR's only code change is twelve test lines in
`internal/service/sqlite_ioerr_linux_test.go`: an independent open read
transaction keeps the failed WAL observable after production cleanup discards
the failed writer. Frozen Core already uses a pinned, query-only connection
without an open read transaction to prevent SQLite's last-close checkpoint.
The two test files differ only in this witness block. The query-only fixture
and its existing focused receipts are retained; this report changes no test
code and does not substitute the alternate witness.

The copied receipts name repaired Core source `9edf59a626c0332a122bb618865673d8b1911a4d`,
the equivalent published repair `ab7d64c5a31653e3a2a14ddb88940e59b4d02892`,
and their documentation reconciliation. Their reported full normal and focused
race results are evidence for those named sources. They do **not** qualify
#109, #112, frozen `e0d27cc`, or the future combined manifest-reader candidate.
The copied documents also preserve interrupted aggregate race/lint/vulnerability
attempts and browser-launch limitations. Their older CI snapshots remain
historical; the freshly observed provider states below take precedence for
this inventory only.

## Exact-head CI, merge states and reviews

All 263 returned check records have the requested `head_sha`. Each check,
status and review collection fit within one 100-record page; names shared by
distinct runs were not collapsed. Workflow collections for main and seven
focus heads were also complete. Separate commit-status endpoints returned
`pending` with **zero status contexts** for every queried head; this does not
override the check-run results.

| Head | Check results at observation | Raw PR merge state |
|---|---|---|
| Main `ead63b5` | 13 success, 1 failure, 5 queued, 3 in progress | Not a PR |
| #90 `839e869` | 9 queued | Open draft; mergeable true, `unstable` |
| #104 `707a4363` | 12 queued | Open draft; mergeable true, `unstable` |
| #108 `38b9bcd0` | 1 success, 1 skipped, 9 queued | Open draft; mergeable true, `unstable` |
| #109 `5db3efc` | 9 queued | Open draft; mergeable true, `unstable` |
| #110 `aeb70db` | 7 queued, 1 skipped | Open non-draft; mergeable true, `unstable` |
| #111 `45356ca` | 9 queued | Open draft; mergeable true, `unstable` |
| #112 `08d4b007` | 7 queued, 1 skipped | Open draft; mergeable true, `unstable` |

Current focus workflow records include
[#109 run 44526219355](https://github.com/djshellshoxxx/circula/actions/runs/44526219355),
[#110 run 44524017877](https://github.com/djshellshoxxx/circula/actions/runs/44524017877),
[#111 run 44528166744](https://github.com/djshellshoxxx/circula/actions/runs/44528166744),
and [#112 run 44529781200](https://github.com/djshellshoxxx/circula/actions/runs/44529781200).
None is recorded as a successful completed workflow here. The optional
cross-platform placeholder's skip on #112 is retained as a skip, not native
qualification. Main's historical failed fuzz check remains alongside newer
successful/pending records.

The remaining open PRs were also refreshed:

| PR | Check results | Raw merge state |
|---|---|---|
| #67, #80 | Each: 2 success, 5 failure, 1 skipped | Each: true, `unstable` |
| #78, #82, #87, #88, #89 | Each: 7 success, 1 skipped | Each: true, `clean` |
| #83 | 7 success, 1 skipped | false, `dirty` |
| #85 | Zero checks | false, `dirty` |
| #86 | 6 success, 1 failure, 1 skipped | true, `unstable` |
| #92, #100 | Each: 9 queued, 1 skipped | Each: true, `unstable` |
| #94 | 7 queued, 1 skipped | null, `unknown` at observation |
| #95 | 8 queued, 1 skipped | true, `unstable` |
| #97 | 5 success, 5 failure, 1 skipped | true, `unstable` |
| #99 | 4 success, 2 failure, 1 queued, 2 in progress | true, `unstable` |
| #102 | 4 success, 1 failure, 4 queued | true, `unstable` |
| #103 | 8 success, 1 skipped, 1 in progress | true, `unstable` |
| #105 | 3 success, 2 failure, 5 queued, 1 skipped | true, `unstable` |
| #106, #107 | Each: 7 queued, 1 skipped | Each: true, `unstable` |

Across all 28 open PR review collections, 21 submitted records were returned.
Every record is `COMMENTED`; none is `APPROVED`. Git mergeability, a clean raw
merge state, source inclusion, and a submitted comment remain distinct from
approval and current-candidate qualification.

### Source-bound old-head failures

Three old-head job logs were inspected to identify whether their failures
represented newly missing Core work:

- [#99 Windows job 135522756570](https://github.com/djshellshoxxx/circula/actions/runs/44513300863/job/135522756570)
  fails the migration fixture's assumption that the head migration creates at
  least two objects, and the standalone help loader's missing `msg` function.
  The frozen candidate already contains both fixture repairs. This is not a
  current Windows pass or a newly diagnosed production failure.
- [#99 quality job 135522757126](https://github.com/djshellshoxxx/circula/actions/runs/44513300863/job/135522757126)
  exits on ten `generic-api-key` scanner findings in audit Git/blob-hash
  evidence. The exact redacted locations/rules are preserved in this report's
  evidence and handed to the separately assigned scanner investigation.
- [#102 quality job 135521077519](https://github.com/djshellshoxxx/circula/actions/runs/44512804697/job/135521077519)
  exits on a `generic-api-key` match in ordinary UI-scope prose at line 133 of
  the preserved 120-branch report. Its blob
  `41ed3e181f0f57ad41fd7a36cd1e31c63c4220dd` is unchanged in frozen Core.
  Original report bytes and detector coverage must be retained when resolving
  that false positive.

These results belong to #99 `0a73ee4` and #102 `f29ebb01`. No failing #112 check
was observed in the recorded snapshot. The scanner investigation is separate
from this documentation change and must establish its own scoped regression
and provider outcome.

## Remaining disposition

Recover PR #104's new manifest validation additively, retain both independent
reader protections, integrate the preserved #111 evidence, and qualify the
exact resulting candidate. The parent-owned full race run on frozen `e0d27cc`
was still in progress during this refresh; no result is inferred. Separate
COMMIT-acknowledgment-loss test work is also outside this publication snapshot.

The fifteen future modules remain isolated. GOV-021 and SEC-026 retain their
previous policy holds and separate migration ownership; unchanged branches
have not acquired integration approval through this refresh. No runtime,
test, workflow, original inventory, main branch, peer branch or remote ref is
changed by this report. Only the six explicitly sourced audit documents and
this additive report/evidence are committed locally. No builds or application
tests were run for this inventory task.
