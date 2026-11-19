# SEC-026 / PR #86 source, policy and readiness review

Reviewed 2026-11-18. This review began at `185273a43a337d25bac7e2cff8ffb83c64ac05e6`; the author's live work then advanced to PR #86 at **`2e2e6f0eba6bed29bed8f2dd5d60699f372760a6`**. Current disposition below is for the latter head. No build, test, new reproduction or probe was run. No owner-branch, root-checkout or remote mutation was performed.

## Current disposition

**Preserve as valuable core implementation under active development, with explicit owner-policy holds and a dependency on draft PR #85. It is not ready for consolidation into main.** This is SEC-026 core functionality, not one of the isolated future modules, and it is not duplicated by the existing authentication/proposal throttles.

The original numeric migration collision and conflicts with current main were resolved by the author's advancing merge. Do not continue to list them as current-head blockers. The remaining release/integration questions are policy approval, qualification of the current stack and its migrations, and completion of independent review. This review does not approve PR #85's underlying recovery-policy changes.

## Exact branch and PR identity

| Item | Current value |
|---|---|
| PR | https://github.com/djshellshoxxx/circula/pull/86 |
| Title | SEC-026: velocity and reciprocal-trade activity signals for administrators and reviewers |
| State | Open, draft true |
| Branch | `claude/sec-026-abuse-signals` |
| Head | `2e2e6f0eba6bed29bed8f2dd5d60699f372760a6` |
| Tree | `90c111732d70bc3f019749e66ee8a0f72228614e` |
| PR base | `claude/gov-021-admin-recovery` at `b20c118b59d43836bd357788914542b904f13872` (#85) |
| Main comparison | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` |
| Relative to PR base | 0 behind / 4 ahead; merge base equals PR base |
| Relative to main | 0 behind / 9 ahead; merge base equals main |
| Latest metadata | created/updated `2026-11-18T08:45:14Z`; mergeable true, mergeable_state unstable |
| Authorship | GitHub account `djshellshoxxx`; body/commits attribute Claude session `session_01KtKj8n2B7N1ZkoQQdz85gS` |

The initial head was based on `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4`, 14 commits behind main and two ahead. It had no associated PR in the first open-PR snapshot. During the review the author added verification documentation and merged #85/current main, then opened the stacked draft. This is concrete ongoing ownership, not abandoned historical code.

The four commits unique to the current PR base are:

- `0f8427412a3bc18f37ddd15453d486e93d5d4607`: SEC-026 implementation.
- `185273a43a337d25bac7e2cff8ffb83c64ac05e6`: traceability.
- `9d8adb11f5a78b77661063c4ee209487b67523fd`: author verification record.
- `2e2e6f0eba6bed29bed8f2dd5d60699f372760a6`: merge #85/current main and renumber the migration.

## Unique behavior and implementation shape

Canonical `docs/specs/09-security-privacy.md:358–364` requires network-configurable proposal-velocity and reciprocal-trade signals, no trade blocking, one signal per window, and same-network administrator/reviewer visibility. Main's `docs/TODO.md:81` explicitly says this implementation is outstanding and that existing throttles are not the feature. The branch supplies it across 49 changed files relative to #85.

- A successful proposal evaluates the proposer's trade count; a completed trade evaluates the pair's completed trades in both directions. Threshold comparison is strictly greater-than. Later reversal/dispute status does not erase the original completion count.
- Event counts use the configured preceding time window; deduplication uses epoch-aligned fixed periods. Signals carry network/member identifiers, kind, threshold, observed count and timestamps, without member-authored text.
- `internal/service/signals.go:91–115` encloses evaluation, signal insert, audit and notices in a store savepoint. Failure is retained for post-commit reporting, rather than returned as a refusal of the trade.
- `internal/store/sqlite_signals.go:90–119` implements savepoint rollback and trims the buffered audit sink. A unique key plus insert conflict handling controls duplicates.
- `ListActivitySignals` uses `open(..., ScopeRead)` and `requireDecider`, then network-scoped SQL and subject filtering. It therefore reuses the established membership, effective-role, live break-glass and administrator-MFA checks. This is source inspection, not a fresh authorization reproduction.
- New API route `GET /api/v1/activity-signals` uses the signed cursor helper and kind filter. The HTML reviewer page uses bounded keyset paging. Both expose current display names, which become anonymized after erasure.
- Signal notices and settings-change notices traverse all active members through a paged helper, rather than stopping at the first 500. Signals are review prompts; there is no reviewed/dismissed workflow.
- Five new policy settings are accepted through the existing policy update, validation, audit and effective-time mechanisms. No ledger-entry mutation is introduced by the SEC-026 layer.

The core signal service, domain rules, policy type, API handler, web handler, repository contract and service regression file are byte-identical between the initial and current heads. The store implementation changed only its migration-number comment. Exact blobs are in `PR86-BLOB-COMPARISON.json`.

## Policy choices requiring disposition

### 1. Default activation and numeric thresholds are executable provisional decisions

Canonical SEC-026 at spec09:359 points to spec12 for recommended defaults. Spec09:453 explicitly leaves whether velocity defaults belong to product or test engineering open; no SEC-026 threshold defaults were found in spec12.

The branch chooses these defaults in `internal/domain/policy.go` and `internal/domain/signals.go:27–42`:

| Setting | Provisional value |
|---|---|
| Enabled | true |
| Velocity | more than 20 proposals in 60 minutes |
| Reciprocal | more than 10 completed two-way trades in 30 days |
| Threshold bounds | 1–1000 |
| Velocity window bounds | 1 minute–1 week |
| Reciprocal window bounds | 1–365 days |

This affects **existing networks**, not just newly created ones: `internal/store/sqlite_network.go:44–45` initializes `DefaultPolicy()` before decoding old stored JSON, so absent settings inherit “on” and these thresholds. The author explicitly labels them owner-pending in the implementation note and PR body. The source behavior should not silently become approved by merging a consolidation PR.

### 2. Hiding signal settings from the members' change log is a canonical governance exception

Canonical `docs/specs/10-governance-operations.md:95–100` (GOV-002) requires every policy change to have before/after audit values **and to be shown to members in a readable network change log**.

The branch's `internal/service/networks.go:389–401` adds five signal keys to `reviewOnlyPolicyKeys`; `gov_terms.go` removes them from both policy-log methods. A signal-only change can therefore leave a version with no visible changes. Auditing and notices to other administrators/reviewers do not satisfy the member-change-log portion by themselves.

This is a direct source/spec discrepancy with an explicitly proposed exception, not an independently reproduced security incident. The author's implementation note acknowledges it under “Known gaps” and requests an owner decision. The existing special rule for `dual_control_threshold` does not automatically approve a new SEC-026 exception.

### 3. Subject exclusion and control over disabling/loosening are explicit product choices

The service, SQL filter, audit-subject selection and notification code exclude the members a signal names, even if they are administrators or reviewers. If every eligible reviewer is a subject, nobody sees the signal. SEC-026 requires authorized same-network visibility but does not settle this particular self-visibility rule; the author explicitly requests a decision.

One administrator can disable or loosen signals through the normal policy path, with an audit and notices to other administrators/reviewers. Whether that must require a second administrator is another author-requested decision. No canonical SEC-026 two-person requirement was found, so the absence of dual approval is a policy limitation to decide, not a reproduced authorization defect.

### 4. Lifetime retention is a new explicit choice under an open retention policy

The new inventory text at `docs/security/personal-data-inventory.md` states that activity signals are append-only, unexported, retain member IDs and counts after erasure, and last for the network's lifetime with DEC-020 still open. The new schema enforces no UPDATE/DELETE, while `docs/dev/erasure.md` lists these rows as unchanged after erasure.

Canonical bases:

- SEC-014, spec09:260–265, requires data minimization and inventory coverage.
- SEC-018, spec09:295–301, requires deletion or irreversible pseudonymization of personal data not needed for ledger integrity/legal retention and makes the trade-off explicit.
- Spec06:256–271 gives lifetime defaults for named ledger/audit/dispute/consistency records, but contains no activity-signal category.
- `docs/DECISIONS.md:838` leaves DEC-020 retention periods open.

The branch does document this choice and includes a test that the display-name canary disappears while the signal row remains identical. That does not establish an approved lifetime-retention policy for this new record type. It also does not by itself prove an erasure violation: references resolve through the anonymized member record, and no member free text is stored. The retention/category rationale needs explicit owner disposition rather than being silently treated as inherited ledger policy.

The SEC-026 layer leaves DEC-014 and A3-F02 unresolved. Its stacked #85 dependency must retain its own separate recovery/break-glass policy review.

## Migration and integration dependency

### Initial collision, now resolved

At initial `185273a`, SEC-026 added `0019_activity_signals.sql`, while #85 added `0019_admin_recovery.sql`. Main's loader in `internal/store/migrate.go:41–45` requires a contiguous sequence by numeric version; merely having different suffixes would not make those two version-19 migrations compatible. This conclusion follows directly from the loader source; no failure probe was run.

Current `2e2e6f0` instead contains:

- Existing migrations 0001 through 0018.
- **`0019_admin_recovery.sql`**, inherited from #85.
- **`0020_activity_signals.sql`**, the renamed SEC-026 migration.

The renamed SQL blob is identical (`7957ef1114867d753737259c8c6eb7ea769e48e3`) to the initial activity-signal migration. No stale `0019_activity_signals` reference was found in the current tree. The current head contains both current main and #85 by ancestry, so the earlier main conflicts in catalogue/traceability are also historical.

### Schema behavior and qualification still required

The activity table is additive: kind/count/time/pair-order CHECKs, composite member/network foreign keys, unique deduplication key, queue index, and proposal/pair-count indexes. Append-only triggers reject update/delete/replacement and ignore duplicate period inserts.

The existing new `TestSEC026SignalMigrationIsAdditive` opens a v18 database, seeds records, upgrades to the current schema and checks row counts plus an empty signal table. On this stack, that exercises the path through both 19 and 20 by construction; this review has not executed it. Direct v19→v20 qualification and the complete final aggregate gates should be accounted for before integration. Recorded migration-name checks in `migrate.go:100–109` also mean already-applied temporary v19 activity-signal databases cannot be casually treated as matching the new v19 administrator-recovery schema.

PR86 is explicitly stacked on #85. Merging or retargeting the entire head to main would carry #85's recovery functionality too. Preserve that dependency and do not bypass its owner-policy gates as part of a SEC-026 merge.

## Test evidence and review limits

The branch adds 26 named `TestSEC026*` functions across domain, service, store, API, web, CLI and messages. They address threshold/window boundaries, reciprocal activity without ledger changes, role/tenant visibility, duplicate concurrency, disabled settings, failure isolation, policy audit, paged notices, erasure, schema constraints and migration.

The PR reports author full tests and race results at the pre-stack verification commit, then focused SEC026/GOV021/migration/contract/OpenAPI/authorization/catalogue results after stacking. It says the full local run is in progress and CI is authoritative. Those are author claims; no new execution result is attributed to this reviewer.

One narrowly scoped future review question is visible in source: the SQL count methods apply an exclusive lower time bound, but no upper bound corresponding to “window ending now.” After a backward host-clock correction, a previously future-dated transaction could therefore be counted. The existing window tests inspect forward boundaries; this review did not run a backward-clock reproduction. This is an unverified source-based edge-case question, not a confirmed security finding or a reason to broaden the current consolidation tests.

## Recommended handoff

1. Preserve draft86 and its original branch; classify as unique core implementation pending policy/stack qualification.
2. Keep the four author-requested decisions and the new retention rationale explicit.
3. Record the latest migration state as **numbering resolved, dependent on #85**, replacing the obsolete collision blocker.
4. Use the eventual exact approved stack head for independent tests and adversarial review; do not transfer the older author's test results to that head automatically.
5. Keep A3-F02 / DEC-014 unresolved and retain #85's separate review.

## Evidence artifacts

- `PR86-EXACT-INVENTORY-2e2e6f0.json`: exact current tree, bases, commits and all 49 paths.
- `PR86-API-METADATA-2e2e6f0.json`: exact PR metadata/body attribution and author verification statements.
- `PR86-BLOB-COMPARISON.json`: unchanged core source and migration rename proof.
- `pr86-full.diff`: complete current PR diff against #85.
- `SEC026-INITIAL-INVENTORY.json`, `SEC026-MERGE-COMPATIBILITY.json`, `sec026-full.diff`: initial185273a evidence only; superseded for current readiness by the stacked head.
