# Branch and PR refresh — 2026-11-19 10:59 UTC

This additive snapshot preserves the [initial 114-branch inventory](BRANCH_INVENTORY.md) and its evidence. It accounts for concurrent work discovered afterward; it does not certify Core completion or authorize any future module.

## Snapshot and provenance

The final enumeration was captured at **2026-11-19 10:59:30 UTC / 02:59:30 America/Los_Angeles**. Exact-head check receipts were collected through **11:03:14 UTC**. An intermediate observation at 10:52:56 UTC contained 119 branches and 101 PRs. These are distinct observations, not corrections to the initial 10:19 UTC inventory.

| Inventory | Initial | Refreshed | Interpretation |
|---|---:|---:|---|
| Remote branches | 114 | 120 | Six new; five existing heads moved; 109 unchanged; none deleted |
| Pull requests | 96 | 102 | Six new drafts, #97–#102 |
| Open PRs | 16 | 20 | #93 merged and #96 closed; six new open PRs |
| Merged PRs | 44 | 45 | #93 merged into an isolated notifications branch |
| Closed without merge | 36 | 37 | #96 superseded by #101 |
| Standalone issues | 0 | 0 | Every enumerated issue record is a PR |
| Remote tags | 0 | 0 | `git ls-remote --tags origin` returned no refs |

`main` remains **`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`**. The current local source base is `5da8bc78b28fb9c272dfa71ab5e5f496a276e098`. Published PR #102 is `c97233fb5b795e85b4a81e19809bb9d9c76a15d1`; both have the identical tree **`69c292b9f7bed54a0534f279bda8c58f5316832c`**. A different commit identity does not change that tree comparison or transfer a CI result from another revision.

[Refresh evidence](evidence/branch-refresh-20261119-0259.json) records all 120 full branch SHAs and dispositions, all 102 PR states and targets, changed paths, incremental scopes, ancestry, exact-head check receipts, review records, and merge-conflict probes. Unchanged branches retain the original content disposition with an explicit reference to the initial record; they are not newly declared complete. The original JSON's SHA-256 remains `8989490d42896e49d05a105ee22dc76a7a4b7c76db47bee327aa8e8717f8b967`.

Enumeration used REST branch pages of 100 and 20, PR pages of 100 and 2, and issue pages of 100 and 2. The clone was fetched and immutable Git objects inspected. The REST tags endpoint was unavailable, so the tag count comes from Git remote enumeration. This refresh ran no full builds or product tests and changed no remote refs, PRs, labels, or reviews.

## Six new branches

Full SHAs and per-file scopes are in the evidence JSON. Short SHAs below identify the same immutable objects.

| New branch | Head | PR and target | Disposition |
|---|---|---|---|
| `master/fnet207-recurring-remote-20261119` | `73a911a9bec3` | [#97](https://github.com/djshellshoxxx/circula/pull/97) → `future/fnet-207-recurring-20261116` | Isolated recurring calendar work; disabled/HOLD and feature-incomplete remain |
| `feature/fnet-204-completion-parallel` | `949d32907f8c` | [#98](https://github.com/djshellshoxxx/circula/pull/98) → `audit/future-feed-cursors-20261119` | Isolated feed work layered onto #92; preserve active worker ownership |
| `integration/core-consolidation-20261119` | `47446b33b01f` | [#99](https://github.com/djshellshoxxx/circula/pull/99) → `main` | Core descendant of #90; reconcile token/storage/UUID repairs with #101 and #102 |
| `audit/future-notifications-bounded-core-20261119` | `be0c071ebb3d` | [#100](https://github.com/djshellshoxxx/circula/pull/100) → `future/fux-104-notifications-20261116` | Core refresh on isolated notifications target; no module activation |
| `codex/api036-completion-20261119` | `baedf4b6670f` | [#101](https://github.com/djshellshoxxx/circula/pull/101) → `integration/comprehensive-audit-20261119` | Active API-036 completion; supersedes #96 |
| `integration/core-recovery-20261119` | `c97233fb5b79` | [#102](https://github.com/djshellshoxxx/circula/pull/102) → `main` | Published local Core recovery; reconcile and qualify the combined result |

The number of future modules remains **15**. The three new future follow-up branches add no new module or permission to integrate one. The original 22 isolated module/follow-up branches now have three additional follow-ups; their disabled/HOLD boundaries remain.

## Five moved existing heads and two PR state changes

| Existing branch | Initial → current head | Verified change and disposition |
|---|---|---|
| `integration/comprehensive-audit-20261119` | `1e2f369a5419` → `f25d10b6b408` | Two changed paths: session template and regression. The next-page link now preserves the composite session cursor. Remains an active Core source ancestor, with #99's authority repair still needed. |
| `audit/future-feed-cursors-20261119` | `8638f3d30f27` → `0fed0480d3b1` | 37 paths carry bounded Core session/token work and the pager correction. No `future/` files changed. Preserve #92/#98 isolation. |
| `audit/future-pwa-browser-20261119` | `f25d037c3cc5` → `f153bf5b18f9` | 39 paths carry the Core refresh and workflow/adoption evidence. No `future/` files changed. #95 remains isolated. |
| `codex/api036-cursor-hardening-20261119` | `81f203555e56` → `94fcd018ac42` | Three changed paths implement bounded, canonical cursor decoding. #96 is now closed unmerged, not an active tests-only draft; preserve its source while #101 owns completion. |
| `future/fux-104-notifications-20261116` | `3bdf8e92d84a` → `797dddf21add` | #93 merged identity-binding repairs plus inherited Core history into this isolated target. The 867-path difference includes that inherited history; it is not 867 missing Core changes. Ten changed paths are under `future/notifications/`. |

All five moves are forward ancestry moves. [#93](https://github.com/djshellshoxxx/circula/pull/93) merged at `2026-11-19T10:33:33Z` into the notifications branch, **not `main`**. [#96](https://github.com/djshellshoxxx/circula/pull/96) closed without merge at `2026-11-19T10:47:51Z` and is superseded by #101. No other initial PR changed open/closed/merged state in this refresh.

API-036 also moved during the refresh itself: the 10:52 observation saw #101 at `6f8ead1951f2d4b72649909b2b4cc83000948afe`; the final observation sees its child `baedf4b6670f6bf39ec593978ce38780c5640215`. That additional commit changes 11 paths, adding completion/database review evidence and adapting the migration rollback and localized-help tests. Current #101 check receipts belong to `baedf4b`, not `6f8ead1`.

## Complete scope of the new PRs

These descriptions combine immutable path/diff inspection with attributed PR descriptions and reviews. Local execution claims in another worker's PR are not independently reproduced by this inventory. The JSON includes the full changed-path sets relative to each PR's merge base and each Core incremental base.

### Core #99: token authority, transaction cleanup, and record IDs

[#99](https://github.com/djshellshoxxx/circula/pull/99) contains #90's exact `f25d10b` head as its direct parent. Its main-relative scope is 416 paths; the independent increment is **33 paths in one commit**. It rechecks live session authority inside token write transactions, before token, notice, or audit writes. Storage changes discard a connection after failed rollback, reject late errors while reading migration metadata, and clean up failed migration commits. New adapter job IDs use UUIDv7; export IDs are allocated before the export's side effects, while existing consistency-run record IDs are retained. Legacy record paths remain accepted. The increment also includes regression sources, query-plan assertions, contract/API documentation, and its own branch-audit records.

The [source review](https://github.com/djshellshoxxx/circula/pull/99#pullrequestreview-5973679547) records a token mutation race in #90: authentication before logout/password reset can reach a later token write. It identifies #99's transactional recheck as the repair and says #90 should not merge alone. This is a `COMMENTED` engineering review, not an `APPROVED` GitHub review or final-head CI qualification. Its rollback, migration, token-authority and UUID work should be preserved and tested in the combined Core candidate, without duplicating the active implementation.

### Core #101: API-036 completion

[#101](https://github.com/djshellshoxxx/circula/pull/101) has **57 paths in two commits** over #90. Its scope includes bounded indexed previous-page session queries; Previous/Next/First navigation; handling disappearing rows; cookie-authenticated revoke-other-sessions; CSRF-protected HTML confirmation; retention of the current session, foreign sessions and API tokens; pending owner MFA invalidation; and atomic audit writes. Migration `0021_session_unrevoked_index.sql` supports the unrevoked-owner query. It also implements the 4,096-byte cursor cap, canonical cursor encodings, filter limits, and an 85-byte session-position cap that supersede #96.

The PR includes API/OpenAPI/authentication fixtures, repository/service/store changes, localization/help and operations documentation, database rollback/performance/query-plan tests, concurrency/stress tests, no-JavaScript browser scenarios, and bounded fuzz campaigns. The latest commit adds SQL measurement/review evidence and updates the migration rollback and localized-help fixtures. Those sources and reported local results require qualification of the actual resulting head; the nine current check runs were queued.

### Core #102: recovered export, pagination, catalogue, and traceability work

[#102](https://github.com/djshellshoxxx/circula/pull/102) changes **56 paths** against main. It reauthorizes exports in the data snapshot; aborts interrupted ZIP responses at the HTTP transport; preserves listing owner/type/search filters with one-row lookahead; adds 196 English catalogue entries and custom/singular unit handling; and restores duplicate-definition rejection in the current Go traceability checker. It also preserves the initial branch inventory and recovery/verification records. Its published tree equals local `5da8bc7`, as verified above.

The PR body records fail-before/pass-after SQLite export regressions and focused web, catalogue, JavaScript and traceability checks. It explicitly leaves the full combined race run, provider CI, broader/browser/security gates, and the real-client interrupted-export regression pending at publication. This refresh does not convert those pending items into passes.

### Isolated #97: recurring calendar rules

[#97](https://github.com/djshellshoxxx/circula/pull/97) changes **eight paths**: seven under `future/recurring/` and its dedicated workflow. It adds daily/weekly/monthly scheduling through `DueWithSchedule`, retains the daily API/key behavior, validates selectors and identities, and uses a bounded time-zone interval walk. Its documented calendar cases cover missing dates, DST gaps/folds, second offsets and current-local-date-only behavior. Tests include an independent oracle and dedicated native/fuzz jobs. The [source review](https://github.com/djshellshoxxx/circula/pull/97#pullrequestreview-5973650673) is `COMMENTED`; it cites a successful Windows module job and explicitly retains pending Linux/macOS/fuzz gates. Durable scheduling, authorization, restart/atomic firing, quotas, retention/policy and UI acceptance remain open. The module remains disabled/HOLD and feature-incomplete.

### Isolated #98: feed storage and private cursors

[#98](https://github.com/djshellshoxxx/circula/pull/98) changes **23 paths**, limited to `future/feed/` and its dedicated workflow. It targets #92; its merge base is #92's earlier `8638f3d` head, before the later Core-only refresh. The scope adds SQLite-backed keyset pagination, immutable insertion identities/tombstones, an insertion fence, bounded candidate/response work, and padded authenticated v3 cursors. The PR describes network/viewer/filter/key bindings, fixed traversal lifetime, v2 compatibility, canonical serialization, typed-nil/key ownership hardening, SQLite integration/query-plan tests, property tests and fuzzing. Its claimed local module coverage and campaigns do not qualify Core. Production joins, key custody, adoption boundaries and other remaining feature gates retain disabled/HOLD status.

### Isolated #100: bounded Core refresh on notifications

[#100](https://github.com/djshellshoxxx/circula/pull/100) changes **38 paths** against the now-merged #93 target: 37 Core/doc paths and one workflow line. The Core blobs bring bounded session/token work and the pager correction from `f25d10b`; no `future/notifications/` module file changes. The workflow adds verbose advisory detail to the existing pinned scanner invocation. It preserves the module's identity-binding fixes, dependencies, scanning semantics and disabled/HOLD boundary. #93's source head had eight successful checks and one optional skip; the notifications merge target and #100's different head require their own receipts.

## Overlap and conflict evidence

Pairwise probes used `git merge-tree --write-tree --name-only --no-messages` on the recorded immutable heads. They created only temporary Git objects, leaving all worktrees, indexes and branches unchanged. An automatically merged file still requires semantic review and tests.

| Pair | Observed textual conflict paths |
|---|---|
| #99 `47446b3` + #101 `baedf4b` | `docs/TRACEABILITY.md`; `internal/store/session_query_plan_test.go` |
| #99 `47446b3` + #102 `c97233f` | `docs/TRACEABILITY.md`; `docs/dev/MESSAGES.md`; `internal/web/messages/en.json`; `internal/web/messages/messages_test.go` |
| #101 `baedf4b` + #102 `c97233f` | `docs/TRACEABILITY.md`; `docs/dev/MESSAGES.md`; `internal/web/messages/en.json`; `internal/web/messages/messages_test.go` |

The independent #99/#101 increments overlap in ten files: `docs/TRACEABILITY.md`, OpenAPI, four `docs/audits/` status/review documents, `docs/dev/SERVICE.md`, API contract spec, `internal/store/session_query_plan_test.go`, and `internal/store/sqlite_person.go`. Preserve the bounded query plans and the distinct token/session authority rules when reconciling those changes. #99's generic transaction/migration recovery and #101's migration-0021 rollback fixtures are related qualification work even where their files merge automatically.

#102's export authorization/streaming changes are in different implementation files from #99's export job-ID allocation, but both affect the export flow and need combined regression coverage. The shared catalogue additions and workflow changes also require deliberate union and regenerated evidence. Selective recovery of one branch while dropping another branch's additions would not satisfy consolidation.

The isolated PRs have different intended targets. #97 and #98 change no Core implementation paths. #100 deliberately carries Core files into an isolated module tree; that does not make it a Core integration candidate.

## Exact-head checks and review status

The refreshed JSON includes every returned check-run ID, name, state, conclusion, completion timestamp, and URL for all 20 open PRs, main, superseded #96, merged #93's source, and its notifications merge target. No check-run list reached the 100-record page limit. Counts include distinct runs with the same job name; duplicate names have not been collapsed into a passing result.

| Ref / PR | Head | Refreshed receipts |
|---|---|---|
| main | `ead63b5ce24b` | 12 success; 1 failure |
| #90 | `f25d10b6b408` | 1 success; 8 queued |
| #92 | `0fed0480d3b1` | 8 queued; 1 skipped |
| #95 | `f153bf5b18f9` | 8 queued; 1 skipped |
| #97 | `73a911a9bec3` | 1 success; 9 queued; 1 skipped |
| #98 | `949d32907f8c` | 9 queued; 1 skipped |
| #99 | `47446b33b01f` | 7 queued; 1 skipped |
| #100 | `be0c071ebb3d` | 9 queued; 1 skipped |
| #101 | `baedf4b6670f` | 9 queued |
| #102 | `c97233fb5b79` | 7 queued; 1 skipped |

All six new PRs are drafts. #97 and #99 have one `COMMENTED` review each; #98, #100, #101 and #102 have none in the returned review records. None has an `APPROVED` review at this checkpoint. Optional native placeholder skips are not native-platform passes. Label changes alone do not qualify a head; preserve the actual subsequent workflow receipt.

Main's failed [fuzz job 134875883683](https://github.com/djshellshoxxx/circula/actions/runs/44296161788/job/134875883683) remains unchanged. The initial log inspection found `FuzzStrictUnmarshal` reaching 41,458 executions before `context deadline exceeded` at 11.02 seconds under a ten-second fuzz budget; it recorded no failing input or parser assertion. #90 already carries execution-count budgets and hard timeouts. Qualify that repair in the combined candidate instead of duplicating it.

Other current failed receipts remain isolated: #94's [Ubuntu job](https://github.com/djshellshoxxx/circula/actions/runs/44491136235/job/135446747495), #86's [browser job](https://github.com/djshellshoxxx/circula/actions/runs/44274333610/job/134698499222), and five checks each on the historical feed heads #67 and #80. #85 has zero check runs. The initial inventory records the known #94 fixture mismatches; this refresh does not claim newly diagnosing the other old failures. Passing source PR receipts (#78/#82/#87/#88/#89) do not qualify their combined descendants.

## Additional actionable Core slices

Two bounded UX-047 slices remain outside the changed handler/template sets of #90, #99, #101 and #102. Their literal text was inspected at local `5da8bc7`; this is evidence of remaining catalogue work, not a new feature proposal.

| Slice | Owned source area | Concrete work and verification |
|---|---|---|
| Trade proposal, review, action and history catalogue | `internal/web/h_trades.go`, `internal/web/h_trade_act.go`; `propose.html`, `propose_review.html`, `trade_accept.html`, `trade_confirm.html`, `trade_detail.html`, `trades.html` | Move remaining titles, field/refusal copy, review sentences, action consequences, confirmations and filter/empty-state labels into catalogue keys. Preserve exact amounts, names, ordinary HTML escaping, CSRF/idempotency and navigation. Verify payer/payee perspectives, custom/singular units, successful and refused actions, and no-JavaScript review/back flows. |
| Core inbox and report catalogue | `internal/web/h_notify.go`; `inbox.html`, `report_form.html`, `admin_reports.html` | Catalogue headings, unread/read actions, report categories/target labels, refusal flashes, filters and administrator handling text. Preserve disclosure behavior, role restrictions, escaped member/listing/note text, and rendered/read report outcomes. Keep existing stored notification payloads unchanged. This is Core UI work, separate from `future/notifications/`. |

Both slices append to the shared `internal/web/messages/en.json`; coordinate those additions after reconciling #90/#101/#102. Existing shared status/transaction-row messages are already catalogued. Avoid wholesale replacement of `base.html`, `render.go`, shared messages, or the generated traceability report. Use distinct key prefixes, meaningful rendered-flow regressions, and the existing catalogue validation. Neither slice requires changes to session APIs, token authority, storage recovery, feature flags, or future modules.

The 19 built requirements without a test citation in the current traceability report are not automatically 19 missing implementations. In particular, #99 already owns record-ID changes, #101 owns session/storage qualification, and the report identifies packaging, deployment, manual accessibility, privacy review and owner decisions that cannot be closed by adding citations. Map meaningful existing evidence before inventing additional tests. Preserve physical read/fsync/truncate and uncertain-commit coverage gaps until a suitable fault harness is available; #99's active cleanup work is not an invitation to duplicate it.

## Integration consequence

Reconcile the current #99, #101 and #102 heads, preserve all their compatible Core changes, regenerate shared evidence, and qualify the resulting exact head. Re-fetch before a merge because concurrent workers are active. The initial 114-branch disposition ledger remains historical evidence; this refresh supplies the six additions, five head updates and two PR state changes. All 15 future modules, policy holds, manual/external assessments, owner decisions, and release/deployment exclusions retain their existing boundaries.
