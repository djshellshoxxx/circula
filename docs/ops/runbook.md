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

Exit `0`: every network passed. Exit `3`: at least one network failed. Capture the output and follow SEC-030. Do not edit
the database by hand.

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

Severity levels and response targets are defined in [`docs/specs/10-governance-operations.md`](../specs/10-governance-operations.md)
and SEC-030. Targets are recommendations until the platform operator approves them (OPS-030).
