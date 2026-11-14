# Incident response procedure

Version 1, 2026-11-04. Specifications: SEC-030 (incident response), OPS-030 (support tiers and severity), OPS-015
(alerts), OPS-033 (failure modes). Related: SEC-013 (break-glass), GOV-023 and GOV-024 (suspension), DAT section 10.2
(repair), DEC-014 (notice period), DEC-022 (support objectives), RSK-026 (breach notification duties).

**Status.** This is the written procedure SEC-030 requires. The response targets in
[Severity levels](#severity-levels) are recommendations: they are not service-level commitments and must not be
presented as such until the platform operator approves them (DEC-022, open). The deadline for notifying affected
parties is set by DEC-014 and counsel review (RSK-026); this document does not assert one. A tabletop drill is still
to be run and recorded before hosted launch ([Drills](#drills)).

Names of the people who hold each role are kept outside this repository, with the alert routing.

## Severity levels

The same three levels apply to incidents and to support requests. Alerts carry the level in their `incident_level`
annotation ([alerts/circula.rules.yml](alerts/circula.rules.yml)); [monitoring.md](monitoring.md#alerts-ops-015) and the
[runbook](runbook.md#incident-contacts-and-severity) use the same names.

| Level | Meaning | Examples | Recommended first response (DEC-022 open) | Recommended restore target (DEC-022 open) | Channel |
|---|---|---|---|---|---|
| `critical` | Ledger integrity or security is at risk, or the instance is down | Ledger verification failure or a frozen network; suspected exposure of credentials, keys or personal data; an authorisation fault; audit anchor mismatch; instance down; migration failure | Within 1 hour of the alert, at any time | Contained at once (a failing network freezes automatically); serving again within the recovery time objective, 4 hours as proposed in [rpo-rto.md](rpo-rto.md) (DEC-016) | Paging route for `severity="critical"` alerts to the on-duty platform operator; the private security channel in [SECURITY.md](../../SECURITY.md) for vulnerability reports |
| `high` | One network cannot post, or a safeguard is degraded while trading continues | Posting refused for a network for a reason other than integrity; backup stale; volume above 80 percent; sustained authentication throttling | Within 4 hours, during the operator's working hours | 1 working day | Team alert route for `severity="warning"` alerts; incident ticket |
| `normal` | Other faults and support requests | A failed scheduled job; a user interface defect; a question from a network administrator | 3 working days (the acknowledgement target in [SECURITY.md](../../SECURITY.md)) | Next planned release, or as agreed with the requester | Ticket or issue tracker; network administrators use their usual contact with the operator |

Approved targets (record here when the platform operator decides DEC-022):

| Level | Approved first response | Approved restore target | Approved by and date |
|---|---|---|---|
| `critical` | not set | not set | DEC-022 open |
| `high` | not set | not set | DEC-022 open |
| `normal` | not set | not set | DEC-022 open |

When in doubt, choose the higher level. A ledger or security incident is always `critical`. The incident lead may
raise or lower the level as facts arrive and records each change.

## Roles

| Role | Responsibility during an incident |
|---|---|
| Incident lead (the on-duty platform operator unless handed over) | Declares the incident, sets and changes the level, coordinates, keeps the incident record and the timeline, decides when the incident is closed |
| Platform operator | Containment and recovery on the host: freeze, maintenance mode, suspension, secret rotation, backups, restore, verification |
| Product architect | Decides on every ledger finding and repair (DAT section 10.2); a ledger incident is not closed without their agreement |
| Security owner | Leads security incidents: scope of exposure, secret rotation, vulnerability handling under [SECURITY.md](../../SECURITY.md), the question to counsel |
| Network administrators of each affected network | Receive the operator's notice, notify their members, take network-level decisions (for example suspending trading, GOV-023) |
| Counsel | Advises on notification duties to regulators and individuals and their deadlines (RSK-026, DEC-014) |

## Detection sources

- Alerts: [monitoring.md](monitoring.md#alerts-ops-015) lists them; the Prometheus rules are in
  [alerts/circula.rules.yml](alerts/circula.rules.yml).
- Ledger verification: the hourly `verify` job, every start, every backup, `circula verify-ledger`, and the
  administrators' consistency check. A failure logs "network frozen" and sets `circula_frozen_networks`.
- Audit chain: `circula audit-anchor verify` exit `3` ([runbook.md](runbook.md#audit-anchor-rb-010)).
- Application logs: error entries such as `api internal error` and `web internal error`. Users see the request id as
  the "reference" on error pages; search the logs for it.
- Reports: abuse reports from members, requests from network administrators, vulnerability reports under
  [SECURITY.md](../../SECURITY.md).
- The audit trail: unexpected administrative or break-glass events (SEC-013).

## Procedure

### 1. Declare and classify

Open an incident record (outside this repository) with: time detected (UTC), source, level, incident lead, affected
networks (by slug), and a timeline that every participant appends to. Use [Severity levels](#severity-levels).

### 2. Contain

- **Ledger verification failure.** The failing network is frozen automatically: its writes are refused (API `423
  network-frozen`, web "Network read-only"), reads and exports work, other networks keep serving. Do not edit the
  database and do not auto-correct. If the cause may affect every network, put the instance in maintenance mode:
  `circula maintenance -config ... on`.
- **Security incident.** Limit the exposure first:
  - Revoke the access of an affected person: `circula reset-password -login NAME -reason ...` ends every session, API
    token and recovery code of that person; `circula reset-mfa` also removes the second factor. This build has no
    command that ends every session of the instance: if that is needed, keep `serve` stopped while the affected
    people are reset one by one.
  - Rotate any secret that may be exposed: the six secrets in [configuration.md](configuration.md#secrets), the web
    signing key ([configuration.md](configuration.md#runtime-state-the-web-signing-key-sec-027)) and the TLS key. Read
    what each rotation invalidates there before rotating it.
  - If a network must stop trading while the incident lasts: `circula suspend start -network <slug> -category
    security_incident -reason "..." -operator "<name>"` ([runbook.md](runbook.md#suspending-a-network-gov-024)). Members keep
    read and export access; the network's administrators are notified in their inbox and audit log.
  - Close any break-glass session that is not needed: `circula break-glass list -network <slug>`, then
    `circula break-glass close -session ID -reason "..."`.
- **Loss of service.** Follow the failure-mode table in [monitoring.md](monitoring.md#failure-modes-ops-033).

### 3. Preserve evidence

Before any recovery step:

1. Take a backup of the current state: `circula backup /var/backups/circula/incident-<id>.db` (works while `serve` runs).
   A backup whose ledger verification fails is still written and is evidence.
2. Keep the damaged database file if `serve` was stopped because of corruption; do not overwrite it.
3. Copy the application logs covering the incident window, the audit mirror file if configured, and the newest audit
   anchor to storage the incident lead controls. Logs contain no credentials by design (SEC-020); treat them as
   restricted anyway (SEC-021).
4. Capture the findings: `circula verify-ledger -network <slug>` (stop `serve` first), and the output of any
   consistency check.
5. Record hashes (`sha256sum`) of the preserved files in the incident record.

### 4. Notify

1. **Affected network administrators first.** The operator notifies the administrators of each affected network, and
   only of that network, with: what happened, what is affected, what was done, what members may notice, and the next
   update time. The deadline is set by DEC-014 and counsel review; it is not asserted here. For a platform suspension
   the software already sends an inbox notice (GOV-024).
2. **Members through their administrators.** The operator does not contact members directly. Administrators inform
   their members, for example with a network announcement.
3. **Regulators and individuals** only as counsel advises (RSK-026).
4. For hosted deployments, the published practice is [docs/platform/incident-notification.md](../platform/incident-notification.md).

### 5. Recover and verify

- Restore only from a verified backup whose `ledger_verification.status` is `ok`
  ([backup-restore.md](backup-restore.md#restoring)), or correct through new ledger transactions (reversal or adjustment
  with dual control). Never edit ledger entries. A direct data change is allowed only as DAT section 10.2 states.
- After any restore or repair: run `circula verify-ledger` (exit `0`), `circula audit-anchor verify` against the newest
  anchor, start `serve`, and check `/readyz`. A frozen network thaws on the next passing verification.
- Confirm with the affected network administrators that balances and recent trades look right; reconcile any gap
  between the backup and the incident (trades made after the backup must be re-entered by the members).
- Record the measured recovery time against the objective in [rpo-rto.md](rpo-rto.md).

### 6. Close and review

The incident lead closes the incident when service is restored, the ledger verifies and notifications are done. Within
the following two weeks, write a blameless post-incident review: timeline, cause, what worked, what did not, and
actions with owners. Keep it in the operator's incident register, give it to the affected network administrators, and
add follow-up defects to `docs/qa/DEFECTS.md` or the issue tracker without personal data.

## Alerts and their levels

| Alert ([alerts/circula.rules.yml](alerts/circula.rules.yml)) | Level |
|---|---|
| `CirculaLedgerVerificationFailed` | `critical` |
| `CirculaDown` | `critical` |
| `CirculaHealthCheckFailing` | `critical` |
| `CirculaMigrationFailed` | `critical` |
| `CirculaBackupStale` | `high` |
| `CirculaVolumeUsageHigh` | `high` |
| `CirculaAuthThrottling` | `high` |
| `CirculaJobFailed` | `normal` |

## Drills

SEC-030 requires an incident drill before hosted launch and at least once a year. Run a tabletop exercise of a ledger
verification failure in one network (the staging test in SEC-030 and OPS-015 checks that only that network freezes,
that an audit event is recorded and that `CirculaLedgerVerificationFailed` fires). Record each drill here:

| Date | Scenario | Participants (roles) | Findings | Actions |
|---|---|---|---|---|
| not yet run | | | | |
