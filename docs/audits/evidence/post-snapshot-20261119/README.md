# Post-snapshot validation evidence

- `session_sql_smoke.py` extracts the six actual production SQL constants, applies all 21 migrations using Python SQLite, verifies indexed seek plans, and exercises bulk revocation scope, rollback, repeat behavior and integrity. `session_sql_smoke.json` records the successful execution at its exact source commit. This is not Go/modernc or service-level qualification. The first two harness attempts stopped while parsing the inline comment on `tokenCols`; the parser was corrected before the recorded successful run.
- `node-messages.execution.json` records the coordinator-observed seven-test production browser-message run: seven passed, zero failed/skipped/cancelled. File digests identify the executed source.
- `core-cursor-security-merged-a50f90b.execution.json` records the attempted driver-neutral production API source regression. Its Go process was killed at the 60-second outer deadline with no test events. The recording wrapper's zero exit is not a test pass. The empty raw log and exact harness are retained.

Provider full-suite, race, browser, native-platform, dependency and fuzz results must be attributed to the final PR head. Historical evidence and source review do not certify that head.

## Resolved catalogue and traceability qualification

The exact combined engineering commit `12af2112fb968f1c72e874e3c18b66e9e2626208` passed `go test -race -p 1 -count 1 -json ./internal/web/messages ./scripts/traceability` with Go 1.26.11. Both packages passed: **28 top-level tests and 69 subtests, zero failed/skipped**, with no race warning. Wall time was 45.816592 seconds and the child process exited zero before the 120-second guard. The metadata records exact arguments, environment, source/tree and file digests; the raw log is retained.

Independent preflight verified the exact 525-key union, all original values, no duplicate catalogue keys, preserved formatting arguments and the localized terms heading. The complete catalogue formatting test and new duplicate-definition CLI guards have explicit passing events. This run qualifies these two packages only, not the full application or database driver.
