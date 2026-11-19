# Consolidation verification results

Toolchain: Go 1.26.11, Linux amd64. Source baseline: main
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Only completed, observed post-restart
commands are counted below. The first executor's interrupted compilation and
race attempts are excluded.

| Check | Observed result | Evidence |
|---|---|---|
| Export admission/snapshot regressions before repair | Failed on all 24 revoked-administrator cases and all four erased-membership cases; retained-history controls did not fail | `evidence/export-auth-before.log` |
| Same export regressions after repair | PASS, 3.093s | `evidence/export-auth-after.log` |
| Web export interruption and listing pagination before repair | Failed for CSV/JSON read/finalization errors and all three pagination cases | `evidence/web-before.log` |
| Focused web candidate, validation/reset rendering, refusal and logging controls | PASS after correcting a fixture to expect the existing filename prefix in export errors | `evidence/web-after.log` |
| Complete message package including 123 new entries | PASS, 0.012s | `evidence/ux-messages.log` |
| Duplicate-definition real-command regressions before repair | Failed as expected in both duplicate cases | `evidence/traceability-duplicates-before.log` |
| Complete traceability package with race detector | PASS, 7.743s | `evidence/traceability-duplicates-after.log` |
| Traceability vet, formatting, repository freshness | PASS on the isolated traceability repair | `evidence/traceability-duplicate-recovery.md` |
| Broader export race regressions, including admission, erasure and retained-history controls | PASS, 83.094s | `evidence/export-race.log` |
| Marketplace message package and actual production-template regressions with race detector | PASS, 1.074s / 10.473s; expected failures captured before the repair | `evidence/marketplace-*-before.log`, `evidence/marketplace-*-after.log` |
| Real TCP client, committed CSV/JSON export interruption | Baseline FAIL / repaired PASS (0.864s / 0.488s) | `evidence/committed-http-before.log`, `evidence/committed-http-after.log` |
| Amount parser fuzz | PASS, 100,000 executions / 4.476s | Tool receipt reproduced in `evidence/fuzz-amount-receipt.md` |
| Full candidate race run on local 5da8bc7 | Interrupted by executor restart; no result receipt | `evidence/verification-attempts.json` |
| CGO-disabled browser binary build and dependency audit | Both killed by the executor; not passing | `evidence/verification-attempts.json` |
| Full candidate Go/race, contract, browser and static/security gates | Still required | Exact published candidate must qualify |

The first web after-run exposed a test expectation error: the existing exporter
wraps a read failure as `export members.csv: ...` or `export full.json: ...`.
The fixture now requires that exact contextual error and still requires one
request-correlated log, the abort sentinel, no valid partial ZIP, and no HTML
suffix. No production assertion or error handling was weakened.

Current main's historical fuzz job failed with the Go coordinator's duration
deadline after 41,458 strict-JSON executions; no failing input was recorded.
Active PR #90 already replaces duration budgets with explicit execution counts.
Its source-head successes and older checks must not be relabeled as passes for
this candidate or the latest #90 tree.

Update this record and retain exact command logs after broader checks finish.
CI receipts must identify the source head and provider checkout/merge revision.
