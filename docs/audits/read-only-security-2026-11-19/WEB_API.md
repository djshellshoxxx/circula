# Circula read-only security review: HTTP API and web boundary

Review date: 2026-11-19 (America/Los_Angeles; 2026-11-20 UTC). Worker: web/API review. Status: source review complete for the coverage described below; two actionable findings, no application changes.

## Scope, provenance, and limits

The review compared main at `aa01563fcc891da60fefebb8335eca27855678e8` with the consolidated Core candidate at `22c767a90f69d286642ef96e6b3c174836637a3c` (PR 125). The candidate was inspected in a detached local worktree at `/workspace/scratch/1b6df59fd7e6/Circula-web-audit`. Line references below use the candidate unless explicitly stated otherwise. The main checkout was not changed. The service and API files establishing both findings are byte-identical between the pinned revisions; relevant web differences are presentation/catalogue changes, not authorization or transaction fixes.

This was a defensive source audit, with inspection of existing test code and an attempted ordinary local baseline. No live Circula deployment was contacted, no exploit or new reproducer was written or run, and no remote issue, comment, commit, or pull request was created. Findings describe the violated invariant and repair requirements rather than attack instructions. The separate authentication, ledger/storage, and operational reviewers own their deeper areas; this note does not repeat their findings as independently verified discoveries.

| Finding | Severity | Confidence | Main | Candidate | Suggested owner |
|---|---|---|---|---|---|
| WAPI-01 — report confidentiality omits administrator/subject conflict | Medium | High, source-confirmed | Present | Present | Reporting/governance service, API/web adapters |
| WAPI-02 — listing `If-Match` is checked outside the write transaction | Low security severity; data-integrity defect | High, source-confirmed | Present | Present | Listings service, API concurrency contract |

Severity reflects the demonstrated source behavior and its prerequisites. These are not claims of outsider account compromise or ledger corruption. Lack of dynamic reproduction limits runtime assurance, but the missing checks and the relevant call paths are directly visible.

## WAPI-01 — a reported administrator receives reporter identity and can mark the complaint handled

### Expected boundary

FR-024 states that the reporter's identity is visible to administrators only **and is not shown to the reported member** (`docs/specs/04-functional-requirements.md:235–241`). SEC-025 independently requires that reports not reveal the reporter's identity to the reported party (`docs/specs/09-security-privacy.md:352–358`). The current product makes this promise on the report form and confirmation (`internal/web/messages/en.json:1405–1406,1417–1418`). Neither promise describes an exception when the subject also has administrator privileges.

### Evidence and exact boundaries

The service protects the administrative queue from ordinary members but does not account for overlap between administrative authority and being the subject:

* `internal/service/reports.go:134–173`, `ListReports`, opens the network and requires administrator authority at lines 140–146. It then reads the network's reports at line 150. Each returned `ReportView` contains the underlying report, including `ReporterID`, and adds the reporter's display name at lines 155–157. Target-member and target-listing labels are resolved at lines 159–167. There is no comparison between the requesting administrator and the target member or listing owner.
* `internal/api/h_reports.go:25–28` serializes `ReporterID` as `reporter_member_id`. The queue adds the reporter's display name at lines 62–74. The HTML queue copies the same name to the page at `internal/web/h_notify.go:208–236`, particularly lines 224–226. These are actual returned fields, not only UI navigation hints.
* `internal/service/reports.go:186–226`, `HandleReport`, requires administrator authority at lines 189–195, then reads the report and accepts an open report at lines 196–205. It never tests whether this administrator is its subject. Lines 211–217 mark the report handled and send the reporter a generic notice that an administrator reviewed it. Lines 220–223 audit the change and return the report.
* Both adapters call that service path: `internal/api/h_reports.go:77–89` and `internal/web/h_notify.go:269–282`. The API's action response also returns the underlying report with its reporter identifier.

The affected actor must have effective same-network administrator authority, including any normal configured MFA requirement. A report must concern that administrator directly, or a listing owned by that administrator. Ordinary membership alone does not grant access; normal network scoping remains in place. This finding does not allege an unauthenticated queue, cross-network access, or automatic suspension caused by filing or handling a report.

### Impact

The person reported can identify the reporter despite the form's unconditional confidentiality assurance. The same person can close the report workflow and generate the review notice without independent handling. This creates a practical retaliation and governance-accountability risk. Marking handled changes the report record and its notice; it does not itself execute a financial or member-status action. The complaint and audit trail remain, and handling does not prevent every future report. The confidentiality/independent-handling failure is distinct from deletion of evidence.

### Adjacent disclosure surface to include in the repair

A template-only fix would leave both the API and service response exposed. A queue-only service fix also needs a review of report-correlated audit reads. Report creation records an event with the report ID and reporter as its subject (`internal/service/reports.go:120–122`). `ListAudit` gives administrators the whole network log (`internal/service/audit_read.go:45–64`); the identity reductions at lines 69–81 apply to the ordinary-member branch. Workers should trace this association and export/cached-response paths before declaring the reporter-confidentiality guarantee satisfied. These adjacent sources establish why repair scope must be broader than concealing one visible column; no additional independent audit-export bypass is claimed here.

### Repair direction

Define a central report-subject conflict predicate. For a member report it resolves the target member; for a listing report it resolves the relevant owner. Apply the predicate consistently to service reads and mutations, and to authorization before returning any idempotent response. The mutation decision belongs in the write transaction. Do not rely on the web adapter, a caller-supplied target, or an administrator-only navigation condition.

Decide explicitly how reports against the only administrator, or against all available administrators, are escalated or held for independent review. Also specify behavior when membership roles change after filing. If the intended authority policy is instead that every administrator can see complaints about themselves, that requires an explicit product/security decision and an honest revision of the user-facing guarantee; the current behavior should not be described as satisfying the existing promise.

### Safe acceptance criteria for the repair worker

1. With ordinary inert fixtures, an administrator who is the reported member cannot receive the reporter's identifier or name from service, API, or HTML report views.
2. The equivalent listing-owner conflict receives the same treatment, and unrelated administrators retain their intended review access.
3. A conflicted subject cannot mark that report handled. The check is inside the mutation transaction and applies to replay responses as well as first execution.
4. Audit, export, and cached-response behavior is reviewed for identity correlation; any intentional exceptions are documented and reflected in the product promise.
5. Cross-network, reviewer-only, ordinary-member, revoked-role, and configured-MFA restrictions retain their current effect.

`internal/service/reports_test.go`, especially `TestReports` and `TestHandleReport`, already checks ordinary reported-member denial, administrator access, and the ordinary member's reduced audit view. The inspected fixtures do not cover the subject also being an administrator. Extend the role-overlap cases rather than treating the existing administrator-versus-member split as proof of the stronger privacy property.

## WAPI-02 — explicit listing ETags do not protect against an intervening authorized write

### Expected boundary

API-026/027 require `If-Match` on listing edits and require a stale validator to produce 412 with no change (`docs/specs/07-api-contract.md:83–88`). API-047 identifies listing PATCH as this conditional operation. The current implementation checks the validator in the API adapter, but the later service mutation does not carry or enforce that expected revision.

### Evidence

* `internal/api/h_listings.go:99–113`: `patchListing` locks the API server's `casMu`, reads the listing through `ownListing`, compares `If-Match` with the current listing ETag, and creates `service.ListingUpdate{ListingID: cur.ID}`. It calls `UpdateListing` later at lines 151–156.
* `internal/api/api.go:49–59` places `casMu` on the API server. Its scope is the adapter, not every writer of listing state.
* `internal/service/listings.go:116–124`: `ListingUpdate` has listing ID, editable fields, clear-price, and status, but no expected revision.
* `internal/service/listings.go:128–188`: `UpdateListing` opens its own write transaction, rechecks active membership and ownership, reads the current listing at lines 141–144, and applies the requested fields. It sets the new update time and persists at lines 180–186, with no comparison to the version seen by the API.
* `internal/web/h_home.go:416–435`: the HTML edit builds the same update type and calls the service directly. It does not take the API mutex. Other legitimate service callers have the same separation.

Consequently, a legitimate service/web write can occur after the API's read/check and before its write. The API then applies its fields to the newer listing despite the explicit validator having become stale. This is a source-confirmed missing transaction invariant, not a measured race result.

### Impact and limits

The affected actor is an authenticated active listing owner, and an intervening authorized edit is required. Overlapping field edits can lose the intervening value while the conditional API operation reports success. The API-instance mutex genuinely serializes its own listing handlers, so this finding requires an authorized writer outside that mutex. The current ETag hashes editable content as well as time (`internal/api/dto.go:155–160`); no timestamp-only collision finding is asserted. Price, description, title, or listing status can be affected according to the requested fields and existing state restrictions.

The service still enforces ownership, same-network resolution, active membership, and final archived state. This finding does **not** show another member modifying a listing, arbitrary financial posting, or a breach of ledger atomicity. It is rated low as a security defect because its direct scope is the owner's listing, while remaining a concrete API data-integrity and concurrency-contract failure.

### Historical relationship

This is a residual instance of the check/act gap identified in the earlier audit, `docs/audit/CODE-AUDIT.md:97–99` under **API-02**. The later remediation record documents transaction-enforced expected revisions for credit-limit and policy changes (`docs/audit/2026-10-28/REMEDIATION-2026-11-18.md:27`). Those repairs should not be taken as evidence that listing writes acquired the same property. Both pinned revisions still have the listing flow described above.

### Repair direction and safe acceptance criteria

Carry an expected listing revision into `UpdateListing` and compare it inside the same write transaction that reads and updates the listing. Return a distinct stale-revision error mapped by the API to 412. Prefer a version that changes reliably for every relevant write rather than relying only on wall-clock resolution. All listing writers should preserve the invariant. Preserve any intentional wildcard/existence semantics separately from matching a specific validator.

Use an ordinary deterministic interleaving fixture to verify that a stale explicit validator is refused after an intervening authorized service edit, with the latest state preserved. Verify that a matching revision still succeeds, that the returned ETag reflects the committed representation, and that non-owner denial remains unchanged. No traffic race or stress demonstration is necessary to establish the repair.

## Boundary controls observed

The review found several substantive controls already implemented. These are observations of the inspected paths, not a certification of all endpoint combinations.

| Area | Observed implementation and evidence | Remaining assurance limit |
|---|---|---|
| Session and CSRF consistency | Shared cookie naming and session-bound HMAC CSRF in `internal/httpx/httpx.go:24–91`; API unsafe cookie requests check the token in `internal/api/auth.go:55–63`. Web cookie flags, token rotation, and validation are in `internal/web/session.go:32–105`. | Cryptographic lifecycle and account recovery received a separate reviewer. Deployment-specific proxy/TLS settings were not tested here. |
| Origin, headers, and caching | Web sets CSP, framing/referrer controls and `Cache-Control: no-store` in `internal/web/middleware.go:36–48`; its origin check starts at line 86. API sets restrictive JSON-oriented CSP, same-origin resource policy and no-store at `internal/api/middleware.go:38–41`. No permissive CORS grant was identified in the inspected middleware. | No browser matrix or external reverse-proxy configuration was exercised. Headers alone do not prove each view safe. |
| Request parsing and body limits | API defaults to 64 KiB in `internal/api/api.go:80–82` and reads with `MaxBytesReader` in `internal/api/decode.go:29`. Web caps request bodies at 1 MiB in `internal/web/middleware.go:62`. Strict JSON/canonical-field code rejects ambiguous fields; structured query handling constrains supported parameters on inspected API routes. | This was not a parser fuzzing campaign. HTML parameter handling varies by form, and every malformed byte sequence was not tested. |
| Pagination | `internal/api/cursor.go` authenticates cursor payloads, constrains encoded size and binds the cursor to its route/context; service page sizes are clamped. | This is not a guarantee that all underlying database work is bounded by page size. No large-dataset profiling was performed. |
| Network and role authority | API resolves the selected network against memberships/token scope (`internal/api/auth.go`); web network/admin wrappers and service calls recheck authority. Adapter-owned administrative reads use `AuthorizeAdminRead` (`internal/api/h_policy.go:99–104`). | Role overlap must still be modeled, as WAPI-01 demonstrates. Deeper service invariants belong to the other reviewers. |
| Replay authorization | `internal/api/idem.go` and `replay_auth.go` recheck current permissions before response replay. Erasure hooks remove retained responses associated with erased users/networks. | WAPI-01's new subject-conflict rule would need to join this authorization model. |
| Export authority and retention | `internal/service/export.go:124–141` scopes own versus administrator exports and checks erasure. Lines 158–176 enforce the durable configured per-member export quota. Lines 181–193 reauthorize in the snapshot before generating content. API job/download, spool-budget, and erasure-hook code were inspected. | Direct HTML streaming and cached API downloads have different lifecycles. No runtime disk, cancellation, or platform-permissions experiment was performed. Do not report HTML exports as unmetered merely because they bypass the API spool mechanism. |
| Output handling | Inspected rendering uses Go `html/template`, with narrow trusted-time formatting and catalogue/template-owned slots. Static `app.js` was inspected; no new executable insertion from member text was identified in that file. API unknown failures become opaque responses (`internal/api/respond.go`). | This is not a complete XSS proof for every localized template or a dependency/browser audit. |

## Resource-bound observations and coordinated ownership

The API has IP/principal request-rate defaults (`internal/api/api.go:26–31,83–87`), bounded request bodies, and export spool accounting. Request rates are not equivalent to an aggregate bound on expensive concurrent password hashing. The authentication reviewer is recording that separate availability issue. This worker confirmed that the inspected web middleware and `cmd/circula/ops_http.go` do not introduce a global hash/admission semaphore; no new load or password-guessing test was performed here.

An additional hardening observation is that `internal/api/idem.go:66–78` attempts to evict expired or completed records at capacity, then inserts a new record even if all existing records remain in progress. The configured entry count therefore is not an absolute admission bound in that state. Practical impact depends on admitted concurrent requests, completion time, timeouts, and upstream limits, which were not measured. This is recorded as a capacity-review item, not promoted to a separately demonstrated denial of service. The historical remediation notes already leave the A3-10 eviction question owner-pending. A worker reviewing this should define an explicit overload result and avoid dropping in-progress ownership records.

Other concerns were not inflated into findings: existing export quotas and snapshot reauthorization were present; malformed/hidden IDs often normalize to not-found; password fields are not populated from generic form redisplay; and trusted template HTML uses were traced where encountered. Questions about Windows spool directory inheritance, sustained database growth, or every possible cancellation ordering require evidence outside this worker's completed scope.

## Coverage ledger

The review read the API route inventory and authorization chain, authentication adapter, common middleware, request decoding and canonical fields, cursor logic, idempotency/replay logic, export spooling/jobs/downloads and erasure hooks, and API handlers for accounts, administrative actions, audit scopes, closure, core members/transactions, dispute review, disputes, dual control, F2, governance, historical balances, listings, notifications, operations, passwords, reports, sessions, suspension, tokens, and transaction actions. Response/error mapping and policy conversion were also inspected. DTOs were followed where relevant to privacy and amount/ETag representation.

The web review covered routing/wrappers, session/CSRF/origin handling, throttling, rendering and form-value handling, static application JavaScript, authentication/recovery/account/security flows, home/listing editing, reporting/notifications, secondary administrative handlers, erasure, own-session management, and scoped-audit views. Transaction confirmation/rendering paths and sensitive templates were sampled. Web governance, all dispute decision/re-review forms, all administrative network forms, every localization catalogue, and every display helper were **not** individually exhaustively reread; the API paths and corresponding service authorization were the stronger focus. Do not translate this scope into a claim that every web template or route was fully audited.

Supporting source included shared HTTP edge utilities, operational HTTP wrapping, reporting/listing/export services, report tests, and selected security/API specs and prior-audit/remediation records. Authentication cryptography, authorization recovery, ledger accounting, database schema/migrations, backup/deployment, dependencies, and CI received separate ownership rather than duplicate complete reviews here.

## Validation performed and result

Source status was clean in the detached candidate worktree after review (`git status --short` returned no changes). The notes are outside the application checkout.

An existing ordinary API baseline was attempted with Go 1.26.11 supplied by the root reviewer, `GOTOOLCHAIN=local`, the shared downloaded module/cache paths, and `GOMAXPROCS=2`:

```text
go test ./internal/api \
  -run '^(TestRouteInventoryMatchesAuthorizationTable|TestWebSessionWorksForAPI|TestSecureSessionCookieWorksForAPI|TestEntriesOrderedByLedgerSeq|TestSingleReadsByID)$' \
  -count=1 -timeout=3m
```

It failed during compilation of `modernc.org/libc` with `compile: signal: killed`; the package reported `[build failed]`. **No selected test assertions executed, and this worker reports no test pass from that run.** The failure is a local runtime/resource limit, not evidence that the application tests fail. The root reviewer subsequently recommended lower memory/parallelism flags. No optional repeat or broadened test run was pursued after this failure. Existing test source was inspected for the report coverage gap; no new tests were added.

## Worker handoff order

1. Assign WAPI-01 to the reporting/governance owner to settle the conflict policy and enforce it across service reads, mutations, replay, and identity-correlated views. Update confidentiality text only in accordance with the chosen policy.
2. Assign WAPI-02 to the listings/API owner to complete the transaction-level revision check using the already-established credit-limit/policy pattern.
3. Keep the in-progress idempotency capacity question and aggregate hashing capacity with the availability review. Preserve their distinct evidence and confidence levels.
4. Run the existing boundary baseline and focused ordinary regression cases in a provisioned CI environment after fixes. Record the exact tested revision; the pinned candidate is not the same as main and these defects are currently present in both.

No patch was applied and no authority was exercised against a running service. This file is a repair handoff, not a claim of comprehensive absence of other vulnerabilities.
