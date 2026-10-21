# Circula Specifications — Index

Status: Draft baseline · Last updated: 2026-10-21 · Start with [00 — Conventions and shared canon](00-conventions.md).

Circula is software for operating community mutual-credit networks. The credit unit is an accounting unit: not money, not a security, not cryptocurrency, and not redeemable for national currency by design. **Specs only — no implementation yet.** Not legal advice; see [19](19-legal-and-regulatory-open-questions.md).

| # | Spec | Prefix | Purpose |
|---|---|---|---|
| 00 | [Conventions and canon](00-conventions.md) | — | Normative language, ID registry, template, shared vocabulary/signs/states/defaults |
| 01 | [Product vision and scope](01-product-vision-and-scope.md) | VIS | Problem, goals, non-goals, principles, success measures, MVP boundary |
| 02 | [Personas, roles, journeys](02-personas-roles-and-journeys.md) | PER, JRN | Roles, permission matrix, core user journeys |
| 03 | [Domain model and mutual-credit rules](03-domain-model-and-mutual-credit-rules.md) | DOM, INV | Entities, balance semantics, invariants, limits, reversals, departure |
| 04 | [Functional requirements](04-functional-requirements.md) | FR-* | MVP capabilities with acceptance criteria; deferred list |
| 05 | [Architecture and trust boundaries](05-architecture-and-trust-boundaries.md) | ARC | Components, data flow, tenancy, quality attributes |
| 06 | [Data, ledger, audit](06-data-ledger-and-audit.md) | DAT | Entities, atomic posting, idempotency, reconciliation, retention |
| 07 | [API and integration contract](07-api-and-integration-contract.md) | API | Operations, errors, pagination, idempotency, versioning |
| 08 | [UX and accessibility](08-ux-and-accessibility.md) | UX | IA, screens, balance presentation, WCAG 2.2 AA |
| 09 | [Security, privacy, abuse](09-security-privacy-and-abuse.md) | SEC | Threat model, controls, incident response |
| 10 | [Governance and network operations](10-governance-and-network-operations.md) | GOV | Admin powers, appeals, succession, closure, charter template |
| 11 | [Deployment, operations, support](11-deployment-operations-and-support.md) | OPS | Hosting models, backup/restore, monitoring, upgrades |
| 12 | [Quality strategy and test plan](12-quality-strategy-and-test-plan.md) | QA | Test layers, gates, E2E acceptance scenarios |
| 13 | [Assumptions, risks, decisions, roadmap](13-assumptions-risks-decisions-roadmap.md) | ASM, RSK, DEC | Registers; deferred sub-spec roadmap |
| 14 | [Glossary](14-glossary.md) | — | Terms; banned/discouraged wording |
| 15 | [Lifecycle state machines](15-lifecycle-state-machines.md) | LIF | Trade, Dispute, Membership, Listing, Adjustment/Reversal |
| 16 | [Network policy catalog](16-network-policy-catalog.md) | POL | Every configurable parameter, defaults, mutability |
| 17 | [Notifications and communications](17-notifications-and-communications.md) | NTF | Event catalogue, channels, preferences, delivery |
| 18 | [Data export and portability](18-data-export-and-portability.md) | PRT | Exports, network package, import hooks |
| 19 | [Legal and regulatory open questions](19-legal-and-regulatory-open-questions.md) | LEG | Questions for counsel, design guardrails |

## Related

- Plans and to-dos: [`../plans/`](../plans/README.md) · Decisions (ADRs): [`../decisions/`](../decisions/README.md) · Research: [`../notes/`](../notes/README.md)
- Consistency check: `python3 scripts/check-specs.py` (links, duplicate and dangling IDs)
- Authoring prompt: [`../../prompts/create-foundational-specs.md`](../../prompts/create-foundational-specs.md)
