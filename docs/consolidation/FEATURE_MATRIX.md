# Circula Core completion matrix

## Current continuation

Current integration authority, completed source, exact verification results,
remaining work and next actions are recorded in [CONTINUATION_20261119.md](CONTINUATION_20261119.md).
Main remains `ead63b5`; this owned continuation targets PR #90 at `9924785`.
The snapshot below is the historical initial recovery record.

## Historical initial-recovery checkpoint

## Scope and baseline

This checkpoint continues the owner's Core-only consolidation directive on
2026-11-19. The fetched main baseline is
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. The authoritative candidate branch is
`integration/core-recovery-20261119`, published as PR #102. The additive
BRANCH_REFRESH_20261119.md accounts for 120 branches and 102 PRs while preserving
the initial 114-branch snapshot. A source implementation, a focused test
pass, and full integration qualification are separate states below.

The canonical requirements remain `docs/specs/`; this matrix does not replace
them or turn historical status rows into new product authorization. The 15
future modules remain isolated and disabled. No release or deployment is part
of this work.

| Area | Status | Current evidence and remaining boundary |
|---|---|---|
| Exact mutual-credit arithmetic and balanced posting | Unintegrated | Established ledger and property tests exist on main; arithmetic/state hardening in active PR #90 still requires its combined qualification. This candidate does not change monetary representations or posting rules. |
| Atomic posting, credit limits, and idempotency | Complete baseline | Existing service/store regressions and recorded QA-001 run cover the established behavior. Full candidate regression/race results must be recorded separately; prior evidence is not a pass for a new tree. |
| Accounts, membership, and administrator authorization | Nearly complete | Existing lifecycle and authorization mechanisms are retained. New export boundary regressions cover live demotion, suspension, removal, and erasure. Remaining UI catalogue migration is independent of policy decisions. |
| Export authorization and erasure | Unintegrated | New same-snapshot authorization fix passes focused real-SQLite regressions; awaiting normal Core integration. See SECURITY_AUDIT.md and TEST_RESULTS.md. |
| Export interruption and archive integrity | Unintegrated | Web adapter now propagates generation/finalization failure as an aborted response, with successful/early-refusal controls. Focused and real-client committed-response regression passes recorded; integration gates remain. |
| Listing pagination | Unintegrated | Owner selection, type, and encoded query survive continuation; one-row lookahead suppresses empty final pages. Actual HTTP regressions pass. |
| Authentication/password proof | Active | Existing reviewed fixes are being consolidated through PR #90. Do not overwrite its branch or substitute an older source-head CI receipt for the current combined tree. |
| API-036 session inventory/revocation | Active | PR #90 owns bounded indexed inventory, public IDs, and selective revocation. PR #101 supersedes closed #96 and adds the active cursor/bulk-revocation continuation. #99 carries the live-session token-mutation repair; #90 must not merge alone before that repair. These tracks require fresh exact-head review and checks. |
| API compatibility and request validation | Nearly complete | Existing OpenAPI/contract, strict JSON, idempotency, and authorization suites are retained. Full current candidate contract and regression checks remain required. |
| SQLite migrations and history integrity | Active | Main migrations are preserved. Active PR #90 adds dispute-policy and session-public-ID migrations; these remain on its isolated branch until qualified. |
| Storage failure qualification | Partial | Existing real SQLITE_FULL, partial-WAL write, and checkpoint write regressions remain. Arbitrary read/fsync/truncate, device faults, uncertain commits, and power-loss qualification are not closed by adapter fault injection. |
| Shared validation/admin-reset catalogue | Unintegrated | 123 English entries, exact amount strings, contextual escaping, and rendered-page checks. Message and focused web tests pass; broader qualification remains. |
| Remaining UX-047 product text | Active | Listing/profile/directory and JavaScript continuations are implemented in this candidate (73 additional entries). Trade/history and inbox/report continuations remain active isolated work items. English externalization does not approve new languages or complete all locale/number/date scope. |
| Requirement traceability | Unintegrated | Recovered duplicate-definition guard passes real-command negative controls, legitimate-reference controls, race tests, vet, and repository freshness checks. Manual/external requirement evidence remains separate. |
| FNET-204 feed | Deferred | PR #92 targets the isolated future feed branch, with disabled/HOLD metadata. Dedicated module passes do not authorize Core adoption. |
| Other future modules | Deferred | All 15 modules retain their isolated development boundary; branch inventory records implementation and audit follow-ups without enabling them. |
| SEC-026 signals, GOV-021 recovery, reviewer reset policy | Blocked | Their recorded owner-policy gates remain unresolved. Do not merge held implementations merely because source branches exist. |
| Manual accessibility and independent external security review | Blocked | Automated or AI-assisted checks do not satisfy manual screen-reader qualification or SEC-033 external review. |
| Licence, release thresholds, deployment | Deferred | Existing owner/external gates remain. This directive explicitly excludes deployment and release publication. |

Update this matrix after each qualified merge. Do not mark the whole Core
complete while active or safely actionable rows remain.
