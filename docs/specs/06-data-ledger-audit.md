# Data model, ledger, and audit requirements

**Status:** Draft  
**Owner:** Circula project owner  
**Last updated:** 2026-10-20  
**Scope:** Conceptual entities, ledger integrity, auditability, retention principles  
**Dependencies:** [Domain model](03-domain-model.md), [security](09-security-privacy.md)  
**Open questions:** Retention periods, deletion/anonymization rules, and treatment of backups after member erasure requests.

## Conceptual entities

- Network: stable identifier, display name, unit label, status, policy version, timestamps.
- Member: stable identifier, network membership status, profile fields chosen by the network, timestamps.
- Account: stable identifier, network and member references, active credit-limit policy, computed or materialized balance.
- Listing: stable identifier, owner, type (offer/request), content, visibility, status, timestamps.
- Transaction: stable identifier, network, proposer, counterparty, amount, description, lifecycle state, idempotency reference, timestamps.
- Ledger entry: stable identifier, account, transaction or adjustment reference, signed amount, effective time, creation time.
- Dispute: stable identifier, challenged transaction, submitter, status, restricted evidence reference, decision history.
- Audit event: stable identifier, actor or system principal, action, target, network, event time, outcome, correlation reference, and minimal before/after context.
- Policy version: stable identifier, network, effective interval, changed fields, approver, and notice reference.

Identifiers MUST be unique within a documented scope and stable across exports. Personal data should be separate from accounting history where practical so identity fields can be minimized without destroying ledger reconciliation.

## Ledger requirements

- DATA-001: The system MUST create both equal-and-opposite trade entries atomically with a unique transaction reference.
- DATA-002: A committed trade MUST have exactly one balanced posting set; for each trade, the sum of its entries is zero.
- DATA-003: The balance MUST be reconstructable as the sum of account entries, and any cached balance MUST be reconcilable.
- DATA-004: The system MUST enforce an idempotency key for retryable state-changing requests, scoped to actor, network, and operation; reuse with a different payload MUST be rejected.
- DATA-005: The system MUST NOT update or delete a posted entry to correct history; it MUST append a linked reversal or adjustment with reason and authorized actor.
- DATA-006: A transaction and associated audit data MUST commit together or be recoverable as an unambiguous operation.
- DATA-007: Duplicate proposals or concurrent acceptance requests MUST not produce duplicate postings.

## Audit and retention

Audit records MUST capture who or what acted, on which network and target, when, what action was attempted, and whether it succeeded. They SHOULD include changed fields rather than full personal-data snapshots. Evidence and sensitive values MUST not be copied into general logs. Access to audit history is role-scoped and itself logged when sensitive records are read or exported.

Retention periods, deletion obligations, and backup expiry must be set before production based on jurisdiction, network policy, and counsel where relevant. The data model MUST make those policies implementable; no specific period is asserted here. Migration MUST preserve IDs, signed amounts, ordering or effective timestamps, idempotency history where needed, and links among original entries and corrections. Every migration MUST have pre/post reconciliation checks and a rollback or recovery plan.
