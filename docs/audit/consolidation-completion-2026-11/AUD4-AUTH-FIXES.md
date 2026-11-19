# AUD4-F06 and AUD4-F04: authentication fixes (AUD4-F02 pending owner decision)

Found by this pass's independent ledger and authorization audit, against `main@ffa2758`. The fix was written against `main@28ed716`; the merge commit is recorded in the merge manifest.

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

## AUD4-F02: administrator takes over a reviewer account (Medium, CWE-284/639): owner decision required

The SA2-01 guard stops network administrators resetting another administrator's password, but not a reviewer's. An
administrator who is a party to a dispute can reset a reviewer who has no second factor, sign in with the temporary
password, and decide their own dispute (probe `TestAUD4_F02_AdminResetsReviewerThenDecidesOwnDispute`: outcome
"upheld"; the administrator's balance went from -5000 to 0). That defeats reviewer independence (DOM-038) for
disputes and GOV-016 challenges.

This finding is **not fixed in this PR**. The independent consolidation review noted two things:

- Extending the operator-only recovery rule (SA2-01, SEC-008 addendum) to reviewers is a new recovery-policy choice.
  DOM-038 and SEC-010 do not select it.
- A current-role guard alone is incomplete. An administrator can revoke the reviewer role, reset the now-ordinary
  account, then grant the role again. This was found by reading the source and has not been executed.

Candidate policies for the owner: operator-only resets for reviewers (a draft change is kept in a separate
policy-pending PR); a required second factor for reviewers; or making a reviewer whose password an administrator
reset ineligible for decisions in which that administrator is a party.

## Fail-before evidence

With the `auth.go` changes stashed, all three new tests fail. With the fix applied, they pass.

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
