# Architecture and API

**Status: Draft framework; technology choices are not finalized.**

The proposed logical system includes a client, an authenticated API/application layer, an invariant-enforcing domain/ledger boundary, persistence, and a separately controlled operations plane.

Every balance-changing path must enforce the same rules regardless of whether it originates in the user interface, API, or future integration. Requests need explicit network scope, server-side authorization, idempotency for retryable mutations, and consistent error behavior.

## Future sections

- Component responsibilities and data flows
- Trust boundaries and network isolation
- Conceptual MVP operations
- Authentication and authorization
- Error format, pagination, versioning, and compatibility
- Integration and extension guidance

See the [architecture](../docs/specs/05-architecture.md) and [API contract](../docs/specs/07-api-contract.md).
