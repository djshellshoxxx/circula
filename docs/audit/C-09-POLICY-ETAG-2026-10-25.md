# C-09: pending currency-name policy validator candidate

Date: 2026-10-25. Status: regression prepared; runtime reproduction blocked.
Repository: `djshellshoxxx/circula`. Scope: local source and tests only.

## Source pin and rationale

Starting checkpoint: `91851e0560c4c7a656968aa56a0f14ecec511d99`.
Last observed public source: `30d18a1626427fa6aa14f70c8df05f3e807efab5`.
The existing independent-founder browser fixture and all earlier local handoff
commits are preserved. One existing inexpensive worker prepared the regression;
the coordinator reviewed its scope, clock setup and assertions.

Production source blobs at the starting checkpoint:

| Path | Git blob SHA |
|---|---|
| `internal/api/h_policy.go` | `dba9626b5eb57fe6281f401d6c94c77a3813c3db` |
| `internal/service/networks.go` | `721e6bc3d46891373c2eadfd16d34b21b24151c4` |
| `internal/api/helpers_test.go` | `0b13f50ccb61277317e9121e60c989299f7471f6` |
| `internal/testutil/testutil.go` | `e2e31f246973e980d455336698cbe825d8033c85` |

The candidate concerns API-026/027/058 in
[the API contract](../specs/07-api-contract.md): a policy representation with a
new pending change should have a changed strong validator, and a distinct update
using the superseded `If-Match` should return 412 without changing state.

Static inspection identifies a possible mismatch: `policyETag` hashes the
network ID and effective `PolicyVersion`; GET adds a `pending` representation
after setting that tag. Scheduled policy updates append a later version while
the effective policy remains unchanged. This is a source-based hypothesis,
not an observed runtime failure.

## Prepared regression

`TestPendingCurrencyNamePolicyPatchChangesETag` in
[`internal/api/more_test.go`](../../internal/api/more_test.go) uses:

- A fixed `2026-03-01T12:00:00Z` API clock, equal to the existing service fake
  clock; neither clock advances.
- Only `currency_name`, required `reason`, and `notice_days: 14` in the scheduled
  PATCH. No amounts, decimal scale, limits, thresholds, financial policy or
  dual-control roles change.
- Pending-state assertions before validator assertions: the name change is
  pending, effective time is exactly 14 days later, and the active version/name
  remain unchanged. The existing wire field is `pending.changes.currency_name`.
- Strong changed ETag checks and agreement with the following GET. These checks
  are nonfatal so the stale-condition assertion can also execute.
- A different name in the stale request, avoiding an identical idempotent replay;
  expected 412 `precondition-failed`, followed by unchanged active/pending state.

All pre-existing test assertions are byte-for-byte preserved after excluding the
new test and required imports. Production source is unchanged. The independent
founder approval fixture is also byte-for-byte unchanged from the starting pin.

## Commands and actual results

With the existing verified local Go toolchain, the final focused attempt was:

```sh
PATH="$PWD/.tools/go1.26.10/go/bin:$PATH" \
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
GOCACHE=/tmp/circula-go-cache GOMODCACHE=/tmp/circula-go-modcache \
go test -count=1 -p 2 ./internal/api \
  -run '^TestPendingCurrencyNamePolicyPatchChangesETag$'
```

Result: exit 1, package **setup failed before compilation**. The local cache is
missing `github.com/google/uuid@v1.6.0`, `golang.org/x/text@v0.43.0`,
`golang.org/x/crypto@v0.58.0` and `modernc.org/sqlite@v1.60.1`.
Reported errors include:

```text
internal/domain/fold.go:6:2: module lookup disabled by GOPROXY=off
internal/store/sqlite.go:15:2: module lookup disabled by GOPROXY=off
FAIL github.com/djshellshoxxx/circula/internal/api [setup failed]
```

Go prints "downloading" while resolving missing modules, but `GOPROXY=off`
refused their lookup; no network download was enabled. An earlier offline API
package attempt encountered the same setup blocker. The old HTTP 403 download
and artifact-byte denials were not retried or routed around.

`gofmt` and `git diff --check` passed. Static preservation checks passed for the
old API assertions and the founder-approval fixture. No aggregate Go tests,
browser run or new CI execution occurred after the focused setup failure.

There is **no runtime failing-before evidence for C-09**, and no production fix
has been made. Preserve the original command/output distinction: a dependency
setup failure is not reproduction of a validator or HTTP-status defect.

## Next range and gates

1. When dependencies are lawfully available locally, run the focused command
   unchanged and preserve the actual result before editing production code.
2. If the pending setup is valid and the validator/status assertions fail,
   inspect `internal/api/h_policy.go` (`policyETag`, GET tag assignment, PATCH
   `requireIfMatch` and response), then the scheduling branch of
   `internal/service/networks.go` and `PolicyLog` in `internal/service/gov_terms.go`.
3. Any fix must consistently account for the pending representation in both
   reads and conditional updates, remain deterministic at a fixed clock, and
   preserve authorization, idempotency and original policy values. Do not use
   wall-clock churn to make the test pass.
4. Run this focused regression first, then the existing policy/listing ETag test
   and API package checks. Keep the unresolved browser fixture validation and
   broader acceptance gates distinct.

Publication remains approval-blocked. No push, PR, comment, merge, CI trigger,
deployment or release was attempted for this candidate.
