# Independent final scope-preservation readback — Circula

**Verdict: PASS — no drift found.** Fresh GitHub reads confirm the published scope boundary remains intact before the coordinator's final historical reconciliation.

## Exact observation times and reference

- Ref and policy-PR reads: **2026-11-21 03:23:28 UTC through 2026-11-21 03:23:29 UTC**; completion is **2026-11-20 19:23:29 PST**.
- Pinned manifest reads: **2026-11-21 03:24:37 UTC through 2026-11-21 03:26:01 UTC**; completion is **2026-11-20 19:26:01 PST**.
- Reference: [published scope proof](https://github.com/djshellshoxxx/circula/blob/9f713dcb2157b66b4e2af7aec68a37b4a9057e6d/docs/consolidation/evidence/core-main-finalization-20261120/scope-review/proof.json) at evidence commit `9f713dcb2157b66b4e2af7aec68a37b4a9057e6d`.
- Actual main: `87651d36558a8e421e41fbd76c514045d88c74ed`.
- Actual tree: `43e65ee91a90ff70b70cc69d0b517b8367a2585f`.
- Ordered parents: `aa01563fcc891da60fefebb8335eca27855678e8`, then `3ef43ac1e05360d2a1641742dc056afe8871879b`.

The native GitHub connector read all **23 explicitly selected refs**: main, fifteen future branches and seven held policy/interface branches. Every head is present and equals the exact head in the published proof. A fresh GitHub main-commit read also matches its recorded tree and ordered parents.

## Five policy PRs remain held

All five freshly read policy PRs remain **open, draft and unmerged**. Their source heads match both their corresponding fresh branch refs and the published proof.

| PR | Fresh source head | State |
|---|---|---|
| [#83](https://github.com/djshellshoxxx/circula/pull/83) | `af54887aa0cd63ba080e2944aa52a5b759b008ca` | Open, draft, unmerged |
| [#85](https://github.com/djshellshoxxx/circula/pull/85) | `0d480a5b348f3d385df0dc9c9a3e43a0b03a8332` | Open, draft, unmerged |
| [#86](https://github.com/djshellshoxxx/circula/pull/86) | `4d349c4d03bfb81760d471a2a7ef4ac4338d9c45` | Open, draft, unmerged |
| [#94](https://github.com/djshellshoxxx/circula/pull/94) | `45b596f12876e3f91c30b996457f7096e1fffe24` | Open, draft, unmerged |
| [#107](https://github.com/djshellshoxxx/circula/pull/107) | `b8cb63bfc6d2a969d972f55937cdc2070d5eba34` | Open, draft, unmerged |

## Seven excluded branches remain outside main

Fresh ref identities are unchanged. Read-only local Git ancestry checks against those exact freshly observed heads confirm that all seven remain outside actual-main ancestry and retain their source-only commits.

| Branch | Exact preserved head | Source-only commits |
|---|---|---:|
| `audit/gov021-final-core-refresh-20261119` | `b8cb63bfc6d2a969d972f55937cdc2070d5eba34` | 11 |
| `audit/gov021-fixture-20261119` | `d922d4f880059f939aeca40cbcb77c60201ff0c7` | 9 |
| `claude/gov-021-admin-recovery` | `0d480a5b348f3d385df0dc9c9a3e43a0b03a8332` | 10 |
| `claude/aud4-f02-reviewer-reset-policy` | `af54887aa0cd63ba080e2944aa52a5b759b008ca` | 2 |
| `audit/sec026-independent-20261119` | `45b596f12876e3f91c30b996457f7096e1fffe24` | 3 |
| `claude/sec-026-abuse-signals` | `4d349c4d03bfb81760d471a2a7ef4ac4338d9c45` | 13 |
| `planning/future-cli-specs-20261110` | `8e2372c064258fb8c53554ae8f6b81957932e885` | 1 |

No new inclusion or policy choice follows from the successful Core merge. The unchanged immutable source identities preserve the prior source/path exclusion proof.

## Fifteen actual manifests remain disabled and on HOLD

For each freshly observed future head, the GitHub connector read its actual `future/<module>/feature.json` through a URL pinned to that commit. The connector returned decoded UTF-8 file bytes. This reviewer parsed each JSON object, required JSON boolean `enabled: false` and string `integration: "HOLD"`, and computed the complete Git blob preimage from the returned bytes.

**All fifteen computed Git blob identities equal the published expected identities.** Each module ID also matches. This verifies both the hold values and byte identity of the actual manifests, independently of documentary status tables.

| Module | Fresh branch head | Verified manifest Git blob |
|---|---|---|
| FNET-201 | `12850290c6bca7720853c33689662d751aced160` | `43bc991b59976c9f12a613fffbca106573f17eb0` |
| FNET-202 | `efc9d696eea2dd3260ecb6c5091802ef31e20572` | `4beec20056a98e995c5e6115b239aded2f15ea9e` |
| FNET-203 | `5200c5bf940f051a86d6e502787af06892200974` | `d8096428f13dac807f88c3b6ac5468176889f9ea` |
| FNET-204 | `108107435645db687f8ac380e6571319f419fa32` | `dea2c3e2ba2600f327cd961e03684843f55aa3ca` |
| FNET-205 | `64f1fef75d167ca322179e3e120214f9a56c6583` | `2507b08b687d64c875480af1b993cfa97f9cfa1a` |
| FNET-206 | `6aac35e226135a53d69c1e6ce74e340dcfe1d85f` | `a1fe86449b5e910833a130748e28536bb0512c85` |
| FNET-207 | `89c6aeb6b4f73292d38a4a0803146ff5b9386e6f` | `d345dbc3fa3b4146ba86020c12f8925930ee404f` |
| FNET-208 | `5eac085005361e973ca8f358e01f039aea718bed` | `44b76724eb5b0b5743e10f8565a1f3f3ca312d38` |
| FNET-209 | `0aeb5a1446798019141a3f6f5bcb1e7a24b70c47` | `547ebe612535c520a037fc3facb516159b863376` |
| FUX-101 | `3415fe24df7b87a2803552b37ca31bf898f7aa8e` | `b9328faac33485ba3cc26650e28c6cdecd587012` |
| FUX-102 | `6028dae56723b8488db6e52e25c83da7888777c1` | `4a4247d78aa8e0d0918a8e333eb2930ba77bf2f0` |
| FUX-103 | `34d0f889631e2d133149a89a276cf9c71fd261a9` | `008d52d9d4d59bf5929181eb191091cb8daacab5` |
| FUX-104 | `54a41b12a314e7652802b71d693e2594c37f172f` | `15827422569370ec231e6bfc11814c7c3991d274` |
| FUX-105 | `54d3af51aa32c022562e8041d1baa149da071db3` | `f1f385d7de5516c094dad9d3a70ea3dc30174259` |
| FUX-106 | `4c3ce1d7b7f2b4ed809669985d5e38981408494b` | `81ac3e69d0ec10ddaf599fd8fb2b6bd4b2e70b43` |

## Preserved evidence and limits

`result.json` contains every requested URL, ref and PR observation, expected and actual source heads, exact returned manifest text, parsed hold state, byte counts, SHA-256 values, computed Git blob identities, and seven ancestry rechecks. It records **zero drift**.

This readback is a timestamped observation, not a lock against later changes. The coordinator must refresh relevant heads before consequential mutations. Actual-main CI completion and historical PR closure remain separate gates; this scope review does not turn the pending macOS run into a pass or lift any policy/future-module hold.

No product suites, migrations, archived programs, workflow restarts, PR/ref mutations, automation updates, settings changes, deployments or shared-worktree edits were performed. Only this assigned intermediate review directory was written. The reviewer remains available for the coordinator's final package and closure-receipt review.
