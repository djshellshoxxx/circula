# Recovered source archive: verified history and remaining reconciliation

Recorded: 2026-11-19T11:37:41.018116+00:00. This is a source recovery and comparison receipt; it does not approve an integration or replace current qualification.

## Recovery outcome

**The older source is recoverable.** A bounded lookup of the recovery record referenced by existing continuation metadata located the exact saved `Circula-Future-Recovery.zip`, version 2. Its 27,776,603 bytes match SHA-256 `a055dfb265625634c2c6bc6da491a634b0c73381dfc8fdecac02f2d9508ed083`. The source archive itself was not changed.

The archive contains 186 entries totaling 28,324,090 uncompressed bytes. Central-directory inspection found no path traversal or symlink entries. Only its Git bundle and six small metadata documents were extracted. The 27,408,215-byte bundle matches SHA-256 `fcdb13b80022e49e95cd2788f4395cb7b31e8f89a4008fc5957d55d6202c4118`. `git bundle verify` reports complete history; all **48 refs exactly match** the supplied inventory. A fresh bare clone passes `git fsck --full`. The sole notice says its default HEAD is unborn; all preserved source refs and objects are present.

The verified object store is `/workspace/scratch/f47ceab29930/evidence/branch-inventory/legacy-recovery/recovered.git`. The untouched ZIP is at the same recovery directory; its bundle and metadata are in `selected/`. The complete 48-ref map is in `bundle-heads.tsv` and [ARCHIVE_RECOVERY.json](ARCHIVE_RECOVERY.json). Recovery usage at verification was 83,106,553 bytes. No existing source file, branch, remote, automation or parent object store was modified by recovery; only the assigned audit records were updated.

## What this changes in the previous inventory

The earlier check correctly found three old workspace directories and two source objects unavailable in the then-fetched repository. That observation was a limit of the available object database, not evidence of permanent loss. Archive verification now resolves the exact source objects:

| Historical source | Recovery result | Relationship to advertised tested module |
| --- | --- | --- |
| Matching `99620acca4459e82c58ea1036af9cbff7effc47d` | Present; ancestor of archived `1a514bdff29ee3f1f0282a9833e64ab6cda51cd5` | Not an ancestor or equal module tree of `12850290c6bca7720853c33689662d751aced160` |
| Favorites `399a4ec9d2544c0bc445c3a8d7ef1b358e59d9cb` | Present; ancestor of archived `47cef5958b7fcdd5d3cbfb5878d74b38b327ce68` | Not an ancestor or equal module tree of `efc9d696eea2dd3260ecb6c5091802ef31e20572` |

The frozen 114-head inventory and 122-head delta still describe their actual advertised remote snapshots. The 48 private recovery refs are a separate population and must not be added to those remote counts. No unavailable code was reconstructed.

## Fifteen module comparisons

Comparisons use the fourteen exact source snapshots in [MODULE_QUALIFICATION.json](MODULE_QUALIFICATION.json), plus the reviewed recurring local source `cdcd2cb38b3af0275e5ceb58242384cfa916be45`, whose whole tree equals published PR 97. All fifteen archive trees differ: seven selected targets are ancestors of archived heads and eight histories diverge. The JSON includes every changed path, status, line count, tree ID and ancestry result. **Added/modified/deleted counts describe the target-to-archive difference, not a proposed patch.**

| Module | Archived current checkpoint | Compared target | Paths added / modified / deleted | History |
| --- | --- | --- | --- | --- |
| FUX-101 pwa | `de148c469bf3c5b0c15dcedd7499be946f4eb09e` | `f25d037c3cc53fbf1ebbf687831300be9bdb749f` | 20 / 21 / 1 | diverged |
| FUX-102 qr | `08a09b2bb196403b26e87de32e3b0ac167cfa75e` | `9db8a323d7dcf430d1a58a4a4d10ca3628aeb401` | 86 / 11 / 46 | diverged |
| FUX-103 discovery | `204dc2691f3a8c914ff2623b714f96e6d8b82968` | `34d0f889631e2d133149a89a276cf9c71fd261a9` | 33 / 9 / 45 | diverged |
| FUX-104 notifications | `ba454e1c14d35bdd90dca8379e890f2bca6281d7` | `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` | 78 / 8 / 25 | diverged |
| FUX-105 messaging | `5375bdf6ad5c48d2913c0581c4f8755b7c0f573a` | `7f92251cdb0d6ed5d9559730045a021a173a278b` | 57 / 5 / 0 | target is ancestor |
| FUX-106 onboarding | `d91c66e8f4285944f2c780ea21801ca819c3b5eb` | `473cab2fb45a7b67ea0001f9fdc7833ffda45eb9` | 22 / 8 / 25 | diverged |
| FNET-201 matching | `1a514bdff29ee3f1f0282a9833e64ab6cda51cd5` | `12850290c6bca7720853c33689662d751aced160` | 89 / 5 / 0 | target is ancestor |
| FNET-202 favorites | `47cef5958b7fcdd5d3cbfb5878d74b38b327ce68` | `efc9d696eea2dd3260ecb6c5091802ef31e20572` | 36 / 9 / 31 | diverged |
| FNET-203 receipts | `91529c5a3a4268ecf3a919e099ab9aca779aab2d` | `bd2fbdbd3adf77b27fa12335ff2827359423aea3` | 34 / 5 / 0 | target is ancestor |
| FNET-204 feed | `4340c712a2ef19eb5dbde32f7cd0d35b061a7ee7` | `8638f3d30f27f4882c4c6424d4c8b248208b6f3d` | 59 / 6 / 4 | diverged |
| FNET-205 analytics | `ef4bdf87d2bc915b26d84e548ea4e6a478079006` | `0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f` | 22 / 4 / 0 | target is ancestor |
| FNET-206 areas | `999dcbf12ffcf2bdcd1764393c5f470fe3c6fbc3` | `8b444b20139e3e687db494b3fe36bb3643148e5b` | 68 / 5 / 0 | target is ancestor |
| FNET-207 recurring | `a755732a02dfe0eaa884691d59a6f748f96039e7` | `cdcd2cb38b3af0275e5ceb58242384cfa916be45` | 73 / 7 / 1 | diverged |
| FNET-208 governance | `6a5af100877aa0117bb43a35eb625dab38ee67f2` | `b4f511372ec26dc63d005bb50d87c56e18fc1a88` | 40 / 4 / 0 | target is ancestor |
| FNET-209 federation | `dc59d9b8349cebefb35278bd37ae5e4f207dd833` | `1b3c92e03007e35e3f190e719a1ea78a807d4102` | 62 / 4 / 0 | target is ancestor |

The notification comparison above uses the earlier tested `9c794a5` snapshot, and federation uses `1b3c92e`. It does not establish compatibility with this pass's later PR 103 transport fixes or PR 105 signing validation. Those fixes remain reviewed work and require explicit reconciliation. Recurring similarly contains new calendar correctness repairs that an older implementation must preserve.

### Source inventory evidence

The recovered code includes real implementations for several items previously described as missing from the advertised snapshots. This inventory establishes their presence; it does not establish their correctness or acceptance completeness.

- Matching has durable preferences, dismissals and intent schemas, `Service.Suggestions`, preference mutations and an HTTP presentation surface.
- Messaging has a SQL store for sending and ordering messages, read state, blocking/reporting/moderation, retention cleanup and owner export, with HTTP surfaces.
- Areas has transactional named-area CRUD, listing assignments, sharing state, retry records and audit tables.
- Governance has durable proposal, eligibility, ballot, result, comment, dispute, idempotency and audit records.
- Receipts adds service, document export, HTTP, certificate and trust source. Analytics adds dashboard and interface source.
- The archive also has broader recurring and federation implementations; the feature author is comparing these against the newly reviewed repairs.

These observations move the corresponding backlog to **recovery, reconciliation and qualification**. They do not justify rebuilding the same services from the earlier minimal primitive. QR, discovery, favorites, onboarding and notifications have substantial additions and deletions compared with already expanded published source; selecting an entire alternate implementation without review would discard current work. Feed and PWA remain under active external ownership.

## Qualification and ownership limits

All fifteen recovered `feature.json` manifests independently retain `enabled=false`, `integration=HOLD` and empty `depends_on_future_features`. Their archived status records remain partial and retain manual, browser, host-contract, operational or policy gates. No feature was enabled, integrated, deployed or released.

The saved progress report contains dated November 16-17 tests and exact tested SHAs. Those runs were **not repeated** during this recovery inventory. The new successful checks establish archive checksum, Git object integrity, exact ref identity and source-tree differences; they are not runtime tests or current-head CI results. Historical completion percentages are not reused.

Before selecting a recovery candidate, refresh live ownership, compare the current published and active-worker paths, preserve recent fixes, review the actual source and run bounded current qualification. Keep the original archive, bundle, metadata and verified objects available. Nothing in this receipt bypasses normal review, CI, protection or approval gates.

## Fresh execution of the recovered module tests

All fifteen exact archived module heads now have passing existing Linux race, vet, formatting, checksum and source-integrity checks with their genuine original dependencies. The previous 14-of-15 result remains preserved as a dated snapshot. A single later bounded notifications retry completed the remaining race gate; no earlier failed command was changed into a pass. These results do not approve known source defects or qualify later repairs.

| Module | Exact recovered head | Latest race | Test pass events | Latest vet | Source unchanged |
|---|---|---|---:|---|---|
| analytics | `ef4bdf87d2bc915b26d84e548ea4e6a478079006` | pass | 19 | pass | True |
| matching | `1a514bdff29ee3f1f0282a9833e64ab6cda51cd5` | pass | 49 | pass | True |
| receipts | `91529c5a3a4268ecf3a919e099ab9aca779aab2d` | pass | 87 | pass | True |
| areas | `999dcbf12ffcf2bdcd1764393c5f470fe3c6fbc3` | pass | 59 | pass | True |
| governance | `6a5af100877aa0117bb43a35eb625dab38ee67f2` | pass | 49 | pass | True |
| messaging | `5375bdf6ad5c48d2913c0581c4f8755b7c0f573a` | pass | 74 | pass | True |
| federation | `dc59d9b8349cebefb35278bd37ae5e4f207dd833` | pass | 46 | pass | True |
| pwa | `de148c469bf3c5b0c15dcedd7499be946f4eb09e` | pass | 37 | pass | True |
| discovery | `204dc2691f3a8c914ff2623b714f96e6d8b82968` | pass | 57 | pass | True |
| onboarding | `d91c66e8f4285944f2c780ea21801ca819c3b5eb` | pass | 43 | pass | True |
| qr | `08a09b2bb196403b26e87de32e3b0ac167cfa75e` | pass | 58 | pass | True |
| notifications | `ba454e1c14d35bdd90dca8379e890f2bca6281d7` | pass | 58 | pass | True |
| favorites | `47cef5958b7fcdd5d3cbfb5878d74b38b327ce68` | pass | 76 | pass | True |
| recurring | `a755732a02dfe0eaa884691d59a6f748f96039e7` | pass | 42 | pass | True |
| feed | `4340c712a2ef19eb5dbde32f7cd0d35b061a7ee7` | pass | 62 | pass | True |

Test pass events include subtests and fuzz seeds; they are not a count of independent requirements or a new fuzz campaign. PWA also passed 33 Node tests and QR 20 Node tests in the preserved first attempt.

Notifications at `ba454e1c14d35bdd90dca8379e890f2bca6281d7`, module tree `be42783f6bbb7cc1fdc443cbcff15fa54e540fc3`, completed its full real-driver race suite in 176.503 seconds wall time: 35 top-level tests, 58 passing test events, no failures or skips. The library reported 71.0% statement coverage and its CLI 83.8%. All 89 archived files were verified against their Git blobs before and after. The exact original `x/sys v0.49.0`, SQLite v1.60.1 and libc v1.77.1 graph was used. The low-memory serial retry adds only linker `-ldflags=-w` to omit DWARF metadata, uses the warmed private cache and monitors disk thresholds. It neither substitutes a driver/version nor filters assertions. No resource stop or compiler diagnostic occurred.

`ARCHIVE_MODULE_QUALIFICATION-attempt3.json` and `qualification-summary-14-of-15.json` preserve the earlier complete 14-of-15 observation. Attempts 1 and 2 retain their low-resource, temporary-directory, compiler and vet failures; the attempt-3 warm matching vet and three x/sys0.48 race completions remain intact. The current aggregate appends attempt 4 and keeps every old check and historical summary. All 952 recovered source blobs and dependency manifests remain unchanged through their recorded checks.

The aggregate records exact commands, toolchain, per-command logs, exit codes, timings and source checks. Tests used Go 1.26.11, CGO enabled and package parallelism 1 with existing real SQLite imports. Only Linux/amd64 was executed. Passing archive tests do not make the separately identified recurring/notification source defects correct, qualify repaired compositions, resolve active feed ownership, or bypass disabled/HOLD, Core integration, native/provider and external/manual gates.
