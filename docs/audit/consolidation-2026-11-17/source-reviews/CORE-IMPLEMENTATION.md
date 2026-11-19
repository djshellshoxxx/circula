# Workstream C: independent core implementation recovery review

Review baseline: `163fe1596bd7cfb8c2a22891629fc4459e6b639c` (merged through PR #69 as `ef54238ee9ebc2184919b6a8ec30c3398df55ed5`). Historical comparison sources include `claude/cool-ramanujan-n2peag@c033cfa448fb89d1a5cc0ca87458b40d2709fa2e` and `haiku/circula-consolidation@6fbfa13fb696d38e8e120377c0eee09af1ac44f0`, plus the exact security source heads recorded by workstream A.

## Consolidation verdict

No additional missing historical core implementation was identified beyond the startup migration gate and regression recovery already merged in PR #69. The reviewed current implementations retain equivalent historical behavior and add relevant newer guards. This is an independent behavioral sample of the branch inventory, not a claim that every possible defect in core has been excluded.

| Area | Source inspection and current evidence |
|---|---|
| Atomic ledger posting and credit limits | `internal/service/posting.go` reads both accounts inside the serialized transaction, checks bounded balances and debit capacity, then writes both entries, balances, transaction and status event. `internal/store/sqlite.go` retains the write gate and `BEGIN IMMEDIATE`. Historical posting differences are the repository abstraction, fixed-at-completion dispute deadline and related adjustment metadata; they do not remove older atomicity guards. Existing `TestParallelAcceptsNeverBreachCreditLimit`, `TestParallelDoubleAcceptSingleSuccess`, `TestQA007OversubscriptionInterleavings` and fault-injection coverage remain present. |
| Idempotency | `internal/service/idempotency.go` is behaviorally identical to the two reviewed historical lines; only its repository import changes. Network/actor/operation/key scope, request fingerprint and result storage remain in the caller's write transaction. `internal/api/replay_auth.go` explicitly maps cached routes to current permission checks and fails closed for unrecognized routes. The replay authority, expiry and stale-finish regression suites are present. |
| History | `internal/service/reversal_links.go` retains keyset pagination across every 500-row page and stable ledger-sequence ordering. Current code adds record-ID validation. `TestReversalsOfResultLimit` checks the large result set; insertion-order and historical balance suites remain present. |
| Authorization, MFA and tenant isolation | `principal.openAny` resolves the member server-side, enforces token network scope before exposing a network and computes mandatory MFA status. `requireDecider` adds the newer MFA guard absent in the historical #35 source. Store account/member getters include `network_id`; account mutations remain tenant-scoped. Current service and API authorization matrices, cross-network ID tests and the PR #69 entry-point regressions remain present. |
| Governance and disputes | Current credit-limit expected revisions are checked within the same transaction as the update. Reversal authorization uses the active-network/admin-required check. Dispute re-review guards prevent reversing under another live dispute. Existing scheduled-policy-preserving operator regressions and dual-control independence tests remain present. No proposal to change break-glass scope is made. |
| Persistence and migrations | Current source retains the recovered H05 suite and stronger migration, disk-full, WAL write-failure, checkpoint recovery and crash tests. These remain bounded qualifications; they do not establish coverage of every fsync/read/uncertain-commit failure. E-02 was independently reproduced and fixed separately below. |
| Backup, restore and exports | Current backup verification preserves signed restore refusal, copy rehash and credential invalidation. Historical memory-fallback exports are superseded by mandatory disk spooling, running reservations and bounded concurrent bytes. `jobs.dropErased` revokes affected running/completed exports and their readers while preserving unrelated networks. NEW-01 backup temporary-file work belongs to workstream D. |
| Privacy and abuse prevention | Current notification erasure uses resource relations, recovery throttle families are removed, frozen-network expiry pages do not starve live work, and forced password-change/current credential checks remain present. Retention choices, WAL sanitization design and future SEC-026 functionality are not silently treated as completed by consolidation. |

## E-02: reproducible missing SQLite hardening

Original `store.Open` and `store.OpenRaw` initialize every connection with `trusted_schema=1`. A stored view can invoke the application-defined `circula_fold` function. Regression before the fix:

- Eight normal connections and two raw connections all return `1`, expected `0`.
- Both open paths evaluate a view calling `circula_fold('CAFÉ')` and return `cafe`, expected an unsafe-use refusal.

This demonstrates the missing schema trust boundary. It does not establish arbitrary code execution or data exfiltration through the current text-folding function.

Local fix commit: `66a0d4a94e3b2650a444037c7e6874d172d7a11d`, branch `integration/store-untrusted-schema-20261117`.

Changed files: `internal/store/sqlite.go`, `internal/store/inspect.go`, `internal/store/trusted_schema_test.go`, `docs/audit/E02-TRUSTED-SCHEMA-2026-11-18.md`.

The production change adds `trusted_schema(0)` to both connection strings, before other pragmas. No schema migration, global test function, financial semantics or policy change. Regression tests prove every connection is covered and direct application calls remain usable.

Qualification: both regression tests pass twice, including twice under the race detector. `go vet ./...` and the complete `go test -count=1 -timeout=25m ./...` pass in a standalone clone, including all 14 packages with tests. Two helper packages have no tests. Initial linked-worktree execution hit a toolchain VCS discovery error in the restore-drill test's nested build; the standalone clone resolves this without changing or disabling the test.

Evidence: `e02-regression-before.log`, `e02-regression-after.log`, `e02-related.log`, `e02-aggregate.log`, and the separate focused race result in `e02-race.log`.

## Additional actionable finding: administrator grant notice truncation

`internal/service/membership.go:noticeAdminGranted` fetches only the first 500 active members and then filters for administrators. An administrator outside that page receives no `admin.granted` notice. This violates the existing function contract and `TestA3_01_AdminGrantNotifiesOtherAdmins` expectation that every other active administrator is notified.

Reproduction on the unmodified PR #69 tree: seed 501 ordinary members with lower-sorted IDs, grant the administrator role to another real member, and inspect the founder's security notices. Expected one new notice; actual zero. The probe fails deterministically in approximately 0.03 seconds.

Evidence: `admin-notice-before.log` (initial probe) and `admin-notice-regression-before.log` (final regression). The retained regression is `internal/service/admin_notice_pagination_test.go` on the separate fix branch.

Implemented with root authorization in local commit `4c26a7ed443d47e339d3bdac7b8f17d3b4e6fe58`, branch `integration/admin-grant-notice-pagination-20261117`: page `ListMembers` using `MemberFilter.AfterID` until exhausted, while retaining the existing active-member, administrator-role, grantee exclusion and network checks. Each read remains capped at 500 rows and every page uses the original write transaction. The focused test verifies early/late administrators each receive exactly one notice, and ordinary, grantee, inactive and foreign-network actors each receive zero. The new regression and all existing A3-01 tests pass twice; the new regression and existing administrator-notice test pass twice under the race detector. `go vet ./...` and the complete `go test -count=1 -timeout=25m ./...` also pass at the exact commit in a standalone clone (all 14 tested packages). Evidence: `admin-notice-regression-after.log`, `admin-notice-race.log`, `admin-notice-aggregate.log`. Independent review and remote integration remain with the root agent.

## Deliberately unresolved

A3-F02 / DEC-014 remains an owner policy question. No authorization change is proposed. All 15 future modules and their associated source branches remain isolated. SEC-026, GOV-021 and incomplete UX-047 catalogue work are product/feature work, not missing historical code recovered by this pass.
