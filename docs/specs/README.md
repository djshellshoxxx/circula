# Circula Foundational Specifications

Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: index of foundational specs | Deps: none | Open questions: see each spec's "Open decisions"

Specification only. No application code exists. Shared terms, normative keywords, ID prefixes and ledger invariants live in [_context.md](_context.md).

| # | Spec | ID prefix | Purpose |
|---|------|-----------|---------|
| – | [Shared context](_context.md) | – | Normative terms, glossary, ledger invariants, header template. |
| 01 | [Product vision and scope](01-product-vision-scope.md) | PRD | Problem, users, goals, non-goals, principles, success measures, MVP boundary. |
| 02 | [Personas, roles, journeys](02-personas-roles-journeys.md) | ROL | Roles, permissions, core user journeys, accessible and error states. |
| 03 | [Domain model and rules](03-domain-model-rules.md) | DOM | Entities, balance semantics, transaction lifecycle, invariants, configurable vs fixed rules. |
| 04 | [Functional requirements](04-functional-requirements.md) | FR | MVP capabilities with priority and acceptance criteria; deferred list. |
| 05 | [System architecture](05-system-architecture.md) | ARC | Logical components, trust boundaries, deployment options, quality attributes. |
| 06 | [Data, ledger, audit](06-data-ledger-audit.md) | DAT | Conceptual entities, atomic posting, idempotency, append-only events, audit content. |
| 07 | [API and integration contract](07-api-integration-contract.md) | API | API principles, auth, errors, pagination, idempotency, versioning, MVP operations. |
| 08 | [UX and accessibility](08-ux-accessibility.md) | UX | Information architecture, core screens, balance clarity, accessibility requirements. |
| 09 | [Security, privacy, abuse](09-security-privacy-abuse.md) | SEC | Assets, threat model, authn/authz, recovery, abuse, logging, backups, incident response. |
| 10 | [Governance and network operations](10-governance-network-operations.md) | GOV | Configurable network rules, change records, appeals, onboarding/offboarding, closure. |
| 11 | [Deployment, operations, support](11-deployment-operations-support.md) | OPS | Deployment models, environments, secrets, backup/restore, monitoring, upgrades. |
| 12 | [Quality strategy and test plan](12-quality-test-plan.md) | QA | Test layers, coverage, scenario acceptance tests, quality gates. |
| 13 | [Assumptions, risks, decisions, roadmap](13-assumptions-risks-decisions-roadmap.md) | RSK | Assumptions, risks, open decisions, deferred work, decision log template. |

## Conventions
- Requirement IDs (`PREFIX-NNN`) are unique and never reused. Open decisions use `OD-<spec#>-N`.
- Each rule has one authoritative home; other specs link to it.
- Legal, tax, financial-regulatory and privacy-law questions are unresolved for qualified counsel; no spec asserts compliance.
