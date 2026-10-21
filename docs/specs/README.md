# Circula foundational specifications

**Status:** Draft baseline for project-owner review  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Foundational product and system requirements; no implementation  
**Dependencies:** None  
**Open questions:** See [decisions and roadmap](13-decisions-risks-roadmap.md).

These documents establish a technology-neutral baseline for a community mutual-credit network platform. The [authoring prompt](../../prompts/create-foundational-specs.md) records the original scope.

## Specification set

| Document | Purpose |
|---|---|
| [Product vision and scope](01-product-vision-scope.md) | Goals, non-goals, MVP boundary, success measures, normative language |
| [Roles and journeys](02-roles-and-journeys.md) | Actors, permissions, and core user journeys |
| [Domain model and mutual-credit rules](03-domain-model.md) | Shared terms, transaction lifecycle, ledger invariants |
| [Functional requirements](04-functional-requirements.md) | MVP capabilities with IDs and acceptance criteria |
| [Architecture and trust boundaries](05-architecture.md) | Logical components, deployment choices, trust boundaries |
| [Data, ledger, and audit](06-data-ledger-audit.md) | Conceptual data, atomic posting, audit and retention |
| [API contract](07-api-contract.md) | Conceptual operations and protocol-independent API requirements |
| [UX and accessibility](08-ux-accessibility.md) | MVP information architecture and accessible balance presentation |
| [Security and privacy](09-security-privacy.md) | Threat baseline, access controls, privacy, abuse handling |
| [Governance and network operations](10-governance-operations.md) | Network policy lifecycle and member/network operations |
| [Deployment and support](11-deployment-operations.md) | Deployment, backup, upgrades, monitoring, recovery |
| [Quality strategy](12-quality-strategy.md) | Verification layers and scenario-based acceptance tests |
| [Assumptions, risks, decisions, roadmap](13-decisions-risks-roadmap.md) | Open decisions, risks, recommended defaults, sequencing |

## Reading order

Read 01–03 for product and domain foundations, 04–10 for behavior and boundaries, then 11–13 for operations and verification. The domain model is authoritative for balance and transaction terms; other documents reference it.

All items are draft requirements. Resolve open decisions before treating recommendations as committed product behavior.
