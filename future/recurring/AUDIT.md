# Independent standalone audit — FNET-207

Date: 2026-11-16. Reviewer: a fresh agent, separate from implementation. Reviewed baseline `971c400` and committed the fixes described below. **Integration HOLD remains. No production approval, core compatibility or browser acceptance is claimed.** Only `future/recurring/` changes.

## Findings fixed

| Finding | Consequence before fix | Fix and reproduced regression |
|---|---|---|
| R207-A01: cached draft uses stale authorization target | Editing a template to an allowed target let an old retry return its prior, newly revoked target/amount after only the edited target was checked. | Validate the cached DTO and reauthorize its own target/amount before returning. `TestAuditDraftRetryAuthorizesCachedTargetAfterEdit`. |
| R207-A02: departure leaves private cached response bodies | Scrubbed template notes/amounts could still be recovered from old create/state/draft retries, including after a template target edit. | Retire responses for affected templates, historical target references and the departing owner. Preserve fingerprints so retries fail instead of replaying. `TestAuditDepartureRetiresCachedPrivateResponses` covers current and historic target cases. |
| R207-A03: silent zero-row writes accepted | SQLite trigger `RAISE(IGNORE)`, a missing serialization row or silently ignored safety inserts could acknowledge unpersisted state/reminders. | Require one persisted row for lock/template/audit/operation/reminder writes. An ignored firing is treated as duplicate only when its logical key actually exists. Bulk response retirement verifies no matching private body remains. `TestAuditSilentWriteFailuresRollBack`, `TestAuditMissingLockAndCorruptRecordFailClosed`, `TestAuditIgnoredDepartureRetirementRollsBack`. |
| R207-A04: existing incompatible schema accepted | `CREATE TABLE IF NOT EXISTS` accepted a table without the required primary key/constraints. | Validate reference tables/index definitions and lock row within the migration transaction; unknown shapes fail and roll back. `TestAuditIncompatibleSchemaRollback`. |
| R207-A05: redundant stored payload can cross its SQL scope | A corrupted serialized Record could disagree with its row's network/owner/ID/state/revision. Cached records had similarly unchecked scope. | Strictly decode and compare stored DTOs against indexed columns and authenticated scope, rejecting invalid cached records/drafts. `TestAuditMissingLockAndCorruptRecordFailClosed`, `TestAuditCachedScopeCorruptionDenied`. |
| R207-A06: form action accepts a protocol-relative path | A request path beginning `//host` could render a form posting private fields and CSRF to another host. | Validate local paths in Handler and exported Render before output. Reject double slash, backslash, query/fragment/control and non-local schemes. `TestAuditFormsRejectExternalActions`. |
| R207-A07: ambiguous JSON aliases and Unicode repair | `Action` plus `action` bypassed exact-key duplicate checks; unpaired escaped surrogates were silently repaired by `encoding/json`. | Reject case aliases and isolated high/low surrogates while preserving valid pairs and escaped literal text. `TestAuditStrictJSONCaseAliasesAndSurrogates`. |
| R207-A08: fold skip targets second physical clock time | `Calendar.Next` could select the second physical occurrence in a fold after the first logical occurrence had passed, wasting a skip. | Canonicalize candidates to the first physical occurrence and advance to the next logical date. `TestAuditNextSkipsSecondPhysicalFold` uses New York's actual 2026 fold. Current Go tzdata keeps Vancouver at UTC-7 after its 2026 change; the earlier Vancouver November 2026 daily fixture alone does not prove a fold. |
| R207-A09: CLI preview cannot find templates beyond first 100 rows | Retained cancelled history could hide an active template from CLI preview. | Add a direct owner-scoped, freshly authorized Get service seam and use it for preview. `TestAuditDirectPreviewNotHiddenBehindCancelledHistory` creates 120 earlier cancelled records. |

## Requirement review

| Requirement | Audit disposition and concrete evidence |
|---|---|
| Owner/network CRUD, quotas and revisions | Implemented. Existing CRUD/conflict/private network tests reproduced; direct Get and corrupt-row tests added. Twenty noncancelled templates per owner remains the documented policy. |
| Fresh membership/terms/capacity and private drafts | Implemented through same-SQL-transaction policy callback. Existing revoked-owner/target/capacity tests reproduced; stale cached-target retry fixed. No proposal/acceptance/posting API exists. |
| Daily/weekly/monthly schedules and bounds | Pinned unmodified `robfig/cron` parser reused only for structured fields. Descriptor/TZ injection, boundary, leap-day, missing month-day and half-hour gap tests reproduced. Actual fold Next regression added. |
| Atomic firing/reminder and duplicate prevention | Same transaction contains logical firing, private reminder, optional host inbox callback and audit. Existing two-connection worker/restart/callback failure tests plus silent-write regression pass. Remote exactly-once delivery is not claimed. |
| Bounded downtime and scan continuation | Existing current-window-only and batches-of-three tests reproduced. Cursors advance over examined/suspended rows; host must retain/wrap continuation and explicitly invoke RunDue. No import starts a job. |
| Pause/resume/skip/cancel/edit | Existing state/revision/skip persistence/timezone-edit guard tests reproduced; duplicate physical-fold skip fixed. Cancel does not permit later draft creation. |
| Departure/retention/restore | Existing backup/reopen, migration rollback, 90-day retention/tombstone tests reproduced. Private cached responses now retire for historical target references and silently ignored retirement fails atomically. Minimal tombstones intentionally remain. |
| HTTP/no-JS/CLI | Existing body/auth/CSRF/X-Network/strict-field/form/escaping/CLI tests reproduced; external form actions and alias/surrogate ambiguity fixed. Host identity/CSRF callbacks remain explicit ports, not a separate login. |
| Persistence and schema integrity | Explicit caller-owned SQL connection; no driver imported by the runtime library. Migrations validate known reference shapes. No root migration registry, global feature flag or background registration changed. |
| Privacy and ledger authority | Notes/default amounts stay in owner-scoped template/draft state, not background reminder payloads. Depart scrubs private response bodies. Source contains no core/future-feature imports or ledger writer. |
| Accessibility and live integration | Native forms/labels, escaping and no-JS request paths tested. Real browser, keyboard/screen-reader, WCAG automation, actual core adapters and full ledger regression remain unperformed external qualification. |

## Reproduced evidence and limits

See `VALIDATION.md` for exact final commands/results. Short calendar fuzzing now exercises both Window and Next, with no backward Next result: 60,023 executions pass. Dependency versions/notices are unchanged; no root license decision is made. Root-module/core regression is not qualified by nested-module evidence. No core file was modified or repaired here.

No unresolved finding identified in this standalone review remains in the audited scope. This is an audit of a candidate reference adapter, not independent production cryptographic/security certification. Actual current-core policy/inbox adapters, network opt-in, scheduler orchestration/fairness/recovery, root API/CLI/OpenAPI routing, browser/manual accessibility, full ledger regression and operational backup/erasure/release approval remain required. Keep HOLD and do not batch-integrate the future modules.
