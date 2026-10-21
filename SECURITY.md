# Security policy

## Reporting a vulnerability

Please do not open a public issue for a security problem. Use the repository's private vulnerability reporting
(Security tab, "Report a vulnerability"). Include the affected version, the steps to reproduce, and the impact you
expect. Do not include real member data, passwords, session tokens or backup files in a report.

The maintainers will acknowledge the report, confirm whether it is a vulnerability, and agree a disclosure date
with you. A ledger integrity or authorisation fault is treated as critical (see `docs/ops/monitoring.md`).

## Supported versions

Only the latest tagged release receives security fixes during the MVP.

## Operator responsibilities

- Keep `/metrics` and the operator interfaces off the public internet (see `docs/ops/configuration.md`).
- Store backups off the host and restrict who can read them (see `docs/ops/backup-restore.md`).
- Rotate secrets after suspected exposure (`docs/specs/11-deployment-operations.md` OPS-010).
