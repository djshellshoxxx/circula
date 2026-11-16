# Circula handoff

## 2026-11-11: beta consolidation (current)

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
