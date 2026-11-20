# Circula authentication and authority review

Review date: 2026-11-19 (America/Los_Angeles; 2026-11-20 UTC). This is a defensive, read-only source review for repair workers. The source-review worker changed no application source, database, configuration, service or remote repository. The coordinating handoff publishes notes only. No exploit, adversarial reproduction, traffic generation, or availability benchmark was run. The reviewer did not execute Go tests; baseline test execution is owned by the coordinating reviewer.

## Revisions and scope

The primary reviewed Core candidate is `22c767a90f69d286642ef96e6b3c174836637a3c` (PR #125, `integration/core-completion-audits-20261119`). The comparison baseline is main at `aa01563fcc891da60fefebb8335eca27855678e8`. The application worktree used for reading is `/workspace/scratch/1b6df59fd7e6/Circula-auth-source`; its tracked and untracked status was clean at review completion. The notes are outside that tree.

All four findings below affect both pinned main and the Core candidate. A path-specific comparison found no changes between those revisions in `internal/service/auth.go`, `principal.go`, `dualcontrol.go`, `recovery.go`, `password.go`, `throttle.go`, `sessions_revoke.go`, `tx_queries.go`, `reads.go`, `export.go`, `internal/store/sqlite_member.go`, or `internal/web/ratelimit.go`. The change to `internal/web/h_auth.go` is authentication-message translation, leaving the relevant admission logic unchanged. Source line references below name the Core candidate unless a different revision is stated.

The held administrator-recovery candidate was separately read at `b8cb63bfc6d2a969d972f55937cdc2070d5eba34` (PR #107). It is not part of the pinned main or Core candidate. Its policy and assurance limits are recorded separately rather than attributed to released Core behavior.

## Finding index

| Local ID | Finding | Priority and confidence | Status |
|---|---|---|---|
| AUTH-01 | Network password-reset authority ignores retained access in departed memberships elsewhere | High confidentiality concern with privileged prerequisites; high confidence in the source mismatch | Newly identified extension of the earlier cross-network reset issue |
| AUTH-02 | Approver ancestry check treats bounded or cyclic traversal as success | Medium integrity concern; high confidence in the control-flow defect | Newly identified edge of the existing independence safeguard |
| AUTH-03 | Membership enumeration returns partial success after an iteration error | Low defensive-hardening issue; high confidence in omitted error check, conditional security impact | Newly identified; no operational failure trigger established |
| AUTH-04 | HTML sign-in does not bound concurrent memory-hard hashing | Medium, unmeasured availability risk; high confidence in missing admission bound | Newly identified admission gap; deployment impact unmeasured |

These are source findings, not four successfully reproduced attacks. The first two concern concrete authority decisions. The third is a fail-open data-completeness hazard with material prerequisites. The fourth identifies absent resource admission controls without claiming an outage threshold or measured resource exhaustion.

## AUTH-01 — Reset authority follows active membership, but credentials still open departed-network history

### Evidence and consequence

`AdminResetPassword` correctly recognizes that the password belongs to a global person, not to one membership. Its guard refuses an ordinary network administrator resetting another current administrator and checks every other membership. However, the check deliberately excludes another membership whose status is `departed`: `internal/service/recovery.go:381–397`, specifically the condition at lines 393–395. The same function then invokes the global `resetPerson` routine and returns the temporary password to the administrator: lines 399–415. `resetPerson` replaces the global password and revokes that person's sessions, API tokens, recovery codes, and pending MFA challenges at lines 296–328. [Reset source](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/recovery.go#L331-L415).

A departed membership retains legitimate read authority. `ListTransactions` explicitly permits a departed member to read their own history (`internal/service/tx_queries.go:65–83`), as does `ListEntries` (`internal/service/reads.go:124–158`). Own-data export creation and download also accept a departed member unless the membership has an erasure record (`internal/service/export.go:107–145`). Consequently, the departed status is not evidence that the global login carries no authority in that other network. [History authorization](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/tx_queries.go#L65-L83); [export authorization](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/export.go#L107-L145).

The resulting boundary failure is that an administrator of the person's current network can control the same global credential that still authorizes private retained history in another network. The administrator need not hold any role in that other network. This does not imply access to every record there, a departed-member write capability, or an MFA bypass.

### Preconditions and severity qualification

The caller must already be an active administrator authorized to use member recovery in the current network. The target must be an active ordinary member there and have a departed membership elsewhere whose confidential data remains accessible. Reading the other history with the returned temporary password also requires knowledge of the target's login and absence of an effective additional factor available only to the target. Network password reset preserves enabled MFA; the report must not imply otherwise. Irrespective of MFA, the reset replaces the person's global password and invalidates credentials across networks. It returns no session, and the web result does not reveal the login. Actual history access requires a later successful authentication. Forced password change verifies the replacement password and is not an independent proof of the original owner's identity; see the independent cross-check for the precise stages.

High priority is justified by crossing the tenant boundary through a tenant-scoped recovery permission. The privilege and account-state requirements should remain in any severity summary. This is a static, high-confidence mismatch between two explicit authorization rules, not an executed account takeover.

### Relationship to earlier audits

The source comments and existing SA2-01 guard already document the danger of giving one network administrator control over a global account. The existing `TestAdminResetRefusesMultiNetworkTarget` in `internal/service/recovery_test.go:439–460` covers an active second membership. The reviewed notes did not identify the retained-history authority of a departed second membership. AUTH-01 extends that earlier finding; it is separate from the already known reviewer-recovery issue AUD4-F02.

### Repair direction and safe acceptance criteria

Recovery authorization should depend on retained authority across the global identity, not solely the membership's active/departed label. At minimum, require platform-level recovery when another membership retains any protected access. Preserve the product's legitimate right for departed members to read their own retained history. Specify separately what qualifies as completely erased or inaccessible before permitting an exception.

Acceptance should establish that a network-local administrator cannot reset a global identity with retained protected access elsewhere; an operator-authorized recovery path still works and is audited; a genuinely single-network eligible member still receives supported recovery; enabled MFA remains intact; and refused recovery leaves the password, credentials, MFA challenges, audit state, and throttles unchanged except for any explicitly designed refusal audit. Include departed-but-retained and erased membership cases. Avoid treating the existing active-membership test as coverage for both.

Suggested worker ownership: authentication service plus privacy/membership policy. Review `AdminResetPassword`, the membership-authority predicate, and recovery tests together.

## AUTH-02 — An incomplete ancestry proof is accepted as an independent approver

### Evidence and consequence

The A3-01 safeguard says the deciding administrator's grant must not descend from the adjustment proposer. `requireIndependentApprover` performs that check in `internal/service/dualcontrol.go:22–53`. The walk is bounded by `maxGrantChain = 32` at lines 19–20 and stops when the current node has already been seen at line 32. All normal loop exits fall through to `return nil` at line 53, including depth exhaustion and a repeated node. Those conditions establish that traversal has stopped; they do not establish an independent root. [Traversal source](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/dualcontrol.go#L19-L54).

`ApproveAdjustment` treats the nil result as authorization at lines 82–90 before continuing toward posting. The same helper is used for rejection (`dualcontrol.go:191–193`) and UI decision eligibility (`260–293`). `LatestAdminGrant` selects the most recent historical administrator grant, including revoked assignments, in `internal/store/sqlite_member.go:175–187`. Reviewed grant/revocation paths do not enforce a maximum ancestry depth or an acyclic invariant that would make every bounded exit safe. [Grant lookup](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/store/sqlite_member.go#L175-L187).

This is an integrity issue in the independence proof when a nontrivial grant history cannot be fully resolved. A bounded search is appropriate for resource safety; a bound should produce a refusal or an explicit indeterminate result rather than successful authorization.

### Preconditions and limits

The action still requires an active member administrator, a valid pending adjustment, a distinct proposer and target, an active network, and the remaining posting guards. The ancestry condition must reach an unresolved depth or cycle before establishing the relevant relationship. No production grant graph or local constructed graph was exercised, and the review does not assert that any deployed network has this state. The independent challenge review narrows the cycle claim: every visited parent is compared before repetition stops the loop, so a short cycle containing the proposer is detected. A cycle excluding that proposer is an indeterminate-provenance/root concern, not independently proof that the same one-parent graph hides a path to that proposer. The depth-exhaustion finding remains substantive.

The code explicitly permits an operator-rooted grant with an empty granter and a founder's self-grant. A repair must distinguish these intentional trust roots from incomplete or inconsistent history. Simply rejecting every empty parent without understanding the grant record would break supported independent-root administration.

### Relationship to earlier audits

This is distinct from AUD4-F01, the known one-direction policy under which an ancestor can approve a descendant's request. The owner-decision appendix and `TestA3_01_IndependentApproverSucceeds` already encode that policy. AUTH-02 concerns success without completing even the existing one-direction check. It can be repaired without silently selecting a new symmetric-independence policy.

### Repair direction and safe acceptance criteria

Represent walk completion explicitly: a proven permitted root may succeed, a prohibited relationship must fail, and depth exhaustion, a repeated node, or unexplained missing history must fail closed. Keep the work bound. Consider whether the authoritative grant model should also prevent inconsistent ancestry on writes, while still treating stored inconsistency safely on reads.

Acceptance should cover a normal independent root, a permitted founder root, a prohibited descendant relationship, the boundary at the configured maximum depth, over-bound history, and a repeated node. Authorization failure must leave the pending decision and ledger untouched. The UI eligibility helper and authoritative approval/rejection methods must agree. These are remediation invariants; no attack sequence or test payload is supplied here.

Suggested worker ownership: governance/dual-control service with membership-store review.

## AUTH-03 — Security consumers can receive an incomplete membership list as a successful result

### Evidence and consequence

`internal/store/sqlite_member.go:109–138` implements `listMembers`. It consumes `rows.Next()`, checks individual scan errors, and calls `rows.Close()` after the loop, but never checks `rows.Err()`. It then loads roles for the rows already collected and returns success. `ListMembershipsOfPerson` delegates to this helper at lines 99–101. The nearby `loadRoles` helper correctly returns `rows.Err()` at line 60, illustrating the missing distinction. [Membership iteration source](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/store/sqlite_member.go#L99-L138).

The transaction query helper returns ordinary `*sql.Rows` (`internal/store/sqlite.go:265–268`), so no intermediate wrapper supplies the omitted check. Go's `database/sql` can close rows as part of a failing `Next`; a later `Close` is not a replacement for examining the iteration error. This behavior was independently cross-checked by the ledger reviewer against the [official database/sql source](https://go.dev/src/database/sql/sql.go).

Completeness matters because the result participates in authority decisions. The cross-network recovery restriction scans this list in `recovery.go:389–397`. Role-specific session lifetimes scan it in `sessions_revoke.go:97–123`, returning the ordinary member lifetime unless a shorter administrator/reviewer lifetime is found. Erasure eligibility also depends on cross-network membership state. A partial list should not be indistinguishable from an authoritative complete list. [Lifetime consumer](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/sessions_revoke.go#L92-L123).

### Preconditions and severity qualification

The defect requires a row-iteration failure after query creation, rather than a query or scan error already handled. A security impact additionally requires the omitted row to matter and subsequent operations to remain viable. Ordinary request cancellation often causes later statements to fail as well, so cancellation alone must not be described as a working authorization bypass. No concrete modernc SQLite failure trigger or affected live execution has been established.

Classify this as Low, defensive hardening with conditional security impact, unless later controlled implementation work supplies stronger evidence. The omitted error check is certain; an externally inducible bypass is not. The concern is particularly worth repairing because the helper is shared by several security-sensitive consumers.

### Repair direction and safe acceptance criteria

Check iteration status before deriving authority or returning data, preserve relevant close errors, and return no successful partial authority set. Review other shared authorization enumerators for the same error-handling pattern without assuming every iterator is affected.

Acceptance should show that a repository fixture returning an iteration error propagates the failure even after producing some rows, while normal empty and complete queries still work. Recovery and role-lifetime consumers should fail closed on that repository error. Such bounded error-path checks are sufficient; stress, corruption, or service attack tests are unnecessary for this repair.

Suggested worker ownership: SQLite repository, with authentication-consumer regression review.

## AUTH-04 — Password hashing lacks aggregate admission and HTML failure counters reserve no in-flight work

### Evidence and consequence

`internal/web/h_auth.go:46–59` checks `ipFail.blocked` and `acctFail.blocked`, invokes `LoginStep`, and only records qualifying failures after it returns. `blocked` only reads the counter; `fail` increments later (`internal/web/ratelimit.go:62–75`). The separate `allow` method atomically consumes an attempt (`43–48`), and registration uses that method (`h_auth.go:109–115`). The sign-in failure budget therefore does not reserve in-flight hashing work. [HTML sign-in source](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/web/h_auth.go#L46-L85); [limiter source](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/web/ratelimit.go#L43-L75).

The service has a useful durable defense against concurrent guessing of the same account: `LoginStep` reserves valid-login attempts before hashing (`internal/service/auth.go:126–158`). `reserveAttemptCtx` applies both login/address and login-wide counters (`internal/service/throttle.go:76–101`). These are per-key budgets, not an aggregate bound on password-hashing resource use. The uniform malformed-login path also invokes dummy verification without durable per-login admission (`auth.go:128–130`).

All product Argon2 evaluations pass through `argon2id`, which directly calls `argon2.IDKey` without a semaphore, bounded queue, or context-aware admission (`internal/service/password.go:56–62`). The default parameters use `64 * 1024` KiB of Argon2 memory, three passes and two threads (`34–37`). That parameter is source evidence; it is not a measured process-memory or outage claim. [Hashing source](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/password.go#L34-L62).

The web/API reviewer independently checked the surrounding Core HTTP wrappers: `cmd/circula/ops_http.go` provides timeouts and maintenance handling, and `internal/web/middleware.go` provides body limits and CSRF checks, but neither supplies an aggregate request/work semaphore. A response deadline does not itself cancel an Argon2 computation. API per-IP throttling narrows that adapter's exposure; it does not supply a shared hashing capacity bound for the HTML path or all service callers.

### Preconditions and severity qualification

This is an unauthenticated availability risk where sign-in is reachable and deployment ingress does not enforce a sufficient independent concurrency budget. Runtime capacity, ingress configuration, process limits, and service scale are unknown. No stress test was run, no effective request rate was measured, and no claim is made that a particular deployment can be taken down with a particular volume.

Medium is a provisional availability rating for an unmeasured resource-admission gap. Keep that qualification prominent. Existing throttles must be credited for the guessing protection they do provide rather than described as absent.

### Repair direction and safe acceptance criteria

Add a small explicit capacity budget shared by expensive password operations, with bounded waiting or prompt refusal when full. Admit work before allocating hashing resources, respect cancellation before admission, and make budget behavior independent of whether an account exists. Review the HTML failure-only counter separately so that concurrent work cannot exceed its intended admission policy merely because failures have not yet completed. Successful-user behavior and lockout resistance should be designed together.

Acceptance should verify a configured upper bound on simultaneous hash evaluations using a benign instrumented hash seam, bounded queue length, prompt handling of canceled waiters, release of capacity after every completion/error path, and consistent known/unknown-account behavior. Test resource accounting rather than intentionally exhausting the host. Record the operator-facing sizing rule and any ingress dependency.

Suggested worker ownership: shared authentication service admission, web sign-in adapter, and deployment capacity guidance. The web/API reviewer has assigned this item to this report to avoid duplicate findings.

## Known open issues and policy limits reconfirmed

### Reviewer recovery and independence remain unresolved (AUD4-F02)

`AdminResetPassword` protects the current administrator role, not the reviewer role (`recovery.go:386–388`). Reviewer/decider conflict checks act on member identity. The previously documented result is that password recovery authority can undermine reviewer independence when the reviewer has no second factor. A present-role-only refusal is incomplete because role history and later regrant matter. This is a known Medium issue, not a newly executed finding. The historical reproduction belongs to the earlier audit, not this review. [Prior audit disposition](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/docs/audit/consolidation-completion-2026-11/AUD4-AUTH-FIXES.md#L48-L65).

Workers should implement the selected recovery/decision-eligibility policy across role revocation and regrant, and reconcile disputes and governance challenges. Possible policies already documented include stronger reviewer recovery authority, required MFA, or disqualification tied to the recovery actor. This audit does not choose that product policy silently.

### One-direction approver independence remains a documented owner decision (AUD4-F01)

The existing check walks the deciding administrator's ancestors. It does not impose symmetric separation between the proposer and decider. The current specification appendix acknowledges founder/descendant limitations and the need for an independently rooted administrator to satisfy stronger separation. Keep this existing Medium policy item separate from AUTH-02. [Documented disposition](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/docs/audit/consolidation-completion-2026-11/AUD4-AUTH-FIXES.md#L71-L80); [owner-decision appendix](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/docs/specs/addenda/SEC-011-owner-decision.md#L59-L66).

### Temporary break-glass authority still permits lasting effects (A3-F02 / DEC-014)

`requireLasting` blocks several permanent administrative operations, but the reviewed password-reset, adjustment, limit, reversal, and invitation paths are not universally subject to it. The prior audit explicitly records reset, trade reversal, and long-lived invitation effects. `OperatorResetPassword` requires a System actor and an audited reason but does not call `RequireBreakGlass`. This is an existing scope/operating-policy issue, not evidence of unauthenticated operator access. Direct host/database access remains a separate trusted-operator assumption. [Current SEC-013 limits](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/docs/specs/addenda/SEC-013.md#L34-L48).

### Session rotation on role change is a specification gap, not proven retained role authority (AUD4-F07)

Role changes do not rotate every session as described in SEC-003. The service rereads current membership roles when authorizing network actions, so this review found no reason to recast that mismatch as persistent privilege after revocation. Document or implement the selected session-rotation policy; do not confuse a live credential with a cached administrator role. The prior audit records the same informational classification.

### MFA and key custody remain deployment-dependent

Core implements TOTP and recovery codes. `RequireMFAForAdmins` is configurable rather than an unconditional product rule (`internal/service/service.go:70–77`). Role-specific session lifetimes are also configurable. TOTP seeds use AES-GCM when `SecretKey` is configured; the implementation intentionally supports a `raw:` base32 representation when no key is configured (`internal/service/totp.go:108–173`). Operators must accurately state which mode is deployed. The review did not inspect production configuration or key custody, so it does not assert that any live database has unencrypted seeds.

MFA recovery codes are ten base32 characters, 50 bits (`totp.go:176–223`); password-recovery codes use 80 random bits (`recovery.go:19–23`). Both are separate single-use mechanisms with throttling and different authentication context. This distinction is recorded for policy review, not claimed as a practical code-guessing result. No WebAuthn implementation was present in the reviewed Core source; WebAuthn was not certified or tested.

## Held administrator-recovery candidate: assurance limit, not merged Core behavior

At `b8cb63bfc6d2a969d972f55937cdc2070d5eba34`, `internal/service/admin_recovery.go` checks that the caller is a System actor and accepts a cleaned operator label in `recoveryDecision` (455–467). The second approval differs by `domain.SameOperator` comparison against the stored first label (621–624), then grants a permanent administrator role. The CLI obtains the operator label as text. This technically distinguishes labels, not independently authenticated human operators. The branch documentation already records this policy HOLD and the lack of a waiting period; it is not a newly discovered claim that Core has a two-person bypass.

The candidate does provide useful transaction-level rechecks of network eligibility, requester status, existing administrators, and the configured MFA requirement; it also records recovery events and member notifications. Those safeguards should be preserved. Before integration, the owner must specify whether two authenticated operator identities are required or whether two-label recording plus an external procedure is the intended control, and the product language must match that guarantee. No remote integration or policy approval was performed during this audit.

## Corroboration for the coordinating operations review

Platform audit events use an empty network scope (`internal/service/audit.go:40–45`). The routine per-network verification and anchor enumeration do not cover that empty scope; the anchor SQL joins against real networks (`cmd/circula/auditanchor.go:75–102`). A search of non-test `verifyAuditChain` callers found the normal per-network verification path and the explicit platform-chain verification inside `InvalidateRestoredCredentials`. Restore therefore supplies an important exception, but not continuous platform-event verification. This corroborates the coordinating reviewer's platform-chain monitoring/integrity gap and should be counted once in the consolidated report. It requires corruption or another ability to affect stored audit state; it is not a remote tampering demonstration.

## Controls verified by source inspection

This review also checked defenses that constrain the findings. `LoginStep` reserves per-account attempts before verification and rechecks the exact person/hash in the session-issuing transaction after Argon2, preserving the recent AUD4-F06 repair (`auth.go:134–187`). Password recovery consumes a code, replaces credentials, revokes prior credentials, and audits within one write unit (`recovery.go:222–263`). Current-password proofs in MFA enrollment and recovery-code generation are rebound to the current hash before sensitive writes, rather than trusting a proof obtained before a concurrent password change (`mfa.go:181–336`; `recovery.go:124–139`).

Session self-management performs a live owner/session gate inside its read/write unit (`sessions_self.go:79–113`), and token mutations reuse that gate for session actors (`tokens.go:22–38`). Network service authorization checks current membership and effective break-glass roles rather than trusting role claims supplied by the HTTP caller (`principal.go`). HTTP password/session/token-management restrictions were cross-checked against the API route choices; trusted service-call capabilities should not be misreported as publicly exposed bearer-token endpoints. These are source observations, not a claim of complete formal assurance.

## Coverage record and unresolved questions

Production files read in the authentication/service lane include `auth.go`, `password.go`, `principal.go`, `mfa.go`, `mfa_login.go`, `totp.go`, `tokens.go`, `recovery.go`, `throttle.go`, `signup_controls.go`, `membership.go`, `join.go`, `networks.go`, `network_access.go`, `sessions_self.go`, `sessions_revoke.go`, `sessions_others.go`, `replay_auth.go`, `breakglass.go`, `dualcontrol_operator.go`, `dualcontrol.go`, `erasure.go`, `accounts.go`, `notices.go`, `timezone.go`, and `ops_read.go` under `internal/service`. Relevant configuration and actor definitions were inspected in `service.go` and the domain package.

Repository coverage included `sqlite_person.go`, `sqlite_mfa.go`, `sqlite_recovery.go`, `sqlite_member.go`, and `sqlite_breakglass.go`, plus the underlying query helper in `sqlite.go`. Cross-boundary authorization was traced through relevant sections of `reads.go`, `tx_queries.go`, `export.go`, `disputes_decide.go`, and `challenges.go`; the web sign-in/recovery handlers and limiter; and API route/password adapters. This is section-level crosswalk coverage, not a claim to have reviewed every line in those other lanes.

Documentation read included README, SECURITY, SEC-007/008/013 addenda, the SEC-011 owner-decision appendix, relevant security/privacy and functional requirements, prior security-audit findings, consolidation authentication disposition, and held administrator-recovery guidance. Existing targeted tests were inspected as evidence of intended behavior, including active cross-network reset and prior password-race repairs. Historical test logs or audit claims were not counted as executions by this reviewer.

The remaining assurance gaps are explicit. Production credentials, data, topology, ingress, operator identity controls, hardware capacity, and live MFA configuration were unavailable. No stress, exploit, fault-injection, browser, or service test was run in this lane. Parent-coordinated baseline checks must be reported with their own exact revision and result. The pinned held administrator-recovery review does not cover every future branch. Cryptographic primitives were inspected at the application-use boundary, not independently cryptanalyzed.

One concurrency question is retained without a vulnerability rating: general network operations accept the actor produced by authentication, whereas self-session/token management additionally rechecks the live credential in the action's transaction. An already admitted request may therefore finish after a later revocation event. That is not automatically a security defect; define whether the contract requires revocation to block only future authentication or to form a barrier for all still-in-flight writes. Do not label it a persistent session-revocation bypass without that contract and stronger evidence.

## Recommended work order

Resolve AUTH-01's global-identity recovery boundary first. Repair AUTH-02's indeterminate traversal result independently of the broader quorum-policy decision. Patch AUTH-03's repository error handling as a small shared defensive correction. Design and verify AUTH-04's bounded hashing admission with explicit sizing. Keep the known reviewer-recovery, approver-independence, break-glass-scope, and held administrator-recovery policy items visible in the release decision; the new fixes do not close those previously documented limits.
