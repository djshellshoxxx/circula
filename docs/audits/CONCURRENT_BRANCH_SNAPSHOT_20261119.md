# Concurrent branch preservation snapshot — 2026-11-19

## Scope and cutoff

Read-only remote enumeration at `2026-11-19T11:16:04.447408+00:00` found **122 branch refs**. This is an additive snapshot; the original 107-branch matrix remains the historical starting record. Main remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. No remote ref, pull request, checkout, or index was changed by this review.

| Snapshot | Branch refs | Meaning |
|---|---:|---|
| Initial audit | 107 | Original source/history inventory |
| First audit publications | 113 | Original 107 plus six branches for PR90–95 |
| 11:16 UTC enumeration | 122 | The prior 113 plus nine branches for PR96–104 |

All 107 original branch names remain. Of their tips, 105 are unchanged; GOV021 and the notifications target advanced normally, with both original tips still ancestors of the current tips. No original branch was deleted. Relative to unchanged main, 52 current branch tips are ancestors and 70 are not. Non-ancestry by itself does not establish unique work.

The six first publication branches are `integration/comprehensive-audit-20261119`, `audit/gov021-fixture-20261119`, `audit/future-feed-cursors-20261119`, `audit/future-notifications-binding-20261119`, `audit/sec026-independent-20261119`, and `audit/future-pwa-browser-20261119`. The nine later branches correspond to PR96–104; each remains preserved, including closed or merged pull requests. Other workstreams retain their own earlier 114- or 120-branch cutoffs; those are different observation times.

## Concurrent candidates

The current anchor for Core comparisons is PR99 `47446b33b01f4993a1caa406516746858bbf0a11`; the notification anchor is PR100 `be0c071ebb3d26527d81a0c5b015effe07d5b1b2`. All four candidates below are drafts authored by the GitHub login `djshellshoxxx`. Branch prefixes are not evidence of a separate owner.

| PR | Exact head | Relative ancestry | Changed paths from PR base | Disposition |
|---|---|---|---:|---|
| [101](https://github.com/djshellshoxxx/circula/pull/101) | `baedf4b6670f6bf39ec593978ce38780c5640215` | 2 unique-side commits / 1 anchor-side commits | 57 | Preserve session completion; reconcile with Core99 before qualification. |
| [102](https://github.com/djshellshoxxx/circula/pull/102) | `f29ebb01ff681a1276f385b8b568e41355b83505` | 2 unique-side commits / 34 anchor-side commits | 63 | Preserve additive export, listing, catalogue and traceability repairs; reconcile overlapping catalogues. |
| [103](https://github.com/djshellshoxxx/circula/pull/103) | `13669485b641475fce526633f47055ef31997098` | 1 unique-side commits / 3 anchor-side commits | 4 | Preserve isolated notification transport repairs; compose with PR100 only on the future branch. |
| [104](https://github.com/djshellshoxxx/circula/pull/104) | `c3c991aa3c12a11edce2874e9444b9d2a99f746c` | 1 unique-side commits / 34 anchor-side commits | 5 | Preserve additive export-verifier hardening; it does not replace Core99. |

None of these four heads is already an ancestor of its comparison anchor, nor is the anchor its ancestor. `git cherry` found zero patch-equivalent non-merge commits and respectively 2, 2, 1 and 1 patch-unique commits against those anchors. This compares complete commit patches; it does not claim that bundled commits contain no previously shared logic.

### PR101: session completion

Head `baedf4b6670f6bf39ec593978ce38780c5640215` adds two commits above Core90 `f25d10b6b4085b80d7de53ecbb55efffeb9d8969`. Scope includes bounded previous/next/first navigation, selective bulk owner-session revocation, MFA challenge invalidation, bounded cursor decoding, migration `0021_session_unrevoked_index.sql`, API/HTML contracts, stress/query-plan/browser regressions, and qualification documentation. It supersedes the earlier cursor follow-up PR96 in scope; the historical branch is retained.

Forty-seven changed paths are untouched by Core99 since the common base; ten contain different changes on both sides. An object-only `git merge-tree` reports textual conflicts in `docs/TRACEABILITY.md` and `internal/store/session_query_plan_test.go`. Resolve the source test intentionally and regenerate traceability after the final combination. The 0021 number also collides with the independently held activity-signals migration if those features are later combined. This is a conditional integration dependency, not a defect in either isolated migration. This inventory is not a full source approval of the 57-file feature.

### PR102: additive Core recovery

Current head `f29ebb01ff681a1276f385b8b568e41355b83505` follows `c97233fb5b795e85b4a81e19809bb9d9c76a15d1`, both above main. The follow-up adds the real-client interrupted-export regression and updated evidence; it does not change production runtime. The combined 63-file candidate adds export authorization inside the data snapshot, interrupted-response aborts, listing pagination repair, English catalogue coverage, and duplicate-definition rejection in traceability generation. It does not contain Core99 ancestry or its token-authority repair.

Fifty-seven changed paths are untouched by Core99 since main; six contain different changes on both sides. The object-only merge conflicts in `docs/TRACEABILITY.md`, `docs/dev/MESSAGES.md`, `internal/web/messages/en.json`, and `internal/web/messages/messages_test.go`. The catalogue and its tests need a semantic union, followed by generation and tests. Security review separately approved the export slice at this exact head, including the real-client regression and its recorded before/after logs; no independent local runtime rerun is claimed here. Broader catalogue/traceability integration remains under the parent review.

### PR103: isolated notification transports

Head `13669485b641475fce526633f47055ef31997098` has merged notification target `797dddf21adddfb34a49e7bda1871a202968dcda` as its sole parent. Exactly four files change under `future/notifications/pushsend/` and `future/notifications/smtpsend/`. The fixes return a true nil error for successful Web Push responses, cap numeric Retry-After values before duration multiplication, and perform verified implicit TLS when an injected SMTP dialer replaces the dependency default.

This reviewer read all four files and the pinned go-mail v0.8.2 dial/TLS/auth implementation. No source blocker was found. The tests use synthetic local transports and check encrypted payloads, certificate/name failures before credentials, cancellation, response bounds, privacy and concurrent messages. The four paths are untouched by PR100; an object-only merge with PR100 is clean. Source approval is for the isolated future module with disabled/HOLD metadata retained. It is not a runtime pass or a production transport/provider matrix. The PR body currently contains duplicated/truncated fuzz-command sections and should be rewritten accurately before final reporting.

### PR104: additive export verification

Head `c3c991aa3c12a11edce2874e9444b9d2a99f746c` is one commit directly above main. It changes five files: the Go export verifier, two new regression files, the Python reader, and its tests. The changes reject ambiguous archive/manifest membership and incomplete, oversized or inconsistent package content. All five paths are untouched by Core99; the object-only merge is clean. Security independently approved this exact verifier slice. Current provider checks and traceability regeneration remain required; Core99 security fixes must also be preserved.

## Exact future qualification records

PWA PR95 `f153bf5b18f933308479eea7fe75c0f60d33ca38` has tree `da5a859ad50bc5bbd4ceb1312c214459c9e555e0`. Notifications PR100 `be0c071ebb3d26527d81a0c5b015effe07d5b1b2` has tree `6efe093ff4856fa73f50544d4f2eafe56382ffad`. Both exact published trees passed the actual local generated-traceability freshness command in clean detached worktrees:

```sh
GOTOOLCHAIN=local GOMAXPROCS=2 GOGC=50 go run -p=1 -mod=readonly ./scripts/traceability -check -o docs/TRACEABILITY.md
```

The Go executable was the repository-pinned Go 1.26.11 toolchain. PWA completed in 2.332 seconds and notifications in 2.388 seconds. These are generated-document checks only; no broad runtime suites were repeated. The generator Input HEAD header identifies its generation base, not a tested-build claim.

PR98 advanced from source-reviewed `949d32907f8cc2fab30ce237d95bb361023e84ba` to `a8aecfe8782df4782af256bbcecdf5fd1469f783` (tree `535ea5bc50e65d547345b258d671118093f2ad44`). Its nine-file follow-up changes documentation, evidence, status metadata and dedicated CI time limits. All 21 Go source/test/dependency files are unchanged. All ten source hashes in the published SQLite qualification receipt match the previously reviewed source. The recorded full SQLite ordinary/race/benchmark passes are attributable evidence from that workstream; this reviewer did not rerun them. The dedicated job limit rose from 20 to 25 minutes and SQLite race timeout from 8 to 12 minutes, retaining all assertions and exact-current-head provider qualification. Disabled/HOLD remains.

At the 11:14 UTC provider observation, the current Core and module jobs for PR92, PR95, PR100 and updated PR98 remained queued, with optional native placeholders skipped. There was no new failure evidence. Earlier successful module runs remain tied to their historical head identities; they do not qualify these updated candidates.

## Method and preservation

The evidence uses `git ls-remote --heads origin`, immutable commit/tree reads, `git merge-base --is-ancestor`, `git rev-list --left-right --count`, `git cherry`, path/blob comparison, and `git merge-tree --write-tree --name-only`. Merge simulation writes Git objects only; it does not move a ref or modify an index or checkout. A clean textual merge is not semantic or runtime qualification.

Retain all original and concurrent source branches. Reconcile one reviewed main candidate while preserving Core99 token authority and each accepted repair. Keep future PR92/95/98/100/103 isolated from main and qualify any later combination at its own exact head. No merge, activation, release or deployment is authorized by this snapshot.

## Bounded refresh — 11:27:08 UTC

The next atomic `git ls-remote --heads origin` observation, at **2026-11-19 11:27:08.020610 UTC**, found **123 branch refs**. This is a fixed cutoff, not a promise to track later writes continuously. The count is the initial 107 plus 16 additions, or the earlier 113 plus ten additions. No branch disappeared. The 122-ref snapshot above remains the record for its earlier observation time; subsequent movements must retain their own timestamps and exact heads.

Compared with the 11:16 snapshot, two existing heads advanced and one branch was added:

| Branch / PR | Previous head | Head at the bounded cutoff | Disposition |
|---|---|---|---|
| `codex/api036-completion-20261119`, PR101 | `baedf4b6670f6bf39ec593978ce38780c5640215` | `f7ac6d22a1bd1718f6f4e3a840da7a28c83d0cb0` | Latest reviewed session continuation preserved in the local combined candidate. |
| `integration/core-consolidation-20261119`, PR99 | `47446b33b01f4993a1caa406516746858bbf0a11` | `0a73ee42b96255601cf2227b1ddf6599ec07a0ef` | Additional cursor/session/recovery ancestry and historical evidence; preservation review described below. |
| `master/fnet209-utf8-20261119`, PR105 | Newly observed branch | `059b0e4508a6c4f8e29f6a4cccd06f92b14ea115` | Isolated federation signing-input repair; retained as pending provenance at this cutoff. |

### Verified PR96 closure

GitHub reports PR96 **closed and unmerged** at `2026-11-19T10:47:51Z`, with its branch still at `94fcd018ac4227ea5a5eb803478b3bb0a1aa3677`. The author's [closing comment](https://github.com/djshellshoxxx/circula/pull/96#issuecomment-7006893786), posted one second earlier, explicitly states that PR101 supersedes the narrower cursor repair while preserving its branch and provenance. This is an observed closed-as-superseded state; it is not a merge or CI pass. PR101's description also records the included canonical, size-bounded cursor hardening.

### PR99 ancestry and evidence preservation

New PR99 head `0a73ee42b96255601cf2227b1ddf6599ec07a0ef` has four parents: prior PR99 `47446b33b01f4993a1caa406516746858bbf0a11`, original PR96 `94fcd018ac4227ea5a5eb803478b3bb0a1aa3677`, earlier PR101 `6f8ead1951f2d4b72649909b2b4cc83000948afe`, and earlier PR102 `c97233fb5b795e85b4a81e19809bb9d9c76a15d1`. Compared with the reviewed combined source `65242f32afab0b47e836e8fbe37cd323d3df75e7`, its three unique history commits are the two original PR96 commits and this preserving merge. The newer PR101/PR102/PR104 work in the combined source must remain intact.

There are 71 new paths relative to that combined source: 58 continuation-evidence files, 12 post-snapshot-evidence files, and one reconciliation report. A pre-existing continuation report also changes, giving 72 unique changed paths. All are documentation/evidence paths. The 57 entries in the incoming continuation `SHA256SUMS` and 11 entries in the incoming post-snapshot `SHA256SUMS` all match their exact published blob bytes.

The evidence documentation explicitly separates completed historical runs, timeouts, missing receipts, coordinator observations, SQL-only checks, selected production-source tests and full-provider gates. Its nine completed future-module race commands belong to that workstream's recorded run; they neither replace nor contradict the separate complete 15-module qualification receipt in this audit. Empty logs without successful completion receipts are not passes. No archived result qualifies a later combined head.

The local preservation merge `ec0efb8a818378c9233b7ec6da1b7598da24c657` retains new PR99 as a parent. Independent comparison confirms **every non-documentation path is identical to reviewed source `65242f32afab0b47e836e8fbe37cd323d3df75e7`**. Its three expected conflicts preserve the reviewed catalogue and message boundaries; generated traceability remains subject to final regeneration. The 525-entry catalogue retains the complete forbidden-reset explanation instead of reverting to the older self-reset-only message.

One archived Python script lost one blank line at EOF to satisfy whitespace validation. Its bytes changed from 5,396 to 5,395 and its parsed Python syntax tree is identical. The archive README records the exact original parent and SHA-256; the historical execution JSON is unchanged. Only the script, provenance README and corresponding `SHA256SUMS` entries changed in that archive, and all 11 current digests independently revalidated. This is an explicitly documented integration copy, not a claimed rerun of the archived test.

### PR105 pending future provenance

[PR105](https://github.com/djshellshoxxx/circula/pull/105) is an open draft targeting `future/fnet-209-federation-20261116`. Its sole parent is original future head `1b3c92e03007e35e3f190e719a1ea78a807d4102`; the published tree is `4d9f608041e65a2a1a297f17614438790240bc04`. Exactly three paths change: `future/federation/envelope.go`, `future/federation/envelope_utf8_test.go`, and `.github/workflows/future-federation.yml`.

The author reports a malformed-UTF-8 signing-input repair and explicitly records that the verifier already rejected those envelopes; this is not presented as a reproduced verification or replay bypass. This inventory verifies the ancestry, three-path scope and unchanged disabled/HOLD manifest only. Full source review, the reported runtime results and current provider qualification are pending within this bounded audit snapshot. Federation stays outside Core, with durable transport/replay/trust/key-lifecycle work and broader feature acceptance unfinished.

Final publication records must name the exact combined head and its actual gates. Later branch advances are separate pending provenance and do not retroactively alter any frozen snapshot or historical validation receipt.
