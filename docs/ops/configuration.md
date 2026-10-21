# Configuration reference

Every setting has one name. You can set it three ways, in this precedence order (OPS-006):

1. A flag: `-listen 127.0.0.1:9000`
2. An environment variable: `CIRCULA_LISTEN=127.0.0.1:9000`
3. A JSON configuration file, passed with `-config <path>` or `CIRCULA_CONFIG`: `{"listen": "127.0.0.1:9000"}`
4. The built-in default

Empty environment variables and empty file values count as unset. Unknown keys in the file are an error, so a typo cannot
silently fall back to a default. `circula serve -check-config` prints the effective value and the source of every setting
without opening the database.

Validation runs before the database is opened or a port is bound. A failure names each invalid setting.

## Settings

| Name (flag, file key) | Environment variable | Default | Required | Meaning |
|---|---|---|---|---|
| `listen` | `CIRCULA_LISTEN` | `127.0.0.1:8080` | no | Address to listen on. Loopback by default; set it explicitly to listen elsewhere. |
| `public-base-url` | `CIRCULA_PUBLIC_BASE_URL` | none | `serve` only | Absolute `http` or `https` URL that users open. Used in links. |
| `database` | `CIRCULA_DATABASE` | none | yes | Path to the SQLite file. Its directory must exist. |
| `backup-dir` | `CIRCULA_BACKUP_DIR` | `<database dir>/backups` | no | Where scheduled, pre-migration, pre-restore and restore records go. |
| `backup-interval` | `CIRCULA_BACKUP_INTERVAL` | `24h` | no | Time between scheduled backups. `0` disables them. |
| `backup-retain-daily` | `CIRCULA_BACKUP_RETAIN_DAILY` | `30` | no | Newest backups kept (1 to 10000). |
| `backup-retain-monthly` | `CIRCULA_BACKUP_RETAIN_MONTHLY` | `12` | no | Months for which the newest backup is also kept. `0` disables. |
| `migrate-mode` | `CIRCULA_MIGRATE_MODE` | `explicit` | no | `explicit`: `serve` refuses to start with pending migrations. `auto`: back up, then migrate at start-up. |
| `session-idle` | `CIRCULA_SESSION_IDLE` | `720h` | no | Idle session timeout (DEC-005). |
| `session-max` | `CIRCULA_SESSION_MAX` | `2160h` | no | Absolute session lifetime. Must not be shorter than `session-idle`. |
| `idempotency-ttl` | `CIRCULA_IDEMPOTENCY_TTL` | `168h` | no | How long idempotency records are kept. |
| `min-password-length` | `CIRCULA_MIN_PASSWORD_LENGTH` | `12` | no | Minimum password length (SEC-002). Cannot be set below 12. |
| `max-body-bytes` | `CIRCULA_MAX_BODY_BYTES` | `1048576` | no | Largest accepted request body. |
| `request-timeout` | `CIRCULA_REQUEST_TIMEOUT` | `30s` | no | Deadline for each API and UI request. |
| `log-level` | `CIRCULA_LOG_LEVEL` | `info` | no | `debug`, `info`, `warn` or `error`. Needs a restart to change. |
| `log-format` | `CIRCULA_LOG_FORMAT` | `json` | no | `json` or `text`. |
| `cookie-secure` | `CIRCULA_COOKIE_SECURE` | `true` | no | `Secure` attribute on session cookies (SEC-022). Reserved: not yet applied by the web adapter. |
| `metrics-allow` | `CIRCULA_METRICS_ALLOW` | `127.0.0.1/32,::1/128` | no | Comma-separated addresses or CIDRs allowed to read `/metrics`. `none` denies all. |

Other settings:

| Name | Environment variable | Notes |
|---|---|---|
| `CIRCULA_OPERATOR_PASSWORD` | environment only | Password for `admin create-operator`. Not a setting; it is read once and never printed. |
| `-config` / `CIRCULA_CONFIG` | | Path to the JSON file. |

Settings that are documented in the specification but not yet implemented (mail, backup encryption keys) are listed in
[README.md](README.md#known-gaps).

## Example configuration file

```json
{
  "public-base-url": "https://circula.example.org",
  "listen": "127.0.0.1:8080",
  "database": "/var/lib/circula/circula.db",
  "backup-dir": "/var/backups/circula",
  "backup-interval": "24h",
  "backup-retain-daily": 30,
  "backup-retain-monthly": 12,
  "migrate-mode": "explicit",
  "request-timeout": "30s",
  "log-format": "json",
  "metrics-allow": "10.20.0.0/16"
}
```

## Start-up checks

On every `serve` start, in this order (before the port opens):

1. Configuration validation.
2. Acquire `<database>.lock`. A second `serve` or `migrate` or `restore` on the same file refuses to start.
3. Read the schema version without migrating. In `explicit` mode, pending migrations stop the start. In `auto` mode, a
   pre-migration backup is taken first and migration is refused if the backup fails.
4. Open the database. Check WAL journal mode and foreign-key enforcement.
5. Verify the ledger for every network. Any failure stops the start (exit code 1) with a message naming the next step.
