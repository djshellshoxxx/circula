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
| `migrate-mode` | `CIRCULA_MIGRATE_MODE` | `explicit` | no | `explicit`: `serve` refuses to start when an existing database has pending migrations. `auto`: back up, then migrate at start-up. In both modes a database file that does not exist yet is created at the current schema (there is no data to protect). |
| `session-idle` | `CIRCULA_SESSION_IDLE` | `720h` | no | Idle session timeout (DEC-005). |
| `session-max` | `CIRCULA_SESSION_MAX` | `2160h` | no | Absolute session lifetime. Must not be shorter than `session-idle`. |
| `session-idle-admin` | `CIRCULA_SESSION_IDLE_ADMIN` | value of `session-idle` | no | Idle timeout for a person who holds the administrator role in any network they belong to (SEC-003). Not longer than `session-idle`. See [Role-specific session timeouts](#role-specific-session-timeouts-sec-003). |
| `session-max-admin` | `CIRCULA_SESSION_MAX_ADMIN` | value of `session-max` | no | Absolute session lifetime for administrators. Not longer than `session-max`. |
| `session-idle-reviewer` | `CIRCULA_SESSION_IDLE_REVIEWER` | value of `session-idle` | no | Idle timeout for a person who holds the dispute reviewer role in any network. Not longer than `session-idle`. |
| `session-max-reviewer` | `CIRCULA_SESSION_MAX_REVIEWER` | value of `session-max` | no | Absolute session lifetime for dispute reviewers. Not longer than `session-max`. |
| `auth-throttle-attempts` | `CIRCULA_AUTH_THROTTLE_ATTEMPTS` | `5` | no | Persistent per-login/address and sensitive-account attempt budget; 4–5. Zero/off is refused. |
| `auth-throttle-global-attempts` | `CIRCULA_AUTH_THROTTLE_GLOBAL_ATTEMPTS` | `100` | no | Persistent login/recovery budget across addresses, from `auth-throttle-attempts` through 100. |
| `auth-throttle-lock` | `CIRCULA_AUTH_THROTTLE_LOCK` | `15m` | no | Persistent authentication lock duration, whole seconds from 15 minutes through 24 hours. |
| `rate-signin-address` | `CIRCULA_RATE_SIGNIN_ADDRESS` | `20/1m` | no | Web: failed sign-ins, recovery attempts and second-factor attempts per client address, as `count/window`. See [Rate limits](#rate-limits-sec-006). |
| `rate-signin-account` | `CIRCULA_RATE_SIGNIN_ACCOUNT` | `5/1m` | no | Web: failed sign-ins and recoveries per client address and login. |
| `rate-password-check` | `CIRCULA_RATE_PASSWORD_CHECK` | `5/10m` | no | Web: wrong current passwords and second-factor codes entered by a signed-in person (password change, new recovery codes, two-step sign-in settings). |
| `rate-register` | `CIRCULA_RATE_REGISTER` | `10/1m` | no | Web: account creations per client address. |
| `rate-setup` | `CIRCULA_RATE_SETUP` | `10/10m` | no | Web: wrong setup secrets per client address. |
| `rate-join` | `CIRCULA_RATE_JOIN` | `10/10m` | no | Web: join requests on a network's join page, per person and per client address (FR-012). `Retry-After` follows the window. Invitation acceptance is not counted here: it is limited only by `network-join-limit` (web and API) and the API limits below. |
| `api-rate-principal` | `CIRCULA_API_RATE_PRINCIPAL` | `600` | no | API: requests per minute per signed-in person or token (API-038). |
| `api-rate-address` | `CIRCULA_API_RATE_ADDRESS` | `1200` | no | API: requests per minute per client address. |
| `network-join-limit` | `CIRCULA_NETWORK_JOIN_LIMIT` | `off` | no | New memberships per network (invitation redemptions and join requests together, web and API), as `count/window` such as `50/24h`; `off` or `0` disables (SEC-006). |
| `signup-grant-cap` | `CIRCULA_SIGNUP_GRANT_CAP` | `1125899906842624` | no | Largest sign-up grant any network may set, in minor units of that network's credit (SEC-032). `0` turns sign-up grants off. The default is the general policy bound, so nothing changes until you lower it. See [Sign-up grant controls](#sign-up-grant-controls-sec-032). |
| `shutdown-grace` | `CIRCULA_SHUTDOWN_GRACE` | `10s` | no | How long in-flight requests may finish after a stop signal (1s to 10m). |
| `idempotency-ttl` | `CIRCULA_IDEMPOTENCY_TTL` | `168h` | no | How long idempotency records are kept. |
| `min-password-length` | `CIRCULA_MIN_PASSWORD_LENGTH` | `12` | no | Minimum password length (SEC-002). Cannot be set below 12. |
| `require-mfa-admins` | `CIRCULA_REQUIRE_MFA_ADMINS` | `false` | no | When `true`, a network administrator without an enabled second factor is refused every administrator-only action with `forbidden` / `mfa-enrolment-required` until they enrol at `/account/security` (SEC-007, DEC-007). See `docs/dev/mfa.md`. |
| `max-body-bytes` | `CIRCULA_MAX_BODY_BYTES` | `1048576` | no | Largest accepted request body. |
| `request-timeout` | `CIRCULA_REQUEST_TIMEOUT` | `30s` | no | Deadline for each API and UI request. |
| `log-level` | `CIRCULA_LOG_LEVEL` | `info` | no | `debug`, `info`, `warn` or `error`. Needs a restart to change. |
| `log-format` | `CIRCULA_LOG_FORMAT` | `json` | no | `json` or `text`. |
| `log-retention` | `CIRCULA_LOG_RETENTION` | `2160h` (90 days) | no | How long `circula prune-logs -dir <dir>` keeps stored log files (24h to 3660d). See "Log retention" in monitoring.md. |
| `maintenance` | `CIRCULA_MAINTENANCE` | `false` | no | Start in maintenance mode (OPS-024): reads (GET, HEAD, OPTIONS) keep working, every other web and API request (sign-in included) answers `503` with `Retry-After` and a friendly page; `/healthz`, `/readyz` and `/metrics` keep working. See [upgrade.md](upgrade.md#maintenance-mode). Turn it on or off while running with `circula maintenance on|off`. |
| `cookie-secure` | `CIRCULA_COOKIE_SECURE` | `true` | no | Marks every cookie `Secure` and names the session cookie `__Host-circula_session` and the CSRF cookie with the `__Host-` prefix too (SEC-022, SA2-14). When `true` the service reads only the `__Host-` names and ignores a plain `circula_session` or CSRF cookie, which a sibling subdomain could plant; after upgrading from a release that used the plain CSRF cookie, people may need to reload once. Keep `true` behind a TLS proxy: TLS ends at the proxy, so the service cannot see it. `false` is for plain-http development only; with `false` the cookie is `circula_session`. |
| `metrics-allow` | `CIRCULA_METRICS_ALLOW` | `127.0.0.1/32,::1/128` | no | Comma-separated addresses or CIDRs allowed to read `/metrics`. `none` denies all. |
| `trusted-proxies` | `CIRCULA_TRUSTED_PROXIES` | `none` | no | Comma-separated addresses or CIDRs of the reverse proxies in front of the service. Only requests whose TCP peer is in this list are believed when they carry `X-Forwarded-For` (client address for rate limits) or `X-Forwarded-Proto`. The client address is the right-most `X-Forwarded-For` entry that is not itself a trusted proxy. Empty: both headers are ignored and every client looks like the proxy. Set it to the proxy's address, for example `127.0.0.1/32`. `serve` logs a warning at start when it looks like it runs behind a reverse proxy (HTTPS `public-base-url`, loopback `listen`) but `trusted-proxies` is empty, because all clients then share one rate-limit bucket (SA2-15). IPv6 clients are limited per /64 prefix, not per address, so a client cannot dodge a limit by rotating addresses within its prefix. |
| `allow-insecure-http` | `CIRCULA_ALLOW_INSECURE_HTTP` | `false` | no | `serve` refuses to start when `public-base-url` is `http://` for a non-loopback host, or when it is `http://` and `listen` is not loopback (SEC-022). Set `true` only for a closed test network. |
| `setup-secret-file` | `CIRCULA_SETUP_SECRET_FILE` | none | no | File holding the setup secret (see below). |
| `metrics-token-file` | `CIRCULA_METRICS_TOKEN_FILE` | none | no | File holding a bearer token that `/metrics` then requires (see below). |
| `backup-key-file` | `CIRCULA_BACKUP_KEY_FILE` | none | no | File holding the backup signing key (see below). |
| `mfa-secret-file` | `CIRCULA_MFA_SECRET_FILE` | none | no | File holding the key that seals two-step sign-in secrets in the database (see below). |
| `backup-encryption-key-file` | `CIRCULA_BACKUP_ENCRYPTION_KEY_FILE` | none | no | File holding the key that encrypts backups at rest (see below). |
| `backup-require-encryption` | `CIRCULA_BACKUP_REQUIRE_ENCRYPTION` | `auto` | no | `auto` (default): `true` whenever `backup-encryption-key` is set, `false` otherwise (SA2-11). `true` makes `restore` refuse plaintext backups and makes backups fail unless an encryption key is configured; configuration is invalid when `true` without a key. `false` lets a restore accept an old plaintext backup even though a key is set. |
| `db-synchronous` | `CIRCULA_DB_SYNCHRONOUS` | `normal` | no | SQLite durability level in WAL mode: `normal` or `full` (SA2-20). See [Durability](#durability-db-synchronous). |
| `tls-cert-file` | `CIRCULA_TLS_CERT_FILE` | none | no | PEM certificate chain. With `tls-key-file`, the service terminates TLS itself (RB-013): minimum TLS 1.2, Go's default cipher suites, HTTP/2. `public-base-url` must then be `https://`. Both files are re-read on `SIGHUP` (`kill -HUP <pid>`); a failed reload logs an error and keeps the old certificate. See [TLS](#native-tls-rb-013). |
| `tls-key-file` | `CIRCULA_TLS_KEY_FILE` | none | no | PEM private key for `tls-cert-file`. On Unix it must not be readable by group or others. Setting only one of the two refuses to start. |
| `audit-anchor-key-file` | `CIRCULA_AUDIT_ANCHOR_KEY_FILE` | none | no | Key that HMACs audit anchors (RB-010). Keep a copy off the database host. See [Audit anchors](#audit-anchors-rb-010). |
| `audit-anchor-with-backups` | `CIRCULA_AUDIT_ANCHOR_WITH_BACKUPS` | `true` | no | Write `<backup>.anchor.json` beside each scheduled backup. Removed together with the backup it belongs to. |
| `audit-mirror-file` | `CIRCULA_AUDIT_MIRROR_FILE` | none | no | Offer committed audit metadata to a bounded, best-effort JSONL mirror (RB-012). See [Audit mirror](#audit-mirror-rb-012). |
| `state-dir` | `CIRCULA_STATE_DIR` | directory of `database` | no | Where runtime state lives: today only the web signing key `circula-web.key` (see below). Created `0700` when missing. |
| `export-spool-dir` | `CIRCULA_EXPORT_SPOOL_DIR` | `<state-dir>/export-spool` | no | Where API export archives are written and kept until they expire. Created `0700` when missing; `serve` refuses to start if it cannot write there. See [Export spool](#export-spool-fr-095-br-06). |

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
| `CIRCULA_BACKUP_KEY` | SA-07. Signs backup manifests (HMAC-SHA-256) and is required to restore them. Without it `restore` refuses unless `-allow-unsigned` is passed, and `serve` warns when scheduled backups run unsigned (SA2-11). See [backup-restore.md](backup-restore.md#authenticity-and-the-backup-key). |

## Durability (`db-synchronous`)

SA2-20. The database runs in WAL mode. `db-synchronous=normal` (default) syncs the WAL only at checkpoints. It never
corrupts the database, but after a power loss or operating-system crash the newest committed transactions can be lost,
together with their audit rows (a posting and its audit event commit in one transaction, so they are lost together and the
ledger and audit chain stay consistent; the trade simply did not happen). A process crash alone loses nothing.
`db-synchronous=full` syncs the WAL on every commit, so a commit that was acknowledged survives power loss.

Measured cost (Go benchmark `BenchmarkPostingSynchronous` in `internal/store`: propose, post two entries and complete a
trade in one transaction, 1000 iterations, 4 vCPU virtual machine, temporary directory of the build sandbox):

| Level | Time per posting (3 runs) |
|---|---|
| `normal` | 0.70, 0.78, 0.86 ms |
| `full` | 1.89, 2.04, 2.41 ms (about 2.7 times `normal`) |

The ratio is above the 2x the project accepts as a default, so `normal` stays the default. In absolute terms `full` still
allows several hundred postings per second on one writer, far above a community network's load, so choose `full` when you
cannot tolerate losing acknowledged trades after a power failure (for example a host without a UPS or battery-backed
storage). Real disks change the numbers: re-run the benchmark on your hardware with
`go test ./internal/store -run NONE -bench PostingSynchronous -benchtime 1000x`. The setting applies to every command that
opens the database.

### Runtime state: the web signing key (SEC-027)

`serve` signs CSRF tokens and the session-start cookie with a 32-byte key kept in `<state-dir>/circula-web.key`
(base64, mode `0600`), created on first start and reused after every restart, so people stay signed in across restarts and
upgrades. It is created atomically and safely when several processes race. `serve` refuses to start, and never replaces
the file, when it is readable by group or others, is not a regular file, or does not hold exactly 32 bytes. Back it up
with your configuration (it holds no ledger data and is not part of a database backup). Rotation: stop the service, delete
the file, start it; every signed cookie and form token stops validating. Ledger data is unaffected.

**Session key rotation (SEC-027, OPS-010).** The key signs CSRF tokens, flash messages and the session-start cookie, but
the session cookie itself is an opaque random token whose SHA-256 is a row in the database. Replacing the key therefore
does not end sessions on its own. Rotate with one command while `serve` is stopped:

```sh
circula revoke-sessions -config /etc/circula/circula.json -reason "routine key rotation" -rotate-web-key
```

It revokes every live session (audited as `sessions.revoked`, platform operator), then writes a fresh key over the old
one atomically. Start `serve` again; everyone signs in again. The ledger and all other records are untouched (tests
`TestRevokeAllSessions` and `TestRevokeSessionsCommand` compare checksums before and after). API tokens are not affected.
For an emergency sign-out of one person use `-login NAME` without `-rotate-web-key`; see
[runbook.md](runbook.md#signing-everyone-out-emergency-revocation-sec-027-ops-010).

Rotation of the other secrets: `setup-secret`, `metrics-token` and `audit-anchor-key` can be replaced at any restart (old
anchors keep verifying only with the old anchor key, so keep it while those anchors matter); `mfa-secret` cannot be rotated
without people re-enrolling (see the secrets table); the backup keys follow
[backup-restore.md](backup-restore.md).

### Role-specific session timeouts (SEC-003)

By default every person has the same `session-idle` and `session-max` (DEC-064). The four `session-*-admin` and
`session-*-reviewer` settings give shorter lifetimes to a person who holds the administrator or dispute reviewer role in
any network where they are not departed. The role is checked on every request: granting the role shortens a running
session at once, removing it restores the member lifetimes. When a person has several roles, the shortest value applies.
A value longer than the member value is refused at start-up. DEC-005 recommends 60 minutes idle and 12 hours absolute for
administrators and reviewers; that is an owner decision (DEC-005 is open) and is **not** the default. To apply the
recommendation:

```json
{"session-idle-admin": "1h", "session-max-admin": "12h", "session-idle-reviewer": "1h", "session-max-reviewer": "12h"}
```

The web page's "you will be signed out soon" warning (UX-030) still counts with the member values.

### Rate limits (SEC-006)

The `rate-*` settings tune the web interface's in-memory, per-process limiters; `api-rate-*` the JSON API's. A `count/window`
value allows `count` events per fixed `window` (whole seconds, `1s` to `744h`) per key; exceeding it answers `429` with
`Retry-After`. The defaults are the limits the product always had. The bounds only reject values that cannot work (count
1 to 1000000); they are not a security recommendation, and the specification has not set the SEC-006 bounds yet. These
limits sit on top of the persistent service throttle configured by `auth-throttle-*`. The defaults remain five attempts
per login/address and 100 across all addresses for a login, followed by a fixed 15-minute lock. Without a client
address the local budget applies. Recovery has a separate counter family. Password checks for sensitive account
operations and second-factor attempts use the local budget and configured lock; second-factor management retains
its separate counter so a stolen session cannot repeatedly lock sign-in.

`auth-throttle-attempts` accepts 4–5, `auth-throttle-global-attempts` accepts the local budget through 100, and
`auth-throttle-lock` accepts whole seconds from 15 minutes through 24 hours. These conservative bounds permit only
stricter guessing limits; they cannot turn throttling off. The lower count bound preserves at least two attempts
when an expired lock restores half the allowance. Success clears the applicable counters; idle history decays
after 24 hours. Settings apply at process start and do not erase stored counters or shorten an already stored lock.
The CLI, environment and configuration-file sources use the same validation and normal precedence.

For example, `{"auth-throttle-attempts":"4","auth-throttle-global-attempts":"20","auth-throttle-lock":"30m"}`
uses four local attempts, twenty across addresses, and a 30-minute lock. These are identity-level controls;
sign-in does not have a network context, so they do not implement a per-network login throttle.

`network-join-limit` is an aggregate per network, counted in the database inside the joining transaction, so it holds
across restarts and concurrent requests: once `count` memberships (join requests, open joins and invitation
redemptions; the founder is not counted) were created within `window`, further joins and redemptions answer `rate-limited` (`429` in the
API) until older ones leave the window. An invitation refused this way is not used up. Existing members repeating a join
are not affected.

### Sign-up grant controls (SEC-032)

Sign-up grants are off by default (grant `0`, DEC-011). When a network sets one:

- It may not exceed `signup-grant-cap` (minor units; `0` turns every grant off). The cap also applies when a grant is
  posted: lowering it does not change stored policies, but from the next start new members of a network whose stored
  grant is above the cap receive the cap, and nothing when the cap is `0`. The next policy change of such a network is
  refused until its grant is lowered to the cap.
- Every member who becomes active receives it: invited members, approved join requests and, in open join mode, people
  who join by themselves. The founder does not.
- Turning on open join mode together with a grant, or raising the grant while both are on, needs an explicit
  confirmation by the administrator (the tick box on the policy and setup forms, `confirm_open_grant_risk: true` in the
  API). The confirmation is recorded as the audit event `policy.open_grant_risk_confirmed`.

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
chain. New anchors use JSON format version 2 and hold the platform audit checkpoint plus every network's checkpoint
(`seq` and `hash`), the time and a database id. The checkpoints are read in one database snapshot. Store the file off-host.

```sh
circula audit-anchor write /var/backups/offsite/anchor-$(date +%F).json
circula audit-anchor verify anchor-2026-10-01.json     # exit 0 intact, exit 3 tail truncated or history rewritten
```

`verify` passes when each covered checkpoint still exists with the same hash, so the chains may have grown since. It fails when the
anchored event is missing (truncation, deleted network), differs (rewrite) or the database id changed. Both commands read
the database without the writer lock, so they work beside `serve`. With `audit-anchor-key-file` set, the anchor carries an
HMAC-SHA-256 and `verify` rejects an edited, unsigned or wrongly keyed file; the key should not live only on the database
host. Scheduled backups write `<backup>.anchor.json` beside the backup unless `audit-anchor-with-backups=false`; copy those
off-host with the backups. Run `verify-ledger` as well to check the complete live chains against their event contents and
recorded heads; anchor verification compares the independently retained checkpoints. Events after a checkpoint are not
covered by that anchor.

Version 1 anchors retain their original signature encoding and remain verifiable. Their success output explicitly says
**network-only coverage; platform audit not covered**. They are never upgraded or treated as evidence about platform
history. Version 2 requires a separate `platform` object. An empty scope is `{ "seq": 0, "hash": "" }`: it covers no past
events and allows later growth. A database with no audit events also has an empty database id, so that checkpoint cannot
establish a database identity until an event exists. Write a new anchor after the first events.

## Export spool (FR-095, BR-06)

`POST /api/v1/exports` streams each archive into a private file (`0600`) in `export-spool-dir` and keeps it there for 24
hours, until the person who made it or (for administrator scopes) any member of the network is erased, or until the
process stops. On Linux and macOS the file is unlinked as soon as it is created, so it never shows in a directory
listing and nothing is left behind after a crash; the disk space is still used while the export lives. On other
platforms the files stay named `circula-export-*.zip` and `serve` deletes leftovers at start.

Bounds, fixed in this build: at most 64 MiB of archives in total, stored and still being written together. Every chunk
is reserved against that cap before it is written, so parallel exports stop with `503 unavailable` instead of filling
the disk. At most 50 exports are held, 5 per person, and 4 run at once; running exports count as held. Plan for
64 MiB of free space in this directory. A tmpfs mount works but its contents count against memory.

The setting fails closed. `serve` creates the directory (`0700`) when it is missing, and refuses to start when the path
is not a directory or a file cannot be created in it. If the directory becomes unusable while the service runs, each
export answers `503 unavailable` and the log records `export spool directory unusable; export refused` with the path.
Archives are never kept in memory instead. Put it on a local disk owned by the service account, and back it up never:
it holds only short-lived copies of data that is in the database.

## Audit mirror (RB-012)

`audit-mirror-file` offers one metadata record after each audit event commits in the database. A single background worker
appends admitted records as JSON lines in FIFO order. Only chain metadata is written (ids, seq, time, actor, action,
target, request id, `prev_hash`, `hash`); summaries, reasons and subject ids stay in the database so that erasure of
personal data is not undone by the mirror. A slow destination does not delay transaction acknowledgement or keep the
store's writer gate occupied. The callback performs bounded encoding and a nonblocking queue offer, with no file writes
or logging on that path. The destination is opened during startup.

The mirror remains **best effort**, with these fixed limits:

- At most **256 queued records**, each at most **16 KiB including its newline**: at most 4 MiB of queued encoded data,
  plus one record being delivered. Oversized metadata is rejected before a large encoding allocation; the encoded size
  is checked again after JSON escaping. Records are dropped whole, never truncated to fit. Full queues drop new offers.
- One worker writes admitted records in order. Failed writes are not retried, because a partial write has an uncertain
  result. A write error or short write may leave a partial JSON line; subsequent records continue in admission order.
  A successful write means the destination accepted those bytes, not that another system durably retained them.
- Shutdown detaches the store callback and allows **two seconds** to drain. At the deadline, queued records are discarded
  and counted. A write already in progress is **unconfirmed** and may finish later. The shutdown caller does not wait for
  a blocked destination `Write` or `Close`; that one existing worker may remain until the destination returns or the
  process exits. No additional worker is started for a timeout or an event.
- A crash can lose admitted records that have not reached the destination. The database audit rows remain the primary
  committed copy under the configured database durability mode; this queue is not a durable outbox or a delivery receipt.

`/metrics` exposes pending records, successful writes, write failures and drops by reason (`queue_full`, `oversize`,
`encoding`, `closed`, `shutdown`). These counters remain readable even while the destination is stalled. The worker logs
the first write failure and coalesces drop summaries to at most once per minute, plus a final summary when it exits;
those messages can be delayed by a blocked destination. An incomplete shutdown is logged with its pending and loss counts.
See [monitoring](monitoring.md#metrics) for the metric names. Check these counters when using mirror gaps as evidence;
absence from this best-effort copy does not establish absence from the database.

The file is only as tamper-proof as its location. Point it at a path the service account can append to but not rewrite
(`chattr +a`, a log-shipper pipe or an object-locked mount) and ship it off-host. Retained `hash` and `prev_hash` values let
you compare audit-chain linkage against the database independently; the mirror does not claim complete delivery.

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
  "db-synchronous": "normal",
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
3. Read the schema version without migrating. In `explicit` mode, pending migrations on an existing database stop the
   start. In `auto` mode, a pre-migration backup is taken first and migration is refused if the backup fails. In both
   modes a database file that does not exist yet is created at the current schema.
4. Open the database. Check WAL journal mode and foreign-key enforcement.
5. Verify the ledger for every network and the platform audit chain. A network that fails is **frozen** (CMD-07, OPS-033): its writes are refused with
   `network-frozen` (API `423`, web page "Network read-only"), reads still work, and every other network serves. The
   failure is logged at error level and `circula_frozen_networks` / `circula_ledger_verification_ok` show it. The hourly
   verification re-checks every network: a network that passes again thaws on its own. Run `circula verify-ledger` for
   the findings and follow the ledger-failure procedure in [monitoring.md](monitoring.md#failure-modes-ops-033).
   A platform audit finding is logged at error level and makes `circula_ledger_verification_ok` zero. It does not add a
   failing network or freeze unrelated financial networks; the existing critical verification alert requires operator review.
6. Log a warning if maintenance mode is on, then open the port.

If the integrity check or the verified backup before a migration fails (step 3, `auto` mode, or `circula migrate`), the
migration is not applied and the start stops (OPS-023, see [upgrade.md](upgrade.md#pre-migration-verification-ops-023)).
