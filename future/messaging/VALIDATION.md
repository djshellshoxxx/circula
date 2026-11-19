# Standalone messaging validation — 2026-11-16

**HOLD. Initial implementation evidence below is retained; independent audit and fixes are recorded in AUDIT.md. Integration/release qualification remains pending.**

Toolchain actually used: Go 1.26.11 linux/amd64. Independent module pins `modernc.org/sqlite v1.60.1` and checksummed runtime dependencies. Root Circula was not built, changed or tested by this feature work.

## Observed commands

```text
$ GOGC=10 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover ./...
ok github.com/djshellshoxxx/circula/future/messaging 2.418s coverage: 72.6% of statements
ok github.com/djshellshoxxx/circula/future/messaging/cmd/circula-messages (cached) coverage: 73.2% of statements

$ GOMAXPROCS=1 go vet ./...
PASS (exit 0, no diagnostics)

$ GOMAXPROCS=1 go test -run=^$ -fuzz=FuzzPrepare -fuzztime=3s -parallel=1 .
fuzz: elapsed: 3s, execs: 8685, new interesting: 27 (total: 32)
PASS
ok github.com/djshellshoxxx/circula/future/messaging 3.022s
```

An initial simultaneous heavy SQLite race build was killed by resource pressure. Final run serialized compilation with `/tmp/circula-future-build.lock` and passed. No behavior was disabled to obtain the pass. Coverage measures statements, not assurance or completion percentage.

## Acceptance matrix

| Requirement | Executed evidence | Result / boundary |
|---|---|---|
| Participant/network authorization | `TestDurableOrderingIdempotencyAndFreshAuthorization`, `TestHTTPAPIAuthCSRFBoundariesAndPrivateErrors` | Wrong network, missing actor, nonparticipant and revoked target/actor fail. Fixture authorizer reads fresh SQL membership; real host adapter remains pending. |
| Concurrent monotonic sends | `TestDurableOrderingIdempotencyAndFreshAuthorization`, `TestMultipleConnectionsSerialize` | 40 goroutine sends and 20 sends through two independent SQLite pools preserve count/sequence under race detector. |
| Idempotency/restart/rollback | `TestDurableOrderingIdempotencyAndFreshAuthorization`, `TestRollbackQuotaAndValidation`, `TestReportPaginationAndHiddenRetry` | Matching retry returns same ID after reopen; conflict fails; injected SQLite trigger abort rolls back message/sequence/idempotency; hidden old body stays hidden on replay. |
| Read positions/private hiding | `TestDurableOrderingIdempotencyAndFreshAuthorization`, `TestOwnViewHideAndActivityDeadline`, `TestMigrationBackupRestoreAndEmptyHide` | Read markers never retreat/exceed sequence; hiding excludes old history/inbox without altering counterpart copy; empty thread hiding works. |
| Blocks/report/moderation/hold | `TestBlocksReportsHoldsAndRetention`, `TestReportPaginationAndHiddenRetry` | Both block directions survive restart; duplicate report keeps original; unknown/hidden/foreign targets fail; unauthorized moderator denied; report cursor works; legal hold restricts expired evidence. |
| Retention/purge/export | `TestBlocksReportsHoldsAndRetention`, `TestOwnViewHideAndActivityDeadline`, `TestNoJSMessagingFormsAndEscaping` | Exact 30-day boundary denies ordinary access/export; send extends inactivity; held content excluded; release/purge removes payload/report/idempotency rows; caller hiding applies to exports. |
| Quotas/input | `TestRollbackQuotaAndValidation`, `TestHTTPAPIAuthCSRFBoundariesAndPrivateErrors`, `FuzzPrepare` | Minute quota, oversized/blank/invalid UTF-8/NUL, invalid retention/reference, canceled contexts, unknown JSON fields, body limit, missing CSRF and bad cursor fail. Day/thread/report caps implemented transactionally; dedicated exhaustion tests for every cap remain useful audit coverage. |
| Literal HTML/no-JS forms | `TestNoJSMessagingFormsAndEscaping` | Script/image payload escaped; HTTP form send/read/hide works without scripts; forms have CSRF/labels/privacy text. Handler-level acceptance, not real-browser/WCAG certification. |
| API privacy/errors | `TestHTTPAPIAuthCSRFBoundariesAndPrivateErrors`, `TestHTTPFactoriesFailClosed` | Private success/error headers no-store; failed authorization/CSRF does not echo text/SQL; missing callbacks fail construction. |
| CLI confidentiality | CLI `TestCLIRequestAndNoCredentialRedirect`, `TestCLIValidation` | Trusted local TLS fixture receives expected token/network/key; body from stdin; redirect rejected; HTTP/userinfo/invalid URL/missing credential rejected. |
| Explicit migration/restore | `TestMigrationBackupRestoreAndEmptyHide` | Migration reruns; consistent SQLite VACUUM snapshot restores messages/hiding. Backup permissions explicitly tightened before restore. No destructive down migration or automatic installer. |
| Ledger independence | Source inspection/module-only diff | No ledger/proposal/acceptance APIs, root imports or cross-feature dependencies. Root regression remains later integration gate. |

## Outstanding qualification

- Fresh independent audit and fixes are recorded in AUDIT.md.
- Real browser/no-JavaScript, automated WCAG and manual screen-reader testing not executed.
- Audit module-level advisory scan reports no vulnerabilities; initial symbol scan exceeded memory and is not claimed as passed.
- `openapi.json` is an unregistered standalone fragment; root API contracts unchanged.
- Real host session/CSRF/membership/network-retention/moderation callbacks, network opt-in, platform backup retention and full core/ledger regression remain integration gates.
- Legal hold is restricted server-readable storage, not encryption or physical erasure of historical WAL/backups.

No test grants approval to merge/deploy/release or batch-integrate. Feature remains disabled; integration HOLD.

## Independent audit continuation

Review fixes and new acceptance are in AUDIT.md. Expanded suite covers no post-expiry legal-hold resurrection, closed-thread initiation quotas, scoped block quotas/cleanup, daily-send exhaustion, strict duplicate/Unicode JSON, route-specific 405, complete-or-error CLI response bounds, no-JS moderation/retention, shortening existing deadlines and operator CLI parity. Final race suite reports library 74.1%, CLI 77.7%; `go vet -p 1 ./...` and `go mod verify` pass. Strict JSON fuzz passes 5,591 executions. Module advisory scan passes; the earlier symbol scan was killed by environment memory (exit137). Manual/browser/core integration gates remain unchanged.

Final counterparty-policy regression and hostile-path action checks pass. Every create/send, including an idempotent retry, now invokes the host's explicit `conversation` authorization action in the same SQLite transaction. The final race run reports library 74.1% and CLI 77.7%; final vet passes.

## Recovery continuation and private search — 2026-11-16

The newer local audited implementation `3b4fbe4497827072df15f75e8e518f89925d6435` was selected after discovering it in the same repository object database. The old outage implementation was preserved at `1b5fe42883c29a2dcfd9e924f7d28facac9b541a`; it was not overwritten or represented as newer work. Current continuation branch: `work/requalify-messaging-20261116`.

Required private-message search was absent. Added bounded, literal, case-sensitive Unicode search with sequence paging through the transactional service, POST JSON API, escaped no-JavaScript forms and CLI stdin. Query text is not placed in URLs or argv. Tests cover tenant/nonparticipant/revoked-member denial, own-view hiding, held-content expiry, persisted sequence paging after reopen, SQL wildcard/quote literals, input bounds, CSRF/method policy, escaped forms, second result page and unchanged unread markers.

Observed on the recovered Go 1.26.11 executable `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`:

```text
GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...
ok github.com/djshellshoxxx/circula/future/messaging 6.123s coverage: 74.4% of statements
ok github.com/djshellshoxxx/circula/future/messaging/cmd/circula-messages 1.337s coverage: 78.4% of statements
```

The same default modernc driver is used for all SQL tests; no cgo test substitution is needed. Original baseline results were separately reproduced before changes (74.1%/77.7%). Targeted search race tests passed before the complete suite.

The coordinator reports browser creation blocked by this runtime; this team does not claim a new browser execution. Search delta still needs a fresh independent review. No production/core tests, external messages, merge, deployment or remote publication occurred in this continuation.

Additional current-source checks passed: `go vet -p 1 ./...` (no diagnostics), `go mod verify` (all modules verified), and `go test -p 1 -run='^$' -fuzz=FuzzStrictJSONDecode -fuzztime=3s -parallel=1 -timeout 90s .` (3,685 executions, 34 baseline cases, 15 new interesting inputs). `git diff --check` and the standalone OpenAPI JSON parse passed.

## Independent search audit and recovered tests — 2026-11-16

Runtime implementation remains `0cc6585f2e6c998f5208c6fe569ea4afbf9cc392`. Audited test checkpoint is `da972087fbfa28e6bf71b3310038f70f105d0055`; it preserves the previously dirty cancellation test, benchmark and `testing.TB` fixture. The benchmark now verifies exact hit/miss counts and a consistent 10,000-message sequence. An independent regression verifies that search immediately before the retention deadline does not prolong visibility. SECURITY-AUDIT.md records the fresh reviewer scope and limits.

Commands below actually ran with the recovered Go 1.26.11 executable, `GOGC=20 GOMAXPROCS=1`, and the default Go build cache. Compilation was serialized within this worker. No browser tests or new vulnerability scan ran.

```text
go test -p 1 -race -count=1 -run TestSearch ./...
ok github.com/djshellshoxxx/circula/future/messaging 1.712s
ok github.com/djshellshoxxx/circula/future/messaging/cmd/circula-messages 1.013s [no tests to run]

go test -p 1 -race -cover -count=1 -timeout 90s ./...
ok github.com/djshellshoxxx/circula/future/messaging 4.499s coverage: 74.4% of statements
ok github.com/djshellshoxxx/circula/future/messaging/cmd/circula-messages 1.218s coverage: 78.4% of statements

go vet -p 1 ./...
PASS (exit 0, no diagnostics)

go mod verify
all modules verified

go test -p 1 -run '^$' -bench BenchmarkSearchTenThousandMessages -benchmem -benchtime=3x -count=1 -timeout 90s .
BenchmarkSearchTenThousandMessages/private  3    633055 ns/op  448101 B/op  953 allocs/op
BenchmarkSearchTenThousandMessages/no-match 3  87842174 ns/op    6813 B/op  150 allocs/op
PASS
ok github.com/djshellshoxxx/circula/future/messaging 0.856s
```

The targeted run preceded the added deadline regression; the complete uncached race run includes that regression and all recovered test source. Broad scalability, real browser/no-JS/WCAG and manual screen-reader evidence remain open. Completion is 90% weighted acceptance evidence solely because the independent search-review criterion is now satisfied, not because benchmarks or Go coverage establish overall completion.

Benchmark environment: linux/amd64, AMD EPYC 9V74, one Go scheduler thread; synthetic private SQLite fixture containing 10,000 messages of 3,900 UTF-8 bytes each, with setup excluded from each subbenchmark timer. Early-hit query returns 100 rows; no-match query returns zero. Three iterations are diagnostic observations, not a stable latency distribution or production service objective. Missing substrings scan the entire remaining conversation under the existing writer reservation. No heavy-client contention, larger retained-history workload or cancellation during an active scan was measured.

## Callback/form/query audit — milestone F, 2026-11-16

Exact tested source `0b1861f50ef1fcecb65eaf3e4227a647ce2dcd58`. Toolchain `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go` (Go 1.26.11), `GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache`; compilers serialized.

- Before fixes, `go test -p 1 -count=1 -run '^TestCallbackAudit' -timeout 90s .` failed: malformed form accepted HTTP 303 / one persisted message, stale token pages HTTP 200 and revoked/replaced CSRF sends HTTP 303 / one message.
- Query-only regression on the previous source failed: oversized query reached Resolve, HTTP 403 / one host invocation.
- Final focused command above: PASS, 0.263 s.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS, library 5.259 s / 76.3%, CLI 1.280 s / 78.4%.
- `go vet -p 1 ./...`: PASS, no diagnostics.
- `go mod verify`: all modules verified.

Raw first red/green, intermediate race/vet and final query red/focused/race/vet/module logs are in `callback-audit-evidence/`, with SHA256SUMS. Intermediate passed source `b81473dd929b541a6d937cdf71c3a157fa32227f` is superseded for final qualification by the source above. No browser, benchmark rerun or new advisory scan is claimed. Existing benchmark remains diagnostic evidence for the unchanged search implementation, not broad scalability qualification.
