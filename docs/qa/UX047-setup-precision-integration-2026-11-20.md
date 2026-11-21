# UX-047 setup precision: integration qualification

Checkpoint: 2026-11-20 Pacific.

**Qualification status: complete for this fix. All applicable provider jobs passed,
and final independent qualification approved with zero blockers.**

[PR #130](https://github.com/djshellshoxxx/circula/pull/130) corrected the setup
amount-validation plural and merged normally. This record separates the local
source checks, PR execution and actual-main execution. Overall UX-047 remains
**partial**.

## Change and source identity

An invalid default credit limit or sign-up grant at precision one previously
produced "1 decimal places" in the error summary and beside the field. The
existing `setup.amount.invalid` catalogue entry now supplies `one` and `other`
forms, and its caller passes integer precision first, localized label second
and the exact fixed-point example third. The catalogue still has 1,458 keys.

The new catalogue and authenticated HTTP regressions cover both amount fields
at precisions 0, 1, 2 and 4. They preserve exact examples, retained and escaped
values, selected precision, summary links, inline errors, ARIA associations and
HTTP 422 before network creation. Parsing, precision bounds, authorization,
CSRF and setup-secret behavior are unchanged.

| Role | Commit |
| --- | --- |
| Previously qualified main / source parent | `87651d36558a8e421e41fbd76c514045d88c74ed` |
| Reviewed source, preserved topic branch | `81f6bf5d0e073d4be78118f5baf3d8946b8347d5` |
| Actual PR CI checkout, synthetic merge | `733c82965f731f61119e431ea93de2d709361fc0` |
| Actual main merge | `fff81fc7f64cb4858c09d4a3a6f1ff0a55a1a337` |

The source, synthetic merge and actual main all have tree
`1f96de30d05b7886736315d07bf5820b6d37d995`. Both merge objects have the expected
ordered parents: previous main, then reviewed source. The twelve changed
paths match the reviewed publication manifest. Remote blobs/tree and the
local checkout were verified byte-for-byte. The topic branch
`codex/ux047-setup-precision-20261120` is preserved.

The [local qualification note](UX047-setup-precision-2026-11-20.md) and its
archive retain the baseline red patch, raw local results, exact five-file
hashes and independent code/standards and acceptance/security reviews. The
HTTP regression is byte-identical between red and green. Only the three
precision-one scenarios failed before the fix; the two parent failure events
are not additional failing scenarios.

The initial local executable build failed because the installed toolchain
selected the workspace parent for Git metadata in a linked worktree. The
identical full-checkout build passed with default VCS stamping enabled. Both
attempts are preserved; no product workaround or stamping override was used
for that normal build.

## Provider execution

- [PR Core run 44869522658](https://github.com/djshellshoxxx/circula/actions/runs/44869522658)
  passed all seven applicable jobs on attempt 1.
- [Actual-main Core run 44871808943](https://github.com/djshellshoxxx/circula/actions/runs/44871808943)
  passed all nine jobs on attempt 1 at
  `fff81fc7f64cb4858c09d4a3a6f1ff0a55a1a337`.
- [Actual-main Scorecard run 44871808928](https://github.com/djshellshoxxx/circula/actions/runs/44871808928)
  succeeded on attempt 1 at the same main commit. Its findings remain recorded.

| Gate | PR result | Actual-main result |
| --- | --- | --- |
| Formatting, vet, lint, vulnerability/secret/helper checks | Passed | Passed |
| Full Linux race suite | 3,272 named passes; six named skips; zero failures | 3,272 named passes; six named skips; zero failures |
| Native Windows suite | Not selected by the PR condition | Passed; 16 test-bearing packages; non-race |
| Native macOS Intel race suite | Not selected by the PR condition | Passed; 16 test-bearing packages; race detector |
| Five target packages | Built, scanned and verified | Built, scanned and verified |
| Reproducibility and local deployment smoke | Passed | Passed |
| Five fuzz campaigns | Passed | Passed |
| API contract and bounded load smoke | Passed | Passed |
| Chromium browser / axe / keyboard suite | 597 passed | 597 passed |
| Scorecard | Not triggered on PR | Execution and SARIF processing succeeded; findings retained |

The PR cross-platform condition produces one skipped pre-matrix placeholder;
it is not a passing native job. The native export-reader workflow is
inapplicable to both this PR and its merge because none of its three exact
path filters changed. Reader unit tests still ran inside the Linux lint job.
No release or manual QA-001 workflow was triggered by this normal merge.

### Full-race artifacts

| Artifact | PR | Actual main |
| --- | --- | --- |
| Provider artifact ID | `14694236500` | `14695042424` |
| ZIP bytes | 138,212 | 138,293 |
| SHA-256 | `497625b5a948c3a5b6a90bcc9cbaafda7926b16c5f9e9e124fabda3eb6086108` | `3de1693d800f8148f44a91ea637785737feafd7d0d774838ead44ab9ee41fe7e` |
| Recorded checkout | `733c82965f731f61119e431ea93de2d709361fc0` | `fff81fc7f64cb4858c09d4a3a6f1ff0a55a1a337` |
| Dirty state / command exit | `false` / `0` | `false` / `0` |

Both artifacts record the same source digest,
`ce4bc23996840a6e10d6e9405ef148e16a0f3d200e67103296d9128b4bd9b1dd`, consistent
with the identical source tree. Both ran
`go test -json -count=1 -race -timeout 30m ./...`. Each contains 3,296 terminal
events: 3,272 named passes, six named skips, 16 package passes and two
no-test-package skips. Named counts include parent tests and subtests. All
fourteen new UX-047 parent/subtest events pass in each execution.

The six named skips are preserved explicitly: three standalone subprocess
helper entry points, the opt-in load test, the opt-in statistical timing
measurement and the privileged foreign-owned-parent fixture. The separate
contract/load job exercises the bounded load scenario. Statistical timing and
privileged ownership are not newly qualified by these full-suite results.

### Other execution details and limits

Both browser logs contain 597 distinct, consecutively numbered passing rows,
with no observed retry, flaky, failed or skipped outcome. These are the
configured automated Chromium, axe and keyboard checks; no new manual or
screen-reader qualification is claimed.

The Windows command is `go test -count=1 ./...`. Its non-verbose output records
sixteen distinct passing packages and two packages without test files; it
does not supply named test/skip counts.

The native macOS Intel command is
`go test -race -count=1 -p 2 -timeout=45m ./...`. Its output also records sixteen
distinct passing packages and two packages without test files, without named
test/skip counts. This is separate native race execution at the actual main.

The five fuzz campaigns reach one million amount-parser executions and
100,000 executions for each remaining target. Contract checks cover 89
operations and 172 operation/status pairs. The bounded main load completes
100 of 100 attempted trades, reports zero unavailable/unexpected errors,
retries or abandoned attempts, and records `ledger_ok=true` with summed
member balances zero. These bounded checks do not approve release-performance
thresholds.

Reproducibility compares the existing Linux/amd64 static-build configuration
with explicit stamps. Its two main builds agree on
`f42029371e8aee84115d99d337dad99babcf80b8ac7eb62fbf0f0eb3184a3e07`.
The separate smoke command checks the unit's command/environment, readiness,
health, database permissions and clean shutdown in a temporary loopback
environment. Actual systemd operation and deployment are separate work.

The five target packages are provider-verified compilation/scan artifacts;
the packaged ZIP was not independently downloaded or rehashed in this review.
Scorecard v5.6.0 completed with a score of 5.9 across eighteen checks. The complete report
and independent review retain its open findings. No licence, policy or
release decision is inferred from successful execution.

## Independent reviews and continuation

The source code/standards, acceptance/security, documentation/evidence and
complete PR-provider reviews found zero blockers. The final actual-main review
independently accepted all nine Core jobs plus Scorecard execution, with zero
blockers and no outstanding jobs in the configured set. A second reviewer
confirmed final run/job/ref and Scorecard linkage. Actual-main qualification
uses its own receipts; prior PR results are not substituted. The consolidated
review is preserved at `provider/main-review/final-main-qualification.md` and
`.json` inside the archive.

A bounded follow-up source review at the merged main examined twelve
catalogue entries that render a cardinal decimal-place count and fourteen
direct production callers. Their singular/plural forms and integer-first
arguments are consistent, and matching existing singular assertions were
found. No further precision-family defect was confirmed. This was static
source/assertion inspection, without new product executions or a whole-UI
audit.

Continue the active UX-047 work from the integrated source with a separately
bounded investigation. Do not reintroduce the completed precision messages or
historical missing-flow lists as unfinished migration. Whole-UI localization
structure remains a separate completion assessment; translation and locale
scope remain product decisions. Held policy work and all fifteen future
modules stay outside this continuation. No deployment, release, branch
deletion or history rewrite occurred.

## Durable evidence

The [integration evidence archive](evidence/UX047-setup-precision-integration-2026-11-20.tar.gz)
contains provider run/job snapshots, all completed job logs, both verified
test-artifact ZIPs and extracted reports, source publication metadata and the
independent review records. Its `MANIFEST.json` records each payload's SHA-256
and size. It contains no compiled product executables.

Archive: **780969 bytes**, **72 payload files plus the manifest**.
SHA-256: `3f82907f20840d91f460f45a22701d44b38161d6b24aad6af53eb3e6c9dfdd25`.

Earlier snapshots and interim reviews preserve their original pending states;
the final run snapshots and final actual-main review govern the completed
qualification. This evidence branch adds documentation and receipts to the
qualified main source; it is not an additional runtime integration.
