# Independent Circula Core scope and remaining-work review

Recorded 2026-11-21 02:38:02 UTC (2026-11-20 Pacific). Reviewer: `circula_security_review`.

## Verdict and source

The scope boundary passes for actual main `87651d36558a8e421e41fbd76c514045d88c74ed`, tree `43e65ee91a90ff70b70cc69d0b517b8367a2585f`. Ordered parents are preceding main `aa01563fcc891da60fefebb8335eca27855678e8` and qualified source `3ef43ac1e05360d2a1641742dc056afe8871879b`.

This review independently inspected committed objects, parsed actual feature manifests, checked ancestry, read live PR state, and compared 23 selected live branch refs with the fresh clone. It did not modify either repository/worktree, execute product tests, reproduce vulnerabilities, or publish remotely. The two files in this intermediate directory preserve those observations; writing them did not rerun the already sufficient scope checks.

## Independently verified boundaries

- Main contains 2,024 tracked files and no paths beneath `future/`, `features/`, `modules/`, `internal/future/` or `cmd/future/`. The prior receipt's 2,140 Git-tree entries also includes directories; these counts do not conflict.
- None of those paths changed relative to preceding main. The runtime/module-file reference search returned no future-module path/identifier matches.
- All fifteen actual future branches' `future/<module>/feature.json` blobs parse with JSON boolean `enabled: false` and string `integration: "HOLD"`. Exact branch, commit and blob identifiers are in `proof.json`.
- All seven held policy/interface tips match the earlier proof, remain outside main ancestry, and retain source-only commits.
- Live PRs #83, #85, #86, #94 and #107 remain open, draft and unmerged at the expected heads. The two additional held branches are the GOV-021 fixture lineage and proposed CLI contract.
- A fresh read-only `git ls-remote` returned all 23 selected refs, with no missing or changed refs.

No finding supports activating a future module or integrating the held policy alternatives. This is a bounded scope review, not a declaration that every security or release criterion is complete. Actual-main post-merge CI is independently owned by the qualification reviewer.

## Security-source gaps still accurately held

Current source confirms the recorded OPS-04 memory issue: Go and Python permit a four-GiB member and retain its bytes plus parsed representations. Integrity checks and complete compressed-stream handling do not establish a practical aggregate/parser memory budget. See `cmd/circula/verify_export.go` and `scripts/read-export.py`. New rejecting defaults require the recorded supported-budget/large-export compatibility decision.

OPS-03 remains: `cmd/circula/fileident_other.go` checks directory type rather than private Windows ACLs, its staging-parent check is a no-op, and link-count admission is permissive. Supported identities, ACL inheritance/ownership/reparse policy, implementation and native acceptance remain separate. The isolated future messaging implementation is not a Core repair.

LED-R02 remains a policy conflict: `internal/service/posting.go` and `transactions.go` disclose exact payer availability in some refusals. The existing actor-aware disclosure choice must be resolved before changing those contracts. GOV-021, AUD4-F02, SEC-026, all-conflicted/system-subject escalation and proposed CLI changes retain their explicit decisions.

## Reconciliation of the older ordinary engineering backlog

The newer `CORE_REMAINING_WORK_20261120.md` and security reconciliation concentrate on selected Core security/policy work. They are not an exhaustive replacement for ordinary acceptance engineering in `docs/audits/REMAINING_BLOCKERS.md`. Conversely, that older document's concrete source claims must not be copied forward without checking the subsequent implementations.

### UX-047: the historical flow list is largely superseded

The older note described a 525-key catalogue. The actual current `internal/web/messages/en.json` has **1,458 entries**. The 1,431-entry second UI batch is documented in `CORE_UI_COMPLETION_20261119.md`; later accepted changes are recorded in the completion-map addendum.

Shared refusals/validation, administrator reset, marketplace/directory, session/browser messages, member/invitation/inactivity pages, policy/network presentation, erasure, shared pages and administrator history already have catalogue implementations and related rendering/message tests. Rebuilding those flows from the old list would duplicate existing work.

The later authentication-fragment gap is also implemented. Current `login.html` and `mfa_enrol.html` use `msgAround` for complete linked sentences. `recovery_codes.html` uses `msgAroundCount` for the complete remaining-code sentence. The accepted completion-map addendum explicitly records this UX-047 slice alongside UX-028 loading. The loading repair subsequently qualified through PR #129/#127; this reviewer did not rerun that suite.

**UX-047 remains partial.** Complete locale/date/number semantics and any genuinely remaining English display literals still require bounded current-source classification. Additional languages remain a product decision. That language choice does not automatically block already accepted English complete-message/formatting engineering. This review did not identify a new concrete missing English flow; it does not certify every literal or promote UX-047 to complete. Start a new slice only after locating its actual remaining source and defining a meaningful rendered behavior check.

### Storage faults: important accepted qualification work remains

Existing current tests already cover real SQLite FULL recovery, the Linux pre-commit-marker WAL-write error, and the Linux checkpoint database-write error preserving a committed WAL. Later work adds:

- `internal/store/member_read_failure_test.go:TestMemberReadFailureAbortsAndRecovers`: delayed `Rows.Next` error handling for member/membership/pagination reads, rollback of staged writes, live recovery and reopen.
- `internal/store/commit_ack_test.go:TestCommitAcknowledgmentFailurePreservesPostingAndReplay`: a real successful COMMIT followed by a driver acknowledgment error, complete durable posting, connection recovery, same-key retry/replay and reopen. The pre-COMMIT rejection remains a separate control.
- Migration metadata and other driver-boundary error checks in `recovery_internal_test.go`.

These are not interchangeable with physical storage faults. The explicit qualification limits in those test comments and `docs/consolidation/evidence/commit-acknowledgment-loss.md` remain correct. The old statement that no COMMIT-acknowledgment test exists is superseded; the larger uncertain-device-outcome backlog is not.

**Still open:** real writable-VFS/OS read, fsync and truncate failures; checkpoint sync/truncate failures; errors around a complete physical commit marker; other device and power-loss acceptance. Migration COMMIT acknowledgment loss is also outside the existing posting regression. A supporting source search found no named `IOERR_FSYNC`, `IOERR_TRUNCATE`, `SQLITE_IOERR_READ`, `SQLITE_IOERR_SHORT_READ`, `xSync` or `xTruncate` fault cases in `internal/` or `cmd/`; search absence alone is not the proof.

This is **accepted ordinary qualification/harness engineering with execution-environment requirements**, not an unresolved governance choice. A useful next separate task is one bounded writable-VFS operation, with demonstrable fault reachability and retained database/WAL evidence. Verify original or fully committed state as appropriate, then restart and same-key replay. Preserve existing assertions. A driver-returned error must remain labeled driver-boundary evidence; synthetic VFS evidence must not be described as physical power-loss validation.

### Requirement evidence: two focused gaps are already completed

The old 15 uncited-built-row list is now **13**. Current `docs/TRACEABILITY.md` includes:

- DAT-001: `TestDefaultIDGeneratorPersistsCanonicalUUIDv7`.
- DAT-002: `TestInjectedIDGeneratorReachesPersistedPerson`.

Both exist in `internal/service/id_generation_test.go` and inspect a created, persisted person. Do not rebuild them or generalize their scope to all identifier classes. The old GOV-017 source locator and VIS-012 notes have also been refreshed in current traceability.

The remaining 13 are VIS-011, VIS-012, ARC-001, ARC-013, ARC-019, ARC-024, DAT-003, UX-007, UX-042, UX-043, UX-058, GOV-017 and OPS-001. They mix source/negative-capability inspection, release/native-artifact evidence and accessibility/content qualification. A missing test citation does not establish missing behavior.

There remain useful ordinary engineering opportunities: representative pointer-release/cancel browser evidence for UX-043; computed reduced-motion behavior for UX-058; and one packaged artifact exercised under both supported configurations, with native smoke evidence where the environment exists. These are separate from hosted launch permission, counsel, licensing, approved security contact, manual assistive-technology work and physical/native release acceptance. Do not add keyword tests or duplicate suites merely to make citation counts zero.

## Concrete next scope

Finish actual-main qualification and proof-based historical reconciliation first. Preserve this scope receipt without moving the qualified runtime merely to stamp status. Then select one genuinely remaining ordinary acceptance slice, preferably a bounded storage-fault harness or a precise browser interaction/motion criterion. Existing policy holds do not prohibit that work, but they do prohibit silently choosing recovery authority, disclosure or new resource-rejection policy.

## Principal source records

All product/source comparisons use main `87651d36558a8e421e41fbd76c514045d88c74ed`. Documentary comparison also reads evidence branch commit `c1a34476de017ab70acc99181ebbe16c337fc0e6`.

- `docs/consolidation/evidence/loading-followup-20261120/core127-main-merge-20261121.md`
- `docs/consolidation/evidence/core-containment-20261120/seven-preserved-holds.json`
- `docs/consolidation/CORE_REMAINING_WORK_20261120.md`
- `docs/consolidation/SECURITY_HANDOFF_RECONCILIATION_20261120.md`
- `docs/audits/REMAINING_BLOCKERS.md`
- `docs/consolidation/CORE_UI_COMPLETION_20261119.md` and `CORE_VERIFICATION_20261119.md`
- `docs/consolidation/evidence/unique-core-20261119/accepted-core-completion-map.md`
- `docs/consolidation/evidence/unique-core-20261119/uncited-built-acceptance.md`
- `docs/qa/sqlite-storage-failure-qualification.md`
- `docs/consolidation/evidence/commit-acknowledgment-loss.md`
