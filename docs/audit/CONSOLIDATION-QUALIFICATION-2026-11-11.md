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

## Focused source review: reversal linkage (UX-017)

Inspected `internal/service/reversal_links.go`, `internal/service/reversal_links_test.go` and `internal/web/ux017_reversal_links_test.go` on the candidate branch.

**Confirmed source behavior:** `ReversalsOf` opens the requested network with read scope, fetches the original transaction, checks `canSeeTx`, returns the same not-found for inaccessible/missing original transactions, queries reversal transactions by `OriginalTxID` and `KindReversal`, and sorts returned entries by ledger sequence then ID. Service regression covers two partial reversals, an unreversed trade, a reversal queried as original, an outsider, a missing transaction and a foreign administrator. Web regression covers original/reversal links, reason text and adjustment history.

**Review finding UX-017-R1 (needs reproduction; not a verified defect):** `maxReversalLinks = 500` bounds the query, but `ReversalsOf` returns a successful result without an indication that additional matching reversals may have been omitted. If the domain permits more than 500 partial reversals of one transaction, History/detail could show an incomplete list. Before changing behavior, establish whether posting limits or minimum reversal amounts make this reachable; if reachable, add a regression with 501 reversals and implement explicit pagination or a documented truncation indicator. Do not silently increase the cap without considering response size and query cost.

**Verification status:** static source review only; no Go or browser tests executed. No defect severity or release blocker assigned without reachability evidence.

## Test-first continuation — reversal result boundary

Added `TestReversalsOfResultLimit` in `internal/service/reversal_links_test.go` on the isolated PR #34 branch. The test attempts 501 one-unit reversals of a 502-unit original and expects a complete result. This is a deliberate regression test of the 500-row limit, **not a passing test**. It may first fail during reversal posting if another domain limit makes 501 partial reversals impossible; in that case the concern is unreachable under those conditions and must be reclassified. The test skips under `go test -short` to avoid imposing 501 postings on fast suites.

Next: run `go test ./internal/service -run '^TestReversalsOfResultLimit$' -count=1 -timeout=15m` on the exact branch head. If it fails with 500 returned, add a bounded paginated store query or an explicit incomplete-result API contract, with focused and full regressions. If the posting itself is refused, document the domain limit instead. No runtime code change is justified until the failure mode is known.

No Go tests were executed in the connector-only session. This draft PR is expected to be red if the truncation path is reachable; it must not be merged as-is.
