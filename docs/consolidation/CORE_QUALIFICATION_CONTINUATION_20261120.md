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

AUTH-04 password-work admission is a separate implementation milestone based
on the reconciled design. It must cover shared service and backup Argon work,
retain credential compatibility and guessing controls, and undergo independent
review and full combined qualification before integration. OPS-04 reader
resource budgets remain a distinct source gap. Financial-error disclosure,
Core Windows storage support, and the previously recorded governance/privacy
decisions are not silently resolved here.

The already accepted feed, PWA, recurring and notification integrations remain
on their independent future branches. Remaining unique notification transport
work is its native workflow and qualification note; runtime and dependencies
are already in the accepted target. The isolated messaging follow-up repairs
Windows path spelling validation before normalization and retains its original
native regression. Both compositions require their own current-head checks.

All fifteen future modules remain disabled and on HOLD. No branch deletion,
force push, deployment, release, or unqualified main merge is authorized by
this evidence record.
