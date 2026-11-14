# Recovery objectives (RPO and RTO)

Specifications: ARC-016, OPS-034, DEC-016. Version 1, 2026-10-21. Owner: product owner (see below).

**Status: proposed defaults from DEC-016. The numbers below are a product decision that the product owner must confirm
before hosted launch.** Until then they are targets that drills measure; they are not a promise to anyone.

| Deployment | RPO (data you may lose) | RTO (time to be serving again) | Off-host copies |
|---|---|---|---|
| Hosted | 1 hour | 4 hours | at least 1 |
| Self-hosted (recommended) | 24 hours (daily backup) | 4 hours | 1 recommended |

ARC-016 lists an initial placeholder of RPO 24 hours for every deployment; DEC-016 sets the stricter hosted figure. The
hosted figure wins for hosted deployments. Self-hosted operators may choose their own numbers and should write them down.

## What meets the objective

- RPO 1 hour: `backup-interval` of 1h or less, with each backup copied off-host (see backup-restore.md, "Off-host
  copies"). The real RPO is the backup interval plus the copy delay.
- RPO 24 hours: the default daily schedule, copied off-host.
- RTO 4 hours: restore the newest backup whose `ledger_verification.status` is `ok` (`circula restore`), run
  `verify-ledger`, start `serve`, check `/readyz`. A drill on an empty database takes seconds; restore time grows with
  database size, so measure it on your own data.
- Shutdown: in-flight requests finish or roll back within `shutdown-grace` (default 10s, ARC-016). Raise it only if your
  slowest request needs more.

## Measurement (OPS-034)

Until the owner confirms the numbers, daily backups are the minimum and every restore drill records its measured
duration. `scripts/restore-drill.sh` and `scripts/restore-drill-full.sh` write `restore_seconds` into the drill record
(`docs/qa/evidence/drill-*.json` for the release gate). Compare it to the RTO at each drill. A drill slower than half
the RTO is a finding: record it in `docs/qa/DEFECTS.md` and act before the margin is gone.

## Decision record

| Field | Value |
|---|---|
| Decision | DEC-016 |
| Owner | product owner (unnamed in the repository; fill in at launch sign-off) |
| Hosted RPO / RTO | 1 hour / 4 hours (proposed) |
| Self-hosted RPO / RTO | 24 hours / 4 hours (recommended) |
| Confirmed on | not yet |
| Evidence | latest drill record under `docs/qa/evidence/` |
