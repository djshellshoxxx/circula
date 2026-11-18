# Consolidation ledger (2026-11-18)

This ledger extends [BRANCH-CONSOLIDATION-2026-11-17.md](../../plans/BRANCH-CONSOLIDATION-2026-11-17.md) and
[PR-CONSOLIDATION-2026-11-17.md](../../plans/PR-CONSOLIDATION-2026-11-17.md). Those files still hold the
dispositions of the 88 branches and 68 PRs reviewed on 2026-11-17. Their live tips were re-checked against
`origin` on 2026-11-18. All are unchanged, including the 15 `future/*` branches and `future/module-catalog-20261116`,
with one exception: `claude/sharp-archimedes-tu76rh` (historical PR #70) has advanced and now carries another
worker's open PR #84 (tip `eb4bb08` at 2026-11-18T09:35Z). Frozen historical PR heads (`refs/pull/N/head`) are
recorded separately from live branch heads.

Method, as in the earlier ledger:

- The clone was unshallowed first, and `refs/pull/*/head` was fetched.
- Ancestry was checked with `git merge-base --is-ancestor`.
- Commit counts used `git rev-list --count origin/main..<tip>`.
- Every remaining commit was compared by content.

No branch was deleted. No history was rewritten. Nothing was force-pushed.

## Branches added since the 2026-11-17 ledger

Snapshot at 2026-11-18T09:35Z (`git ls-remote --heads origin`): **105 remote branches**, not counting `origin/HEAD`.
That is the 96 at baseline plus 9 created in this pass, for PRs #78–#83 and #85–#87. PR #84 reuses the historical
#70 branch.

| Branch | Tip | PR | Unique work | Disposition |
|---|---|---|---|---|
| `claude/circula-pr-consolidation-q6q18n` | `163fe15` | #69 | none (in `main`) | merged before this pass |
| `integration/admin-grant-notice-pagination-20261117` | `cb25b41` | #73 | none | merged before this pass |
| `integration/erasure-fixture-stability-20261117` | `fc31dc9` | #76 | none | merged before this pass |
| `integration/release-threshold-check-20261117` | `ec98cab` | #74 | none | merged before this pass |
| `integration/throttle-documentation-20261117` | `83de536` | #75 | none | merged before this pass |
| `integration/browser-parity-clock-20261117` | `2f6b3e4` | #77 | UX-005 parity test clock fix | **merged** `78c695f` |
| `integration/store-untrusted-schema-20261117` | `b1cd8f3` | #71 | E-02 | **merged** `28ed716` |
| `integration/backup-private-staging-20261117` | `faed490` | #72 | NEW-01 | **merged** `a62071d` |
| `claude/ux-047-auth-catalogue` | `dd3e82e` | #79 | UX-047 authentication slice | **merged** `cdc8112` |
| `claude/backup-path-hardening` | `09ef8a5` | #81 | AUD4-D01/D02 | **merged** `b439807` |
| `claude/aud4-auth-fixes` | `6785863` | #82 | AUD4-F06/F04 | open; CI on the combined tree |
| `claude/aud4-f03-dispute-timelines` | `679a301` | #87 | AUD4-F03 (partial) | open; CI |
| `claude/aud4-f02-reviewer-reset-policy` | `af54887` | #83 | AUD4-F02 candidate | open draft; **owner decision** |
| `claude/gov-021-admin-recovery` | `59cf699` | #85 | GOV-021 | open draft; **owner gates** (SEC-013 decision 7, operator identity, objection period, expiry) |
| `claude/sec-026-abuse-signals` | `4d349c4` | #86 | SEC-026, stacked on #85 | open draft; **owner gates** (defaults, GOV-002 exception, subject hiding, retention) |
| `claude/fnet-204-signed-cursors` | `0952c94` | #80 | FNET-204 cursor hardening, future only | open draft against #67's branch; never into `main` |
| `claude/sharp-archimedes-tu76rh` | `eb4bb08` (advanced; historical #70 branch) | #84 | 2026-10-28 Low/Info triage | another worker's open draft; not reviewed here |
| `claude/circula-consolidation-audit-0sydm1` | this PR | #78 | this record | open draft |

## Pull requests

| PR | Before | After |
|---|---|---|
| #67 | open draft (future) | unchanged; #80 is stacked on it |
| #71, #72, #77 | open | merged |
| #78–#87 | — | opened in this pass (#84 by another worker) |

Unique work preserved: everything above. Every open PR is either a fix waiting only on CI or an owner-gated
draft. No PR was closed, and no unique code was discarded.
