# Circula code audit

Scope: whole tree at `claude/circula-build`, HEAD `94c77c9` (audit began at `920f9dc`; `94c77c9` fixed the lint findings listed in section 1).
Compared against `docs/specs/*` and `docs/dev/SERVICE.md`. Code was read, not modified. Probes were run in a scratch copy outside the repo.
Every finding below was verified in code or by a probe test. Nothing speculative.

## 1. Tool results

| Check | Result |
|---|---|
| `go vet ./...` | clean |
| `gofmt -l .` | clean |
| `go test -race -count=1 ./...` | all 6 packages pass, no data race reported (service ~50 s under race) |
| `golangci-lint` v2.5.0 (CI-pinned) | at `920f9dc`: 8 issues (errcheck x2, gosec x5, unconvert x1). At `94c77c9`: 0 issues, exit 0 |
| `go test -cover` (total, with `-coverpkg`) | 78.4 %. cmd 56 %, api 83 %, domain 80 %, service 83 %, web 76 %. The store package alone reports 19.7 %, but it is exercised through service tests |
| govulncheck | not run (needs network via `go run`; CI runs it) |

Probes (scratch copy, `internal/service/probe_test.go`, not committed):
- P1: an `INSERT` made inside `Store.View` succeeded and persisted.
- P2: 8 parallel failed `Login` calls for one login left `login_throttle.failures = 1`.
- P3: 3 `VerifyLedger` calls added 3 audit events to the network chain.

## 2. Ledger invariants (LED-1..7), state machines

| Invariant | Verdict | Evidence / caveat |
|---|---|---|
| LED-1 entries sum 0, balances sum 0 | holds | Two-entry post in `service/posting.go` `postTx`. DB trigger `tx_complete_needs_entries` and `entries_only_for_open_tx` (`store/migrations/0001_init.sql`). Network-total zero is NOT a DB constraint, only checked by `VerifyLedger`. |
| LED-2 append-only | holds with one gap | Triggers cover entries, audit, audit_subjects, status events, dispute events, policies, consistency runs. `accounts.balance` (the cache) has no guard trigger, so raw SQL can corrupt it. Only `VerifyLedger` would catch that (LED-3 detection, as designed). |
| LED-3 cache == sum | holds | `AddBalance` is called only from `postTx`. Check in `store/sqlite_verify.go` `BalanceMismatches`. |
| LED-4 atomic + idempotent | holds | One `store.Update` per command, `idempotent()` writes the key in the same unit. Failed commands roll back, so keys stay reusable. See API-07 and AUTH-01 for edge cases outside the ledger path. |
| LED-5 limit at post time, serialised | holds, service-level only | `checkCapacity` runs in `postTx` under write gate + `BEGIN IMMEDIATE`. No DB-level check. Single process enforced only by lock file in `serve` (see OPS-02). |
| LED-6 audit same unit | holds for ledger/membership/dispute/policy changes | Every command calls `Service.audit`/`emit` inside the unit. Login/logout/session creation are not audited (not ledger state). |
| LED-7 integers only | holds | `int64`, `AddChecked` overflow guard, `MaxPolicyAmount = 1<<50`, DB `CHECK(amount > 0)`. |

State machines: `domain/statemachine.go` transaction transitions match the DB trigger `tx_status_flow` exactly (proposed -> completed/declined/cancelled/expired; completed -> disputed/reversed; disputed -> completed/reversed). Dispute transitions are service-level only (no DB trigger). Member and network status transitions are service-level only (DB has only value CHECKs). `accepted` is never persisted, as documented.

Documented deviation confirmed: DAT-012 (one reversal per original) is not enforced; cumulative partial reversals are capped by `CHECK (reversed_total <= amount)`. API-056 text "second reversal returns 409" therefore does not hold for partial reversals.

## 3. Findings

Severity: critical / high / med / low. No critical finding. Line numbers are at HEAD `94c77c9`.

### 3.1 Service and store

**SVC-01 high. `VerifyLedger` modifies the database, blocks all writers, and is invoked synchronously everywhere**
- `internal/service/verify.go:30` runs the check inside `s.update` (write gate + `BEGIN IMMEDIATE`), `:71` inserts a `consistency_runs` row, `:75` appends an audit event (`ActVerify`) that extends the hash chain.
- Spec: OPS-016 ("MUST NOT modify any data", "leaves a checksum of the database unchanged", "runnable while the service is running (read-only, consistent snapshot)"), FR-052 ("runs read-only"), OPS-012 ("long-running work MUST NOT hold the write lock").
- Consequences verified in code: full-table scans (`store/sqlite_verify.go` `BalanceMismatches`, `TxProblems`, `ReversalMismatches`, audit walk) hold the only write slot, so every posting waits behind it. Callers: API `POST /admin/consistency-checks` (`api/h_ops.go`), web `ledgerCheck` (`web/h_admin2.go:216`), startup, hourly job (`cmd/circula/serve.go:279,326`), `migrate`, `restore`, `verify-ledger`. On a large ledger a scan longer than the 10 s busy timeout makes other processes fail (see OPS-01). Probe P3: each run adds an audit event.
- Fix: split into a read-only `VerifyLedger` on `Store.View` (returns report, no write). Record the run and audit event in a separate short write unit only when the caller asks (API/web admin action). Make CLI/job/startup/backup callers use the read-only path. Add the OPS-016 test (database checksum unchanged).

**SVC-02 med. `Store.View` is not read-only**
- `internal/store/sqlite.go:69-72`: comment says "Writes through it fail", but `View` runs `BEGIN` (deferred) then `COMMIT`, so a write upgrades the lock and commits. Probe P1 confirmed an `InsertPerson` through `View` persisted, bypassing the in-process write gate (`gate` is taken only in `Update`).
- No current caller writes inside `View` (checked all `s.view(` sites), so this is latent, but any future mistake silently skips the gate and the serialisation LED-5 depends on.
- Fix: run view connections with `PRAGMA query_only=ON` (set per conn inside `run` for `View`, reset after), or open a second read-only `sql.DB` (`mode=ro`) for views. Add a test that a write in `View` errors.

**AUTH-01 med. Login throttle has a lost-update race, so parallel guesses bypass SEC-006**
- `internal/service/auth.go:99` reads `failures` in a `view`; password verification runs outside any transaction; `:126` writes `failures+1` in a later `update`. Probe P2: 8 parallel failures recorded `failures = 1`. A burst of N guesses is counted as 1, and none sees the lock.
- The only login entry is the web form (`web/h_auth.go` `loginPost`) with a per-IP limit of 10 per 60 s, which bounds practical impact; the service-level guarantee in the SERVICE.md/SEC-006 sense is still broken.
- Also: the `PutThrottle` error is discarded (`:125` `_ = s.update`), and `Login` never calls `validLogin`, so any string (up to the 1 MiB web body) becomes a `login_throttle` primary key (see DATA-01).
- Fix: do the increment in SQL inside one `update` (`INSERT ... ON CONFLICT DO UPDATE SET failures = failures + 1 RETURNING failures`) and check the lock in that same unit before verifying; validate and length-cap the login first.

**AUTH-02 low. Store errors during authentication become "401 unauthenticated"**
- `internal/service/auth.go` `Authenticate` (`:156`) and `internal/service/tokens.go:72` `AuthenticateToken` map every `s.view` error (including SQLITE_BUSY) to `ErrUnauthenticated`. A transient DB failure logs users out (web redirects to `/login`; API clients get 401 and may discard tokens).
- Fix: map only `ErrNotFound` to unauthenticated; return other errors as internal.

**AUTH-03 med (spec gap). `ChangePassword` is unreachable**
- `Service.ChangePassword` (`service/auth.go:182`) has no caller outside tests (grep of `internal`, `cmd`). No web form, no API route, no CLI. SEC-002/SEC-003 acceptance ("after password change, a session from another browser is rejected") cannot be exercised by a user; a member cannot change a password at all.
- Fix: add web form + API route, call `ChangePassword`; or record the P0 deferral.

**SVC-03 low. Accept does not re-check the default flag**
- `ProposeTransaction` enforces `DefaultFlagBlocksPayer` (`service/transactions.go:87`) but `AcceptTransaction` (`:146`) does not. A payer flagged after the proposal still completes the payment. DOM-025 says only that the flag's configured effect is "for example, blocking new payer proposals", so this may be intended; it should be a recorded decision.

**SVC-04 med. Denied or removed members get a success response when "joining" again**
- `service/join.go` `JoinNetwork` (`:189`) and `RedeemInvitation` (`:144`) return the existing membership with `nil` error when one exists, including `departed` (denied applicants are stored as `departed` by `DecideJoinRequest`, so they can never re-apply; UNIQUE `(network_id, person_id)`).
- `web/h_auth.go:222` `afterJoin` then flashes "You have joined X" for any non-pending status and redirects into the network (`netH` only intercepts `pending`).
- Fix: return `invalid-transaction-state`/`forbidden` for `departed` (and `suspended`) from join paths; record whether re-application after denial is wanted (FR-011).

**SVC-05 low. Reads that audit use write transactions; API credit-limit PUT writes three audit rows**
- `ListAudit` (`service/audit_read.go:54`), `AdminGetMemberBalance` (`service/accounts.go:73`), `Export` (`service/export.go:95`) take the write gate. Documented in SERVICE.md.
- `api/h_accounts.go:62,73` calls `AdminGetMemberBalance` before and after `SetCreditLimit`, so one `PUT .../credit-limit` writes `account.viewed` x2 + `limit.changed`. Audit noise and two extra write-lock acquisitions per request.
- Fix: expose a non-audited internal read for ETag computation, or compute the ETag from the account row inside the same service call.

**DATA-01 low. Nothing purges expired rows**
- No job or query deletes expired `idempotency_keys` (only on reuse of the same key, `store/sqlite_audit.go` `DeleteIdem`), expired/revoked `sessions`, or old `login_throttle` rows (grep for `DELETE FROM` finds only those two statements). Tables grow without bound; `login_throttle` accepts attacker-chosen keys (AUTH-01).
- DAT section 9 retention is therefore not implemented. Fix: add a periodic purge to the job runner (`cmd/circula/serve.go`) via a service method.

**SVC-06 low. `View`-based verify of audit chain ignores platform scope**
- `service/verify.go:84` `verifyAuditChain` checks only the network's chain. Platform events (`network_id = ''`: registrations, password changes, token create/revoke) are hash-chained by `emit` but never verified. Add a platform-scope check to `circula verify-ledger`.

### 3.2 JSON API (`internal/api`)

**API-01 high. Cookie names differ between web sign-in and API, so a browser session cannot call the API**
- `internal/web/session.go:18` sets `cm_session`; `internal/api/auth.go:15` reads only `__Host-circula_session` / `circula_session`. `grep` finds no bridge. Tests in `api/authn_test.go` hand-craft `circula_session`, hiding it.
- Spec: API-008 (P0, browser clients authenticate with the cookie from the sign-in flow, `__Host-` prefix). Also CSRF token schemes differ (`api/auth.go:24` HMAC with constant key vs `web/session.go` random-key double submit).
- Fix: one cookie name constant shared by both packages (use `__Host-` where Secure), one CSRF scheme, and an end-to-end test that signs in through the web and calls the API.

**API-02 med. `If-Match` optional on member actions**
- `api/h_admin.go:115`: precondition checked only when the header is present ("optional for these actions"). API-026/027 (P0) require 428 when missing for membership status. Also the check-then-act is two separate store units (ETag read at `:116`, change after), and `casMu` is process-local and used by API handlers only; web edits (`web/h_home.go:321` `listingEdit`, `web/h_admin.go:154` `memberActionPost`) bypass it, so a web edit can slip between the API's ETag check and its write (lost update on listing/policy/limit).
- Fix: make `If-Match` mandatory for approve/suspend/reinstate/remove/roles; carry an expected-version into the service call and compare inside the write unit.

**API-03 med. `GET /accounts/me/entries` is ordered by transaction id, not `ledger_seq`**
- `api/h_core.go:133-135,656` ("Order: transaction id descending"; derived from `ListTransactions`). API-045 (P0) requires `ledger_seq` descending. Proposals complete later than they are created, so IDs and posting order differ; `LedgerSeq` is shown but not the sort key; cursor is a tx id.
- Fix: add `ListEntries(member, afterSeq, limit)` to the service (store already has `entries(account_id, ledger_seq)` index) and page by `ledger_seq`.

**API-04 med. Bounded O(n) scans give wrong answers beyond fixed caps**
- `api/h_core.go:60-75` `findDirectory` (20 x 100 = 2000 members) used by `GET /members/{id}` -> 404 for an existing member past the 2000th; `names` (`:96`, 10 x 100) silently drops `counterparty_display_name` past 1000 members and for non-active counterparties; `api/h_admin.go:86` `adminMember` (2000) is called AFTER the change (`:140`, `:164`) so a role/suspend change succeeds but the client gets 404; `api/h_listings.go:71` `ownListing`; web `findMember` `web/h_admin.go:131` (500, no paging) makes members beyond the 500th unmanageable in the UI (list is oldest-first, so the newest members are unreachable); `web/h_admin.go:31` pending count capped at 500.
- Root cause: the service lacks single-object reads (`GetMember`, `GetListing`, `GetDisputeByTx`) and an entries list, so adapters scan.
- Fix: add those service methods; delete the scan helpers.

**API-05 med. Exports are fully buffered in memory, and the 50-slot store is global**
- `api/h_export.go:63` uses `MemSink`, then builds a zip in a second buffer; `api/jobs.go:28,78` caps total at 50 exports / 64 MiB across ALL users and networks. FR-095 says rows are never held in memory (the service streams; the adapter undoes it). One member can request `mine` exports until the cap is hit, answering everyone else with 503 for 24 h (`jobTTL`). A `full` export of a large network risks OOM.
- Also exports and consistency-check results vanish on restart (`api/jobs.go`), and the check id returned (`api/h_ops.go:104` `uuid.NewString()`) is not the `consistency_runs.id` stored in the DB.
- Fix: per-person quota, stream to a temp file (or response), persist run ids from `VerifyLedger`.

**API-06 low. Idempotency record stuck after a handler panic**
- `api/idem.go:124-170`: `begin` creates a not-done record, `finish` is called only on normal return. A panic in `rt.h` is recovered in `edge` (`api/middleware.go:47`) but the record stays `done=false`; every retry with that key gets 409 `idempotency-request-in-progress` until the 24 h TTL (`api/api.go:74`). Fix: `defer` a failing `finish` in `runIdempotent`.

**API-07 low. Invitation token written to logs and echoed**
- `api/route.go:56` logs `r.URL.Path` on internal errors; for `POST /invitations/{token}/accept` the path contains the token. `api/respond.go` `writeProblem` echoes the path in `instance`. API-012 forbids credentials in logs. (Metrics/access log in `cmd/circula/ops_http.go:88` correctly logs only the route template.) Fix: log the route pattern (`r.Pattern`), mask `{token}`.

**API-08 low. Wrong error code for malformed idempotency key**
- `api/idem.go:133` returns `idempotency-key-required` for an invalid format. Not in the code catalogue; should be `validation-failed`/`invalid-header`.

**API-09 low. Rate-limit and client IP**
- `api/middleware.go:92` and `web/ratelimit.go:50` key on `RemoteAddr`. Behind a reverse proxy (the documented non-loopback deployment, `serve.go:164` warns for it) all users share one IP bucket (API 1200/min, web login 10/min), and `X-Forwarded-For` is never read. There is no trusted-proxy setting. `limiter.allow` (`middleware.go:114-118`) sweeps the whole map on each call once it exceeds 50 000 keys, which is O(n) under the mutex per request.
- Fix: add a trusted-proxy CIDR setting and derive the client IP from it; sweep on a timer.

### 3.3 Web UI (`internal/web`)

**WEB-01 high. Setup secret, secure-cookie and CSRF-key options are never wired; anyone can register and create networks**
- `cmd/circula/serve.go:155` calls `web.NewHandler(svc)` (zero `Options`). `SetupSecret` (`web/web.go:23`) is therefore always empty: `/setup` lets any signed-in person create and activate a network (`web/h_auth.go:140-172`), and `/register` is open (`h_auth.go:92`). Spec: FR-002 (P0: one-time setup secret, invalidated after first use, not logged) is unbuilt; no config key exists (`settingDefs`, `cmd/circula/config.go:40`).
- `config.go:57` defines `cookie-secure` ("reserved until the web adapter reads it"); `Config.CookieSecure` is parsed and never used. Cookies get `Secure` only if the request has TLS or `X-Forwarded-Proto: https` (`web/session.go:27`), a header any client can send and a TLS-terminating proxy often does not. SEC-003/SEC-022 gap.
- `public-base-url` (required) is only logged (`serve.go:163`). Invitation links use `r.Host` (`web/h_admin.go:271`). CSRF `sameOrigin` compares `Origin` to `r.Host` (`web/middleware.go:69`), which breaks behind proxies that rewrite Host.
- Also, the comparison of the setup secret (`h_auth.go:147`) is not constant time.
- Fix: map `setup-secret`/`cookie-secure`/`public-base-url` into `web.Options`; make the secret one-time (store used flag in DB, audited); use `PublicBaseURL` for links and origin checks.

**WEB-02 med. First-100 / first-N views hide older items**
- `web/h_home.go:75` dashboard reads the newest 100 transactions only: an older proposal that still needs the member's acceptance is not shown under "Needs attention" (FR-053). `web/h_disputes.go:78` review queue and `:64` disputes list: 100 newest, no paging, so oldest open disputes (the ones nearest deadline) drop off the queue. `web/h_trades.go:265` finds the dispute for a disputed trade by scanning the newest 100 disputes instead of filtering by `TxID` (the store supports `DisputeFilter.TxID`, the service does not expose it).
- Fix: status-filtered, paged service queries (`ListTransactions(statuses=[proposed])`), `ListDisputes` with `TxID`.

**WEB-03 low. Per-request full-directory loads**
- `web/h_home.go:130` `directoryAll` loads up to 10 000 members per call; used in propose form and POST (`h_trades.go:85,124`); `web/format.go:196,209` load directory/admin lists for names. N+1 style, fine for tens of members, not for thousands. Same fix as API-04.

**WEB-04 low. Session-expiry warning missing**
- UX-030 (P0: warn at least 2 minutes before expiry with an option to extend). `static/app.js` and templates contain no such code (grep for expiry/extend found nothing); sessions have no extend path beyond the idle refresh in `Authenticate`.

### 3.4 CLI and operations (`cmd/circula`)

**CMD-01 med. Backup verification mutates the backup**
- `cmd/circula/snapshot.go:157` verifies the snapshot by opening `.partial` with `store.Open` and calling `svc.VerifyLedger` (via `verifyFile`, `:110-125`), which writes a `consistency_runs` row and an audit event per network into the snapshot (SVC-01). The integrity check (`:147`) ran BEFORE this write, and the SHA-256 (`:164`) is taken after, so the manifest checksums a file that differs from the live database (extra audit events with request id `backup-verify` that exist only in the backup). `removeSideFiles(partial)` at `:162` then deletes any `-wal`/`-shm` left beside the snapshot, so the post-verify state of the file is never integrity-checked.
- Fix: read-only verify (SVC-01) on the snapshot opened `mode=ro`; or verify, then discard, then copy.

**CMD-02 med. Operator commands write without the single-writer lock**
- `cmdVerifyLedger` (`commands.go:350`) and `cmdAdmin create-operator` (`:381`) call `openDatabase` but never `acquireLock` (only `migrate` `:119`, `restore` `:238`, and `serve` lock). While `serve` runs they become a second writer process (VerifyLedger writes; `RegisterPerson` writes). OPS-011 ("MUST NOT be shared with another process other than the app's own backup routine") and ARC-011 are violated. The in-process write gate does not span processes; only SQLite's lock does, with a 10 s busy timeout.
- Fix: for `verify-ledger` use the read-only path (SVC-01) and no lock; for `create-operator` either require the lock (refuse if `serve` runs) or route through the running service.

**CMD-03 low. `cmd` reads SQLite directly**
- `snapshot.go:63` lists networks with raw SQL against the store's DB (`st.DB()`); `schema.go` `inspectSchema` and `ops_http.go:202` read `schema_migrations`; `serve.go:242` `checkPragmas` queries pragmas. Duplicates schema knowledge outside `internal/store`; a renamed table breaks the CLI silently. Fix: add `store.ListNetworks`, `store.SchemaState`, `store.Pragmas` and use them.

**CMD-04 low. Hourly verify job floods the audit log**
- `serve.go:279,326` runs `VerifyLedger` for every network each hour, each run appending an audit event and a run row (SVC-01): 24 x N rows/day, visible to admins and included in every export. Startup, `migrate`, and `restore` add more.

**CMD-05 med. SQLITE_BUSY surfaces as 500, not retryable**
- `store/sqlite.go:31,81` busy timeout 10 s; the resulting `store: begin: ... database is locked` is a plain error, not a `domain.Error`, so `api/respond.go` `fromErr` returns `internal-error` (500) and `web` renders the error page. grep finds no mapping for busy/locked. OPS-011/OPS-012 acceptance ("returns a retryable error", "A posting that times out waiting for the lock leaves no partial state") needs a distinguishable retryable code (503 + `Retry-After`). Partial state is not left (verified: rollback in `run`), only the error class is wrong.

**CMD-06 low. Durability and runbook text**
- `store/sqlite.go:33` hard-codes `synchronous(NORMAL)`; OPS-011 wants a documented setting the operator can read (documented in `docs/ops/monitoring.md:67`, but not configurable). The same runbook line says to "reconcile recent activity from the audit trail" after power loss, but audit rows commit in the same unit as the data, so a lost transaction loses its audit row too. Fix text, or offer `synchronous=FULL` as a setting for ledger-grade durability.

**CMD-07 low. Startup refuses to serve all tenants if one network fails verification**
- `serve.go:143-146`. Matches OPS-016/OPS-020 as written, but a single failing network takes every network offline, and the check itself is slow and write-locking (SVC-01). Consider serving read-only or per-network degrade; record the decision.

**CMD-08 low. `restore` leaves a failing database in place**
- `commands.go:209` `cmdRestore`: the database is replaced (rename) before the post-restore verification; on failure it exits 3 but the failed data stays and `serve` then refuses to start. A pre-restore backup exists, so recovery is possible; documented in the message. Acceptable, listed for completeness.

### 3.5 Docs, dead code, maintainability

**DOC-01 low. `docs/dev/SERVICE.md` is stale**
- Line 7 "no HTTP, UI or CLI yet"; section 6 lists CSRF (adapter), second-process detection (CLI) and backups before migration (CLI) as open; all are now implemented in `internal/web`, `cmd/circula`. Line "Writes through it fail" in code (SVC-02) also repeated in design intent.

**DEAD-01 low. Service code with no production caller**
- `CloseNetwork` (FR-068 P1), `AdminListTransactions`, `BalanceAt` (FR-054 P1), `ChangePassword` (AUTH-03), `domain.TxStatus.IsTerminal`, `web.NewHandlerWith` and `api.NewHandlerWithOptions` (the options they carry are the unreachable settings in WEB-01). Config keys never read: `cookie-secure`, `public-base-url` (log only). Each is tested but cannot be reached by an operator or user.

**MAINT-01 low. Duplicated layers**
- Two idempotency layers (in-memory `api/idem.go`, durable `service/idempotency.go`) with different scopes (API: person+key, path in fingerprint; service: network+person+operation). API replay is lost on restart for commands the service does not make idempotent (departure, member actions, invitations, consistency checks, exports). Two CSRF schemes (API-01). Four copies of "scan the list to find one item" (API-04). `web/refusal.go` and `api/respond.go` each maintain their own error-code mapping.

## 4. Spec-to-code gaps (P0/P1 not built or only partly built)

Legend: P0 = MVP-blocking, P1 = MVP. "Partial" = exists in service but missing at an interface.

| ID | Pri | Status | Notes |
|---|---|---|---|
| FR-002 first-admin setup secret | P0 | not built | WEB-01 |
| SEC-003 password change (and its acceptance tests) | P0 | partial | AUTH-03 |
| SEC-008 account recovery | P0 | not built | SERVICE.md lists as not implemented; no route, no token table |
| SEC-013 operator break-glass | P0 | not built | no operator role in code; `admin create-operator` only creates a person, with no special rights |
| SEC-018 retention / erasure / pseudonymisation | P0 | not built | no code touches `display_name` after departure |
| SEC-012 notices to affected members | P0 | not built | (FR-038 notifications, P1, also not built) |
| UX-030 session expiry warning | P0 | not built | WEB-04 |
| API-008 browser session cookie for API | P0 | broken | API-01 |
| API-026/027 `If-Match` on membership status | P0 | partial | API-02 |
| API-045 entries ordered by `ledger_seq` | P0 | deviates | API-03 |
| FR-011 / API-059 deny a join request | P0 | partial | service `DecideJoinRequest(approve=false)` and web "deny" exist; API has only `approve` route (`api/routes.go:57`) |
| OPS-016 / FR-052 verification must not modify data | P0 | violated | SVC-01 |
| OPS-011/OPS-012 retryable error on busy DB | P0 | violated | CMD-05 |
| OPS-011 no second writer process | P0 | violated | CMD-02 |
| FR-053 pending items view | P1 | partial | WEB-02 (older items hidden) |
| FR-054 balance at past date | P1 | partial | service `BalanceAt` only, no UI/API |
| FR-064 default flag | P1 | partial | web only; no API route |
| FR-067 admin overview | P1 | partial | web shows pending-member count only |
| FR-068 network closure | P1 | partial | `CloseNetwork` has no UI/API caller |
| FR-024 report listing/member | P1 | not built | |
| FR-038 notifications and inbox | P1 | not built | |
| FR-076 response by other party; FR-078 one re-review | P1 | not built | |
| API-010 `admin` token scope | P1 | 501 | `api/h_tokens.go:63` |
| API-064 announcements | P1 | 501 | `api/routes.go:20,70,71` |
| API-077/078 OpenAPI + compatibility fixtures | P1 | not built | no OpenAPI file in tree |
| DAT-012 one reversal per original | P1 | deviation | documented; see section 2 |
| DAT-047 10 000-tx migration fixture; DAT-048 timing | P0/P1 | not built | only one migration exists, no fixture test |
| FR-095 export correctness/performance gates | P1 | not built | no large-data export test; API buffers (API-05) |
| SEC-007 MFA, SEC-011 dual control | P1 | not built | |

## 5. Test gaps

- Zero-coverage or near-zero: `runServe` 16 %, all job runner code (`expire`, `verify`, `backups`, `untilNextBackup`, `backupNow`) 0 %, `cmdBackup` 0 %, `cmdAdmin`/`readOperatorPassword` 0 %, `preMigrationBackup` 0 %, `checkPragmas` 0 %, `cmdMigrate` 23 %.
- API: `reviewDispute`, `withdrawDispute`, `retryDispute` 0 %, `getExport` 0 %, `roleAction` 7 %.
- Web: `invitationRevoke` 0 %, `disputeFail` 0 %, `joinNetPost` 26 %.
- Missing tests that would have caught findings: concurrent failed logins (AUTH-01); write inside `View` (SVC-02); `VerifyLedger` leaves DB checksum unchanged (SVC-01, OPS-016 acceptance); backup file equals source plus no extra audit rows (CMD-01); API call using the cookie that the web sign-in sets (API-01); handler panic then retry with same key (API-06); more than 100/500/2000 members or disputes (API-04, WEB-02); busy-database request returns retryable code (CMD-05); export of a large ledger (API-05, FR-095); a departed or denied person calling join (SVC-04).
- Positive: property tests, fault injection (`testutil/fault.go`), atomicity, append-only raw SQL, concurrency and cross-network isolation tests exist and pass under `-race`.

## 6. Suggested order of work

1. SVC-01 (read-only verify) then CMD-01, CMD-02, CMD-04 follow from it.
2. WEB-01 and API-01 (wire options; one cookie name and CSRF scheme).
3. SVC-02 (`query_only` views), AUTH-01 (atomic throttle), CMD-05 (busy -> 503).
4. API-02, API-03, API-04 plus the missing service reads.
5. Spec gaps in section 4 marked P0.
6. Remaining low items and test additions in section 5.
