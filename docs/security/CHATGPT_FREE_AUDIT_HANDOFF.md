# Circula security audit handoff

Date: 2026-11-15
Audit branch: `chatgpt/circula-security-audit-20261115`
Audit handoff commit: `c55e225fc2df23665a16d02c1bb8f05a67a98669`
Authoritative application baseline: PR #35 branch `claude/cool-ramanujan-n2peag`
PR #35 baseline commit: `c033cfa448fb89d1a5cc0ca87458b40d2709fa2e`
Repository default branch: `main`
Audit handoff PR: #42 (draft; targets PR #35 integration branch)
Scope: security review only. No application code, financial rules, migrations, or production data were changed.

## Repository state

- Repository: `djshellshoxxx/circula` (Go, SQLite).
- PR #35 is the documented integration baseline. Its handoff identifies it as a draft PR into `main`; its branch is 345 commits ahead of main and 0 behind at the checked comparison.
- PR #38: foundational specs. PR #40: QA-042 requirement-ID drift auditor. PR #41: future-only receipt/certificate specs; it explicitly says no current feature code is introduced.
- PR #35 handoff says QA-042 full traceability regeneration remains incomplete, alongside owner gates and SEC-033.
- PR #35 baseline commit `c033cfa` has a completed successful GitHub Actions run (run 43685188544). This is CI evidence, not proof of security.
- No historical branches were merged or changed during this audit.

## Completed: Priority 1 — ledger and financial integrity (source review only)

### Positive controls observed

- Posting is centralized in `internal/service/posting.go:postTx`: it reads both accounts, checks bounded arithmetic and payer capacity, obtains a ledger sequence, writes equal/opposite entries, updates cached balances, updates transaction state and writes a status event.
- SQLite writes use an in-process gate and `BEGIN IMMEDIATE` in `internal/store/sqlite.go:Update/run`.
- Service-level idempotency records are written in the same unit of work; the implementation hashes request payloads and rejects reuse with a different payload (`internal/service/idempotency.go`).
- `internal/service/atomicity_test.go:TestAcceptFaultInjectionLeavesNoTrace` injects failures across entry, balance, transaction, status-event, audit and idempotency writes and checks rollback.
- `internal/service/concurrency_test.go` contains concurrent acceptance and credit-limit tests.
- SQLite migrations define append-only triggers for entries and audit events; transaction posting invariants are also checked by `VerifyLedger`.
- These are implementation observations, not a claim that the ledger is secure. Tests were not executed in this environment.

### Historical A1 findings rechecked against PR #35

- A1-01 — departure settlement posting after closure/suspension/admin-required: **fix present in source**. `settleOnDeparture` now calls `p.requireNetworkActive()` before posting.
- A1-02 / A1-11 — scheduled decimal-scale changes reinterpret stored amounts: **fix present in source per remediation record**; inspect `UpdatePolicy` and regression tests again during a runtime-capable pass.
- A1-03 — re-review reverses a trade while another live dispute is open: **fix present in source**. `decideReReview` refuses reversal when `t.Status == disputed`.
- A1-04 — reversal can post while network is administrator-required: **CONFIRMED by source trace; runtime PoC not run**. `networkOpenErr` / `requireNetworkActive` rejects `AdminRequired`, but `internal/service/reversal.go:reverse` checks only `net.Suspended()`. Both `ResolveDispute` and `RetryDisputePosting` call `reverse` without a network-active gate. An administrator/reviewer can therefore resolve or retry a dispute and post a reversal while the network is read-only because it has no active administrator. Severity: Low under the existing A1 classification.
- A1-05 — expired-proposal sweep starvation: **CONFIRMED by source trace; runtime PoC not run**. `internal/store/sqlite_tx.go:ListExpiredProposals` selects the oldest expired proposals globally with a limit and no network filter. `internal/service/transactions.go:ExpireProposals` then skips frozen networks. If the first limited batch is continually occupied by frozen-network proposals, active-network proposals behind it can remain proposed indefinitely. Severity: Low.
- A1-06 — incomplete proposal cancellation on member departure: **CONFIRMED by source trace; runtime PoC not run**. `internal/service/departure.go:cancelOpenProposals` fetches only 500 proposed transactions and does not paginate. More than 500 open proposals involving a departing member leave excess proposals active, retaining reservations until expiry and referencing a departed party. Severity: Low.
- A1-07 — final closure export is not retried after a one-time export failure: **CONFIRMED by source trace; runtime PoC not run**. `FinaliseClosures` first calls `finaliseOne`, which sets `closed_at`; it then calls `recordFinalExport` in a separate unit. `ListClosures(true)` only returns rows with `closed_at = 0`. If export/recording fails after closure commits, the next sweep no longer selects that closure, so its final checksum/reconciliation may never be recorded. Severity: Low in the historical A1 report.
- A1-08 to A1-10 and remaining informational findings: not fully rechecked in this pass. Do not mark resolved.

### Tests and limitations

- Read PR #35 source patches for the posting path, SQLite transaction wrapper, transaction store, migrations, reversal/dispute/departure/closure paths, and the historical A1 report.
- Reviewed existing test source for atomicity, concurrent acceptance, departure settlement and reversal behavior.
- No shell/Go runtime was available through the connected GitHub interface. No `go test`, `go test -race`, fuzz, mutation, or PoC execution is claimed.
- Findings A1-04 through A1-07 are confirmed by code-path inspection but still need focused reproductions and regression tests before remediation is accepted.
- No destructive or production tests were run.

## Completed: Priority 2 — authentication, authorization and IDOR (source review only)

### Positive controls observed

- `service.openAny` resolves membership and token network scope server-side on every service call; callers do not supply roles.
- `requireAdmin` enforces active membership, the administrator role and configured MFA requirement for administrator operations using that gate.
- API bearer tokens are stored by hash, have expiry/revocation checks, and produce actors carrying only the stored scopes and optional network restriction.
- Token creation rejects scopes wider than the creator's and requires membership in a requested network.
- Session authentication checks revocation, absolute expiry and idle expiry. Password change revokes all sessions.
- Cookie-authenticated API unsafe methods use a session-bound CSRF token. Web forms use same-origin checks and CSRF tokens.
- `GetTransaction` applies object-level visibility checks; members cannot retrieve unrelated transactions. `GetMember` returns full details only to the subject or an administrator, and a reduced directory view to eligible active members.
- `TestAuthorizationMatrix` covers ten actor classes across service operations. `route_inventory_test.go` inventories API, web and ops routes and checks route authorization metadata. Tests were inspected but not executed.

### Confirmed findings

- **A3-F02 — Medium — break-glass administrators can make lasting changes. CONFIRMED by current source trace; historical A3 audit includes PoC evidence, but no current-head PoC was run.** `principal.requireLasting()` exists but is called only by a subset of write paths. Current code paths such as `CreateAdjustment`, `ApproveAdjustment`, `SetCreditLimit`, `AdminResetPassword`, `ReverseTransaction`, `DecideJoinRequest`, network suspension/resumption and other admin actions do not consistently call it. In addition, `requireIndependentApprover` returns success immediately for `p.bgAdmin`, bypassing the new-admin delay and appointment-chain independence check. A temporary break-glass admin can therefore create permanent balance changes, permanent credit-limit changes, reset another member's password, reverse a trade, or approve a pending adjustment immediately. The temporary role expires, but these effects do not. This contradicts the current code comment that break-glass access must not leave lasting changes. Remediation is blocked on an owner decision about the permitted break-glass scope; do not implement an allow-list or broaden powers without that decision.
- **A3-F03 — Low — configured MFA requirement is bypassed for decider operations. CONFIRMED by source trace; runtime test not run.** `principal.requireAdmin()` checks `mfaMissing`, but `principal.requireDecider()` does not. Admins without an enrolled second factor can reach decider-only paths such as dispute resolution (which can post a reversal) and challenge decisions. The intended policy must clarify whether an administrator using reviewer/decider powers must satisfy the administrator MFA requirement; the historical A3 audit treats this as a confirmed conformance weakness.

### Owner decision required: A3-F02 / SEC-013 / DEC-014

The repository's SEC-013 addendum explicitly leaves open whether a break-glass administrator can use all administrator powers. The audit must not silently choose.

The owner must decide:
1. **Strict temporary access:** break-glass grants read/export access only, with no balance changes, adjustment approvals, credit-limit changes, credential resets, trade reversals, membership decisions, or network policy/state changes; or
2. **Explicitly bounded emergency powers:** name each permitted state-changing action, state its approval/notice requirements, explain how persistent effects are controlled after the 4-hour grant expires, and update the normative specification and tests.

Separately, SEC-013 still records owner/process gaps for a second approver, notice to affected administrators within 24 hours, and a published operator access policy. These are governance/verification gaps, not independently demonstrated code exploits in this pass.

### Areas not fully verified

- No current-head tests were run. Historical A3 PoC evidence is not a substitute for reproducing findings on the PR #35 commit.
- No exhaustive audit of every service method or object identifier was completed. No additional IDOR bypass was confirmed in the sampled paths.
- Token/session revocation races, API-token survival after password change, MFA enrollment/recovery flows and every route-specific role gate need deeper testing.
- Break-glass owner decision remains unresolved; do not mark A3-F02 fixed.

## Remaining audit areas

1. Priority 3 — cryptography and key management.
2. Priority 4 — SQLite migrations, backup/restore, persistence and recovery.
3. Priority 5 — privacy and data protection.
4. Priority 6 — HTTP/API/browser security.
5. Priority 7 — deployment, infrastructure and secrets.
6. Priority 8 — CI/CD and software supply chain.
7. Verify SEC-033, PR #38 requirements, PR #40 QA-042 implementation, historical branches and current CI.
8. Recheck A1-08 to A1-10 and all lower-severity ledger findings.
9. Reproduce A1-04 through A1-07, A3-F02 and A3-F03 on a runtime-capable environment and add regression tests before accepting remediation.

## Exact next task

Continue with Priority 3. Trace key generation, storage, rotation, encryption/decryption, password hashing, token generation, signature verification, nonce/replay handling and cryptographic error paths. Inspect implementations and vectors; do not replace protocols or change production behavior during the audit. Update this handoff after completing that area.
