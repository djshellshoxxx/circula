# Circula Specifications Index

Status: Draft v0.3 · Owner/role: Product architect · Last updated: 2026-10-24 · Scope: index of the foundational specifications in this directory, implementation addenda, and status evidence · Dependencies: none · Open questions: see [13](13-assumptions-risks-decisions.md) and [DECISIONS.md](../DECISIONS.md)

Circula is software for running community mutual-credit networks. It is not a guarantee that credits have monetary value or that members will accept them. Read the glossary and invariants in [00](00-conventions.md) first; every other document uses those terms.

## Reading order

Start with [00](00-conventions.md) (conventions, glossary, invariants LED-1..LED-7, DEC-001..DEC-004). Then read the domain and ledger documents (03, 06), followed by the requirements that use them (04, 07, 08), and then the cross-cutting documents (09 to 12). Read [13](13-assumptions-risks-decisions.md) for open decisions and [DECISIONS.md](../DECISIONS.md) for decisions recorded during the build, before planning any milestone. Treat build-status sections in the specs as historical evidence for the snapshot each section names; they do not establish current branch status. The current checkpoint's evidence and limits are in [RESUME-2026-10-24](../audit/RESUME-2026-10-24.md).

## Index

| File | Prefix | Purpose | Status |
|---|---|---|---|
| [00-conventions.md](00-conventions.md) | (shared) | Normative terms, requirement ID scheme, authoritative glossary, platform invariants LED-1..LED-7, configurable policy defaults, shared technical decisions DEC-001..DEC-004 | Draft v0.2; historical build snapshot labelled |
| [01-vision-scope.md](01-vision-scope.md) | VIS | Problem, users, goals, non-goals, principles, success measures, MVP boundary | Draft v0.1; historical build-status snapshot |
| [02-personas-journeys.md](02-personas-journeys.md) | PER | Roles, core user journeys, permissions and error states at requirements level | Draft v0.1; historical build-status snapshot |
| [03-domain-model.md](03-domain-model.md) | DOM | Domain entities, balance semantics, transaction lifecycle, policies versus invariants | Draft v0.1; historical build-status snapshot |
| [04-functional-requirements.md](04-functional-requirements.md) | FR | MVP functional requirements with priorities and acceptance criteria; deferred capabilities | Draft v0.1; historical build-status snapshot |
| [05-architecture.md](05-architecture.md) | ARC | Components, trust boundaries, data flow, deployment options, architectural qualities | Draft v0.1; historical build-status snapshot |
| [06-data-ledger-audit.md](06-data-ledger-audit.md) | DAT | Conceptual data model, ledger storage rules, audit events, retention and migration principles | Draft v0.1; historical build-status snapshot |
| [07-api-contract.md](07-api-contract.md) | API | Conceptual API operations, errors, pagination, idempotency, versioning and compatibility | Draft; API-specific as-built alignment is explicitly tied to its named historical source |
| [08-ux-accessibility.md](08-ux-accessibility.md) | UX | Information architecture, core screens, balance presentation, accessibility requirements | Draft v0.1; historical build-status snapshot |
| [09-security-privacy.md](09-security-privacy.md) | SEC | Assets, actors, threat model, authentication, authorization, privacy, abuse resistance, secure logging, backups, incident response | Draft v0.1; historical build-status snapshot |
| [10-governance-operations.md](10-governance-operations.md) | GOV | Configurable network rules, change control, membership, disputes, appeals, negative balances, suspension, closure, export, platform commitments | Draft v0.1; historical build-status snapshot |
| [11-deployment-operations.md](11-deployment-operations.md) | OPS | Deployment models, configuration and secrets, embedded storage, backup and restore, migrations, monitoring endpoints, failure recovery | Draft v0.1; historical build-status snapshot |
| [12-quality-test-plan.md](12-quality-test-plan.md) | QA | Test layers, ledger invariant and concurrency tests, authorization, disputes, migrations, accessibility, backup and export tests, scenario tests, quality gates | Draft v0.1; historical build-status snapshot |
| [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md) | ASM / RSK / DEC | Assumptions, risks, decision register with recommended defaults, roadmap, legal questions for counsel, decision log template | Draft v0.2; historical build-status snapshot labelled |
| [../TRACEABILITY.md](../TRACEABILITY.md) | all | Requirement ID to status, code path and test name, for the build snapshot named in that file | Historical traceability evidence; not a current verification claim |
| [../DECISIONS.md](../DECISIONS.md) | DEC | Decisions made during the build (DEC-028 onward), open owner decisions raised by it, and open questions resolved into decisions | Living log |

## Conventions used across the set

- Normative terms MUST, SHOULD and MAY are defined in [00 section 1](00-conventions.md#1-normative-terms).
- Requirement IDs follow `PREFIX-NNN` with a three-digit number and are never reused. Priorities are P0 (MVP-blocking), P1 (MVP), P2 (post-MVP).
- Each requirement states its rationale and testable acceptance criteria.
- Build-status sections describe the named historical snapshot only. Status values are **built**, **partial**, **not built** and **deferred**; they do not prove that an acceptance test or release gate has passed. [TRACEABILITY.md](../TRACEABILITY.md) contains evidence for its named snapshot. Current checkpoint evidence is in [RESUME-2026-10-24](../audit/RESUME-2026-10-24.md).
- Resolved open questions (OQ-*) are removed from the open lists and recorded as decisions in [DECISIONS.md](../DECISIONS.md). Questions still open are listed in each file's open-question section.
- Legal, tax, money-services, consumer-protection, privacy and financial-regulatory questions are unresolved and reserved for qualified counsel. No document in this set asserts legal or regulatory compliance.
- Decisions that are not yet made are recorded in [13](13-assumptions-risks-decisions.md) with recommended defaults. A default is not a decision.

## Implementation addenda

These addenda record implementation details or owner questions raised during build and review work. They are supporting evidence, not replacements for the authoritative requirements in 00–13 and do not themselves settle an open product or owner decision. The owner of each core spec decides whether and how to reconcile an addendum; update that core spec and its decision log when adopting a change. Until then, follow the core spec and record any known implementation divergence in the current checkpoint evidence.

| Addendum | Topic and stated status | Authority limit |
|---|---|---|
| [OPS-019 / SEC-028 backup encryption](addenda/OPS-019-SEC-028.md) | Backup encryption details; marked implemented | Operational evidence only; deployment/security owners reconcile against 09 and 11. |
| [SEC-007 second factor](addenda/SEC-007.md) | TOTP and recovery-code implementation; marked implemented | Does not decide whether MFA is mandatory; DEC-007 remains open. |
| [SEC-008 account recovery](addenda/SEC-008.md) | Channel-free recovery implementation; marked implemented with deviations | Does not settle DEC-009 or replace the core recovery requirement. |
| [SEC-011 dual control implementation](addenda/SEC-011.md) | Adjustment approval behavior; marked implemented with gaps | Does not amend the broader dual-control requirement or decide uncovered actions. |
| [SEC-011 owner decision](addenda/SEC-011-owner-decision.md) | Options for extending dual control; marked open | Proposal only; owner decision is pending. |
| [SEC-012 security notices](addenda/SEC-012.md) | Notices for own-account security changes; marked partly implemented | Narrow scope; does not satisfy or replace all member-facing administrative notices in SEC-012. |
| [SEC-013 break-glass trail](addenda/SEC-013.md) | Operator access trail; marked implemented in part | Does not close remaining gaps or settle DEC-014. |
| [SEC-018 erasure and anonymisation](addenda/SEC-018.md) | Departed-member erasure; marked implemented with owner decisions open | Does not settle retention, closure or counsel questions. |
| [SEC-027 runtime keys](addenda/SEC-027-keys.md) | Persisted web-signing key and setup secret; marked implemented | Implementation detail; reconcile with the authoritative requirements and operating docs. |

An earlier specification-only source, local ref `origin/claude/sweet-turing-m3ftja` at `01b6e5c`, organized glossary, lifecycle, policy, notification, portability, and legal material as separate documents 14–19. The current baseline keeps the prompt's foundational scope consolidated in 00–13. That branch is source material for reconciliation, not an automatic authority to restore those six files wholesale or to create new feature-specific specs.
