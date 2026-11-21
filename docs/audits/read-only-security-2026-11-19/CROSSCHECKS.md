# Independent challenge reviews

The following source-only reviews challenged selected findings for counterevidence, affected-version equality and overstated impact. Material qualifications are incorporated into the main reports. These are independent worker passes within the same AI-assisted audit, not external human certification. No dynamic test result is claimed here.

## Independent static cross-check: reporting, closure, and recovery

Date: 2026-11-19 America/Los_Angeles (2026-11-20 UTC). Reviewed the peer notes and independently traced the cited Core source at `22c767a90f69d286642ef96e6b3c174836637a3c`. A targeted comparison found the relevant service, repository, and API source unchanged from main `aa01563fcc891da60fefebb8335eca27855678e8`. This cross-check used source/specification/existing-test inspection only. No test, reproduction, service call, or application modification was performed.

### Disposition

| Finding | Independent assessment | Severity qualification |
|---|---|---|
| WAPI-01 | Upheld. Administrative authority is checked, but administrator/subject overlap is not. Both reporter disclosure and self-handling follow from the service source. | Medium confidentiality/governance issue; same-network administrator required. Handling preserves the report and audit trail rather than deleting evidence. |
| LED-R01 | Upheld. Fresh export generation time is inside the hashed JSON bytes and therefore inside the closure checksum. | Medium operational-integrity/correctness issue; normal elapsed time suffices for a false mismatch. This is not a false match or a ledger-writing defect. |
| LED-R04 | Upheld with a narrow concurrency prerequisite. The fresh schedule is read but its due date is not checked in the mutation transaction. | Medium conditional lifecycle correctness; cancellation must precede the old deadline, a replacement must exist, and the retained batch must span that change. No likelihood or runtime reproduction established. |
| AUTH-01 | Upheld after rechecking password-change and MFA boundaries. Resetting access is immediate; obtaining usable account access is a separate condition. | High cross-network confidentiality concern only with the stated access prerequisites. Reset does not itself issue a session or bypass enabled MFA. |

### WAPI-01 — No counterevidence restores reported-party confidentiality

`internal/service/reports.go:134–173` requires an administrator, then returns every selected report with its `ReporterID` and resolved display name. `HandleReport` at `186–226` requires the same role and marks a report handled without comparing the actor to the member target or the listing owner. `internal/api/h_reports.go:25–28,62–74,84–89` returns the identifier through both the queue and action result. The comment on `reportDTO` itself states that the reporter is never shown to the reported member, so this is not merely a discrepancy inferred from a distant product goal. [Report service](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/reports.go#L125-L226); [API representation](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/api/h_reports.go#L7-L28).

Counterevidence limits the claim: ordinary reported members and reviewer-only members cannot open the queue; normal effective-admin and configured MFA guards still apply. Filing and handling do not themselves hide a listing, suspend a member, or move value. A handled report remains available through the handled/all queue and audit; the reporter can file a new report after handling (`reports_test.go:119–123`). The finding is disclosure and lack of independent handling, not deletion or irrevocable suppression of all future complaints.

The spec does contain both an administrator-only audience and an unqualified reported-party exclusion (FR-024 at `04-functional-requirements.md:235–241`; SEC-025 at `09-security-privacy.md:352–358`). Those conditions need to hold simultaneously when a person has both roles. Existing tests separate ordinary members from the administrator and do not cover that intersection (`reports_test.go:26–79,147–168`).

**Repair invariant:** A report subject must not receive reporter-identifying fields or handle that report merely because they also hold administrator authority. Apply a centrally resolved conflict predicate in service reads, mutation transactions, and both durable and adapter replay authorization. Preserve subject identity through target lifecycle changes and role changes; an unavailable target lookup must not silently erase the conflict. Unrelated administrators should retain legitimate review access. Specify the independent handling path when every available administrator is conflicted.

Audit correlation is real supporting evidence: report creation includes report ID and reporter actor/subject (`reports.go:120–122`), and the administrator branch of `ListAudit` returns the full network log (`audit_read.go:45–64`). Redacting one queue column cannot establish the advertised guarantee. Review audit/export/cached representations as a single confidentiality property, while retaining records internally for accountability. Do not promise protection against identity someone voluntarily writes into a free-text report unless the product explicitly undertakes that broader redaction problem.

### LED-R01 — Timestamp contamination is sufficient even with unchanged ledger tables

The source trace is complete: `Export` assigns `GeneratedAt: s.now()` (`export.go:198–199`); `writeJSON` places it in the header written through `countingWriter` (`export_write.go:101–106`); `countingWriter` hashes those bytes (`export.go:243–260`); and `ManifestChecksum` includes the resulting per-file hash (`closure_schedule.go:297–305`). Recording and later verification each call a new JSON export (`311–316,375–381`). The default clock is wall time, truncated to milliseconds (`service.go:166–181`). [JSON header](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/export_write.go#L93-L106); [closure comparison](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/closure_schedule.go#L297-L381).

Counterevidence does not remove the issue: a fixed fixture clock or two exports in the same stored millisecond can yield the same metadata. The existing closure checksum test holds time fixed after finalization (`closure_test.go:84–98`). The `Erased` flag acknowledges a separate legitimate content change after erasure; it does not normalize timestamps before erasure. Even if GOV-025 is interpreted as requiring only verification of one saved artifact, the public `VerifyClosureExport` operation currently compares newly generated bytes with an older checkpoint and exposes that result.

**Repair invariant:** Unchanged committed checkpoint data must yield the same versioned digest after elapsed time and restart. Volatile export metadata must either be excluded from the certified representation or reproduced from persisted checkpoint metadata. Relevant committed-field changes must still affect the digest. Keep ordinary downloadable generation metadata if desired, but distinguish artifact generation from checkpoint identity.

Historical handling needs special care. `SetClosureFinal` persists sequence, checksum and reconciliation totals, not the original export timestamp (`sqlite_closure.go:64–66`). `ClosedAt` is written in an earlier transaction and is not proven equal to the later export time. Do not silently infer the old timestamp, overwrite an old digest, or claim a historical checkpoint verified under a new algorithm. Preserve provenance and explicitly represent legacy/unverifiable cases. Maintain the existing final-export retry behavior and intentional erasure interpretation.

### LED-R04 — Fresh schedule existence is checked; freshness and due-ness are not

The batch query selects all unfinished schedules, including future ones (`sqlite_closure.go:75–78`). `FinaliseClosures` stores those records, evaluates `Due(s.now())` outside the finalizing write transaction, and passes only the network ID onward (`closure_schedule.go:174–207`). `finaliseOne` reads the current network/closure but checks only stored network status, closure existence, and completion (`217–229`). It then cancels proposals and can close the network (`230–278`). Repository network reads do not apply derived lifecycle status (`sqlite_network.go:34–60`). [Finalizer](https://github.com/djshellshoxxx/circula/blob/22c767a90f69d286642ef96e6b3c174836637a3c/internal/service/closure_schedule.go#L170-L278).

The narrow prerequisites matter. A cancellation with no replacement is safe under the current inner `nil` check. A cancellation after the old deadline is refused by `CancelClosure` because the normal service open derives the closing state (`closure_schedule.go:41–47,129–162`). The source concern requires a batch retaining the old record across a permitted pre-deadline cancellation and a replacement whose current due date is still future. The outer saved-record condition can then differ from the current object that is mutated. A long-running batch can make this possible; no occurrence rate or actual scheduling race has been established.

Writer serialization correctly prevents changes inside the finalization transaction, but it does not validate a business predicate evaluated before that transaction. An unresolved dispute may block final closure, yet proposal cancellation occurs before that blocker check (`230–262`), so “the blocker prevented closure” is insufficient to show a future schedule was untouched.

**Repair invariant:** Before the first cancellation, notification, audit event, or closure mutation, the current schedule in the same write transaction must exist, be unfinished, be due under the current clock, and satisfy the relevant current freeze condition. Carry and compare an immutable schedule identity/revision if work is intended to apply only to the batch's selected schedule; replaced or canceled work should become a no-op. Preserve completion idempotency, closed-network final-export retries, and the legitimate due-schedule blocker behavior. Acceptance should assert absence of *all* premature effects, not only unchanged final network status.

### AUTH-01 — Separate global reset, session issuance, forced change, and second factor

The eligibility mismatch remains: the reset guard ignores other `departed` memberships (`recovery.go:389–397`), while own history and export authorization retain their rights (`tx_queries.go:65–83`, `reads.go:124–158`, `export.go:107–145`). `MyMemberships` also returns departed memberships rather than filtering them (`networks.go:198–213`), and the HTTP network wrappers do not impose a general departed-member read ban. All other memberships must satisfy the reset guard; a different non-departed membership would cause refusal.

The four stages must remain distinct in the final report:

1. **Global credential replacement:** `resetPerson` changes the password, sets the forced-change flag, revokes all sessions and API tokens, deletes password-recovery codes, and deletes pending MFA challenges (`recovery.go:296–328`). These effects do not require the resetting administrator to know the login or the target's factor. The reset result returns a temporary password, not a session. The web response contains display name and temporary password, not the login (`h_recover.go:133–157`).
2. **Session access:** A person using the replacement password still needs the correct login. `LoginStep` sends an enabled-MFA person into the challenge path before inserting a session (`auth.go:165–194`). `Login` explicitly refuses to return a password-only session when that challenge is required (`mfa_login.go:20–31`).
3. **Second factor:** Reset does not delete or disable the MFA record. `CompleteMFA` must validate the person's factor or MFA recovery code before issuing a session (`mfa_login.go:84–134`). The network reset cannot be described as bypassing that requirement. Enabled MFA therefore prevents the asserted history access unless the resetting party separately possesses a valid factor. It does not prevent the credential-replacement and revocation effects.
4. **Forced password change:** The web blocks ordinary screens until change (`web/session.go:182–194`). That restriction is not independent owner verification: `ChangePassword` verifies the currently stored password, which is now the returned temporary one, then clears the flag and revokes existing sessions (`auth.go:252–328`). It does not require the superseded password. Keep this as an explanation of why forced change does not establish owner identity, not as a request sequence.

The precise confidentiality claim is therefore: an eligible same-network administrator can replace a global password despite retained protected access in another, departed membership; when the login is known and no unavailable second factor blocks authentication, that replacement can enable access to the target's own retained other-network history. It is not immediate session issuance, universal MFA bypass, all-record access, or departed-member write authority. Preserve those qualifiers when ranking AUTH-01 as High.

**Repair invariant:** Network-local recovery cannot replace a global identity that retains protected authority elsewhere without the appropriate global/owner recovery authorization. Failed eligibility leaves credentials and factor state intact. Preserve both departed-member read rights and the existing MFA challenge, password-proof, forced-change, and revocation controls; weakening those controls is unnecessary to fix the boundary.

## Independent challenge review — operations, ancestry, listing CAS

Date: 2026-11-19 America/Los_Angeles (2026-11-20 UTC). Static source review only. No application edits, tests, new reproductions, remote writes, or live calls.

Pins: main `aa01563fcc891da60fefebb8335eca27855678e8`; candidate `22c767a90f69d286642ef96e6b3c174836637a3c`. A direct Git comparison found no changes between these revisions in the relevant production files listed below. The cross-check therefore supports both-pin attribution, not merely candidate attribution.

### Verdicts

| Finding | Verdict | Material qualification |
|---|---|---|
| OPS-01 — platform audit chain omitted from routine verification/anchors | Retain Medium integrity-monitoring gap | Requires storage corruption or database write authority for the claimed tampering consequence. Restore explicitly checks the platform chain. An optional correctly operated audit mirror offers separate evidence. |
| OPS-02 — synchronous mirror holds writer gate after commit | Retain Medium, deployment-dependent availability | It retains the in-process store gate, not SQLite's transaction write lock after COMMIT. Other processes and pure read units are not inherently blocked by that gate. Context cancellation can end queued callers, but cannot end the already blocked mirror write. |
| AUTH-02 — ancestry traversal succeeds on incomplete walk | Retain depth-bound finding; qualify cycle impact | Depth exhaustion can omit a prohibited ancestor. A short cycle alone does not show that an ancestor was missed: any proposer actually traversed is checked before repetition stops the loop. Keep cycle/missing-root handling as an indeterminate-provenance hardening case, not a separately demonstrated relationship bypass. |
| WAPI-02 — listing conditional write outside transaction | Retain Low integrity/concurrency defect | The same API handler instance serializes API contenders. The gap requires another authorized writer outside that mutex, such as the owner's HTML edit. The current ETag hashes editable content as well as time; do not add a timestamp-only collision claim. |

### OPS-01 — source trace and counterevidence

`internal/service/audit.go:40-45` places platform audit events in the empty network scope. `cmd/circula/snapshot.go:139-157` enumerates `OperatorNetworks` and invokes `VerifyLedger` for real network IDs. `internal/service/verify.go:41-90` verifies the requested network, including its chain at lines 77-79. Startup and scheduled checks use that enumerator (`cmd/circula/serve.go:169-177,459-468`); the CLI and backup callers also point to it. No production empty-scope verification caller was found except restored-credential invalidation.

`cmd/circula/auditanchor.go:75-102` obtains heads by joining `networks` to `audit_events`, which omits the empty platform scope. The anchor comparison at lines 139-151 examines only recorded network heads. `anchorDBID` at lines 59-71 hashes a selected event ID; it does not commit to platform event content or the remainder of the platform chain. It is not substitute coverage for that chain.

Counterevidence is real and should remain visible: `internal/service/restore_credentials.go:25-45` appends a platform restore event and verifies the platform chain before committing, rolling back on reported inconsistency. Further, `cmd/circula/auditmirror.go:55-65` emits each committed event without filtering out empty network IDs; when configured and independently retained, that mirror can supply additional forensic evidence. The optional mirror does not change what a routine verification or anchor result actually covers.

The existing OPS-01 text is appropriately bounded. Avoid language such as “audit verification is bypassed by any user,” “all audit integrity is broken,” or “restore never checks platform history.” This is missing assurance for one important scope. No new prerequisite for obtaining database write access was found.

### OPS-02 — source trace and scope correction

`internal/store/sqlite.go:140-147` acquires a per-`SQLite` writer gate and releases it only when `s.run` returns. `sqlite.go:221-229` commits, sets `done=true`, and synchronously invokes the post-commit sink before returning. `cmd/circula/serve.go:148-154` installs `auditMirror.write` when configured. The sink calls `os.File.Write` under its mutex without a context or write deadline (`cmd/circula/auditmirror.go:47-65`). Its own comments explicitly list log-shipper pipes as an intended destination (`:14-24`). The sink failure branch runs only after the write returns.

This supports the reported availability risk for a stalled configured destination. It also identifies precise limits:

- The database mutation is committed. A caller's delayed or lost acknowledgement must not be described as a rolled-back or half-posted ledger operation.
- COMMIT releases SQLite's transaction lock. The remaining gate belongs to that store instance; another process is not held behind this application gate.
- `View` bypasses the writer gate (`sqlite.go:150-156`), and service reads delegate directly to it (`internal/service/service.go:217-218`). Pure reads may continue. Some HTTP authentication/read flows may perform writes, so neither “all reads continue” nor “the entire database is locked” is justified without tracing those flows.
- Waiting `Update` calls can return on their own canceled contexts (`sqlite.go:140-145`). That does not release the first caller's blocked context-free mirror write or the gate it retains.
- This depends on enabling the optional mirror and an actual stalled destination. A local regular file with prompt failure is not evidence of indefinite blocking, and no failure duration or workload threshold was measured.

Bounded asynchronous delivery or an explicit durable outbox remains appropriate, provided event ordering, bounded resource use, failure visibility, and shutdown policy are specified. The existing “best effort” comment describes lost lines and returned write errors; it does not make a synchronous file write nonblocking.

### AUTH-02 — retain the bound defect, narrow the cycle claim

`internal/service/dualcontrol.go:19-53` caps traversal at 32 iterations and returns nil after loop exhaustion or a repeated node. Inside each iteration it compares the parent to the proposer at lines 48-49. `ApproveAdjustment` treats nil as permission at lines 82-90. The grant lookup uses the latest historical grant (`internal/store/sqlite_member.go:175-187`). `internal/service/membership.go:164-216` and the role schema at `internal/store/migrations/0001_init.sql:73-83` have no maximum ancestry-depth enforcement, so no write-side invariant proves that all legitimate chains fit the traversal bound.

The concrete depth issue remains: a prohibited relationship beyond the visited edges is not evaluated, and exhaustion is accepted. The relevant source does not require raw database access to have long legitimate grant history, although this review constructed no history and makes no production-frequency claim.

For cycles, the report should distinguish incomplete provenance from an actually missed proposer. Every visited edge is checked before the next loop condition. If the proposer is in a short encountered cycle, the parent comparison encounters it before repeating a node. A cycle excluding the proposer establishes no path to that proposer in the current one-parent graph; its lack of a trusted root is a separate reason to reject or flag inconsistent provenance, not by itself proof of the stated descendant relationship. Role revocation/regrant and selection of only the newest edge complicate historical ancestry, but this cross-check did not establish an additional dynamic-history bypass.

Permitted trust roots and exceptions are explicit: temporary break-glass administrator at lines 27-29, founder self-grant at 41-42, and an operator-rooted empty granter after the applicable wait. Do not convert those approved semantics into findings by applying a generic “all empty parents fail” rule. Existing direct/short-chain refusal and founder-direction tests at `internal/service/a3b_test.go:34-89` support these distinctions; they were read, not run.

Recommended final framing: “Depth exhaustion accepts an incomplete independence check; cyclic or unexplained missing-root histories also lack an explicit failure result and need defensive handling.” Keep this separate from the accepted one-direction policy AUD4-F01. Medium remains reasonable for the depth case given privileged actors and unusual history; cycle impact should remain qualified.

### WAPI-02 — valid gap with a real compensating mutex

`internal/api/h_listings.go:99-113` locks `casMu`, reads the listing, and checks the supplied validator; its call to `UpdateListing` occurs at 151-156. `casMu` belongs to an API server instance (`internal/api/api.go:49-59`). It is a real compensating control against concurrent listing PATCH handlers using that instance, so the finding should not be described as an unconstrained race between those handlers.

`internal/service/listings.go:116-124` carries no expected revision, and `128-188` opens a later write unit, checks ownership, loads current state, and applies the update without rechecking the validator. The owner's HTML edit calls this service directly (`internal/web/h_home.go:416-435`), outside the API mutex. Store writer serialization protects each write unit but does not extend backward over the adapter's already completed read. Thus an intervening authorized edit can make an explicit validator stale without preventing the later conditional write.

The current ETag includes ID, update time, status, title, description, and price (`internal/api/dto.go:155-160`). It is not merely a millisecond timestamp, and source does not support adding an ordinary changed-field timestamp-collision finding. `internal/api/decode.go:201-219` also deliberately accepts wildcard existence semantics; a wildcard need not detect intervening edits, so the report correctly confines the gap to explicit validators.

Ownership and archive-finality guards remain inside the service transaction (`listings.go:135-147`). Only overlapping fields can lose an intervening value; untouched fields are taken from the newly loaded record. Retain Low severity and the owner-only data-integrity scope. No other member's edit authority or ledger value movement was established.

### Equality and verification record

Git source comparison covered the four operations files (`auditanchor.go`, `auditmirror.go`, `snapshot.go`, `serve.go`), service audit/verification/restore/dual-control/membership/listings/runtime wrappers, store transaction/member code and role schema, API listings/server/DTO/If-Match code, and HTML listing update. No relevant production difference was found between the pinned SHAs. The detached worktree remained clean. No dynamic validation, exploitation, or test result is claimed by this cross-check.
