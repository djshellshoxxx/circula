# System architecture and trust boundaries

**Status:** Draft recommendation  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Logical architecture and deployment boundaries; no vendor selection  
**Dependencies:** [Product scope](01-product-vision-scope.md), [domain model](03-domain-model.md)  
**Open questions:** Hosted versus self-hosted MVP; isolation topology; service-level objectives.

## Logical components

1. **Client:** presents network-scoped views and submits user actions; it is untrusted for authorization and accounting.
2. **Application/API layer:** authenticates requests, authorizes network roles, validates commands, enforces idempotency, and coordinates state transitions.
3. **Ledger/domain service:** applies transaction rules and creates balanced postings atomically. It is the authoritative path for all balance-changing actions.
4. **Persistence layer:** stores network configuration, membership, listings, transaction lifecycle, ledger entries, and audit records.
5. **Operations plane:** handles deployment configuration, backups, monitoring, upgrades, and platform support separately from network governance.

Clients and any external integration are untrusted. Authentication does not imply authorization. A user may participate in multiple networks; all reads and writes MUST be scoped to an explicit network context. The application boundary MUST prevent one network’s data from being returned or modified in another network’s request.

## Data flow

A client submits a command with authenticated identity, explicit network, and idempotency key where the operation can create state. The API validates syntax and role scope, the domain service checks current network policy and account state, and a transaction commits all required ledger entries and corresponding audit data. The response identifies the resulting state and stable resource reference. Read models may be optimized, but MUST be reconcilable to the ledger.

## Deployment recommendation

Recommend a modular application with support for self-hosting as an architectural quality, while initially validating one reference deployment model. Whether the first release is hosted, self-hosted, or both is an unresolved product decision. Do not make the platform operator a default network governor. For self-hosting, the operator of an instance can administer the software but has no product-level license to alter a network’s ledger without an attributable authorized event.

## Architecture requirements

- ARC-001: All balance-changing paths MUST pass through the same invariant-enforcing domain boundary.
- ARC-002: The system MUST isolate network data at every read and write boundary and verify scope server-side.
- ARC-003: Ledger posting and its audit event MUST commit or fail as one logical operation.
- ARC-004: Components SHOULD be modular enough that storage and deployment choices can change without changing mutual-credit semantics.
- ARC-005: The architecture MUST support export of a network’s data without dependence on a proprietary hosted-only interface.
- ARC-006: Failure responses MUST make uncertain commit outcomes detectable and safely resolvable by idempotent retry or status lookup.

## Quality attributes

Correctness and traceability take priority for posting. Availability, accessibility, maintainability, portability, and exportability targets must be quantified after pilot deployment assumptions are selected; see [quality strategy](12-quality-strategy.md).
