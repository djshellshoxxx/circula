# Actionable Core scope reconciliation — 2026-11-20

Source root: `/workspace/scratch/4d30fa819181/circula`. Review began at
`ca019903b33a762cb86dcc52f98e9c8de34a794d` and finished after the audit-mirror and
documentation integration at `01685005573bf36bcbec07f06a4c14c8bb4a73fa`. The
current audit-mirror metrics and documentation were inspected in the working
tree before that integration committed. No repository edits or runtime tests
were performed by this review. Historical status labels were used only to find
candidate requirements, not as evidence that behavior is absent.

**Full Core feature completion is not established yet.** Five concrete areas
remain implementable under existing requirements: migration semantic rollback,
posting outcome metrics, announced loading feedback, ordinary two-member
transaction reports with exact-transaction dispute links, and the remaining
authentication message fragments. Existing support, terms, parser, timing and
several security/lifecycle absence notes are stale. Release/manual evidence and
explicitly held policy/future-module work are excluded from that implementation
list.

## 1. Concrete remaining implementation

| Requirement | Current implementation / actual gap | Accepted authority | Smallest remaining action |
|---|---|---|---|
| **OPS-023, P0** — semantic migration verification before commit | `internal/store/migrate.go:migrateTo` executes each migration, inserts its version, and commits before service ledger verification. `cmd/circula/commands.go:cmdMigrate` verifies only after `openStore` has applied migrations. `snapshot.go:snapshotWith` explicitly records ledger verification as not run for an older-schema backup. Verified backups, SQLite integrity checks, transactional DDL failure rollback and refusal to serve after failed final verification exist; these do not supply pre-commit semantic invariant refusal. | OPS-023 explicitly requires before/after verification for ledger-affecting migrations and refusal of a newly introduced invariant failure before commit. No policy/threshold decision is needed for that correctness rule. | Introduce a schema-compatible invariant check within the migration transaction, preserving the before-state comparison and rolling back migration data/version changes on a newly introduced failure. Retain the verified backup and final startup checks. This is production behavior, not merely another test citation. Root assigned this to storage completion. |
| **OPS-014, P0; ARC-022, P1** — posting attempt/failure outcomes | `internal/service/observe.go` exposes only `Posted` and `Replayed`; `Service.update` invokes them only after a successful commit. `postTx` increments the committed-posting accumulator. `cmd/circula/metrics.go` exports committed postings/replays but has no posting attempt/failure-by-reason series. `docs/ops/monitoring.md` explicitly acknowledges refused-posting and service-authentication visibility gaps. Latency is keyed only by route, while OPS-014 asks for route/status-class breakdown. | OPS-014 explicitly names attempts, successes and failures by reason class; ARC-022 names posting success/failure counts. Their correctness and bounded labels do not depend on numerical alert thresholds. | Extend the observer and operation outcome bookkeeping to count attempts and classified refusals/rollbacks without treating a rolled-back posting as committed or a replay as a second posting. Use fixed reason classes, preserve existing metrics, wire the production service, and add the missing latency status dimension. Root owns the posting outcome work. |
| **UX-028, P0** — loading state | `internal/web/templates/base.html` has flash/error/freshness/session live regions. `static/app.js` handles error focus, offline/history freshness and session extension only. There is no reusable announced loading state for normal list/detail navigation or form submission. Browser navigation is still functional; empty/error/content states already exist. | UX-028 expressly requires an announced loading state. Existing progressive-enhancement/no-script behavior does not authorize adding a JavaScript dependency to core navigation. No new product policy is required. | Add one catalogue-backed loading/status enhancement to the shared shell for ordinary navigation/submission, reset it on return/cancellation, and preserve native navigation, validation and no-script operation. Root assigned the shared frontend files separately. |
| **SEC-025, P1** — ordinary two-member transaction reports | `reports.go`, the report table's check constraint in `0008_notify_reports.sql`, `api/h_reports.go`, `web/h_notify.go:reportTarget`, and the report form currently support only listing/member targets. There is no report action on `trade_detail.html`. | SEC-025 explicitly permits a reporter to report a transaction they are party to and requires refusing unrelated transactions. No accepted decision/addendum defers that ordinary case. FR-024 supplies existing limits, no automatic effects, administrator-only reporter identity and confidentiality from the subject. | Add the transaction target through a forward migration, service, API and HTML. Require the reporter to be a party and resolve the subject as the other member when both parties are distinct real members. Extend the shared queue/audit/export conflict predicate and handling/replay checks together. Preserve existing active-member restrictions, duplicate/rate limits, idempotency and no ledger/dispute mutation. Do not borrow dispute-window or completed-state restrictions that the reporting contract does not impose. |
| **SEC-025, P1** — report/dispute association | Reports and their views/DTOs have no dispute association. A transaction's existing dispute navigation is not a report-to-dispute relation. `DisputeFilter.TxID` and `ListDisputes` already provide exact-transaction lookup; `GetDisputeForTx` returns only the newest case. | The same network and exact transaction ID are an objective relation under SEC-025. No new governance outcome or escalation privilege is required. | Derive the relation in the authorized report projection, including later and historical disputes on the exact transaction. Keep report identities/details out of the general party-visible dispute page/DTO. Do not silently link every dispute sharing a member/listing or only the newest case. |
| **UX-047, P1** — remaining complete-message migration | The recovered catalogue is extensive, but `mfa_enrol.html` still assembles `auth.enrol.already_on` + link + `page_word`; `login.html` assembles recovery sentences around links; `recovery_codes.html` combines `auth.codes.have`, a raw count, `auth.codes.unused`, and literal punctuation. `docs/dev/MESSAGES.md` explicitly records these original authentication fragments and still calls UX-047 partial. | English-only scope is accepted. Existing `messages.Around`/count/markup-slot helpers provide complete positional messages. Translation-language choice is separate and does not block this existing English flow. | Finish the authentication sentence/count fragments with existing complete-message markup slots, preserving links, strong emphasis, escaping and exact visible English. Do not add a language selector or alter authentication semantics. This is smaller than a new localization framework. |

The transaction-report/dispute findings were independently inspected in
[`report_scope_reconciliation.md`](report_scope_reconciliation.md).

## 2. Report scope precision — do not convert these into a blanket hold

| Requirement / case | Existing authority and boundary | Concrete disposition |
|---|---|---|
| SEC-025 reviewer reads; FR-024 reporter identity; PER-003 data minimization | SEC-025 names reviewers in the restricted audience; FR-024 limits reporter identity to administrators; PER-003 limits reviewer data to what the dispute requires. Existing queue and handling are administrator-only. DEC-072 concerns dispute decisions, not report creation. | A same-dispute, independent-reviewer, read-only projection with identity redaction is a feasible interpretation of all three constraints. It does not imply full-queue access, handling authority or reporter identity. Keep this separate from the first administrator/report-target implementation slice; do not claim it implemented or invent a blanket reviewer-read hold. |
| SEC-025 one-member/system-account transactions | Adjustment/grant transactions may have only one human party. `CreatedBy`, `OriginalTxID`, and `RelatedTxID` do not themselves declare a reported human subject. | Specify the subject for this case before claiming universal transaction reporting. Do not silently choose the creating administrator or an original counterparty. This missing definition does not block two-real-member records. |
| SEC-025 “related” disputes for member/listing reports | Current reports carry no selected case context. Shared membership/listing alone does not establish that all disputes are related. | Define explicit context/selection and authorization before broad association. Exact-transaction links are independently actionable. |
| WAPI-01 all available administrators conflicted | The integrated `REPORT_PRIVACY_20261120.md` explicitly holds the operational escalation procedure. Existing System service authority remains, with no dedicated operator report command. | Preserve records and subject exclusions. Do not invent a role grant, quorum, support procedure or escalation command. This hold is separate from normal report creation/linking. |

## 3. Stale absence notes and already present mechanisms

| Requirement(s) | Current source evidence | Remaining smallest action |
|---|---|---|
| **GOV-027** support-page link | `templates/admin_home.html` links `/help#support-boundaries`; `templates/help.html` has the support, exception and access-policy explanations. | Correct the historical absence note. Do not build another support page or make support commitments. |
| **PER-010 / GOV-009; historical QA-017 note** terms/onboarding | `service/gov_terms.go` publishes append-only terms, records acceptance and checks current versions; `join.go:RedeemInvitationTerms/JoinNetworkTerms` enforce joining acceptance. `web/h_auth.go`, `h_gov.go` and API handlers expose the flows. | Correct stale terms-absence notes. The release/browser evidence is separate; no new onboarding policy is needed. |
| **DAT-048** migration duration | `store/golden_migration_test.go:TestGoldenMigrationV1With10000Transactions` measures migration/verification time, logs it, and optionally appends to `CIRCULA_MIGRATION_TIMING_FILE`. | Retain actual release-fixture timing during qualification. This is release evidence, not absent timing instrumentation; do not confuse it with OPS-023's real transactional gap. |
| **ARC-020 / DAT-041** reference parser | `scripts/read-export.py` is the standalone Python-stdlib reader, documented in `docs/api/export-schema.md`. `service/export.go` provides adapter-independent exports; closure uses the same ledger export implementation. | Refresh the absent-parser note. No duplicate reader or compulsory new CLI surface follows from ARC-020's CLI-or-background acceptance wording. |
| **UX-022 / UX-023 / UX-024 / UX-031** ordinary form/time/status mechanisms | `base.html` has linked error summaries and retained-value field helpers; `app.js` focuses errors. Amount helpers/name-specific refusal text are present. `render.go:fmtTime/rc.at` emits zoned text and machine-readable `time`; `h_account_f2.go` edits the zone. `render.go` and base shell persist suspended notices. | Broad accessibility qualification remains separate. No missing mechanism was found from these historical notes; UX-028 remains the specific loading gap above. |
| **PER-013 / PER-022** freshness | `base.html` renders snapshot time and `app.js` preserves it while announcing offline/history-return staleness. | Keep actual behavior; do not invent another offline subsystem. |
| **PER-015 / DOM-025** limit warnings, dual control and suspension | `notify_limits.go:RunLimitWarnings` is called by `cmd/circula/jobs_limits.go`; `dualcontrol.go` implements adjustment approval; network suspension has service/API/HTML implementations. | Refresh broad absence claims. Wider dual-control authority remains its explicit SEC-011 owner decision. |
| **PER-016** departure notices/export offer | `departure.go` notifies counterparties when departure cancels proposals; `templates/leave.html` offers a personal export. | Refresh the absence note; retain settlement policy boundaries. |
| **GOV-011 / GOV-016** removal challenge and dispute re-review | `challenges.go:OpenChallenge/DecideChallenge` handles removal, member credit-limit and listing-removal cases with independence; dispute outcome uses its own re-review workflow. | Correct “challenge not built.” General policy-change objection is distinct; see decision boundary below. |
| **GOV-001 / DOM-030** policy binding | Current spec now distinguishes each policy's binding event rather than requiring one blanket proposal snapshot; `networks.go`, proposal/acceptance and dispute-policy binding implement those rules. | Preserve the accepted binding rules. A historical “policy version stored on each transaction not verified” note does not justify a new universal snapshot. |
| **SEC-003 / SEC-004 / SEC-006 / SEC-032** existing controls | `sessions_revoke.go` and `cmd/newService` wire role lifetimes; `web/session.go` binds signed-in CSRF tokens to the session; `signup_controls.go` and CLI config wire network join budgets and grant caps/open-grant confirmation. | Refresh mechanism-absence notes. Default-policy approvals remain separate. No fresh security audit or changed privileges were undertaken here. |
| **SEC-027 / OPS-010** runtime key persistence/rotation | `cmd/statekey.go` and `specs/addenda/SEC-027-keys.md` define persisted private keys and the stop/delete/restart rotation procedure. | Correct the missing-procedure note. Do not add a new key policy from that historical note. |
| **OPS-024 / OPS-033** maintenance/freeze | Maintenance command/middleware exists; `service/freeze.go` keys freezes by network, and serving/verification paths use it. | Refresh absence notes; deployment drills remain qualification. |
| **VIS-002** disclaimer placement | Every page footer already displays `footer.credits`, including “not guaranteed to be accepted.” The inspected README lacks that explicit statement and there is no named network About page. | This is a documentation/placement gap, not absent mutual-credit behavior: add the existing non-guarantee statement to README and expose/reconcile the named network About acceptance placement using current network information. Do not invent financial or support promises. |

## 4. Explicitly excluded work and accepted deviations

- **Existing owner/contract gates:** SEC-026 activity signals, GOV-021 recovery,
  reviewer-reset independence, SEC-011 wider dual control, SEC-013/ARC-009
  break-glass powers, SEC-014/018 retention, and the all-conflicted report
  procedure retain the holds recorded in `docs/audits/REMAINING_BLOCKERS.md`,
  the relevant addenda and DECISIONS. SEC-002's breached-password source and
  role/MFA defaults retain DEC-006/005/007. Merely having a configurable
  implementation does not settle a default or external source.
- **GOV-006 policy objections:** the three current challenge kinds are not a
  general policy objection. GOV-016 permits a policy objection and describes
  recording it, while DEC-021 remains explicitly open about the decision
  method/objection right. Do not relabel this built; keep the exact recorded
  choice separate from removal challenges and from mandatory ordinary Core
  fixes. Voting remains deferred.
- **Accepted as-built alternatives:** DAT-012 uses cumulative reversal caps
  under DEC-040; DAT-024 uses an audited time-limited limit change under
  DEC-039; DOM-036 routes disputed reversals through disputes under DEC-052.
  These historical partial labels do not authorize replacing the accepted
  behavior. GOV-022/VIS-009 import remains deferred under DEC-061.
- **MFA presentation:** SEC-007's implemented addendum explicitly chooses
  text/`otpauth://` provisioning without a QR dependency. Do not treat an old
  “no QR/WebAuthn” note as automatic authorization for new authentication
  methods. UX-053 accessibility assessment remains a separate qualification.
- **Release/manual/external evidence:** per-target native smoke, hosted
  execution, exact release fixture timing, screen-reader/manual accessibility,
  numerical budgets/reference hardware, counsel, licence, support commitments,
  vulnerability-contact approval and independent professional security review
  are not new runtime feature implementations in this pass.
- **All 15 future modules** remain isolated under their recorded HOLDs. Missing
  production adapters there do not become Core work through this review.

No status was changed and no optional test was added to fill a citation. The
five implementation areas in section 1 should be completed and qualified
before describing the accepted Core feature set as complete; the narrower
report contract questions and existing external/policy gates must remain
explicit afterwards.

## Transaction-report implementation follow-up — 2026-11-20

The ordinary two-member SEC-025 transaction report and exact-transaction dispute
relation identified above are implemented in local commit
`d885af837b7d5b1a10c7b0c885265a6504b684dd` (base `0168500`). The source-only
findings above describe the earlier reviewed head. Migration 0023 preserves old
report records and constraints and has a late-DDL rollback/retry fixture.

Focused Go 1.26.11 race execution passed 31 top-level tests and 58 subtests
across store, service, API, HTML, catalogue and API contract packages, with
zero failures, skips or race reports. See `transaction_reports_verification.json`
and the maintained `docs/consolidation/TRANSACTION_REPORTS_20261120.md` note in
the transaction-report branch. This does not promote the full SEC-025 status:
system-account subject semantics, reviewer report projection, explicit
member/listing case relations and the existing all-conflicted operational
procedure remain distinct boundaries. Other gap rows are retained as review-time
findings for their separately assigned completion work.
