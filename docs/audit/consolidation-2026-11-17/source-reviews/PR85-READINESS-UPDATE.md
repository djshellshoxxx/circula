# PR85 readiness update: author corrections through e11d292

Date: 2026-11-18. Reviewer: core_recovery (C), Caveman source review. This updates the live disposition of the findings in `PR85-GOV021-READINESS.md`; that earlier report remains an exact record of b20c118. **No PR85 Go build, test, race run, new probe, tracked source edit or remote mutation was performed.** Agent B has the local Go slot for its separate PR87 work.

## Disposition

**Preserve PR85 as active, valuable core work and keep the owner-policy hold.** The author has corrected the specific bounded-scan, continuation and closing-test gaps in source, and has added mandatory reason/operator labeling, a smaller operator view and an access audit for listing. The listing must no longer be described as wholly unaudited. Permanent recovery authority, actual two-person authentication and the remaining operator-access policy still require an owner decision. Source corrections are not an independent runtime pass; qualify the eventual approved integration head after the policy decision.

| Item | Exact value |
|---|---|
| PR | https://github.com/djshellshoxxx/circula/pull/85 |
| Branch | `claude/gov-021-admin-recovery` |
| New reviewed head | `e11d2920a325b910bf75c1e431208b94ebb6045b` |
| Tree | `b1876f8e1fd2bcee9eba8f0e854f236798915234` |
| Metadata base / merge base against cdc8112 | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` |
| Prior reviewed head | `b20c118b59d43836bd357788914542b904f13872` |
| Forward commits | `a5b258eab5268da0892ad3b2826e9418c8d77f6c` corrections; `e11d292...` regenerated traceability |
| Forward scope | 18 files, 725 additions / 117 deletions |
| Refreshed PR metadata | Open, draft, mergeable, not merged |

Metadata is preserved in `PR85-E11D292-METADATA.json`; exact source blob IDs in `PR85-E11D292-SOURCE-OBJECTS.json`; full forward diff in `PR85-b20-e11-forward.patch`.

## Updated finding ledger

| Prior finding | Status at e11d292 | Evidence and remaining gate |
|---|---|---|
| C85-01: permanent recovery path versus canonical SEC-013 / actual two-person approval | **Owner-policy hold remains; now explicitly documented** | Design record adds an “Owner gate” section; TODO names permanent rights-changing authority versus SEC-013 Decision 7, typed labels, waiting/objection period and provisional expiry. No owner approval, canonical specification amendment or authenticated second staff principal was introduced. |
| C85-02: operator listing without reason/access audit | **Reason/audit/output minimization corrected in source; operator-access policy still held** | List now requires System actor, operator label and reason; reads and audit commit in one write transaction; records one platform event and one event in each represented network, with requester subjects; returns a restricted view. There is still no break-glass or time-bounded access grant, independently authenticated second approver, or separate access notice. |
| C85-03a: first-500 open-request scans | **Corrected in source; supplied boundary tests added** | Both request eligibility and lazy expiry call an indexed `GetOpenAdminRecovery(network, member)` lookup. A request cannot be hidden by newer requests from other members. |
| C85-03b: inaccessible later CLI/web pages | **Corrected in source; supplied paging tests added** | CLI supports bounded `-limit` and `-after`; web adds request and event continuation links. API continuation receives explicit coverage. |
| C85-04: closing test masked by seven-day expiry | **Corrected test source** | Old 31-day masked case removed; replacement explicitly keeps the request unexpired across the network closing date and checks the closing-specific result, no grant and unchanged approved request. |
| C85-05: migration coordination | **No new migration change in this delta** | PR85 still has migration 0019. Earlier SEC-026 numbering correction to 0020 and its stack dependency were already recorded in the prior report; this review does not assert a newly duplicated number or requalify the latest PR86 head. Keep stack coordination with its owning workstream. |

## Operator listing correction and limits

`internal/service/admin_recovery.go` adds `AdminRecoveryListInput` and replaces the raw request pointer in `AdminRecoveryOperatorView` with explicit display fields. `ListAdminRecoveryOperator` now calls `recoveryDecision` for System-actor, operator-label and reason validation, validates the page cursor, and performs the read inside `s.update`.

The transaction collects the networks and requesters actually shown, emits `admin_recovery.listed` at platform scope and in every represented network, and returns no result if the transaction/audit fails. Network events carry requester subjects so their own audit views include the access. The member recovery view continues to remove the operator label/reason; a future appointed administrator can inspect the full event. Networks with no returned requests do not receive a spurious member-data access event. An empty listing still records the platform operation.

The output is limited to request/network identifiers, status and expiry, whether an administrator is still required, an eligibility boolean, requester identity/display name, membership date rounded to a UTC date, and first-approval identity/time. It no longer returns raw recovery reasons, raw membership status, or full request objects. `cmd/circula/admin_recovery.go` requires `-operator` and `-reason` for `list`, and documents that the read is audited.

This meaningfully repairs the missing trail. It does not settle the canonical operator-access controls: SEC-013 and ARC-009 also discuss second staff approval, bounded access and notice. Listing still uses a typed label under a System actor, has no break-glass session/time limit, and emits audit records without a separate access notification. Do not mark the broader access-policy question resolved solely because the audit is now present.

One documentation precision note: the new view comment/design says no person ID is “read”. `GetMember` selects `m.person_id`, and eligibility uses it for the required MFA lookup. The restricted DTO does not expose that ID, which is the useful guarantee. Prefer “not returned/displayed” for that claim; this is an internal-read/output wording distinction, not a new exposure finding.

## Scan and continuation corrections

`internal/store/sqlite_admin_recovery.go:GetOpenAdminRecovery` selects by both network and requester, restricted to requested/approved states; no row is returned as nil. The existing partial unique index allows at most one such row. Both `requestable` and `closeExpiredRecovery` use it, so the previously identified 500-row omission no longer follows from the implementation.

Expiry now advances by ascending request-ID keyset in batches of 100, rather than relying on successful closures to shrink the first batch. Supersession similarly advances through descending open-request IDs in pages of 500. The corresponding repository signature and query were updated together. No schema migration is added.

CLI pages default to 50 rows, accept 1–500, and print a continuation ID when a full page is returned. The page explicitly says more *may* follow, which handles an exactly-full final page without falsely promising another record.

The member page now uses `after` for 50-request pages and `events_after` for 100-event pages. It retains chain-order event display, but adds the missing continuation link, so later actions are reachable instead of permanently hidden behind the oldest 200. Request and event pagers are independent. Existing API signed cursors are unchanged in production and now receive explicit end-to-end list-continuation coverage.

## Supplied regression coverage inspected, not executed

| Test / location | What the source now checks |
|---|---|
| `TestGOV021OpenRequestLookupBeyondFirstPage`, service bounds file | An expired and an unexpired member request behind 501 newer open requests; expired one closes and can be replaced, open one produces the intended duplicate-request conflict; approval supersedes all remaining pages. |
| `TestGOV021ExpiryJobCoversEveryBatch`, same file | 150 expired requests span multiple 100-row batches; all become closed. |
| `TestGOV021ClosingNetworkRefusesAppointment`, same file | Creates request one day before the scheduled close, advances two days, explicitly asserts it is still within its seven-day lifetime, checks closing-specific rejection, no role grant, approved state retained and new request refused. |
| `TestGOV021OperatorListIsAudited`, same file | Mandatory label/reason, System-only access, represented-network audit and requester subjects, quiet-network exclusion, paging, restricted view, reduced member event and full administrator audit. |
| `TestGOV021AdminRecoveryCLIListPaging` and updated CLI regression | Required flags, limit bounds, continuation and end-of-list behavior; audit counts/reasons and absence of login fields. |
| `TestGOV021AdminRecoveryListsContinue`, new API file | Walks both signed-cursor lists across pages without duplicate IDs, verifies total counts and rejects using one list's cursor for the other. |
| `TestGOV021AdminRecoveryPagePaging`, web | 51 requests / 102 steps cross both page limits; continuation exposes the oldest request and remaining steps. |
| Updated `TestGOV021RecoveryRepository`, store | Member-targeted open lookup, no open row for a refused request, expiry cursor boundary and query-plan use of the partial unique index. |

The design document claims author fail-before evidence for the indexed lookup and closing guard. This reviewer did not run those cases or independently repeat those claims because root explicitly held runtime work while policy remains unresolved. No green build or race result is asserted for e11d292 here.

## Remaining readiness decisions

The canonical GOV-021 text, SEC-013 addendum and DEC-014 were not changed in this forward delta. Recovery still creates a permanent ordinary admin role without a break-glass session, while the existing addendum describes the temporary grant as the operator rights-changing path. Two typed labels still do not prove two separate staff. The new “Owner gate” makes these limits more visible, but does not authorize them. Preserve A3-F02 / DEC-014 as unresolved.

The seven-day expiry and potential waiting/objection period also remain owner questions. Existing secondary notes are unchanged: service-vs-schema operator-label normalization is not identical; application full-export treatment of an in-progress recovery remains a portability clarification. No new ledger semantics, permission expansion beyond the already-held recovery design, API route, or migration was added by the corrections reviewed here.

Recommended handoff: retain the policy-held draft, update the readiness ledger to credit the concrete repairs, and avoid repeating obsolete claims that listing lacks any audit or all request/history paths remain bounded to their first page. After the owner chooses and records the authorized recovery/access model, align the canonical/addendum/runbook claims, refresh the dependent stack, independently review the resulting exact tree, and run required qualification on that tree.
