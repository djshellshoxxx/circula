# Product Vision and Scope

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Product owner |
| Last updated | 2026-11-11 |
| Scope | Why Circula exists, who it serves, what it sets out to achieve, what it deliberately does not do, the principles that guide trade-offs, how success is observed, and where the MVP boundary lies. |
| Dependencies | [00-conventions.md](00-conventions.md) (language, identifiers, glossary, baseline decisions). |
| Related documents | [Personas, roles, and journeys](02-personas-roles-and-journeys.md), [Domain model and mutual-credit rules](03-domain-model-and-mutual-credit-rules.md), [Functional requirements](04-functional-requirements.md), [Assumptions, risks, decisions, and roadmap](13-assumptions-risks-decisions-and-roadmap.md) |

Terms in **bold** on first use follow the [shared glossary](00-conventions.md#6-shared-glossary). This document owns the `VIS` identifier prefix. It states intent and boundaries; rules, requirements, and acceptance criteria live in the documents named as authoritative homes in [00-conventions.md](00-conventions.md#7-authoritative-homes).

## 1. Problem

Local communities, cooperatives, time-banks, and trade circles often have plenty of unused capacity (skills, tools, space, time) and unmet needs that sit side by side, yet members cannot match them because there is no shared way to record who has given and who has received. Groups that try to run a mutual-credit network today face recurring problems:

1. **Bookkeeping by hand.** Spreadsheets and chat threads record trades inconsistently. Balances drift, entries are edited after the fact, and nobody can show how a figure was reached.
2. **No agreed record of consent.** A trade is claimed by one side and later contested by the other because no step captured both parties agreeing.
3. **Unclear limits.** Without an enforced credit limit, one member can accumulate a large negative balance that the group never agreed to carry.
4. **Opaque decisions.** Administrators correct errors or write off losses without a visible, reviewable record, which erodes trust.
5. **Hard to find each other.** Offers and requests are scattered, so credit that could circulate sits idle.
6. **Lock-in and fragility.** A network's history lives in one organizer's files and disappears when that person leaves.

Existing general-purpose tools either assume a payment model that does not fit a community accounting unit, or are bespoke systems that a small group cannot operate or audit. Circula aims to give such groups a correct, inspectable, self-operable ledger and the surrounding tools to run it.

## 2. Intended users

Circula serves four kinds of people. Roles, permissions, and journeys are defined in [02-personas-roles-and-journeys.md](02-personas-roles-and-journeys.md); this section names them only to fix the audience of the goals below.

| User | What they need from Circula |
|---|---|
| **Network member** | Find offers and requests, propose and accept trades, see a trustworthy balance and history, raise a dispute. |
| **Network administrator** | Set up a network, choose its policy, admit and remove members, post adjustments with a recorded reason, act on dispute outcomes. |
| **Dispute reviewer** | See the evidence for a contested transaction or adjustment and record a resolution under network policy. |
| **Platform operator** | Install, upgrade, back up, and support an instance that hosts one or more networks, without needing access to any network's ledger decisions. |

Typical networks are small to mid-sized groups whose members know one another or share a locality or interest. Network size targets are an open question (see [OQ-VIS-2](#9-open-questions)).

## 3. Goals

Each goal has a priority (see [priority levels](00-conventions.md#4-priority-levels)), a rationale, and a success measure that can be observed. Numeric thresholds are decided in [section 7](#7-success-measures).

<a id="vis-001"></a>**VIS-001** | Priority P0
- **Goal.** Circula MUST give every network a ledger whose balances can be reconstructed exactly from immutable postings, with no silent change to any balance.
- **Rationale.** Trust in a mutual-credit network rests on members believing the numbers. Rules are defined in [DOM](03-domain-model-and-mutual-credit-rules.md) and the ledger decision [DEC-002](13-assumptions-risks-decisions-and-roadmap.md#dec-002).
- **Success measure.** For every network, the sum of all balances is zero, and recomputing every balance from postings reproduces the displayed balance, in 100% of automated checks.

<a id="vis-002"></a>**VIS-002** | Priority P0
- **Goal.** Circula MUST let two members agree on a trade and record it only after both have consented.
- **Rationale.** Removes disputes about whether a trade was agreed; consent is the basis of every posting.
- **Success measure.** Every completed transaction carries a recorded proposal by one party and acceptance by the other; no transaction completes without both.

<a id="vis-003"></a>**VIS-003** | Priority P0
- **Goal.** Circula MUST enforce each account's credit limit at the moment a transaction completes, including under concurrent requests.
- **Rationale.** Limits are the network's main protection against unbounded negative balances ([DEC-004](13-assumptions-risks-decisions-and-roadmap.md#dec-004)).
- **Success measure.** In concurrency tests, no ordinary transaction takes any account beyond its credit limit.

<a id="vis-004"></a>**VIS-004** | Priority P0
- **Goal.** Circula MUST let a network administrator configure network policy within platform invariants and MUST record every policy change.
- **Rationale.** Networks differ in limits, expiry, and visibility; the platform fixes only what protects ledger correctness.
- **Success measure.** Each policy change appears in the audit history with actor, time, old value, and new value.

<a id="vis-005"></a>**VIS-005** | Priority P0
- **Goal.** Circula MUST let members publish and browse offers and requests inside their network.
- **Rationale.** Credit only circulates if members can find each other.
- **Success measure.** A member can publish a listing and another member of the same network can find it through the directory without leaving the application.

<a id="vis-006"></a>**VIS-006** | Priority P0
- **Goal.** Circula MUST give members a balance and a history in which direction and meaning are unmistakable.
- **Rationale.** Confusion over whether a balance is owed or earned is a main cause of disputes. Presentation rules live in [08-ux-and-accessibility.md](08-ux-and-accessibility.md).
- **Success measure.** In usability checks with the participants described in [OQ-VIS-3](#9-open-questions), participants correctly state whether a sample balance means credit earned or credit spent. Threshold needs product decision.

<a id="vis-007"></a>**VIS-007** | Priority P0
- **Goal.** Circula MUST provide a dispute process for completed transactions and adjustments that ends in a recorded resolution.
- **Rationale.** Members need a defined route to challenge an outcome; unresolved grievances end networks.
- **Success measure.** Every dispute reaches a terminal status with a recorded resolution and reviewer, and every resulting ledger change is a traceable reversal or adjustment.

<a id="vis-008"></a>**VIS-008** | Priority P0
- **Goal.** Circula MUST keep an append-only audit history of every action that changes ledger state, policy, membership, roles, or access.
- **Rationale.** Visible decisions keep administrators accountable and allow independent checking.
- **Success measure.** For a sampled set of state-changing actions, each has exactly one audit event naming actor, object, time, and context.

<a id="vis-009"></a>**VIS-009** | Priority P1
- **Goal.** Circula SHOULD let a network export its members' data, transactions, and audit history in documented, non-proprietary formats.
- **Rationale.** Prevents lock-in and lets a network leave an instance or close with its record intact.
- **Success measure.** A network's export can be read without Circula software, and an independent recomputation of balances from the exported postings matches the displayed balances.

<a id="vis-010"></a>**VIS-010** | Priority P0
- **Goal.** Circula MUST be operable by a small volunteer group on its own infrastructure, with one instance able to host several isolated networks ([DEC-001](13-assumptions-risks-decisions-and-roadmap.md#dec-001)).
- **Rationale.** The target users rarely have engineering staff; self-hosting is the baseline, not an advanced option.
- **Success measure.** A person with general system-administration skills, following only the published instructions, brings up an instance with one network. The time limit needs product decision.

<a id="vis-011"></a>**VIS-011** | Priority P0
- **Goal.** Circula MUST be usable by members with assistive technology and on small screens in the MVP client ([DEC-006](13-assumptions-risks-decisions-and-roadmap.md#dec-006)).
- **Rationale.** A community tool excludes the people it exists for if it fails on a phone browser or with a screen reader.
- **Success measure.** Every MVP journey is completable with keyboard only and at a 320 px viewport width. The conformance target is set in [08-ux-and-accessibility.md](08-ux-and-accessibility.md).

<a id="vis-012"></a>**VIS-012** | Priority P1
- **Goal.** Circula SHOULD collect only the personal data a network needs to operate ([DEC-007](13-assumptions-risks-decisions-and-roadmap.md#dec-007)).
- **Rationale.** Small communities hold sensitive information about neighbors; data not collected cannot leak.
- **Success measure.** The documented list of personal data fields collected at sign-up contains only those justified in [09-security-privacy-and-abuse-resistance.md](09-security-privacy-and-abuse-resistance.md).

## 4. Non-goals

These are outside the product's purpose, not merely postponed. Items that are only postponed are listed in [section 8.2](#82-explicitly-deferred).

<a id="vis-013"></a>**VIS-013** | Non-goal
Circula does not make credits redeemable for national currency, and the platform does not convert between credits and national currency ([DEC-008](13-assumptions-risks-decisions-and-roadmap.md#dec-008)).

<a id="vis-014"></a>**VIS-014** | Non-goal
Circula does not guarantee that credits have any monetary value, that any member will accept credits in exchange for goods or services, or that any member will repay a negative balance.

<a id="vis-015"></a>**VIS-015** | Non-goal
Circula does not set or enforce the price, quality, safety, or legality of the goods and services members trade. Members and networks agree these between themselves.

<a id="vis-016"></a>**VIS-016** | Non-goal
Circula does not issue or manage a cryptographic token, run a distributed ledger, or present credits as a security or investment.

<a id="vis-017"></a>**VIS-017** | Non-goal
Circula does not decide a network's governance. The platform supplies configurable controls and records; each network chooses and owns its rules, and software enforcement is distinct from governance policy ([10-governance-and-network-operations.md](10-governance-and-network-operations.md)).

<a id="vis-018"></a>**VIS-018** | Non-goal
Circula does not provide legal, tax, or regulatory advice, and no specification in this directory asserts that any design is compliant or exempt. Such questions are recorded as counsel questions ([section 9](#9-open-questions)).

<a id="vis-019"></a>**VIS-019** | Non-goal
Circula does not give platform operators a way to alter a network's ledger outside the rules in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md); operating an instance confers no ledger authority by default.

<a id="vis-020"></a>**VIS-020** | Non-goal
Circula does not claim member anonymity. Network membership and trading activity are visible according to network policy.

## 5. Guiding principles

When requirements pull in different directions, these principles settle the question. Earlier principles take precedence over later ones only where a conflict is recorded in the decision log.

<a id="vis-021"></a>**VIS-021** | Principle: Correctness before convenience
A feature that cannot preserve the ledger invariants does not ship, however useful it appears. Ledger postings are never edited or deleted; mistakes are corrected by a new, traceable pair of postings.

<a id="vis-022"></a>**VIS-022** | Principle: Nothing happens silently
Every change to balances, policy, membership, roles, or access leaves a visible record. A member can always see why their balance is what it is.

<a id="vis-023"></a>**VIS-023** | Principle: Consent of both parties
A trade moves balances only after both parties have agreed. Administrators correct and arbitrate through recorded adjustments and reversals, not by moving balances unilaterally and invisibly.

<a id="vis-024"></a>**VIS-024** | Principle: Network autonomy within platform invariants
Each network sets its own participation, limit, dispute, and governance policy. The platform restricts only what protects ledger integrity, and the line between invariant and policy is drawn once, in [03-domain-model-and-mutual-credit-rules.md](03-domain-model-and-mutual-credit-rules.md).

<a id="vis-025"></a>**VIS-025** | Principle: Accounting unit, not value claim
Everything the product says and shows treats the credit unit as a record of reciprocity inside one network. Wording and screens avoid implying redemption or monetary worth.

<a id="vis-026"></a>**VIS-026** | Principle: One path to the ledger
The user interface and the public API reach the ledger by the same path and enforce identical rules ([DEC-006](13-assumptions-risks-decisions-and-roadmap.md#dec-006)).

<a id="vis-027"></a>**VIS-027** | Principle: Operable and portable by the community
A network can run Circula itself, inspect what it holds, and take its data elsewhere. Open formats and documented behavior are preferred over convenience features that create dependence on one operator.

<a id="vis-028"></a>**VIS-028** | Principle: Collect little, show only what policy allows
Data not needed to run a network is not collected. Visibility of balances and member information follows network policy, and the default is the narrower visibility.

<a id="vis-029"></a>**VIS-029** | Principle: Say what is unknown
Uncertain choices are recorded as assumptions or open decisions with options and a recommended default. The specifications never present a recommendation as settled.

## 6. What Circula is and is not

**Circula is software for operating a network.** It records agreed trades between members of a network as balanced postings in that network's own credit unit, shows balances and history, and gives administrators and dispute reviewers the controls and records they need to run the network under its chosen rules.

**Circula does not itself guarantee that credits have monetary value, nor that members will accept them.** Whether a credit is worth anything to a member, and whether another member will provide goods or services in return for it, is decided by the people in the network. The platform records those decisions; it does not back, insure, or promise them. This statement is the product-level form of [VIS-014](#vis-014) and applies to every specification, screen, and message the product produces.

**The credit unit is an accounting unit.** It is a named unit with a fixed number of decimal places, chosen per network, used to record reciprocity. The platform performs no conversion to or from national currency and offers no redemption ([DEC-008](13-assumptions-risks-decisions-and-roadmap.md#dec-008), [VIS-013](#vis-013)). It is not a cryptocurrency, not a security, and not a deposit.

| Circula is | Circula is not |
|---|---|
| Software a platform operator installs and a network uses | An operator-run service that holds members' credits on their behalf |
| A double-entry, append-only record of agreed trades | A store of value or a promise of value |
| A set of configurable controls for network policy | A source of governance rules for any network |
| A directory where members find offers and requests | A marketplace that sets prices or vouches for quality |
| An audit trail of decisions | A guarantee that every decision was fair or lawful |

## 7. Success measures

Measures are observable from the product, its automated checks, or defined research. Where a number is needed and not yet chosen, the measure is marked **threshold needs product decision** and is tracked as an open question. Test methods live in [12-quality-strategy-and-test-plan.md](12-quality-strategy-and-test-plan.md).

| No. | Measure | How observed | Linked goals | Threshold |
|---|---|---|---|---|
| 1 | Ledger integrity: sum of all balances per network equals zero, and recomputed balances equal displayed balances. | Automated invariant check on every test run and on a scheduled check per instance. | VIS-001 | 100% of checks pass (fixed by the invariant). |
| 2 | Credit-limit enforcement under concurrent requests. | Concurrency test suite. | VIS-003 | Zero limit violations (fixed by the invariant). |
| 3 | Audit coverage: state-changing actions with exactly one audit event. | Sampled comparison of actions to audit events. | VIS-004, VIS-008 | 100% of sampled actions. |
| 4 | Trade completion: share of proposed transactions that reach Completed rather than Declined, Cancelled, or Expired. | Counts by transaction status per network. | VIS-002, VIS-005 | Threshold needs product decision. |
| 5 | Time from proposal to acceptance. | Timestamps on proposal and acceptance. | VIS-002 | Threshold needs product decision. |
| 6 | Dispute resolution: share of disputes with a recorded resolution, and time to resolution. | Dispute status and timestamps. | VIS-007 | Every dispute resolved or open with a named reviewer; time threshold needs product decision. |
| 7 | Balance comprehension: share of test participants who correctly interpret a sample balance, available credit, and a pending transaction. | Moderated usability check. | VIS-006 | Threshold needs product decision. |
| 8 | Accessibility: MVP journeys completable by keyboard only and at 320 px width. | Automated and manual accessibility checks. | VIS-011 | All MVP journeys; conformance level needs product decision. |
| 9 | Self-hosting effort: time for a person following only published instructions to reach a working instance with one network. | Timed walkthrough. | VIS-010 | Threshold needs product decision. |
| 10 | Export fidelity: balances recomputed from an exported file equal the displayed balances. | Automated export round-trip check. | VIS-009 | 100% of tested networks. |
| 11 | Active use: share of members in a network who take part in at least one transaction or listing in a period. | Activity counts per network, aggregated without member identifiers. | VIS-005 | Threshold and period need product decision. |

Measures 4, 5, 7, and 11 describe network health, not product correctness; a low value signals a question for the network, not necessarily a defect.

## 8. MVP boundaries

The MVP is the smallest product that lets a community run a mutual-credit network correctly. The capability list below is the scope; requirement identifiers, priorities, and acceptance criteria are in [04-functional-requirements.md](04-functional-requirements.md), and dependency order is in [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md).

### 8.1 In scope for the MVP

| # | Capability | Goal links |
|---|---|---|
| 1 | **Network setup.** Create a network, name its credit unit and decimal places, set initial policy, and create the network account. | VIS-004, VIS-010 |
| 2 | **Membership.** Invite, admit, suspend, and remove members; assign roles; create one member account per active membership. | VIS-004, VIS-012 |
| 3 | **Member directory and listings.** Publish, edit, withdraw, and browse offers and requests within a network. | VIS-005 |
| 4 | **Transaction proposal and acceptance.** Propose a transfer, accept, decline, or cancel it, and complete it with atomic credit-limit checks. | VIS-002, VIS-003 |
| 5 | **Balance and history.** Show balance, available credit, and transaction history with unmistakable direction. | VIS-001, VIS-006 |
| 6 | **Administrator controls.** Configure policy, post adjustments with a recorded reason, reverse transactions, and manage members. | VIS-004, VIS-023 |
| 7 | **Disputes.** Raise a dispute on a completed transaction or adjustment, review it, and record a resolution. | VIS-007 |
| 8 | **Audit history.** Append-only record of state-changing actions, viewable by authorized roles. | VIS-008 |
| 9 | **Basic exports.** Export members, transactions, postings, and audit history in documented non-proprietary formats. | VIS-009 |

Supporting foundations the MVP also needs, owned by other documents: email-based invitation and recovery ([DEC-007](13-assumptions-risks-decisions-and-roadmap.md#dec-007)), a responsive web client over the public API ([DEC-006](13-assumptions-risks-decisions-and-roadmap.md#dec-006)), and self-hostable deployment of a modular monolith ([DEC-001](13-assumptions-risks-decisions-and-roadmap.md#dec-001), [DEC-010](13-assumptions-risks-decisions-and-roadmap.md#dec-010)).

### 8.2 Explicitly deferred

Deferred items are priority P2: named to fix the boundary, not designed in this pass, and not to be built into the MVP in anticipation. Each is recorded in the roadmap in [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md).

| Deferred item | Boundary note |
|---|---|
| Mobile apps | The MVP client is a responsive web application. Native clients would use the same public API. |
| Federation between networks | No cross-network transactions, shared accounts, or shared identity. Each network's ledger is isolated. |
| Tokenization and blockchain | No token issuance and no distributed ledger ([VIS-016](#vis-016)). |
| Payment and currency conversion | No exchange with national currency and no payment-provider connection ([VIS-013](#vis-013)). |
| Reputation scoring | No computed trust or rating of members. |
| Advanced analytics | Only the basic counts needed for [section 7](#7-success-measures); no dashboards beyond them. |
| Third-party integrations | No integration-specific contracts. The public API is the extension point. |
| Paid hosting and monetization features | No subscription, billing, or paid administration features. A hosted offering, if any, runs the same software ([DEC-001](13-assumptions-risks-decisions-and-roadmap.md#dec-001)). |

### 8.3 Boundary rules

1. A capability not listed in 8.1 is out of the MVP unless a decision-log entry adds it.
2. A deferred item MUST NOT constrain the MVP design except where the roadmap records a deliberate extension point.
3. A change to 8.1 or 8.2 requires a matching update to [04-functional-requirements.md](04-functional-requirements.md) and the decision log in the same change.

## 9. Open questions

Each question is mirrored as a `DEC-` entry in [13-assumptions-risks-decisions-and-roadmap.md](13-assumptions-risks-decisions-and-roadmap.md) (see [conventions](00-conventions.md#5-document-structure)).

| ID | Question | Decision needed | Decided by |
|---|---|---|---|
| OQ-VIS-1 | Counsel question: does operating a network on Circula, or offering a hosted instance, raise money-services, securities, tax, or consumer-protection questions in the jurisdictions where early networks will run? Product behavior to review: members record credit balances, negative balances are permitted within limits, and administrators can write off balances. | Which jurisdictions apply and what review is required before any launch. Does not block technology-neutral requirements. | Project owner with qualified counsel |
| OQ-VIS-2 | What range of network sizes (members, transactions per day) must the MVP support? | Target figures that feed architecture and test thresholds. Recommended default: record a target range before architecture is accepted. | Project owner |
| OQ-VIS-3 | Who are the first real networks, and can representatives take part in the usability checks behind measures 7 and 9? | Named pilot communities or an agreed substitute. | Project owner |
| OQ-VIS-4 | What numeric thresholds apply to measures 4 to 9 and 11? | A threshold and measurement period for each, or a decision to track them without a threshold. | Product owner |
| OQ-VIS-5 | Is "Circula" the final product name, and which open-source license applies? | Name and license, recorded before the repository is made public. A license choice may carry counsel questions. | Project owner |
| OQ-VIS-6 | Is the language "credit unit", used throughout, acceptable to the pilot communities, or do networks need a different label in the interface? | Whether the per-network unit name is sufficient or the product needs further wording controls. | Product owner |

## 10. Change history

| Date | Change |
|---|---|
| 2026-11-11 | Initial draft. |
