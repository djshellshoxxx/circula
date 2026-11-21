# Security handoff reconciliation and password admission

Date: 2026-11-20 UTC. Disposition: pinned source reconciliation and subsequent implementation receipt. The original audit evidence remains unchanged.

## Subsequent AUTH-04 implementation

The admission design below was implemented at local source
`0bd96859034c7341015dbaf56a137f6cb84f5b55` and integrated with the published
PR #127 repair composition at local commit
`025ad9d3b23abd29c99dbe3ea376ff1622844313`. An independent reviewer found no
unresolved finding in that implementation and verified all 28 recorded source
and execution-log hashes against committed Git blobs.

The ordinary application now shares one fixed two-operation admission limit
across service instances, startup hashing and backup passphrase derivation.
There are no queued waiters. Busy admission precedes credential repository
work and durable guessing reservations, propagating the existing retryable
503 response. A ticket covers sequential verification and replacement hashing;
cancellation retains capacity until synchronous work finishes. Credential
formats, historical PHC costs, guessing controls and final password rechecks
remain unchanged. This is an operation-count bound, not a byte or RSS ceiling.

The [implementation description](../dev/password-work-admission.md) records
the full call graph and compatibility limits. The
[execution receipt](evidence/password-admission-20261120/README.md) and its
manifest distinguish overlapping runs: 116 distinct top-level tests and 239
distinct named test/subtest cases passed, zero failed, and the existing opt-in
timing test skipped. Targeted race runs and changed-code lint passed. The
adapted benchmark wiring completed three one-iteration smoke variants; no
hardware-sizing claim follows from them.

AUTH-04's aggregate-admission source gap is implemented. The final combined
branch still requires full qualification before merge. Of the fourteen source
findings, eleven now have substantive implementations in the candidate;
OPS-04 remains a reader-resource source gap, LED-R02 requires a disclosure
decision, and OPS-03 requires a platform/support decision and native work.
The table below retains the earlier source pin so historical and subsequent
evidence are not conflated.

## Scope and evidence

[PR #128](https://github.com/djshellshoxxx/circula/pull/128), source commit `896d69f472652ed54c051ec82efcb7f162d1d2c4`, records a read-only security review against main `aa01563fcc891da60fefebb8335eca27855678e8` and PR #125 `22c767a90f69d286642ef96e6b3c174836637a3c`. Its fourteen findings describe those revisions. The original reports, evidence and checksums under [the audit directory](../audits/read-only-security-2026-11-19/README.md) remain unchanged.

This separate review reconciles those findings with [PR #127 source `21933a3c650690db5a8e1e44cc236d8d1b5cdb24`](https://github.com/djshellshoxxx/circula/tree/21933a3c650690db5a8e1e44cc236d8d1b5cdb24). Ten findings have substantive repairs in that source. At that pin, two source gaps remained; one finding needed a disclosure-policy decision and one needed a platform/support decision followed by native implementation and qualification. The later AUTH-04 milestone above updates that source-gap count.

**Implemented does not mean independently runtime-qualified or closed.** This worker inspected source, existing tests and the original handoff; it did not execute new tests, attack requests, exhaustion experiments or native Windows checks. The integrating owner must attach successful checks for the final exact commit. Earlier CI results and this source review do not qualify a later composition. No additional verified Core defect was established in the bounded challenge review of the ten repairs.

Source paths in the table refer to the exact PR #127 pin above. Earlier repair provenance is recorded in [the Core continuation record](UNIQUE_CORE_CONTINUATION_20261119.md).

## Fourteen-finding disposition

| ID | Disposition at the reviewed pin | Implemented invariant or remaining work |
|---|---|---|
| AUTH-01 | Implemented; final qualification required | `internal/service/recovery.go` refuses network-local reset when the person has any other-network membership, including departed and anonymized memberships. Operator recovery and eligible single-network recovery remain. `recovery_authority_test.go` covers retained authority and MFA. This conservative predicate does not introduce new recovery powers. |
| AUTH-02 | Implemented; final qualification required | `internal/service/dualcontrol.go` allows only a completed permitted founder/operator root. Missing provenance, repeated nodes and exhaustion of the bounded traversal refuse approval. Existing directional independence, wait and break-glass policies remain separate. See `dualcontrol_ancestry_test.go` and its internal companion. |
| AUTH-03 | Implemented; final qualification required | `internal/store/sqlite_member.go` closes member rows and checks `rows.Err()` before role loading and successful return. A partial authority enumeration is no longer returned as success. |
| AUTH-04 | Open at the reviewed pin; subsequently implemented above | `internal/service/password.go` still calls Argon2 without shared aggregate admission. Per-account guessing controls remain effective within their scope. No deployment capacity or exhaustion threshold was measured. See the historical design below and the subsequent implementation receipt above. |
| WAPI-01 | Implemented for subject confidentiality; escalation policy remains | Report subject filtering occurs before pagination; reads, handling, replay and correlated audit views check current authority. Full/audit exports refuse retained report conflicts, including handled reports. Member, listing and transaction subjects are covered. Existing system authority remains. An all-admins-conflicted escalation procedure is still an owner decision, not an invented privilege. See `reports.go`, `sqlite_notify.go`, `audit_read.go`, `export.go` and [report privacy](REPORT_PRIVACY_20261120.md). |
| WAPI-02 | Implemented; final qualification required | `internal/service/listings.go` rechecks `ExpectedRevision` inside the write unit; stale explicit API conditions return 412. Content-sensitive ETags, wildcard/unconditional semantics, ownership and archive finality remain. |
| LED-R01 | Implemented with explicit compatibility limit | `closure_checksum.go` binds version-one closure representation to persisted `ClosedAt.UTC()`. Legacy version-zero and unsupported checkpoints are refused explicitly; their historical hashes are not silently replaced or reinterpreted. |
| LED-R02 | Policy decision required | `transactions.go` and `posting.go` still return exact payer availability in some refusals. The existing specifications disagree about disclosure. Record actor-aware disclosure policy before changing service errors, API detail and web wording; preserve atomic refusal and idempotency. |
| LED-R03 | Implemented; final qualification required | `closure_checksum.go` streams digest-only output through context-aware counting/hash writers to `io.Discard`. It no longer retains the full serialized export in `MemSink` for this purpose. |
| LED-R04 | Implemented; final qualification required | `closure_schedule.go` compares the current immutable announcement tuple and rechecks due state and freeze conditions in the write unit. A canceled or replaced saved candidate cannot finalize the replacement prematurely. |
| OPS-01 | Implemented with explicit legacy scope | `internal/service/verify.go`, `cmd/circula/snapshot.go` and `auditanchor.go` include the platform audit chain in routine checks and version-two anchors. Network counts/freezes remain separate. Version-one signed bytes and their limited coverage remain identifiable. |
| OPS-02 | Implemented for the documented best-effort mirror | `cmd/circula/auditmirror.go` uses bounded FIFO delivery with a size cap, one worker, a nonblocking store callback, observable failures/drops and bounded shutdown. Optional sink I/O no longer retains the application writer gate indefinitely. The mirror remains best effort, not a durable outbox. |
| OPS-03 | Platform/support decision and native work required | `cmd/circula/fileident_other.go` does not enforce private Windows ACLs: parent validation only checks directory type, staging validation is a no-op and link-count checking is permissive. Manual NTFS ACL provisioning remains a prerequisite. Define supported identities, ACL/reparse behavior and native acceptance evidence. Isolated future messaging work is not a Core repair. |
| OPS-04 | Open source gap | Go's `verify_export.go` permits a four-GiB member and materializes bytes before parsed objects; `scripts/read-export.py` likewise retains member contents. Archive correctness repairs do not establish practical member/aggregate/parser memory budgets. Add injectable small limits and bounded/streaming parsing without dropping valid-export compatibility. |

The original QG-01 repository protection policy and QG-02 head-specific CI failure are separate from the fourteen source findings. Refresh their live metadata and final-head checks independently; do not copy the original PR #125 CI status onto PR #127. Future notification findings also remain outside this Core total.

## Historical AUTH-04 design — reviewed before implementation

### Smallest coherent change

Use **one shared process admission object for password work**, with a small positive active-operation limit and **zero queued waiters initially**. When full, refuse promptly. This bounds simultaneous work and waiting without introducing a dispatcher, unbounded goroutines or a second scheduler. Admission must be shared by every `Service` instance in the server process; a fresh limiter per request, adapter or service would multiply the advertised capacity. Configuration is fixed before serving. A common internal package can also expose the same object to the backup code.

Acquire one private operation ticket before account-dependent lookup, durable attempt reservation, recovery-code consumption or credential mutation. Public-input validation and already-known actor checks may precede it. A ticket covers the entire synchronous credential operation, including sequential old-password verification and new-password hashing. Pass it explicitly to private helpers; do not reacquire it in `checkCurrentPassword`, `verifyLoginPassword` or the second hash of a change. Do not hide a reusable bypass in `context.Context`, and do not share one ticket across parallel goroutines.

This intentionally holds a slot across short repository units as well as hashing. It is simpler to audit, ensures an overload refusal has not already charged a guessing attempt, and guarantees at most the configured number of credential operations can reach Argon2 simultaneously. All hashing must continue to occur outside SQLite write units.

The existing `domain.CodeUnavailable` is a suitable opaque retryable capacity result. It already maps to HTTP 503; confirm `Retry-After` behavior in both adapters and update its currently database-specific comments. Do not translate overload into an incorrect-password result or into the per-account `CodeRateLimited` result. A rejected operation performs no credential mutation and does not reserve an authentication guess. Existing admitted-attempt reservation, per-account/global guessing bounds, recovery-code consumption and final password-proof rechecks remain unchanged.

### Coverage and error propagation

The coverage inventory is the production call graph, not just HTML login:

| Entry point | Required coverage |
|---|---|
| `New` | Dummy PHC generation uses the shared admission path; constructor errors propagate. Nonblocking admission needs no new wait or background hash goroutine. |
| `registerPerson` | Registration, duplicate-registration paths and operator provisioning. |
| `LoginStep` | Known, unknown, malformed and unusable-hash cases use the same identity-neutral admission rule. Admitted unknown/unusable cases retain their dummy Argon work. |
| `ChangePassword` | One ticket covers current-password proof and new hash; retain final stored-hash comparison. |
| `GenerateRecoveryCodes` | Admission precedes its durable proof-attempt reservation; retain the current-hash recheck. |
| `RecoverPassword` | Admission precedes account/code-dependent work; retain the cheap code check before the new hash and transactional code consumption afterward. |
| `AdminResetPassword`, `OperatorResetPassword` | Temporary-password hashing is covered without changing authority or MFA. |
| `checkCurrentPassword`, `DisableMFA` | Cover MFA enrolment/confirmation and disable flows, including their durable proof budgets and current-hash checks. |
| `cmd/circula/backupcrypt.go:deriveBackupKey` | A second production Argon2 call exists here. Scheduled backup passphrase derivation can share a server process with authentication, so use the same admission object if claiming a process-wide Argon bound. Preserve backup framing and parameters. Separate CLI processes have separate budgets. |

Make the private hashing path context/error-aware, including a `(bool, error)` distinction for password verification. Thread cancellation and admission failure through every caller rather than collapsing them to `false`. Prefer a private ticket parameter that makes bypassing admission an explicit code error; leave only the actual cryptographic call at the bottom of the stack. The startup path and direct helper tests/benchmarks must be adapted deliberately.

### Cancellation, equivalence and existing seams

Check context cancellation before acquisition, immediately after acquisition and before starting each hash. Argon2's synchronous function is not interruptible through this API: if cancellation arrives during a hash, retain its ticket until computation returns, then propagate cancellation before issuing credentials or starting a protected write. Never release capacity while an abandoned background goroutine still computes. Release the ticket exactly once on every return path. An attempt already reserved before a later cancellation remains charged; do not refund shared throttle state using a stale counter.

The zero-waiter design has no queue cancellation or queue fairness problem. Admission treats account identities identically; it does not guarantee request-order fairness under sustained load. If a queue is subsequently justified, it needs an explicit finite length, wait deadline and cancellation removal. Do not add an unbounded channel or a goroutine per waiting request.

Preserve the current known/unknown work-equivalence contract for admitted attempts, including the dummy fallback for unusable stored PHCs. Existing valid historical PHCs retain their recorded cost. Saturated responses must not disclose whether an account exists. Keep overload/cancellation metrics separate from incorrect credentials and avoid identity labels.

The test-only `argonObserver` is **reentrant**: `aud4_auth_test.go` and `password_proof_test.go` synchronously perform a legitimate password change/reset while observing another proof. Never invoke it while holding an admission bookkeeping mutex. Those race tests must inject an admission capacity that permits the two overlapping operations, keeping both subject to normal accounting. A capacity-one accounting test should use a nonreentrant inert hash seam instead. Do not bypass the limiter for nested service calls, weaken the stale-proof assertions, or move the observer after the protected write merely to make tests pass.

### Sizing and honest bounds

The default PHC requests 64 MiB of Argon memory, but `parsePHC` accepts historical memory parameters up to two GiB. An operation-count limit therefore bounds concurrency; it is **not** a promise that every admitted hash costs 64 MiB or that a given container's total RSS is safe. Preserve Argon strength and existing PHC compatibility. Select and document the positive concurrency limit using benign measurements, actual process memory headroom, the largest supported stored parameters and backup work. Do not silently reject or downgrade historical credentials to fit an assumed memory budget. DEC-006 minimum-hardware qualification remains separate.

A stricter byte-weighted admission budget can follow if required, but it needs an explicit compatibility plan for a single historical hash larger than the configured budget and ordering that does not consume guesses merely while waiting for capacity. Do not claim this first operation-count patch supplies that additional guarantee. The verified AUTH-04 defect is the absent aggregate admission bound; the proposed first milestone closes that specific invariant without inventing new password policy.

### Required benign qualification

Use a deterministic inert hashing seam and small bounded fixtures to establish the active cap, prompt full-capacity refusal, absence of retained waiters, exact release on errors and shared accounting across two service instances and backup work. Test cancellation before admission, between sequential hashes and while an inert computation is held; a canceled in-flight computation must still occupy its slot until it returns.

Inspect every production hash call site after migration. Run the existing enumeration-equivalence, durable guessing-budget, stale-password-proof, recovery/MFA and API/web error-contract suites under explicit test capacity. Keep their original behavioral assertions. Add ordinary capacity-refusal assertions showing no session, challenge, password replacement or recovery material is issued and no guess is reserved before admission. Then run the race detector and complete final-head CI. No exhaustion experiment or exploit reproduction is needed.

This section records the design reviewed before implementation. The subsequent
milestone above attaches the implementation, focused regression results and
independent review to an exact source commit. Full combined qualification
remains a separate integration requirement.
