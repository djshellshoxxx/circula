# Branch consolidation (2026-11-17 assignment, executed 2026-11-18)

> Historical preparation snapshot. Its counts, branch heads and pending-merge statements describe that earlier review. The [executed consolidation record](../audit/consolidation-2026-11-17/README.md) provides refreshed inventories, exact merges and completed closures.

Companion to [PR-CONSOLIDATION-2026-11-17.md](PR-CONSOLIDATION-2026-11-17.md), which holds the per-PR
dispositions. This file records branch ancestry, patch equivalence and recovered commits against
`main@b53f757682bff5f488be9c2686c2e8aa8ad4054a`. It supersedes the branch tables in
[branch-consolidation-report.md](branch-consolidation-report.md) (2026-11-11), which remains as history.

## Method

```bash
git fetch --unshallow origin        # the session clone was shallow; ancestry is wrong without this
git fetch --all --prune
git fetch origin '+refs/pull/*/head:refs/remotes/pr/*'
git merge-base --is-ancestor <tip> origin/main
git rev-list --count origin/main..<tip>
git cherry -v origin/main <tip>     # '+' = no patch-equivalent commit on main
git diff --name-only $(git merge-base origin/main <tip>) <tip>
```

Every `+` commit on a branch with product code was then checked by content on `main` (see the code-line detail
in the PR ledger). Merge commits on those branches carry no merge-only hunks (`git diff-tree --cc`).

## Summary

| Measure | Count |
|---|---|
| Remote branches (excluding the integration branch) | 88 |
| Ancestors of `main` (fully contained) | 42 |
| Not ancestors | 46 |
| of which isolated future-module branches (15 modules, catalogue, PR #66 and #67 sources) | 18 |
| of which squash-merged into `main` (fix PRs #45, #51–#54; #68 and its sources) | 8 |
| of which content already on `main`, superseded or obsolete | 12 |
| of which archived as historical documentation | 7 |
| of which blocked by an owner decision (no PR) | 1 (`planning/future-cli-specs-20261110`) |
| Commits recovered into core code | 1 (`ebb4e298` → `cb79676`) |
| Regression tests added | 2 (`91a26f2`, from the PR #42 audit follow-up) plus `TestStartupMigrationGate` and `TestServeCreatesNewDatabaseInExplicitMode` |

No branch was deleted, rebased or force-pushed.

## Lineage (verified)

- `main@4b422ad` (prompt only) is the root. The product line is
  `claude/circula-security-features` → `claude/circula-build` (#4) → … → `claude/happy-noether-444cum` (#19) →
  `claude/circula-beta-completion-1npjw6` (#31) and `claude/quirky-einstein-l9ni8m` (#33/#43) →
  `codex/circula-core-completion-20261116` (#56) → #57–#62, #64, #65 → #68 (`b53f757`). All of it is in `main`.
- `claude/cool-ramanujan-n2peag` (#35) is a parallel beta follow-up line that forks from the `claude/quirky-einstein-l9ni8m` line at `8b4c8ba` (the #43 head);
  #40 merged into it and #42 is stacked on it. `main` received equivalent or stronger versions of its work
  through #56 and #65.
- `haiku/circula-consolidation` (base of #34) builds on `claude/happy-noether-444cum` (#19 head `34ccb81`). Its four unique
  patches (H05 tests, UX-021 test, CI runner and push changes) are on `main` under the same content.
- All 15 `future/*` module branches and `future/module-catalog-20261116` fork from `main@2eacf77` and touch only
  `future/<module>/` (own `go.mod`) or `docs/plans/`. They need a rebase before any integration.
- The spec-draft branches (#1, #2, #3, #5, #29, #38) and `planning/future-cli-specs-20261110` fork from
  `main@4b422ad`. The canonical specs come from a separate lineage (`dbd85c8`).

## Branches not contained in `main` (46)

| Branch | Tip | Commits not in `main` | Non-patch-equivalent | Files | PR | Classification |
|---|---|---|---|---|---|---|
| `chatgpt/circula-consolidation-review-20261111` | `02f9f0a7` | 22 | 10 | 53 | 34 | Patch-equivalent / content present on main |
| `chatgpt/circula-departure-pagination-20261115` | `8abff32c` | 4 | 4 | 2 | 54 | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-erasure-notification-20261115` | `26b57163` | 1 | 0 | 1 | 53 | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-ledger-auth-guard-20261115` | `29e46e50` | 3 | 3 | 2 | 51 | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-mfa-decider-20261115` | `4ffb5c78` | 4 | 4 | 2 | 52 | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-qa042-traceability-audit-20261114` | `fe832fac` | 37 | 28 | 95 | 40 | Superseded (Go traceability generator); rest as cool-ramanujan |
| `chatgpt/circula-secure-json-write-20261115` | `e02085a6` | 8 | 8 | 4 | 45 | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-security-audit-20261115` | `36a80335` | 45 | 35 | 96 | 42 | Audit notes archived (`dca5ea9`); code as cool-ramanujan |
| `chatgpt/consolidation-delta-qualification-20261114` | `8f742877` | 1 | 1 | 1 | 36 | Obsolete docs (superseded) |
| `chatgpt/favorites-owner-scoped-collection-20261116` | `372a8f97` | 4 | 4 | 10 | 66 | Future-only; merged into FNET-202 branch (#66) |
| `chatgpt/feed-signed-cursors-20261116` | `7a528c61` | 4 | 4 | 10 | 67 | Future-only, open PR #67 |
| `chatgpt/zero-codex-continuation-spec-20261109` | `3d9c5619` | 1 | 1 | 1 | 20 | Integrated or archived (`ec44a2c`) |
| `checkpoint/circula-handoff-20261025` | `3d0080a9` | 6 | 6 | 7 | 13 | Integrated or archived (`ec44a2c`) |
| `claude/circula-completeness` | `fcd9fc58` | 1 | 1 | 1 | 7 | Obsolete docs (superseded) |
| `claude/compassionate-cannon-txdlz1` | `047e92e8` | 6 | 4 | 9 | 68 | Already integrated (squash-merged in #68) |
| `claude/cool-ramanujan-n2peag` | `c033cfa4` | 40 | 30 | 95 | 35 | Partially integrated; remaining partial commit ebb4e298 recovered (`cb79676`); rest present, superseded or obsolete |
| `claude/gracious-ptolemy-dscc7v` | `1d165989` | 3 | 1 | 5 | 30 | Patch-equivalent / content present on main |
| `claude/kind-einstein-v5ycss` | `3bfb2f78` | 1 | 1 | 1 | 37 | Obsolete docs (superseded) |
| `claude/lucid-archimedes-upisa8` | `d2c1c609` | 15 | 15 | 15 | 38 | Superseded |
| `claude/quirky-babbage-y9fpmm` | `92d204dc` | 1 | 1 | 1 | 39 | Obsolete docs (superseded) |
| `claude/relaxed-hopper-8q4vhq` | `2da1a293` | 22 | 22 | 17 | 29 | Conflicting but recoverable docs: archived (`a2b9680`, `c327f0b`), not merged into canonical specs |
| `claude/sweet-turing-m3ftja` | `01b6e5c7` | 13 | 13 | 36 | 2 | Conflicting but recoverable docs: archived (`a2b9680`, `c327f0b`), not merged into canonical specs |
| `codex/circula-integration-plan-20261030` | `127943ba` | 1 | 1 | 3 | 16 | Integrated or archived (`ec44a2c`) |
| `codex/dashboard-amounts-meter-20261116` | `d53196cc` | 1 | 1 | 3 | 63 | Already integrated (squash-merged in #68) |
| `dependabot/go_modules/golang.org/x/text-0.44.0` | `4a6ea9ab` | 1 | 1 | 2 | 48 | Already integrated (squash-merged in #68) |
| `docs/foundational-spec-shells` | `d96f2e75` | 1 | 1 | 16 | 3 | Superseded |
| `docs/foundational-specifications` | `22e9202f` | 31 | 31 | 39 | 1 | Conflicting but recoverable docs: archived (`a2b9680`, `c327f0b`), not merged into canonical specs |
| `docs/readme-and-wiki` | `db782649` | 1 | 1 | 2 | 5 | Superseded |
| `future/fnet-201-matching-20261116` | `12850290` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-202-favorites-20261116` | `efc9d696` | 9 | 9 | 42 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-203-receipts-20261116` | `bd2fbdbd` | 2 | 2 | 10 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-204-feed-20261116` | `5d3819c3` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-205-analytics-20261116` | `0b296510` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-206-areas-20261116` | `8b444b20` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-207-recurring-20261116` | `0437e548` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-208-governance-20261116` | `b4f51137` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fnet-209-federation-20261116` | `1b3c92e0` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fux-101-pwa-20261116` | `a6326b87` | 7 | 7 | 44 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fux-102-qr-20261116` | `9db8a323` | 8 | 8 | 58 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fux-103-discovery-20261116` | `34d0f889` | 7 | 7 | 56 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fux-104-notifications-20261116` | `3bdf8e92` | 4 | 4 | 34 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fux-105-messaging-20261116` | `7f92251c` | 2 | 2 | 8 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/fux-106-onboarding-20261116` | `473cab2f` | 6 | 6 | 35 | - | Future-only (isolated module, `enabled:false`, HOLD) |
| `future/module-catalog-20261116` | `dfa4779d` | 3 | 3 | 5 | - | Future-only (catalogue; SHAs stale) |
| `haiku/circula-consolidation` | `6fbfa13f` | 16 | 4 | 52 | - | Patch-equivalent / content present on main |
| `planning/future-cli-specs-20261110` | `8e2372c0` | 1 | 1 | 1 | - | Blocked by owner decision (indexed in archive; conflicts with exit codes and DOM-011) |

## Branches contained in `main` (42)

`git merge-base --is-ancestor <tip> origin/main` is true for each (tip, PR at tip):

`chatgpt/circula-expiry-starvation-20261115` (2eacf771, -), `chatgpt/zero-codex/audit-20261110-a` (2e729346, 22), `chatgpt/zero-codex/ci-macos-race-budget-20261110` (2078bf09, 26), `chatgpt/zero-codex/ci-macos15-retirement-20261110` (1760551f, 25), `chatgpt/zero-codex/cookie-policy-adapter-20261110-b` (7e848100, 28), `chatgpt/zero-codex/decidable-expiry-20261110-a` (b61a8e71, 27), `chatgpt/zero-codex/docs-20261110-a` (a376a4ed, 23), `chatgpt/zero-codex/qualify-pr19-20261110-a` (63b033d8, 21), `chatgpt/zero-codex/terms-return-20261110-b` (7620d7f5, 24), `claude/awesome-darwin-y6eaiz` (827e651f, -), `claude/charming-gauss-19fbfa` (8b4c8ba2, 43), `claude/circula-beta-completion-1npjw6` (bffcede8, 31), `claude/circula-build` (fc488445, 4), `claude/circula-finish` (d350474e, 8), `claude/circula-security-features` (f8278a04, 6), `claude/cool-gauss-vdxlod` (64fa4f68, 55), `claude/happy-babbage-dcumwz` (dc97f7d5, -), `claude/happy-noether-444cum` (34ccb81a, 19), `claude/quirky-einstein-l9ni8m` (4a8db9e8, 33), `claude/sharp-archimedes-tu76rh` (fe111799, 12), `claude/vibrant-brahmagupta-3uofnj` (33b200e0, 15), `codex/circula-checkpoint-recovery-20261116` (12175ec3, 59), `codex/circula-common-ui-catalog-20261116` (a112c979, 62), `codex/circula-consolidate-20261030` (7643fb61, 17), `codex/circula-core-checkpoint-20261116` (15375a8d, -), `codex/circula-core-completion-20261116` (e3c60a62, 56), `codex/circula-evidence-20261116` (f6ea3213, 58), `codex/circula-financial-messages-20261116` (cf5abf5c, 61), `codex/circula-message-catalogs-20261116` (96059bb4, 60), `codex/circula-noscript-qa` (ba575129, -), `codex/circula-qualification-20261116` (dc89212a, 57), `codex/circula-responsive-qa` (04e11989, -), `codex/circula-resume-20261024` (f4bfe6bc, -), `codex/circula-staging-refresh-docs` (9c0bc500, 18), `codex/circula-support-boundaries-20261028` (837d51b0, 14), `consolidation/http-persistence-regressions-20261116` (050798c9, 65), `consolidation/recover-future-specifications` (3a776e63, 64), `dependabot/go_modules/github.com/getkin/kin-openapi-0.153.0` (3fa76cdb, 47), `docs/fnet-203-optional-trade-receipts` (3d6ed55b, 41,44), `docs/future-usability-network-specs-20261111` (f4a051c1, 32), `main` (b53f7576, -), `work/spec-completion` (d39bfef1, 9).

## Recovered commits

| Destination | Source | Notes |
|---|---|---|
| `cb79676` and review follow-up | `claude/cool-ramanujan-n2peag@ebb4e298` (PR #35) | OPS-025 start-up migration gate pinned by helper and `runServe` tests; docs corrected. Adapted, not cherry-picked: the branch version hard-coded schema 16 |
| `91a26f2` | Audit follow-up in `chatgpt/circula-security-audit-20261115@36a80335` (PR #42) | New tests only; production fixes were already on `main` from #51 and #52 |
| `ec44a2c`, `a2b9680`, `c327f0b`, `78a9372`, `dca5ea9`, `3be5b7f` | Branches of #1, #2, #13, #16, #20, #29, #38, #42, future modules | Documentation archive and status; see the PR ledger |

## Not recovered, with reason

| Source | Reason |
|---|---|
| `claude/cool-ramanujan-n2peag` `.github/workflows/ci.yml` | Older action pins (checkout v4.3.1, setup-go v5.6.0); `main`'s CI was redesigned (#33, #46) |
| `claude/cool-ramanujan-n2peag` `export-dir` setting (`13beb485`) | Conflicts with `main`'s `export-spool-dir`, which fails closed and has no in-memory fallback |
| QA-042 Python drift audit (`a4fd31b2`, `513b978c`, `7fd7b14e`) | Superseded by `scripts/traceability` with `make traceability-check` in CI |
| `66df03d3` Go cache keys | `actions/setup-go` already keys the cache on the root `go.sum` |
| `tools/check_specs.py` (#29), `scripts/check-specs.py` and `scripts/gen-traceability.py` (#2) | Incompatible ID rules; superseded tooling. The traceability generator scans `.py` files, so copying them would add false citations |
| `prompts/next-milestone-circula.md` (#16), continuation prompts (#37, #39), `d0fcd404` | Obsolete instructions based on a prompt-only `main` |
