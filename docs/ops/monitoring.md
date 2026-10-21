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
| `circula_schema_version`, `circula_schema_pending_migrations` | gauge | Applied schema version and migrations the binary knows that the database lacks. |
| `circula_database_size_bytes`, `circula_database_wal_size_bytes` | gauge | Sizes of the database and WAL files. |

Metrics that the specification requires but the service cannot yet count (posting attempts, idempotent replays,
authentication failures as seen by the service) are listed under known gaps in [README.md](README.md#known-gaps).

The verification job runs hourly, and every start and every backup also verifies the ledger.

## Alerts (OPS-015)

Owners are roles. Assign a named person per instance and record them outside this repository.

| Alert | Condition | Severity | Owner role | First response |
|---|---|---|---|---|
| Ledger verification failure | `circula_ledger_verification_ok == 0` for any scrape after a run, or `circula_ledger_verification_failing_networks > 0` | critical | Platform operator; product architect informed | Follow SEC-030. Do not restore over the live database. Run `circula verify-ledger` to capture the findings, take a backup, then decide with the product architect. |
| Instance unhealthy | `/healthz` returns non-200 for more than 5 minutes | critical | Platform operator | Check the process and the database file. See the disk-full and corruption rows below. |
| Backup stale | `time() - circula_backup_last_success_timestamp_seconds > 26*3600` (daily schedule) | recommended | Platform operator | Check `backup-interval`, the disk, and the `backup` job failures in logs. Run `circula backup` manually. |
| Volume usage high | Host volume holding the database or backups above 80 percent | recommended | Platform operator | Free space by expiring old backups only through retention. Do not delete the WAL file. |
| Authentication throttling | `rate(circula_http_rate_limited_total[10m])` sustained above the agreed rate | warning | Security owner | Review the source addresses in the edge logs. Follow SEC-030 if the pattern looks like an attack. |
| Migration failure | `increase(circula_migrations_total{result="failed"}[1h]) > 0` or `result="refused"` | critical | Platform operator | The database is at its previous version. Read the start-up error, then follow [upgrade.md](upgrade.md). |
| Scheduled job failure | `increase(circula_job_runs_total{result="failed"}[1h]) > 0` | warning | Platform operator | Read the logged error for the named job. |

## Failure modes (OPS-033)

| Failure | Detection | Immediate action | Recovery | Owner role |
|---|---|---|---|---|
| Crash during a posting | Process exits; restart | None needed. Postings are atomic (one transaction). | Restart `serve`; run `verify-ledger`. | Platform operator |
| Power loss | Host restarts | None needed beyond restart. Durability is `synchronous=NORMAL` in WAL mode, so the most recent committed transactions can be lost. | Start `serve`; run `verify-ledger`; reconcile recent activity from the audit trail. | Platform operator |
| Disk full | Writes fail; the volume alert fires | Free space by means that do not touch the ledger or WAL. | Restart after space is restored; run `verify-ledger`. | Platform operator |
| Database corruption | `PRAGMA integrity_check` fails in a backup; `verify-ledger` fails | Stop `serve`. Keep the damaged file. | Restore the newest verified backup (see [backup-restore.md](backup-restore.md)). Reconcile the gap. | Platform operator |
| Ledger verification failure | Alert above; startup refusal | Stop serving. Capture findings. | Follow SEC-030. Do not auto-correct. Restore only after the product architect agrees. | Platform operator; product architect |
| Mail outage | Not applicable: mail is not implemented | None | None | |
| Suspected secret exposure | Report, or alert | Rotate the secret (OPS-010). Invalidate sessions by rotating the session key once that key exists. | Confirm the impact with the security owner. | Security owner |
| Clock error on host | Logs and timestamps out of order | Fix the clock. Expiry runs on the server clock and recovers on restart. | Restart if needed. Check proposal expiry through the audit trail. | Platform operator |

## Scheduled jobs (OPS-027, OPS-028)

- `expiry`: every 5 minutes, and once at start-up. Expires proposals whose expiry time has passed through the service's
  expiry path. It never posts entries.
- `verify`: every hour, and once at start-up. Records a consistency run for each network.
- `backup`: every `backup-interval`, catching up after a restart. Retries after 15 minutes if it fails.

Each job uses UTC and the server clock. The jobs are idempotent: running one twice has no further effect.
