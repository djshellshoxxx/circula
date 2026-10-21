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

1. Stop `circula serve`. There is no separate maintenance mode yet (see the known gaps). Stopping the service refuses
   new requests.
2. Replace the binary with the new version. Check the checksum against `SHA256SUMS` first:

   ```sh
   sha256sum -c SHA256SUMS --ignore-missing
   ```

3. Apply migrations explicitly:

   ```sh
   circula migrate
   ```

   This takes a `pre-migration` backup, applies the pending migrations, then verifies every network. Each migration runs in
   its own transaction, so a failed migration leaves the database at its previous version. The command exits with code `3`
   if the ledger check fails after migration. In that case, restore the `pre-migration` backup with the previous binary.

4. Start the service. In `explicit` mode (the default), `serve` refuses to start if migrations are still pending.

   ```sh
   circula serve
   curl -s http://127.0.0.1:8080/readyz
   circula verify-ledger
   ```

5. Record the upgrade: version, time, backup file, and the result of `verify-ledger`.

Rolling upgrades are not supported for a single-node embedded deployment.

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

`restore` refuses a backup whose schema is newer than the binary, so an older binary cannot restore a newer backup. Rolling
back to an older binary is only safe with a backup taken before the upgrade.
