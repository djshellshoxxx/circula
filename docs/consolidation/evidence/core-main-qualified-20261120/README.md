# Circula Core: qualified main and completed reconciliation

**Core finalization is complete.** All thirteen applicable actual-main jobs
succeeded, their full logs identify the exact merged checkout, and the four
historical PRs were closed after fresh source/ref checks. The final job completed
at **19:22:54 America/Los_Angeles on 20 November 2026**. Reconciliation completed
at **19:26:42**, and the final PR-description/open-list readback completed at
**19:27:14** that evening.

This package supersedes the pending status of the earlier
[preservation package](../core-main-finalization-20261120/README.md). Its frozen
provider observations remain valid historical records. This is documentary
publication on the existing evidence branch; the branch retains older runtime
and must never be merged wholesale into main.

## Qualified source and execution

| Identity | Verified value |
|---|---|
| Actual main | `87651d36558a8e421e41fbd76c514045d88c74ed` |
| Whole tree | `43e65ee91a90ff70b70cc69d0b517b8367a2585f` |
| First parent | `aa01563fcc891da60fefebb8335eca27855678e8` |
| Second parent | `3ef43ac1e05360d2a1641742dc056afe8871879b` |
| Source SHA-256 over 2,023 inputs | `9de685071744f0b9e260d53fc7d934cb4289c3b10f84a020db8e319227b77100` |

| Actual-main workflow | Result | Evidence |
|---|---|---|
| [Core 44856110228](https://github.com/djshellshoxxx/circula/actions/runs/44856110228) | 9/9 jobs successful | Builds, Windows tests, Linux/macOS race tests, reproducibility, fuzz, lint/vulnerability checks, browser/accessibility/keyboard, API/load. |
| [Native readers 44856110230](https://github.com/djshellshoxxx/circula/actions/runs/44856110230) | 3/3 jobs successful | 49 tests on each of Linux, macOS and Windows. |
| [Scorecard 44856110197](https://github.com/djshellshoxxx/circula/actions/runs/44856110197) | 1/1 job successful | Analysis execution/upload succeeded; findings remain explicit. |

The independent final provider observation was at **19:23:28 Pacific**; the
coordinator independently observed the same complete composition at **19:23:46**.
All three are successful `push` runs at actual main. Every job record and every
saved log identifies that same commit. No required job was missing, skipped,
cancelled or failed. Conditional platform steps and named test skips are recorded
separately. No product suite was rerun during this evidence review.

The [full qualification review](qualification-review.md),
[machine-readable result](qualification-result.json), and
[coordinator provider summary](root-provider-summary.json) retain the precise
checks, commands, times and limits. The source's pre-merge twelve successful jobs
remain separate execution evidence; none was substituted for these main results.

## Independently counted Linux results

The [original actual-main Linux ZIP](actual-main-linux-evidence.zip) is artifact
**14689307829**, exactly **138,870 bytes**, SHA-256
`df5b84d700ad1193afe6244178384fc8ae7fa02eeab2fbe8482220661af89a54`.
Its provider digest, upload-log digest and independently computed digest agree.
It records exact main, `dirty=false`, exit code zero and uncached race execution.

| Result category | Pass | Fail | Skip |
|---|---:|---:|---:|
| Named tests and subtests | 3,258 | 0 | 6 |
| Top-level names, included above | 1,293 | 0 | 6 |
| Subtests, included above | 1,965 | 0 | 0 |
| Packages | 16 | 0 | 2 |
| All terminal events, including packages | 3,274 | 0 | 8 |

The 3,282 terminal-event total is not a count of passing tests. Top-level passes
comprise 1,270 Test functions and 23 Fuzz entries. Six named skips cover an
ownership-guarded fixture, three subprocess helper entries, optional load and
optional statistical login timing. The separate API/load job passed its scenario.
The two package skips are helper packages without tests. Runtime skip text is not
present in the compact artifact; source-guard interpretation is explicitly labeled.

The [coordinator's independent ZIP check](root-linux-verification.json) also
verifies terminal passes for WAL rollback/recovery and both persisted ID tests.
Browser execution separately passed **597** tests. These counts are from actual
main and are not copied from the pre-merge artifact.

## Completed historical reconciliation

| PR | Verified disposition | Preserved source |
|---|---|---|
| [#109](https://github.com/djshellshoxxx/circula/pull/109) | Closed as incorporated through #127; ancestor of main, zero source-only commits. | `5db3efc084e6a807bb4702c09d3235490d42b71d` |
| [#112](https://github.com/djshellshoxxx/circula/pull/112) | Closed as incorporated through #127; ancestor of main, zero source-only commits. | `7462138fb4f7110ea4f705c6f09742a2b5a253e4` |
| [#111](https://github.com/djshellshoxxx/circula/pull/111) | Closed with six identical documents and explicit alternate WAL-fixture supersession; not an ancestor. | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` |
| [#104](https://github.com/djshellshoxxx/circula/pull/104) | Closed after runtime supersession and separate documentary recovery; not whole-tree inclusion. | `bcdc569cbc483761bf2d0e2d7594636ac1713d7d` |

All four source branches remain at their original heads. The retained receipts
contain before-and-after snapshots of each PR and its direct source ref, plus an
unchanged main-ref snapshot after each close. Only the previously owned status
prefix was replaced; the original
description remainder is byte-identical. Each PR is closed and unmerged by this
administrative action. #127's status prefix now records completed qualification
and reconciliation; it remains merged. The open-list readback contains only the
five intentionally held policy drafts.

The [compact reconciliation receipt](reconciliation-result.json) records source
identity, timestamps, classification, body hashes and preservation checks. The
full before/after provider records are in the archive below. The earlier
[independent historical proof](../core-main-finalization-20261120/history-review/REVIEW.md)
retains all source comparisons and recovery limits, including #104's **102
unique documents**, **5,206,983 original bytes**, and archive SHA-256
`6d7ad0e06ef2bb9a4cd1a70626ae328a8384828dd439e07c4bf07246ff37d385`.
Its 38 original Python test method ASTs remain within the current 49-method suite.

## Scope preserved and work remaining

The [fresh independent scope review](scope-review.md) and
[complete observations](scope-review.json) confirm 23 unchanged refs: main,
fifteen future heads and seven excluded policy/interface heads. All fifteen actual
manifests were freshly fetched at pinned heads, parsed and checked against their
Git blob identities. Every module remains `enabled: false` and
`integration: "HOLD"`. Policy PRs #83/#85/#86/#94/#107 remain open/draft/unmerged;
the seven held heads remain outside main. That review preceded the administrative
closures; the later reconciliation receipt supplies their final state.
The frozen scope report began while macOS qualification was pending. Its reference
to that pending job is historical and is superseded by the final qualification
record, which verifies macOS success at 19:22:54 Pacific.

Scorecard success is execution success. Its score is **5.9 across eighteen
criteria**, with missing branch protection, no sampled GitHub approval reviews,
license and other process findings preserved. It is not evidence that those
controls are satisfied or that an independent professional security audit occurred.

The [current completion and continuation handoff](../../CORE_POSTMERGE_CONTINUATION_20261120.md)
separates ordinary remaining engineering from policy and release decisions.
Physical storage-failure qualification, selected UI/locale and packaged-native
acceptance remain legitimate engineering work. Reader budgets, Core Windows
private-file rules, disclosure/recovery/retention policy and other held decisions
remain explicit. The separate Core-feature and future-module owners continue
their work. Finalization does not mean whole-product or release completion.

## Independent publication review

The [final publication review](publication-review/report.md) approved the corrected
sixteen-path documentary composition, with twenty-one integrity checks passed
and no remaining blockers. Its [structured result](publication-review/result.json)
records the original findings and verified corrections. The coordinator then
added this unchanged review, its supporting records and archive manifest before
regenerating the final package checksums. No runtime or workflow files are part
of this publication.

The complete eight-file review directory is retained in
[publication-review.tar.gz](publication-review.tar.gz), including its original
checksum manifest, independent closure audit, fresh provider readback and review
script. The [publication archive manifest](publication-archive-manifest.json)
records every member hash and length. Supporting-file names in the readable
publication report refer to `publication-review/` inside that archive.

## Retained originals and reproducibility

| File | Purpose |
|---|---|
| [actual-main-review.tar.gz](actual-main-review.tar.gz) | The entire frozen independent review: 54 files, including thirteen full decoded logs, original Linux ZIP, full provider records, intermediate observations, inspector scripts and checksum manifests. |
| [qualification-archive-manifest.json](qualification-archive-manifest.json) | Archive hash/size and every member's byte count/SHA-256. Paths begin with `actual-main-review/`. |
| [provider-and-reconciliation.tar.gz](provider-and-reconciliation.tar.gz) | Full coordinator final CI/main read, four guarded before/after closures, #127 description update and final open-PR list. |
| [provider-archive-manifest.json](provider-archive-manifest.json) | Archive hash/size and all three original receipt member hashes. |
| [log-manifest.json](log-manifest.json) | Decoded UTF-8 and gzip hashes for all thirteen logs, including explicit checkout witnesses. Paths are relative to the review directory inside its archive. |
| [SHA256SUMS](SHA256SUMS) | Final hashes for all other files in this published package. |

Archive members were verified byte-for-byte after construction. Both archives
use fixed timestamps, ownership and ordering. The readable qualification report,
JSON result, log manifest, original ZIP and scope records are exact copies of the
reviewers' sources; their internal relative path descriptions refer to the frozen
review archive. Earlier pending reports are explicitly timestamped historical
observations. The final result supersedes them.

Publication uses a normal expected-head fast-forward from evidence commit
`9f713dcb2157b66b4e2af7aec68a37b4a9057e6d`, with an exact staged-tree comparison
and a readback of the resulting commit and every changed blob. Once that
publication is verified, only the completed finalization automation is to be
paused. Its separate feature and future-module tasks retain their ownership.
