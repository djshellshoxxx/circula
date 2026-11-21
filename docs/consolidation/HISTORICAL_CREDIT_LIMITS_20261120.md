# Historical credit-limit verification — 2026-11-20 Pacific

OPS-016 requires checking a posted debit against the limit that applied when it
was accepted. Today's policy or an expired/replaced account override cannot
establish that historical fact. This implementation records future decisions
and reports the older evidence gap explicitly.

## Posting and adoption boundary

Migration `0024_historical_credit_limits.sql` adds immutable
`posting_credit_checks`. During that one migration it marks the exact identities
of existing negative entries as legacy-unverified, with all decision inputs
NULL. Runtime writers cannot create another legacy marker, replace an existing
row, update it or delete it. Posting time and sequence values do not decide
whether a later entry is legacy.

For every new negative entry, the central posting operation records the actual
policy version, checked time, prior balance, system-account status, override,
override expiry and effective limit in the same write transaction as the two
entries, balances, transaction state, audit and idempotency effects. Failure to
write the evidence rolls the operation back. Durable replay creates no second
posting or witness. Trades, reversals, adjustments and grants use the same path.

## Verification and coverage

The shared verifier streams all entries in account and ledger order and
reconstructs prior balances with exact integer arithmetic. For each witnessed
debit it compares the recorded account, time and prior balance, resolves the
immutable policy version, applies the recorded override at the exact expiry
boundary, and evaluates the resulting debit. Incoming credits improve the
position and are not rejected merely because a later limit reduction leaves
the account below its current limit. System debits retain their explicit
exemption.

Unusable scalar inputs, missing policy evidence and invalid policy objects or
default limits cannot be presented as checked evidence. Legitimate omitted
fields retain the application's established historical policy defaults. A read
error returns an error, never a partial successful report.

| Result field | Meaning |
|---|---|
| `status: complete` | Every debit has usable evidence or its explicit system exemption. The separate invariant result may still fail. |
| `status: partial` | At least one debit is legacy-unverified or has unusable forward evidence. |
| `checked_debits` | Debits evaluated against their historical limit. |
| `system_debits` | Witnessed system debits exempt from credit limits. |
| `legacy_unverified_debits` | Exact pre-adoption debits whose old decision inputs were not retained. Their existence alone is not an invariant failure. |
| `unverifiable_debits` | Forward debits with missing or unusable evidence; these also produce findings. |
| Missing coverage | An older saved run did not record this information, or an aggregate contains unknown coverage. It does not mean complete. |

The coverage status is independent of `OK`. Existing findings and verification
errors still determine failure and consistency-freeze behavior. Migration
admission skips the unavailable witness check only for schemas before 24; it
requires the check after adoption.

## Operator and member-facing surfaces

New consistency runs persist the optional coverage object. API checks, the CLI,
administrator check results and the retained administrator overview expose it.
CLI and backup aggregation exclude the platform audit scope and preserve unknown
coverage if any financial scope is unknown. Counts are checked for invalid
values and overflow.

Backup manifests omit the field when unknown so older canonical payloads and
signatures remain valid. New version 2 signatures authenticate every coverage
field. The older version 1 signature payload remains unchanged. These additions
do not change export schema 1, entry CSV fields or closure checkpoint version 1
bytes; an exported balance is not silently promoted to a historical-limit proof.

## Source and qualification

Core source `48bb4c6` is integrated as `2d3a163`; the decoder and service
regressions in `61ed3e7` are integrated as `e3f2a5d`, with final focused evidence
cases from `6bb9750` integrated as `56ec7f5`. Consumer source `db27af0` is
integrated as `5c389fe`.

The initial core selection passed four race tests. The final expanded core
selection at clean `6bb9750` passed 30 top-level tests and 32 subcases under race:
integrity 1.012 seconds, store 13.704 seconds and service 25.323 seconds. It
includes exact expiry, all posting kinds, complete rollback/replay, saved-run
coverage, large legacy totals, read-only checks, original exports and closure
compatibility. The source stayed clean and unchanged throughout the run.

The consumer selection passed 23 top-level tests and 22 subcases under race,
including real snapshot propagation, old signature compatibility,
authentication of every new field, API authorization and retained HTML states.
Independent review's policy-decoder finding is repaired and covered by the
later core selection; legitimate omitted defaults remain supported. The final
combined normal/race and provider gates are recorded separately in
[the continuation](UNIQUE_CORE_CONTINUATION_20261119.md).

The old history gap remains a data limitation after successful qualification.
No current policy, clock inference or fabricated override fills it.
