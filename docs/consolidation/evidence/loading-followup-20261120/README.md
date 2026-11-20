# Loading qualification follow-up — 2026-11-20

This package records the actual browser failure on Core source
`bea5de25cf76b66d87689034e2529f9b40c6dff6`, fresh independent review of its
correction, and focused execution against repair
`eef0c6d8c832ce994db1019eb5e7d6a41800b8ef`. It supplements the frozen earlier
[complete qualification evidence](../final-qualification-20261120/README.md)
and [subsequent future-integration receipt](../future-ref-review-late-20261120/README.md).
Earlier failures, source pins and skips remain unchanged.

## Source and review boundary

The repair's complete tree is `029e5fdda2d03347389397576ec08701ae04cec2` and
its sole parent is the current Core source above. Relative to that parent,
PR129 changes only `e2e/tests/loading.spec.js`. Relative to earlier locally
qualified Core `be9a5731bc93dcbf7b57301a8d2538a13af360bd`, the complete delta is
exactly that test file, `internal/web/static/app.js`, and
`scripts/test_app_messages.cjs`.

Two fresh reviewers separately examined production behavior and test
qualification. No blocking finding remained in this scoped correction. The
[independent review](independent-review.md) records listener ordering,
cancellation, native submission and retained assertions. Source and dependency
hashes are in the [execution receipt](local-checks/qualification.json) and were
checked against committed blobs again during packaging. No Go, dependency,
workflow, policy or future-module change is introduced by this correction.

## Actual executed checks

| Check | Source | Result |
|---|---|---|
| Node loading/message helper suite | Exact repair `eef0c6d…` | 16 passed, zero failed/skipped, exit 0 |
| Old-production negative control | Exact current helper plus exact `be9a573…` app.js in an isolated temporary directory | 15 passed, one expected failure, exit 1; only the new synchronous-feedback regression fails |
| JavaScript syntax | Exact repair | Exit 0 |
| Complete changed-source whitespace | `be9a573…` to `eef0c6d…` | Exit 0 |
| Playwright discovery | Exact repair, unchanged Playwright 1.63.0 dependency lock | 597 cases in 20 files, including all four loading cases; zero browser tests executed locally |
| Provider browser job `136617856750` | Exact repair `eef0c6d…`, run `44831668852` | 597 passed, zero failed; all four loading cases passed |
| Provider Linux race job `136617856302` | Exact repair `eef0c6d…`, same run | Failed: service package reached the 20-minute Go timeout; not a qualifying pass |

The mixed-source negative control is deliberately distinct from exact-candidate
execution. It establishes that the regression detects the earlier deferred
feedback. Its failure is retained as expected evidence, not converted to a
candidate pass or suppressed.

The four-entry original manifest and the decoded bytes of all three raw logs
are preserved in `local-checks/`. The negative-control log is transported as
lossless gzip because its original failure excerpt contains whitespace-only
lines; the [transport receipt](local-log-transport.json) verifies its exact bytes.
The other original files remain plain text. The execution receipt records exact
commands, source identities, Node
version, environment, duration, exit code, event counts and limitations. The
earlier Chromium socket-permission failure was not retried or bypassed.

## Original provider failure

[Core run 44815660965](https://github.com/djshellshoxxx/circula/actions/runs/44815660965)
on `bea5de25…` passed eight of its nine jobs and failed browser job
`136563225100`. That job reports **595 passing cases and two failures**, both
at the old `readLoadingState(outgoing)` predicate after a document request was
deliberately held. Its complete decoded UTF-8 log is stored losslessly in
`browser-failure-136563225100.log.gz`. The
[packaging receipt](source-and-packaging-verification.json) records its
136,343 uncompressed bytes and SHA-256
`a7f244047ba6f5a6c32c8ae37e6d67d955868c0ed83e17d8f48979b4302d2d61`.

The new observer captures the genuine DOM/ARIA values within the initiating
native event, after the production listener has run. A uniquely marked console
message transports the snapshot without asking the pending document to run a
later evaluation. Real held GET/POST requests, submitted values, focus, enabled
controls, destination/success and reset assertions remain. Provider browser
execution subsequently passed all 597 cases, as recorded in the
[success receipt](browser-success-receipt.json) and losslessly compressed log.
DOM/ARIA observations do not independently measure rendered pixels or
screen-reader announcements.

## Linux deadline repair

The same exact-head provider run finished with **six successful jobs, one failed
Linux race job, and one conditional native-matrix placeholder skipped**. The
skip is not counted as a pass. Its [failure receipt](linux-timeout-receipt.json)
and complete log preserve the `20m0s` package panic and service elapsed time of
`1200.102s`. `TestQA001InvariantSequences` had run for 23 seconds when the package
deadline expired. The log contains no assertion-failure or data-race marker;
that observation does not establish completion of the suite.

The earlier verified full-race archive records a successful service package in
`1294.982s`, exceeding the old deadline. The scoped workflow repair changes the
Linux package timeout from 20 to 30 minutes and job timeout from 30 to 45 minutes.
All test selectors, race instrumentation, property seeds, fuzz budgets, platform
matrices, assertions, action pins and reporting remain. The
[patch](linux-budget-change.patch) and
[local verification](linux-budget-local-verification.json) record the complete
change, semantic YAML comparison, action-pin check, repository hygiene check and
whitespace check. Its new exact source must finish provider CI before integration.

The [independent workflow review](linux-budget-independent-review.md) cleared
local commit `a4becfda0f10d5fa3790bf0ee4aeff6552dfeca1`. Published successor
`11739c7f8f59455b4dc480294cab4d79f7933cb9` has identical complete tree
`43e65ee91a90ff70b70cc69d0b517b8367a2585f` and sole parent `eef0c6d…`; the
[publication receipt](linux-budget-publication.json) and post-fetch comparison
verify both. The [provider snapshot](provider-after-budget-publication.json)
records seven applicable jobs queued and a conditional placeholder skipped in
run44838129155. That snapshot does not claim the new source passed.

## Verification and subsequent actions

`SHA256SUMS` covers this package except itself. From this directory, run:

```sh
sha256sum -c SHA256SUMS
gzip -cd browser-failure-136563225100.log.gz | sha256sum
python3 - <<'PY'
from pathlib import Path
import gzip, hashlib
root = Path('local-checks')
for line in (root / 'SHA256SUMS').read_text().splitlines():
    expected, name = line.split(None, 1)
    path = root / name
    data = path.read_bytes() if path.exists() else gzip.decompress(Path(str(path) + '.gz').read_bytes())
    assert hashlib.sha256(data).hexdigest() == expected, name
print('All four original checksums match decoded bytes')
PY
```

The [PR80 reconciliation receipt](pr80-reconciliation.json) records a separate
verified duplicate cleanup: its complete history and original tests were
already preserved in the accepted feed target. PR80 was closed without another
merge; its source branch remains. The independent future-ref package supplies
the complete source, job and closure readbacks.

The [PR87 reconciliation receipt](pr87-reconciliation.json) separately records
the dispute-timeline work already incorporated in accepted main. Its complete
18-path comparison, original tests and OpenAPI additions were verified before
closure. No new merge or source-branch deletion occurred. The
[Core containment review](../core-containment-20261120/README.md) identifies older
Core PRs that can only be reconciled after qualified integration into actual main.

The [current-source overlay](current-source-containment-overlay.json) preserves
the independently checked ancestry/hold conclusions after the workflow-only
change. Separately, all 102 missing PR104 documentary paths were recovered in
the [historical package](../pr104-documentary-recovery-20261120/README.md).
The [coordinator verification](pr104-coordinator-verification.json) repeats its
integrity checks, matches every archived Git blob to the original source, and
confirms exact equality with the containment review's missing-path set. Durable
publication and actual-main integration remain separate PR-closure prerequisites.

Later provider results and integration outcomes are recorded separately in
the continuation report and provider receipts; a source review, discovery,
older-head pass or unchanged subset does not qualify a new complete Core head.
