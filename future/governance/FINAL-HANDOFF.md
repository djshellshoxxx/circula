# FNET-208 master milestone 1 handoff

Branch: `codex/future-governance-master-20261116`. Worktree: `/workspace/scratch/4f14005536ba/work-governance`. Recovered predecessor: `e4381aa19292f1d4b24f09e1de51f3e2e5a2930b`. Exact tested implementation: `8b74e7db0e237df15acb46a27609f6015eadb4ed`. Later milestone commit only records documentation/evidence.

## Completed functionality

Strict bounded query grammar before host authentication; signed cursor validation before mutations; URL/body proposal agreement while native forms retain current URL; empty-cursor compatibility; fresh same-session authorization after CSRF token callbacks before private detail/list/error reads; fail-closed private error reload; bounded strict form parsing before host CSRF lookup can cache partial values.

Changes: web.go and master_regression_test.go plus master documentation, README, manifest and contract. No service/schema/dependency/core changes. No automatic financial policy application. Source tests exercise fresh callback revocation/replacement, member disable, cancellation, malformed/ambiguous transport and no-mutation rejection.

## Validation

Final source focused PASS0.126s, uncached race PASS5.824s/75.3%, vet and module verify PASS. Parser-only fuzz1,742 executions at earlier correction with unchanged selected parser source. Static notice-path/HOLD/JSON/diff checks passed. See VALIDATION.md for commands, earlier failures and exact evidence scope.

## Reuse and architecture

No new upstream code. Existing GOV.UK Frontend6.6.0/a11bd3316363ab4702fcef81d6ce06a947c464ef native markup adaptation, MIT license; modernc.org/sqlite v1.60.1 test driver and exact pinned imported dependencies retain notices. UPSTREAM-REUSE-MATRIX.md and THIRD_PARTY_NOTICES.md map all retained material. Runtime remains host-supplied database/sql and trusted current Authority callbacks, no automatic registration or background job.

## Remaining work and rotation

The coordinator independently reviewed both the initial and follow-up corrections; no additional defect was identified. Weighted acceptance is84/100 under the explicit master categories. A fresh next team should verify this handoff before the next milestone. Browser/keyboard/screen-reader/translation acceptance is unexecuted. Host operating retention/backup policy and actual production auth/admin-MFA/source contract need qualification; core integration is separately deferred and unauthorized. No known uncorrected defect is claimed from this bounded inspection; passing tests do not prove absence of others. No whole-product completion.

Next team: fresh governance reviewer and accessibility/operations qualification specialist. Preserve the exact branch; do not reset from older source. Read master directive and these current records before further source changes. Stop implementation at this committed milestone; reopen only from a fresh team's verified handoff.

```sh
cd /workspace/scratch/4f14005536ba/work-governance/future/governance
export GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB
export GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache
/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go test -p 1 -race -cover -count=1 -timeout 90s ./...
/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go vet -p 1 ./...
```

Acquire coordinator compiler slot first. Only future/governance may change. Keep enabled=false and integration HOLD. No push or other GitHub writes, core wiring, root dependencies/routes/migrations, main merge, deployment or release.
