# Core qualification continuation — 2026-11-20

## Current integration boundary

The previous consolidation, PR #99, merged into `main` at
`aa01563fcc891da60fefebb8335eca27855678e8` after all nine jobs in
[its combined CI run](https://github.com/djshellshoxxx/circula/actions/runs/44598890076)
passed. The independently inspected later
[main run](https://github.com/djshellshoxxx/circula/actions/runs/44728309016)
also passed all nine jobs. The prior continuation task completed its scoped
future-branch merges and disabled itself.

The current Core candidate is [PR #127](https://github.com/djshellshoxxx/circula/pull/127),
initially inspected at `21933a3c650690db5a8e1e44cc236d8d1b5cdb24`.
PR #125, #112, #109 and the advanced #102 are already ancestors. The refreshed
remote inventory has 148 branch heads. Historical #104 verifier runtime is
already represented, and its older Python reader must not replace newer
compatibility safeguards. PR #111's alternate WAL fixture is superseded by the
retained accepted repair. Their source branches remain preserved.

The original documentation-only PR #128 at
`896d69f472652ed54c051ec82efcb7f162d1d2c4` is incorporated without changing its
reports or checksums. Its findings are pinned historical observations. The
[separate reconciliation](SECURITY_HANDOFF_RECONCILIATION_20261120.md) records
which ten of fourteen source findings already have substantive repairs in
PR #127 and which source or policy gaps remain.

The first repair composition was published as
`d6843366dc06d723f8d9efe07242997c858a6b3f`, with verified tree
`733dacda5cdbe1e12f7ab84079af2b7615cd7c81`. Subsequent AUTH-04 implementation
is integrated locally at `025ad9d3b23abd29c99dbe3ea376ff1622844313`, preserving
that published commit as its parent. It adds shared two-operation password-work
admission across authentication, recovery/MFA, startup and passphrase backups.
Its [pinned execution receipt](evidence/password-admission-20261120/README.md)
records 116 distinct top-level tests and 239 distinct named cases passed, zero
failed and one existing opt-in timing skip. Independent review found no
unresolved issue within the admission scope. The
[security reconciliation](SECURITY_HANDOFF_RECONCILIATION_20261120.md) now
distinguishes eleven implemented source findings from the three remaining
source/policy/platform gaps. Full combined-source checks remain required.

The first complete normal Go run of this composition, at local source
`a9cdcb80179df1469b1443dcb79f1df00b1857fb`, completed in 377.345 seconds with
1,292 top-level tests passed, one failed and six skipped (3,456 passing events
including subtests). The failure exposed an existing audit-anchor fixture that
could leave an already matching hash prefix unchanged. A reviewed test-only
repair guarantees a different digest and checks that exactly one fixture row
changed, retaining the original clean/growth/rewrite/truncation assertions.
The [complete-run summary](evidence/core127-qualification-20261120/auth-full-normal-before-fixture.json)
preserves the failed result and original source; it is not reported as a pass.
The corrected composition requires a new complete race run and final-head CI.

The first combined secret scan also matched nine source-file digests in the
AUTH-04 execution manifest as credential-shaped map entries. Their metadata
representation now uses explicit path/sha256 rows, with all 22 source paths
and digests verified unchanged. Execution logs and scanner rules are unchanged;
the final complete scan must pass on the publication composition.

## Exact failures and repairs

The initial PR #127
[run 44712003434](https://github.com/djshellshoxxx/circula/actions/runs/44712003434)
finished with five successful and four failed jobs. Linux race tests, API/load,
five bounded fuzz targets, package builds and reproducibility passed on that
old head. Native export-reader qualification separately passed on all three
platforms. Those results do not qualify this later repair composition.

| Observed issue | Repair and preserved boundary |
|---|---|
| Windows migration 0023/0025 digest mismatch | A scoped `.gitattributes` rule keeps migration SQL at LF during checkout. Original SQL bytes and exact review digests remain unchanged. A real Git checkout with `core.autocrlf=true` reproduced the mismatch, restored both reviewed digests after the rule, and still rejected changed SQL. |
| macOS service package timed out at 25 minutes | Increase only the native macOS package budget to 45 minutes and job budget to 60 minutes. The timed-out stack was actively parsing a new SQLite fixture; the current subtest was only two seconds old. Keep every test, race instrumentation, `-count=1`, `-p 2`, runners and other platform budgets. Native completion remains required. |
| G710 on a report return redirect | Construct the destination from the authorized case identifier, an escaped path segment and the rooted network prefix. Do not use a form display link as the redirect target. Context-free reports retain their network-home destination. No lint suppression was added. |
| Two loading assertions waited for their own held navigation | Capture outgoing-document nodes before native activation, then poll the retained handles. Preserve announcement text, live-region role, busy state, focus, enabled controls, submitted values and final navigation/reset assertions. Production loading behavior is unchanged. |
| Missing `dispute_reports` accessibility coverage | Exercise the actual independent-reviewer report page, require HTTP 200 and its heading/back link, run the existing light/dark axe checks, and register that real check in the coverage map. No route exclusion or accessibility rule was added. |
| Historical credit-evidence null alias handling | Apply the existing explicit-null rejection to every case-equivalent default-limit field recognized by the policy decoder. Preserve omitted historical defaults and valid numeric aliases. This is a damaged/imported-history correctness boundary; ordinary policy writes already use canonical validated JSON. |

The portability and web patches received independent source review. A fresh
review of the migration invariant gate, historical credit accounting, signed
coverage and report privacy/context found the last decoder issue above and no
additional concrete blocking defect within that scope. Policy decisions for
system-account report subjects and all-conflicted escalation remain explicit.

## Executed local qualification

[Execution receipts](evidence/core127-qualification-20261120/combined-verification.json)
identify the actual source and distinguish successful checks from incomplete
attempts. The combined Go source at local commit
`2d45ef0ef27f746d11990c07fa65e469c2199184` passed:

- Focused store/web race tests: three top-level tests and 28 subtests, zero
  failed/skipped, two successful packages and no race report.
- Production message/keyboard helper tests: 23 passed, zero failed, skipped
  or cancelled.
- Focused production-web Gosec analysis: zero issues. This is narrower than
  the full provider lint gate.
- Standalone migration review/scanner tests: two passed, zero failed/skipped,
  as recorded in the portability receipt.
- Current traceability, both changed JavaScript syntax checks, workflow action
  pins/permissions, repository hygiene and whitespace checks.
- Directory secret scanning on the recorded pre-publication source: no leaks
  found. The final published composition still requires its own provider scan.

The full local lint attempt was killed with exit 137 before a report; it is
not a pass. Configured local Chromium aborted before page creation with
`socket() failed: Operation not permitted (1)`; no browser product assertion
executed. These limits leave the complete provider gates mandatory.

Publication must verify that the provider tree equals the reviewed local tree,
refresh the current branch heads, and use an expected-head fast-forward update.
The final publication SHA and live check receipts belong in the PR body so
recording successful CI does not invalidate its own tested commit.

## Next bounded work and preserved future streams

The [148-head refresh](BRANCH_REFRESH_20261120.md) and complete dependency graph
are pinned observations; subsequent mutations are recorded separately. After
the [independent four-module review](FUTURE_FOUR_SOURCE_REVIEW_20261120.md), the
coordinator reread all 40 successful applicable job records and completed the
following normal merges. All four provider merge trees equal their qualified
source trees, both parents match the reviewed target/source, and every original
source branch remains available. The
[post-merge receipt](evidence/future-merges-20261120.json) records exact pins.

| PR | Existing isolated target | Verified merge commit |
|---|---|---|
| #113 | `future/fnet-203-receipts-20261116` | `5200c5bf940f051a86d6e502787af06892200974` |
| #114 | `future/fnet-208-governance-20261116` | `5eac085005361e973ca8f358e01f039aea718bed` |
| #115 | `future/fnet-206-areas-20261116` | `6aac35e226135a53d69c1e6ce74e340dcfe1d85f` |
| #120 | `future/fux-102-qr-20261116` | `6028dae56723b8488db6e52e25c83da7888777c1` |

Each source passed seven applicable Core jobs and three dedicated native module
jobs. Each older Core workflow also returned one skipped, unexpanded native
matrix placeholder: the configured Windows/macOS Core executions did not run.
Those skips are not passes. QR's dedicated jobs qualify Node camera-lifecycle
tests, not a real camera or a fresh QR Go/SQLite suite. All four manifests remain
disabled/HOLD, with their explicit host, policy and activation gates preserved.

PR #103 subsequently merged into the existing notification future branch as
`54a41b12a314e7652802b71d693e2594c37f172f`. All nine Core jobs, the dedicated
SQLite job and all three native transport jobs passed at source
`66b23f8e9e37ce6efddf56c0215c6f29f741d489`. The coordinator verified both
parents and exact merge tree `774205e9de006c00181c8ef09e192ccb42489e92`.
Only the previously reviewed native workflow and qualification document differ
from the accepted target; runtime and the disabled/HOLD manifest are unchanged.
The [notification merge receipt](evidence/notification-native-merge-20261120.json)
preserves these exact references.

The [remaining-work table](FUTURE_MODULE_REMAINING_WORK_20261120.md) credits the
current implementations in all fifteen streams and estimates remaining work
separately from policy decisions and independent production qualification. It
does not treat old prototype descriptions or completion percentages as current
implementation inventories.

The [Core remaining-work table](CORE_REMAINING_WORK_20261120.md) separately
records the reader-resource and Windows-support gaps, preserved policy/interface
decisions and reference-hardware/final-qualification boundaries, with explicitly
conditional engineering estimates.

AUTH-04 password-work admission has passed its scoped implementation tests and
independent review; its final combined qualification is still an integration
gate. OPS-04 reader resource budgets remain a distinct source gap.
Financial-error disclosure,
Core Windows storage support, and the previously recorded governance/privacy
decisions are not silently resolved here.

The already accepted feed, PWA, recurring and notification integrations remain
on their independent future branches. Remaining unique notification transport
work is its native workflow and qualification note; runtime and dependencies
are already in the accepted target. The isolated messaging follow-up repairs
Windows path spelling validation before normalization and retains its original
native regression. Both compositions require their own current-head checks.

Additional isolated prose repairs were published on the existing federation
recovery PR #117 at `267aa7d08bbc41e89a429eec7a077c5c0b1f9778` and onboarding
retention PR #121 at `20e6c8479459f837915eb31a5d668e2bd470e5a8`. Each changes
only its qualification document after ordinary prose matched the secret
scanner. Complete directory scans passed after the wording repairs. Runtime,
tests, workflows, scanner rules and disabled/HOLD manifests retain their
original blobs; fresh provider checks remain required.

All fifteen future modules remain disabled and on HOLD. No branch deletion,
force push, deployment, release, or unqualified main merge is authorized by
this evidence record.
