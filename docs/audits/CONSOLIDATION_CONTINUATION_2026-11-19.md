# Circula core consolidation continuation — 2026-11-19

## Scope and publication boundary

This continuation inspects the live repository, recovers unfinished core work, merges concurrent compatible work without replacing its history, and repairs additional verified defects. It does not deploy, publish a release, delete remote branches, enable future modules, or choose unresolved governance policy.

Starting main: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.
Starting consolidation candidate: PR #90 at `244e1356201e6477eab4694d82947f2058f3b0d8`.
Concurrent candidates incorporated: `1e2f369a541963540d93c3b7f455b052827839c7` and its session-link correction `f25d10b6b4085b80d7de53ecbb55efffeb9d8969`.
Continuation branch: `integration/core-consolidation-20261119`.

The final provider commit and current CI/merge receipt are recorded on the continuation pull request. Local source hashes below identify implementation provenance; GitHub publication may produce a different commit with the same tree. A source review, SQL smoke, or prior-head CI result is never substituted for final combined-head qualification.

## Discovery and disposition

The [branch matrix](BRANCH_CONSOLIDATION_MATRIX.md), [complete inventory](BRANCH_INVENTORY_2026-11-19.json), [pinned refs](BRANCH_REFS_2026-11-19.tsv), and [dependency graph](BRANCH_DEPENDENCY_GRAPH_2026-11-19.md) cover:

- 113 actual starting remote branch heads.
- 22 recovered local refs.
- All 95 pull-request records, including 36 closed without merging.
- Zero advertised tags.
- 564 observed workflow-run records.
- 2,894 exact branch ancestry relationships.

The normalized JSON is lossless and reconstructs its original detailed representation using the included inventory generator's `--expand-json` option. It contains merge bases, divergence, unique commits, changed files, patch equivalence, exact source hashes, dependency relationships, PR provenance, conflict paths and observed CI.

| Starting remote disposition | Count |
|---|---:|
| Already ancestors of starting main | 52 |
| In the starting PR #90 ancestry, including PR #90 itself | 6 |
| Clean merges producing no additional candidate tree changes | 3 |
| Patch-equivalent historical branch | 1 |
| Historical work recovered or superseded | 23 |
| Future-only streams and supporting topics | 22 |
| Product/interface decisions still required | 6 |
| Total | 113 |

Fresh merge simulations found 24 retained branches that conflict with each target. Against main, 52 were ancestors and 37 other merges were clean; against starting PR #90, 58 were ancestors and 31 other merges were clean. These are properties of preserved branches, not unresolved conflicts in the final consolidation worktree.

PR #49 and #50 no longer have advertised source branches, but their PR heads remain recovered and their SQLite/crypto dependency versions are already present. The generically named recovered local `audit/ledger-features-20261119` includes unapproved GOV-021 and SEC-026 implementations; it is deliberately not merged wholesale.

No additional compatible core implementation was found outside PR #90 and the bounded-session recovery. Competing historical specifications, superseded source and stronger successor implementations remain linked rather than silently discarded.

### Baseline verification

Starting main's latest `ci` run, [44296161788](https://github.com/djshellshoxxx/circula/actions/runs/44296161788), failed. The latest successful whole `ci` workflow found on an ancestor is [44274085093](https://github.com/djshellshoxxx/circula/actions/runs/44274085093), at `b4398074ae4ca72559235ca50242cfcb7a934eec`. This distinction prevents scorecard/dependabot successes or an older PR result from being mistaken for current main qualification.

## Core recovery and engineering

### Bounded session and token inventories

Recovered local session commits `853a38826dd661406536753a865a27ee29a4fd84` and `04413b1a8ba0fe7a4118333fa23ac828f673cea6` supply the missing API-036 conformance:

- Forward migration 0020 assigns independent public session selectors, preserves existing credentials and lifetimes, and retains legacy INSERT compatibility.
- Immutable owner/keyset identities and direct owner-scoped lookup avoid credential-derived selectors and full-list revocation searches.
- Indexed keyset queries return at most 100 retained rows per batch; the service scans at most 20 batches per session-list request.
- Empty pages can carry continuation when retained dead rows exhaust that work budget. Deleted boundary rows do not break continuation.
- HTML and JSON consume bounded pages; the API changes its cursor namespace so old ordering positions cannot be reused.
- Token inventory's pre-existing 100-row API limit is enforced by indexed SQL before materialization. Tokens outside that list remain independently addressable and revocable.

These bounds describe inventory queries. Existing role-specific authentication can still enumerate memberships when computing session lifetimes.

Concurrent PR #90 publication `1e2f369` contained the same session recovery and an equivalent token SQL cap. The continuation preserves that exact head and its stronger token owner/revocation tests, retains the production token query-plan regression, and removes the weaker duplicate token API test. No unique runtime behavior is lost.

A subsequent native CI result exposed a session pagination regression: a dynamic URL prefix caused `html/template` to filter the composite `millis:publicID` cursor as an apparent URL scheme. Commit `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` now uses a literal query context and tests the decoded continuation link on both populated and empty pages. The continuation merges that correction without rewriting its history.

### SQLite transaction and migration failure recovery

Source commits `b0852f0f855b09ffff1d22f25d0ec6ec39548814` and `c237b718e7d739c0a98bd2a46283caf01608cef2` fix three failure paths:

1. A failed ROLLBACK no longer returns a potentially active transaction to the pool. The connection is discarded while the original callback error or panic is preserved.
2. Migration discovery checks `Rows.Err()` after iteration; a late metadata read error cannot masquerade as a complete migration history.
3. Migration statement and COMMIT failures both roll back or discard the connection before returning.

Driver-boundary tests retain the actual SQLite engine. They cover callback error/panic, failed cleanup, a clean subsequent transaction without leaked writes, incomplete migration metadata, metadata/COMMIT failures with successful or failed cleanup, unchanged schema/version, and healthy retry. The SQLite-full fixture rearms its connection-local limit for each attempt because discarded connections are correctly replaced.

These tests do not simulate physical power loss or an error acknowledged after a successful commit. Those fault classes remain separate qualification work.

### Token issuance after credential revocation

Source commit `fe8324fca1e106493eb3e583d7c35ac732db2824` closes the window between HTTP authentication and token mutation. Session-backed token creation and revocation now recheck live session ownership, expiry, revocation and stored password restrictions inside the write transaction. A stale session cannot create a durable replacement credential after logout, password change or account reset.

Trusted direct service callers retain their existing scope contract and must satisfy current stored password restrictions. HTTP token issuance remains session-only. Tests cover active and direct callers, missing/expired/revoked/foreign sessions, forced changes, storage errors, logout, password changes and administrator/operator resets. Refusals must leave tokens, security notices and audit state unchanged.

### UUIDv7 compatibility completion

Source commit `a8c1348e27f994f93d5259e5ebbae01ff42377e4` closes API-005/D-14. New export jobs and the unrecorded consistency-job fallback allocate UUIDv7 identifiers. Existing record IDs, including UUIDv4 jobs, remain unchanged and addressable. Export identifier allocation happens before export side effects. OpenAPI and API documentation describe this compatibility boundary.

### Previously completed work retained

The starting candidate already preserves source PRs #82, #87, #78, #88 and #89, including password-proof race fixes, immutable dispute-policy binding, exact ledger aggregate arithmetic, historical policy defaults, recovery throttle correction, setup precision validation, account session UI/API, onboarding message catalogues and bounded CI fuzz budgets. This continuation preserves that work rather than claiming to have newly authored it.

## Validation and evidence

The [continuation evidence manifest](evidence/continuation-20261119/README.md) distinguishes exact commands, source scope, successes and interrupted attempts. At candidate publication:

| Verification | Executed result and scope |
|---|---|
| Bounded session service tests | Three tests passed with race detection using production service sources and the driver-neutral fixture. |
| Token authorization state tests | One parent plus 18 command/state cases passed with race detection using production service sources. No failed/skipped cases or race report. |
| UUIDv7 allocation compatibility | One test passed with race detection, including a second invocation containing every production API source file. |
| SQL migration/query smoke | Python SQLite 3.53.1 applied all 20 migrations to 20,002 session rows and 40,000 token rows; preserved credential/lifetime data, assigned legacy selectors, used indexed bounded queries, passed integrity and foreign-key checks. This is not a Go/modernc execution. |
| Production package build | Domain, repository, service, API, web and store-contract production packages built successfully at the recorded intermediate source. |
| Python tooling | Export reader 10 tests, QA runner 4, thresholds 8 and hygiene 4 passed. |
| Formatting, source diff, workflow pins | Passed on the recorded source. Final-head provider checks still apply. |
| Full Go/SQLite, race, contract/load, browser, native platforms, lint/vulnerability, packaging and reproducibility | Required on the published combined candidate. Consult the continuation PR's exact-head results; source-PR successes do not qualify it. |

Local execution disconnected twice during this continuation. Interrupted compilation, incomplete aggregate runs, cancelled baseline-proof attempts and a cancelled fuzz attempt are not passes. No assertion that a regression failed before its fix is based on those interrupted attempts.

Migrations remain forward-only under accepted DEC-060. Transaction rollback is tested; automated schema downgrade is not supported. Operational rollback uses verified backup/restore.

## Future modules remain independent

All 15 streams remain disabled and on integration HOLD. Nested Go modules are not covered by the root application's `go test ./...`. [Future-module qualification](FUTURE_MODULE_QUALIFICATION.md) records the detailed source-pinned prior review and explicit database suites. This continuation refreshes the source inventory and records additional local/provider observations separately. Its local rerun completed default race suites for nine pinned modules and formatting/vet for eleven. Favorites exceeded its outer deadline after partial results (72 passed, one skipped); QR exceeded its outer deadline without terminal test results. Four modules were not attempted before the runner was cancelled. Those interrupted runs are not successes, and prior pinned qualification remains separately attributed.

The feed and notifications repair PRs target only their respective future branches. [PR #93](https://github.com/djshellshoxxx/circula/pull/93) merged at `797dddf21adddfb34a49e7bda1871a202968dcda` after its eight relevant checks passed. Its tree exactly equals reviewed source `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` (`d551498bd5c3c144acedf72f9320704ba55c3404`), and the merged manifest remains disabled/HOLD. [PR #92](https://github.com/djshellshoxxx/circula/pull/92) subsequently advanced to `0fed0480d3b129351c96cc0f62bd19479acceab7` with the reviewed Core session-link correction and requires fresh qualification.

The [PWA qualification PR #95](https://github.com/djshellshoxxx/circula/pull/95) also targets its existing future branch. Its browser suite passed, but the broader action policy rejected an undeclared `actions/setup-node` adoption. Source `4a754c2632f40c8e6c77f8d2a25686bd6e7f04f3`, published as `05e618b4145f85095b21a75a8fd15675ca2b5659`, adds the reviewed official v6.0.0 action to the allowlist and contributor documentation. The original failure was reproduced, and the patched policy, shell syntax and whitespace checks passed. All existing full-SHA and permissions guards remain intact; provider CI must qualify the new head. A successful module test or isolated repair merge does not authorize enabling that module in core.

Observed provider evidence includes:

- [Feed job](https://github.com/djshellshoxxx/circula/actions/runs/44491232304/job/135447075278), repair at `8638f3d30f27f4882c4c6424d4c8b248208b6f3d`: module race suite passed at 98.8% root-package statement coverage; cursor fuzzing completed 100,000 executions. A subsequent branch advance requires its own current-head checks.
- [Notifications job](https://github.com/djshellshoxxx/circula/actions/runs/44491239025/job/135447099448), repair at `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae`: explicit SQLite race suite passed at 71.0% root-package statement coverage; transport packages were not exercised. Vulnerability scanning reported zero reachable/imported-package vulnerabilities and one advisory in a required module that was not shown as called.
- [PWA job](https://github.com/djshellshoxxx/circula/actions/runs/44496951525/job/135466500874), qualification at `f25d037c3cc53fbf1ebbf687831300be9bdb749f`: 22 Node tests (zero failed/skipped), Go race tests at 97.5% root-package coverage, and the real Chromium install/cache/privacy/offline/no-JS/reflow harness passed using Chrome for Testing 151.0.7922.34. Its tested PR merge tree was `0a623e0587e8b011a816c84c93ee67fd0ec6e602`. Remaining core CI and platform/manual acceptance are distinct.

Coverage percentages above describe specific packages and commands, not feature completion.

### Remaining work estimates

These are engineering planning ranges, not measured completion percentages or delivery promises. They assume one experienced engineer, accepted decisions and stable test infrastructure. They exclude approval delays, external assessments and deployment. Integration should remain sequential and independently reviewed.

| Unfinished core area | Remaining work | Indicative effort |
|---|---|---|
| Complete English catalogue / UX-047 | Remaining templates, refusal/field messages, JavaScript and consistent locale/date/number handling; language selection remains a separate decision. | 5–10 engineer-days |
| Storage fault qualification | Read/fsync/truncate/checkpoint faults, ambiguous commit acknowledgment, restart/retry and committed-entry preservation. | 3–7 engineer-days |
| Reviewer recovery / dual-control independence | Approve an independence invariant, close revoke/reset/regrant and appointment paths, then adversarially qualify. | 3–6 engineer-days after decision |
| GOV-021 administrator-unavailable recovery | Reconcile operator and temporary/permanent authority, objection/expiry policy, migrations and end-to-end acceptance. | 4–8 engineer-days after decision |
| SEC-026 activity signals | Resolve subject visibility, threshold/disable authority and retention; qualify isolated implementation on final core. | 2–5 engineer-days after decision |
| Privacy/erasure retention | Resolve retained fields, audit free text and WAL treatment, then implement and test accepted rules. | 3–6 engineer-days after decision |
| Acceptance/release evidence | Manual accessibility, reference-hardware budgets, backup/restore drills, external review and policy/licence decisions. | 2–5 engineering days plus external scheduling; no release authorized here |

| Future module | Reusable work and remaining integration dependencies | Indicative remaining effort |
|---|---|---|
| FNET-201 Matching | Bounded pure matching logic; durable interests/preferences, authorized snapshots, ranking/caps and accessible suggestions. | 8–15 engineer-days |
| FNET-202 Favorites | Service, memory/SQL contracts, CLI/web; current core authorization, opt-in/migration/restore and browser acceptance. | 5–10 engineer-days |
| FNET-203 Receipts | Party-scoped TXT/JSON projections; authoritative/redacted snapshots and downloads. Signing requires a separate decision. | 4–8 engineer-days unsigned; signing adds 5–10 after policy |
| FNET-204 Feed | Permission-aware projection and repaired signed cursors; persistent snapshots/joins, preferences, moderation, no-JS UI and key lifecycle. | 12–22 engineer-days |
| FNET-205 Analytics | Checked integer aggregates and cohort suppression; reporting periods, reversals, differencing protection, authorized exports and accessible UI. | 10–18 engineer-days after privacy decisions |
| FNET-206 Areas | Coarse labels without coordinates; durable authorized CRUD, listing association, deletion/restore and accessible selection. | 6–12 engineer-days |
| FNET-207 Recurring reminders | Daily timezone/DST logic; durable schedules, atomic firing dedupe/restart, weekly/monthly rules and manual draft UI. | 10–18 engineer-days |
| FNET-208 Governance | Advisory eligibility/tally primitives; durable snapshots/unique ballots, closing rules, audit/export and accessible voting. | 12–22 engineer-days after policy decisions |
| FNET-209 Federation | Signed discovery envelopes; trust/consent/revocation, durable replay and transport, key custody and interoperability. No shared ledger. | 20–35 engineer-days after protocol decisions |
| FUX-101 PWA | Offline public shell and verified cache policy; production route adapter, cross-browser/device installation and manual accessibility. | 5–10 engineer-days plus device access |
| FUX-102 QR | Durable tokens, renderer/scanner, CLI/web; current auth/migrations, camera/no-camera acceptance, permission/race/dependency review. | 8–15 engineer-days |
| FUX-103 Discovery | Taxonomy/media sanitization, cursors and SQL/web; production visibility/storage limits, moderation/backup lifecycle and browser acceptance. | 10–18 engineer-days |
| FUX-104 Notifications | Preferences/encrypted targets/outbox/transports; current core event/session adapters, worker crashes, real delivery/egress/credentials, bounce/CLI/UI. | 15–25 engineer-days |
| FUX-105 Messaging | Participant access and bounded text/retention primitives; durable idempotent messages, current permissions, legal-hold/abuse rules and UI. | 15–25 engineer-days after retention decisions |
| FUX-106 Onboarding | Durable checklist and isolated synthetic practice server; current production auth/migrations, distinct-origin configuration and browser/restore acceptance. | 7–12 engineer-days |

No estimate changes a module's HOLD status or resolves an owner decision.

## Final integration rule

Before merging, refresh main and candidate, resolve any newly introduced conflicts, and inspect all checks on the exact published head. Use normal expected-head PR merge permission. Preserve the starting branches and recoverable PR refs. If a check or infrastructure gate remains unresolved, report that concrete blocker and retain the reviewable candidate; do not describe pending work as a completed main merge.

