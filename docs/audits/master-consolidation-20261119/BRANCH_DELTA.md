# Post-snapshot branch and PR delta

This report preserves the original **114-head / 96-PR inventory** and records a separate live observation after the code-stage publication of [#104](https://github.com/djshellshoxxx/circula/pull/104). The remote advertisement was read at **2026-11-19T11:03:38.121867+00:00**. The observed main commit remains [ead63b5ce24b](https://github.com/djshellshoxxx/circula/commit/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625).

## Observed change

| Measure | Count |
|---|---:|
| Original advertised heads | 114 |
| Advertised heads at this observation | 122 |
| Original heads still unchanged | 109 |
| Existing heads that advanced | 5 |
| Newly advertised heads | 8 |
| Original heads no longer advertised | 0 |
| Newly observed PR records, #97–#104 | 8 |
| Total known PR records, including the preserved snapshot | 104 |

The original classifications remain historical facts for the recorded SHAs: 52 integrated, 11 active-worker blocked, 17 experimental future, 5 policy blocked, and 29 duplicate or superseded. They have not been silently reassigned using newer heads. The original JSON and Markdown content hashes are retained in the accompanying delta JSON.

## Changed and new heads

| Branch | Change | Observed head | Related PRs | Disposition |
|---|---|---|---|---|
| `audit/future-feed-cursors-20261119` | advanced existing head | [0fed0480d3b1](https://github.com/djshellshoxxx/circula/commit/0fed0480d3b129351c96cc0f62bd19479acceab7) | [#92](https://github.com/djshellshoxxx/circula/pull/92) | active worker: reserved feed stream |
| `audit/future-pwa-browser-20261119` | advanced existing head | [f153bf5b18f9](https://github.com/djshellshoxxx/circula/commit/f153bf5b18f933308479eea7fe75c0f60d33ca38) | [#95](https://github.com/djshellshoxxx/circula/pull/95) | active isolated future qualification |
| `codex/api036-cursor-hardening-20261119` | advanced existing head | [94fcd018ac42](https://github.com/djshellshoxxx/circula/commit/94fcd018ac4227ea5a5eb803478b3bb0a1aa3677) | [#96](https://github.com/djshellshoxxx/circula/pull/96), [#101](https://github.com/djshellshoxxx/circula/pull/101) | superseded within active API-036 stream |
| `future/fux-104-notifications-20261116` | advanced existing head | [797dddf21add](https://github.com/djshellshoxxx/circula/commit/797dddf21adddfb34a49e7bda1871a202968dcda) | [#93](https://github.com/djshellshoxxx/circula/pull/93) | isolated target advanced after PR 93 merge |
| `integration/comprehensive-audit-20261119` | advanced existing head | [f25d10b6b408](https://github.com/djshellshoxxx/circula/commit/f25d10b6b4085b80d7de53ecbb55efffeb9d8969) | [#90](https://github.com/djshellshoxxx/circula/pull/90), [#99](https://github.com/djshellshoxxx/circula/pull/99), [#101](https://github.com/djshellshoxxx/circula/pull/101) | active worker: reserved Core and API stream |
| `audit/future-notifications-bounded-core-20261119` | newly advertised head | [be0c071ebb3d](https://github.com/djshellshoxxx/circula/commit/be0c071ebb3d26527d81a0c5b015effe07d5b1b2) | [#100](https://github.com/djshellshoxxx/circula/pull/100) | active isolated Core refresh |
| `codex/api036-completion-20261119` | newly advertised head | [baedf4b6670f](https://github.com/djshellshoxxx/circula/commit/baedf4b6670f6bf39ec593978ce38780c5640215) | [#101](https://github.com/djshellshoxxx/circula/pull/101) | active worker: reserved API-036 completion |
| `feature/fnet-204-completion-parallel` | newly advertised head | [949d32907f8c](https://github.com/djshellshoxxx/circula/commit/949d32907f8cc2fab30ce237d95bb361023e84ba) | [#98](https://github.com/djshellshoxxx/circula/pull/98) | active worker: reserved FNET-204 completion |
| `integration/circula-master-consolidation-20261119` | newly advertised head | [c3c991aa3c12](https://github.com/djshellshoxxx/circula/commit/c3c991aa3c12a11edce2874e9444b9d2a99f746c) | [#104](https://github.com/djshellshoxxx/circula/pull/104) | reviewed Core source candidate; gates blocked |
| `integration/core-consolidation-20261119` | newly advertised head | [47446b33b01f](https://github.com/djshellshoxxx/circula/commit/47446b33b01f4993a1caa406516746858bbf0a11) | [#99](https://github.com/djshellshoxxx/circula/pull/99) | active worker: reserved Core consolidation |
| `integration/core-recovery-20261119` | newly advertised head | [c97233fb5b79](https://github.com/djshellshoxxx/circula/commit/c97233fb5b795e85b4a81e19809bb9d9c76a15d1) | [#102](https://github.com/djshellshoxxx/circula/pull/102) | active worker: reserved Core recovery |
| `master/fnet207-recurring-remote-20261119` | newly advertised head | [73a911a9bec3](https://github.com/djshellshoxxx/circula/commit/73a911a9bec387381633c031d8a4083c9fc724ad) | [#97](https://github.com/djshellshoxxx/circula/pull/97) | reviewed isolated recurring candidate; gates pending |
| `master/notification-transport-tests-20261119` | newly advertised head | [13669485b641](https://github.com/djshellshoxxx/circula/commit/13669485b641475fce526633f47055ef31997098) | [#103](https://github.com/djshellshoxxx/circula/pull/103) | reviewed isolated notification candidate; gates pending |

## Ownership and integration consequences

[#90](https://github.com/djshellshoxxx/circula/pull/90), [#99](https://github.com/djshellshoxxx/circula/pull/99) and [#101](https://github.com/djshellshoxxx/circula/pull/101) remain active Core/API streams. The prior cursor-hardening [#96](https://github.com/djshellshoxxx/circula/pull/96) is **closed without merging** and has an active replacement in #101. None of these observations is a handoff or authority to cherry-pick their source. [#92](https://github.com/djshellshoxxx/circula/pull/92) and [#98](https://github.com/djshellshoxxx/circula/pull/98) remain the reserved feed stream.

[#102](https://github.com/djshellshoxxx/circula/pull/102) is the separate Core recovery covering export authorization/streaming, web pagination, catalogue work, shared CI and traceability. Its complete 56-path comparison has no overlap with the five verifier paths in #104. The shared workflow and generated traceability document remain reserved to their owners.

[#93](https://github.com/djshellshoxxx/circula/pull/93) has merged into the isolated notifications target at [797dddf21add](https://github.com/djshellshoxxx/circula/commit/797dddf21adddfb34a49e7bda1871a202968dcda); this is not a main merge. A read-only comparison confirms that target has no file differences from PR 93's former head. [#100](https://github.com/djshellshoxxx/circula/pull/100) is a separate active refresh, while [#103](https://github.com/djshellshoxxx/circula/pull/103) adds only the four reviewed transport paths. The latter's tree equals the local reviewed candidate, and #100 has no transport-path overlap.

[#97](https://github.com/djshellshoxxx/circula/pull/97) and [#103](https://github.com/djshellshoxxx/circula/pull/103) remain **disabled, incomplete and on HOLD**. Their isolated source reviews do not adopt feature policy or enable Core adapters. [#95](https://github.com/djshellshoxxx/circula/pull/95) remains another worker's PWA qualification draft.

### Historical path overlap is checked against actual blobs

The broad old-base comparisons for #92, #93 and #95 list `scripts/read-export.py` and `scripts/test_read_export.py`. Both files at their current heads or merged target are byte-for-byte equal to main `ead63b5`, as shown by exact Git blob comparisons in the JSON. These are inherited main files, not competing reader changes. The active Core PR comparisons #90, #99, #101 and #102 do not include the five verifier repair paths.

## Reviewed candidates and outstanding gates

[#104](https://github.com/djshellshoxxx/circula/pull/104) at [c3c991aa3c12](https://github.com/djshellshoxxx/circula/commit/c3c991aa3c12a11edce2874e9444b9d2a99f746c) exactly matches the reviewed five-file local candidate `6e8f5f1473084e7a14beccb92d3db9ea547554d4`. Source review found no remaining blocking issue within that diff. Its generated DAT-040 traceability mapping still requires the shared owner's handoff; required current-head CI, normal approvals and protection rules remain effective.

At the recorded code-stage heads, #97's module workflow run was queued and its Core workflow run pending; both #103 workflow runs were queued; #104's Core workflow run was queued. These are workflow-level observations, not an assertion that every individual job remains incomplete. No pending or absent result is counted as passing. See [REVIEW.md](REVIEW.md) and [REVIEW.json](REVIEW.json) for source receipts, exact blobs, observed workflow IDs and the reviewed traceability patch.

## Older continuation-workspace provenance

A separate read-only provenance check at **2026-11-19 11:23:54 UTC** examined historical continuation metadata. The three referenced parent workspaces are absent: `/workspace/scratch/3ef42c6ef5d8`, `/workspace/scratch/4f14005536ba`, and `/workspace/scratch/2395b07c0520`. They do not add live repositories to the original local ownership census.

At that observation, the exact historical matching commit `99620acca4459e82c58ea1036af9cbff7effc47d` was unavailable in the fetched local object database; the provider also returned “No commit found for SHA” for it in this repository. Its ancestry or content equivalence with advertised matching head `12850290c6bca7720853c33689662d751aced160` therefore could not be established from those checks. The historical favorites prefix `399a4ec` also did not resolve locally; no full available source object had yet been supplied for comparison with advertised favorites head `efc9d696eea2dd3260ecb6c5091802ef31e20572`.

This is a provenance limit, not proof that the current remote heads contain every piece of old unpublished work or that such work is missing. No unavailable source was reconstructed, no guessed code was published, and no existing automation was changed. The accompanying JSON retains the exact checks and provider response.

## Later source-archive recovery

A subsequent bounded check of the referenced recovery record located and verified the saved source ZIP, its complete-history Git bundle and all **48 preserved refs**. Both exact historical matching and favorites commits are now available in the verified archive. Their earlier unavailability in the fetched repository is preserved above as a dated observation, and is no longer an unresolved source-recovery limit. See [ARCHIVE_RECOVERY.md](ARCHIVE_RECOVERY.md) and [ARCHIVE_RECOVERY.json](ARCHIVE_RECOVERY.json) for exact checksums, refs, module tree comparisons and the canonical read-only object store.

All fifteen archived module trees differ from the selected tested targets. Recovered services and persistence must be reconciled before describing them as unimplemented or starting replacement work. At the initial read-only recovery observation, archived tests had not yet been rerun. The later fresh Linux qualification passed the existing suites for 14 exact archived module heads; notifications remains unqualified for race after compiler resource failures. See ARCHIVE_MODULE_QUALIFICATION.json for all attempts and source identities. Activation remains HOLD, and these private refs do not change the frozen 122 advertised-head count.

## Scope of this delta

The JSON retains all 122 advertised head names/SHAs, the 13 changed/new head records, selected refreshed PR metadata, and complete current PR path sets. Those path sets are explicitly identified by their current PR base/head; they are not mislabeled as fresh old-head-to-new-head diffs. The 109 unchanged branches were not exhaustively re-audited.

This observation precedes an evidence-only follow-up to #104. That follow-up will have a new SHA and needs its own checks while retaining the reviewed source. Refs and PR metadata can move after any observation. No merge, deletion, rewrite, release, deployment or handoff was performed by this inventory review.



## Subsequent remote readback — 2026-11-19 13:29 UTC

The original 114-head inventory and the first 122-head delta above remain frozen. A later read-only `ls-remote` observation advertises **134 heads and no tags**: 108 initial heads are unchanged, six existing heads have changed, twenty heads are new, and none of the initial heads has disappeared. Protected main remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Twelve newly fetched PR records (#105–#116) bring the known metadata set to 116. Complete refs, observation times and metadata scope are retained in the appended JSON snapshot.

### Independently verified recovery publications

| PR | Published head | Sole parent | Full tree / changed paths |
|---|---|---|---|
| [#115](https://github.com/djshellshoxxx/circula/pull/115) areas | `684ca5132690e24104aa389aac16870c18bbc7fd` | `8b444b20139e3e687db494b3fe36bb3643148e5b` | `f5de54ced2f6fd363372a4588c292ab900eeef65` / 104 |
| [#114](https://github.com/djshellshoxxx/circula/pull/114) governance | `fbd92a22f06af7bcd96957397958b33e2706807a` | `b4f511372ec26dc63d005bb50d87c56e18fc1a88` | `36ca2f142be776095a97d51cf1caee15d04926e6` / 73 |
| [#116](https://github.com/djshellshoxxx/circula/pull/116) messaging | `62623635d9711fbafc9b9660ac8860f1eb916dbb` | `7f92251cdb0d6ed5d9559730045a021a173a278b` | `951e68d1316d0efd8503408d15a3fc555dd4848e` / 90 |

All three are open drafts targeting their unchanged original module branches. Every remote complete tree equals its frozen local candidate, and every remote changed-path set equals the reviewed local set. Their only paths outside the owned module are its dedicated new workflow and the two authorized exact accepted-main baseline files. This confirms source identity; current-head provider checks and normal approval remain separate.

### Refreshed active ownership

External draft [PR #112](https://github.com/djshellshoxxx/circula/pull/112), head `08d4b0078e8c308ae7f353849783913f4302882a`, targets PR #109’s `feature/core-catalogue-continuation-20261119` at `5db3efc084e6a807bb4702c09d3235490d42b71d`. Its complete 84-path set includes Core administration handlers, shared messages/templates, session/render/refusal code, `docs/TRACEABILITY.md` and consolidation evidence. Those paths are actively reserved. No unfinished UI/admin source is copied, and its traceability change does not constitute a handoff for our pending DAT-040 refresh.

| PR | Current head | State and reservation |
|---|---|---|
| [#90](https://github.com/djshellshoxxx/circula/pull/90) | `839e86935767288e594914ad711486616a604eca` | Draft Core consolidation; API/auth/shared paths reserved. |
| [#92](https://github.com/djshellshoxxx/circula/pull/92) | `db80ed34af09ad6b28e533da6d4b82b8250f3711` | Draft feed work; topic merge of #98 is not release qualification or handoff. |
| [#94](https://github.com/djshellshoxxx/circula/pull/94) | `45b596f12876e3f91c30b996457f7096e1fffe24` | Draft independent SEC-026 work targeting #90; reserved. |
| [#95](https://github.com/djshellshoxxx/circula/pull/95) | `0b0d70624591941c0a71a225b7390903dc4ab97c` | Draft PWA/browser work; reserved. |
| [#96](https://github.com/djshellshoxxx/circula/pull/96) | `94fcd018ac4227ea5a5eb803478b3bb0a1aa3677` | Closed without merge; source remains preserved, no independent eligibility inferred. |
| [#99](https://github.com/djshellshoxxx/circula/pull/99) | `0a73ee42b96255601cf2227b1ddf6599ec07a0ef` | Draft Core consolidation; reserved. |
| [#100](https://github.com/djshellshoxxx/circula/pull/100) | `98852afbd1cd2e31a38d6405b6eee36ad3d7e737` | Draft notification/Core refresh; reserved. |
| [#101](https://github.com/djshellshoxxx/circula/pull/101) | `f7ac6d22a1bd1718f6f4e3a840da7a28c83d0cb0` | Merged into the #90 topic; no Core release/handoff inferred. |
| [#102](https://github.com/djshellshoxxx/circula/pull/102) | `f29ebb01ff681a1276f385b8b568e41355b83505` | Draft Core/shared catalogue recovery; reserved. |
| [#106](https://github.com/djshellshoxxx/circula/pull/106) | `d55554408ef591371768259905c70ca58d68facb` | Draft federation README change targeting #105; README reserved. |
| [#107](https://github.com/djshellshoxxx/circula/pull/107) | `b8cb63bfc6d2a969d972f55937cdc2070d5eba34` | Draft GOV-021/Core refresh; unrelated to isolated advisory governance. |
| [#109](https://github.com/djshellshoxxx/circula/pull/109) | `5db3efc084e6a807bb4702c09d3235490d42b71d` | Draft Core catalogue continuation targeting #90; reserved. |
| [#110](https://github.com/djshellshoxxx/circula/pull/110) | `aeb70db0255d482eb45aa6e0515f09fc32518d81` | Open non-draft CI repair targeting #97; normal checks/approval and handoff remain required. |
| [#111](https://github.com/djshellshoxxx/circula/pull/111) | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` | Draft API-036 qualification targeting #90; reserved. |
| [#112](https://github.com/djshellshoxxx/circula/pull/112) | `08d4b0078e8c308ae7f353849783913f4302882a` | Draft Core administration/UI continuation targeting #109; reserved. |

Metadata also records analytics #108 at `38b9bcd0e049d97167d793baba933d5e79d8030d`, receipts #113 at `3cc55f7966a6321c9fa4e587596ea7daf0f9d152`, and the previously verified Core checkpoint #104 at `707a4363a86922d152ed2d203c9ea184fe4b323f`. This snapshot does not re-audit those complete source deltas or declare their provider gates successful. Later stacked recurring/federation publication belongs to a later observation.


## Archive qualification follow-up — 2026-11-19 13:31 UTC

The remaining unchanged-archive notification race suite passed on one bounded serial warm-cache retry at exact `ba454e1c14d35bdd90dca8379e890f2bca6281d7`: 35 top-level tests/58 pass events, no failures/skips, all 89 original files unchanged. All fifteen archived module heads now have passing existing Linux suites and associated recorded vet/format/checksum checks. The earlier 14-of-15 aggregate and every killed/interrupted attempt remain preserved; this adds a new terminal result. It does not qualify known source defects, repaired compositions, provider/native jobs, full feature completion or Core activation. See [ARCHIVE_RECOVERY.md](ARCHIVE_RECOVERY.md) and [ARCHIVE_MODULE_QUALIFICATION.json](ARCHIVE_MODULE_QUALIFICATION.json).


## Stacked recovery publication readback — 2026-11-19 13:38 UTC

A later independent remote snapshot advertises **136 heads, no tags and 118 known PR records**. All 134 previously advertised heads remain unchanged, and the only two new refs are the owned stacked recovery branches below. Relative to the initial 114-head snapshot, 108 heads are unchanged, six existing heads have changed, 22 are new, and none has disappeared. Main remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. The earlier114/122/134 observations remain preserved.

| Draft PR | Published head | Sole parent / target | Complete tree / changed paths |
|---|---|---|---|
| [#117](https://github.com/djshellshoxxx/circula/pull/117) federation | `f50aaa4b599c2c0272d37cdc910f567b76c451f8` | `059b0e4508a6c4f8e29f6a4cccd06f92b14ea115` / `master/fnet209-utf8-20261119` | `0873c4516d633cc56304c6d35a54fe1e812ab71d` / 78 |
| [#118](https://github.com/djshellshoxxx/circula/pull/118) recurring | `035dd1c0d7c3de901cb1d2e1854ac4fde4cbdad4` | `73a911a9bec387381633c031d8a4083c9fc724ad` / `master/fnet207-recurring-remote-20261119` | `d41e47b2a4961983f8881e5162096c588973158c` / 88 |

The complete remote trees and changed-path sets equal the final local recurring `2389be3631f632c037f85fe23a691edd6475d630` and federation `879676cf7ba143f8bb33647cb6221653b7bac70d` candidates. Their runtime, tests and dependency files remain identical to tested `e79cd7a3` and `ca9af754` respectively. Later manifest changes affect only descriptive status/audit/qualification fields; ID, disabled/HOLD, dependency boundaries, scope and remaining gates are unchanged. Each owned native workflow explicitly permits its existing primitive-PR source branch as the stacked base.

External #110 and #106 source refs remain unchanged. Their CI-repair/README ownership and normal handoff gates are not bypassed. No source-target merge, Core activation, release or successful current-head provider result follows from this readback.
