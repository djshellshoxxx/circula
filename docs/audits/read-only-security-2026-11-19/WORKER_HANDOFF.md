# Repair-worker handoff

All entries start **OPEN** at main `aa01563fcc891da60fefebb8335eca27855678e8` and candidate `22c767a90f69d286642ef96e6b3c174836637a3c`. Logical ownership below is not a named person's assignment. Qualifications in the lane reports remain part of each work packet. No implementation or configuration change accompanies these notes.

## Work order

1. Address global-identity recovery and report confidentiality. In parallel, fix incomplete ancestry traversal and shared membership error handling.
2. Repair closure checkpoint determinism, schedule revalidation and checksum-only memory use as coordinated, separately reviewable changes.
3. Settle the financial-error privacy rule and implement it across adapters. Complete transactional listing revision enforcement.
4. Add platform audit coverage and bounded mirror/hash work. Qualify Windows storage and operator-reader budgets.
5. Restore current-head qualification and prepare repository protections. Review existing policy holds before making a release decision.

An owner decision is needed for an actual unresolved product/security policy. Prepare the patch, tests, compatibility proposal and evidence that make the decision reviewable. Routine implementation of an established invariant does not need a new policy decision.

## Authority, privacy and HTTP work

| ID / owner | Primary files | Required invariant and acceptance evidence |
|---|---|---|
| AUTH-01 / authentication and privacy | `internal/service/recovery.go`; `tx_queries.go`, `reads.go`, `export.go` | Network-local recovery cannot replace a global identity's credentials while protected authority remains elsewhere, including departed-but-retained access. Preserve single-network recovery, operator recovery, MFA and history retention. Refusal leaves credential state unchanged. Record erased-membership policy. [Detail](AUTH.md) |
| AUTH-02 / governance | `internal/service/dualcontrol.go`; `internal/store/sqlite_member.go` | Authorization succeeds only on a completed permitted ancestry result. Depth exhaustion and inconsistent provenance cannot be success. Preserve explicit founder/operator/break-glass roots and the separate direction policy. Approval, rejection and eligibility agree; refusal leaves ledger/pending state unchanged. [Detail](AUTH.md) |
| AUTH-03 / repository | `internal/store/sqlite_member.go`; recovery/session-lifetime consumers | Iteration errors propagate after partial rows; no successful partial authority set reaches consumers. Use an inert repository-error fixture. No corruption or cancellation-based bypass demonstration is needed. [Detail](AUTH.md) |
| AUTH-04 / authentication capacity | `internal/service/password.go`, `auth.go`, `throttle.go`; `internal/web/h_auth.go`, `ratelimit.go` | Shared admission bounds simultaneous hashes before allocation. Queue length/waiting are bounded; canceled waiters leave; every result releases capacity. Known/unknown accounts receive consistent admission. Use a benign hash seam and document sizing. [Detail](AUTH.md) |
| WAPI-01 / reports and governance | `internal/service/reports.go`, `audit_read.go`, replay/export views; API/web handlers | Under the current promise, a report subject cannot learn reporter identity or handle their own report. Include listing ownership, role changes, replay and identity correlation. Independent admins retain intended access. Define escalation when all admins are conflicted. Mutation conflicts are transactional. [Detail](WEB_API.md) |
| WAPI-02 / listing concurrency | `internal/service/listings.go`; `internal/api/h_listings.go`, `dto.go`; HTML writer | Carry the expected revision into the write transaction. A stale explicit validator refuses without overwriting an intervening edit. Preserve ownership, archive finality, field-specific updates and wildcard semantics. Existing ETags already include editable content. [Detail](WEB_API.md) |

## Financial and lifecycle work

| ID / owner | Primary files | Required invariant and acceptance evidence |
|---|---|---|
| LED-R01 / checkpoint format | `internal/service/closure_schedule.go`, `export.go`, `export_write.go` | Unchanged content verifies after elapsed time/restart. Use a versioned deterministic representation or retained exact metadata. Document old-checkpoint compatibility; do not replace historical hashes silently or assume `ClosedAt` reconstructs export time. Authorized committed-content changes alter the digest. [Detail](LEDGER.md) |
| LED-R02 / privacy/error contract | `internal/service/transactions.go`, `posting.go`; `internal/api/respond.go`; web refusal mapping and specs 02/04/09 | Settle availability-sharing policy. Prefer typed actor-aware errors: non-payers receive useful refusals without another member's exact financial values; authorized own detail remains. Cover proposal, acceptance, reversal and adapters. Preserve refusal atomicity/idempotency. [Detail](LEDGER.md) |
| LED-R03 / export resources | `internal/service/closure_schedule.go`, `export.go` | Digest-only paths stream to a bounded/discarding sink with the same snapshot/hash. Memory does not retain all serialized bytes. Bounded fixtures cover cancellation, errors and retry. Coordinate format changes with LED-R01. [Detail](LEDGER.md) |
| LED-R04 / lifecycle jobs | `internal/service/closure_schedule.go`; `internal/store/sqlite_closure.go` | Recheck current schedule due date and expected identity/revision before every effect in the write unit. Replaced/canceled candidates are no-ops. Future replacements keep proposals, status and notice periods. Preserve blockers, freeze semantics and retry. [Detail](LEDGER.md) |

## Operations work

| ID / owner | Primary files | Required invariant and acceptance evidence |
|---|---|---|
| OPS-01 / audit integrity | `internal/service/verify.go`; `cmd/circula/snapshot.go`, `auditanchor.go` | Cover platform/network scopes explicitly in routine verification and a versioned anchor. Define empty/legacy semantics and failure behavior. Legacy network-only anchors cannot imply platform coverage. [Detail](OPERATIONS.md) |
| OPS-02 / audit delivery | `internal/store/sqlite.go`; `cmd/circula/auditmirror.go` | A stalled optional sink cannot indefinitely retain the application writer gate. Choose bounded delivery/outbox with explicit loss, order, backpressure, observability and shutdown semantics. Acknowledgement follows policy; committed writes remain committed. [Detail](OPERATIONS.md) |
| OPS-03 / Windows storage | `cmd/circula/util.go`, `statekey.go`, `fileident_other.go`, config/staging/spool; packaging | Validate/establish private storage with native security descriptors. Cover database/sidecars, keys, plaintext staging and exports with a second unprivileged principal. Define service identities and reparse-point policy. Manual ACL guidance is a prerequisite, not enforcement evidence. [Detail](OPERATIONS.md) |
| OPS-04 / local readers | `cmd/circula/verify_export.go`; `scripts/read-export.py` | Separate small manifest, member-count, aggregate-byte and configurable budgets. Stream where practical. Inject small budgets for limit coverage; ordinary exports still validate. Preserve candidate archive-correctness repairs. [Detail](OPERATIONS.md) |

## Delivery work

**QG-01 — Repository owner:** prepare a branch/ruleset policy requiring agreed reviews and exact-commit checks, with explicit emergency/bypass authority and separate tag controls. Capture before/after evidence when implemented. This audit changes no permissions/settings.

**QG-02 — CI/evidence owner:** correct scanner-control metadata for the exact composition; preserve positive/negative controls and narrow exceptions. Run the complete combined job on the repair commit. Keep native Core execution distinct from cross-build and Python checks. [Observed evidence](evidence/github-observations.json).

## Isolated feature work

FTR-N01 and FTR-N02 are **Low** pre-integration items on the notification pin, outside Core and excluded from the 14-item Core total. Fence dispatch and per-target outcomes against stale lease ownership, while documenting at-least-once delivery. Settle whether external recipients/providers may learn internal identifiers from signed, unencrypted link claims. Federation needs an explicit identity-continuity policy for endpoint/key changes under an existing remote label. Keep the features disabled/HOLD pending their own integration gates. [Feature evidence and criteria](FEATURES.md).

## Existing owner decisions

| Existing issue | Required explicit decision | Source detail |
|---|---|---|
| AUD4-F02 | Reviewer recovery authority and later decision eligibility, including revocation/regrant | [Auth limits](AUTH.md) |
| AUD4-F01 / SEC-011 | Direction of independence and independently rooted administrator availability | [Auth limits](AUTH.md) |
| A3-F02 / DEC-014 | Lasting mutations allowed under temporary break-glass authority | [Auth limits](AUTH.md) |
| GOV021 held branch | Two labels plus procedure or independently authenticated operators; waiting/escalation | [Held branch](AUTH.md) |
| A3-07 | Voluntary-departure debt-settlement approval or explicit accepted policy | [Ledger decisions](LEDGER.md) |
| SA2-12 | Financial action types and related/split actions covered by dual control | [Ledger decisions](LEDGER.md) |
| A1-04-G2 | Legitimate dispute-resolution route during closure freeze | [Ledger decisions](LEDGER.md) |
| Legacy scale changes | Inventory and treatment of existing scheduled scale changes | [Ledger decisions](LEDGER.md) |

## Required closure record

Append for each finding: repair SHA; invariant; required policy/compatibility decision; exact checks/results; platform/configuration assumptions; independent review; remaining limitation. Distinguish **source-confirmed**, **runtime-verified**, **policy-accepted**, and **not yet verified**. Leave ineffective or blocked repairs open with the concrete reason. Use ordinary deterministic fixtures, controlled seams and bounded data suitable for CI. Repository fixtures do not establish assessment of a live deployment.
