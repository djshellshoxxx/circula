# Functional requirements and acceptance criteria

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Foundational MVP capabilities only  
**Dependencies:** [Domain model](03-domain-model.md), [roles](02-roles-and-journeys.md)  
**Open questions:** Default member invitation flow, network closure behavior, and dispute deadlines.

Priority: **P0** required for MVP; **P1** desirable for initial pilot. Acceptance criteria are observable, technology-independent checks.

| ID | Priority | Requirement and rationale | Acceptance criteria |
|---|---|---|---|
| FR-001 | P0 | An authorized administrator can create and configure a network. | Create a network with name, unit label, participation policy, and initial administrator; unauthorized user cannot create or alter it. |
| FR-002 | P0 | A network can admit, suspend, reinstate, and depart members. | Each transition records actor, time, and outcome; suspended members cannot initiate new trades; history remains available to authorized parties. |
| FR-003 | P0 | Members can publish and manage offers and requests. | Listing includes owner, description, status, and visibility; owner or authorized administrator can change status; closed listings are not presented as active. |
| FR-004 | P0 | Members can propose and accept or decline a trade. | Proposal shows amount, both counterparties, descriptions, and expected balance effects; only required parties can accept; posting revalidates current constraints and is atomic. |
| FR-005 | P0 | Members can review balances and transaction history. | Displayed balance reconciles to posted entries; pending items are visibly separate; filters do not omit corrections from the underlying history. |
| FR-006 | P0 | Members can raise disputes and authorized reviewers can record outcomes. | A dispute is linked to the challenged transaction; submitter receives a case reference; decisions and evidence access are permission-checked and recorded. |
| FR-007 | P0 | Administrators can perform permitted policy and membership actions. | Each change shows its scope and consequence, requires authorization, records before/after values where appropriate, and cannot mutate past postings silently. |
| FR-008 | P0 | The system records audit history for material actions. | A permitted auditor can find actor, action, target, timestamp, and outcome for network administration, posting, correction, and dispute decisions. |
| FR-009 | P0 | An authorized network user can export network records. | Export is scoped to the selected network and caller’s permission; includes ledger entries and links needed to reconcile balances; export is logged. |
| FR-010 | P0 | The system explains unavailable or rejected operations. | Insufficient credit, permission failure, validation error, and stale-state rejection identify the reason and whether posting occurred; safe retry does not duplicate a posting. |
| FR-011 | P1 | Network members can inspect policy and its change history. | Current policy and effective date are visible to members; prior changes identify approver, time, and notice reference. |
| FR-012 | P1 | Administrators can export a human-readable audit subset. | Export respects role scope, identifies time range and filters, and does not expose secrets or unrelated personal data. |

## Deferred

Mobile applications, federation, tokenization, conversion or redemption to national currency, reputation scoring, advanced analytics, automated credit decisions, payment processing, and integrations require separate scope approval and specifications. They are not implied by the requirements above.
