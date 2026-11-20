# Circula branch uniqueness review — 2026-11-19 Pacific

Observed UTC: `2026-11-20T06:10:10.674254+00:00`.

## Exact source

- Main: `aa01563fcc891da60fefebb8335eca27855678e8`.
- Candidate: `22c767a90f69d286642ef96e6b3c174836637a3c` (PR #125).
- Candidate tree: `9c1ad082bfd7befb9914000c79d5974f1017cc27`.
- All 145 remote names and SHAs match the freshly fetched consolidation checkout. Merge simulations ran in an isolated local mirror. The symbolic `refs/remotes/origin/HEAD` (short name `origin`) is excluded; the original alias-inclusive receipt is preserved separately.

| Classification | Tips |
|---|---:|
| Ancestor of candidate | 66 |
| Future module or supporting topic | 43 |
| Recorded policy/interface hold | 7 |
| Historical source recovered or superseded | 23 |
| Patch-equivalent history | 1 |
| Clean merge with identical tree | 3 |
| Core peer runtime recovered; documentation/test alternative | 2 |
| Total | 145 |

The 66 candidate ancestors include 60 main ancestors and six Core continuation tips. Remaining tips produce three exact no-op merges, 34 clean changed merges and 42 conflict simulations. Conflicts belong to retained sources, not the current integration worktree.

## Recoverable Core source

PR #125 composes the unique compatible Core implementation: session insertion/migration regressions, interrupted exports, archive membership and size/file-kind checks, additive Python manifest/compression handling, 1,431 English messages, singular precision correction, lost-COMMIT-ack tests and diagnostic/browser-fixture repairs. Qualify this source rather than recreating those features.

Fresh clean merges of `chatgpt/circula-ledger-auth-guard-20261115`, `chatgpt/circula-mfa-decider-20261115` and `codex/dashboard-amounts-meter-20261116` each produce the exact candidate tree. Erasure-notification has zero patch-unique commits; later accepted edits cause its conflict.

PR #104 at `bcdc569cbc483761bf2d0e2d7594636ac1713d7d` adds 102 documentation/evidence paths, zero runtime paths, in its simulated merge. Its Go verifier is byte-identical to the candidate. Candidate Python files preserve its manifest parity plus nine compression-guard and 89 regression lines absent from the older topic. Whole-file replacement would lose those additions.

PR #111 at `45356ca10e1ecd6e6441e0171910dd918b36f5d9` contains an alternate WAL observation fixture. Candidate already retains the pinned query-only witness serving that purpose and the six peer qualification documents. No missing production behavior was found.

All clean non-noop runtime additions are confined to future modules; remaining clean additions are documentation/old prompts. Historical dispositions were reconciled with preserved source reviews and newer refreshes; their tips have not changed. Keep those refs as provenance. The 43 future/support tips cover the existing fifteen disabled modules. Seven held tips preserve GOV-021, SEC-026, reviewer-reset or future CLI decisions.

## Scope and next steps

No additional compatible Core branch implementation was found outside PR #125. The complete source-pinned records are in `candidate_branch_compare.json`, including commit/path lists, merge results, patch counts and dispositions. This inventory is not runtime qualification; provider status is managed separately by the integration worker.

Read newer `docs/consolidation/CORE_VERIFICATION_20261119.md` and `CORE_UI_COMPLETION_20261119.md` before acting on old incomplete-feature lists. Older catalogue and COMMIT-ack gaps are now implemented. Remaining acceptance, storage fault, locale and recorded external/owner gates retain their stated scope.

Bounded independent source review of the Go export safeguards and diagnostic recorder found no additional blocker in those diffs. A Python nested-object manifest-merge compatibility gap was repaired separately in local commit `ed3e095f31053d9b7becaab1a89f309a6f166fca`; all 45 reader tests pass. Its source-specific record is `export_nested_verification.md`. The inventory itself remains pinned to PR #125.

No remote branch was deleted, rewritten, pushed or merged during this review.
