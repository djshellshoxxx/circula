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

See [backup-restore.md](backup-restore.md#restoring). Stop the service first.

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
