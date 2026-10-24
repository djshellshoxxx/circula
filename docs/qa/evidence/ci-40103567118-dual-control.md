# CI 40103567118: dual-control fixture diagnostic

Date: 2026-10-24. Status: bounded local candidate; browser validation pending.
Repository: `djshellshoxxx/circula`. Published source under test:
`30d18a1626427fa6aa14f70c8df05f3e807efab5`.
Local diagnostic starts at `59b24ba7ffb51fe6cdcaa9c3fef274556cf6aba3`;
that unpushed handoff and its ancestry are preserved.

## Recorded CI result

[Run 40103567118](https://github.com/djshellshoxxx/circula/actions/runs/40103567118)
completed with eight successful non-E2E jobs: vet/lint/vulnerability checks,
Ubuntu and macOS race tests, Windows tests, cross-build, API/load smoke, fuzz
smoke and reproducible build. These are remote results for the published SHA,
not local runs or results for the candidate below.

The [E2E job 120900564017](https://github.com/djshellshoxxx/circula/actions/runs/40103567118/job/120900564017)
failed: 98 passed, one failed and three did not run. The reported failure in
`e2e/tests/security.spec.js:223` followed keyboard activation of `Approve and
post`; the expected success element `#flash` was absent within 5000 ms.
The read-only job log confirms the counts and references both screenshots and
the error context beneath
`test-results/security-axe-and-keyboard--1da1d-with-a-second-administrator/`:
`test-failed-1.png`, `test-failed-2.png` and `error-context.md`.

## Artifact evidence and access limit

Preserve the original screenshot/error-context artifact; do not replace it with
candidate-run output:

- Name: `e2e-test-results`; artifact ID: `12563504549`.
- Source run/head: `40103567118` /
  `30d18a1626427fa6aa14f70c8df05f3e807efab5`.
- GitHub-reported archive size: `167877` bytes.
- GitHub-reported SHA-256:
  `f470d9b95d8e01d288a07095d9a6cdf15b944347570c7ad5f66e7891f9d9ef00`.
- GitHub-reported expiry: `2026-10-31T14:44:56Z`; not expired at inspection.
- The read-only artifact connector returned a reusable ZIP file reference
  (`file_00000000d4d081f5aae23489b5386faf`). Local retrieval from the returned
  `*.oaiusercontent.com` URL failed with
  `URLError: <urlopen error Tunnel connection failed: 403 Forbidden>`.

No local archive bytes, screenshot or error-context contents were obtained.
The digest above is GitHub metadata, not an independently computed local hash.
No alternative download route was attempted after the denial. The original
artifact remains referenced on GitHub and by the connector file reference;
complete local archival remains blocked and should be completed before expiry
if an authorized access route becomes available.

## Source-confirmed fixture eligibility error

1. [`secfixture.js`](../../../e2e/secfixture.js) appoints `admin2` through the
   founder's `grant-admin` action immediately before restarting the test server.
2. [`requireIndependentApprover`](../../../internal/service/dualcontrol.go)
   enforces the 24-hour new-admin delay and rejects approval by an administrator
   appointed by the requester. The original test direction violates both rules;
   waiting 24 hours alone would not satisfy the grant-independence rule.
3. [`pendingAct`](../../../internal/web/h_dualcontrol.go) renders a refusal as
   HTTP 409 on the pending-list page and sets a success flash only after approval.
   The [actual form](../../../internal/web/templates/admin_adjust_pending.html)
   posts directly to the approval action; visibility of the button does not
   establish eligibility under the service's grant-chain rules.
4. [`createAdjustment`](../../../internal/service/adjustments.go) permits an
   active administrator to request a pending adjustment. Approval then separately
   checks independence, requester/target separation and current authority.
5. Existing regression
   `TestA3_01_IndependentApproverSucceeds` in
   [`a3b_test.go`](../../../internal/service/a3b_test.go) uses the valid direction:
   an appointed administrator requests and the founder approves. The founder's
   creation-time root grant does not descend from the requester and has the
   explicit root-grant exemption from the new-admin wait.

The local candidate swaps only the two login identities in this E2E scenario:
`admin2` requests; founder `admin` approves by keyboard. An explanatory comment
records why those roles are eligible. The existing button visibility, absence of
self-approval, keyboard, flash and axe assertions are retained. There are no
longer waits, skipped assertions, backdated grants, disabled eligibility checks,
backend changes or new dependencies.

This source path explains the expected refusal and is consistent with the CI
symptom. It is not screenshot corroboration or a successful browser reproduction.

## Local checks and next validation

- `node --check e2e/tests/security.spec.js`: passed.
- `git diff --check`: passed.
- `scripts/check-hygiene.sh`: passed.
- `scripts/check-action-pins.sh`: passed.
- Coordinator source review: candidate matches the existing positive service
  regression and preserves security and accessibility assertions.
- Full local browser/service rerun: not performed. The previously recorded
  SQLite module download HTTP 403 remains unresolved; it was not retried.
- No push, PR creation, merge or CI trigger was attempted during this diagnostic.
  The earlier remote-publication approval rejection remains unresolved.

When dependency and publication access are authorized, run the focused service
regressions, the dual-control browser scenario, then the full security browser
suite and required CI gates. Do not infer success for the three tests that CI
did not run.

```sh
go test -count=1 ./internal/service -run '^TestA3_01_(IndependentApproverSucceeds|NewAdminWaits|SybilAdminCannotApprove)$'
make build
cd e2e
CIRCULA_BIN=../bin/circula npx playwright test tests/security.spec.js --grep 'dual-control pending adjustments'
CIRCULA_BIN=../bin/circula npx playwright test tests/security.spec.js
```
