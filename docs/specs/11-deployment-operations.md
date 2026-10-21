# Deployment, operations, and support

**Status:** Draft recommendation  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Deployment and operational capabilities; vendor-neutral  
**Dependencies:** [Architecture](05-architecture.md), [security](09-security-privacy.md), [quality strategy](12-quality-strategy.md)  
**Open questions:** Hosted/self-hosted first release, supported platform matrix, support ownership, and service targets.

## Deployment options

The architecture should remain portable and support a documented reference deployment. The first production model—hosted, self-hosted, or both—must be selected before release. If both are supported, each needs a clear responsibility boundary for infrastructure, identity, backups, upgrades, incident response, and member support. No infrastructure vendor is selected here.

## Operational requirements

- OPS-001: Deployments MUST separate development, test, and production configuration and MUST keep secrets out of source control and ordinary logs.
- OPS-002: Operators MUST have documented backup schedules, retention, encryption/access controls, and restore procedures.
- OPS-003: A restore exercise MUST verify ledger reconciliation and network isolation before restored data is returned to service.
- OPS-004: Upgrades and migrations MUST have versioned steps, compatibility notes, preflight checks, and a recovery path.
- OPS-005: Monitoring MUST report service health, storage capacity, backup success, migration status, and errors that could affect transaction outcomes without exposing member secrets.
- OPS-006: Operational failures MUST preserve enough correlation information to determine whether a state-changing request committed.
- OPS-007: Support responsibilities and escalation routes MUST be documented for each deployment model.
- OPS-008: Networks MUST be able to export records before service discontinuation or network closure.

## Recovery and support

Runbooks SHOULD cover outage communication, data restore, partial migration, suspected compromise, failed export, and uncertain transaction outcome. Restore procedures must avoid creating a second writable copy of the same network ledger. Service-level targets, recovery point objective, and recovery time objective depend on the selected deployment and pilot expectations and are open decisions.
