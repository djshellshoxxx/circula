# Core loading qualification and evidence recovery — 2026-11-20

## Current outcome

The browser loading correction has independent source review, an effective
regression test, and an actual provider run with **597 browser cases passed**.
That source's complete CI still failed its Linux service-package deadline.
A separately reviewed workflow correction is published and awaiting fresh CI.
Accepted main remains unchanged; final Core integration is not complete.

| Role | Exact source |
|---|---|
| Accepted main | `aa01563fcc891da60fefebb8335eca27855678e8` |
| [Core PR127](https://github.com/djshellshoxxx/circula/pull/127), `integration/unique-core-completion-20261120` | `bea5de25cf76b66d87689034e2529f9b40c6dff6` |
| [Repair PR129](https://github.com/djshellshoxxx/circula/pull/129), `repair/core127-e2e-observer-20261120` | `11739c7f8f59455b4dc480294cab4d79f7933cb9` |
| Repair whole tree | `43e65ee91a90ff70b70cc69d0b517b8367a2585f` |
| Browser-qualified parent | `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef` |
| Reviewed local workflow commit | `a4becfda0f10d5fa3790bf0ee4aeff6552dfeca1`; whole tree equals published repair |
| Evidence branch | `audit/core127-final-evidence-20261120`; predecessor `49dc8acc781790635fca8ed32882a7a27971c8a1` |

At **23:39:32 UTC**, [repair CI44838129155](https://github.com/djshellshoxxx/circula/actions/runs/44838129155)
had seven applicable jobs queued and one conditional native-matrix placeholder
skipped. Neither state is a pass. Both PRs remain draft. Repair integration
requires all applicable jobs to pass and refreshed source/target review. The
resulting Core head then requires all nine Core and three native-reader jobs
before main merge. Later outcomes require separate provider receipts.

## Engineering and executed validation

The [loading evidence package](evidence/loading-followup-20261120/README.md)
contains original failures, focused-test logs, provider browser success, exact
source fingerprints, independent reviews and the Linux correction.

Production announces catalogue-backed loading text and `aria-busy` synchronously
during native navigation/submission. Cancellation cleanup, validation, origin and
target boundaries, focus and native form behavior remain. The new regression
checks both states before timers execute. Its negative control substitutes the
exact older production script and fails only the new regression.

The browser harness reads actual DOM/ARIA state within the initiating event,
after the production listener. A unique console marker transports that snapshot
without waiting on a blocked outgoing document's later evaluation. Real held
GET/POST requests, payload, enabled controls, focus, destination and reset
assertions remain. This evidence does not independently measure rendered pixels
or screen-reader announcements.

| Executed check | Source and result |
|---|---|
| Node helpers | `eef0c6d…`: **16 passed, 0 failed, 0 skipped**, exit 0. |
| Old-production negative control | Current helpers plus old `be9a573…` script: **15 passed, 1 expected failure**, exclusively the synchronous-feedback regression. |
| Syntax/whitespace | JavaScript syntax and complete changed-source whitespace passed. |
| Playwright discovery | **597 cases in 20 files**, four loading cases; zero local browser executions. |
| Core CI44815660965 | `bea5de25…`: **8 jobs passed, browser job failed**, 595 cases passed/2 failed. |
| Native readers44815660893 | `bea5de25…`: **3 jobs passed**. |
| Repair browser136617856750 | `eef0c6d…`: **597 passed**, all four loading cases passed. |
| Complete repair CI44831668852 | `eef0c6d…`: **6 jobs passed, 1 Linux job failed, 1 conditional placeholder skipped**. |
| Linux workflow comparison | Parsed YAML differs only in package 20→30 minutes and job 30→45 minutes. |
| Workflow local checks | Whitespace, action pins and hygiene passed; hygiene executed four passing Python tests. |
| New repair CI44838129155 | `11739c7…`: pending at this checkpoint. |

Linux job136617856302 failed with `test timed out after 20m0s`, service elapsed
`1200.102s`. QA001 had run only 23 seconds; current seeds had run 1–3 seconds.
The complete log has no assertion-failure, race or fatal-deadlock marker. A stack
snapshot cannot establish universal absence of deadlocks. Independently verified
prior full-race terminals record service PASS at **1294.982 seconds**, QA001 PASS
at 104.690 seconds and all 50 configured race-mode QA001 seeds passing. No Go or
root module/checksum file changed between that race source and the repair parent.
The new bounded deadlines preserve every test, race flag, seed, fuzz budget,
matrix and evidence step. Published parent, workflow blob and whole tree were
fetched and checked against the independently reviewed source.

Workspace disk exhaustion was resolved by clearing only the explicit task-owned
Go build cache. Sources, module downloads, evidence and worktrees remained intact.
No release, deployment, branch deletion or shared-history rewrite occurred.

## Branch refresh and reconciliation

The [original inventory and dependency graph](BRANCH_REFRESH_20261120.md) remain
frozen. The [late independent refresh](evidence/future-ref-review-late-20261120/README.md)
accounts for **152 nonsymbolic remote branches**: 134 original tips unchanged,
14 advanced, four added and none removed. Every advanced old tip remains an
ancestor. The additions are an evidence branch and three loading-repair branches;
no unrelated feature stream appeared. The workflow repair advances an existing ref.

The coordinator fetched all branches, tags and **129 PR heads**. The
[ref record](evidence/loading-followup-20261120/ref-refresh.json) also lists
**26 local branches**, including isolated worker branches, and zero tags. The
[provider snapshot](evidence/loading-followup-20261120/provider-after-budget-publication.json)
records **15 open PRs** after two proof-based closures.

The [independent local-head inventory](evidence/local-branch-review-20261120/README.md)
examined all 26 local heads against 152 remote tips and 961 remote-reachable
commits. It records exact bases, unique commits, paths, blob identities and
preservation witnesses. No uncovered runtime, test or build change was found.

| Local preservation class | Heads |
|---|---:|
| Exact commit already in remote history | 4 |
| Complete tree published under a different commit identity | 11 |
| Selective changes preserved in remote source/history | 3 |
| Runtime preserved, older documentary variants retained locally | 4 |
| Documentary branches represented in the coordinator's pinned staging tree | 4 |
| **Total** | **26** |

The older documentary variants retain all 22 original source digests and original
qualification notes. Their differences are normalization/explanation, a generated
HEAD stamp, or 61 additional report lines. Staging preservation is explicitly
distinguished from the subsequent evidence-branch publication. Local worker
branches and every original remote source remain preserved.

| PR | Verified reconciliation and preservation |
|---|---|
| [80](https://github.com/djshellshoxxx/circula/pull/80) | Original signed-cursor work is already in feed target `108107435645db687f8ac380e6571319f419fa32`. All eight original paths, three identical test files, complete retained validation body and later hardening were accounted for. Source `0952c94a5dbab4044d7673ce9d51cec9063ab97f` and its branch remain. Closed without a new merge. |
| [87](https://github.com/djshellshoxxx/circula/pull/87) | Dispute-timeline source `b6275747df66ede81ee4267e2cd6a2595f3bd680` is already in accepted main. All 18 original paths, seven identical files, original tests and OpenAPI additions were reconciled. Versioned policy history supersedes ambiguous timestamp reconstruction. Closed without another merge; source branch remains. |

The [Core containment proof](evidence/core-containment-20261120/README.md) covers
16 PRs at its original observation and preserves PR87's later closure separately.
Six heads (#90/#102/#109/#112/#125/#128) are candidate ancestors, conditionally
reconcilable after qualified actual-main integration and refreshed source checks.
All 23 original #128 audit tree entries and 22 manifest checks match. PR111's six
documents are identical; its alternate WAL fixture is explicitly superseded by
the retained qualified setup, not classified as ancestor containment. PR104's
runtime comparison preserves all 38 original Python tests plus 11 more. Its
unique documentary files are recovered separately.

The independent reviewer carried the containment proof forward to `11739c7…`:
only the CI workflow changed, all 16 ancestry classifications remain, and the
seven held sources remain outside the candidate. Live reads independently
confirmed PR87 closed without merging, its source branch unchanged, 15 PRs open,
and all five held policy PRs still open. This overlay is separate from CI success.

Five policy PRs (#83/#85/#86/#94/#107) and **seven policy/interface branches**
remain preserved. Authority, retention, visibility and interface choices remain
unresolved. Historical simulated conflicts remain in the inventory; this
continuation leaves no unresolved merge operation. Mergeability is not validation.

## Verified future-development merges

Across these continuation checkpoints, **eleven verified merges** comprise ten
actual independent future-target merges and one intermediate messaging recovery
merge. The [original eight-merge evidence](QUALIFICATION_CHECKPOINT_20261120.md#eight-verified-merges-in-this-continuation)
and [later three-merge proof](evidence/future-ref-review-late-20261120/README.md)
retain exact source heads, ordered parents, target refs and whole-tree equality.

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
| [#116](https://github.com/djshellshoxxx/circula/pull/116) | `future/fux-105-messaging-20261116` | `54d3af51aa32c022562e8041d1baa149da071db3` |
| [#117](https://github.com/djshellshoxxx/circula/pull/117) | `future/fnet-209-federation-20261116` | `0aeb5a1446798019141a3f6f5bcb1e7a24b70c47` |
| [#118](https://github.com/djshellshoxxx/circula/pull/118) | `future/fnet-207-recurring-20261116` | `89c6aeb6b4f73292d38a4a0803146ff5b9386e6f` |

The last three sources completed **32 successful jobs**; messaging/federation
each also had one conditional Core native placeholder skipped. All 15 actual
future-target manifests remain `enabled:false`, `integration:HOLD`. Core imports
no future implementation and has no changed future path. Source branches remain.
These integrations do not select activation or establish host/device/peer acceptance.

## PR104 documentary recovery and federation evidence

The [recovery package](evidence/pr104-documentary-recovery-20261120/README.md)
preserves **all 102 missing original documentary files**, totaling **5,206,983
bytes**, from source `bcdc569cbc483761bf2d0e2d7594636ac1713d7d`. The deterministic
lossless archive is 1,205,996 bytes, SHA-256
`6d7ad0e06ef2bb9a4cd1a70626ae328a8384828dd439e07c4bf07246ff37d385`.
Every original blob identity and byte hash was checked; the coordinator repeated
the verifier and matched all 102 entries against the original Git source tree.
All 205 nested ZIP payloads and both identical original manifests verify. All
56 original and 44 nested JSON documents parse. Original wording, failures,
patches and dates remain historical evidence, without replacing present documents.

This restores the **ten previously unavailable federation logs and their receipt**.
Independent recount verifies 36 top-level test passes, 105 subtests, three fuzz
targets and 137 fuzz seeds: **281 named passing terminal events**, zero failures
or skips, one package pass and recorded 74.5% coverage. Three separate fuzz logs
each finish 100,000 executions and PASS. All ten original steps record exit 0.
These are recovered historical results, not new product executions.

Accepted federation `0aeb5a1446798019141a3f6f5bcb1e7a24b70c47` matches 19 of 21
historical fingerprints: all 17 Go files, `go.mod` and `go.sum`; workflow and
manifest metadata differ. The historical commit and two declared trees are
unavailable locally and returned supported GitHub 404 responses through the
checked connection. This limits independent source/tree reconstruction without
invalidating preservation of the verified documentary bytes. Current provider
qualification and real peer/TLS acceptance remain separate.

The original decoded payload scan actually returned **exit 1 with 24 detections**
across 307 payloads/8,465,994 bytes. An independent reviewer classified all as
documentary false positives: Git commit IDs, ordinary status/topic phrases and
explicit notification checksum fields. Notification source bytes were unavailable
for recomputation, and that limit remains recorded. No scanner rule, configuration,
exception or original byte was altered. Authored recovery metadata separately
scanned with exit 0 and zero findings. The raw payload scan is not called clean.

This closes PR104's documentary gap once published and verified. Its older Python
alternatives remain excluded under the separate containment proof; no old runtime
or future feature was merged by this recovery. PR104 closure remains conditional
on durable evidence preservation and qualified actual-main integration.

## Earlier complete validation remains source-specific

The [full raw qualification package](evidence/final-qualification-20261120/README.md)
records the earlier whole-source race pass: **1,293 top-level passes** (1,270 Test
functions and 23 fuzz-seed entrypoints), 3,255 passing terminal events including
subtests, zero failures, six explicit skips, 16 packages passed and two helpers
with no tests. It took 3,450.203 seconds without a race report. The earlier normal
run failed one audit-anchor fixture; its corrected fixture passed the full race
run. That original failure remains preserved.

Five fuzz campaigns completed **1,400,000 executions**. Full configured lint
reported zero issues; the actual Go advisory scan returned zero finding events,
and module verification passed. A separate load/correctness probe passed two
tests, completed 100 trades/200 writes, and verified a zero-sum ledger. These are
recorded-source results. New loading/workflow revisions require current provider
gates; old whole-source passes are not relabeled as tests of a newer head.

Manual assistive-technology acceptance, reference-hardware sizing, physical
failure/production restore drills and actual future device/peer integrations
remain separate from automated synthetic tests.

## Remaining engineering by feature and module

The [Core work table](CORE_REMAINING_WORK_20261120.md) and
[15-module work table](FUTURE_MODULE_REMAINING_WORK_20261120.md) retain detailed
scope and exclusions. These unchanged ranges are engineering judgment in focused
engineer-days after the stated decisions, including implementation, regressions
and documentation for an experienced maintainer. They exclude decision/review
waits, CI queues, hardware, devices, services and rollout. Overlapping ranges
are not additive calendar promises.

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

### All fifteen preserved future modules

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

## Next concrete action

Complete repair129's applicable current-head jobs, inspect any failure, and
normally integrate it into127 with expected-head guards. Refresh main and all
new-head Core/native-reader gates before main merge. After verified integration,
recheck and reconcile proven older PRs, with unique documentary work durably
preserved. All policy holds and future streams remain. No release or deployment
is included. Original evidence snapshots remain unchanged; later events require
separate receipts.
