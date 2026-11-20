# Local branch preservation review — 2026-11-20

Snapshot: **2026-11-20T23:48:15.555304+00:00**. Examined **26 local heads**, **152 remote branch heads**, and **961 commits reachable from those remote heads**. This supplements the remote/future-branch audit with all local development and worker branches.

**No uncovered runtime, test, or build path was found.** All local branches remain preserved. This is a source/history proof; no product builds, tests, provider CI, fetches, or remote writes were performed for this receipt. Current CI results are recorded separately by the coordinator.

## Pinned comparison streams

| Remote stream | Exact head |
| --- | --- |
| `main` | `aa01563fcc891da60fefebb8335eca27855678e8` |
| `integration/unique-core-completion-20261120` | `bea5de25cf76b66d87689034e2529f9b40c6dff6` |
| `repair/core127-e2e-observer-20261120` | `11739c7f8f59455b4dc480294cab4d79f7933cb9` |
| `audit/core127-final-evidence-20261120` | `49dc8acc781790635fca8ed32882a7a27971c8a1` |

Pending documentary staging: `cca252843196edb423def4c530049d31affca10b`. Root-owned uncommitted changes are excluded. The reviewer input head is `6eafbaad996fb846f3a71be735f57196164d9d37`; the commit adding this receipt necessarily advances that one branch after the recorded snapshot.

## Method and interpretation

[inventory.json](inventory.json) records every exact head/tree, selected comparison branch, merge base, every local-only commit and subject, and every changed path with base/local/remote blob IDs. Corresponding published streams are chosen by development ownership, rather than choosing an unrelated future stream merely because it has fewer differing commits. For each path it distinguishes current-remote equality, same-path remote-history witnesses, and whole-tree/history preservation. A historical witness proves recovery and retention, not that an earlier version must replace a later repair.

The classification does not treat different commit IDs as missing work. Merge ancestry, equal whole trees, selective source preservation, and pending documentary publication are separate findings. An empty changed-path list on an ancestral head means it adds nothing after its merge base; it does not mean the branch contains no files.

| Classification | Count | Meaning |
| --- | ---: | --- |
| History | 4 | The exact local commit is reachable from a current remote head. |
| Equal tree | 11 | A remote-reachable commit has the identical complete tree under another commit ID. |
| Preserved changes | 3 | Selectively integrated changes have current or historical same-path source witnesses. |
| Document variants | 4 | Runtime is preserved; older documentary forms remain on local branches with the qualified equivalence below. |
| Pending evidence | 4 | Documentary work is retained locally and every changed-path blob matches the pinned coordinator staging tree. |

## All local heads

| Local branch | Exact head | Local-only commits versus selected remote | Classification |
| --- | --- | ---: | --- |
| `audit/core-loading-continuation-20261120` | `cca252843196edb423def4c530049d31affca10b` | 3 | Pending evidence |
| `audit/core127-final-evidence-20261120` | `c372bb94af819daeaf34b98beaf8da44e1fa3c14` | 1 | Equal tree |
| `audit/federation-compose-20261120` | `7d565ac2e3c1d7973d802b755b89a0ce267d0ac6` | 2 | Equal tree |
| `audit/pr104-historical-evidence-20261120` | `72cf3a267c72e588f37fbd31ed5b3410037f3284` | 1 | Pending evidence |
| `docs/branch-refresh-20261120` | `43e02a16c3c0eb854d34c587381ea24ab0e749bf` | 2 | Document variants |
| `fix/analytics-fuzz-qualification-20261120` | `d7edb701f6b941998deaacf43ccbc8e2a49e05d0` | 1 | Equal tree |
| `fix/core-anchor-fixture-20261120` | `dc12dac950714c66337dbd2e98edc15eab0fa640` | 4 | Document variants |
| `fix/core-credit-evidence-null-20261120` | `c1aadfd91c38b360291094fbd53949b39ebff4bb` | 1 | Preserved changes |
| `fix/core-linux-qualification-budget-20261120` | `a4becfda0f10d5fa3790bf0ee4aeff6552dfeca1` | 1 | Equal tree |
| `fix/core-password-admission-20261120` | `0bd96859034c7341015dbaf56a137f6cb84f5b55` | 1 | Document variants |
| `fix/core127-portability-20261120` | `ca62063638fa67d32d0c56e976698c1077541106` | 3 | Preserved changes |
| `fix/core127-web-ci-20261120` | `82e61847339dd08fa8d69ebf57bc0c4c6bbf2e38` | 1 | Preserved changes |
| `fix/federation-doc-qualification-20261120` | `99d6f4456260290fc3616599710b50de77585ad5` | 1 | Equal tree |
| `fix/future-messaging-path-20261120` | `882c6f39fc44c67388b1e0f1f7af5cf14384dc27` | 1 | Equal tree |
| `fix/onboarding-doc-qualification-20261120` | `02564643d01a7fa14006efd220283e476907a36c` | 1 | Equal tree |
| `integrate/core-auth-admission-20261120` | `69ca540dd9a1c01471de613e476b76babb87b31a` | 8 | Equal tree |
| `integrate/core-loading-qualification-20261120` | `bea5de25cf76b66d87689034e2529f9b40c6dff6` | 0 | History |
| `integration/future-notification-native-20261120` | `e9e94fd079a718c534945548eb7308debab0ef9b` | 2 | Equal tree |
| `main` | `aa01563fcc891da60fefebb8335eca27855678e8` | 0 | History |
| `recover/fnet207-recurring-consolidation-20261120` | `86ab145b1e3f0e02c6441d1c94faa0de2b741453` | 2 | Equal tree |
| `repair/core127-final-ci-20261120` | `e3cb6d202317406242868dbc6d5a6843b409dff3` | 7 | Equal tree |
| `review/core-containment-20261120` | `11912dfa28112d6c3bb9065478021b248f217660` | 1 | Pending evidence |
| `review/core-loading-independent-20261120` | `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef` | 0 | History |
| `review/core-loading-tests-20261120` | `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef` | 0 | History |
| `review/future-isolated-four-20261120` | `041af1e5777ba63d3e5d3d02332c7edc6cb6cce1` | 2 | Document variants |
| `review/future-ref-refresh-20261120-late` | `6eafbaad996fb846f3a71be735f57196164d9d37` | 1 | Pending evidence |

## Documentary differences and pending publication

[documentary-equivalence.json](documentary-equivalence.json) records the additional checks for the four older documentary variants. All 22 original password-admission source path/digest pairs are preserved; the published manifest converts the map to explicit rows and appends one explanation of that representation change. All original qualification notes remain in order and every other field is identical. The generated traceability file changes only its input HEAD stamp. The Core continuation report retains every original line and adds 61 lines. These are qualified content-preservation results, not byte-identical whole-document claims.

The four pending documentary branches are `audit/core-loading-continuation-20261120`, `audit/pr104-historical-evidence-20261120`, `review/core-containment-20261120`, and `review/future-ref-refresh-20261120-late`. Their 33, 10, 13, and 10 changed-path sets respectively are byte-identical in the pinned `cca2528` staging tree. These counts overlap because the staging branch includes the worker receipts. The containment worker is based on `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef`; comparing it to the repair stream avoids mislabeling inherited loading changes as new documentary-worker product work.

No local branch requires an additional runtime recovery change based on this snapshot. Keep all local heads until the coordinator publishes and verifies the pending evidence and completes the Core integration checks. The previous `future-ref-review-late-20261120` package is unchanged.

## Receipt verification

This directory contains the machine-readable inventory, qualified documentary proof, this overview, and [SHA256SUMS](SHA256SUMS). JSON parsing, count consistency, same-path/staging assertions, checksum verification, Git whitespace checks, and the repository-configured secret scanner are the applicable documentary checks. Product validation is outside this bounded review.
