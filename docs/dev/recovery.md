# Account recovery runbook (SEC-008)

Circula keeps no email address, so recovery uses codes and people. Design and
deviations: `docs/specs/addenda/SEC-008.md`.

## What members should do now

Sign in, open **Recovery codes** (`/account/recovery-codes`), enter the
password, and write down the ten codes. They are shown once. Making new codes
replaces all old ones. Keep the sheet away from the password manager.

## Case 1: member lost the password but has a code

1. Open `/recover` (link it from your sign-in help).
2. Enter login, one unused code (case and dashes do not matter) and a new password.
3. Everything the member had is signed out (sessions and API tokens). The
   remaining codes are deleted: sign in and make a new batch.

Five wrong attempts for a login lock recovery for that login (15 minutes,
doubling up to 24 hours). The normal sign-in is not affected. An unknown login
looks exactly like a wrong code, so do not tell members "that login does not exist".

## Case 2: member lost the password and the codes

Ask a network administrator (Network admin, Members, **Reset password**).

1. Check the person's identity out of band (in person, phone you know).
2. Give a reason (it is recorded). The page shows a temporary password once.
3. Hand it over in person or by a trusted channel. Do not paste it into chat
   history you do not control.
4. The member signs in with it and is sent to the password page; every other
   page refuses until the password is changed. All their sessions, API tokens
   and recovery codes were revoked: they should make new codes afterwards.

Rules: you cannot reset yourself; only active members of your network;
10 resets per hour per administrator; resetting another administrator is
recorded as `high_risk`. Look for `member.password_reset` in the network audit
log and `person.password_reset` in the platform log.

## Case 3: the only administrator lost access (or an administrator lost everything)

Another administrator of the same network resets them (Case 2). If there is no
other administrator, the platform operator does it on the server:

```
systemctl stop circula      # or however you run serve; the command needs the database lock
circula reset-password -login ana@example.org -reason "ticket 42: lost password and codes, verified by phone"
```

The temporary password is printed once on stdout (never logged; do not run
under a recorder that stores it). Restart `serve`. The person signs in, is
forced to change the password, then should make new recovery codes. The event
is `person.password_reset` with actor type `platform_operator` and your reason.
Verify identity as strictly as for an administrator reset: the operator can
take over any login.

## What gets revoked by every reset path

Password, all sessions, all API tokens, all recovery codes, the login's and
recovery throttles. Forced change flag set (admin/operator) or cleared
(self-service recovery, `ChangePassword`).

## Troubleshooting

| Symptom | Cause |
|---|---|
| "login or recovery code is not correct" | wrong code, used code, unknown login or typo; all look the same on purpose |
| "too many attempts" on `/recover` | login locked by wrong guesses; wait or use an administrator reset (it clears the lock) |
| API returns 403 `password-change-required` | the person was reset; they must change the password in the web UI (`/account/password`) or `POST /api/v1/me/password` with a session |
| Administrator sees "too many password resets" | 10 per rolling hour; wait or ask another administrator |
| `reset-password` says database in use | stop `serve` first |

## Operating notes

- Codes are 80-bit random and stored as salted SHA-256; they are not
  recoverable. A lost sheet means Case 2.
- Nothing in audit, logs or error pages contains a code, a password or a temporary password.
- Migration `0004_recovery.sql` adds `persons.must_change_password` and
  `account_recovery_codes`; it is forward-only (take a backup first, as `migrate` does).
