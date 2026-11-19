# Ledger aggregate cancellation after integer overflow

## Finding

The SQLite verifier's `sumSaturating` fallback latched an int64 endpoint on
the first overflowing prefix and ignored every later value. Per-account
bounds do not bound the sum of an arbitrary prefix across a network. For
`B = 2^62`, `[B, B, B, -B, -B, -B]` is balanced and each value is allowed by
the account-bound trigger, but the old fallback returned `MaxInt64`.
Reversing the signs returned `MinInt64`; a later cancellation could also
leave a small nonzero total or change the final sign.

`Totals` feeds the actual consistency checker. The defect can report a
balanced network as inconsistent. It does not change monetary records.
The claim in historical migration 0015 that individual bounds prevent
whole-network SUM overflow is insufficient; that migration remains intact.

## Repair

The SQL SUM fast path remains. When SQLite reports integer overflow, Go
accumulates every row exactly with `math/big.Int`, checks the row iterator
error, and clamps only a final total outside int64. In-range results retain
their exact value, including zero and both int64 endpoints. Genuine
out-of-range totals remain nonzero, so corruption is still reported.

No schema, posting, cached balance, ledger entry, audit row, or historical
policy changes. The exceptional fallback uses space proportional to the
size of the aggregate integer, without loading all rows into memory.

## Regression evidence

`internal/store/totals_overflow_test.go` covers eleven signed order and
boundary cases, including cancellation after positive/negative overflow,
alternating order, residual +1/-1, a sign reversal, exact int64 endpoints,
and final out-of-range totals. A separate persisted balanced four-account
fixture calls `Service.VerifyLedger`; it keeps every schema guard and
does not edit historical entries. Large repository postings keep that
fixture small; current service transaction limits are lower.

SQLite reproduction confirmed the three overflow examples above. Source
review independently approved the implementation and regression design.
Go execution is pending the shared environment's SQLite compilation:
both concurrent and serial initial compilation were killed for memory
pressure. This note does not claim runtime qualification until test logs
are recorded by the integration audit.

## Integer-only QA guard follow-up

The first lightweight domain run caught an overbroad QA rule: it rejected
every `math/big` import, including exact integer arithmetic. The guard now
resolves import aliases and permits only `Int` and `NewInt` from that
package. Dot/blank imports, Float/Rat APIs, and inferred float conversions
remain rejected. Fourteen guard fixtures cover accepted integer uses and
unsafe aliases/conversions. The existing ordinary float and SQL column
guards remain in place.

After this repair, `go test -buildvcs=false -p=1 -count=1 ./internal/domain`
passed locally (0.148 seconds). This qualifies the domain and guard tests;
the SQLite-backed ledger regressions still require integration CI.
