# Branch, provider and remaining-work refresh after PR #119

The bounded enumeration found **137 remote branches and 119 pull requests**:
seven new branches and seven new PRs since the preserved 130-branch snapshot.
There were **35 open, 47 merged and 37 closed-unmerged PRs**. Main remained
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` in the subsequent focus refresh.
No branch, PR, protection setting or deployment was changed by this review.

The new Core contribution requiring investigation was PR #119's session-browser
fixture. Its proposed reload does not address the observed failure. The actual
provider accessibility snapshot and a subsequent HTTP reproduction identify a
same-named error-summary link and checkbox. A narrow keyboard-helper correction
has separate before/after Node evidence, HTTP evidence and independent source
review; a browser rerun remains required. The other six new PRs recover isolated
future modules and preserve their disabled/HOLD boundary.

## Collection and preservation

Branch/PR pages were read at **2026-11-19 13:45:56–13:46:16 UTC**. Their counts
were **100 + 37 branches** and **100 + 19 PRs**, with unique branch names and all
PR numbers 1–119 present. At that observation, all 130 previously observed
branch heads were unchanged; none was deleted. The later movement of newly
created PR #119 is recorded separately below. Focus identities were refreshed
at **14:04:24 UTC** and their checks/statuses at **14:04:32 UTC**. These separate
observations are not presented as one atomic remote snapshot.

The [machine-readable evidence](evidence/branch-refresh-post119-20261119.json)
retains request URLs, observation times, exact branch/PR identities, check/job
states, changed-path blobs, relevant source patches and selected failure
metadata. It also verifies the original eight inventory/report files and all
six previously copied PR #111 documents byte-for-byte. The report was prepared
on isolated branch `docs/provider-refresh-post119-20261119`, after local HTTP
receipt `1cc3e7068ba2f1d43e1d1c026bd591932bee171e`, with Core source base
`789f082a337eac18f9628976846536e6230f5975`.

The unchanged [114-branch inventory](BRANCH_INVENTORY.md),
[120-branch refresh](BRANCH_REFRESH_20261119.md),
[128-branch/109-PR refresh](BRANCH_REFRESH_POST109_20261119.md), and
[130-branch/112-PR refresh](BRANCH_REFRESH_POST112_20261119.md) retain their
original timestamps and qualification boundaries. Their four JSON files are
also unchanged. The six PR #111 documents still match immutable source
`45356ca10e1ecd6e6441e0171910dd918b36f5d9`; their older qualification receipts
do not qualify later UI or combined source.

This follow-up used read-only REST. It did not repeat the prior Git tag
advertisement, fetch all refs, enumerate standalone issues or inspect releases.
The previous zero-tag result remains attached to its earlier observation.

Original base documents remain the
[specification index](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/README.md),
[conventions and invariants](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/00-conventions.md),
[functional requirements](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/04-functional-requirements.md),
[ledger/audit specification](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/06-data-ledger-audit.md),
[security specification](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/09-security-privacy.md),
[quality plan](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/specs/12-quality-test-plan.md),
and [decision log](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/DECISIONS.md).
The original branch/PR ledgers and historical archive remain linked from the
preserved 130-branch report; competing archived text is not promoted into a new
requirement by this refresh.

## Focus heads and actual provider states

All eight focus PRs were open and reported `mergeable: true` with
`mergeable_state: unstable`. All were draft except #110. The exact latest
observed heads, bases and check states are:

| PR | Head | Base | Checks at 14:04:32 UTC |
|---|---|---|---|
| [#90](https://github.com/djshellshoxxx/circula/pull/90) | `839e86935767288e594914ad711486616a604eca` | Main `ead63b5` | 3 success, 1 running, 1 failure, 4 queued |
| [#109](https://github.com/djshellshoxxx/circula/pull/109) | `5db3efc084e6a807bb4702c09d3235490d42b71d` | #90 `839e869` | 2 success, 1 running, 6 queued |
| [#112](https://github.com/djshellshoxxx/circula/pull/112) | `08d4b0078e8c308ae7f353849783913f4302882a` | #109 `5db3efc` | 7 queued, 1 skipped matrix job |
| [#104](https://github.com/djshellshoxxx/circula/pull/104) | `707a4363a86922d152ed2d203c9ea184fe4b323f` | Main `ead63b5` | 12 queued, including the native reader workflow |
| [#110](https://github.com/djshellshoxxx/circula/pull/110) | `aeb70db0255d482eb45aa6e0515f09fc32518d81` | Recurring `73a911a` | 3 success, 4 queued, 1 skipped |
| [#111](https://github.com/djshellshoxxx/circula/pull/111) | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` | #90 `839e869` | 1 success, 8 queued |
| [#108](https://github.com/djshellshoxxx/circula/pull/108) | `38b9bcd0e049d97167d793baba933d5e79d8030d` | Analytics `0b29651` | 1 success, 2 running, 7 queued, 1 skipped |
| [#119](https://github.com/djshellshoxxx/circula/pull/119) | `153cec2e0645c9c51be1c7da0668c5d2bb503321` | #99 `0a73ee4` | 7 queued, 1 skipped |

#90's successful jobs are native Windows tests, API contract/load smoke and
release-target builds; macOS is running. #109 has successful Windows tests and
release-target builds, with macOS running. #111 has passed contract/load. #108's
success and two running jobs belong to the isolated analytics workflow. These
results do not qualify the later local combination or another PR's head.

Main's check endpoint returns 22 records across multiple runs, including older
checks. The current CI run
[44515980005](https://github.com/djshellshoxxx/circula/actions/runs/44515980005)
has **eight successes and the browser job running**. Its 22-record aggregate
contains 20 successes, one running job and one older failure: the failed fuzz
job from run 44296161788 on November 18. That older failure is not a new failure
of the current run. All queried combined-status endpoints return `pending`
with zero status contexts; check-run records provide the meaningful job state.

#112's current run was created before its cross-platform label was present.
The label is now present, but that does not retroactively execute the skipped
matrix. Qualification must observe the intended matrix on the appropriate new
source event. No skipped or queued job is treated as a pass.

### Why jobs remained queued

The early job records for #109, #111 and #112 have `runner_id: 0`, an empty
runner name and no executed steps. The first two queried runs have no pending
deployment approvals. The inspected CI source uses a concurrency group scoped
to the PR/event, rather than one group shared across all these PRs. It has no
job `needs` or `environment` gate explaining these queued jobs.

Observed hosted-runner allocation is delayed: #90's run was created at 12:21:47,
its Windows job started at 13:38:05, and its quality job at 13:42:27. Subsequent
successful jobs show that the queue is moving. At the earlier repository-wide
observation there were 29 queued workflows and two in-progress workflows;
individual job records can be ahead of a workflow-level status update.

These observations support **runner-allocation delay**, but they do not expose
the account-level cause. No billing, quota, outage or approval explanation is
asserted without evidence. There is no authorized reason to bypass the gates.

### Actual failure and merge-policy visibility

#90's [quality job 135556801602](https://github.com/djshellshoxxx/circula/actions/runs/44523175368/job/135556801602)
failed in Gitleaks with 15 `generic-api-key` findings in existing audit hashes
and UI prose. The raw metadata preserves rule, file and line locations without
duplicating finding values. This is a real failed provider job on `839e869`;
later lint/vulnerability steps in that job did not run. It is not a failure
result for #112's queued source.

The independently owned repair at
`dd3357d3cb155bbe29de47d1bb788bff38d7d35f` uses exact rule/path/value allowlists
and real-scanner positive controls while preserving original inventory bytes.
Its local qualification does not retroactively change the old provider result.
The later combined candidate still needs its own scanner and provider results.

Legacy main protection returned **403, resource not accessible by integration**.
The accessible ruleset collection was empty; the effective branch-rule endpoint
was rejected by the connector's endpoint allowlist. Therefore the actual
required-check, required-review and merge-queue configuration remains unknown.
The focus review records include one `COMMENTED` review on #90 and no
`APPROVED` review. A mergeable response does not establish merge eligibility.

The normal proposed integration path remains #112 into #109, #109 into #90,
then #90 into main, with expected source identities and applicable gates
verified at each stage. This report performs none of those operations.

## New branches and contribution disposition

| New PR | Initial head | Scope and disposition |
|---|---|---|
| [#113](https://github.com/djshellshoxxx/circula/pull/113) | `3cc55f7966a6321c9fa4e587596ea7daf0f9d152` | Isolated receipts recovery; 63 changed paths |
| [#114](https://github.com/djshellshoxxx/circula/pull/114) | `fbd92a22f06af7bcd96957397958b33e2706807a` | Isolated advisory-governance recovery; 73 paths |
| [#115](https://github.com/djshellshoxxx/circula/pull/115) | `684ca5132690e24104aa389aac16870c18bbc7fd` | Isolated named-areas recovery; 104 paths, collected as 100 + 4 |
| [#116](https://github.com/djshellshoxxx/circula/pull/116) | `62623635d9711fbafc9b9660ac8860f1eb916dbb` | Isolated messaging recovery; 90 paths |
| [#117](https://github.com/djshellshoxxx/circula/pull/117) | `f50aaa4b599c2c0272d37cdc910f567b76c451f8` | Isolated federation recovery; 78 paths |
| [#118](https://github.com/djshellshoxxx/circula/pull/118) | `035dd1c0d7c3de901cb1d2e1854ac4fde4cbdad4` | Isolated recurring recovery; 88 paths |
| [#119](https://github.com/djshellshoxxx/circula/pull/119) | `4f42cbc57d146a765e5867434cab3680b4898209` | Four Core fixture changes; later moves to `153cec2` with three audit-document representation changes |

All seven were newly opened drafts. The six future-module feature manifests
retain `enabled: false`, `integration: HOLD`, and empty future-dependency lists.
They do not introduce a new module identity or authorize Core activation.
Their implementation percentages and local module receipts are not Core release
acceptance, and this inventory does not rerun or certify the modules.

The only Core source in #113–#117 is the accepted baseline restoration of
`cmd/circula/auditanchor.go` and
`internal/service/departure_pagination_regression_test.go`. Their blobs are
respectively `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` and
`3b117ecfe2f01e7e2363b9d201503cde6e0f5595`, already byte-identical in frozen
Core `e0d27cc`. #118 has no new Core source; its non-module path is its dedicated
workflow. These contributions need no duplicate Core source adoption.

#104's manifest-parity source has already been recovered additively in the
local report base, preserving the existing decoder-completion guard and all
three compression regressions. Its 41-test local Python pass remains bounded
to that combined reader. The dedicated native reader jobs above remain queued.
The [separate reconciliation evidence](evidence/export-reader-reconciliation-20261119/)
retains exact sources, fail-before controls and the ten peer evidence documents.

### PR #119: each changed contribution

PR #119 moved from `4f42cbc` to `153cec2` after the initial enumeration. Its
old checks were cancelled at 13:49:47; they are not failures or successes for
the newer head. The later file list contains seven paths:

| Contribution | Disposition against the current Core candidate |
|---|---|
| Migration rollback fixture | Preserve the current fixture, which fails after the current head migration's SQL but before its version transaction commits, verifies rollback and reopens. The peer pins the older multi-object migration 0015; no wholesale replacement is needed. |
| Support-boundary template loader | Preserve the current production `messages.Text` and `messages.Around` registration. The peer supplies a function returning the key itself, which does not exercise actual catalogue rendering. |
| WAL witness | Preserve the focused-qualified pinned query-only connection. The peer's alternate open read transaction recovers the same last-connection observation behavior and is not missing production source. |
| Browser session reload | Not adopted. Actual failure and HTTP state contradict the asserted CSRF-rotation cause; the later missing-confirmation response still creates the same-named link/checkbox collision. |
| Three audit representation rewrites | Not adopted. Original inventory and execution records remain byte-for-byte preserved; the independently qualified scanner configuration addresses the verified false positives. |

The browser failure was **587 passed, one failed** in
[job 135522757115](https://github.com/djshellshoxxx/circula/actions/runs/44513300863/job/135522757115)
on PR #99 source `0a73ee42b96255601cf2227b1ddf6599ec07a0ef`. Its downloaded
artifact, exact accessibility snapshot and selected failure log are retained in
[session keyboard qualification](../qa/session-keyboard-confirmation-qualification.md).
The correction constrains boolean-field matches to the focused native checkbox
while preserving Tab traversal, Space toggle and Enter submission. The test-only
baseline has three failures and three passing controls; the corrected combined
Node suite has 13 passes. A separate real HTTP probe against clean `e0d27cc`
passes in 1.721 seconds and confirms unchanged CSRF/session state, the collision,
and successful original-token confirmation. Independent source review found no
issues. **No browser after-pass is claimed.**

## Remaining work: current ownership and genuine gaps

The current [Core UI checkpoint](CORE_UI_COMPLETION_20261119.md) records 1,431
English messages. The older 525-key text in the preserved peer blocker document
and the explicitly historical initial-recovery portion of
[REMAINING_WORK.md](REMAINING_WORK.md) must not reopen completed template work.
The checkpoint's successful-COMMIT/failed-acknowledgment item has since received
[actual test-only qualification](evidence/commit-acknowledgment-loss.md) and
[independent review](evidence/commit-acknowledgment-loss-independent-review.md).
It is no longer an unowned implementation gap. Python parity, scanner repair
and the confirmed keyboard fixture also have assigned owners and source-bound
receipts.

The following work remains actionable within ordinary Core engineering:

1. Complete qualification of the actual combined candidate: preserve full
   failure diagnostics and resolve the aggregate race outcome, then require
   applicable Go/static/vulnerability, bounded-fuzz, contract/load, browser,
   native-target and reproducibility results. These are already coordinated
   workstreams; another cold SQLite build would duplicate them. A failed or
   interrupted aggregate command is not a pass because other packages passed.
2. Run the corrected session fixture in a supported browser/provider. The
   local Node adapter and HTTP reproduction address selection and server state;
   they do not replace browser focus/navigation or the complete browser suite.
3. Extend real storage-failure qualification where a suitable bounded physical
   fault harness exists: read, fsync, truncate, checkpoint sync/truncate and
   other device failures. Existing disk-full/WAL/checkpoint-write and new commit
   acknowledgment-loss tests do not establish those physical guarantees.
4. Inspect and close meaningful acceptance gaps identified in
   [BUILT-REQUIREMENT-EVIDENCE.md](../audit/consolidation-2026-11-17/source-reviews/BUILT-REQUIREMENT-EVIDENCE.md):
   canonical production UUIDv7 output and injected-generator boundaries;
   balance-help semantics; semantic deferred-route exclusion; text-only dispute
   evidence; pointer cancellation and reduced-motion behavior; and one packaged
   artifact's single-/multi-network mode parity. These are qualification
   opportunities, not newly demonstrated runtime defects. Current tests must be
   inspected before adding regressions, and tests must prove a real boundary
   rather than merely repeat implementation.
5. Keep source/PR disposition current at integration, satisfy actual protection
   and review gates, verify resulting main, and reassess remaining Core work.

The bounded UI audit found no additional confirmed unowned product/UI slice
beyond the keyboard fixture. This does not establish all of UX-047: complete
locale/date/number behavior and applicable full-browser qualification remain
distinct, while additional translation languages are a product decision.

The recorded owner/external holds also remain: reviewer-recovery independence;
dual-control appointment and reset policy; GOV-021 temporary/operator authority
and its independent migration; SEC-026 visibility/disable/retention policy;
privacy retention and post-erasure WAL treatment; approved numerical release
budgets/reference hardware; manual screen-reader acceptance; SEC-033 independent
professional review; SEC-031 approved disclosure/contact arrangements; licence,
jurisdiction and support decisions; and explicit future-module adoption.
Automated or AI-assisted engineering checks cannot supply those decisions.
Deployment and release publication remain outside this continuation's scope.

This report and its JSON are additive evidence. They change no Core source,
policy, original inventory bytes or remote state, and claim no full-suite,
browser, merge or release result beyond the separately identified observations.
