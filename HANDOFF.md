# Circula handoff

## 2026-11-14: beta follow-up integration (current)

Branch `claude/cool-ramanujan-n2peag`, draft PR https://github.com/djshellshoxxx/circula/pull/35 into `main`. It builds on
`claude/quirky-einstein-l9ni8m@8b4c8ba` (PR #33, which contains #31) and ports the unique work of
`chatgpt/circula-consolidation-review-20261111` (PR #34) and `claude/gracious-ptolemy-dscc7v` (PR #30). Results and
commands: [docs/releases/beta-readiness-report.md](docs/releases/beta-readiness-report.md) (2026-11-14 section).

Done in this pass:
- Ported from #34: web-ux slice (UX-017, UX-049, GOV-010, SEC-016, UX-021), H05 persistence tests (fixtures fixed for
  migration 0016), reversal-history pagination (UX-017-R1 confirmed red at 500 of 501, green with the fix), go.sum cache
  keys. Ported from #30: the HTTP-level acceptance scenario.
- Export slice integrated with the BR-06 fixes (shared in-flight byte budget, `export-dir` setting, bounded and logged
  memory fallback, manifest `reconciliation` documented). BR-04 fixed (erasure drops network-wide exports).
- arch-guards (ARC-002/004/008, QA-012, DAT-045) and sec-ops-docs (SEC-014/020/030, OPS-015/030) integrated after
  conflict resolution and 0016 adaptations. All eight preserved WIP slices are now in the tree.
- qa-robustness: QA-001 long-run knobs plus nightly job, QA-007 (100 interleavings), QA-010 (real process crash),
  QA-019 (expiry notice, new kind `trade.expired`), QA-026 (forced migration failure), QA-028 (backup under load).
  QA-024 was already met.
- BR-05 timing test. migrate-mode rule for a new database pinned (`startupMigrationGate`) and documented.
- Spec status rows and TRACEABILITY rows updated for the IDs above.

Next, in order:
1. Owner: review and merge PR #35, then close the PRs it supersedes (#19, #30, #31, #33, #34; others per
   [docs/plans/branch-consolidation-report.md](docs/plans/branch-consolidation-report.md)).
2. QA-042: full regeneration of `docs/TRACEABILITY.md` (only the rows touched here were refreshed).
3. Remaining P0 rows marked partial in `docs/specs/` (QA-001 release size waits on DEC-023; SEC-030 drills; OPS-015 staging
   alerts; ARC-002 snapshot.go exception; QA-030 independent reconciliation script).
4. Backup file names have one-second resolution: a restore in the same second as a pre-migration backup is refused
   ("destination already exists", nothing changed). Low; consider a unique suffix.
5. Owner gates below; independent security review (SEC-033); manual screen-reader run.

## 2026-11-11: beta consolidation

Branch `claude/circula-beta-completion-1npjw6`, a draft PR into `main`. It is the integration branch: it starts from
`claude/gracious-ptolemy-dscc7v@3c2a577` (PR #30) and merges PRs #21–#28, part of #13 and four preserved WIP
slices. Read these first:

- [docs/plans/branch-consolidation-report.md](docs/plans/branch-consolidation-report.md): every branch and the decision on it.
- [docs/plans/circula-completion-matrix.md](docs/plans/circula-completion-matrix.md): feature states with evidence, and the dependency graph.
- [docs/security/security-review.md](docs/security/security-review.md): findings BR-01..BR-06.
- [docs/releases/beta-readiness-report.md](docs/releases/beta-readiness-report.md): exact SHA, commands and results, blockers.

Done: C-09 policy ETag fix; DOM-017/DAT-051 dispute deadline fixed at completion (migration 0016); beta ledger and
session tests; release workflow gates; e2e WCAG/contrast/flow coverage (578 tests); lint at 0 issues.

Next, in order:
1. Owner: review and merge this PR into `main`, then close the PRs it supersedes (see the consolidation report).
2. Export slice: fix review findings BR-06 (bound in-flight bytes, configurable spool directory, document the
   manifest field), then integrate.
3. Rebase web-ux, arch-guards and sec-ops-docs onto this branch, review them, integrate.
4. CI: cut duplicate runs; move the macOS/Windows matrix to main or nightly (measure first).
5. BR-04: drop admin full-export artefacts on member erasure. BR-05: timing check for enumeration.
6. Owner gates below; independent security review (SEC-033); manual screen-reader run.

---

## Earlier handoff (2026-11-04, kept for history)

Updated 2026-11-04. Owned branch: `claude/happy-noether-444cum`, draft PR
https://github.com/djshellshoxxx/circula/pull/19 (base `main`, which holds prompts only).
Started from the product tip `codex/circula-noscript-qa@ba57512`, which contains the #17 → #18
chain and the responsive/no-script QA branches. Earlier handoffs are preserved in Git history
and under `docs/audit/`.

## Done in this pass
- `94728b1` DEF-001: the invitation return link (`next`) was double-escaped, so the return path
  was lost after Create an account / Sign in. This was the only failing e2e case at `b231dc5`.
  Fixed in the templates. Regression test: `TestReturnLinksDecodeToTheOriginalPath`.
- `c34d947`: remediates audit 2026-10-28 Medium findings A1-01, A1-02/A1-11, A1-03, A3-F01 and
  A3-F16. Each PoC is kept as a regression test, and those tests fail with the fix reverse-applied.
  A3-F02 is confirmed but waits on owner decision DEC-014. Report:
  [docs/audit/2026-10-28/REMEDIATION-2026-11-04.md](docs/audit/2026-10-28/REMEDIATION-2026-11-04.md).

- `ce7dcf0`, `d98734c`: merged reviewed slices `wf/auth-sessions` and `wf/gov-policy`. They add
  SEC-027/OPS-010 `circula revoke-sessions` (with `-rotate-web-key`), SEC-003 role session
  lifetimes, SEC-006 configurable web/API limits and `network-join-limit`, SEC-032 grant cap plus
  audited open-mode confirmation, GOV-011 listed removal reasons and conflicted-admin refusal, and
  DOM-031 stored scale-change detection. Defaults keep the earlier behaviour.
- `a2d20b1`: fixes the four blocking findings from an independent review of those merges, each
  with a regression test that fails without the fix.

## Verified locally (Go 1.26.10, this container)
- At `ba57512`: `go build ./...`, `go vet ./...` and `go test -count=1 ./...` pass in all 9 packages.
- At `94728b1`: the full Playwright suite passes, 325 of 325.
- At `c34d947`: `go vet ./...` passes; `go test` passes for service, store, cmd/circula, web, api and domain.

## Open owner questions raised this pass
- GOV-011: should `transaction_id` be mandatory for fraud or breach removals? The conflict check
  only sees the trade named in the request plus undecided disputes and re-reviews.
- DOM-031: should a scale change stored by an earlier build be neutralised once it is in force?
  It is currently detected, audited and logged as a warning only.
- SEC-032: should open self-joins be excluded from the grant even after confirmation? Currently
  they receive it, per DOM-027/DEC-051. Also: how should a lowered cap affect stored grants?
- SEC-006: per-login throttle values, numeric bounds for the rate settings (OPS-008) and a
  per-network login limit.
- DEC-005, DEC-006 and DEC-011 values. DECISIONS.md rows DEC-051, DEC-064 and DEC-065 are stale
  against the built mechanisms; they are owner records and were not edited.

## Paused 2026-11-11 (owner instruction)
- Work stopped at the owner's request. No agents or workflows are running.
- Unmerged, unreviewed slice work is preserved as patch series in
  [docs/audit/wip-slices-2026-11-11/](docs/audit/wip-slices-2026-11-11/README.md), with base, head,
  IDs and state for each.
- Next steps to resume:
  1. Review each implemented slice, then fix and merge it: web-ux, e2e-axe, e2e-flows, arch-guards, sec-ops-docs.
  2. Finish boundary-fuzz, export and release-chain.
  3. Implement qa-robustness.
  4. Cut duplicate CI runs and move the macOS/Windows test matrix to main only, measured as in
     the scratch notes: macOS race 7.9 min, Ubuntu race 6.8 min, e2e 4.7 min.
  5. Regenerate docs/TRACEABILITY.md (QA-042).
- Last CI result: `76e2c89` passed every job (push and PR runs). CI for later heads had not
  finished when work was paused.

## Owner gates (not decided here)
- Licence; DEC-005, DEC-011, DEC-013, DEC-014, DEC-021, DEC-022, DEC-023 and DEC-025; UX-047 locale scope.
- Release thresholds; pilot jurisdictions; the manual screen-reader run (UX-056, QA-035).
- Independent security review (SEC-033); A3-F02 break-glass scope.
- Merging to `main`, releases and deployments.
