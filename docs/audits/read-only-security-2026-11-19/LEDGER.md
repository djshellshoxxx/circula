# Circula — ledger, financial authorization, and closure review

Date: 2026-11-19 (America/Los_Angeles; 2026-11-20 UTC). Reviewer scope: defensive source review, repair notes only.

## Provenance and limits

The reviewed main revision is `aa01563fcc891da60fefebb8335eca27855678e8`. The consolidated Core candidate is `22c767a90f69d286642ef96e6b3c174836637a3c` (PR #125). Review used a detached local worktree at `/workspace/scratch/1b6df59fd7e6/Circula-ledger-review`; the original clone is `/workspace/scratch/1b6df59fd7e6/Circula-audit`.

Production code under `internal/service`, `internal/store`, and `internal/domain` is identical between these two revisions. Their differences in those directories are five store test files. The API error and closure handlers cited below are also identical. `internal/web/refusal.go` differs only in an unrelated decimal-validation string match. Consequently, all four current findings below apply to both pinned revisions. Line references identify candidate source; cited production lines have the same numbering in main.

This review read source, schema, existing tests, specifications, and prior audit dispositions. It did not exercise a running deployment, create adversarial requests, write a reproduction, generate abnormal workloads, or modify application files. It did not independently run the Go tests. The root audit owns baseline test and dependency-check results; their results should be attached separately, without attributing them to this source review. Existing test names below describe useful coverage and repair targets, not execution receipts.

The audit establishes source-level behavior, not deployment exploitability. In particular, the closure race and memory-exhaustion outcomes have not been measured or reproduced. No new way to violate zero-sum posting, post without consent, cross a network boundary, or overrun debit capacity was established in this owned review. That statement is limited to the paths and assumptions inspected; it is not a security certification.

## Findings for workers

| ID | Priority / severity | Current behavior | Confidence | Main | Candidate |
|---|---|---|---|---|---|
| LED-R01 | Medium, correctness of integrity checks | Closure checksums incorporate a fresh generation timestamp, so later verification of unchanged ledger data normally mismatches | High from complete source trace; no runtime reproduction | Present | Present |
| LED-R02 | Medium privacy; requires specification reconciliation | API insufficient-credit errors reveal another member's exact available capacity in cases where ordinary balance reads are forbidden | High disclosure confidence; privacy intent is contradictory | Present | Present |
| LED-R03 | Medium, availability | Closure finalization and verification buffer the entire ledger export although they only use its digest | High allocation confidence; actual exhaustion threshold unmeasured | Present | Present |
| LED-R04 | Medium, conditional lifecycle correctness | A closure job can act on a replacement schedule without rechecking its current due date | High missing-check confidence; concurrency consequence is source-derived | Present | Present |

These findings are not remote code execution, unauthorized money creation, or confirmed production incidents. Severity reflects the specific outcome and prerequisites described below.

### LED-R01 — Closure checksum includes a timestamp that changes on every verification

**Owner:** ledger export / closure worker. **Applies to:** both pinned SHAs. **Suggested priority:** fix before relying on closure certification or shipping a closure-integrity status to users.

**Evidence.**

- `internal/service/export.go:198-199` creates each export manifest with `GeneratedAt: s.now()`.
- `internal/service/export_write.go:95-106` includes that `generated_at` value in the JSON document header and writes the header through `countingWriter`.
- `internal/service/export.go:243-260` hashes all bytes written to that writer; the file digest therefore includes the generation timestamp.
- `internal/service/closure_schedule.go:297-305` computes the closure checksum from each manifest file's name, row count, and digest.
- `internal/service/closure_schedule.go:311-336` records the checksum from a newly generated JSON ledger export.
- `internal/service/closure_schedule.go:354-381` later generates another JSON ledger export and directly compares its new checksum with the stored value.
- `internal/service/service.go:168,181` uses the current wall clock by default and truncates it to milliseconds; it does not bind export time to the closure checkpoint.
- `internal/api/h_closure.go:74-80` exposes the resulting `match` value.

**Result and prerequisites.** An authorized administrator verifying a legitimately closed, unchanged network at a different export timestamp receives a mismatch. No attacker or database modification is required. The checksum of the serialized file legitimately changes because its header changes. The comparison treats this expected change as a change in the certified ledger.

This prevents a normal later check from serving its stated purpose. It can cause false integrity alarms, unnecessary investigation, and eventual disregard of a check whose negative result has become routine. It does not modify balances, create a false positive match, or by itself let tampering pass. Medium is a correctness / operational-integrity rating, not a claim of a monetary exploit.

**Requirement and existing-test gap.** `docs/specs/10-governance-operations.md:306-312` (GOV-025) requires a final checksum that matches a recomputed value. `internal/service/closure_test.go:84-98` computes and verifies the checksum without advancing the fake clock between those operations. Compatibility coverage in `internal/service/adjustment_relation_test.go:219-229` and the post-closure departure check in `internal/service/departure_test.go:29-50` likewise use unchanged fixture time. Those checks are compatible with this defect and do not establish later wall-clock verification.

This differs from historical A1-07: the missing-final-export retry is already repaired. It also differs from A1-01: settlement after closure is already blocked. Neither repair removes volatile export metadata from the digest input.

**Repair direction.** Define a versioned, deterministic closure-checkpoint representation. One option is to hash canonical ledger content while excluding generation metadata. Another is to persist the exact checkpoint metadata, including the original export timestamp, and reuse it for verification. Keep generation time in ordinary downloadable exports if useful, but distinguish it from the value the closure digest commits to.

Treat existing stored checksums explicitly. `closed_at` is not guaranteed to equal the original export's generation time because closure and export commit in separate phases. Do not silently substitute a new timestamp or overwrite a historical checkpoint and call it verified. Preserve the old algorithm/version and provenance, with an explicit compatibility or migration decision for checkpoints lacking enough metadata to reconstruct their original bytes.

**Safe acceptance criteria.** For a normal fixture with no ledger changes, closure verification must return a match after elapsed time and after process restart. The digest must still change for an authorized, controlled fixture change to a committed field. The final-export retry must remain idempotent. A documented test must cover how existing checkpoints are presented and verified; no undocumented rewrite of historical hashes is acceptable. Run existing closure, export-format, event-order, and adjustment-relation compatibility tests alongside that invariant.

### LED-R02 — API refusal details expose another member's available credit

**Owner:** service error model, API serialization, and privacy/specification owner. **Applies to:** both pinned SHAs. **Suggested priority:** settle the disclosure contract and use actor-aware error details.

**Evidence.**

- `internal/service/transactions.go:115-118` makes the counterparty the payer for a proposal in which the proposer will be paid.
- `internal/service/transactions.go:140-159` reads that payer's reserved outgoing amount and computes available capacity. On insufficient credit, the returned domain error includes the exact formatted value, clipped at zero.
- `internal/service/posting.go:19-30` similarly embeds the debited account's balance plus effective credit limit in a posting refusal. A non-proposing payee accepting a payment proposal can receive that refusal.
- `internal/api/respond.go:123-145` carries recognized domain error messages into `apiError.detail`; `internal/api/respond.go:161-165` serializes that detail to the client.
- By comparison, `internal/service/accounts.go:52-65` returns only the caller's own balance view. An administrator's read of another member's balance is a separately authorized and audited operation beginning at `internal/service/accounts.go:70-78`.
- The HTML error path, `internal/web/refusal.go:177-181`, reads the caller's own `GetBalance` instead of echoing the service message. That avoids the same direct disclosure there, although showing one's own amount for a counterparty failure can itself be confusing.

**Prerequisites and information revealed.** The relevant caller is an authenticated, active member allowed to trade in the same network, with the normal write, membership, terms, counterparty, and amount gates satisfied up to the capacity refusal. The caller need not hold administrator or reviewer privileges. A posting refusal additionally requires a proposal the caller may legitimately accept.

The disclosed number is available capacity, not necessarily the balance in isolation. At proposal time it is `max(0, balance + effective limit - reserved outgoing)`; the posting check uses `max(0, balance + effective limit)`. Limits and reservations affect what can be inferred about the standalone balance. The observation is constrained to same-network financial information; no cross-network access or credentials were found. A failed proposal returns before insertion, notification, or the success audit event at `transactions.go:165-178`, so the affected counterparty receives no proposal notification for that refusal.

**Specification conflict must be resolved explicitly.** The permission matrix at `docs/specs/02-personas-journeys.md:91-94` denies ordinary members a view of another member's balance and describes administrator balance/limit reads as audited. T-10 at `docs/specs/09-security-privacy.md:114` identifies disclosure of another member's balance/history as a threat. However, the trade journey at `docs/specs/02-personas-journeys.md:128-139` explicitly asks to show the effect on the payer's credit and the available amount on failure, including the request-to-be-paid direction. FR-031 at `docs/specs/04-functional-requirements.md:259-264` similarly asks for payer availability and the exceeded limit.

Accordingly, this is a confirmed disclosure with a policy contradiction, not an unqualified claim that implementation violates a single clear specification. If sharing availability with any active counterparty is intentional, document that exception and its audit/consent consequences. It should not remain hidden behind a statement that financial data is private to the member and audited administrators.

**Repair direction.** Prefer typed, privacy-safe insufficient-credit errors. Show precise balance, limit, reservation, or available amounts only when the actor is the payer or already has the appropriate explicit authority. A non-payer can receive a useful refusal explaining that the counterparty cannot cover the trade without exposing the amount. Apply the same rule to proposal, acceptance, reversals, and all adapters. Do not rely on the HTML template alone to redact a service message the API still returns. Reconcile the journey and FR wording with the agreed privacy rule.

**Safe acceptance criteria.** Table-driven role/party tests should assert that refusal DTOs and HTML output shown to an ordinary non-payer contain no other member's balance, effective limit, reserved amount, or exact availability. Owners should retain any authorized own-account detail. Existing same-network/cross-network denial behavior, refusal atomicity, idempotency, and notification behavior must remain intact. This can be checked entirely with ordinary fixture accounts and deterministic service errors.

### LED-R03 — Checksum-only closure exports retain all ledger bytes in memory

**Owner:** closure / export resource-budget worker. **Applies to:** both pinned SHAs. **Suggested priority:** remove the unnecessary materialization before large networks rely on closure.

**Evidence.** `internal/service/closure_schedule.go:311-316` and `354-379` both call `Export(... ExportLedger, FormatJSON, &MemSink{})`. They use only the resulting manifest/checksum, not the collected file contents. `internal/service/export.go:423-438` defines `MemSink` as a map of files whose `Write` appends every byte to an unbounded byte slice. The manifest's per-file digest is already computed incrementally by `countingWriter` at `export.go:243-260`, and JSON rows stream through it at `internal/service/export_write.go:93-146`.

**Prerequisites and effect.** A legitimately large network reaches scheduled closure, or an authorized administrator requests final-checksum verification. The full uncompressed ledger document is retained in the serving process's heap. The allocation scales with total ledger size, despite the caller needing only a digest. A sufficiently large export can cause memory pressure, termination, or unavailability affecting other networks in that process. The resource threshold depends on deployment memory and ledger size and was not tested.

Scheduled finalization is a system-actor operation. The ordinary per-actor export quota explicitly exempts system actors at `internal/service/export.go:163-172`; a quota also cannot bound the size of one successful export. This path does not pass through the API's bounded disk export spool. The administrator verification handler at `internal/api/h_closure.go:74-80` directly calls the same service operation.

Closure status commits before `recordFinalExport`; an export failure leaves the checksum phase pending. The current retry logic at `closure_schedule.go:180-190` correctly retries that phase, but retries do not remove the allocation cause. This is an availability/resource defect, not evidence that the ledger commits partially or loses its accounting entries.

**Requirement and prior-audit status.** FR-095 at `docs/specs/04-functional-requirements.md:546-550` requires streaming without holding the full ledger in memory. The current status text and API spool repair address HTTP exports. Historical `docs/audit/CODE-AUDIT.md:110-111` discussed API buffering, not these remaining checksum-only service callers. Do not reopen the repaired API buffering finding as if its fix were absent; repair the uncovered closure path.

**Repair direction.** Use a sink that accepts and discards bytes while the existing counting/hash wrapper constructs the manifest, or factor a dedicated streaming checksum operation with the same canonical representation. If a downloadable final artifact must also be retained, store it in the bounded file spool with explicit lifecycle and cleanup. Do not allocate the full ledger merely to obtain its digest. Keep single-snapshot reads and export authorization/audit semantics intact. Resource budgets should apply to verification and background jobs, not only the download adapter.

**Safe acceptance criteria.** A normal growing fixture must produce the same canonical checksum through the verification path and the artifact path while verification memory remains bounded independently of total serialized ledger bytes. Check success, read failure, cancellation, and finalization retry cleanup. Retain authorization tests for administrator-only closure checks. Use bounded fixture sizes suitable for ordinary CI; no process-exhaustion demonstration is required.

### LED-R04 — Closure finalizer does not revalidate the current schedule inside its write unit

**Owner:** network lifecycle / scheduled-job worker. **Applies to:** both pinned SHAs. **Suggested priority:** close the stale-schedule window before relying on notice guarantees.

**Evidence.**

- `internal/service/closure_schedule.go:174-194` loads pending closure records in a read unit. `internal/store/sqlite_closure.go:75-93` selects all open schedules, including future ones.
- `closure_schedule.go:196-207` subsequently checks `c.Due(s.now())` on each saved record and passes only its network ID to `finaliseOne`.
- `closure_schedule.go:217-229` rereads the network and its current closure in a write transaction. Its guard checks stored network status, closure existence, and whether it is done. It does not check that the current closure is due, or that it is the schedule captured by the job.
- The method then cancels open proposals at `230-254` and can set the network closed and record the current schedule as done at `264-278`.
- `CancelClosure` at `129-162` can delete an open schedule before its due date. `ScheduleClosure` at `68-125` can create a replacement with a new minimum notice period. `internal/store/sqlite_network.go:34-56` loads the current closure without applying the service's derived closing status; `finaliseOne` does not call `lifecycle`.

**Prerequisites and consequence.** This is a conditional concurrency issue: a closure batch remains in progress across a legitimate cancellation and replacement of a schedule near its old deadline. Cancellation must occur before the old deadline; the service correctly refuses cancellation after that date. A long-running batch can retain the old record while the current schedule has moved to a future date. The outer check can regard the saved record as due, while the inner operation acts on the replacement merely because it exists.

The result can be closure and proposal cancellation before the replacement's announced date, with the read-only/erasure lifecycle beginning from an unintended completion. With unresolved disputes, the method can still cancel proposals before discovering the blocker. No unauthorized caller is needed, but the relevant administrative scheduling and job timing must coincide. No runtime race was reproduced, and no estimate of production likelihood is claimed.

This is distinct from ordinary concurrent postings: the SQL write transaction prevents interleaving inside finalization. The problem is that the business predicate was evaluated against an earlier object and is not rechecked against the object actually mutated. GOV-025 at `docs/specs/10-governance-operations.md:306-311` requires the announced date and minimum notice to control closure.

**Repair direction.** Inside the same write unit that cancels proposals and closes the network, verify that the freshly loaded schedule is due. Prefer identifying the expected schedule explicitly, for example by immutable schedule ID/revision, and treat a replaced or canceled candidate as a no-op. Make the current schedule authoritative. Also review the placement of the consistency-freeze guard so a job queued before a freeze does not rely solely on its earlier outer-loop check.

**Safe acceptance criteria.** A deterministic scheduler test should hold a batch while an ordinary schedule change is committed and assert that the current future schedule remains active, its proposals remain unchanged, and no closure audit/notification is emitted. Separately assert that the current due schedule closes once, canceled schedules remain canceled, blocked closures retain their intended behavior, and existing final-export retry coverage still passes. The test should assert lifecycle invariants rather than provide an attack request sequence.

## Existing owner decisions that remain material

These are not new findings and should retain their existing IDs and decision owners. Current source was checked so workers do not mistake historical awareness for a completed repair.

| Existing issue | Current source / disposition | Remaining decision or work |
|---|---|---|
| A3-07: voluntary departure can forgive debt | `internal/service/departure.go:255-291` applies the network's `settle-to-zero` rule. The extra dual-control threshold check at `275-280` is conditional on an administrator removing another member; voluntary departure can settle the debt without that approval. `docs/specs/addenda/SEC-011-owner-decision.md:62` explicitly leaves this policy open. | Decide whether voluntary debt settlement requires approval, enters a pending settlement process, or remains an accepted network rule. This moves the negative position to the system account; it does not break ledger zero sum. |
| SA2-12: dual control applies to selected action types, not every change to credit position | `internal/service/reversal.go:84-111` permits reversals of completed non-reversal transactions subject to capacity and network gates, without a dual-control threshold. Credit-limit and policy increases also remain outside a single uniform approval rule. `docs/specs/addenda/SEC-011-owner-decision.md` lists the broader scope decision. | Establish approved action scope, treatment of related/split actions, and independent approval requirements before extending the mechanism. Do not call a field named `DualControlThreshold` a universal credit-position control. |
| A1-04-G2: closure freeze and dispute reversals can block each other | `internal/service/reversal.go:87-89` rejects reversal while closing; `internal/service/closure.go:10-22` treats disputed transactions as closure blockers; `closure_schedule.go:145-149` prevents canceling a closure after its date. `docs/audit/2026-10-28/REMEDIATION-2026-11-18.md:13,157` records the open owner decision. | Decide whether a dispute-only posting exception is allowed while closing, or explicitly specify another resolution route. Do not imply that dismissing or withdrawing a meritorious dispute is an adequate technical repair. |
| Legacy policy-scale changes stored by older builds | `internal/service/networks.go:430-536` reports existing scheduled scale changes without rewriting them. Current setters prevent new decimal-scale changes. | Inventory historical affected networks and retain the documented owner/operator decision. Reporting a legacy condition is not proof that it was repaired. |

Grant-ancestry independence and membership-list iteration errors are owned by the separate authentication/authorization review. The ledger reviewer independently checked the `database/sql` behavior behind the latter: an iteration error can auto-close the rows, after which a later `Close` can return nil; the loop needs `Rows.Err`. A security outcome remains conditional on a non-cancellation read error and subsequent successful operations, not merely normal request cancellation. No driver-specific trigger was demonstrated. Do not count that shared observation twice.

## Controls that held in source, and old findings not reopened

The following are useful repair constraints. They must survive follow-up work.

- **Atomic posting and capacity.** `internal/store/sqlite.go:138-156` serializes writers with an in-process gate and `BEGIN IMMEDIATE`, while read units use a consistent read-only snapshot. `internal/service/posting.go:40-120` reads both accounts inside the posting unit, checks bounded payer/payee balances and the payer's effective limit, then creates the opposing entries and completion event. Caller audit and idempotency work share the same write unit. Source did not reveal a separate check-then-write gap in ordinary accept/post paths.
- **Bounded money.** `internal/domain/money.go` uses checked/bounded integer arithmetic; policy bounds and proposal/request limits reduce arithmetic exposure in the inspected posting paths. Current `internal/store/sqlite_verify.go` uses exact wide-accumulator fallback when SQLite aggregate sums exceed the SQL integer range. The recently repaired aggregate-overflow finding is not open in either pinned revision.
- **Durable idempotency.** `internal/service/idempotency.go:36-80` scopes records by network, actor, operation, and key; compares the fingerprint while the record is live; and records the result in the same unit as the mutation. Errors do not consume a key. Read/adaptor response replay has separate authorization controls and is covered by the peer review. The application deliberately permits key reuse after configured expiry; this should not be described as a timeless exactly-once guarantee.
- **Schema constraints and immutability.** The reviewed schema uses composite network scoping for ledger references and checks/triggers for transaction amounts, two-entry completion, append-only ledger events, balance bounds, adjustment relationships, and dispute policy binding. This defense is strongest against accidental application misuse and does not replace protecting raw database access.
- **Dispute policy binding.** Migration `internal/store/migrations/0019_dispute_policy_binding.sql`, `internal/store/sqlite_dispute.go`, and the current dispute service bind a new dispute to a policy version. The equal-timestamp/clock-rollback AUD4-F03 issue is repaired for newly created disputes; earlier timestamp-only descriptions are stale. Historical backfill necessarily retains the limitations described in its migration/audit notes.
- **Dispute independence.** The inspected decision path excludes parties and relevant prior actors, and reversal authorship filtering happens before the one-row query limit. Do not reopen the earlier page/filter independence issue. The deeper grant-chain issue belongs to the auth reviewer.
- **Departure and closure hardening already present.** Settlement now checks network-active state; open proposal cancellation drains pages; a closed network with a missing final export remains eligible for retry; erasure waits for the checksum phase. LED-R01, R03, and R04 concern separate remaining properties.
- **Exports and history.** Export data is reauthorized inside the consistent data snapshot. Network-wide export scopes require administrator authority; own-history exports preserve the documented suspended/departed-member rights until erasure. CSV values receive formula-trigger handling. No new credential export path was identified in the ledger table selection.
- **Verification scope.** `internal/service/verify.go` and `internal/store/sqlite_verify.go` check the ledger/cache/reversal aggregates and audit-chain structure in a read-only snapshot. Their success is an internal consistency check; it should not be marketed as proof against a privileged actor who can rewrite the database and all unkeyed audit material. The root operations review covers anchoring and durability qualifications.

## Coverage ledger

This lists the material inspected by this reviewer, not every file in the repository. “Read” means source inspection, not execution or formal verification.

| Area | Material read |
|---|---|
| Transaction lifecycle, posting, idempotency | `internal/service/transactions.go`, `posting.go`, `adjustments.go`, `reversal.go`, `idempotency.go`, `service.go`, `principal.go`, `network_access.go`; relevant `replay_auth.go` paths |
| Adjustments and authorization boundaries | `internal/service/dualcontrol.go`, `dualcontrol_operator.go`; coordination with separate auth reviewer on grant ancestry |
| Disputes | `internal/service/disputes.go`, `disputes_decide.go`, `disputes_response.go`, `disputes_deadlines.go` |
| Network policy, limits, reporting | `internal/service/networks.go`, `policy_revision.go`, `credit_limit_revision.go`, `accounts.go`, `histbal.go`, `reports.go` |
| Departure, closure, exports | `internal/service/departure.go`, `closure.go`, `closure_schedule.go`, `export.go`, `export_write.go`; relevant API `respond.go`, `h_closure.go` and HTML `refusal.go` lines |
| Integrity/audit | `internal/service/verify.go`, `audit.go`; `internal/store/sqlite_verify.go` and current aggregate repair notes |
| Storage | `internal/store/sqlite.go`, `sqlite_tx.go`, `sqlite_account.go`, `sqlite_dispute.go`, `sqlite_pending_adjustment.go`, `sqlite_network.go`, `sqlite_member.go`, `sqlite_closure.go`, `migrate.go` |
| Domain/schema | `internal/domain/money.go`, `policy.go`, `statemachine.go`, `fold.go`; migrations 0001, 0002, 0005, 0011, 0014, 0015, 0016, 0017, 0018, 0019. Other migration files inventoried, with auth/session/operations coverage assigned to peer reviewers |
| Existing test source inspected or located for handoff | Closure, closure retry, departure, adjustment relationship compatibility; concurrency, randomized ledger invariant, and idempotency test entry points. Named tests are not claimed as executed here |
| Specifications / prior audits | README, SECURITY, CONTRIBUTING; relevant sections of specs 02, 03, 04, 09, 10; SEC-011 and owner decision addenda; A1 ledger report and its remediation disposition; current consolidation notes for AUD4-F03 and aggregate arithmetic |

Remaining boundaries include real deployment configuration and load, malicious/compromised-host behavior, full browser rendering and adapter contracts, and native SQLite failure modes. Those were not simulated. No application patch accompanies these notes. Each current finding includes its own narrowly scoped repair and invariant-based validation target so another worker can implement it without treating the audit as approval for unrelated policy changes.
