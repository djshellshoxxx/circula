# Final documentary publication review

**Verdict: APPROVE — no publication blockers remain.** The final qualification and closure claims are supported by the retained provider records, original artifact bytes and complete job logs. Both documentary wording issues identified during review were corrected without changing frozen source evidence.

Reviewed at **2026-11-21 03:37:06 UTC**. Repository: `djshellshoxxx/circula`.

| Review identity | Value |
|---|---|
| Evidence base | `9f713dcb2157b66b4e2af7aec68a37b4a9057e6d` |
| Initial staged tree | `9964506f0e1e9bcfdf56e7f77c5c50166600afc6` |
| Corrected reviewed staged tree | `4e25db7841065ee7e5245bb779fb1f74c3234a14` |
| Actual runtime main | `87651d36558a8e421e41fbd76c514045d88c74ed` |
| Actual runtime tree | `43e65ee91a90ff70b70cc69d0b517b8367a2585f` |
| Reviewed staged paths | 16, all documentary |
| Independent integrity checks | 21 passed; zero failed |

The reviewed change contains only the canonical continuation handoff and the new `core-main-qualified-20261120` evidence package. No runtime, workflow, test, dependency or other product files changed. The existing evidence and earlier dated observations are preserved.

## Qualification is tied to actual main

The raw final independent provider snapshot and the separate coordinator snapshot both identify the exact actual-main commit, tree and ordered parents. The three runs are completed successful `push` executions on main:

| Workflow | Run | Jobs |
|---|---:|---:|
| Core CI | 44856110228 | 9/9 successful |
| Native export readers | 44856110230 | 3/3 successful |
| Scorecard | 44856110197 | 1/1 successful execution |

The job identities match the expected sets. No required job is missing, skipped, cancelled or failed in those sets. Conditional skipped steps remain distinguishable from skipped jobs and test events. All **13 full decoded logs** match their saved byte counts and SHA-256 values; each contains an explicit checkout command followed by the exact actual-main SHA. The original compressed logs are byte-identical to the frozen independent source directory.

The final macOS race job completed at **2026-11-21 03:22:54 UTC**, or **November 20, 19:22:54 Pacific**. Its log contains 16 package pass lines. The browser log records **597 passed**. Each native reader log independently records **49 tests and OK** on its respective Linux, macOS or Windows runner.

The final independent observation is **03:23:28 UTC** and the coordinator observation is **03:23:46 UTC**. These receipts qualify actual-main runs; the earlier synthetic/premerge execution is not substituted for them. Scorecard execution success is correctly limited: its retained result is **5.9 across 18 criteria**, with its findings preserved.

## Original Linux evidence and recomputed counts

The actual-main ZIP is exactly **138,870 bytes**, SHA-256:

`df5b84d700ad1193afe6244178384fc8ae7fa02eeab2fbe8482220661af89a54`

The digest and size match provider artifact **14689307829**, the workflow/head metadata, the upload log and the untouched original download. Both ZIP members match their saved member digests and sizes; the archive integrity check passes. Its recorded command is the uncached race suite, with the exact main SHA, `dirty=false` and exit code 0.

The recorded source digest matches the separate frozen calculation over **2,023 included source inputs**:

`9de685071744f0b9e260d53fc7d934cb4289c3b10f84a020db8e319227b77100`

The publication review independently recomputed the terminal-event totals from the ZIP:

| Category | Pass | Fail | Skip | Total |
|---|---:|---:|---:|---:|
| Named tests and subtests | 3,258 | 0 | 6 | 3,264 |
| Top-level named entries | 1,293 | 0 | 6 | 1,299 |
| Subtests | 1,965 | 0 | 0 | 1,965 |
| Packages | 16 | 0 | 2 | 18 |
| All terminal events | 3,274 | 0 | 8 | 3,282 |

The top-level passing entries consist of **1,270 Test functions and 23 Fuzz entries**. The named/subtest categories are nested; they must not be summed as independent test populations. The report correctly avoids calling all 3,282 terminal events passes. There are no duplicate terminal identities or unknown terminal actions. The coordinator's three reconciliation-relevant event records, including WAL recovery and persisted identifier checks, occur verbatim among passing events in the original artifact.

These results establish the recorded CI executions. They do not independently prove whole-requirement, operational, policy, hardware-performance or release acceptance.

## Archive and copy integrity

All **14 entries** in the package SHA256SUMS match the exact package file set, with no duplicate entries or missing files. All three nested manifests pass. The two archives contain unique, sorted, regular files with safe relative paths, normalized ownership/mode/time metadata and zero gzip timestamps. Both archives were rebuilt independently from their inspected member bytes; both rebuilds are **byte-identical** to the staged archives.

| Archive | Files | Bytes | SHA-256 |
|---|---:|---:|---|
| `actual-main-review.tar.gz` | 54 | 349,937 | `34ead5191abd628a83bec932bacbed3a6a02d7aed803d72436cb50ffdd73d8c4` |
| `provider-and-reconciliation.tar.gz` | 3 | 103,869 | `f2f8e57a4ba130c900c3403c6eca8ed7be25edd64cd3cb33ee3eba9168e951c9` |

Every archive member matches its manifest length and digest, and every member is byte-identical to its original scratch source. This includes the full independent review, intermediate observations, thirteen complete logs, original ZIP and all three full provider/reconciliation JSON receipts. The readable qualification report/result, log manifest, original ZIP and scope report/result are also exact copies of their frozen sources. All **20 relative Markdown links** in the canonical handoff and principal reports resolve.

The archives were inspected as data. No archived scripts or payloads were executed.

## Historical reconciliation and preserved descriptions

An additional independent closure audit verified the raw before/after receipts, compact hashes, local ancestry and scope classifications:

| PR | Disposition | Original description remainder preserved |
|---|---|---:|
| #109 | Closed/unmerged; exact source is an ancestor, zero source-only commits | 3,395 bytes |
| #112 | Closed/unmerged; exact source is an ancestor, zero source-only commits | 8,530 bytes |
| #111 | Closed/unmerged; documented supersession, not ancestry | 4,454 bytes |
| #104 | Closed/unmerged; documented runtime supersession and document recovery, not whole-tree inclusion | 6,739 bytes |

All four exact source heads and direct refs match before and after. Each saved post-close main ref remains at the qualified actual-main SHA. The four PR closure timestamps fall within their respective saved before/after intervals; reconciliation completed at **03:26:42 UTC / 19:26:42 Pacific**. All twelve recorded before/after/tail SHA-256 values independently recompute.

PR #127 remains merged at the exact main commit. Its source, merge timestamp and closure timestamp are preserved, and its **31,849-byte** original description remainder is unchanged. Only the owned status prefix was updated. The merged PR description and final open list were verified at **03:27:14 UTC / 19:27:14 Pacific**.

The administrative closure language accurately distinguishes #109/#112 ancestry from #111/#104 supersession. It preserves the earlier #104 recovery limits and does not claim whole-tree inclusion or total product completion.

## Scope and fresh provider readback

The copied scope report/result are byte-identical to the earlier independent review. Their stored actual manifest contents were reparsed and their lengths, SHA-256 values and Git blob identities independently recalculated: **all 15 modules remain boolean `enabled: false` and `integration: "HOLD"`**. All 23 recorded refs match their expected heads, and all seven excluded policy/interface heads remain outside main in the retained ancestry checks. The separate scope observations retain their original timestamps: refs/PRs at **03:23:28–03:23:29 UTC**, and manifest retrieval/verification through **03:26:01 UTC**.

A further native GitHub read at **03:33:11 UTC / 19:33:11 Pacific** confirms main is still the exact qualified commit. The complete open-PR list contains exactly **#83, #85, #86, #94 and #107**, all open drafts, all at their expected unchanged heads, with null merge/closure timestamps. The full fresh responses are retained in this review directory.

The canonical handoff keeps the outstanding engineering work, policy decisions, performance evidence and release acceptance explicit. It preserves separate feature/future-module ownership and permits retirement only of the finalization automation after verified publication.

## Corrections resolved during review

1. **Receipt wording:** the initial README described a main refresh immediately before every closure. The retained per-action JSON independently supports before PR/source-ref checks and after PR/source-ref/main checks; it contains no separate before-main field. The README now states exactly what the archive retains. No missing receipt was reconstructed and no frozen record changed.
2. **Historical macOS wording:** the original scope report still contains its pending-macOS sentence. The package README now explicitly identifies that sentence as historical and superseded by the final successful qualification at 19:22:54 Pacific. The original scope files remain unchanged.

The child closure audit is frozen with its original finding. This final publication review records the independently checked resolution, avoiding an apparent unresolved issue when the historical child report is read separately.

## Publication followthrough and review limits

This approval covers the corrected **16-path staged documentary composition** identified above. The coordinator will preserve this frozen review directory, add the readable final report/result, regenerate the package manifest and verify the resulting documentary scope before expected-head publication from `9f713dcb2157b66b4e2af7aec68a37b4a9057e6d`. Remote publication and post-publication blob readback remain coordinator actions; this report does not claim the final evidence commit is already published.

The review performed read-only repository/provider inspection and evidence verification. It did not execute product suites, change shared checkout files, mutate PRs/refs/automation or claim that remaining feature work is complete. Timestamped observations are not a lock against later changes.

Supporting material is retained in `integrity-result.json`, `fresh-provider-readback.json`, `verify_package.py`, and `closure-audit/report.md` / `closure-audit/result.json`. The final `SHA256SUMS` covers every other file in this frozen review directory.
