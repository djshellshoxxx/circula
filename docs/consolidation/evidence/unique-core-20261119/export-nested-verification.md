# Nested manifest compatibility repair

Local reviewed commit: `ed3e095f31053d9b7becaab1a89f309a6f166fca`. Tree: `1e36b67212192ef1b15b4e497711f9b079c46c1f`.
Base: `22c767a90f69d286642ef96e6b3c174836637a3c` (PR #125).

The standalone reader rejected a valid reconciliation summary split between repeated object fields or supported case aliases. Go's JSON decoder reuses nested struct pointers and retains earlier fields; the Python adapter replaced the entire nested object.

The repair carries only known fields from the previous reconciliation/account object, recursively retaining nested account data. Later scalar values still replace earlier values, scalar nulls remain no-ops, pointer nulls clear prior state, earlier type errors remain errors, and unknown extension data is not repeatedly copied. Existing archive, compression, bounds, field-type and reconciliation checks remain in the full suite.

Four regression groups cover actual ZIP JSON/CSV exports with duplicate/aliased split objects, empty/scalar-null later objects, later-value/type-error enforcement, and pointer-null resets. Before the production change: four groups, sixteen failures in 3.374 seconds. After: all 45 reader tests pass in 33.314 seconds. Whitespace checks pass and the worktree is clean.

A standalone probe ran the actual Go decoder with the production nested pointer field shapes and retained both the earlier entry/nonzero counts and later digest. That probe is decoder evidence, not a full Go command-package execution. Raw Python logs, the probe source and source-blob/digest details are in this analysis directory and `export_nested_verification.json`.

Only `scripts/read-export.py` and `scripts/test_read_export.py` changed. No push or remote operation was performed. Cross-platform provider results for the earlier reader do not qualify this later repair; the integrated head retains its native reader workflow.
