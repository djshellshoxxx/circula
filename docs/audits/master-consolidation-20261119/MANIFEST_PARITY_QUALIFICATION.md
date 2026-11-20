# Python export manifest qualification

This is the frozen author-source receipt. Later composition, publication and provider results are recorded separately in `EXECUTION_CHECKPOINT.md` and `CORE_QUALIFICATION.json`. The named original probe, fixture and log files below are preserved with exact bytes and hashes inside `MANIFEST_PARITY_EVIDENCE.json`.

Final source: `9e95b73861be23d0f46121f534e6c3e46d83bc1a`, tree `ca441d471000c1ae24ca87c28255a0c467c6e2f9`.

Integration from `6e8f5f1473084e7a14beccb92d3db9ea547554d4` requires both commits, in order: `56c09536a01fe6658ee98fe573029eac77c6ae82` then `9e95b73861be23d0f46121f534e6c3e46d83bc1a`. Only `scripts/read-export.py` and `scripts/test_read_export.py` changed. The worktree is clean and frozen. No remote writes were made for this repair.

## Behavior

Known manifest, file, reconciliation, and account-digest fields are type-checked at every occurrence before duplicate keys collapse. Integer fields retain the existing signed 64-bit bounds. Timestamp validation follows the pinned Go decoder, including its existing legacy spellings. Missing or null optional metadata is not newly required.

Known field aliases are resolved in JSON member order using ASCII case matching plus the Go simple-fold long-s and Kelvin equivalences. Later scalar values replace earlier values; later scalar null leaves an earlier value intact. Slice/pointer null remains explicit for existing final checks. Unknown Unicode extensions, including the FI ligature field, remain ignored. There is no blanket duplicate-key ban and no full Unicode casefold.

Both manifest and member JSON reject the nonstandard NaN, Infinity, and -Infinity tokens, including inside ignored extensions. Valid finite extension values and strings bearing those names remain supported.

## Evidence

- Original type regressions: 48 failing subcases before the first repair; retained in `implementation-fail-before.txt`.
- Alias/constant regressions against unchanged production at 56c0953: 7 selected methods, 48 failing subcases, 3.084 seconds. These include false acceptance cases and two valid alias-only controls previously rejected.
- Final command: `python3 -m unittest discover -s scripts -p test_read_export.py` — 38 tests pass in 23.950 seconds (wall 24.121 seconds). Exact source hashes are in `alias-constants-full-suite.json`.
- All 27 retained actual ZIP fixtures match expected results, with 5 positive controls; `alias-constants-fixture-replay.json` contains every exit and diagnostic.
- The eight reviewer manifests were decoded using the exact Go manifest declarations in a standard-library-only probe. Six type/syntax errors reject, the unknown ligature control accepts, and the later ROWS alias decodes to 1 despite 0 physical rows. This is decoder evidence, not a full command-package test. Exact command/output/environment are in `go-manifest-alias-probe-results.json`; the isolated temporary cache was removed after completion.
- `git diff --check` and staged whitespace checks passed.

The first full follow-up run had one test assertion failure: production correctly returned `unsupported schema_version 2`, while the new test expected the generic failure diagnostic. Only the assertion changed before the passing run; that intermediate result remains recorded separately.

## Review and limits

Independent reviewer `/root/branch_recovery` signed off the frozen complete 6e..9e diff with no remaining blocker in the repaired scope. The reviewer separately reran all eight alias/constant ZIP fixtures (seven rejects, ligature control accepted) and all six malformed-size ZIP probes (two valid controls accepted, four mismatches rejected). No combined Core or provider pass is claimed by this receipt. Existing archive/module qualification runs were not repeated.
