# 12 Quality Strategy and Test Plan

Status: Draft v0.1 · Owner/role: Quality lead (test engineering) with Product architect · Last updated: 2026-10-21 · Scope: test layers, required coverage for ledger invariants, concurrency, retries, authorization, disputes, migrations, accessibility, backup and restore, and export correctness; scenario end-to-end tests; quality gates; traceability for the MVP · Dependencies: [00](00-conventions.md) (LED-1..LED-7), [03](03-domain-model.md), [06](06-data-ledger-audit.md), [07](07-api-contract.md), [08](08-ux-accessibility.md), [09](09-security-privacy.md), [10](10-governance-operations.md), [11](11-deployment-operations.md), [13](13-assumptions-risks-decisions.md) · Open questions: section 10

Normative terms follow [00 section 1](00-conventions.md#1-normative-terms). Test requirements use the prefix `QA`. Requirements from other documents are verified here by reference; this document does not restate them, it maps tests to their IDs.

**Core principle.** A passing test suite does not prove legal, tax or financial correctness of a network's rules, accessibility conformance beyond the checks actually run, or fitness of any deployment. Each claim in a release note MUST be backed by a recorded test or check run against the stated build. A check that was not performed MUST be recorded as not performed.

---

## 1. Test layers

| Layer | Purpose | Runs in | Tooling |
|---|---|---|---|
| L0 Static checks | Formatting, vet and lint, static security analysis, secret scan, dependency scan, static checks on migrations | Every change | To be chosen by engineering; recorded in [13](13-assumptions-risks-decisions.md) if changed |
| L1 Unit | Domain rules as pure functions: policy validation, state transitions, money arithmetic | Every change | Standard test runner for the language |
| L2 Property and invariant | Random sequences of domain operations, invariant checks after every step | Every change (short), merge and release (long) | Property-based testing library |
| L3 Storage integration | Real embedded database, transactions, migrations, idempotency records, audit atomicity | Every change | Real database file in temporary directories |
| L4 Concurrency | Concurrent postings, retries and lock contention on the real storage | Merge and release | Multi-client harness with injected delays |
| L5 API contract | Versioned JSON endpoints, error format, pagination, idempotency, compatibility against the contract in [07](07-api-contract.md) | Every change | Contract tests generated from or checked against the contract |
| L6 Interface integration | Server-rendered flows with scripting disabled, form handling, CSRF, headers | Every change | HTTP-level tests |
| L7 Authorization matrix | Every route against every role and membership state | Every change | Table-driven test generated from the route inventory |
| L8 Security tests | Session, cookie, rate-limit, logging, export and secret scans; dependency checks | Every change (subset); merge and release (full) | Mixed |
| L9 Accessibility | Automated checks, keyboard-only flows, screen reader checks | Merge (automated); release (manual) | Automated checker plus manual protocol |
| L10 Browser end-to-end | Scenario tests (section 5) in real browsers | Merge and release | Browser automation |
| L11 Backup, restore and migration | Backup during load, restore, verification, migration from every released schema | Merge (subset); release (full) | Scripted drills on fixture data |
| L12 Export correctness | Reconciliation of exports against ledger and balances; isolation; neutralisation | Every change (unit and integration); release (full) | Automated comparison |
| L13 Packaging and smoke | Build for each DEC-003 target, checksum, start, health, one posting | Merge (build); release (run) | CI runners per platform where available |
| L14 Exploratory and review | Structured manual exploration, code review of ledger and security paths | Before release | Session charters recorded |
| L15 Performance and soak | Measured throughput, latency and long-run stability on reference hardware | Release candidate | Load harness; results recorded with hardware |

---

## 2. Ledger invariant tests

**QA-001** · P0 · Invariant property tests
- Requirement: The test suite MUST include property-based tests that generate random sequences of domain operations (create network, add member, change policy, propose, accept, decline, cancel, expire, complete, reverse, adjust, suspend, depart) across at least two networks, and check LED-1, LED-2, LED-3, LED-5, LED-7 and GOV-007 after every operation.
- Rationale: Invariants must hold for sequences no one wrote by hand.
- Acceptance criteria:
  - The short run used on every change executes a fixed number of sequences (recommended default: 1,000 sequences of at most 50 operations; recorded per CI configuration).
  - The long run used on merge and release executes at least 100,000 sequences (recommended default; DEC-023 pending).
  - Any failing sequence is saved as a reproducible seed and fixture file, and the failure is reported with the seed.
  - Zero invariant violations across the release run.

**QA-002** · P0 · Balance derivation and cache agreement
- Requirement: Tests MUST verify that every cached balance equals the sum of entries for its account after each operation in QA-001 and after each concurrency scenario in section 3 (LED-3).
- Rationale: Cached balances are a performance optimisation that must never diverge.
- Acceptance criteria:
  - Zero mismatches in the release run (test).
  - A deliberately corrupted cache value in a seeded test is detected by the verification operation (OPS-016; negative test).

**QA-003** · P0 · Equal and opposite postings
- Requirement: Tests MUST confirm that every completed transaction has exactly two entries, with equal magnitude and opposite sign, on two different accounts, and that every reversal and adjustment has the same property (LED-1; [06](06-data-ledger-audit.md)).
- Rationale: The core mutual-credit property.
- Acceptance criteria:
  - No completed transaction in any property run has other than two entries (test).
  - No transaction in any run has entries that sum to a non-zero value (test).
  - A negative test that attempts to post a single-sided entry is refused (test).

**QA-004** · P0 · Credit limit boundary tests
- Requirement: Tests MUST cover the credit-limit boundary: a posting that lands exactly on the negative limit is accepted; one minor unit beyond is refused; a zero limit refuses any negative balance; a limit reduction below current balance follows GOV-019; and the system account is exempt as LED-5 states.
- Rationale: Off-by-one errors in credit limits are a primary money-handling defect class (LED-7).
- Acceptance criteria:
  - Each listed case has a named test with its expected result (test).
  - The boundary tests run for limits of zero, one minor unit, and a large value (test).

**QA-005** · P0 · Integer money handling
- Requirement: Tests MUST confirm that no money value is represented as a floating-point type in the domain, storage, API or export code paths, and that inputs with fractional minor units or decimal separators in the wrong format are rejected (LED-7; SEC-024).
- Rationale: Floating-point rounding is a silent error source.
- Acceptance criteria:
  - A static check finds no floating-point type used for money (static check, L0).
  - Amounts with a fractional part beyond the configured scale are rejected (test).
  - Arithmetic overflow near the maximum integer value is refused without wrapping (test).

**QA-006** · P0 · Audit atomicity
- Requirement: Tests MUST confirm LED-6: for every state-changing operation, a fault injected after the business write and before commit leaves neither the business change nor the audit event; a successful operation leaves exactly one audit event of the expected type.
- Rationale: Audit gaps are repudiation risks (T-08, T-09).
- Acceptance criteria:
  - Fault injection at each step boundary of the posting path yields zero partial states (test; at least one injection per step).
  - Each state-changing route produces exactly one audit event (test; L7 matrix extension).

---

## 3. Concurrency, retries and idempotency

**QA-007** · P0 · Concurrent postings against one account
- Requirement: Tests MUST drive many concurrent postings against the same account and the same network, including the case where the sum of requested amounts exceeds the available credit, and verify that no account goes below its credit limit (LED-5), that no posting is lost or double-applied, and that the set of successful postings is consistent with some serial order.
- Rationale: Serialised per-account checks are the only protection against over-limit postings under concurrency (T-05).
- Acceptance criteria:
  - Over-subscription scenario: with N concurrent requests whose total exceeds the limit, the final balance is never below the negative limit at any observed point (test).
  - The number of successful postings equals the number that fit in the available credit (test).
  - Final balances equal the sum of accepted postings (QA-002).
  - Scenario repeated for at least 100 random interleavings per CI configuration (test).

**QA-008** · P0 · Concurrent postings across accounts
- Requirement: Tests MUST confirm that concurrent trades among disjoint pairs of members in the same network neither deadlock nor serialise unnecessarily beyond the write-lock behaviour in OPS-012, and that the network-wide invariant (sum of balances is zero) holds.
- Rationale: Deadlocks would freeze a network.
- Acceptance criteria:
  - A run with 100 concurrent disjoint trades completes within a timeout recorded per DEC-023 and produces zero deadlocks (test).
  - The network sum of balances is zero after the run (test).

**QA-009** · P0 · Idempotent retries
- Requirement: Tests MUST confirm LED-4 for retried requests: the same idempotency key and identical payload return the original result and create no new entries, transactions or audit events; the same key with a different payload is refused with a conflict error; and a retry after a timeout whose original committed returns the committed result.
- Rationale: Network retries are routine; duplicate postings are a severe error (T-06).
- Acceptance criteria:
  - Ten identical retries produce exactly one transaction and one pair of entries (test).
  - A different payload with the same key is refused and leaves no change (test).
  - A retry after a simulated timeout returns the original status (test).
  - Each command accepts the key for at least 24 hours (API-020), replays return the original result with the replay header (API-022), and key reuse with a different body returns the conflict response (API-023) (test).
  - After the key retention period in [06](06-data-ledger-audit.md) section 9 expires, the key is free for reuse as DAT-028 states (test with injected clock).

**QA-010** · P0 · Crash and restart during posting
- Requirement: Tests MUST interrupt a posting process at injected points and restart it; the result after restart MUST be either the fully committed state or the original state, with no partial entries (LED-4; OPS-033).
- Rationale: Crash consistency is the foundation of the atomicity claim.
- Acceptance criteria:
  - Injection at each step boundary, followed by restart, produces one of the two valid states (test).
  - The verification operation passes after each restart (test).

**QA-011** · P1 · Lock contention behaviour
- Requirement: Tests MUST confirm that under sustained contention a posting that cannot acquire the write lock within the timeout returns a retryable error and leaves no partial state (OPS-012).
- Rationale: Predictable failure under load is needed for clients to retry safely.
- Acceptance criteria:
  - Under a seeded contention scenario, timeouts are returned as retryable and no partial state is found (test).

---

## 4. Authorization, disputes and governance

**QA-012** · P0 · Authorization matrix
- Requirement: The authorization matrix test MUST cover every route (interface and API) for each role (anonymous, member, departed member, suspended-network member, network administrator, dispute reviewer, platform operator without break-glass, platform operator with break-glass) and for principals of another network. Expected results MUST be stored in a reviewed table, and a mismatch MUST fail the build (SEC-009, SEC-010, SEC-013, SEC-015).
- Rationale: Authorization failures are the most probable disclosure and elevation defects (T-10, T-15).
- Acceptance criteria:
  - The route inventory equals the authorization table: a route missing from the table fails the build (test).
  - Zero unexpected allows across the matrix (test).
  - Every deny response for an object of another network is indistinguishable from a not-found response (test).

**QA-013** · P0 · Dispute lifecycle
- Requirement: Tests MUST cover the dispute state machine in [10](10-governance-operations.md) (GOV-013, GOV-014, GOV-016): raising within and outside the window; the response deadline with and without response; reviewer recusal; decision with and without reasons; reversal and adjustment outcomes; appeal within and outside the period; and that the transaction status moves to `disputed` and back as [03](03-domain-model.md) defines.
- Rationale: Disputes change money and must be deterministic.
- Acceptance criteria:
  - Each transition in the state machine has at least one test, and each invalid transition is refused with a test (test).
  - A dispute outcome never edits the original transaction (test; QA-014).
  - Deadlines are tested with an injected clock across daylight-saving and year boundaries in UTC (test).

**QA-014** · P0 · Reversal and adjustment non-destructiveness
- Requirement: Tests MUST confirm that after any reversal, adjustment, dispute outcome or write-off, the original transaction and its entries are unchanged in storage, and that the balances equal the sum of the original and corrective entries (LED-2; GOV-015).
- Rationale: Traceability of corrections is required for dispute evidence.
- Acceptance criteria:
  - A before and after comparison of stored original records shows no difference (test).
  - Balances match the sum of all entries (QA-002).

**QA-015** · P0 · Governance rules
- Requirement: Tests MUST cover the configurable policies in [10](10-governance-operations.md) section 2, including bounds (GOV-005), prospective-only application (GOV-001), notice rules (GOV-003), decimal lock (GOV-004), membership lifecycle (GOV-008 to GOV-012), and administrator continuity (GOV-020, GOV-021).
- Rationale: Governance rules are product behaviour, and changes to them must be provable.
- Acceptance criteria:
  - One test per rule in the listed requirements (traceability check).
  - Boundary values for each bound are tested at the boundary and one step beyond (test).

**QA-016** · P0 · Dual control and conflicts
- Requirement: Tests MUST cover SEC-010 and SEC-011: self-approval refused; conflicted reviewer refused; last-administrator removal refused; approval expiry closes the request with an audit event and no posting.
- Rationale: These are the principal administrative misuse controls (T-16).
- Acceptance criteria:
  - Each refusal has a test that asserts zero ledger change and one audit event for the refused attempt, where the policy requires one (test).

---

## 5. Scenario end-to-end acceptance tests

**QA-017 to QA-026** · P0 · Scenario acceptance set
- Requirement: The following scenarios are the minimal acceptance set for the MVP. Every scenario is P0. Each covers a core journey defined for the MVP and MUST pass with the key assertions shown. A scenario that is skipped MUST be recorded with a reason (QA-038).
- Rationale: End-to-end journeys catch integration defects that layer tests miss, and they map to the core journeys in the MVP specification.
- Acceptance criteria: each row's key assertions hold in the release run. Each scenario runs in a real browser with scripting disabled for the core steps and with scripting enabled for the same steps, and each also runs against the JSON API where the step is available there.

| ID | Scenario | Key assertions |
|---|---|---|
| QA-017 | Invite-based join: an administrator invites a member; the member accepts the terms and sets up an account | Member appears in the directory only for members of the network; terms version recorded (GOV-008, GOV-009) |
| QA-018 | Trade: a member lists an offer; a second member proposes a trade; the offerer accepts; the trade completes | Two entries sum to zero; both balances change by equal and opposite amounts; history shows both parties; audit event exists (LED-1, LED-6) |
| QA-019 | Decline and expiry: a proposal is declined; another expires after the configured period | No entries; statuses `declined` and `expired`; notices sent (OPS-028) |
| QA-020 | Credit limit boundary: a member trades up to exactly the limit, then one unit further | Exact-limit trade accepted; one-unit-further refused with a clear message; balance unchanged after refusal (LED-5) |
| QA-021 | Dispute and reversal: a completed trade is disputed; the reviewer upholds; a reversal posts | Original unchanged; reversal equal and opposite; both parties notified; decision reasons visible to parties (GOV-014, GOV-015) |
| QA-022 | Administrator policy change: an administrator reduces a credit limit | Change log entry; notice to member before effect; existing negative balance unchanged; new increases refused (GOV-002, GOV-003, GOV-019) |
| QA-023 | Departure: a member leaves a network with an open proposal | Open proposal cancelled with counterparty notice; member frozen; self-export offered; balance unchanged (GOV-010) |
| QA-024 | Backup and restore: a backup is taken during trades, the instance is destroyed, and the backup restored | Verification passes; balances equal pre-backup values; restore audit event present (OPS-017, OPS-020) |
| QA-025 | Network export and re-import: a network is exported with a reconciliation summary and imported into a clean instance | Reconciliation sums to zero; balances and entries equal (GOV-022, SEC-017) |
| QA-026 | Upgrade with rollback: an instance is upgraded through a migration that is then forced to fail, and rolled back | Pre-upgrade backup taken; failed migration leaves the prior schema; rollback restores the pre-upgrade state (OPS-022 to OPS-024) |

---

## 6. Migrations, backup and restore

**QA-027** · P0 · Migration from every released schema
- Requirement: Each released schema version MUST have a fixture database, and the test suite MUST migrate each fixture to the current head and run the verification operation before and after (OPS-022, OPS-023).
- Rationale: Migrations are the most likely path to silent data damage.
- Acceptance criteria:
  - Every fixture migrates to head with zero invariant failures (test).
  - Running the migration again changes nothing (test).
  - A fixture with more than 10,000 entries migrates within a time recorded per DEC-023 (performance test).

**QA-028** · P0 · Backup under load and restore
- Requirement: Tests MUST take a backup during concurrent postings, restore it into a clean instance, and verify integrity, the verification operation, the manifest checksum and that all committed postings before the backup time exist (OPS-017, OPS-020).
- Rationale: Backups are tested by restoring, not by existence.
- Acceptance criteria:
  - Restore completes with zero integrity errors (test).
  - Verification passes; posting refuses until verification passes on a seeded corrupted backup (test).
  - Restore duration is recorded (performance record, DEC-016).

**QA-029** · P0 · Restore drill evidence
- Requirement: Each release that changes storage or migrations MUST have a recorded restore drill (OPS-021), and the release gate MUST check for the drill record.
- Rationale: The drill is the evidence that the recovery procedure works.
- Acceptance criteria:
  - The drill record names the backup, duration and outcome (release check).

---

## 7. Export correctness

**QA-030** · P0 · Export reconciliation
- Requirement: Tests MUST confirm that each export type in GOV-022 contains entries whose sum is zero per transaction and whose per-account sums equal the balances at the export snapshot time, and that the reconciliation summary is recomputed independently by the test (L12).
- Rationale: An export that does not reconcile is not usable for disputes or migration.
- Acceptance criteria:
  - Zero reconciliation differences across the property corpus (test).
  - An export taken during concurrent postings reconciles to its own snapshot time (test).

**QA-031** · P0 · Export isolation and secrecy
- Requirement: Tests MUST confirm SEC-015 and SEC-017 for every export: no record of another network; no credential, token or hash; a member self-export contains only transactions in which the member is a party.
- Rationale: Export is a high-volume disclosure path (T-12).
- Acceptance criteria:
  - Canary scans of export files find zero matches (test).
  - Two-network dataset export has zero foreign identifiers (test).

**QA-032** · P0 · Spreadsheet-safe export
- Requirement: Tests MUST confirm that any cell beginning with a character that spreadsheet software can treat as a formula is neutralised in CSV exports (SEC-017).
- Rationale: Member-entered text (listing titles, dispute narratives) can contain formulas.
- Acceptance criteria:
  - A seeded set of formula-like values (for example leading equals, plus, minus, at-sign, tab and carriage-return characters) is written neutralised (test).

---

## 8. Accessibility and interface

**QA-033** · P0 · Automated accessibility checks
- Requirement: Every interface page and state listed in [08](08-ux-accessibility.md) MUST be checked by an automated accessibility checker in the merge gate, with zero violations at the conformance level set in DEC-024 (recommended: WCAG 2.2 level AA) for the rules the tool can test.
- Rationale: Automated checks catch regressions cheaply; they do not prove conformance (see QA-035).
- Acceptance criteria:
  - Zero automated violations at the chosen level on each listed page and state (test).
  - Each accepted exception is recorded with a reason and an expiry date (documentation check).

**QA-034** · P0 · Keyboard and scripting-disabled operation
- Requirement: Core flows (registration, invitation acceptance, trade proposal, acceptance, decline, balance and history view, dispute raise) MUST be completable using the keyboard only and with scripting disabled (SEC-023; [08](08-ux-accessibility.md)).
- Rationale: The product requires progressive enhancement, not script dependence.
- Acceptance criteria:
  - Each flow completes in the browser test with keyboard input only (test).
  - Each flow completes with scripting disabled (test).
  - Focus is visible at every step (test; automated plus manual sample).

**QA-035** · P0 · Manual accessibility review before release
- Requirement: Before each release, the core flows MUST be reviewed manually with at least one screen reader on each major platform that the release claims to support, and the review record MUST list the screen reader and browser combinations used. The supported combinations are defined in [08](08-ux-accessibility.md) and DEC-024.
- Rationale: Automated tools cannot test many accessibility requirements.
- Acceptance criteria:
  - The review record lists each combination and the outcome for each core flow (release check).
  - No open accessibility defect of the highest severity class at release (release check).

**QA-036** · P0 · Balance direction and status presentation
- Requirement: Tests MUST confirm that balance direction, negative balance, available credit, pending and completed transaction states are presented with text as well as colour, and that the same meaning is conveyed to assistive technology (UX requirements in [08](08-ux-accessibility.md)).
- Rationale: Balance direction errors are a practical harm (member misreads a debt).
- Acceptance criteria:
  - For each state in the balance display matrix, the rendered text and accessible name are asserted (test).
  - A contrast check passes for each state colour in both light and dark presentations (automated check).

---

## 9. Quality gates

Gates define the evidence required before a change moves forward. A failed gate MUST block the move, and a gate MUST NOT be waived by reducing a threshold in this document.

**QA-037** · P0 · Change gate (every merge request)
- Requirement: Every change MUST pass L0, L1, the short L2 run, L3, L5, L6, the L7 authorization matrix, the secret and dependency scans in L8, and the static checks on migrations.
- Acceptance criteria:
  - All listed checks pass on the change commit, with the results recorded in the change record (CI record).

**QA-038** · P0 · Integration gate (branch merged to the release line)
- Requirement: Merged work MUST pass the change gate and, in addition, L4 concurrency, the full L2 run at the recommended long-run size, L9 automated accessibility, L10 scenario tests QA-017 to QA-026, L11 subset (migration from the latest fixture and a backup-restore run), L12 export tests, and L13 build for each DEC-003 target.
- Acceptance criteria:
  - All checks pass on the merged commit (CI record).
  - Any scenario test that is skipped is listed in the change record with a reason.

**QA-039** · P0 · Release gate
- Requirement: A release candidate MUST pass the integration gate and, in addition: full migration fixtures (QA-027), restore drill record (QA-029), manual accessibility review (QA-035), L14 exploratory session records, L15 performance results on recorded reference hardware, L13 run smoke tests on each DEC-003 target or a documented reason for each target not run, SEC-033 independent security review record, OPS-026 release notes, and checksum and bill of materials (OPS-035).
- Acceptance criteria:
  - Each item exists and is linked from the release record (release check).
  - Zero open P0 defects that fail a P0 requirement in [09](09-security-privacy.md), [10](10-governance-operations.md), [11](11-deployment-operations.md) or this document (release check).
  - Zero failing ledger invariant tests (QA-001, QA-002, QA-003).

**QA-040** · P0 · Quality thresholds
- Requirement: The following thresholds apply. Values marked "pending DEC-023" are recommended defaults that the product owner MUST confirm or change through a recorded decision before the release gate; the gate MUST NOT pass with an unconfirmed value.
  - Ledger invariant failures in any run: zero (absolute).
  - Authorization matrix mismatches: zero (absolute).
  - Tests of P0 requirements failing: zero (absolute).
  - Branch coverage of the posting, balance and credit-limit code: recommended at least 90 percent, pending DEC-023. Coverage is a signal; it does not replace QA-001 to QA-007.
  - Latency and throughput: targets are set on recorded reference hardware, pending DEC-023. No latency figure is claimed until measured.
  - Login response time difference for existing and non-existing accounts: threshold set in SEC-005, pending DEC-023.
- Acceptance criteria:
  - The release record lists each threshold, its value, and its decision reference (documentation check).

**QA-041** · P1 · Test determinism
- Requirement: Tests that depend on time MUST use an injected clock. Tests that depend on randomness MUST record their seed. Tests MUST NOT depend on external network services.
- Rationale: Flaky tests hide real defects and are often ignored.
- Acceptance criteria:
  - No test calls the system clock directly outside the clock adapter (static check).
  - A failing randomised test reports its seed and reproduces with it (test).

## 10. Traceability and evidence

**QA-042** · P0 · Traceability matrix
- Requirement: A traceability matrix MUST map every P0 and P1 requirement in [03](03-domain-model.md) to [11](11-deployment-operations.md) and this document to at least one automated test, documented inspection or drill. Each matrix row MUST name the test identifier, layer, and last result with its date and build.
- Rationale: Verification evidence is required; requirements without tests are assumed to be unverified.
- Acceptance criteria:
  - Every P0 requirement has at least one row (documentation check).
  - No row points to a test that does not exist (cross-check).
  - The matrix is regenerated or reviewed for each release (release check).

**QA-043** · P1 · Test data
- Requirement: Test fixtures MUST be synthetic. Fixtures derived from any real data MUST be anonymised under [09](09-security-privacy.md) SEC-014 and MUST NOT be committed unless the anonymisation has been reviewed.
- Rationale: Test repositories are widely copied.
- Acceptance criteria:
  - A scan of fixture files finds no real personal-data pattern (automated scan).

**QA-044** · P1 · Evidence retention
- Requirement: Test reports, gate records, drill records and the traceability matrix MUST be kept with each release for at least the release's supported lifetime, recommended to be one year after release end-of-support.
- Rationale: Evidence is needed for audits and incident reviews.
- Acceptance criteria:
  - Each release record links to its evidence (release check).

---

## 11. Open questions

1. Concrete values for QA-001 run sizes, QA-007 interleavings and coverage threshold (DEC-023).
2. Choice of property-based testing library, browser automation and accessibility checker (recorded in [13](13-assumptions-risks-decisions.md) when selected).
3. Accessibility conformance level and supported screen reader and browser combinations (DEC-024).
4. Vulnerability severity threshold for dependency scans (SEC-029).
5. Reference hardware for performance claims (DEC-023).
6. Independent security review scope, provider and timing (SEC-033).
7. Whether scenario tests QA-017 to QA-026 must also run on each supported platform or only on the primary platform.
