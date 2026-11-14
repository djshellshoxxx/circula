# Deployment, Operations and Support
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Supported deployment modes, environments, configuration and secrets handling, backup and restore, monitoring, upgrades, migrations, support responsibilities and failure recovery for hosted and self-hosted deployments | Deps: [_context.md](./_context.md), [05](./05-system-architecture.md), [06](./06-data-ledger-audit.md), [13](./13-assumptions-risks-decisions-roadmap.md) | Open questions: 10, see "Open decisions"

## Purpose

This spec sets the operational requirements that a deployment of Circula MUST meet. It covers how a deployment is run, configured, backed up, restored, monitored, upgraded, migrated, supported and recovered.

It does not decide the deployment model. The supported models (centralized hosted, self-hosted, or both) and their trade-offs are decided in [05 section 5](./05-system-architecture.md#5-deployment-models) under OD-05-1, with the default recorded as RSK-026 in [13](./13-assumptions-risks-decisions-roadmap.md). This spec applies to whichever models are in force and states where hosted and self-hosted requirements differ.

Ledger integrity rules (balance reconstruction, append-only records, idempotent posting) are owned by [06](./06-data-ledger-audit.md). This spec cites them and does not restate them. Platform-plane and network-plane separation is owned by [05](./05-system-architecture.md) (ARC-013, ARC-014).

## Terms used in this spec

- **Deployment**: one running instance of the Circula software with its storage, configuration and secrets. Per [05 section 5](./05-system-architecture.md#5-deployment-models), the MVP runs one network per deployment.
- **Deployment class**: the operating mode of a deployment, either **hosted** (run by the platform operator) or **self-hosted** (run by a party the network chooses). Availability and recovery targets are set per class (ARC-029, OD-05-5).
- **Environment role**: the purpose of a deployment, either **production** (holds real network data) or **non-production** (pre-production, test or development).
- **Recovery point**: a verified backup or snapshot from which a deployment can be restored to a known state.
- **Operator**: the party responsible for running a deployment. For hosted deployments this is the platform operator. For self-hosted deployments this is the self-hosting party.

## Requirements

### 1. Deployment modes and classes

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-001 | MUST | Every deployment MUST record its deployment class (hosted or self-hosted) and the network it serves. | Targets, responsibilities and checks differ by class. | For each deployment, a status report shows its class and network. A deployment with no recorded class does not start. |
| OPS-002 | MUST | Hosted and self-hosted deployments MUST run the same release artifacts. Requirements in this spec apply to both classes unless a section below says otherwise. | Implements ARC-020. | Two deployments of the same release, one per class, pass the same operational checks in section 9. |
| OPS-003 | MUST | A self-hosted deployment MUST be operable from the published deployment checklist and documentation, without access to the platform operator's systems. | Implements ARC-019. | Clean-install check by a tester with no platform-operator access completes and passes the post-upgrade checks in OPS-053. |
| OPS-004 | MUST NOT | A hosted deployment MUST NOT rely on an operational capability that a self-hosted deployment lacks. | Implements ARC-020 and RSK-026. | Feature-parity check of operational commands between the two classes shows no hosted-only operational path. |

### 2. Environments

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-010 | MUST | Each deployment MUST be assigned exactly one environment role, production or non-production. | Prevents test activity against real network data. | A deployment with no role, or with two, fails its startup check. |
| OPS-011 | MUST | Non-production deployments MUST NOT hold data copied from a production network unless the copy is made by a recorded platform-plane action (ARC-014). | Keeps real member data out of test systems without a traceable copy. | Scan of a non-production deployment after a seeded copy shows an audit event for each copy and no copy without one. |
| OPS-012 | SHOULD | Each deployment class SHOULD have a non-production environment that runs the release candidate and the same migration path as production. | Enables rehearsal of upgrades and migrations (OPS-052, DAT-105). | Release checklist records a rehearsal in such an environment before each production upgrade. |
| OPS-013 | MAY | Operators MAY create short-lived non-production environments for testing. | Flexibility for test work. | Created environments are subject to OPS-010 and OPS-011. |

### 3. Configuration

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-020 | MUST | Environment-specific configuration MUST be supplied from outside the release artifact. Release artifacts MUST NOT contain deployment-specific values. | Lets one artifact run in every environment (OPS-002). | The same artifact, started with two configurations, behaves as each configuration specifies, without a rebuild. |
| OPS-021 | MUST | Startup MUST validate all required configuration. A missing or invalid required value MUST stop startup, and the error MUST name the key but MUST NOT print its value. | Fails early with no silent defaults for critical settings. | Fault injection with each required key missing and each malformed: startup stops, and no error output contains the injected value. |
| OPS-022 | MUST | Each configuration key MUST be documented with its purpose, type, whether it is required, its default if any, and whether it is secret. Documentation MUST be generated from or checked against the same definition the software validates against. | Keeps operator documentation accurate for self-hosted operators. | A check fails the build when a key is defined without documentation, or documented but not defined. |
| OPS-023 | MUST | Network policy values (credit limits, dispute windows and similar, per [03](./03-domain-model-rules.md)) MUST be changed through the network administration interface and recorded as policy versions, not through deployment configuration. | Keeps network governance separate from platform operation (ARC-013). | A deployment-configuration change to a network policy value is rejected or ignored, and a policy change produces a new policy version. |
| OPS-024 | MUST | Secrets MUST NOT appear in source control, release artifacts, configuration documentation, logs, audit event content, exports (ARC-025), or error responses. | Limits exposure of credentials and keys. | After a test run with seeded canary secrets, a scan of all listed outputs finds none. |
| OPS-025 | MUST | Secrets MUST be supplied to a deployment through a mechanism that is controlled by the operator and kept separate from network data. The mechanism is not chosen in this spec (OD-11-7). | Operator controls credentials; network data does not carry them. | Restoring a backup of network data does not restore any secret. |
| OPS-026 | MUST | Access to secrets on a platform plane MUST require a platform-plane permission that is not implied by any network-administrator permission (ARC-013). | Keeps credential access separate from governance. | Access test with a network-administrator account confirms no secret read or change is possible. |
| OPS-027 | MUST | Every secret change or rotation MUST produce an audit event (ARC-014) naming the actor, the secret's identifier and the time. The audit event MUST NOT contain the secret value. | Traceable key handling without exposing keys. | Rotation drill produces one audit event per rotated secret, and the event contains no value (OPS-024 scan). |
| OPS-028 | MUST | A documented procedure MUST exist for rotating each secret class and for responding to suspected compromise of a secret. Rotation MUST be possible without editing or deleting ledger data. | Ensures key rotation is routine and safe. | Rotation drill completes with the deployment serving requests before and after, and with entries and audit events unchanged by the rotation. |
| OPS-029 | MUST | After a compromise response revokes a secret, the deployment MUST reject the revoked secret within the bound set by OD-11-1. | Bounds exposure time. | Test presents the revoked secret after the response and observes rejection within the bound. |

### 4. Backup and restore

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-030 | MUST | Backups MUST cover every record class that the platform treats as network data: accounts, transactions, entries, transition records, idempotency records, audit events, policy versions and configuration of the network (see [06](./06-data-ledger-audit.md) and ARC-025 in [05](./05-system-architecture.md#43-data-exportability)). | A partial backup cannot restore a reconstructable ledger. | A restore drill (OPS-033) recovers every listed class; ARC-025-style counts match before and after. |
| OPS-031 | MUST | Each backup MUST be verified at creation by an integrity check. A backup that fails verification MUST NOT be recorded as a recovery point. | Prevents restores from corrupt data. | Corrupted backup fixture is not listed as a recovery point. |
| OPS-032 | MUST | Backup frequency, retention and drill interval MUST meet the minimums set under OD-11-2. | Gives a defined recovery position. | Backup schedule and retention settings are compared with the recorded minimums; a deployment below a minimum fails its operational check. |
| OPS-033 | MUST | Restore MUST be tested by a drill at the interval set under OD-11-2. Each drill MUST restore to a non-production environment and run the reconstruction check (DAT-050) and reconciliation (ARC-016). | Proves backups are usable, not only present. | Drill report shows zero reconstruction mismatches and zero reconciliation mismatches. |
| OPS-034 | MUST | A restore MUST either complete fully or leave the target unchanged. A partially restored deployment MUST NOT accept posting or serve balances as current. | Avoids partial state (ARC-030, invariant 7 in [_context](./_context.md)). | Interruption drill at several points leaves the target in one of the two allowed states. |
| OPS-035 | MUST | A restore MUST name the target deployment and network, MUST require confirmation by the operator, and MUST produce an audit event (ARC-014) naming the recovery point used. | Prevents restoring one network's data into another. | Restore without confirmation is refused. The audit event exists and names the recovery point. |
| OPS-036 | MUST | A restored deployment MUST NOT accept new postings until the reconstruction check (DAT-050) passes on the restored data. | Ensures conservation before use. | Restore with an injected mismatch keeps posting unavailable and reports the mismatch as an audit event. |

### 5. Monitoring

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-040 | MUST | Each deployment MUST expose, to its operator, these health signals: process availability, storage availability, whether posting is available, age of the most recent verified recovery point, result of the most recent reconciliation, and the state of the most recent migration. | Operators need the signals that predict failure. | Each signal is reported in a status view or interface and changes correctly under fault injection. |
| OPS-041 | MUST | Measured availability and posting latency MUST be reported against the targets recorded for the deployment class (ARC-029, OD-11-1). | Makes targets verifiable. | Report shows measured and target values side by side for each class. |
| OPS-042 | MUST | A reconciliation mismatch (ARC-016) MUST raise an alert to the operator and to the network administrator within the bound set under OD-11-1. The alert MUST NOT trigger an automatic balance change. | Ensures prompt human review without silent correction (invariant 3). | Injected mismatch produces an alert in both recipient roles within the bound, and balances are unchanged. |
| OPS-043 | MUST | Operational logs MUST be structured, MUST NOT contain secrets (OPS-024), and MUST NOT contain member personal data beyond identifiers needed to diagnose a fault. | Limits exposure in operational data. | Log scan after a test run finds no secret canaries and no personal data fields outside the allowed identifier set. |
| OPS-044 | SHOULD | Operational logs and monitoring data SHOULD follow a retention period shorter than ledger and audit retention, set under OD-11-10. | Limits data kept outside the ledger (see [06](./06-data-ledger-audit.md) Retention). | Retention settings are compared with the recorded period for operational data. |

### 6. Upgrades

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-050 | MUST | Each release MUST document its upgrade path from the previous supported version, including any migration it runs and any data it changes. | Operators can plan upgrades. | Release notes for each release contain the section; a check fails the release without it. |
| OPS-051 | MUST | Before an upgrade that touches ledger data, the operator MUST take and verify a recovery point (DAT-104). | Enables recovery from a failed upgrade. | Upgrade runner refuses to start without a verified recovery point recorded. |
| OPS-052 | MUST | Each upgrade to production MUST be rehearsed on a copy of production data in a non-production environment before it runs in production (DAT-101, DAT-105). | Surfaces failures before real data is affected. | Release checklist records a rehearsal with equal per-account balances before and after. |
| OPS-053 | MUST | After an upgrade, the reconstruction check (DAT-050) MUST pass, balances MUST equal their pre-upgrade values, and reconciliation (ARC-016) MUST report zero mismatches. Each upgrade MUST produce an audit event naming the versions involved. | Confirms the upgrade preserved the ledger. | Post-upgrade check report is stored with the audit event and shows the three results. |
| OPS-054 | MUST | Each release MUST state whether it supports rollback to the previous release, and if not, MUST document the roll-forward procedure (DAT-104). | Operators know their recovery path before upgrading. | Release notes contain one of the two statements. |
| OPS-055 | SHOULD | Upgrades SHOULD keep ledger writes frozen no longer than the window set under OD-11-3. | Keeps trading available during upgrades. | Measured write freeze in a rehearsal is within the recorded window. |
| OPS-056 | MUST | Each release MUST state its supported-version window (OD-11-4). Versions outside the window MUST NOT receive security fixes from the software project. | Gives operators a predictable support horizon. | Release notes state the window; a check confirms it matches the recorded policy. |

### 7. Migrations

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-060 | MUST | Schema and data migrations MUST be applied only by the versioned migration runner. Manual changes to production data MUST NOT occur except through a documented and audited procedure. | Keeps migration history reconstructable (DAT-103). | Attempted unversioned change in production is refused or detected by the migration history check. |
| OPS-061 | MUST | Migrations MUST meet DAT-100 to DAT-104. The migration runner MUST refuse a migration that lacks the recovery point or roll-forward procedure required by DAT-104. | Applies the data rules at the operational gate. | Migration without a recovery point is refused (see also OPS-051). |
| OPS-062 | MUST | Each migration MUST record its expected duration and write-freeze impact for a stated data size, measured in a rehearsal (OPS-052). | Lets operators schedule migrations. | Rehearsal report contains both values for the stated size. |
| OPS-063 | SHOULD | Migrations SHOULD be written so that the software of the previous supported version can still read the migrated data during the upgrade window. | Supports rolling or staged upgrades and rollback where offered. | Test runs previous-version reads against migrated fixtures. |

### 8. Support responsibilities

Responsibilities are assigned as follows. Roles are defined in [02](./02-personas-roles-journeys.md) and [_context](./_context.md).

| Activity | Hosted deployment | Self-hosted deployment |
|---|---|---|
| Software releases and release notes | Software project | Software project |
| Running the deployment (infrastructure, uptime, monitoring) | Platform operator | Self-hosting party |
| Backups, restore drills and recovery points | Platform operator | Self-hosting party |
| Secrets and rotation | Platform operator | Self-hosting party |
| Upgrades and migrations | Platform operator | Self-hosting party |
| Network policy, membership and dispute outcomes | Network administrator | Network administrator |
| Support for members about network activity | Network administrator | Network administrator |
| Support for the deployment itself | Platform operator | Per OD-11-5 |

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-070 | MUST | Each deployment MUST record its operator, and each operator MUST publish a contact channel for platform-plane incidents. | Members and administrators need a known point of contact. | Status report shows operator and contact; a check fails a deployment missing either. |
| OPS-071 | MUST | The platform operator of a hosted deployment MUST notify the affected network administrator of platform-plane incidents that affect posting or balance availability. Notification timing and channels are set under OD-11-6. | Keeps administrators informed of outages affecting their network. | Incident drill records a notification to the administrator for each affecting incident, within the OD-11-6 timing. |
| OPS-072 | MUST | A self-hosting party MUST be responsible for operating its deployment as listed in section 8. The software project MUST publish the documentation needed to do so (OPS-003, OPS-050). | Clear ownership in self-hosted use. | Documentation review confirms each listed self-hosted activity has a procedure. |
| OPS-073 | MUST | Platform-operator access to network data for support MUST follow a documented procedure, MUST be limited to that procedure, and MUST produce an audit event (ARC-014). Access scope is governed by OD-06-5 in [06](./06-data-ledger-audit.md). | Keeps support access bounded and traceable. | Support access without the procedure is detected in the access log check. |
| OPS-074 | SHOULD | Incident reviews SHOULD be written in a blameless format and SHOULD be completed within the period set under OD-11-6. | Improves recovery over time. | Each Sev-level incident (as defined by the operator) has a review within the period. |

This spec assigns no availability or response obligation to the software project for self-hosted deployments. Any such obligation is a commercial matter outside this specification (OD-11-5).

### 9. Failure recovery

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-080 | MUST | Each operator MUST have a documented runbook for at least these failures: storage unavailable; posting unavailable while reads remain (ARC-030); failed write or partial write; reconciliation mismatch; failed migration; failed upgrade; loss of storage; secret compromise; corruption detected by the reconstruction check. | Operators act consistently under stress. | Runbook index lists each failure; a review confirms each has steps, owner and recovery target. |
| OPS-081 | MUST | After any failure and recovery, a posting retried with the same idempotency key MUST yield one posting (see the Idempotency section of [06](./06-data-ledger-audit.md#idempotency), invariant 5 in [_context](./_context.md)). | Prevents double-counting after crashes and retries. | Crash-and-retry drill at several points posts each transaction exactly once. |
| OPS-082 | MUST | Recovery MUST NOT delete or alter audit events or ledger entries to complete. Corrections MUST use adjustments (invariant 4 in [_context](./_context.md)). | Recovery keeps history intact. | Post-recovery comparison of entries and audit events against the pre-failure set shows no removed or changed record. |
| OPS-083 | MUST | When the reconstruction check (DAT-050) or reconciliation (ARC-016) reports a mismatch, new postings for the affected network MUST be refused until the mismatch is resolved and a resolution is recorded as an audit event. | Stops compounding of a fault. | Injected mismatch refuses posting; recording a resolution re-enables it. |
| OPS-084 | MUST | Each runbook MUST state the recovery point or procedure used and the expected data loss, which MUST be compared with the targets recorded under OD-11-1. | Connects procedures to targets. | Runbook review confirms the statement and the comparison. |

### 10. Numeric targets

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| OPS-090 | MUST | Each deployment class MUST have recorded values for: recovery point objective, recovery time objective, availability, posting latency, reconciliation alert latency, and secret-revocation bound (OPS-029). | ARC-029 requires measured targets; OD-05-5 requires a minimum set here. | Targets for each class exist in the recorded decision set and are referenced by the measurements in OPS-041. |
| OPS-091 | MUST | This spec MUST NOT state numeric values for the targets in OPS-090 until product input resolves OD-11-1. Until then, values are recorded as "unset" and a deployment for production MUST NOT be declared ready. | Numeric targets are product decisions, not engineering defaults. | Spec and deployment readiness check show "unset" values block production readiness. |

## Open decisions

Option lists and defaults follow the rules in [_context.md](./_context.md). Open-decision IDs use the `OD-11-N` form.

- **OD-11-1 Numeric recovery, availability and alert targets.** Options: (a) product owner sets one minimum per deployment class in this spec before the MVP gate; (b) operator sets values per deployment, at or above a platform minimum published per release, consistent with OD-05-5; (c) targets set only for hosted deployments. **Default: (b), with the platform minimum set by the product owner.** Numeric values are product input. Also sets the secret-revocation bound (OPS-029).
- **OD-11-2 Backup frequency, retention and restore-drill interval.** Options: (a) platform sets minimum frequency, retention and drill interval, operators may exceed them; (b) operator chooses freely; (c) network administrator chooses through network policy. **Default: (a).** Minimum values are product input. Governs OPS-032 and OPS-033.
- **OD-11-3 Write-freeze window for upgrades and migrations.** Options: (a) numeric maximum per deployment size set by product; (b) no write freeze, online migrations only; (c) announced maintenance windows, with a numeric maximum from product. **Default: (c).** Numeric maximum is product input. Governs OPS-055 and OPS-062.
- **OD-11-4 Supported-version window.** Options: (a) latest release plus one prior minor release; (b) latest major release only; (c) designated long-term releases. **Default: (a).** Length of the window is product input. Governs OPS-056.
- **OD-11-5 Support for self-hosted deployments.** Options: (a) release documentation, upgrade notes and a public issue channel only; (b) support offered by the platform operator under separate terms; (c) support provided by the network administrator. **Default: (a).** Any commercial support terms are outside this pass (monetization is out of scope). No legal or compliance statement is made here.
- **OD-11-6 Hosted incident notification and review timing.** Options: (a) product-defined numeric timing for notification and for post-incident review; (b) best-effort timing with no stated target; (c) timing by severity level. **Default: (c), with numeric values from product.** Governs OPS-071 and OPS-074.
- **OD-11-7 Secret delivery mechanism.** Options: (a) supplied at startup from an operator-controlled store, outside network data; (b) read from a file mounted at startup; (c) fetched at runtime from a secret service. **Default: (a).** No vendor or product is selected. Governs OPS-025.
- **OD-11-8 Secret rotation interval.** Options: (a) on demand and after suspected compromise only; (b) scheduled at an operator-set interval; (c) scheduled at a platform-minimum interval. **Default: (b), with the interval recorded by each operator.** Governs OPS-028. Interval is product input if (c) is chosen.
- **OD-11-9 Number of networks per hosted deployment.** Options: (a) one network per deployment in the MVP, matching [05 section 5](./05-system-architecture.md#5-deployment-models); (b) several networks per hosted deployment, which requires OD-05-7 to be resolved first. **Default: (a).** Governs OPS-001 and OPS-035.
- **OD-11-10 Retention of operational logs and monitoring data.** Options: (a) shorter than ledger and audit retention, with the period set by product; (b) same as ledger and audit retention; (c) no retention limit. **Default: (a).** Period is product input. Governs OPS-044 and interacts with OPS-043 and the Retention section of [06](./06-data-ledger-audit.md).

## Related docs

- [_context.md](./_context.md): shared terms, normative keywords, ledger invariants and ID prefixes.
- [05 System architecture](./05-system-architecture.md): deployment models (section 5, OD-05-1), platform and network planes (ARC-013, ARC-014), availability targets (ARC-029, OD-05-5), exportability (ARC-025 to ARC-027), portability (ARC-019, ARC-020), time source (OD-05-6).
- [06 Data, ledger and audit](./06-data-ledger-audit.md): reconstruction check (DAT-050, DAT-052), migration principles (DAT-100 to DAT-106), idempotency, retention, operator access (OD-06-5).
- [13 Assumptions, risks, decisions and roadmap](./13-assumptions-risks-decisions-roadmap.md): deployment model default (RSK-026) and self-hosting operational risk (RSK-021).
- [01 Product vision and scope](./01-product-vision-scope.md): MVP boundaries that bound this spec.
- [02 Personas, roles and journeys](./02-personas-roles-journeys.md): roles used in the responsibilities table.
- [03 Domain model and rules](./03-domain-model-rules.md): network policy values referenced in OPS-023.
- Security spec (SEC prefix): not yet authored. Requirements for protecting backups and secret stores belong there and are referenced, not restated, once written.
