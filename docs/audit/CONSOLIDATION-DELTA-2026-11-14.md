# Cross-branch consolidation delta qualification — 2026-11-14

## Source pins
- Primary PR #33: `claude/quirky-einstein-l9ni8m@8b4c8ba23c99d4418348d76ff8cb38a72e7b958d`, stacked on PR #31.
- Alternative PR #34: `chatgpt/circula-consolidation-review-20261111@02f9f0a7bb87f54611e2fe0b0837a519d2bcca9e`, stacked on `haiku/circula-consolidation`.
- GitHub compare reported **diverged**, 26 primary-only commits and 22 alternative-only commits. These are *not* 48 independent features; overlapping histories and files require per-change review.
- This branch is based on the primary SHA. No alternative application code has been copied or merged.

## Distinct alternative files worth porting selectively
1. `internal/service/reversal_links.go` and `internal/service/reversal_links_test.go`: UX-017 reversal history and access-control tests. The test includes a 501-partial-reversal boundary case. Port source and tests together only after checking the primary service API, transaction schema and authorization semantics. Avoid blindly transplanting a stale implementation.
2. `internal/store/persistence_regression_test.go`: database migration rollback, restart and ledger-integrity regression coverage. It predates the primary branch's migration 0016; adapt the fixture and the test's migration-head assumptions before running. Do not change production migration history to satisfy stale tests.
3. `docs/audit/CI-WORKFLOW-QUALIFICATION-20261111.md` and `docs/audit/CONSOLIDATION-QUALIFICATION-2026-11-11.md`: documentation-only evidence, compare against current `docs/ops/ci.md` and handoffs before importing.

## Conflicting / shared areas
- `HANDOFF.md`, `.github/workflows/ci.yml`, session and governance services, web handlers/templates, and many E2E fixtures are modified on both sides. Prefer the primary's current code, compare individual hunks, and import only proven missing behavior.
- The primary has dispute deadline migration 0016, policy ETag fix, release gates, and updated browser tests. These must remain intact.
- PR #35 is a documentation prompt, not an application integration.

## Recommended execution order
1. Confirm PR #33 head has not changed since the source pin and check current CI/ownership.
2. Port isolated reversal-history source and tests into a *separate* branch; run `go test -count=1 ./internal/service`, then `go test -race ./...`. If the 501-item test exposes truncation, fix with bounded pagination or explicit error, not silent truncation.
3. Adapt persistence regression tests for schema 0016; run `go test -count=1 ./internal/store` and the full race suite. Preserve the forward-only migration contract.
4. Review and integrate the already queued export BR-06, BR-04 erasure, BR-05 enumeration timing, arch-guards, web-ux and sec-ops-docs slices without duplicating active ownership.
5. Validate Docker/Compose and release CI, regenerate traceability and update the handoff with exact SHAs and test results.

## Verification status
This is a **read-only source comparison and documentation commit**. No Go, browser, container or CI tests were run for this qualification. No merge to `main`, deployment, release or owner-gated policy change was performed.
