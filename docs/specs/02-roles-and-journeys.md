# Roles and core journeys

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** MVP actors, permissions, and journey outcomes  
**Dependencies:** [Product scope](01-product-vision-scope.md), [domain model](03-domain-model.md)  
**Open questions:** Whether a network may assign multiple dispute reviewers to a case.

## Roles

| Role | Typical capabilities | Limits |
|---|---|---|
| Visitor/applicant | View public network information; submit a membership request where enabled | No account balance or private member data |
| Member | Maintain permitted profile fields; manage own offers/requests; propose, accept, or dispute eligible transactions; view own balance and history | Cannot alter another member’s balance or network policy |
| Network administrator | Configure network policy; manage membership and listings; appoint roles; review operational records; initiate authorized adjustments | Every privileged action is attributable and auditable; cannot erase ledger history |
| Dispute reviewer | Read evidence needed for assigned disputes; record findings and permitted resolution | Must not decide a case where policy requires recusal; scope is limited to assigned cases |
| Platform operator | Maintain platform service, deployments, and support tooling | Platform access does not confer authority to govern a network or alter its ledger |

Role combinations and separation-of-duty rules are network policy, subject to platform-wide audit controls. Access is denied by default when a role or scope is missing.

## Core journeys

1. **Join a network:** applicant reads network terms and unit explanation, submits requested information, and receives a pending, approved, or declined outcome. The interface explains next steps without exposing other applicants.
2. **Offer or request value:** member creates, edits, pauses, or closes a listing; visibility and contact options are shown before publication.
3. **Propose and accept a trade:** proposer selects the counterparty, amount, description, and any required terms. The counterparty reviews both balance impacts and accepts or declines. Acceptance is revalidated against current authorization and credit limits before posting.
4. **Review account activity:** member sees current balance, configured limit, available credit, pending items, completed entries, and linked corrections in chronological history.
5. **Raise and resolve a dispute:** member submits a reason and evidence; the case is acknowledged, assigned, updated, decided under network policy, and retained with its outcome.
6. **Administer network rules:** administrator reviews the impact, announces a proposed change, applies it through a permission-checked action, and leaves a timestamped record of who changed what and when.

## Journey requirements

- ROLE-001: The system MUST show the user’s effective role and network scope before presenting privileged actions.
- ROLE-002: The system MUST return an understandable access-denied state when an action is unauthorized and MUST NOT partially apply that action.
- ROLE-003: A member MUST be able to distinguish pending, declined, completed, disputed, reversed, and adjusted activity.
- ROLE-004: Core journeys MUST be operable by keyboard and provide clear validation and recovery guidance.
- ROLE-005: Administrative and reviewer actions MUST record actor, target, time, action, and outcome in the audit record, subject to privacy minimization.

## Error and accessibility states

For network unavailable, stale data, expired session, validation failure, insufficient credit, and denied access, the interface MUST identify whether any action was committed. Retry guidance MUST not encourage duplicate transactions; transaction retries use the idempotency behavior in the [API contract](07-api-contract.md).
