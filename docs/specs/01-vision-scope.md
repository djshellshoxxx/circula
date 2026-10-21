# 01 Product Vision and Scope

Status: Draft v0.1 · Owner: Product architect · Last updated: 2026-10-21 · Scope: product purpose, users at a high level, goals, non-goals, principles, success measures, MVP boundary · Depends on: [00-conventions.md](00-conventions.md) (glossary, ID prefixes, invariants LED-1..7, configurable defaults) · Open questions: section 10

Terms such as *network*, *member*, *credit*, *transaction* and *system account* are defined once in the glossary of [00-conventions.md](00-conventions.md#3-glossary-authoritative). This file does not redefine them. Normative terms (MUST, SHOULD, MAY) follow [00-conventions.md](00-conventions.md#1-normative-terms).

---

## 1. Purpose and statement of limits

Circula is software for operating a **community mutual-credit network**. Members provide goods or services, receive a positive balance when they provide value, and use credits earned to obtain value from other members. Circula records these exchanges as balanced ledger postings, enforces the network's configured rules, and keeps an audit trail.

Circula is **not** a currency, payment system, or redemption channel. The software:

- does **not** guarantee that credits have monetary value;
- does **not** guarantee that any member or merchant will accept credits;
- does **not** convert credits to or from national currency, and does **not** hold or move national currency on behalf of members in the MVP;
- does **not** make a network's governance decisions. Each network decides its own participation, limit, dispute and governance rules within the bounds the platform allows (see [03-domain-model.md](03-domain-model.md)).

Whether credits have value, and whether members accept them, is a property of each network's community and rules. The specification describes how Circula behaves; it does not make claims about the economic or legal status of the credit unit.

## 2. Problem

Communities that lack easy access to cash-based exchange often trade informally. Common tools (shared spreadsheets, chat threads, paper tallies) have known weaknesses:

- balances are edited in place, so errors and disputes cannot be reconstructed;
- nothing checks that both sides of a trade were recorded consistently;
- negative balances and limits are enforced by convention, not by a system;
- there is no clear record of who approved a trade, changed a rule, or resolved a complaint;
- members cannot easily see what they have earned, spent, or still have available.

Circula addresses these by making the ledger, the rules, and the audit trail explicit and checkable.

## 3. Intended users

The four roles are defined in [02-personas-journeys.md](02-personas-journeys.md):

- **Network member**: joins a network, lists offers and requests, proposes and accepts trades, reviews balance and history, raises disputes.
- **Network administrator**: sets up and configures a network, manages membership and limits, makes audited adjustments, handles governance changes.
- **Dispute reviewer**: reviews disputed transactions and records decisions.
- **Platform operator**: runs the Circula software deployment (hosted or self-hosted) and keeps it available, backed up and upgraded.

## 4. Goals

Goals are the outcomes the MVP must enable. Each goal is traced to requirements.

- **G1. Trustworthy books.** Every completed trade is two balanced postings; balances can always be recomputed from postings; no silent edits. (LED-1..LED-7; DOM; DAT.)
- **G2. Clear rules per network.** Each network can set its own credit limit, expiry, dispute window and membership join mode within stated bounds. (DOM-028..DOM-031; FR-003.)
- **G3. Low-friction trades.** A member can find a counterparty, propose a trade, and see the result without staff help. (FR-021..FR-035; PER journeys.)
- **G4. Understandable balances.** A member can always tell whether they are in credit or have used credit, how much is still available, and why. (UX; FR-050..FR-054.)
- **G5. Recourse.** A member can contest a completed trade, and the outcome is recorded with reasons and reversed by new postings, never by editing. (DOM-017..DOM-021; FR-070..FR-078.)
- **G6. Accountability.** Every administrative or reviewer action is attributable and visible to the appropriate roles. (FR-080..FR-084.)
- **G7. Portability.** Members and administrators can export their data in open formats. (FR-090..FR-095.)

## 5. Non-goals (for the MVP and this specification pass)

- Converting credits to national currency, redemption, cash-out, or any payment-processing function.
- Tokenization, blockchain or cryptocurrency representation of credits.
- Investment, yield, or speculative features.
- Cross-network trading, federation, or shared ledgers between networks.
- Reputation or credit scoring, and automated decisions about members.
- Native mobile applications.
- Third-party integrations (payments, accounting, identity providers, messaging) beyond what is needed for basic email delivery if decided later (see section 10).
- Advanced analytics and reporting beyond the exports in [04-functional-requirements.md](04-functional-requirements.md).
- Monetization features, hosted subscriptions, or paid administration features. These are acknowledged as possible future work and are out of scope for this pass.
- Legal, tax, money-services, consumer-protection or privacy compliance claims. These are open questions for qualified counsel (see section 10).

## 6. Guiding principles

These principles resolve trade-offs when requirements conflict. A reviewer SHOULD cite a principle when rejecting a change.

1. **The ledger is the truth.** Derived views (balances, dashboards, caches) MUST be reproducible from postings.
2. **Corrections add, they do not edit.** Completed records are never altered; errors are fixed by new, attributable transactions.
3. **Consent before debit.** No member's balance decreases because of a trade they did not accept.
4. **Each network governs itself, within platform bounds.** Configurable policy is per network; platform invariants are not configurable.
5. **Say what the number means.** Every balance, limit and pending amount is shown with direction and status in words as well as numbers.
6. **Minimise personal data.** Collect and retain only what a trade, membership, dispute or audit requires.
7. **Technology-neutral requirements.** Behavioural requirements do not depend on a framework, database vendor or hosting provider unless the repository decides so (see [00-conventions.md](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log)).
8. **Honest scope.** Features that are deferred are named as deferred; the product does not imply they exist.

## 7. Success measures

These measures indicate whether the MVP achieves its goals. Thresholds marked **TBD** require a product decision before they are binding. Measures are reviewed at each release gate (see [12-quality-test-plan.md](12-quality-test-plan.md)).

| # | Measure | Target | Goal |
|---|---|---|---|
| SM-1 | Ledger invariant violations detected in production (balance mismatch, unbalanced transaction, over-limit posting) | 0 | G1 |
| SM-2 | Share of completed trades whose both postings and audit event were written atomically (checked by the consistency command) | 100% | G1, G6 |
| SM-3 | Time from proposal to completed trade for a pilot member in a pilot network, median | TBD (product decision) | G3 |
| SM-4 | Share of members who can state their available credit correctly on a usability task | TBD (product decision; measured in UX testing) | G4 |
| SM-5 | Share of disputes with a recorded decision and rationale | 100% of closed disputes | G5 |
| SM-6 | Share of export runs whose totals match the ledger checksum | 100% | G7 |
| SM-7 | Accessibility defects open at release for the MVP screens | TBD (threshold per [08-ux-accessibility.md](08-ux-accessibility.md)) | G3, G4 |

## 8. MVP boundary

The MVP includes the P0 and P1 capabilities in [04-functional-requirements.md](04-functional-requirements.md): network setup, membership, directory and listings, trade proposal and acceptance, balances and history, administrator controls, disputes, audit history and CSV/JSON exports.

The MVP excludes everything listed as deferred in [04-functional-requirements.md](04-functional-requirements.md#10-deferred-capabilities-explicit-list). Deferred items are not partially implemented behind hidden switches.

## 9. Requirements

### VIS-001 · P0 · Scope statement is accurate
- **Statement.** Circula documentation and user-facing text MUST describe the software as a tool for operating a mutual-credit network, and MUST NOT describe credits as currency, investment, or redeemable value.
- **Rationale.** Misdescribing the credit unit creates legal and financial risk that the platform cannot control (section 1).
- **Acceptance.**
  - A review checklist for every user-facing string and every spec file finds no statement that credits are redeemable, convertible, investable or guaranteed.
  - A grep of the repository for the terms "redeem", "cash out", "guaranteed value", "investment" returns only hits inside this file's disclaimers or in deferred-capability notes.

### VIS-002 · P0 · Network acceptance is not guaranteed
- **Statement.** The MVP interface and documentation MUST state that acceptance of credits is governed by each network, not guaranteed by the platform.
- **Rationale.** Members must not infer a promise of acceptance from the software.
- **Acceptance.** The network "about" page and the README (when created) contain the statement; QA checks it on the release candidate.

### VIS-003 · P0 · Networks are isolated tenants
- **Statement.** No data, listing, balance, transaction, dispute or audit record MAY be visible to, or referenced from, a different network.
- **Rationale.** The glossary defines the network as a tenant boundary.
- **Acceptance.** Cross-tenant automated tests (see [12-quality-test-plan.md](12-quality-test-plan.md)) show that a principal of network A receives "not found" for every identifier of network B across all read and write endpoints.

### VIS-004 · P0 · Ledger invariants are enforced and tested
- **Statement.** Invariants LED-1 to LED-7 ([00-conventions.md](00-conventions.md#4-platform-wide-invariants-not-configurable)) MUST hold in every release.
- **Rationale.** Ledger integrity is the first goal (G1).
- **Acceptance.** Each LED invariant has at least one automated test at the layer named in [12-quality-test-plan.md](12-quality-test-plan.md), and all pass in the release pipeline.

### VIS-005 · P0 · Per-network rule configuration
- **Statement.** Each network MUST be able to set its own values for the configurable policies listed in [00-conventions.md](00-conventions.md#5-configurable-network-policies-defaults) and in [03-domain-model.md](03-domain-model.md#13-configuration-versus-invariants), within the bounds stated there.
- **Rationale.** Goal G2; principle 4.
- **Acceptance.** Two networks on the same deployment can hold different values for each configurable policy, and a change in one does not alter the other (automated test).

### VIS-006 · P0 · MVP completeness gate
- **Statement.** Every P0 and P1 requirement in [04-functional-requirements.md](04-functional-requirements.md) MUST have a passing acceptance test before the MVP is declared complete.
- **Rationale.** Prevents silent scope reduction.
- **Acceptance.** The traceability matrix in [12-quality-test-plan.md](12-quality-test-plan.md) lists a passing test ID for each FR P0 and P1 item, or an approved, recorded deferral.

### VIS-007 · P0 · Deployment shape
- **Statement.** The MVP MUST be deliverable as one application artifact that can be self-hosted, and MUST be usable by a hosted operator without changes to the application's behaviour.
- **Rationale.** DEC-001 in [00-conventions.md](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log); the deployment choice is detailed in [05-architecture.md](05-architecture.md) and [11-deployment-operations.md](11-deployment-operations.md).
- **Acceptance.** A clean-machine install from the release artifact reaches a working network with default configuration, with no hosted services required.

### VIS-008 · P1 · Accessibility baseline
- **Statement.** MVP screens MUST meet the accessibility requirements in [08-ux-accessibility.md](08-ux-accessibility.md).
- **Rationale.** Members with disabilities must be able to trade and check balances (G3, G4).
- **Acceptance.** The release gate in [12-quality-test-plan.md](12-quality-test-plan.md) includes an accessibility review with zero open blocking defects.

### VIS-009 · P1 · Data exportability
- **Statement.** Members and administrators MUST be able to export their data in the formats defined in FR-090 to FR-095.
- **Rationale.** Goal G7; principle 6.
- **Acceptance.** A round-trip test exports a network's ledger and checks that totals and per-member balances recomputed from the export equal the source (see FR-091).

### VIS-010 · P1 · Data minimisation
- **Statement.** Every personal data field MUST appear in a data inventory with its purpose and retention rule, and fields outside the inventory MUST NOT be stored.
- **Rationale.** Principle 6; the security baseline in [09-security-privacy.md](09-security-privacy.md).
- **Acceptance.** The inventory exists in [06-data-ledger-audit.md](06-data-ledger-audit.md) or [09-security-privacy.md](09-security-privacy.md); the schema test fails if an undocumented personal field is added.

### VIS-011 · P1 · No monetization in the MVP
- **Statement.** The MVP MUST NOT include billing, subscription, payment or paid-feature code paths, enabled or disabled.
- **Rationale.** Section 5; monetization is outside this pass.
- **Acceptance.** A repository search for billing and payment-provider identifiers finds none in the MVP build; a reviewer confirms in the release checklist.

### VIS-012 · P2 · Deferred capabilities stay inactive
- **Statement.** Capabilities in the deferred list of [04-functional-requirements.md](04-functional-requirements.md#10-deferred-capabilities-explicit-list) MUST NOT be reachable from any MVP interface or API route.
- **Rationale.** Principle 8.
- **Acceptance.** Route inventory test confirms no deferred route exists in the MVP build.

## 10. Open questions

- **OQ-VIS-1 (product owner).** Target MVP pilot profile: number of networks, members per network, and whether pilots are hosted, self-hosted or both. Affects SM-3 and deployment choice.
- **OQ-VIS-2 (product owner).** Thresholds for SM-3, SM-4 and SM-7.
- **OQ-VIS-3 (counsel, before launch).** Whether the statements in VIS-001 and VIS-002 need jurisdiction-specific wording. This spec makes no legal assertion.
- **OQ-VIS-4 (product owner).** Whether email delivery of notifications is part of the MVP. Currently treated as a P2 item; the FR list currently includes in-app notification only.
- **OQ-VIS-5 (project owner).** Licence (DEC-004 in [00-conventions.md](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log)). Not decided here.

Open questions in this file are labelled `OQ-VIS-n` and are to be consolidated into [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).
