# Monitoring, alerts and failure modes

Specifications: OPS-013 to OPS-015, OPS-027, OPS-032, OPS-033.

## Endpoints

| Path | Who can call it | Returns |
|---|---|---|
| `/healthz` | anyone who can reach the listener | Coarse status only: `{"status":"ok"|"unavailable","version":"..."}`. `503` when the database is unreachable or its schema does not match the binary. No member, network, balance or configuration data. |
| `/readyz` | anyone who can reach the listener | `{"status":"ready"}` once serving, `503` while starting, shutting down, or when the database check fails. |
| `/metrics` | clients whose address is in `metrics-allow` (default loopback), not a `trusted-proxies` address, no forwarding headers, and the bearer token when `metrics-token` is configured | Prometheus text exposition format. `403` for everyone else, `401` for a missing or wrong token. |
| `/api/...` | per the API contract | The JSON API, served under `/api/v1` (not stripped). |
| `/...` | per the UI | The web UI. |

Every response carries `X-Request-Id`. A caller-supplied value is kept if it is short and printable. Otherwise the service
generates one. Logs record the route template, status and duration. They never record query strings, headers or bodies.

Do not expose `/metrics` through a reverse proxy. The service refuses `/metrics` requests that carry any proxy header
(`X-Forwarded-For`, `Forwarded`, `X-Forwarded-Host`, `X-Forwarded-Proto`, `X-Real-IP`, `Via`) and requests whose peer is a
`trusted-proxies` address. A proxy on the same host that strips those headers still looks like loopback, so set
`metrics-token` (SA-15): Prometheus then sends `Authorization: Bearer <token>` (`authorization: {credentials_file: ...}`),
and a forwarded `/metrics` without it gets `401`.

## Metrics

Names use the `circula_` prefix. Labels hold only fixed vocabularies (route templates, status classes, job and result names).
No label carries a member, network, balance or personal data.

| Metric | Type | Meaning |
|---|---|---|
| `circula_build_info{version}` | gauge | Always 1. |
| `circula_process_start_time_seconds`, `circula_uptime_seconds` | gauge | Process start time and uptime. |
| `circula_http_requests_total{route,class}` | counter | Requests by route template (`/api/`, `/`, `/healthz`, `/readyz`, `/metrics`) and status class (`2xx`...). |
| `circula_http_request_duration_seconds_{sum,count}{route}` | counter | Latency by route template. |
| `circula_http_auth_failures_total` | counter | Responses with status 401. |
| `circula_http_rate_limited_total` | counter | Responses with status 429. |
| `circula_job_runs_total{job,result}` | counter | Scheduled jobs: `expiry`, `verify`, `backup`; result `ok` or `failed`. |
| `circula_migrations_total{result}` | counter | Migration runs at start-up (`applied`, `failed`, `refused`). |
| `circula_ledger_verification_ok` | gauge | Last ledger verification: `1` pass, `0` fail, `-1` not run in this process. |
| `circula_ledger_verification_failing_networks` | gauge | Networks that failed the last verification. |
| `circula_ledger_verification_timestamp_seconds` | gauge | Time of the last verification. |
| `circula_backup_last_success_timestamp_seconds` | gauge | Time of the last successful backup in this process (`0` if none). |
| `circula_ledger_postings_total` | counter | Ledger postings committed since start (OPS-014). |
| `circula_idempotent_replays_total` | counter | Retries answered from a stored idempotency record since start (OPS-014). |
| `circula_frozen_networks` | gauge | Networks refusing writes after a ledger verification failure (CMD-07). Alert when above 0. |
| `circula_maintenance_mode` | gauge | `1` while maintenance mode is active (OPS-024). |
| `circula_schema_version`, `circula_schema_pending_migrations` | gauge | Applied schema version and migrations the binary knows that the database lacks. |
| `circula_database_size_bytes`, `circula_database_wal_size_bytes` | gauge | Sizes of the database and WAL files. |

Metrics that the specification requires but the service cannot yet count (refused posting attempts, authentication
failures as seen by the service) are listed under known gaps in [README.md](README.md#known-gaps).

The verification job runs hourly, and every start and every backup also verifies the ledger.

## Alerts (OPS-015)

The conditions below ship as Prometheus rules in [alerts/circula.rules.yml](alerts/circula.rules.yml). Load the file
with `rule_files`, name the scrape job of `/metrics` `circula`, and route `severity="critical"` to paging. A Go test
(`cmd/circula/alert_rules_test.go`) fails when a rule reads a metric the binary does not export; the only other series
the rules use are Prometheus' `up` and, in the group `circula-host`, node_exporter's `node_filesystem_avail_bytes` and
`node_filesystem_size_bytes`.

Owners are roles. Assign a named person per instance and record them outside this repository. The incident level is
the level of the incident procedure, [incident-response.md](incident-response.md#severity-levels) (`critical`, `high`,
`normal`); every first response starts there (SEC-030).

| Alert | Condition | Severity | Incident level | Owner role | First response |
|---|---|---|---|---|---|
| `CirculaLedgerVerificationFailed` | `circula_ledger_verification_ok == 0`, or `circula_ledger_verification_failing_networks > 0`, or `circula_frozen_networks > 0` | critical | `critical` | Platform operator; product architect informed | Follow SEC-030. The failing network is already frozen (read-only); the others serve. Do not restore over the live database. Take a backup, run `circula verify-ledger` (stop `serve` first) to capture the findings, then decide with the product architect. |
| `CirculaDown` | `up{job="circula"} == 0` for 5 minutes | critical | `critical` | Platform operator | Follow SEC-030. Check the process and the start-up error: a failed or refused migration, a held lock or a configuration error stops `serve` before it listens. |
| `CirculaHealthCheckFailing` | `/healthz` answered only `503` for 5 minutes (from `circula_http_requests_total{route="/healthz"}`; needs a prober calling `/healthz`) | critical | `critical` | Platform operator | Follow SEC-030. Check the database file and the disk; see the disk-full and corruption rows below. |
| `CirculaMigrationFailed` | `increase(circula_migrations_total{result=~"failed\|refused"}[1h]) > 0`. Today `serve` exits on a failed or refused migration before it listens, so `CirculaDown` is the alert that fires. | critical | `critical` | Platform operator | Follow SEC-030. The database is at its previous version. Read the start-up error, then follow [upgrade.md](upgrade.md). |
| `CirculaBackupStale` | No backup by this `serve` process for more than 26 hours (recommended default for the daily schedule): the last-success gauge is older than 26 hours, or it is `0` and uptime exceeds 26 hours | warning | `high` | Platform operator | Check `backup-interval`, the disk, and the `backup` job failures in logs. Run `circula backup` manually. Backups taken only with the command are not seen by the gauge. |
| `CirculaVolumeUsageHigh` | Filesystem holding the database or backups above 80 percent (recommended default; node_exporter; set the `mountpoint` matcher for your host) | warning | `high` | Platform operator | Free space by expiring old backups only through retention. Do not delete the WAL file. |
| `CirculaAuthThrottling` | `rate(circula_http_rate_limited_total[10m])` above the agreed rate. The rate is not decided: the alert stays silent until you define the recording rule `circula:auth_throttling_alert_threshold:rate10m` (see the header of the rule file). | warning | `high` | Security owner | Review the source addresses in the edge logs. Follow SEC-030 if the pattern looks like an attack. |
| `CirculaJobFailed` | `increase(circula_job_runs_total{result="failed"}[1h]) > 0` (the job name arrives as `exported_job`) | warning | `normal` | Platform operator | Read the logged error for the named job. Raise the level if it is `verify` or `backup`. |

## Failure modes (OPS-033)

| Failure | Detection | Immediate action | Recovery | Owner role |
|---|---|---|---|---|
| Crash during a posting | Process exits; restart | None needed. Postings are atomic (one transaction). | Restart `serve`; run `verify-ledger`. | Platform operator |
| Power loss | Host restarts | None needed beyond restart. Durability is `db-synchronous=normal` (default) in WAL mode, so the most recent committed transactions can be lost, each with its audit row; set `db-synchronous=full` to keep every acknowledged commit ([configuration.md](configuration.md#durability-db-synchronous)). | Start `serve`; run `verify-ledger`; reconcile recent activity with the members (a lost transaction leaves no audit row either). | Platform operator |
| Disk full | Writes fail; the volume alert fires | Free space by means that do not touch the ledger or WAL. | Restart after space is restored; run `verify-ledger`. | Platform operator |
| Database corruption | `PRAGMA integrity_check` fails in a backup; `verify-ledger` fails | Stop `serve`. Keep the damaged file. | Restore the newest verified backup (see [backup-restore.md](backup-restore.md)). Reconcile the gap. | Platform operator |
| Ledger verification failure | Alert above; start-up or hourly verification log line "network frozen" | The failing network is frozen automatically: writes get `network-frozen` (423), reads work, other networks serve. Optionally put the whole instance in maintenance mode (`circula maintenance on`). Capture findings. | Follow SEC-030 ([incident-response.md](incident-response.md)). Do not auto-correct. Restore only after the product architect agrees. A network thaws on the next passing verification (hourly job or restart). | Platform operator; product architect |
| Mail outage | Not applicable: mail is not implemented | None | None | |
| Suspected secret exposure | Report, or alert | Rotate the secret (OPS-010). Sign everyone out with `circula revoke-sessions -reason ... -rotate-web-key` ([runbook](runbook.md#signing-everyone-out-emergency-revocation-sec-027-ops-010)). | Confirm the impact with the security owner. | Security owner |
| Clock error on host | Logs and timestamps out of order | Fix the clock. Expiry runs on the server clock and recovers on restart. | Restart if needed. Check proposal expiry through the audit trail. | Platform operator |

## Scheduled jobs (OPS-027, OPS-028)

- `expiry`: every 5 minutes, and once at start-up. Expires proposals whose expiry time has passed through the service's
  expiry path. It never posts entries.
- `verify`: every hour, and once at start-up. Verifies every network read-only, freezes any that fail and thaws any that
  pass again (it records no consistency run; administrators record one explicitly).
- `purge`: every hour. Deletes expired sessions, throttle rows and idempotency records.
- `backup`: every `backup-interval`, catching up after a restart. Retries after 15 minutes if it fails.

Each job uses UTC and the server clock. The jobs are idempotent: running one twice has no further effect.

## Application logs (SEC-020, OPS-032)

Logs are structured: `log-format=json` (default) writes one JSON object per line, `text` one `key=value` line. Every
entry has `time` (UTC), `level`, `msg` (the event name, for example `request`, `api internal error`,
`web internal error`, `scheduled job failed`) and, for entries about a request, `request_id`. The UI and the API keep
the `X-Request-Id` the edge assigned, so the request id on an error page ("reference") finds both the access-log entry
and the error entry.

Everything `serve` logs goes through `internal/logx`, which:

- replaces the value of any field whose name refers to a password, secret, token, session, cookie, CSRF token, TOTP or
  OTP, recovery code, API key or authorization with `[REDACTED]`;
- removes invitation and API tokens, bearer values and `key=value` credentials from free text such as error messages,
  and masks email addresses and international phone numbers;
- escapes newlines, other control characters, Unicode line separators and bidirectional or invisible format
  characters, so text a user typed cannot end an entry or forge another one.

The access log records method, route template, status and duration; never the path, query, headers or body. Monetary
amounts and balances are not logged; they are in audit events. Tests: `internal/logx`, `TestLogsCarryNoCanaryAcrossFlows`
(`internal/web`), `TestAPILogForgeryAndCredentialsInErrors` (`internal/api`), `TestAccessLogCarriesNoCredentials`
(`cmd/circula`). `log-level` needs a restart to change ([configuration.md](configuration.md)).

## Log retention (SEC-021, DEC-019)

`circula` writes logs to standard error (`log-format`, `log-level`). Whatever stores them (systemd journal, a log file,
a shipper) must keep them for the period in `log-retention` (default 90 days) and no longer. Rules:

- Logs hold request ids, route templates and status classes. They hold no secrets, passwords or tokens (SEC-027).
- Only operator staff who need them may read them. Make log files `0600` and the directory `0700`.
- Log storage is part of the backup and deletion procedures: include it in off-host copies only if it is covered by the
  same retention; delete it with the deployment.
- For log files in a directory, run `circula prune-logs -dir /var/log/circula` daily (a systemd timer or cron job). It
  deletes regular `*.log`, `*.log.*` and `*.jsonl` files older than `log-retention` and ignores everything else. With
  journald, set `MaxRetentionSec=90d` in `journald.conf` instead.
