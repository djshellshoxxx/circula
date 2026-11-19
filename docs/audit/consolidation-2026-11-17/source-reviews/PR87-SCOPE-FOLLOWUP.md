# PR #87 — scope-correction follow-up

Reviewed 2026-11-18T09:29:12.694221+00:00. Exact delta from `1a3ef0fad836067b71857d961659459d976e7e15` to `679a3016d77c3cf34e218ecb296f1d42cc0969c8`; new tree `d88d4da959d34989248c3d172da56dabf24a272e`. The new commit's sole parent is the previously reviewed head.

**The earlier documentation objection is resolved.** AUD4-F03 is now explicitly **partial remediation** in the code comment, audit note and canonical GOV-001 implementation note. Both independently reproduced conditions—same-millisecond policy writes and host-clock rollback—remain open. The audit no longer says the historical answer can never change or dismisses the timing as uncontrollable. It records the actual response/waiver and re-review consequences and a future durable-binding/backfill design item. No migration or design approval is introduced.

## Exact change scope

Only three of 979 tracked files change; the other 976 Git blobs are identical. No file is added or removed.

| File | Change |
|---|---|
| `docs/audit/consolidation-completion-2026-11/AUD4-F03.md` | Documentation correction |
| `docs/specs/10-governance-operations.md` | Documentation correction |
| `internal/service/disputes_deadlines.go` | Comment-only; executable bytes unchanged |

The corrected audit says no migration **was added**, replacing the unsupported assertion that none was needed. Its prospective-policy explanation is expressly limited to ordinary forward-time changes. Its distinction between product-admin API controls and operator/host clock behavior is appropriately narrower than the earlier blanket timing claim. The current GOV-001 implementation note explicitly repeats both residuals; the normative prospective-policy requirement itself remains unchanged.

## Executable and test equivalence

- The repository contains 594 Go source files. **593 are byte-identical** across the two commits.
- The sole changed Go file, `internal/service/disputes_deadlines.go`, adds/removes only complete ordinary `//` comment lines. No compiler or line directive changes. Removing only such complete comment lines from each file leaves **3276 identical bytes**, SHA-256 `cfd1bf41463d63bbb4cdfdd4342e2a20037e6436b5d75ef491ec93d5b6978cab`. Executable whitespace, strings and all other code bytes are identical; this is direct byte comparison, not a compiler-run claim.
- All **322 `_test.go` files are byte-identical**, including every previously executed supplied regression and guard.
- The repository interface, SQLite historical-policy query, reusable store contract and legacy-policy regression have unchanged Git blobs. No runtime selection, timestamp comparison, tenant/decision guard, schema, API or test behavior changed.

The full new-head source and blob/hash inventory is in `PR87-FOLLOWUP-679a301.json`; the exact three-file patch is retained as `PR87-FOLLOWUP-679a301.diff.txt`.

## Evidence carried forward and readiness

The earlier [source/regression review](PR87-SOURCE-REVIEW.md) remains applicable to this identical executable and test content: ordinary cases and selected existing guards pass; literal-baseline and old-selector controls reproduce the original defect; same-ms and rollback variants reproduce the remaining defect. **No command was rerun on this new commit**, and those prior executions are not relabeled as new-head runs.

Source review is now acceptable **as explicitly partial remediation**, subject to current-head CI and the coordinator's combined-tree qualification. Keep AUD4-F03 partly remediated. The future policy-version/deadline binding and backfill still require a separate reviewed design; no owner-policy choice, migration or full-finding closure is inferred here.

This follow-up ran no Go command or test, changed no tracked source or refs, and made no remote write.
