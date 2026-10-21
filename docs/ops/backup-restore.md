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
  - `encrypted`, `cipher`, `plaintext_bytes`, `plaintext_sha256`: present on encrypted backups (see below). `bytes` and
    `sha256` then describe the ciphertext file on disk
  - `hmac_sha256`: present when `backup-key` is configured (see below). It also covers the encryption fields

Each snapshot is verified before it is named: it must pass the integrity check and, when its schema matches this binary,
the ledger verification. A backup is written under a `.partial` name first and renamed only after every check passed.
A backup whose ledger verification failed is still kept, because it is evidence. Its manifest says `failed`.

The backup is the exact snapshot of the source (CMD-01). Ledger verification records its own run in the database it
checks, so it runs on a throw-away copy of the snapshot; the digest of the snapshot is taken before the checks and
compared after them, and a backup that changed is discarded. Backup files and manifests are created 0600 and
`backup-dir` is 0700 (SA-06).

## Encryption at rest (OPS-019, SEC-028)

Set `backup-encryption-key` and every backup (manual, scheduled, `pre-migration`, `pre-restore`) is written encrypted as
`circula-<UTC time>.db.enc` with its manifest. Without it, backups are plaintext `.db` files and `serve` and `backup`
warn. Restoring an old plaintext backup stays possible unless `backup-require-encryption` is set.

Generate a key and store it on the host that takes and restores backups, apart from the backup storage:

```sh
umask 077
circula keygen > /etc/circula/backup-encryption-key     # 32 random bytes, base64
export CIRCULA_BACKUP_ENCRYPTION_KEY_FILE=/etc/circula/backup-encryption-key
```

Also keep a copy of the key in your password manager or escrow: **a backup cannot be restored without its key, and there
is no recovery**. Key and backups must live in separate places (SEC-028). Any passphrase of 16 or more characters is also
accepted and is stretched with Argon2id (64 MiB, per file salt); the random key is stronger and is the recommendation.
`backup-encryption-key` must differ from `backup-key`.

Format. AES-256-GCM over 64 KiB chunks (standard library only). The file starts with the header `CIRCBK1`, the key
derivation id, the chunk size, a random 16-byte salt and an 8-byte nonce prefix. The key is HKDF-SHA-256 of the random key
and the salt (or Argon2id of a passphrase). Each chunk uses nonce = prefix and a counter, and authenticates the header,
its index and a final-chunk flag as additional data. A flipped bit, reordered, duplicated, dropped or appended chunks, a
truncation (also exactly at a chunk boundary) and a wrong key all fail with a clear error, and no plaintext is released
as a database.

How a backup is made. `VACUUM INTO` writes the plaintext snapshot into a private `0700` work directory
(`.circula-bk-*`) next to the destination. The integrity check and ledger verification run there, the snapshot is
encrypted to `*.db.enc.partial`, decrypted again in memory and compared with the verified plaintext, and only then renamed.
The work directory is overwritten and removed on success and on every failure. If the process is killed, the next backup
in that directory removes work directories older than an hour. Overwriting is best effort on copy-on-write file systems
and SSDs: keep the host disk encrypted too.

Restore decrypts transparently when the key is configured (`CIRCULA_BACKUP_ENCRYPTION_KEY` or
`backup-encryption-key-file`). `restore` refuses, before changing anything: an encrypted backup with no key, a wrong key,
a modified or truncated file, a manifest that disagrees with the file about being encrypted, and (with
`backup-require-encryption`) any plaintext backup.

Key rotation. Set the new key, take a new backup, and keep the old key for as long as old backups are kept. Old backups
open only under the key that wrote them. To re-encrypt an old backup: restore it into a scratch database with the old
key, then `circula backup` under the new key.

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
   With `backup-key` configured it also requires a valid manifest signature (below).
3. If the database exists, it refuses unless `-overwrite` is given. With `-overwrite`, it first writes a `pre-restore`
   backup of the current database. If that backup fails, nothing changes.
4. Copies the backup to a temporary file, checks SQLite's integrity, and refuses a backup that needs migration.
5. Verifies the ledger and every network's audit hash chain inside the backup itself, on a throw-away copy (SA-07).
   A backup that fails exits with code `3` and the current database is not touched.
6. Moves the current database aside, moves the verified copy into place, opens it and checks the journal mode and
   foreign keys. If any step fails the previous database is put back unchanged (CMD-08); otherwise the old files are
   deleted. A copy of the old data stays in the `pre-restore` backup from step 3.
7. Prints the backup time, states that data written after it may be missing and must be reconciled from other sources,
   and writes `restore-<time>.json` to `backup-dir`.

## Authenticity and the backup key

Threat (SA-07). The manifest sits next to the backup and holds the checksum, so it protects against corruption, not
against a person who can write to the backup storage. Such a person can edit a backup (for example replace a password
hash with their own), recompute the checksum, and have the restore install it. Ledger and audit-chain verification
catch edits to balances, entries and audit rows, but not edits to accounts, password hashes, sessions or tokens.

Control. Set `CIRCULA_BACKUP_KEY` (or `backup-key-file`), a secret kept outside the backup location, on the host that
takes and restores backups. Then every manifest carries an HMAC-SHA-256 over its format, time, schema version, size and
digest, and `restore` refuses a manifest that is unsigned or whose signature does not verify under the key. Without the
key, `restore` still verifies the ledger and audit chains, but prints a warning that the backup is not authenticated:
restore only files you trust. Rotate the key by taking a new backup under the new key; old backups verify only under the
key that signed them, so keep old keys with the old backups. After restoring any backup you did not make yourself, make
everyone change their password; a password change revokes the person's sessions.

A failing ledger or audit chain is never overridden: fix the cause or restore an earlier backup (SEC-030).

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
operator roles that need them. Hosted deployments keep one copy outside the primary failure domain (OPS-019). With
`backup-encryption-key` set the copies are encrypted; never store the key beside them. Without it, the copy location must
itself be encrypted.
