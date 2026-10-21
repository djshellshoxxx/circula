# Product vision and scope

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Product intent and initial release boundary  
**Dependencies:** None  
**Open questions:** Hosting model, initial target network profile, and measurable launch thresholds; see [decisions](13-decisions-risks-roadmap.md).

## Purpose

Circula is software for operating community mutual-credit networks. It helps members discover offers and requests, agree on exchanges, and keep a shared record of credits created and used within their network. A network’s accounting unit records obligations within that network. Circula does not guarantee that credits have monetary value, that members will accept them, or that any balance is redeemable for national currency.

## Problem and intended users

Small communities often coordinate exchanges through informal spreadsheets, messages, or paper records. These methods can make balances difficult to reconcile, transactions hard to trace, and governance unclear. Intended users include local exchange networks, cooperatives, community groups, their members, and administrators.

## Goals

- Make balances and the transactions that explain them understandable.
- Enforce consistent, reviewable accounting rules across user and API operations.
- Give each network transparent controls for participation, credit limits, disputes, and governance.
- Support member discovery of goods and services without implying guaranteed availability.
- Let a network export its records in a usable format.
- Keep the initial design portable and avoid unnecessary personal-data collection.

## Non-goals for the MVP

Circula does not provide national-currency payment processing, credit redemption, investment products, a cryptocurrency or token, automated credit scoring, legal or tax advice, escrow, or a guarantee that a member will perform. Mobile applications, federation, advanced analytics, and third-party integrations are deferred unless later approved as separate specifications.

## Guiding principles

1. **Accounting clarity:** every completed trade has equal-and-opposite postings.
2. **Traceability:** corrections are linked, attributable events; history is not silently rewritten.
3. **Local governance:** networks configure policy within platform-wide accounting and security invariants.
4. **Member understanding:** show credit direction, available credit, and transaction state plainly.
5. **Data control:** collect only data needed to operate the network and support member rights.
6. **Portability:** export and migration are first-class operational needs.

## MVP boundary and success measures

MVP includes network setup, membership, member directory and listings, transaction proposal and acceptance, balances and history, administration, disputes, audit history, and basic exports. The initial success measures should track: successful onboarding, successful completion and reconciliation of transactions, dispute resolution completion, export/restore correctness, and accessibility defects. Numeric targets are open decisions pending a pilot profile.

## Normative language

**MUST** means required for conformance. **SHOULD** means recommended unless a documented reason justifies another choice. **MAY** means optional. These terms express product requirements, not legal conclusions.
