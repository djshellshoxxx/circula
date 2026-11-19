# Consolidation execution report

> Closing publication checkpoint: the addendum below records the published Core992 source and isolated follow-ups. The [combined core review](CORE_COMPOSITION_20261119.md) and [frozen 123-ref snapshot](CONCURRENT_BRANCH_SNAPSHOT_20261119.md#bounded-refresh--112708-utc) retain their original source and observation time. Current PR heads and checks govern later qualification.

Audit date: **2026-11-18 Pacific / 2026-11-19 UTC**. This report records observed implementation, publication and qualification. Its current checkpoint is still pending combined provider gates; it is not a declaration that all work is merged or release-ready. The PR receipts below are authoritative for later state changes.

## Closing publication checkpoint (2026-11-19)

The reviewed Core composition was published in [PR #90](https://github.com/djshellshoxxx/circula/pull/90) as
[`992478594e5902f57a39f3772739bee6ab495d24`](https://github.com/djshellshoxxx/circula/commit/992478594e5902f57a39f3772739bee6ab495d24),
tree `3725d720ce7fc41f148b560e5f06c8f9bf5ab221`. Independent comparison of
all 1,470 leaves matched the reviewed local tree, including every mode and
blob SHA. Its preserved source parents are recorded in the commit and the
[composition checkpoint](CORE_COMPOSITION_20261119.md). Main remains
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`; **zero main merges** occurred.

The following are published-source receipts, not claims that their new CI
has passed. The two earlier qualified topic merges, #91 and #93, remain
recorded separately below.

| Open coordinator PR | Published head at closing checkpoint | Target / boundary |
|---|---|---|
| [#90 core](https://github.com/djshellshoxxx/circula/pull/90) | `992478594e5902f57a39f3772739bee6ab495d24` | `main`; draft, final provider/review gate pending |
| [#92 feed](https://github.com/djshellshoxxx/circula/pull/92) | `db80ed34af09ad6b28e533da6d4b82b8250f3711` | `future/fnet-204-feed-20261116`; draft, disabled/HOLD |
| [#94 signals](https://github.com/djshellshoxxx/circula/pull/94) | `45b596f12876e3f91c30b996457f7096e1fffe24` | Core integration branch; draft, default off and policy-held |
| [#95 PWA](https://github.com/djshellshoxxx/circula/pull/95) | `0b0d70624591941c0a71a225b7390903dc4ab97c` | `future/fux-101-pwa-20261116`; draft, disabled/HOLD |
| [#100 notifications](https://github.com/djshellshoxxx/circula/pull/100) | `98852afbd1cd2e31a38d6405b6eee36ad3d7e737` | `future/fux-104-notifications-20261116`; draft, disabled/HOLD |
| [#106 federation documentation](https://github.com/djshellshoxxx/circula/pull/106) | `d55554408ef591371768259905c70ca58d68facb` | `master/fnet209-utf8-20261119`; draft, README-only correction |
| [#107 GOV-021 refresh](https://github.com/djshellshoxxx/circula/pull/107) | `b8cb63bfc6d2a969d972f55937cdc2070d5eba34` | `claude/gov-021-admin-recovery`; draft and policy-held |

The closing bounded `git ls-remote --heads` read observed **125 live remote
branch refs at 12:10:21.881995 UTC**. This confirms the earlier observation of
125 distinct names: the frozen 123-ref inventory at 11:27:08 UTC plus the
two specifically verified #106/#107 additions. Both observations retain
their timestamps; neither is an assurance about later concurrent writes.
All 107 original names remain: 105 original tips are unchanged, while the
GOV-021 and notification topic tips advanced with their original commits
still ancestral, confirmed by native Git ancestry checks. The closing
open-PR collection returned 23 records. There are **nine coordinator-created repair/integration
PRs: #90–95, #100, #106 and #107**. These are separate from concurrent source
PRs and from the number of defects repaired.

There are **two qualified original-topic merges**, #91 and #93, and **two
integration-only inclusions**: #101 closed as merged into the Core integration
branch at `99247859` (11:39:56 UTC), and #98 closed as merged into the feed
integration branch at `db80ed34` (11:57:10 UTC). Neither inclusion is a main
merge or a new qualification result. All original histories remain preserved.

The PWA, feed and notification publications were independently checked against
their local reviewed trees: 1,517, 1,505 and 1,510 leaves respectively, with no differences.
PWA retains all 44 module files and its existing browser launch behavior.
Feed retains all 21 reviewed Go source/test/dependency files from #98.
Notifications retains all 38 module files from reviewed #103, the earlier
AAD/UTF-8 repairs and the verbose dependency diagnostic. These compositions
retain the critical Core992 token-authority, session and export repairs.
Fresh combined-head CI remains required; no duplicate runtime suites were
run merely to compose unchanged module source.

At **11:59:01 UTC**, exact-Core992 run
[44516848989](https://github.com/djshellshoxxx/circula/actions/runs/44516848989)
had **all nine configured jobs queued, zero completed passes and zero completed
failures**. Queue state is not a test failure or a pass. The final main gate
remains external: current-head provider results, required reviews and normal
merge controls must complete. Owner-policy, manual accessibility, professional
security-review and release gates remain as described in
[REMAINING_BLOCKERS.md](REMAINING_BLOCKERS.md).

The [closing remote-head snapshot](evidence/closing-publication-20261119/remote-heads.json)
and [original-branch preservation receipt](evidence/closing-publication-20261119/original-branch-preservation.json)
retain the complete named observations and ancestry checks in the repository.

This closing change updates only the six audit reports and those two evidence files. Non-documentation
blobs and `docs/TRACEABILITY.md` remain exactly Core992; no runtime test or
trace regeneration is needed for this documentation-only follow-up. Its
publication will produce a new PR90 head. **Current PR heads and checks are
authoritative after that publication**; the source-bound results below must
not be relabeled as passes on the new head. The isolated branches require no
runtime-source refresh for this documentation change.

## Starting and ending revisions

| Revision | SHA / evidence |
|---|---|
| Starting main | [`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`](https://github.com/djshellshoxxx/circula/commit/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625) |
| Main at this report checkpoint | Same starting SHA; no new main merge has yet been claimed |
| First published combined core | [`354ab550166de10d1727a9990d8e08dc981545b8`](https://github.com/djshellshoxxx/circula/commit/354ab550166de10d1727a9990d8e08dc981545b8), exact reviewed tree `621ada23969d9fdb3abb17ac6179de8b2b48551d` |
| Published core with ledger/recovery/policy repairs | [`9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c`](https://github.com/djshellshoxxx/circula/commit/9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c), exact reviewed tree `6694ae050b395f94b5be4e43aba1a906d0d52017` |
| Core with reviewed session footer fixture | [`244e1356201e6477eab4694d82947f2058f3b0d8`](https://github.com/djshellshoxxx/circula/commit/244e1356201e6477eab4694d82947f2058f3b0d8), tree `7ada979757198fb9c19726858d17c897dfdca1b9`; requires the subsequent API-036 bounded-session correction before merge |
| Latest core candidate and merge receipt | [PR #90](https://github.com/djshellshoxxx/circula/pull/90); check its exact head and current checks, not an earlier successful source PR |
| GOV-021 fixture merged into its topic | [`0d480a5b348f3d385df0dc9c9a3e43a0b03a8332`](https://github.com/djshellshoxxx/circula/commit/0d480a5b348f3d385df0dc9c9a3e43a0b03a8332), [PR #91](https://github.com/djshellshoxxx/circula/pull/91), merged 2026-11-19 at 09:20:26 UTC into `claude/gov-021-admin-recovery`; main-adoption policy hold remains |
| Notifications repair merged into its topic | [`797dddf21adddfb34a49e7bda1871a202968dcda`](https://github.com/djshellshoxxx/circula/commit/797dddf21adddfb34a49e7bda1871a202968dcda), [PR #93](https://github.com/djshellshoxxx/circula/pull/93), merged 2026-11-19 at 10:33:33 UTC into `future/fux-104-notifications-20261116`; exact resulting tree matches qualified source and module adoption remains held |

Git data publication uses the connected provider because the shell Git client had no push credentials. Every published tree is compared with the local reviewed tree before creating a commit/ref. Updates are non-force and guarded by the expected old head. Publication commits record local source SHAs; a local SHA is not falsely presented as a remotely retrievable commit. Original remote source heads remain reachable and their branches are preserved.

## API-036 focused continuation, 2026-11-19

The complete session-management follow-up is [PR #101](https://github.com/djshellshoxxx/circula/pull/101),
based on concurrent PR90 head `f25d10b6b4085b80d7de53ecbb55efffeb9d8969`. Published
implementation `6f8ead1951f2d4b72649909b2b4cc83000948afe` exactly matches local
reviewed tree `a56e533e88275c05498c39db2c97c5f15b2af2fa`. It includes previous-page
keyset navigation, current-preserving bulk sign-out, atomic MFA/audit handling,
migration 0021 and cursor resource/encoding repairs. Narrower draft #96 was
closed as superseded, preserving its branch and code in #101.

[API036_COMPLETION.md](API036_COMPLETION.md) records implementation, branch
provenance, independent reviews, real SQLite/Go stress measurements and actual
failed/interrupted checks. [API036_DATABASE_REVIEW.md](API036_DATABASE_REVIEW.md)
links the portable SQL probe and preserved measurement JSON. These receipts
remain bound to their stated #101 source. Since #101 was included in the
Core integration branch, current-head full Go/race/vet/check, browser, fuzz
and native platform receipts on #90 govern combined qualification. The first
full run found two fixture compatibility failures now repaired without removing
assertions. Local Chromium aborts before assertions because socket() is denied.
All future modules remain isolated and disabled. No main merge, release or
deployment is asserted by this continuation record.

## Evidence-backed counts

| Measure | Observed count / definition |
|---|---|
| Starting real remote branch refs inspected | **107**, including main; excludes symbolic origin/HEAD |
| Starting tips already ancestral to main | **52**, including main |
| Starting non-ancestor tips reviewed | **55** |
| Later remote refs preserved at bounded cutoff | **123**, observed 2026-11-19 at 11:27:08 UTC; all 107 original names retained |
| Closing bounded live-ref snapshot | **125**, observed 2026-11-19 at 12:10:21.881995 UTC; independently confirms frozen 123 plus the two named #106/#107 additions |
| Original-tip preservation at closing snapshot | **105 unchanged + 2 advanced with original tip ancestral**; all 107 original branch names remain |
| Open PRs at closing snapshot | **23** returned by the bounded open-PR collection; separate from the initial 89-PR inventory |
| Non-ancestor clean merge simulations / conflicts | **32 / 23** |
| Non-ancestor zero-tree-change merges | **3**; retained without claiming new behavior |
| PRs inventoried | **89**: 10 open, 79 closed; pagination exhausted |
| Standalone issues found | **0**; issue endpoint records were all PRs |
| Original source PR heads preserved in core candidate | **5**: #82, #87, #88, #78, #89 |
| Additional concurrent core PR histories composed | **5**: #96, #99, #101, #102, #104; latest reviewed heads and limits in CORE_COMPOSITION_20261119.md |
| Coordinator-created repair/integration PRs through closing publication | **9**: #90–95, #100, #106 and #107; excludes concurrent source PRs and does not count defects, completed features or main merges |
| New main merges at this checkpoint | **0**; pending current-head gates |
| Isolated topic merges observed | **2**: #91 into GOV-021 and #93 into notifications; configured active gates passed at each exact source and each resulting tree equals its qualified source tree |
| Integration-only source inclusions | **2**: #101 into Core integration at `99247859`, #98 into feed integration at `db80ed34`; neither is a main merge |
| Original refs deleted or force-pushed | **0**; all 107 starting refs retained |
| Future modules inspected / qualified for core activation | **15 / 0** |
| Default future race / vet suites passed | **15 / 15** each |
| Explicit future SQLite-tagged whole-module race suites passed | **5 / 5** |
| PWA JavaScript tests passed | **22 / 22**; historical provider Chromium run also passed, while the later core refresh requires new checks |

“Preserved in a candidate,” “merged into main,” “repaired source,” and “runtime-qualified” are different counts. The final PR/head receipt must be checked before treating a pending candidate as merged. Repair work covers the core dispute source, both feed source PRs, notifications, independent SEC-026 and the GOV-021 fixture; it does not make their unresolved product policies approved.

## Bugs repaired and features advanced

1. **Dispute rule stability:** completed #87 with an immutable policy version stored at opening, exact-version readers, forward migration/backfill guards and equal/reversed-clock regressions.
2. **Ledger reconciliation:** fixed cancellation after intermediate int64 overflow using exact aggregate arithmetic, without changing historical financial data. Added eleven arithmetic-order/boundary cases and a real balanced-ledger verification fixture. The monetary-safety guard permits only exact `big.Int` APIs and continues to reject floats, including aliases/inferred conversions.
3. **Password throttling/recovery:** preserved #82's credential-proof race fixes; corrected web HTTP 429 handling and cleared obsolete proof locks only after successful authorized recovery transactions.
4. **Historical policy compatibility:** readable rule logs now apply the same missing-field defaults as execution readers, preserve explicit zero and leave stored JSON intact.
5. **Setup input handling:** validate decimal precision before formatting a rejection example; add accessible field errors and bounded malicious-input regressions.
6. **Onboarding catalogue:** externalised 51 setup/invitation/open/approval-join messages with exact amounts, ungrouped terms IDs, complete throttle sentences and contextual escaping. Full UX-047 remains partial.
7. **Account sessions:** completed the owner-scoped list/revoke flow with migration-0020 immutable public selectors, indexed keyset reads bounded to 20 × 100 retained rows per request, direct confirmation/revocation lookup and HTML/API continuation. The required API-036 correction was independently reviewed; focused service/template and real SQLite checks passed. Fresh full provider qualification remains pending.
8. **Feed module:** combined #67/#80 cursor work, rejected contradictory duplicate source records, and repaired nil/zero codec behavior. Independent race execution reached 98.8% coverage; cursor fuzzing passed 66,869 executions. Dedicated nested-module CI preserves the HOLD boundary.
9. **Notifications module:** repaired ambiguous encrypted-target context and invalid-UTF-8 identity normalization, preserved safe legacy compatibility and documented fail-closed recovery. Explicit SQLite-tagged race suite passed at 71.0% root-package coverage.
10. **Held topic branches:** independent SEC-026 now defaults off, exposes settings changes in member history, fixes touch sizing and excludes future-dated events after clock rollback. GOV-021's missing accessibility fixture is supplied without importing that policy-blocked feature into core.
11. **CI and documentation:** replaced duration-only fuzz termination with execution budgets and hard deadlines; preserved the earlier audit/evidence archive, corrected stale current-status descriptions and created all six requested audit reports plus portable future-module evidence.
12. **PWA qualification:** [PR #95](https://github.com/djshellshoxxx/circula/pull/95) adds provider execution of the existing browser suite after the local launch was blocked. Original runtime, harness, lockfile, assets and disabled/HOLD manifest are unchanged. Workflow/source/static checks and the provider browser harness passed at f25d037; the closing `0b0d7062` composition requires its own current-head checks.

13. **API-token inventory:** moved the existing first-100 response contract into the repository query, with an owner/ID index, preserved ordering and owner-scoped revocation beyond the cap, plus repository and HTTP regressions.

14. **Federation signing input:** independent review of #105 reproduced false signing success for all 30 malformed-UTF-8 cases against the parent source. Every returned envelope was already refused by the original verifier. The repaired exact source passed an independent race run (1.146s, 87.5% coverage) and vet; this establishes a signing-input repair, not a verifier/replay bypass. #106 corrects only the README's workflow-isolation claim. Both remain outside Core and on HOLD.

Detailed finding IDs, regression names and reviewer roles are in [SECURITY_REVIEW.md](SECURITY_REVIEW.md). [FEATURE_COMPLETION_STATUS.md](FEATURE_COMPLETION_STATUS.md) separates implemented, partial, superseded and blocked work.

The later compatible core streams also close token authority inside the write
transaction, failed-rollback connection reuse, migration cleanup, UUIDv7 job
allocation, export snapshot authority, incomplete HTTP archives, ambiguous ZIP
verification, listing lookahead/filter loss and duplicate requirement checking.
The full session navigation/bulk command and reconciled 525-key catalogue are
implemented. Independent reviewers approved the source composition; focused
tests and exact limits are in [the composition checkpoint](CORE_COMPOSITION_20261119.md).
Full current-head provider gates remain pending, so this is not a completed main
consolidation or release-readiness claim.

## Before and after testing

| Gate | Starting evidence | Recorded candidate evidence / limit |
|---|---|---|
| Main scheduled CI | [44296161788](https://github.com/djshellshoxxx/circula/actions/runs/44296161788): 8 successful jobs, strict-JSON fuzz deadline failure | Combined PR jobs required at final head. Initial run [44483334411](https://github.com/djshellshoxxx/circula/actions/runs/44483334411) was superseded/cancelled; [44484102222](https://github.com/djshellshoxxx/circula/actions/runs/44484102222) started qualification of `9b1fc6a0`. Neither is silently a final-head pass. |
| Amount fuzz | Baseline 678,832 executions | **PASS, 1,000,000 executions**, 68.601 seconds locally with pinned Go and hard timeout |
| Strict-JSON fuzz | Baseline 41,458 executions then timer-completion error | **PASS, 100,000 executions**, 18.770s against actual production API source files plus the original fuzzer; unrelated SQLite test fixtures excluded, normal package CI still required |
| Domain, message and repository race packages | Existing baseline provider coverage | **PASS** locally after exact-integer guard repair: 3.738s / 1.134s / 1.661s |
| Actual web catalogue/template rendering | Existing authentication catalogue tests | **PASS**, 6.625s, actual production web source files plus authentication/onboarding render tests; explicitly excludes SQLite-dependent web integration fixtures |
| Full core Go/race/migration/contract suite locally | Build attempts exceeded SQLite compiler budget | **Not qualified locally**. Source reviews/SQL trigger checks do not replace the provider runtime gate |
| Full native core race suite | Existing baseline provider coverage | **PASS on `244e135`**, macOS job [135445609909](https://github.com/djshellshoxxx/circula/actions/runs/44490739417/job/135445609909), completed 09:31:22 UTC; API, contract, service, store and web packages passed. This predates the required bounded-session follow-up and does not qualify its future changes. |
| GOV-021 fixture provider gates | Original topic browser fixture was missing | **7/7 configured jobs PASS** on `d922d4f` in [44486475798](https://github.com/djshellshoxxx/circula/actions/runs/44486475798); optional native placeholder skipped. Merge `0d480a5` has the exact same tree. Original #85 retains policy and core/schema-conflict blockers. |
| Tooling | Existing CI gates | Export reader 10 tests, QA runner 4, threshold checker 8, action-pin/permission and hygiene checks **PASS** |
| Future modules | Stale manifests and root CI did not qualify nested modules | 15 default race/vet and 5 explicit SQLite whole-suite passes; [portable commands, outputs and attempts](FUTURE_MODULE_QUALIFICATION.json) |
| PWA browser | Local attempt **BLOCKED before assertions** by sandbox socket denial | Historical provider run44496951525/job135466500874 **PASS** at f25d037, including 22 Node tests and the real Chromium harness; closing `0b0d7062` head requires its own checks |

## Main readiness and next priorities

At this checkpoint main is unchanged, and the combined candidate is not declared release-ready. Required reviews and exact-head provider checks must pass before a normal PR merge. The inspection could not read legacy protection settings (403), so no absence-of-protection assumption is made.

The remaining priorities are: qualify and merge the reviewed core repairs; finish account-session and onboarding browser acceptance; consolidate verified future repairs into their own targets; resolve reviewer/operator and SEC-026 policy decisions; continue full catalogue migration and uncovered storage-fault testing; then complete manual accessibility, external security review and reference-hardware release criteria. See [REMAINING_BLOCKERS.md](REMAINING_BLOCKERS.md).

No deployment, release tag, branch deletion, force push, blanket conflict-resolution strategy or alteration of historical ledger balances was performed.
