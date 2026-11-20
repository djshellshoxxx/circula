# Independent review of PR #127 web CI repairs — 2026-11-20

Reviewed peer source: `82e61847339dd08fa8d69ebf57bc0c4c6bbf2e38`.
Base: `21933a3c650690db5a8e1e44cc236d8d1b5cdb24`.
Result: **No blocking source issue found in this three-file patch.**
This is technical source review, not a separate-account GitHub approval or
a substitute for current-head execution.

## Redirect construction

`internal/web/h_notify.go` constructs the success destination from the
current network's rooted path and the authorized case identifier, with
`url.PathEscape` applied to that one path segment. The identifier originates
from `GetReportDisputeContext`; target matching and transactional
`CreateReport` checks remain intact. Context-free reports still return to the
network home. Display-link fields no longer select the redirect destination.

The earlier G710 diagnostic is a source-analysis result, not proof of an
exploitable redirect. This change makes the trusted construction explicit.
The current lint command must establish whether its analyzer accepts it.

## Accessibility route coverage

`e2e/tests/a11y.spec.js` adds an actual request for the seeded independent
administrator's dispute report summaries. The seed dispute is between Alice
and Bob; the administrator satisfies the existing independent-decider reader
contract. The test requires HTTP 200, the exact report heading, the correct
back link, and the existing light/dark axe helper.

The template registry now references that real test. It does not mark the
page layout-only or remove the coverage assertion. Existing Go report tests
continue to cover populated summaries, authorization and confidentiality.

## Loading-state assertions

`e2e/tests/loading.spec.js` captures outgoing-document nodes before ordinary
keyboard or button activation. Polling reads those retained nodes through
their JS handle while the document response is intentionally withheld.
It avoids the locator assertion's pending-navigation wait.

Both checks retain exact announcement text, busy state and positive control
state assertions. They also check the live-region role and focus. The POST
check retains method, confirmation step, amount, counterparty, CSRF and
idempotency fields. The held response is released, native activation is
awaited, the handle is disposed, and the destination's reset state is checked.

The production `app.js` is unchanged. Its cancellation-aware delayed
announcement contract is preserved. No test, race mode, positive assertion,
native submission or required provider gate was removed.

## Evidence and limits

The complete peer commit diff, route constructor, service authorization,
seed fixture and light/dark axe helper were read. `git show --check` on the
reviewed commit exited 0. This review does not rerun heavy tests already
reserved by the coordinator.

Author-reported Node/static results are separately recorded by the author.
The local browser cannot launch in the managed executor, so neither this
review nor file-only checks establish a browser pass. Current-head lint,
real browser execution and complete combined qualification remain required.
