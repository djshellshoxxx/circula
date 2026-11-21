# Circula: read-only adversarial security review

**Review date:** November 19, 2026, America/Los_Angeles (November 20 UTC).
**Disposition:** repair handoff; application and settings unchanged.
**Repository:** [djshellshoxxx/circula](https://github.com/djshellshoxxx/circula).

## Assessment

The review identified **14 current source findings and hardening items: one High, ten Medium and three Low**, plus two delivery/qualification gaps. A separate review records two additional Low pre-integration notification items, excluded from the Core total. Those totals include conditional availability and local-configuration risks; they are not a count of demonstrated attacks. No Critical finding was established. The immediate concerns are a network administrator's recovery authority over a global identity, confidentiality for reports about administrators, incomplete independence checks, and the reliability of closure checks and scheduling.

The findings apply to both pinned Core revisions below. Main and the candidate share the same production service, storage and domain code; the candidate does not resolve these findings. Existing controls provide substantial protection, including transaction-scoped posting, current role checks, session-bound CSRF, strict API parsing, credential invalidation during restore, and authenticated backup framing. Follow-up work should preserve these controls.

This was an AI-assisted, multi-worker source review with independent challenge review of selected findings. The work inspected source, existing test code, repository metadata and selected benign baseline checks. It contacted no deployed Circula service, exercised no production accounts or databases, and created no exploit or new vulnerability reproduction. The only repository changes in the handoff are these audit documents and supporting evidence. No deployment, merge, release approval, or repository-setting change is included.

## Revisions and scope

| Subject | Exact revision | Interpretation |
|---|---|---|
| Main | [`aa01563fcc891da60fefebb8335eca27855678e8`](https://github.com/djshellshoxxx/circula/tree/aa01563fcc891da60fefebb8335eca27855678e8) | Baseline for the notes-only branch |
| Consolidated Core candidate, PR #125 | [`22c767a90f69d286642ef96e6b3c174836637a3c`](https://github.com/djshellshoxxx/circula/tree/22c767a90f69d286642ef96e6b3c174836637a3c) | Primary source-reading pin; candidate tree `9c1ad082bfd7befb9914000c79d5974f1017cc27` |
| Held administrator-recovery candidate, PR #107 | [`b8cb63bfc6d2a969d972f55937cdc2070d5eba34`](https://github.com/djshellshoxxx/circula/tree/b8cb63bfc6d2a969d972f55937cdc2070d5eba34) | Separate policy-HOLD review, not Core behavior |

Main contains 1,442 tracked files and the candidate 1,719; both contain 283 production Go files and 21 migrations. The remote-head snapshot inventories 145 branches. Fifteen canonical future-feature manifests declare `enabled: false` and `integration: HOLD`; neither Core pin contains `future/`. These are inventory counts, not claims of semantic review of every file or historical branch. Detailed coverage and excluded areas are recorded in [COVERAGE_AND_VALIDATION.md](COVERAGE_AND_VALIDATION.md), each lane report, and the [branch inventory](evidence/branch-inventory.json).

The observed main and PR #125 heads remained at the pinned revisions when refreshed during final assembly. GitHub qualification/settings evidence was refreshed at 2026-11-20 06:21:03 UTC (November 19, 22:21:03 Pacific). Later commits require a difference review; the notes make no assertion about unexamined revisions.

## Findings and worker ownership

Severity expresses the stated consequence and prerequisites. **Source-confirmed** means the reviewed control flow supports the finding; it does not mean an attack was executed. Availability thresholds, native-platform behavior and deployment reachability remain unmeasured unless explicitly stated.

| ID | Severity | Finding and consequence | Important limit | Suggested owner |
|---|---|---|---|---|
| AUTH-01 | **High** | Network-local password reset ignores departed memberships that still authorize private history elsewhere. A local recovery permission can replace a global credential spanning that retained authority. | Requires an authorized network administrator and eligible target. Reset returns no session and preserves MFA. History access also requires the login and successful authentication. | Authentication and membership privacy |
| AUTH-02 | Medium | Approval ancestry traversal returns success at its depth bound, accepting an incomplete independence check. | Privileged actor and sufficiently deep/unresolved history. A short cycle containing the proposer is detected; cycle handling is a separate provenance-hardening case. | Governance / membership |
| AUTH-03 | Low | Shared membership enumeration omits `rows.Err()`, potentially treating an incomplete authority set as complete. | No concrete modernc SQLite failure trigger or externally inducible bypass established. Ordinary cancellation is not claimed as a bypass. | SQLite repository / authentication |
| AUTH-04 | Medium, unmeasured availability | Shared memory-hard hashing has no aggregate admission budget, and HTML failure counters do not reserve in-flight work. | Durable per-account guessing controls exist. Actual exposure depends on ingress and process capacity; no stress test. | Authentication / HTTP capacity |
| WAPI-01 | Medium | A reported administrator can receive the reporter's identity and mark the complaint handled, conflicting with the confidentiality promise. | Same-network administrator authority required. Handling preserves the complaint and audit trail. | Reporting / governance / adapters |
| WAPI-02 | Low | Listing `If-Match` validation precedes a separate service write; an intervening authorized non-API write can make the validator stale without refusal. | The same API instance's mutex is real protection. Remaining scope is the owner's listing and overlapping fields. | Listings / API |
| LED-R01 | Medium, integrity-check correctness | Closure digests include fresh export time; later verification can mismatch an unchanged ledger. | False integrity alarm, not a demonstrated false match or balance mutation. | Closure / export format |
| LED-R02 | Medium, privacy-policy conflict | API refusal details can disclose a counterparty's exact available capacity. | Availability is not necessarily balance alone. Specifications conflict on whether this amount should be shown. | Service errors / API / privacy owner |
| LED-R03 | Medium, conditional availability | Closure digest-only paths retain the whole serialized ledger in `MemSink`. | Requires a sufficiently large legitimate export; exhaustion threshold was not measured. | Closure / export resources |
| LED-R04 | Medium, conditional lifecycle correctness | The finalizer can act on a replacement closure schedule without checking that the current schedule is due. | Requires legitimate pre-deadline cancellation/replacement while a saved job batch remains in progress. No runtime race was reproduced. | Lifecycle / scheduled jobs |
| OPS-01 | Medium, integrity-monitoring | Routine verification and external anchors omit the platform audit chain. | Integrity consequence requires corruption or database write access. Restore does verify the platform chain; an independently retained mirror may supply evidence. | Verification / audit formats |
| OPS-02 | Medium, deployment-dependent availability | Synchronous mirror delivery retains the store's application writer gate after commit. | Optional mirror must stall. SQLite's transaction lock is already released; pure read units and other processes are not inherently blocked by this gate. | Store / audit delivery |
| OPS-03 | Medium, local-configuration dependent | Core's Windows confidentiality depends on manual NTFS ACL provisioning. | Correctly restricted storage addresses the prerequisite. No native Windows execution or remote read path was established. | Windows packaging / storage |
| OPS-04 | Low, local-tool availability | Export verification allows multi-gigabyte members while materializing bytes and parsed data in memory. | Explicit local verifier input, not an HTTP upload vulnerability. No exhaustion test. | Operator CLI / Python reader |

The full reports contain source paths and line ranges, prerequisites, historical relationships, repair direction and acceptance criteria:

- [AUTH.md](AUTH.md): identity, recovery, sessions, MFA, grant ancestry and held administrator recovery.
- [WEB_API.md](WEB_API.md): reporting confidentiality, conditional listing updates and HTTP boundary coverage.
- [LEDGER.md](LEDGER.md): ledger/closure findings, financial policy limits and already-repaired issues.
- [OPERATIONS.md](OPERATIONS.md): verification, mirrors, Windows storage, local readers and delivery controls.
- [WORKER_HANDOFF.md](WORKER_HANDOFF.md): work packets and completion evidence for each repair.
- [CROSSCHECKS.md](CROSSCHECKS.md): independent challenge-review verdicts and qualifications.
- [FEATURES.md](FEATURES.md): separately pinned isolated notification/federation review.

## What should be addressed first

**Recovery and report confidentiality are the first authority/privacy work.** AUTH-01 needs an authority-based cross-network recovery predicate, preserving departed members' legitimate history access. WAPI-01 needs a central report-subject conflict rule enforced in service reads, mutation transactions, replay and report-correlated identity views. Concealing a template column is insufficient. Both fixes need ordinary role-overlap and retained-history fixtures.

**Repair incomplete approval proofs without silently changing quorum policy.** AUTH-02 can distinguish a proven permitted root from exhaustion or inconsistent provenance. The previously documented one-direction independence policy is a separate owner decision. The bounded walk should remain bounded while refusing indeterminate authorization.

**Make closure certification and scheduling reliable before relying on their guarantees.** LED-R01 needs a versioned deterministic checkpoint representation and an honest compatibility story for stored hashes. LED-R04 needs the current due date and schedule identity rechecked inside the transaction before any proposal cancellation, notice or closure. LED-R03 should remove complete in-memory materialization from checksum-only paths while preserving consistent snapshot semantics.

**Reconcile the financial-error privacy contract.** LED-R02 is a disclosure supported by source, but the journey prose requests some of the same detail the privacy matrix restricts. Introduce actor-aware typed errors after settling the rule. Replacing HTML text alone leaves API disclosure unchanged.

**Complete operational and resource controls.** Platform audit coverage should be explicit in verification and anchors. Hash admission and mirror delivery need bounded work policies, observable overload/failure and cancellation semantics. Windows support needs native ACL evidence. Local archive readers need budgets appropriate to parsers that retain multiple representations.

## Delivery and qualification gaps

**QG-01: main has no enforced branch protection or active inherited/repository ruleset in the observed metadata.** The branch response reports `protected: false`, disabled protection and no required status contexts; the rulesets response is empty. This does not permit anonymous writes, but write-capable credentials are not constrained by these review/check gates. Prepare the intended owner-reviewed protection policy. Evidence: [github-observations.json](evidence/github-observations.json).

**QG-02: the candidate's combined lint/security job is failing.** On the reviewed head, [job 136036460029](https://github.com/djshellshoxxx/circula/actions/runs/44660259330/job/136036460029) stops at the scanner-control metadata check. Its expected path/rule/line findings include audit documents absent from the composition, so expected and observed metadata differ. This is not evidence of a newly leaked credential. The later full secret scan in that shell and subsequent formatting, vet, lint and vulnerability steps do not run. Restore accurate positive/negative scanner controls and narrow exceptions, then rerun on the repair commit. Disabling the scanner or exempting all audit documents would not satisfy the handoff.

Other head-specific CI checks passed: Ubuntu tests, browser/accessibility/keyboard checks, API contract/load smoke, cross-builds, reproducibility, fuzz smoke and the Python reader on three operating systems. Native Core macOS/Windows tests were skipped. Cross-compilation and Python portability do not supply native Core execution evidence. These external results were inspected; this audit did not rerun those jobs.

## Validation performed here

On the pinned candidate, the existing domain, shared HTTP and logging Go suites passed. Module verification passed. The **package-level** vulnerability scan reported no vulnerabilities. The existing Python export-reader suite passed **41 tests**, and the message/keyboard helper suite passed **15 tests**. Workflow pin/permission checks passed.

The full source/call-graph vulnerability analysis was killed. Selected existing CLI and API baselines were stopped during `modernc.org/libc` compilation (`signal: killed`), before test assertions. These are incomplete checks, not evidence of application assertion failures. The smaller package scan must not be substituted for a completed call-graph scan. Logs, commands, toolchain provenance and limits are in [COVERAGE_AND_VALIDATION.md](COVERAGE_AND_VALIDATION.md).

## Existing decisions still open

The audit does not count these as new discoveries: reviewer password-recovery authority (AUD4-F02), one-direction independent approval (AUD4-F01), lasting effects under temporary break-glass authority (A3-F02 / DEC-014), voluntary debt settlement (A3-07), broader dual-control action scope (SA2-12), closure/dispute resolution deadlock (A1-04-G2), and legacy stored scale-change disposition. Session rotation on role change remains a specification gap; current role rereading limits its authority impact. Held GOV021 distinguishes operator labels, not independently authenticated human operators, and remains a policy HOLD.

Current code contains several prior repairs: disputes persist policy-version binding; exact aggregate-overflow handling is present; settlement after closure is gated; final-export retry exists; credential proofs are rebound after password changes; and restored bearer credentials are invalidated. Preserve those fixes and avoid reopening superseded descriptions as current defects. The lane reports cite their source and prior disposition.

## Completion rule for follow-up workers

A finding closes when a focused patch implements its invariant, required policy decisions are recorded, relevant ordinary regression checks pass on the resulting commit, and an independent reviewer checks the repair and retained controls. A merge, unrelated passing test suite or earlier audit label is insufficient evidence. Update status with the repair SHA and evidence while preserving the original source pin and limitations.
