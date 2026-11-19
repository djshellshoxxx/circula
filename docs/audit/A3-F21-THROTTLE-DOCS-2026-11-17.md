# A3-F21: authentication throttle documentation

Reviewed baseline: `ef54238ee9ebc2184919b6a8ec30c3398df55ed5` (PR #69 merge).

## Finding and correction

The SEC-007 and SEC-008 addenda still described a 15-minute lock that doubled
to 24 hours. SEC-007 also described one shared counter for MFA sign-in and
session-management operations, immediately before its dated A3-F16 amendment
explained why those counters had been separated.

The documents now describe the existing behavior in
`internal/service/throttle.go`, `throttle_config.go`, `mfa.go` and `recovery.go`:

- Default local attempt budget: 5; operators may tighten it to 4.
- Login/recovery budget across client addresses: 100 by default, configurable
  down to the local budget. MFA factor checks use their per-person counters.
- Fixed lock: 15 minutes by default, configurable up to 24 hours in whole
  seconds. Repeated failures do not double its duration.
- Expired locks restore half the attempt allowance. Failure history decays
  after 24 hours beyond the stored lock end.
- MFA sign-in and session-management attempts use separate counters. A
  sign-in lock also refuses management calls; management failures do not
  lock sign-in. A valid factor clears both counters.

The recovery runbook now distinguishes defaults from operator settings and
the per-address budget from the login-wide budget. The dated A3-F16 history
remains recorded. No code, migration, financial rule, retention choice or
break-glass policy changes. A3-F02 / DEC-014 remains unresolved.

## Adjacent recovery corrections

The recovery addendum and runbook also retained instructions that predated
existing hardening. They now state that ordinary network administrators
cannot reset administrator accounts or accounts with any non-departed
membership in another network (SA2-01a). Those accounts require the operator.
Invalid recovery codes are rejected before Argon2 password hashing
(A3-04 / BR-05). Only an admitted attempt with a currently valid unused code
reaches that expensive step. The code is still consumed atomically in the
password-reset transaction. These corrections add no recovery capability.

## Existing regression evidence

The source behavior is covered by:

- `TestConfiguredAuthenticationThrottleLoginAndGlobal`
- `TestConfiguredAuthenticationThrottleRecoveryAndSensitivePassword`
- `TestConfiguredAuthenticationThrottleMFA`
- `TestAuthenticationThrottleRejectsUnsafeServiceConfiguration`
- `TestStolenSessionCannotLockOwnerOutOfMFA`
- `TestMFASignInLockNotRenewableByOneFailure`
- `TestMFASessionThrottleBoundAndUnlock`
- `TestPersistentAuthThrottleSettings`
- `TestPersistentAuthThrottleServiceWiring`
- `TestAdminResetGuards`
- `TestAdminResetRefusesMultiNetworkTarget`
- `TestA3_04_RecoverSkipsHashingBeforeCodeCheck`
- `TestRecoveryRefusalsDoEqualWork`

All 13 named tests passed against the reviewed baseline with this documentation
patch (`go test -count=1`, filtered to these tests in `./internal/service` and
`./cmd/circula`). `git diff --check` also passed. No new regression test was
needed because this patch changes documentation only.

This is a factual documentation reconciliation, not approval of unresolved
security or governance decisions or completion of the broader security review.
