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
| `shutdown-grace` | `CIRCULA_SHUTDOWN_GRACE` | `10s` | no | How long in-flight requests may finish after a stop signal (1s to 10m). |
| `idempotency-ttl` | `CIRCULA_IDEMPOTENCY_TTL` | `168h` | no | How long idempotency records are kept. |
| `min-password-length` | `CIRCULA_MIN_PASSWORD_LENGTH` | `12` | no | Minimum password length (SEC-002). Cannot be set below 12. |
| `require-mfa-admins` | `CIRCULA_REQUIRE_MFA_ADMINS` | `false` | no | When `true`, a network administrator without an enabled second factor is refused every administrator-only action with `forbidden` / `mfa-enrolment-required` until they enrol at `/account/security` (SEC-007, DEC-007). See `docs/dev/mfa.md`. |
| `max-body-bytes` | `CIRCULA_MAX_BODY_BYTES` | `1048576` | no | Largest accepted request body. |
| `request-timeout` | `CIRCULA_REQUEST_TIMEOUT` | `30s` | no | Deadline for each API and UI request. |
| `log-level` | `CIRCULA_LOG_LEVEL` | `info` | no | `debug`, `info`, `warn` or `error`. Needs a restart to change. |
| `log-format` | `CIRCULA_LOG_FORMAT` | `json` | no | `json` or `text`. |
| `log-retention` | `CIRCULA_LOG_RETENTION` | `2160h` (90 days) | no | How long `circula prune-logs -dir <dir>` keeps stored log files (24h to 3660d). See "Log retention" in monitoring.md. |
| `cookie-secure` | `CIRCULA_COOKIE_SECURE` | `true` | no | Marks every cookie `Secure` and names the session cookie `__Host-circula_session` (SEC-022). Keep `true` behind a TLS proxy: TLS ends at the proxy, so the service cannot see it. `false` is for plain-http development only; with `false` the cookie is `circula_session`. |
| `maintenance` | `CIRCULA_MAINTENANCE` | `false` | no | Start in maintenance mode (OPS-024): reads (GET, HEAD, OPTIONS) keep working, every other web and API request (sign-in included) answers `503` with `Retry-After` and a friendly page; `/healthz`, `/readyz` and `/metrics` keep working. See [upgrade.md](upgrade.md#maintenance-mode). Turn it on or off while running with `circula maintenance on|off`. |
| `metrics-allow` | `CIRCULA_METRICS_ALLOW` | `127.0.0.1/32,::1/128` | no | Comma-separated addresses or CIDRs allowed to read `/metrics`. `none` denies all. |
| `trusted-proxies` | `CIRCULA_TRUSTED_PROXIES` | `none` | no | Comma-separated addresses or CIDRs of the reverse proxies in front of the service. Only requests whose TCP peer is in this list are believed when they carry `X-Forwarded-For` (client address for rate limits) or `X-Forwarded-Proto`. The client address is the right-most `X-Forwarded-For` entry that is not itself a trusted proxy. Empty: both headers are ignored and every client looks like the proxy. Set it to the proxy's address, for example `127.0.0.1/32`. |
| `allow-insecure-http` | `CIRCULA_ALLOW_INSECURE_HTTP` | `false` | no | `serve` refuses to start when `public-base-url` is `http://` for a non-loopback host, or when it is `http://` and `listen` is not loopback (SEC-022). Set `true` only for a closed test network. |
| `setup-secret-file` | `CIRCULA_SETUP_SECRET_FILE` | none | no | File holding the setup secret (see below). |
| `metrics-token-file` | `CIRCULA_METRICS_TOKEN_FILE` | none | no | File holding a bearer token that `/metrics` then requires (see below). |
| `backup-key-file` | `CIRCULA_BACKUP_KEY_FILE` | none | no | File holding the backup signing key (see below). |
| `mfa-secret-file` | `CIRCULA_MFA_SECRET_FILE` | none | no | File holding the key that seals two-step sign-in secrets in the database (see below). |
| `backup-encryption-key-file` | `CIRCULA_BACKUP_ENCRYPTION_KEY_FILE` | none | no | File holding the key that encrypts backups at rest (see below). |
| `backup-require-encryption` | `CIRCULA_BACKUP_REQUIRE_ENCRYPTION` | `false` | no | `true` makes `restore` refuse plaintext backups and makes backups fail unless an encryption key is configured. Configuration is invalid when `true` without a key. |
| `tls-cert-file` | `CIRCULA_TLS_CERT_FILE` | none | no | PEM certificate chain. With `tls-key-file`, the service terminates TLS itself (RB-013): minimum TLS 1.2, Go's default cipher suites, HTTP/2. `public-base-url` must then be `https://`. Both files are re-read on `SIGHUP` (`kill -HUP <pid>`); a failed reload logs an error and keeps the old certificate. See [TLS](#native-tls-rb-013). |
| `tls-key-file` | `CIRCULA_TLS_KEY_FILE` | none | no | PEM private key for `tls-cert-file`. On Unix it must not be readable by group or others. Setting only one of the two refuses to start. |
| `audit-anchor-key-file` | `CIRCULA_AUDIT_ANCHOR_KEY_FILE` | none | no | Key that HMACs audit anchors (RB-010). Keep a copy off the database host. See [Audit anchors](#audit-anchors-rb-010). |
| `audit-anchor-with-backups` | `CIRCULA_AUDIT_ANCHOR_WITH_BACKUPS` | `true` | no | Write `<backup>.anchor.json` beside each scheduled backup. Removed together with the backup it belongs to. |
| `audit-mirror-file` | `CIRCULA_AUDIT_MIRROR_FILE` | none | no | Append every audit event (chain metadata only) to this JSONL file after commit (RB-012). See [Audit mirror](#audit-mirror-rb-012). |
| `state-dir` | `CIRCULA_STATE_DIR` | directory of `database` | no | Where runtime state lives: today only the web signing key `circula-web.key` (see below). Created `0700` when missing. |

## Secrets

Six secrets exist. None is accepted as a flag (a flag would show in the process list) or as a key in the JSON file.
Supply each either as an environment variable or as a file named by the matching `*-file` setting, not both. A secret
file must be a regular file and, on Unix, must not be readable by group or others (`chmod 600`). One trailing newline is
ignored. Each secret must have at least 16 characters; use 32 random characters or more
(`head -c 32 /dev/urandom | base64`). `-check-config` prints only `<set>` or `<unset>`.

| Environment variable | Purpose |
|---|---|
| `CIRCULA_SETUP_SECRET` | FR-002. With a secret, the network-creation page (`/setup`) asks for it and refuses without it. The secret is spent by its first successful use and stays spent across restarts (SEC-028): the use is recorded in the audit log (`setup.secret_used`). To allow exactly one more network, stop the service and run `circula setup-reset` (audited as `setup.secret_reset`, system actor), then start it again; change the secret too if it may have leaked. Remove it from the configuration once the first network exists. Without a secret, any signed-in person may create a network, and `serve` logs a warning at start. Wrong guesses are rate limited per client address. |
| `CIRCULA_METRICS_TOKEN` | SA-15. When set, `/metrics` also requires `Authorization: Bearer <token>`. Set it whenever a reverse proxy runs on the same host: such a proxy reaches `/metrics` from loopback. Independently, `/metrics` always refuses requests carrying any proxy header (`X-Forwarded-*`, `Forwarded`, `X-Real-IP`, `Via`) and requests from a `trusted-proxies` address. |
| `CIRCULA_MFA_SECRET` | SEC-007. When set, authenticator seeds are stored AES-256-GCM sealed under a key derived from it; otherwise they are stored base32-encoded. Losing or changing it makes sealed seeds unreadable: affected people must disable (recovery code) and re-enrol. Set it before people enrol. See `docs/dev/mfa.md`. |
| `CIRCULA_BACKUP_ENCRYPTION_KEY` | OPS-019, SEC-028. Encrypts backups (AES-256-GCM). 32 random bytes in base64 (`circula keygen`, recommended) or a passphrase of 16+ characters. Must differ from the backup key. Without it backups are plaintext and `serve` warns. See [backup-restore.md](backup-restore.md#encryption-at-rest-ops-019-sec-028). |
| `CIRCULA_AUDIT_ANCHOR_KEY` | RB-010. HMAC key for audit anchors written by `circula audit-anchor write` and the backup job. |
| `CIRCULA_BACKUP_KEY` | SA-07. Signs backup manifests (HMAC-SHA-256) and is required to restore them. See [backup-restore.md](backup-restore.md#authenticity-and-the-backup-key). |

### Runtime state: the web signing key (SEC-027)

`serve` signs CSRF tokens and the session-start cookie with a 32-byte key kept in `<state-dir>/circula-web.key`
(base64, mode `0600`), created on first start and reused after every restart, so people stay signed in across restarts and
upgrades. It is created atomically and safely when several processes race. `serve` refuses to start, and never replaces
the file, when it is readable by group or others, is not a regular file, or does not hold exactly 32 bytes. Back it up
with your configuration (it holds no ledger data and is not part of a database backup). Rotation: stop the service, delete
the file, start it; every signed cookie and form token stops validating and people sign in again. Ledger data is
unaffected.

Other settings:

| Name | Environment variable | Notes |
|---|---|---|
| `CIRCULA_OPERATOR_PASSWORD` | environment only | Password for `admin create-operator`. Not a setting; it is read once and never printed. |
| `-config` / `CIRCULA_CONFIG` | | Path to the JSON file. |

Settings that are documented in the specification but not yet implemented (mail) are listed in
[README.md](README.md#known-gaps).

## Native TLS (RB-013)

Set `tls-cert-file` and `tls-key-file` to serve HTTPS without a proxy. The service accepts TLS 1.2 and 1.3 only and sends
`Strict-Transport-Security` on every application response (HSTS is tied to the request being HTTPS, or to an `https://`
`public-base-url`). Renew the certificate by replacing the files and sending `SIGHUP`; established connections keep the old
certificate. Plain HTTP on the same port is not served. Reverse-proxy deployments are unchanged: leave the settings empty
and keep `trusted-proxies`. The existing refusal still applies: an `http://` `public-base-url` with a non-loopback listen
address does not start unless `allow-insecure-http=true` (or TLS is configured). An `https://` `public-base-url` on a
non-loopback address without TLS keeps starting with a warning, since that is the proxy layout.

## Audit anchors (RB-010)

The audit hash chain and its head live in the database file, so someone who can write that file can recompute the whole
chain. An anchor is a small JSON file holding, for every network, the chain head (`seq` and `hash`), the time and a database
id. Store it off-host.

```sh
circula audit-anchor write /var/backups/offsite/anchor-$(date +%F).json
circula audit-anchor verify anchor-2026-10-01.json     # exit 0 intact, exit 3 tail truncated or history rewritten
```

`verify` passes when each anchored event still exists with the same hash, so the chain may have grown since. It fails when the
anchored event is missing (truncation, deleted network), differs (rewrite) or the database id changed. Both commands read
the database without the writer lock, so they work beside `serve`. With `audit-anchor-key-file` set, the anchor carries an
HMAC-SHA-256 and `verify` rejects an edited, unsigned or wrongly keyed file; the key should not live only on the database
host. Scheduled backups write `<backup>.anchor.json` beside the backup unless `audit-anchor-with-backups=false`; copy those
off-host with the backups. Limit: an anchor proves history up to its time. Events after the newest anchor are not covered.

## Audit mirror (RB-012)

`audit-mirror-file` makes the service append one JSON line per audit event after the database commit. Only chain metadata is
written (ids, seq, time, actor, action, target, request id, `prev_hash`, `hash`); summaries, reasons and subject ids stay in
the database so that erasure of personal data is not undone by the mirror. Limits: best effort (a crash between commit and
write, or a write error, loses lines; the first failure is logged), and the file is only as tamper-proof as its location.
Point it at a path the service account can append to but not rewrite (`chattr +a`, a log-shipper pipe or an object-locked
mount) and ship it off-host. The mirror lets you compare `hash` and `prev_hash` linkage against the database independently.

## Secure delete (RB-003)

Every connection runs `PRAGMA secure_delete=ON`: freed pages are overwritten with zeros, so text deleted from any table
(sessions, throttle rows, expired idempotency keys, purged rows) does not linger in the file. Cost: one extra page write per
freed page, negligible for this write volume. Backups taken before the delete, older WAL segments already copied elsewhere
and filesystem snapshots are outside its reach. A canary test (`internal/store/securedelete_test.go`) guards the behaviour.

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
  "metrics-allow": "10.20.0.0/16",
  "trusted-proxies": "127.0.0.1/32",
  "setup-secret-file": "/etc/circula/setup-secret"
}
```

## Start-up checks

On every `serve` start, in this order (before the port opens):

1. Configuration validation, including the transport rule for `public-base-url` (SEC-022).
2. Acquire `<database>.lock`. A second `serve` or `migrate` or `restore` on the same file refuses to start.
3. Read the schema version without migrating. In `explicit` mode, pending migrations stop the start. In `auto` mode, a
   pre-migration backup is taken first and migration is refused if the backup fails.
4. Open the database. Check WAL journal mode and foreign-key enforcement.
5. Verify the ledger for every network. A network that fails is **frozen** (CMD-07, OPS-033): its writes are refused with
   `network-frozen` (API `423`, web page "Network read-only"), reads still work, and every other network serves. The
   failure is logged at error level and `circula_frozen_networks` / `circula_ledger_verification_ok` show it. The hourly
   verification re-checks every network: a network that passes again thaws on its own. Run `circula verify-ledger` for
   the findings and follow the ledger-failure procedure in [monitoring.md](monitoring.md#failure-modes-ops-033).
6. Log a warning if maintenance mode is on, then open the port.

If the integrity check or the verified backup before a migration fails (step 3, `auto` mode, or `circula migrate`), the
migration is not applied and the start stops (OPS-023, see [upgrade.md](upgrade.md#pre-migration-verification-ops-023)).
