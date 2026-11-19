# Consolidation ledger (2026-11-18)

This ledger extends [BRANCH-CONSOLIDATION-2026-11-17.md](../../plans/BRANCH-CONSOLIDATION-2026-11-17.md) and
[PR-CONSOLIDATION-2026-11-17.md](../../plans/PR-CONSOLIDATION-2026-11-17.md). Those files still hold the
dispositions of the 88 branches and 68 PRs reviewed on 2026-11-17. The live tips were re-checked against
`origin`; the 2026-11-18T09:35Z snapshot has three differences from that historical branch ledger:

- `main` advanced normally from `b53f7576` to `b4398074` through the reviewed merges below.
- `claude/quirky-babbage-y9fpmm` is `af69702be08aaf14cbe9a705ddf7f08796d5e976`, while closed PR #39
  remains frozen at `92d204dc027320b1b26a05fd86e58232d89394ca`. This live-ref difference already existed
  at the start of the integration review. Its additional Python traceability tooling was reviewed and is
  superseded by the current Go implementation in `scripts/traceability` and CI; PR #39 stays closed and
  its branch is preserved. The historical baseline is not evidence that this live tip equals the frozen PR head.
- `claude/sharp-archimedes-tu76rh`, recorded as `fe111799` in the old ledger and later used by PR #70,
  advanced to `eb4bb08` and now carries another worker's open PR #84.

The other 85 historical branch tips match, including the 15 `future/*` branches and
`future/module-catalog-20261116`. Frozen historical PR heads (`refs/pull/N/head`) are recorded separately
from live branch heads. The baseline and 09:35Z inventory remain dated snapshots, not a claim that later
PRs or branch updates cannot occur.

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

After the 09:35Z snapshot, #82 advanced normally from `6785863` to
`e127acc9877edebeca818b75a5d48ce12511eca8` to include the reproduced MFA/account-recovery password-proof
repair. Its independent review and focused regressions pass. The full local combined race suite and pinned
lint subsequently passed on candidate `17152dadad91d4e1c994ea807e7d6e2497513aa8`, tree
`1e47c9c30165ca27866887533586c90323e1630e`, which also includes #84 at `eb4bb08` and #87 at `679a301`.
This is a candidate qualification, while `main` remains `b4398074`: #82/#84/#87 were still unmerged with
seven of seven provider jobs queued per PR at 2026-11-18T10:07:24Z. This follow-up does not change the
historical 105-branch snapshot. See [TEST-EVIDENCE](TEST-EVIDENCE.md).

Subsequent verified update at **2026-11-18T10:37:58Z**: the integration coordinator merged #84 as
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`, tree `717b32995b82772e3d6ff644297b7d05a74464b5`,
after its seven required provider jobs passed. [MERGE-MANIFEST](MERGE-MANIFEST.md) pins the source,
CI run, review, parents and actual merge receipt; [SECURITY-AUDIT](SECURITY-AUDIT.md) records completion
of A2-08 and A3-F10. The 09:35Z branch inventory and 10:07Z candidate checkpoint above remain dated
historical observations. No new repository-wide branch or PR count is asserted here. #82/#87 remain
unmerged with provider CI pending at this update.

## Pull requests (earlier pass snapshot)

| PR | Before | After |
|---|---|---|
| #67 | open draft (future) | unchanged; #80 is stacked on it |
| #71, #72, #77 | open | merged |
| #78–#87 | — | opened in this pass (#84 by another worker) |

Unique work preserved: everything above. Every open PR is either a fix waiting only on CI or an owner-gated
draft. No PR was closed, and no unique code was discarded.
