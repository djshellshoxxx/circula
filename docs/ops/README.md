# Operations guide

This guide covers running a Circula instance. Specifications it implements: [`docs/specs/11-deployment-operations.md`](../specs/11-deployment-operations.md).

| Document | Use it for |
|---|---|
| [configuration.md](configuration.md) | Every setting, its default, and how to set it |
| [backup-restore.md](backup-restore.md) | Taking, checking, pruning and restoring backups; restore drills |
| [upgrade.md](upgrade.md) | Upgrading, migrating and rolling back |
| [monitoring.md](monitoring.md) | Health and readiness endpoints, metrics, alert conditions, failure modes |

## Environments (OPS-004)

| Environment | Where | Data |
|---|---|---|
| Development | Developer machine | Synthetic only |
| Continuous integration | Ephemeral runners | Synthetic only |
| Staging (hosted only) | Mirrors production configuration | Anonymised copy; see SEC-014 and SEC-018 |
| Production | Hosted or self-hosted | Real |

Production data must never be copied to development or CI.

## Deployment shape (OPS-002)

- One `circula serve` process per database file. A second process refuses to start while the first holds
  `<database>.lock`.
- The database is a single SQLite file in WAL mode. Its directory must be writable.
- Put a TLS-terminating reverse proxy in front of the listener. Do not forward `/metrics` or `/healthz` to the public
  internet. `serve` refuses an `http://` `public-base-url` on a non-loopback address unless `allow-insecure-http` is set.
- Tell the service about the proxy: set `trusted-proxies` to the proxy's address and make the proxy send
  `X-Forwarded-For` (client address) and `X-Forwarded-Proto: https`. Without `trusted-proxies` all clients share the
  proxy's address in the rate limiters (one noisy client can then block sign-in for everyone), and forwarded headers are
  ignored. Keep `cookie-secure=true` (default): cookies are then `Secure` and `__Host-` prefixed, and the service sends
  `Strict-Transport-Security` when `public-base-url` is `https://`.
- Set `public-base-url` to the exact address users type. Invitation links use it, and form posts are accepted only with
  that `Origin`.
- Files: the process sets umask `077`. The database, its WAL files, backups, manifests and the lock file are owner-only
  (0600) and `backup-dir` is 0700; `serve` tightens a database file or backup directory it finds wider. Keep the
  database directory itself private to the service account.
- Set a setup secret (FR-002) before exposing the first run, see [configuration.md](configuration.md#secrets).
- Backups go to `backup-dir` on the same host by default. Copy them off the host (OPS-019).

## Ownership

Roles are named in the alert and failure-mode tables. The platform operator role owns the instance; the product architect
owns the specification. Names of people are kept outside this repository.

## Known gaps

These are not implemented in the current `circula` build. Each one is a deliberate deferral or a missing service hook. Do not
rely on them until the gap is closed and this list is updated.

- **Restore audit event (OPS-020).** `restore` writes a local record in the backup directory (`restore-<time>.json`). The
  ledger has no restore event yet.
- **Maintenance mode (OPS-024).** There is no command that refuses writes while the service stays up. Upgrades stop `serve`
  instead.
- **Per-network freeze (OPS-033).** A ledger verification failure at start-up stops the instance from serving, rather than
  freezing one network. The service layer has no per-network posting gate yet.
- **Pre-migration verification (OPS-023).** Only the post-migration verification runs. The pre-migration backup is the
  rollback point.
- **Posting metrics (OPS-014).** Posting attempts, successes, failures, idempotent replays and authentication failures are
  not counted inside the service. `/metrics` counts HTTP 401 and 429 responses instead.
- **Mail (OPS-029, DEC-009).** No mail settings exist yet.
- **Secrets from files (OPS-009).** No setting currently holds a secret. The operator password is read from
  `CIRCULA_OPERATOR_PASSWORD` or stdin only.
- **Cookie security.** `cookie-secure` is parsed and reported, but the web adapter does not read it yet.
- **Log level without restart (OPS-032).** The log level needs a restart.
- **Disk-usage percentage.** `/metrics` reports file sizes, not the percentage of the volume in use. Use the host's
  monitoring for the disk alert in OPS-015.
- **Backup schedule.** `backup-interval` is a fixed interval, not a cron expression.
- **Long streaming responses.** `request-timeout` bounds each request, so an export that takes longer is cut off.
