# Traceability matrix

Status: generated from one status table at the build pass · Last updated: 2026-10-21 · Scope: every requirement ID in 00 to 06 and 08 to 13, the API contract IDs in 07, and the decision register IDs in 13 · Depends on: all spec files, [DECISIONS.md](DECISIONS.md)

This matrix answers three questions for each ID: what the requirement says (short title and priority, from the spec), what state the code is in (status), and where it lives (code path and test names). Only symbols and files that exist in the repository are cited; the citations were checked by resolving each name against the Go, SQL, template and script files. A citation proves that the code or test exists. It does not prove that the test passes at a given build.

## Snapshot and test status

Snapshot: HEAD `925f637` (branch claude/circula-build). The statuses were first read at `31b1015` and re-checked against the commits that followed (`a006a90` OpenAPI description, `f8e0b84` security research report, `925f637` keyset paging for directory and admin members). Untracked test files added by other work during the pass were present at the final run (cmd/circula/cli_regress_test.go, cmd/circula/jobs_regress_test.go, internal/api/regress_internal_test.go, internal/api/regress_scale_test.go, internal/service/regress_rejoin_test.go); they are not committed and are not cited here.

Checks run in this pass: `go vet ./...` exit 0 (final run). `go test -count=1 ./...` at the final snapshot: ok for cmd/circula, internal/api, internal/domain, internal/httpx, internal/service, internal/store, internal/web (internal/testutil has no tests). An earlier run in this pass, before `925f637`, failed two directory-paging tests (`TestDirectoryPagingReachesEveryMember`, `TestDirectoryNameFilterCoversAllMatches`) in an untracked file; they pass on the final run. Not run: the race detector, browser end-to-end tests, an accessibility tool, load tests, backup-under-load, restore drills, the power-loss test, and `govulncheck`.

## Status values

- **built**: behaviour exists in the code and at least one test or observed run covers it (test names listed).
- **partial**: exists with the gaps stated in the note.
- **not built**: no implementation.
- **deferred**: out of MVP by the spec or by a recorded decision.
- Decision-register rows (DEC-001 to DEC-027 in 00 and 13) use decision states: decided, open or deferred. The owner decisions are in [DECISIONS.md](DECISIONS.md).

Counts (requirement IDs, excluding decision-register rows): built 290, deferred 9, not built 41, partial 153.


## 00 Conventions and invariants

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| LED-1 | P0 | Sum of all entries per transaction = 0; sum of all balances per network = 0. | built | `internal/service/posting.go` (postTx)<br>`internal/store/migrations/0001_init.sql`<br>`internal/service/verify.go` (VerifyLedger) | `internal/service/trades_test.go:TestTradeScenarioBalances`<br>`internal/service/property_test.go:TestRandomisedLedgerInvariants` | Two postings per trade in one unit; DB trigger; network sum checked by verify |
| LED-2 | P0 | Entries are append-only; no UPDATE/DELETE on entries; corrections are new transactions. | built | `internal/store/migrations/0001_init.sql` | `internal/store/sqlite_test.go:TestTriggersBlockUpdateDeleteOnAppendOnlyTables`<br>`internal/service/audit_test.go:TestAppendOnlyThroughRawAccess` | Append-only triggers on entries, audit, status/dispute events, policies, runs; cache row unguarded (CODE-AUDIT LED-2) |
| LED-3 | P0 | Balance is derivable by summing entries; cached balance MUST equal the sum (verified by a  | built | `internal/store/sqlite_account.go` (AddBalance)<br>`internal/service/verify.go` (VerifyLedger) | `internal/service/audit_test.go:TestVerifyDetectsBalanceCorruptionAndBalanceAt` | AddBalance only inside postTx; VerifyLedger compares cache with sum |
| LED-4 | P0 | Posting both sides is atomic; retry with same idempotency key returns the original result. | built | `internal/service/idempotency.go` (idempotent) | `internal/service/idempotency_test.go:TestIdempotentRetryReturnsOriginal`<br>`internal/service/atomicity_test.go:TestAcceptFaultInjectionLeavesNoTrace` | One store.Update per command; idempotent() writes key in same unit |
| LED-5 | P0 | No posting may take a non-system account below −credit limit at post time (checked inside  | built | `internal/service/posting.go` (checkCapacity) | `internal/service/concurrency_test.go:TestParallelAcceptsNeverBreachCreditLimit`<br>`internal/service/trades_test.go:TestAcceptRechecksLimit` | checkCapacity inside write unit; service-level only, no DB constraint |
| LED-6 | P0 | Every state change writes an audit event in the same atomic unit. | built | `internal/service/audit.go` (audit) | `internal/service/atomicity_test.go:TestEveryCommandWritesAuditEvent`<br>`internal/service/atomicity_test.go:TestAuditFailureRollsBackStateChange` | Audit written in every command unit; login and session events not audited |
| LED-7 | P0 | Amounts are integers > 0 for trades; no floating point anywhere in money handling. | built | `internal/domain/money.go` (AddChecked)<br>`internal/domain/policy.go` (MaxPolicyAmount) | `internal/domain/domain_test.go:TestNoFloatsInMoneyPackages`<br>`internal/domain/domain_test.go:TestParseAmount` | int64 minor units; AddChecked overflow guard; MaxPolicyAmount; DB CHECK amount>0 |
| DEC-001 | - | Single self-hostable binary + optional hosted deployment of the same binary. Implementatio | decided, built | `internal/store/sqlite.go` (Open)<br>`internal/store/sqlite.go` (SQLite)<br>`internal/store/migrate.go` (migrate)<br>`scripts/package-release.sh` | - | Single Go binary; embedded SQLite (pure-Go driver) behind store; PostgreSQL not built |
| DEC-002 | - | Server-rendered HTML (progressive enhancement, no JS required for core flows) + versioned  | decided, built | `internal/web/web.go` (NewHandlerWith)<br>`internal/api/api.go` (NewHandlerWithOptions)<br>`internal/api/routes.go` | `internal/api/trade_test.go:TestFullTradeFlow` | Server-rendered HTML and /api/v1 both call service layer |
| DEC-003 | - | Release targets: windows/amd64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, plus | open, partly built | `scripts/package-release.sh`<br>`.github/workflows/ci.yml` | - | CI builds the five targets via package-release.sh; smoke run on each target not performed |
| DEC-004 | - | Product license undecided (owner decision); repo ships without LICENSE until decided; reco | open, not built | `SECURITY.md` | - | Licence undecided; no LICENSE file in repo (see 13 register) |

## 01 Vision and scope

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| VIS-001 | P0 | Scope statement is accurate | partial | `internal/web/templates/help.html`<br>`internal/web/templates/dashboard.html` | - | Help text and dashboard copy avoid redeemable/money wording; no review checklist run |
| VIS-002 | P0 | Network acceptance is not guaranteed | partial | `internal/web/templates/help.html` | - | Help text says nobody is obliged to accept credits; no network "about" page |
| VIS-003 | P0 | Networks are isolated tenants | built | - | `internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound`<br>`internal/api/authn_test.go:TestAuthorisation` | Cross-network ids return not-found in service and API |
| VIS-004 | P0 | Ledger invariants are enforced and tested | built | - | `internal/service/property_test.go:TestRandomisedLedgerInvariants`<br>`internal/service/trades_test.go:TestTradeScenarioBalances`<br>`internal/service/audit_test.go:TestAuditChainDetectsTampering` | LED invariant tests present; run in go test |
| VIS-005 | P0 | Per-network rule configuration | built | `internal/service/networks.go` (UpdatePolicy) | `internal/domain/statemachine_test.go:TestPolicyBounds`<br>`internal/service/admin_test.go:TestPolicyChanges` | Policy stored per network; bounds tested; prospective versions |
| VIS-006 | P0 | MVP completeness gate | not built | - | - | Completeness gate not met: P0/P1 gaps listed in 04 and 09-11 status tables |
| VIS-007 | P0 | Deployment shape | partial | `scripts/package-release.sh`<br>`.github/workflows/ci.yml` | - | One artefact per target from package-release.sh; clean-machine install not tested |
| VIS-008 | P1 | Accessibility baseline | partial | - | `internal/web/a11y_test.go:TestPageStructure`<br>`internal/web/a11y_test.go:TestFormLabelsAndAutocomplete` | Structural accessibility tests only; WCAG tooling not run |
| VIS-009 | P1 | Data exportability | partial | `internal/service/export.go` (Export) | `internal/service/export_test.go:TestFullExportJSON`<br>`internal/service/export_test.go:TestLedgerExportCSV` | Export round trip implemented; import (needed for round trip) deferred |
| VIS-010 | P1 | Data minimisation | partial | `internal/testutil/testutil.go` (Member)<br>`internal/testutil/testutil.go` (Person) | - | Entity list in 06 exists; no personal-data inventory document |
| VIS-011 | P1 | No monetization in the MVP | built | `internal/service/transactions.go` (ProposeTransaction) | - | No billing or payment-provider code; "payment" is trade wording only |
| VIS-012 | P2 | Deferred capabilities stay inactive | built | `internal/api/routes.go` | - | No route for deferred capabilities in routes.go |

## 02 Personas and journeys

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| PER-001 | P0 | Network member (role) | built | `internal/service/accounts.go` (GetBalance)<br>`internal/service/transactions.go` (ProposeTransaction)<br>`internal/service/reads.go` (ListEntries) | - | Member role: balance, listings, trades, disputes, export |
| PER-002 | P0 | Network administrator (role) | partial | `internal/service/membership.go` (AssignRole) | `internal/service/membership_test.go:TestAdministratorSuccession`<br>`internal/service/reversal_test.go:TestReversalAuthorization` | Admin limits and self-restrictions built; dual control, notices and reviewer-conflict audit partly missing |
| PER-003 | P0 | Dispute reviewer (role) | partial | `internal/service/disputes_decide.go` (ResolveDispute) | `internal/service/dispute_test.go:TestDisputeDecisionAuthorization`<br>`internal/service/dispute_test.go:TestDisputeVisibility` | Reviewer decides with independence checks; admin without reviewer role can also decide (open decision DEC-072) |
| PER-004 | P0 | Platform operator (role) | partial | `cmd/circula/commands.go` (cmdAdmin) | - | No operator role routes; break-glass and operator access record not built |
| PER-010 | P0 | J1 Join a network | partial | `internal/service/join.go` (RedeemInvitation)<br>`internal/service/join.go` (JoinNetwork)<br>`internal/service/join.go` (DecideJoinRequest) | `internal/service/membership_test.go:TestJoinModes` | Invite, approval and open joins built; onboarding terms (GOV-009) not built |
| PER-011 | P0 | J2 Offer or request goods and services | built | `internal/service/listings.go` (CreateListing)<br>`internal/service/listings.go` (UpdateListing)<br>`internal/service/listings.go` (ListListings) | - | Create, edit, pause, archive listing; browse by type and text |
| PER-012 | P0 | J3 Propose and accept a trade | built | `internal/service/transactions.go` (ProposeTransaction)<br>`internal/service/transactions.go` (AcceptTransaction) | `internal/api/trade_test.go:TestFullTradeFlow` | Propose, accept, decline, cancel; review step; idempotent commands |
| PER-013 | P0 | J4 Review balance and history | partial | `internal/service/accounts.go` (GetBalance)<br>`internal/service/tx_queries.go` (ListTransactions) | `internal/web/hardening_test.go:TestDashboardShowsOldProposalBehindManyTrades` | Balance and history views built; stale-data indicator (PER-022) not built |
| PER-014 | P0 | J5 Raise a dispute | partial | `internal/service/disputes.go` (OpenDispute)<br>`internal/service/disputes_decide.go` (ResolveDispute) | `internal/service/dispute_test.go:TestOpenDisputeRules` | Dispute open, review and decision built; other-party response (FR-076) not built |
| PER-015 | P0 | J6 Administer network rules | partial | `internal/service/membership.go` (SuspendMember)<br>`internal/service/adjustments.go` (CreateAdjustment)<br>`internal/service/membership.go` (SetCreditLimit)<br>`internal/service/networks.go` (UpdatePolicy) | - | Admin controls built; limit warnings, dual control and network suspension not built |
| PER-016 | P1 | J7 Leave a network or be offboarded | partial | `internal/service/departure.go` (DepartMember) | `internal/service/admin_test.go:TestDepartureBlockersAndAdminOffboarding` | Departure with blockers built; counterparty notices and self-export offer not built |
| PER-017 | P1 | J8 Close a network | partial | `internal/service/networks.go` (CloseNetwork) | - | CloseNetwork in service only; no web or API route calls it |
| PER-020 | P0 | Error states are specific and safe | built | - | `internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound` | Not-found is uniform for other-network and missing ids |
| PER-021 | P0 | Empty states explain the next step | partial | `internal/web/templates/history.html` | - | Empty-state copy in templates; no UX test run |
| PER-022 | P1 | Loading and stale data | not built | - | - | No stale-data indicator on balance views |

## 03 Domain model

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| DOM-001 | P0 | Network tenant boundary | built | - | `internal/store/sqlite_test.go:TestCrossNetworkReferencesRejected`<br>`internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound` | Every entity carries network id; cross-network refs refused |
| DOM-002 | P0 | One member and one account per person per network | built | `internal/service/join.go` (JoinNetwork) | `internal/service/hardening_test.go:TestRejoinSemantics` | One member per person per network; second join returns existing |
| DOM-003 | P0 | Membership states and join modes | built | - | `internal/domain/statemachine_test.go:TestMemberTransitionsExhaustive` | Membership states and transitions table-tested |
| DOM-004 | P0 | Account states | built | - | `internal/domain/statemachine_test.go:TestAccountStateDerivedFromMembership` | Account state derived from membership |
| DOM-005 | P0 | Integer minor units, no floating point | built | `internal/domain/money.go` (ParseAmount) | `internal/domain/domain_test.go:TestAmountRoundTripProperty`<br>`internal/domain/domain_test.go:TestNoFloatsInMoneyPackages` | Integer minor units; parser rejects fractions and zero |
| DOM-006 | P0 | Balance semantics | built | `internal/service/accounts.go` (GetBalance)<br>`internal/web/format.go` (balanceVM)<br>`internal/web/format.go` (makeBalance) | - | Balance is sum of signed entries; sign convention in UI |
| DOM-007 | P0 | Credit limit and effective limit | built | `internal/service/membership.go` (SetCreditLimit) | `internal/domain/statemachine_test.go:TestEffectiveLimit` | Effective limit: override else default |
| DOM-008 | P0 | Available credit | built | `internal/store/sqlite_account.go` (ReservedOutgoing) | `internal/service/trades_test.go:TestAvailableCreditTable` | Available = balance + limit - reserved outgoing |
| DOM-009 | P0 | Transaction kinds | built | `internal/domain/types.go` (TxKind) | `internal/domain/statemachine_test.go:TestTxStateMachineExhaustive` | Transaction kinds trade, reversal, adjustment, grant |
| DOM-010 | P0 | Allowed transitions | built | `internal/domain/statemachine.go` (NextTxStatus) | `internal/domain/statemachine_test.go:TestTxStateMachineExhaustive` | Transition table enforced; DB trigger tx_status_flow |
| DOM-011 | P0 | Consent before debit | built | - | `internal/service/trades_test.go:TestConsentBothDirections`<br>`internal/service/hardening_test.go:TestUnsolicitedRequestsDoNotFreezeCredit` | Payer and payee consent; requests do not post without accept |
| DOM-012 | P0 | Proposal checks and capacity reservation | built | `internal/service/transactions.go` (ProposeTransaction) | `internal/service/trades_test.go:TestReservationAndRelease`<br>`internal/service/hardening_test.go:TestRequestExposureAndCounts` | Reservation on payer's own proposals; requests refined per SA-08 (DEC-035) |
| DOM-013 | P0 | Proposal immutability and exits | built | `internal/service/transactions.go` (ExpireProposals) | `internal/service/trades_test.go:TestExpiry` | Terms immutable; expiry is system transition with no posting |
| DOM-014 | P0 | Acceptance posts atomically and re-checks capacity | built | `internal/service/transactions.go` (AcceptTransaction) | `internal/service/concurrency_test.go:TestParallelDoubleAcceptSingleSuccess` | Acceptance re-checks both parties and limit in one unit |
| DOM-015 | P0 | Completed transactions have exactly two equal and opposite entries | built | `internal/service/posting.go` (postTx) | `internal/service/trades_test.go:TestTradeScenarioBalances` | Completed trades have exactly two equal and opposite entries |
| DOM-016 | P0 | Idempotent proposal, acceptance and posting | built | `internal/service/idempotency.go` (idempotent) | `internal/service/idempotency_test.go:TestIdempotencyKeyReusedWithDifferentPayload` | Idempotency key on retryable commands; key reuse refused |
| DOM-017 | P0 | When a dispute may be opened | built | `internal/service/disputes.go` (OpenDispute) | `internal/service/dispute_test.go:TestOpenDisputeWindowAndNoPostings` | Dispute window from completion; opening creates no entries |
| DOM-018 | P0 | Dispute lifecycle and outcomes | built | `internal/service/disputes_decide.go` (ResolveDispute) | `internal/domain/statemachine_test.go:TestDisputeStateMachineExhaustive`<br>`internal/service/dispute_test.go:TestResolveOutcomes` | Dispute states and outcomes with rationale and decider |
| DOM-019 | P0 | Reversal construction | built | `internal/service/reversal.go` (ReverseTransaction) | `internal/service/reversal_test.go:TestFullReversal` | Reversal is new transaction; reversal cannot be reversed |
| DOM-020 | P0 | Full and partial reversal and the cumulative cap | built | - | `internal/service/reversal_test.go:TestPartialReversalCumulativeCap` | Cumulative reversed total capped at original amount |
| DOM-021 | P0 | Reversal blocked by the limit | built | `internal/service/disputes_decide.go` (RetryDisputePosting) | `internal/service/dispute_test.go:TestUpheldReversalBlockedThenOverride` | Upheld reversal blocked by limit goes to posting_pending |
| DOM-022 | P0 | Adjustments | built | `internal/service/adjustments.go` (CreateAdjustment) | `internal/service/admin_test.go:TestAdjustments` | Adjustment: reason, system counter-entry, own-account refused |
| DOM-023 | P0 | Corrections are new transactions | built | - | `internal/service/audit_test.go:TestAppendOnlyThroughRawAccess` | No edit or delete of completed records; corrections are new transactions |
| DOM-024 | P0 | Member departure | built | `internal/service/departure.go` (DepartMember) | `internal/service/admin_test.go:TestDepartureRetain`<br>`internal/service/admin_test.go:TestDepartureSettleToZero` | Departure preconditions, retain and settle-to-zero rules |
| DOM-025 | P1 | Default handling | partial | `internal/service/membership.go` (SetDefaultFlag) | `internal/service/hardening_test.go:TestAcceptRechecksDefaultFlag` | Default flag built (no ledger effect); limit-warning job (DOM-025.1) not built |
| DOM-026 | P0 | System account | built | `internal/store/sqlite_account.go` (GetSystemAccount) | `internal/service/trades_test.go:TestTradeScenarioBalances` | One system account per network; exempt from limit |
| DOM-027 | P1 | Sign-up grants | built | `internal/service/adjustments.go` (postSignupGrant) | `internal/service/admin_test.go:TestSignupGrant` | Sign-up grant posted from system account on activation; off by default (DEC-051) |
| DOM-028 | P0 | Invariants are not configurable | built | - | `internal/domain/statemachine_test.go:TestPolicyBounds` | No configuration key for invariants |
| DOM-029 | P0 | Configurable policies and bounds | built | `internal/domain/policy.go` (Validate)<br>`internal/domain/policy.go` (DefaultPolicy) | `internal/domain/statemachine_test.go:TestPolicyBounds` | Policy bounds and unknown-field refusal; default MaxTxAmount differs (DEC-056) |
| DOM-030 | P0 | Policy changes take effect prospectively | built | `internal/store/sqlite_network.go` (AppendPolicy)<br>`internal/service/networks.go` (UpdatePolicy) | `internal/service/admin_test.go:TestPolicyChanges` | Policy versions appended; changes audited with old and new values |
| DOM-031 | P0 | Decimals fixed after first posting | built | `internal/service/networks.go` (UpdatePolicy) | `internal/service/admin_test.go:TestPolicyChanges` | Decimals locked after first posting |
| DOM-032 | P0 | Limit changes do not post | built | - | `internal/service/trades_test.go:TestLimitReductionDoesNotPost`<br>`internal/service/trades_test.go:TestAcceptRechecksLimit` | Limit change posts nothing; reduction blocks only new debt |
| DOM-034 | P2 | Mutual reversal | deferred | - | - | Mutual reversal without dispute is post-MVP |
| DOM-035 | P0 | Audit coupling | built | `internal/service/audit.go` (audit) | `internal/service/atomicity_test.go:TestEveryCommandWritesAuditEvent` | Audit event in the same unit for each state change |
| DOM-036 | P1 | Administrative reversal | partial | `internal/service/reversal.go` (ReverseTransaction) | `internal/service/reversal_test.go:TestReversalAuthorization`<br>`internal/service/reversal_test.go:TestReversalBlockedByLimit` | Admin reversal built; requires completed status, disputed only via dispute (DEC-052) |
| DOM-037 | P0 | Administrators cannot act on their own transactions | built | - | `internal/service/reversal_test.go:TestReversalAuthorization`<br>`internal/service/membership_test.go:TestAdministratorSuccession` | Admin cannot act on own account or own trades |
| DOM-038 | P0 | Reviewer independence | partial | `internal/service/disputes_decide.go` (ResolveDispute) | `internal/service/dispute_test.go:TestDisputeDecisionAuthorization` | Party and conflict refusal built; admin without reviewer role may decide (DEC-072) |
| DOM-039 | P0 | Suspended members | built | - | `internal/service/membership_test.go:TestSuspension`<br>`internal/service/trades_test.go:TestDeclineCancelExitRules` | Suspended members may decline, cancel and view; cannot propose or accept |
| DOM-040 | P0 | Listings have no balance effect | built | - | `internal/service/trades_test.go:TestListingsDoNotTouchLedger` | Listings write no entries |

## 04 Functional requirements

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| FR-001 | P0 | Create a network | built | `internal/service/networks.go` (CreateNetwork)<br>`internal/web/h_auth.go` (setupPost) | `internal/service/admin_test.go:TestNetworkLifecycle` | Create network with policy; system account; audit |
| FR-002 | P0 | First administrator bootstrap | partial | `internal/web/h_auth.go` (setupPost)<br>`internal/web/h_auth.go` (secretEqual) | `internal/web/hardening_test.go:TestSetupSecretEnforced` | One-time setup secret; used flag is per process, so a restart re-enables it while the secret is set |
| FR-003 | P0 | Change configurable policies | built | `internal/service/networks.go` (UpdatePolicy)<br>`internal/web/h_admin2.go` (policyPost) | `internal/service/admin_test.go:TestPolicyChanges` | Policy change within bounds, versioned and audited |
| FR-004 | P0 | Network activation | built | `internal/service/networks.go` (ActivateNetwork) | `internal/service/admin_test.go:TestNetworkLifecycle` | Setup to active once an admin exists |
| FR-005 | P0 | Decimals fixed after first posting | built | `internal/service/networks.go` (UpdatePolicy) | `internal/service/admin_test.go:TestPolicyChanges` | Decimals fixed after first posting |
| FR-006 | P0 | Network isolation in every interface | built | - | `internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound`<br>`internal/api/authn_test.go:TestAuthorisation` | Cross-network identifier test covers web and API |
| FR-010 | P0 | Join by invitation | built | `internal/service/join.go` (CreateInvitation)<br>`internal/service/join.go` (RedeemInvitation)<br>`internal/web/h_admin.go` (invitationCreate)<br>`internal/web/h_auth.go` (joinInvitePost) | `internal/service/membership_test.go:TestInvitations` | Single-use invitation token hashed; redemption creates member |
| FR-011 | P0 | Join by approval | built | `internal/service/join.go` (DecideJoinRequest)<br>`internal/web/h_admin.go` (memberActionPost) | `internal/api/hardening_test.go:TestAPIDenyJoinRequest` | Approve and deny applicants; deny via API and web |
| FR-012 | P1 | Join by open link | partial | `internal/service/join.go` (JoinNetwork)<br>`internal/web/h_auth.go` (joinNetPost) | `internal/service/membership_test.go:TestJoinModes` | Open join creates active member; join rate limit per abuse baseline not built |
| FR-013 | P0 | Membership lifecycle states | built | - | `internal/domain/statemachine_test.go:TestMemberTransitionsExhaustive` | Membership transitions table-tested |
| FR-014 | P0 | Voluntary departure | built | `internal/service/departure.go` (DepartMember, PreviewDeparture)<br>`internal/web/h_gov.go` (leaveForm, leavePost) | `internal/service/admin_test.go:TestDepartureBlockersAndAdminOffboarding`<br>`internal/web/gov_h1_test.go:TestLeavePages` | Departure in service, JSON API and the web leave form |
| FR-015 | P0 | Administrative offboarding | built | `internal/web/h_admin.go` (memberActionPost) | `internal/service/admin_test.go:TestDepartureBlockersAndAdminOffboarding` | Admin removal with mandatory reason; last-admin guard |
| FR-016 | P1 | Member profile minimum | built | `internal/service/auth.go` (RegisterPerson) | - | Display name is the only required profile field |
| FR-017 | P0 | Duplicate join is idempotent | built | `internal/service/join.go` (JoinNetwork) | `internal/service/hardening_test.go:TestRejoinSemantics` | Duplicate join returns existing membership |
| FR-020 | P0 | Directory visible to active members only | built | `internal/service/membership.go` (ListDirectory)<br>`internal/service/member_page.go` (ListDirectoryPage) | `internal/service/membership_test.go:TestJoinModes` | Directory returns active members only; keyset paging and exact counts since 925f637 |
| FR-021 | P0 | Create a listing | built | `internal/service/listings.go` (CreateListing) | `internal/service/trades_test.go:TestListingsDoNotTouchLedger` | Create offer or request listing; no entries |
| FR-022 | P0 | Edit, pause, archive own listing | built | `internal/service/listings.go` (UpdateListing)<br>`internal/web/h_home.go` (listingEdit) | `internal/api/more_test.go:TestPolicyAndListingETags` | Edit, pause, resume, archive own listing |
| FR-023 | P0 | Search and browse listings | built | `internal/service/listings.go` (ListListings)<br>`internal/web/h_home.go` (listings) | `internal/api/more_test.go:TestCursorPagination` | Filter by type and text; paginated |
| FR-024 | P1 | Report a listing or member | not built | - | - | Member reports not built |
| FR-025 | P1 | Members-only visibility of listings | partial | `internal/api/route.go` (wrap) | - | Members-only pages require session; no-index directive absent |
| FR-030 | P0 | Propose a trade | built | `internal/service/transactions.go` (ProposeTransaction)<br>`internal/web/h_trades.go` (proposePost) | - | Propose to active member in same network |
| FR-031 | P0 | Proposal validation and effect preview | built | `internal/service/transactions.go` (ProposeTransaction)<br>`internal/web/templates/propose_review.html` | - | Amount, limit and effect preview on review page |
| FR-032 | P0 | Accept and post a trade atomically | built | `internal/service/transactions.go` (AcceptTransaction) | `internal/service/concurrency_test.go:TestParallelAcceptsNeverBreachCreditLimit` | Accept posts both entries atomically; limit re-checked |
| FR-033 | P0 | Decline and withdraw | built | `internal/service/transactions.go` (DeclineTransaction)<br>`internal/service/transactions.go` (CancelTransaction) | - | Decline and withdraw release reservation |
| FR-034 | P0 | Proposal expiry | built | `internal/service/transactions.go` (ExpireProposals)<br>`cmd/circula/serve.go` (expire) | `internal/service/trades_test.go:TestExpiry` | Scheduled expiry, idempotent, injected clock |
| FR-035 | P0 | Idempotent propose, accept and other commands | built | `internal/service/idempotency.go` (idempotent) | `internal/service/concurrency_test.go:TestIdempotentConcurrentPropose` | Idempotency on propose, accept, decline, cancel, reverse |
| FR-036 | P0 | Concurrent acceptance safety | built | - | `internal/service/concurrency_test.go:TestParallelDoubleAcceptSingleSuccess`<br>`internal/service/concurrency_test.go:TestParallelAcceptsNeverBreachCreditLimit` | Concurrent accepts for last credit: one succeeds |
| FR-037 | P1 | Note and listing reference on a trade | built | `internal/service/transactions.go` (ProposeTransaction) | - | Plain-text note and listing reference on proposal |
| FR-038 | P1 | In-app notifications and inbox | partial | `internal/web/templates/dashboard.html`<br>`internal/service/reads.go` (ListPendingForActor) | - | Dashboard lists pending items; no inbox, unread state or outcome notices |
| FR-050 | P0 | Balance and available credit view | built | `internal/service/accounts.go` (GetBalance)<br>`internal/web/format.go` (makeCard) | - | Balance, limit, available, reserved and time shown |
| FR-051 | P0 | Transaction history | built | `internal/service/tx_queries.go` (ListTransactions)<br>`internal/web/templates/history.html` | - | Reverse-chronological history with status and counterparty |
| FR-052 | P0 | Balance integrity check | built | `internal/service/verify.go` (VerifyLedger)<br>`internal/web/h_admin2.go` (ledgerCheck)<br>`cmd/circula/commands.go` (cmdVerifyLedger) | `internal/service/hardening_test.go:TestVerifyLedgerIsReadOnly` | Read-only ledger verification from admin page and CLI |
| FR-053 | P1 | Pending items view | built | `internal/service/reads.go` (ListPendingForActor) | `internal/service/hardening_test.go:TestListPendingForActorPagesBeyondOneHundred` | Pending view paged; dashboard shows 100 newest pending |
| FR-054 | P1 | Balance at a past date | partial | `internal/service/accounts.go` (BalanceAt) | `internal/service/audit_test.go:TestVerifyDetectsBalanceCorruptionAndBalanceAt` | BalanceAt in service only; no UI or API |
| FR-060 | P0 | Set a member's credit limit override | built | `internal/service/membership.go` (SetCreditLimit)<br>`internal/api/h_accounts.go` (putCreditLimit) | `internal/api/more_test.go:TestCreditLimitAndMemberAdmin` | Set or remove override with reason; audited |
| FR-061 | P0 | Suspend and restore a member | built | `internal/service/membership.go` (SuspendMember)<br>`internal/service/membership.go` (RestoreMember)<br>`internal/web/h_admin.go` (memberActionPost) | `internal/service/membership_test.go:TestSuspension` | Suspend and restore with reason; audited |
| FR-062 | P0 | Administrative adjustment | built | `internal/service/adjustments.go` (CreateAdjustment)<br>`internal/web/h_admin2.go` (adjustPost) | `internal/service/admin_test.go:TestAdjustments` | Admin adjustment with reason; two entries |
| FR-063 | P0 | Manage invitations and join requests | built | `internal/service/join.go` (ListInvitations)<br>`internal/service/join.go` (RevokeInvitation)<br>`internal/web/h_admin.go` (invitationRevoke) | - | List, revoke invitations; approve or deny join requests |
| FR-064 | P1 | Default flag | partial | `internal/service/membership.go` (SetDefaultFlag)<br>`internal/web/h_admin.go` (memberActionPost) | - | Default flag in web admin only; no API route |
| FR-065 | P1 | Assign dispute reviewers | built | `internal/service/membership.go` (AssignRole)<br>`internal/api/h_admin.go` (roleAction)<br>`internal/service/membership.go` (changeRole) | - | Assign or remove reviewer; reviewer cannot review own dispute |
| FR-066 | P1 | Administrative reversal | built | `internal/service/reversal.go` (ReverseTransaction)<br>`internal/api/h_admin.go` (adminReverse) | `internal/service/reversal_test.go:TestReversalAuthorization` | Admin reversal via API with reason |
| FR-067 | P1 | Administrator overview | partial | `internal/web/h_admin.go` (adminHome) | - | Admin home shows pending-member count only |
| FR-068 | P1 | Network closure | partial | `internal/service/networks.go` (CloseNetwork) | - | CloseNetwork refuses with blockers; no web or API caller |
| FR-069 | P0 | Administrator succession | built | `internal/service/membership.go` (changeRole) | `internal/service/membership_test.go:TestAdministratorSuccession` | Last-admin guard; role changes audited |
| FR-070 | P0 | Open a dispute | built | `internal/service/disputes.go` (OpenDispute)<br>`internal/web/h_disputes.go` (disputePost) | `internal/service/dispute_test.go:TestOpenDisputeRules` | Open dispute within window by party on completed trade |
| FR-071 | P0 | Dispute does not change postings | built | - | `internal/service/dispute_test.go:TestOpenDisputeWindowAndNoPostings`<br>`internal/service/dispute_test.go:TestWithdrawDispute` | Postings unchanged on open, withdraw, dismiss |
| FR-072 | P0 | Reviewer independence | built | - | `internal/service/dispute_test.go:TestDisputeDecisionAuthorization` | Party and proposer conflicts refused |
| FR-073 | P0 | Decide a dispute | built | `internal/service/disputes_decide.go` (ResolveDispute)<br>`internal/web/h_disputes.go` (disputeAct) | `internal/service/dispute_test.go:TestResolveOutcomes` | Dismissed, upheld, partially upheld with rationale |
| FR-074 | P0 | Upheld reversal blocked by the limit | built | `internal/service/disputes_decide.go` (RetryDisputePosting) | `internal/service/dispute_test.go:TestUpheldReversalBlockedThenOverride` | Blocked reversal enters posting_pending; override then retry |
| FR-075 | P1 | Withdraw a dispute | built | `internal/service/disputes.go` (WithdrawDispute) | `internal/service/dispute_test.go:TestWithdrawDispute` | Opener withdraws before decision |
| FR-076 | P1 | Response by the other party | not built | - | - | Response by other party and response period not built |
| FR-077 | P0 | Dispute window enforcement and deadlines | built | `internal/service/disputes.go` (OpenDispute) | `internal/service/dispute_test.go:TestOpenDisputeRules` | Window measured from completion with server clock |
| FR-078 | P1 | One review of a decision | not built | - | - | Re-review of a decision not built |
| FR-080 | P0 | Every state change is audited | built | - | `internal/service/atomicity_test.go:TestEveryCommandWritesAuditEvent` | Sweep: every command writes one audit event |
| FR-081 | P0 | Audit event content | built | `internal/service/audit.go` (audit) | `internal/service/audit_test.go:TestAuditEventContent` | Audit event carries required fields |
| FR-082 | P0 | Audit visibility by role | partial | `internal/service/audit_read.go` (ListAudit)<br>`internal/web/h_admin2.go` (auditPage) | `internal/service/audit_test.go:TestAuditVisibilityForMembers` | Admin reads full audit; member and reviewer views exist in service only |
| FR-083 | P0 | Audit entries are append-only | built | - | `internal/service/audit_test.go:TestAppendOnlyThroughRawAccess` | No update or delete path; triggers |
| FR-084 | P1 | Audit search and filter | built | `internal/service/audit_read.go` (ListAudit)<br>`internal/web/h_admin2.go` (auditPage)<br>`internal/api/h_ops.go` | - | Admin audit filters by action, actor, target, dates; paged |
| FR-085 | P2 | Tamper-evident audit chain | built | `internal/service/audit.go` (AuditHash) | `internal/service/audit_test.go:TestAuditChainDetectsTampering`<br>`internal/service/hardening_test.go:TestAuditTailTruncationDetected` | Hash chain built ahead of P2 deferral (DEC-044); not keyed |
| FR-090 | P0 | Member exports own history | built | `internal/service/export.go` (Export)<br>`internal/web/h_admin2.go` (exportMine) | `internal/service/export_test.go:TestMemberExportIsScoped` | Member exports own history as CSV or JSON |
| FR-091 | P0 | Admin exports the network ledger | built | `internal/service/export.go` (Export)<br>`internal/web/h_admin2.go` (exportsPost) | `internal/service/export_test.go:TestLedgerExportCSV` | Admin ledger export with manifest and checksums |
| FR-092 | P0 | Admin exports the member list | built | `internal/service/export.go` (ExportMembers)<br>`internal/service/export.go` (Export) | - | Members export scope with role and limit fields |
| FR-093 | P0 | Admin exports the audit history | built | `internal/service/export.go` (ExportAudit)<br>`internal/service/export.go` (Export) | - | Audit export scope with manifest |
| FR-094 | P1 | Full network export for closure or portability | built | `internal/service/export.go` (ExportFull)<br>`internal/service/export.go` (Export) | `internal/service/export_test.go:TestFullExportJSON` | Full export scope incl. closed networks |
| FR-095 | P1 | Export correctness and performance gates | partial | `internal/service/export.go` (MemSink)<br>`internal/api/h_export.go` (createExport) | - | Export buffered in memory by API adapter; no streaming or size gates |

## 05 Architecture

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| ARC-001 | P0 | Single binary for both deployment modes | built | `scripts/package-release.sh`<br>`cmd/circula/main.go` (main) | - | One executable per target; no edition split |
| ARC-002 | P0 | Domain service layer is the only write path | partial | `internal/service/service.go` (Service)<br>`internal/store/sqlite.go` (Update)<br>`cmd/circula/commands.go` (cmdAdmin) | - | Service is the write path; cmd reads SQLite directly for some checks |
| ARC-003 | P0 | Adapters contain no business rules | built | `internal/service/verify.go` (VerifyLedger)<br>`internal/service/posting.go` (checkCapacity) | - | Adapters map and render; rules in service |
| ARC-004 | P0 | Storage behind a repository interface | partial | `internal/store/sqlite.go` (Open)<br>`internal/store/iface.go` | - | Store interface exists but lives beside SQLite; no contract suite for another store |
| ARC-005 | P0 | Ledger component owns posting | built | `internal/service/posting.go` (postTx)<br>`internal/store/sqlite_account.go` (AddBalance) | - | Only postTx writes entries and balances |
| ARC-006 | P0 | Negative-balance check is serialised per account | built | `internal/service/posting.go` (checkCapacity) | `internal/service/concurrency_test.go:TestParallelAcceptsNeverBreachCreditLimit` | Per-account serialisation via write gate and BEGIN IMMEDIATE |
| ARC-007 | P0 | Network tenant isolation | built | - | `internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound` | Network scoping in queries and tests |
| ARC-008 | P0 | Platform and network governance are separate | partial | `cmd/circula/commands.go` (cmdRestore)<br>`cmd/circula/commands.go` (cmdAdmin) | - | No operator API route; host-level CLI restore and SQL access remain outside service |
| ARC-009 | P1 | Operator access to network data is break-glass only | not built | - | - | Break-glass access not built |
| ARC-010 | P0 | Trust boundaries are enumerated and enforced | partial | - | `cmd/circula/ops_test.go:TestBodyLimitApplied`<br>`internal/api/strict_test.go:TestStrictBodyHandling` | Input limits and tests exist; trust-boundary checklist and fuzzing not run |
| ARC-011 | P0 | Concurrency model is explicit | built | `cmd/circula/lock_other.go` (acquireLock) | `cmd/circula/backup_test.go:TestLockRefusesSecondHolder`<br>`cmd/circula/hardening_test.go:TestOperatorCommandsTakeTheLock` | One writer per database file; lock refuses second holder |
| ARC-012 | P1 | Background jobs are idempotent and restart-safe | built | `cmd/circula/serve.go` (expire)<br>`cmd/circula/serve.go` (purge) | `cmd/circula/jobs_regress_test.go:TestJobExpiryMovesOnlyElapsedProposals`<br>`internal/service/hardening_test.go:TestPurge` | Expiry and purge idempotent; export artefacts not resumable |
| ARC-013 | P0 | Deployment modes are both supported in the MVP | built | `cmd/circula/serve.go` (runServe) | - | Same artefact for self-hosted and multi-network use; hosted launch not done |
| ARC-014 | P2 | Single-process MVP; horizontal scaling is deferred | built | `cmd/circula/lock_other.go` (acquireLock) | - | Single process constraint documented and enforced by lock (P2 item) |
| ARC-015 | P0 | Correctness is verified by invariant checks | partial | `internal/service/verify.go` (VerifyLedger)<br>`internal/service/verify.go` (VerifyAndRecordLedger) | `internal/service/audit_test.go:TestVerifyDetectsBalanceCorruptionAndBalanceAt` | Verify on demand and in hourly job; scheduled runs are not recorded (SVC-01) |
| ARC-016 | P1 | Availability and recoverability targets | partial | `cmd/circula/snapshot.go` (snapshot)<br>`cmd/circula/serve.go` (shutdownGracePeriod) | `cmd/circula/backup_test.go:TestSnapshotWritesVerifiedManifestAndRestores` | Online backup and restore built; RPO and RTO not recorded (DEC-016); shutdown grace is 30 s, spec says 10 s |
| ARC-017 | P0 | Accessibility is a build property of the HTML adapter | partial | - | `internal/web/a11y_test.go:TestPageStructure`<br>`internal/web/a11y_test.go:TestFormLabelsAndAutocomplete` | Semantic server-rendered HTML; no automated WCAG tool in CI |
| ARC-018 | P1 | Maintainability through enforced module boundaries | partial | `.github/workflows/ci.yml` | - | Layering by convention; no CI import-rule check |
| ARC-019 | P0 | Portability across release targets | built | `scripts/package-release.sh`<br>`.github/workflows/ci.yml` | - | CI builds all five targets from one commit |
| ARC-020 | P0 | Data exportability is an architectural property | partial | `internal/service/export.go` (Export)<br>`internal/service/export.go` (ExportFull) | - | Export through service; documented reference parser absent |
| ARC-021 | P1 | Domain portability | built | `internal/domain/types.go` (IDGen)<br>`internal/domain/types.go` (Clock) | - | Clock and IDs injected into service |
| ARC-022 | P1 | Observability without sensitive content | partial | `cmd/circula/metrics.go` (newMetrics)<br>`internal/service/purge.go` (RedactSecrets) | `cmd/circula/ops_test.go:TestRequestIDAndRoutesMounted` | Structured logs, request IDs, metrics; posting counters not present |
| ARC-023 | P1 | Configuration and secrets | partial | `cmd/circula/config.go` (loadSecrets) | `cmd/circula/config_test.go:TestPrecedence`<br>`cmd/circula/hardening_test.go:TestSecretsLoadingAndRedaction` | Config from file, env and flags; secret files; no repository secret scan in CI |
| ARC-024 | P1 | Dependencies are minimal and reviewed | not built | `go.mod` | - | No dependency licence manifest |
| ARC-025 | P2 | Extension points are deferred | deferred | `internal/api/routes.go` | - | Federation, mobile, payment and tokenisation not built; no route |

## 06 Data, ledger and audit

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| DAT-001 | P0 | Primary identifiers for all entities MUST be UUIDv7 in canonical 36-character lowercase te | built | `internal/domain/types.go` (IDGen) | - | Identifiers are UUIDv7 text |
| DAT-002 | P0 | Identifiers MUST be generated by the domain layer through an injected generator (ARC-021), | built | `internal/domain/types.go` (IDGen) | - | IDs from injected generator |
| DAT-003 | P0 | Identifiers MUST NOT encode personal data, network names, or sequence values that reveal m | built | `internal/domain/types.go` (IDGen) | - | IDs carry no personal data |
| DAT-004 | P1 | Identifiers exposed in URLs and APIs MUST be the same as internal identifiers, so that the | built | `internal/httpx/httpx.go` (SessionToken)<br>`internal/service/purge.go` (RedactSecrets) | - | Invitation and session tokens random and stored hashed |
| DAT-005 | P0 | Every network-scoped table MUST have a non-null network id and a foreign key (or equivalen | built | `internal/store/migrations/0001_init.sql` | `internal/store/sqlite_test.go:TestCrossNetworkReferencesRejected` | Composite foreign keys with network id |
| DAT-006 | P0 | A LedgerEntry MUST reference exactly one Transaction and exactly one Account, both in the  | built | `internal/store/sqlite_tx.go` (InsertEntry) | `internal/store/sqlite_test.go:TestTransactionIntegrityTriggers` | Entry references one transaction and one account in same network |
| DAT-007 | P0 | A Transaction of kind `trade`, `adjustment`, or `reversal` MUST have exactly two LedgerEnt | built | `internal/service/posting.go` (postTx) | `internal/store/sqlite_test.go:TestTransactionIntegrityTriggers` | Two entries, equal and opposite, for trade, adjustment, reversal |
| DAT-008 | P0 | `amount_minor` MUST be an integer greater than zero. Signed amounts on entries MUST be non | built | `internal/store/migrations/0001_init.sql` | `internal/domain/domain_test.go:TestMigrationsAvoidFloatColumns` | Integer amounts with CHECK constraints |
| DAT-009 | P0 | Account balances per network MUST sum to zero, including the system account (LED-1). | built | `internal/service/verify.go` (VerifyLedger)<br>`internal/store/sqlite_verify.go` (BalanceMismatches)<br>`internal/store/sqlite_verify.go` (Totals) | - | Network sum checked by verify |
| DAT-010 | P0 | LedgerEntry, TransactionStatusEvent, AuditEvent, DisputeEvent and ConsistencyCheckRun rows | built | - | `internal/store/sqlite_test.go:TestTriggersBlockUpdateDeleteOnAppendOnlyTables` | Triggers block update and delete on append-only tables |
| DAT-011 | P0 | Enforcement of DAT-010 MUST be tested, including attempts through the repository layer and | built | - | `internal/service/audit_test.go:TestAppendOnlyThroughRawAccess` | Enforcement tested via raw SQL |
| DAT-012 | P1 | Reversal transactions MUST reference the original transaction; a transaction MUST NOT be r | partial | `internal/service/reversal.go` (ReverseTransaction)<br>`internal/store/sqlite_verify.go` (ReversalMismatches) | - | One reversal per original not enforced; cumulative cap instead (DEC-040) |
| DAT-013 | P1 | Each adjustment MUST carry a non-empty reason (maximum length to be set in 03) and MUST re | built | `internal/service/adjustments.go` (CreateAdjustment) | - | Adjustment reason required; counter-account is system account |
| DAT-014 | P0 | Posting a transaction (LED-4) MUST insert its status event, both ledger entries, update bo | built | `internal/service/posting.go` (postTx) | `internal/store/sqlite_test.go:TestUpdateRollsBackOnError` | Posting writes status, entries, balances, audit in one transaction |
| DAT-015 | P0 | Before posting, the storage transaction MUST read the current cached balance of every affe | built | `internal/service/posting.go` (checkCapacity) | - | Limit check under write gate before posting |
| DAT-016 | P0 | Each committed posting batch receives a `ledger_seq` value from a per-network counter that | built | `internal/service/reads.go` (ListEntries)<br>`internal/store/sqlite_network.go` (NextLedgerSeq) | `internal/api/hardening_test.go:TestEntriesOrderedByLedgerSeq` | ledger_seq per network; history ordered by it |
| DAT-017 | P0 | Times MUST be stored in UTC with at least millisecond precision. Display time zones are a  | built | `internal/store/sqlite.go` (fromMS)<br>`internal/store/sqlite.go` (msPtr) | - | Times stored as UTC milliseconds |
| DAT-018 | P0 | `available_minor` for a payer MUST be computed as `cached_balance_minor + credit_limit_min | built | `internal/store/sqlite_account.go` (ReservedOutgoing)<br>`internal/store/sqlite_account.go` (sumProposed) | - | Available computed from balance, limit and proposed reservations |
| DAT-019 | P0 | Cached balances MUST be updated only inside a posting (DAT-014) or a documented repair (se | built | `internal/store/sqlite_account.go` (AddBalance) | - | Cache changed only in posting |
| DAT-020 | P0 | The consistency check (ARC-015) MUST verify, per network, that the sum of entries per acco | built | `internal/service/verify.go` (VerifyLedger)<br>`internal/store/sqlite_verify.go` (BalanceMismatches)<br>`internal/store/sqlite_verify.go` (TxProblems) | - | Verify checks per-account, per-transaction and network sums |
| DAT-021 | P1 | Balance history for an account MUST be reconstructable at any `ledger_seq` by summing entr | built | `internal/service/accounts.go` (BalanceAt) | - | BalanceAt sums entries up to a sequence (service only) |
| DAT-022 | P0 | A reversal creates a new transaction of kind `reversal` with entries equal and opposite to | built | `internal/service/reversal.go` (ReverseTransaction) | `internal/service/reversal_test.go:TestFullReversal` | Reversal is new transaction; original unchanged |
| DAT-023 | P1 | An adjustment creates a new transaction of kind `adjustment` posting equal and opposite en | built | `internal/service/adjustments.go` (CreateAdjustment) | - | Adjustment against system account with audit |
| DAT-024 | P1 | A reversal or adjustment that would take a non-system account below its negative limit MUS | partial | `internal/service/membership.go` (SetCreditLimit) | `internal/service/trades_test.go:TestLimitOverrideExpires` | Override path is time-limited audited limit change; no general admin override flag |
| DAT-025 | P0 | An IdempotencyRecord MUST be written in the same storage transaction as the state change i | built | `internal/service/idempotency.go` (idempotent)<br>`internal/store/sqlite_audit.go` (PutIdem) | - | Idempotency record written in same unit |
| DAT-026 | P0 | The record key MUST be unique over (network id, actor id, operation, key). Concurrent dupl | built | `internal/store/sqlite_audit.go` (GetIdem)<br>`internal/service/idempotency.go` (idempotent) | - | Record key over network, person, operation and key |
| DAT-027 | P0 | The record MUST store a fingerprint of the normalised request body. Reuse of a key with a  | built | `internal/service/idempotency.go` (idempotent) | `internal/service/idempotency_test.go:TestIdempotencyKeyReusedWithDifferentPayload` | Fingerprint of request; reuse refused |
| DAT-028 | P1 | Idempotency records MUST expire after the retention period in section 9. An expired record | built | `internal/store/sqlite_audit.go` (DeleteExpiredIdem) | `internal/service/idempotency_test.go:TestIdempotencyRecordExpires` | Records expire after TTL (7 days) |
| DAT-029 | P0 | Audit events MUST be append-only (DAT-010) and MUST be written in the same storage transac | built | `internal/service/audit.go` (audit) | - | Audit in same unit as change |
| DAT-030 | P1 | Each network's audit chain SHOULD be tamper-evident: each event stores a hash of its conte | built | `internal/service/audit.go` (AuditHash) | `internal/service/audit_test.go:TestAuditChainDetectsTampering` | Per-network hash chain with head; unkeyed (DEC-044) |
| DAT-031 | P0 | An audit event MUST NOT contain passwords, session identifiers, token values, invitation t | built | - | `internal/service/audit_test.go:TestAuditEventContent` | Audit excludes secrets and tokens |
| DAT-032 | P1 | Audit events MUST be exportable in the same format as the ledger export (section 8). | partial | `internal/service/export.go` (ExportAudit) | - | Audit export scope exists; format identical to ledger export not checked |
| DAT-033 | P0 | Members MAY see audit events that concern their own transactions and accounts, with actor  | partial | `internal/service/audit_read.go` (ListAudit) | `internal/service/audit_test.go:TestAuditVisibilityForMembers` | Member view of own audit exists in service; no member UI or API |
| DAT-034 | P0 | Network administrators MUST be able to read the network's full audit history. Access MUST  | built | `internal/service/audit_read.go` (ListAudit) | `internal/service/authz_test.go:TestAdminBalanceViewIsAudited` | Admin audit read is itself audited |
| DAT-035 | P0 | Dispute reviewers MUST be able to read audit events that concern disputes they are assigne | partial | `internal/service/audit_read.go` (ListAudit) | - | Reviewer audit scope in service; no reviewer UI |
| DAT-036 | P0 | Platform operators MUST NOT read audit events except through the break-glass path in ARC-0 | built | `internal/service/audit_read.go` (ListAudit) | - | No operator read path to audit |
| DAT-037 | P0 | Exports MUST use documented, open, machine-readable formats. The MVP format set is: CSV fo | built | `internal/service/export.go` (ManifestFile) | `internal/service/export_test.go:TestLedgerExportCSV` | CSV and JSON export with manifest and SHA-256 |
| DAT-038 | P0 | Exports MUST include ledger entries, transactions, status history, accounts with cached ba | built | `internal/store/sqlite_export.go` (exportQuery)<br>`internal/store/sqlite_export.go` (StreamTable) | - | Ledger export includes transactions, status, accounts, entries, audit |
| DAT-039 | P0 | An export MUST be taken from a consistent snapshot of one `ledger_seq` value and the manif | built | `internal/service/export.go` (Export)<br>`internal/service/export.go` (ManifestFile) | - | Export manifest states ledger_seq of one snapshot |
| DAT-040 | P0 | A checksum mismatch or row count mismatch detected during verification MUST fail the expor | partial | `internal/service/export.go` (ManifestFile) | - | Manifest checksums exist; no verifier command for export packages |
| DAT-041 | P1 | The export format MUST be documented in a versioned schema description, and a reference re | not built | - | - | No versioned export schema document or reference reader |
| DAT-042 | P0 | Schema migrations MUST be forward-only, versioned, and applied in order with a recorded ve | built | `internal/store/migrate.go` (loadMigrations)<br>`internal/store/migrate.go` (migrate) | `internal/store/sqlite_test.go:TestMigrationsIdempotentAndVersioned` | Forward-only embedded migrations with recorded version |
| DAT-043 | P0 | Before any migration on a non-empty database, a backup MUST be taken and verified. | built | `cmd/circula/serve.go` (preMigrationBackup) | - | Pre-migration backup before migration on non-empty database |
| DAT-044 | P0 | After every migration, the consistency check (ARC-015, DAT-020) MUST pass before the appli | built | `cmd/circula/serve.go` (checkPragmas)<br>`internal/service/verify.go` (VerifyLedger)<br>`cmd/circula/commands.go` (cmdMigrate) | - | Post-migration verification gates serving |
| DAT-045 | P1 | Schema changes SHOULD follow expand-then-contract: add new structures, backfill, switch re | partial | `internal/store/migrate.go` (loadMigrations) | - | Expand-then-contract is a practice; not enforced by tests |
| DAT-046 | P0 | Migrations MUST NOT modify rows of append-only tables, except a migration that only adds n | built | `internal/store/migrations/0002_hardening.sql` | `internal/store/hardening_test.go:TestMigration0002BackfillsAuditHeads` | Migrations do not change ledger rows (0002 backfills audit heads only) |
| DAT-047 | P0 | Each release MUST be tested with a migration from the previous released schema, using a fi | not built | - | - | No 10,000-transaction fixture and no migration from prior released schema |
| DAT-048 | P1 | Migration duration on the fixture MUST be recorded per release so operators can estimate d | not built | - | - | Migration duration not recorded |

## 07 API contract (owned by the 07 author; status recorded here, not edited there)

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| API-001 | P0 | The HTML adapter and the JSON API MUST call the same domain services. Each MVP operation i | built | `internal/api/h_tx.go` (propose)<br>`internal/api/h_tx.go` (acceptTx)<br>`internal/service/transactions.go` (ProposeTransaction) | - | Adapters call the same service methods |
| API-002 | P0 | The JSON API MUST be served under the path prefix `/api/v1` of a network-scoped deployment | built | `internal/api/routes.go` (v1)<br>`internal/api/routes.go` | - | Routes under /api/v1 with network context |
| API-003 | P0 | Requests and responses MUST use `application/json; charset=utf-8`. A request with a body M | built | - | `internal/api/strict_test.go:TestStrictBodyHandling` | JSON content type enforced on bodies |
| API-004 | P0 | Money values MUST be JSON strings containing a canonical base-10 integer of minor units (f | built | `internal/api/decode.go` (parseMinor) | `internal/api/strict_test.go:TestStrictBodyHandling` | Money as strings of minor units; numbers rejected |
| API-005 | P0 | Timestamps MUST be RFC 3339 in UTC with a `Z` suffix and millisecond precision. Identifier | built | - | `internal/api/more_test.go:TestCursorPagination` | RFC 3339 UTC; UUIDv7 ids |
| API-006 | P0 | Resource names MUST be plural nouns. `GET` MUST NOT change state. `POST` creates a resourc | built | `internal/api/routes.go` | - | Plural resource names; GET has no side effects |
| API-007 | P0 | Balance views MUST include the derived field `direction` with value `in_credit` (balance > | built | `internal/api/dto.go` | - | direction field on balance views |
| API-008 | P0 | Browser clients MUST authenticate with a session cookie issued by the sign-in flow (HTML). | built | - | `internal/api/hardening_test.go:TestWebSessionWorksForAPI`<br>`internal/api/hardening_test.go:TestSecureSessionCookieWorksForAPI` | Web session cookie accepted by API |
| API-009 | P1 | Non-browser clients MUST authenticate with a personal API token presented as `Authorizatio | built | `internal/api/authn_test.go` | `internal/api/authn_test.go:TestTokenScopesAndRevocation` | Bearer API tokens |
| API-010 | P1 | Each token MUST have a name, one or more scopes from the set `read` (read-only operations  | partial | `internal/api/h_core.go` (notImpl) | `internal/api/authn_test.go:TestTokenScopesAndRevocation` | Read and write scopes built; admin scope returns 501 |
| API-011 | P1 | A token MUST NOT authorise token creation, sign-in, or changes to a token's own scopes. A  | built | `internal/api/h_tokens.go` (createToken)<br>`internal/api/routes.go` | - | Token cannot create tokens; creation needs session |
| API-012 | P0 | Credentials MUST NOT be accepted in query strings. A request carrying a session identifier | built | - | `internal/api/authn_test.go:TestCredentialsInQueryRejected` | Credentials in query string rejected |
| API-013 | P0 | Unauthenticated requests to non-public operations MUST return 401 with `WWW-Authenticate`  | built | - | `internal/api/authn_test.go:TestUnauthenticatedIs401WithChallenge` | 401 with challenge; 403 for unauthorised |
| API-014 | P0 | Cookie-authenticated requests with unsafe methods (POST, PATCH, PUT, DELETE) MUST include  | built | - | `internal/api/authn_test.go:TestCookieSessionNeedsCSRF` | CSRF header required for cookie unsafe requests |
| API-015 | P0 | Cross-origin browser requests MUST NOT receive credentialed CORS responses. The API MUST N | built | `internal/web/middleware.go` | - | No CORS headers emitted |
| API-016 | P0 | Every operation MUST check the caller's membership status and role against the operation's | built | - | `internal/api/authn_test.go:TestAuthorisation` | Membership and role checked per operation |
| API-017 | P0 | A request for a resource that exists in another network, or that the caller is not permitt | built | - | `internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound` | Foreign and invisible ids return not-found |
| API-018 | P0 | Admin operations MUST require the network administrator role on the caller's current membe | partial | `internal/api/h_disputes.go` (resolveDispute)<br>`internal/api/h_disputes.go` (reviewDispute) | - | Reviewer operations accept admin without reviewer role (DEC-072) |
| API-019 | P0 | The platform operator role MUST NOT have any route under `/api/v1` that mutates network da | built | `internal/api/routes.go` | - | No operator route under /api/v1 mutates data |
| API-020 | P0 | Every command that changes ledger state, membership, or dispute state MUST accept an `Idem | built | `internal/api/idem.go` | `internal/service/idempotency_test.go:TestIdempotencyKeyRequiredAndScoped` | Idempotency-Key required on commands |
| API-021 | P0 | The key MUST be 1 to 255 printable ASCII characters (RFC 9110 field value rules). Clients  | built | `internal/api/idem.go` | `internal/api/hardening_test.go:TestIdempotencyKeyHeaderCode` | Key validated as printable ASCII (space accepted, OBS-5) |
| API-022 | P0 | A retry with the same key and an equivalent body MUST return the status and body of the or | built | - | `internal/service/idempotency_test.go:TestIdempotentRetryReturnsOriginal` | Replay returns original with Idempotent-Replayed |
| API-023 | P0 | Reusing a key with a different request body (after normalisation) MUST return 422 (`idempo | built | - | `internal/service/idempotency_test.go:TestIdempotencyKeyReusedWithDifferentPayload` | Reuse with different body returns 422 |
| API-024 | P0 | If a request with the same key is still in progress, the second request MUST return 409 (` | built | `internal/api/idem.go` | `internal/api/internal_test.go:TestPanicReleasesIdempotencyKey` | In-progress retry returns 409 with Retry-After |
| API-025 | P0 | A command that fails without changing state (for example, validation error or insufficient | built | - | `internal/service/idempotency_test.go:TestFailedCommandDoesNotConsumeKey` | Failed commands do not record key |
| API-026 | P0 | Responses for mutable resources with conflict risk (policy, listing, credit limit, members | partial | `internal/web/h_admin.go`<br>`internal/api/decode.go` | - | ETags on policy, listing, limit; membership actions optional for If-Match |
| API-027 | P0 | A mismatched `If-Match` MUST return 412 (`precondition-failed`) and no change. A missing ` | partial | `internal/api/decode.go`<br>`internal/api/h_admin.go` (memberAction) | - | 412 and 428 implemented; If-Match optional on member actions |
| API-028 | P0 | Transaction actions MUST be validated against the current status inside the atomic unit (0 | built | - | `internal/domain/statemachine_test.go:TestTxStateMachineExhaustive` | Action validated against current status in unit |
| API-029 | P0 | All error responses MUST use `application/problem+json` as defined by RFC 9457 with the me | built | `internal/api/respond.go` | `internal/api/hardening_test.go:TestInvalidHeaderTitleIsJSON` | problem+json responses |
| API-030 | P0 | Errors MUST include the extension members `code` (stable machine-readable code from the ca | built | `internal/api/respond.go` | `cmd/circula/ops_test.go:TestRequestIDAndRoutesMounted` | code and request_id in errors |
| API-031 | P0 | `type` MUST be a stable URN of the form `urn:circula:problem:<code>`, not a URL that must  | built | `internal/api/respond.go` | - | urn:circula:problem type |
| API-032 | P0 | Server errors (5xx) MUST NOT reveal internal details. The response MUST contain `code` `in | built | - | `cmd/circula/ops_test.go:TestPanicRecoveredAsInternalError` | 500 hides detail |
| API-033 | P1 | `detail` text MUST be in English in the MVP. Clients MUST use `code` for logic and MUST NO | built | `internal/api/respond.go` | - | English detail text |
| API-034 | P0 | List operations MUST use cursor pagination. Requests take `limit` (integer 1 to 100, defau | built | `internal/api/cursor.go` | `internal/api/more_test.go:TestCursorPagination` | Cursor pagination with limit |
| API-035 | P0 | Ordering MUST be stable and total: a unique sort key (for example `ledger_seq` or creation | built | - | `internal/api/hardening_test.go:TestEntriesOrderedByLedgerSeq` | Stable ordering by id or ledger_seq |
| API-036 | P1 | Total counts are not required. Listing endpoints MUST NOT run unbounded queries; an endpoi | built | `internal/api/cursor.go`<br>`internal/store/sqlite.go` (limitOf) | - | Lists bounded to 500 per page |
| API-037 | P0 | Request bodies MUST be limited to 64 KiB by default. Oversize bodies MUST be rejected with | built | `internal/api/decode.go` | `internal/api/strict_test.go:TestBodySizeLimit` | API body limit 64 KiB; server setting default 1 MiB (DEC-079) |
| API-038 | P1 | Rate limits MUST apply per authenticated principal and per client IP for unauthenticated a | partial | `internal/api/middleware.go` (newLimiter) | `internal/api/strict_test.go:TestRateLimit` | Per-client-IP limit (1200/min); per-principal limit not separate |
| API-039 | P0 | Responses containing balances, ledger entries, transactions, disputes, exports, or token m | built | `internal/web/middleware.go` | - | Cache-Control no-store on every response |
| API-040 | P0 | `GET /network` | built | `internal/api/h_core.go` (getNetwork) | - | GET network |
| API-041 | P0 | `GET /me` | built | `internal/api/h_core.go` (getMe) | `internal/api/more_test.go:TestInvitationsDisputesDepartureExport` | GET me |
| API-042 | P0 | `GET /members` | built | `internal/api/h_core.go` (listMembers)<br>`internal/service/member_page.go` (ListDirectoryPage) | - | Directory list active members, keyset-paged since 925f637 |
| API-043 | P0 | `GET /members/{id}` | built | `internal/api/h_core.go` (getMember) | `internal/api/hardening_test.go:TestSingleReadsByID` | Member detail, visibility rules |
| API-044 | P0 | `GET /accounts/me` | built | `internal/api/h_core.go` (getAccount) | - | Balance fields incl. direction and available |
| API-045 | P0 | `GET /accounts/me/entries` | built | `internal/api/h_core.go` (listEntries) | `internal/api/hardening_test.go:TestEntriesOrderedByLedgerSeq` | Entries ordered by ledger_seq |
| API-046 | P0 | `GET /listings`, `POST /listings` | built | `internal/api/h_listings.go` (listListings)<br>`internal/api/h_listings.go` (createListing) | - | Listings list and create |
| API-047 | P1 | `PATCH /listings/{id}` | built | `internal/api/h_listings.go` (patchListing) | `internal/api/strict_test.go:TestPatchRequiresMergePatchType` | PATCH listing with merge patch and If-Match |
| API-048 | P0 | `POST /transactions` | built | `internal/api/h_tx.go` (propose) | `internal/api/trade_test.go:TestFullTradeFlow` | Propose trade |
| API-049 | P0 | `GET /transactions`, `GET /transactions/{id}` | built | `internal/api/h_tx.go` (getTx)<br>`internal/api/h_tx.go` (listTx) | - | Transaction list and detail; non-party sees not-found |
| API-050 | P0 | `GET /transactions/{id}/events` | built | `internal/api/h_tx.go` (txEvents) | - | Status events endpoint |
| API-051 | P0 | `POST /transactions/{id}/accept` | built | `internal/api/h_tx.go` (acceptTx) | `internal/api/trade_test.go:TestFullTradeFlow` | Accept returns completed and posted_ledger_seq |
| API-052 | P0 | `POST /transactions/{id}/decline` | built | `internal/api/h_tx.go` (declineTx) | - | Decline |
| API-053 | P0 | `POST /transactions/{id}/cancel` | built | `internal/api/h_tx.go` (cancelTx) | - | Cancel by proposer |
| API-054 | P1 | `POST /disputes`, `GET /disputes`, `GET /disputes/{id}` | built | `internal/api/h_disputes.go` (openDispute)<br>`internal/api/h_disputes.go` (listDisputes)<br>`internal/api/h_disputes.go` (getDispute) | - | Disputes create, list, get |
| API-055 | P1 | `POST /disputes/{id}/resolve` | partial | `internal/api/h_disputes.go` (resolveDispute) | - | Resolve endpoint built; outcome names differ (dismissed, upheld, partially_upheld) |
| API-056 | P1 | `POST /admin/transactions/{id}/reversal` | partial | `internal/api/h_admin.go` (adminReverse) | - | Admin reversal endpoint; partial reversals allowed so second reversal rule differs (DEC-040) |
| API-057 | P1 | `POST /admin/adjustments` | built | `internal/api/h_admin.go` (adminAdjust) | - | Adjustment endpoint with reason |
| API-058 | P0 | `GET /admin/policy`, `PATCH /admin/policy` | built | `internal/api/h_policy.go` (getPolicy)<br>`internal/api/h_policy.go` (patchPolicy) | `internal/api/more_test.go:TestPolicyAndListingETags` | Policy GET and PATCH with ETag |
| API-059 | P0 | `GET /admin/members`, `POST /admin/members/{id}/approve`, `/deny`, `/suspend`, `/reinstate | built | `internal/api/h_admin.go` (adminListMembers)<br>`internal/api/h_admin.go` (memberAction) | - | Member list and approve, deny, suspend, reinstate, remove |
| API-060 | P0 | `PUT /admin/accounts/{id}/credit-limit` | built | `internal/api/h_accounts.go` (putCreditLimit) | `internal/api/more_test.go:TestCreditLimitAndMemberAdmin` | Credit limit PUT with If-Match and reason |
| API-061 | P1 | `POST /admin/invitations`, `POST /invitations/{token}/accept` | built | `internal/api/h_ops.go` (createInvitation)<br>`internal/api/h_ops.go` (acceptInvitation) | `internal/service/membership_test.go:TestInvitations` | Invitation create and accept |
| API-062 | P0 | `GET /admin/audit-events` | built | `internal/api/h_ops.go` (listAudit) | - | Audit list with filters |
| API-063 | P0 | `POST /admin/consistency-checks`, `GET /admin/consistency-checks/{id}` | built | `internal/api/h_ops.go` (runCheck)<br>`internal/api/h_ops.go` (getCheck) | - | Consistency check run and get; id is the stored run id |
| API-064 | P1 | `POST /admin/announcements`, `GET /admin/announcements` (and member read via `/network/ann | not built | `internal/api/h_core.go` (notImpl) | - | Announcements return 501 |
| API-065 | P1 | `POST /exports`, `GET /exports/{id}`, `GET /exports/{id}/artefact` | partial | `internal/api/h_export.go` (createExport)<br>`internal/api/h_export.go` (getArtefact) | `internal/api/hardening_test.go:TestExportCapPerPerson` | Export jobs built; artefacts held in memory and lost on restart |
| API-066 | P1 | `GET /tokens`, `POST /tokens`, `DELETE /tokens/{id}` | built | `internal/api/h_tokens.go` (listTokens)<br>`internal/api/h_tokens.go` (createToken)<br>`internal/api/h_tokens.go` (revokeToken) | `internal/service/auth_test.go:TestAPITokens` | Token list, create (session only), revoke |
| API-067 | P1 | `POST /me/membership/departure` | built | `internal/api/h_core.go` (depart) | - | Departure endpoint |
| API-068 | P0 | No API response MAY report a successful state change that did not commit. A success respon | built | `internal/api/h_tx.go` (acceptTx) | - | Success returned after commit |
| API-069 | P0 | Every write response that changes balances MUST return the resulting account view or the p | built | `internal/api/h_tx.go` (acceptTx) | - | Posting result includes posted_ledger_seq |
| API-070 | P0 | The API MUST NOT accept client-supplied balances, ledger entries, cached totals, `ledger_s | built | - | `internal/api/strict_test.go:TestStrictBodyHandling` | Client-supplied balances and ledger fields rejected |
| API-071 | P0 | Contract tests MUST run the same scenario set through the HTML and API surfaces and compar | partial | - | `internal/api/trade_test.go:TestFullTradeFlow` | Same service used by both surfaces; no shared scenario contract test |
| API-072 | P0 | An invariant violation found during a request MUST return 500 (`internal-error`), roll bac | built | `internal/api/respond.go` | - | Invariant violation returns 500 and rolls back |
| API-073 | P0 | The major version appears in the path (`/api/v1`). A breaking change requires a new major  | built | `internal/api/routes.go` (v1) | - | Version in path |
| API-074 | P0 | Non-breaking changes within v1 are: adding optional request fields, adding response fields | partial | - | - | Compatibility rules stated; no compatibility fixtures |
| API-075 | P0 | Breaking changes include: removing or renaming fields, changing types or the meaning of a  | partial | - | - | Rules stated; enforcement is review only |
| API-076 | P1 | When a v1 operation is deprecated, responses MUST include `Deprecation` (RFC 9745) and `Su | not built | - | - | No deprecated operations; deprecation headers not implemented |
| API-077 | P1 | The API MUST publish a machine-readable description (OpenAPI 3.1 recommended) per release, | partial | `docs/api/openapi.yaml` | - | OpenAPI 3.1 description published at docs/api/openapi.yaml (since a006a90); no contract test validates live responses against it |
| API-078 | P1 | Backward-compatibility tests MUST run each release against recorded requests from the prev | not built | - | - | No recorded requests from previous release |
| API-079 | - | Outbound webhooks or event streams | deferred | - | - | Webhooks deferred |
| API-080 | - | Third-party integrations and OAuth provider role | deferred | - | - | Third-party integrations deferred |
| API-081 | P2 | Language negotiation (`Accept-Language`) for error details | deferred | - | - | Accept-Language negotiation deferred |
| API-082 | - | GraphQL or other query-style interfaces | deferred | - | - | Query-style interface not planned |

## 08 UX and accessibility

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| UX-001 | P0 | Navigation MUST show only sections the current person's role permits. A hidden section MUS | built | `internal/web/templates/base.html` | `internal/web/security_test.go:TestAuthorisationAndIsolation` | Navigation by role; hidden pages return not-found or forbidden |
| UX-002 | P0 | Every MVP core screen in section 3 MUST be reachable from Home in no more than three activ | partial | `internal/web/templates/dashboard.html`<br>`internal/web/templates/base.html` | - | Home reachable in three activations not measured |
| UX-003 | P1 | A person who belongs to more than one network MUST see a network switcher, and each networ | built | `internal/web/templates/networks.html` | `internal/web/flow_test.go:TestLoginTradeFlow` | Network switcher and separate network pages |
| UX-004 | P0 | `In credit` when balance > 0 | built | `internal/web/format.go` (balanceVM)<br>`internal/web/format.go` (makeBalance) | `internal/web/format_test.go:TestUnitForPlural` | In credit, In debt, Level labels |
| UX-005 | P0 | Direction MUST be conveyed by the sign character and the word. Colour MAY reinforce but MU | partial | `internal/web/format.go` | - | Sign and words used; greyscale check not run |
| UX-006 | P0 | The accessible name of a balance MUST spell out the sign in words, for example "In debt by | built | `internal/web/format.go`<br>`internal/web/format.go` (balanceVM) | - | Accessible name spells out direction in words |
| UX-007 | P0 | Help text MUST explain the sign convention in plain language: "In credit" means the member | built | `internal/web/templates/help.html` | - | Help text explains sign convention |
| UX-008 | P0 | Amounts MUST be shown with the network's credit unit name and the decimal places in the ne | built | `internal/web/format.go` (unitFor) | `internal/web/format_test.go:TestUnitForPlural` | Network unit name and decimals; no currency symbols |
| UX-009 | P0 | The Home balance card MUST show, as separate labelled values: current balance with directi | built | `internal/web/format.go` (makeCard)<br>`internal/web/templates/dashboard.html` | - | Balance card shows balance, limit, reserved, available |
| UX-010 | P0 | Available to spend MUST NOT be presented as the balance. Its label MUST include the words  | built | `internal/web/templates/dashboard.html` | - | Available labelled as available to spend with help link |
| UX-011 | P0 | When the member is in debt, the card MUST show the portion of the credit limit in use as t | built | `internal/web/format.go`<br>`internal/web/format.go` (makeCard) | - | Text states credit in use against limit |
| UX-012 | P1 | When the member's in-use credit reaches the network's warning threshold (default proposed  | not built | - | - | Credit-use warning notice not built (OQ-UX-02) |
| UX-013 | P0 | Each transaction line MUST state the viewer's direction in words: "You paid Alex 5.00" (ba | built | `internal/web/format.go` (statusLabel)<br>`internal/web/format.go` | - | Direction in words per transaction line |
| UX-014 | P0 | Pending trades (status `proposed`) MUST be listed in a section labelled "Pending" and MUST | built | `internal/web/templates/dashboard.html` | - | Pending section excluded from balance |
| UX-015 | P0 | Completed trades MUST show "Completed" with the posting date and time (member's time zone, | built | `internal/web/templates/history.html` | - | Completed rows show posted time |
| UX-016 | P0 | Each status MUST have a text label and a one-line explanation in the detail view. The labe | built | `internal/web/format.go` (statusLabel)<br>`internal/web/format.go` (statusExplain) | - | Status labels and explanations; accepted not shown |
| UX-017 | P1 | Reversals and adjustments MUST be shown as separate lines linked to the original, with the | partial | `internal/web/templates/history.html` | - | Reversal and adjustment rows exist; link to original not verified |
| UX-018 | P0 | Accepting a trade, proposing a trade, creating a reversal or adjustment, and raising a dis | built | `internal/web/templates/propose_review.html`<br>`internal/web/templates/trade_confirm.html` | - | Review step before accept, propose, dispute |
| UX-019 | P0 | After a successful posting the confirmation message MUST state the new balance and directi | built | `internal/web/session.go` (setFlash)<br>`internal/web/h_trade_act.go` (tradeActionPost) | - | Confirmation message after posting |
| UX-020 | P0 | Refusals MUST state the cause in plain language and the fact that nothing changed, for exa | built | `internal/web/refusal.go` | - | Refusal copy names cause and that nothing changed |
| UX-021 | P0 | Validation errors MUST identify the field in text, describe the problem, and suggest a cor | partial | `internal/web/refusal.go` | - | Error summary and field messages present; not verified against all codes |
| UX-022 | P0 | On a failed submission, the page MUST show an error summary at the top that links to each  | partial | `internal/web/refusal.go` | - | Error summary present; focus move and value retention not verified |
| UX-023 | P0 | Amount inputs MUST state the accepted format in the label or helper text ("Amount in credi | partial | `internal/web/templates/propose.html` | `internal/web/a11y_test.go:TestFormLabelsAndAutocomplete` | inputmode decimal on amount fields; format text in helper |
| UX-024 | P1 | Dates and times MUST be shown in the member's chosen time zone with its abbreviation, in a | not built | `internal/web/render.go` | - | Member time zone not supported; times shown in server zone |
| UX-025 | P0 | Server failures (5xx) MUST show a generic message with the request reference (07 API-030)  | built | - | `internal/web/security_test.go:TestErrorPagesHideInternals` | Generic 5xx page with request reference |
| UX-026 | P0 | Asynchronous results (for example "Trade sent" after a background change) MUST be announce | partial | `internal/web/templates/base.html` | - | Live region on some pages; asynchronous announcements not verified |
| UX-027 | P0 | Disabled controls MUST be accompanied by visible text explaining why they are disabled and | partial | `internal/web/templates/propose.html` | - | Disabled controls explained in text; not audited for all |
| UX-028 | P0 | Every list and detail screen MUST define four states: loading (announced to assistive tech | partial | `internal/web/templates/history.html`<br>`internal/web/templates/error.html` | - | Loading state not implemented; empty and error states present |
| UX-029 | P0 | A permission failure MUST show a page that explains the person does not have access, with  | built | `internal/web/refusal.go`<br>`internal/web/templates/error.html` | - | Permission failure page with link home |
| UX-030 | P0 | Sessions that expire MUST warn the person at least 2 minutes before expiry (default; produ | built | `internal/web/h_account.go` (sessionExtend)<br>`internal/web/static/app.js` | `internal/web/hardening_test.go:TestSessionExpiryWarning` | Two-minute warning with Stay signed in extension |
| UX-031 | P1 | Suspended members MUST see a persistent notice explaining that they can view but not propo | partial | `internal/web/templates/dashboard.html` | - | Suspended notice not verified in templates |
| UX-032 | P0 | Destructive or irreversible-looking actions (remove member, cancel trade, revoke token, cl | built | `internal/web/h_admin.go` (memberActionForm)<br>`internal/web/h_trade_act.go` (tradeActionForm) | - | Confirmation step names object for remove and cancel |
| UX-033 | P0 | All core screens MUST reflow to a single column at a 320 CSS pixel viewport width without  | partial | `internal/web/static/style.css`<br>`docs/qa/LIGHT-QA.md` | - | Single-column layout with media queries; 320px checked by manual QA only |
| UX-034 | P0 | Text MUST remain readable and all functionality available when text is resized to 200 perc | partial | `internal/web/static/style.css` | - | Resize and reflow not measured |
| UX-035 | P0 | Interactive targets MUST be at least 24 by 24 CSS pixels; targets SHOULD be at least 44 by | partial | `internal/web/static/style.css` | - | Target sizes not measured |
| UX-036 | P0 | The layout MUST NOT lock orientation; portrait and landscape MUST both be usable. | partial | `internal/web/static/style.css` | - | No orientation lock; not tested |
| UX-037 | P1 | Tables with more than four columns MUST collapse into labelled stacked rows at narrow widt | not built | `internal/web/static/style.css` | - | Stacked rows for wide tables not built |
| UX-038 | P0 | Every function reachable with a pointer MUST be reachable and operable with the keyboard a | partial | `docs/qa/LIGHT-QA.md` | - | Keyboard operation verified manually for sign-in and trade form only |
| UX-039 | P0 | Focus order MUST follow reading order. | partial | `internal/web/templates/base.html` | - | Reading order by template; not automated |
| UX-040 | P0 | A visible focus indicator MUST be present on every focusable element, and focused elements | partial | `internal/web/static/style.css` | - | Focus outline present in CSS; not automated |
| UX-041 | P0 | Each page MUST provide a skip link to the main content. | built | `internal/web/templates/base.html` | `internal/web/a11y_test.go:TestPageStructure` | Skip link to main |
| UX-042 | P0 | Any action that uses dragging MUST have a single-pointer or keyboard alternative. The MVP  | built | `internal/web/static/app.js` | - | No drag interactions |
| UX-043 | P0 | Multi-point or path-based gestures MUST NOT be required. Actions MUST fire on pointer rele | built | `internal/web/templates/propose.html` | - | Actions fire on submit and link activation |
| UX-044 | P0 | Each page MUST have a unique title that includes the network name and screen name. Heading | built | `internal/web/templates/base.html` | `internal/web/a11y_test.go:TestPageStructure` | Page title includes network name and screen |
| UX-045 | P0 | Structures (lists, tables with header cells and scope, forms with labels, landmarks: heade | built | `internal/web/templates/base.html` | `internal/web/a11y_test.go:TestPageStructure` | Semantic lists, tables, landmarks |
| UX-046 | P0 | The document language MUST be set. The MVP language is English; all strings MUST be extern | built | `internal/web/templates/base.html` | - | Document language en |
| UX-047 | P1 | The UI MUST be structured for localisation: no concatenated sentences in code, plural and  | not built | - | - | Strings not externalised into catalogues |
| UX-048 | P0 | Form labels MUST be visible text; the accessible name MUST contain the visible label text. | built | - | `internal/web/a11y_test.go:TestFormLabelsAndAutocomplete` | Visible labels and autocomplete |
| UX-049 | P1 | Help links MUST appear in the same location on every page of the same section. | partial | `internal/web/templates/base.html` | - | Help link in same place not verified |
| UX-050 | P0 | Copy MUST NOT use the words "money", "cash", "coins", "tokens" (except for the API token s | built | `internal/web/templates/help.html` | - | Copy avoids money and cash wording for credits |
| UX-051 | P0 | Sign-in and recovery MUST NOT require the person to solve a cognitive test (for example, t | built | `internal/web/templates/login.html` | - | Sign-in form allows paste; no cognitive test |
| UX-052 | P0 | Account recovery flows MUST use the same error message for a recognised and an unrecognise | partial | - | `internal/service/hardening_test.go:TestRegisterIsUniform` | Recovery not built; registration response uniform |
| UX-053 | P0 | Two-factor or other additional verification, if adopted, MUST be available with a non-cogn | not built | - | - | Second factor not built |
| UX-054 | P0 | Automated accessibility checks MUST run on every page template in CI with the WCAG 2.2 AA  | partial | `internal/web/a11y_test.go` | - | No automated WCAG checker in CI; structural tests only |
| UX-055 | P0 | Each release MUST pass a keyboard-only walkthrough of the core flows: sign-in, propose tra | not built | `docs/qa/LIGHT-QA.md` | - | Keyboard-only walkthrough not recorded for release |
| UX-056 | P0 | Each release MUST have a screen-reader pass of the same core flows on at least one desktop | not built | `docs/qa/LIGHT-QA.md` | - | Screen-reader pass not recorded |
| UX-057 | P0 | Colour contrast MUST meet 4.5:1 for body text and 3:1 for large text and for non-text UI b | partial | `internal/web/static/style.css` | - | Dark scheme token set present; contrast not measured |
| UX-058 | P1 | Motion MUST respect the operating system "reduce motion" setting. No content MAY flash mor | built | `internal/web/static/style.css` | - | Reduced motion media query present |
| UX-059 | P1 | The application MUST provide a way to close or extend time-limited content (session warnin | built | `internal/web/h_account.go` (sessionExtend)<br>`internal/web/static/app.js` | - | Session warning can be extended |
| UX-060 | P0 | Accessibility defects found in release testing MUST be logged and classified. Any WCAG 2.2 | partial | - | - | Defect logging process not evidenced in repo |

## 09 Security and privacy

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| SEC-001 | P0 | Password storage | partial | `internal/service/password.go` (DefaultArgon2Params)<br>`internal/service/service.go` (MinPasswordLen) | `internal/service/auth_test.go:TestPasswordStorage` | Argon2id PHC with per-password salt; benchmark for DEC-006 not recorded |
| SEC-002 | P0 | Password policy | partial | `internal/service/password.go` (validatePassword) | `internal/service/auth_test.go:TestPasswordPolicy` | Length 12 enforced; breached-password check not wired (PasswordDenied unset) |
| SEC-003 | P0 | Sessions | partial | `internal/service/auth.go` (Authenticate)<br>`internal/service/auth.go` (Logout)<br>`internal/service/auth.go` (ChangePassword) | `internal/service/auth_test.go:TestLoginSessionLifecycle`<br>`internal/service/auth_test.go:TestChangePasswordRevokesSessions` | Server sessions, cookie attributes, password change revokes; role-specific timeouts not built |
| SEC-004 | P0 | CSRF protection | partial | `internal/httpx/httpx.go` (CSRF) | `internal/web/security_test.go:TestCSRFRejections`<br>`internal/httpx/httpx_test.go:TestCSRFScheme` | CSRF tokens on unsafe methods; token bound to random cookie, not to session |
| SEC-005 | P0 | Uniform responses and anti-enumeration | partial | `internal/service/auth.go` (Login) | `internal/service/hardening_test.go:TestRegisterIsUniform`<br>`internal/service/auth_test.go:TestLoginDoesNotRevealExistence` | Login and sign-up responses uniform; recovery not built |
| SEC-006 | P0 | Rate limiting and backoff for authentication | partial | `internal/service/auth.go` (Login) | `internal/service/hardening_test.go:TestLoginThrottleIsAtomicUnderConcurrency`<br>`cmd/circula/hardening_test.go:TestTrustedProxiesSetting` | Atomic per-login throttle and per-address limiter via trusted proxies; no per-network or invitation limits |
| SEC-007 | P1 | Multi-factor authentication for privileged roles | not built | - | - | MFA not built |
| SEC-008 | P0 | Account recovery | not built | - | - | Account recovery not built |
| SEC-009 | P0 | Server-side authorization on every request | built | - | `internal/service/authz_test.go:TestAuthorizationMatrix`<br>`internal/api/authn_test.go:TestAuthorisation` | Default deny; authorisation matrix test |
| SEC-010 | P0 | Role separation | built | - | `internal/service/dispute_test.go:TestDisputeDecisionAuthorization`<br>`internal/service/membership_test.go:TestAdministratorSuccession`<br>`internal/service/reversal_test.go:TestReversalAuthorization` | Self-approval, conflicts, last-admin and operator-grant refusals |
| SEC-011 | P1 | Dual control for value-moving administrative actions | not built | `internal/service/adjustments.go` (CreateAdjustment) | - | No dual control; adjustments post at once |
| SEC-012 | P0 | Administrative action audit and notice | partial | `internal/service/audit.go` (audit) | `internal/service/atomicity_test.go:TestEveryCommandWritesAuditEvent` | Audit in same unit for admin actions; notices to affected members not built |
| SEC-013 | P0 | Platform operator access | partial | `cmd/circula/commands.go` (cmdAdmin) | - | No operator role reads network data in product; break-glass not built |
| SEC-014 | P0 | Data minimization | partial | `internal/service/auth.go` (RegisterPerson) | - | Minimal profile fields; data inventory document absent |
| SEC-015 | P0 | Tenant isolation between networks | built | - | `internal/service/authz_test.go:TestCrossNetworkIdsAreNotFound`<br>`internal/store/sqlite_test.go:TestCrossNetworkReferencesRejected` | Network-scoped queries and cross-network tests |
| SEC-016 | P0 | Directory visibility and identity display | partial | `internal/service/membership.go` (ListDirectory) | - | Members-only directory; no-index directive absent |
| SEC-017 | P0 | Export controls | built | `internal/api/jobs.go` (jobTTL) | `internal/service/export_test.go:TestCSVFormulaNeutralisation`<br>`internal/service/export_test.go:TestMemberExportIsScoped` | Member and admin scopes; CSV formula neutralisation; 24h download expiry |
| SEC-018 | P0 | Retention and erasure | not built | `internal/service/departure.go` (DepartMember) | - | No pseudonymisation or erasure on departure |
| SEC-019 | P0 | Audit event access and content | partial | `internal/service/audit_read.go` (ListAudit) | `internal/service/audit_test.go:TestAuditEventContent` | Audit excludes secrets; admin-only read path; member read in service only |
| SEC-020 | P0 | Secure logging | partial | `internal/service/purge.go` (RedactSecrets)<br>`cmd/circula/ops_http.go` (routeOf) | `internal/service/hardening_test.go:TestRedactSecrets` | RedactSecrets and route templates in logs; no log-wide canary scan |
| SEC-021 | P1 | Log retention and access | not built | - | - | Log retention not implemented |
| SEC-022 | P0 | Transport security | built | `cmd/circula/config.go` (checkTransport) | `cmd/circula/hardening_test.go:TestPlainHTTPRefusedUnlessExplicit`<br>`internal/web/hardening_test.go:TestSecureCookiesAndHSTS` | Refuses plain HTTP on non-loopback unless explicit; Secure cookies forced |
| SEC-023 | P0 | Browser security headers | built | - | `internal/web/security_test.go:TestHeadersAndCookies`<br>`internal/httpx/httpx_test.go:TestSecureAndHSTS` | CSP, frame-ancestors, no-store, same-origin scripts |
| SEC-024 | P0 | Output encoding and input validation | built | `internal/httpx/httpx.go` (SafeNext) | `internal/web/hardening_test.go:TestSafeNextRejectsBrowserTricks`<br>`internal/domain/domain_test.go:TestParseAmount` | Safe next parsing; html/template; integer amounts |
| SEC-025 | P1 | Abuse reporting | not built | - | - | Abuse reporting not built |
| SEC-026 | P1 | Basic velocity and pattern signals | not built | - | - | Velocity and reciprocal signals not built |
| SEC-027 | P0 | Secrets management | partial | `cmd/circula/config.go` (loadSecrets)<br>`cmd/circula/config.go` (readSecretFile) | `cmd/circula/hardening_test.go:TestSecretsLoadingAndRedaction` | Secrets from env or files; rotation procedure not documented for session key |
| SEC-028 | P0 | Backup security | partial | `cmd/circula/snapshot.go` (manifestMAC)<br>`cmd/circula/snapshot.go` (checkManifestAuth) | `cmd/circula/hardening_test.go:TestSignedBackupManifest`<br>`cmd/circula/perm_unix_test.go:TestFilesAreOwnerOnly` | Backup manifest signed with key; backup not encrypted at rest; files 0600 |
| SEC-029 | P1 | Dependency and supply-chain hygiene | partial | `.github/workflows/ci.yml`<br>`scripts/package-release.sh` | - | Module versions pinned; govulncheck and SBOM module list in CI; dependency checksum failure test not present |
| SEC-030 | P0 | Incident response | partial | `docs/ops/monitoring.md`<br>`docs/ops/runbook.md` | - | Incident contacts and failure table in docs/ops; drills not recorded |
| SEC-031 | P1 | Vulnerability disclosure | built | `SECURITY.md` | - | SECURITY.md with private reporting channel; owner approval of contact pending (DEC-026) |
| SEC-032 | P1 | Sign-up grant and Sybil controls | partial | `internal/service/adjustments.go` (postSignupGrant) | - | Grant defaults to zero; cap and open-mode confirmation not built |
| SEC-033 | P1 | Security test obligations | partial | - | - | Tests cover most P0 SEC items; independent review not performed |
| SEC-034 | P2 | Tamper-evident audit chain (deferred option) | deferred | `internal/service/audit.go` (AuditHash) | - | Tamper-evident option deferred; unkeyed chain already present (DEC-044) |

## 10 Governance and operations

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| GOV-001 | P0 | Prospective-only policy changes | partial | `internal/store/sqlite_network.go` (AppendPolicy) | `internal/service/admin_test.go:TestPolicyChanges` | Policy versions prospective; policy version stored on each transaction not verified |
| GOV-002 | P0 | Recording policy changes | built | `internal/service/networks.go` (UpdatePolicy)<br>`internal/service/gov_terms.go` (PolicyLog)<br>`internal/web/h_gov.go` (rulesPage) | `internal/service/gov_h1_test.go:TestPolicyNoticePeriod`<br>`internal/web/gov_h1_test.go:TestPolicyNoticePages` | Changes audited with before and after; member-visible change log |
| GOV-003 | P0 | Notice before adverse changes | built | `internal/service/networks.go` (UpdatePolicy, noticeDays)<br>`internal/domain/terms.go` (Adverse)<br>`internal/store/sqlite_gov.go` | `internal/service/gov_h1_test.go:TestPolicyNoticePeriod`<br>`internal/service/gov_h1_test.go:TestPolicyNoticeCancel` | 14-day default notice, shortened only with a reason shown in the log, inbox notice to members, cancellation |
| GOV-004 | P0 | Decimal scale lock | built | `internal/service/networks.go` (UpdatePolicy) | `internal/service/admin_test.go:TestPolicyChanges` | Decimals lock after first completed transaction |
| GOV-005 | P1 | Policy validation bounds | built | `internal/domain/policy.go` (Validate) | `internal/domain/statemachine_test.go:TestPolicyBounds` | Bounds with boundary tests |
| GOV-006 | P1 | Policy decision method | partial | `internal/service/networks.go` (UpdatePolicy) | - | Unilateral admin change built; objection right not built |
| GOV-007 | P0 | No governance override of ledger invariants | built | - | `internal/service/property_test.go:TestRandomisedLedgerInvariants` | Corrective actions are transactions; property tests |
| GOV-008 | P0 | Join modes | built | - | `internal/service/membership_test.go:TestJoinModes`<br>`internal/service/membership_test.go:TestInvitations` | Invite, approval, open modes; audit per decision |
| GOV-009 | P0 | Onboarding | built | `internal/service/gov_terms.go`<br>`internal/service/join.go` (RedeemInvitationTerms, JoinNetworkTerms)<br>`internal/web/h_gov.go` | `internal/service/gov_h1_test.go:TestTermsGovernance`<br>`internal/web/gov_h1_test.go:TestTermsPages` | Versioned terms; acceptance at join; re-acceptance before trading after an update |
| GOV-010 | P0 | Voluntary departure | partial | `internal/service/departure.go` (DepartMember) | `internal/service/admin_test.go:TestDepartureRetain`<br>`internal/service/gov_h1_test.go:TestDeparturePreviewAndNotices` | Blockers, retain and settle rules built; counterparties notified; self-export only linked, not offered in the flow |
| GOV-011 | P0 | Removal by administrator | partial | `internal/web/h_admin.go` (memberActionPost) | `internal/service/admin_test.go:TestDepartureBlockersAndAdminOffboarding`<br>`internal/service/gov_h1_test.go:TestDeparturePreviewAndNotices` | Admin removal with reason; member notified; challenge not built |
| GOV-012 | P1 | Inactivity | not built | - | - | Inactivity rule not built |
| GOV-013 | P0 | Dispute window and eligibility | built | `internal/service/disputes.go` (OpenDispute) | `internal/service/dispute_test.go:TestOpenDisputeRules` | Window and party rule; dispute keeps entries unchanged |
| GOV-014 | P0 | Dispute process | partial | `internal/service/disputes_decide.go` (ResolveDispute)<br>`internal/service/disputes.go` (OpenDispute) | - | Statement and decision with reasons; response period and deadlines not built |
| GOV-015 | P0 | Reversals and adjustments | built | `internal/service/reversal.go` (ReverseTransaction)<br>`internal/service/adjustments.go` (CreateAdjustment) | `internal/service/reversal_test.go:TestFullReversal` | Reversal and adjustment are new transactions; originals unchanged |
| GOV-016 | P0 | Challenges and appeals | not built | - | - | Challenges and appeals not built |
| GOV-017 | P1 | Dispute evidence | built | `internal/api/routes.go` | - | No dispute upload route |
| GOV-018 | P0 | Recovery of negative balances | partial | `internal/service/adjustments.go` (CreateAdjustment) | - | Trades continue within limits; write-off by approved adjustment not built (SEC-011) |
| GOV-019 | P0 | Reduction of credit limit below current negative balance | built | - | `internal/service/trades_test.go:TestLimitReductionDoesNotPost`<br>`internal/service/trades_test.go:TestAcceptRechecksLimit` | Limit reduction posts nothing; increases refused |
| GOV-020 | P0 | Administrator continuity | partial | - | `internal/service/membership_test.go:TestAdministratorSuccession` | Last-admin guard built; read-only administrator-required state not built |
| GOV-021 | P0 | Recovery when no administrator is available | not built | - | - | Platform-assisted recovery not built |
| GOV-022 | P0 | Export contents and portability | partial | `internal/service/export.go` (Export) | `internal/service/export_test.go:TestFullExportJSON` | Member, network and full exports built; reconciliation summary in manifest; import not built |
| GOV-023 | P0 | Network suspension by an administrator | not built | `internal/domain/types.go` (NetworkStatus) | - | Network suspension not built (status has no suspended value) |
| GOV-024 | P0 | Platform suspension | not built | - | - | Platform suspension not built |
| GOV-025 | P0 | Network closure | partial | `internal/service/networks.go` (CloseNetwork) | `internal/service/admin_test.go:TestNetworkLifecycle` | Closure refuses with blockers and freezes postings; notices, final export and erasure not built |
| GOV-026 | P1 | Published platform policies | not built | - | - | Hosted platform policy documents not written |
| GOV-027 | P1 | Support boundaries | partial | `docs/ops/runbook.md` | - | Support boundary in docs/ops; not linked from admin UI |

## 11 Deployment and operations

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| OPS-001 | P0 | Same artifact for both models | built | `scripts/package-release.sh`<br>`cmd/circula/serve.go` (runServe) | - | One artefact for both models |
| OPS-002 | P0 | Single-process deployment | built | `cmd/circula/serve.go` (runServe) | `cmd/circula/jobs_regress_test.go:TestServeStartsAndStopsCleanly` | Single process; embedded database; loopback default |
| OPS-003 | P0 | Supported platforms | partial | `scripts/package-release.sh` | - | Five targets built in CI with checksums; per-target smoke run not performed |
| OPS-004 | P1 | Environments | partial | `README.md` | - | Environments described in docs/ops/README.md; no staging refresh process |
| OPS-005 | P2 | Horizontal scaling and alternative stores | deferred | `internal/store/sqlite.go` (Open) | - | Alternative store and multi-node deferred; no SQLite import outside store |
| OPS-006 | P0 | Configuration sources and precedence | built | `cmd/circula/commands.go` (resolveFlags) | `cmd/circula/config_test.go:TestPrecedence`<br>`cmd/circula/config_test.go:TestEverySettingAcceptedByEverySource` | Flags, environment, file, defaults precedence |
| OPS-007 | P0 | Validation at start-up | built | `cmd/circula/config.go` (buildConfig) | `cmd/circula/config_test.go:TestInvalidValuesNameEachSetting`<br>`cmd/circula/config_test.go:TestCheckConfigDoesNotCreateDatabase` | Validation before serving; check-config does not create database |
| OPS-008 | P0 | Required settings and defaults | built | `cmd/circula/config.go` (settingDefs) | `cmd/circula/config_test.go:TestRequiredSettingsAreNamed` | Settings present with documented defaults |
| OPS-009 | P0 | Secrets handling | partial | `cmd/circula/config.go` (readSecretFile)<br>`cmd/circula/config.go` (secretDefs) | `cmd/circula/hardening_test.go:TestSecretsLoadingAndRedaction` | Secret files read; inline secret warning not separately tested |
| OPS-010 | P1 | Secret rotation | partial | `docs/ops/backup-restore.md` | - | Backup key rotation documented; session key rotation not implemented |
| OPS-011 | P0 | Storage configuration | built | `internal/store/sqlite.go` (Open)<br>`cmd/circula/serve.go` (checkPragmas)<br>`internal/store/sqlite.go` (DefaultBusyTimeout) | - | WAL, foreign keys, busy timeout, startup pragma check; durability fixed (CMD-06) |
| OPS-012 | P0 | Write concurrency and bounded transactions | built | - | `internal/web/hardening_test.go:TestBusyDatabaseIs503`<br>`internal/store/hardening_test.go:TestBusyDatabaseIsRetryable` | Short write units; busy returns retryable 503 |
| OPS-013 | P0 | Metrics and health endpoints | built | `cmd/circula/ops_http.go` (healthz)<br>`cmd/circula/ops_http.go` (readyz)<br>`cmd/circula/ops_http.go` (metricsAccess) | `cmd/circula/ops_test.go:TestHealthzReportsCoarseStatusOnly`<br>`cmd/circula/hardening_test.go:TestMetricsGate` | healthz, readyz, metrics with allow-list and token |
| OPS-014 | P0 | Required operational metrics | partial | `cmd/circula/metrics.go` (newMetrics) | `cmd/circula/ops_test.go:TestMetricsExposeRequiredSeriesWithRouteTemplates` | HTTP and job metrics built; posting attempt and idempotent-replay counters absent |
| OPS-015 | P0 | Alerts | partial | `docs/ops/monitoring.md` | - | Alert table in docs/ops/monitoring.md; alerts not configured or tested in staging |
| OPS-016 | P0 | Ledger verification command | built | `internal/service/verify.go` (VerifyLedger)<br>`cmd/circula/commands.go` (cmdVerifyLedger) | `cmd/circula/cli_regress_test.go:TestVerifyLedgerLeavesDatabaseChecksumUnchanged` | Read-only verify with checksum test; CLI and admin entry points |
| OPS-017 | P0 | Backup method and consistency | built | `cmd/circula/snapshot.go` (snapshot) | `cmd/circula/backup_test.go:TestSnapshotWritesVerifiedManifestAndRestores` | Online snapshot with manifest and checksum; load test not run |
| OPS-018 | P0 | Backup schedule and retention | built | `cmd/circula/snapshot.go` (pruneBackups) | `cmd/circula/backup_test.go:TestPruneKeepsDailyAndNewestPerMonth`<br>`cmd/circula/jobs_regress_test.go:TestJobScheduledBackupAndRetention` | Daily schedule with 30 daily and 12 monthly retention |
| OPS-019 | P0 | Backup storage and protection | partial | `docs/ops/backup-restore.md` | `cmd/circula/perm_unix_test.go:TestFilesAreOwnerOnly` | Backups not encrypted; off-host copy is an operator procedure |
| OPS-020 | P0 | Restore procedure and verification | partial | `cmd/circula/commands.go` (cmdRestore)<br>`cmd/circula/commands.go` (swapDatabase)<br>`cmd/circula/commands.go` (writeRestoreRecord) | `cmd/circula/hardening_test.go:TestRestoreRefusesBackupFailingEmbeddedChecks` | Restore refuses while serving (lock) and verifies; restore audit event not in ledger |
| OPS-021 | P0 | Restore drills | partial | `docs/ops/backup-restore.md` | - | Drill procedure documented; no drill records |
| OPS-022 | P0 | Versioned, forward-only migrations | built | `internal/store/migrate.go` (loadMigrations)<br>`cmd/circula/schema.go` (checkSchemaNotNewer) | `internal/store/sqlite_test.go:TestMigrationsIdempotentAndVersioned` | Versioned forward-only migrations; newer schema refused |
| OPS-023 | P0 | Migration safety | partial | `cmd/circula/commands.go` (cmdMigrate)<br>`cmd/circula/serve.go` (preMigrationBackup) | - | Post-migration verification only; pre-migration verification and injected-failure test not present |
| OPS-024 | P0 | Upgrade procedure | partial | `docs/ops/upgrade.md` | - | Upgrade sequence documented; maintenance mode not built |
| OPS-025 | P1 | Migration mode | built | - | `cmd/circula/serve_test.go:TestServeRefusesPendingMigrationsInExplicitMode` | Explicit migrate mode default; auto mode takes backup |
| OPS-026 | P0 | Versioning and release notes | partial | `docs/releases/TEMPLATE.md`<br>`cmd/circula/main.go` (version) | - | Release notes template exists; version set by build flag |
| OPS-027 | P0 | Idempotent scheduled jobs | built | `cmd/circula/serve.go` (jobRunner)<br>`cmd/circula/serve.go` (expire)<br>`cmd/circula/serve.go` (purge)<br>`cmd/circula/serve.go` (verify)<br>`cmd/circula/serve.go` (backups) | `cmd/circula/jobs_regress_test.go:TestJobRunStartsAllJobsAndStops` | Expiry, purge, verify and backup jobs idempotent; last run recorded in metrics |
| OPS-028 | P0 | Expiry does not bypass posting | built | `internal/service/transactions.go` (ExpireProposals) | `internal/service/trades_test.go:TestExpiry` | Expiry uses same service path; no entries |
| OPS-029 | P1 | Notification delivery | not built | - | - | No mail and no stored notices |
| OPS-030 | P1 | Support tiers and severity | partial | `docs/ops/runbook.md` | - | Severity levels in docs/ops/runbook.md; not linked from incident procedure |
| OPS-031 | P1 | Support access | partial | `docs/ops/runbook.md` | - | Support boundary documented; break-glass not built |
| OPS-032 | P0 | Structured logs | built | `cmd/circula/serve.go` (newLogger) | - | Structured JSON logs; log level needs restart |
| OPS-033 | P0 | Failure recovery requirements | partial | `docs/ops/monitoring.md`<br>`cmd/circula/serve.go` (runServe) | - | Failure table documented; per-network freeze not built |
| OPS-034 | P1 | Recovery objectives | not built | - | - | RPO and RTO not set |
| OPS-035 | P1 | Packaging | partial | `scripts/package-release.sh` | - | Binary, checksum and module list per target; signing deferred; no secret scan in package |

## 12 Quality and test plan

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| QA-001 | P0 | P0 | partial | - | `internal/service/property_test.go:TestRandomisedLedgerInvariants` | Property tests run in go test; sequence counts and operation list not verified |
| QA-002 | P0 | P0 | built | `internal/service/verify.go` (VerifyLedger) | `internal/service/property_test.go:TestRandomisedLedgerInvariants` | Cached balance agreement checked after random operations |
| QA-003 | P0 | P0 | built | - | `internal/store/sqlite_test.go:TestTransactionIntegrityTriggers`<br>`internal/service/trades_test.go:TestTradeScenarioBalances` | Two-entry and zero-sum checks |
| QA-004 | P0 | P0 | partial | - | `internal/service/trades_test.go:TestAcceptRechecksLimit` | Limit boundary covered by acceptance tests; zero and one-unit boundary cases not named |
| QA-005 | P0 | P0 | built | - | `internal/domain/domain_test.go:TestNoFloatsInMoneyPackages`<br>`internal/domain/domain_test.go:TestAddChecked` | No float types; overflow refused |
| QA-006 | P0 | P0 | built | - | `internal/service/atomicity_test.go:TestAcceptFaultInjectionLeavesNoTrace`<br>`internal/service/atomicity_test.go:TestAuditFailureRollsBackStateChange` | Fault injection at accept and audit failure |
| QA-007 | P0 | P0 | partial | - | `internal/service/concurrency_test.go:TestParallelAcceptsNeverBreachCreditLimit` | Parallel accept test present; 100 interleavings not configured |
| QA-008 | P0 | P0 | not built | - | - | Disjoint concurrent trades test not present |
| QA-009 | P0 | P0 | built | - | `internal/service/idempotency_test.go:TestIdempotentRetryReturnsOriginal`<br>`internal/service/idempotency_test.go:TestIdempotencyRecordExpires` | Retry, reuse and expiry tests |
| QA-010 | P0 | P0 | partial | `internal/testutil/testutil.go`<br>`internal/testutil/fault.go` | - | Fault store wrapper exists; injection at each posting step not complete |
| QA-011 | P0 | P0 | partial | - | `internal/store/hardening_test.go:TestBusyDatabaseIsRetryable` | Busy retryable test; seeded contention scenario not present |
| QA-012 | P0 | P0 | built | - | `internal/service/authz_test.go:TestAuthorizationMatrix` | Authorisation matrix test |
| QA-013 | P0 | P0 | partial | - | `internal/domain/statemachine_test.go:TestDisputeStateMachineExhaustive` | Dispute state machine tests; response deadline and appeal not built |
| QA-014 | P0 | P0 | built | - | `internal/service/reversal_test.go:TestFullReversal` | Originals unchanged after reversal |
| QA-015 | P0 | P0 | partial | - | `internal/domain/statemachine_test.go:TestPolicyBounds`<br>`internal/domain/statemachine_test.go:TestMemberTransitionsExhaustive` | Policy bound and lifecycle tests; notice and objection not built |
| QA-016 | P0 | P0 | partial | - | `internal/service/dispute_test.go:TestDisputeDecisionAuthorization`<br>`internal/service/membership_test.go:TestAdministratorSuccession` | Self-approval, conflict and last-admin tests; approval expiry not built |
| QA-017 | P0 | P0 | partial | - | `internal/web/flow_test.go:TestJoinByInvitationLink` | Invitation join tested at HTTP level; terms acceptance not built |
| QA-018 | P0 | P0 | partial | - | `internal/api/trade_test.go:TestFullTradeFlow`<br>`internal/web/flow_test.go:TestLoginTradeFlow` | Trade flow tested over HTTP and API; no browser end-to-end suite |
| QA-019 | P0 | P0 | partial | - | `internal/service/trades_test.go:TestDeclineCancelExitRules`<br>`internal/service/trades_test.go:TestExpiry` | Decline and expiry tested; notices not built |
| QA-020 | P0 | P0 | partial | - | `internal/service/trades_test.go:TestAcceptRechecksLimit` | Limit boundary scenario tested at service level; browser flow not run |
| QA-021 | P0 | P0 | partial | - | `internal/service/dispute_test.go:TestUpheldReversalBlockedThenOverride`<br>`internal/service/reversal_test.go:TestFullReversal` | Dispute and reversal tested; party notices not built |
| QA-022 | P0 | P0 | partial | - | `internal/service/trades_test.go:TestLimitReductionDoesNotPost` | Limit reduction tested; notice not built |
| QA-023 | P0 | P0 | partial | - | `internal/service/admin_test.go:TestDepartureRetain` | Departure tested; notices and self-export offer not built |
| QA-024 | P0 | P0 | partial | - | `cmd/circula/backup_test.go:TestSnapshotWritesVerifiedManifestAndRestores` | Snapshot restore tested; restore audit event not in ledger; no load |
| QA-025 | P0 | P0 | deferred | - | - | Import of exported data is deferred (04 section 10) |
| QA-026 | P0 | P0 | partial | - | `cmd/circula/hardening_test.go:TestSwapDatabaseRollsBack` | Restore rollback tested; upgrade with forced migration failure not present |
| QA-027 | P0 | P0 | partial | - | `internal/store/sqlite_test.go:TestMigrationsIdempotentAndVersioned` | Migration tests on current schemas; fixture per released version absent |
| QA-028 | P0 | P0 | partial | - | `cmd/circula/hardening_test.go:TestBackupEqualsSource` | Restore integrity tested; backup under concurrent load not run |
| QA-029 | P0 | P0 | not built | - | - | No restore drill records |
| QA-030 | P0 | P0 | partial | - | `internal/service/export_test.go:TestLedgerExportCSV`<br>`internal/service/export_test.go:TestFullExportJSON` | Export totals tested; independent reconciliation script not present |
| QA-031 | P0 | P0 | built | - | `internal/service/export_test.go:TestMemberExportIsScoped` | Member scope and canary checks in export tests |
| QA-032 | P0 | P0 | built | - | `internal/service/export_test.go:TestCSVFormulaNeutralisation` | Formula neutralisation test |
| QA-033 | P0 | P0 | partial | `internal/web/a11y_test.go` | `internal/web/a11y_test.go:TestPageStructure` | Structural checks in Go tests; no automated WCAG checker in CI |
| QA-034 | P0 | P0 | partial | `docs/qa/LIGHT-QA.md` | - | Keyboard and scripting-disabled operation checked manually for some flows |
| QA-035 | P0 | P0 | not built | `docs/qa/LIGHT-QA.md` | - | Manual screen-reader review not recorded |
| QA-036 | P0 | P0 | partial | `internal/web/format.go` (balanceVM) | `internal/web/format_test.go:TestUnitForPlural` | Balance text tested; contrast not checked |
| QA-037 | P0 | P0 | partial | `.github/workflows/ci.yml` | - | CI runs vet, lint, race tests, govulncheck; L8 and migration static checks partial |
| QA-038 | P0 | P0 | partial | `.github/workflows/ci.yml` | - | Integration gate partly automated; scenario browser tests absent |
| QA-039 | P0 | P0 | not built | - | - | Release gate evidence (drills, reviews, performance) not complete |
| QA-040 | P0 | P0 | not built | - | - | Thresholds pending DEC-023; not confirmed |
| QA-041 | P0 | P0 | partial | `internal/testutil/testutil.go` | - | Injected clocks used; seeds recorded for randomised tests only partly |
| QA-042 | P0 | P0 | partial | - | - | Traceability matrix produced in docs/none; test dates and builds to be added at each release |
| QA-043 | P0 | P0 | partial | `internal/testutil/testutil.go` | - | Fixtures synthetic; no automated personal-data scan |
| QA-044 | P0 | P0 | not built | - | - | Evidence retention process not defined |

## 13 Decision register

| ID | Pri | Requirement (short) | Status | Code path(s) | Test name(s) | Note |
|---|---|---|---|---|---|---|
| DEC-005 | - | - | open, partly built | `internal/service/auth.go` (Authenticate) | - | Single idle (720 h) and absolute (2160 h) timeouts for all roles; role split not built |
| DEC-006 | - | - | open, partly built | `internal/service/password.go` (DefaultArgon2Params) | - | Min length 12 and Argon2id built; benchmark and breached-password list not |
| DEC-007 | - | - | open, not built | - | - | No second factor built |
| DEC-008 | - | - | open, partly built | `internal/store/sqlite.go` (Open) | - | One shared database with network-scoped rows; per-network files not built |
| DEC-009 | - | - | open, not built | - | - | No recovery channel and no mail |
| DEC-010 | - | - | open, partly built | `internal/service/departure.go` (DepartMember) | - | Retain default and settle-to-zero rule built; counsel review open |
| DEC-011 | - | - | open, partly built | `internal/service/adjustments.go` (postSignupGrant) | - | Grant off by default; approval or invite requirement and caps not built |
| DEC-012 | - | - | open, not built | - | - | Dispute response period and timelines not built |
| DEC-013 | - | - | open, not built | - | - | Dual control not built |
| DEC-014 | - | - | open, not built | - | - | Break-glass not built |
| DEC-015 | - | - | open, partly built | `internal/service/audit.go` (AuditHash) | - | Unkeyed hash chain built; keyed or external publication deferred |
| DEC-016 | - | - | open, not built | - | - | RPO and RTO not recorded |
| DEC-017 | - | - | decided, built | `cmd/circula/commands.go` (resolveFlags)<br>`cmd/circula/config.go` (settingDefs) | - | Flags, environment, file, defaults; explicit migrate default |
| DEC-018 | - | - | decided, built | `cmd/circula/ops_http.go` (metricsAccess) | - | Text exposition; loopback allow-list and optional token |
| DEC-019 | - | - | open, not built | - | - | Log retention not implemented |
| DEC-020 | - | - | open, not built | - | - | Retention periods not set |
| DEC-021 | - | - | open, partly built | `internal/service/networks.go` (UpdatePolicy) | - | Unilateral administrator change built; objection right not built |
| DEC-022 | - | - | open, not built | - | - | Support objectives not set |
| DEC-023 | - | - | open, not built | - | - | Numerical thresholds not confirmed |
| DEC-024 | - | - | open, not built | - | - | Accessibility level and browser list not recorded |
| DEC-025 | - | - | open, not built | - | - | Notice periods not implemented |
| DEC-026 | - | - | open, partly built | `SECURITY.md` | - | Private vulnerability reporting named in SECURITY.md; owner approval not recorded |
| DEC-027 | - | - | deferred | - | - | Code signing and notarisation deferred |
