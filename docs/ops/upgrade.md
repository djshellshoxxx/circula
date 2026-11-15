# Upgrading and migrating

Specification: OPS-022 to OPS-026. Migrations are forward-only. Rollback means restoring the pre-upgrade backup with the
previous binary.

## Before you start

1. Read the release notes for the target version (`docs/releases/<version>.md`). Note any breaking change, required operator
   action, schema change and whether a backup is required.
2. Check the current state:

   ```sh
   circula version
   circula migrate -dry-run          # pending migrations for this binary, nothing changed
   curl -s http://127.0.0.1:8080/readyz
   ```

3. Take a verified backup:

   ```sh
   circula backup /var/backups/circula
   ```

   If the command exits with code `3`, stop and investigate before upgrading.

## Upgrade sequence (OPS-024)

1. Enter maintenance mode so clients get a clear `503` instead of a dropped connection: `circula maintenance on`
   (see [Maintenance mode](#maintenance-mode)). Then stop `circula serve`.
2. Replace the binary with the new version. Check the checksum against `SHA256SUMS` first:

   ```sh
   sha256sum -c SHA256SUMS --ignore-missing
   ```

3. Apply migrations explicitly:

   ```sh
   circula migrate
   ```

   This checks the live database's integrity, takes a `pre-migration` backup and verifies it from disk (see
   [Pre-migration verification](#pre-migration-verification-ops-023)), applies the pending migrations, then verifies every
   network. Each migration runs in
   its own transaction, so a failed migration leaves the database at its previous version. The command exits with code `3`
   if the ledger check fails after migration. In that case, restore the `pre-migration` backup with the previous binary.

4. Start the service. In `explicit` mode (the default), `serve` refuses to start if migrations are still pending.

   ```sh
   circula serve
   curl -s http://127.0.0.1:8080/readyz
   circula verify-ledger
   ```

5. Leave maintenance mode: `circula maintenance off` (a freshly started `serve` also honours the file switch, so turn
   it off after the checks above pass).
6. Record the upgrade: version, time, backup file, and the result of `verify-ledger`.

Rolling upgrades are not supported for a single-node embedded deployment.

## Maintenance mode

Maintenance mode (OPS-024) keeps reads working and refuses writes. Reads (`GET`, `HEAD`, `OPTIONS`) on the web UI and
the API are served as usual, so members can still look at balances and history. Every other method, sign-in and
sign-out included, gets `503 Service Unavailable` with a `Retry-After` header. The web UI shows a short "Back soon"
page; the API returns a `application/problem+json` document with code `maintenance`. Nothing is written by a refused
request. `/healthz`, `/readyz` and `/metrics` keep working, so monitors and load balancers still see the instance
(`circula_maintenance_mode` is `1`). People who are not yet signed in cannot sign in while it is on.

Two switches, either turns it on:

- `circula maintenance on|off|status [-retry-after SECONDS]`: creates or removes `<database>.maintenance` (mode 0600,
  holding the Retry-After seconds, default 300). A running `serve` notices within about a second, no restart. The command
  works with `serve` stopped or running.
- The `maintenance` setting (`-maintenance`, `CIRCULA_MAINTENANCE`, file key) starts the instance in maintenance mode.
  `circula maintenance off` does not clear a setting that is `true`; change the configuration and restart.

If the file exists but cannot be read, the instance stays in maintenance mode (fails closed).

## Pre-migration verification (OPS-023)

Before any migration of an existing database, `circula migrate` and `serve` in `auto` mode:

1. run `PRAGMA integrity_check` on the live database and stop if it is not `ok`;
2. take a `pre-migration` backup;
3. re-read that backup from disk and check it against its manifest: SHA-256 and size, the manifest signature when
   `backup-key` is configured, `integrity_check` ok (recorded and recomputed), and that it holds exactly the schema
   version the live database had.

Any failure stops with `migrations not applied`. The ledger check of the pre-migration copy is `not-run` because it holds
the older schema; the ledger is verified right after the migration. A database with no recorded migrations (a new file)
has nothing to protect and skips the backup.

## Migration timing (DAT-047, DAT-048)

Every release is tested by migrating a deterministic fixture of 10,000 completed transactions across 61 accounts (60
members and the system account, with a valid audit chain) from the first released schema (v1) to the current schema,
then verifying the ledger and audit chain. The test logs the duration; to keep the record per release:

```sh
CIRCULA_MIGRATION_TIMING_FILE=docs/releases/migration-timing.log \
  go test ./internal/store -run TestGoldenMigrationV1With10000Transactions -v
```

Reference run (development container, Go 1.26, v1 to v2): migration 27 ms, ledger verification 201 ms for 10,000
transactions. Real databases with more audit rows or slower disks take longer; plan downtime with margin and measure on a
copy (`circula migrate -dry-run` reports the pending versions without changing anything).

## What changes for operators in this release line

Review these before you restart; each is a behaviour change, not a schema change.

- `restore` refuses a backup it cannot authenticate: configure `backup-key-file`, or pass `-allow-unsigned` for files you
  trust (SA2-11).
- `backup-require-encryption` is now `auto`: with a `backup-encryption-key`, plaintext backups are refused on restore.
  Pass `-backup-require-encryption false` to restore an old plaintext backup.
- `serve` warns when scheduled backups run without `backup-key`.
- New setting `db-synchronous` (`normal` default; `full` for power-loss durability), see
  [configuration.md](configuration.md#durability-db-synchronous).
- With `cookie-secure=true` only `__Host-` cookies are read. People signed in with the old plain CSRF cookie reload the
  page once; sessions are unaffected.
- `circula unlock -login NAME` clears a login lock-out ([runbook.md](runbook.md#unlocking-a-locked-out-person)).
- Per-IP limits key IPv6 clients on their /64; set `trusted-proxies` behind a proxy (start-up warning otherwise).

Upgrades are tested from the first schema: `TestUpgradeFromMigration0001` builds a database as migration 0001 wrote it,
runs `circula migrate` and checks the data, SQLite integrity and foreign keys, and ledger and audit-chain verification.

## Migration mode

- `explicit` (default, recommended, and required for hosted deployments): migrations run only through `circula migrate`.
- `auto`: `serve` takes a `pre-migration` backup and then migrates at start-up. Use it only where a backup is already taken
  before every start.

If the backup needed for `auto` mode fails, migration does not run and `serve` does not start.

## Rollback

Stop the service. Restore the pre-upgrade backup with the previous binary:

```sh
circula restore -database /var/lib/circula/circula.db -overwrite /var/backups/circula/<pre-upgrade backup>.db
```

Add `-allow-unsigned` if no `backup-key` is configured. `restore` refuses a backup whose schema is newer than the binary, so an older binary cannot restore a newer backup. Rolling
back to an older binary is only safe with a backup taken before the upgrade.
