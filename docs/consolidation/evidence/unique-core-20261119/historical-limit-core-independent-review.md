# Historical credit-limit evidence: independent core review

## Scope and provenance

Source reviewed: `48bb4c69b43b392cdbaf3db81dbc186120d3a9ba`, integrated by the
parent as `2d3a163`. The review checkout contains the same atomic commit as
`cbcbaeb`, before the consumer-only implementation. Review covers the new
repository contract, migration 0024, posting witness capture, streaming reader,
shared evaluator, migration gate option, saved-run serialization and overview
repository reader. The parent owns final integrated verification.

This was source review. It did not execute malformed-database scenarios,
disable storage protections, reproduce an exploitation workflow, or test a
live target. The core author separately reported four initial focused race
passes; this report does not independently qualify that log yet. Consumer
tests are an author-run qualification of the reviewer's separate consumer
implementation, not independent review of that implementation.

## Finding: null or unusable policy can be labelled usable evidence

**Status at this checkpoint: confirmed by the core author; bounded repair
approved by the parent and in progress.**

`internal/store/sqlite_credit_limits.go`, in `CreditLimitHistory`, initializes
`domain.DefaultPolicy()`, unmarshals the saved `policy_json` into it, then sets
`r.Policy` when unmarshalling returns no error. JSON `null` leaves that default
value unchanged. The reader therefore supplies a non-nil policy for an absent
policy object. It also does not validate the default credit-limit range.
`internal/integrity/credit_limits.go` only tests the effective limit after
applying an override; an otherwise usable override can leave an invalid
default-limit input unexamined.

The resulting risk is a false claim that the historical evidence was usable,
and therefore complete coverage, for a policy representation that cannot
support that claim. Existing application policy writes produce validated
objects, so this is a bounded verification/decoder correctness finding rather
than a claimed ordinary request path for changing policy history.

Recommended repair: require a non-null JSON object for the new historical
policy decoder, retain the existing defaults for legitimately omitted legacy
fields, and validate the default-limit field's domain bounds. Use controlled
decoder fixtures to distinguish valid omitted defaults from null/non-object
and invalid-limit representations. Keep ordinary policy selection and posting
decisions unchanged. Do not expand this repair into a redesign of all policy
readers.

## Other reviewed boundaries

- `internal/service/posting.go` captures the actual payer, policy version,
  millisecond decision time, prior balance, system exemption, override/expiry
  and effective limit in the same update unit as the negative entry. A failed
  witness insert returns through the existing rollback path. The posting
  capacity decision and observer boundary are retained.
- Migration 0024 records the exact existing negative-entry identities with
  no invented limit inputs. It installs the new-marker, replacement, update
  and delete guards after the one-time adoption insert. Later debits cannot
  become legacy merely because their clock or sequence resembles an older
  entry. New witnesses require the referenced negative entry, matching network,
  posting time and system marker, and an existing policy version.
- `CreditLimitHistory` reads through the caller's transaction in stable
  account/ledger-sequence/entry-ID order. It includes positive entries for
  balance reconstruction. The evaluator uses exact `big.Int` accumulation;
  only debits undergo the historical limit condition. Stream and scan errors
  return to the caller without a successful partial report.
- The immutable witness preserves the system exemption actually observed.
  The evaluator also compares it with the current account marker and reports
  a disagreement. Existing account writes update balances or override fields;
  no supported account-conversion path was found. No additional system-marker
  regression was identified.
- The entry-driven stream does not enumerate orphan witness rows. Normal
  writes are protected by entry and policy foreign keys plus append-only
  guards. Hypothetical external changes with constraints disabled were not
  treated as an ordinary-runtime regression or used to justify wider tests.
- Findings and coverage remain separate: a witnessed condition can fail with
  complete coverage; a legacy marker alone produces partial coverage without
  failure. An unusable forward witness produces a finding and increments the
  unverifiable count. Recorded-prior-balance mismatch is a finding while the
  actual debit condition is evaluated against reconstructed history.
- `internal/store/migration_integrity.go` permits only pre-24 schemas to skip
  the then-unavailable witness read. Version 24 and later require it. The
  migration's old debits become explicit legacy markers, so migration does not
  invent historical inputs or historical violations.
- `LastConsistencyRun` returns nil coverage for old SQL NULL values. Its JSON
  errors propagate. The overview repository now reuses that reader within the
  existing view, preventing divergent saved-check projections. The consumer
  service DTO must carry that pointer without translating nil into complete.
- Export schema 1 and closure checksum code are not changed by this core
  commit. The new database table is retained by SQLite snapshots; portable
  export limitations are documented explicitly.

No other concrete source-level regression was identified in this bounded
review. This is AI source review, not GitHub approval, external security
qualification, a full integrated suite pass, or deployment acceptance.

## Decoder repair source-review addendum

Reviewed follow-up `61ed3e7bb5ca9790f728854c671568e14b630b5c`, integrated as
`e3f2a5d` and present at root `f55ab56`. The reported finding is addressed in
source: the new row decoder requires a policy object, marks a null policy or
out-of-range default limit unusable, and retains omitted legacy defaults.
Non-object or malformed policy representations return a decoding error.
Witness override and effective-limit magnitudes also receive their domain
bounds check. The row scanner was extracted without changing the query,
ordering, visitor boundary or `rows.Err()` propagation.

The added decoder fixtures use constant SELECT rows and preserve database
history and guards. They distinguish a legitimate omitted default from a
missing policy, null policy, malformed/non-object policy and invalid limit
under an otherwise positive override. Additional pure-evaluator and service
fixtures cover expiry, exact prior balances, later prospective changes,
posting atomicity and saved-run coverage. No additional concrete regression
was identified in this bounded delta.

This closes the reported finding at the source-review level. The follow-up's
Go qualification was pending when this addendum was written. The reviewer
did not execute a second Go selection or reuse the earlier four-test result
as proof of this later decoder source. Parent will retain its actual result
and exact-source receipt separately.
