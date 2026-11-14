# Product Vision and Scope
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Problem, users, goals, non-goals, principles, success measures and MVP boundaries for Circula | Deps: [_context.md](./_context.md) | Open questions: 6, see "Open decisions"

## Purpose

This spec states why Circula exists, who it serves, what it must and must not do, and where the first delivery stops. Shared terms (Network, Account, Balance, Offer, Request, Transaction, and others), normative keywords, and ledger invariants are defined once in [_context.md](./_context.md) and are used here without redefinition. Normative keywords MUST, SHOULD and MAY have the meanings given there.

## Problem

Groups of people who exchange goods and services often have no shared, auditable way to record value provided and value received. Informal tallies are hard to verify, disagree between participants, and do not survive changes in membership. Circula provides a software system that a community can run to record such exchanges as balanced ledger entries under rules that the community itself sets.

## Product statement

- **PRD-001** (MUST) Circula MUST be described, in all product-facing and public text, as software for operating a Network. Circula MUST NOT be described as a currency, a payment system, an investment, or a service that guarantees any value or acceptance.
  - Rationale: Prevents the software from being read as a promise about credits that the software cannot control.
  - Verify: Review all product copy, onboarding text and public documentation. Each statement about credits matches the "Credit semantics" section of [_context.md](./_context.md). Zero statements claim monetary value, redeemability, or guaranteed acceptance.

- **PRD-002** (MUST) Circula MUST preserve every platform invariant listed in [_context.md](./_context.md) (ledger invariants 1 to 7) for every feature in every delivery. Network policy MUST NOT override them.
  - Rationale: Balance integrity is the basis for trust in any Network.
  - Verify: Each feature has an acceptance test that checks the affected invariants. Any feature that fails an invariant test blocks its release.

- **PRD-003** (MUST) Each Network MUST be able to set its own values for the configurable items named in [_context.md](./_context.md) (participation, credit limit, dispute windows, visibility defaults, join rules and governance). The platform MUST NOT impose a single global value for any of them.
  - Rationale: Networks differ in size, purpose and risk tolerance.
  - Verify: Configure two Networks with different values for each item. Each Network reads back only its own values, and no operation in one Network changes the other.

## Intended users

Roles are defined in [_context.md](./_context.md). Detailed journeys are out of scope for this spec.

- **PRD-004** (MUST) The product MUST support four roles: Platform operator, Network administrator, Dispute reviewer and Member. Each role MUST have a distinct permission set, which MUST be defined in the roles and journeys spec (ROL prefix).
  - Rationale: Separation of duties keeps policy, dispute decisions and daily use apart.
  - Verify: For each role, a test lists permitted and forbidden actions. A Member cannot perform Network administrator or Dispute reviewer actions, and the reverse holds too.

- **PRD-005** (SHOULD) The primary user is a Member who wants to find goods or services from other Members and to record what they provide. Secondary users are Network administrators who set up and run a Network.
  - Rationale: Member use drives the value of a Network; administrator use decides whether a Network can start.
  - Verify: Usability tests in the UX spec (UX prefix) include a Member task and an administrator setup task. Both are completed in a usability session with a recorded success rate target (see PRD-902).

## Goals

- **PRD-006** (MUST) A Network administrator MUST be able to create a Network, set its policy values from PRD-003, and invite or admit Members, without a Platform operator changing code or data by hand.
  - Verify: A scripted run creates a Network, sets all configurable items, admits a Member, and completes one Transaction, with no direct database or code change.

- **PRD-007** (MUST) A Member MUST be able to record an Offer and a Request, and see Offers and Requests from other Members of the same Network.
  - Verify: Two Member accounts in one Network create an Offer and a Request. Each sees the other's listing. A Member in a second Network does not see them (subject to PRD-904).

- **PRD-008** (MUST) Two Members MUST be able to agree on a Transaction, complete it, and see both Balances change by equal and opposite amounts.
  - Verify: Complete a Transaction of value V. The provider's Balance rises by V and the receiver's Balance falls by V. The Network sum of Balances stays at zero.

- **PRD-009** (MUST) Any Member MUST be able to view their Balance and the list of posted entries that produced it.
  - Verify: The sum of listed entries equals the displayed Balance for every Account in the test set.

- **PRD-010** (MUST) A participant MUST be able to raise a Dispute on a Transaction, and a Dispute reviewer MUST be able to record a resolution. Any resolution that changes Balances MUST do so through an Adjustment, not an edit.
  - Verify: Raise a Dispute, resolve it, and confirm that the history shows the original entries unchanged, plus new Adjustment entries that reference them.

## Non-goals

These items are excluded from this specification pass. They MUST NOT be built as part of the MVP.

- **PRD-011** (MUST NOT) The MVP MUST NOT include payments, cash-out, conversion to or from national currency, or any cryptocurrency or token feature.
  - Verify: Feature inventory for the MVP contains none of these. Code and configuration search finds no payment or token integration.

- **PRD-012** (MUST NOT) The MVP MUST NOT include paid monetization, hosted subscriptions, or paid administration features.
  - Rationale: Monetization is outside the scope of this specification pass.
  - Verify: Feature inventory review, as in PRD-011.

- **PRD-013** (MUST NOT) This specification MUST NOT make legal or compliance conclusions. Any such question MUST be recorded as an open decision (RSK prefix).
  - Verify: Review of this spec and of any spec derived from it finds no statement of legal status, regulatory approval or compliance outcome outside the open decisions list.

## Guiding principles

- **PRD-014** (SHOULD) Ledger history is never rewritten. Corrections are shown as new, referenced entries.
  - Rationale: Members need to trust that history cannot be changed quietly.
  - Verify: The ledger contains no update or delete path for posted entries, and every correction shows a link to its original.

- **PRD-015** (SHOULD) Each Network's rules are visible to its Members in plain language before they join.
  - Rationale: Members choose whether to join based on the rules.
  - Verify: Before admission, a prospective Member can read every configured policy value for the Network they are joining.

- **PRD-016** (SHOULD) The platform collects only the data needed for Balance, Transaction, Dispute and audit functions.
  - Rationale: Less data reduces risk for Members and Networks. Detailed privacy rules belong to the security and privacy spec (SEC prefix).
  - Verify: A data inventory lists each stored field with its function. Each field maps to a requirement in an authoritative spec.

- **PRD-017** (SHOULD) Each Network operates independently. A decision in one Network MUST NOT alter another Network's Balances, rules or listings.
  - Verify: Run identical operations in two Networks and confirm that changes in one do not appear in the other.

## Success measures

Targets are not set in this spec. Each measure MUST have a recorded definition, data source and target before the MVP release (see PRD-902).

- **PRD-018** (MUST) The share of completed Transactions whose entries fail the balance check MUST be zero.
  - Verify: Daily report over all Networks. Any non-zero value is a defect.

- **PRD-019** (SHOULD) Track time from Network creation to first completed Transaction.
  - Verify: The metric is computed from audit events for each Network with at least one completed Transaction.

- **PRD-020** (SHOULD) Track the share of Transactions that receive a Dispute.
  - Verify: The metric uses Dispute records divided by completed Transactions, per Network and in total.

- **PRD-021** (SHOULD) Track the number of Networks with at least a set number of active Members, where "active" is set by the product owner when targets are recorded (PRD-902).
  - Verify: The definition of "active" and the threshold are recorded before this metric is reported.

## MVP boundaries

The MVP is the smallest delivery that satisfies PRD-002 and PRD-006 to PRD-010 for at least one Network.

- **PRD-022** (MUST) The MVP MUST include: Network creation and policy settings; Account admission; Offer and Request listings; Transaction proposal, completion and Balance view; ledger entry history; Dispute raise and resolution; and audit events for each security- or accounting-relevant action.
  - Verify: Each listed capability has at least one passing end-to-end test.

- **PRD-023** (MUST) The MVP MUST run one Network end to end. Support for several Networks in one installation is not required (see PRD-901).
  - Verify: The end-to-end test in PRD-022 passes on a fresh install with one Network.

- **PRD-024** (MAY) The MVP MAY include automated matching of Offers and Requests.
  - Verify: If included, matching produces suggestions only. It MUST NOT create a Transaction without both participants' agreement.

- **PRD-025** (MUST NOT) The MVP MUST NOT allow Transactions or Balance-visible listings across Networks.
  - Verify: Cross-network query tests return no records.

## Open decisions

Each item gives options and a recommended default. The default applies until a decision is recorded in this spec.

- **PRD-901** Tenancy: how many Networks one installation may host.
  - Options: (a) one Network per installation; (b) many Networks per installation with strict data separation.
  - Default: (a) for the MVP. Option (b) is reconsidered after the MVP.

- **PRD-902** Success measure targets and data sources (PRD-018 to PRD-021).
  - Options: (a) set targets from pilot data; (b) set provisional targets now and review after the first pilot.
  - Default: (b). Targets are recorded in the metrics spec before any external release.

- **PRD-903** Account membership: whether one person may hold Accounts in several Networks.
  - Options: (a) one Account per person per Network; (b) one person identity across Networks, with separate Accounts.
  - Default: (a). Option (b) affects the admission and identity model and needs its own decision.

- **PRD-904** Visibility of listings across Networks (PRD-007).
  - Options: (a) listings visible only inside their Network; (b) listings optionally shared to other Networks by policy.
  - Default: (a). Option (b) would require a cross-network rule set and is not in the MVP (PRD-025).

- **PRD-905** Identity verification before admission. Any question about legal or regulatory duties here is out of scope under PRD-013.
  - Options: (a) none at platform level, with Network policy deciding; (b) a platform-defined minimum that every Network must meet; (c) a Network-defined check run by a named external party.
  - Default: (a). The choice is a product and governance decision, and any legal review is separate.

- **PRD-906** Client scope for the MVP: which devices and platforms must be supported.
  - Options: (a) a responsive web application only; (b) web plus native mobile applications.
  - Default: (a). Platform choices are not made in this spec.

## Related docs

- [_context.md](./_context.md): shared terms, normative keywords, credit semantics, ledger invariants and the spec header template.
- Foundational specs with prefixes ROL, DOM, FR, ARC, DAT, API, UX, SEC, GOV, OPS, QA and RSK are in this directory (see [README.md](./README.md)). They link to this spec and MUST NOT redefine its requirements.
