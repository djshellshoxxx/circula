# Circula Specifications Index

Status: Draft v0.1 · Owner/role: Product architect · Last updated: 2026-10-21 · Scope: index of the foundational specifications in this directory · Dependencies: none · Open questions: see [13](13-assumptions-risks-decisions.md)

Circula is software for running community mutual-credit networks. It is not a guarantee that credits have monetary value or that members will accept them. Read the glossary and invariants in [00](00-conventions.md) first; every other document uses those terms.

## Reading order

Start with [00](00-conventions.md) (conventions, glossary, invariants LED-1..LED-7, DEC-001..DEC-004). Then read the domain and ledger documents (03, 06), followed by the requirements that use them (04, 07, 08), and then the cross-cutting documents (09 to 12). Read [13](13-assumptions-risks-decisions.md) for open decisions before planning any milestone.

## Index

| File | Prefix | Purpose | Status |
|---|---|---|---|
| [00-conventions.md](00-conventions.md) | (shared) | Normative terms, requirement ID scheme, authoritative glossary, platform invariants LED-1..LED-7, configurable policy defaults, shared technical decisions DEC-001..DEC-004 | Draft v0.1 |
| [01-vision-scope.md](01-vision-scope.md) | VIS | Problem, users, goals, non-goals, principles, success measures, MVP boundary | Being written |
| [02-personas-journeys.md](02-personas-journeys.md) | PER | Roles, core user journeys, permissions and error states at requirements level | Being written |
| [03-domain-model.md](03-domain-model.md) | DOM | Domain entities, balance semantics, transaction lifecycle, policies versus invariants | Being written |
| [04-functional-requirements.md](04-functional-requirements.md) | FR | MVP functional requirements with priorities and acceptance criteria; deferred capabilities | Being written |
| [05-architecture.md](05-architecture.md) | ARC | Components, trust boundaries, data flow, deployment options, architectural qualities | Being written |
| [06-data-ledger-audit.md](06-data-ledger-audit.md) | DAT | Conceptual data model, ledger storage rules, audit events, retention and migration principles | Being written |
| [07-api-contract.md](07-api-contract.md) | API | Conceptual API operations, errors, pagination, idempotency, versioning and compatibility | Being written |
| [08-ux-accessibility.md](08-ux-accessibility.md) | UX | Information architecture, core screens, balance presentation, accessibility requirements | Being written |
| [09-security-privacy.md](09-security-privacy.md) | SEC | Assets, actors, threat model, authentication, authorization, privacy, abuse resistance, secure logging, backups, incident response | Draft v0.1 |
| [10-governance-operations.md](10-governance-operations.md) | GOV | Configurable network rules, change control, membership, disputes, appeals, negative balances, suspension, closure, export, platform commitments | Draft v0.1 |
| [11-deployment-operations.md](11-deployment-operations.md) | OPS | Deployment models, configuration and secrets, embedded storage, backup and restore, migrations, monitoring endpoints, failure recovery | Draft v0.1 |
| [12-quality-test-plan.md](12-quality-test-plan.md) | QA | Test layers, ledger invariant and concurrency tests, authorization, disputes, migrations, accessibility, backup and export tests, scenario tests, quality gates | Draft v0.1 |
| [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md) | ASM / RSK / DEC | Assumptions, risks, decision register with recommended defaults, roadmap, legal questions for counsel, decision log template | Draft v0.1 |

Files 01 to 08 are being written in a parallel pass. Their links resolve once those files are committed.

## Conventions used across the set

- Normative terms MUST, SHOULD and MAY are defined in [00 section 1](00-conventions.md#1-normative-terms).
- Requirement IDs follow `PREFIX-NNN` with a three-digit number and are never reused. Priorities are P0 (MVP-blocking), P1 (MVP), P2 (post-MVP).
- Each requirement states its rationale and testable acceptance criteria.
- Legal, tax, money-services, consumer-protection, privacy and financial-regulatory questions are unresolved and reserved for qualified counsel. No document in this set asserts legal or regulatory compliance.
- Decisions that are not yet made are recorded in [13](13-assumptions-risks-decisions.md) with recommended defaults. A default is not a decision.
