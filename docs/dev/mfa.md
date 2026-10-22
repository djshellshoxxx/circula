# Second factor, recovery codes and security notices

Specs: SEC-007, SEC-012. Decisions and test map: `docs/specs/addenda/SEC-007.md`, `SEC-012.md`.

## For people

- `/account/security`: status, replace recovery codes, turn off, security notices.
- `/account/security/mfa/enrol`: Start setup asks for your current password (A3-03), then shows the setup key (and `otpauth://` URI); enter the password again with the 6-digit code to confirm; ten recovery codes are shown once. Confirming ends every other session of the person (the one you are using stays).
- Sign-in with MFA: password, then `/login/mfa` asks for the app code or one recovery code (5 minutes).

## For operators

| Setting | Effect |
|---|---|
| `require-mfa-admins` (default `false`) | Administrators without MFA get `forbidden` / `mfa-enrolment-required` on admin-only operations. Turn it on after announcing it: an administrator must enrol first at `/account/security` (that page works without admin rights). |
| `mfa-secret-file` / `CIRCULA_MFA_SECRET` | Seals authenticator seeds in the database (AES-256-GCM). Set it before people enrol. If lost, sealed seeds are unusable: people disable MFA with a recovery code and re-enrol, or an operator deletes the `person_mfa` row. Without it seeds are stored base32-encoded. |

Lost device and lost recovery codes: there is no self-service path. Verify the person out of band, stop `serve`, then:

```sh
circula reset-mfa -login NAME -reason "ticket 42: lost phone and codes, verified by phone"
```

It deletes the second factor, recovery codes and pending challenges, ends every session, writes a security notice for the person and an audit event (`mfa.reset`, platform operator, with the reason). It leaves the password alone; add `circula reset-password` when the password may be known to someone else. Administrator password resets (`AdminResetPassword`, `OperatorResetPassword`) and `RecoverPassword` do not remove the second factor, only revoke sessions, tokens and pending challenges.

Backups contain the sealed (or raw) seeds and hashed recovery codes: encrypt backups.

## For developers

Service (`internal/service`, files `mfa.go`, `mfa_login.go`, `notices.go`, `totp.go`):

```go
BeginMFAEnrolment(ctx, actor, password) (*MFAEnrolment, error)  // secret + URI; resumes pending for 15 min; password throttled (recgen key)
ConfirmMFAEnrolment(ctx, actor, password, code) ([]string, error) // enables; revokes the actor's other sessions; returns 10 recovery codes once
OperatorResetMFA(ctx, systemActor, login, reason) error         // `circula reset-mfa`
DisableMFA(ctx, actor, password, code) error                    // code = TOTP or recovery
RegenerateRecoveryCodes(ctx, actor, totpCode) ([]string, error)
MFAStatus(ctx, actor) (*MFAStatus, error)
LoginStep(ctx, login, password) (*LoginResult, error)           // Token xor MFAChallenge
CompleteMFA(ctx, challenge, code) (token, *Person, error)
Login(ctx, login, password)                                     // legacy; error code "mfa-required" for MFA people
ListSecurityNotices / CountUnseenSecurityNotices / AckSecurityNotices
```

Adapters that sign people in must use `LoginStep`/`CompleteMFA`. Wrong or replayed codes return `validation-failed` (field `code`, management calls) or `unauthenticated` (CompleteMFA); locks return `rate-limited`.

### Using recovery codes from other flows (account recovery)

```go
ok, err := svc.VerifyAndConsumeRecoveryCode(tx, personID, userInput)
```

Runs inside the caller's `store.Tx`; marks one unused code used atomically (`UPDATE ... WHERE used_at IS NULL`), so concurrent use of one code succeeds once. It writes no audit, notice or throttle: the caller must rate-limit (reuse the `\x1fmfa:`+person throttle via the exported flows or its own key), audit and call `notify`. Returns `false` (no error) for malformed, unknown, used or another person's codes.

### Emitting a security notice

`s.notify(tx, personID, domain.Notice..., detail)` inside the same write unit as the change. Add new kinds in `internal/domain/mfa.go` and the text in `internal/web/h_security.go`.

### Data (migration 0003)

`person_mfa` (sealed seed, enabled, `last_step`), `recovery_codes` (salted SHA-256, `used_at`), `mfa_challenges` (hashed bearer, 5 min), `security_notices`. Throttle rows reuse `login_throttle` with key `"\x1fmfa:"+personID`.

Never log or audit codes, seeds or challenges.
