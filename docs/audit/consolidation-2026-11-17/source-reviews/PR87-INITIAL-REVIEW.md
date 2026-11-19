# PR #87 — independent source and regression review

Reviewed 2026-11-18T09:25:01.611417+00:00. Caveman focused-review workflow; all eighteen changed files inspected. This reviewer made no tracked source change or remote mutation.

- Branch: `claude/aud4-f03-dispute-timelines`.
- Exact reviewed head: `1a3ef0fad836067b71857d961659459d976e7e15`.
- Tree: `1fc439e5a9a1bb90553d59952d02d117a4858871`.
- Compared main/base: `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` (tree `f41365f8ee4f42f4cbd33fb4e1a78cad85caedd1`).

## Readiness

**Useful partial remediation, with independently verified ordinary-case fixes. Do not close AUD4-F03 as fully fixed.** The supplied regressions, repository contracts and selected existing authorization/timing guards pass on the exact source. Timestamp variants reproduce the remaining same-millisecond case and an additional host-clock-rollback case. The source comment and audit note's unconditional history-immutability claims need qualification before presenting this as a completed security fix. Current-head CI and the coordinator's combined-tree qualification remain separate requirements.

This change implements the pre-existing “Dispute timelines … Prospective” rule in canonical GOV-001/section 2. It changes neither configured bounds/defaults nor who may change policy, respond, decide or request a re-review. It adds no migrations, dependencies, ledger semantics or future-module code. The as-built notes now identify opening as the reference point. Existing DEC-012/counsel questions and A3-F02/DEC-014 remain unresolved; this review does not infer a new owner-policy decision.

## Finding

`internal/store/sqlite_network.go:L71`: **Medium retained residual:** greatest-version selection by `effective_at <= OpenedAt` can select a policy written after opening. Preserve this as an unresolved timestamp-history case; do not claim a persisted or immutable opening version.

`internal/service/disputes_deadlines.go:L20` and `docs/audit/consolidation-completion-2026-11/AUD4-F03.md:L24`: append-only history does **not** make this query's answer immutable. Qualify the claim for equal-millisecond and backward-wall-clock writes. The audit note at L26 already acknowledges one tie case but incorrectly dismisses timing as uncontrollable and omits rollback.

The implementation stores millisecond UTC wall-clock values (`Service.now`) and orders qualifying policies by version. There is no stored policy-version or write-order link on the dispute. A later append therefore replaces the answer if its effective timestamp is at or before the old opening time. `UpdatePolicy` can append after a host-clock rollback when the original policy remains in force and no later pending version blocks the change. Append-only triggers preserve rows but cannot establish chronological write order from wall time.

Two variants of the author's existing `TestDisputeTimelinesFixedAtOpen` reproduce this on unchanged candidate production source:

1. **Same millisecond:** remove only the one-hour clock advance before the policy update. The dispute and later update have the same stored timestamp.
2. **Clock rollback:** advance two hours before opening, then move the injected wall clock back one hour before the update. The network's original policy predates both operations, so the service accepts the update with an effective time before the dispute's opening.

Both produce assertion failures for the moved response deadline, premature overdue reminder, resolution without response/waiver and removal of the original fourteen-day re-review window. These are retained portions of the original bug, which previously applied after any policy change; the evidence does not show a new ledger or authorization regression. The tests establish the conditions deterministically, not their production frequency. A later durable-version or write-order repair needs its own design and regression review; none was invented or implemented here.

## Verified behavior and boundaries

- **Historical selection:** SQLite scopes by network and selects the greatest version with effective time at or before the opening timestamp. Both raw and wrapped repository contracts pass: scheduled versions wait, a superseding earlier-effective version suppresses a cancelled future version, equal times select the larger version, pre-history/unknown networks are not found, and another network's policy cannot interfere.
- **Existing and new disputes:** changes made at strictly later timestamps no longer shorten or extend earlier disputes. Later disputes use changed values. Disputes opened during notice retain their original policy after the scheduled date. Missing historical policy fails closed instead of falling back to current policy.
- **Decision and tenant guards:** current actor/network/party/independence checks remain before the timeline decision gate. The historical lookup changes only timeline values, not authorization. Existing decision, visibility, responder, re-review and expiry tests pass.
- **Timestamp boundaries:** the response gate permits a decision exactly at its deadline; re-review remains inclusive at its last millisecond. The existing DST/year test passes all 28 zone/date subcases, including one millisecond on either side of the relevant windows. The two residual conditions above remain explicit exceptions to immutability.
- **Legacy policies:** missing historical JSON fields use the same `DefaultPolicy` behavior as `GetNetwork`; the supplied legacy test passes.
- **Presentation:** the service and web use the same historical window source; the web form's inclusive endpoint aligns with the service's millisecond-boundary contract. No new public API schema field or route is introduced.

## Executed checks

All runs used Go 1.26.11, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, `CGO_ENABLED=1`, `GOGC=50`, `GOMEMLIMIT=512MiB`, explicit workspace `GOCACHE`, `-mod=readonly -race -count=1 -p 1 -timeout=10m`. Local Go work was coordinated with Agent C and ran sequentially. No aggregate suite was run.

| Scope | Outcome | Wrapper wall time | Evidence |
|---|---|---:|---|
| Five supplied service/store/web regressions | PASS | 23.359 s | `PR87-FIXED-REGRESSIONS.log` |
| Two policy-history repository contract variants | PASS | 2.173 s | `PR87-FIXED-POLICY-CONTRACT.log` |
| Literal baseline + supplied web test | Expected assertion failure | 6.950 s | `PR87-BASELINE-WEB.log` |
| Candidate with old current-policy selector | Expected assertion failure | 11.824 s | `PR87-CURRENT-POLICY-CONTROL.log` |
| Candidate: same-ms clock variant | Expected assertion failure | 11.264 s | `PR87-SAME-MS.log` |
| Candidate: host-clock-rollback variant | Expected assertion failure | 10.859 s | `PR87-CLOCK-ROLLBACK.log` |
| Six existing authorization/visibility/response/re-review/DST tests | PASS | 24.200 s | `PR87-EXISTING-GUARDS.log` |

The five supplied fixed cases pass with package runtimes service 3.235 s, store 1.369 s and web 1.737 s. The two contract variants pass in a 1.760 s store invocation. The literal baseline web test fails by hiding the re-review form/date and rejecting the request; the exact test file is unchanged from the reviewed PR.

The service before-control reinstates only `tx.GetNetwork(d.NetworkID).Policy` in the shared selector while retaining the new DTO/interface seams, so all supplied assertions compile unchanged. All three supplied service tests then fail at their intended behavioral assertions. This is a clearly labeled mutation, not a claim that the final service test file compiled unchanged on the old baseline. The same-ms and rollback variants each fail at the expected assertions, with package runtimes 0.666 s and 0.648 s respectively.

Post-test integrity verification compared **979 tracked blobs** in the fixed snapshot against the exact reviewed Git tree: **zero mismatches**. Snapshot, overlay and timestamp-variant provenance; test commands/environments/exit codes; inert full diff; and source hashes are retained under `PR87-*` in this workstream. `PR87-SOURCE-REVIEW.json` is the machine-readable index.

## Complete changed-file review

| Changed file | Review scope |
|---|---|
| `docs/TRACEABILITY.md` | Generated citation inventory only; requirement statuses are unchanged. |
| `docs/api/openapi.yaml` | Existing response/re-review policy descriptions aligned with the at-opening lookup; no new endpoint or authorization scope. |
| `docs/audit/consolidation-completion-2026-11/AUD4-F03.md` | Finding and supplied evidence retained; unconditional history-immutability and uncontrollable-timing claims require correction. |
| `docs/dev/SERVICE.md` | Service timeline documentation; corrects stale platform-constant description of the configurable re-review window. |
| `docs/specs/03-domain-model.md` | As-built note only; DOM-017 completion-time immutable dispute window remains unchanged. |
| `docs/specs/04-functional-requirements.md` | FR-076/078 as-built descriptions identify the opening policy; durations, default questions and role rules remain unchanged. |
| `docs/specs/10-governance-operations.md` | Implementation note under existing GOV-001 and prospective dispute-timeline table; no new default or owner policy resolution. |
| `internal/domain/closure.go` | One shared response/re-review deadline calculation; exact 24-hour durations and inclusive re-review endpoint. |
| `internal/repository/iface.go` | Adds tenant-scoped historical policy read contract with legacy defaults and not-found behavior. |
| `internal/service/dispute_timelines_test.go` | Three supplied policy-change regressions; shortening, lengthening, later disputes, scheduled dates and exact response deadline. |
| `internal/service/disputes_deadlines.go` | Shared lookup/view/reminder behavior; fail closed for missing history; retained equal/backdated timestamp limitation. |
| `internal/service/disputes_decide.go` | Response gate and view use historical timelines; existing current authorization, independence, waiver and posting logic retained. |
| `internal/service/disputes_response.go` | Re-review eligibility uses shared historical window; party, lifecycle, uniqueness and reason checks retained. |
| `internal/store/policy_in_force_test.go` | Legacy JSON-default compatibility and parity with GetNetwork at the same instant. |
| `internal/store/sqlite_network.go` | Network-scoped effective_at<=instant query, greatest version wins, same defaults as GetNetwork. |
| `internal/storetest/storetest.go` | Repository contracts cover scheduled/cancelled versions, equal-date version order, not-found and tenant separation. |
| `internal/web/dispute_timelines_test.go` | Supplied browser-handler regression reproduces literal baseline failure and candidate success. |
| `internal/web/h_dispute_review.go` | Uses service re-review deadline and inclusive comparison; existing party/activity/reviewer conditions retained. |

## Coordinator handoff

Before merge, correct the unconditional source/audit claims or explicitly reconcile them in reviewed current documentation, retain AUD4-F03 as partially remediated with both timestamp residuals, and require successful CI on the actual proposed head plus the required combined qualification. No additional owner choice was identified for the compatible prospective-policy correction itself. This reviewer did not approve a migration, close the residual, or issue a GitHub review/merge action.
