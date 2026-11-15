# QA-042 traceability audit checkpoint

- Base: PR #35 `claude/cool-ramanujan-n2peag` at `c4d427670a59920b4a185875c2b15fca29cf41d1`.
- `scripts/traceability_audit.py` (PR #40): read-only detector of status-table IDs missing from the matrix.
  Its tests: `python3 -m unittest discover -s scripts -p 'traceability_audit_test.py' -v` (3 pass, run 2026-11-15).
- 2026-11-15, finish pass: `scripts/traceability.py` regenerates `docs/TRACEABILITY.md` (`--write`) and fails on drift
  (`--check`), including citations of files, symbols or tests that no longer exist. Tests:
  `python3 -m unittest scripts/traceability_test.py` (10 pass). Both test files and `--check` run in
  `scripts/check-hygiene.sh`, which CI's lint job runs on every PR.
- First full regeneration: 3 rows added (DAT-049, DAT-050, DAT-051); 7 stale citations corrected to the renamed
  symbols; API-064 (built), API-077 (built) and API-065 (note) re-checked against code and tests.
- Still open: per-row last result, date and build (they live in CI logs); built rows that cite no test
  (`python3 scripts/traceability.py` prints `built_without_tests`).
