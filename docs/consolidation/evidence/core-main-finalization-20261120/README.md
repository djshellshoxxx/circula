# Actual-main verification and historical reconciliation — 20 November 2026

This package records fresh independent review after Core PR #127 reached `main`.
It supplements the historical receipts without changing the qualified runtime.
The coordinator owns publication and PR reconciliation. The reviewers performed
read-only source, provider-log, artifact, and preservation checks; they did not
rerun product suites or make remote changes.

## Current source and qualification boundary

| Item | Exact value |
|---|---|
| Actual main | `87651d36558a8e421e41fbd76c514045d88c74ed` |
| Previous main, first parent | `aa01563fcc891da60fefebb8335eca27855678e8` |
| Qualified Core source, second parent | `3ef43ac1e05360d2a1641742dc056afe8871879b` |
| Actual main and source tree | `43e65ee91a90ff70b70cc69d0b517b8367a2585f` |
| Executed synthetic PR checkout | `6eb49aa2b2c7859fd2a8ceda207ddbb01eaa9d65` |
| Preceding evidence commit | `c1a34476de017ab70acc99181ebbe16c337fc0e6` |

The synthetic checkout has the same ordered parents and complete tree as the
actual main merge. The independent CI reviewer read all twelve pre-merge job
logs. The coordinator separately read all twelve successful job records and
both Git commit objects. The Linux artifact's recorded source digest was
independently recomputed over 2,023 source files in clean actual main, excluding
the generated `docs/TRACEABILITY.md`, and matches exactly.

These facts establish the tested source composition. They do not turn a
pre-merge execution into an actual-main push execution.

The [provider observation](provider-observation-20261120T183747.json) is pinned
to **20 November 2026, 18:37:47 America/Los_Angeles** (21 November, 02:37:47 UTC):

| Workflow | Execution result at that observation |
|---|---|
| [Pre-merge Core 44843655337](https://github.com/djshellshoxxx/circula/actions/runs/44843655337) | Nine applicable jobs completed successfully. |
| [Pre-merge native readers 44843655247](https://github.com/djshellshoxxx/circula/actions/runs/44843655247) | Three jobs completed successfully. |
| [Actual-main Core 44856110228](https://github.com/djshellshoxxx/circula/actions/runs/44856110228) | Nine jobs queued. |
| [Actual-main native readers 44856110230](https://github.com/djshellshoxxx/circula/actions/runs/44856110230) | Three jobs queued. |
| [Actual-main scorecard 44856110197](https://github.com/djshellshoxxx/circula/actions/runs/44856110197) | One job completed successfully. |

A [later provider observation](provider-observation-20261120T184325.json) at
**18:43:25 America/Los_Angeles** records three successful jobs (Core build,
macOS reader, scorecard), five running jobs, and five queued jobs, all on the
same actual-main SHA. No required job had failed; final qualification remained
pending.

Later provider observations belong in separate dated receipts. Historical PR
closure still requires complete applicable post-merge qualification and fresh
source-ref checks. No queued result is recorded as a pass.

## Evidence counts corrected and preserved

The [independent artifact review](test-evidence/test-evidence-review.md) preserves
the [original provider ZIP](test-evidence/linux-source-test-evidence.zip), its
byte hashes, exact source identity, terminal counts, and six named skip reasons.
The ZIP is 137,945 bytes with SHA-256
`d4368bca6e6d52bd22ac3e8a1acef798ac501322652aaa500a80315197ed8476`.

| Linux pre-merge classification | Passed | Failed | Skipped |
|---|---:|---:|---:|
| Named tests, including subtests | 3,258 | 0 | 6 |
| Top-level named tests | 1,293 | 0 | 6 |
| Passing subtests | 1,965 | 0 | 0 |
| Packages | 16 | 0 | 2 |
| All terminal events | 3,274 | 0 | 8 |

The 3,282 total is a terminal-event count, not a pass count. The two skipped
packages are helpers without tests. Three named skips are subprocess helpers
whose parent tests passed. The other named skips concern a privileged ownership
fixture, the separately executed opt-in load test, and opt-in statistical login
timing. Terminal JSON omits skip text; the review distinguishes source-based
classification from a recorded runtime reason.

The other source job logs record 597 passing browser cases, 49 Python reader
tests on each native platform, and five passing fuzz campaigns totaling
1,400,000 executions. The separate load job records 100 completed trades and
200 committed writes. These remain attributed to their own pre-merge jobs.

The [traceability review](test-evidence/test-evidence-review.md) accounts for
all thirteen built rows without test citations and all eleven unresolved frozen
legacy code-citation rows. All are inherited; no newly uncited built row was
introduced. DAT-001 and DAT-002 now have meaningful persisted-person regressions
with passing events in the artifact. Do not invent tests or resurrect old stubs
to make historical citation counts reach zero. API-010's administrative token
scope remains the explicitly documented D-07 contract deviation.

## Four historical PRs: preservation review complete

The [fresh independent historical review](history-review/REVIEW.md) and
[complete recomputation](history-review/proof.json) use actual main rather than
the earlier candidate. Live PR and source-ref observations are preserved.

| PR | Correct classification | Reconciliation after post-merge gates |
|---|---|---|
| #109 | Source is an ancestor of actual main; zero source-only commits. | Close as incorporated through #127, without another merge. |
| #112 | Source is an ancestor of actual main; zero source-only commits. | Close as incorporated through #127, without another merge. |
| #111 | Six documents are identical; the retained WAL witness setup supersedes the alternative fixture. The following 1,877 bytes and assertions are identical. | Close with explicit fixture supersession; never claim source ancestry. |
| #104 | All 118 original paths accounted for: fourteen identical, two Python alternatives superseded, 102 documentary paths separately recovered. All 38 original Python test-method ASTs are unchanged in the current 49-method suite. | Close with explicit runtime supersession and documentary recovery; never claim whole-tree inclusion. |

The original PR104 recovery archive was checked again against original Git
bytes, modes, blob identities, lengths and hashes: 102 files / 5,206,983 bytes,
205 nested ZIP payloads, byte-identical internal/external manifests, and all
nine package checksums. Original and nested JSON parse checks also passed.
No archived program was executed. Historical host/source reconstruction and
scanner limitations remain unchanged in the original recovery package.

The [non-executable reconciliation plan](reconciliation-plan.json) records exact
expected source heads and all conditions. All four PRs were still open and draft
when reviewed. Their branches remain preserved. #90, #102, #125 and #128 were
already automatically marked merged following #127; their history is retained.

## Scope and unfinished work

The [independent scope review](scope-review/report.md) and
[structured proof](scope-review/proof.json) verify 2,024 tracked files, no future
runtime paths/imports, all fifteen actual target manifests disabled and on HOLD,
all seven held policy/interface heads outside main, and the five policy PRs
#83/#85/#86/#94/#107 still open/draft at their reviewed heads.

That review also corrects stale backlog interpretation. The old 525-entry
catalogue description predates the current 1,458-entry catalogue and later
sentence/count repairs. Existing identifier, late driver-read, and successful
COMMIT-acknowledgment regressions should not be rebuilt. Remaining physical
SQLite read/fsync/truncate/checkpoint qualification, pointer cancellation,
reduced-motion evidence, and same-artifact/native smoke qualification are
ordinary engineering or environment-dependent acceptance, not governance holds.

Separate policy/platform decisions still govern reader resource budgets and
large-export compatibility, Core Windows private-file ownership/ACL behavior,
financial refusal disclosure, administrative/reviewer recovery, abuse-signal
visibility/retention, report escalation, interface choices, and reference-hardware
budgets. This review does not select those decisions or claim complete release
acceptance.

## Scorecard execution is not an all-clear

The [extracted provider result](scorecard-result.json) preserves the actual-main
scorecard findings. The workflow ran and uploaded its analysis successfully.
It still reports no enforced branch protection, no GitHub approval reviews among
its sampled changesets, no license file, and other supply-chain/process gaps.
Independent worker review is distinct from a submitted GitHub approval. The
automated scanner's zero detected vulnerabilities is limited to its own check;
it does not close known source findings or supply a professional external audit.

## Preservation and publication

`SHA256SUMS` covers every file in this new package except itself. Nested original
history-review checksums remain intact. The original provider ZIP and independent
review files are copied without byte changes. New documentary receipts preserve
their exact observation times. The existing evidence branch is intentionally
based on older runtime: publish these documents there with an expected-head
update, and never merge that whole branch over qualified main.

This continuation changes no product source, dependency, workflow, test assertion,
policy, future-module state, or release/deployment setting.
