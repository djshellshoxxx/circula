# PR #85: GOV-021 source and policy readiness review

Review snapshot: 2026-11-18 08:45 UTC. Reviewer: core_recovery (C), using Caveman investigate-first and review. This was a read-only source/specification review. No Go build, test, race run, new probe, tracked source edit, or remote write was performed in this assignment.

## Disposition

**Preserve as active, valuable core development; keep draft and do not merge this head yet.** This is an implementation of a previously missing core recovery requirement, not a superseded historical branch or an isolated future module. The independent review found unresolved policy alignment, request/history pagination gaps, and a lifecycle regression blind spot. Concurrent draft PR86 is now stacked on this work; its author has resolved the initial migration-number collision. External author ownership remains intact.

| Item | Exact value |
|---|---|
| PR | https://github.com/djshellshoxxx/circula/pull/85 |
| Head branch | `claude/gov-021-admin-recovery` |
| Head | `b20c118b59d43836bd357788914542b904f13872` |
| Tree | `cfd87fc26b42367f97dfa89a6a63ce82fb9ec440` |
| Base and merge base | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` (`main`) |
| PR state | Open, draft, mergeable; not merged |
| Diff | 5 commits; 36 files; 3,070 additions / 59 deletions |
| Review worktree | `/workspace/scratch/9ceb45b04668/audit-c-gov021` (detached at exact head; clean) |
| Metadata snapshot | `audit-c/PR85-METADATA.json` |

Mergeable is GitHub's conflict assessment, not a readiness or CI verdict. Metadata was refreshed at the end of this review and still reports this exact head.

## New behavior and surfaces

An authenticated active member of an active network with no active administrator can request their own appointment. A configured administrator-MFA requirement is checked. The request is idempotent and has a provisional seven-day lifetime. An operator CLI records a first approval; a second approval with a different typed operator label grants a permanent ordinary administrator role. Eligibility and network state are rechecked within the write transaction. Other requests are then superseded. Refusal and expiry are separate audited terminal states.

New member interfaces are GET/POST `/api/v1/admin-recovery/requests`, GET `/api/v1/admin-recovery/events`, and GET/POST `/n/{slug}/admin-recovery`. Request creation requires a session cookie; API tokens cannot initiate it. Web/API CSRF and idempotency conventions are retained. Operator actions exist only under `circula admin-recovery list|approve|refuse`, not HTTP.

The new `admin_recovery_requests` table, repository interface and migration impose state-transition and append-only constraints, a unique open request per member/network, and a matching active administrator-role reference for applied requests. No existing table/data is rewritten by the new migration. The implementation calls a newly extracted `grantRoleTx` helper also used by existing ordinary role assignment; existing administrator-grant notice behavior is preserved.

Source-level positives include same-transaction role/state/audit/notices, explicit tenant checks, reduced member-facing audit views that omit operator labels/reasons, paginated notification delivery to active and suspended members, CAS transitions, approval-time eligibility/MFA checks, and a global expiry job that iterates batches. No new ledger semantics or account/entry mutation was found in the recovery path. Runtime qualification of these properties is still required at the integration head.

One cross-cutting API change moves network/membership resolution ahead of the cookie-only requirement for network routes (`internal/api/route.go`). It preserves outsider not-found behavior while members using bearer tokens still get forbidden on session-only routes. This affects existing session-only network routes too, so the existing authorization matrix belongs in integration qualification.

## Findings and required decisions

### C85-01 — Owner-policy alignment is not established

Canonical `docs/specs/10-governance-operations.md` GOV-021 (lines 269–276) explicitly ties recovery to the SEC-013 break-glass process. SEC-013 in `docs/specs/09-security-privacy.md` requires documented reason, second staff approval, audit, notice and minimum necessary access. `docs/DECISIONS.md` DEC-014 (line 834) still leaves scope, second approval and notice policy open.

This PR deliberately creates a separate process with **no break-glass session or grant**, and its final effect is a permanent ordinary administrator role. The CLI always uses `SystemActor("cli-admin-recovery")` (`cmd/circula/admin_recovery.go:67`); the two staff identities are free-text labels. `domain.SameOperator` rejects case/whitespace variants of the same label, but does not authenticate two independent people. The author explicitly acknowledges that one person can type two labels in `docs/audit/consolidation-completion-2026-11/GOV-021.md:112–127` and the PR body.

This is a procedural approval model whose acceptability needs an explicit owner decision; it is not evidence that the service technically enforces two independent staff principals. Do not interpret the new feature as closing A3-F02 / DEC-014. The author also identifies unresolved waiting/objection period, seven-day expiry, authenticated operator identities and possible nomination of another consenting member.

**Exact addendum correction:** PR85 does not rewrite SEC-013 addendum Decision 7. That existing text still says the only operator rights-changing action is `break-glass grant-admin`, with temporary session-bound grants and no permanent role assignment. The PR appends a separate-process note under “Not done” item 1. The result needs explicit normative alignment, rather than treating that note as an approved exception. Likewise, changing the GOV-021 status row and TODO to “built” does not resolve the policy decision or qualify the feature.

Smallest next step: the owner decides whether this narrowly scoped permanent recovery process is authorized outside a break-glass session and whether documented procedural second-person approval is acceptable. Record that decision and align canonical/addendum text and implementation. Otherwise adapt the implementation to the existing required procedure. Do not silently change broader break-glass authority.

### C85-02 — New operator personal-data read lacks the required access trail

`ListAdminRecoveryOperator` (`internal/service/admin_recovery.go:346–390`) requires a System actor, then reads requester display name, membership status and join date, with an all-networks listing option. It does not require an operator label, reason, session/time limit, or record an audit of this access. The CLI list operation invokes it directly (`cmd/circula/admin_recovery.go:74–88`).

SEC-013 covers member personal data through product interfaces. ARC-009 (`docs/specs/05-architecture.md:180–187`) requires reason, time limit, notice and access audit for operator support reads. The member's request and approval-time audit are useful, but this new read path itself does not implement those controls. This is a source-confirmed policy/control gap, not a remotely reachable HTTP disclosure claim. Decide and document the narrowly required access authorization, then implement an audited, appropriately bounded listing or explicitly approved policy exception.

### C85-03 — Request eligibility/expiry and user interfaces truncate their data

`requestable` (`internal/service/admin_recovery.go:181–201`) and `closeExpiredRecovery` (443–461) each inspect only `ListAdminRecovery(netID, true, "", 500)` once. The repository orders requests by descending ID (`internal/store/sqlite_admin_recovery.go:60–74`). There is no per-requester predicate or continuation in either helper.

Therefore, with more than 500 open rows, a member's older request can be absent from both checks. An unexpired one is missed by the intended open-request precondition; an expired one is not closed before insertion. The schema's unique open-member index (`0019_admin_recovery.sql:43–45`) still prevents duplicates, so this is not a demonstrated double appointment. The member can nevertheless be offered a new request that cannot be inserted, and the expired row remains blocking until the global expiry job or an explicit operator resolution reaches it. The existing global expiry job is correctly batched; that does not make the direct request path's promised lazy-expiry behavior complete.

Smallest implementation correction: use a targeted open-request lookup by network/member, or iterate every relevant page in both helpers. Add existing-request and expired-request regressions with the target beyond the first 500 request rows. The supplied >500-member notice test covers the roster, not this request dataset.

Separately, the CLI fetches a fixed 200 requests with no cursor option or loop (`cmd/circula/admin_recovery.go:75`). The member page fetches only 50 requests and 200 events (`internal/web/h_admin_recovery.go:47,52`) and has no pagination controls. Events are read in ascending sequence order (`internal/store/sqlite_audit.go:99`), so the page repeatedly displays the **oldest** 200 recovery events and hides later actions. API pagination is available, but does not make the web page or CLI listing complete. Add usable continuation or an explicitly labeled bounded recent view with access to the full history, and boundary regressions.

These consequences are derived from exact query/call paths. No runtime reproduction was run under the source-only assignment.

### C85-04 — “Network closing” regression passes through the expiry guard first

The `network closing` case of `TestGOV021RechecksAtEachApproval` (`internal/service/admin_recovery_test.go:409–418`) creates a request, approves once, advances 31 days, then asserts only invalid-state. The request lifetime is seven days. `ApproveAdminRecovery` calls `expireOne` before network lifecycle/state checks (`internal/service/admin_recovery.go:523`). Thus request expiry already explains the expected rejection; the test does not establish that the closing-network guard was exercised.

Keep the request unexpired when the network becomes closing, and verify no role is granted and the request was not merely expired by the attempt. For example, advance near the scheduled close before creating the request. The source currently does contain lifecycle/network checks; this is a regression coverage gap, not a demonstrated absent production guard.

### C85-05 — Migration numbering corrected in the newer SEC-026 stack

PR85 adds `internal/store/migrations/0019_admin_recovery.sql`. An earlier concurrent SEC-026 head, `185273a43a337d25bac7e2cff8ffb83c64ac05e6` (tree `621cb4c05219f4f9dd459cbf8f7d588fb0d1aad2`), added `0019_activity_signals.sql`; those two objects were independently inspected in this review. That collision is now historical, not a current blocker for PR85.

At 08:45 UTC the author opened draft PR86 with head `2e2e6f0eba6bed29bed8f2dd5d60699f372760a6`, tree `90c111732d70bc3f019749e66ee8a0f72228614e`. Agent A independently inspected that exact source and confirmed both `0019_admin_recovery.sql` and **`0020_activity_signals.sql`**, with preceding versions 0001–0018. PR85 head b20c118 and main cdc8112 are ancestors of the new PR86 head; PR86 is four commits ahead of PR85.

Thus the current coordination issue is the stack dependency: PR86 includes this recovery implementation and its unresolved readiness conditions. Keep the author-owned stack coherent, regenerate affected traceability when it changes, and qualify upgrades/backup/restore on the eventual combined head. Do not request a redundant migration rename based on the older SEC-026 head. No competing source patch was made.

## Additional completion notes

- Schema label distinctness is `lower(trim(...))`, while service `SameOperator` uses Unicode case folding and collapses all internal whitespace. The schema does not repeat the service's full normalization, despite the PR body's broad claim. Existing service tests cover Unicode whitespace; schema tests cover simpler ASCII variants. Correct the claim or strengthen the documented defense-in-depth rule. This is not a demonstrated service-path bypass.
- Recovery table contents are intentionally absent from application exports, while applied roles and audit effects are exported. `internal/service/export.go:98–105` lists full-export tables and omits recovery requests. Clarify how an in-progress recovery request should behave under GOV-022 migration/portability. Whole SQLite backups naturally retain the table. Do not expand export access to operator data without an explicit scope decision.
- Member identity checking is the authenticated member/account relationship plus an operational identity-checking procedure described in the runbook. No evidence attachment or independently authenticated staff principal is stored. Describe that boundary accurately.
- No browser, accessibility, migration or backup execution was performed here. Existing web tests inspect rendering/CSRF and do not constitute full browser qualification.

## Existing tests useful for the eventual required gate

| Area | Supplied or existing tests and scope |
|---|---|
| Approval and audit | `TestGOV021AppointmentNeedsSecondApproval`, `TestGOV021RecoveryAuditVisibleToNetwork`: first-vs-second approval, same-label variants, visible reduced audit, both audit scopes |
| Ledger invariants | `TestGOV021RecoveryChangesNoBalanceOrEntry`: digests across ledger tables through request, approval, refusal, expiry and supersession; `VerifyLedger` |
| Eligibility and lifecycle | `TestGOV021RequestEligibility`, `TestGOV021RechecksAtEachApproval`, `TestGOV021RequestExpiry`; closing subtest needs the correction above |
| Atomic/concurrent effects | `TestGOV021ConcurrentSecondApprovals`: eight competing second approvals, one role and applied audit |
| Notices and privacy | `TestGOV021NoticesReachAllMembers`: >500-member delivery with inactive/other-network exclusions; `TestGOV021ErasedRequesterLeavesNoPersonalData` |
| Break-glass separation | `TestGOV021SeparateFromBreakGlass`: permanent recovery role survives temporary break-glass expiry; documents the proposed policy, not owner approval |
| Schema and repository | `TestGOV021RecoverySchemaGuards`, `TestGOV021RecoveryRepository`: illegal transitions, append-only guards, FK/role association, CAS and paging/expiry boundaries |
| CLI and web | `TestGOV021AdminRecoveryCLI`, `TestGOV021AdminRecoveryCLIUsage`, `TestGOV021NoOperatorCommandAppointsDirectly`, `TestGOV021AdminRecoveryPage` |
| API | Added cases in `internal/contract/contract_test.go:TestAPIMatchesOpenAPI`; updated route authorization inventory and OpenAPI; rerun existing authorization matrix because middleware ordering changes |
| Migration integration | Existing `TestUpgradeFromEverySchemaVersion` discovers the new migration and covers previous schema versions; also retain startup migration, CLI upgrade, rollback, backup/restore and aggregate required gates on the final combined tree |

The author reports unit/vet/lint/hygiene/traceability and package race passes at **43d8e87**, before this head's merge of current main, and says exact-head full unit/race runs were still underway. Those are author claims, not independent verification by this review. Traceability still records test results as “not recorded”. No green result for an older tree should substitute for required checks on the final integration tree.

## Handoff

Keep PR85 open as the external author's draft. Share C85-01/02 for the owner policy decision and C85-03/04 for focused author corrections. C85-05 records the current SEC-026 stack dependency and already-corrected numbering. Preserve A3-F02 / DEC-014 as unresolved. After authorized policy alignment and repairs, independently review the new exact head and run the required qualification gates; do not close, supersede or merge this head based solely on its broad feature coverage.
