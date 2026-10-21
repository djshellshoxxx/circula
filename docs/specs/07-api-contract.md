# API and integration contract

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Protocol-independent MVP contract  
**Dependencies:** [Domain model](03-domain-model.md), [data and ledger](06-data-ledger-audit.md), [security](09-security-privacy.md)  
**Open questions:** API protocol, authentication provider, and public versus private endpoint scope.

## Principles

The user interface and API are peers at the trust boundary: neither is trusted to enforce ledger rules alone. The server MUST authenticate callers, authorize every action against network scope, validate state transitions, apply current policy, and return a stable outcome. A client MUST NOT choose authoritative actor identity or resulting balances.

## Conceptual operations

- Read network metadata and effective policy.
- Request, approve, suspend, reinstate, or leave membership.
- List and manage offers/requests within visibility scope.
- Create, inspect, accept, decline, cancel, and dispute a transaction.
- Read account balance and paginated ledger history.
- Record an authorized adjustment or reversal.
- Read and export permitted audit and network records.

These operations describe capability, not a commitment to REST, GraphQL, or a specific framework.

## Contract requirements

- API-001: Every request MUST carry or resolve an explicit network context; server-side authorization MUST verify membership and role in that network.
- API-002: Balance-changing operations MUST use the canonical domain rules in the [domain model](03-domain-model.md), including current credit-limit checks.
- API-003: Retryable mutations MUST accept an idempotency key and return the original outcome for an identical retry; same key with different payload MUST return a conflict.
- API-004: Errors MUST use a consistent machine-readable code and human-readable message, with field details when applicable. They MUST state whether the mutation was committed, rejected, or has an unknown outcome.
- API-005: List operations MUST support stable pagination and explicit sorting; page tokens MUST not expose data across network scopes.
- API-006: API versions and compatibility behavior MUST be documented before external clients are supported. Breaking changes MUST be versioned or preceded by a documented migration period.
- API-007: Responses MUST distinguish proposed, pending, posted, disputed, reversed, and declined state without presenting proposals as completed balance changes.
- API-008: Exports MUST be permission-scoped and auditable; sensitive fields MUST be excluded unless necessary and authorized.
- API-009: Authentication credentials and secrets MUST never be returned in ordinary responses or written to logs.

## Error behavior

Validation errors identify the field and correction when safe. Authorization failures do not disclose whether an inaccessible resource exists. Stale state or insufficient credit returns the current actionable reason without posting. An uncertain network failure can be resolved using an idempotency key or operation-status lookup; retrying MUST NOT duplicate effects.
