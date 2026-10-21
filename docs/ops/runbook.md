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
