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
