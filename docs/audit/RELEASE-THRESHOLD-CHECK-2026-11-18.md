# Release threshold decision check

Baseline: `ef54238ee9ebc2184919b6a8ec30c3398df55ed5` (merged PR #69).

## Reproduction

`scripts/release-gate.sh` searched the entire `docs/qa/THRESHOLDS.md` file for
`UNCONFIRMED`. The explanatory footer contains that word independently of the
Decision cells. In an isolated fixture, replacing every unconfirmed Decision
cell with a dated DEC-023 reference still produced `[FAIL] thresholds` and exit
status 1. All unrelated checks were explicitly skipped for this classification
probe; it was not release qualification or a policy approval.

## Change

`scripts/check_thresholds.py` checks the threshold table's Decision cells.
Proposed thresholds require `DEC-023 YYYY-MM-DD` with a valid date. Existing
zero-valued absolute QA-040 constraints retain their `n/a (absolute)` marker.
The checker refuses missing tables, empty tables, malformed rows, missing
decisions, unconfirmed decisions, invalid dates and duplicate threshold names.
All ten existing core threshold names must appear exactly once. Only the three
existing absolute threshold names can use the zero-valued exemption; additional
rows require dated decisions normally. Removing required rows or changing them
into prose does not remove the release condition.
Instructional prose does not count as a decision. Blank lines cannot hide later
unconfirmed rows.

The release gate invokes this checker. CI runs the new regression suite alongside
the existing Python tooling suites. Canonical threshold values and decisions are
unchanged. The real document still fails because DEC-023 remains unconfirmed.

## Verification

- Eight new test methods, including malformed-decision cases and full-script
  fixtures, pass with `python3 -m unittest discover -s scripts -p
  'test_check_thresholds.py'`.
- The full-script confirmed fixture succeeds only in classifying its Decision
  cells. Unrelated checks remain `SKIPPED`, and human review requirements remain
  `MANUAL`; that result does not qualify a release.
- The full-script fixture using the unchanged current threshold document exits
  1 and records a failed threshold check.
- Python tooling suites, shell syntax, repository hygiene, action pins,
  traceability and diff whitespace checks pass.

## Independent review correction

The first checker revision (`ef8c3a6`, not published) accepted a table after all
seven proposed rows were removed or converted into tab-separated prose. It also
accepted an invented zero-valued absolute row. The independent reviewer
reproduced these cases at both checker and full-script levels. New regressions
failed in ten subcases before the follow-up fix. The explicit required-name set
and restricted absolute-name set now refuse every case, including a single
omitted required row and an invented absolute row appended to an otherwise
confirmed table. A valid additional row with a dated decision still passes.

This correction does not approve DEC-023, tag or publish a release, deploy the
application, or satisfy any manual or external qualification gate.
