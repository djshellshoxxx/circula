# PR #78 follow-up: source readiness and focused correction

## Result

Reviewed remote head `bcacd848dea66cac8b7909aa1e9e9dac7fdf84e4`, tree `1408a8f38b20edb29b1201b6d32284b4258b61b7`, against the earlier `1d84c6d` review. The author's update resolved most requested corrections and preserved all eight useful audit documents. Five remaining documentation corrections are committed locally for the coordinator's review and publication.

- Local commit: `6b0fb7816faec6a7c93fa7e483ee6805ce54ebf1`.
- Tree: `3a50bdaa564ac2a3d80ffcf35c59227ee9cf3176`.
- Sole parent: `bcacd848dea66cac8b7909aa1e9e9dac7fdf84e4`.
- Branch: `integration/audit78-accuracy-20261118`.
- Worktree: `/workspace/scratch/9ceb45b04668/audit-b-pr78-docs`; clean.

**The corrected candidate has no remaining required factual correction identified in this review.** Its publication requires a fresh PR #78 head check and the coordinator's independent review. Existing CI and merge-order gates remain in force.

## What changed

1. **CI commands:** C5 now distinguishes Linux race/20m, macOS race/25m with `-p 2`, and Windows `go test -count=1 ./...` without an explicit Go timeout or race detector. The author-local `40m` override and Makefile default remain separate.
2. **Completed PR #81 evidence:** AUD4-D01/D02 now name merge `b439807`, reviewed source `09ef8a5`, and run `44264463917` with nine successful required jobs. D02 explicitly retains the privileged foreign-owner test's lack of independent/CI execution; its author-reported UID0 result is not promoted into retained proof.
3. **Historical versus live refs:** the ledger explicitly separates main's normal advance, quirky-babbage's `af69702` live head versus frozen PR #39 `92d204dc`, and the reused70/84 branch. Parsing all 88 historical branch pins gives exactly those three differences and 85 matching tips. The quirky-babbage implementation remains superseded; no reopening or branch deletion is proposed.
4. **MFA follow-up chronology:** C2 and the evidence records now identify the subsequently reproduced stale-password proof defect and published repair `e127acc9877edebeca818b75a5d48ce12511eca8`, tree `84ea230a25cf30a56f4f43b1a401abea6dc1b7fb`. They accurately attribute 12 fail-before / 12 pass-after and 21 passing focused race top-level tests, plus independent review `5954151240`. PR #82 remains unmerged and its current-head CI/combined qualification is pending at this handoff. The additional finding does not change the original 55-finding denominator or DEC-014 policy.
5. **Qualification scope and merge attribution:** the manifest now gives local qualification per row. PR #77's diff review plus CI is not described as a separate local execution. The recorded external merge actions are distinguished from observations made by the parallel integration coordinator.

Exactly these five files changed, all under `docs/audit/consolidation-completion-2026-11/`: `CODE-AUDIT.md`, `CONSOLIDATION-LEDGER.md`, `MERGE-MANIFEST.md`, `SECURITY-AUDIT.md`, `TEST-EVIDENCE.md`.

## Author corrections retained

AUD4-F03 is now partial in all relevant reports, with both the independently reproduced same-millisecond and host-clock-rollback residuals. Durable binding/backfill remains outstanding. GOV-021 / SEC-026 links target their held source commits. The feature matrix credits existing Linux checkpoint-write/restart/replay coverage while retaining broader storage-fault gaps. The privileged-test attribution states the missing retained log and independent/CI execution.

`BASELINE.md` is byte-identical to `bcacd848`: the author-session 96-branch snapshot is preserved. The 105-remote-branch count remains explicitly 09:35Z. PR #88's later creation does not invalidate that timestamped inventory and belongs to the coordinator/C's separate review.

C7's new specific PR #83 citation is **supported**, so it is unchanged. F verified run `44265609316`, job `134669246743`, source `cb95bb1894cd3b178f99ff3c54e0b04651d68cff`: the job executed 09:07:39–09:08:18 UTC and failed the traceability-freshness step. The retained diagnostic names the stale DOM-038 row and `make traceability`. The earlier 08:39 queued snapshot did not describe this later terminal result. This verification does not approve the held reviewer-reset policy.

## Scope, checks and remaining gates

The large two-head diff from `1d84c6d` contains inherited PR #81/main changes. Both `bcacd848` and this local candidate differ from main `b4398074` only in the eight audit Markdown documents; no new runtime, test or workflow change is introduced by PR #78. No further production defect was discovered in this documentation follow-up.

`git diff --check` passes; only the five authorized Markdown files changed; all eight documents were inventoried with committed blob/SHA256/size/mode in the JSON companion. The held GOV-021 / SEC-026 pinned targets exist. Three pre-existing relative links to `AUD4-AUTH-FIXES.md` are absent on PR #78 alone and resolve from PR #82's pinned source: merge #82 before #78 as planned, or replace those links if the order changes.

No Go commands or remote writes were made. The correction does not claim that queued CI or the running combined `17152` race suite passed. The coordinator owns subsequent current-head CI, merge receipts and final repository inventory.
