# Security and integrity review

> Closing publication: [the final addendum](CONSOLIDATION_FINAL_REPORT.md#closing-publication-checkpoint-2026-11-19) records Core992, isolated follow-ups, the timestamped 125-ref snapshot and pending gates. The [combined core review](CORE_COMPOSITION_20261119.md), [frozen 123-ref snapshot](CONCURRENT_BRANCH_SNAPSHOT_20261119.md#bounded-refresh--112708-utc) and earlier execution receipts retain their original scope. Current PR heads and checks govern subsequent qualification.

Review date: **2026-11-18 Pacific / 2026-11-19 UTC**. Baseline `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`; core candidate [PR #90](https://github.com/djshellshoxxx/circula/pull/90). This is an engineering review with independent AI-assisted reviewers, not the external SEC-033 release assessment.

## Core findings and repair status

“Repaired” below describes the source change. Final qualification is the current-head result in [CONSOLIDATION_FINAL_REPORT.md](CONSOLIDATION_FINAL_REPORT.md); a pending runtime gate is never represented as passed.

| ID | Finding and practical effect | Repair and regression | Independent review |
|---|---|---|---|
| CORE-01 | Reconstructing dispute policy by wall time lets a same-millisecond or backward-clock policy revision change an existing dispute's response/re-review deadlines. PR #87 fixed ordinary forward time only. | Persist `disputes.policy_version` in the serialized opening transaction; exact-version reads; migration 0019 with mapping failure rollback and immutable update/REPLACE guards. `TestDisputeTimelinesSurviveEqualOrReversedPolicyClocks` plus migration/contract/guard tests. | Ledger reviewer approved the actual diff and legacy-default behavior; SQL migration/trigger checks executed. Full Go runtime gate required. |
| CORE-02 | The overflow fallback for network totals stopped accumulating after an overflowing prefix. Valid individually bounded positive/negative balances could cancel to zero yet be reported corrupt, potentially freezing a healthy network. | Exact `big.Int` accumulation on the exceptional path, final-only int64 clamp, row-error check. Eleven signed-order/residual/boundary cases; `TestVerifyLedgerAcceptsLargeBalancedTotals`. SQL reproduction demonstrated zero/one exact totals despite SQL prefix overflow. | Security reviewer approved source and real-ledger fixture. Monetary float guard is retained and narrowed to exact integer APIs. Runtime gate required. |
| CORE-03 | The web password form reported durable service throttling as validation HTTP 422. The durable budget can exhaust before the adapter budget. | Explicit `CodeRateLimited` → HTTP 429. `TestPasswordChangeDurableThrottleStatus` covers the lower service budget, unchanged password, finite expiry and subsequent success. | Security reviewer checked the branch, budget ordering and clock/session lifetime. |
| CORE-04 | Successful reset/recovery left the old password's exhausted proof-attempt budget intact, blocking the mandatory password change for the recovered user. | Clear `recgen` only inside successful administrator/operator/code-recovery transactions. Regression covers all three modes, denied-reset lock retention, forced-change flag and revoked sessions. | Ledger reviewer verified authorization, atomicity and unchanged unrelated HTTP budgets. |
| CORE-05 | Historical policy-log readers decoded old JSON into zero values while execution readers seeded defaults, displaying false changes and inconsistent effective settings. | Seed `domain.DefaultPolicy()` for both current and previous log entries. Regression upgrades a v18 fixture, compares full/paged logs with network reads, preserves explicit zero and checks unchanged stored JSON. | Security reviewer checked page/predecessor semantics and no data rewrite. |
| CORE-06 | Network setup used unvalidated decimal precision when formatting a rejected amount's example. A large authenticated input could control `strings.Repeat` allocation; malformed integer input was silently converted to zero. | Check integer parsing and `0..MaxDecimals` before any amount formatting; accessible select error. Regression uses safe moderate `4096` and malformed/negative/out-of-range/overflow inputs and asserts no network creation. | Ledger reviewer approved guard placement, bounded regression and ARIA references. |
| CORE-07 | The initial session inventory paginated its response but materialized all retained owner sessions; confirmation and revocation also scanned the set. API-036 requires bounded listing queries. | Migration 0020 adds immutable random public selectors and an owner/time/selector index. Each request uses at most 20 indexed reads of 100 retained rows, with usable empty-page continuation. Confirmation/revocation use direct owner lookup. Legacy signed cursors are invalidated; credentials and lifetimes are preserved. Persistence, work-cap, equal-time, owner and HTML/API regressions added. | Ledger and root independently approved storage/service and adapter source; actual production-service race tests and five template states passed. Real SQLite seek/guard evidence passed; fresh full provider qualification remains required. |
| CORE-08 | The existing API token inventory truncated its response to 100 after an unbounded owner query, violating API-036 as retained tokens accumulated. | Apply LIMIT 100 in the repository and use the migration-0020 owner/ID index. Preserve ascending IDs, retained metadata, the unpaginated response and direct revocation beyond the cap. Repository and API regressions cover limits, order, owner isolation and credential privacy. | Ledger independently approved the source and regression semantics. Extracted production SQL with all 20 migrations used the owner index and exactly 1,713 VM steps for both 270 and 27,000 retained-token fixtures; integrity and owner revocation checks passed. Full Go/HTTP qualification remains pending. |

The candidate also preserves PR #82's credential-proof protections: password changes are durably throttled; login/session issuance, MFA enrollment/disable/code replacement and password-recovery code generation recheck the password material in their serialized mutation. A password change/reset that wins the race invalidates the old proof. The complete inherited source and its original review remain in [AUD4-AUTH-FIXES.md](../audit/consolidation-completion-2026-11/AUD4-AUTH-FIXES.md).

## Additional composed findings, 2026-11-19

The following repairs are present in the reviewed composition described in
[CORE_COMPOSITION_20261119.md](CORE_COMPOSITION_20261119.md). Their regression
source is retained; full provider qualification of the combined head is pending.
These are defect categories, not a count of independently exploitable vulnerabilities.

| ID | Confirmed defect / effect | Repair and evidence boundary |
|---|---|---|
| CORE-09 | API-token mutation could rely on session/password authority checked before the write transaction. | Revalidate live mutation authority within both token create/revoke write units. Preserve the existing owner/scope contract. Security and ledger reviewers verified both calls and exact PR99 regression blobs. |
| CORE-10 | A failed rollback could return an unusable SQLite connection to the pool. | Attempt cleanup with a background context and discard the connection via driver.ErrBadConn when rollback fails; preserve the original operation error or panic. Retain real SQLite fault regressions and their source-specific receipts. |
| CORE-11 | Migration row iteration errors and failed COMMIT cleanup could leave incorrect completion state. | Check rows.Err and perform rollback/discard cleanup after failed commit. Current-head Go-driver and fault gates remain required. |
| CORE-12 | New export and unrecorded consistency-job identifiers did not follow the documented UUIDv7 contract. | Allocate before export side effects and use UUIDv7 for new records; preserve existing identifiers and addressability. Source-pinned adapter tests are recorded separately from full qualification. |
| CORE-13 | Export admission and the later data snapshot could observe different membership, role, MFA, break-glass or erasure authority. | Shared authorization runs inside the data view before the first sink write. Eligible suspended/departed own-history access remains. Revocation/erasure/retained-history regressions are preserved; no continuous post-snapshot cancellation guarantee is added. |
| CORE-14 | An interrupted export could be finalized or followed by an HTML error, misleading a client about archive completeness. | Propagate the HTTP abort sentinel through web/operator recovery. Preserve PR102's real-client regression that receives ZIP bytes before the injected failure and then observes an incomplete response/archive. |
| CORE-15 | Archive verification could accept ambiguous membership, unsafe metadata or data inconsistent with declared sizes. | PR104 rejects duplicate/reserved/unsafe members and declarations, validates schema and file types, caps allocation and verifies actual lengths/content. All 22 Python reader tests pass; Go verifier and combined driver/provider gates remain required. |
| CORE-16 | Listings advertised an extra page for exact terminal batches and could lose active filters. | Fetch a real lookahead row, keep the 50-row page contract, and preserve ownership/type/trimmed encoded search in the complete escaped next URL. Focused marketplace template tests pass; current-head handler/browser tests remain required. |
| CORE-17 | Regenerating traceability could conceal duplicate requirement definitions. | Reject duplicates during the check workflow while retaining diagnostic generation and valid repeated references. Actual traceability race tests pass at the composed source. |

## API-036 focused completion review, 2026-11-19

[PR #101](https://github.com/djshellshoxxx/circula/pull/101) preserves the concurrent
PR90 bounded inventory and HTML cursor fix, then completes selective bulk
revocation and previous-page navigation. Independent database, security and
adapter reviews verified published implementation `6f8ead1951f2d4b72649909b2b4cc83000948afe`
against exact tree `a56e533e88275c05498c39db2c97c5f15b2af2fa`; no unresolved
blocking source finding remained. The [completion record](API036_COMPLETION.md)
contains scope, provenance, defects, measurements and qualification limits.

Confirmed additional defects now have regressions: noncanonical/oversized signed
cursors; raw numeric-position error allocations proportional to input; malformed
HTML query escaping discarded by convenience parsing; dynamic pager context
losing a composite cursor; and disappearing newer pages skipping a surviving
boundary. Forward termination is explicitly preserved. Encoding aliases did not
bypass HMAC verification or ownership.

Bulk revocation rechecks the current stored session and owner within its write
transaction, preserves that credential and foreign/API-token authority, clears
pending owner MFA challenges, and atomically writes only person/count/current
disposition audit data. Review covered CSRF, confirmation, effective role
lifetimes, expired/revoked/missing targets, audit failure and concurrent
revocation. Actual focused production tests passed; broader execution gates and
external SEC-033 release review remain separate.

## Separate future-module and topic repairs

| ID | Scope | Finding / correction | Qualification boundary |
|---|---|---|---|
| FUT-01 | FNET-204 feed | Nil/zero cursor codec could panic. Refuse unavailable codec safely; preserve signature, rotation, expiry and actor/filter binding. | Isolated module race suite independently rerun, 98.8% statement coverage; no core activation. |
| FUT-02 | FNET-204 feed | Conflicting duplicate source records made cursor pagination depend on input order. Reject conflicting records in either order. | Fail-before regression, repaired race suite and 20-second `FuzzOpenCursor` (66,869 executions) passed. |
| FUT-03 | FUX-104 notifications | Pipe-joined encryption context allowed distinct accepted `(network, member)` tuples to share authenticated data. | Use unambiguous versioned encoding for ambiguous tuples, preserve safe legacy rows, never fall back to ambiguous legacy context. Security reviewer approved actual code/tests; module qualification recorded separately. |
| FUT-04 | FUX-104 notifications | Invalid UTF-8 identities could normalize through JSON and collide in deduplication/signed-unsubscribe representations. | Reject invalid UTF-8 at relevant identity boundaries; tests cover all tuple components and valid Unicode. Isolated module only. |
| FUT-05 | FNET-209 federation | `Sign` reported success after malformed UTF-8 was normalized by JSON; the returned envelope did not preserve the caller's strings. Original `Verify` already rejected it. | [#105](https://github.com/djshellshoxxx/circula/pull/105) rejects all five invalid string fields before marshaling. Independent 30-case parent-source reproduction, exact-source race (1.146s, 87.5%) and vet pass; no reproduced verifier or replay-identity bypass. |
| FUT-06 | FUX-104 notification transports | A successful push could return a non-nil interface containing a typed nil; Retry-After arithmetic could overflow; a custom SMTP dialer needed explicit verified implicit-TLS negotiation. | Reviewed #103 implementations and real HTTP/TLS/SMTP regression source are preserved in #100 at `98852afb`. Combined-head runtime and dependency checks remain required; module adoption stays held. |
| TOPIC-01 | SEC-026 | Default-enabled provisional thresholds, hidden member policy changes, undersized checkbox and future-dated records counted after clock rollback. | Independent branch defaults off, shows settings in member history, uses existing 44-pixel control layout and counts only `(since, now]`. Reviewed; unresolved subject-visibility policy prevents main integration. |
| TOPIC-02 | GOV-021 | Existing administrator-recovery page absent from the browser accessibility fixture registry. | Add the existing page fixture on the GOV-021 branch only. This does not resolve the recovery authority conflict. |

### Closing federation and dependency evidence

The independent #105 review used exact source
`059b0e4508a6c4f8e29f6a4cccd06f92b14ea115` (tree
`4d9f608041e65a2a1a297f17614438790240bc04`).
[`TestSignRejectsMalformedUTF8`](https://github.com/djshellshoxxx/circula/blob/059b0e4508a6c4f8e29f6a4cccd06f92b14ea115/future/federation/envelope_utf8_test.go)
tests five fields against six malformed encodings. Applying these regressions
to only the original parent `envelope.go` reproduced false `Sign` success in
all 30 cases; each envelope was rejected by the original verifier.
`TestVerifyRejectsMalformedUTF8Representations` also passed before the repair.
The exact repaired source passed `go test -race -count=1 -cover -timeout=30s ./...`
with a 1.146s package result (2.530s command wall time) at 87.5% coverage,
plus `go vet ./...`. This independent run includes
the valid-Unicode fixture and fuzz seeds, not another fuzz campaign. Its
`runtime-receipt.json`, counterfactual and race/vet logs are retained in the
checksummed handoff archive `security-federation-review-059b0e4.tar.gz`.
[#106](https://github.com/djshellshoxxx/circula/pull/106) at `d5555440` corrects
only the README's workflow-isolation claim. Federation remains disabled/HOLD;
durable replay, transport, trust and core-adapter acceptance are unfinished.

The official [GO-2026-5932 record](https://vuln.go.dev/ID/GO-2026-5932.json)
affects the OpenPGP family within the unchanged `golang.org/x/crypto v0.58.0`
requirement, not every package in that module. Its recorded `x/crypto` range
ends at `v0.58.1-0.20261009181718-2f1f834330b2`, while the separately extracted
`golang.org/x/crypto/openpgp` module remains affected at all versions; that
boundary is not an OpenPGP safety fix. The
[notification composition note](https://github.com/djshellshoxxx/circula/blob/98852afbd1cd2e31a38d6405b6eee36ad3d7e737/docs/audit/consolidation-2026-11-17/source-reviews/NOTIFICATION-TRANSPORT-COMPOSITION.md)
preserves the scope. Historical job135447099448 reported zero called and zero
imported-package vulnerabilities plus one unnamed required-module advisory.
The official lookup does not retroactively identify that unnamed warning or
certify current reachability. Offline attempts stopped before analysis on
unavailable metadata. The current verbose provider scan remains required;
no dependency bump or warning suppression is claimed.

## Existing protections preserved and checked

The review traced same-network lookups and not-found behavior; session/API-token scope separation; current authorization for replayed cached responses; CSRF on unsafe cookie routes; strict JSON/query validation; contextual template escaping; generic sign-in/recovery failures; immutable ledger/history triggers; serialized capacity checks and atomic posting/audit/idempotency; bounded input and pagination; private temporary files; authenticated backups; export staging, expiry, revocation and erasure invalidation. Relevant regression families are indexed in [TRACEABILITY.md](../TRACEABILITY.md) and the current CI suite. Inspection of these controls is not a penetration-test certificate.

No exploit against a live deployment was attempted. All new fixtures are synthetic. No historical financial rows, passwords, personal records, production configuration or release artifacts were modified. Repository hygiene and redacted secret scanning remain CI gates; the baseline provider vulnerability/secret checks passed on the baseline, not automatically on this new candidate.

## Unresolved decisions with security consequences

1. **Reviewer recovery (AUD4-F02 / #83):** choose and enforce an invariant that survives role removal and regrant. A present-role-only reset guard is insufficient. Options already recorded include operator-only recovery, required reviewer MFA or decision ineligibility after administrator reset.
2. **Dual-control independence (AUD4-F01 / SEC-011):** independent account names do not establish independent people when an administrator can appoint the approving identity. Select a quorum/rooted-approver/delay mechanism and test appointment/reset/regrant sequences.
3. **Break-glass scope (A3-F02 / DEC-014):** accepted temporary session-bound rights and proposed broader authority need reconciliation before adopting permanent administrator recovery or extending privileged actions.
4. **GOV-021:** authenticate two independent approving operators, settle expiry/objection rules and reconcile the permanent role with SEC-013 decision 7. The topic fixture repair cannot stand in for these decisions.
5. **SEC-026:** decide subject visibility, threshold/disable authority and retention. Default-off repair prevents automatic adoption of provisional behavior but does not approve the feature.
6. **Privacy and operations:** retention of last activity/time zone/challenge reasons and WAL erasure behavior remain recorded questions. Manual screen-reader qualification, reference-hardware thresholds and professional external review remain release gates.

## Historical dispute migration limit

For pre-0019 disputes, backfill freezes the policy selected by the prior timestamp reconstruction. If no applicable version exists, the migration aborts transactionally rather than guessing. If several historical versions are ambiguous because of equal or reversed clocks, the database cannot prove which version the original operation actually observed. Preserve the evidence and use the documented operator review process; do not rewrite ledger balances, outcomes or audit history to manufacture certainty.

## Review process and residual risk

Implementers worked in separate branches/worktrees. Critical ledger and security diffs were read by a different worker; reviewers inspected source and tests rather than accepting implementation summaries. Git data publication compared exact tree IDs with the reviewed local tree. Review does not replace compilation, race tests, migration fixtures, contract validation or browser execution. [REMAINING_BLOCKERS.md](REMAINING_BLOCKERS.md) lists the concrete qualification gaps and responsible next actions.
