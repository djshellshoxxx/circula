# Independent UX-047 investigation — 2026-11-28

## Exact source and ownership

Investigated the fresh private worktree `ux047-independent-main` at main
`3faeed76c41467a75b09bc75e7185a98a92f8219` (verified with `git rev-parse HEAD`).
This was a read-only investigation. No product file, test, branch, PR or provider
record was changed. No future module was inspected or executed.

The investigation excluded the owned #142 notification/inbox implementation and
catalogue/test files, plus #143's render/account/shared-shell/JavaScript/session
warning implementation and tests. Reading catalogue entries to understand callers
did not change that ownership. Policy decisions and held PRs were not investigated.

## Verdict

No confirmed eligible wrong-number, wrong-plural or raw-label wiring defect was
found in the bounded scan. There is no evidence-backed failing regression proposal
from this investigation. This is not a claim that whole-UI localization, UX-024,
or UX-047 is complete.

## Evidence and eliminated candidates

* Enumerated all 37 catalogue entries with a `one` form and inspected direct core
  production callers. Count-bearing calls use the relevant count first; the
  previously completed precision-family callers likewise pass precision first.
  No new precision-family defect was confirmed.
* `overview.stale` and `overview.invites` have a second duration argument expressed
  in hours. A hypothetical one-hour or sub-hour configuration could produce an
  incorrect duration word or truncated count, but this is not reachable from the
  current service: `internal/service/overview.go` fixes `OverviewStaleAfter` to
  48 hours and `OverviewInviteWithin` to 72 hours and assigns those constants to
  every returned overview. This candidate was ruled out, not reported as a bug.
* Dispute list/detail/review template `.Status` and `.Category` fields already
  receive display labels from their handlers. Transaction rows map every current
  status through `statusLabel`; member status/role and break-glass status likewise
  use existing catalogue entries. Unknown-value fallbacks preserve raw historical
  data deliberately. Raw-looking template property names alone do not establish
  a migration gap.
* The listing's dynamically constructed `listing.tag.*` key covers each current
  domain status (`active`, `paused`, `archived`). No missing current status label
  was identified.
* Searched direct `strconv.Itoa`, `strconv.FormatInt` and `fmt.Sprintf` calls outside
  owned files. Remaining uses were Retry-After headers, signature/protocol input,
  export names, version identifiers and ledger/cursor identifiers. These are not
  evidence of an unformatted display count.
* Searched template `printf`, `.Days`, `.Hours`, `.Count`, `.Total` and common
  status/category/role fields. Counted directory/member/queue/inactivity/closure
  notices already use catalogue messages. Other numeric uses were identifiers or
  machine comparisons. The admin overview's `.Count` is a conditional link gate;
  its displayed `.Text` is already formatted.
* Searched explicit time formatting outside the owned files. No additional direct
  presentation `time.Format` caller was found beyond the existing messages date
  helpers. This does not close #143's separately documented nil-viewer context
  assessment.
* Inspected `format.go`, `refusal.go` and member action/role label wiring. No
  existing-catalogue key was found to be wired to a wrong current value. Member
  action labels added by erasure/reset-password initialization are present; their
  apparent absence from the initial action map is not a missing-label defect.

## Execution and limits

Only filesystem/Git identity checks and source searches/reads were run. No Go,
browser, Node, race or HTTP test was executed for this investigation. The known
prior resource-killed SQLite compilation attempts were not repeated. No local
execution or acceptance claim is inferred from prior authors' or hosted results.

This scan specifically sought a small repair that could reuse existing catalogue
entries without overlapping #142/#143 paths. It did not assess translations,
new-locale scope, manual screen readers, independent security review, held owner
policies, future modules or whole-UI acceptance.
