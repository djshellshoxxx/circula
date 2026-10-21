# Quality thresholds (QA-040, DEC-023)

Status: PROPOSED. The product owner has not confirmed DEC-023. Per QA-040 the release gate MUST NOT pass with an
unconfirmed value, so `scripts/release-gate.sh` reports "thresholds unconfirmed" as a blocking item until the
`Decision` column below is filled with a recorded decision reference and date.

| Threshold | Value | Source | Measured on this repo | Decision |
|---|---:|---|---|---|
| Ledger invariant failures in any run | 0 (absolute) | QA-040 | tests green | n/a (absolute) |
| Authorization matrix mismatches | 0 (absolute) | QA-040 | tests green | n/a (absolute) |
| Failing P0 tests | 0 (absolute) | QA-040 | tests green | n/a (absolute) |
| Property-test long run (merge/release) | 100,000 sequences | QA-001 recommended | short run in CI only | UNCONFIRMED |
| Interleavings per CI configuration | 100 | QA-007 recommended | not 100 yet | UNCONFIRMED |
| 100 disjoint concurrent trades finish within | 60 s, zero deadlocks | QA-008 (test `TestQA008DisjointConcurrentTrades`) | ~6 s under -race, 4 vCPU | UNCONFIRMED |
| Branch coverage, posting/balance/credit-limit code | >= 90 % | QA-040 recommended | not measured | UNCONFIRMED |
| Throughput / latency targets | none claimed | QA-040: set on recorded reference hardware | docs/qa/LOAD-TEST.md (about 250-360 trades/s, single writer) | UNCONFIRMED |
| Login timing difference, existing vs missing account | per SEC-005 | QA-040 | not measured | UNCONFIRMED |
| Migration of fixture with >10,000 entries | time recorded | QA-027 | golden migration test exists | UNCONFIRMED |

How to confirm: record a decision in `docs/DECISIONS.md` (DEC-023) with the final numbers, then replace each
UNCONFIRMED with `DEC-023 <date>`. The release record (docs/releases/vX.Y.Z.md) copies this table.
