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
