# Assumptions, risks, decisions, and roadmap

**Status:** Draft decision register  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Unresolved product and architecture choices, material risks, sequencing  
**Dependencies:** All foundational specifications  
**Open questions:** Items below require project-owner selection before implementation commitments.

## Assumptions

| ID | Assumption | Impact if false |
|---|---|---|
| A-001 | Each network uses one unit of account at a time. | Multi-unit accounting changes ledger, display, exchange, and export models. |
| A-002 | A basic exchange is between two member accounts and one amount. | Group trades or split settlement require expanded transaction semantics. |
| A-003 | Circula records community accounting positions; it does not promise currency redemption or acceptance. | Any redemption or payment feature materially changes product and legal review. |
| A-004 | Networks govern their own participation and dispute rules within platform invariants. | A centralized governance model changes platform/operator permissions. |

## Decisions to make

| ID | Decision | Options | Recommended default | Needed by |
|---|---|---|---|---|
| DEC-001 | Initial deployment | Hosted; self-hosted; both | Choose one reference deployment for MVP while keeping boundaries portable | Architecture implementation |
| DEC-002 | Pending credit reservation | No reservation until posting; reserve at proposal; network-configurable | Do not reserve in MVP; recheck at acceptance and define race-safe posting | Ledger implementation |
| DEC-003 | Network/account topology | One account per member per network; multiple accounts | One account per member per network for MVP | Data model freeze |
| DEC-004 | Membership admission | Invite; application/approval; open registration | Administrator-approved invitation/application | Pilot workflow |
| DEC-005 | Accessibility target | Select applicable WCAG conformance level and evaluation method | Adopt a stated WCAG 2.2 level before public launch | UX implementation |
| DEC-006 | Retention and deletion | Define per record class and jurisdiction | Separate identity data from ledger references; set periods after counsel review | Production readiness |
| DEC-007 | API protocol and authentication | Select during architecture design | Keep conceptual contract protocol-neutral until deployment needs are known | API implementation |
| DEC-008 | Performance and recovery goals | Set latency, availability, RPO/RTO values | Measure pilot baseline, then ratify numeric gates | Production readiness |

## Risks and mitigations

- **Unclear unit meaning or member expectations:** explain the unit and acceptance limits during onboarding and before confirmation.
- **Unbalanced or duplicate accounting:** centralize posting, use atomic transactions and idempotency, and test concurrent retries.
- **Administrator misuse:** least privilege, attributable actions, role history, and a member-visible governance process.
- **Credit-limit race conditions:** recheck limits at commit under concurrency control; define pending reservation behavior.
- **Sensitive dispute evidence exposure:** restrict evidence access, log reads/exports, and minimize general logs.
- **Network closure leaves unclear balances:** define closure, export, open-dispute, and balance treatment before pilot.
- **Legal/regulatory uncertainty:** record product behavior and intended jurisdictions for qualified counsel; do not claim compliance.

## Roadmap dependency order

1. Resolve DEC-001 through DEC-004 and confirm pilot assumptions.
2. Freeze domain vocabulary, transaction lifecycle, and ledger invariants.
3. Specify storage/API/auth choices and detailed interaction designs as separate follow-on work.
4. Implement MVP requirements with invariant-driven tests.
5. Exercise export, restore, accessibility, and incident procedures before production.
6. Commission jurisdiction-specific legal/privacy review before launch and before any monetary conversion, redemption, or payment functionality.

## Decision log template

For each future decision record: ID, date, status, context, decision, alternatives considered, consequences, owner, and links to affected specs. Keep accepted decisions in one authoritative log and update this register to link them.
