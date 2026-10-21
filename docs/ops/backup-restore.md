# Backup and restore

Specifications: OPS-017 to OPS-021, OPS-034. Backups are produced by SQLite's online snapshot (`VACUUM INTO`), which reads a
consistent view while the service keeps posting. Copying the live database file is not a supported backup.

## What a backup contains

Each backup is two files:

- `circula-<UTC time>.db`: the snapshot. Its name encodes the UTC time to the second.
- `circula-<UTC time>.db.manifest.json`: the manifest. It records:
  - `created_at` (UTC), `reason` (`scheduled`, `manual`, `pre-migration`, `pre-restore`)
  - `app_version`, `schema_version`
  - `bytes` and `sha256` of the `.db` file
  - `integrity_check` (`ok` when SQLite's integrity check passed)
  - `ledger_verification`: `status` (`ok`, `failed`, or `not-run`), network counts, and the reason when not run

Each snapshot is verified before it is named: it must pass the integrity check and, when its schema matches this binary,
the ledger verification. A backup is written under a `.partial` name first and renamed only after every check passed.
A backup whose ledger verification failed is still kept, because it is evidence. Its manifest says `failed`.

## Taking a backup

Manual, to a file or into a directory:

```sh
circula backup /var/backups/circula            # writes a timestamped file into the directory
circula backup /var/backups/circula/before-change.db
```

The command exits with code `3` when the snapshot's ledger verification failed. The backup itself is still written.

Scheduled: `serve` takes a backup every `backup-interval` (default 24 hours) into `backup-dir` and then applies retention.
Retention keeps the newest `backup-retain-daily` backups (default 30) and, for each of the newest
`backup-retain-monthly` months (default 12), the newest backup of that month. Pruned backups are removed with their
manifests and only their names are logged.

Before any migration of an existing database, a backup is taken with reason `pre-migration`.

## Verifying a backup

Run this on a copy, or on the same host with `serve` stopped or running:

```sh
sha256sum -c <(printf '%s  %s\n' "$(jq -r .sha256 circula-X.db.manifest.json)" circula-X.db)
circula verify-ledger -database /path/to/a/copy.db
```

`restore` checks the checksum and the size against the manifest before it changes anything.

## Restoring

Restores go into a stopped instance or a new instance, never over a running one (OPS-020).

```sh
systemctl stop circula                         # or stop the process
circula restore -database /var/lib/circula/circula.db /var/backups/circula/circula-20261020T030000Z.db
```

What `restore` does, in order:

1. Reads the manifest and checks the file's SHA-256 and size. It refuses a backup whose schema is newer than the binary.
2. Takes the single-writer lock. It refuses if `serve` is running.
3. If the database exists, it refuses unless `-overwrite` is given. With `-overwrite`, it first writes a `pre-restore`
   backup of the current database. If that backup fails, nothing changes.
4. Copies the backup to a temporary file, checks SQLite's integrity, and refuses a backup that needs migration. The
   copy is then moved into place, and the old WAL and shared-memory files are removed.
5. Verifies the ledger for every network. It prints the backup time and states that data written after it may be
   missing and must be reconciled from other sources.
6. Writes `restore-<time>.json` to `backup-dir`.

A restore whose ledger verification fails exits with code `3`. `serve` will refuse to start on that database. Restore an
earlier verified backup, or follow the incident procedure in SEC-030.

Then start the service again:

```sh
circula serve
curl -s http://127.0.0.1:8080/readyz
```

## Restore drills (OPS-021)

Run a drill before each release that changes storage or migrations, once a quarter for hosted deployments, and once a
year for self-hosted deployments. Use a separate host or a separate directory:

1. Pick the newest backup with `ledger_verification.status` of `ok`.
2. `circula restore -database /tmp/drill/circula.db <backup>` and time the command.
3. Record the date, backup file name, duration, and the ledger result in the drill log.
4. Start `circula serve` against the drilled copy with a different `listen` port. Check `/readyz`.

A failed drill blocks the release until the cause is recorded and fixed.

## Off-host copies

Copy each backup and its manifest off the host, and copy the restore record too. Restrict who can read the copies to the
operator roles that need them. Hosted deployments keep one copy outside the primary failure domain (OPS-019). Encryption at
rest is not yet implemented (see [README.md](README.md#known-gaps)); until then, the copy location must be encrypted.
