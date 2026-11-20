# Independent review: dispute reviewer report projection

Date: 2026-11-20. Reviewer: `/root/review_integration` (AI).

## Provenance and scope

Reviewed worker `1995ad07957bbbf2c55c95a10373dda88e4fff91` and integrated
`2489a2eeadded31677443b1e08ea6f14cd3eaa60`. All 24 changed paths have identical
bytes between those commits. This review read the runtime diff, existing
authority/snapshot/cursor helpers, regression fixtures and retained evidence.
It ran no live requests or disclosure reproduction. A later, separately
authorized repair and benign repository-failure test are identified below.

## Finding and assigned repair

**P2 — preserve the new route's repository lookup errors.** In
`internal/service/dispute_reports.go`, `ListDisputeReports` replaces every
`GetDispute` error with `domain.NotFound("dispute")`. Store `GetDispute` already
normalizes only missing rows and preserves typed unavailable/read errors.
`Service.view` and `SQLite.run` return the callback error; they cannot restore
a discarded cause. An ordinary temporary lookup failure therefore becomes
HTTP 404 instead of retryable 503, and other read failures lose diagnostics.
This conclusion follows directly from source and adapter error mappings.

Parent assigned the narrow repair from
`d324b9d7b129e5b723e59ea1d8ccc0dac8f10cfd`. It is committed as
`7dc19820aaf06a25e9210d91e775043071b508de`: normalize only `CodeNotFound`, return
other errors unchanged. Its benign fixture wraps only the selected valid
dispute's repository lookup after ordinary setup, preserving the real snapshot
and `GovRepo` policy-clock extension. It covers unavailable, wrapped unavailable,
ordinary read failure, cancellation, deadline and plain/wrapped not-found.
The older `GetDispute` detail method's pre-existing error pattern is explicitly
outside this patch. Focused execution evidence is recorded separately.

## Remaining reviewed boundaries

No additional actionable finding in the requested projection boundaries:

- **Minimal data:** explicit service/API/HTML structures contain only report
  ID, category and status; they do not embed the administrator report. Reporter
  identity, notes, handler identity and timestamps are excluded. IDs, accepted
  categories and open/handled statuses are checked before returning rows;
  invalid stored content causes a generic error and discards the page output.
- **Independent authority:** each page resolves a current active reviewer or
  administrator and applies the existing party/proposer/acceptor/reversal
  participant check. The existing trusted System read remains explicit.
  Queue access and handling rights are not expanded.
- **One snapshot:** principal, dispute, transaction, independence lookup and
  report selection run within the same SQLite read snapshot. Each later page
  opens and authorizes a fresh snapshot.
- **Filter before pagination:** the service derives the exact transaction
  from the network-scoped dispute. SQLite applies network, unresolved-subject
  exclusion and target predicates before the keyset boundary/order/limit.
  Callers cannot select a target, and row-iteration errors propagate.
- **Cursor/navigation:** the API accepts only limit/cursor, using the existing
  binding to dispute path, network and person. HTML paging retains the dispute
  path. Both adapters perform fresh service authorization on page load.
- **Authority-only links:** `CanReadTransactionReports` depends on independent
  authority without reading report presence/counts. Ineligible detail readers
  receive no report URL; general dispute lists receive no report context.
  Existing party detail reads continue on expected independence refusal;
  unexpected independence-query failures propagate.
- **Existing behavior:** full administrator DTOs, handling, creation/replay,
  notifications, audit and export paths are not widened. This projection adds
  no migration or database write. The category helper preserves the vocabulary.

## Evidence independently inspected

Receipt `analysis/dispute_report_projection_verification.json` at review time:
SHA-256 `404e465d5ab9f22e9aaaa97cebee90c51fed2796b6c484adbc3c301b9989724a`.
The reviewer parsed the raw logs and verified each recorded hash:

| Log | Observed outcome | SHA-256 |
|---|---|---|
| `dispute-report-focused.jsonl` | Five package selections passed: 26 top-level and 31 subcases; service compiler killed before tests. | `c9bb9e432cf36a751426eb810fc81b7e8b3cdcc2065a6ff9230aab07959655e8` |
| `dispute-report-service-retry.jsonl` | Compiler killed before execution; no test pass. | `e95bb551bd4de8001245519f1b6fd9268abb3a4e72688414eab46ccdcef88941` |
| `dispute-report-service-relocated-cache.jsonl` | Unchanged service selection passed: 15 top-level and 27 subcases, 35.018 s. | `8c16e1b44110ce170e41bdbb21435c254aeac6406143fce055cd833271ec1286` |

The successful selections account for six packages, 41 top-level passes and
58 subcases, with no executed-test failure or race warning. Both killed builds
remain retained inconclusive attempts. Fixtures cover narrow fields, exact-case
paging, current-role revocation/suspension, party and reversal conflicts,
invalid stored categories, hidden party links, read-only behavior, route
authorization and unchanged full administrator details. The read-error finding
is additional source-review feedback, not an executed failure in those logs.

This is AI review and inspection of focused evidence, not GitHub approval,
external security qualification, a full integrated suite pass or deployment
acceptance.

## Repair verification addendum

The isolated repair commit `7dc19820aaf06a25e9210d91e775043071b508de` passed
`TestDisputeReportsPreserveDisputeReadErrors` and all seven of its subcases
under Go 1.26.11 with `-race -p 1 -count=1`, one CPU, 256 MiB memory target and
workspace build-temporary files. The full service package compiled; its test
elapsed time was 2.247 s, total command wall time 41.986 s. The source hashes
did not change during execution. This closes the single finding in the
isolated repair; parent integration remains separately tracked.

Raw log: `analysis/dispute-report-read-errors-focused.jsonl`, SHA-256
`8b3a7334c79a6d720b3d3544b4dc82c293c0a2ac86ba91a50bd7e84c5606e258`.
Receipt: `analysis/dispute_report_read_errors_verification.json`, retaining
the committed source ID, both source-file hashes, exact command/environment
and complete result. No additional full-suite claim is made by this check.
