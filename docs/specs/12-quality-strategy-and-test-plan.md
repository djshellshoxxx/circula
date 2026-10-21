# 12 — Quality Strategy and Test Plan

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | Quality lead (with Product architect) |
| Last updated | 2026-10-21 |
| Scope | Test layers, required coverage of ledger invariants and risk areas, quality gates with thresholds, definition of done, test data and environments, CI stages, requirement traceability, and the end-to-end acceptance scenarios `QA-E2E-nn` |
| Depends on | [00](00-conventions.md), [02](02-personas-roles-and-journeys.md), [03](03-domain-model-and-mutual-credit-rules.md), [04](04-functional-requirements.md), [05](05-architecture-and-trust-boundaries.md), [06](06-data-ledger-and-audit.md), [07](07-api-and-integration-contract.md), [08](08-ux-and-accessibility.md), [09](09-security-privacy-and-abuse.md), [11](11-deployment-operations-and-support.md), [15](15-lifecycle-state-machines.md), [17](17-notifications-and-communications.md), [18](18-data-export-and-portability.md) |
| Open questions | QA-OD-1 to QA-OD-10 (section 11); registered in [13](13-assumptions-risks-decisions-roadmap.md) as `DEC-nnn` with the local label as alias (canon 5.8 item 8) |

## 1. Purpose and scope

This document defines how Circula proves that it meets the other specifications. It owns the `QA` prefix: quality requirements (`QA-nnn`), quality gates (`QA-GATE-nn`) and end-to-end acceptance scenarios (`QA-E2E-nn`). It does not restate requirements; it cites them by ID and says which test technique and which gate verifies each. It is technology-neutral (canon section 4): no test framework, database, browser-automation tool or CI product is named. Where a technique needs a capability (for example "a property-based test generator" or "an automated accessibility rule engine"), the capability is specified, not the product.

The principal quality risk is an incorrect or double ledger posting under concurrency, retry or failure ([15 section 9.2](15-lifecycle-state-machines.md), [05](05-architecture-and-trust-boundaries.md) ARC-036). Test effort is weighted accordingly.

In scope: the Platform software, its migrations, backup/restore tooling, exports and the documentation-visible behaviour of notifications. Out of scope: the third-party message transport's internal behaviour, the independent security assessment's content (SEC-095 owns it; this document only gates on its completion), and legal conclusions ([19](19-legal-and-regulatory-open-questions.md)).

## 2. Definitions used here

Terms follow the [glossary](14-glossary.md) and canon 5.1. Additional local terms (not redefinitions):

| Term | Meaning in this document |
|---|---|
| **Gate** | A measurable pass/fail condition evaluated automatically in the CI pipeline (section 9) or recorded by a named role at release. |
| **Suite** | A named, versioned collection of tests (for example the *ledger invariant suite*). |
| **Race test** | A test that runs two or more competing events (rows R1 to R18 of [15 section 9.2](15-lifecycle-state-machines.md)) in parallel many times against the reference datastore and asserts the stated resolution. |
| **Oracle** | An independent, deliberately simple model (for example a reference calculator that sums postings from scratch) against which system output is compared. |
| **Synthetic dataset** | Generated, non-personal test data of a declared envelope (Pilot, Standard, Stretch per [05 section 9.1](05-architecture-and-trust-boundaries.md)). |
| **Reference datastore** | The real storage technology of the reference deployment; required for concurrency testing by DAT-040. |
| **Covers tag** | A machine-readable annotation on a test or scenario naming the requirement IDs it verifies (section 10). |

## 3. Quality principles

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-001 | P0 | The ledger invariant suite (section 5) **MUST** run on every pull request in reduced form and on every release candidate in full form, and a single failing invariant check **MUST** block merge to the release branch. | INV-001 to INV-016 are the product's core promise; ARC-036 sets zero tolerance. | CI gate QA-GATE-01 |
| QA-002 | P0 | Tests that assert concurrency, atomicity, isolation or idempotency **MUST** run against the reference datastore, not only against a mock or in-memory substitute. | Isolation behaviour is technology-specific (DAT-040). | Inspection of suite configuration; CI gate QA-GATE-05 |
| QA-003 | P0 | Every defect found in production or in the release gate that concerned a ledger result, authorization decision, money-amount parse, recovery flow or export figure **MUST** get a regression test before the fix is released. | Prevents recurrence (SEC-097). | Inspection: defect record links a test ID |
| QA-004 | P0 | Test data **MUST NOT** contain real personal data; production data **MUST NOT** be copied to a non-production environment unless irreversibly anonymised and approved as a recorded decision. | Privacy by design; SEC-100 minimisation. | Inspection; data-scan check in CI |
| QA-005 | P0 | Tests **MUST** be deterministic: time is injected (never wall-clock sleeps for correctness), randomness is seeded and the seed is printed on failure so any failure can be replayed. | Flaky tests destroy trust in the gates. | Flake-rate gate QA-GATE-12 |
| QA-006 | P1 | Each test layer **SHOULD** own the assertions it is cheapest at; a behaviour verified at a lower layer **SHOULD NOT** be re-verified through the UI except as part of a QA-E2E scenario. | Fast feedback; low maintenance. | Inspection at review |
| QA-007 | P0 | Specification text **MUST NOT** be considered implemented until each P0 requirement it contains has at least one passing automated or recorded manual test linked by a covers tag (section 10). | Traceability. | Traceability check QA-GATE-11 |

## 4. Test layers

| ID | Pri | Layer | Purpose | What it MUST include | Typical owner |
|---|---|---|---|---|---|
| QA-010 | P0 | **Unit** | Verify pure logic in isolation: amount parsing and formatting (FR-TRD-002, UX-089), limit arithmetic (DOM-010 to DOM-014), state-machine guards, policy bounds (POL-001, POL-002), authorization decision functions, hash and fingerprint functions (DAT-022, API-082). | Table-driven boundary cases for every numeric guard (min, min-1, max, max+1, exactly at floor, floor-1, at cap, overflow of the 64-bit integer type per DAT-009); parse rejection of ambiguous or non-canonical decimals. | Developers |
| QA-011 | P0 | **Property-based / invariant** | Find sequences nobody thought of. Generated random sequences of Trade, Reversal, Adjustment, departure, policy-change, expiry and cancel operations (single-threaded and concurrent) against the real ledger code, with an oracle recomputing state from postings only. | Assertions after every step: INV-001, INV-005 to INV-008, INV-011, INV-015, INV-016, limit-flag correctness (DOM-027, DAT-020), `account_seq` gapless (DAT-035). Shrinking of failing sequences to a minimal replayable case. At least the volume in QA-GATE-02. | Developers, QA |
| QA-012 | P0 | **Model-based lifecycle** | Prove that exactly the transitions of [15](15-lifecycle-state-machines.md) exist. | Every (state x event x actor class) combination for Trade, Dispute, Membership, Listing, Adjustment/Reversal; build fails if an unlisted combination succeeds (LIF-053); history record and audit written (LIF-008, LIF-021). | Developers |
| QA-013 | P0 | **Integration** | Verify components with real collaborators: application logic with the reference datastore, transactional outbox with job runner (DAT-016), idempotency store (DAT-024), reconciliation checks R1 to R14 (DAT-045), recompute and repair (DAT-042 to DAT-044), erasure (DAT-072 to DAT-079), timers (LIF-055). | Fault injection at each step of the posting unit of work (DAT-012, DAT-015) proving all-or-nothing. | Developers |
| QA-014 | P0 | **Contract / API** | Prove the HTTP contract matches the OpenAPI document (API-130): schemas, error catalogue and codes, unknown-field rejection (API-007), idempotency header rules (API-080 to API-088), pagination, least-data responses (API-008), version compatibility (API-122). | Contract tests generated from the OpenAPI document; a breaking-change detector comparing against the previous release; every operation x every documented error code exercised at least once. | Developers |
| QA-015 | P0 | **Concurrency and stress** | Show serial equivalence under parallel load (DAT-029 to DAT-041) and the race matrix. | One race test per row R1 to R18 (section 7); account-level stress with many parallel acceptances; deadlock and lock-order test (DAT-031); lock-wait timeout behaviour (DAT-033). | Developers, QA |
| QA-016 | P0 | **End-to-end (acceptance)** | Verify journeys through the real client and API with a real datastore, as a user would. | All scenarios QA-E2E-01 to QA-E2E-24 (section 12). | QA |
| QA-017 | P0 | **Accessibility** | Verify WCAG 2.2 Level AA (D-14, UX-082). | Automated rule-engine run on every screen state in CI (UX-085); manual keyboard-only and screen-reader audit of JRN-001 to JRN-009 on at least two assistive-technology/browser pairs [DECISION NEEDED: UX-OD-3, see QA-OD-5]; audit record stored with the release. Details in section 8. | QA, Design |
| QA-018 | P0 | **Security** | Verify the controls of [09](09-security-privacy-and-abuse.md): static analysis (SAST), dynamic analysis (DAST) against a running instance, dependency and licence scanning, secret scanning, authorization-matrix tests (SEC-020), cross-Network negative tests (SEC-022), rate-limit tests (SEC-055), log-canary scans (SEC-062), fuzzing of amount parsing, request parsing and recovery (SEC-097). Gates G1 to G5 in section 9.2. | Security champion, QA |
| QA-019 | P0 | **Migration** | Verify DAT-082 to DAT-086 and OPS-048, OPS-049. | Section 7.6. | Developers, Ops |
| QA-020 | P0 | **Backup-restore drill** | Verify OPS-023 to OPS-030 and DAT-051. | Section 7.7. | Ops |
| QA-021 | P0 | **Export correctness and reconciliation** | Verify [18](18-data-export-and-portability.md) and FR-EXP-004. | Section 7.8. | Developers, QA |
| QA-022 | P0 | **Performance and capacity** | Verify ARC-040 to ARC-042, ARC-046, DAT-049. | Load tests with the Standard synthetic dataset (Stretch for ARC-041), recording p95 and p99; soak test of at least [DECISION NEEDED: soak duration; recommended 4 hours] at Standard load with zero invariant violations and no resource growth trend. | Performance owner |
| QA-023 | P0 | **Chaos / failure injection** | Verify degradation and recovery (ARC-039, LIF-052, API-067, NTF-096). | Process crash between commit and hand-off; crash after commit before response; datastore failover/loss of connection mid-posting; message-transport outage during a Trade; job runner killed mid-sweep; clock skew between instances; disk-full on file store; duplicate and delayed timer runs (LIF-055). After each: reconciliation passes and no double or partial posting. | Ops, QA |
| QA-024 | P1 | **Usability** | Verify UX-112 (usability study with defined participants and measures) and the plain-language targets UX-108. | Study report stored before MVP declared complete. | Design |
| QA-025 | P1 | **Copy and compliance lint** | Verify banned terms (UX-029, DOM-001, INV-014): no "pay", "price", "charge", redemption, cash-out, interest wording on any user-visible surface, including emails and exports. | Automated lint over string catalogues, templates and export headers, plus manual review of marketing copy shipped with the Platform. | QA, Product |

## 5. Ledger invariant coverage (INV-001 to INV-016)

Every invariant has a primary technique, at least one deterministic example test, and a gate. "Property suite" is QA-011; "Race" is QA-015; "Recon" is the reconciliation check set R1 to R14 ([06](06-data-ledger-and-audit.md) DAT-045). Gate names are defined in section 9.

| ID | Primary technique | Additional techniques (minimum) | Pass condition | Gate |
|---|---|---|---|---|
| INV-001 Postings sum to zero | Property suite: every generated entry type (trade, reversal, adjustment, opening_balance, departure_settlement) checked after commit | Unit: entry builder; storage-level rejection test of an unbalanced entry (DAT-013); Recon | 0 violations across the generated volume; storage rejects an injected unbalanced entry | QA-GATE-01, QA-GATE-02 |
| INV-002 Atomic two-sided posting | Fault injection at every step of the posting unit of work (DAT-012, DAT-015): crash, timeout, constraint failure | Concurrent reader test: no reader ever observes one side only; audit-write failure aborts posting (DAT-015) | After each injected fault the ledger holds either the whole entry or nothing; readers never see a partial entry | QA-GATE-01, QA-GATE-05 |
| INV-003 Append-only (three audited exceptions) | Privilege test: application credentials attempt UPDATE/DELETE on JournalEntry and Posting and are refused (DAT-018) | API test: no operation mutates ledger (API-005); migration test (DAT-084); DR restore and disposal paths each produce an Audit Event | Zero successful mutations outside the three exceptions; each exception leaves an audit trail | QA-GATE-01, QA-GATE-08 |
| INV-004 Corrections only by Reversal/Adjustment | Model-based lifecycle (QA-012) plus API surface test: no operation edits or deletes a posted entry | Test that every correction carries `reverses_journal_entry_id`/`corrects_journal_entry_id` and an Audit Event (DAT-019); acyclicity (DAT-008) | No correction path other than the two; links present | QA-GATE-01 |
| INV-005 Floor held on Trade posting | Property suite with limits at boundaries; Race R5 | Unit boundary table (exactly at floor accepted, one minor unit below rejected); concurrent-accept stress (FR-TRD-016: 100 parallel accepts against a Recipient who can afford exactly N) | Exactly the affordable number accepted; no balance below floor; rest rejected with `credit_limit_exceeded` and Trade still `proposed` | QA-GATE-01, QA-GATE-05 |
| INV-006 Cap held on Trade posting | Property suite with `max_balance` set; Race R5 on the credit side | Boundary table; `max_balance` change vs accept race (R7) | No balance above cap after a Trade posting; reversal-caused over-cap is flagged not rejected (DOM-027) | QA-GATE-01, QA-GATE-05 |
| INV-007 Network sum zero | Property suite (after every step) and Recon on every release candidate and after every restore (DAT-051) | Per-Network isolation check (INV-011); includes System Accounts | Sum is exactly 0 for every Network in every run | QA-GATE-01, QA-GATE-09 |
| INV-008 Cached equals recomputed | Property suite oracle recomputing from postings only; recompute procedure test (DAT-042) | Injected cache corruption is detected, shown as unconfirmed balance (FR-BAL-003, UX-027), repaired with audit (DAT-044), and does not halt postings (DAT-048); `account_seq` gapless (DAT-035) | 0 mismatches; injected drift detected within one reconciliation run | QA-GATE-01, QA-GATE-09 |
| INV-009 Idempotency | Retry/idempotency suite (section 7.2): same key + same payload N times; same key + different payload | Fault injection after commit before response (API-083, API-067); concurrent duplicate submission (API-084); cross-Principal and cross-Network key isolation (API-081); late retry beyond TTL for posting operations (API-086) | Exactly one Journal Entry, Audit Event and notification regardless of retries; mismatched payload rejected with no side effects (DAT-026) | QA-GATE-01, QA-GATE-05 |
| INV-010 Audit Event for governance and corrections | Parameterised test: every Adjustment, Reversal, policy change, role change and dispute resolution path asserted to write exactly one Audit Event naming actor, time, reason and target (FR-AUD-001) | Audit failure aborts the action (DAT-015); append-only audit (FR-AUD-002, DAT-055); denied attempts logged (DAT-060) | 100% of enumerated action paths produce the event; zero without | QA-GATE-01 |
| INV-011 No cross-Network postings | Cross-Network negative test for every Network-scoped endpoint family (SEC-022); storage-level reference check (DAT-005) | Property suite with two Networks in one run: no posting or read crosses; export isolation (FR-NET-012) | 0 cross-Network references; 100% endpoint families covered | QA-GATE-01, QA-GATE-06 |
| INV-012 No admin/Operator Trade or silent value change | API/UI surface test: no operation or screen exists for admin- or Operator-initiated Trades (FR-ADM-010, UX-020, PER-028); authorization matrix cells denied (SEC-020) | Adjustment visibility test (QA-E2E-09); Operator capability test (FR-ADM-017, PER-006) | Every attempt by every role other than the Counterparty is rejected; all admin value changes are visible Adjustments | QA-GATE-01, QA-GATE-06 |
| INV-013 Unit identity immutable after first entry | Unit and integration: update path checked in same transaction (DAT-037) | Race: first Journal Entry vs `unit_scale` change (QA-E2E-24); API rejects after first entry (API `update_network_profile`) | Exactly one of the two wins; never both | QA-GATE-01, QA-GATE-05 |
| INV-014 No money-like claims | Copy and compliance lint (QA-025); feature inventory inspection | Absence tests: no transaction type, account type, endpoint or export field for redemption, cash-out, interest or conversion; DOM-015 scheduled-job inspection (no job posts balances changes) | 0 lint hits; inspection checklist signed per release | QA-GATE-10 |
| INV-015 Trade validity | Table-driven guard test: each failing guard separately (same party, different Network, non-active membership, amount 0, below min, above max, scale overflow) at proposal and at acceptance (DOM-019) | Property suite generating invalid candidates; departing exception (DOM-035) tests | Every invalid candidate rejected with the specific guard (LIF-014) and no ledger effect | QA-GATE-01 |
| INV-016 Closed account has zero balance | Closure atomicity test: `departure_settlement` entry and `closed` transition in one unit (DOM-036); Race R16 (finalize vs final Trade) | Property suite including departure sequences; no entry references a closed account; reversal of a Trade touching a closed account rejected (DOM-026) | Closed accounts always at 0 with no later entry | QA-GATE-01, QA-GATE-05 |

## 6. Required coverage by risk area

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-030 | P0 | **Concurrent transactions.** The suite **MUST** include (a) FR-TRD-016's 100-parallel-accept scenario against one Recipient, (b) parallel accepts across overlapping account pairs in opposite directions to exercise lock ordering (DAT-031), (c) parallel accepts mixed with limit and policy changes (DAT-034), (d) lock-wait timeout behaviour returning a retryable error without partial effect (DAT-033), and (e) account sequence gaplessness (DAT-035) after every run. | Serial equivalence is the central correctness claim (DAT-029). | Race and stress suites; QA-GATE-05 |
| QA-031 | P0 | **Retries and idempotency.** Tests **MUST** cover every Posting operation in API-080 and every **R**-marked operation in 07 section 6: replay returns identical status and body with `Idempotent-Replayed: true` (API-083), key reuse across operation, Principal or Network (API-081), in-progress duplicate (API-084), domain rejection not consuming the key (API-085), missing key (`missing_idempotency_key`), client clock/ID manipulation (API-088), and notification suppression on replay (NTF-043). | INV-009. | Contract and integration suites; QA-GATE-05 |
| QA-032 | P0 | **Authorization.** The permission matrix of [02 section 5](02-personas-roles-and-journeys.md) (rows PER-010 to PER-064) **MUST** be loaded as a machine-readable test table covering every role (Visitor, Applicant, Member, Dispute Reviewer, Network Administrator, Platform Operator) x every action row, allow and deny, plus the condition column (for example Counterparty-only, `balance_visibility` values, quorum-bound "Q" actions), for both the API and, where a control exists, the UI. The coverage report **MUST** show 100% of cells (SEC-020). Additional tests: object-level authorization with identifier substitution (API-015, API-016, SEC-019), "same not-found for forbidden and absent" (UX-011, DAT-062), role/suspension change effective within 60 seconds on all instances (SEC-023), single acting capacity for multi-role users (PER-004, UX-009), and reviewer limited to assigned Disputes (PER-005, FR-AUD-005). | Authorization defects are the most common serious flaw; the matrix is the contract. | Generated tests; QA-GATE-06 |
| QA-033 | P0 | **Disputes.** Tests **MUST** cover every transition DS1 to DS7 ([15](15-lifecycle-state-machines.md)); reviewer conflict exclusion and reassignment (FR-DSP-007, LIF-022); remedy atomic with resolution (LIF-021); settlement consent binding (LIF-023); remedy leaving an account out of limits flagged not blocked (FR-DSP-010, DOM-027, DAT-020); Dispute never changing balances alone (LIF-020, D-12); SLA flag and escalation (FR-DSP-012, FR-DSP-013, LIF-024); races R12 and R13. | Disputes are the legitimate route to correct the ledger. | Model-based, race, QA-E2E-08 |
| QA-034 | P0 | **Migrations.** See section 7.6. | Ledger history must survive upgrades. | QA-GATE-08 |
| QA-035 | P0 | **Accessibility (WCAG 2.2 AA).** See section 8. | D-14, UX-082. | QA-GATE-07 |
| QA-036 | P0 | **Backup restoration.** See section 7.7. | OPS-026, OPS-027. | QA-GATE-09 |
| QA-037 | P0 | **Export correctness.** See section 7.8. | FR-EXP-004, PRT-017 to PRT-021. | QA-GATE-09 |
| QA-038 | P0 | **Notifications.** Deduplication key (NTF-042), no notification on idempotent replay (NTF-043), one reminder per (object, type) (NTF-047), no duplicate content across a user's roles (NTF-054), dispatch latency (NTF-044, FR-NTF-003), delivery failure never blocking the business action (FR-NTF-008), mandatory security/ledger notices not suppressible (FR-NTF-004), no personal free text or credentials in message bodies (FR-NTF-005, NTF-068), fault-injection suite of NTF-096. | Silent or duplicated notices break the transparency promise of INV-012. | Integration and fault-injection; QA-E2E-18 |
| QA-039 | P0 | **Timers.** Every timer in [15 section 10](15-lifecycle-state-machines.md) **MUST** be tested for idempotency, concurrent execution on multiple workers, late execution and duplicate execution (LIF-055), actor `system` with reason `timer:<name>` (LIF-056), and missed-run catch-up (LIF-015). | Expiry and reminders run unattended. | Integration; QA-GATE-05 |

## 7. Specialised test designs

### 7.1 Race tests (the race gate)

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-040 | P0 | Each row R1 to R18 of [15 section 9.2](15-lifecycle-state-machines.md) **MUST** have an automated race test running the competing events in parallel at least [DECISION NEEDED: iterations per race test; LIF-051 recommends 1,000; see QA-OD-2] times with randomised start offsets, asserting the stated resolution, a single winner, `illegal_transition` (or the row-specific outcome) for the loser, no partial effects, and INV-001 and INV-007 afterwards. | LIF-051 is owned by doc 15; this requirement makes it a release gate. | QA-GATE-05 |
| QA-041 | P0 | Race tests **MUST** be run on at least two instances of the application against one datastore (to defeat in-process locking masking a defect), and **MUST** include a run with artificially widened windows (injected delays between read and write) in addition to natural timing. | Single-process runs hide cross-instance races. | Suite configuration inspection |
| QA-042 | P0 | A race test failing once in the full iteration count **MUST** fail the gate; it **MUST NOT** be retried until green. The failing seed and interleaving **MUST** be stored with the build. | A rare ledger race is a Sev-1 in production. | CI gate rule |
| QA-043 | P0 | The model-based test (LIF-053) and the timer-safety test (LIF-055) **MUST** be part of the same race gate. | They are the cheapest net for illegal transitions and timer duplicates. | QA-GATE-05 |

Note on gate references: doc 09 and doc 15 are expected to reference race-test gates by ID. At the time of writing, doc 15 defines the race requirement as LIF-051 (with LIF-052, LIF-053 and LIF-055 as companions) and does not define the further race-gate IDs the brief for this document expected (numbers above 057); see QA-OD-1. Until reconciled, the race gate in this document (QA-GATE-05) cites LIF-051, LIF-052, LIF-053 and LIF-055.

### 7.2 Retry and idempotency matrix

For each Posting operation (`accept_trade`, `create_adjustment`, `approve_adjustment`, `accept_reversal`, `approve_reversal`) and a representative **R**-operation that does not post (for example creating a Trade proposal):

| Case | Expected result | Cites |
|---|---|---|
| Same key, same payload, 50 sequential retries | One Journal Entry, one Audit Event, one notification set; identical response body | API-083, INV-009, NTF-043 |
| Same key, same payload, 50 concurrent | One executes; others `idempotency_in_progress` (409, `Retry-After`) or the replayed result; never two postings | API-084 |
| Same key, different amount / counterparty / description | `idempotency_conflict` (422), no side effects | API-082, DAT-026 |
| Same key, different operation or path | `idempotency_conflict` | API-081 |
| Same key from another Principal or Network | Treated as unrelated | API-081 |
| Crash after commit before response, then retry | Original outcome returned; no second posting | API-083, API-067, DAT-024 |
| Rejection by a domain guard, then retry after fixing cause | Evaluated fresh | API-085 |
| Retry long after the 24-hour window for a Posting operation | Still no double post | API-086 |
| Missing key on an **R** operation | `missing_idempotency_key`, nothing executed | API-080 |
| Balance changed between first attempt and replay | Replay returns stored outcome without re-evaluating guards | DAT-025 |

### 7.3 Concurrency and stress

| Case | Setup | Assertion |
|---|---|---|
| Same-account accepts | Recipient with limit allowing exactly N Trades; 100 `proposed` Trades; accept all in parallel | Exactly N `completed`; 100-N remain `proposed` with specific guard error; balance equals floor; INV-005, INV-007, INV-008 hold (FR-TRD-016, DOM-020) |
| Opposite-direction pairs | A to B and B to A Trades accepted in parallel, many iterations | No deadlock; every Trade terminal or retryable; bounded wait (DAT-031, DAT-033) |
| Limit change vs accept | Admin lowers `credit_limit` while accepts run | Result equals some serial order (DAT-034, R7) |
| Departure vs accept | Member enters `departing` during accept (R9); finalize vs final Trade (R16) | One wins; INV-016, INV-015 hold |
| Quorum approvals | Last two approvals of an Adjustment in parallel (R14) | Exactly one posting |
| Mixed soak | Random mix of all operations at Standard load for the soak duration | Zero invariant violations; reconciliation passes at end (DAT-045) |

### 7.4 Property-based test design

1. The generator produces sequences of operations drawn from the legal and illegal event set of each machine, with parameters biased toward boundaries (amounts at limits, zero, one minor unit above or below, maximum 64-bit values).
2. Execution is run (a) sequentially, (b) with 2 to 16 concurrent workers, (c) with random fault injection (abort a unit of work, restart an instance).
3. The oracle holds a per-Network model: balances, limits, states. After each step the system's visible state must equal the oracle's, all INV checks run, and every response must be one of the documented outcomes.
4. On failure the sequence is shrunk to a minimal case and stored as a permanent regression example (QA-003).

### 7.5 Dispute and correction tests

Beyond QA-033: reversal of a Trade whose Provider has since spent the credit leaves Provider over limit with flag `limit_exceeded_by_reversal` and the account still creditable but not debitable (DOM-012, DOM-027, R11); a second Reversal request for the same Trade fails (R10, DOM-025); Reversal rejected when either account is `closed` (DOM-026); Adjustment targeting the approving admin's own account refused (DOM-032, FR-ADM-008, FR-ADM-015); override reason mandatory when outside limits (DOM-033).

### 7.6 Migration tests

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-050 | P0 | Every migration **MUST** be exercised on a copy of a Standard-envelope synthetic dataset (and on a snapshot restored from the previous release) before release, recording duration, locking behaviour and the stated rollback strategy (DAT-083, DAT-086). | Rehearsal catches data-dependent failures. | Migration job record; QA-GATE-08 |
| QA-051 | P0 | After every migration touching ledger-related tables, reconciliation R1 to R14 **MUST** pass and the pre- and post-migration ledger digests (entry hashes, IntegrityCheckpoint) **MUST** be identical (DAT-084, DAT-085, DAT-022). | Migrations must not change ledger values. | QA-GATE-08 |
| QA-052 | P0 | Migrations **MUST** be tested for interruption and re-run (idempotent, resumable) and for expand/contract compatibility: release N-1 application code against release N schema, and release N code against N-1 schema during the transition (OPS-048). | Rolling deployments and aborted upgrades. | QA-GATE-08 |
| QA-053 | P0 | A test **MUST** assert that no migration step updates or deletes a Journal Entry or Posting value, changes an identifier or alters an `entry_hash` input (static scan of migration scripts plus runtime privilege check). | INV-003 versioned-migration exception is narrowly bounded. | QA-GATE-08 |
| QA-054 | P0 | Upgrade followed by rollback to the previous release (or documented roll-forward) **MUST** be tested at gate G4 (SEC-094 table) and a restore from an N-1 backup into release N **MUST** pass (OPS-034). | Release rollback must be real. | QA-GATE-08 |

### 7.7 Backup and restore drill tests

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-060 | P0 | The drill procedure **MUST** implement the five pass criteria of OPS-027 as machine-checked results: restore within tier RTO; application start and smoke test; reconciliation with zero findings; row counts and latest Journal Entry identifier equal at the backup point; keys recovered by a custodian other than the configurer. | Objective pass/fail. | Drill automation output |
| QA-061 | P0 | At each release candidate, a drill **MUST** restore the latest backup of a Standard-envelope staging deployment into an isolated scratch environment and pass QA-060. Hosted production drills run at least every 7 days and self-hosted at least every 90 days [DECISION NEEDED: OPS-OD-5, see QA-OD-6]. | OPS-026. | QA-GATE-09 |
| QA-062 | P0 | The drill **MUST** include point-in-time restore to a chosen timestamp within the PITR window (OPS-028) and verify that Trades accepted after the target are absent, and that no partial entry exists. | Recovery from logical corruption. | Drill record |
| QA-063 | P0 | After restore the application **MUST NOT** accept writes until reconciliation passes, and an Audit Event of the restore **MUST** exist (OPS-030, DAT-051); the drill **MUST** assert write rejection during the check. | Prevents compounding corruption. | Drill assertion |
| QA-064 | P0 | The drill **MUST** replay the erasure and anonymisation register after restore and assert that members erased after the backup point are anonymised again (OPS-029, DAT-079); and that credentials/session material is not restored from a Network archive (PRT-044). | Erasure must survive restore. | Drill assertion |
| QA-065 | P0 | A drill **MUST** also restore with a deliberately injected failure (corrupted backup segment, wrong key) and prove that the failure is detected, alarms and does not leave a half-restored environment reachable (OPS-032). | A detected failure is better than a silent one. | Negative drill |
| QA-066 | P1 | Measured RPO and RTO per drill **SHOULD** be recorded and compared with the tier targets: hosted at most 5 minutes / 4 hours; self-host recommended 15 minutes / 8 hours; self-host minimum 24 hours / 24 hours [DECISION NEEDED: OPS-OD-3]. | Trend visibility. | Drill record |

### 7.8 Export correctness and reconciliation tests

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-070 | P0 | An independent verifier (separate from export code, implementing the normative algorithm of [18](18-data-export-and-portability.md) PRT-021) **MUST** be run on every export type produced in the release suite: it recomputes entry counts, per-entry zero sum, per-account balances, `ledger_total` and checksums from the package alone and compares them to the manifest (FR-EXP-004, PRT-017, PRT-018, PRT-021). | Exports must be self-verifying by third parties. | QA-GATE-09 |
| QA-071 | P0 | Export figures **MUST** equal the live ledger for the same as-of snapshot: sum of every member's exported balance equals zero including System Accounts; each member statement satisfies opening + entries = closing (FR-BAL-011, DAT-052); no amount is a float, carries a currency symbol or is locale-formatted (PRT-008). | Reconciliation. | Automated comparison |
| QA-072 | P0 | Exports **MUST** be consistent under concurrent postings (single consistent snapshot, PRT-010) and **MUST NOT** block postings; verified by running an export while the mixed soak workload runs (ARC-046). | Export of a live Network. | Integration |
| QA-073 | P0 | Visibility and anonymisation parity: the export of every dataset **MUST** contain only what the requester can see in the UI/API (PRT-036, FR-EXP-008, FR-EXP-005); `redaction` counts are correct (PRT-040); closed-Member and erased-Member projections match UI. | No data leakage via export. | Differential test: API vs export |
| QA-074 | P0 | CSV injection: cells beginning with `=`, `+`, `-`, `@`, tab or CR **MUST** be neutralised (PRT-009); reproducibility: same scope and as-of yields byte-identical content apart from the generation timestamp (FR-EXP-007, PRT-011). | Safe for spreadsheet users. | Unit and golden-file |
| QA-075 | P1 | Round trip: a verified `network_archive` **SHOULD** import into an empty deployment with identical balances verified by reconciliation (PRT-043, OPS-006, OPS-033). | Portability proof. | Integration |
| QA-076 | P0 | Export access control and audit: only the roles and quorum of the permission matrix may export (PER-032, PER-033, SEC-105); each export lifecycle event creates an Audit Event (PRT-032). | Governance of bulk data. | Authorization matrix; audit test |

### 7.9 Security testing

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-080 | P0 | SAST **MUST** run on every pull request and fail on new high-severity findings not triaged with a recorded justification (SEC-094 gate G2). | Early defects. | CI |
| QA-081 | P0 | DAST (authenticated and unauthenticated, per role) **MUST** run against a deployed staging instance on every release candidate; critical and high findings block release without a recorded exception (G3, G4). | Runtime flaws. | CI/release |
| QA-082 | P0 | Dependency (known-vulnerability and licence) and secret scanning **MUST** run on every pull request and nightly on the default branch (G2, SEC-092, SEC-072). | Supply chain. | CI |
| QA-083 | P0 | The authorization-matrix suite (QA-032), cross-Network negative tests (SEC-022), rate-limit tests (SEC-055, API-111 returning `rate_limited` before any state change), log-canary scan (SEC-062: seeded fake credentials, tokens, emails and free text must not appear in logs), CSRF checks on cookie-authenticated mutations (API-012) and session behaviour (SEC-012 to SEC-015) **MUST** pass at gate G3. | Defence in depth. | CI |
| QA-084 | P0 | Recovery and enumeration tests: registered and unregistered emails produce identical content, status and (within measurement tolerance) timing (SEC-026, SEC-056); token single-use, expiry and no automatic sign-in (SEC-027, SEC-028); cool-off and restricted period (SEC-029, SEC-030); notices sent (SEC-031). | Account-takeover resistance. | Integration and timing test |
| QA-085 | P1 | Fuzzing of money-amount parsing, API request parsing and recovery flows **SHOULD** run in CI or nightly; every finding becomes a regression test (SEC-097). | Parser defects are classic. | Nightly job |
| QA-086 | P0 | The ASVS L2 checklist delta is signed off and the independent assessment (SEC-095) is completed before first hosted production launch; unresolved critical/high findings block launch. | G4, G5. | Release record |

## 8. Accessibility testing (WCAG 2.2 AA)

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-090 | P0 | An automated accessibility rule engine **MUST** run in CI on every screen (SCR-01 to SCR-30) in every state of UX-013 (loading, empty, error, forbidden) and at 320 CSS px and desktop widths; any new violation of a WCAG 2.2 A or AA rule **MUST** fail the build (UX-085). | Prevents regression. | QA-GATE-07 |
| QA-091 | P0 | A manual keyboard-only audit **MUST** be performed at each release candidate for journeys JRN-001 to JRN-009: no keyboard trap, logical focus order, visible focus with 3:1 contrast, skip link, focus moved to heading on route change, dialog focus management (UX-054 to UX-058). | Automation covers only part of WCAG. | Audit record stored with release |
| QA-092 | P0 | A manual screen-reader audit **MUST** be performed on at least two assistive-technology/browser pairs [DECISION NEEDED: UX-OD-3; QA-OD-5] covering the Trade proposal and acceptance flow (UX-037 to UX-044), history table amount cells ("Spent -12.00 hours", UX-063), balance panel (UX-022 to UX-028), live-region announcements and assertive alerts (UX-068, UX-069), form error summary and field mapping (UX-072 to UX-074), and session-expiry and step-up dialogs (UX-101, UX-103). | Real assistive-technology behaviour. | Audit record |
| QA-093 | P0 | Visual criteria **MUST** be verified: contrast 4.5:1 text / 3:1 large text and components (UX-066), colour not the sole signal (UX-067, FR-BAL-010), reflow at 320 px with no horizontal scroll and no loss of function (UX-046, UX-048), target size at least 24 x 24 CSS px (UX-049), primary action never obscured (UX-051), text spacing and zoom to 200%. | WCAG 2.2 AA specifics. | Automated plus manual |
| QA-094 | P0 | Content criteria **MUST** be verified: unique page titles "Screen name - Network name" and one level-one heading (UX-008), reading-level target of about grade 8 measured by a readability tool on member-facing strings (UX-108), no reliance on drag, path-based or timed gestures. | Cognitive accessibility. | Automated lint plus review |
| QA-095 | P0 | An accessibility conformance report (criterion-by-criterion, with failures and remediation) **MUST** be stored per release (UX-082(c)); a release with an open WCAG 2.2 AA failure on a Trade, balance, sign-in, recovery or departure screen is blocked. | Evidence. | QA-GATE-07 |
| QA-096 | P1 | Testers with disabilities or experienced assistive-technology users **SHOULD** take part in at least one pre-release audit per year; the accessibility statement is kept current (UX-084). | Real-world validity. | Audit record |

## 9. Quality gates and CI pipeline

### 9.1 Quality gates with measurable thresholds

| Gate | Pri | What is measured | Threshold | Blocks | Source |
|---|---|---|---|---|---|
| QA-GATE-01 Ledger invariant suite | P0 | Deterministic invariant tests for INV-001 to INV-016 (section 5) | 100% pass; 0 violations. Reduced form (seeded subset) on each PR; full form on release candidate | PR merge (reduced), release (full) | ARC-036, QA-001 |
| QA-GATE-02 Property-based volume | P0 | Generated operations with oracle checking, sequential and concurrent | 0 violations in at least 100,000 generated operations per release candidate (ARC-036); at least 5,000 per PR [DECISION NEEDED: QA-OD-3] | Release | ARC-036 |
| QA-GATE-03 Unit/integration coverage | P1 | Line and branch coverage of ledger posting, limit arithmetic, authorization policy, idempotency and lifecycle modules | Branch coverage at least 95% on those modules; at least 80% overall [DECISION NEEDED: QA-OD-4]. Coverage is a floor, not the objective (QA-006) | PR merge | QA-OD-4 |
| QA-GATE-04 Contract conformance | P0 | Every operation x documented error code exercised; request/response schemas validated; breaking-change detector clean unless a major version is declared | 100% operations; 0 undeclared breaking changes (API-122) | PR merge | API-130 |
| QA-GATE-05 Race and concurrency gate | P0 | Race tests R1 to R18 (QA-040), model-based combinations (LIF-053), timer safety (LIF-055), 100-parallel accept (FR-TRD-016), retry/idempotency matrix (section 7.2), fault injection (LIF-052) | 100% pass at every iteration; iterations per race test at least 1,000 [DECISION NEEDED: QA-OD-2, LIF-051]; two application instances; zero retries-to-green | Release candidate; reduced iteration count (for example 100) on PR for touched areas | LIF-051, LIF-052, LIF-053, LIF-055, DAT-040 |
| QA-GATE-06 Authorization and isolation | P0 | Permission-matrix cells (SEC-020), cross-Network negative tests per endpoint family (SEC-022), object-level tests (API-015, API-016) | 100% of matrix cells (allow and deny); 100% of Network-scoped endpoint families; 0 failures | Release candidate (G3) | SEC-020, SEC-022 |
| QA-GATE-07 Accessibility | P0 | Automated rule engine on all screen states; manual keyboard and screen-reader audit; conformance report | 0 new A/AA violations in CI; 0 open AA failures on core screens at release; audits for JRN-001 to JRN-009 complete | PR merge (automated), release (manual) | UX-082, UX-085 |
| QA-GATE-08 Migration | P0 | Rehearsal, reconciliation, digest equality, N-1/N compatibility, rollback | All pass; ledger digests identical; reconciliation 0 findings | Release candidate containing a migration | DAT-082 to DAT-086, OPS-048, OPS-049 |
| QA-GATE-09 Recovery and export integrity | P0 | Restore drill (OPS-027 five criteria), reconciliation R1 to R14 (DAT-045), export verifier (QA-070) | Drill passes within tier RTO; reconciliation 0 findings (Critical and non-Critical); verifier 0 mismatches; reconciliation of 1,000,000 Postings within 10 minutes [DECISION NEEDED: DEC-DAT-10] | Release candidate | OPS-026, OPS-027, DAT-049, DAT-051, FR-EXP-004 |
| QA-GATE-10 Copy and non-monetary | P0 | Banned-term lint across UI strings, templates, exports and API docs; absence-of-feature inspection | 0 hits; checklist signed | Release candidate | DOM-001, INV-014, UX-029 |
| QA-GATE-11 Traceability | P0 | Every P0 and P1 `FR-*` requirement has at least one covers-tagged passing test or recorded manual test (section 10) | 100% of P0; 100% of P1 or a recorded deferral decision; 0 dangling IDs | PR merge (check), release (report) | QA-007, 00 section 2.2 |
| QA-GATE-12 Test health | P0 | Flaky-test rate (tests with non-deterministic results over 20 reruns) | 0 flaky tests in QA-GATE-01 and QA-GATE-05 suites; at most 1% elsewhere, quarantined with owner and date [DECISION NEEDED: QA-OD-4] | Release | QA-005 |
| QA-GATE-13 Performance | P0 | Standard-envelope load test | Trade acceptance server time at most 300 ms p95 and 1 s p99; balance and recent-history read at most 150 ms p95; directory search at most 500 ms p95; sustain at least 50 acceptances/s per deployment with 0 limit violations [DECISION NEEDED: ARC-OD-1]; full Network export within 15 minutes and member export within 2 minutes without blocking postings (ARC-046) | Release candidate | ARC-040, ARC-046 |
| QA-GATE-14 Stretch performance | P1 | Stretch-envelope load test | Same targets hold, or measured gap recorded in the roadmap | Report | ARC-041 |
| QA-GATE-15 Client performance | P1 | Mid-range phone, throttled connection reference profile | Largest contentful paint at most 2.5 s at p75 on main member screens | Report | ARC-042 |
| QA-GATE-16 Notification SLO | P0 | Dispatch latency in a staged run | In-app record present and email handed to transport within 60 seconds of commit for 95% of events [DECISION NEEDED: FR-OD-14] | Release candidate | NTF-044, FR-NTF-003 |
| QA-GATE-17 Chaos | P0 | Failure-injection suite (QA-023) | After every injected failure: reconciliation 0 findings; no duplicate or partial posting; degradation as ARC-039 | Release candidate | ARC-039, NTF-096 |
| QA-GATE-18 Defect exit | P0 | Open defects by severity at release | 0 open Sev-1/Sev-2 ledger, security or data-loss defects; no open Sev-3 without a recorded decision | Release | QA-003 |

### 9.2 Security gates G1 to G5 (from doc 09, SEC-094)

Doc 09 defines the secure development lifecycle gates; this document records them as pipeline stages and names the evidence stored. They are verbatim in intent; doc 09 owns the definition.

| Gate | When | Required test evidence (this document) | Pipeline stage (9.3) |
|---|---|---|---|
| G1 Design | Any feature touching roles, data flows, credentials or exports | Threat-model delta and privacy-inventory delta (SEC-099) reviewed; decision recorded if a Sensitive action is added; test plan lists new authorization-matrix rows and abuse cases before implementation | Design review (not automated); recorded in the change |
| G2 Pull request | Every PR | Review per SEC-093 for auth, authorization, ledger, recovery, export, logging code; SAST clean or triaged; secret scan; dependency scan; unit tests for authorization decisions | Stage 2 |
| G3 Merge to release branch | Every release candidate | Full authorization-matrix test (SEC-020); cross-Network negative tests (SEC-022); log-canary scan (SEC-062); rate-limit tests (SEC-055); ledger invariant suite (QA-GATE-01) | Stages 4 to 6 |
| G4 Release | Every release | SBOM and provenance if adopted; ASVS L2 checklist delta signed off (SEC-115); no open critical/high without recorded exception; upgrade and rollback tested (QA-054) | Stages 8 to 10 |
| G5 Pre-launch and yearly | First hosted launch, then annually or after major change | Independent assessment (SEC-095); restore drill (QA-061); incident tabletop; secret-rotation exercise (SEC-073) | Out-of-band; recorded in the release record |

### 9.3 CI pipeline stages

| Stage | Name | Contents | Typical trigger | Fails the build when |
|---|---|---|---|---|
| 1 | Static checks | Format and lint, copy lint (QA-025), OpenAPI validity (API-130), traceability check (QA-GATE-11), migration static scan (QA-053) | Every push | Any finding |
| 2 | Fast tests and security-on-PR | Unit tests, model-based lifecycle (reduced), SAST, secret scan, dependency scan (G2) | Every PR | Any failure; new high findings |
| 3 | Integration and contract | Integration suite on reference datastore, contract tests, reduced invariant suite and reduced race iterations (QA-GATE-01, 04, 05 reduced) | Every PR | Any failure |
| 4 | Full invariant and property run | Full property volume (QA-GATE-02), full invariant suite | Release candidate; nightly | Any violation |
| 5 | Race, concurrency and chaos | Full race iterations on two instances (QA-GATE-05), failure injection (QA-GATE-17), timer duplication | Release candidate; nightly | Any failure |
| 6 | Security and authorization | Authorization matrix, cross-Network, rate limits, log canaries, DAST (G3) | Release candidate | Any failure; critical/high finding |
| 7 | Accessibility | Automated engine on all screen states (every PR for changed screens, all screens at RC) | PR; RC | New violation |
| 8 | Migration and recovery | Migration rehearsal and reconciliation (QA-GATE-08), restore drill (QA-GATE-09), export verification (QA-070) | Release candidate | Any failure |
| 9 | End-to-end | All QA-E2E scenarios against a staging deployment built from the release artifact (OPS-009) | Release candidate | Any scenario failure |
| 10 | Performance and soak | Load, Stretch (report), soak, notification SLO (QA-GATE-13, 14, 15, 16) | Release candidate; weekly | Threshold missed |
| 11 | Release record | Manual audits (accessibility, usability), conformance report, ASVS delta, exceptions list, traceability report, defect exit (QA-GATE-18) | Release | Missing evidence |

The artifact tested in stages 4 to 10 **MUST** be byte-identical to the artifact deployed (OPS-009). Gates evaluate the same suite versions on retries; re-running to obtain a green result is prohibited for QA-GATE-01 and QA-GATE-05 (QA-042).

## 10. Traceability approach

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-100 | P0 | Every test, scenario and manual audit step **MUST** carry a covers tag listing the requirement IDs it verifies (for example `covers: FR-TRD-016, INV-005, DAT-029`). Tags are machine-readable and live next to the test. | Tool-checkable linkage. | QA-GATE-11 |
| QA-101 | P0 | The traceability tool **MUST** (a) parse all spec documents for requirement IDs and priorities, (b) parse all covers tags, (c) fail when a P0 or P1 `FR-*` requirement has no covering test, (d) fail on any tag or document reference to an ID that does not exist (00 section 2.2 rule 2), (e) report Withdrawn IDs still tagged, and (f) emit a matrix (requirement x test x layer x last result) stored with the release. | Dangling references and uncovered requirements are release defects. | Tool run; negative self-test (seeded gap makes the tool fail) |
| QA-102 | P0 | Every P0 and P1 `FR-*` requirement in [04](04-functional-requirements.md) **MUST** be covered by at least one test at a layer that can actually observe the behaviour; the given/when/then cell in the requirement row is the minimum scenario. P2 requirements need design-for evidence only (for example the PRT-052 hook attributes exist). | Requirement-to-test completeness. | QA-GATE-11 |
| QA-103 | P0 | Every INV (section 5), every `LIF` race row R1 to R18 and every cell of the permission matrix **MUST** map to a test ID in the matrix; the matrix **MUST** be generated, not hand-maintained. | The three highest-risk catalogues. | Generated report |
| QA-104 | P0 | A requirement whose verification is Inspection or Demonstration **MUST** have a recorded checklist item (reviewer, date, release) rather than a test; the tool treats it as covered only if the record exists for the current release. | Not every requirement is automatable. | Release record |
| QA-105 | P1 | Persona and journey coverage: each persona ([02](02-personas-roles-and-journeys.md), PER-001) is traceable to at least one QA-E2E scenario and each journey JRN-001 to JRN-011 to at least one scenario or manual audit; the table in section 12.2 is the starting map. | PER-001. | Inspection |
| QA-106 | P0 | A requirement change (new, changed or withdrawn ID) **MUST** be accompanied in the same change set by test updates or an explicit "no test impact" note; the tool flags changed requirement text whose covering tests did not change. | Prevents drift. | CI check on spec changes |

## 11. Test data, environments, and definition of done

### 11.1 Test data

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| QA-110 | P0 | The project **MUST** provide a synthetic-data generator producing Networks at Pilot, Standard and Stretch envelopes ([05 section 9.1](05-architecture-and-trust-boundaries.md)) with realistic distributions (a few very active Members, a long tail, some at floor and cap, closed Members, departing Members, disputed Trades, Reversals and Adjustments), seeded and reproducible (QA-005). | Realistic load and edge shapes. | Generator self-test; OPS-012 |
| QA-111 | P0 | Fixtures **MUST** include at least two Networks with different `unit_scale` (0, 2, 4), `membership_mode`, `balance_visibility` and `admin_quorum` (1, 2, 3) values, and Members holding Memberships in both Networks, to exercise isolation and quorum. | Policy variants are the common source of defects. | Fixture inspection |
| QA-112 | P0 | Fixtures **MUST** contain adversarial inputs: names and descriptions with RTL text, combining characters, very long strings, markup, CSV-formula prefixes, and identifiers that look like other Networks' identifiers. | Injection and rendering defects. | Suite inspection |
| QA-113 | P0 | Test personas **MUST** cover each role and state: Visitor, Applicant, Member, Member with suspended / departing / closed Membership, Dispute Reviewer, Network Administrator (alone, and one of several), Platform Operator, and one person in two Networks. | Matrix coverage. | Fixture inspection |
| QA-114 | P0 | Time **MUST** be controllable in tests (injected clock) so expiry, notice periods, dispute windows, cool-offs and reminder timers are testable without real waiting (LIF-007). | Determinism. | Suite design inspection |
| QA-115 | P0 | A message-transport test double **MUST** record every handed-over message with its deduplication key and support injected outage, bounce and duplicate delivery. | Notification tests. | NTF-004 interface |

### 11.2 Environments

| Environment | Purpose | Data | Required properties |
|---|---|---|---|
| Developer / ephemeral | Unit, integration, contract, model-based, property tests | Synthetic, small | Reference datastore for concurrency suites (QA-002); disposable per run |
| CI | Stages 1 to 5 | Synthetic | At least two application instances against one datastore for race tests (QA-041) |
| Staging | End-to-end, performance, DAST, migration rehearsal, restore drills | Synthetic at Standard (OPS-012) | Same artifact and topology class as production, configuration differences only (OPS-009); message-transport double or sandbox |
| Scratch (restore) | Restore drills, archive import (OPS-013) | Restored backup | Isolated from production network; credentials separated (OPS-032) |
| Production | Reconciliation job, backup freshness, status view, canary checks | Real | No tests that mutate member data; only read-only synthetic probes in a dedicated test Network if adopted [DECISION NEEDED: QA-OD-8] |

### 11.3 Definition of done

| ID | Pri | A change is Done only when | Verification |
|---|---|---|---|
| QA-120 | P0 | Code reviewed (SEC-093 code-owner review where required); all PR-stage gates green; covers tags present for every requirement touched; new behaviour has tests at the cheapest sufficient layer and, if user-visible, a QA-E2E scenario or an update to one. | PR checklist |
| QA-121 | P0 | Any change touching ledger posting, limits, idempotency, lifecycle, authorization, recovery, export or migration additionally has: property-suite coverage or a stated reason, a race test if it adds or alters a shared-state interaction, and an authorization-matrix row update. | PR checklist |
| QA-122 | P0 | UI changes pass the automated accessibility engine for all affected states, have been keyboard-tested by the author, copy passes the lint, and strings are externalised (D-13). | PR checklist |
| QA-123 | P0 | Specification impact handled: the owning document updated (or "no spec impact" stated), new IDs registered, no dangling references (00 section 2.2). | Traceability check |
| QA-124 | P0 | Observability: new failure modes emit structured logs without personal data (SEC-062) and, where operator action is needed, an alert with an owner (OPS-036). | Review |
| QA-125 | P0 | Documentation and release notes updated, including migrations and rollback strategy (OPS-047, DAT-083). | Release checklist |
| QA-126 | P0 | A release is Done when all P0 gates in section 9.1 are green on the release candidate, the release record (section 9.3 stage 11) is complete, and every open exception has a named owner, justification and expiry. | Release record |

## 12. End-to-end acceptance scenarios

### 12.1 Conventions

- Scenarios run through the real client and API against a staging deployment built from the release artifact, with the injected clock (QA-114). They additionally assert, after the final step, that reconciliation passes with zero findings and the Network's balances sum to zero (INV-007) unless stated otherwise.
- Actors: Alice, Bob, Chidi, Dana are Members; Nia is a Network Administrator; Omar is a second Network Administrator; Rae is a Dispute Reviewer; Olu is the Platform Operator. "Net1" and "Net2" are separate Networks. Amounts use `unit_scale = 2` unless stated.
- Every scenario is P0 unless marked P1. Each cites only IDs that exist in the specification set at the time of writing. Cited IDs are covers tags (QA-100).
- Assertions about copy refer to the plain-language labels of UX-034 and the banned-term list (UX-029).

### 12.2 Scenario index and journey map

| ID | Title | Pri | Journeys / personas touched |
|---|---|---|---|
| QA-E2E-01 | Join and first trade | P0 | JRN-001, JRN-002 (Applicant to Member) |
| QA-E2E-02 | Credit-limit rejection and recovery | P0 | Member |
| QA-E2E-03 | Concurrent accepts on the same account | P0 | Member, system |
| QA-E2E-04 | Retry with the same idempotency key | P0 | Member, client |
| QA-E2E-05 | Same key, changed payload rejected | P0 | Member, client |
| QA-E2E-06 | Expiry race at the boundary | P0 | Member, system |
| QA-E2E-07 | Cancel versus accept race | P0 | Member |
| QA-E2E-08 | Dispute upheld to Reversal with limit-breach flag | P0 | Member, Reviewer, Administrator |
| QA-E2E-09 | Adjustment visibility | P0 | Administrator, Member |
| QA-E2E-10 | Departure with negative balance | P0 | Member, Administrator |
| QA-E2E-11 | Admin quorum | P0 | Administrator |
| QA-E2E-12 | Policy change with notice | P0 | Administrator, Member |
| QA-E2E-13 | Export reconciliation | P0 | Administrator, Member |
| QA-E2E-14 | Restore drill | P0 | Platform Operator |
| QA-E2E-15 | Account recovery flow | P0 | Member, Visitor |
| QA-E2E-16 | Cross-Network isolation | P0 | Member in two Networks, Administrator |
| QA-E2E-17 | Keyboard-only trade with screen-reader checks | P0 | Member using assistive technology |
| QA-E2E-18 | Notification deduplication | P0 | Member, Reviewer |
| QA-E2E-19 | Unknown outcome after timeout | P0 | Member, client |
| QA-E2E-20 | Suspension versus accept race | P0 | Member, Administrator |
| QA-E2E-21 | Over-limit account after limit lowered | P0 | Administrator, Member |
| QA-E2E-22 | Operator has no ledger or content power; break-glass | P0 | Platform Operator |
| QA-E2E-23 | Migration preserves the ledger | P0 | Platform Operator |
| QA-E2E-24 | Unit identity locked by first entry | P0 | Administrator |

### 12.3 Scenarios

**QA-E2E-01 — Join and first trade**
Cites: FR-MEM-001, FR-MEM-002, FR-MEM-003, FR-MEM-005, FR-MEM-006, FR-LST-001, FR-LST-003, FR-LST-005, FR-TRD-001, FR-TRD-002, FR-TRD-004, FR-TRD-006, FR-TRD-007, FR-TRD-022, FR-BAL-001, FR-BAL-004, INV-001, INV-002, INV-007, INV-015, DOM-023, UX-037, UX-038, API-080.
- Given Net1 in `approval` mode with `default_credit_limit` 50.00, `unit_name` "hours", and Bob already an active Member with a published Offer.
- When Alice applies with email and the Network's application fields and accepts the rules text, verifies her email, and Nia approves her application with a reason
- Then Alice has exactly one Member Account at balance 0.00 with credit limit 50.00 and Bob and Nia receive nothing they should not, and Alice sees her application status and then "active".
- When Alice finds Bob's Offer in the directory and proposes a Trade as Recipient, amount 12.00, and sees the Review step stating in words who is credited and debited and the resulting balance before sending
- Then the Trade is `proposed`, no balance has changed, `available_to_spend` is unchanged, and Bob is notified.
- When Bob accepts
- Then the Trade is `completed` in one Journal Entry with two Postings (Alice −12.00, Bob +12.00); Alice's balance is −12.00 shown with the words "You have received 12.00 hours more than you have provided" and not styled as an error; Bob's is +12.00; both histories show the entry; both are notified; the Network sum is 0.

**QA-E2E-02 — Credit-limit rejection and recovery**
Cites: FR-TRD-003, FR-TRD-007, FR-TRD-008, FR-TRD-009, FR-BAL-009, INV-005, DOM-019, DOM-014, LIF-014, UX-004, UX-025, UX-079.
- Given Alice's balance is −45.00, credit limit 50.00, and Bob has proposed to Alice (Bob as Provider) a Trade of 10.00
- When Alice accepts
- Then the acceptance is rejected with the specific limit reason (`credit_limit_exceeded`), the message states the largest amount Alice could accept (5.00) without internal identifiers, the Trade remains `proposed`, balances are unchanged, and the Audit/history shows no posting.
- When Alice receives 20.00 from Chidi (Chidi accepts a proposal by Alice as Provider) and then accepts Bob's Trade
- Then it completes and Alice's balance is −35.00, never below −50.00 at any observable point.

**QA-E2E-03 — Concurrent accepts on the same account**
Cites: FR-TRD-016, FR-TRD-007, FR-TRD-008, INV-005, INV-007, INV-008, DOM-020, DAT-029, DAT-030, DAT-035, LIF-051, R5.
- Given Alice's available_to_spend is exactly 30.00 and 100 `proposed` Trades from 100 distinct Providers of 1.00 each, all addressed to Alice
- When Alice's 100 accept requests (each with a distinct idempotency key) are issued in parallel from several client sessions
- Then exactly 30 Trades are `completed` and 70 remain `proposed` each with a limit rejection; Alice's balance equals her floor; her `account_seq` values are gapless; cached balance equals the sum of postings; the Network sum is 0. The scenario is repeated 100 times with randomised order.

**QA-E2E-04 — Retry with the same idempotency key**
Cites: INV-009, FR-TRD-015, API-080, API-081, API-083, API-067, DAT-024, DAT-025, NTF-043, UX-041.
- Given a `proposed` Trade of 8.00 awaiting Bob's acceptance
- When Bob's accept request (key K) commits but the response is dropped, and the client retries with key K and the same payload five times
- Then every response is the original success with the same `journal_entry_id` and header `Idempotent-Replayed: true`; exactly one Journal Entry, one Audit Event and one set of notifications exist; balances changed once. Repeating with 50 concurrent retries yields at most one execution (others `idempotency_in_progress` or replay).

**QA-E2E-05 — Same key, changed payload rejected**
Cites: INV-009, API-082, API-085, DAT-026, FR-TRD-015.
- Given Alice created a Trade proposal of 5.00 to Bob with key K
- When a request with key K is sent again changing the amount to 50.00
- Then it is rejected with `idempotency_conflict` (422), the original proposal is unchanged, no second proposal exists. When a request with a fresh key K2 and amount 50.00 is sent, it is evaluated independently.

**QA-E2E-06 — Expiry race at the boundary**
Cites: FR-TRD-012, FR-TRD-013, LIF-015, LIF-055, LIF-056, DOM-021, R2, UX-079, NTF-047.
- Given `proposal_expiry_days` = 7 and a `proposed` Trade created at T0 with `expires_at` = T0 + 7 days
- When the injected clock is set one second before `expires_at` and Bob accepts
- Then it completes. In a second run, when the clock is set exactly to `expires_at` and Bob accepts at the same moment the expiry timer runs
- Then exactly one of the two outcomes occurs per the rule `now < expires_at`: at or after the boundary the accept fails reporting expiry, the Trade is `expired` with history actor `system` and reason `timer:<name>`, no ledger effect, and exactly one expiry notification is sent. Running the timer twice and late changes nothing further.

**QA-E2E-07 — Cancel versus accept race**
Cites: FR-TRD-011, FR-TRD-006, LIF-003, LIF-005, LIF-013, R1, INV-002, INV-007.
- Given a `proposed` Trade from Alice to Bob
- When Alice's cancel and Bob's accept are issued simultaneously, repeated 1,000 times on fresh Trades
- Then in every run exactly one wins: either `cancelled` with no ledger effect and Bob's accept answered `illegal_transition`, or `completed` with one Journal Entry and Alice's cancel answered `illegal_transition` with direction to use a Reversal; no run shows both effects or a partial posting; Network sum 0.

**QA-E2E-08 — Dispute upheld to Reversal with limit-breach flag**
Cites: FR-DSP-001, FR-DSP-002, FR-DSP-006, FR-DSP-007, FR-DSP-008, FR-DSP-009, FR-DSP-010, FR-ADM-009, FR-ADM-015, FR-TRD-023, FR-BAL-006, FR-AUD-001, INV-004, INV-010, DOM-025, DOM-027, DAT-020, LIF-020, LIF-021, LIF-022, D-12.
- Given a `completed` Trade of 40.00 (Bob Provider, Alice Recipient) inside `dispute_window_days`, and Bob has since spent his credit so that a Reversal would take Bob below his floor
- When Alice raises a Dispute (reason, description, remedy sought)
- Then the Dispute is `open`, no balance changed, Trade state is still `completed`, and both parties can see it.
- When Rae (not a party, no conflict) takes assignment (`under_review`) and resolves `resolved_upheld` with a written reason, and Nia (non-party) with the required quorum applies the Reversal
- Then a Reversal Journal Entry exactly negates the original Postings, is linked to the original Journal Entry, the Dispute and an Audit Event; the Trade becomes `reversed`; Bob's account is over limit and the entry carries the flag `limit_exceeded_by_reversal`; Bob can still receive credit but cannot be debited by a Trade until balance recovers; history shows the Reversal labelled with reason and links; both parties are notified; Network sum is 0.
- And when Rae is Bob (a party), assignment and resolution are refused and the Dispute is reassigned; and a second Reversal request for the same Trade fails.

**QA-E2E-09 — Adjustment visibility**
Cites: FR-ADM-007, FR-ADM-008, FR-ADM-015, FR-BAL-006, FR-AUD-004, FR-NTF-004, INV-004, INV-010, INV-012, DOM-030, DOM-031, DOM-032, DOM-033, SEC-035, UX-045, PER-033.
- Given `admin_quorum` = 1 and Alice's balance 10.00
- When Nia posts an Adjustment of −4.00 against Alice's account versus `network_clearing` with a mandatory reason
- Then it posts as an `adjustment` Journal Entry (not a Trade); Alice is notified immediately and the notice cannot be disabled; Alice's history shows it labelled "Adjustment" with reason, actor role and link; Alice's activity view (SCR-17) lists the Audit Event; Nia cannot adjust her own account; no endpoint or screen allows Nia to create a Trade for Alice; other Members cannot see the Audit Event.
- And when the Adjustment would leave Alice outside her limits, an explicit override reason is required and the entry is flagged.

**QA-E2E-10 — Departure with negative balance**
Cites: FR-MEM-012, FR-MEM-013, FR-MEM-014, FR-MEM-015, FR-MEM-017, FR-TRD-020, INV-016, INV-015, DOM-035, DOM-036, DOM-037, LIF-028, LIF-029, POL-001, R9, R16, DAT-072.
- Given Dana is active with balance −20.00 and `departure_settlement_mode` = `settle`
- When Dana starts departure
- Then the Membership is `departing`, her Listings are archived, the settlement summary states in plain words that she needs to provide 20.00 of goods/services, and she can only propose or accept Trades that move her balance toward 0 without crossing it; a Trade crossing 0 is rejected.
- When her balance reaches 0 through such Trades and the finalize step runs
- Then the Membership becomes `closed` in one atomic step, personal data is anonymised per D-09 while opaque account IDs remain, no further entry may reference her account, and a later return requires a new Membership.
- Variant A (`write_off`): the balance −20.00 is zeroed by a `departure_settlement` entry against `network_clearing` in the same atomic step as `closed`, with an Audit Event naming mode, amount and sign. Variant B (`settle`, window expires at non-zero): the Membership stays `departing`, admins are notified, no automatic entry is posted. Variant C: a final Trade racing the finalize results in exactly one winner (R16).

**QA-E2E-11 — Admin quorum**
Cites: FR-ADM-002, FR-ADM-003, FR-ADM-018, FR-ADM-019, FR-ADM-015, PER-008, SEC-035, SEC-009, DOM-032, R14, LIF-051, UX-103.
- Given Net1 with three active admins (Nia, Omar, Pia) and `admin_quorum` = 2
- When Nia initiates an Adjustment without recent step-up
- Then the action requires re-authentication (`reauthentication_required`), the step-up dialog is focus-managed, and after step-up the request is `pending` with one approval (Nia's).
- When Nia tries to approve again, then it is refused; when Omar and Pia approve simultaneously
- Then exactly one posting results (the later approval is idempotent or `illegal_transition`); a target admin cannot approve an Adjustment affecting their own account; a pending approval older than the approval window expires without effect; attempting to set `admin_quorum` above the number of active admins or below 1 is rejected.

**QA-E2E-12 — Policy change with notice**
Cites: FR-ADM-001, FR-ADM-002, FR-ADM-004, FR-ADM-005, FR-ADM-014, FR-NTF-002, INV-010, DOM-014, POL-001, POL-002, D-11, LIF-024.
- Given `max_trade_amount` is 100.00, `policy_notice_days` = 14, a `proposed` Trade of 80.00 exists, and the Network has 200 active Members
- When Nia (with quorum) changes `max_trade_amount` to 50.00 with a reason
- Then the change is `pending` until approved, then scheduled for a date at least 14 days later; every active Member receives a notice with old value, new value, reason and effective date; the policy history lists it; an Audit Event exists; behaviour is unchanged until the effective date.
- When the effective date passes
- Then new proposals above 50.00 are rejected, and acceptance of the existing 80.00 proposal evaluates the policy in force at commit (it fails the bound guard and stays `proposed`). A value outside platform bounds is rejected at entry. A change flagged as a safety fix may take effect sooner only with its recorded justification.

**QA-E2E-13 — Export reconciliation**
Cites: FR-EXP-001, FR-EXP-003, FR-EXP-004, FR-EXP-005, FR-EXP-007, FR-EXP-008, FR-BAL-011, PRT-001, PRT-002, PRT-008, PRT-009, PRT-010, PRT-017, PRT-018, PRT-021, PRT-036, PRT-040, DAT-052, INV-007, INV-008, ARC-046, SEC-105.
- Given a Standard-envelope synthetic Net1 including closed Members, Reversals, Adjustments and `limit_flag` entries, with the mixed soak workload running
- When Nia (with quorum and step-up) requests a `network_admin` export and Alice requests a `member_self` export
- Then the Network export completes asynchronously within 15 minutes with progress and a notification, without blocking postings; the independent verifier recomputes entry counts, per-entry zero sums, per-account balances and `ledger_total` from the package alone with zero mismatches against the manifest and against the live ledger at the snapshot; sum of all exported balances including System Accounts is 0; Alice's statement satisfies opening + entries = closing; no amount is a float or locale-formatted; formula-prefixed text is neutralised; closed Members are anonymised as in the UI and `redaction` counts match; Alice's export contains only data she can see; a repeat export for the same as-of is byte-identical apart from the generation timestamp; export lifecycle Audit Events exist.

**QA-E2E-14 — Restore drill**
Cites: OPS-023, OPS-026, OPS-027, OPS-028, OPS-029, OPS-030, OPS-032, DAT-051, DAT-078, DAT-079, INV-007, INV-008, INV-003, SEC-073.
- Given a staging deployment at the Standard envelope with continuous traffic, a recent backup, a recorded erasure after the backup point, and key custodians other than the configurer
- When Olu runs the drill into an isolated scratch environment, and separately a point-in-time restore to T
- Then restore finishes within the tier RTO; the application passes the smoke test; reconciliation reports zero findings; row counts and the latest Journal Entry ID equal those at the backup point; writes are refused until reconciliation passes; an Audit Event of the restore exists; Trades accepted after T are absent with no partial entry; the member erased after the backup is anonymised again by replaying the register; the restored environment is not reachable from production; a deliberately corrupted backup or wrong key is detected, alarmed, and leaves no usable half-restored environment; a failed drill is raised as Sev-2.

**QA-E2E-15 — Recovery flow**
Cites: FR-MEM-019, SEC-026, SEC-027, SEC-028, SEC-029, SEC-030, SEC-031, SEC-056, FR-NTF-004, FR-NTF-005, PER-015, JRN-008, UX-072.
- Given Alice has lost her only authentication factor, and a Visitor enters an unregistered email
- When both submit the "I can't sign in" form
- Then the responses are identical in content and status (timing within tolerance) and do not reveal registration.
- When Alice follows the single-use link (at most 15 minutes old) and reaches the recovery page
- Then she is not signed in automatically; she must set a new credential first; because she holds no registered credential or code, completion waits for the cool-off period and a security notice is sent to her previous channels; Alice's sensitive actions are restricted for the restricted period; the link cannot be reused or used after expiry; no administrator or Operator can complete the recovery on her behalf (PER-015); emails contain no credentials beyond the intended link.

**QA-E2E-16 — Cross-Network isolation**
Cites: INV-011, FR-NET-012, FR-MEM-004, PER-003, SEC-019, SEC-022, SEC-057, API-003, API-015, UX-002, UX-011, DAT-005, DAT-062, FR-EXP-003.
- Given Net1 and Net2 with different `unit_scale`, and Chidi holding Memberships in both, Nia admin of Net1 only
- When Chidi, signed in for Net1, requests Net2's Trade, Listing, Dispute and Audit identifiers through every Network-scoped endpoint family and via a forged `network_id` in the body; and Nia requests the same
- Then every attempt returns the same "not found" as a non-existent object with no hint of existence; Chidi's two Memberships have separate balances, states and roles; a Trade cannot name a counterparty from the other Network; a cross-Network Journal Entry cannot be created at the storage level; Net1's exports, notifications and admin views contain no Net2 data; the persistent header always names the Network Chidi acts in.

**QA-E2E-17 — Keyboard-only trade with screen-reader checks**
Cites: UX-054, UX-055, UX-056, UX-057, UX-037, UX-039, UX-042, UX-044, UX-063, UX-068, UX-069, UX-072, UX-082, UX-046, UX-049, FR-TRD-001, FR-TRD-004, D-14.
- Given a Member using only the keyboard and a screen reader, at a 320 CSS px viewport and again at 200% zoom
- When she uses the skip link, opens New Trade, chooses the counterparty with the keyboard-operable chooser (without typing the exact name), enters the amount in the decimal text field with its persistent unit label, submits with an invalid amount, corrects it, passes the Review step, goes Back to Edit (values and focus preserved), and sends
- Then every control is reachable and operable with no keyboard trap; focus order matches visual order; focus is visible with sufficient contrast; the error summary receives focus and lists each error with how to fix it; status messages are announced through the polite live region and blocking errors through the assertive region; on route change focus moves to the new level-one heading and the page title updates; history amount cells read fully (for example "Spent −12.00 hours"); no horizontal scrolling and no loss of function; targets are at least 24 x 24 CSS px; automated rule-engine run reports zero A/AA violations on each state visited. (Manual audit step recorded per QA-091, QA-092.)

**QA-E2E-18 — Notification deduplication**
Cites: NTF-042, NTF-043, NTF-047, NTF-054, NTF-044, NTF-096, NTF-068, FR-NTF-003, FR-NTF-005, FR-NTF-008, DAT-016, API-083, INV-009.
- Given Rae holds Member and Dispute Reviewer roles and is assigned a Dispute whose Trade she is not party to; Nia holds Administrator and Member roles; the transport double is configured to duplicate deliveries once and to fail once
- When a Trade is accepted with five client retries (same key), a timer fires twice for the same expiry reminder, a Dispute assignment event is delivered to a user holding two roles, and the transport fails after commit
- Then each recipient has at most one notification per (event id, recipient, channel, template_key); retries and duplicate timers create none extra; a user holding several roles receives the content once; the process crash between commit and hand-off results in the notification being sent after recovery exactly once (transactional outbox); the transport failure is recorded and the business action is not blocked; the in-app notification exists within 60 seconds and email is handed over within the dispatch latency; message bodies contain no credentials and no Trade description free text.

**QA-E2E-19 — Unknown outcome after timeout**
Cites: UX-041, UX-003, API-067, API-083, API-084, INV-009, FR-TRD-015, FR-BAL-003, DAT-038.
- Given Bob's accept request is committed server-side but the connection times out before the response
- When the client shows the unknown-outcome message and Bob uses "check status"/retries with the same key
- Then the client first says "We couldn't confirm whether this went through" and does not display the Trade or balance as completed until the server confirms; the retry returns the original outcome; balance shown carries the sequence it reflects; no duplicate posting; the assertive alert is announced to assistive technology.

**QA-E2E-20 — Suspension versus accept race**
Cites: FR-MEM-010, FR-MEM-011, FR-TRD-021, LIF-016, LIF-051, LIF-018, R6, R18, INV-015, INV-012.
- Given a `proposed` Trade between Alice and Bob
- When Nia suspends Bob and Alice (or Bob) accepts at the same time, repeated 1,000 times
- Then per commit order: either the suspension commits first and the accept fails (Trade remains `proposed` with the reason shown) or the accept commits first and the Trade stays `completed`; suspension never changes any balance or limit; a `suspended` Member cannot propose, accept or be named in a new proposal, and their Listings are hidden; on reinstatement they reappear as before; no admin can accept on Bob's behalf.

**QA-E2E-21 — Over-limit account after limit lowered**
Cites: FR-ADM-006, FR-ADM-008, FR-BAL-009, DOM-012, DOM-013, R7, INV-005, UX-004, UX-025, FR-NTF-001.
- Given Alice's balance −35.00 with credit limit 50.00
- When Nia lowers Alice's `credit_limit` to 20.00 with a reason
- Then Alice's balance is unchanged at −35.00 (no silent mutation), she is notified, her screen explains in words that further spending is blocked until her balance recovers while receiving credit is still possible, `available_to_spend` shows 0 (not negative), a debit Trade is rejected and a credit Trade is accepted; once balance reaches −20.00 or above she can spend again. A race between the lowering and an accept (R7) resolves by commit order using the limit in force at commit.

**QA-E2E-22 — Operator has no ledger or content power; break-glass**
Cites: FR-ADM-017, FR-AUD-006, PER-006, INV-012, INV-003, SEC-042, SEC-043, DAT-064, SEC-065.
- Given Olu holds `platform_operator` and no standing access to production data
- When Olu attempts through every interface to read a Net1 Trade, post or edit an entry, accept a Trade, or act as a Member
- Then each attempt is denied (not-found for content), and only platform-level audit events are visible. When Olu requests break-glass with a reason and ticket reference and it is granted
- Then access is time-limited, an Audit Event with `network_id` set is visible to that Network's administrators, an alert fires, and even then no capability to post, edit, reverse or delete exists.

**QA-E2E-23 — Migration preserves the ledger**
Cites: DAT-082, DAT-083, DAT-084, DAT-085, DAT-086, OPS-048, OPS-049, INV-003, INV-008, DAT-022, DAT-023.
- Given a Standard-envelope staging deployment on release N-1 with an IntegrityCheckpoint, and release N containing a schema migration
- When the migration runs, is interrupted midway and re-run, and then release N is rolled back or rolled forward per its declared strategy
- Then the migration is idempotent and resumable; no Journal Entry or Posting value, identifier or `entry_hash` input changed; ledger digests before and after are identical; reconciliation R1 to R14 passes with zero findings before the migration is marked complete; release N-1 application code ran correctly against the expanded schema during transition; the `migration.applied` Audit Event exists.

**QA-E2E-24 — Unit identity locked by first entry**
Cites: INV-013, DAT-037, DOM-007, FR-NET-004, FR-NET-005, API-130.
- Given a Network in `setup` with no Journal Entry
- When the first admin edits `unit_scale` while a first posting (an opening balance) is issued at the same instant, repeated many times
- Then in every run exactly one outcome occurs: the change commits and the posting then uses the new scale, or the posting commits and the change is rejected; the pre-launch warning about immutability is shown (FR-NET-004); after any entry exists, edits to `unit_name`, `unit_symbol` and `unit_scale` through the UI and API are rejected.

## 13. Assumptions and open decisions

Local labels `QA-OD-n` are aliases for `DEC-nnn` in [13](13-assumptions-risks-decisions-roadmap.md) (canon 5.8 item 8). Proposed defaults are shown.

| ID | Open decision | Options | Proposed default | Affects |
|---|---|---|---|---|
| QA-OD-1 | **Race-test gate IDs.** Doc 09 and the brief for this document were expected to define dedicated race-test gate requirements numbered 059 and 061 in doc 15; doc 15 currently defines the race requirement as `LIF-051` (with `LIF-052`, `LIF-053`, `LIF-055`) and defines no LIF IDs above `LIF-057`. | (a) keep LIF-051 as the single authority and map it to QA-GATE-05; (b) add dedicated race-gate requirements to doc 15 after its current last LIF ID | (a); no dangling reference is introduced here. Owner of doc 15 to confirm | Section 7.1, QA-GATE-05 |
| QA-OD-2 | Iterations per race test | 100 / 1,000 / 10,000 | 1,000 on release candidate; 100 on PR for touched areas (aligns with LIF-051) | QA-040, QA-GATE-05 |
| QA-OD-3 | Property-test volume | 100,000 (ARC-036) per release; per-PR volume 1,000 / 5,000 / 20,000 | 100,000 per release candidate, 5,000 per PR | QA-GATE-02 |
| QA-OD-4 | Coverage and flake thresholds | branch coverage 80/90/95% on critical modules; flake rate 0/1/2% | 95% critical modules branch coverage; 80% overall; 0 flakes in critical suites, 1% elsewhere | QA-GATE-03, QA-GATE-12 |
| QA-OD-5 | Assistive-technology/browser pairs for manual audit (UX-OD-3) | 2 / 3 / 4 pairs | 2 pairs minimum, covering one desktop and one mobile screen reader | QA-092, QA-GATE-07 |
| QA-OD-6 | Restore drill frequency (OPS-OD-5) and tier RPO/RTO (OPS-OD-3) | hosted 1 / 7 / 30 days; self-host 30 / 90 / 180 days | As OPS-026: hosted 7 days automated, self-host 90 days; plus every release candidate in staging | QA-061, QA-GATE-09 |
| QA-OD-7 | Soak duration | 1 / 4 / 24 hours | 4 hours per release candidate, 24 hours quarterly | QA-022 |
| QA-OD-8 | Production synthetic probes (a dedicated test Network in production) | none / read-only probes / full synthetic Network | none for MVP self-hosted; read-only probes for hosted | Section 11.2 |
| QA-OD-9 | Traceability tool and location: the conventions reference `docs/plans/traceability.md`; the directory does not yet exist | create plans document and tool in the first implementation increment / defer | Create before the first P0 implementation merge; until then manual matrix is a draft-phase substitute | QA-101 |
| QA-OD-10 | Independent verifier ownership (QA-070): written by the export team or by a separate party/repository to guarantee independence | same team different module / separate repository / external | Separate module with no shared export code, reviewed by someone outside the export change | QA-070 |

Assumptions: ASM (to register in doc 13): (1) the reference datastore of the first implementation is available in CI at the cost of build time; (2) the OpenAPI document (API-130) is the machine-readable contract source; (3) the permission matrix of 02 section 5 can be exported to a machine-readable table without changing its meaning; (4) injected-clock support exists in all timer code (LIF-007).

## 14. References

- Internal: [00 conventions](00-conventions.md); [02 personas, roles, journeys](02-personas-roles-and-journeys.md) (permission matrix, JRN, PER-001); [03 domain model](03-domain-model-and-mutual-credit-rules.md) (INV-001 to INV-016, DOM); [04 functional requirements](04-functional-requirements.md); [05 architecture](05-architecture-and-trust-boundaries.md) (ARC-036, ARC-040 to ARC-046, section 9.1); [06 data, ledger, audit](06-data-ledger-and-audit.md) (DAT-012 to DAT-051, DAT-082 to DAT-086); [07 API](07-api-and-integration-contract.md) (API-080 to API-088, API-130); [08 UX](08-ux-and-accessibility.md) (UX-054 to UX-069, UX-082, UX-085); [09 security](09-security-privacy-and-abuse.md) (SEC-020, SEC-022, SEC-094, gates G1 to G5); [11 operations](11-deployment-operations-and-support.md) (OPS-023 to OPS-034, OPS-048); [14 glossary](14-glossary.md); [15 lifecycle state machines](15-lifecycle-state-machines.md) (LIF-051 to LIF-056, race matrix 9.2); [16 policy catalog](16-network-policy-catalog.md); [17 notifications](17-notifications-and-communications.md) (NTF-042 to NTF-054, NTF-096); [18 export and portability](18-data-export-and-portability.md); [19 legal questions](19-legal-and-regulatory-open-questions.md).
- External: WCAG 2.2, W3C Recommendation, <https://www.w3.org/TR/WCAG22/> (accessed 2026-10-21). OWASP ASVS 5.0.0 (as cited by SEC-115), <https://owasp.org/www-project-application-security-verification-standard/> (accessed 2026-10-21).
