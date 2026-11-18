# AUD4-F06, AUD4-F04, AUD4-F02: authentication fixes

Found by this pass's independent ledger and authorization audit, against `main@ffa2758`. Fixed on `main@28ed716`.

## AUD4-F06: sign-in that straddles a password change (Medium, CWE-367)

`LoginStep` read the password hash in one write unit and ran Argon2 outside any unit. It then inserted the session
in a second unit without re-reading the hash. A change, recovery or reset that committed between those steps revoked
every session, and the sign-in then created a fresh session for the superseded password. That breaks SEC-003
("after a password change, a session from another browser is rejected"). `ChangePassword` had the same gap: two
changes that both verified the old password could both apply.

Fix (`internal/service/auth.go`):

- `LoginStep` re-reads the person and hash inside the session-issuing unit and refuses if either changed. Write units
  are serialised, so a change commits either before this check, and is seen, or after the session exists, and
  revokes it.
- `ChangePassword` replaces only the hash it verified (compare-and-swap).
- `ChangePassword` deletes pending second-factor challenges, as recovery and resets already do (SA2-13).

Regressions:

- `TestLoginStraddlingPasswordChangeGetsNoSession` covers self-service change and administrator reset. It is
  deterministic through the `ObserveArgon2` seam.
- `TestChangePasswordReplacesOnlyTheVerifiedHash`.

## AUD4-F04: change-password guess rate (Low, CWE-307)

`ChangePassword` compared the current password without a durable throttle. The only limit was the API's in-memory
limit of 5 per minute, which a restart resets. A stolen session could make about 300 guesses an hour.

Fix: count the attempt with `reserveAttempt(recgenKey(person))` before comparing. This is the same budget as
`checkCurrentPassword` and recovery-code generation (A3-03). A successful change clears it.

Regression: `TestChangePasswordGuessesAreThrottled`.

## AUD4-F02: administrator takes over a reviewer account (Medium, CWE-284/639)

The SA2-01 guard stopped network administrators resetting another administrator's password, but not a reviewer's.
An administrator who was a party to a dispute could reset a reviewer who had no second factor, sign in with the
temporary password, and decide their own dispute. That defeats reviewer independence (DOM-038) for disputes and
GOV-016 challenges.

Fix (`internal/service/recovery.go`): a reviewer's password, like an administrator's, can only be reset by the
platform operator (`circula reset-password`). Ordinary members are unchanged.

Regression: `TestAdminCannotResetReviewerPassword`.

**Owner note:** this widens the SA2-01 operator-only rule to reviewers. It is the conservative, reversible choice.
Requiring a second factor for reviewers is an alternative the owner may prefer.

## Fail-before evidence

With the `auth.go` and `recovery.go` changes stashed, all four new tests fail. With the fix applied, they pass.

## Not fixed here (owner decisions)

- **AUD4-F01 (Medium):** the dual-control independence check (A3-01) runs only in one direction. An administrator
  can appoint a second login and approve that login's above-threshold request. Refusing both directions would make
  dual control impossible in networks where every administrator descends from the founder. The
  [SEC-011 owner-decision appendix](../../specs/addenda/SEC-011-owner-decision.md) treats this as an accepted side
  effect, and `TestA3_01_IndependentApproverSucceeds` encodes it. It needs an owner decision: configurable quorum,
  operator-rooted second approver, or a proposer-side approval delay.
- **AUD4-F07 (Info):** role grants and revocations do not revoke or rotate sessions, as SEC-003 requires. There is no
  practical effect, because roles are re-read on every request. Either amend the spec or revoke on role change.
- **A3-F02 (DEC-014):** still reproducible. A break-glass administrator can also reset member passwords, reverse
  trades and create 90-day invitations, in addition to the known adjustment, credit-limit and approval paths.
