# Focused source refresh of stale specification notes

Reviewed source: `ac25cb72d3d1ad09f0de265352408fd6f52c8d8e`, 2026-11-19.
Scope: nine requirement notes in six specification files. Historical status
labels, summary counts, normative requirements and owner decisions remain
unchanged. Only obsolete absence claims are replaced with current source
and test evidence. This is not a whole-specification re-audit or a passing
execution receipt.

| Requirement | Current source evidence | Limit retained |
|---|---|---|
| FR-095 | `internal/service/export_write.go` streams rows; `internal/api/h_export.go` and `export_spool.go` create disk-backed artifacts and reserve stored/in-flight bytes. `TestExportStreamsWithBoundedMemory` measures a 150,000-row fixture against a buffering control; `TestParallelLargeExportsStayBounded` tests concurrency and the shared cap. | Product size and duration thresholds remain OQ-FR-3. Test source does not establish this candidate's peak-memory result. |
| ARC-004 | Interfaces are in `internal/repository`, and the contract harness in `internal/storetest` depends on those interfaces. SQLite and pass-through-wrapper entry points are in `internal/store/contract_test.go`; `TestRepositoryServiceAndContractSuiteHaveNoDriverDependency` checks indirect imports. | No second production database adapter is claimed. Cross-target release builds and current contract execution require their own receipts. |
| VIS-010, SEC-014 | `docs/security/personal-data-inventory.md` classifies columns and records purpose, retention, export and erasure. `internal/store/pii_inventory_test.go` checks schema coverage, row completeness and forbidden names; `TestOverLengthFreeTextRejectedPerField` checks field errors. | Retention decisions and the inventory's explicit unresolved erasure/activity gaps remain visible. No compliance conclusion is drawn from schema naming checks. |
| QA-001 | The sequence harness defaults to 250 cases, or 50 under race. `.github/workflows/qa001-manual.yml` runs `scripts/run_qa001.py` on an explicit seed range, defaults to 100,000 sequences, and retains logs/metadata/failure fixtures. | This workflow is manual investigation tooling. It does not approve a merge/release workload or demonstrate a completed 100,000-sequence run. |
| QA-018 | `e2e/tests/keyboard.spec.js` drives proposal and acceptance to completion; `zz-noscript.spec.js` checks completed history and the balance view without scripts. API and server-rendered HTTP tests also exist. | These browser cases alone do not cover the complete listing-origin scenario or every ledger/audit assertion. Current-head execution is separate. |
| QA-019 | `internal/api/qa019_expiry_test.go` and `internal/web/qa019_expiry_test.go` send real adapter requests, advance an injected clock and call scheduled expiry; they check terminal states, unchanged monetary data and once-only notices. | The HTML test is an HTTP test, not a real-browser expiry qualification. |
| QA-033 | `.github/workflows/ci.yml` runs Playwright/axe; `e2e/tests/a11y.spec.js` includes exhaustive page-template registration. `e2e/fixtures.js` selects WCAG 2.0/2.1 A/AA and 2.2 AA tags in light/dark schemes. | Automated rules do not establish complete state coverage or manual accessibility acceptance. Current candidate receipts remain required. |
| OPS-023 | `cmd/circula/premigrate.go` checks live SQLite integrity and verifies a backup before upgrading; its tests include plain/encrypted backups and corrupt-database refusal. `TestFailingMigrationRollsBackAndReopensCleanly` plants a mid-migration DDL conflict; `TestUpgradeFromEverySchemaVersion` checks prior versions. | SQLite integrity and transactional DDL rollback do not establish full ledger-invariant comparison before each migration commits, or the migration-metric acceptance criterion. |

Validation for this documentation change: referenced paths and named tests
were inspected, existing partial classifications were preserved, and
whitespace/link checks were run. Full application execution remains subject
to the integration candidate's CI gates; no historical receipt is promoted
to current-head evidence here.
