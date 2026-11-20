# Post-snapshot branch reconciliation — 2026-11-19

## Scope and evidence

This addendum supplements the pinned [113-head inventory](BRANCH_CONSOLIDATION_MATRIX.md) and [original advertised refs](BRANCH_REFS_2026-11-19.tsv). It does not rewrite that initial snapshot or substitute later heads for earlier review evidence.

A fresh `git ls-remote --heads --tags https://github.com/djshellshoxxx/circula.git` completed at **2026-11-19 02:46:15 Pacific / 10:46:15 UTC**. It advertised **117 branches, no tags**: four new branches, four changed existing heads, and no removed starting branches. The other 109 original heads are unchanged. `main` remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.

- Original advertised-ref SHA-256: `d3bee5f890d4cfd67b9c1da5f3fbbc0ed2515dd42d2dad0447866a6bbab8468f`.
- Refreshed advertised-ref SHA-256: `f1e91890e6f7ba676699d47c20960fa3393d268499deaceb082de5f590e3991a`.
- The complete refreshed ref set is reconstructable by applying the eight rows below to the original TSV. Branch counts exclude local worktrees and PR-only refs.

PR metadata was read directly from GitHub alongside this refresh. New PRs #96–#99 are open drafts at the recorded heads. CI and later edits remain separate, moving observations; none of the source review below is a runtime test pass or permission to enable a future module.

## Four newly advertised branches

| Branch | Exact head | PR and intended target | Disposition at this snapshot |
|---|---|---|---|
| `codex/api036-cursor-hardening-20261119` | `94fcd018ac4227ea5a5eb803478b3bb0a1aa3677` | [#96](https://github.com/djshellshoxxx/circula/pull/96), into `integration/comprehensive-audit-20261119` | Core cursor hardening; assigned separate current-source review. Do not count it as already integrated. |
| `master/fnet207-recurring-remote-20261119` | `73a911a9bec387381633c031d8a4083c9fc724ad` | [#97](https://github.com/djshellshoxxx/circula/pull/97), into `future/fnet-207-recurring-20261116` | Isolated calendar milestone; source review below found no blocking defect. Current-head required checks remain a merge gate. |
| `feature/fnet-204-completion-parallel` | `949d32907f8cc2fab30ce237d95bb361023e84ba` | [#98](https://github.com/djshellshoxxx/circula/pull/98), into `audit/future-feed-cursors-20261119` | Further isolated feed work; requires complete independent review and current-head qualification. This addendum does not approve its implementation. |
| `integration/core-consolidation-20261119` | `47446b33b01f4993a1caa406516746858bbf0a11` | [#99](https://github.com/djshellshoxxx/circula/pull/99), into `main` | Published core candidate. Final operations, follow-up evidence, required native checks and eventual merge outcome belong to its current-head report. |

PR #97 advanced from the initially reported `b92a7aa4211d9d99d028572ab908469b83a4bda2` before this refresh. Review and merge qualification must use `73a911a9bec387381633c031d8a4083c9fc724ad`, which adds the native platform matrix.

## Four changed existing heads

| Branch | Original snapshot | Refreshed head | Scope / preservation |
|---|---|---|---|
| `audit/future-feed-cursors-20261119` | `8638f3d30f27f4882c4c6424d4c8b248208b6f3d` | `0fed0480d3b129351c96cc0f62bd19479acceab7` | PR #92 isolated feed path. The latest follow-up from `c4732bc95f2f1a65e642c356b04ed916b1dfa4b4` changes only the core session template and its render test; no feed module or manifest change. |
| `audit/future-pwa-browser-20261119` | `f25d037c3cc53fbf1ebbf687831300be9bdb749f` | `f153bf5b18f933308479eea7fe75c0f60d33ca38` | PR #95 remains a separate PWA topic; its current source/checks must be assessed independently. |
| `future/fux-104-notifications-20261116` | `3bdf8e92d84af2c660c615bcf956cf76752e86ae` | `797dddf21adddfb34a49e7bda1871a202968dcda` | PR #93 merged only into the notification module target. Tree equality and preserved HOLD state verified below. |
| `integration/comprehensive-audit-20261119` | `244e1356201e6477eab4694d82947f2058f3b0d8` | `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` | PR #90 now includes bounded session/token inventory and the rendered cursor-link correction. Earlier snapshot remains a provenance reference, not qualification of this new head. |

The initial remote refs and commits are retained. This reconciliation performs no remote writes, branch deletions or history rewrites.

## PR #97 independent source review

**Reviewed head:** `73a911a9bec387381633c031d8a4083c9fc724ad`.

**Base:** `0437e54897c6d9cfb3f2577807b82dd038493850`, `future/fnet-207-recurring-20261116`.

Complete PR scope is eight files: seven under `future/recurring/` and the new `.github/workflows/future-recurring.yml`. The review read the full production source, both added test files, existing tests, README, validation record, upstream notes, unchanged `SPEC-SNAPSHOT.md`, manifest and module declaration. No build or runtime test was executed during this review.

| Commit | Purpose |
|---|---|
| `640224122a8b83f26313ddbe2a5cb04b49de6808` | Add structured weekly/monthly rules and repair exact calendar/identity behavior. |
| `b92a7aa4211d9d99d028572ab908469b83a4bda2` | Add dedicated isolated-module qualification and bounded fuzzing. |
| `73a911a9bec387381633c031d8a4083c9fc724ad` | Add native Ubuntu, macOS Intel and Windows qualification. |

### Findings and compatibility

No blocking source defect was identified for consolidation into the existing isolated recurring branch, subject to successful current-head required checks. This is not approval to integrate recurring behavior into Circula Core or declare the full feature complete.

- `Template` retains its layout; `Due` remains the daily wrapper. `DueWithSchedule` is additive, and valid existing daily logical-key construction is preserved.
- Daily, weekly and monthly rules operate only on the current local date. Invalid/contradictory rules are refused. Missing monthly dates and DST gaps skip; folds choose the first exact physical occurrence. These are explicitly provisional module semantics.
- The resolver examines bounded timezone intervals, computes candidates using exact second offsets and verifies the wall-time round trip. A separate `found` flag preserves the valid zero instant. The backward interval traversal avoids reliance on stale extrapolated interval ends.
- Invalid UTF-8 identities are rejected before JSON key construction, and the host-dependent `Local` zone is rejected. No arbitrary cron parser or third-party dependency is introduced.
- The source remains a pure calculator. Membership and activation values are documented as trusted adapter snapshots; no HTTP authorization boundary, persistence, background worker, proposal acceptance or ledger write is added.
- Tests cover rule boundaries, leap/century dates, UTC/local date crossings, whole-date skips, one-hour/half-hour DST transitions, key compatibility, tenant and activation checks, invalid encoding, historical second offsets and the zero instant. The independent minute oracle has 3,456 comparisons; the fuzz target asserts exact wall-time, no-future, key and boundary properties. These are inspected assertions, not fresh execution results.
- The dedicated workflow retains pinned actions, `contents: read`, no persisted checkout credentials, bounded jobs/fuzzing and explicit disabled/HOLD/standard-library-only checks. It executes native Linux/macOS race suites and an ordinary native Windows suite; Windows is not presented as race-qualified. The workflow does not modify shared CI or root dependencies.

### Required boundaries and remaining work

`future/recurring/feature.json` still declares `integration: "HOLD"`, `enabled: false` and `calendar-core-tested-feature-incomplete`, with no dependency on another future module. The unchanged specification remains proposed, non-authoritative future planning.

The following remain unfinished and must not be implied by merging this isolated milestone:

- Durable templates and firing records; atomic reminder insertion and uniqueness under restart/concurrent workers.
- Pause, skip, edit and cancellation audit; quotas, retention and timezone-edit policy.
- Current production authorization and transaction adapters; explicitly confirmed draft handling.
- Accessible HTTP, CLI and user-interface workflows; migration/backup/restore acceptance and core adoption decisions.

Reported local coverage and fuzz results in `VALIDATION.md` remain the original author's source-pinned evidence. This review does not independently re-execute them or replace provider qualification at `73a911a9bec387381633c031d8a4083c9fc724ad`.

## PR #92 and #93 reconciliation receipts

PR #92 at `0fed0480d3b129351c96cc0f62bd19479acceab7` is two commits ahead of the previously reviewed `c4732bc95f2f1a65e642c356b04ed916b1dfa4b4`: core fix `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` and its preserving merge. The complete two-file delta is `internal/web/templates/sessions.html` (+1/-1) and `internal/web/session_templates_test.go` (+25/-2). The fix preserves the composite `millis:publicID` cursor in a literal query context; assertions require round-trip cursor equality and reject fragments or the template failsafe. Feed source, workflow, traceability and manifest are unchanged in this follow-up. Its current manifest remains disabled/HOLD and feature incomplete.

PR #93 merged at `797dddf21adddfb34a49e7bda1871a202968dcda`. GitHub Git objects show this merge and reviewed source `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` share exactly tree `d551498bd5c3c144acedf72f9320704ba55c3404`; the comparison contains zero changed files. The merge preserves original target `3bdf8e92d84af2c660c615bcf956cf76752e86ae` and reviewed source as parents. The manifest read at the merge commit retains `integration: "HOLD"`, `enabled: false` and `isolated-service-and-transports-implemented-feature-incomplete`.

These receipts establish source and preservation facts. The final integration report records execution and merge gates separately, using the exact heads that were actually qualified.

## Second snapshot and inventory freeze — 10:55:40 UTC

A final narrow `git ls-remote --heads --tags https://github.com/djshellshoxxx/circula.git` completed at **2026-11-19 10:55:40 UTC**. It advertised **120 branches and no tags**. All 117 heads from the preceding snapshot are unchanged; three branches are newly advertised, and none was removed. Relative to the original 113-head inventory, the cumulative delta is seven added branches and the same four changed heads recorded above. `main` remains `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.

| Newly advertised branch | Exact head | PR and intended target | Disposition at freeze |
|---|---|---|---|
| `audit/future-notifications-bounded-core-20261119` | `be0c071ebb3d26527d81a0c5b015effe07d5b1b2` | [#100](https://github.com/djshellshoxxx/circula/pull/100), into `future/fux-104-notifications-20261116` at `797dddf21adddfb34a49e7bda1871a202968dcda` | Open draft, created at 10:46:24 UTC. Independent future-module source review assigned separately; no merge or qualification claim in this inventory. |
| `codex/api036-completion-20261119` | `6f8ead1951f2d4b72649909b2b4cc83000948afe` | [#101](https://github.com/djshellshoxxx/circula/pull/101), into `integration/comprehensive-audit-20261119` at `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` | Open draft, created at 10:46:41 UTC. Session navigation and revocation source review assigned separately to the security worker. |
| `integration/core-recovery-20261119` | `c97233fb5b795e85b4a81e19809bb9d9c76a15d1` | No PR mapping established in this narrow refresh. | Newly observed concurrent recovery head; recorded for preservation and reported to the integration lead. Its source, dependencies and CI were not reviewed here. |

- Preceding 117-head advertised-ref SHA-256: `f1e91890e6f7ba676699d47c20960fa3393d268499deaceb082de5f590e3991a`.
- Frozen 120-head advertised-ref SHA-256: `c267f961814e49553d7845ae0715ade21ec49de613636d32a73419d01ffb0a26`.
- The complete frozen ref set is reconstructable from the original TSV, the eight changes in the first snapshot and the three added rows in this section. Original inventory artifacts remain unchanged.

### PR #96 local integration receipt

The local consolidation history now contains merge `a50f90bb120d07620159cd2d824b48e92425018b`, **“merge: integrate PR96 bounded canonical cursor decoding”**. Its parents are `a3be13a066484708bf549e079c52d32fe1159d8c` and reviewed PR #96 head `94fcd018ac4227ea5a5eb803478b3bb0a1aa3677`. This records local preservation of the cursor hardening. It does not establish that PR #96 has been merged remotely or that the resulting candidate has passed CI; current-head qualification is still pending in the integration record.

This section freezes branch discovery at the stated observation. Later branch movement, reviews, CI results and integration actions belong to separate final-operation evidence; they must not be substituted into either pinned snapshot. This narrow refresh performed no builds, runtime tests or remote mutations.

## Reviewed recovery and local integration after the frozen ref snapshot

The previously unmapped `integration/core-recovery-20261119` head is [PR #102](https://github.com/djshellshoxxx/circula/pull/102), still at `c97233fb5b795e85b4a81e19809bb9d9c76a15d1` when mapped. Security, UI and branch workers reviewed its complete compatible Core recovery. It introduces no future-module or root dependency changes.

Local integration preserves PR #96 at `a50f90bb120d07620159cd2d824b48e92425018b`, PR #101 at `f143d90c147786183d382371a83535451427e2f5`, and PR #102 at `12af2112fb968f1c72e874e3c18b66e9e2626208`. The query-plan conflict preserves six strict indexed-seek cases; four recovery conflicts preserve both message/test/documentation continuations and regenerate traceability. The catalogue union contains exactly 525 keys with no changed preexisting values. These are implementation/source-review results, not combined-head CI passes.

PR #98's complete 23-file feed addition, PR #97's 8-file recurring addition, and PR #100's notification refresh received independent source approval for their respective isolated branches, conditional on their current checks. PR #100 keeps all 36 notification module files unchanged from the verified PR #93 merge and adds only the reviewed Core refresh plus verbose vulnerability reporting. All three remain disabled/HOLD and are not included in Core.

The final publication/merge receipt belongs to PR #99. Discovery counts remain pinned to the explicit snapshots above.
