# Independent Circula integration review

Reviewed on 2026-11-20. Base: `22c767a90f69d286642ef96e6b3c174836637a3c`. Integrated source reviewed: `29375f6a3901ceeca41580ee7cd0e263c76f9ed5`.

## Result

No unresolved blocking finding in the reviewed continuation diff. Two concrete issues found during review were repaired: repeated manifest file arrays did not preserve Go's existing element state, and the new acceptance preview could refuse an otherwise successful concurrent idempotent command. The fixes and their regression coverage are described below.

This is a bounded independent source review with selected local checks. It is not a GitHub approval, an external security qualification, full combined-head CI, or a claim that every specification requirement was independently requalified.

## Findings resolved during review

### Repeated manifest arrays must reuse existing file structs

At `acdc348`, the nested reconciliation/accounts repair correctly retained fields in repeated non-null pointer objects. An ordinary valid export still failed when a later `files` array contained partial file objects. Go's `encoding/json` reused the previous slice elements; the Python reader discarded their earlier `name`, size, row count and hash fields.

The standalone Go 1.26.11 probe uses the production field shapes and a local members export with correct size, row count and SHA-256. It established these decoder semantics: partial objects and null elements retain populated file fields; a nonempty shrink followed by regrowth retains previously populated backing slots; whole-array `[]` and `null` reset that state. The old Python reader failed the valid package with a missing `name`.

Commit `b4e724a93d159fb1438595ed4d0fd5b2ea19bf42` now maintains populated slot state across repeated arrays, clears it on the two reset cases, validates every recognized field occurrence, and copies only recognized file fields. The independent ordinary package passes the repaired reader. The added tests cover aliases, literal duplicate names, null elements, shrink/regrowth, resets, overriding values and earlier type errors. I found no further issue in this bounded parity repair.

Evidence: `manifest_files_parity_probe.go`, `reviewer_files_parity/`, `reviewer_files_python_result.json` (before), `reviewer_files_go_probe.log`, and `reviewer_files_python_after.log` (after). These prove decoder behavior and the Python package check; the standalone Go probe is not a production Go export-command execution.

### Acceptance confirmation must reach atomic service idempotency

The preview change at worker commit `7bca03e` introduced a confirmation regression. The handler read a proposed transaction, then read balance in another snapshot before calling the service. A permitted same-key acceptance committed between those reads. At the supported balance boundary, the stale preview attempted to project the amount again and returned HTTP 422 before the service could return the already committed result.

I qualified the interleaving in an isolated exported source tree using a real SQLite store, real service, authenticated HTTP flow and a one-shot callback after a completed read transaction. The diagnostic uses runtime call frames only to select the read boundary; it changes no transaction interfaces. The prior handler at `22c767a` passed, the new preflight handler failed, and the corrected handler passed. All three runs used race instrumentation. The checks also verified completed status, exactly two posting entries and exactly one balance change. The boundary fixture seeds an account balance directly; this is a handler/idempotency check, not a ledger reconciliation qualification.

Integrated commit `29375f6` (worker commit `debf7fc2655fd1064ed27a260d206997c703e106`) invokes the service before constructing refusal views, retains service authorization/state/idempotency checks, and redirects using the canonical returned transaction ID. GET previews retain their arithmetic checks. Refusal-rendering failures use the actual view error, preserving its error status.

The permanent `TestTradeAcceptanceConcurrentReplayAtBalanceBoundary` uses a stable direct-handler/store boundary without call-frame matching or scheduler timing. I independently confirmed that this test fails against the earlier handler with HTTP 422. The author recorded its passing corrected run and related trade-flow coverage. On the corrected HTTP diagnostic, the old preflight window is absent and the callback occurs during the later redirected read; the permanent direct-handler test separately covers the post-command balance-read replay.

Evidence: `acceptance-between-reads-original.log` (pass, 2.675s), `acceptance-between-reads-current.log` (expected failure), `acceptance-permanent-before.log` (expected failure), `acceptance-between-reads-fixed.log` (pass, 2.342s), and `functional-acceptance-stable-after.log` (author's pass, 2.482s). The separately retained `acceptance-control-preparation-retry.log` is a preparation failure/retry and is not counted as the original-handler control. The fixed handler SHA-256 is `68ff0e342a8c662c0fabf16c1d255150d5d39530e02861af3675ec315fcf67df`, identical to the integrated source.

## Other reviewed changes

| Area | Review conclusion and evidence limits |
| --- | --- |
| Scanner controls, `0a2705a` | Raw encoded evidence and normalized temporary JSON copies receive separate controls. Normalization asserts identical JSON meaning and physical line count. Exact rule/path/value exceptions, replacement and extension checks, relocated-path and adjacent-secret controls remain. The scanner configuration, workflows and earlier evidence files are unchanged. I inspected source and the recorded ten scanner controls; I did not rerun the scanner. |
| Member enumeration, `0def6ce` | Checking `rows.Err()` after closure prevents a late read failure from returning a partial result and committing a staged write. The test uses the real SQLite driver with one injected row-read failure, verifies rollback, then verifies recovery through the pool and a reopened database. I inspected the code and the before/after logs: four new failing cases before, four top-level tests with ten subcases passing afterward under race instrumentation. This does not test physical storage failure or fsync. |
| Nested manifest pointers, `acdc348` | Repeated reconciliation/accounts objects retain only known fields, null pointers reset state, scalar nulls preserve earlier values, and earlier type errors remain errors. I inspected the production change, regression tests and recorded 45-test Python result. The later slice repair's recorded full suite passes 49 tests. |
| Trade previews, `e7488ac` | The proposal review applies the transaction policy limit, bounded balance arithmetic and checked available-credit arithmetic. Available credit can legitimately exceed the balance bound by the credit limit. Acceptance reflects whether a payer-created proposal already reserved credit. Terminal transactions are not projected again. Confirmation delegates policy and replay semantics to the service; the concurrency correction above is included. |
| Dispute text, `25bc803` and `62f81eb` | The web paths count Unicode code points in alignment with service limits. The initial decision checks the actual stored rationale length, including surrounding whitespace. Tests cover mixed two-, three- and four-byte characters at 2,000 and 2,001 code points, review/confirmation behavior, unchanged state after refusal and preserved stored text. I reviewed source and tests rather than repeating the author's suite. |
| Operator JSON files, `63e40d8` | Configuration and audit-anchor verification require one non-null object followed by EOF. Existing `UseNumber`, known-field validation, anchor signature/version handling and surrounding whitespace behavior remain. Tests exercise direct parsing, real configuration preflight, signed anchor verification and unchanged database bytes. I inspected the runtime delta, 143-line test file and recorded six top-level/26-subcase passing race run. The recorded first combined before-run was killed and is explicitly separated from the completed anchor retry. |

## Verification boundary

I executed only the focused manifest decoder/package checks and isolated acceptance before/control/after checks described above. Other test results are inspected author/parent evidence, not a second independent execution. No production environment or remote repository was mutated. Review-created files are confined to the analysis area; the parent and functional worker own all production edits and final combined-head qualification.
