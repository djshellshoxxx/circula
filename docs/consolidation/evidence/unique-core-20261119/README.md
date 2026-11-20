# Unique Core continuation evidence

This directory preserves source-specific review and verification receipts for
the current [continuation](../../UNIQUE_CORE_CONTINUATION_20261119.md). Each log
retains its observed success, failure or interruption. A focused result is not
a full integration result. `SHA256SUMS` covers the files in this directory,
excluding the manifest itself; it is an integrity inventory, not a signature.

| Files | Scope |
|---|---|
| `branches.json`, `branch-uniqueness-review.md` | Frozen 145-branch inventory against PR #125; excludes the symbolic origin/HEAD alias. |
| `source-map.json` | Local-to-provider commit/tree mapping for published checkpoints. |
| `scanner-independent-review.md` | Bounded independent review of the scanner-control repair. Full controls are preserved in the neighboring `scanner-json-controls-20261119` directory. |
| `storage-read-before.log`, `storage-read-after.log` | Late SQLite member-iterator error regression and positive transaction/connection controls. |
| `export-nested-*`, `export-files-*`, `reviewer-files-*` | Nested-object and repeated-file-array Python/Go manifest parity, including full 49-test reader pass. |
| `functional-*` | Dispute Unicode, trade preview, raw rationale, browser-form and related service/HTTP checks. Browser test listing does not constitute execution. |
| `acceptance-*`, `integration-independent-review.md` | Concurrent trade acceptance regression discovered in review, its source comparison, repair and permanent regression. |
| `operator-*`, `acceptance-review.md` | Exactly-one-object operator configuration and audit-anchor input behavior, with source-specific verification record. |
| `authority-*` | Retained cross-network authority and bounded grant-provenance checks; the killed attempt remains identified separately. |
| `listing-*` | Atomic conditional owner edits, API 412 behavior and retained ETag/wildcard semantics. |
| `closure-*` | Scheduled-job boundary, versioned streaming checkpoints, migration, API/OpenAPI, targeted race, lint and hygiene. Worker source hashes are recorded in the continuation. |
| `platform-audit-*` | Platform verification and versioned-anchor checks, including a retained incomplete race attempt. |
| `report-privacy-*` | Ordinary role-overlap privacy, pagination, replay and export authorization checks. |
| `id-generation-*` | Production default UUIDv7 and injected IDs persisted through person registration. |
| `audit-mirror-*` | Bounded best-effort delivery, inert paused destination, ordering, failure counters and bounded drain. |
| `integration-second-review.md` | Later independent source review, with exact included and excluded source boundaries. |
| `browser-preflight.*` | Environment-only local Chromium launch failure before any product assertion. |
| `uncited-built-acceptance.md` | Bounded mapping of the 15 built requirements without automated test citations, preserving manual/owner/external acceptance boundaries. |

Raw rendered-HTML logs retain their original trailing whitespace; source/prose
whitespace checks exclude these byte-preserved `.log` files.

Raw logs may refer to their original temporary worktrees; the checked-in copy
preserves the evidence even after those worktrees disappear. No credentials or
production data were used. Provider run URLs and final combined qualification
are recorded in the continuation when observed; this directory does not turn
pending jobs into passing results.


## Accepted-feature completion checkpoint — 2026-11-20 Pacific

The additional records preserve the distinction between author execution,
independent source review and combined qualification:

- `migration-integrity-*` and `migration-service-compatibility.log` cover the
  pre-commit migration gate at `d65f5a0`, including the original-schema golden
  upgrade. `migration-baseline-fixes.log` records the successful corrected legacy-fixture
  and inventory recheck. The original combined failures remain in the combined
  run log.
- `ui-*` records the loading, complete authentication messages and network About
  slice integrated at `e8c8f6a`. The Playwright list proves discovery only.
- `transaction-report-focused.jsonl` and its receipt qualify worker `d885af83`,
  integrated at `f4fce9b`. The independent review also checks the combined
  catalogue and migration-gate seams. Its review does not execute tests.
- `posting-metrics-tests.log` and its receipt record ten focused race passes on
  worker `4630a0a`, integrated at `a29a094`. The two earlier files retain the
  corrected test-constant and fixture-assumption failures.
- `combined-race-0168500.*` records a failed combined run: 1,176 passing
  test/subtest events, two store test failures, and killed CLI/service/web
  processes. The two genuine store fixture/inventory failures were fixed at
  `09e4674` and `4b56b15` and then passed under race; the killed work is incomplete.
- `lint-f4f718b.*` records a clean-source lint attempt blocked by a killed
  service-test compiler. It is not a lint pass.
- `helper-results.json` ties 65 Python helper passes and 23 Node helper passes
  to their actual source and file hashes. Hygiene passed at `f4f718b` after the
  required migration-table-rebuild explanation was added, without changing SQL.

`source-map.json` now covers publication through remote `c45c475` / local
`c710487`. Later code and this evidence commit require their own publication and
qualification; the mapping is not a recursive claim about its own commit.

### Reviewer projection, authentication and combined checks

- `dispute-report-projection-verification.json` identifies the worker and
  integrated source. Five packages passed first; the full service package
  passed unchanged after the Go cache and temporary build files were moved
  from RAM-backed storage to disk. The earlier killed attempts remain in the
  evidence. In total, six packages recorded 41 top-level and 58 subtest passes
  under race. This is still a focused selection.
- `reviewer-projection-independent-review.md` records the separate source
  review. Its read-error finding is fixed in `0773308`;
  `dispute-report-read-errors-*` records the focused race check, including
  preservation of ordinary not-found behavior and propagation of read errors.
- `authentication-metrics-focused.*`, its receipt, and the verification map
  record 22 top-level and two subtest race passes. The optional observer's
  bounded classes do not count storage/cancellation/policy validation as bad
  credentials, and the outer sign-in operation avoids double-counting.
- `combined-normal-c710487.*` records a failed full normal run with 3,317 pass
  events, six skips and one migration-review test failure. Every package
  compiled. CLI, service, API, contract and web package tests passed; the store
  package failed its unregistered migration-0023 rebuild guard. The six skips
  are identified individually in the receipt.
- `compatible-rebuild-check.jsonl` records the exact reviewed migration-0023
  registry repair, preservation/rollback/retry fixture and unchanged scanner
  controls passing under race. An exact SQL digest scopes that registry entry;
  it is not a waiver for arbitrary DROP/RENAME statements.
- `lint-0773308.*` retains the three style findings. `lint-c9dbc84.*` records
  the succeeding full lint run, zero issues, with clean unchanged source.

These receipts predate the historical-credit witness and member/listing case
context. They do not qualify those later additions or the final combined tree.


### Publication representations

`evidence-transformations.json` records original and published hashes for four
receipts whose path/checksum maps were represented as explicit path/sha256 rows,
and for two published combined-run logs whose generated test request identifiers
were redacted. All source checksum values and test result events are preserved.
The original raw logs and receipts remain in the analysis workspace; the earlier
report-privacy receipt is also preserved in its existing Git history. These
six published files are documented derivatives, not byte-identical raw logs.
The unchanged secret scanner flagged the original noncredential values; no
scanner configuration, finding rule, or control fixture was relaxed. The
expanded-evidence scan's fifteen findings are retained with this exact scope.
