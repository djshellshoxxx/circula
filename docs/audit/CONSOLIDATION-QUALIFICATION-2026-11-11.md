# Circula consolidation qualification — 2026-11-11

## Scope and provenance

This review is based on the GitHub compare API, comparing `claude/happy-noether-444cum` to `haiku/circula-consolidation`. The comparison returned **ahead by 84 commits, behind by 0**. The haiku branch is a descendant of the original application branch, not an independent patch that should be blindly cherry-picked. No tests were run in this review, and no production readiness is claimed.

## Already accumulated in the haiku branch

- Core/regression tests: `cmd/circula/acceptance_scenario_test.go`, `internal/store/persistence_regression_test.go`, `internal/service/reversal_links_test.go`.
- UX and governance: `internal/web/ux017_reversal_links_test.go`, `internal/web/ux021_refusal_test.go`, `internal/web/gov010_leave_export_test.go`, `internal/web/sec016_member_ref_test.go` and associated templates/handlers.
- Accessibility and browser QA: new `e2e/tests/a11y-governance.spec.js`, `contrast.spec.js`, `greyscale.spec.js`, `kbd-admin.spec.js`, `target-size.spec.js`, plus changes to navigation/responsive tests and fixtures.
- CI and documentation: modified `.github/workflows/ci.yml`; `docs/audit/CI-WORKFLOW-QUALIFICATION-20261111.md`; user/developer wiki source pages; handoff and audit ledgers.

These paths are *present in the compare*, not necessarily verified or complete. The original pause handoff's 'unmerged slices' list is historical and must be reconciled with the newer branch before work is assigned.

## Qualification sequence

1. Freeze exact branch SHA and inspect the latest commit and CI statuses. Keep original PR #19 and other workers' branches unchanged.
2. Compare the eight preserved patch series with current haiku branch paths and implementation semantics. Mark each item integrated / partial / absent, citing tests and commits. Do not reapply already integrated patches.
3. Run `gofmt`, `go vet ./...`, `go test -count=1 ./...`, focused regression tests and Playwright e2e on the **exact candidate head**. Check CI workflow change against the documented limitations, especially PR push trigger coverage and macOS cancellation. Record actual outcomes, not historical passes.
4. Independently review ledger balance and reversal invariants, authorization/dual-control, idempotency, recovery, and audit durability; fix reproduced failures on an isolated branch with red/green regression evidence.
5. Finish only confirmed missing portions of boundary-fuzz, export, release-chain and qa-robustness. Rebuild `docs/TRACEABILITY.md` from exact source and tests.
6. Prepare a reviewed integration PR; do not merge to `main`, deploy or release until owner gates are satisfied.

## Owner-held gates

Licence, undecided DEC policies (including DEC-014/A3-F02), independent security review, manual screen-reader testing, pilot/release thresholds and any financial-policy changes remain explicitly unapproved.

## Verification in this review

- GitHub compare API: `haiku/circula-consolidation` ahead 84, behind 0 relative to `claude/happy-noether-444cum`.
- Paths above were returned as additions/modifications by that comparison.
- **Not executed:** Go tests, browser tests, patch application, merge, deployment or release.
