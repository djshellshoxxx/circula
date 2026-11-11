# Circula Specifications

This directory holds the foundational specifications for Circula, software for operating community mutual-credit networks. They are the baseline a future implementation team builds from. They describe behavior, not code, and they are deliberately technology-neutral.

Start with [00-conventions.md](00-conventions.md): it defines the normative words, the requirement-ID scheme, the shared glossary, and the ten baseline decisions every other document assumes. Then read [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md), the authoritative home for every rule about how balances may change.

| # | Document | Purpose | ID prefix |
|---|---|---|---|
| 00 | [Conventions and shared glossary](00-conventions.md) | Normative language, ID scheme, document template, glossary, authoritative homes, baseline decisions. | none |
| 01 | [Product vision and scope](01-product-vision-and-scope.md) | Problem, intended users, goals, non-goals, principles, success measures, MVP boundary. | VIS |
| 02 | [Personas, roles, and core user journeys](02-personas-roles-and-journeys.md) | The four roles, the permissions matrix, and the core journeys with error and accessibility states. | PER, JRN |
| 03 | [Domain model and mutual-credit rules](03-domain-model-and-mutual-credit-rules.md) | Entities, lifecycles, ledger invariants, transaction/reversal/adjustment/limit/dispute/departure rules, policy catalogue. | DOM |
| 04 | [Functional requirements and acceptance criteria](04-functional-requirements.md) | MVP capabilities with stable IDs, priorities, rationale, and testable acceptance criteria. | FR |
| 05 | [System architecture and trust boundaries](05-system-architecture.md) | Components, data flow, trust boundaries, deployment options, architectural qualities, concurrency model. | ARCH |
| 06 | [Data model, ledger, and audit requirements](06-data-model-ledger-and-audit.md) | Conceptual entities, identifiers, atomic posting, append-only storage, audit event contents, retention, migration, export formats. | DATA |
| 07 | [API and integration contract](07-api-and-integration-contract.md) | API principles, authentication, error format, pagination, idempotency, versioning, operation catalogue. | API |
| 08 | [UX and accessibility requirements](08-ux-and-accessibility.md) | Information architecture, core screens, balance presentation rules, responsive and accessibility requirements. | UX |
| 09 | [Security, privacy, and abuse-resistance baseline](09-security-privacy-and-abuse-resistance.md) | Assets, actors, threat model, and requirements for authentication, authorization, recovery, administrator misuse, logging, backups, incident response, privacy. | SEC |
| 10 | [Governance and network operations](10-governance-and-network-operations.md) | What administrators configure, how changes are announced, appeals, onboarding/offboarding, succession, dispute handling, default recovery, closure. | GOV |
| 11 | [Deployment, operations, and support](11-deployment-operations-and-support.md) | Deployment models, environments, configuration and secrets, backup/restore, monitoring, upgrades, support, failure recovery. | OPS |
| 12 | [Quality strategy and test plan](12-quality-strategy-and-test-plan.md) | Test layers, coverage requirements for invariants and concurrency, quality gates, end-to-end acceptance scenarios. | QA, E2E |
| 13 | [Assumptions, risks, decisions, and roadmap](13-assumptions-risks-decisions-and-roadmap.md) | Assumptions, open decisions with options and recommended defaults, risks with mitigations, dependency order, decision log. | ASM, RISK, DEC |

## How the documents fit together

- **03 (domain model)** owns the ledger rules. **06 (data)**, **07 (API)**, **08 (UX)**, **09 (security)**, and **12 (quality)** each enforce or verify those rules from their own angle and cite them by ID rather than restating them.
- **02 (personas)** owns the permissions matrix; **07** and **09** enforce it.
- **10 (governance)** separates what a community decides from what the software enforces or records.
- **13 (decisions)** is where every open question from every document lands, with options and a recommended default.

## Status

All documents are `Draft` as of 2026-11-11. This pass produced specifications only; no application code exists yet.

## Checking the specifications

Every requirement identifier is unique across the set and anchored so that links of the form `03-domain-model-and-mutual-credit-rules.md#dom-001` resolve. Before merging a change to any specification, run the checker, which fails on duplicate or undefined identifiers, broken relative links or anchors, missing required sections, and banned adjectives:

```bash
python3 tools/check_specs.py docs/specs
```
