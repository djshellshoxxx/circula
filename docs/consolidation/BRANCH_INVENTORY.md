# Circula branch inventory — 2026-11-19

> Later consolidation: [combined recovery and validation status](../audits/CONSOLIDATION_CONTINUATION_2026-11-19.md). This file retains its original source snapshot and execution scope; final candidate status is recorded in PR #99.

This is a source and integration snapshot, not a completion certificate. It accounts for every remote branch visible during the pass and distinguishes ancestry, behavioral recovery, and test evidence.

## Snapshot and evidence

- Fetched `main`: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Its observed main CI has a strict-JSON fuzz deadline failure; this is not an all-green main qualification.
- Active Core integration PR [#90](https://github.com/djshellshoxxx/circula/pull/90): `1e2f369a541963540d93c3b7f455b052827839c7`. Its current-head jobs were queued at this snapshot. API-036 branches remain isolated.
- This consolidation pass: `integration/core-recovery-20261119`; inventory branch: `audit/branch-inventory-20261119`.
- Snapshot: **2026-11-19 02:19 America/Los_Angeles (10:19 UTC)**. Re-fetch before merging; other workers are active.
- **114 remote branches**, **96 PRs** (16 open, 44 merged, 36 closed without merge), **zero standalone issues**, **zero tags**.
- [Machine-readable evidence](evidence/branch-inventory-20261119.json) contains every full head SHA, full merge base, unique commit SHA/subject, changed path, test/documentation path, PR, patch-equivalence count, fetched check-run receipt, and disposition.
- Preserve the existing [branch ledger](../plans/BRANCH-CONSOLIDATION-2026-11-17.md) and [PR ledger](../plans/PR-CONSOLIDATION-2026-11-17.md). Their historical tests apply to their recorded revisions.

## Method and limits

The clone is not shallow. All 114 true remote heads were reconciled with paginated branch enumeration; symbolic `origin/HEAD` is excluded. REST enumeration includes Dependabot PRs #46–#50, omitted by the initial search result. All 96 issue records represent PRs. Closed does not mean merged.

For each branch this pass recalculated merge base, ancestry to main and PR #90, commits outside main, changed paths, and `git cherry` patch equivalence. Counts below measure ancestry, not valuable missing features. Merge commits appear in the commit count but not ordinary patch IDs. Non-equivalent commits can still be behaviorally recovered, superseded, isolated, or held for policy.

Twenty-seven historical source tips match their recorded content dispositions. The changed `claude/quirky-babbage-y9fpmm` tip was inspected separately below. No original branch was changed, deleted, rebased, or force-pushed. No speculative trial merges were performed for contained or out-of-scope branches. PR mergeability is scoped to the recorded PR head and target; it does not certify security or policy readiness.

## Disposition totals

| Disposition | Branches | Meaning |
|---|---:|---|
| Main baseline | 1 | Fetched main; current fuzz failure recorded |
| Heads contained in main | 51 | Direct ancestry |
| Core source heads recovered into PR #90 | 5 | Pending verified integration |
| Active Core integration | 1 | PR #90 needs current-head qualification |
| Active API cursor follow-up | 1 | PR #96 is a test-first draft |
| Isolated future-module work | 22 | Fifteen modules, catalogue, six follow-up branches |
| Governance/security policy holds | 5 | Preserve code without selecting unresolved policy |
| Historical recovered/superseded/archived | 28 | Includes one changed tip reviewed below; future CLI remains held |
| **Total** | **114** | Every observed branch has a disposition; feature acceptance remains separate |

## Current Core integration priorities

PR #90 contains the exact source heads of #78, #82, #87, #88, and #89. It also supplies durable dispute-policy binding missing from #87, exact ledger aggregate accumulation, password/recovery fixes, onboarding catalogue work, API-036 storage bounds, and the fuzz-workflow repair. Source PR checks do not qualify this combined tree. Review its latest head, complete required tests/reviews, and merge through the normal expected-head process.

PR #96 adds intentionally failing tests for oversized and noncanonical API cursor encodings. Its original head is based on an older #90 revision. Preserve active worker isolation until the repair and current-head qualification exist.

Unowned Core work described by the current audit includes remaining UX-047 catalogue flows, storage read/fsync/truncate and uncertain-commit qualification, and missing requirement evidence. Governance policies, language selection, manual screen-reader assessment, professional review, and release criteria remain distinct decisions or external gates.

## Core source branches awaiting integration

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `claude/aud4-auth-fixes` | `82dc18e07f6b` / `ead63b5ce24b` | 9 / 5 / 8 | [#82](https://github.com/djshellshoxxx/circula/pull/82) | Recovered into active Core integration PR #90; pending verified merge into main |
| `claude/aud4-f03-dispute-timelines` | `b6275747df66` / `ead63b5ce24b` | 13 / 7 / 25 | [#87](https://github.com/djshellshoxxx/circula/pull/87) | Recovered into active Core integration PR #90; pending verified merge into main |
| `claude/circula-consolidation-audit-0sydm1` | `0e5296a71f3f` / `ead63b5ce24b` | 10 / 7 / 8 | [#78](https://github.com/djshellshoxxx/circula/pull/78) | Recovered into active Core integration PR #90; pending verified merge into main |
| `integration/consolidation-execution-record-20261118` | `73244357ca13` / `b4398074ae4c` | 1 / 1 / 279 | [#89](https://github.com/djshellshoxxx/circula/pull/89) | Recovered into active Core integration PR #90; pending verified merge into main |
| `nodata/doc-sweep-20261118` | `866adc8f2c30` / `ead63b5ce24b` | 3 / 2 / 4 | [#88](https://github.com/djshellshoxxx/circula/pull/88) | Recovered into active Core integration PR #90; pending verified merge into main |

## Active Core branches

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `integration/comprehensive-audit-20261119` | `1e2f369a5419` / `ead63b5ce24b` | 32 / 21 / 397 | [#90](https://github.com/djshellshoxxx/circula/pull/90) | Authoritative active Core audit/API-036 candidate; draft PR #90; current-head verification pending |

## Active cursor follow-up

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `codex/api036-cursor-hardening-20261119` | `81f203555e56` / `ead63b5ce24b` | 32 / 21 / 387 | [#96](https://github.com/djshellshoxxx/circula/pull/96) | Active API-036 cursor-hardening tests; draft PR #96; preserve worker isolation |

## Isolated future modules and follow-ups

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `audit/future-feed-cursors-20261119` | `8638f3d30f27` / `ead63b5ce24b` | 38 / 25 / 399 | [#92](https://github.com/djshellshoxxx/circula/pull/92) | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `audit/future-notifications-binding-20261119` | `9c794a5b9e6f` / `ead63b5ce24b` | 37 / 24 / 423 | [#93](https://github.com/djshellshoxxx/circula/pull/93) | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `audit/future-pwa-browser-20261119` | `f25d037c3cc5` / `ead63b5ce24b` | 39 / 27 / 431 | [#95](https://github.com/djshellshoxxx/circula/pull/95) | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `chatgpt/favorites-owner-scoped-collection-20261116` | `372a8f97ae01` / `2eacf7718f5b` | 4 / 4 / 10 | [#66](https://github.com/djshellshoxxx/circula/pull/66) | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `chatgpt/feed-signed-cursors-20261116` | `7a528c611bff` / `2eacf7718f5b` | 4 / 4 / 10 | [#67](https://github.com/djshellshoxxx/circula/pull/67) | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `claude/fnet-204-signed-cursors` | `0952c94a5dba` / `2eacf7718f5b` | 5 / 5 / 11 | [#80](https://github.com/djshellshoxxx/circula/pull/80) | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-201-matching-20261116` | `12850290c6bc` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-202-favorites-20261116` | `efc9d696eea2` / `2eacf7718f5b` | 9 / 9 / 42 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-203-receipts-20261116` | `bd2fbdbd3adf` / `2eacf7718f5b` | 2 / 2 / 10 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-204-feed-20261116` | `5d3819c3b3af` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-205-analytics-20261116` | `0b2965101e1d` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-206-areas-20261116` | `8b444b20139e` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-207-recurring-20261116` | `0437e54897c6` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-208-governance-20261116` | `b4f511372ec2` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fnet-209-federation-20261116` | `1b3c92e03007` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fux-101-pwa-20261116` | `a6326b8725fe` / `2eacf7718f5b` | 7 / 7 / 44 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fux-102-qr-20261116` | `9db8a323d7dc` / `2eacf7718f5b` | 8 / 8 / 58 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fux-103-discovery-20261116` | `34d0f889631e` / `2eacf7718f5b` | 7 / 7 / 56 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fux-104-notifications-20261116` | `3bdf8e92d84a` / `2eacf7718f5b` | 4 / 4 / 34 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fux-105-messaging-20261116` | `7f92251cdb0d` / `2eacf7718f5b` | 2 / 2 / 8 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/fux-106-onboarding-20261116` | `473cab2fb45a` / `2eacf7718f5b` | 6 / 6 / 35 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |
| `future/module-catalog-20261116` | `dfa4779d05a4` / `2eacf7718f5b` | 3 / 3 / 5 | — | Future module only; preserve disabled/HOLD boundary; no automatic Core integration |

## Governance/security policy holds

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `audit/gov021-fixture-20261119` | `d922d4f88005` / `cdc8112cf80c` | 9 / 8 / 39 | [#91](https://github.com/djshellshoxxx/circula/pull/91) | Policy hold; preserve isolated branch pending authority/visibility/retention decisions |
| `audit/sec026-independent-20261119` | `26f350b8aa43` / `ead63b5ce24b` | 33 / 21 / 421 | [#94](https://github.com/djshellshoxxx/circula/pull/94) | Policy hold; preserve isolated branch pending authority/visibility/retention decisions |
| `claude/aud4-f02-reviewer-reset-policy` | `af54887aa0cd` / `cdc8112cf80c` | 2 / 2 / 3 | [#83](https://github.com/djshellshoxxx/circula/pull/83) | Policy hold; preserve isolated branch pending authority/visibility/retention decisions |
| `claude/gov-021-admin-recovery` | `0d480a5b348f` / `cdc8112cf80c` | 10 / 8 / 39 | [#85](https://github.com/djshellshoxxx/circula/pull/85) | Policy hold; preserve isolated branch pending authority/visibility/retention decisions |
| `claude/sec-026-abuse-signals` | `4d349c4d03bf` / `b4398074ae4c` | 13 / 9 / 76 | [#86](https://github.com/djshellshoxxx/circula/pull/86) | Policy hold; preserve isolated branch pending authority/visibility/retention decisions |

## Historical dispositions and recovery

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `chatgpt/circula-consolidation-review-20261111` | `02f9f0a7bb87` / `3c2a577ddf31` | 22 / 10 / 53 | [#34](https://github.com/djshellshoxxx/circula/pull/34) | Patch-equivalent / content present on main |
| `chatgpt/circula-departure-pagination-20261115` | `8abff32c1b4a` / `3fee2a399a94` | 4 / 4 / 2 | [#54](https://github.com/djshellshoxxx/circula/pull/54) | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-erasure-notification-20261115` | `26b57163d965` / `3fee2a399a94` | 1 / 0 / 1 | [#53](https://github.com/djshellshoxxx/circula/pull/53) | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-ledger-auth-guard-20261115` | `29e46e5016a0` / `3fee2a399a94` | 3 / 3 / 2 | [#51](https://github.com/djshellshoxxx/circula/pull/51) | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-mfa-decider-20261115` | `4ffb5c78d6b4` / `3fee2a399a94` | 4 / 4 / 2 | [#52](https://github.com/djshellshoxxx/circula/pull/52) | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-qa042-traceability-audit-20261114` | `fe832faced82` / `8b4c8ba23c99` | 37 / 28 / 95 | [#40](https://github.com/djshellshoxxx/circula/pull/40) | Superseded (Go traceability generator); rest as cool-ramanujan |
| `chatgpt/circula-secure-json-write-20261115` | `e02085a6aa1a` / `a4ccc84eb460` | 8 / 8 / 4 | [#45](https://github.com/djshellshoxxx/circula/pull/45) | Already integrated (squash-merged; fix + test verified on main) |
| `chatgpt/circula-security-audit-20261115` | `36a80335abf5` / `8b4c8ba23c99` | 45 / 35 / 96 | [#42](https://github.com/djshellshoxxx/circula/pull/42) | Audit notes archived (`dca5ea9`); code as cool-ramanujan |
| `chatgpt/consolidation-delta-qualification-20261114` | `8f742877fb53` / `8b4c8ba23c99` | 1 / 1 / 1 | [#36](https://github.com/djshellshoxxx/circula/pull/36) | Obsolete docs (superseded) |
| `chatgpt/zero-codex-continuation-spec-20261109` | `3d9c56196625` / `4b422ad1e857` | 1 / 1 / 1 | [#20](https://github.com/djshellshoxxx/circula/pull/20) | Integrated or archived (`ec44a2c`) |
| `checkpoint/circula-handoff-20261025` | `3d0080a9c436` / `30d18a162642` | 6 / 6 / 7 | [#13](https://github.com/djshellshoxxx/circula/pull/13) | Integrated or archived (`ec44a2c`) |
| `claude/circula-completeness` | `fcd9fc587848` / `9a86ea69c419` | 1 / 1 / 1 | [#7](https://github.com/djshellshoxxx/circula/pull/7) | Obsolete docs (superseded) |
| `claude/compassionate-cannon-txdlz1` | `047e92e88a0c` / `b7c27b44199c` | 6 / 4 / 9 | [#68](https://github.com/djshellshoxxx/circula/pull/68) | Already integrated (squash-merged in #68) |
| `claude/cool-ramanujan-n2peag` | `c033cfa448fb` / `8b4c8ba23c99` | 40 / 30 / 95 | [#35](https://github.com/djshellshoxxx/circula/pull/35) | Partially integrated; remaining partial commit ebb4e298 recovered (`cb79676`); rest present, superseded or obsolete |
| `claude/gracious-ptolemy-dscc7v` | `1d165989e217` / `3c2a577ddf31` | 3 / 1 / 5 | [#30](https://github.com/djshellshoxxx/circula/pull/30) | Patch-equivalent / content present on main |
| `claude/kind-einstein-v5ycss` | `3bfb2f78fb0a` / `4b422ad1e857` | 1 / 1 / 1 | [#37](https://github.com/djshellshoxxx/circula/pull/37) | Obsolete docs (superseded) |
| `claude/lucid-archimedes-upisa8` | `d2c1c6090a05` / `4b422ad1e857` | 15 / 15 / 15 | [#38](https://github.com/djshellshoxxx/circula/pull/38) | Superseded |
| `claude/quirky-babbage-y9fpmm` | `af69702be08a` / `8b4c8ba23c99` | 41 / 30 / 104 | [#39](https://github.com/djshellshoxxx/circula/pull/39) | Superseded Python traceability implementation after old PR #39 head; inherited #35/#40 product changes accounted for; duplicate-ID check behavior remains an explicit tooling follow-up |
| `claude/relaxed-hopper-8q4vhq` | `2da1a293ebf2` / `4b422ad1e857` | 22 / 22 / 17 | [#29](https://github.com/djshellshoxxx/circula/pull/29) | Conflicting but recoverable docs: archived (`a2b9680`, `c327f0b`), not merged into canonical specs |
| `claude/sweet-turing-m3ftja` | `01b6e5c7cfc5` / `4b422ad1e857` | 13 / 13 / 36 | [#2](https://github.com/djshellshoxxx/circula/pull/2) | Conflicting but recoverable docs: archived (`a2b9680`, `c327f0b`), not merged into canonical specs |
| `codex/circula-integration-plan-20261030` | `127943ba7aa2` / `24f417c944e4` | 1 / 1 / 3 | [#16](https://github.com/djshellshoxxx/circula/pull/16) | Integrated or archived (`ec44a2c`) |
| `codex/dashboard-amounts-meter-20261116` | `d53196cc6907` / `bf33325d4ac7` | 1 / 1 / 3 | [#63](https://github.com/djshellshoxxx/circula/pull/63) | Already integrated (squash-merged in #68) |
| `dependabot/go_modules/golang.org/x/text-0.44.0` | `4a6ea9ab3dff` / `b7c27b44199c` | 1 / 1 / 2 | [#48](https://github.com/djshellshoxxx/circula/pull/48) | Already integrated (squash-merged in #68) |
| `docs/foundational-spec-shells` | `d96f2e75634b` / `4b422ad1e857` | 1 / 1 / 16 | [#3](https://github.com/djshellshoxxx/circula/pull/3) | Superseded |
| `docs/foundational-specifications` | `22e9202f19f5` / `4b422ad1e857` | 31 / 31 / 39 | [#1](https://github.com/djshellshoxxx/circula/pull/1) | Conflicting but recoverable docs: archived (`a2b9680`, `c327f0b`), not merged into canonical specs |
| `docs/readme-and-wiki` | `db7826495ec2` / `4b422ad1e857` | 1 / 1 / 2 | [#5](https://github.com/djshellshoxxx/circula/pull/5) | Superseded |
| `haiku/circula-consolidation` | `6fbfa13fb696` / `3c2a577ddf31` | 16 / 4 / 52 | — | Patch-equivalent / content present on main |
| `planning/future-cli-specs-20261110` | `8e2372c06425` / `4b422ad1e857` | 1 / 1 / 1 | — | Blocked by owner decision (indexed in archive; conflicts with exit codes and DOM-011) |

## Heads contained in main

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `chatgpt/circula-expiry-starvation-20261115` | `2eacf7718f5b` / `2eacf7718f5b` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/audit-20261110-a` | `2e729346d629` / `2e729346d629` | 0 / 0 / 0 | [#22](https://github.com/djshellshoxxx/circula/pull/22) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/ci-macos-race-budget-20261110` | `2078bf096435` / `2078bf096435` | 0 / 0 / 0 | [#26](https://github.com/djshellshoxxx/circula/pull/26) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/ci-macos15-retirement-20261110` | `1760551f5b03` / `1760551f5b03` | 0 / 0 / 0 | [#25](https://github.com/djshellshoxxx/circula/pull/25) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/cookie-policy-adapter-20261110-b` | `7e8481004801` / `7e8481004801` | 0 / 0 / 0 | [#28](https://github.com/djshellshoxxx/circula/pull/28) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/decidable-expiry-20261110-a` | `b61a8e719030` / `b61a8e719030` | 0 / 0 / 0 | [#27](https://github.com/djshellshoxxx/circula/pull/27) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/docs-20261110-a` | `a376a4ed5924` / `a376a4ed5924` | 0 / 0 / 0 | [#23](https://github.com/djshellshoxxx/circula/pull/23) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/qualify-pr19-20261110-a` | `63b033d842a2` / `63b033d842a2` | 0 / 0 / 0 | [#21](https://github.com/djshellshoxxx/circula/pull/21) | Contained in main; historical test scope retained |
| `chatgpt/zero-codex/terms-return-20261110-b` | `7620d7f51913` / `7620d7f51913` | 0 / 0 / 0 | [#24](https://github.com/djshellshoxxx/circula/pull/24) | Contained in main; historical test scope retained |
| `claude/awesome-darwin-y6eaiz` | `827e651fe151` / `827e651fe151` | 0 / 0 / 0 | [#10](https://github.com/djshellshoxxx/circula/pull/10) | Contained in main; historical test scope retained |
| `claude/backup-path-hardening` | `09ef8a5aa25f` / `09ef8a5aa25f` | 0 / 0 / 0 | [#81](https://github.com/djshellshoxxx/circula/pull/81) | Contained in main; historical test scope retained |
| `claude/charming-gauss-19fbfa` | `8b4c8ba23c99` / `8b4c8ba23c99` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `claude/circula-beta-completion-1npjw6` | `bffcede8c7dc` / `bffcede8c7dc` | 0 / 0 / 0 | [#31](https://github.com/djshellshoxxx/circula/pull/31) | Contained in main; historical test scope retained |
| `claude/circula-build` | `fc488445af5f` / `fc488445af5f` | 0 / 0 / 0 | [#4](https://github.com/djshellshoxxx/circula/pull/4) | Contained in main; historical test scope retained |
| `claude/circula-finish` | `d350474e4952` / `d350474e4952` | 0 / 0 / 0 | [#8](https://github.com/djshellshoxxx/circula/pull/8) | Contained in main; historical test scope retained |
| `claude/circula-pr-consolidation-q6q18n` | `163fe1596bd7` / `163fe1596bd7` | 0 / 0 / 0 | [#69](https://github.com/djshellshoxxx/circula/pull/69) | Contained in main; historical test scope retained |
| `claude/circula-security-features` | `f8278a047392` / `f8278a047392` | 0 / 0 / 0 | [#6](https://github.com/djshellshoxxx/circula/pull/6) | Contained in main; historical test scope retained |
| `claude/cool-gauss-vdxlod` | `64fa4f685780` / `64fa4f685780` | 0 / 0 / 0 | [#55](https://github.com/djshellshoxxx/circula/pull/55) | Contained in main; historical test scope retained |
| `claude/happy-babbage-dcumwz` | `dc97f7d5b0ae` / `dc97f7d5b0ae` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `claude/happy-noether-444cum` | `34ccb81a54ab` / `34ccb81a54ab` | 0 / 0 / 0 | [#19](https://github.com/djshellshoxxx/circula/pull/19) | Contained in main; historical test scope retained |
| `claude/quirky-einstein-l9ni8m` | `4a8db9e80509` / `4a8db9e80509` | 0 / 0 / 0 | [#43](https://github.com/djshellshoxxx/circula/pull/43), [#33](https://github.com/djshellshoxxx/circula/pull/33) | Contained in main; historical test scope retained |
| `claude/sharp-archimedes-tu76rh` | `eb4bb08b15d4` / `eb4bb08b15d4` | 0 / 0 / 0 | [#84](https://github.com/djshellshoxxx/circula/pull/84), [#70](https://github.com/djshellshoxxx/circula/pull/70), [#12](https://github.com/djshellshoxxx/circula/pull/12), [#11](https://github.com/djshellshoxxx/circula/pull/11) | Contained in main; historical test scope retained |
| `claude/ux-047-auth-catalogue` | `dd3e82e27ddb` / `dd3e82e27ddb` | 0 / 0 / 0 | [#79](https://github.com/djshellshoxxx/circula/pull/79) | Contained in main; historical test scope retained |
| `claude/vibrant-brahmagupta-3uofnj` | `33b200e00c08` / `33b200e00c08` | 0 / 0 / 0 | [#15](https://github.com/djshellshoxxx/circula/pull/15) | Contained in main; historical test scope retained |
| `codex/circula-checkpoint-recovery-20261116` | `12175ec348d5` / `12175ec348d5` | 0 / 0 / 0 | [#59](https://github.com/djshellshoxxx/circula/pull/59) | Contained in main; historical test scope retained |
| `codex/circula-common-ui-catalog-20261116` | `a112c979932b` / `a112c979932b` | 0 / 0 / 0 | [#62](https://github.com/djshellshoxxx/circula/pull/62) | Contained in main; historical test scope retained |
| `codex/circula-consolidate-20261030` | `7643fb616208` / `7643fb616208` | 0 / 0 / 0 | [#17](https://github.com/djshellshoxxx/circula/pull/17) | Contained in main; historical test scope retained |
| `codex/circula-core-checkpoint-20261116` | `15375a8dc875` / `15375a8dc875` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `codex/circula-core-completion-20261116` | `e3c60a62c065` / `e3c60a62c065` | 0 / 0 / 0 | [#56](https://github.com/djshellshoxxx/circula/pull/56) | Contained in main; historical test scope retained |
| `codex/circula-evidence-20261116` | `f6ea3213e663` / `f6ea3213e663` | 0 / 0 / 0 | [#58](https://github.com/djshellshoxxx/circula/pull/58) | Contained in main; historical test scope retained |
| `codex/circula-financial-messages-20261116` | `cf5abf5cfb00` / `cf5abf5cfb00` | 0 / 0 / 0 | [#61](https://github.com/djshellshoxxx/circula/pull/61) | Contained in main; historical test scope retained |
| `codex/circula-message-catalogs-20261116` | `96059bb4b8ae` / `96059bb4b8ae` | 0 / 0 / 0 | [#60](https://github.com/djshellshoxxx/circula/pull/60) | Contained in main; historical test scope retained |
| `codex/circula-noscript-qa` | `ba5751291819` / `ba5751291819` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `codex/circula-qualification-20261116` | `dc89212a8bf1` / `dc89212a8bf1` | 0 / 0 / 0 | [#57](https://github.com/djshellshoxxx/circula/pull/57) | Contained in main; historical test scope retained |
| `codex/circula-responsive-qa` | `04e11989b811` / `04e11989b811` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `codex/circula-resume-20261024` | `f4bfe6bc3f81` / `f4bfe6bc3f81` | 0 / 0 / 0 | — | Contained in main; historical test scope retained |
| `codex/circula-staging-refresh-docs` | `9c0bc500cb19` / `9c0bc500cb19` | 0 / 0 / 0 | [#18](https://github.com/djshellshoxxx/circula/pull/18) | Contained in main; historical test scope retained |
| `codex/circula-support-boundaries-20261028` | `837d51b0d537` / `837d51b0d537` | 0 / 0 / 0 | [#14](https://github.com/djshellshoxxx/circula/pull/14) | Contained in main; historical test scope retained |
| `consolidation/http-persistence-regressions-20261116` | `050798c9cf94` / `050798c9cf94` | 0 / 0 / 0 | [#65](https://github.com/djshellshoxxx/circula/pull/65) | Contained in main; historical test scope retained |
| `consolidation/recover-future-specifications` | `3a776e630455` / `3a776e630455` | 0 / 0 / 0 | [#64](https://github.com/djshellshoxxx/circula/pull/64) | Contained in main; historical test scope retained |
| `dependabot/go_modules/github.com/getkin/kin-openapi-0.153.0` | `3fa76cdb5901` / `3fa76cdb5901` | 0 / 0 / 0 | [#47](https://github.com/djshellshoxxx/circula/pull/47) | Contained in main; historical test scope retained |
| `docs/fnet-203-optional-trade-receipts` | `3d6ed55b2e44` / `3d6ed55b2e44` | 0 / 0 / 0 | [#44](https://github.com/djshellshoxxx/circula/pull/44), [#41](https://github.com/djshellshoxxx/circula/pull/41) | Contained in main; historical test scope retained |
| `docs/future-usability-network-specs-20261111` | `f4a051c1c1af` / `f4a051c1c1af` | 0 / 0 / 0 | [#32](https://github.com/djshellshoxxx/circula/pull/32) | Contained in main; historical test scope retained |
| `integration/admin-grant-notice-pagination-20261117` | `cb25b41febd4` / `cb25b41febd4` | 0 / 0 / 0 | [#73](https://github.com/djshellshoxxx/circula/pull/73) | Contained in main; historical test scope retained |
| `integration/backup-private-staging-20261117` | `faed4906f541` / `faed4906f541` | 0 / 0 / 0 | [#72](https://github.com/djshellshoxxx/circula/pull/72) | Contained in main; historical test scope retained |
| `integration/browser-parity-clock-20261117` | `2f6b3e4f1b70` / `2f6b3e4f1b70` | 0 / 0 / 0 | [#77](https://github.com/djshellshoxxx/circula/pull/77) | Contained in main; historical test scope retained |
| `integration/erasure-fixture-stability-20261117` | `fc31dc9a6b8c` / `fc31dc9a6b8c` | 0 / 0 / 0 | [#76](https://github.com/djshellshoxxx/circula/pull/76) | Contained in main; historical test scope retained |
| `integration/release-threshold-check-20261117` | `ec98caba7328` / `ec98caba7328` | 0 / 0 / 0 | [#74](https://github.com/djshellshoxxx/circula/pull/74) | Contained in main; historical test scope retained |
| `integration/store-untrusted-schema-20261117` | `b1cd8f38450a` / `b1cd8f38450a` | 0 / 0 / 0 | [#71](https://github.com/djshellshoxxx/circula/pull/71) | Contained in main; historical test scope retained |
| `integration/throttle-documentation-20261117` | `83de536ee373` / `83de536ee373` | 0 / 0 / 0 | [#75](https://github.com/djshellshoxxx/circula/pull/75) | Contained in main; historical test scope retained |
| `work/spec-completion` | `d39bfef12d4a` / `d39bfef12d4a` | 0 / 0 / 0 | [#9](https://github.com/djshellshoxxx/circula/pull/9) | Contained in main; historical test scope retained |

## Main baseline

| Branch | Head / merge base | Commits / non-equivalent patches / files | PR | Disposition |
|---|---|---:|---|---|
| `main` | `ead63b5ce24b` / `ead63b5ce24b` | 0 / 0 / 0 | — | Authoritative main baseline |

## Current-head CI snapshot

The 16 open PR heads and main were refreshed through check-runs. Other branch records explicitly say their checks were not refreshed. Optional skipped jobs are not passes. Source-head success does not qualify a new merge tree.

| Target | Exact head | Observed checks |
|---|---|---|
| [#67](https://github.com/djshellshoxxx/circula/pull/67) | `7a528c611bffdf92ef9f8cbddb98e8b088eb5036` | 5 failure; 1 skipped; 2 success |
| [#78](https://github.com/djshellshoxxx/circula/pull/78) | `0e5296a71f3f3802f36e14d847c0abf7b1dbf882` | 1 skipped; 7 success |
| [#80](https://github.com/djshellshoxxx/circula/pull/80) | `0952c94a5dbab4044d7673ce9d51cec9063ab97f` | 5 failure; 1 skipped; 2 success |
| [#82](https://github.com/djshellshoxxx/circula/pull/82) | `82dc18e07f6b22b9c34ef026f833391705b30067` | 1 skipped; 7 success |
| [#83](https://github.com/djshellshoxxx/circula/pull/83) | `af54887aa0cd63ba080e2944aa52a5b759b008ca` | 1 skipped; 7 success |
| [#85](https://github.com/djshellshoxxx/circula/pull/85) | `0d480a5b348f3d385df0dc9c9a3e43a0b03a8332` | No check runs at this SHA |
| [#86](https://github.com/djshellshoxxx/circula/pull/86) | `4d349c4d03bfb81760d471a2a7ef4ac4338d9c45` | 1 failure; 1 skipped; 6 success |
| [#87](https://github.com/djshellshoxxx/circula/pull/87) | `b6275747df66ede81ee4267e2cd6a2595f3bd680` | 1 skipped; 7 success |
| [#88](https://github.com/djshellshoxxx/circula/pull/88) | `866adc8f2c30abd294843e43a2ba42a6fe5f7b67` | 1 skipped; 7 success |
| [#89](https://github.com/djshellshoxxx/circula/pull/89) | `73244357ca136874452a6cd826ffa91ac8c3c434` | 1 skipped; 7 success |
| [#90](https://github.com/djshellshoxxx/circula/pull/90) | `1e2f369a541963540d93c3b7f455b052827839c7` | 9 queued |
| [#92](https://github.com/djshellshoxxx/circula/pull/92) | `8638f3d30f27f4882c4c6424d4c8b248208b6f3d` | 2 in_progress; 1 skipped; 6 success |
| [#93](https://github.com/djshellshoxxx/circula/pull/93) | `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` | 2 queued; 1 skipped; 6 success |
| [#94](https://github.com/djshellshoxxx/circula/pull/94) | `26f350b8aa437d315c993c5555cb0b9fa2b7eba4` | 1 failure; 1 skipped; 6 success |
| [#95](https://github.com/djshellshoxxx/circula/pull/95) | `f25d037c3cc53fbf1ebbf687831300be9bdb749f` | 7 queued; 1 skipped; 1 success |
| [#96](https://github.com/djshellshoxxx/circula/pull/96) | `81f203555e562bc45d86a53341b9b5c44d0b550f` | 7 queued; 1 skipped |
| main | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` | 1 failure; 12 success |

### Main fuzz failure and existing recovery

[Run 44296161788, job 134875883683](https://github.com/djshellshoxxx/circula/actions/runs/44296161788/job/134875883683) checked out the main SHA above. `FuzzParseAmount` passed 678,832 executions. `FuzzStrictUnmarshal` reached 41,458 executions, then failed at 11.02 seconds with `context deadline exceeded` under `-fuzztime 10s`. No failing input or parser assertion is recorded; this does not establish a product defect.

PR #90 already replaces duration budgets with 1,000,000 amount-parser and 100,000 strict-JSON executions, each under a three-minute hard timeout. Assertions and provenance remain. Qualify this existing repair rather than duplicate it.

### SEC-026 fixture failures remain isolated

[Run 44491136235, job 135446747495](https://github.com/djshellshoxxx/circula/actions/runs/44491136235/job/135446747495) fails on the independent SEC-026 candidate. `TestSEC026SignalFailureIsLogged` expects a warning after signal generation was made explicitly opt-in; its log is empty. `TestSEC026PolicyFormSignalSettings` expects `20 to 5` and `60 to 15`, while the restored member-visible log renders settings snapshots. Investigate these fixtures on the held branch; they do not authorize signals in Core.

## Changed historical branch: PR #39 source

`claude/quirky-babbage-y9fpmm` points to `af69702be08aaf14cbe9a705ddf7f08796d5e976`; closed PR #39 and the old ledger record `92d204dc027320b1b26a05fd86e58232d89394ca`. The old “obsolete prompt only” disposition misses later work.

The branch merges the already-accounted #35 frontier and #40 traceability work; both merge commits have no merge-only hunks. Its independent tip adds a Python traceability generator/tests, generated inventory, CI/hygiene wiring, and four tracked bytecode caches. Main uses the Go generator with parser, citation, deterministic-output, test-discovery and execution-evidence tests. Do not restore obsolete Python or caches.

One semantic difference needs recovery: old Python rejects duplicate status IDs, while Go reports duplicates and checks freshness only. A regenerated matrix containing duplicates can therefore satisfy freshness. Add and test a strict duplicate gate in the Go tool. This snapshot records the finding; it does not claim the follow-up is implemented.

Subsequent recovery: the Go qualification guard and real-command regressions are now implemented in the consolidation branch. The [separate recovery record](evidence/traceability-duplicate-recovery.md) contains observed fail-before and full package race evidence. The snapshot above retains the historical branch and CI state it originally measured.

## Integration boundaries and next action

Preserve future-module heads. Feed #67/#80/#92, notifications #93, and PWA #95 target isolated branches and retain disabled/HOLD status. Independent SEC-026 #94 removes the original GOV-021 dependency but retains subject-visibility, threshold-authority and retention decisions. #83 and #85 retain their governance decisions.

Qualify current PR #90 and its cursor follow-up, integrate eligible Core work through the normal PR process, and refresh against the resulting main. Close or retarget source PRs only after verification. Preserve source refs and historical evidence. This inventory represents no release or deployment.
