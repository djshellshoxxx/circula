# PR #84 source review at 7f98b3d

Observed 2026-11-18. Scope: read-only source and Git-history review; no Go build, test, reproduction, or probe was run for this review. Work stayed in `audit-a/`, including an isolated bare repository for fetching the new commit and calculating merge trees. Root checkout and refs were not changed.

## Disposition

**Valuable new core work; preserve as an active draft. Do not classify the live branch as fully integrated or superseded merely because PR #70 merged.** The current head contains a compatible-looking retention fix and a useful API-contract regression, plus new audit evidence. No source blocker was found in those two slices. This is not a merge approval: the author explicitly has an additional clock-anomaly logging fix in progress, the draft remains open, and the latest merged head has not received independent runtime qualification in this review.

The PR body attributes the work to Claude Code session `session_01KtKj8iYFjoCXACy4iwbKai`. GitHub author is `djshellshoxxx`. This is a different session from the `session_01KtKj8n2B7N1ZkoQQdz85gS` attribution on the other active consolidation drafts. Treat it as a separate concurrent worker; no ownership handoff was inferred.

## Exact identity and ancestry

| Item | Value |
|---|---|
| PR | https://github.com/djshellshoxxx/circula/pull/84 |
| State | Open; draft true; mergeable true; mergeable_state unstable at metadata read |
| Live branch | `claude/sharp-archimedes-tu76rh` |
| Reviewed head | `7f98b3db8d032fc107599a3da4a6e1747c3e00aa` |
| Reviewed tree | `c7077653183021a006821c71926be3a089da502b` |
| Base branch / SHA | `main` / `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` |
| Merge base | Same as base SHA |
| Behind / ahead | 0 / 5 |
| Head last updated | `2026-11-18T08:27:48Z` |
| PR #70 frozen head | `1ee699f4e41c2779f32bd27063b14a4ff22e3a7d` |
| PR #70 exact merge commit | `e67bad5e2dbee95cff05536aa79f53f3afbccc84` |

All five `git merge-base --is-ancestor` checks returned 0: frozen70 → merge70; frozen70 → main; frozen70 → live84; merge70 → main; main → live84. PR #70's second merge parent is its exact frozen head. There is no evidence of rewriting or loss of that history. GitHub still reports the frozen head for merged PR #70, independently of the moving branch name.

The complete 104-branch API snapshot at `2026-11-18T08:39:54.457Z` preserves **all 89 original branch names**. Eighty-seven original heads are unchanged. The two changes are forward advances: main from `b53f757682bff5f488be9c2686c2e8aa8ad4054a` to `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a`, and sharp-archimedes from frozen70 to reviewed84. Final preservation language needs this explicit branch-head exception; “all original non-main heads unchanged” is no longer true.

## Complete change inventory

Ten files differ from the current main tree: one production file, three test files, and six documentation/evidence files.

| Files | Unique value |
|---|---|
| `cmd/circula/snapshot.go` | Retains future-dated backup entries separately from configured daily/monthly slot counts; adds `pruneBackupsAt` clock injection |
| `cmd/circula/backup_test.go` | Four deterministic retention cases; exact removed set; checks manifests, anchors, and future-file bytes |
| `cmd/circula/jobs_regress_test.go` | Exercises the normal `backupNow` path with a future file and verifies new backup integrity and retained older backup |
| `internal/api/problem_catalogue_test.go` | Checks the documented catalogue against `codeStatus` and source `apiError` literals |
| `docs/specs/07-api-contract.md` | Records four already-emitted error codes and narrows existing OpenAPI coverage gaps |
| `docs/specs/11-deployment-operations.md` | Records the future-file retention behavior while explicitly leaving anomaly logging open |
| `docs/TRACEABILITY.md` | Adds retention implementation/test citations and regenerated input header |
| `docs/audit/2026-10-28/REMEDIATION-2026-11-18.md` | New historical-baseline triage, validation claims, unresolved items, and preserved overridden verdicts |
| `docs/audit/2026-10-28/wip-fixes/README.md` | Appends one explicitly unapplied patch-reference row |
| `docs/audit/2026-10-28/wip-fixes/challenge-reason-erasure.patch.txt` | Preserves unreviewed PRIV-02 proposal as inert text only |

Four non-merge commits are all patch-unique against main (`git cherry` reports `+`); the fifth ahead commit merges current main:

- `bf494d01c50afe43216e51557828413e49b9312b`: API catalogue and regression.
- `155caabccef1afbee32fc025f9acd9b673234faa`: retention implementation and regressions.
- `8bd8e45b62b491bb4856aa5bee67dd2584c8a52e`: remediation evidence.
- `da04d4506b0951f24bba5c77e96d7fb9a64cdbfa`: validation evidence/corrected race note.
- `7f98b3db8d032fc107599a3da4a6e1747c3e00aa`: merge of current main.

No financial-rule code, ledger semantics, authorization implementation, migrations, dependency versions, workflows, or future modules are changed. The only executable production change is the prune algorithm. The privacy patch remains a `.txt` evidence file and must not be applied as part of this review.

## A3-F10: narrow the old “fixed” classification

This is a residual part of the original finding, not an independently reproduced new defect in this review. The original `docs/audit/2026-10-28/A3-circula-security-ops.md:206–215` describes three effects: schedule suppression, premature pruning of real backups, and missing clock-anomaly logs. Its original evidence already records one real backup surviving where two were configured.

Current main's `cmd/circula/serve.go:524–540` skips future-dated entries in `untilNextBackupAt`. This fixes cadence and is covered by the existing `TestJobUntilNextBackupIgnoresFutureFiles`. The narrower row in `docs/audit/CORE-COMPLETION-2026-11-16.md:30` accurately claims that scope.

Current main's `pruneBackups` still fills daily and monthly slots from the complete filename-sorted entry list. The PR's change separates entries satisfying `entry.at.After(now)`, retains them without consuming slots, and applies the unchanged count selection to the remaining entries. Exactly-now entries remain eligible. Existing past backups retained by the old algorithm remain retained by the new algorithm. No retention count or period is changed.

The author supplies fail-before and passing evidence for loss of real retention slots, including the default 30/12 case, a one-hour future name with 1/1 counts, exactly-now, multiple future files, and the scheduled backup caller. Those are author/reviewer claims preserved in the new report; they were not rerun here. The new tests' assertions match the described mechanism by source inspection.

**Still incomplete at this exact head:** anomaly logging. The existing OPS-033 requirement at `docs/specs/11-deployment-operations.md:341–350` includes logging host-clock anomalies. PR84 changes no `serve.go` code, and its body explicitly says a fixer and reviewer are preparing that follow-up.

The broad statement in `docs/security/security-review.md:78–84` that A3-F10 is fixed needs qualification. Recommended current disposition: **cadence fixed on main; retention fix under draft #84 review; clock-anomaly logging pending.** If #84 advances and its logging change is verified, update those substatuses using the actual final head and evidence.

Keeping all future files may preserve more files than configured counts; that conservative preservation is explicit in the new documentation. The author notes the associated retention-owner question as a minor review point. No general retention-policy decision or DEC-020 revision is introduced here.

## API catalogue slice

The four new rows document behavior already emitted by the API: `network-frozen` 423, `approval-required` 403, `password-change-required` 403, and `terms-acceptance-required` 409. The change adds no production API status or authority behavior. It leaves previous-release fixtures (API-078) open.

The new test scans current non-test Go sources and checks literal error constructors plus `codeStatus` against section 7.2. It fails on unsupported positional constructors, dynamic codes outside `fromErr`, or non-literal status fields. The author documents future limitations: inferred element types can escape this AST pattern, and dynamic expressions inside `fromErr` are exempt. Those are bounded coverage limitations, not evidence of a missing current runtime code. A wording nit in the password-change row should retain the existing sign-out exception from SEC-008; the PR's own review records it.

## Compatibility with reviewed #81 and #82

| Candidate | Shared changed paths | Private merge-tree result |
|---|---|---|
| #81 `09ef8a5aa25f11ee4d96f043e7d686e97598b7e3` | `cmd/circula/snapshot.go` only; #81 changes `sweepStaleTemp`, #84 changes `pruneBackups` / adds `pruneBackupsAt` | Clean; tree `ab8fe1d653705d2aba3a14dddcadcce0bc7382c5` |
| #82 `5c39abdb89faa505533e51d3878b3b119957ff32` | None | Clean; tree `75a332396acd61754d879b4a40e3a8722167daab` |

These are textual/history compatibility checks only, calculated in the private bare repository. They do not establish aggregate test success. #84 does not duplicate either reviewed implementation. Its current base includes merged #79.

## Historical evidence and policy preservation

All six files originally changed by PR #70 are byte-identical at frozen70 and current main. At reviewed84, five remain byte-identical and the sixth, `wip-fixes/README.md`, has only one added reference row; no original evidence is removed. Exact blob IDs are recorded in `PR84-ANCESTRY-AND-OVERLAP.json`.

The new triage report is useful historical evidence, but its baseline is `1ee699f` / `e67bad5`, not its latest merge head. References to “open #71–#75” and the count of one finding covered by open #75 are stale as current-state claims. Preserve the dated provenance and reconcile the live status separately; #75 is already merged. Do not overwrite the old test claims as though rerun on `7f98b3d`.

The report explicitly leaves A3-F02 / DEC-014 unresolved. PRIV-02, PRIV-03, disputed closing-network reversal policy, and member-limit notice policy stay pending. Its archived patch is not an authorized policy resolution.

## Recommended handoff

1. Record frozen PR70 and moving source branch as distinct inventory entities, preserving PR70's exact merge receipt.
2. Preserve draft84 with an active unique-core-work disposition; coordinate with its attributed worker before taking ownership.
3. Reconcile the A3-F10 substatuses and stale “open PR” statements in final consolidation evidence.
4. Re-review any new logging commit at its exact head, then run the relevant regression, aggregate, and required CI gates when the draft is ready.
5. Do not apply archived privacy proposals or resolve DEC-014.

Artifacts: `PR84-API-METADATA.json`, `PR84-ANCESTRY-AND-OVERLAP.json`, `PR84-ORIGINAL89-PRESERVATION.json`, and `pr84-full.diff`.
