# Quality Strategy and Test Plan
Status: Draft | Owner: Quality lead (role) | Updated: 2026-11-14 | Scope: test layers, required coverage, requirement traceability, end-to-end acceptance scenarios, quality gates | Deps: [_context](./_context.md), [03](./03-domain-model-rules.md), [04](./04-functional-requirements.md), [06](./06-data-ledger-audit.md), [07](./07-api-integration-contract.md), [08](./08-ux-accessibility.md), [09](./09-security-privacy-abuse.md), [11](./11-deployment-operations-support.md) | Open questions: 7, see "Open decisions"

## Purpose

This spec defines how the platform is verified. It sets the required test layers, the minimum coverage per risk area, the traceability from every test to the requirement IDs it checks, a small set of end-to-end acceptance scenarios, and measurable quality gates.

It does not restate the rules being tested. Each rule has one authoritative home in the linked specs. This spec cites those IDs and does not define new behavior.

Status of this document: it is a plan. No tests have been run against the platform to support any statement here.

## Conventions

Normative terms (MUST, SHOULD, MAY) and the ID format follow [_context](./_context.md). IDs in this spec use the prefix `QA`. Open decisions use `OD-12-N`. Requirement IDs cited here (DOM, FR, DAT, API, UX, SEC, OPS, and others) are defined only in the linked specs.

## 1. Test layers

Each layer has a defined scope. A change MUST run every layer marked "required" for its change class (QA-001).

| Layer | Scope | Required when |
|---|---|---|
| L1 Unit | Pure domain rules: balance computation, state transitions, input validation. | Any domain rule change. |
| L2 Invariant (generated sequences) | Randomly generated sequences of propose, accept, complete, dispute, adjust, and reverse operations, checked against the ledger invariants after every committed step. | Any change to posting, ledger, dispute, or adjustment code. |
| L3 Storage integration | Real transactional storage: constraints, permissions, row locking, idempotency record atomicity. | Any schema or write-path change. |
| L4 Concurrency | Parallel clients acting on shared accounts and transactions. Includes deterministic interleavings and sustained contention runs. | Any write-path change. |
| L5 Fault injection | Process termination, storage errors, and client timeouts injected at each write step. | Any write-path change. |
| L6 API contract | Every operation: success result, refusal codes, idempotency semantics, versioning. | Any interface change. |
| L7 Authorization matrix | Role x operation x network x account/transaction state. | Any role, permission, or operation change. |
| L8 Migration | Rehearsed schema and data migrations with reconstruction checks before and after. | Any migration. |
| L9 Backup and restore drill | Restore into an isolated target, then reconstruction check and behavior checks. | Any backup, restore, or storage change; on the drill schedule. |
| L10 Export verification | Recompute and compare exported data against source records. | Any export change. |
| L11 Accessibility | Automated scan per screen plus manual assistive-technology review of MVP flows. | Any UI change; manual review per release. |
| L12 End-to-end acceptance | Scenarios in section 6. | Every release candidate (see OD-12-5). |
| L13 Non-functional | Contention and latency measured against thresholds in section 7. | Release candidates and scheduled runs. |

## 2. Requirements

### 2.1 Strategy and coverage

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-001 | MUST | A change MUST run every test layer marked "required" for its change class in section 1, and the results MUST be recorded per change. | Prevents layer skipping on high-risk code. |
| QA-002 | MUST | Test environments and test data MUST use synthetic identities and synthetic amounts only. No production personal data MAY appear in CI or development environments. Use of a production-derived copy is governed by OD-12-7. | Limits exposure of member data during testing. |
| QA-003 | MUST | Every requirement ID with priority MUST in the linked specs MUST map to at least one automated test or to a named manual check. A review or script MUST find zero unmapped MUST IDs before release. | Ensures no rule is untested. |
| QA-004 | MUST | Every test MUST cite at least one requirement ID, except performance tests that cite a gate in section 7. | Keeps tests traceable to rules. |

### 2.2 Ledger invariants

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-010 | MUST | After every committed operation in L2 sequences, the signed sum of all account balances MUST equal zero. | Balanced invariant holds across all paths. |
| QA-011 | MUST | Every completed transaction MUST be tested to produce exactly two entries that are equal and opposite. Any other entry count MUST fail the test. | Checks the posting shape. |
| QA-012 | MUST | Tests MUST attempt update and delete of posted entries and audit events using the application identity. Every attempt MUST be refused at the storage layer. | Append-only is enforced below application code. |
| QA-013 | MUST | Every change to a stored balance MUST be traced in tests to exactly one posting and one audit event. A balance change with no such link MUST fail. | No silent mutation. |
| QA-014 | MUST | For every account in every L2 history and after every fault-injection run, the balance recomputed from entries MUST equal the stored balance. | Reconstructable balances. |
| QA-015 | MUST | Tests MUST compute the balance of an account as of any sequence number S and compare it with an independent replay of entries up to S. | Point-in-time correctness. |
| QA-016 | MUST | Tests of corrections MUST confirm that each correction is a new entry referencing the original, and that the original entry is byte-for-byte unchanged afterwards. | Traceable correction. |
| QA-017 | MUST | Credit-limit boundary tests MUST show: a posting that leaves the balance exactly at the floor succeeds; a posting that leaves it one minimum unit below the floor is refused. | Boundary correctness of the floor check. |
| QA-018 | MUST | Tests of lowering a credit limit MUST confirm that existing entries and balances are unchanged, and that later debits beyond the new floor are refused. | Policy change does not rewrite history. |
| QA-019 | MUST | Tests MUST submit non-integer, negative-zero, and out-of-range amounts. Each MUST be refused, and no entry MUST be written. | Integer minor-unit amounts only. |
| QA-020 | MUST | Every refused mutating operation MUST leave no trace in ledger, transaction, dispute, or audit records beyond the refusal record the interface defines. | No partial effects from refusals. |

### 2.3 Concurrent transactions

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-030 | MUST | Tests with N parallel postings on one account MUST show that final balance equals the sum of accepted postings and that no accepted posting drives the balance below the floor. | Per-account serialization. |
| QA-031 | MUST | Tests with overlapping multi-account postings MUST show a deterministic lock order and no deadlock over the contention runs defined in OD-12-2. | Multi-account safety. |
| QA-032 | MUST | Tests MUST construct an interleaving in which two debits each pass the floor check alone but together would breach it. Exactly one MUST succeed. | Check and write are atomic together. |
| QA-033 | MUST | Fault injection at each write step of a posting MUST leave either all entries, the posting record, the status change, and the idempotency record committed, or none of them. | All-or-nothing posting. |
| QA-034 | MUST | Under concurrency, sequence numbers within a network MUST be unique and strictly increasing in commit order. | Reliable ordering for reconstruction. |
| QA-035 | MUST | A race between accept and cancel on the same PROPOSED transaction MUST yield exactly one terminal outcome. The losing request MUST be refused with a defined code. | Single status outcome. |
| QA-036 | MUST | In a race between completion and a dispute raise on the same transaction, a dispute MUST NOT be accepted against a transaction that is not COMPLETED, and completion MUST be refused while a dispute is open. Tests MUST check both orderings. | Dispute and completion do not overlap. |
| QA-037 | SHOULD | Sustained contention runs SHOULD execute at the load level and duration set by OD-12-2, and SHOULD record throughput and retry counts. | Detects rare races and livelock. |

### 2.4 Retries and idempotency

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-040 | MUST | A repeated request with the same key and equivalent content MUST return the original result and MUST NOT create a second posting. | Retry safety. |
| QA-041 | MUST | A repeated request with the same key and different content MUST be refused with the defined code and MUST NOT change any record. | Key reuse is detected. |
| QA-042 | MUST | Fault injection between the change and the idempotency record MUST NOT leave a posting without its record or a record without its posting. | Atomic guard. |
| QA-043 | MUST | Tests MUST show that the same key string used by two callers, or in two networks, produces two independent operations. | Key scope. |
| QA-044 | MUST | Tests MUST show that an idempotency record is not removed while the record or posting it guards exists. | Retention of guards. |
| QA-045 | MUST | A client retry loop run against injected timeouts and dropped responses MUST end with exactly one posting per transaction and posting type. | End-to-end retry safety. |
| QA-046 | MUST | Completion MUST be tested to be unique per transaction regardless of how many concurrent or repeated completion requests are sent. | Single completion. |
| QA-047 | SHOULD | Read operations SHOULD be tested to succeed without an idempotency key and SHOULD be tested to create no records. | Reads stay side-effect free. |

### 2.5 Authorization

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-050 | MUST | The L7 matrix MUST cover every MVP operation for every role, network, and relevant state. Each cell MUST be an explicit allow or deny. An operation not granted to a role MUST be denied by the test. | Deny by default is verified, not assumed. |
| QA-051 | MUST | Every MVP operation MUST be tested without credentials and MUST be refused. | No anonymous access. |
| QA-052 | MUST | Calls made with only the platform operator role MUST be refused for ledger writes, dispute decisions, and access to ledger or audit content. | Operator boundary. |
| QA-053 | MUST | A request for a record in another network MUST produce a response indistinguishable from a request for a record that does not exist, in code, shape, and timing class. | No existence leak across networks. |
| QA-054 | MUST | A credential whose role grant was revoked MUST be refused on the next request, with no retained grant after revocation is committed. | Revocation takes effect. |
| QA-055 | MUST | Each privileged operation tested MUST produce an audit event. A missing event MUST fail the test. | Privileged actions are recorded. |
| QA-056 | MUST | A member MUST see only their own balance and entries unless network visibility settings allow more. Tests MUST cover both settings. | Visibility rules. |
| QA-057 | MUST | A dispute reviewer who is a participant in the disputed transaction MUST be refused every review action on that dispute. | Impartiality check. |
| QA-058 | MUST | Tests MUST cover the read matrix for ledger and audit data, including that bulk export is available only to network administrators. | Read scope. |

### 2.6 Disputes

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-060 | MUST | Raising a dispute MUST be tested to change no balances and no entries, and to change only the transaction status defined by the rules. | Dispute is not a posting. |
| QA-061 | MUST | A second open dispute on the same transaction MUST be refused. | One open dispute per transaction. |
| QA-062 | MUST | Tests MUST show that a dispute can be raised only on a COMPLETED transaction and only by a participant. Other states and non-participants MUST be refused. | Eligibility. |
| QA-063 | MUST | A reversal or adjustment with zero, two, or an unlisted authority MUST be refused. A reversal with exactly one valid authority MUST succeed. | Single authority. |
| QA-064 | MUST | For each dispute outcome, tests MUST check the exact entries produced. UPHELD MUST produce no entries. Other outcomes MUST produce only the entries the rules define. | Outcome effects are exact. |
| QA-065 | MUST | Reversal and adjustment postings MUST be tested against the credit-limit check and against the exceptions the rules state. | Limit behavior for corrections. |
| QA-066 | MUST | Completion of a transaction with an open dispute MUST be refused with the defined code. | Dispute blocks completion. |
| QA-067 | MUST | Tests MUST show that a resolved dispute cannot be reopened through any operation. | Final resolution. |

### 2.7 Migrations

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-070 | MUST | For each migration rehearsal, the reconstruction check MUST pass before and after, and every account balance after MUST equal its value before. | Data integrity through migration. |
| QA-071 | MUST | A snapshot diff MUST show no updated, deleted, or renumbered entries, postings, transitions, or audit events. | Append-only through migration. |
| QA-072 | MUST | Identifiers, sequence numbers, and timestamps MUST be identical before and after the migration for all migrated records. | Stable references. |
| QA-073 | MUST | Tests MUST show that each migration records its version and application, and that an audit event is produced. | Migration traceability. |
| QA-074 | MUST | The migration runner MUST be tested to refuse out-of-order application and to refuse to run when a pre-check fails. | Safe ordering. |
| QA-075 | MUST | Tests MUST show that a migration that cannot be rolled back is refused unless a recovery point exists. | Recovery before irreversible change. |
| QA-076 | MUST | Every rehearsal MUST run on a production-like copy before a production upgrade, and MUST record measured duration and write-freeze impact against the expected values. | Rehearsal evidence. |
| QA-077 | SHOULD | The previous supported software version SHOULD be tested against the migrated schema. | Rollback and mixed-version safety. |
| QA-078 | SHOULD | Measured write-freeze time SHOULD be checked against the window in OD-12-3. | Bounded operational impact. |

### 2.8 Backup and restore

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-080 | MUST | A backup inventory test MUST show that every network-owned record class is included in backups. | Complete backups. |
| QA-081 | MUST | A corrupted backup copy MUST be detected by the integrity check at creation and at restore. | Backup integrity. |
| QA-082 | MUST | Each restore drill MUST run at the interval set by the operations spec. After restore, the reconstruction check MUST pass and balances MUST equal their values at backup time. | Restore proves recoverability. |
| QA-083 | MUST | A restore forced to fail partway MUST leave the target unchanged. | All-or-nothing restore. |
| QA-084 | MUST | Tests MUST show that a restore requires a named target and confirmation, and that it produces an audited administrative action. | Controlled restore. |
| QA-085 | MUST | A restored deployment MUST refuse postings until the reconstruction check passes. Tests MUST attempt a posting before and after the check. | Restored state is verified before use. |
| QA-086 | MUST | When a restore rolls back state, tests MUST show that the rolled-back period is recorded. | Rollback is visible. |
| QA-087 | MUST | Tests MUST show that backups are encrypted at rest and that restore access is limited to the defined roles. | Backup confidentiality. |
| QA-088 | MUST | Tests MUST show that recovery steps do not delete or alter audit events or ledger entries. | No history rewriting during recovery. |

### 2.9 Export correctness

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-090 | MUST | For each network export, balances recomputed from the exported entries MUST equal the balances stored at the export sequence, and the signed sum MUST be zero. | Export is self-consistent. |
| QA-091 | MUST | An export MUST contain only records of the requested network. Tests MUST seed a second network and confirm none of its records appear. | Network isolation in export. |
| QA-092 | MUST | Exports MUST be scanned for credentials, session data, and second-factor material. Any match MUST fail the test. | Export is free of secrets. |
| QA-093 | MUST | Each export MUST produce an audit event with requester, time, and scope. A missing event MUST fail the test. | Exports are recorded. |
| QA-094 | MUST | Tests MUST show that an export is delivered only through the authenticated session that requested it. | Export delivery control. |
| QA-095 | MUST | A member export MUST contain no identifying data about other members beyond what the rules allow. Tests MUST check with seeded counterparties. | Counterparty privacy. |
| QA-096 | MUST | Audit export tests MUST show that every audit event in scope is present, with no omissions and no duplicates, compared with the source count and sequence range. | Audit completeness. |
| QA-097 | MUST | Export headers, field descriptions, and generated text MUST be scanned for any statement that credits have monetary value, are redeemable, or are a security or cryptocurrency. Any match MUST fail the test. | Credit semantics in output. |
| QA-098 | MUST | If the member export of own entry history is implemented, tests MUST show it is machine-readable and complete for the member's entries. If it is not implemented, this requirement is not applicable. | Conditional on the export feature. |
| QA-099 | SHOULD | Two exports taken at the same sequence SHOULD be byte-identical in content, excluding the generation timestamp. | Deterministic exports aid verification. |

### 2.10 Accessibility

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-110 | MUST | Each MVP screen MUST pass an automated accessibility scan at the target level, with zero violations at the severity threshold in OD-12-4. | Baseline automated coverage. |
| QA-111 | MUST | Automated checks MUST confirm text contrast meets the ratio for the target level. | Contrast. |
| QA-112 | MUST | Tests MUST confirm that status, error, and balance-sign information is also conveyed by text or shape, not color alone. | Color independence. |
| QA-113 | MUST | Tests MUST confirm every control exposes name, role, and state to assistive technology. | Programmatic semantics. |
| QA-114 | MUST | Tests MUST confirm every data table associates each cell with its column and row headers. | Table semantics. |
| QA-115 | MUST | Tests MUST confirm each page declares its language, and that content in another language is marked. | Language identification. |
| QA-116 | MUST | Every MVP flow in section 6 MUST be completable using the keyboard alone. Tests MUST confirm no single-key shortcut is the only route to any action. | Keyboard operability. |
| QA-117 | MUST | Each release MUST have a manual review with assistive technology of every MVP flow, with the result recorded. Automated scans alone do not satisfy this requirement. | Automated scans miss many issues. |
| QA-118 | SHOULD | Non-essential motion SHOULD be checked to be avoidable and to stop or pause on request where used. | Motion sensitivity. |

### 2.11 End-to-end acceptance

Scenarios are in section 6. Each MUST pass on every release candidate.

| ID | Priority | Requirement (testable) | Rationale |
|---|---|---|---|
| QA-120 | MUST | Scenario E2E-1 (section 6) MUST pass. | Core happy path. |
| QA-121 | MUST | Scenario E2E-2 MUST pass. | Limit refusal. |
| QA-122 | MUST | Scenario E2E-3 MUST pass. | Retry safety. |
| QA-123 | MUST | Scenario E2E-4 MUST pass. | Dispute lifecycle. |
| QA-124 | MUST | Scenario E2E-5 MUST pass. | Cross-network isolation. |
| QA-125 | MUST | Scenario E2E-6 MUST pass. | Restore readiness. |
| QA-126 | MUST | Scenario E2E-7 MUST pass. | Migration safety. |

### 2.12 Quality gates

Thresholds marked "product decision" are not set by this spec. Gate IDs are part of the release checklist.

| ID | Priority | Gate | Threshold | Threshold status |
|---|---|---|---|---|
| QA-130 | MUST | Invariant suite (QA-010 to QA-020) | 0 failures | Fixed |
| QA-131 | MUST | Reconstruction check (DAT-050) in every CI run that touches ledger code | 0 mismatches | Fixed |
| QA-132 | MUST | Authorization matrix (QA-050 to QA-058) | 0 unexplained allow cells; 0 open critical or high defects | Severity definitions: OD-12-6 |
| QA-133 | MUST | Concurrency and fault-injection suites (QA-030 to QA-036, QA-042, QA-045) | 0 failures across the run count set in OD-12-2 | Product decision: OD-12-2 |
| QA-134 | MUST | Automated accessibility scan (QA-110) | 0 violations at the severity set in OD-12-4 | Product decision: OD-12-4 |
| QA-135 | MUST | Restore drill (QA-082) and migration rehearsal (QA-076) | Completed within the interval and window recorded per the operations spec | Linked: see OD-12-3 |
| QA-136 | SHOULD | Line and branch coverage for domain-rule modules | Reported per release; numeric minimum not set | Product decision: OD-12-1 |
| QA-137 | MUST | Traceability check (QA-003) | 0 unmapped MUST IDs | Fixed |

## 3. Traceability

Each row lists the requirement IDs whose behavior the QA test checks. Columns link to the authoritative spec: [DOM](./03-domain-model-rules.md), [FR](./04-functional-requirements.md), [DAT](./06-data-ledger-audit.md), [API](./07-api-integration-contract.md), [UX](./08-ux-accessibility.md), [SEC](./09-security-privacy-abuse.md), [OPS](./11-deployment-operations-support.md). "—" means no ID in that domain applies. 

| QA | DOM | FR | DAT | API | UX | SEC | OPS |
|---|---|---|---|---|---|---|---|
| QA-010 | DOM-020 | — | DAT-009 | API-061 | — | — | — |
| QA-011 | DOM-016 | — | — | API-060 | — | — | — |
| QA-012 | DOM-021 | — | DAT-040 | API-063 | — | — | — |
| QA-013 | DOM-022 | — | DAT-041 | API-065 | — | — | — |
| QA-014 | DOM-025 | — | DAT-050 | — | — | — | — |
| QA-015 | — | — | DAT-051 | — | — | — | — |
| QA-016 | DOM-023 | — | DAT-012 | API-064 | — | — | — |
| QA-017 | DOM-027 | — | — | API-066 | — | — | — |
| QA-018 | DOM-029 | — | — | API-072 | — | — | — |
| QA-019 | — | — | DAT-008 | API-007 | — | — | — |
| QA-020 | — | — | — | API-022 | — | — | — |
| QA-030 | DOM-028 | FR-024 | DAT-031 | — | — | — | — |
| QA-031 | — | — | DAT-032 | — | — | — | — |
| QA-032 | — | — | DAT-031 | API-067 | — | — | — |
| QA-033 | DOM-026 | — | DAT-030 | API-060 | — | — | — |
| QA-034 | — | — | DAT-010 | — | — | — | — |
| QA-035 | DOM-010, DOM-012 | — | — | — | — | — | — |
| QA-036 | DOM-052, DOM-054 | — | — | API-070 | — | — | — |
| QA-037 | DOM-028 | — | — | — | — | — | — |
| QA-040 | DOM-024 | — | DAT-022 | API-041 | — | — | — |
| QA-041 | — | — | — | API-042 | — | — | — |
| QA-042 | — | — | DAT-023 | API-044 | — | — | — |
| QA-043 | — | — | DAT-025 | API-043 | — | — | — |
| QA-044 | — | — | DAT-024 | — | — | — | — |
| QA-045 | DOM-024 | — | — | API-041 | — | — | OPS-081 |
| QA-046 | — | — | — | API-045 | — | — | — |
| QA-047 | — | — | — | API-046, API-004 | — | — | — |
| QA-050 | — | — | — | API-011 | — | SEC-009 | — |
| QA-051 | — | — | — | API-010 | — | — | — |
| QA-052 | — | FR-011 | — | API-012 | — | — | — |
| QA-053 | — | — | DAT-081 | API-013 | — | — | — |
| QA-054 | — | — | — | API-015 | — | SEC-010 | — |
| QA-055 | — | — | — | API-014 | — | — | — |
| QA-056 | — | FR-030 | DAT-084 | — | — | — | — |
| QA-057 | DOM-051 | — | — | — | — | — | — |
| QA-058 | — | — | DAT-080, DAT-083 | — | — | — | — |
| QA-060 | DOM-052 | FR-036 | — | — | — | — | — |
| QA-061 | DOM-054 | FR-040 | — | — | — | — | — |
| QA-062 | DOM-050 | — | — | — | — | — | — |
| QA-063 | DOM-042 | FR-041 | — | — | — | — | — |
| QA-064 | DOM-053, DOM-041 | — | — | — | — | — | — |
| QA-065 | DOM-027, DOM-043 | — | — | — | — | — | — |
| QA-066 | — | — | — | API-070 | — | — | — |
| QA-067 | DOM-054 | — | — | — | — | — | — |
| QA-070 | — | — | DAT-052, DAT-101 | — | — | — | OPS-053 |
| QA-071 | — | — | DAT-100 | — | — | — | — |
| QA-072 | — | — | DAT-102 | — | — | — | — |
| QA-073 | — | — | DAT-103 | — | — | — | — |
| QA-074 | — | — | — | — | — | — | OPS-061 |
| QA-075 | — | — | DAT-104 | — | — | — | — |
| QA-076 | — | — | — | — | — | — | OPS-052, OPS-062 |
| QA-077 | — | — | — | — | — | — | OPS-063 |
| QA-078 | — | — | DAT-105 | — | — | — | OPS-055 |
| QA-080 | — | — | — | — | — | — | OPS-030 |
| QA-081 | — | — | — | — | — | — | OPS-031 |
| QA-082 | — | — | DAT-050 | — | — | — | OPS-033 |
| QA-083 | — | — | — | — | — | — | OPS-034 |
| QA-084 | — | — | — | — | — | SEC-042 | OPS-035 |
| QA-085 | — | — | — | — | — | — | OPS-036 |
| QA-086 | — | — | — | — | — | SEC-043 | — |
| QA-087 | — | — | — | — | — | SEC-041 | — |
| QA-088 | — | — | — | — | — | — | OPS-082 |
| QA-090 | — | FR-045 | DAT-050 | — | — | SEC-059 | — |
| QA-091 | — | FR-045 | DAT-081 | — | — | — | — |
| QA-092 | — | — | — | — | — | SEC-062 | — |
| QA-093 | — | — | — | — | — | SEC-060 | — |
| QA-094 | — | — | — | — | — | SEC-061 | — |
| QA-095 | — | FR-046 | DAT-084 | — | — | — | — |
| QA-096 | — | — | — | — | — | SEC-063 | — |
| QA-097 | DOM-082 | FR-047 | — | — | — | — | — |
| QA-098 | — | FR-044 | — | — | — | — | — |
| QA-099 | — | — | DAT-051 | — | — | — | — |
| QA-110 | — | — | — | — | UX-067 | — | — |
| QA-111 | — | — | — | — | UX-061 | — | — |
| QA-112 | — | — | — | — | UX-062 | — | — |
| QA-113 | — | — | — | — | UX-063 | — | — |
| QA-114 | — | — | — | — | UX-064 | — | — |
| QA-115 | — | — | — | — | UX-065 | — | — |
| QA-116 | — | — | — | — | UX-044 | — | — |
| QA-117 | — | — | — | — | UX-060 | — | — |
| QA-118 | — | — | — | — | UX-066 | — | — |
| QA-120 to QA-126 | see section 6 | see section 6 | see section 6 | see section 6 | see section 6 | see section 6 | see section 6 |
| QA-130 | DOM-020 | — | DAT-050 | — | — | — | — |
| QA-131 | DOM-025 | — | DAT-050 | — | — | — | OPS-083 |
| QA-132 | — | — | — | API-011 | — | SEC-009 | — |
| QA-133 | DOM-028 | FR-024 | DAT-031 | API-040 | — | — | — |
| QA-134 | — | — | — | — | UX-060, UX-067 | — | — |
| QA-135 | — | — | DAT-052 | — | — | SEC-045 | OPS-033 |
| QA-136 | — | — | — | — | — | — | — |
| QA-137 | — | — | — | — | — | — | — |

## 4. Coverage of required areas

| Area | Requirements | Layers |
|---|---|---|
| Ledger invariants | QA-010 to QA-020 | L1, L2, L3 |
| Concurrent transactions | QA-030 to QA-037 | L3, L4, L5 |
| Retries and idempotency | QA-040 to QA-047 | L3, L5, L6 |
| Authorization | QA-050 to QA-058 | L6, L7 |
| Disputes | QA-060 to QA-067 | L2, L6 |
| Migrations | QA-070 to QA-078 | L8 |
| Backup and restore | QA-080 to QA-088 | L9 |
| Export correctness | QA-090 to QA-099 | L10 |
| Accessibility | QA-110 to QA-118 | L11 |
| End-to-end acceptance | QA-120 to QA-126 | L12 |
| Quality gates | QA-130 to QA-137 | all |

## 5. Test data and environments

- Test identities and amounts are synthetic (QA-002).
- Each network-scoped test MUST seed at least two networks so that isolation tests (QA-053, QA-091) can detect leaks.
- Fault injection and concurrency tests MUST be reproducible from a recorded seed or schedule. A failing run MUST keep its seed or schedule.
- Clock-dependent tests (listing expiry, acceptance windows, dispute windows) MUST use controllable time. Configured windows are network policy; this spec does not set their values.

## 6. End-to-end acceptance scenarios

Each scenario runs against a deployment built from the release candidate. Steps are described in behavioral terms. Expected results cite the rules that define them.

**E2E-1 Core exchange (QA-120).** Member A proposes a transaction with member B as counterparty. B accepts. Both parties confirm completion. Expected: one COMPLETED transaction; two equal and opposite entries; balances of A and B change by equal and opposite amounts; signed sum over the network is zero; audit events exist for each step. Rules: FR-018, FR-019, DOM-015, DOM-016, DOM-020, DOM-070.

**E2E-2 Limit refusal (QA-121).** Member A has a balance one unit above the floor after a prior exchange. A proposes and completes a transaction that would breach the floor. Expected: completion is refused with the defined code; no entries, no transaction status change, no balance change; a refused mutating operation leaves no trace. Rules: DOM-027, API-066, API-022.

**E2E-3 Retry under network fault (QA-122).** A client submits a completion with an idempotency key. The response is dropped. The client retries three times with the same key and content. Expected: exactly one posting; the retries return the original result; no second posting exists after the retries. Rules: API-041, DAT-022, DOM-024, API-045.

**E2E-4 Dispute lifecycle (QA-123).** A participant raises a dispute on a COMPLETED transaction. A second participant attempts a second dispute on it. A reviewer who is not a participant resolves it as REVERSED. Expected: raising changes no balances; the second dispute is refused; the reversal cites the dispute outcome as its single authority; the reversal entries reference the original entries; balances recompute correctly; the dispute cannot be reopened. Rules: ROL-052, DOM-050, DOM-051, DOM-054, DOM-041, DOM-042, DOM-053.

**E2E-5 Cross-network isolation (QA-124).** A member of network A requests a transaction ID that belongs to network B, and a transaction ID that does not exist. Expected: both responses are indistinguishable in code and shape. No data from network B is returned. Rules: API-013, DAT-081, API-002.

**E2E-6 Restore readiness (QA-125).** A restore drill loads a backup into an isolated target. Expected: the restore either completes or leaves the target unchanged; the reconstruction check passes; a posting is refused before the check passes and accepted after it. Rules: OPS-033, OPS-034, OPS-036, DAT-050.

**E2E-7 Migration rehearsal (QA-126).** A copy of the data is migrated with the release's migration set. Expected: reconstruction check passes before and after; every account balance is unchanged; identifiers, sequence numbers, and timestamps are unchanged; migration version and application are recorded. Rules: DAT-100, DAT-101, DAT-102, DAT-103, OPS-052.

## 7. Quality gates and thresholds

Gate table entries are in section 2.12. Gates marked "Fixed" use thresholds that follow directly from the rules (zero failures, zero violations of an invariant). Gates marked "Product decision" MUST NOT be treated as final until the matching open decision is resolved. Until then, the gate is reported but does not block release.

## Open decisions

**OD-12-1 Numeric coverage minimum (QA-136).**
- Options: (a) report line and branch coverage per release, no minimum; (b) fixed branch-coverage minimum on domain-rule modules; (c) rule-to-test traceability as the only numeric gate (QA-003, QA-137), with coverage reported.
- Default: (c) for MVP. Revisit after the first two releases using observed defect data.

**OD-12-2 Concurrency and contention run size (QA-037, QA-133).**
- Options: (a) small fixed run per change, larger run on release candidates; (b) one large run per release only; (c) continuous nightly runs with a per-change smoke run.
- Default: (a). Exact parallelism, iteration count, and duration are set by the quality lead and the platform owner. They are not fixed here.

**OD-12-3 Migration write-freeze and rehearsal limits (QA-078, QA-135).**
- Options: (a) take the window and duration limits from the operations spec; (b) set separate limits here; (c) set limits per release.
- Default: (a). Limits are not stated in the operations spec as numeric values in this pass. QA-135 stays "linked" until the operations spec states them or this decision sets them.

**OD-12-4 Accessibility scan severity gate (QA-110, QA-134).**
- Options: (a) zero violations of any severity; (b) zero critical and serious violations, with minor issues tracked; (c) zero violations at the target level only, with a documented exception list.
- Default: (b) for MVP. The target conformance level itself is decided in [OD-08-1](./08-ux-accessibility.md) and is not restated here.

**OD-12-5 End-to-end suite cadence (section 6).**
- Options: (a) full E2E suite on every release candidate, subset on merges; (b) full suite nightly; (c) full suite on every merge.
- Default: (a). Cost and run time are weighed by the quality lead.

**OD-12-6 Release-blocking defect severities (QA-132).**
- Options: (a) any critical or high defect in invariant, authorization, or restore tests blocks release; (b) only critical blocks; (c) severity set per release by the release owner.
- Default: (a). Severity names and definitions are to be written into the release process.

**OD-12-7 Source of data for migration rehearsal and restore drills (QA-002, QA-076).**
- Options: (a) fully synthetic data only; (b) an anonymized copy of production data, with the anonymization method defined in the security spec; (c) a restricted production-derived copy kept under the same access controls as production.
- Default: (a) for routine CI; (b) or (c) for rehearsals only, after the method is decided. The choice affects privacy design and is an open product and design decision, not a settled requirement.

## Related docs

- [_context](./_context.md): shared terms, ID prefixes, ledger invariants.
- [01 Product vision and scope](./01-product-vision-scope.md)
- [02 Personas, roles, journeys](./02-personas-roles-journeys.md): ROL-006, ROL-052.
- [03 Domain model and rules](./03-domain-model-rules.md)
- [04 Functional requirements](./04-functional-requirements.md)
- [05 System architecture](./05-system-architecture.md): ARC-016, ARC-025.
- [06 Data, ledger, audit](./06-data-ledger-audit.md)
- [07 API and integration contract](./07-api-integration-contract.md)
- [08 UX and accessibility](./08-ux-accessibility.md)
- [09 Security, privacy, abuse](./09-security-privacy-abuse.md)
- [10 Governance and network operations](./10-governance-network-operations.md)
- [11 Deployment, operations, support](./11-deployment-operations-support.md)
- [13 Assumptions, risks, decisions, roadmap](./13-assumptions-risks-decisions-roadmap.md)
