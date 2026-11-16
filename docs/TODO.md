# Circula TODO

Date: 2026-11-16. Branch: `claude/cool-gauss-vdxlod`. Single consolidated list of open work. Every item is open. Nothing is marked done.

## Source of truth

- Status text in docs is checked against code. `(verified)` = checked in code on 2026-11-16. `UNVERIFIED` = not checked.
- Main sources: `HANDOFF.md` (2026-11-11), `docs/plans/circula-completion-matrix.md`, `docs/security/security-review.md`, `docs/releases/beta-readiness-report.md`, `docs/qa/DEFECTS.md`.
- `docs/audit/CONTINUATION.md` is historic. Only items still true per code are kept.
- Owner decisions live in `docs/DECISIONS.md`. This file records the questions only and decides nothing.

Legend: `eng` = can do now. `owner` = needs owner decision. `ext` = external (manual test, independent review, licence, pilot).

## 1. Ledger and security (eng)

- [ ] L9 Fault injection runs at the store interface only. Add tests with real SQLite I/O errors — eng — src: matrix L9
- [ ] L12 Status-event ordering is checked by count only. Check order — eng — src: matrix L12
- [ ] A4 Same-network member-page reads are tested only lightly. Add tests — eng — src: matrix A4
- [ ] A6 / BR-05 Sign-in and recovery timing equivalence not measured. No timing test found in `*_test.go` (verified) — eng — src: matrix A6, security-review BR-05
- [ ] BR-04 Admin full-export artefacts survive member erasure. `internal/api/erasure_hook.go` `OnMemberErased` drops only entries keyed to the erased person (verified), so admin-scope artefacts stay up to 24 h. Drop them on erasure — eng — src: security-review BR-04, readiness #6
- [ ] BR-06 Export spool: bound in-flight bytes, configurable spool directory, document manifest field, then integrate. No spool code in non-test Go source (verified). Slice patch `export` (base `a2d20b1`, head `c512f9e`, unverified) — eng — src: security-review BR-06, HANDOFF step 2, matrix W10, `docs/audit/wip-slices-2026-11-11/README.md`
- [ ] FR-054 Timestamp-based historical balance read. `service.BalanceAt` takes `ledger_seq` only (verified: `internal/service/accounts.go:96`); `internal/histbal/histbal.go` notes the gap — eng — src: CONTINUATION step 4
- [ ] Re-verify the 31 Low and 18 Info findings of the 2026-10-28 audit. Status not re-verified (UNVERIFIED) — eng — src: security-review "Earlier findings still open"

## 2. Build, CI, release

- [ ] CI on the current head has not run. Last green run was `76e2c89`. Later heads did not finish (UNVERIFIED) — eng — src: readiness #1, HANDOFF "Paused"
- [ ] govulncheck not run locally (`vuln.go.dev` refused by proxy). Confirm it passes in CI — eng — src: readiness #2, security-review "Dependencies"
- [ ] O4 CI: cut duplicate runs. Move macOS 15 and Windows test matrix to main or nightly. Measure first (macOS race 7.9 min, Ubuntu race 6.8 min, e2e 4.7 min). `ci.yml` still runs `[macos-15-intel, windows-2022]` (verified) — eng — src: matrix O4, HANDOFF step 4
- [ ] O2 Container: `deploy/Dockerfile` and compose not built or run (no Docker here). Run `docker build -f deploy/Dockerfile .` on a Docker host — eng — src: matrix O2, readiness #8
- [ ] Review, fix, merge slice `web-ux` (`4995130`; UX-017 partial). Patch series in `docs/audit/wip-slices-2026-11-11/` — eng — src: HANDOFF step 3, wip README
- [ ] Review, fix, merge slice `e2e-axe` (`c1d83a8`) — eng — src: HANDOFF step 1, wip README
- [ ] Review, fix, merge slice `e2e-flows` (`e3ddeba`; review was in progress) — eng — src: HANDOFF step 1, wip README
- [ ] Review, rebase onto this branch, merge slice `arch-guards` (`cf62791`; ARC-002, QA-012 partial) — eng — src: HANDOFF step 3, wip README
- [ ] Review, rebase onto this branch, merge slice `sec-ops-docs` (`f35a0f0`; test results not confirmed) — eng — src: HANDOFF step 3, wip README
- [ ] Finish and verify slice `boundary-fuzz` (`187e2a5`, WIP, unverified) — eng — src: HANDOFF "Paused" step 2, wip README
- [ ] Finish and verify slice `release-chain` (`808868b`, WIP, unverified) — eng — src: HANDOFF "Paused" step 2, wip README
- [ ] Implement slice `qa-robustness` (QA-001, QA-007, QA-010, QA-024, QA-028, QA-026, QA-019). Not started — eng — src: HANDOFF "Paused" step 3, wip README
- [ ] Readiness report: refresh after CI and slice work. "Deployment instructions verified" is still Partial — eng — src: readiness acceptance table
- [ ] First tagged release run of `release.yml` (SBOM, scans, release-notes gate). Release BR-03 gate not exercised on a real tag — owner — src: matrix O3, security-review BR-03

## 3. Docs and specs (eng)

- [ ] Regenerate `docs/TRACEABILITY.md` (QA-042). File header says last updated 2026-10-21 (verified), so it predates the 2026-11-11 changes — eng — src: HANDOFF "Paused" step 5
- [ ] Update stale spec status columns: FR-002, DAT-047, DAT-049, DAT-033, DAT-035, FR-082. Code shows these built — eng — src: matrix header, matrix W6
- [ ] Re-audit historical rows in `docs/audit/COMPLETENESS.md` against code. Prioritise P0 over stale status text — eng — src: CONTINUATION step 3
- [ ] Reconcile the 28 P0 items marked partial in `docs/specs/12-quality-test-plan.md` with the matrix — eng — src: matrix Estimate

## 4. Owner decisions (questions only; not decided here)

- [ ] DEC-005 Session timeouts. Single 720 h idle and 2160 h absolute are built for all roles. Decide whether admin (60 min idle, 12 h absolute) and reviewer values are built — owner — src: DECISIONS.md DEC-005, DEC-064
- [ ] DEC-011 Sign-up grants in MVP. Off by default. Decide approval or invite requirement and caps (not built). Also OQ-DOM-3 / DEC-051 — owner — src: DECISIONS.md DEC-011, matrix
- [ ] DEC-013 Dual control for adjustments and write-offs. Not built. Confirm rule (two or more administrators) — owner — src: DECISIONS.md DEC-013
- [ ] DEC-014 Platform operator break-glass controls and notice period. Not built. Proposed 4 h expiry. Also gates A3-F02 break-glass scope — owner — src: DECISIONS.md DEC-014, security-review A3-F02
- [ ] DEC-021 Governance method for policy changes. Objection right not built. Confirm unilateral change with notice and objection — owner — src: DECISIONS.md DEC-021
- [ ] DEC-022 Support objectives and any external commitment. Not set — owner — src: DECISIONS.md DEC-022
- [ ] DEC-023 Numerical thresholds: property-test run sizes, interleaving counts, coverage, performance, timing-equality threshold. Not confirmed. Also release thresholds — owner — src: DECISIONS.md DEC-023, HANDOFF owner gates
- [ ] DEC-025 Notice periods: 14 days for adverse policy changes, 30 days for network closure. Not implemented — owner — src: DECISIONS.md DEC-025
- [ ] DEC-072 Who may decide a dispute: reviewer role only, or an administrator too? Code lets a non-reviewer admin decide (verified: `internal/service/disputes.go` returns `ActorAdmin`) — owner — src: matrix W5, DECISIONS.md DEC-072
- [ ] L16 Re-review window reads the policy in force at decision time, outside DOM-017. Confirm intent — owner — src: matrix L16, readiness #9
- [ ] GOV-011 Should `transaction_id` be mandatory for fraud or breach removals? Conflict check sees only the named trade and open disputes — owner — src: HANDOFF "Open owner questions"
- [ ] DOM-031 Should a scale change stored by an earlier build be neutralised once in force? Currently detected, audited and logged as a warning only — owner — src: HANDOFF "Open owner questions"
- [ ] SEC-032 Should open self-joins be excluded from the sign-up grant after confirmation? How should a lowered cap affect stored grants? — owner — src: HANDOFF "Open owner questions"
- [ ] SEC-006 / A5 Per-login throttle values, numeric bounds for rate settings (OPS-008), per-network login limit — owner — src: matrix A5, HANDOFF "Open owner questions"
- [ ] A3-F02 Break-glass scope. Waits on DEC-014 — owner — src: security-review "Earlier findings still open"
- [ ] UX-047 Locale scope — owner — src: HANDOFF "Owner gates"
- [ ] DEC-051, DEC-064, DEC-065 rows are stale against built mechanisms. Owner to update the records (not edited here) — owner — src: HANDOFF "Open owner questions", DECISIONS.md
- [ ] W5 / DEC-072 follow-up: if the answer is reviewer-only, change `service/disputes.go` and tests — owner then eng — src: matrix W5

## 5. External and manual gates (ext)

- [ ] UX-056 / QA-035 Manual screen-reader run per `docs/qa/SCREEN-READER.md`. Log any A-block defect in `docs/qa/DEFECTS.md`. `DEFECTS.md` has no open rows today (verified) — ext — src: matrix W9, HANDOFF "Owner gates", readiness #4
- [ ] SEC-033 Independent security review. Not done; the 2026-11-11 review is AI-assisted only — ext — src: security-review header, readiness #4, HANDOFF "Owner gates"
- [ ] Pilot jurisdictions. No decision record found (UNVERIFIED) — ext — src: HANDOFF "Owner gates"
- [ ] Licence. No `LICENSE` file in repo (verified). Choose licence and add it — ext — src: HANDOFF "Owner gates", readiness #3
- [ ] Manual penetration test, Windows-specific code path review, container image hardening — ext — src: security-review "Not covered"

## 6. Repo hygiene

- [ ] Open PR triage: decide close or merge for each open PR; see worker F report if present — owner — src: HANDOFF step 1, `docs/plans/branch-consolidation-report.md`
- [ ] Review and merge draft PR `claude/circula-beta-completion-1npjw6` into `main`, then close the PRs it supersedes — owner — src: HANDOFF step 1
- [ ] Remote branch clutter on origin (many `chatgpt/*` and `claude/*` branches): decide which to delete or keep — owner — src: `git branch -a`
