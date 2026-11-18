# PR consolidation ledger (2026-11-17 assignment, executed 2026-11-18)

Authoritative record of the disposition of every Circula pull request. Branch-level ancestry and patch
equivalence are in [BRANCH-CONSOLIDATION-2026-11-17.md](BRANCH-CONSOLIDATION-2026-11-17.md).

- Repository: `djshellshoxxx/circula`.
- Starting baseline: `main@b53f757682bff5f488be9c2686c2e8aa8ad4054a` (merge of PR #68). `main` did not move during
  the work (re-checked before the PR was opened).
- Integration branch: `claude/circula-pr-consolidation-q6q18n`, created from that SHA. The ending SHA and CI run
  are recorded in the consolidation pull request.
- Inventory: 68 PRs (19 open, 49 closed) and 88 remote branches, re-queried through the GitHub API and
  `git fetch --all --prune` plus `refs/pull/*/head`.
- Method: `git merge-base --is-ancestor`, `git cherry -v origin/main <head>` (patch IDs), author-date and subject
  matching, then a **content** check of every non-equivalent commit against `main` (file identity, test and
  symbol names, behaviour). Ancestry alone was not treated as proof. Squash-merged PRs were checked by
  diffing their touched files against `main` and by reverting each fix on a scratch copy.

> **Clone pitfall.** The session clone was shallow (grafted roots `2e323a1`, `e41d48f`), so every pre-November
> PR first appeared to have 100+ unique commits. `git fetch --unshallow` restored full history. Always check
> `git rev-parse --is-shallow-repository` before ancestry work.

## Baseline verification (`b53f757`, Go 1.26.11, Linux)

| Check | Result |
|---|---|
| `go mod verify` | all modules verified |
| `go vet ./...` | clean |
| `go test -count=1 ./...` | 14/14 packages ok |
| `go test -race -count=1 -timeout=40m ./...` (CGO on) | 14/14 packages ok |
| golangci-lint | not run locally: the installed v2.5.0 is built with go1.25 and refuses this module; CI pins v2.15.0 |
| govulncheck, e2e, native Windows/macOS | left to CI |

## Disposition summary

| Measure | Count |
|---|---|
| PRs examined | 68 |
| Starting open PRs | 19 |
| Already fully integrated (head is an ancestor of `main`) | 38, all closed except #32 |
| Closed PRs squash-merged or superseded, content verified on `main` | #45, #48, #49, #50, #51, #52, #53, #54, #63, #68 (and #40, #66 merged into other branches) |
| Open PRs whose unique work is already on `main` (close as superseded) | 11: #3, #5, #7, #30, #32, #34, #35, #36, #37, #38, #39 |
| Open PRs whose unique work was recovered or archived by this consolidation (close after) | 7: #1, #2, #13, #16, #20, #29, #42 |
| Open PRs whose unique code was ported into core | #35 (ebb4e298, migrate-mode gate) |
| Open PRs kept open | 1: #67 (future module FNET-204, incomplete, owner decisions) |
| Starting open PRs resolved | 18 of 19 = 94.7 % |

"Resolved" means the PR has a verified disposition and is closed or will be closed with a pointer to the
replacement. It is PR cleanup only; it says nothing about feature completion (see `docs/TODO.md`).

## Recovered into core by this consolidation

| Commit | Source | Change | Tests |
|---|---|---|---|
| `cb79676` | PR #35, `claude/cool-ramanujan-n2peag@ebb4e298` | Extract the existing start-up condition into `startupMigrationGate` (no behaviour change); pin OPS-025 for new, current, pending and empty-file databases in both modes; correct the `migrate-mode` help text, `docs/ops/configuration.md` and `deploy/compose.yaml`, which said `migrate` must run before the first start | `TestStartupMigrationGate` (8 cases) fails if the `!state.Exists` guard is removed; `go test ./cmd/circula/` ok |
| `91a26f2` | PR #42 audit notes (A1-04, A3-F03); fixes from #51/#52 | Service entry-point regressions: reviewer `ResolveDispute` cannot post a reversal while an administrator is required; administrators without MFA cannot start or resolve reviews under `require-mfa-admins` | `TestResolveDisputeRefusesReversalWhenAdministratorRequired`, `TestDeciderEntryPointsRequireAdministratorMFA`; each fails when its fix is removed |

Archived (documentation only, each file carries a provenance banner):

| Commit | Source | Destination |
|---|---|---|
| `ec44a2c` | #16 `127943ba`, #20 `3d9c5619`, #13 `001f09aa` | `docs/audit/VALIDATED-CONSOLIDATION-2026-10-30.md`, `docs/handoffs/archive/2026-11-09-pr20-chatgpt-zero-codex-continuation-spec.md`, `docs/qa/evidence/ci-40103567118-dual-control.md` |
| `a2b9680`, `c327f0b`, `78a9372` | #1 `22e9202f`, #2 `01b6e5c7`, #29 `2da1a293`, #38 `d2c1c609`, `planning/future-cli-specs-20261110@8e2372c0` | `docs/archive/specs-historical/` (index, `pr1/`, `pr2/`, `pr29/`), `docs/research/PRIOR-ART-MUTUAL-CREDIT.md`, `docs/research/REGULATORY-PRIVACY-QUESTIONS.md` |
| `dca5ea9` | #42 `36a80335` | `docs/audits/CHATGPT_SECURITY_AUDIT_HANDOFF_2026-11-15.md`; status of each finding in `docs/security/security-review.md` |
| `3be5b7f` | future branches | Module branch/status table in `docs/future-features/README.md` |

## Open PRs at the start (19)

| PR | Source SHA | Unique work (content check) | Integrated? | Tests | Disposition | Evidence |
|---|---|---|---|---|---|---|
| [#1](https://github.com/djshellshoxxx/circula/pull/1) | `22e9202f` | 31 commits: short spec drafts, crypto/business brainstorms, early wiki | Specs superseded by `docs/specs/`; brainstorms archived | none (docs) | Archived, close | `docs/archive/specs-historical/pr1/`; crypto/tokenization ideas are out of core scope; IDs collide with canonical specs |
| [#2](https://github.com/djshellshoxxx/circula/pull/2) | `01b6e5c7` | 13 commits: 20 spec drafts, research notes, plans, Python scripts | Unique docs 15, 16, 17, 19 archived; two research notes moved to `docs/research/` | none (docs) | Archived, close | `docs/archive/specs-historical/README.md` lists the unique items (race matrix R1–R18, cross-parameter policy rules) and conflicts (DEC-054, SEC-011) |
| [#3](https://github.com/djshellshoxxx/circula/pull/3) | `d96f2e75` | Empty spec shells | Nothing unique | none | Close, superseded | No requirements in the files |
| [#5](https://github.com/djshellshoxxx/circula/pull/5) | `db782649` | README/wiki stub saying "no code yet" | Obsolete | none | Close, superseded | `README.md` on `main` describes the built product |
| [#7](https://github.com/djshellshoxxx/circula/pull/7) | `fcd9fc58` | 1 commit: `docs/HANDOFF.md` of 2026-10-22; its code reached `main` through `claude/circula-build` (#4) | Yes (code); handoff superseded | CI 16/18 passed, 2 cancelled (retired macOS 14 runner) | Close, superseded | `docs/audit/COMPLETENESS.md`; base `claude/circula-build` is an ancestor of `main` |
| [#13](https://github.com/djshellshoxxx/circula/pull/13) | `3d0080a9` | 6 commits: C-09 ETag regression, e2e fixture, handoffs | C-09 test on `main` (`internal/api/more_test.go` `TestPendingCurrencyNamePolicyPatchChangesETag`, ported as `addc229`/`2cd1ec8`); fixture superseded by the A3-01 rewrite of `e2e/tests/security.spec.js`; CI diagnosis archived | C-09 test passes on `main` | Archived, close | `ec44a2c`; `docs/audit/C-09-POLICY-ETAG-2026-10-25.md` |
| [#16](https://github.com/djshellshoxxx/circula/pull/16) | `127943ba` | 1 commit: handoff, CI receipt, prompt | Receipt archived (only source of CI run ids 41047232276/41047238538); handoff already under `docs/audit/HANDOFF-*-2026-10-30.md`; prompt obsolete | n/a | Archived, close | `ec44a2c` |
| [#20](https://github.com/djshellshoxxx/circula/pull/20) | `3d9c5619` | Agent-lane process spec | Archived (cited by `docs/handoffs/CHATGPT_*_LATEST.md`) | n/a | Archived, close | `ec44a2c` |
| [#29](https://github.com/djshellshoxxx/circula/pull/29) | `2da1a293` | 22 commits: full competing rewrite of 14 specs + `tools/check_specs.py` | Full Markdown set archived; not merged into `docs/specs/` (ID collisions; five rules contradict DEC-035, DEC-039, DEC-040, DEC-050, FR-020/025) | none (docs) | Archived, close; unique items listed as owner decisions | `docs/archive/specs-historical/pr29/` and index |
| [#30](https://github.com/djshellshoxxx/circula/pull/30) | `1d165989` | 3 commits; 2 patch-equivalent; `0a33e3cc` HTTP acceptance test | Yes: `cmd/circula/acceptance_http_test.go` is byte-identical on `main` (via #65) | `TestAcceptanceScenarioHTTP` passes on `main` | Close, superseded | [report row GP](#code-line-detail) |
| [#32](https://github.com/djshellshoxxx/circula/pull/32) | `f4a051c1` | 15 future-module specs + research | Yes: head is an ancestor of `main`; `git diff f4a051c1 origin/main` on its 18 paths shows only two added links in the README | n/a | Close, fully integrated | Also recovered by #64 (`3a776e6`) |
| [#34](https://github.com/djshellshoxxx/circula/pull/34) | `02f9f0a7` | 10 non-equivalent commits on `haiku/circula-consolidation` | Yes: H05 tests, UX-021 reason-code test, reversal >500 pagination, CI runner/push changes are all on `main` | `TestReversalsOfResultLimit`, H05 suite pass on `main` | Close, superseded | [Code-line detail](#code-line-detail) |
| [#35](https://github.com/djshellshoxxx/circula/pull/35) | `c033cfa4` | 30 non-equivalent commits | 26 present on `main` (often stronger), 1 partial (ported, `cb79676`), 3 QA-042 Python drift-audit commits and 1 CI cache key obsolete, docs historical | Evidence tests pass on `main`; `TestStartupMigrationGate` added | Close, superseded after `cb79676` | Must not be merged as is: older action pins, `export-dir` conflicts with `export-spool-dir`, duplicate tests under other names |
| [#36](https://github.com/djshellshoxxx/circula/pull/36) | `8f742877` | 1 doc qualifying cross-branch deltas | All items it lists are on `main` (`reversal_links.go`, `ux017_reversal_links_test.go`, `persistence_regression_test.go`) | CI 7/8 passed, 1 skipped | Close, superseded | |
| [#37](https://github.com/djshellshoxxx/circula/pull/37) | `3bfb2f78` | Continuation prompt for specs | Obsolete premise (prompt-only `main`) | n/a | Close, superseded | `docs/specs/` has 26 files |
| [#38](https://github.com/djshellshoxxx/circula/pull/38) | `d2c1c609` | 15 commits: condensed 13-spec set | Covered by canonical specs; one unique item (GOV-032 no-eligible-reviewer behaviour) recorded for DEC-072 | none (docs) | Close, superseded | Archive index |
| [#39](https://github.com/djshellshoxxx/circula/pull/39) | `92d204dc` | Prompt: finish features and consolidate PRs | Its ground truth is stale; its work items were done by #56, #57, #61, #68 and this consolidation | n/a | Close, superseded | |
| [#42](https://github.com/djshellshoxxx/circula/pull/42) | `36a80335` | #35's commits plus 5 audit-note commits | Audit notes archived; every finding re-checked on `main`; 2 entry-point regressions added | `91a26f2` | Archived, close | `docs/security/security-review.md#consolidation-re-check-2026-11-18` |
| [#67](https://github.com/djshellshoxxx/circula/pull/67) | `7a528c61` | FNET-204 signed tenant-bound feed cursors (future module, base `future/fnet-204-feed-20261116`) | Not into core, by design | Module tests pass at `pr/67` (`go test -race`, 84.5 % coverage); PR CI failed 5 jobs at `7a528c61`; `gofmt -l` flags both new files | **Keep open** | See [PR #67 review](#pr-67-review) |

## Closed PRs (49)

| PRs | State | Verification |
|---|---|---|
| #4, #6, #8, #9, #10, #11, #12, #14, #15, #17, #18, #19, #21–#28, #31, #33, #41, #43, #44, #46, #47, #55–#62, #64, #65 | Head is an ancestor of `main` | `git merge-base --is-ancestor` |
| #45, #51, #52, #53, #54 | Squash-merged security fixes | Each fix is on `main` and its test fails when the fix is reverted. #53's query was later replaced by #56 (`TestNotificationErasureUsesResourceRelations`) |
| #48, #49, #50 | Closed, superseded by #68 | `main` has `golang.org/x/text v0.44.0`, `modernc.org/sqlite v1.62.0`, `golang.org/x/crypto v0.59.0` |
| #63 | Closed, superseded by #68 | Its three files are identical on `main` |
| #68 | Squash-merged as `b53f757` | Head tree equals `main`'s tree for its nine files |
| #40 | Merged into `claude/cool-ramanujan-n2peag` | QA-042 Python drift audit, superseded by `scripts/traceability` and `make traceability-check` (its 41 "missing" IDs are ASM/RSK rows the generator deliberately excludes) |
| #66 | Merged into `future/fnet-202-favorites-20261116` | Head `372a8f97` is an ancestor of that branch; future-only |

## Code-line detail

Content check of every non-equivalent commit on `claude/cool-ramanujan-n2peag` (CR, #35),
`haiku/circula-consolidation` (HK), `chatgpt/circula-consolidation-review-20261111` (CG, #34) and
`claude/gracious-ptolemy-dscc7v` (GP, #30). Merge commits on these branches carry no merge-only hunks
(`git diff-tree --cc`).

| Commit | Subject | Class | Evidence on `main` |
|---|---|---|---|
| CR `136e4169`, `5d007a9b`; HK `f8ce7494` | H05 persistence regression tests | present | `internal/store/persistence_regression_test.go`, byte-identical to CR head (via #65) |
| CR `81259994`; HK `f83ae1c9` | Removal above dual-control threshold sends a valid reason | present | `internal/web/ux021_refusal_test.go` `TestRemoveAboveDualControlThresholdRefusesOnForm` |
| CR `6631d8b0`, `888b4f23`; CG `fbdcfc11`, `d8e4e0ba` | Reversal history beyond 500 records | present | `internal/service/reversal_links.go` `ReversalsOf` keyset paging; `TestReversalsOfResultLimit` (stronger: checks IDs and order) |
| CR `41683b4f`; GP `0a33e3cc` | HTTP-level acceptance scenario | present | `cmd/circula/acceptance_http_test.go`, byte-identical |
| CR `13beb485` | BR-06 bounded in-flight export bytes, spool directory | present, superseded by a stronger design | `internal/api/jobs.go` `exportTicket.reserve`, `export-spool-dir`, `cmd/circula/exportspool.go`; no in-memory fallback |
| CR `ef1815af` | BR-04 erasure drops network export archives | present | `internal/api/jobs.go` `dropErased`; `TestErasureDropsNetworkExports` |
| CR `e742cf90` | Arch guards adapted to migration 0016 | present | `internal/store/expand_contract_test.go` `TestContractScannerForms` |
| CR `763cbbf9` | SEC-020 structured redacting log | present (main is a superset) | `internal/logx`; `cmd/circula/log_safety_test.go` |
| CR `b50f7041` | Classify dispute-deadline columns | present | `docs/security/personal-data-inventory.md` |
| CR `73726ed4` | QA-019 expiry notifies both parties | present | `internal/service/transactions.go` `ExpireProposals`; `qa019_*_test.go` in service, api, web |
| CR `25db3295`, `58e061e9`, `c977d885`, `c4b8b0fd`, `71739f1d`, `2d50e8f6` | QA-007, QA-010, QA-028, QA-026, QA-001, BR-05 tests | present under other names | `TestQA007OversubscriptionInterleavings`, `TestQA010CrashDuringPosting`, `TestQA028BackupUnderLoadAndRestore`, `TestQA026UpgradeWithFailedMigrationAndRollback`, `TestQA001InvariantSequences` + `qa001-manual.yml`, `TestLoginRefusalsDoEqualWork` (deterministic, replaces wall-clock timing) |
| CR `c4d42766`; HK `3c3ebebe`, `df895628` | CI: race timeout, macOS 15 Intel, push on `main` only | present | `.github/workflows/ci.yml` |
| CR `ebb4e298` | Pin migrate-mode rule for a new database | **partial → ported** | `cb79676` |
| CR `66df03d3` | Key Go caches on `go.sum` | obsolete | `actions/setup-go` with `cache: true` already keys on the root `go.sum` |
| CR `513b978c`, `a4fd31b2`, `7fd7b14e` | QA-042 Python drift audit and CI step | obsolete | Superseded by the Go generator `scripts/traceability` |
| CR `d0fcd404`, `3bfcfe3d`, `07b1bbc5`, `45ad179a`, `fe832fac`, `c033cfa4`; CG `d900316f`, `026c6eee`, `1a06ae41`, `02f9f0a7` | Prompts, handoffs, status rows, qualification notes | docs-only historical | Superseded by current handoff, generated traceability and `security-review.md`; branches preserved |

Counts (41 distinct commits): 26 present, 1 partial (ported), 4 obsolete, 10 historical docs, 0 valuable work missing.

## PR #67 review

Scope: `future/feed/cursor_auth.go` (+48) and `cursor_auth_test.go` (+14) on top of `future/fnet-204-feed-20261116`.
It is isolated from core and is not merged by this consolidation.

- Good: HMAC-SHA256 with `hmac.Equal`, 32-byte minimum key, network bound inside the signed payload,
  versioned payload, restricted kinds, uniform error; format mirrors `internal/api/cursor.go`.
- Medium: no expiry (the `t` field is the row time, not the issue time); no key ID or rotation; no actor, route or
  filter binding (`internal/api/cursor.go` binds path, network, person and filter), so a cursor works for any
  member of the same network.
- Low: no domain separation from core cursor MACs; `Seal`/`Open` are not wired into `Page()` and there is no
  round-trip test; the single tamper test may fail on signature length rather than the MAC; `gofmt -l` flags both
  files; network ID readable in the token; module README and VALIDATION.md not updated.

| Outstanding item | Can be done without an owner decision? |
|---|---|
| Wire into `Page()`, reuse the core cursor pattern, round-trip and tamper tests, gofmt | Yes |
| Expiry mechanism | Mechanism yes; TTL value is an owner choice |
| Key ID and verify-set interface for rotation | Yes (interface); key custody, persistence and cadence need the owner |
| Actor/filter binding | Yes |
| Persistent source joins | Module-local SQL draft only; a core schema/adapter needs owner approval |
| Revocation and mute behaviour | Needs owner policy (mute/follow semantics, retention, moderation) |
| Browser and CLI regression coverage | Module-local harness yes |

This session may push only to `claude/circula-pr-consolidation-q6q18n`, so no work was pushed to the PR #67
branch. The items marked "yes" are the next engineering step for that module, on its own branch.

## Remaining owner decisions raised by this consolidation

1. Specification drafts (`docs/archive/specs-historical/README.md`): DEC-035/039/040/050 versus the #29
   contrary rules; public directory; positive balance cap, large-amount confirmation, member blocking,
   email-change verification, own sign-in activity, organizations as members; configurable administrator quorum
   (SEC-011); banned-terms copy lint versus DEC-054; value-capture principles; CLI exit codes and write paths;
   no-eligible-reviewer behaviour (DEC-072); legal gates H/S/P.
2. Security: A3-F02 break-glass scope (DEC-014); SEC-011 scope (SA2-12); PRIV-01/02 retention (SEC-014/018);
   WAL checkpoint design for PRIV-03.
3. FNET-204 (PR #67): signing-key custody and rotation cadence, cursor TTL, mute/follow semantics, core schema for
   persistent source joins.

## Rules followed

No branch was deleted, no history rewritten, nothing force-pushed, nothing merged into `main`, no release or
deployment. Closed PRs can be reopened; their branches and `refs/pull/<N>/head` remain.
