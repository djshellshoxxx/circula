# Circula consolidation qualification checkpoint — 2026-11-20

The published Core candidate has completed the full local race suite, five bounded fuzz campaigns, full configured lint, a vulnerability scan against the published Go advisory database, module verification and a separate load/correctness probe. Verified independent merges and pending exact-head provider gates are distinguished below. Main remains unchanged; this checkpoint does not declare project completion or production qualification.

## Source and observation boundaries

| Role | Exact identity |
|---|---|
| Accepted `main` | `aa01563fcc891da60fefebb8335eca27855678e8` |
| Published [Core PR #127](https://github.com/djshellshoxxx/circula/pull/127) | `integration/unique-core-completion-20261120` at `be9a5731bc93dcbf7b57301a8d2538a13af360bd` |
| Published complete tree | `7ba56914025bccada81f90a326e05233c45189a7` |
| Complete race execution source | `36112a8614f4719415ec2e208521aef28ff3e69c` |
| Final static, fuzz and load checkout | `69ca540dd9a1c01471de613e476b76babb87b31a`; complete tree equals published tree |
| Original complete normal execution source | `a9cdcb80179df1469b1443dcb79f1df00b1857fb` |

Only documentation/evidence differs between the race source and published candidate; final composition review established unchanged executable source. Static, fuzz and load qualification used the exact published whole tree. The [source-equivalence receipt](evidence/final-qualification-20261120/source-equivalence.json) and [raw-output index](evidence/final-qualification-20261120/README.md) preserve these distinctions.

Prior [PR #99](https://github.com/djshellshoxxx/circula/pull/99) established accepted main after nine successful jobs; the separately inspected [later main run](https://github.com/djshellshoxxx/circula/actions/runs/44728309016) also passed all nine. These historical results do not qualify PR #127. The original provider observation is **18:14:31 UTC**; verified merges were first refreshed at **18:19:52 UTC**, including onboarding's intervening merge. Ref preservation was recorded at **18:23:46 UTC**. Later observations are identified separately.

## Branch inventory, dependencies and preservation

The [148-head inventory](BRANCH_REFRESH_20261120.md) covers 128 PRs and 752 recorded workflow runs. Its [machine-readable ledger](evidence/branch-refresh-20261120.json) contains exact bases, unique commits, changed paths, patch equivalence, merge simulations, conflicts, manifests and input hashes. Its [dependency graph](BRANCH_REFRESH_20261120.md#dependency-graph) preserves 4,637 ancestor/descendant relations through 206 reduced edges, with separate PR-target and declared-feature dependency layers.

The inventory is pinned to candidate `025ad9d3b23abd29c99dbe3ea376ff1622844313`. Its ref, PR and workflow collections were separate observations with explicit overlays. It does not claim all 148 heads were retested at `be9`. Against its pinned candidate, 102 merge simulations were clean and 46 had textual conflicts; 71 clean results were exact no-ops. These measurements do not establish readiness.

| Recorded disposition | Heads |
|---|---:|
| Independent future module, HOLD | 44 |
| Policy/interface decision, HOLD | 7 |
| Historically recovered or superseded | 23 |
| History already contained in the pinned candidate | 67 |
| Audit record preserved for reconciliation | 1 |
| Patch-equivalent historical implementation | 1 |
| Historical exact no-op | 3 |
| Core peer recovered with documented/test alternative | 2 |

The [preservation snapshot](evidence/final-qualification-20261120/ref-preservation-snapshot.json) records 148 remote-tracking branches, 18 local branches and zero tags; symbolic `origin/HEAD` is separate. All 15 actual future targets retain disabled/HOLD manifests and empty declared cross-feature dependencies. Core imports no future-module implementation. Original branch names and recoverable commits remain preserved; no remote branch was deleted or shared history rewritten.

## Core engineering recovered and repaired

The [accepted completion map](evidence/unique-core-20261119/accepted-core-completion-map.md), [recovery history](UNIQUE_CORE_CONTINUATION_20261119.md) and [continuation receipt](CORE_QUALIFICATION_CONTINUATION_20261120.md) retain original source references and selective-integration decisions. PRs #125, #112, #109 and advanced #102 are ancestors. Selected repairs were integrated as commits, not additional whole-branch merges. The test-only anchor repair originated at `60e98b8229a211fcf24c762c3c1ba960bb21879c`; shared password admission was integrated at `025ad9d3b23abd29c99dbe3ea376ff1622844313`.

Recovered Core includes pre-commit migration invariant refusal/rollback; posting-time historical credit evidence; transaction reports with exact dispute context, subject confidentiality and independent-reviewer projections; conditional listing revisions; retained-membership recovery safeguards; bounded authorization ancestry; versioned closure checksums and platform audit anchors; bounded best-effort audit mirroring; posting outcome metrics; and shared interface/message/loading improvements. Database/API forward migrations, compatibility rules and regressions are recorded in the linked histories.

| Repair area | Result and preserved constraint |
|---|---|
| Password work, AUTH-04 | Shared two-operation admission covers authentication, recovery/MFA, startup and passphrase backups. Busy requests refuse before credential repository/guess-reservation work; credential formats and historical PHC compatibility remain. This is an operation-count bound, not an RSS ceiling. |
| Native migration/CI portability | Scoped LF checkout rules preserve migration 0023/0025 bytes and reviewed digests. macOS receives the observed necessary test budget; selection, race instrumentation and assertions remain. |
| Report/API correctness | Redirects use authorized case IDs and escaped rooted paths. Historical credit decoding consistently rejects explicit null aliases while preserving omitted defaults and valid numeric aliases. |
| Browser coverage | Loading tests retain outgoing-document handles without waiting on their own blocked navigation. Actual `dispute_reports` light/dark accessibility coverage was added; no checks were excluded. |
| Audit-anchor fixture | Replacement guarantees a changed digest byte and exactly one affected row. Clean, growth, rewrite-detection and truncation assertions remain. |
| Evidence scanning | Synthetic documentation/hash metadata was clarified without changing source digests or weakening scanner rules. |

The [security reconciliation](SECURITY_HANDOFF_RECONCILIATION_20261120.md) records implementations for 11 of the original 14 source findings. OPS-04 reader resource use, OPS-03 Core Windows private files and LED-R02 disclosure policy remain explicit. This is source-finding disposition, not universal security closure.

Duplicate alternatives were handled narrowly: historical #104 verifier runtime is already represented while newer Python compatibility protections remain; #111's alternate WAL fixture is superseded by the accepted repair. Original documentation-only #128 at `896d69f472652ed54c051ec82efcb7f162d1d2c4` remains intact. These decisions neither discard unrelated unique work nor decide the seven held policy/interface branches.

## Eight verified merges in this continuation

Seven merges reached actual independent future targets; #122 reached an intermediate messaging recovery branch. The original [seven-merge receipt](evidence/final-qualification-20261120/verified-merges.json) preserves its 18:19:52 observation. The later [analytics verification](evidence/final-qualification-20261120/analytics-merge-verification.json) adds the eighth without rewriting that snapshot. Receipts record source branches/SHAs, targets, both parents, exact trees and times.

| PR | Target branch | Verified merge commit |
|---|---|---|
| [#103](https://github.com/djshellshoxxx/circula/pull/103) | `future/fux-104-notifications-20261116` | `54a41b12a314e7652802b71d693e2594c37f172f` |
| [#113](https://github.com/djshellshoxxx/circula/pull/113) | `future/fnet-203-receipts-20261116` | `5200c5bf940f051a86d6e502787af06892200974` |
| [#114](https://github.com/djshellshoxxx/circula/pull/114) | `future/fnet-208-governance-20261116` | `5eac085005361e973ca8f358e01f039aea718bed` |
| [#115](https://github.com/djshellshoxxx/circula/pull/115) | `future/fnet-206-areas-20261116` | `6aac35e226135a53d69c1e6ce74e340dcfe1d85f` |
| [#120](https://github.com/djshellshoxxx/circula/pull/120) | `future/fux-102-qr-20261116` | `6028dae56723b8488db6e52e25c83da7888777c1` |
| [#121](https://github.com/djshellshoxxx/circula/pull/121) | `future/fux-106-onboarding-20261116` | `4c3ce1d7b7f2b4ed809669985d5e38981408494b` |
| [#122](https://github.com/djshellshoxxx/circula/pull/122) | `master/recover-messaging-20261119` | `d232d6507cb198081434284084a4ced7ed6df27c` |
| [#108](https://github.com/djshellshoxxx/circula/pull/108) | `future/fnet-205-analytics-20261116` | `64f1fef75d167ca322179e3e120214f9a56c6583` |

Onboarding merged at **18:15:49 UTC**, after the provider snapshot recorded it open. Analytics merged at **18:30:20 UTC**, from `master/recover-analytics-20261119` at `0b67eb44811165e947d8db5ba25a3d868af892ff`; its [final individual gates](evidence/final-qualification-20261120/analytics-final-gates.json) show seven applicable Core and three native module successes. Each retained the skipped conditional Core native-matrix placeholder.

Earlier #113/#114/#115/#120 also passed seven applicable Core and three dedicated jobs while preserving that skip. QR's native jobs exercise camera-lifecycle Node tests, not physical cameras. Notifications passed all nine Core jobs, its SQLite job and three native transport jobs. No merge activates a module or deploys anything.

## Pending exact-head provider gates

The frozen [18:14:31 provider snapshot](evidence/final-qualification-20261120/provider-snapshot.json) preserves individual job IDs, heads and conclusions. Its onboarding/analytics open states are superseded only by the later receipts above. The four remaining candidates are open/draft and textually mergeable in that snapshot.

| PR and source → target | Exact published head | Jobs at 18:14:31 UTC |
|---|---|---|
| [#127](https://github.com/djshellshoxxx/circula/pull/127): `integration/unique-core-completion-20261120` → `main` | `be9a5731bc93dcbf7b57301a8d2538a13af360bd` | Nine Core and three native Python-reader jobs queued. |
| [#116](https://github.com/djshellshoxxx/circula/pull/116): `master/recover-messaging-20261119` → `future/fux-105-messaging-20261116` | `d232d6507cb198081434284084a4ced7ed6df27c` | Seven Core and three native module jobs queued; one Core placeholder skipped. |
| [#117](https://github.com/djshellshoxxx/circula/pull/117): `master/recover-federation-20261119` → `future/fnet-209-federation-20261116` | `d00b5ca897ad7b6fbebed6572a9270845ef9c2c0` | Seven Core and three native module jobs queued; one Core placeholder skipped. |
| [#118](https://github.com/djshellshoxxx/circula/pull/118): `master/recover-recurring-20261119` → `future/fnet-207-recurring-20261116` | `18741aa25c364e7294de8bec0ff714d0dc1ed9fc` | Nine Core jobs queued; native module: one running, two queued. |

Core [run 44784049761](https://github.com/djshellshoxxx/circula/actions/runs/44784049761) covers native tests, browser/accessibility/keyboard, API/load, fuzz, lint/vulnerability, builds and reproducibility; [run 44784050251](https://github.com/djshellshoxxx/circula/actions/runs/44784050251) covers Python readers. Queued/running/skipped jobs are not passes. Future candidates require all applicable shared jobs and dedicated module jobs; final #127 requires full native Core qualification.

Recurring `18741aa…` and federation `d00b5ca…` received source/provenance review against their actual future targets. Federation preserves #105/#106 histories through normal composition and reconciles its README with recovered standalone behavior. The inventory's 46 historical simulation conflicts remain documented; no unresolved textual conflict appears in this remaining-PR snapshot.

## Executed validation

| Check | Executed result |
|---|---|
| [Complete race suite](evidence/final-qualification-20261120/full-race-summary.json) | `go test -race -p 1 -count=1 -timeout=45m -json ./...`, exit 0, 3,450.203 seconds. **1,293 top-level passes: 1,270 Test functions plus 23 fuzz-seed entrypoints; 3,255 passing terminal events including subtests; zero failures; six skips.** Sixteen packages passed; two helpers contain no tests. No race report. |
| [Original complete normal suite](evidence/final-qualification-20261120/full-normal-summary.json) | Exit 1: **1,292 top-level passes, one fixture failure, six skips**; 3,456 passing events including subtests. `TestAuditAnchorWriteVerifyAndTamper` could perform a no-op replacement. The corrected test passed in the complete race suite. No later complete normal pass is claimed. |
| [Five current fuzz campaigns](evidence/final-qualification-20261120/critical-fuzz/campaign-summary.json) | `FuzzParseAmount`: 1,000,000 executions; `FuzzStrictUnmarshal`, `FuzzOpenCursor`, `FuzzSessionPosition`, `FuzzSessionSelector`: 100,000 each. **1,400,000 executions total; all five exited 0.** Count-bounded single-worker campaigns retain assertions. |
| [Full static/dependency qualification](evidence/final-qualification-20261120/core-static-summary.json) | Full configured lint: exit 0, twelve active linters, **zero reported issues**. Actual `https://vuln.go.dev` source/symbol scan: exit 0, **zero findings**, database last modified `2026-11-11T02:51:11Z`. `go mod verify`: exit 0, all modules verified. |
| [Load and QA008 correctness probe](evidence/final-qualification-20261120/load-smoke-summary.json) | **Two tests passed, zero failed/skipped**. Four workers, twenty members, twenty-five trades per worker produced 100 completed trades, zero credit refusals and 200 committed writes; ledger verification passed with sum zero. QA008 separately exercised 100 disjoint trades. This is not production capacity qualification. |

The six race skips were the foreign-owned-parent fixture requiring Unix root; `TestBackupWorkLeaseChild`, `TestRestoreSwapCrashChild` and `TestQA010CrashChild` subprocess entrypoints; opt-in `TestLoginTimingStatistical`; and opt-in `TestLoadConcurrentTrades`. The latter subsequently ran in the separate load probe. `internal/storetest` and `internal/testutil` are no-test packages, not additional skipped tests.

The two full runs contain the same top-level test names. The race run exercises 202 fewer nested cases because of unchanged existing race budgets: 200 fewer QA001 sequences and two fewer ledger-property seeds. Those source files were not altered; the repaired anchor fixture changes from failure to pass. This explains the different event totals without treating reduced nested counts as missing top-level coverage.

The suites exercise implemented API, database, migration/invariant, transaction, recovery and interface contracts. They do not establish physical-failure or production restore acceptance. Existing lint exclusions were unchanged. The earlier full-lint attempt killed with exit 137 remains a failed/incomplete attempt; the later complete success is separately recorded.

Local Chromium originally aborted before page creation with `socket() failed: Operation not permitted (1)`; no product browser assertion executed. Final provider browser/native gates remain mandatory. Federation review verified five bundled historical raw-log hashes, but ten combined-run logs referenced by its older receipt are absent from this workspace and bundle. Their historical 36-test/74.5%-coverage/three-fuzz results were not independently recounted. Current-head provider evidence, real Core adapters, independent peer/TLS interoperability and operational/key-custody qualification remain distinct requirements.

## Remaining Core work and conditional effort

These are focused engineer-days for an experienced maintainer after the specified decisions, including code, regressions and documentation. They exclude decision/review waits, CI queues, hardware, devices, external services and rollout. Overlapping ranges cannot be added into a calendar promise. The [detailed Core table](CORE_REMAINING_WORK_20261120.md) supplies exact constraints; its previously pending full-race item is updated by the successful run above.

| Core item | Remaining engineering | Days | Separate prerequisite |
|---|---|---:|---|
| OPS-04 | Bounded/streaming Go/Python export readers and aggregate/parser accounting. | 7–15 | Supported budgets, overrides and large-export compatibility. |
| OPS-03 | Core Windows staging/output ACL, ownership and reparse behavior. | 5–10 | Service identities/support policy and native acceptance; messaging's repair is separate. |
| LED-R02 | Actor-aware service/JSON/HTML refusal detail. | 2–4 | Resolve contradictory availability-disclosure requirements. |
| GOV-021 | Reconcile retained administrative-recovery implementations and authority/session boundaries. | 5–10 | Recovery authority, independence, waiting and audit policy. |
| AUD4-F02 | Reviewer-reset contract and credential/MFA regressions. | 3–6 | Reset actors and independent authority. |
| SEC-026 | Abuse-signal visibility, retention and export/erasure lifecycle. | 5–10 | Data and access policy. |
| Report escalation/subjects | System-account subjects and all-administrators-conflicted handling. | 3–6 | Escalation actor and evidence visibility. |
| Competing CLI contract | Contract/document reconciliation; implementation remains unscoped. | 2–4 | Select intended interface. |
| DEC-006 | Reference-hardware password-work measurements and operating documentation. | 1–3 | Hardware/headroom; smoke benchmarks are insufficient. |
| Final qualification | Finish exact-head provider gates; address observed failures. | No new implementation estimate | Queue/runtime dependent; assistive-technology and operational drills remain separate. |

## All fifteen independent future streams

The [source-pinned module table](FUTURE_MODULE_REMAINING_WORK_20261120.md) records commits, manifest blobs, recovered functionality and exclusions. Every module remains disabled/HOLD with no Core imports. Estimates credit existing implementations and exclude hypothetical activation; the same effort assumptions apply. Completed recurring composition and federation composition/README reconciliation are no longer residual tasks.

| Module | Remaining engineering | Days | Separate decision/qualification |
|---|---|---:|---|
| FNET-201 Matching | Persistent preferences/dismissals, authorized snapshots, term ranking/caps, rescoring, explanation/no-JS and Core adapter. | 15–25 | Ranking/retention; optional cycles excluded. |
| FNET-202 Favorites | Core policy/trade/session/departure/proposal-confirmation adapters and documentation. | 5–10 | Browser, restore and host qualification; persistence/interfaces exist. |
| FNET-203 Receipts | Authenticated immutable reads, protected signing adapters and interoperability fixtures. | 7–15 | Custody/trust and external verifier acceptance; ephemeral artifacts remain intentional. |
| FNET-204 Feed | Authoritative projection/permissions/replay, preferences, keys and host interfaces. | 10–20 | Retention, mute/follow and cursor lifecycle. |
| FNET-205 Analytics | Durable immutable releases, query budgets, chosen correction rules and historical Core snapshots. | 10–20 | Privacy decisions and production budgets. |
| FNET-206 Areas | Principal/MFA/listing/store/CSRF adapters and retention/backup integration. | 5–10 | Revocation/restore acceptance; geometry/maps excluded. |
| FNET-207 Recurring | Core transaction/policy/inbox adapters and durable fair scheduler continuation. | 7–15 | Archival/erasure policy and operational/browser qualification. |
| FNET-208 Governance | Core authority/MFA/store/policy-history and retention/backup adapters. | 7–15 | Advisory-only policy and accessibility/rollout selection. |
| FNET-209 Federation | Membership/source/departure/delete adapters; selected capacity/epoch/compaction and signing-material management. | 10–20 | Protocol/peer/custody/rate decisions and real TLS interoperability; settlement excluded. |
| FUX-101 PWA | Real routing/templates/CSP/session/CSRF and disable/decommission integration. | 3–7 | Mobile installation, browser/BFCache and assistive-technology acceptance. |
| FUX-102 QR | Same-transaction membership/limits/proposal adapters, opt-in routes, alias custody/restore and documentation. | 5–10 | Physical camera/mobile/CSP qualification; renderer/scanner/storage/CLI exist. |
| FUX-103 Discovery | Core listing/membership sync, authenticated host, production blobs/retention/driver and documentation. | 7–15 | Media/retention policy, sanitizer and production-size acceptance. |
| FUX-104 Notifications | Core transaction/auth/worker lifecycle, bounce handling, operator CLI, accessible preferences and restart/restore integration. | 10–20 | Live SMTP/Web Push/custody acceptance; transports exist. |
| FUX-105 Messaging | Core auth/CSRF/reference/retention/audit, schema/routes and defined history/contention/cancellation workloads. | 7–15 | Unmeasured qualification gap, not a demonstrated production contention defect. |
| FUX-106 Onboarding | Current-auth/terms/member routes, existing separate practice-origin/cookie configuration, retention operations and documentation. | 5–10 | Curriculum/isolation/browser acceptance; checklist and practice server exist. |

## Publication checkpoint

The publication branch for this report and packaged evidence is `audit/core127-final-evidence-20261120`, based on published `be9a573…`. This evidence-only branch preserves the Core qualification head and executable source. Original outputs remain source-bound and losslessly preserved; the package checksum manifest covers the completed reports and artifacts. The publication commit is recorded in PR #127 separately so publishing evidence does not restart its source qualification.

Next actions are to refresh exact source/target heads and individual jobs, repair observed failures, and merge qualified compatible candidates. Evidence publication is separate from changing the qualified Core candidate. No release or deployment is included. Policy choices, missing host adapters and real-environment acceptance remain documented rather than silently supplied by consolidation.
