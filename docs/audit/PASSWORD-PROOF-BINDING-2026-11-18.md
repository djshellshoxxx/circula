# Bind password proofs to protected writes

Reviewed baseline: `6785863252d2910553b1f4a26396f14b4405d6fa` (PR #82).
This is an additional finding beyond the original 55-finding audit ledger.

## Reproduction and scope

`BeginMFAEnrolment`, `ConfirmMFAEnrolment`, `DisableMFA`, and
`GenerateRecoveryCodes` verified a captured password hash outside the later
write transaction. A password change or administrator reset could commit in
between. The request's already authenticated actor remained usable even after
its session was revoked, and the protected action accepted the old proof.

`TestPasswordProtectedChangesRejectSupersededPassword` uses the existing
`ObserveArgon2` test seam to commit the legitimate change during verification.
It covers a self-service change and an administrator reset for each of six
scenarios: fresh enrolment, resumed enrolment, confirmation, disabling with TOTP,
disabling with a recovery code, and generating account recovery codes. All
twelve cases fail against the baseline source: the stale operation succeeds;
enrolment/recovery operations disclose secrets or codes, and all but resumption
also change durable state. Confirmation and disabling still require the valid
factor supplied by the test; this finding does not bypass that requirement.

## Repair

Carry the verified hash into the protected write and read the current hash
there, before returning a pending secret, consuming a factor, changing MFA,
or replacing recovery codes. A mismatch returns the existing password-field
validation error for MFA, or `unauthenticated` for account recovery generation.
The comparison and protected effects share the serialized write transaction.
Argon2 remains outside that transaction.

Existing attempt reservations and throttle families are unchanged. The fix
changes no schema, reset authority, governance rule, or break-glass policy;
A3-F02 / DEC-014 remains unresolved.

## Verification

The new regression snapshots the factor, used/unused recovery-code hashes,
password hash, audit count and security-notice count after the legitimate
password change/reset. The rejected operation preserves that state, returns
no secret, and the original session remains revoked. After a self-service
change, a fresh password proof can reuse the same TOTP/recovery code, proving
the rejected operation did not consume it.

- Baseline source plus the new test: all twelve expected failures;
  service package `0.189s`.
- Fixed source, the same test: all twelve pass; service package `0.179s`.
- Focused race run: the new test and twenty existing MFA/password/recovery
  tests pass; service package `14.099s`, no race report. The existing selection
  covers enrolment, wrong-password validation, shared password throttling,
  factor throttling, scope checks, replay, recovery-code replacement, session
  revocation, and the PR #82 password/login races.

Commands use `GOMAXPROCS=2`, `GOGC=50`, `GOMEMLIMIT=512MiB`, an explicit build
cache, `-p=1`, and `-count=1`. The focused regression command is:

```sh
go test -p=1 -count=1 ./internal/service \
  -run '^TestPasswordProtectedChangesRejectSupersededPassword$' -v
```

Whole-service/aggregate qualification and independent review are separate
integration gates; the focused results above do not claim those gates passed.
