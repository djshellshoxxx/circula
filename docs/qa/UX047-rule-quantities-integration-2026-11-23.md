# UX-047 rules-log quantity integration qualification

Checkpoint: 2026-11-23 Pacific. Product and native qualification follow-up are integrated; every configured applicable final-main check completed successfully.

This qualifies one bounded English catalogue correction. UX-047 remains partial;
whole-UI completion, translation scope and project/release completeness are not
claimed. The completed #130 setup/cardinal precision work was not repeated.

## Exact revisions and preservation

| Object | Revision |
| --- | --- |
| Starting main | `fff81fc7f64cb4858c09d4a3a6f1ff0a55a1a337` |
| Product correction | `037f10a7d8729ab7e3add4c06e724e30b8e8cd77` |
| Final PR #131 source | `a2b4229031cb18ff3517ad429b86e54c9edc329a` |
| Qualified synthetic PR checkout | `7f775778c728d6cbc7c5b79f65be19309e78cf6f` |
| PR #131 normal-merge main | `d3c5ce67a252a9111e98729c7b967ecb9cc7cdc8` |
| PR #131 shared source/synthetic/main tree | `004ff13caaf1532a30d488432110f2499e443a68` |

The PR #131 merge has parents starting main and final source. Its product code
remains unchanged by the native qualification follow-up:

| Follow-up #132 object | Revision |
| --- | --- |
| Starting main / observed timeout | `d3c5ce67a252a9111e98729c7b967ecb9cc7cdc8` |
| Windows-budget code | `089c3a196d436e8c3fb4c71d8949e57529b0382a` |
| Final follow-up source | `ae4f369b5f8e506e804bebc386d010cb66a189c7` |
| Qualified follow-up synthetic | `701780d8ea0aa2c21ff24c0627001567edc2ba0a` |
| Final actual-main integration | `9cb4cc0250db487d6fd4760227ce906c4ea5f58f` |
| Shared follow-up source/synthetic/final-main tree | `38b6ed1aefc9d83fbd73581dc047d7b8ebc1e1ba` |

The follow-up preserves the complete Windows workload and `-count=1`, adds an
explicit bounded twenty-minute package timeout inside the unchanged thirty-minute
job, and changes no product code. The original default ten-minute alarm was
observed while a newly started test was in SQLite-close cleanup; it was not a
reported assertion failure. This does not prove that arbitrary I/O always finishes.
The original Windows failure log remains in the archive.

The follow-up Mac attempt 1 was cancelled near its sixty-minute job boundary
without an assertion failure or Go forty-five-minute package alarm. Its completed
CLI/API/contract packages took 2.535/2.857/5.077 times as long as the prior pass
with the same image, Go version and command. The exact cause was not logged;
the job-cap interpretation remains an inference. Independent review supported
one full unchanged-head Mac-only retry, preserving race instrumentation,
`-count=1`, `-p 2`, the forty-five-minute package limit and sixty-minute job limit.
That retry passed. GitHub carries the other eight successful job records into
attempt 2 with new IDs but their original execution timestamps; they are reused
evidence, not eight new executions. All original evidence is retained.
Follow-up source checkpoints
link maintained final qualification conditionally so completed work is not rerun.
The follow-up topic `codex/ux047-windows-qualification-20261123` is also preserved. Topic
`codex/ux047-rule-quantities-20261123` remains at the reviewed head. Current refs,
tree equivalence and all seven applicable #131 PR jobs were checked before its
normal merge. All nine applicable #132 PR jobs were checked before its separate
normal merge. Repository automatic branch deletion was off; post-merge ref readback
confirmed preservation. No force push, history rewrite, branch deletion,
deployment, release or licence selection occurred.

## Correction and acceptance

The rules log previously converted policy quantities with strconv before passing
them as strings into complete catalogue sentences. Allowed values 1000 and 3650
therefore bypassed catalogue number formatting. `fmtAny` now passes int/int64
quantities through the new `rules.quantity` entry. The catalogue has 1,459 keys.
The three visible monetary keys still use exact fixed-point domain formatting;
version identifiers and numeric-looking user text stay ungrouped. Public reads
still filter the private dual-control threshold. No auth, tenant, ledger,
SQLite, policy mutation, audit or balance behavior changed.

The five product/test files are `internal/web/h_gov.go`,
`internal/web/messages/en.json`, `internal/web/messages/messages_test.go`,
`internal/web/messages/terms_rules_leave_test.go` and
`internal/web/terms_rules_leave_catalog_test.go`. A two-line addition in
`.github/workflows/ci.yml` runs the three affected tests before the unchanged
full race recorder. Five related documentation files update the source handoff,
TODO, message guidance, historical DEC-054 wording and acceptance/traceability
record. The PR #131 diff contains exactly eleven files; all eleven blobs
were read back exactly. The four-file qualification follow-up contains only CI and documentation changes. No new requirement/test symbol was introduced, so the
generated traceability inventory is unchanged; its freshness check passed.

## Executed proof

| Qualification | Exact evidence and result |
| --- | --- |
| Native red regression | Run `45232559048`, completed failing job `137895869325`, source `52d6c943b97b3502b6a8f1a7269c7f150e2270ba`: messages selection passed; the rules subtest failed four int/int64 grouping assertions and two rendered-row assertions. The full race step was skipped after the expected focused failure. The overall workflow was later cancelled by its successor after the failed job/log completed. |
| Corrected PR | Run `45233048885`, attempt 1: seven applicable jobs passed; conditional native matrix skipped. Linux artifact: 3,278 named passes, six named skips, zero failures; sixteen passing packages plus two no-test packages. All six added regression subtest events passed. Browser suite: 597 passed. |
| Original actual main | Run `45237650449`, attempt 1: eight jobs passed, including Linux race, macOS race (sixteen passing packages plus two no-test packages) and 597 browser tests; native Windows failed the original default ten-minute package alarm. This original run is a failure, not a qualified all-platform main pass. |
| Native qualification follow-up PR | Run `45241575868`, latest attempt 2 with eight initial same-head gate results carried forward and a full Mac-only retry: all nine current-head gate records successful. Full Mac retry passed sixteen test-bearing packages plus two no-test packages, with service 1351.914s/store 789.219s/web 518.693s/messages 1.324s; command/race/count=1/p=2/package 45m/job 60m unchanged. Windows also passed sixteen test-bearing packages plus two no-test; browser 597; Linux 3,278 named passes/six skips/zero failures and all six added subtest events passed. |
| Final actual main | Run `45259862932`, attempt 1: all nine jobs succeeded on the first attempt. Linux race: 3,278 named passes, six skips, zero failures; all six added events passed. Browser: 597 passed. Native Windows (non-race) and macOS Intel (race) each passed sixteen test-bearing packages plus two no-test packages. Windows service/store/web/messages: 418.169/378.039/136.100/0.127s; Mac: 1401.507/758.408/558.034/1.313s. Both used `-count=1`; Mac retained `-race -p 2 -timeout=45m`, Windows `-timeout=20m`. No CI finalizer remains for this bounded slice. |
| Actual-main Scorecard | Run `45237649791`, attempt 1 succeeded for exact main; score 5.9 across eighteen checks. Policy/licence findings remain open. The final actual-main Scorecard run `45259862745` also succeeded for exact `9cb4cc0`, again 5.9 across eighteen checks; findings remain open. |
| Local available checks | Node browser-message tests: sixteen passed, zero failures/skips, on files matching their pinned Git blobs. JSON uniqueness/format and whitespace checks passed. No local Go compiler was available; no local Go execution is claimed. |

PR artifact ZIP SHA-256:
`ca29bdc3c92f44fd17f5c0196b9ae3c079a55263a35f82b346bfb951e55a5f23`
(138,776 bytes), independently matched to the provider digest. It records clean
synthetic head `7f775778`, exit zero and source digest
`ee83674270aa1313e75f7a298f36ce9a76fd3e4f0cf0f810fd9c5e4d92dc6534`.
Original-main Linux artifact: SHA-256 `5757f3a544c66cabae9c6f986628dde65f552c16814934e140c9042b7df7cfa8` (138,602 bytes), provider digest verified. Clean actual head `d3c5ce67`, exit zero, the same source digest as its PR; 3,278 named passes, six named skips, zero failures.
Follow-up PR Linux ZIP: SHA-256 `0a330ee5dde77c29d499dd56f6db161396537e2fe4ecaa39242a9d5b25c38927` (138,164 bytes), provider digest verified. Clean synthetic head `701780d8`, exit zero and source digest `d6287cc6a4d99735fb105cc9d8aea880706a8e5171537645a2a7177ed0b59886`; 3,278 named passes, six named skips, zero failures. This physical Linux execution was retained from attempt 1 in the latest attempt 2 status records; it was not rerun.
Final-main Linux artifact 14860918103: SHA-256
`c19a245b3527d26fdaa559cb447092c25e43230331549a6e221185dd542555c9`
(138,714 bytes), independently matched to the provider digest. It records clean
actual head `9cb4cc0250db487d6fd4760227ce906c4ea5f58f`, exit zero and source
digest `d6287cc6a4d99735fb105cc9d8aea880706a8e5171537645a2a7177ed0b59886`,
the same as its tree-equivalent PR #132 checkout. All 3,278 named passes, six
explicit skips and six added regression subtest passes were independently
recounted from the original ZIP; sixteen packages passed and two had no tests.

The six explicit default-suite skips remain skips, not passes:
`TestBackupStagingRefusesForeignOwnedParent`, `TestBackupWorkLeaseChild`,
`TestRestoreSwapCrashChild`, `TestLoadConcurrentTrades`,
`TestLoginTimingStatistical` and `TestQA010CrashChild`. Raw named events are kept
in the artifacts; parent/helper or opt-in coverage is not inferred from the skip count.
The separate native export-reader workflow was inapplicable: none of either PR's
changed paths matched its three exact path filters.

## Independent review and evidence boundaries

Independent number/type investigation, browser/diff review, fresh acceptance and
documentation review, PR-provider review and actual-main verification are
recorded in the archive. An initial malformed CI insertion was independently
blocked, rebuilt with literal callback replacement and verified as exactly two
added lines with all original gates intact. A stale provider merge SHA was
identified as the red checkout; only the fresh tree-equivalent synthetic commit
was used for green qualification. No result is claimed from the malformed or
superseded workflows. Final acceptance and provider reviews approved exact-main qualification with zero blockers. A separately authored read-only validator verified all payload hashes/sizes, manifest inventory, the four original ZIPs and their exact revisions/counts, and absence of compiled executable payloads. Publication readback is recorded separately in the maintained handoff.

The [evidence archive](evidence/UX047-rule-quantities-integration-2026-11-23.tar.gz)
contains provider logs/metadata, verified Linux artifacts, source diff/readback,
reviews and local-check summary. Its `MANIFEST.json` records payload SHA-256 and
size. Archive SHA-256: `1f5cd874c221c638379a5c479e7f5ae3e27e709a25b30a9aaa02e69a594d81ef`; 941,645 bytes,
89 payloads plus manifest. No compiled executables are included.
The preserved evidence branch adds exactly this report and the archive over the
qualified main revision. Publication readback is recorded in the maintained PR
handoff; the evidence-only commit is not presented as a separately executed
product qualification.

These native assistant-worker reviews are not the independent external security
assessment. No manual desktop/mobile screen-reader pass is fabricated. Six
skips, bounded fuzz/load and packaging/reproducibility/sandbox-smoke limits stay
explicit. Neither native cleanup success nor this CI budget adjustment closes
storage read/fsync/truncate or uncertain-commit qualification. Scorecard success
does not approve licence, branch policy or other
findings. SEC-026 visibility/threshold/retention and GOV-021 authority/identity
decisions, translation/locale scope and other owner/manual/external release
gates remain unresolved. All fifteen future modules and held policy PRs
#83/#85/#86/#94/#107 were untouched.

## Fresh continuation

Once the final record and archive are published and read back, ownership of
this bounded slice is released. UX-047 remains the single active partial Core
feature; this run stops at that boundary.
Fetch actual main and newest Core PR heads/checks before editing. Read HANDOFF.md,
docs/TODO.md, specifications, decisions, this final record and the maintained
[PR handoff](https://github.com/djshellshoxxx/circula/pull/131#issuecomment-7109225318).
Continue the single active partial feature UX-047 with a different bounded
current-source investigation and fresh independent workers. Do not repeat #130
setup/cardinal precision, #131 rules-log quantities or #132 native finalization, revive historical missing
flow lists, select held SEC-026/GOV-021 work, or inspect/run future modules.
Write a real regression before any confirmed correction, retain financial and
privacy boundaries, run focused proof and required exact-head CI, publish an
accurate handoff and preserve every branch. Do not deploy, release, choose a
licence, invent owner approvals or use paid coding/API services.
