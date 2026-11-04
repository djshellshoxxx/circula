# Decision log

Status: living log · Last updated: 2026-10-21 · Scope: decisions made during the build (accepted as built), open owner decisions raised by the build, owner decisions still open from [13](specs/13-assumptions-risks-decisions.md), and the open questions that were resolved into decisions · Depends on: [13](specs/13-assumptions-risks-decisions.md) (register and template), [TRACEABILITY.md](TRACEABILITY.md)

Entries use the template in [13 section 1.3](specs/13-assumptions-risks-decisions.md#13-decision-log-template). An accepted entry records what the build does; it is not an owner sign-off unless its status says so. Open entries give the options and a recommended default, which is not a decision (13 section 1.2). Numbers are never reused. Entries DEC-001 to DEC-027 keep the numbering of 00 and 13; new decisions start at DEC-028.

## 1. Accepted as built (DEC-028 to DEC-071)

### DEC-028 · Go 1.26 toolchain

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: Language and stdlib choice for the service (DEC-001).
- Decision: Module pins `go 1.26.10`; CI reads go.mod. Toolchain moved 1.25.9 -> 1.25.14 -> 1.26.10 for stdlib CVE fixes (commits e7677f4, ef34159, 5aee3ae).
- Rationale: Stdlib net/http and html/template security fixes flagged by govulncheck.
- Recommended default and whether adopted: Go 1.26 (DEC-001 baseline)
- Requirements or documents changed: 00 section 6; 13 register
- Verification impact: govulncheck runs in CI (ci.yml); go vet clean in this pass
- Review or revisit: each Go minor release
- Counsel review required: no
- Code and tests: `.github/workflows/ci.yml` (ci.yml)

### DEC-029 · SQLite through a pure-Go driver, WAL, foreign keys, busy timeout

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead / operations lead
- Context and requirements: Storage default (DEC-001, ARC-004, OPS-011).
- Decision: modernc.org/sqlite (no cgo). Every connection opens with WAL, foreign_keys on, busy_timeout 5 s (`DefaultBusyTimeout`) and synchronous=NORMAL. Migrations are embedded and forward-only.
- Rationale: Keeps all five release targets cgo-free (DEC-003). synchronous=NORMAL is the only durability setting today (see DEC-075).
- Recommended default and whether adopted: As DEC-001
- Requirements or documents changed: 05 ARC-004; 11 OPS-011
- Verification impact: TestBusyDatabaseIsRetryable TestMigrationsIdempotentAndVersioned
- Review or revisit: before hosted launch
- Counsel review required: no
- Code and tests: `internal/store/sqlite.go` (DefaultBusyTimeout); `internal/store/hardening_test.go` (TestBusyDatabaseIsRetryable); `internal/store/sqlite_test.go` (TestMigrationsIdempotentAndVersioned)

### DEC-030 · One writer process per database file

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: ARC-011, OPS-011, SA/CMD-02 (second writer process).
- Decision: A lock file (per-OS lock in cmd/circula) is held by `serve`, `migrate`, `restore` and operator commands that write. Inside a process a write gate serialises `Update`; across processes SQLite `BEGIN IMMEDIATE` serialises writers. A second `serve` exits with 'another circula process holds the database lock'.
- Rationale: SQLite gives single-writer semantics; the lock stops a second writer from silently sharing the file.
- Recommended default and whether adopted: Single process (ARC-014)
- Requirements or documents changed: 05 ARC-011; 11 OPS-011
- Verification impact: TestLockRefusesSecondHolder TestOperatorCommandsTakeTheLock
- Review or revisit: multi-node design
- Counsel review required: no
- Code and tests: `cmd/circula/lock_other.go` (acquireLock); `cmd/circula/backup_test.go` (TestLockRefusesSecondHolder); `cmd/circula/hardening_test.go` (TestOperatorCommandsTakeTheLock)

### DEC-031 · Ledger verification is read-only; recorded runs are an explicit admin action

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: OPS-016, FR-052, SVC-01 (verify used to write audit rows and run rows).
- Decision: `VerifyLedger` runs on `Store.View` and writes nothing. `VerifyAndRecordLedger` (admin API and web check only) also appends a consistency run and a `ledger.verified` audit event. Startup, hourly, migrate, restore and backup-verify callers use the read-only path.
- Rationale: Scheduled verification must not add audit rows or hold the write gate; an admin-initiated check is still recorded.
- Recommended default and whether adopted: Read-only verification plus recorded admin run
- Requirements or documents changed: 05 ARC-015; 11 OPS-016
- Verification impact: TestVerifyLedgerIsReadOnly TestVerifyLedgerLeavesDatabaseChecksumUnchanged
- Review or revisit: if scheduled runs must be recorded (ARC-015 wording)
- Counsel review required: no
- Code and tests: `internal/service/verify.go` (VerifyLedger); `internal/service/verify.go` (VerifyAndRecordLedger); `internal/service/hardening_test.go` (TestVerifyLedgerIsReadOnly); `cmd/circula/cli_regress_test.go` (TestVerifyLedgerLeavesDatabaseChecksumUnchanged)

### DEC-032 · Read views are enforced read-only at the connection

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: SVC-02 (writes inside View were not blocked).
- Decision: `Store.View` sets `PRAGMA query_only=ON` on its connection for the unit and resets it afterwards, so a write inside a view fails.
- Rationale: Keeps the write gate and LED-5 serialisation from being bypassed by a future mistake.
- Recommended default and whether adopted: n/a
- Requirements or documents changed: docs/dev/SERVICE.md section 1
- Verification impact: TestViewIsReadOnly
- Review or revisit: never
- Counsel review required: no
- Code and tests: `internal/store/hardening_test.go` (TestViewIsReadOnly); `internal/store/sqlite.go` (query_only)

### DEC-033 · Cookie names, Secure flag and CSRF scheme shared by web and API

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead / security owner
- Context and requirements: API-008, SEC-003, SEC-004, SEC-022 (web and API cookie names differed; Secure flag not forced).
- Decision: One constant set in internal/httpx: `__Host-circula_session` when Secure applies, otherwise `circula_session`; CSRF cookie `cm_csrf` with an HMAC-signed double-submit token. `cookie-secure` (default true) forces Secure; HSTS is sent on HTTPS. The API reads the same cookie.
- Rationale: A browser sign-in must also work for API calls; Secure must not depend on a header the client sends.
- Recommended default and whether adopted: cookie-secure=true
- Requirements or documents changed: 07 API-008; 09 SEC-003, SEC-004
- Verification impact: TestWebSessionWorksForAPI TestSecureCookiesAndHSTS TestCSRFScheme
- Review or revisit: if CSRF must bind to the session (see SEC-004 status)
- Counsel review required: no
- Code and tests: `internal/httpx/httpx.go` (SessionCookieSecure); `internal/httpx/httpx.go` (CSRF); `internal/api/hardening_test.go` (TestWebSessionWorksForAPI); `internal/web/hardening_test.go` (TestSecureCookiesAndHSTS); `internal/httpx/httpx_test.go` (TestCSRFScheme)

### DEC-034 · First-network setup secret

- Status: accepted (as built), partly
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner
- Context and requirements: FR-002 (one-time setup secret, invalidated after first use, not logged).
- Decision: `setup-secret-file` or `CIRCULA_SETUP_SECRET` must match before `/setup` creates a network; compared in constant time. After one successful use the process refuses further setups (in-memory flag). A restart clears the flag while the secret is still configured.
- Rationale: Stops anonymous sign-ups creating networks (SA-03) without a platform role model.
- Recommended default and whether adopted: Secret required when set; unset disables the gate
- Requirements or documents changed: 04 FR-002
- Verification impact: TestSetupSecretEnforced secretEqual
- Review or revisit: persist the used flag in the database
- Counsel review required: no
- Code and tests: `internal/web/h_auth.go` (secretEqual); `internal/web/h_auth.go` (setupPost); `internal/web/hardening_test.go` (TestSetupSecretEnforced)

### DEC-035 · Request-proposal reservation rule

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: DOM-011, DOM-012, SA-08 (a request froze the payer's credit without consent).
- Decision: A payer's reservation is counted only for proposals the payer made. A request (payee proposes) is checked against the payer's available credit when made, its open amount must fit, and a proposer may hold at most `MaxOpenRequestsPerPair` (5) open requests per payer and `MaxOpenProposalsPerMember` (200) open proposals. Both refusals return rate-limited. Acceptance re-checks the hard limit.
- Rationale: Consent before reservation (principle 3); blocks credit-freezing by unsolicited requests.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 03 DOM-012; 13 (none)
- Verification impact: TestUnsolicitedRequestsDoNotFreezeCredit TestRequestExposureAndCounts TestOpenProposalsPerMemberCap
- Review or revisit: when DOM-012 wording is revised
- Counsel review required: no
- Code and tests: `internal/service/transactions.go` (ProposeTransaction); `internal/service/hardening_test.go` (TestUnsolicitedRequestsDoNotFreezeCredit); `internal/service/hardening_test.go` (TestRequestExposureAndCounts); `internal/service/hardening_test.go` (TestOpenProposalsPerMemberCap)

### DEC-036 · `accepted` is never persisted; audit records proposed to completed

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: OQ-DOM-1, OQ-ARC-04 (whether accepted is a separate persisted or audited state).
- Decision: Acceptance and posting run in one unit. `accepted` is not stored and no separate audit event is written; the audit trail shows `proposed` -> `completed`.
- Rationale: Matches the atomic accept-and-post design (DOM-014) and avoids a two-phase state.
- Recommended default and whether adopted: Recommended default in 03 adopted
- Requirements or documents changed: 03 OQ-DOM-1 (moved here); 05 OQ-ARC-04 (moved here); 08 UX-016
- Verification impact: TestTxStateMachineExhaustive TestAcceptFaultInjectionLeavesNoTrace
- Review or revisit: see DEC-074 (visibility)
- Counsel review required: no
- Resolves: OQ-DOM-1, OQ-ARC-04 (the question text is removed from the open list in its spec)
- Code and tests: `internal/domain/statemachine_test.go` (TestTxStateMachineExhaustive); `internal/service/atomicity_test.go` (TestAcceptFaultInjectionLeavesNoTrace)

### DEC-037 · Proposal expiry fixed when the proposal is created

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: OQ-DOM-7 (recalculate expiry on policy change?).
- Decision: `ExpiresAt` is set from the policy in force at proposal time and stored on the transaction. Policy changes do not move it. A system job (`ExpireProposals`, every 5 minutes) expires elapsed proposals with no posting.
- Rationale: Prospective policy changes (DOM-030, GOV-001).
- Recommended default and whether adopted: No recalculation (recommended default adopted)
- Requirements or documents changed: 03 OQ-DOM-7 (moved here)
- Verification impact: TestExpiry
- Review or revisit: never without a new decision
- Counsel review required: no
- Resolves: OQ-DOM-7 (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/transactions.go` (ExpireProposals); `internal/service/trades_test.go` (TestExpiry)

### DEC-038 · Departed members cannot rejoin; suspended members are refused

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: FR-011, FR-017 (re-application after denial or removal); SVC-04.
- Decision: A departed person has no second membership (UNIQUE network+person). Joining again returns the existing pending or active membership unchanged; a suspended or departed person gets `forbidden` and no change. Re-application needs a new DOM-003 transition.
- Rationale: DOM-003 has no transition out of departed; a success reply for a refused person misled users.
- Recommended default and whether adopted: Not supported
- Requirements or documents changed: 04 FR-011 (open point closed as not supported)
- Verification impact: TestRejoinSemantics
- Review or revisit: if re-application after denial is wanted
- Counsel review required: no
- Code and tests: `internal/service/hardening_test.go` (TestRejoinSemantics); `internal/service/join.go` (JoinNetwork)

### DEC-039 · Upheld reversal blocked by the limit: posting_pending, audited override, retry

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: OQ-DOM-2, DOM-021, FR-074 (what happens when an upheld reversal would breach LED-5).
- Decision: The dispute moves to `posting_pending` and no entry is written. The resolution is a time-limited, reasoned credit-limit override by an administrator (`SetCreditLimit` with an expiry), after which `RetryDisputePosting` posts the reversal. No path posts past the limit.
- Rationale: Both paths in the recommended default were built; the ledger rule is not bent.
- Recommended default and whether adopted: Both paths (recommended default adopted)
- Requirements or documents changed: 03 OQ-DOM-2 (moved here)
- Verification impact: TestUpheldReversalBlockedThenOverride TestLimitOverrideExpires
- Review or revisit: if holding until recovery is also wanted
- Counsel review required: no
- Resolves: OQ-DOM-2 (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/disputes_decide.go` (RetryDisputePosting); `internal/service/dispute_test.go` (TestUpheldReversalBlockedThenOverride); `internal/service/trades_test.go` (TestLimitOverrideExpires)

### DEC-040 · Partial reversals: cumulative cap instead of one reversal per original

- Status: accepted (as built), deviation from DAT-012
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: DOM-020 (cumulative partial reversals) versus DAT-012 (one reversal per original).
- Decision: Any number of reversals may reference one original, provided the cumulative reversed total never exceeds the original amount (DB CHECK `reversed_total <= amount`, checked by VerifyLedger). A full reversal sets `reversed`.
- Rationale: DOM-020 requires partial reversals; DAT-012 contradicts it.
- Recommended default and whether adopted: DOM-020 governs
- Requirements or documents changed: 06 DAT-012; 07 API-056
- Verification impact: TestPartialReversalCumulativeCap ReversalMismatches
- Review or revisit: when DAT-012 is revised
- Counsel review required: no
- Code and tests: `internal/service/reversal_test.go` (TestPartialReversalCumulativeCap); `internal/store/sqlite_verify.go` (ReversalMismatches)

### DEC-041 · Uniform response on sign-up

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner
- Context and requirements: SEC-005, SA-10 (registration revealed which logins exist).
- Decision: A taken login returns the same success page as a new one, with a decoy person and an audit row (`person.register_duplicate`). No auto sign-in follows sign-up; the message is the same for both outcomes.
- Rationale: Anti-enumeration. Trade-off: a user who already has an account is not told so.
- Recommended default and whether adopted: Uniform response
- Requirements or documents changed: 09 SEC-005
- Verification impact: TestRegisterIsUniform TestLoginDoesNotRevealExistence
- Review or revisit: if out-of-band confirmation is added
- Counsel review required: no
- Code and tests: `internal/service/auth.go` (RegisterPerson); `internal/service/hardening_test.go` (TestRegisterIsUniform); `internal/service/auth_test.go` (TestLoginDoesNotRevealExistence)

### DEC-042 · Login throttling: atomic per login, per address, and web limits

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner
- Context and requirements: SEC-006, AUTH-01, SA-04 (lost-update race let parallel guesses bypass the lock).
- Decision: The failure check and count run in one write unit before the password is verified (at most five evaluated guesses per window per login). Web limits: 20 failures/60 s per address, 5 failures/60 s per login, 5 password-change failures/600 s, 10 registrations/60 s, 10 setup attempts/600 s. Successful sign-ins do not use the budget. No per-network or invitation-redemption limit.
- Rationale: Makes the account lock hold under concurrency.
- Recommended default and whether adopted: Values as built
- Requirements or documents changed: 09 SEC-006
- Verification impact: TestLoginThrottleIsAtomicUnderConcurrency TestLoginThrottleKeyIsBounded TestSuccessfulLoginsDoNotUseTheBudget
- Review or revisit: before hosted launch (DEC-006 bounds)
- Counsel review required: no
- Code and tests: `internal/service/auth.go` (Login); `internal/api/middleware.go` (newLimiter); `internal/service/hardening_test.go` (TestLoginThrottleIsAtomicUnderConcurrency); `internal/service/hardening_test.go` (TestLoginThrottleKeyIsBounded); `internal/web/hardening_test.go` (TestSuccessfulLoginsDoNotUseTheBudget)

### DEC-043 · Trusted proxies decide the client address

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: SA-05, API-09 (one proxy address throttled every user).
- Decision: `trusted-proxies` (default none) lists CIDRs whose X-Forwarded-For and X-Forwarded-Proto headers are believed. Otherwise the socket address is used.
- Rationale: Headers from untrusted clients must not change rate limits or the Secure decision.
- Recommended default and whether adopted: none
- Requirements or documents changed: 11 OPS-008; 09 SEC-006
- Verification impact: TestTrustedProxiesSetting TestIPRateLimitUsesTrustedProxyClientIP TestClientIP
- Review or revisit: never
- Counsel review required: no
- Code and tests: `internal/httpx/httpx.go` (ClientIP); `cmd/circula/hardening_test.go` (TestTrustedProxiesSetting); `internal/api/hardening_test.go` (TestIPRateLimitUsesTrustedProxyClientIP); `internal/httpx/httpx_test.go` (TestClientIP)

### DEC-044 · Audit hash chain (unkeyed) with a per-scope head

- Status: accepted (as built); owner to confirm
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: DAT-030, FR-085, SEC-034, DEC-015, OQ-DAT-04, SA-09 (tail truncation undetected).
- Decision: Each audit event carries an unkeyed SHA-256 over its content and the previous hash. An `audit_heads` row holds the newest (seq, hash) per scope and is advanced in the same unit; VerifyLedger compares it. Keyed chains or external publication of the head are not built.
- Rationale: Detects edits and tail deletion by anyone without the database trigger rights; a full rewrite is still undetected.
- Recommended default and whether adopted: Unkeyed chain plus head (DEC-015 recommends deferring the full chain; this is a lighter form)
- Requirements or documents changed: 06 DAT-030; 09 SEC-034
- Verification impact: TestAuditChainDetectsTampering TestAuditTailTruncationDetected TestAuditHeadTracksAppendsAndIsProtected
- Review or revisit: before hosted launch
- Counsel review required: no
- Resolves: OQ-DAT-04 (as built, owner to confirm) (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/audit.go` (AuditHash); `internal/service/audit_test.go` (TestAuditChainDetectsTampering); `internal/service/hardening_test.go` (TestAuditTailTruncationDetected); `internal/service/hardening_test.go` (TestAuditHeadTracksAppendsAndIsProtected)

### DEC-045 · Backup manifests signed with an operator key; restore checks the signature

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner / operations lead
- Context and requirements: SEC-028, OPS-019, SA-07 (a backup editor could recompute the checksum).
- Decision: When `backup-key-file` or CIRCULA_BACKUP_KEY is set, each manifest carries an HMAC-SHA-256 and `restore` refuses an unsigned or unverifiable manifest. Without the key, restore warns that the backup is unauthenticated. Backup files are not encrypted (OPS-019 open).
- Rationale: Integrity against a person with write access to backup storage; confidentiality is still open.
- Recommended default and whether adopted: Key recommended; encryption open
- Requirements or documents changed: 11 OPS-019; ops backup-restore.md
- Verification impact: TestSignedBackupManifest TestBackupCommandSignsManifestWithKey TestRestoreRefusesBackupFailingEmbeddedChecks
- Review or revisit: with DEC-008 and encryption decision
- Counsel review required: no
- Code and tests: `cmd/circula/snapshot.go` (checkManifestAuth); `cmd/circula/snapshot.go` (manifestMAC); `cmd/circula/hardening_test.go` (TestSignedBackupManifest); `cmd/circula/cli_regress_test.go` (TestBackupCommandSignsManifestWithKey)

### DEC-046 · Retention purge is a service method run by the job runner

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: SA-12, DATA-01, SEC-018 (no purge of dead sessions, throttle rows, idempotency rows).
- Decision: `Service.Purge` deletes dead sessions, stale throttle rows and expired idempotency records in batches of 1000, one write unit each. It never touches ledger, audit or membership data. The job runner calls it hourly.
- Rationale: Bounded growth without touching ledger data (LED-2).
- Recommended default and whether adopted: Hourly
- Requirements or documents changed: 06 section 9; 11 OPS-027
- Verification impact: TestPurge TestPurgePrimitives TestJobPurgeRemovesIdleSessions
- Review or revisit: when personal-data erasure is specified (SEC-018)
- Counsel review required: no
- Code and tests: `internal/service/purge.go` (Purge); `internal/service/hardening_test.go` (TestPurge); `cmd/circula/jobs_regress_test.go` (TestJobPurgeRemovesIdleSessions)

### DEC-047 · Export artefacts held in memory for an interim period

- Status: accepted (as built), interim
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: FR-095, API-05, SA-11 (exports buffered; one member could exhaust a global cap).
- Decision: Artefacts are kept in process memory for 24 h. Limits: 50 live exports and 64 MiB globally, and a per-person cap (`maxExportsPerPerson`). Export calls are also limited per member per network per hour (`ExportsPerHour`, default 30). Artefacts are lost on restart.
- Rationale: Keeps the MVP simple; known to fail FR-095 streaming at large sizes.
- Recommended default and whether adopted: Interim; replace with streamed files
- Requirements or documents changed: 04 FR-095; 07 API-065
- Verification impact: TestExportCapPerPerson TestExportQuotaPerActor
- Review or revisit: before pilot with large networks (OQ-FR-3)
- Counsel review required: no
- Code and tests: `internal/api/jobs.go` (jobTTL); `internal/service/export.go` (MemSink); `internal/api/hardening_test.go` (TestExportCapPerPerson); `internal/service/hardening_test.go` (TestExportQuotaPerActor)

### DEC-048 · Busy database is a retryable 503

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: OPS-012, CMD-05 (busy surfaced as 500).
- Decision: SQLITE_BUSY past the busy timeout maps to `unavailable` (HTTP 503, Retry-After). Nothing is written.
- Rationale: Clients can retry safely.
- Recommended default and whether adopted: 503 with Retry-After
- Requirements or documents changed: 07 section 7.2; 11 OPS-012
- Verification impact: TestBusyDatabaseIs503 TestBusyMapsTo503 TestIsBusy
- Review or revisit: never
- Counsel review required: no
- Code and tests: `internal/store/sqlite.go` (isBusy); `internal/web/hardening_test.go` (TestBusyDatabaseIs503); `internal/api/internal_test.go` (TestBusyMapsTo503)

### DEC-049 · Departure default: retain balance; settle-to-zero by adjustment

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: OQ-DOM-4 (default departure rule).
- Decision: Default `departure_rule` is `retain`: balance stays on the closed account. `settle-to-zero` is available and posts a reasoned adjustment to the system account. Both are configurable per network.
- Rationale: Settlement meaning depends on network governance (03 OQ-DOM-4 reasoning).
- Recommended default and whether adopted: retain (recommended default adopted)
- Requirements or documents changed: 03 OQ-DOM-4 (moved here); 10 section 2
- Verification impact: TestDepartureRetain TestDepartureSettleToZero
- Review or revisit: counsel review (RSK-020)
- Counsel review required: yes (RSK-020)
- Resolves: OQ-DOM-4 (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/departure.go` (DepartMember); `internal/service/admin_test.go` (TestDepartureRetain); `internal/service/admin_test.go` (TestDepartureSettleToZero)

### DEC-050 · Platform bounds on policy values

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: OQ-DOM-5, OQ-DOM-6, OQ-DAT-06 (platform maxima for expiry, dispute window, decimals, reason length).
- Decision: Proposal expiry 1 to 365 days. Dispute window 1 to 3650 days. Limit warning 0 (off) to 3650 days. Decimals 0 to 4. Amounts and limits up to 2^50 minor units. Reasons and free text up to 500 characters.
- Rationale: Keeps minor-unit magnitudes well inside int64 (LED-7).
- Recommended default and whether adopted: Values as built (recommended 4 for decimals adopted)
- Requirements or documents changed: 03 section 13; 06 OQ-DAT-06 (moved here)
- Verification impact: TestPolicyBounds TestAddChecked
- Review or revisit: if a longer dispute window is needed
- Counsel review required: no
- Resolves: OQ-DOM-5, OQ-DOM-6, OQ-DAT-06 (the question text is removed from the open list in its spec)
- Code and tests: `internal/domain/policy.go` (Validate); `internal/domain/policy.go` (MaxProposalExpiryDays); `internal/domain/money.go` (MaxDecimals); `internal/domain/statemachine_test.go` (TestPolicyBounds)

### DEC-051 · Sign-up grant is off by default

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner (confirm)
- Context and requirements: OQ-DOM-3, DEC-011, SEC-032 (sign-up grant in MVP and default amount).
- Decision: Default `signup_grant` is 0 (disabled). When set, the grant posts from the system account when a member becomes active, as an audited grant. Caps and the open-join confirmation required by SEC-032 are not built.
- Rationale: Recommended default; free balances attract Sybil accounts (T-18).
- Recommended default and whether adopted: Disabled (adopted); SEC-032 controls open
- Requirements or documents changed: 03 OQ-DOM-3 (moved here); 09 SEC-032
- Verification impact: TestSignupGrant
- Review or revisit: before enabling in any open network
- Counsel review required: no
- Resolves: OQ-DOM-3 (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/adjustments.go` (postSignupGrant); `internal/service/admin_test.go` (TestSignupGrant)

### DEC-052 · Administrative reversal: completed only; disputed trades only through a dispute

- Status: accepted (as built), deviation
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: DOM-036, DOM-010 (admin reversal of a disputed trade).
- Decision: An administrator reversal needs status `completed`. A `disputed` trade is reversed only by the dispute decision.
- Rationale: One path for disputed money; avoids two reviewers acting on one trade.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 03 DOM-036
- Verification impact: TestReversalAuthorization TestReversalBlockedByLimit
- Review or revisit: if admin reversal of disputed trades is required
- Counsel review required: no
- Code and tests: `internal/service/reversal.go` (ReverseTransaction); `internal/service/reversal_test.go` (TestReversalAuthorization)

### DEC-053 · Administrators only invite and approve; admins see balances as limit and balance only; directory shows active members

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner (confirm)
- Context and requirements: OQ-PER-1, OQ-PER-3, OQ-PER-4, OQ-FR-1 (who may invite; admin view of balances; suspended directory access).
- Decision: Only administrators create invitations; approval is administrator-only. An administrator who is a member sees another member's balance and limit (audited `account.viewed`). The directory lists active members to active members and administrators; suspended and departed members see no directory.
- Rationale: Matches the recommended defaults in 02 and 04.
- Recommended default and whether adopted: Recommended defaults adopted
- Requirements or documents changed: 02 OQ-PER-1/3/4 and 04 OQ-FR-1 (moved here)
- Verification impact: TestAdminBalanceViewIsAudited TestInvitations TestJoinModes
- Review or revisit: OQ-PER-1 if member invitations are wanted
- Counsel review required: no
- Resolves: OQ-PER-1, OQ-PER-3, OQ-PER-4, OQ-FR-1 (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/accounts.go` (AdminGetMemberBalance); `internal/service/join.go` (CreateInvitation); `internal/service/membership.go` (ListDirectory); `internal/service/authz_test.go` (TestAdminBalanceViewIsAudited)

### DEC-054 · MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner (confirm)
- Context and requirements: OQ-UX-03, OQ-UX-05, OQ-UX-07, OQ-FR-5, OQ-FR-6, OQ-DAT-05 (scope questions).
- Decision: UI and API text in English only; strings are not externalised (UX-047 not built). Listing prices optional. Open join mode is built as a configuration option (FR-012). Trades have exactly two parties and one amount. No per-member remembered preferences. Entries are never partially settled.
- Rationale: Recommended defaults in 02, 04, 06, 08.
- Recommended default and whether adopted: Recommended defaults adopted
- Requirements or documents changed: 04 OQ-FR-5/6; 06 OQ-DAT-05; 08 OQ-UX-03/05/07 (moved here)
- Verification impact: TestJoinModes TestListingsDirectoryExports
- Review or revisit: pilot feedback
- Counsel review required: no
- Resolves: OQ-UX-03, OQ-UX-05, OQ-UX-07, OQ-FR-5, OQ-FR-6, OQ-DAT-05 (the question text is removed from the open list in its spec)
- Code and tests: `internal/service/membership_test.go` (TestJoinModes); `internal/web/admin_test.go` (TestListingsDirectoryExports)

### DEC-055 · Dark colour scheme follows the operating system

- Status: accepted (as built); contrast not measured
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Interface lead
- Context and requirements: OQ-UX-01 (dark scheme in MVP).
- Decision: A dark token set is applied under prefers-color-scheme: dark. The contrast check required by UX-057 has not been run for it.
- Rationale: Low cost; UX-057 still applies to both schemes.
- Recommended default and whether adopted: Include (adopted); verify contrast
- Requirements or documents changed: 08 OQ-UX-01 (moved here)
- Verification impact: none run for contrast
- Review or revisit: before release gate
- Counsel review required: no
- Resolves: OQ-UX-01 (the question text is removed from the open list in its spec)
- Code and tests: `internal/web/static/style.css` (style.css)

### DEC-056 · Default transaction cap of 1,000,000.00 credits

- Status: accepted (as built), deviation from 03/00 wording
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner (confirm)
- Context and requirements: DOM-029 and 00 section 5 (max transaction amount: set by administrator, no network-wide default).
- Decision: `MaxTxAmount` defaults to 100,000,000 minor units (1,000,000.00 at 2 decimals) so a new network has a cap.
- Rationale: A cap is needed for validation; the spec gave no value.
- Recommended default and whether adopted: Owner to set the value
- Requirements or documents changed: 00 section 5; 03 DOM-029
- Verification impact: DefaultPolicy TestPolicyBounds
- Review or revisit: first pilot
- Counsel review required: no
- Code and tests: `internal/domain/policy.go` (DefaultPolicy); `internal/domain/statemachine_test.go` (TestPolicyBounds)

### DEC-057 · Configuration sources, precedence and secret files

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: OPS-006, OPS-009, DEC-017; 11 section 12 item 4.
- Decision: Precedence: flags, then CIRCULA_ environment variables, then the configuration file, then defaults. Secrets (setup, metrics token, backup key) are accepted only from the environment or a `*-file` path, never inline. `settingDefs` is the single list of settings. Migration mode is a setting (`migrate-mode`), set by file or flag.
- Rationale: Predictable overrides in containers and service managers.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 11 OPS-006, OPS-009; ops configuration.md
- Verification impact: TestPrecedence TestEverySettingAcceptedByEverySource TestSecretsLoadingAndRedaction
- Review or revisit: never
- Counsel review required: no
- Resolves: 11 section 12 item 4 (DEC-017) (the question text is removed from the open list in its spec)
- Code and tests: `cmd/circula/commands.go` (resolveFlags); `cmd/circula/config.go` (readSecretFile); `cmd/circula/config_test.go` (TestPrecedence); `cmd/circula/hardening_test.go` (TestSecretsLoadingAndRedaction)

### DEC-058 · Metrics exposition format and access

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: OPS-013, DEC-018; 11 section 12 item 3.
- Decision: Text exposition format at /metrics. Default access is loopback (`metrics-allow`); an optional bearer token (`metrics-token-file`) is also accepted. Labels use route templates, never raw paths or member ids.
- Rationale: Common format; no member data in labels (T-11).
- Recommended default and whether adopted: Loopback allow-list (recommended adopted)
- Requirements or documents changed: 11 OPS-013; ops monitoring.md
- Verification impact: TestMetricsGate TestMetricsExposeRequiredSeriesWithRouteTemplates TestRouteLabelsAreTemplates
- Review or revisit: if a scrape network is needed
- Counsel review required: no
- Resolves: 11 section 12 item 3 (DEC-018) (the question text is removed from the open list in its spec)
- Code and tests: `cmd/circula/ops_http.go` (metricsAccess); `cmd/circula/hardening_test.go` (TestMetricsGate); `cmd/circula/ops_test.go` (TestRouteLabelsAreTemplates)

### DEC-059 · Backup schedule and retention

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: OPS-017, OPS-018; 11 section 12 item 7.
- Decision: Scheduled backup every 24 h (fixed interval, not cron). Keep 30 daily and the newest backup of each of 12 months. Same default for self-hosted and hosted. Snapshots use an online copy (VACUUM INTO) with a manifest.
- Rationale: Recommended defaults in 11 OPS-018.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 11 OPS-017, OPS-018; ops backup-restore.md
- Verification impact: TestPruneKeepsDailyAndNewestPerMonth TestJobScheduledBackupAndRetention TestSnapshotWritesVerifiedManifestAndRestores
- Review or revisit: hosted RPO decision (DEC-016)
- Counsel review required: no
- Resolves: 11 section 12 item 7 (the question text is removed from the open list in its spec)
- Code and tests: `cmd/circula/snapshot.go` (pruneBackups); `cmd/circula/snapshot.go` (snapshot); `cmd/circula/backup_test.go` (TestPruneKeepsDailyAndNewestPerMonth)

### DEC-060 · Migrations: forward-only, explicit by default, pre-migration backup, post-migration verification

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: OPS-022 to OPS-025, DAT-042 to DAT-044.
- Decision: Migrations are embedded, versioned and forward-only. `migrate-mode` defaults to explicit (serve refuses pending migrations). Auto mode takes a backup first. A pre-migration backup is taken for non-empty databases and the post-migration verification gates serving. No automated downgrade.
- Rationale: Downgrades on a ledger are unsafe; rollback is by restore.
- Recommended default and whether adopted: explicit
- Requirements or documents changed: 06 DAT-042..044; 11 OPS-022..025
- Verification impact: TestServeRefusesPendingMigrationsInExplicitMode TestMigrationsIdempotentAndVersioned TestMigration0002BackfillsAuditHeads
- Review or revisit: at each schema change
- Counsel review required: no
- Code and tests: `cmd/circula/serve.go` (preMigrationBackup); `cmd/circula/schema.go` (checkSchemaNotNewer); `cmd/circula/serve_test.go` (TestServeRefusesPendingMigrationsInExplicitMode)

### DEC-061 · Export scopes and file formats

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: FR-090 to FR-094, DAT-037 to DAT-040, SEC-017.
- Decision: Scopes: mine (member), ledger, members, audit, full (admin; full also for closed networks). Formats CSV and JSON; a manifest lists files, row counts and SHA-256, and the ledger_seq of the snapshot. CSV cells starting with spreadsheet formula characters are neutralised. Downloads expire after 24 h.
- Rationale: Open formats; checksum for the recipient.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 04 FR-090..094; 06 section 8
- Verification impact: TestMemberExportIsScoped TestCSVFormulaNeutralisation TestFullExportJSON TestLedgerExportCSV
- Review or revisit: when an import format is specified (deferred)
- Counsel review required: no
- Code and tests: `internal/service/export.go` (ExportMembers); `internal/service/export.go` (ExportFull); `internal/service/export.go` (ManifestFile); `internal/service/export_test.go` (TestCSVFormulaNeutralisation)

### DEC-062 · Identifiers are UUIDv7 from an injected generator

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Ledger and data lead
- Context and requirements: OQ-DAT-01 (UUIDv7 or ULID).
- Decision: All entity ids are UUIDv7 text, generated in the service through the injected `IDGen`. Tests use deterministic ids.
- Rationale: Time-ordered index locality; deterministic tests (ARC-021).
- Recommended default and whether adopted: UUIDv7 (recommended adopted)
- Requirements or documents changed: 06 OQ-DAT-01 (moved here)
- Verification impact: IDGen TestSingleReadsByID
- Review or revisit: never
- Counsel review required: no
- Resolves: OQ-DAT-01 (the question text is removed from the open list in its spec)
- Code and tests: `internal/domain/types.go` (IDGen); `internal/service/service.go` (uuid.NewV7)

### DEC-063 · Disputes cover completed trades only

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: FR-070, DOM-017; reversals and adjustments are not disputable.
- Decision: Only completed trades can be disputed. A reversal cannot be reversed (DOM-019).
- Rationale: Keeps dispute scope to the trade model.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 03 DOM-017, DOM-019
- Verification impact: TestOpenDisputeRules
- Review or revisit: if grants or adjustments need recourse
- Counsel review required: no
- Code and tests: `internal/service/disputes.go` (OpenDispute); `internal/service/dispute_test.go` (TestOpenDisputeRules)

### DEC-064 · Session lifetimes: one pair of values for all roles

- Status: accepted (as built), partly
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner
- Context and requirements: DEC-005, SEC-003, OPS-008.
- Decision: Idle timeout 720 h (30 days) and absolute timeout 2160 h (90 days) for every role. Separate administrator values (60 min idle, 12 h absolute in DEC-005) are not built.
- Rationale: Simplest build; DEC-005 default for members.
- Recommended default and whether adopted: Members' values for all roles
- Requirements or documents changed: 09 SEC-003; 13 DEC-005
- Verification impact: TestSessionOrderingAndPasswordFloor TestLoginSessionLifecycle
- Review or revisit: DEC-005 decision
- Counsel review required: no
- Code and tests: `internal/service/auth.go` (Authenticate); `cmd/circula/config_test.go` (TestSessionOrderingAndPasswordFloor)

### DEC-065 · Argon2id password hashing: interim parameters

- Status: accepted (as built), benchmark pending
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner
- Context and requirements: SEC-001, DEC-006 (parameters chosen by benchmark).
- Decision: Argon2id with 64 MiB memory, time 3, parallelism 2, 16-byte salt, stored as PHC strings so parameters can change.
- Rationale: Above the OWASP minimum; benchmark on minimum hardware not yet recorded.
- Recommended default and whether adopted: Benchmark pending (DEC-006)
- Requirements or documents changed: 09 SEC-001; 13 DEC-006
- Verification impact: TestPasswordStorage TestPasswordPolicy
- Review or revisit: DEC-006
- Counsel review required: no
- Code and tests: `internal/service/password.go` (DefaultArgon2Params); `internal/service/auth_test.go` (TestPasswordStorage)

### DEC-066 · Two idempotency layers

- Status: accepted (as built), known duplication
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: LED-4, DAT-025..028, API-020..025, MAINT-01 (audit).
- Decision: The service keeps durable idempotency records per network, person and operation, kept 7 days (`IdempotencyTTL`). The HTTP adapter keeps an in-memory replay cache for 24 h (10,000 entries) keyed by person and key. Replay across restarts is only guaranteed by the service layer.
- Rationale: Service records give the ledger guarantee; the API cache returns identical responses.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 07 section 5; docs/dev/SERVICE.md section 2
- Verification impact: TestIdempotentRetryReturnsOriginal TestPanicReleasesIdempotencyKey
- Review or revisit: when refactoring the adapter
- Counsel review required: no
- Code and tests: `internal/service/idempotency.go` (idempotent); `internal/api/idem.go` (idem.go); `internal/api/internal_test.go` (TestPanicReleasesIdempotencyKey)

### DEC-067 · Browser security headers and no third-party assets

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Security owner
- Context and requirements: SEC-023, DEC-002.
- Decision: Every HTML response sends a Content-Security-Policy that allows same-origin scripts only and forbids framing; no third-party scripts or fonts are loaded. API responses send a stricter CSP.
- Rationale: Progressive enhancement with a strict policy.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 09 SEC-023
- Verification impact: TestHeadersAndCookies TestSecureAndHSTS
- Review or revisit: if a CDN is ever wanted (SEC-029)
- Counsel review required: no
- Code and tests: `internal/web/security_test.go` (TestHeadersAndCookies); `internal/httpx/httpx_test.go` (TestSecureAndHSTS)

### DEC-068 · Structured JSON logs and route-template metric labels

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: OPS-032, ARC-022, SEC-020.
- Decision: Logs are JSON by default (`log-format`), levels set by `log-level` (restart needed). Request ids on every response. Metric labels use route templates and status classes only.
- Rationale: No paths with tokens and no member ids in logs or metrics.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 11 OPS-032; ops monitoring.md
- Verification impact: TestRouteLabelsAreTemplates TestRequestIDAndRoutesMounted
- Review or revisit: when log-level reload is wanted
- Counsel review required: no
- Code and tests: `cmd/circula/serve.go` (newLogger); `cmd/circula/ops_test.go` (TestRouteLabelsAreTemplates)

### DEC-069 · Single application process

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: ARC-014, OPS-002, DEC-030.
- Decision: One `serve` process per database file, enforced by the lock (DEC-030). Horizontal scaling is not supported.
- Rationale: SQLite single writer; scaling is out of MVP scope.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 05 ARC-014; 11 OPS-002
- Verification impact: TestLockRefusesSecondHolder
- Review or revisit: with an alternative store (OPS-005)
- Counsel review required: no
- Code and tests: `cmd/circula/backup_test.go` (TestLockRefusesSecondHolder)

### DEC-070 · Server clock and injected time source for every time rule

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: DOM-013, DOM-017, OPS-027, ARC-021.
- Decision: Expiry, dispute windows, sessions and backup cadence use the service clock (UTC), injected through `Config.Clock`. Host time zone does not affect job timing.
- Rationale: Testable with a fake clock; restart-safe.
- Recommended default and whether adopted: As built
- Requirements or documents changed: 05 ARC-021; 11 OPS-027
- Verification impact: TestJobExpiryMovesOnlyElapsedProposals TestJobEveryRepeatsUntilCancelled
- Review or revisit: never
- Counsel review required: no
- Code and tests: `internal/domain/types.go` (Clock); `cmd/circula/jobs_regress_test.go` (TestJobEveryRepeatsUntilCancelled)

### DEC-071 · Build and test tooling in CI

- Status: accepted (as built)
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Engineering lead
- Context and requirements: 12 section 11 item 2 (choice of testing and checking tools).
- Decision: Go standard testing (no property-test library: randomised invariant tests are written in-house), `go vet`, golangci-lint v2 (CI pinned), govulncheck v1.1.4, race-enabled test run, action-pin check script, and a five-target package build.
- Rationale: Small dependency set (go.mod lists three direct modules).
- Recommended default and whether adopted: As built
- Requirements or documents changed: 12 section 11 item 2
- Verification impact: ci.yml
- Review or revisit: when browser or accessibility tooling is chosen (QA-033, QA-036)
- Counsel review required: no
- Resolves: 12 section 11 item 2 (the question text is removed from the open list in its spec)
- Code and tests: `.github/workflows/ci.yml` (ci.yml); `scripts/check-action-pins.sh` (check-action-pins.sh)

## 2. Open owner decisions raised by the build (DEC-072 to DEC-079)

### DEC-072 · Who may decide a dispute: reviewer role only, or an administrator too?

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner
- Context and requirements: FR-065 and API-018 (reviewer operations need the reviewer role; reviewer does not grant admin). DOM-038 and the service comment allow 'reviewer or administrator'. The build allows an administrator without the reviewer role to resolve (LIGHT-QA BUG-3).
- Options: (a) reviewer role required; administrators who are not parties may also hold the role. (b) administrators may decide (as built); align FR-065/API-018 wording to DOM-038.
- Decision: Not decided.
- Rationale: Separation of duties versus small networks with one administrator (ASM-010).
- Recommended default and whether adopted: (a): require the reviewer role, and allow an administrator to hold it; a party or conflicted person never decides.
- Requirements or documents changed: 02 PER-003; 03 DOM-038; 04 FR-065; 07 API-018
- Verification impact: TestDisputeDecisionAuthorization (current behaviour)
- Review or revisit: before pilot
- Counsel review required: no
- Code and tests: `internal/api/h_disputes.go` (resolveDispute); `internal/service/dispute_test.go` (TestDisputeDecisionAuthorization)

### DEC-073 · Administrator read access to transactions they are not party to

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner
- Context and requirements: API-049 says a non-party gets not-found. The build returns 200 to an administrator for any transaction in the network, with audit for the read (LIGHT-QA OBS-4).
- Options: (a) admin oversight of all transactions, audited, and API-049 reworded. (b) restrict to parties and reviewers of a dispute.
- Decision: Not decided.
- Rationale: Oversight versus data minimisation (VIS-010, SEC-013).
- Recommended default and whether adopted: (a) with audit, if the owner accepts admin oversight of a network it runs
- Requirements or documents changed: 07 API-049; 02 PER-002
- Verification impact: GetTransaction (current behaviour)
- Review or revisit: before pilot
- Counsel review required: no
- Code and tests: `internal/service/tx_queries.go` (GetTransaction); `internal/api/h_tx.go` (getTx)

### DEC-074 · Should an accepted-but-not-posted state ever be visible?

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect
- Context and requirements: OQ-DOM-1 and OQ-ARC-04 were resolved as built (DEC-036): accepted is never persisted. Owner to confirm this holds, or define a two-phase flow.
- Options: (a) keep internal (as built). (b) add a counterparty confirmation step; revise 03, 06, 08 and the audit trail.
- Decision: Not decided; DEC-036 stands until changed.
- Rationale: A two-phase flow would change DOM-014 and the audit model.
- Recommended default and whether adopted: (a) keep accepted internal, as built, until a two-phase flow is decided
- Requirements or documents changed: 03 DOM-010, OQ-DOM-1; 05 OQ-ARC-04; 08 UX-016
- Verification impact: TestTxStateMachineExhaustive
- Review or revisit: before pilot
- Counsel review required: no
- Code and tests: `internal/domain/statemachine.go` (NextTxStatus)

### DEC-075 · Durability setting for the embedded database

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Operations lead
- Context and requirements: OPS-011, OPS-033 (power loss may lose recent commits with synchronous=NORMAL; runbook text said to reconcile from the audit trail, which is in the same commit).
- Options: (a) keep synchronous=NORMAL and correct the runbook text. (b) add a setting (`durability`) with FULL for ledger-grade durability at a write-speed cost.
- Decision: Not decided.
- Rationale: Trade-off between durability and throughput (ASM-001, ASM-009).
- Recommended default and whether adopted: (b) offered as a setting; default NORMAL until measured
- Requirements or documents changed: 11 OPS-011, OPS-033; ops monitoring.md
- Verification impact: none run (power-loss drill not performed)
- Review or revisit: before hosted launch
- Counsel review required: no
- Code and tests: `cmd/circula/serve.go` (checkPragmas); `internal/store/sqlite.go` (DefaultBusyTimeout)

### DEC-076 · Scope of a ledger verification failure at start-up

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product architect with operations lead
- Context and requirements: OPS-033 and OPS-016 say only the affected network freezes. The build refuses to serve every network if any fails at start-up (CMD-07); no per-network posting gate exists.
- Options: (a) whole instance refuses (as built). (b) per-network posting freeze with a service gate and an audit event.
- Decision: Not decided.
- Rationale: Isolation of tenants versus simpler failure handling.
- Recommended default and whether adopted: (b), with (a) until the gate is built
- Requirements or documents changed: 11 OPS-033; 05 ARC-007
- Verification impact: TestServeStartsAndStopsCleanly (current behaviour)
- Review or revisit: before hosted launch
- Counsel review required: no
- Code and tests: `cmd/circula/serve.go` (runServe); `cmd/circula/serve.go` (checkPragmas)

### DEC-077 · Is the reviewer's identity shown to the parties?

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner
- Context and requirements: OQ-FR-4. The API returns reviewer_id on a resolved dispute; the web dispute view does not name the reviewer.
- Options: (a) show to administrators and both parties. (b) show to administrators only.
- Decision: Not decided.
- Rationale: Accountability (SEC-010) versus reviewer safety.
- Recommended default and whether adopted: (a), as in 04 FR-073 wording
- Requirements or documents changed: 04 OQ-FR-4; 09 section 9
- Verification impact: GetDispute (current behaviour)
- Review or revisit: before pilot
- Counsel review required: no
- Code and tests: `internal/service/disputes_decide.go` (GetDispute); `internal/service/disputes_decide.go` (disputeView)

### DEC-078 · What actor identity may a member see on audit events about their trades?

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: Product owner with security owner
- Context and requirements: OQ-DAT-02. The service reduces member views to a reduced form; no member UI or API exposes audit yet (FR-082 partial).
- Options: (a) role or pseudonymous label only. (b) counterparty display name.
- Decision: Not decided.
- Rationale: Minimisation (VIS-010) versus dispute evidence.
- Recommended default and whether adopted: (a)
- Requirements or documents changed: 06 OQ-DAT-02; 09 SEC-019
- Verification impact: TestAuditVisibilityForMembers
- Review or revisit: when a member audit view is built
- Counsel review required: no
- Code and tests: `internal/service/audit_read.go` (ListAudit); `internal/service/audit_test.go` (TestAuditVisibilityForMembers)

### DEC-079 · Request body size: API default 64 KiB or the 1 MiB server setting

- Status: open
- Dates: opened 2026-10-21; as-built record written 2026-10-21 (this pass)
- Owner who decides: API lead
- Context and requirements: API-037 says 64 KiB. The API adapter enforces 64 KiB (`MaxBodyBytes` default). The server setting `max-body-bytes` defaults to 1 MiB (OPS-008) and applies to the edge.
- Options: (a) 64 KiB for API and 1 MiB for web forms. (b) one value for both.
- Decision: Not decided.
- Rationale: Web forms with long text and API limits differ in purpose.
- Recommended default and whether adopted: (a), with the two values documented
- Requirements or documents changed: 07 API-037; 11 OPS-008
- Verification impact: TestBodyLimitApplied TestBodySizeLimit
- Review or revisit: with 07 next revision
- Counsel review required: no
- Code and tests: `internal/api/strict_test.go` (TestBodySizeLimit); `cmd/circula/config.go` (MaxBodyBytes)

## 3. Open owner decisions carried from 13 (not renumbered)

These are decided by the owner in [13](specs/13-assumptions-risks-decisions.md). The table gives the status after the build and the recommended default in 13. A default is not a decision.

| ID | Question (13) | Status after build | Recommended default (13) |
|---|---|---|---|
| DEC-004 | Product licence (undecided; repository ships without LICENSE) | open, not built: Licence undecided; no LICENSE file in repo (see 13 register) | AGPL-3.0 or MIT (recommendation in 00 section 6) |
| DEC-005 | Session idle and absolute timeouts | open, partly built: Single idle (720 h) and absolute (2160 h) timeouts for all roles; role split not built | Members: idle 30 days (matches the default in [06](specs/06-data-ledger-audit.md) section 9), absolute 90 days. Administrators and reviewers: idle 60 minutes, absolute 12 hours |
| DEC-006 | Password minimum length, maximum accepted length, breached-password check source, Argon2id or bcrypt parameters | open, partly built: Min length 12 and Argon2id built; benchmark and breached-password list not | Minimum 12, accept at least 64; offline breached-password check enabled; Argon2id, parameters chosen by benchmark on minimum hardware |
| DEC-007 | Mandatory second factor for privileged roles | open, not built: No second factor built | Mandatory for network administrators and platform operators; optional for members and reviewers |
| DEC-008 | Hosted storage isolation: one database file per network, or one shared file with tenant scoping | open, partly built: One shared database with network-scoped rows; per-network files not built | One database file per network in hosted mode. Revisit if operations cost is too high |
| DEC-009 | Recovery channel and mail dependency | open, not built: No recovery channel and no mail | Verified email for recovery; admin-assisted recovery with dual control (SEC-011) when no verified contact exists; notices are in-app and by mail when configured |
| DEC-010 | Departure settlement: residual positive and negative balances | open, partly built: Retain default and settle-to-zero rule built; counsel review open | Freeze account; keep residual balances recorded; residual negative balance may be written off only by approved adjustment (GOV-018); no redemption in any form |
| DEC-011 | Sign-up grants in MVP | open, partly built: Grant off by default; approval or invite requirement and caps not built | Off by default (grant zero); when enabled, approval or invite join mode required |
| DEC-012 | Dispute and appeal timelines | open, not built: Dispute response period and timelines not built | Dispute window 30 days after completion; response 7 days; decision target 14 days after response deadline; appeal within 14 days; no default outcome on silence |
| DEC-013 | Dual control for adjustments and write-offs | open, not built: Dual control not built | Dual control whenever the network has two or more administrators; single administrator otherwise, with the restriction shown to members |
| DEC-014 | Platform operator break-glass controls and notice period | open, not built: Break-glass not built | Two-person approval and audit trail readable by affected administrators; access expires after 4 hours (the proposal in [05](specs/05-architecture.md) ARC-009); notice to affected administrators within 24 hours where legally permitted (recommended, counsel review) |
| DEC-015 | Tamper-evident audit chain | open, partly built: Unkeyed hash chain built; keyed or external publication deferred | Defer to post-MVP (SEC-034); keep stable per-network sequence in the data model |
| DEC-016 | Backup recovery point and recovery time objectives, off-host copy policy | open, not built: RPO and RTO not recorded | Hosted: recovery point at most 1 hour, recovery time at most 4 hours, one off-host copy. Self-hosted: daily backups, off-host copy recommended |
| DEC-019 | Log retention period | open, not built: Log retention not implemented | 90 days, then pruned |
| DEC-020 | Retention of personal data and of closed-network records | open, not built: Retention periods not set | Retention periods to be set after counsel review; backups follow the stated retention (SEC-028) |
| DEC-021 | Governance decision method for policy changes | open, partly built: Unilateral administrator change built; objection right not built | Unilateral administrator change with notice and objection rights; member voting deferred |
| DEC-022 | Support objectives and any external commitment | open, not built: Support objectives not set | Documented targets, not presented as service levels until the platform operator approves them |
| DEC-023 | Numerical thresholds: property-test run sizes, interleaving counts, coverage, performance, timing-equality threshold | open, not built: Numerical thresholds not confirmed | Values in QA-001, QA-007, QA-008, QA-040 as recommended there; each confirmed on reference hardware before release |
| DEC-024 | Accessibility conformance level and supported browser and screen reader combinations | open, not built: Accessibility level and browser list not recorded | WCAG 2.2 level AA; current evergreen browsers; one screen reader per major platform, recorded |
| DEC-025 | Notice periods for adverse policy changes and for closure | open, not built: Notice periods not implemented | 14 days for adverse policy changes; 30 days for network closure |
| DEC-026 | Vulnerability disclosure contact | open, partly built: Private vulnerability reporting named in SECURITY.md; owner approval not recorded | A role-based contact approved by the owner; no personal address published without approval |

## 4. Open questions resolved into decisions

Each OQ below was removed from the open list in its spec and recorded here. Questions that remain open are listed after the table.

| Question | Original text (verbatim, from the spec before this pass) | Resolved by | As-built answer |
|---|---|---|---|
| OQ-DOM-1 | **OQ-DOM-1.** Whether `accepted` is written to the audit trail as its own event, or only `completed` is recorded. Recommended default: record the single atomic event `proposed → completed` only. | DEC-036 | `accepted` is never persisted; audit records proposed to completed |
| OQ-ARC-04 | **OQ-ARC-04 (assumption):** Whether `accepted` is ever externally visible. This spec assumes acceptance and posting occur in one atomic unit in the MVP, so the `accepted` status is internal. If a two-phase flow is required (for example, a counterparty confirmation after acceptance), 03 and 06 must be revised. | DEC-036 | `accepted` is never persisted; audit records proposed to completed |
| OQ-DOM-7 | **OQ-DOM-7.** Whether proposal expiry is recalculated on policy change (DOM-030). Recommended default: no recalculation. | DEC-037 | Proposal expiry fixed when the proposal is created |
| OQ-DOM-2 | **OQ-DOM-2.** Resolution path for upheld reversals blocked by LED-5 (DOM-021). Options: time-limited audited limit override; hold until balance recovers; both. Recommended default: both, with the override requiring a reason and an expiry. | DEC-039 | Upheld reversal blocked by the limit: posting_pending, audited override, retry |
| OQ-DAT-04 | **OQ-DAT-04 (decision needed):** Tamper-evident audit chain (DAT-030) is recommended as P1; confirm whether it is required for the MVP. | DEC-044 | Audit hash chain (unkeyed) with a per-scope head |
| OQ-DOM-4 | **OQ-DOM-4.** Default departure rule. Recommended default: `retain`, because settle-to-zero is an accounting event whose meaning depends on the network's governance. | DEC-049 | Departure default: retain balance; settle-to-zero by adjustment |
| OQ-DOM-5 | **OQ-DOM-5.** Platform maximum for proposal expiry and dispute window. | DEC-050 | Platform bounds on policy values |
| OQ-DOM-6 | **OQ-DOM-6.** Platform maximum for decimals. Recommended: at most 4, to keep minor-unit magnitudes well within 64-bit range; to be confirmed against the storage design. | DEC-050 | Platform bounds on policy values |
| OQ-DAT-06 | **OQ-DAT-06 (decision needed):** Maximum length of adjustment reasons and listing text. To be set in 03 and 04. | DEC-050 | Platform bounds on policy values |
| OQ-DOM-3 | **OQ-DOM-3.** Whether sign-up grants are part of the MVP or post-MVP, and their default amount. Recommended default: disabled. | DEC-051 | Sign-up grant is off by default |
| OQ-PER-1 | **OQ-PER-1 (product owner).** Whether members may be invited by other members (not only administrators) in invite mode. Current spec: administrators only; FR-010 is written accordingly. | DEC-053 | Administrators only invite and approve; admins see balances as limit and balance only; directory shows active members |
| OQ-PER-3 | **OQ-PER-3 (product owner).** Whether an administrator who is also a member may see other members' balances. Current spec: yes, limit and balance only, audited (matrix). | DEC-053 | Administrators only invite and approve; admins see balances as limit and balance only; directory shows active members |
| OQ-PER-4 | **OQ-PER-4 (UX).** Whether a suspended member can still view the directory. Current spec: no; they keep personal history and export only. | DEC-053 | Administrators only invite and approve; admins see balances as limit and balance only; directory shows active members |
| OQ-FR-1 | **OQ-FR-1 (product owner).** Whether non-administrator members may send invitations in invite mode (see also OQ-PER-1). Current spec: administrators only. | DEC-053 | Administrators only invite and approve; admins see balances as limit and balance only; directory shows active members |
| OQ-UX-03 | **OQ-UX-03 (decision needed):** MVP languages beyond English (UX-046, UX-047). | DEC-054 | MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices |
| OQ-UX-05 | **OQ-UX-05 (assumption):** Trades always have exactly two parties and a single amount; no multi-party or split-payment screens exist in the MVP. | DEC-054 | MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices |
| OQ-UX-07 | **OQ-UX-07 (assumption):** Trade acceptance and proposal screens do not need a per-member "remember my choice" preference in the MVP. | DEC-054 | MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices |
| OQ-FR-5 | **OQ-FR-5 (product owner).** Whether listing prices are required or optional for the MVP. Current spec: optional (FR-021). | DEC-054 | MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices |
| OQ-FR-6 | **OQ-FR-6 (product owner).** Whether FR-012 (open join) is in the MVP or only invite and approval. Current spec: P1, so it may slip without blocking the MVP gate if the product owner records a deferral. | DEC-054 | MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices |
| OQ-DAT-05 | **OQ-DAT-05 (assumption):** Entries and transactions are never partially settled; a trade is either fully posted or not posted (no multi-leg or partial payments in MVP). | DEC-054 | MVP scope as built: English only, optional prices, open join present, no multi-party trades, no remembered choices |
| OQ-UX-01 | **OQ-UX-01 (decision needed):** Whether the MVP includes a dark colour scheme. If it does, UX-057 applies to both schemes. | DEC-055 | Dark colour scheme follows the operating system |
| OQ-DAT-01 | **OQ-DAT-01:** UUIDv7 (recommended) or ULID for identifiers. Either satisfies DAT-001; owner preference needed. | DEC-062 | Identifiers are UUIDv7 from an injected generator |

Still open (not resolved by the build): OQ-API-01, OQ-API-02, OQ-API-03, OQ-API-04, OQ-API-05, OQ-API-06, OQ-ARC-01, OQ-ARC-02, OQ-ARC-03, OQ-ARC-05, OQ-DAT-02, OQ-DAT-03, OQ-DAT-04, OQ-DOM-8, OQ-FR-2, OQ-FR-3, OQ-FR-4, OQ-PER-2, OQ-UX-02, OQ-UX-04, OQ-UX-06, OQ-VIS-1, OQ-VIS-2, OQ-VIS-3, OQ-VIS-4, OQ-VIS-5.
