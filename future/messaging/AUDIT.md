# Independent messaging audit — 2026-11-16

**Integration HOLD. Fresh audit of implementation baseline `b698fbc`.** This reviewer did not implement the baseline. Scope is the isolated `future/messaging` module, its SQL/service/auth ports, HTTP/templates/CLI, tests, contracts and dependency inventory. No root application change, merge, deployment or publication is performed.

## Findings and fixes

| Finding | Effect before review | Fix and regression evidence |
|---|---|---|
| MSG-AUD-01 — expired evidence resurrection | A moderator could place a new hold after an unheld thread expired, then read content that ordinary retention had made unavailable. A released expired hold could also be reopened. | Deny a new hold on expired unheld content. Existing pre-expiry holds remain usable; release cannot resurrect expired content. `TestAuditExpiredContentCannotAcquireNewHold` verifies both cases and later purge. |
| MSG-AUD-02 — admission quota bypass | Repeated create/close operations bypassed the live-thread cap; directed blocks allowed unlimited nonexistent targets. | Transactional initiation budgets (60/minute, 1000/day) include closed/purged tombstones; idempotent retries do not consume them again. Blocks require a fresh active target and cap 1000 rows/member; unblocking departed targets remains possible. `TestAuditCreateQuotaSurvivesClosedChurn`, `TestAuditBlockScopeAndQuota`, `TestAuditDailySendQuota` cover churn, exact expiry and daily sends. |
| MSG-AUD-03 — ambiguous Unicode/JSON | `encoding/json` silently replaced malformed UTF-8/unpaired surrogate escapes, accepted duplicate/case-aliased names and accepted null as a zero struct. | Bound before host authentication callbacks and before parsing; validate UTF-8 and paired escapes; require object roots; reject duplicate/case aliases with a depth-16 bound, unknown fields and trailing documents. `TestAuditStrictJSONRejectsAmbiguity` and `FuzzStrictJSONDecode`. Literal U+FFFD and escaped backslashes remain valid text. |
| MSG-AUD-04 — silent CLI truncation | A legitimate page containing many HTML-sensitive characters could expand beyond the CLI's 1 MiB cap, emit broken JSON and still report success. | Read at most 4 MiB plus one byte; copy complete accepted pages, reject excess without emitting a partial page. `TestCLILargePageCompleteOrError` covers a 1.5 MiB accepted page and an oversized response. |
| MSG-AUD-05 — missing operator paths | Moderator and retention operations existed only in JSON APIs; CLI lacked these and conversation-list continuation. | Escaped, CSRF-protected no-JS moderation/report/hold and maintenance screens; shared-authority CLI `reports`, `held`, `moderate`, `purge`, `retention`, `-cursor`. `TestAuditNoJSModerationAndRetention`, `TestCLIAdminCommandsAndConversationCursor`. Ordinary members cannot read review evidence. |
| MSG-AUD-06 — retention policy could only affect new threads | A newly shortened network maximum had no module operation to shorten existing deadlines. | Explicit separately authorized `ShortenRetention`, API/form/CLI, checks current network policy and uses `min(existing, maximum)`. It does not change activity, extend a deadline or release a hold. `TestAuditNoJSModerationAndRetention` verifies immediate exact-boundary expiry and no re-extension. Host policy synchronization remains an actual integration requirement. |
| MSG-AUD-08 — missing pair-policy authorization seam | Active-member checks exposed no explicit actor/counterparty callback for a network relationship policy. | Add a mandatory transactional `conversation` action for create/send and their retries. `TestAuditCurrentPairPolicyOnMutationAndReplay` verifies current relationship denial cannot be bypassed by retry while historical participant reads remain available. |
| MSG-AUD-07 — method contract/body bounds | Known route method mismatches returned generic 400 and authentication callbacks ran before the module body cap. | Apply the 64 KiB cap before authentication and return RFC 9457 405 with route-specific Allow. `TestAuditMethodContracts` and the existing body/CSRF/private-error tests. |

## Requirement review

| Requirement | Reproduced evidence / disposition |
|---|---|
| Current tenant/member/participant isolation | Original fresh SQL membership, foreign actor/network, nonparticipant and revoked-target regressions pass. Every unit reserves SQLite's writer then invokes the trusted authorizer in that transaction. Actual host callbacks remain a later integration gate, not client-provided booleans. |
| Ordered durable writes and idempotency | Original 40-send goroutine and two-pool concurrency tests, conflict/reopen tests and trigger-failure rollback pass. Send/sequence/activity/idempotency commit together. Closed-thread retries may return committed results; no new send occurs. |
| Blocks, reports and quotas | Both directions, unblock, hidden/foreign/missing report targets and first-report dedupe pass; fresh block-target/quota and create-churn regressions added. Existing minute-send, report-pagination and SQL quota logic reviewed; daily-send exhaustion tested independently. |
| Own-view deletion, read/unread | Monotonic markers, hidden-through sequence, hidden empty threads, hidden replay/export and fresh new-message visibility tests pass. Hiding does not delete another participant's copy. |
| Retention, holds, purge and export | Exact inactivity boundary, extension only on new committed send, held evidence restrictions, release, purge/restart and closed SQLite backup/restore pass. Late hold resurrection fixed. Purge removes body/report/idempotency data; physical WAL/backup erasure is not claimed. |
| Reference policy, no trade authority | Reference authorization runs under the same transaction on creation. No ledger/proposal/acceptance interface or cross-feature import exists. Whole-core ledger regression is pending integration. |
| Rendering and private transport | Handler tests reproduce literal script/image escaping, no-store/error headers, current host auth/CSRF and strict JSON/body constraints. Added operator forms remain script-free with labels and escaped evidence. No E2EE or remote-image/preview claim. TLS-only host mounting/cookies/origin policy remains host qualification. |
| Local action paths | Fixed `/conversations/{validated-id}` form actions and query-prefixed encoded pagination never use request URL.Path as an action. `TestAuditActionsRemainLocalForHostilePaths` verifies protocol-relative hostile paths cannot create an external POST action. |
| HTTP/CLI parity | Member and separately privileged paths are present; CLI stdin avoids text in argv, credential redirects are refused, response completion enforced, paging/operator commands tested. Local fragment/help/contract updated; no root API advertised or mounted. |
| SQL migration and recovery | Explicit idempotent migration and private regular-file constructor; FK and secure-delete requirements reviewed; existing migration/backup/restore tests pass. Added admission index is additive and installed by explicit rerun. |
| Supply chain | Pinned SQLite dependency/runtime notices reviewed; `go mod verify` passes. `govulncheck -scan=module` reports no known vulnerabilities. Initial symbol analysis was killed by environment memory; no successful reachability scan is claimed. |

## Executed validation

Go 1.26.11 linux/amd64, synthetic principals and temporary SQLite/TLS fixtures only.

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...`: passed; final exact results in VALIDATION.md.
- `GOMAXPROCS=1 go vet -p 1 ./...`: passed.
- `go mod verify`: all modules verified.
- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -run='^$' -fuzz=FuzzStrictJSONDecode -fuzztime=3s -parallel=1 -timeout 90s .`: passed 5,591 executions, seven baseline seeds, 27 new interesting cases.
- `GOFLAGS=-buildvcs=false ... govulncheck -scan=module`: no vulnerabilities found. Default symbol scan was terminated by resource limits (exit 137); that attempt is not a pass.

## Remaining qualification

No unfixed implementation finding from this review is being concealed as a completion gate. Real-browser/no-JS behavior, automated WCAG and manual screen-reader acceptance were not executed. Actual host network opt-in, TLS/session/origin/CSRF and fresh role/reference/retention adapters, privileged-action audit integration, schema adoption, operational key/session controls, backups/legal-hold retention and full core/ledger regression remain separate gates. No production keys, listeners, workers or feature registration are added. This review cannot establish that all possible defects are absent, and grants no integration/release approval.
