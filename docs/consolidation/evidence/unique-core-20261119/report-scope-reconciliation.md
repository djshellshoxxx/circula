# SEC-025 report scope reconciliation

Source-only review of `/workspace/scratch/4d30fa819181/circula` at
`ca019903b33a762cb86dcc52f98e9c8de34a794d` on 2026-11-20. The root worktree also
contained ongoing parent-owned documentation changes; no repository file was
edited and no tests were run in this review.

## Finding

**Transaction abuse reporting and report-to-dispute association are genuine
remaining Core behavior under SEC-025. They are not future-module work and no
explicit accepted deferral was found.** The narrow case of a member reporting
an ordinary two-party trade, with private links to disputes on that same trade,
has enough stated authority to implement without a new governance decision.
That bounded completion must not be described as all of SEC-025 while the
broader identity/role and non-trade target semantics below remain unresolved.

| Requirement / sub-scope | Current source | Contract and authority | Smallest remaining action |
|---|---|---|---|
| FR-024: listing/member reports | `internal/service/reports.go` supplies `ReportListing`, `ReportMember`, `CreateReport`, the administrator queue and handling. `internal/web/h_notify.go:reportTarget` and API `h_reports.go` expose those targets. The integrated privacy follow-up excludes report subjects from queue, handling, correlated audit reads and complete audit/full exports. | FR-024 explicitly requires active-member reporting of listings/members, an administrator queue, no automatic target state changes, and reporter identity visible only to administrators and never the reported member. | Existing behavior for these two target types is implemented; preserve its independent-administrator checks when extending targets. |
| SEC-025: transaction report creation | `createReport` has no transaction branch; all other target types fail validation. `0008_notify_reports.sql` constrains `target_type` to `listing` or `member`. `reportTarget` cannot resolve transactions and `trade_detail.html` offers dispute actions but no report action. | SEC-025 expressly requires reporting a transaction in which the reporter is a party and refusing unrelated transactions. No report-specific accepted decision or addendum defers this. `domain.Transaction.IsParty` already defines payer/payee membership, and `service.loadParty` is the existing service authorization primitive. A dispute's completion/window restrictions are not report restrictions. | Add a transaction target through a forward migration, service validation, API and ordinary trade detail/report form. Preserve existing active-member/network, category/note, duplicate, rate-limit and idempotency behavior. For the unambiguous two-member trade case, determine the reported party from the non-reporting counterparty and extend the central subject checks, SQL queue/audit predicate and complete-export refusal together. Reporting must not open/resolve a dispute or change ledger/transaction state. |
| SEC-025: related dispute link for transaction reports | `repository.Report`, report DTO and `ReportView` contain no dispute reference. `DisputeView` contains no report relation. `DisputeFilter.TxID`, `ListDisputes`, and `GetDisputeForTx` already locate disputes by transaction; the last returns only the newest dispute. Existing trade-to-dispute navigation is not a report-to-dispute link. | SEC-025 says a report must be linked to any related dispute. A shared network and exact transaction ID provide an objective relation for a transaction report. There is no need to redefine dispute outcomes, decider roles, time windows, or ledger behavior to expose that relation privately. | Add the same-transaction relation to the authorized report projection/queue, covering disputes created before or after the report and retained dispute history. A derived join can avoid a second independently mutable relationship. Keep report identity/existence out of the general party-visible dispute DTO/page; any reciprocal report panel needs the same report subject/reader checks. Do not equate the current newest-dispute helper with all related history. |
| SEC-025 versus FR-024: reviewer visibility | `ListReports` and `HandleReport` require administrator authority; API/HTML routes are administrator queue/handling routes. Reviewer access is absent. | SEC-025 names administrators and dispute reviewers as the restricted report audience; FR-024 specifically limits reporter identity to administrators. PER-003 further limits reviewer personal data to what is needed for the dispute. These constraints permit a narrower design than granting reviewers the existing full administrator DTO. DEC-072 concerns deciding disputes, not report visibility or handling. | A related-dispute-only, read-only report projection for independent reviewers, with reporter identity redacted and subject exclusions preserved, is an implementable way to honor all three constraints. Treat reviewer handling powers, access to the whole report queue, or disclosure of reporter identity as separate authority questions; none is expressly granted by the report requirements. Do not invent a blanket hold on reviewer reads merely because these broader choices exist. |
| SEC-025: non-trade transactions and broader “related” semantics | `domain.Transaction` includes `trade`, `reversal`, `adjustment`, and `grant`; system-account payer/payee member IDs can be empty. `CreatedBy`, `OriginalTxID`, and `RelatedTxID` exist but do not declare a report subject. Disputes are restricted to completed trades in `OpenDispute`. Listing/member reports have no selected dispute/transaction context. | SEC-025 says “transaction” broadly. The other party is clear for an ordinary two-member trade, but an adjustment/grant may have one member and a system account; automatically treating the creating administrator or an original-trade counterparty as the reported person is not stated. Likewise, sharing a listing/member does not by itself prove that every dispute is related to a report. No explicit deferral was found; these are unresolved contract details, not proof of a general policy hold. | Before claiming complete transaction coverage, specify the report subject for one-member/system transactions, treatment of reversals/related adjustments, and how an explicit related dispute is selected/validated for listing/member reports. Preserve the usable two-party implementation slice and list unsupported cases honestly until that scope is defined. |
| WAPI-01: every available administrator conflicted | The integrated note `docs/consolidation/REPORT_PRIVACY_20261120.md` records retained reports, no subject handling, and existing System service authority. There is no dedicated operator report-handling command or established escalation procedure. | This is already an explicitly held operational decision. It is separate from authorizing party-only transaction creation or a private same-transaction relation. | Keep this operational hold; do not add a role grant, operator escalation privilege, quorum, or support procedure as part of ordinary report implementation. |

## Canonical sources and limits

- `docs/specs/09-security-privacy.md`, SEC-025 requirement and its three acceptance
  criteria supply the transaction/party/privacy/link contract.
- `docs/specs/04-functional-requirements.md`, FR-024 acceptance supplies the
  existing administrator-only identity constraint and no automatic effects.
- `docs/specs/02-personas-journeys.md`, PER-003 constrains reviewer data access.
- `docs/specs/README.md` says historical build statuses do not establish current
  implementation, future specifications do not authorize Core implementation,
  and addenda do not settle open owner decisions.
- `docs/DECISIONS.md` and `docs/specs/addenda/` contain no report-specific accepted
  decision that removes SEC-025's transaction or linking requirements. DEC-072
  and DEC-073 concern dispute decision authority and non-party transaction
  reads, respectively; neither is a reason to hold a party-only report create.

This review establishes source gaps and an implementation boundary only. It
does not assert execution results, universal confidentiality qualification, or
completion of SEC-025.
