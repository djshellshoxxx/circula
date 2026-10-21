# Domain model and mutual-credit rules

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Shared MVP accounting vocabulary and transaction semantics  
**Dependencies:** [Product scope](01-product-vision-scope.md)  
**Open questions:** Policy for third-party transactions, negative-balance recovery, and exact treatment of pending reservations.

## Terms

- **Network:** a separately governed group using one accounting unit and policy set.
- **Member:** a person or organization admitted to a network.
- **Account:** a member’s ledger position within one network. Membership and account status are related but distinct.
- **Credit unit:** an accounting unit defined by a network. It is not presumed to be a cryptocurrency or redeemable currency.
- **Balance:** signed net of posted ledger entries for an account. Positive means net value provided; negative means net value received.
- **Credit limit:** network policy cap on an account’s permitted negative balance.
- **Offer/request:** a member-authored listing of value available or sought; it is not a transaction commitment.
- **Transaction:** a proposed exchange between accounts with a positive amount and description.
- **Dispute:** a case challenging a transaction or its fulfillment.
- **Adjustment:** an authorized, reasoned accounting event correcting or applying a network decision.
- **Audit event:** an attributable record of a relevant action or state transition.

## Platform invariants and configurable policy

Platform-wide invariants MUST preserve ledger integrity, authorization, auditability, and tenant isolation. Network administrators MAY configure member eligibility, account limits, acceptance requirements, dispute procedures, and listing rules. A network may not configure a policy that permits unbalanced completed trades, silent ledger edits, cross-network posting, or unauthorized access.

## Transaction lifecycle

A transaction moves through proposed, accepted, completed/posted, declined, cancelled, disputed, reversed, or otherwise resolved states. Exact state labels may be refined by implementation, but state transitions MUST be explicit and auditable.

- A proposal does not change posted balances.
- Acceptance is an explicit, attributable action by the required parties.
- Before posting, the system MUST recheck member status, permissions, transaction amount, and applicable credit limits using current state.
- A completed trade posts equal and opposite entries atomically: for amount A, one account changes by +A and the other by -A. Network-wide sum of trade postings is zero.
- A failed or declined proposal creates no balance movement.
- A correction MUST create a linked reversal or adjustment with reason, actor, timestamp, and reference to the original; it MUST NOT erase or rewrite the original posting.
- Retries MUST not create duplicate postings. See [ledger requirements](06-data-ledger-audit.md) and [API contract](07-api-contract.md).

## Negative balances, departure, and default

A debit MUST NOT cause an account to exceed its active negative-balance limit unless an explicit, auditable policy exception is authorized. A pending transaction should not reserve credit in the MVP unless the network policy explicitly selects that behavior; this remains an open decision because concurrent proposals can otherwise compete for the same headroom. Membership departure MUST preserve historical records and MUST NOT silently forgive or transfer a balance. Recovery, suspension, write-off, and closure procedures are governance decisions, not presumed guarantees of collection.

## Rule ownership

Balance arithmetic, equal-and-opposite posting, transaction uniqueness, tenant isolation, and immutable correction history are platform rules. Admission, standard credit limits, proposal expiry, dispute deadlines, and local review authority are network policies, enforced only where configured.
