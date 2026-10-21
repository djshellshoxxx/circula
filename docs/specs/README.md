# Circula specification index

**Status:** Initial shell set | **Last updated:** 2026-10-20

These files establish the documentation map for Circula’s foundational specifications. The repository prompt calls for technology-neutral requirements for community mutual-credit networks. This pass creates titled section shells only; the headings are not approved requirements or design decisions.

## Foundations

| File | Purpose | Status |
|---|---|---|
| [Specification conventions and document control](00-conventions-and-document-control.md) | Defines shared requirements language, status metadata, identifiers, and review conventions for this specification set. | Shell |
| [Product vision and scope](01-product-vision-and-scope.md) | Sets Circula’s purpose, intended users, boundaries, and MVP outcomes. | Shell |
| [Personas, roles, and core user journeys](02-personas-roles-and-user-journeys.md) | Defines people and roles that interact with a network and the baseline journeys they must complete. | Shell |
| [Domain model and mutual-credit rules](03-domain-model-and-mutual-credit-rules.md) | Establishes shared domain terms and the platform-wide rules for mutual-credit transactions. | Shell |
| [Functional requirements and acceptance criteria](04-functional-requirements-and-acceptance-criteria.md) | Lists foundational MVP capabilities as uniquely identified, verifiable requirements. | Shell |
| [System architecture and trust boundaries](05-system-architecture-and-trust-boundaries.md) | Describes component responsibilities, trust boundaries, data flows, and deployment choices without prematurely selecting a stack. | Shell |
| [Data model, ledger, and audit requirements](06-data-model-ledger-and-audit.md) | Defines conceptual data, accounting integrity, auditability, retention, and migration requirements. | Shell |
| [API and integration contract](07-api-and-integration-contract.md) | Defines technology-neutral interface behavior and API obligations for preserving domain invariants. | Shell |
| [UX and accessibility requirements](08-ux-and-accessibility-requirements.md) | Sets MVP information architecture and accessible presentation requirements. | Shell |
| [Security, privacy, and abuse-resistance baseline](09-security-privacy-and-abuse-resistance.md) | Defines the baseline threat model and observable protections for accounts, networks, transactions, and data. | Shell |
| [Governance and network operations](10-governance-and-network-operations.md) | Separates member-governed network policy from software enforcement and operational procedures. | Shell |
| [Deployment, operations, and support](11-deployment-operations-and-support.md) | Defines the operations needed to deploy, maintain, restore, and support Circula. | Shell |
| [Quality strategy and test plan](12-quality-strategy-and-test-plan.md) | Defines verification layers and required scenarios for correctness and operational readiness. | Shell |
| [Assumptions, risks, decisions, and roadmap](13-assumptions-risks-decisions-and-roadmap.md) | Records unresolved choices, assumptions, risks, dependencies, and the sequence for later specification work. | Shell |
| [Cryptographic foundations and decentralization boundaries](14-cryptographic-foundations-and-decentralization.md) | Frames cryptography and decentralization choices for research and decision; it does not assert anonymity or select a protocol. | Shell |

## Authoring notes

- Use `00-conventions-and-document-control.md` for normative terms and stable requirement identifiers.
- Keep each rule in one authoritative spec and link to it from other documents.
- Treat monetary redemption, tokenization, anonymity, legal compliance, and specific cryptographic protocols as undecided unless separately evidenced and approved.
- The crypto/decentralization foundation is included to capture the project’s stated direction; it frames alternatives and questions rather than selecting a chain, token, identity system, or deployment model.
- The project prompt excludes implementation and feature-specific sub-specifications from this pass.

## Repository context

The repository currently contains `prompts/create-foundational-specs.md`. No README, contribution guide, license, or existing specification documents were present in the inspected default-branch tree.