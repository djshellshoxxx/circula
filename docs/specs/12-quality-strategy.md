# Quality strategy and test plan

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Verification expectations for foundational behavior  
**Dependencies:** [Functional requirements](04-functional-requirements.md), [data and ledger](06-data-ledger-audit.md), [UX](08-ux-accessibility.md), [operations](11-deployment-operations.md)  
**Open questions:** Numeric availability, performance, recovery, and accessibility gates.

## Verification layers

1. **Domain/unit tests:** signed-balance arithmetic, limit checks, lifecycle transitions, policy evaluation, reversals.
2. **Persistence/integration tests:** atomic paired posting, uniqueness, rollback, idempotency, migration and reconciliation.
3. **Authorization tests:** role matrix, network isolation, reviewer scope, export permissions, account recovery.
4. **API contract tests:** error shape, pagination stability, versioning, conflict behavior, safe retry.
5. **Accessibility tests:** automated semantic checks plus keyboard and screen-reader review of member, balance, transaction, and dispute journeys.
6. **Operational tests:** backup restoration, migration recovery, monitoring alerts, export/import reconciliation.
7. **End-to-end tests:** scenario flows below using realistic concurrent and failure conditions.

## Quality requirements

- QA-001: Every ledger invariant in DATA-001 through DATA-007 MUST have a direct automated test.
- QA-002: Concurrent acceptance of the same proposal MUST result in at most one posting set.
- QA-003: Concurrent proposals MUST not permit a posted debit beyond the configured limit; behavior for pending reservation must match documented network policy.
- QA-004: Repeating a mutation with the same key and payload MUST return the same outcome; a changed payload with the same key MUST fail without additional effects.
- QA-005: A restored backup MUST reconcile account balances to ledger entries and retain links to corrections.
- QA-006: Exported records MUST reconcile to source records and MUST respect network and role scope.
- QA-007: Authorization tests MUST include attempts to access another network by changing identifiers or pagination tokens.
- QA-008: Accessibility checks MUST include keyboard-only completion of proposal, acceptance, balance review, and dispute creation.

## End-to-end acceptance scenarios

- A member proposes a trade, the counterparty accepts, both balances change by equal and opposite amounts, and both histories display the same transaction reference.
- Two concurrent accepts arrive for one proposal; one completes, the other receives the existing outcome or a conflict, and no duplicate balance movement occurs.
- A trade that would exceed the accepter’s negative-balance limit is rejected with no ledger entries.
- A member disputes a completed trade; the reviewer records an outcome and an authorized reversal or adjustment is linked to the original.
- An authorized administrator exports a network, restores it in a test environment, and reconciles all balances; an unauthorized member cannot export restricted records.

Set numeric targets only after target deployment and pilot volumes are known. Do not mark a build production-ready based solely on unit-test success.
