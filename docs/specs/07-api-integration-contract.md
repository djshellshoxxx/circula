# API and Integration Contract
Status: Draft | Owner: Product architect (role) | Updated: 2026-11-14 | Scope: Conceptual interface contract for MVP operations: design principles, access expectations, error, pagination, idempotency, versioning, and ledger enforcement at the interface | Deps: [_context.md](./_context.md), [01](./01-product-vision-scope.md), [02](./02-personas-roles-journeys.md), [03](./03-domain-model-rules.md), [13](./13-assumptions-risks-decisions-roadmap.md) | Open questions: 7, see "Open decisions"

## Purpose

This spec defines the contract that any programmatic interface to Circula MUST satisfy. It states what operations exist in the MVP, who may invoke them, what outcomes they must produce, and which rules they must enforce. It does not define endpoints, methods, payload encodings, or a framework. Those are implementation choices that MUST conform to this contract.

The user interface and any programmatic interface are two entry points to one set of domain rules. This spec exists so that neither entry point can weaken the rules owned by [03](./03-domain-model-rules.md) and the ledger invariants in [_context.md](./_context.md).

## Scope

In scope: interface design principles, authentication and authorization expectations, error reporting, pagination, idempotency, versioning and compatibility, the MVP operation set at a conceptual level, and the ledger rules every operation must enforce at the interface boundary.

Out of scope:
- Role definitions and journeys: [02](./02-personas-roles-journeys.md). This spec references roles and does not redefine them.
- Domain entities and mutual-credit rules: [03](./03-domain-model-rules.md). This spec references DOM IDs.
- Data storage, ledger schema, and audit record format: the DAT prefix (not yet written).
- Credential issuance, session lifetime, and transport security: the SEC prefix (not yet written).
- Interface rules for user-facing screens, error wording in the UI, and accessibility: the UX prefix (not yet written).
- Target latency, load, and test-suite definitions: the QA prefix (not yet written).

Normative terms (MUST, SHOULD, MAY) are defined in [_context.md](./_context.md) and are used here with that meaning.

## Requirements

### Design principles

**API-001** (MUST) Every operation MUST call the same domain rules that the user interface calls. An interface MUST NOT hold its own copy of posting, limit, standing, or dispute logic.
- Rationale: two implementations of one rule drift apart, and drift in posting logic breaks ledger invariants.
- Verify: a contract test runs one scenario set through the user interface path and the programmatic path. Resulting entries, balances, statuses, and audit events are identical.

**API-002** (MUST) Every operation MUST act within exactly one network. The network MUST be determined from the caller's authenticated scope and MUST NOT be inferred from request content alone. A request that references a record in another network MUST be refused.
- Rationale: [DOM-001](./03-domain-model-rules.md) and [PRD-025](./01-product-vision-scope.md).
- Verify: a transaction proposal naming accounts from two networks is refused, and no account, transaction, or entry record is written.

**API-003** (MUST) No operation MUST accept a caller-supplied balance value, a caller-supplied entry list for posting, or a caller-supplied audit event. Balances, entries, and audit events MUST be produced by the platform as a side effect of permitted operations.
- Rationale: [ROL-064](./02-personas-roles-journeys.md) and invariant 3 in [_context.md](./_context.md).
- Verify: schema review of every MVP operation confirms no such field. Fuzzing with extra balance or entry fields yields refusal or those fields are ignored, and no record changes outside the operation's stated effect.

**API-004** (SHOULD) Read operations SHOULD have no side effects on records, counters shown to members, or balances.
- Verify: before and after any sequence of read calls, record counts and stored values are unchanged.

**API-005** (MUST) Operations MUST return data only to callers whose role in the network permits it. A member's response MUST NOT contain another member's balance or entries, except as [ROL-043](./02-personas-roles-journeys.md) allows for a counterpart's transaction amount and status.
- Verify: for each role and each read operation, an access test confirms out-of-scope data is absent from the response.

**API-006** (MUST) Timestamps on ledger entries, audit events, transaction state changes, and dispute records MUST be assigned by the server clock. Client-supplied timestamps MUST NOT be used for these records or for any window check.
- Rationale: [RSK-010](./13-assumptions-risks-decisions-roadmap.md) and [RSK-034](./13-assumptions-risks-decisions-roadmap.md).
- Verify: with a client clock skewed by a fixed offset, stored timestamps and dispute-window results match server time.

**API-007** (MUST) Monetary amounts MUST be exchanged as integers in the network's minor unit. Non-integer, floating-point, or unit-less amounts MUST be refused with a validation error.
- Rationale: [RSK-033](./13-assumptions-risks-decisions-roadmap.md) default (a). If that decision changes, this requirement MUST be revised.
- Verify: requests with fractional, exponent-form, or missing-unit amounts are refused, and no record is written.

### Authentication and authorization expectations

**API-010** (MUST) Every MVP operation MUST require an authenticated caller. No MVP operation is public. See open decision OD-07-6.
- Verify: an unauthenticated call to each operation in the MVP operations table is refused with an authentication outcome and returns no data.

**API-011** (MUST) Authorization MUST be checked for every operation against the caller's effective roles in the target network. Effective rights are the union of held roles, limited as stated in [ROL-003](./02-personas-roles-journeys.md) to [ROL-005](./02-personas-roles-journeys.md) and [ROL-021](./02-personas-roles-journeys.md).
- Verify: a role-by-operation matrix test covers every cell of the permissions matrix in [02](./02-personas-roles-journeys.md). Each cell's outcome matches the matrix.

**API-012** (MUST) An identity holding only the platform operator role MUST NOT be authorized for any operation that changes ledger state, policy, roles, or dispute outcomes in any network.
- Verify: for each mutating operation, a platform-operator-only identity receives a permission outcome and no record changes.

**API-013** (MUST) A permission outcome MUST NOT reveal whether a record exists in another network or belongs to another member. A request for a record the caller may not see MUST produce the same outcome shape and code as a request for a record that does not exist.
- Verify: test compares responses for an existing foreign record and a nonexistent identifier. Code and body structure are identical.

**API-014** (MUST) Each privileged operation (role grant or revocation, policy change, reviewer assignment, offer or request removal, dispute resolution, dispute-linked adjustment) MUST write its audit event in the same atomic unit as its effect. A success outcome MUST NOT be returned unless both are committed.
- Rationale: [ROL-006](./02-personas-roles-journeys.md), [ROL-061](./02-personas-roles-journeys.md), [ROL-021](./02-personas-roles-journeys.md).
- Verify: fault injection between effect and audit write yields no committed effect and an error outcome.

**API-015** (MUST) A credential that has been revoked, or whose role grant has been removed, MUST be refused on the next operation that requires it, subject to the bound in OD-07-7. Credential lifecycle rules are owned by the SEC prefix (not yet written).
- Verify: revoke-then-call test in each role.

### Error format

**API-020** (MUST) Every refusal MUST include: (a) a stable, machine-readable code from a published catalog; (b) a human-readable message that names the cause and the next available action, as [ROL-072](./02-personas-roles-journeys.md) requires; (c) a correlation identifier usable in support requests; and (d) for validation failures, the path of each offending field.
- Verify: a test iterates every code in the catalog and checks all four elements are present and non-empty.

**API-021** (MUST) The error catalog MUST distinguish at minimum: validation failure; authentication required; permission denied; not found; credit limit reached ([ROL-033](./02-personas-roles-journeys.md)); invalid state transition for the transaction or dispute lifecycle; dispute open, blocking completion ([ROL-052](./02-personas-roles-journeys.md)); idempotency key conflict (API-042); account not active ([DOM-003](./03-domain-model-rules.md)); and pending, not yet confirmed ([ROL-075](./02-personas-roles-journeys.md)).
- Verify: each condition triggers its own code in a test.

**API-022** (MUST) A refused mutating operation MUST leave no trace in ledger, transaction, dispute, or audit records. It MUST NOT create an entry, change a balance, or emit a success audit event.
- Verify: row counts and stored values across all postable tables are unchanged after each refusal path.

**API-023** (MUST) Error messages MUST NOT include stack traces, storage internals, identifiers of other networks, or any other member's private data.
- Verify: fault-injection review of responses for each error code.

**API-024** (SHOULD) For the same condition, the interface and the user interface SHOULD return the same error code. Wording MAY differ by channel.
- Verify: mapping table between UI states and API codes is reviewed for each release.

### Pagination

**API-030** (MUST) Every list operation whose result can grow (offers and requests, transactions, ledger entries, dispute lists, audit events visible to the caller) MUST paginate. A call without a page size MUST receive at most the platform maximum.
- Verify: a call over a collection larger than the maximum returns at most the maximum and a continuation indicator.

**API-031** (MUST) Pagination MUST use a documented, stable ordering and an opaque continuation token. Continuation tokens MUST NOT expose storage offsets or other internal positions.
- Verify: documented ordering matches observed order in tests. Tokens are not decodable into record keys by a caller.

**API-032** (MUST) Traversal of ledger history MUST be complete and non-duplicating while new entries are appended. Each entry that existed when traversal began MUST appear exactly once.
- Rationale: [PRD-014](./01-product-vision-scope.md) and ledger history is append-only.
- Verify: concurrent-append test during full traversal of an account's history.

**API-033** (SHOULD) The platform maximum page size and default page size SHOULD be fixed platform values, not network policy. Their values are set in the QA spec. See OD-07-4.

### Idempotency

**API-040** (MUST) Every operation that creates a transaction, changes transaction state, posts a completion, raises a dispute, submits a join request, or changes a role or policy value MUST accept a caller-supplied idempotency key.
- Verify: each such operation rejects a request without a key where the key is required by this requirement.

**API-041** (MUST) A retry with the same key and equivalent request content MUST return the result of the first execution and MUST NOT create a second entry, transaction, dispute, or audit event. This is invariant 5 in [_context.md](./_context.md).
- Verify: the same request sent N times (N at least 10, with concurrent sends) yields one set of records and identical responses.

**API-042** (MUST) A retry with the same key and different request content MUST be refused with the idempotency key conflict code (API-021) and MUST write nothing.
- Verify: keyed request, then altered request with same key, yields conflict and unchanged records.

**API-043** (MUST) Idempotency keys MUST be scoped to one network and one authenticated caller. The same key value used in another network or by another caller MUST NOT match.
- Verify: test with equal key values across two callers and two networks.

**API-044** (MUST) The idempotency record MUST be committed in the same atomic unit as the operation's effect. A retry after a crash following commit MUST return the original result.
- Rationale: [RSK-011](./13-assumptions-risks-decisions-roadmap.md), [RSK-005](./13-assumptions-risks-decisions-roadmap.md).
- Verify: fault injection after commit and before response; retry returns original result.

**API-045** (MUST) Completion of a given transaction MUST be unique per transaction regardless of how many distinct keys are submitted. A second completion of a completed transaction MUST be refused or return the original result, and MUST NOT post again.
- Rationale: [ROL-035](./02-personas-roles-journeys.md). This is a backstop for keys that differ between clients.
- Verify: two completion calls with different keys from two sessions yield one posting.

**API-046** (SHOULD) Read operations SHOULD NOT require idempotency keys.

### Versioning and compatibility

**API-050** (MUST) Every interface MUST carry an explicit version identifier visible to the caller. A breaking change MUST be published under a new version identifier. Breaking changes include removing or renaming a field or operation, changing the meaning of a field or code, tightening validation, and changing the ledger effect of an operation.
- Verify: a published contract diff between consecutive versions lists every change, and each breaking change is under a new identifier.

**API-051** (MUST) Within one version, changes MUST be additive only: new optional request fields, new response fields, and new error codes. Clients MUST be able to treat an unknown error code as a generic refusal with the same handling as the closest documented category.
- Verify: the previous version's client test suite passes unchanged against the new release.

**API-052** (MUST) The ledger effect of an operation MUST NOT change within a version. Ledger invariants apply to every version simultaneously ([_context.md](./_context.md), invariant 2 and invariant 4 for history).
- Verify: the operation table's ledger effects are asserted in each version's test suite.

**API-053** (MUST) A deprecated version or operation MUST remain functional for the deprecation window in OD-07-5 and MUST signal deprecation to callers in documentation and in responses.
- Verify: a deprecated operation returns a deprecation signal and succeeds for the full window.

**API-054** (MUST) Transaction representations MUST include the identifier of the network policy version in force when the transaction was created. Policy changes MUST NOT alter this value on existing transactions.
- Rationale: [RSK-017](./13-assumptions-risks-decisions-roadmap.md) and [ROL-062](./02-personas-roles-journeys.md).
- Verify: change a credit limit, then read an earlier transaction. Its recorded policy version is unchanged.

### Ledger invariants at the interface

These requirements restate how the invariants in [_context.md](./_context.md) apply at the interface. They do not replace the invariant definitions. The DAT prefix (not yet written) owns the storage mechanics.

**API-060** (MUST) Every posting MUST be atomic. All entries of a transaction post together or none posts (invariant 7).
- Verify: fault injection at each step of a posting yields all-or-nothing.

**API-061** (MUST) Every completed transaction MUST have entries that sum to zero across the participating accounts (invariant 1). A posting that would produce a non-zero sum MUST be refused and MUST NOT be partially written.
- Verify: property test over randomized transactions; every completed transaction sums to zero.

**API-062** (MUST) Entries for a posting MUST be derived by the platform from the agreed transaction terms. Callers MUST NOT choose entry accounts, signs, or amounts beyond the transaction terms.
- Verify: API-003 test plus review that completion carries no entry input.

**API-063** (MUST) No operation MUST update or delete a posted entry or an audit event (invariant 2). The MVP operation set MUST NOT include any such operation, and any attempt MUST be refused.
- Rationale: [RSK-004](./13-assumptions-risks-decisions-roadmap.md).
- Verify: catalog review plus negative tests for update and delete on each record type.

**API-064** (MUST) Corrections, reversals, and dispute effects MUST be new entries that reference the original entry (invariant 4). The original entry MUST remain readable.
- Verify: after a dispute adjustment, the original and adjustment are both returned by history reads, with a reference between them.

**API-065** (MUST) Every change to a balance MUST be produced by a posting that references an audit event (invariant 3).
- Verify: for each posting in a randomized sequence, a linked audit event exists.

**API-066** (MUST) Every posting that debits an account MUST check the account's credit limit (see [DOM-005](./03-domain-model-rules.md) and [DOM-027](./03-domain-model-rules.md)). A posting that would take the balance below the limit floor MUST be refused with the credit limit code (API-021).
- Verify: boundary tests at limit, limit minus one minor unit, and limit plus one.

**API-067** (MUST) Credit-limit checks and the posting they guard MUST be evaluated against a consistent view of the account. Concurrent postings to one account MUST NOT jointly exceed the limit.
- Verify: concurrent proposals and completions against one account near its limit never produce a balance below the floor.

**API-068** (MUST) A balance returned by any operation MUST equal the sum of that account's posted entries at the time of the read (invariant 6, [ROL-042](./02-personas-roles-journeys.md)).
- Verify: reconciliation job compares returned balances to recomputed sums for all accounts after a randomized sequence.

**API-069** (MUST) Only ACTIVE accounts MAY propose, accept, or complete transactions, and only ACTIVE accounts MAY have postings created against them ([DOM-003](./03-domain-model-rules.md)).
- Verify: attempts from SUSPENDED and DEPARTED accounts are refused with the account not active code.

**API-070** (MUST) Completion MUST be refused while a dispute on the transaction is open ([ROL-052](./02-personas-roles-journeys.md)). Raising a dispute MUST NOT change balances.
- Verify: completion during an open dispute is refused. Dispute raise leaves entries unchanged.

**API-071** (MUST) A policy change MUST apply only to operations created after it takes effect. It MUST NOT alter posted entries or the outcome of existing transactions or disputes ([ROL-062](./02-personas-roles-journeys.md)).
- Verify: policy change during an open transaction; the transaction completes or refuses under the policy version recorded on it.

**API-072** (MUST) Lowering a credit limit MUST NOT alter existing balances. Further debits that would breach the new limit MUST be refused until the account is within limit ([ROL-063](./02-personas-roles-journeys.md)).
- Verify: lower limit below current floor; balance unchanged; next debit refused.

### MVP operations (conceptual)

The table lists the MVP operations at a conceptual level. Each row is normative: the stated actor MUST be the only role permitted to invoke the operation, and the ledger effect MUST hold exactly.

**API-080** (MUST) Each operation in the table below MUST have exactly the stated ledger effect. An operation whose effect differs from its row is non-conforming.
- Verify: a contract test per row asserts the ledger effect, including zero entries where stated.

**API-081** (MUST NOT) The MVP operation set MUST NOT include any operation for payments, cash-out, currency conversion, or any token or cryptocurrency feature ([PRD-011](./01-product-vision-scope.md)).
- Verify: catalog review at each gate.

| Operation | Actor (role) | Ledger effect |
|---|---|---|
| Create network | Platform operator provisions; Network administrator configures ([PRD-006](./01-product-vision-scope.md)) | None. Creates network scope and policy record. Audit event. |
| Set or change policy value | Network administrator | None on entries or balances. Audit event with old and new value, actor, reason ([ROL-061](./02-personas-roles-journeys.md)). Applies to later operations only (API-071). |
| Submit join request | Prospective member | None. Pending request record. |
| Decide join request | Network administrator | None on balances. On approval, creates one account at zero balance with no entries ([ROL-011](./02-personas-roles-journeys.md)). Audit event. |
| Change account standing | Network administrator | None on entries. Audit event ([DOM-072](./03-domain-model-rules.md)). |
| Create, edit, pause, close offer or request | Member (own listings) | None ([ROL-022](./02-personas-roles-journeys.md)). |
| Remove offer or request | Network administrator | None on balances. Reason and audit event ([ROL-021](./02-personas-roles-journeys.md)). |
| Propose transaction | Active member (proposer) | None. Transaction in status proposed ([ROL-031](./02-personas-roles-journeys.md)). |
| Accept proposal | Counterpart only | None. Credit limit check per API-066 and API-069. Refused with reason if the check fails. |
| Decline or withdraw proposal | Counterpart (decline) or proposer (withdraw), before acceptance | None. Status change and audit event. |
| Complete transaction | Participant(s) per OD-07-1 | Balanced entries posted atomically (API-060, API-061). Refused while a dispute is open (API-070). Idempotent (API-045). |
| Raise dispute | Participant of an accepted or completed transaction, within dispute window | None. Transaction status disputed. Blocks completion (API-070). |
| Submit dispute statement | Either participant | None. |
| Assign dispute reviewer | Platform: from network roster; Network administrator: assigns and revokes roster ([ROL-066](./02-personas-roles-journeys.md)) | None. Audit event. |
| Record dispute decision: uphold | Assigned dispute reviewer | None. Reason and audit event. Transaction status restored. |
| Record dispute decision: resolve | Assigned dispute reviewer | New adjustment entries referencing original entries (API-064). Balanced (API-061). Reason and audit event. |
| Post non-dispute adjustment | Per OD-07-2 | New entries referencing original. Audit event. |
| Read own balance, limit, headroom, entries | Member | None. |
| Read network account balances and history | Network administrator (subject to [02](./02-personas-roles-journeys.md) open decision OD-6) | None. |
| Read assigned dispute record | Assigned dispute reviewer | None. |
| Export own history | Member ([ROL-045](./02-personas-roles-journeys.md), MAY) | None. |

Operations not listed are not MVP operations and MUST NOT be exposed.

## Open decisions

- **OD-07-1 Completion confirmers.** Who must invoke Complete for a transaction to post.
  - Options: (a) either participant; (b) both participants; (c) proposer only.
  - Default: (b). Matches the mutual-agreement model in [PRD-008](./01-product-vision-scope.md). Changing the default changes the Complete row in the operations table.

- **OD-07-2 Non-dispute adjustments.** Whether any role may post an adjustment outside dispute resolution.
  - Options: (a) none in MVP; (b) administrator requests, reviewer approves; (c) reviewer only, on referral.
  - Default: (a). Consistent with [ROL-064](./02-personas-roles-journeys.md).

- **OD-07-3 Idempotency key retention.** How long keys and their stored results are kept.
  - Options: (a) retained with the ledger record they guard; (b) network-configured with a platform minimum; (c) fixed short window.
  - Default: (a). Keeps API-041 true for any retry. Storage cost is accepted. Follows the retention default in [13](./13-assumptions-risks-decisions-roadmap.md) (RSK-036).

- **OD-07-4 Page size limits.** Whether page size limits are fixed or network-configured.
  - Options: (a) fixed platform maximum and default; (b) network-configured within a platform maximum.
  - Default: (a). Exact values to be set with the QA spec (QA prefix, not yet written).

- **OD-07-5 Deprecation window.** Minimum time a deprecated version or operation stays functional.
  - Options: (a) at least one full release cycle; (b) fixed calendar period; (c) set per release.
  - Default: (a), minimum one full release cycle. Exact period to be recorded in the OPS spec (not yet written).

- **OD-07-6 Public reads.** Whether any operation may be called without authentication.
  - Options: (a) none in MVP; (b) limited public read of network listing metadata.
  - Default: (a). Option (b) would also need a visibility decision under [RSK-032](./13-assumptions-risks-decisions-roadmap.md) and [02](./02-personas-roles-journeys.md) OD-5.

- **OD-07-7 Role-change effect timing.** Maximum delay between a role revocation and its enforcement on in-flight sessions.
  - Options: (a) enforced on the next operation; (b) bounded delay of a stated number of seconds.
  - Default: (a). Option (b) is acceptable only if the bound is written into the SEC spec.

## Related docs

- [_context.md](./_context.md): shared terms, ledger invariants, normative terms.
- [01 Product vision and scope](./01-product-vision-scope.md): PRD-006 to PRD-010, PRD-011, PRD-014, PRD-025.
- [02 Personas, roles and journeys](./02-personas-roles-journeys.md): roles, journeys, permissions matrix, ROL IDs cited above.
- [03 Domain model and rules](./03-domain-model-rules.md): DOM IDs cited above.
- [13 Assumptions, risks, decisions, roadmap](./13-assumptions-risks-decisions-roadmap.md): RSK IDs cited above.
- Not yet written, referenced by prefix only: DAT (data, ledger, audit), SEC (security), UX (user experience), QA (quality), OPS (operations).
