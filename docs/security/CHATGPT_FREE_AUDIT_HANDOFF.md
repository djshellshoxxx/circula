# Circula security audit handoff

Date: 2026-11-15
Audit branch: `chatgpt/circula-security-audit-20261115`
Authoritative application baseline: PR #35 branch `claude/cool-ramanujan-n2peag`
Repository default branch: `main`
Baseline commit SHA: not returned by the available GitHub connector; do not guess. The comparison against main reports the PR #35 branch is 345 commits ahead and 0 behind, with merge base `4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7`.
Audit scope: security review only. No application code, financial rules, migrations, branches, or production data were changed.

## Repository state

- Repository: `djshellshoxxx/circula` (Go, SQLite).
- PR #35 is the documented integration baseline. Its handoff identifies it as a draft PR into `main`; the PR description claims a 345-commit integration. Treat its reported test results as claims until current-head CI is independently checked.
- PR #35's changed-file list contains the full application, tests, migrations, security reports and specifications.
- PR #38: foundational specs. PR #40: QA-042 requirement-ID drift auditor. PR #41: future-only receipt/certificate specs; it explicitly says no current feature code is introduced.
- PR #35 handoff says QA-042 full traceability regeneration remains incomplete, alongside owner gates and SEC-033.
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
- A1-02 / A1-11 — scheduled decimal-scale changes reinterpret stored amounts: **fix present in source per remediation record**; inspect `UpdatePolicy` and regression tests again during the next runtime-capable pass.
- A1-03 — re-review reverses a trade while another live dispute is open: **fix present in source**. `decideReReview` refuses reversal when `t.Status == disputed`.
- A1-04 — reversal can post while network is administrator-required: **CONFIRMED by source trace; runtime PoC not run**. `networkOpenErr` / `requireNetworkActive` rejects `AdminRequired`, but `internal/service/reversal.go:reverse` checks only `net.Suspended()`. Both `ResolveDispute` and `RetryDisputePosting` call `reverse` without a network-active gate. An administrator/reviewer can therefore resolve or retry a dispute and post a reversal while the network is read-only because it has no active administrator. The reverse path needs the same complete posting gate as other value-moving operations. Severity: Low under the existing A1 classification; raise only if the wider threat model supports greater impact.
- A1-05 — expired-proposal sweep starvation: **CONFIRMED by source trace; runtime PoC not run**. `internal/store/sqlite_tx.go:ListExpiredProposals` selects the oldest expired proposals globally with a limit and no network filter. `internal/service/transactions.go:ExpireProposals` then skips frozen networks. If the first limited batch is continually occupied by frozen-network proposals, active-network proposals behind it can remain proposed indefinitely. Severity: Low (liveness/financial fairness).
- A1-06 — incomplete proposal cancellation on member departure: **CONFIRMED by source trace; runtime PoC not run**. `internal/service/departure.go:cancelOpenProposals` fetches only 500 proposed transactions and does not paginate. More than 500 open proposals involving a departing member leave excess proposals active, retaining reservations until expiry and referencing a departed party. Severity: Low.
- A1-07 — final closure export is not retried after a one-time export failure: **CONFIRMED by source trace; runtime PoC not run**. `FinaliseClosures` first calls `finaliseOne`, which sets `closed_at`; it then calls `recordFinalExport` in a separate unit. `ListClosures(true)` only returns rows with `closed_at = 0`. If export/recording fails after closure commits, the next sweep no longer selects that closure, so its final checksum/reconciliation may never be recorded. Severity: Low in the historical A1 report; impact includes loss of the closure integrity receipt.
- A1-08 to A1-10 and remaining informational findings: not fully rechecked in this pass. Do not mark resolved.

### Tests and limitations

- Read PR #35 source patches for the posting path, SQLite transaction wrapper, transaction store, migrations, reversal/dispute/departure/closure paths, and the historical A1 report.
- Reviewed existing test source for atomicity, concurrent acceptance, departure settlement and reversal behavior.
- No shell/Go runtime was available through the connected GitHub interface. No `go test`, `go test -race`, fuzz, mutation, or PoC execution is claimed.
- Findings A1-04 through A1-07 are confirmed by code-path inspection but still need focused reproductions and regression tests before remediation is accepted.
- No destructive or production tests were run.

## Remaining audit areas

1. Priority 2 — authentication, authorization, IDOR.
2. Priority 3 — cryptography and key management.
3. Priority 4 — SQLite migrations, backup/restore, persistence and recovery.
4. Priority 5 — privacy and data protection.
5. Priority 6 — HTTP/API/browser security.
6. Priority 7 — deployment, infrastructure and secrets.
7. Priority 8 — CI/CD and software supply chain.
8. Verify SEC-033, A3-F02 break-glass scope, PR #38 requirements, PR #40 QA-042 implementation, historical branches and current CI.
9. Recheck A1-08 to A1-10 and all lower-severity ledger findings.

## Exact next task

Continue with Priority 2. Trace authentication and session lifecycle from HTTP/API entry points through service authorization and persistence. Test every sensitive object operation for server-side ownership and role checks. Do not change production behavior during the audit. Update this handoff after completing that area.
