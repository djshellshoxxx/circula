# Operator runbook

Short procedures for the common tasks. The details are in the linked documents.

## First start on a new host

1. Install the binary from a verified release (`sha256sum -c SHA256SUMS`).
2. Write a configuration file (see [configuration.md](configuration.md)) and check it:
   `circula serve -config /etc/circula/circula.json -check-config`.
3. Create the database: `circula migrate -config /etc/circula/circula.json`.
4. Create the first operator: `printf '%s\n' "$PASSWORD" | circula admin create-operator -config ... -login NAME`.
5. Start the service under your service manager, with `serve -config ...`.
6. Check `/healthz`, `/readyz`, and that `/metrics` is refused from outside the allowed range.
7. Take a first backup: `circula backup <backup-dir>`, and copy it off the host.

## Daily checks

- `/readyz` returns ready.
- `circula_backup_last_success_timestamp_seconds` is less than 26 hours old.
- No critical alert is open.

## Ledger verification

```sh
circula verify-ledger -config /etc/circula/circula.json
```

Exit `0`: every network passed. Exit `3`: at least one network failed. Capture the output and follow SEC-030
([incident-response.md](incident-response.md)). Do not edit the database by hand.

`verify-ledger` and `admin create-operator` take the single-writer lock (`<database>.lock`) and refuse with "in use by
another circula process" while `serve` runs: stop the service first (CMD-02). Verification of a running service is done
by the hourly job and the admin consistency check in the UI and API, which record their runs.

## Audit anchor (RB-010)

Weekly (or after each backup), copy the newest `<backup>.anchor.json` or run `circula audit-anchor write <file>` and keep the
file off the database host. To check for tampering with an older anchor: `circula audit-anchor verify <file>`. Exit `3` means
the live audit chain no longer contains the anchored head: treat as an integrity incident, preserve the database file and the
off-host backups, and restore a backup whose `verify` passes. After any restore, run `verify` against the latest anchor.
See [configuration.md](configuration.md#audit-anchors-rb-010).

## Maintenance mode

```sh
circula maintenance -config /etc/circula/circula.json -retry-after 600 on   # writes answer 503 within a second, reads keep working
circula maintenance -config /etc/circula/circula.json status
circula maintenance -config /etc/circula/circula.json off
```

Details in [upgrade.md](upgrade.md#maintenance-mode).

## A network is frozen

`circula_frozen_networks > 0` or a log line "network frozen": that network failed ledger verification. Its members can
read but not write (API `423 network-frozen`, web "Network read-only"). Other networks are unaffected. Stop `serve`, run
`circula verify-ledger -network <slug>`, capture the findings (SEC-030) and decide with the product architect. If you
restore a verified backup, the network is verified again at start and serves normally when it passes.

## Suspending a network (GOV-024)

Only for a legal order, a serious security incident, persistent abuse of the platform, or non-performance of a written
hosting agreement. Never for a reason about members' trades or balances.

    circula suspend start -network <slug> -category legal_order|security_incident|platform_abuse|hosting_agreement -reason "..." -operator "<your name>"
    circula suspend lift  -network <slug> -reason "..." -operator "<your name>"

Run with the service stopped (it takes the same lock as `break-glass`). Trading and posting stop; members keep read and
export access. Each network administrator gets an inbox notice, and the event (`network.platform_suspended`) is written
at platform scope and to the network's audit log, where its administrators can read it. The network's own administrators
cannot lift a platform suspension, and lifting it does not lift an administrator suspension. DEC-014 allows withholding
the notice where law prohibits it; this build always sends it.

## Restoring after a fault

See [backup-restore.md](backup-restore.md#restoring). Stop the service first. `restore` needs `backup-key` (and
`backup-encryption-key` for encrypted backups) in its configuration; without `backup-key` it refuses unless you pass
`-allow-unsigned` for a file you trust (SA2-11).

## Unlocking a locked-out person

Repeated wrong passwords lock a login for a while. To clear the lock before it expires, with `serve` stopped or running as
the command allows:

```sh
circula unlock -config /etc/circula/circula.json -login NAME
```

The command clears the throttle for that login only and is audited. It does not reset the password; use the recovery flow
for that. Confirm the person's identity out of band first.

## Lost authenticator and recovery codes

After verifying the person out of band, with `serve` stopped:

```sh
circula reset-mfa -config /etc/circula/circula.json -login NAME -reason "ticket 42: lost phone and codes"
```

This removes the second factor, recovery codes and pending challenges, ends every session, notifies the person and writes an
audit event as the platform operator. The password is untouched: also run `circula reset-password` if it may be
compromised. `circula unlock` also clears the password-guess lock used by recovery-code and MFA setup.

## Signing everyone out (emergency revocation, SEC-027, OPS-010)

Use this after a suspected leak of session cookies, of the database file or of the web signing key, after a host
compromise, or as part of rotating the web signing key. Sessions are server-side rows: deleting or replacing the key file
alone does **not** end them, only this command does. With `serve` stopped (the command needs the database lock):

```sh
# every session of every person, and a fresh web signing key
circula revoke-sessions -config /etc/circula/circula.json -reason "incident 7: suspected cookie leak" -rotate-web-key

# only one person's sessions (for example a lost laptop)
circula revoke-sessions -config /etc/circula/circula.json -login NAME -reason "ticket 42: lost laptop"
```

- `-reason` is required (3 to 500 characters) and is recorded in the platform audit event `sessions.revoked` (platform
  operator), together with the number of sessions and the scope (`all` or `person`). No secret is printed or stored.
- `-rotate-web-key` also replaces `<state-dir>/circula-web.key` atomically (mode `0600`), so CSRF tokens, flash messages
  and session-start cookies issued before stop validating. It cannot be combined with `-login`.
- The ledger and every other record are not touched (tested: `TestRevokeAllSessions`, `TestRevokeSessionsCommand`).
  API tokens are separate credentials and stay valid; people revoke their own tokens, and a password reset
  (`circula reset-password`) revokes a person's tokens too.
- Start `serve` again. People sign in again; nothing else changes for them.

## Reverse proxy checklist

- Set `trusted-proxies` to the proxy address, or every client shares one rate-limit bucket (`serve` warns at start).
  IPv6 clients are limited per /64.
- Keep `cookie-secure=true`: cookies carry the `__Host-` prefix and plain-name cookies are ignored over HTTPS. A proxy must
  not rewrite or strip `Set-Cookie` attributes (`Secure`, no `Domain`, `Path=/`).
- Block `/metrics` at the proxy and set `metrics-token-file`.

## Releasing

Tag only a commit that is already on the default branch: the release workflow fails when the tag's commit is not an
ancestor of `origin/<default branch>` (SA2-19). Protect `v*` tags in the repository settings so only maintainers can
create them, and keep `docs/releases/<tag>.md` in the tagged commit.

## Upgrading

See [upgrade.md](upgrade.md).

## Adding an operator

```sh
printf '%s\n' "$PASSWORD" | circula admin create-operator -config /etc/circula/circula.json -login NAME
```

The command creates a person. It does not grant a network role. Network roles are assigned inside each network by its
administrators.

## Incident contacts and severity

The incident procedure is [incident-response.md](incident-response.md) (SEC-030): severity levels, roles, detection,
containment, evidence, notification, recovery and review. Its levels apply to incidents and support requests alike
(OPS-030):

- `critical`: ledger integrity or security at risk, or the instance down. Page the on-duty platform operator.
- `high`: one network cannot post, or a safeguard (backups, disk, throttling) is degraded.
- `normal`: other faults and support requests.

The response targets there are recommendations, not service levels, until the platform operator approves them
(DEC-022). Contacts and the people holding each role are recorded outside this repository. Each alert in
[monitoring.md](monitoring.md#alerts-ops-015) names its level.
