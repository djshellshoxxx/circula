# Completeness audit: P0/P1 vs current code

Branch claude/circula-completeness, head df8ae78. Read-only audit; statuses re-derived from code and tests, not from spec status lines, docs/TRACEABILITY.md or docs/audit/CODE-AUDIT.md (all stale: several rows below are built now that the specs call not built, e.g. FR-024, FR-038, FR-076, FR-078, UX-012, UX-024, UX-053, SEC-007, SEC-008, SEC-011, SEC-013, SEC-018, SEC-028, DAT-047, OPS-023, OPS-024).

Method: grep/read of routes, services, store, templates, workflows, cmd; tests cited by name where read. Tests were not executed in this pass. UI-quality rows (UX-033..040 etc.) keep `partial` where no automated or recorded evidence exists. Scope: P0 and P1 only; API-001..039 are the cross-cutting rows from the 07 body, API-040+ from its operation table.

Path prefixes: `S/`=internal/service, `A/`=internal/api, `W/`=internal/web, `ST/`=internal/store, `C/`=cmd/circula. Status: built / partial / not built / deviation (built differently from spec, recorded as D-xx or DEC).


## 04 Functional requirements

| ID | Pri | Status | Evidence |
|---|---|---|---|
| FR-001 | P0 | built | S/networks.go CreateNetwork; S/admin_test.go |
| FR-002 | P0 | built | S/setup_state.go (state in platform audit, survives restart; reset via `setup-reset`); S/setup_state_test.go, W/setup_state_test.go |
| FR-003 | P0 | built | S/networks.go UpdatePolicy; domain TestPolicyBounds; S/admin_test.go TestPolicyChanges |
| FR-004 | P0 | built | S/networks.go; S/admin_test.go |
| FR-005 | P0 | built | S/networks.go decimals lock; domain/policy.go |
| FR-006 | P0 | built | S/authz_test.go TestCrossNetworkIdsAreNotFound; A/authn_test.go TestAuthorisation |
| FR-010 | P0 | built | S/join.go RedeemInvitation; A/routes.go:70-71; W/h_auth.go:23-24 |
| FR-011 | P0 | built | S/join.go DecideJoinRequest; A/routes.go:62; A/hardening_test.go TestAPIDenyJoinRequest |
| FR-012 | P1 | partial | open join built (S/join.go JoinNetwork, W/h_auth.go:25); no join-specific rate limit (W/web.go limiters: ip/acct/pw/reg/setup only) |
| FR-013 | P0 | built | domain/statemachine_test.go |
| FR-014 | P0 | built | S/departure.go; A/routes.go:27; W/h_gov.go leaveForm/leavePost (fixed in claude/cc-h1) |
| FR-015 | P0 | built | S/membership.go; W/h_admin.go memberActions[remove]; S/membership_test.go |
| FR-016 | P1 | built | S/join.go; display name only required |
| FR-017 | P0 | built | S/membership_test.go TestJoinModes; S/regress_rejoin_test.go |
| FR-020 | P0 | built | S/member_page.go; ST/member_page.go; S/member_page_test.go |
| FR-021 | P0 | built | S/listings.go; A/routes.go:31-32; W/h_home.go:24-25 |
| FR-022 | P0 | built | S/listings.go UpdateListing; W/h_home.go:26-27; A PATCH /listings/{id} |
| FR-023 | P0 | built | S/listings.go ListListings (kind, q, paged) |
| FR-024 | P1 | built | S/reports.go; A/h_reports.go routes.go:83-85; W/h_notify.go:14-17; S/reports_test.go |
| FR-025 | P1 | partial | session required for members pages (W/middleware.go); no robots/noindex directive anywhere (grep noindex/robots empty) |
| FR-030 | P0 | built | S/transactions.go ProposeTransaction; S/trades_test.go |
| FR-031 | P0 | built | W/templates/propose_review.html; W/flow_test.go |
| FR-032 | P0 | built | S/posting.go postTx/checkCapacity; S/atomicity_test.go; TestAcceptRechecksLimit |
| FR-033 | P0 | built | S/transactions.go; S/trades_test.go |
| FR-034 | P0 | built | S/purge.go / expiry job; injected clock; S/trades_test.go |
| FR-035 | P0 | built | S/idempotency.go; S/idempotency_test.go TestIdempotentRetryReturnsOriginal |
| FR-036 | P0 | built | S/concurrency_test.go TestParallelAcceptsNeverBreachCreditLimit, TestParallelDoubleAcceptSingleSuccess |
| FR-037 | P1 | built | S/transactions.go note + listing ref |
| FR-038 | P1 | built | S/notify.go (13 kinds), S/notify_limits.go, W/h_notify.go inbox, A/h_notify.go; S/notify_test.go |
| FR-050 | P0 | built | S/accounts.go; W/templates/dashboard.html; A GET /accounts/me |
| FR-051 | P0 | built | S/reads.go ListEntries; A TestEntriesOrderedByLedgerSeq |
| FR-052 | P0 | built | S/verify.go; W/h_admin.go:27; C/commands.go verify-ledger |
| FR-053 | P1 | built | W/hardening_test.go TestDashboardShowsOldProposalBehindManyTrades |
| FR-054 | P1 | partial | S/accounts.go:96 BalanceAt; no UI/API caller |
| FR-060 | P0 | built | S/membership.go SetCreditLimit; A routes.go:69 |
| FR-061 | P0 | built | S/membership.go Suspend/Reinstate; A routes.go:63-64 |
| FR-062 | P0 | built | S/adjustments.go; A routes.go:56; W/h_admin.go:24; dual-control above threshold S/dualcontrol.go |
| FR-063 | P0 | built | W/h_admin.go:19-21; S/join.go; A routes.go:61-62,70 |
| FR-064 | P1 | partial | S/membership.go:185 SetDefaultFlag; W/h_admin.go:214 only; no API route |
| FR-065 | P1 | built | S/membership.go role assign; A routes.go:66-67; S/dispute_test.go |
| FR-066 | P1 | built | A/routes.go:55 adminReverse; S/reversal.go; S/reversal_test.go; W/h_admin_f2.go reverse UI |
| FR-067 | P1 | built | S/overview.go AdminOverview; W/templates/admin_home.html; W/f2_test.go |
| FR-068 | P1 | partial | S/networks.go:127 CloseNetwork; no web/API/CLI caller (grep) |
| FR-069 | P0 | built | S/membership.go AssignRole; S/membership_test.go TestAdministratorSuccession |
| FR-070 | P0 | built | S/disputes.go OpenDispute; S/dispute_test.go TestOpenDisputeRules |
| FR-071 | P0 | built | S/dispute_test.go |
| FR-072 | P0 | built | S/disputes_decide.go; TestDisputeDecisionAuthorization |
| FR-073 | P0 | built | S/disputes_decide.go; A routes.go:49 |
| FR-074 | P0 | built | S/disputes_decide.go posting_pending; A routes.go:52 retry-posting |
| FR-075 | P1 | built | A routes.go:51 withdraw; W/h_disputes.go |
| FR-076 | P1 | built | S/disputes_response.go; A routes.go:80; S/disputes_response_test.go |
| FR-077 | P0 | built | S/disputes.go window from completion, server clock |
| FR-078 | P1 | built | A routes.go:81-82 re-review + decision; S/disputes_response.go; W/templates/dispute_view.html |
| FR-080 | P0 | built | S/atomicity_test.go TestEveryCommandWritesAuditEvent |
| FR-081 | P0 | built | S/audit.go; domain/audit.go |
| FR-082 | P0 | partial | admin read: A routes.go:72, W/h_admin.go:26; member/reviewer scopes in S/audit_read.go only, no UI/API |
| FR-083 | P0 | built | ST/migrations/0001_init.sql triggers; ST/sqlite_test.go TestTriggersBlockUpdateDeleteOnAppendOnlyTables |
| FR-084 | P1 | built | A/h_admin.go listAudit filters; W/templates/admin_audit.html |
| FR-090 | P0 | built | S/export.go; W/h_home.go:20 POST /export; S/export_test.go |
| FR-091 | P0 | built | S/export.go TestLedgerExportCSV; W/h_admin.go:28-29 |
| FR-092 | P0 | built | S/export.go members scope |
| FR-093 | P0 | built | S/export.go audit scope + manifest |
| FR-094 | P1 | built | S/export.go TestFullExportJSON |
| FR-095 | P1 | partial | service streams (S/export_write.go StreamTable) but API artefact held in memory 24 h (A/h_export.go); per-person cap only (A/hardening_test.go TestExportCapPerPerson); no size gate |

## 05 Architecture

| ID | Pri | Status | Evidence |
|---|---|---|---|
| ARC-001 | P0 | built | cmd/circula single binary; scripts/package-release.sh |
| ARC-002 | P0 | partial | service is write path; C/ reads store directly for backup/verify/inspect (ST/inspect.go, C/schema.go) |
| ARC-003 | P0 | built | A/ and W/ call service; domain/ holds rules |
| ARC-004 | P0 | partial | ST/iface.go interface; only SQLite implementation, no cross-store contract suite |
| ARC-005 | P0 | built | S/posting.go postTx sole writer |
| ARC-006 | P0 | built | write gate + BEGIN IMMEDIATE ST/sqlite.go; S/concurrency_test.go |
| ARC-007 | P0 | built | S/authz_test.go TestCrossNetworkIdsAreNotFound |
| ARC-008 | P0 | partial | no operator /api route (A/routes.go); operator CLI + break-glass (S/breakglass.go) audited; host SQL access outside service |
| ARC-009 | P1 | built | S/breakglass.go, C/breakglass.go, W/h_breakglass.go; S/breakglass_test.go, W/breakglass_erasure_test.go |
| ARC-010 | P0 | partial | input limits/tests (A/strict_test.go TestBodySizeLimit); no fuzz tests (`func Fuzz` none); no trust-boundary checklist |
| ARC-011 | P0 | built | C/lock*.go single-writer lock |
| ARC-012 | P1 | built | S/purge.go idempotent jobs; C/jobs_regress_test.go |
| ARC-013 | P0 | built | one artefact both models; hosted launch not done (non-code) |
| ARC-015 | P0 | partial | S/verify.go VerifyLedger; hourly job in C/ does not call RecordConsistencyRun (only explicit API/admin run records, S/verify.go:97) |
| ARC-016 | P1 | deviation | C/serve.go:36 shutdownGracePeriod=30s vs spec 10s; backup/restore built; RPO/RTO unrecorded |
| ARC-017 | P0 | partial | semantic templates; W/a11y_test.go; e2e/tests/a11y.spec.js axe run via `make e2e`, NOT in ci.yml |
| ARC-018 | P1 | partial | layering by convention; no import-rule lint (no depguard in golangci config) |
| ARC-019 | P0 | built | .github/workflows/ci.yml build job -> scripts/package-release.sh |
| ARC-020 | P0 | partial | export manifest (S/export_write.go); no reference parser/schema doc |
| ARC-021 | P1 | built | service Config clock/ID injection |
| ARC-022 | P1 | built | C/metrics.go circula_ledger_postings_total + replays; request IDs; JSON logs |
| ARC-023 | P1 | partial | config precedence + secret files (C/config.go); no secret scan in ci.yml |
| ARC-024 | P1 | not built | no licence manifest; package-release.sh emits Go module list only |

## 06 Data, ledger, audit

| ID | Pri | Status | Evidence |
|---|---|---|---|
| DAT-001 | P0 | built | UUIDv7 IDs; S/service.go id() |
| DAT-002 | P0 | built | injected generator |
| DAT-003 | P0 | built | IDs random UUIDv7 |
| DAT-004 | P1 | built | hashed invitation/session/api tokens; S/auth.go, S/tokens.go |
| DAT-005 | P0 | built | composite FKs ST/migrations/0001_init.sql |
| DAT-006 | P0 | built | 0001_init.sql entries FKs |
| DAT-007 | P0 | built | S/posting.go; S/property_test.go |
| DAT-008 | P0 | built | CHECKs 0001_init.sql; domain/money.go AddChecked |
| DAT-009 | P0 | built | S/verify.go network sum |
| DAT-010 | P0 | built | append-only triggers 0001_init.sql |
| DAT-011 | P0 | built | S/audit_test.go TestAppendOnlyThroughRawAccess |
| DAT-012 | P1 | deviation | one-reversal-per-original replaced by cumulative cap (DEC-040); S/reversal.go |
| DAT-013 | P1 | built | S/adjustments.go reason required |
| DAT-014 | P0 | built | S/atomicity_test.go TestAcceptFaultInjectionLeavesNoTrace |
| DAT-015 | P0 | built | S/posting.go checkCapacity under write unit |
| DAT-016 | P0 | built | ledger_seq; A TestEntriesOrderedByLedgerSeq |
| DAT-017 | P0 | built | UTC ms ints |
| DAT-018 | P0 | built | S/accounts.go |
| DAT-019 | P0 | built | AddBalance only in postTx |
| DAT-020 | P0 | built | S/verify.go; S/audit_test.go TestVerifyDetectsBalanceCorruptionAndBalanceAt |
| DAT-021 | P1 | built | S/accounts.go BalanceAt (service only) |
| DAT-022 | P0 | built | S/reversal.go; S/reversal_test.go |
| DAT-023 | P1 | built | S/adjustments.go system counter-account |
| DAT-024 | P1 | partial | time-limited credit-limit override (A routes.go:69 expires_at); no general override flag |
| DAT-025 | P0 | built | S/idempotency.go in-unit |
| DAT-026 | P0 | built | S/idempotency.go key scope |
| DAT-027 | P0 | built | S/idempotency_test.go; A 422 idempotency-key-reused |
| DAT-028 | P1 | built | TTL purge S/purge.go |
| DAT-029 | P0 | built | S/atomicity_test.go TestAuditFailureRollsBackStateChange |
| DAT-030 | P1 | built | hash chain S/audit.go; TestAuditChainDetectsTampering |
| DAT-031 | P0 | built | RedactSecrets; audit excludes tokens |
| DAT-032 | P1 | partial | audit export scope S/export.go; format parity unverified |
| DAT-033 | P0 | partial | S/audit_read.go member-safe summaries; no member UI/API route |
| DAT-034 | P0 | built | admin audit read audited (A/h_admin.go; S/audit_read.go) |
| DAT-035 | P0 | partial | reviewer scope in service; no reviewer UI |
| DAT-036 | P0 | built | no operator audit read path; break-glass record itself visible to admins |
| DAT-037 | P0 | built | S/export_write.go CSV+JSON, manifest SHA-256 |
| DAT-038 | P0 | built | S/export_test.go |
| DAT-039 | P0 | built | manifest ledger_seq snapshot |
| DAT-040 | P0 | partial | manifest checksums exist; no verifier command for export packages (C/snapshot.go verifyBackupFile is for backups) |
| DAT-041 | P1 | not built | no versioned export schema doc or reference reader |
| DAT-042 | P0 | built | ST/migrate.go + 9 embedded migrations |
| DAT-043 | P0 | built | C/premigrate.go; C/premigrate_test.go |
| DAT-044 | P0 | built | post-migration verify gate; C/serve.go |
| DAT-045 | P1 | partial | practice only; no test enforcing expand-contract |
| DAT-046 | P0 | built | migrations do not touch ledger rows; ST/golden_migration_test.go verifies ledger after migrate |
| DAT-047 | P0 | built | ST/golden_migration_test.go: 10,000 tx fixture built at schema v1, migrated, ledger+audit verified |
| DAT-048 | P1 | not built | ST/migrate.go records no duration |

## 07 API contract

| ID | Pri | Status | Evidence |
|---|---|---|---|
| API-001 | P0 | built | A and W call same S/ commands; internal/contract/contract_test.go |
| API-002 | P0 | built | A/routes.go /api/v1 prefix |
| API-003 | P0 | built | A/decode.go; A/strict_test.go TestStrictBodyHandling |
| API-004 | P0 | built | A/dto.go string money; domain/money.go |
| API-005 | P0 | built | A/dto.go ts() |
| API-006 | P0 | built | A/strict_test.go TestPatchRequiresMergePatchType |
| API-007 | P0 | built | A/dto.go direction |
| API-008 | P0 | built | httpx/httpx.go __Host- cookie; W/session.go HttpOnly Lax Secure; A TestSecureSessionCookieWorksForAPI |
| API-009 | P1 | built | S/tokens.go hashed bearer tokens |
| API-010 | P1 | partial | A/h_tokens.go scopes read/write; `admin` scope returns 501 (D-07) |
| API-011 | P1 | built | A/routes.go:95 cookieOnly create; token cannot create tokens |
| API-012 | P0 | built | A/middleware.go:59 credentials-in-query |
| API-013 | P0 | built | A/respond.go:155 WWW-Authenticate |
| API-014 | P0 | built | httpx CSRF (bound to session token); A/internal_test.go |
| API-015 | P0 | built | no CORS enabled |
| API-016 | P0 | built | S/principal.go status/role check |
| API-017 | P0 | built | S/authz_test.go |
| API-018 | P0 | built | A/authn_test.go TestAuthorisation |
| API-019 | P0 | built | no operator route in A/routes.go |
| API-020 | P0 | built | A/idem.go; routes with idem |
| API-021 | P0 | built | A TestIdempotencyKeyHeaderCode |
| API-022 | P0 | built | A/idem.go:162 Idempotent-Replayed |
| API-023 | P0 | built | A/idem.go 422 |
| API-024 | P0 | built | A/idem.go:154 409 in-progress |
| API-025 | P0 | built | A/idem.go failures not replayed as success |
| API-026 | P0 | built | ETag on policy/listing/limit; A/decode.go |
| API-027 | P0 | built | A/decode.go:206 428; 412 precondition-failed |
| API-028 | P0 | built | S/atomicity + domain/statemachine_test.go |
| API-029 | P0 | built | A/respond.go problem+json |
| API-030 | P0 | built | A/respond.go code, request_id, errors |
| API-031 | P0 | built | A/respond.go:159 urn:circula:problem: |
| API-032 | P0 | built | internal-error body; A/respond.go |
| API-033 | P1 | built | English detail |
| API-034 | P0 | built | A/cursor.go envelope next_cursor |
| API-035 | P0 | built | A/cursor.go integrity-protected cursors |
| API-036 | P1 | built | limits; tokens list capped 100 |
| API-037 | P0 | built | A/decode.go:28 MaxBytesReader; TestBodySizeLimit |
| API-038 | P1 | built | A/middleware.go limiter; TestRateLimit, TestIPRateLimitUsesTrustedProxyClientIP |
| API-039 | P0 | built | A/middleware.go:41 no-store |
| API-040 | P0 | deviation | requires authentication (no public route) - D/OQ-API-06; A/h_core.go |
| API-041 | P0 | deviation | returns memberships across all networks (D-01) |
| API-042 | P0 | built | A/h_core.go listMembers |
| API-043 | P0 | built | A/routes.go:24 |
| API-044 | P0 | built | A/h_accounts.go |
| API-045 | P0 | built | A TestEntriesOrderedByLedgerSeq |
| API-046 | P0 | built | A/h_listings.go; idem required (stricter) |
| API-047 | P1 | built | A/h_listings.go patchListing |
| API-048 | P0 | built | A/h_tx.go propose; A/trade_test.go TestFullTradeFlow |
| API-049 | P0 | built | A/h_tx.go |
| API-050 | P0 | built | A/h_tx.go txEvents |
| API-051 | P0 | built | A/h_tx.go acceptTx |
| API-052 | P0 | built | A/h_tx.go declineTx |
| API-053 | P0 | built | A/h_tx.go cancelTx |
| API-054 | P1 | built | A/h_disputes.go |
| API-055 | P1 | built | A/h_disputes.go resolveDispute (vocabulary aligned) |
| API-056 | P1 | deviation | admin only; spec's reviewer-via-dispute path not on this route (D-10); reversal via dispute resolve exists |
| API-057 | P1 | built | A/h_admin.go adminAdjust; above dual-control threshold returns pending (A/dualcontrol_test.go) |
| API-058 | P0 | built | A/h_policy.go |
| API-059 | P0 | deviation | all six actions built; remove allowed with non-zero balance (D-06), If-Match optional (D-05) |
| API-060 | P0 | built | A/h_admin.go putCreditLimit |
| API-061 | P1 | built | A/routes.go:70-71 |
| API-062 | P0 | built | A/h_admin.go listAudit |
| API-063 | P0 | deviation | results kept in memory 24 h / 200 (D-11); runs recorded only when RecordConsistencyRun used |
| API-064 | P1 | built | A/routes.go:75-77,20; S/announcements.go |
| API-065 | P1 | deviation | synchronous export; no queued/running/failed states, artefact in memory 24 h (D-13) |
| API-066 | P1 | built | A/h_tokens.go; A/routes.go:94-96 |
| API-067 | P1 | built | A/routes.go:27; S/departure.go |

## 08 UX and accessibility

| ID | Pri | Status | Evidence |
|---|---|---|---|
| UX-001 | P0 | built | W/render.go nav by role; W/security_test.go |
| UX-002 | P0 | partial | three-activation reachability not measured |
| UX-003 | P1 | built | W/templates/networks.html |
| UX-004 | P0 | built | W/flow_test.go:71 'In credit'/'In debt' |
| UX-005 | P0 | partial | sign + words; greyscale check not run |
| UX-006 | P0 | built | W/format.go; W/format_test.go |
| UX-007 | P0 | built | W/templates/help.html |
| UX-008 | P0 | built | W/format.go decimals, no symbols |
| UX-009 | P0 | built | W/templates/dashboard.html |
| UX-010 | P0 | built | dashboard.html |
| UX-011 | P0 | built | W/flow_test.go:75 'You are using 25.00 of your 100.00 credit limit' |
| UX-012 | P1 | built | S/notify_limits.go NotifLimitWarning; policy limit_warning_days (W/h_admin2.go:23) |
| UX-013 | P0 | built | W/format.go |
| UX-014 | P0 | built | dashboard.html pending section |
| UX-015 | P0 | built | history.html |
| UX-016 | P0 | built | W/render.go status labels |
| UX-017 | P1 | partial | reversal/adjustment rows shown; link to original not verified in history.html/trade_detail.html |
| UX-018 | P0 | built | propose_review.html, trade_accept.html, dispute_decide_review.html, admin_reverse_review.html |
| UX-019 | P0 | built | W/flow_test.go 'Trade completed.' |
| UX-020 | P0 | built | W/refusal.go |
| UX-021 | P0 | partial | error summary + aria-invalid/describedby (base.html:58-59); not verified for every error code |
| UX-022 | P0 | built | error summary focus (static/app.js:5), value retention P.Val (base.html:59) |
| UX-023 | P0 | built | inputmode decimal (base.html:59); format hint |
| UX-024 | P1 | built | S/timezone.go; W/h_account_f2.go; templates/time_zone.html; PATCH /me |
| UX-025 | P0 | built | W/templates/error.html:3 request reference |
| UX-026 | P0 | built | role=status flash (base.html:32) |
| UX-027 | P0 | partial | disabled-control text not audited |
| UX-028 | P0 | partial | no loading state; empty/error states present |
| UX-029 | P0 | built | W/refusal.go forbidden page |
| UX-030 | P0 | built | W/static/app.js session warning; /session/extend W/h_account.go:17 |
| UX-031 | P1 | built | W/render.go:213,233 suspended notice |
| UX-032 | P0 | built | admin_member_act.html confirm; W/h_trade_act.go |
| UX-033 | P0 | partial | @media max-width 40rem (static css:59); 320px manual only (docs/qa/LIGHT-QA.md U9) |
| UX-034 | P0 | partial | reflow not measured |
| UX-035 | P0 | partial | target size not measured |
| UX-036 | P0 | partial | no orientation lock; untested |
| UX-037 | P1 | built | table.stack css:64; used in 4 templates |
| UX-038 | P0 | partial | e2e/tests/keyboard.spec.js covers many flows, local `make e2e` only, not CI |
| UX-039 | P0 | partial | template order; not automated |
| UX-040 | P0 | partial | focus outline in CSS; keyboard.spec.js checks indicator on tab stops (local only) |
| UX-041 | P0 | built | skip link base.html |
| UX-042 | P0 | built | no drag UI |
| UX-043 | P0 | built | submit/link activation |
| UX-044 | P0 | built | base.html:6 title |
| UX-045 | P0 | built | W/a11y_test.go TestPageStructure |
| UX-046 | P0 | built | base.html:2 lang=en |
| UX-047 | P1 | not built | strings inline in templates/Go; no catalogue |
| UX-048 | P0 | built | W/a11y_test.go TestFormLabelsAndAutocomplete |
| UX-049 | P1 | partial | help link placement unverified |
| UX-050 | P0 | built | copy review; help.html |
| UX-051 | P0 | built | paste allowed |
| UX-052 | P0 | built | W/h_recover.go, S/recovery.go recovery codes; uniform register/login responses |
| UX-053 | P0 | built | S/mfa.go, S/totp.go, W/h_security.go (opt-in; RequireMFAForAdmins in C/config.go:101) |
| UX-054 | P0 | partial | axe-core in e2e/tests/a11y.spec.js, run only by `make e2e`; not wired in .github/workflows/ci.yml |
| UX-055 | P0 | partial | automated keyboard.spec.js exists; no recorded manual walkthrough per release |
| UX-056 | P0 | partial | e2e/tests/aria.spec.js proxy + docs/qa/SCREEN-READER.md script; NO human screen-reader run |
| UX-057 | P0 | partial | dark tokens (css:2); dark contrast not measured |
| UX-058 | P1 | built | prefers-reduced-motion css:66 |
| UX-059 | P1 | built | session extend |
| UX-060 | P0 | partial | docs/qa/DEFECTS.md log + classification; gate counts open A-block |

## 09 Security and privacy

| ID | Pri | Status | Evidence |
|---|---|---|---|
| SEC-001 | P0 | partial | Argon2id PHC (S/password.go); DEC-006 benchmark not recorded (password.go:31) |
| SEC-002 | P0 | partial | len>=12 enforced; PasswordDenied hook (S/service.go:26) never set by cmd/ - no breached list |
| SEC-003 | P0 | partial | server sessions, idle+absolute (S/auth.go:227, C/config.go:51), password change revokes; no role-specific timeouts |
| SEC-004 | P0 | built | W/session.go:64-90 CSRF token HMAC of session token when signed in; same-origin check |
| SEC-005 | P0 | built | uniform login/register/recover responses; W/recover_test.go |
| SEC-006 | P0 | partial | login_throttle table + exponential lockout (S/auth.go:89), per-IP limiter; no per-network/invitation limits |
| SEC-007 | P1 | built | S/mfa.go, S/totp.go, S/mfa_login.go; S/mfa_test.go, W/mfa_flow_test.go; opt-in |
| SEC-008 | P0 | built | S/recovery.go recovery codes; C/recover.go reset-password; S/recovery_test.go |
| SEC-009 | P0 | built | S/authz_test.go matrix |
| SEC-010 | P0 | built | S/membership_test.go; dual-control self-approval refused S/dualcontrol_test.go |
| SEC-011 | P1 | built | S/dualcontrol.go, S/adjustments.go:103; off by default (threshold 0, operator `dual-control` command) |
| SEC-012 | P0 | built | S/notices.go secNotice; W/h_security.go; (admin-action notices to affected members via S/notify.go limited kinds) |
| SEC-013 | P0 | built | S/breakglass.go, C/breakglass.go; docs/dev/break-glass.md |
| SEC-014 | P0 | partial | minimal fields; no data-inventory document in docs/ |
| SEC-015 | P0 | built | S/authz_test.go |
| SEC-016 | P0 | partial | directory members-only; no noindex/robots |
| SEC-017 | P0 | built | S/export.go formula neutralisation; A 24h expiry |
| SEC-018 | P0 | built | S/erasure.go, ST/sqlite_erasure.go, W/h_erasure.go; S/erasure_test.go |
| SEC-019 | P0 | partial | audit excludes secrets; member read service-only |
| SEC-020 | P0 | partial | RedactSecrets; canary only in S/auth_test.go, no log-wide scan |
| SEC-021 | P1 | not built | no log retention setting |
| SEC-022 | P0 | built | C/config.go:456 refuses http on non-loopback; Secure/__Host- cookies |
| SEC-023 | P0 | built | W/middleware.go CSP etc; W/security_test.go |
| SEC-024 | P0 | built | safeNext; html/template |
| SEC-025 | P1 | built | S/reports.go (FR-024) |
| SEC-026 | P1 | not built | no velocity/reciprocal signals (grep empty) |
| SEC-027 | P0 | partial | secrets via env/file; C/statekey.go, docs/specs/addenda/SEC-027-keys.md; session key rotation not implemented |
| SEC-028 | P0 | built | C/backupcrypt.go AES-256-GCM chunked; opt-in (backup-require-encryption default false); backupcrypt_test.go |
| SEC-029 | P1 | partial | govulncheck in ci.yml:44; module list only (no SBOM format); no checksum-failure test |
| SEC-030 | P0 | partial | docs/ops/runbook.md; no drill records |
| SEC-031 | P1 | built | SECURITY.md |
| SEC-032 | P1 | partial | default credit limit 0 (domain/policy.go:58); no cap/open-mode confirmation |
| SEC-033 | P1 | partial | tests cover most; docs/audit/SECURITY-AUDIT*.md internal; no independent review |

## 10 Governance and operations

| ID | Pri | Status | Evidence |
|---|---|---|---|
| GOV-001 | P0 | partial | policy versions prospective (ST network_policies); version not stored on transactions |
| GOV-002 | P0 | built | audited before/after; S/gov_terms.go PolicyLog; W/h_gov.go rulesPage; GET /network/policy-log (claude/cc-h1) |
| GOV-003 | P0 | built | S/networks.go UpdatePolicy noticeDays; ST network_policies effective_at as-of read; migration 0012 (claude/cc-h1) |
| GOV-004 | P0 | built | S/networks.go decimals lock |
| GOV-005 | P1 | built | domain/statemachine_test.go TestPolicyBounds |
| GOV-006 | P1 | partial | unilateral admin change; no objection mechanism |
| GOV-007 | P0 | built | S/reversal.go, adjustments; property tests |
| GOV-008 | P0 | built | S/join.go; S/membership_test.go TestJoinModes |
| GOV-009 | P0 | built | S/gov_terms.go; S/join.go *Terms; gate in transactions.go; migration 0012 (claude/cc-h1) |
| GOV-010 | P0 | partial | S/departure.go blockers/retain/settle; counterparty notices built (claude/cc-h1); self-export only linked |
| GOV-011 | P0 | partial | admin removal with reason; member notice built (claude/cc-h1); no challenge |
| GOV-012 | P1 | not built | no inactivity rule |
| GOV-013 | P0 | built | S/disputes.go |
| GOV-014 | P0 | partial | statement, response (S/disputes_response.go), decision with reasons; no response deadline/auto-escalation |
| GOV-015 | P0 | built | S/reversal.go |
| GOV-016 | P0 | partial | re-review request/decision (A routes.go:81-82) is one level of challenge; no formal appeal body/time limits |
| GOV-017 | P1 | built | no upload route |
| GOV-018 | P0 | partial | trades continue within limits; write-off via adjustment (dual control optional) |
| GOV-019 | P0 | built | S/membership.go SetCreditLimit |
| GOV-020 | P0 | partial | last-admin guard; no read-only admin-required state |
| GOV-021 | P0 | partial | break-glass grant-admin (docs/dev/break-glass.md) + reset-password; no member-facing recovery of lost-admin network beyond operator |
| GOV-022 | P0 | partial | S/export.go member/network/full; import deferred |
| GOV-023 | P0 | not built | no network suspension (networks status has no suspended) |
| GOV-024 | P0 | not built | no platform suspension |
| GOV-025 | P0 | partial | CloseNetwork S/networks.go:127 refuses with blockers; no caller, notices, final export, erasure |
| GOV-026 | P1 | not built | no hosted platform policy docs |
| GOV-027 | P1 | partial | docs/ops runbook boundary; not linked from admin UI |

## 11 Deployment and operations

| ID | Pri | Status | Evidence |
|---|---|---|---|
| OPS-001 | P0 | built | one artefact |
| OPS-002 | P0 | built | C/serve.go |
| OPS-003 | P0 | partial | ci.yml build job + checksums; no per-target smoke run |
| OPS-004 | P1 | partial | docs/ops/README.md; no staging refresh |
| OPS-006 | P0 | built | C/config.go; C/config_test.go |
| OPS-007 | P0 | built | check-config; C/config_test.go |
| OPS-008 | P0 | built | C/config.go defaults |
| OPS-009 | P0 | partial | secret files (C/cmd_secrets.go); inline-secret warning untested |
| OPS-010 | P1 | partial | backup key rotation documented; session key rotation absent |
| OPS-011 | P0 | built | ST/sqlite.go pragmas; ST/hardening_test.go |
| OPS-012 | P0 | built | short units; busy -> 503 |
| OPS-013 | P0 | built | C/ops_http.go; C/ops_test.go |
| OPS-014 | P0 | built | C/metrics.go postings/replays/jobs; C/metrics_hooks_test.go |
| OPS-015 | P0 | partial | docs/ops/monitoring.md alert table; not configured/tested |
| OPS-016 | P0 | built | S/verify.go read-only; CLI+admin |
| OPS-017 | P0 | built | C/snapshot.go manifest+checksum; C/backup_test.go |
| OPS-018 | P0 | built | C/jobs_limits.go retention 30d/12m |
| OPS-019 | P0 | built | C/backupcrypt.go; opt-in encryption; off-host copy is operator procedure |
| OPS-020 | P0 | partial | restore refuses while serving and verifies (C/commands.go:325); restore record is local file not ledger audit (commands.go:421) |
| OPS-021 | P0 | partial | docs/ops/backup-restore.md:150 procedure; no drill records |
| OPS-022 | P0 | built | ST/migrate.go newer schema refused |
| OPS-023 | P0 | built | C/premigrate.go pre-migration integrity+backup+verify; C/premigrate_test.go; post-migration gate |
| OPS-024 | P0 | built | C/maintenance.go; C/maintenance_test.go; docs/ops/upgrade.md |
| OPS-025 | P1 | built | migrate mode default explicit |
| OPS-026 | P0 | partial | docs/releases/TEMPLATE.md; release.yml:33 requires notes file; no notes for any version yet |
| OPS-027 | P0 | built | C/jobs_limits.go; C/jobs_regress_test.go |
| OPS-028 | P0 | built | S/purge.go |
| OPS-029 | P1 | partial | no mail by design; in-app inbox + security notices (S/notify.go, S/notices.go) |
| OPS-030 | P1 | partial | docs/ops/runbook.md severities; linkage thin |
| OPS-031 | P1 | built | docs/ops/runbook.md boundary + break-glass |
| OPS-032 | P0 | built | JSON logs |
| OPS-033 | P0 | built | S/freeze.go per-network freeze; S/freeze_test.go, W/frozen_test.go, A/frozen_test.go |
| OPS-034 | P1 | not built | RPO/RTO unset (DEC-016 open) |
| OPS-035 | P1 | partial | scripts/package-release.sh SHA256SUMS + module list; no signing, no secret scan |

## 12 Quality and test plan

| ID | Pri | Status | Evidence |
|---|---|---|---|
| QA-001 | P0 | partial | S/property_test.go seeds 1-6; counts/op list unverified |
| QA-002 | P0 | built | S/property_test.go TestRandomisedLedgerInvariants |
| QA-003 | P0 | built | property_test.go |
| QA-004 | P0 | partial | TestQA004LimitBoundary (limits 0, 1, large); GOV-019 and system-account cases not named |
| QA-005 | P0 | built | domain TestNoFloatsInMoneyPackages, TestParseAmount |
| QA-006 | P0 | built | S/atomicity_test.go |
| QA-007 | P0 | partial | S/concurrency_test.go; not 100 interleavings |
| QA-008 | P0 | partial | TestQA008DisjointConcurrentTrades (100 trades, 60 s limit pending DEC-023) |
| QA-009 | P0 | built | S/idempotency_test.go |
| QA-010 | P0 | partial | fault store; not every posting step |
| QA-011 | P0 | partial | busy-retry test; no seeded contention scenario (S/loadtest_test.go is load) |
| QA-012 | P0 | built | S/authz_test.go; A/authn_test.go |
| QA-013 | P0 | partial | S/dispute_test.go, disputes_response_test.go; no response deadline |
| QA-014 | P0 | built | S/reversal_test.go |
| QA-015 | P0 | partial | policy tests; notice/objection not built |
| QA-016 | P0 | built | S/dualcontrol_test.go (approval expiry S/dualcontrol.go:20), S/membership_test.go |
| QA-017 | P0 | partial | invitation join tested; terms acceptance not built |
| QA-018 | P0 | built | W/flow_test.go, A/trade_test.go, internal/contract/contract_test.go (browser e2e covers a11y/keyboard only) |
| QA-019 | P0 | partial | decline/expiry tested; party notices partly (notify_test.go) |
| QA-020 | P0 | partial | service-level boundary; browser flow not run |
| QA-021 | P0 | built | dispute+reversal+notifications S/notify_test.go, W/f2_test.go |
| QA-022 | P0 | partial | S/notify_test.go limit warning; limit-reduction notice to member partial |
| QA-023 | P0 | partial | departure tested; counterparty notices/self-export offer missing |
| QA-024 | P0 | partial | C/backup_test.go restore; restore not in ledger audit; no load |
| QA-025 | P0 | partial | import deferred by spec (04 s10) - not counted as gap |
| QA-026 | P0 | partial | C/premigrate_test.go failure path; no forced-failure upgrade E2E |
| QA-027 | P0 | built | ST/golden_migration_test.go (v1 fixture -> current) |
| QA-028 | P0 | partial | restore integrity tested; backup under concurrent load not run |
| QA-029 | P0 | partial | scripts/restore-drill.sh + docs/qa/EVIDENCE.md; one drill on an empty DB, none on real data |
| QA-030 | P0 | partial | export totals tested; no independent reconciliation script |
| QA-031 | P0 | built | S/export_test.go canary/scope |
| QA-032 | P0 | built | S/export_test.go formula neutralisation |
| QA-033 | P0 | partial | e2e job in ci.yml (make e2e, axe); not yet observed green on GitHub |
| QA-034 | P0 | partial | e2e/tests/keyboard.spec.js local; scripting-disabled flows via Go tests |
| QA-035 | P0 | partial | same as UX-056: script written, not run by a human |
| QA-036 | P0 | partial | balance text tested; contrast via axe local only |
| QA-037 | P0 | partial | scripts/check-hygiene.sh migration numbering/destructive check in CI |
| QA-038 | P0 | partial | e2e job in ci.yml; not yet observed green on GitHub |
| QA-039 | P0 | partial | scripts/release-gate.sh + docs/qa/RELEASE-GATE.md; manual items (SR, L14, SEC-033, L13) never auto-pass |
| QA-040 | P0 | partial | docs/qa/THRESHOLDS.md proposed values; DEC-023 owner confirmation missing, gate fails until then |
| QA-041 | P0 | partial | injected clocks; seeds only in property test |
| QA-042 | P0 | partial | docs/TRACEABILITY.md exists (stale: snapshot 925f637) |
| QA-043 | P0 | partial | scripts/check-hygiene.sh PII scan in CI |
| QA-044 | P0 | partial | docs/qa/EVIDENCE.md retention process; release-notes link check manual |

## Summary

| Priority | built | partial | not built | deviation | total |
|---|---|---|---|---|---|
| P0 | 217 | 79 | 12 | 4 | 312 |
| P1 | 51 | 24 | 9 | 4 | 88 |
| all | 268 | 103 | 21 | 8 | 400 |

Note: QA-025 (import) is marked partial but is deferred by the spec; treat as out of scope.

Per doc:

| Doc | built | partial | not built | deviation |
|---|---|---|---|---|
| 04 Functional requirements | 56 | 8 | 0 | 0 |
| 05 Architecture | 12 | 9 | 1 | 1 |
| 06 Data, ledger, audit | 39 | 6 | 2 | 1 |
| 07 API contract | 60 | 1 | 0 | 6 |
| 08 UX and accessibility | 40 | 17 | 3 | 0 |
| 09 Security and privacy | 18 | 13 | 2 | 0 |
| 10 Governance and operations | 8 | 13 | 6 | 0 |
| 11 Deployment and operations | 22 | 11 | 1 | 0 |
| 12 Quality and test plan | 13 | 25 | 6 | 0 |

## Ranked P0 gaps

Order: not built, then partial, then deviation; swarm = owned by security swarm area (auth.go, MFA, recovery, dual control, break-glass, erasure, backup encryption, release.yml, cookie hardening, login throttle); free = ours.

| # | ID | Status | Fix sketch | Size | Owner |
|---|---|---|---|---|---|
| 1 | GOV-003 | not built | policy changes take effect after notice period; auto-announcement + effective_at | M | free |
| 2 | GOV-009 | not built | terms versions table, acceptance at join, re-accept on change | L | free |
| 3 | GOV-023 | not built | add suspended network status; block writes | M | free |
| 4 | GOV-024 | not built | platform-level suspension flag + operator command | M | free |
| 5 | UX-056 | not built | run + record screen-reader pass | M | free |
| 6 | UX-060 | not built | define defect log process | S | free |
| 7 | QA-008 | not built | add disjoint concurrent trades test | S | free |
| 8 | QA-029 | not built | record restore drill | S | free |
| 9 | QA-035 | not built | same as UX-056 | M | free |
| 10 | QA-039 | not built | assemble release gate evidence | M | free |
| 11 | QA-040 | not built | decide thresholds DEC-023 | S | free |
| 12 | QA-044 | not built | define evidence retention | S | free |
| 13 | ARC-002 | partial | route CLI direct-store reads via service read-only methods | M | free |
| 14 | ARC-004 | partial | extract store contract test suite run against SQLite | M | free |
| 15 | ARC-008 | partial | document/enforce no operator route; add test that routes lack operator scope | S | free |
| 16 | ARC-010 | partial | add Fuzz tests for decode/amount/cursor/csv; trust-boundary checklist | M | free |
| 17 | ARC-015 | partial | call RecordConsistencyRun from hourly verify job | S | free |
| 18 | ARC-017 | partial | run axe e2e job in ci.yml | S | free |
| 19 | ARC-020 | partial | publish export schema + reference reader/parser | M | free |
| 20 | DAT-033 | partial | same as FR-082 member route | M | free |
| 21 | DAT-035 | partial | reviewer audit page, same work as FR-082 | S | free |
| 22 | DAT-040 | partial | add `circula verify-export <zip>` re-hashing files vs manifest | S | free |
| 23 | FR-014 | partial | add member web 'Leave network' form + confirm calling DepartMember | S | free |
| 24 | FR-082 | partial | add member/reviewer audit routes (API + page) over S/audit_read.go | M | free |
| 25 | GOV-001 | partial | store policy_version on each transaction (migration 0010) | M | free |
| 26 | GOV-002 | partial | member-visible policy change log page from network_policies | S | free |
| 27 | GOV-010 | partial | notify counterparties on departure + offer self-export | M | free |
| 28 | GOV-011 | partial | notify member on admin removal + challenge path | M | free |
| 29 | GOV-014 | partial | response deadline + overdue state/job | M | free |
| 30 | GOV-016 | partial | formal appeal timeline/reviewer rule beyond one re-review | M | free |
| 31 | GOV-018 | partial | document/write-off flow via adjustment with dual control | S | free |
| 32 | GOV-020 | partial | admin-required read-only network state when no admin | M | free |
| 33 | GOV-021 | partial | member-facing lost-admin recovery request flow over break-glass | M | swarm |
| 34 | GOV-022 | partial | import of export package (deferred in spec) | L | free |
| 35 | GOV-025 | partial | expose CloseNetwork (admin UI+API), notices, final export | M | free |
| 36 | OPS-003 | partial | smoke-run each target in CI/manual | M | free |
| 37 | OPS-009 | partial | test inline-secret warning | S | free |
| 38 | OPS-015 | partial | ship alert rules and test in staging | M | free |
| 39 | OPS-020 | partial | write restore event into audit after restore | S | free |
| 40 | OPS-021 | partial | run and record restore drill | S | free |
| 41 | OPS-026 | partial | write first release notes file | S | swarm |
| 42 | SEC-001 | partial | benchmark argon2 params on min hardware, record DEC-006 | S | swarm |
| 43 | SEC-002 | partial | embed breached-password list; set PasswordDenied in cmd | S | swarm |
| 44 | SEC-003 | partial | role-specific session timeouts (admin shorter) | S | swarm |
| 45 | SEC-006 | partial | per-network/invitation/join limits | M | swarm |
| 46 | SEC-014 | partial | write personal-data inventory doc | S | free |
| 47 | SEC-016 | partial | same as FR-025 | S | free |
| 48 | SEC-019 | partial | expose member-safe audit read (see FR-082) | S | free |
| 49 | SEC-020 | partial | log-wide canary scan test | S | free |
| 50 | SEC-027 | partial | session key rotation w/ overlap + doc | M | swarm |
| 51 | SEC-030 | partial | record incident drill | S | free |
| 52 | UX-002 | partial | measure/assert 3-activation reach | S | free |
| 53 | UX-005 | partial | greyscale check | S | free |
| 54 | UX-021 | partial | audit all error codes in forms | M | free |
| 55 | UX-027 | partial | audit disabled controls | S | free |
| 56 | UX-028 | partial | loading states | S | free |
| 57 | UX-033 | partial | 320px e2e check | S | free |
| 58 | UX-034 | partial | reflow e2e check | S | free |
| 59 | UX-035 | partial | target size check | S | free |
| 60 | UX-036 | partial | orientation test | S | free |
| 61 | UX-038 | partial | extend keyboard.spec and run in CI | S | free |
| 62 | UX-039 | partial | reading-order check | S | free |
| 63 | UX-040 | partial | focus check in CI | S | free |
| 64 | UX-054 | partial | same: wire `make e2e` into CI | S | free |
| 65 | UX-055 | partial | record per-release keyboard walkthrough | S | free |
| 66 | UX-057 | partial | measure dark-scheme contrast | S | free |
| 67 | QA-001 | partial | assert op counts/list in property test | S | free |
| 68 | QA-004 | partial | name zero/one-unit boundary tests | S | free |
| 69 | QA-007 | partial | 100-interleaving run | S | free |
| 70 | QA-010 | partial | fault injection at each posting step | M | free |
| 71 | QA-011 | partial | seeded contention scenario | S | free |
| 72 | QA-013 | partial | response-deadline tests (after GOV-014) | S | free |
| 73 | QA-015 | partial | notice/objection tests (after GOV-003/006) | S | free |
| 74 | QA-017 | partial | terms tests (after GOV-009) | S | free |
| 75 | QA-019 | partial | expiry/decline notice tests | S | free |
| 76 | QA-020 | partial | browser limit-boundary e2e | S | free |
| 77 | QA-022 | partial | limit-reduction notice test | S | free |
| 78 | QA-023 | partial | departure notice tests | S | free |
| 79 | QA-024 | partial | restore under load test | M | free |
| 80 | QA-026 | partial | forced-migration-failure upgrade test | S | free |
| 81 | QA-028 | partial | backup under concurrent load | S | free |
| 82 | QA-030 | partial | independent reconciliation script | S | free |
| 83 | QA-033 | partial | same: wire e2e into CI | S | free |
| 84 | QA-034 | partial | scripting-disabled e2e | S | free |
| 85 | QA-036 | partial | contrast check in CI | S | free |
| 86 | QA-037 | partial | add migration static check | S | free |
| 87 | QA-038 | partial | same: wire e2e into CI | S | free |
| 88 | QA-041 | partial | record seeds in all random tests | S | free |
| 89 | QA-042 | partial | regenerate traceability matrix | S | free |
| 90 | QA-043 | partial | PII scan of fixtures | S | free |
| 91 | API-040 | deviation | accept as deviation or add public route | S | free |
| 92 | API-041 | deviation | accept deviation D-01 | S | free |
| 93 | API-059 | deviation | accept deviation D-06 | S | free |
| 94 | API-063 | deviation | persist check results via RecordConsistencyRun | S | free |

## Ranked P1 gaps

Order: not built, then partial, then deviation; swarm = owned by security swarm area (auth.go, MFA, recovery, dual control, break-glass, erasure, backup encryption, release.yml, cookie hardening, login throttle); free = ours.

| # | ID | Status | Fix sketch | Size | Owner |
|---|---|---|---|---|---|
| 1 | ARC-024 | not built | generate licence manifest in package-release.sh | S | free |
| 2 | DAT-041 | not built | same: versioned export schema doc + reader | M | free |
| 3 | DAT-048 | not built | record migration duration | S | free |
| 4 | GOV-012 | not built | inactivity job: flag then suspend per policy | M | free |
| 5 | GOV-026 | not built | write hosted platform policy docs | S | free |
| 6 | OPS-034 | not built | decide and record RPO/RTO (DEC-016) | S | free |
| 7 | SEC-021 | not built | log retention config + rotation doc | M | free |
| 8 | SEC-026 | not built | velocity/reciprocal signals report for admins | L | free |
| 9 | UX-047 | not built | externalise strings | L | free |
| 10 | API-010 | partial | implement admin token scope | M | free |
| 11 | ARC-018 | partial | add depguard import rules | S | free |
| 12 | ARC-023 | partial | add gitleaks/trufflehog step to ci.yml | S | free |
| 13 | DAT-024 | partial | general admin override flag or amend spec | S | free |
| 14 | DAT-032 | partial | test audit export format parity | S | free |
| 15 | DAT-045 | partial | test enforcing expand-then-contract | M | free |
| 16 | FR-012 | partial | add join rate limiter in W/web.go and API | S | free |
| 17 | FR-025 | partial | add `<meta name=robots noindex>` + X-Robots-Tag header + /robots.txt disallow | S | free |
| 18 | FR-054 | partial | add GET /accounts/me/balance-at + history UI | S | free |
| 19 | FR-064 | partial | add API route for default flag | S | free |
| 20 | FR-068 | partial | wire CloseNetwork to admin UI/API (same as GOV-025) | S | free |
| 21 | FR-095 | partial | stream export to response, size gate | M | free |
| 22 | GOV-006 | partial | objection record/vote on policy change | L | free |
| 23 | GOV-027 | partial | link support boundary from admin UI | S | free |
| 24 | OPS-004 | partial | document staging refresh | S | free |
| 25 | OPS-010 | partial | same: session key rotation | M | swarm |
| 26 | OPS-029 | partial | optional mail/webhook channel | L | free |
| 27 | OPS-030 | partial | link severities from incident procedure | S | free |
| 28 | OPS-035 | partial | sign artefacts (cosign), secret scan in release.yml | M | swarm |
| 29 | SEC-029 | partial | emit CycloneDX SBOM in package-release.sh/release.yml | S | swarm |
| 30 | SEC-032 | partial | cap on default credit limit + open-mode confirm | S | free |
| 31 | SEC-033 | partial | commission independent review | L | free |
| 32 | UX-017 | partial | link reversal/adjustment rows to original | S | free |
| 33 | UX-049 | partial | verify help link placement | S | free |
| 34 | API-056 | deviation | accept deviation D-10 | S | free |
| 35 | API-065 | deviation | async job states if exports get large | M | free |
| 36 | ARC-016 | deviation | set shutdown grace to spec or amend spec; record RPO/RTO | S | free |
| 37 | DAT-012 | deviation | accept as deviation DEC-040; update spec | S | free |

## Notes on ownership

- swarm rows are listed for tracking only; do not touch auth.go, MFA, recovery, dual control, break-glass, erasure, backup encryption, release.yml, cookie hardening or login throttle code from this branch.
- Largest free P0 items: GOV-003, GOV-009, GOV-023/024/025 (governance lifecycle), then FR-082/DAT-033/035 (member and reviewer audit views), then CI wiring of the existing e2e/axe suite (UX-054, QA-033, QA-038) and ops evidence (OPS-021, QA-029, QA-039, UX-056).
- Many remaining P0 `partial` rows are evidence/measurement gaps (drills, screen-reader pass, thresholds), not missing features.

