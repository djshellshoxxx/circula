# Data, Ledger and Audit

Status: Draft | Owner: Product architect | Updated: 2026-11-14 | Scope: data-level shape, identifiers, relationships, retention, integrity mechanics, audit-event contents, access and migration for ledger and audit data | Deps: [01](./01-product-vision-scope.md), [02](./02-personas-roles-journeys.md), [03](./03-domain-model-rules.md), [13](./13-assumptions-risks-decisions-roadmap.md), [_context](./_context.md) | Open questions: 7, see Open decisions

## Purpose

This spec defines how ledger and audit data is shaped, identified, related, retained, protected, and migrated. It does not define business rules. Rules for what a transaction may do, when a balance may change, and which dispute outcomes exist are owned by [03-domain-model-rules.md](./03-domain-model-rules.md). This spec cites them by ID (DOM-nnn) and adds the data-level expectations needed to enforce them.

Terms follow the [shared glossary](./_context.md#glossary). Ledger invariants 1 to 7 are stated in [_context](./_context.md#ledger-invariants-platform-wide-not-configurable) and realized here by DAT requirements.

This spec names no database product, storage engine, or schema language. It describes conceptual entities, not tables.

## Conceptual entities

- **Posting**: the atomic unit of ledger change. One posting is created per completed transaction (DOM-016), per adjustment (DOM-040), and per settlement or write-off recorded under a network rule (DOM-062, DOM-064). A posting belongs to one network, references one causing record and one audit event, and owns its entries.
- **Entry**: one signed integer amount posted to one account. Every entry belongs to exactly one posting.
- **Transition record**: one append-only record of an attempted or applied change to a transaction status (DOM-010).
- **Idempotency record**: the key, requester scope, request fingerprint, and stored result of a state-changing command.
- **Audit event**: the append-only record defined by DOM-070. This spec defines its stored shape.
- **Reconciliation finding**: the recorded result of a failed balance reconstruction check (DAT-052).

Other entities (network, member, account, offer, request, transaction, dispute, adjustment) are defined in [03](./03-domain-model-rules.md) and are referenced here, not redefined.

## Requirements

### Identifiers and relationships

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-001 | MUST | Every posting, entry, transition record, idempotency record, and audit event has an opaque identifier. Every entity named in DOM-001 has one too. Identifiers are assigned once at creation, are never reassigned, and are unique across all networks in a deployment. | Stable references make audit and reconstruction possible. | Two records created in different networks have different identifiers. An attempt to change an existing record identifier is rejected. |
| DAT-002 | MUST | Identifiers MUST NOT encode or be derived from personal data, display names, or network names. | Keeps identifiers stable when descriptive data changes and avoids leaking personal data through references. | Changing a member's display name or contact detail changes no identifier in any ledger or audit record. |
| DAT-003 | SHOULD | Identifiers SHOULD NOT reveal creation order or record counts to other networks or to unauthenticated callers. | Limits cross-network inference (see OD-06-7). | Sampled identifiers from two networks show no sequential pattern. Identifiers returned to a member do not allow counting records by probing. |
| DAT-004 | MUST | Relationships are fixed: each account references exactly one member and one network. Each entry references exactly one account and one posting. Each posting references exactly one network, one causing record (a transaction, an adjustment, or a rule-defined settlement), and one audit event. Each transaction references a provider account and a receiver account (DOM-009) and MAY reference one offer or request. An adjustment references the transaction and the original entry identifiers it corrects (DOM-040). | Explicit relationships make postings and corrections traceable. | A record with a missing, extra, or mismatched reference is rejected. An entry whose account and posting belong to different networks is rejected. |
| DAT-005 | MUST | Referential integrity for DAT-004 is enforced by the storage layer or an equivalent mechanism that application code cannot bypass. No orphan entry, posting, transition record, or audit event may exist. | Application checks alone are insufficient under retries and concurrency. | A direct write that references a missing posting or audit event is refused, including writes that skip the service layer. |
| DAT-006 | MUST | Every network-owned record carries its network reference, and every read path filters by network. Except for platform operations defined under DAT-082, no query returns records from more than one network. | Implements the isolation in DOM-001 at the data layer. | A query scoped to network A returns no record whose network reference is B, including when identifiers are guessed. |
| DAT-007 | MUST | At most one account exists per (member, network) pair. The storage layer enforces this, including under concurrent creation. | Implements DOM-002. | Two concurrent account creations for the same pair produce one account. |

### Ledger record shape

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-008 | MUST | Entry amounts are signed integers in network minor units (DOM-009; unit is decided in [03](./03-domain-model-rules.md) OD-08). No fractional, floating-point, or variable-scale representation is stored. | Exact arithmetic is required for DOM-020. | A non-integer amount is rejected. A stored amount exported and re-imported is bit-identical. |
| DAT-009 | MUST | The signed sum of a posting's entries is zero before the posting commits. A posting with a non-zero sum is rejected. | Implements invariant 1 (DOM-016, DOM-020). | A posting of +100 and -99 is rejected and writes no records. |
| DAT-010 | MUST | Each entry and posting receives a sequence number at commit. Within a network, sequence numbers are unique and strictly increase in commit order. Gaps caused by rolled-back attempts are allowed. | Enables reconstruction as of a point in history (DAT-051). | Under concurrent postings in one network, all sequence numbers are unique and ordered consistently with commit order. |
| DAT-011 | MUST | Each entry and posting stores a UTC commit timestamp. | Unambiguous ordering and audit timing (DOM-070). | Stored timestamps are UTC regardless of server or client locale. |
| DAT-012 | MUST | A correction entry stores the identifier of the entry it corrects and a correction type. A correction that takes an account below its floor carries the correction flag required by DOM-043 and the audit reference required by DOM-043. | Makes corrections traceable (DOM-023). | A correction without a referenced entry is rejected. A correction below the floor without the flag is rejected. |
| DAT-013 | MUST | Transaction status changes are recorded in an append-only transition history. Each record stores the from-status, to-status, actor reference, UTC timestamp, audit event reference, and outcome (APPLIED or REJECTED). Rejected attempts (DOM-010) are recorded here. | Transition history makes DOM-010 checkable from stored data. | For a sample transaction, the current status equals the last APPLIED to-status, and every REJECTED attempt appears once. |

### Idempotency

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-020 | MUST | A posting is keyed by (network, transaction, posting type) (DOM-024). The storage layer enforces uniqueness of that key. | Retries must not double-post (invariant 5). | Ten concurrent identical completion requests yield one posting and one pair of entries. |
| DAT-021 | MUST | Every state-changing command (propose, accept, reject, cancel, complete, adjust, dispute action, standing change, policy change) carries a client-generated idempotency key. The key is stored with a fingerprint of the request body and the result. | Network retries are normal. | Replaying the same key and body returns the stored result and writes no entries or audit events. Reusing the key with a different body is rejected. |
| DAT-022 | MUST | A retried request returns the original result, including the same posting, record, and audit event identifiers. | Callers must be able to rely on a retry returning the first outcome. | A retry after a simulated timeout returns identifiers equal to the first response. |
| DAT-023 | MUST | An idempotency record is written in the same atomic unit as the change it guards (DAT-030). | A committed change always has its key, and a key never exists without its change. | Fault injection between the change and key writes leaves both or neither. |
| DAT-024 | MUST | No idempotency record is removed while the posting or record it guards exists. | Keeps retry safety for the life of the guarded record. | A retention process run leaves every idempotency record whose guarded record still exists. |
| DAT-025 | MUST | Idempotency keys are scoped to the requesting identity. The same key string from two requesters does not collide. | Prevents one member from interfering with another member's retries. | Two requesters submitting the same key string each receive their own result. |

### Atomicity and concurrency

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-030 | MUST | A posting's entries, posting record, transaction status change, idempotency record, and audit event commit in one atomic unit (DOM-016, DOM-026). | Implements invariant 7. | Failure injected after each intermediate write leaves either all of these records or none. |
| DAT-031 | MUST | The balance floor check (DOM-027) and the entry writes for every account in a posting run in the same atomic unit, under per-account serialization (DOM-028). | A check outside the write unit can be invalidated by a concurrent posting. | Two concurrent debits of 60 against a limit of 100 with balance 0 produce exactly one committed entry. |
| DAT-032 | MUST | When a posting touches several accounts, the accounts are locked in a single deterministic order. | Prevents deadlock between postings that share accounts. | Concurrent postings that share two accounts in opposite order all complete or fail cleanly, and invariants hold. |

### Append-only integrity and reconstruction

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-040 | MUST | Application identities have no update or delete permission on entries, postings, transition records, or audit events. This restriction is enforced below application code (by privileges or storage-level controls). | Implements invariant 2 and DOM-021 and DOM-071 at every layer that can reach the data. | An update or delete issued with application credentials is refused by the storage layer. |
| DAT-041 | MUST | A stored balance changes only in the same atomic unit as the posting that causes it (DOM-022). A balance value cannot be written alone. | Implements invariant 3. | A balance write with no matching posting is refused. |
| DAT-042 | SHOULD | Audit events SHOULD carry tamper-evidence as decided in OD-06-3. | Audit trustworthiness (DOM-071). | Per the chosen option in OD-06-3, altering a stored audit event is detected by the verification procedure. |
| DAT-050 | MUST | For every account, the balance recomputed from its entries equals the stored balance (DOM-025, invariant 6). | Enables integrity checks and audit. | After any test sequence, the reconstruction check reports no mismatch. |
| DAT-051 | MUST | The balance of an account as of sequence number S is computable from entries with sequence numbers at most S. | Supports point-in-time audit and dispute review. | For a sample account and sequence S, the computed balance equals the balance recorded at commit of S. |
| DAT-052 | MUST | The reconstruction check (DAT-050) runs before every migration that touches ledger data (DAT-101) and at the interval set by network policy. The default interval is at least once per 24 hours. A mismatch creates a reconciliation finding and an audit event with a system actor. | Early detection of silent mutation (invariant 3). | An injected balance mismatch is reported as a finding and produces one audit event. |
| DAT-053 | MUST | A mismatch is never corrected silently. Correction is made only by an adjustment with authority (DOM-023, DOM-042) or by a recovery procedure that writes an audit event stating its reason. | Implements DOM-022. | A balance repair that writes no audit event is rejected by the procedure. |

### Retention

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-060 | MUST | No automated process removes postings, entries, transition records, or audit events for the life of the network. | Preserves history and conservation (DOM-061, DOM-022). | A retention run over a fixture with aged ledger and audit data leaves all of those records intact. |
| DAT-061 | MUST | Personal data held outside ledger and audit records is kept only for the period and purpose set by network policy (OD-06-6). When the period ends, the data is removed or pseudonymized by the method set in OD-06-2. | Limits personal data to what is needed. | After the period ends, the profile fields are absent or pseudonymized, and ledger identifiers still resolve. |
| DAT-062 | MUST | Records of a DEPARTED account are retained under DOM-061. Their identifiers remain resolvable after any action taken under DAT-061. | History must stay readable. | After profile pseudonymization for a departed account, its entries and balance are still readable and the network sum check (DOM-020) passes. |
| DAT-063 | SHOULD | Operational records (for example request logs) SHOULD be kept out of ledger and audit stores. Their retention is defined by the security spec (SEC prefix). | Keeps ledger and audit data to its purpose. | No ledger or audit table contains operational-log fields. |

### Audit event content

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-070 | MUST | Each audit event stores: its identifier; network reference; actor reference (member account, administrator identity, dispute reviewer identity, or system); action code; target references; UTC occurrence time; outcome (APPLIED or REJECTED); reason where DOM-070 requires one; old and new values for policy and standing changes (DOM-072); the idempotency key reference or request correlation identifier; and the posting reference when one exists. | Implements DOM-070 and DOM-072 as stored fields. | For each action listed in DOM-070, one event exists with every applicable field populated. An administrative action with no reason is rejected. |
| DAT-071 | MUST | Action codes come from a closed, versioned set. Adding a code requires a recorded change. | Keeps audit records machine-checkable. | An event with an unknown action code is rejected. |
| DAT-072 | MUST | The reason field is bounded in length. It MUST NOT be required to contain personal data. The system MUST NOT copy profile data into it. Automatically generated reasons contain identifiers and codes only. | Data minimization in audit (see Data minimization). | A reason beyond the length limit is rejected. Generated reasons contain no names or contact details. |
| DAT-073 | MUST | Audit events MUST NOT store contact details, profile fields, message content, dispute narrative text, or client network metadata (client network metadata is decided in OD-06-4). Narratives are held in the dispute record, and the audit event references its identifier. | Keeps audit records minimal while still attributable. | A schema review of audit events finds none of the listed fields. |
| DAT-074 | MUST | Old and new values stored for policy or standing changes contain only the changed setting. | Prevents audit records from carrying unrelated configuration or personal data. | A credit-limit change records the limit values and no other settings. |

### Access

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-080 | MUST | Read access to ledger and audit data follows the [permissions matrix](./02-personas-roles-journeys.md#permissions-matrix). By default: a network administrator reads events and entries for their network; a member reads entries and events for their own account and for transactions they are party to, with audit fields limited to action, time, outcome, and transaction reference; a dispute reviewer reads events for disputes assigned to them; platform operators have no content access (OD-06-5). | Least privilege, consistent with DOM-051. | Each role's read requests are tested against the matrix. A member request for another member's entries is denied under default visibility. |
| DAT-081 | MUST | No role can read audit events or ledger records of another network. | Implements DOM-001. | A cross-network read request by any role returns no records. |
| DAT-082 | MUST | Any access to ledger or audit content by platform operator personnel is recorded as an audit event that states its reason. Operator metadata access (counts, health) is not content access and is exempt. | Operator access must be attributable. | An operator content read produces one audit event with a reason. |
| DAT-083 | MUST | Bulk export of ledger or audit data is limited to network administrators and is scoped to one network. | Prevents bulk exposure of personal data. | A bulk export request by a member or operator is refused. An export contains records of one network only. |
| DAT-084 | MUST | A member can see another member's balance or entries only where network visibility policy allows it. The default is no visibility beyond what a transaction shows the member. Visibility defaults are network policy ([_context](./_context.md#configurable-network-policy-vs-invariant)). | Limits exposure of balances. | Under default policy, a member's request for a non-party account balance is denied. |

### Data minimization

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-090 | MUST | Ledger and audit records hold identifiers and enumerated values, not personal data. The only person-linked value is a member or account identifier. Display names, contact details, and profile fields live in member records outside the ledger. | Removing personal data from history keeps pseudonymization possible without touching the ledger. | The ledger schema contains no name, contact, address, or free-text personal field. Deleting a member's profile record leaves all ledger records intact. |
| DAT-091 | MUST | Adding a data field to a ledger or audit record requires a recorded decision that cites this spec and a minimization review. | Keeps data growth deliberate. | A schema change without the decision reference fails review. |

### Migration principles

| ID | Priority | Requirement | Rationale | Testable criterion |
|---|---|---|---|---|
| DAT-100 | MUST | Migrations MUST NOT update, delete, or renumber existing entries, postings, transition records, idempotency records, or audit events. A migration that changes storage representation MUST preserve the logical content of each record. | Implements invariant 2 across schema change. | A migration dry run on a fixture produces matching logical checksums before and after. |
| DAT-101 | MUST | Before and after any migration that touches ledger data, the reconstruction check (DAT-050) passes, and each account balance equals its pre-migration value. | Proves conservation survives migration. | A dry run on a production-shaped copy shows equal balances per account. |
| DAT-102 | MUST | Identifiers, sequence numbers, and timestamps are preserved across migrations. | Keeps references and point-in-time reads valid. | Identifiers and sequence numbers sampled before and after a migration are equal. |
| DAT-103 | MUST | Each migration is versioned, its application is recorded, and it produces an audit event with an actor and version identifiers. | Migration history must be reconstructable. | The migration history lists every applied version, and each has one audit event. |
| DAT-104 | MUST | Before a migration that cannot be reversed by a defined rollback, a recovery point is taken and a roll-forward procedure is documented. A migration without either is blocked. | Limits the impact of a failed migration. | A migration without a recovery point or procedure is refused by the migration runner. |
| DAT-105 | SHOULD | Migrations SHOULD run without freezing ledger writes longer than a window defined in the operations spec (OPS prefix). | Keeps trading available. | Measured write-freeze during a migration rehearsal is within the defined window. |
| DAT-106 | MAY | A network MAY be exported to another deployment. If it is, the export carries identifiers, sequence numbers, entries, transition records, and audit events. Import MUST pass DAT-050 before the network becomes active. | Portability without silent loss. | An export and import round trip yields equal identifiers, sequences, and balances, and the reconstruction check passes. |

## Open decisions

- **OD-06-1 Member identity scope.** Options: (a) a member is identified per network, with no cross-network person record; (b) a global identity record with per-network membership. **Default: (a).** It matches the isolation in DOM-001 and minimizes data (DAT-090). Choosing (b) requires revisiting DAT-090 and DAT-002.
- **OD-06-2 Method of personal-data removal.** Options: (a) no removal while the account exists; (b) pseudonymize profile fields and keep opaque identifiers in the ledger; (c) delete profile fields and keep opaque identifiers. **Default: (b).** This spec makes no legal conclusion about whether any option is sufficient. Legal review is required before launch.
- **OD-06-3 Tamper-evidence for audit events.** Options: (a) access controls only (DAT-040); (b) each audit event stores a digest linking it to the previous event in the same network; (c) (b) plus periodic anchoring outside the platform. **Default: (b).** The digest algorithm is not chosen here.
- **OD-06-4 Client network metadata in audit events.** Options: (a) not stored in audit events; (b) stored with a short, separately defined retention. **Default: (a).** The security spec decides any exception.
- **OD-06-5 Platform operator access to content.** Options: (a) no content access without network administrator authorization; (b) support read access with a required reason and audit event; (c) unrestricted access. **Default: (a).** Option (c) is excluded by DAT-082.
- **OD-06-6 Retention periods for personal data outside the ledger.** Options: (a) one platform-wide value; (b) a per-network setting with a platform default. **Default: (b).** The numeric period is not set in this spec and is for the product owner to decide.
- **OD-06-7 Sequence scope.** Options: (a) sequence numbers are per network; (b) one sequence across the deployment. **Default: (a).** Networks are isolated (DOM-001), and a global sequence would expose cross-network activity (DAT-003).

## Related docs

- [_context](./_context.md): glossary, normative terms, ledger invariants, header template.
- [01 Product vision and scope](./01-product-vision-scope.md): purpose and scope of the platform.
- [02 Personas, roles and core journeys](./02-personas-roles-journeys.md): roles and the permissions matrix referenced by DAT-080.
- [03 Domain model and mutual-credit rules](./03-domain-model-rules.md): the DOM rules this spec realizes.
- [13 Assumptions, risks, decisions and roadmap](./13-assumptions-risks-decisions-roadmap.md): decision log that records changes to the open decisions above.
