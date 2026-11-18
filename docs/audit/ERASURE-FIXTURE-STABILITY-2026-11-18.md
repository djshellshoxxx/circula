# Erasure regression fixture stability — 2026-11-18

## Finding and deterministic reproduction

Aggregate race qualification reported `TestAnonymiseMemberKeepsLedger` failing
because `"Ana"` remained in `persons.password_hash`. The test scans every
database column with SQLite `LIKE`; an unrelated member's random Argon2 salt
or digest can contain that three-letter sequence. This is a test false positive.

At base `ef54238ee9ebc2184919b6a8ec30c3398df55ed5`, assigning Ben a valid
fixture password hash with the salt `AnaAAAAAAAE` reproduces the exact failure.
Ben successfully signs in with the fixture password before the failed scan.
No production erasure behavior is changed to obtain this reproduction.

## Test repair

The fixture uses `Ana García-López` and `zucchini harvest` as the name and
authored-text markers. Their spaces and punctuation cannot appear in base64
salts or digests. The test confirms that the login, name, shared text marker,
complete memo, listing text and dispute statement are present before erasure
and absent afterward. The same values are checked in the export.

Every database column, including `password_hash`, remains in the scan. The
target password must become the unusable marker, and its original hash must
disappear from the database. Ben's and the administrator's stored hashes must
remain byte-for-byte unchanged. Ben's deliberately colliding credential must
still authenticate after erasure. Existing ledger, identity, session, audit,
authorization and irreversible-erasure assertions remain in place.

## Verification

Using Go 1.26.11:

- The deterministic collision fails with the original scan and passes with
  the repaired fixture; the repaired test passes 20 consecutive runs.
- All 11 tests selected by `(Anonymise|Erase|Erasure)` pass twice, both normally
  and under `go test -race ./internal/service`.
- `go vet ./internal/service` passes.
- Ten temporary fault injections after erasure are detected: retained name,
  login, memo, name-only memo fragment, shared-text-only memo fragment, listing
  title, listing description, dispute statement, target password hash, and
  replacement of an unrelated password hash. The test file is restored
  byte-for-byte after those probes.

Scope is the seeded erasure workflow. This test-only repair introduces no
database migration, runtime behavior or policy change.
