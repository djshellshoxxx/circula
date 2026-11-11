# Specification Conventions and Shared Glossary

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Product architect |
| Last updated | 2026-11-11 |
| Scope | Rules every Circula specification follows: normative language, requirement identifiers, document structure, priority levels, shared vocabulary, and the baseline decisions that the other specifications build on. |
| Dependencies | None. Every other document under `docs/specs/` depends on this one. |
| Related documents | [Specification index](README.md), [Domain model and mutual-credit rules](03-domain-model-and-mutual-credit-rules.md), [Assumptions, risks, decisions, and roadmap](13-assumptions-risks-decisions-and-roadmap.md) |

## 1. Purpose

Circula is software for operating community mutual-credit networks. The specifications in this directory are written by several authors and must read as one body of work. This document fixes the shared rules so that every specification uses the same words for the same things, numbers its requirements the same way, and builds on the same baseline decisions.

When a specification and this document disagree on a convention, this document wins. When two specifications disagree on a rule, the document named as the **authoritative home** of that rule in [section 7](#7-authoritative-homes) wins, and the other document MUST be corrected to link to it rather than restate it.

## 2. Normative language

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **NOT RECOMMENDED**, **MAY**, and **OPTIONAL** in these specifications are to be interpreted as described in [BCP 14](https://www.rfc-editor.org/info/bcp14) ([RFC 2119](https://www.rfc-editor.org/rfc/rfc2119), [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174)) when, and only when, they appear in all capitals.

Practical meaning:

| Term | Meaning in Circula specifications |
|---|---|
| MUST / MUST NOT | A platform-wide rule. The software enforces it in every network and no network policy can switch it off. A build that violates it is non-conforming. |
| SHOULD / SHOULD NOT | Expected behavior. A deviation is allowed only with a recorded reason in the decision log (see [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md)). |
| MAY | Permitted but optional. Often marks something a network administrator can configure. |

Non-normative phrases used with fixed meaning:

| Phrase | Meaning |
|---|---|
| **Recommended default** | The value or behavior a fresh installation or a new network starts with. A network administrator may change it unless the rule says otherwise. |
| **Open decision** | A choice the project owner has not made. The specification records options and a recommended default; implementers do not treat the recommendation as settled. |
| **Deferred** | Out of scope for the MVP. Named so the boundary is explicit, not to be designed in this pass. |
| **Counsel question** | A legal, tax, money-services, consumer-protection, privacy, or financial-regulatory question. Recorded as the product behavior that needs review and the decision needed before launch. Never answered in these documents. |

Words to avoid: "secure", "user-friendly", "intuitive", "robust", "scalable", "fast", and similar adjectives without a measurable condition. Replace them with observable behavior ("rejects the request with error code `limit_exceeded`", "operable with keyboard only", "restores a backup within 60 minutes").

## 3. Requirement identifiers

Every requirement, invariant, assumption, risk, and decision has a stable identifier of the form `PREFIX-NNN` (three digits, zero-padded, starting at 001).

Rules:

- An identifier is assigned once and never renumbered or reused. A withdrawn requirement keeps its identifier and gains the status `Withdrawn`, with a link to the replacement.
- Each prefix is owned by exactly one document. Only the owning document may define identifiers with that prefix.
- Other documents refer to a requirement by identifier and link, for example `[DOM-001](03-domain-model-and-mutual-credit-rules.md#dom-001)`. They MUST NOT restate the normative text.
- Every requirement has at least: identifier, priority, normative statement, rationale. Functional requirements additionally have acceptance criteria. Decisions, assumptions, and risks use the templates in [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md).
- Anchors: each identifier is written as a heading or a bold label with an explicit HTML anchor so that links resolve, for example `<a id="prefix-nnn"></a>**PREFIX-NNN**`.

Prefix registry:

| Prefix | Meaning | Owning document |
|---|---|---|
| VIS | Vision, goals, non-goals, success measures | [01-product-vision-and-scope.md](01-product-vision-and-scope.md) |
| PER | Personas, roles, permissions | [02-personas-roles-and-journeys.md](02-personas-roles-and-journeys.md) |
| JRN | Core user journeys | [02-personas-roles-and-journeys.md](02-personas-roles-and-journeys.md) |
| DOM | Domain model and mutual-credit rules (platform invariants, lifecycles) | [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) |
| FR | Functional requirements with acceptance criteria | [04-functional-requirements.md](04-functional-requirements.md) |
| ARCH | Architecture, components, trust boundaries, qualities | [05-system-architecture.md](05-system-architecture.md) |
| DATA | Data model, ledger storage, audit, retention, migration | [06-data-model-ledger-and-audit.md](06-data-model-ledger-and-audit.md) |
| API | API and integration contract | [07-api-and-integration-contract.md](07-api-and-integration-contract.md) |
| UX | UX and accessibility requirements | [08-ux-and-accessibility.md](08-ux-and-accessibility.md) |
| SEC | Security, privacy, abuse resistance | [09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md) |
| GOV | Governance and network operations | [10-governance-and-network-operations.md](10-governance-and-network-operations.md) |
| OPS | Deployment, operations, support | [11-deployment-operations-and-support.md](11-deployment-operations-and-support.md) |
| QA | Quality strategy, test layers, quality gates | [12-quality-strategy-and-test-plan.md](12-quality-strategy-and-test-plan.md) |
| E2E | Scenario-based end-to-end acceptance tests | [12-quality-strategy-and-test-plan.md](12-quality-strategy-and-test-plan.md) |
| ASM | Assumptions | [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md) |
| RISK | Risks and mitigations | [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md) |
| DEC | Decisions (made and open) | [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md) |

## 4. Priority levels

| Priority | Meaning |
|---|---|
| P0 | Required for the MVP. The MVP cannot ship without it. |
| P1 | Expected in the MVP. May slip to the first post-MVP release only with a recorded decision. |
| P2 | Deferred beyond the MVP. Listed to fix the scope boundary; not designed in this pass. |

Platform invariants in the domain model are always P0.

## 5. Document structure

Every specification starts with the header table used at the top of this document (Status, Owner role, Last updated, Scope, Dependencies, Related documents) and ends with two sections: **Open questions** and **Change history**.

- **Status** is one of `Draft`, `Review`, `Accepted`, `Superseded`.
- **Owner role** names a role, not a person (for example "Product architect", "Security reviewer").
- **Last updated** is an ISO 8601 date taken from the environment on the day of editing, never guessed.
- **Dependencies** lists the documents whose terms or rules this document relies on.
- **Open questions** lists unresolved items in the form `OQ-<PREFIX>-n` (per-document numbering) with the decision needed and who decides. Each open question is mirrored as a `DEC-` entry in the decision log.
- **Change history** is a short dated list of substantive edits.

Markdown rules: one `#` title per document; ATX headings; tables for structured requirements; Mermaid fenced blocks for diagrams (GitHub renders them) with a plain-text summary beneath each diagram for readers without rendering; relative links only within the repository; no trailing whitespace; wrap lines at the author's discretion.

## 6. Shared glossary

Short definitions for orientation. Full semantics, lifecycles, and invariants live in the [domain model](03-domain-model-and-mutual-credit-rules.md); the glossary never overrides it.

| Term | Definition |
|---|---|
| **Platform** | The Circula software as a whole, independent of who runs it. |
| **Instance** | One running deployment of the platform. An instance hosts one or more networks. |
| **Platform operator** | The organization or person who runs an instance: installs, upgrades, backs up, and supports it. Has no role inside any network's ledger by default. |
| **Network** | A community that trades in its own credit unit under its own rules, hosted on an instance. The unit of governance, policy, and ledger isolation. |
| **Network administrator** | A member of a network who holds the administrator role: configures network policy, manages memberships, posts adjustments, and acts on dispute outcomes. |
| **Dispute reviewer** | A member of a network who holds the reviewer role: examines disputes and records a resolution according to network policy. May be the same person as an administrator unless policy forbids it. |
| **Member** | A person or organization that has joined a network. A member belongs to exactly one network per membership; the same person may hold memberships in several networks. |
| **Membership** | The relationship between a person or organization and one network, with its own status and roles. |
| **Account** | A ledger account inside one network. Every active membership has exactly one member account in the MVP. Networks also have system accounts. |
| **System account** | An account owned by the network itself rather than by a member. The MVP requires one, the **network account**, used as the counterparty for adjustments such as write-offs so that the ledger always balances. |
| **Credit unit** | The accounting unit of one network, with a network-chosen name (for example "credits", "hours") and a fixed number of decimal places. It is not money, not a security, and not redeemable for national currency through the platform. |
| **Amount** | A positive integer count of the smallest fraction of a credit unit (minor units). Never a floating-point value. |
| **Posting** | One immutable accounting entry: a debit or a credit of an amount to one account, always created as one half of a pair. Also called a ledger entry. |
| **Balance** | The sum of all credits minus all debits posted to an account. Derived from postings, never stored as the only source of truth. May be negative. |
| **Credit limit** | The largest negative balance an account may reach through ordinary transactions, expressed as a non-negative amount. A credit limit of 500 means the balance may fall to −500 but not below. |
| **Positive limit** | An optional largest positive balance an account may reach through ordinary transactions. Unset means no positive limit. |
| **Available credit** | Balance plus credit limit: the amount an account can still spend through ordinary transactions. Pending transactions do not reduce it (see [DEC-004](13-assumptions-risks-decisions-and-roadmap.md#dec-004)). |
| **Listing** | A member-published description of something offered or requested. An **offer** describes goods or services the member will provide; a **request** describes goods or services the member seeks. |
| **Transaction** | A proposed transfer of an amount from a payer account to a payee account in the same network, with a description, which moves through a lifecycle and, when completed, creates exactly one pair of postings. |
| **Payer** | The account debited by a transaction. **Payee**: the account credited. |
| **Transaction status** | The lifecycle state of a transaction: Proposed, Completed, Declined, Cancelled, Expired, Reversed. |
| **Reversal** | A completed transaction's effect undone by a new, opposite pair of postings that reference the original. The original postings are never deleted or edited. |
| **Adjustment** | An administrator-initiated pair of postings with a recorded reason, used to correct errors or to record governance outcomes such as a write-off. Always balanced against another account, usually the network account. |
| **Dispute** | A member's formal challenge to a completed transaction or an adjustment, handled by a dispute reviewer under network policy, with a recorded resolution. |
| **Audit event** | An append-only record of who did what, to which object, when, and from which context, for every action that changes ledger state, policy, membership, roles, or access. |
| **Network policy** | The set of rules a network administrator can configure within platform invariants, for example default credit limit, proposal expiry, visibility of balances to other members. |
| **Platform invariant** | A rule enforced by the software in every network that no policy can relax. |
| **Idempotency key** | A client-supplied unique token on a state-changing request so that a retried request has the same effect as the first. |

## 7. Authoritative homes

| Topic | Authoritative home |
|---|---|
| Vocabulary and conventions | This document |
| Roles and permissions matrix | [02-personas-roles-and-journeys.md](02-personas-roles-and-journeys.md) |
| Entity definitions, lifecycles, ledger invariants, what is policy versus invariant | [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md) |
| MVP capability list and acceptance criteria | [04-functional-requirements.md](04-functional-requirements.md) |
| Components, trust boundaries, deployment topology, architectural qualities | [05-system-architecture.md](05-system-architecture.md) |
| Storage shape of the ledger, identifiers, audit event contents, retention, migration | [06-data-model-ledger-and-audit.md](06-data-model-ledger-and-audit.md) |
| API shape, errors, pagination, idempotency, versioning | [07-api-and-integration-contract.md](07-api-and-integration-contract.md) |
| Screens, presentation of balances, accessibility | [08-ux-and-accessibility.md](08-ux-and-accessibility.md) |
| Threat model, authentication, recovery, rate limiting, logging, incident response | [09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md) |
| What administrators decide, how decisions are announced, appeals, departure and default policy | [10-governance-and-network-operations.md](10-governance-and-network-operations.md) |
| Environments, configuration, backups, upgrades, support | [11-deployment-operations-and-support.md](11-deployment-operations-and-support.md) |
| Test layers, quality gates, end-to-end scenarios | [12-quality-strategy-and-test-plan.md](12-quality-strategy-and-test-plan.md) |
| Assumptions, risks, decision log, roadmap | [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md) |

## 8. Baseline decisions

These decisions were taken while writing this pass so that the documents agree. Each is recorded in full, with alternatives, in the [decision log](13-assumptions-risks-decisions-and-roadmap.md#decision-log). They are **recommended baselines**, not commitments by the project owner; any of them can be reopened there.

| ID | Decision | Why it matters here |
|---|---|---|
| [DEC-001](13-assumptions-risks-decisions-and-roadmap.md#dec-001) | Circula is **self-hostable first**. A hosted offering, if any, runs the same software. One instance can host many networks (multi-network), and each network's data is isolated by the software. | Fixes the deployment topology all documents assume. |
| [DEC-002](13-assumptions-risks-decisions-and-roadmap.md#dec-002) | The ledger is **double-entry and append-only**. Every completed trade, reversal, and adjustment is exactly one balanced pair of immutable postings. Balances are derived from postings. | The core correctness rule; restated nowhere, enforced everywhere. |
| [DEC-003](13-assumptions-risks-decisions-and-roadmap.md#dec-003) | Every network has a **network account** (a system account). Write-offs and governance adjustments post against it, so the sum of all balances in a network is always zero. | Makes losses visible instead of silently destroying credits. |
| [DEC-004](13-assumptions-risks-decisions-and-roadmap.md#dec-004) | A transaction **posts at the moment both parties have consented**, and credit-limit checks run atomically at that moment. Proposed transactions do not reserve available credit. | Simplest model that is still correct under concurrency; the reservation model is recorded as an alternative. |
| [DEC-005](13-assumptions-risks-decisions-and-roadmap.md#dec-005) | Amounts are **integers in minor units** with a per-network number of decimal places (0 to 2). No floating-point arithmetic anywhere in the ledger. | Prevents rounding drift. |
| [DEC-006](13-assumptions-risks-decisions-and-roadmap.md#dec-006) | The MVP client is a **responsive web application** that uses the public API; there is no privileged internal path to the ledger. | The API and UI enforce identical rules because they are the same path. |
| [DEC-007](13-assumptions-risks-decisions-and-roadmap.md#dec-007) | **Email** is the MVP channel for invitations, notifications, and account recovery. | Fixes the minimum identity data collected. |
| [DEC-008](13-assumptions-risks-decisions-and-roadmap.md#dec-008) | The credit unit is an **accounting unit only**. The platform performs no conversion to or from national currency and offers no redemption. | Keeps counsel questions from shaping the ledger design. |
| [DEC-009](13-assumptions-risks-decisions-and-roadmap.md#dec-009) | The ledger requires a **transactional store with serializable or equivalent guarantees** for postings; no database vendor is chosen. | Technology-neutral, but rules out stores that cannot post atomically. |
| [DEC-010](13-assumptions-risks-decisions-and-roadmap.md#dec-010) | The initial architecture is a **modular monolith** with the ledger as an isolated core module behind an explicit interface. | Smallest design that keeps the ledger testable alone and portable later. |

## 9. Writing rules

1. **Observable, not aspirational.** Every normative sentence names a behavior a test can check.
2. **One home per rule.** Define once, link everywhere. If you find yourself restating a rule, replace the restatement with the identifier and a link.
3. **Mark uncertainty.** Anything not decided is an open question with options and a recommended default, never a silent requirement.
4. **No legal conclusions.** Describe the product behavior and the decision needed; do not say a design is compliant, lawful, or exempt.
5. **No monetary framing.** Do not describe credits as money, currency, payment, cash, deposits, or redeemable value. "Credit", "credit unit", "balance", "transaction", and "trade" are the approved terms. Note: the word "payer"/"payee" is used for the debited/credited account because no shorter unambiguous term exists; it does not imply money.
6. **Minimize personal data in examples.** Use placeholder names and addresses such as `member@example.org`.
7. **Dates from the environment.** Never guess a date.
8. **Cite external sources** with URL and access date when a technical standard is relied on.
9. **Scope discipline.** Foundational capabilities only. Anything that reads like a feature sub-specification (detailed interaction flows, polished visual design, integration-specific contracts) is deferred by name.

## 10. Open questions

| ID | Question | Decision needed | Decided by |
|---|---|---|---|
| OQ-CONV-1 | Should specifications adopt a formal ADR directory (`docs/adr/`) in addition to the decision log table? | Whether the decision log in document 13 is sufficient until implementation begins. | Project owner |

## 11. Change history

| Date | Change |
|---|---|
| 2026-11-11 | Initial draft. |
