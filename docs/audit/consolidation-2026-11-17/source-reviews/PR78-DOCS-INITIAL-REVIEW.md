# PR #78 — independent documentation readiness review

Exact reviewed head: `1d84c6d5b849af1e1ad19e71f95462873f96973f`.
Tree: `bb2a1f320164098f25d2ce2ac822020a115d98c9`.
Merge base with fetched main: `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a`.
Review observation: 2026-11-18 09:33–09:35 UTC.

**Preserve all eight documents. They contain useful author-session evidence and synthesis, but this head needs factual, scope and link corrections before merge.** The diff adds only eight Markdown files, totaling 351 lines. No runtime or tests change. This reviewer ran no tests or Go commands and made no tracked-source, ref or remote mutation.

## Concrete merge conditions

1. **Inventory and branch history — CONSOLIDATION-LEDGER L5–6/L19.** The fetched real-remote inventory contains **105 branches: 96 at the author's baseline plus nine new branches**, not 104 = 96 + 8. Distinguish frozen historical PR heads from live branch refs. Merged PR #70 remains frozen at `1ee699f`; its preserved source branch advances for PR #84, now at fetched `eb4bb08b15d41295aa2c4b9f93040124ce4d66e9`. Record that reviewed forward advance and expected main movement. Timestamp and refresh current rows, or label them as a historical snapshot.

2. **AUD4-F03 — FEATURE-MATRIX L18, CODE-AUDIT C3 and SECURITY-AUDIT L27.** Use **partial remediation through historical lookup**. Include both same-millisecond writes and host-clock rollback, which independent test variants reproduce. Reference corrected PR #87 head `679a3016` and retain the outstanding durable-binding/backfill design item. This is a technical residual; no new owner policy was approved.

3. **MFA source claim — CODE-AUDIT C2.** `checkCurrentPassword` callers are not read-only. In `internal/service/mfa.go`, `BeginMFAEnrolment` calls it at L214 before a later `PutMFA`; `ConfirmMFAEnrolment` calls it at L280 before `PutMFA`, recovery-code replacement and session revocation. Remove the reassurance and track these paths for separate review. This source counterexample is **not an independently reproduced exploit**; no race probe was run here.

4. **Timeouts — CODE-AUDIT C5.** Makefile `test-race` at L39 has **no explicit timeout**. Linux CI uses `-race -timeout 20m` in `ci.yml` L88; macOS uses `-race -count=1 -p 2 -timeout=25m` at L133; Windows' ordinary command adds none at L137. Forty minutes is an author-local override, not the recorded Makefile/CI configuration. Correct the text without changing the gates.

5. **Traceability CI history — CODE-AUDIT C7.** Agent F's complete PR #82 inventory contains four superseded runs cancelled before any runner or step executed, followed by a queued run. All 28 required jobs in those four runs were cancelled with zero runners/steps; these are not failed traceability runs. The retained PR #83 snapshot is queued and does not substantiate a failure. Attribute any local regeneration failure to its actual author/log; require a specific failed run/job for a CI-failure claim. Evidence: `audit-f/final-ci/pr78-c7-ci-claim-verification.json`.

6. **Held-branch links — BLOCKERS L12–13.** Replace relative `GOV-021.md` and `SEC-026.md` links with commit-pinned links on the reviewed #85/#86 branches. Those files will remain absent from main while the PRs are held. `AUD4-AUTH-FIXES.md` links are also absent from current main, but should resolve after the planned #82 merge; verify them when #78 merges. The D01/D02 link resolves after merging current main, which already contains #81.

7. **Historical state and execution attribution.** `MERGE-MANIFEST.md` ends at the four-merge checkpoint `cdc8112` and still calls #81 open, while the ledger and TEST-EVIDENCE already record #81 merged. Label that manifest as a historical checkpoint or append current receipts. Narrow its blanket local-qualification prerequisite: #77's own row records only diff review. For TEST-EVIDENCE L32's root-only D02 fail-before claim, request the author's exact head, command, UID/platform and terminal before/after results, or label it as unverified author evidence. The author's environment could be UID 0; our UID 1000 skip does not prove the claim false. Keep authored local runs separate from independent verification.

8. **L9 precision — FEATURE-MATRIX L25–26.** Existing source includes `TestSQLiteCheckpointWriteFailurePreservesCommittedTrade`, with restart/replay checks after a checkpoint-write failure. Identify that specific Linux regression and cite its executed result where available. Distinguish broader read, fsync, truncate, other checkpoint modes and uncertain-commit qualification still outstanding.

## Useful material to retain

| Document | Unique value and treatment |
|---|---|
| `BASELINE.md` | Keep the author's historical session-start snapshot: main `ffa2758`, 96 branches and then-open PRs. Do not replace it with the coordinator's earlier 89-branch snapshot. |
| `CONSOLIDATION-LEDGER.md` | Incremental mapping of newly created and reused branches, PR ownership and held work. Correct and timestamp current claims. |
| `MERGE-MANIFEST.md` | Exact merge/CI references for #77/#71/#72/#79 and author-local qualification. Preserve the checkpoint and distinguish later state. |
| `FEATURE-MATRIX.md` | Requirement-level progress, owner gates and the isolated 15-module inventory. Correct F03 and L9 scope. |
| `CODE-AUDIT.md` | Process lessons, migration coordination and platform boundaries. Correct C2, C5 and C7. |
| `SECURITY-AUDIT.md` | Separate AUD4 findings, withdrawn hypotheses, reviewed areas and explicit unreviewed areas. Preserve DEC-014 and external SEC-033; align F03 and verification states. |
| `TEST-EVIDENCE.md` | Author-environment run summaries, CI references and before-controls. Add precise head/environment/privilege attribution. |
| `BLOCKERS.md` | Owner-decision and external-gate register. Make held-feature references usable without merging their code. |

The GOV-021/SEC-026 policy holds, resolved 0020 migration-number status, future-module isolation, A3-F02/DEC-014 residual, external SEC-033 review and manual screen-reader gates should remain. Do not merge held feature code to repair documentation links.

## Evidence boundaries and handoff

The 105-branch count is a fetched-ref observation, not a promised final count after the evidence PR. The companion JSON retains all observed branch heads, each document's blob/hash, unresolved links and concrete findings. The historical BASELINE remains valid as its own timestamped observation.

After corrections, refresh the actual PR head, verify the docs-only scope and links against the intended merge tree, and require current-head CI. None of these eight documents should be discarded merely because the coordinator has a separate evidence package.
