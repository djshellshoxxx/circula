# PR #82 independent fail-before evidence — 2026-11-18

## Exact inputs

| Item | Git object |
|---|---|
| Fixed PR head | `5c39abdb89faa505533e51d3878b3b119957ff32` |
| Fixed tree | `302beed8428063a8df33ab68ba3256c7e1976f5f` |
| Baseline main | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` |
| Baseline `internal/service/auth.go` blob | `527afe551387125448e0198eb945a24fa95e5448` |
| Fixed `internal/service/auth.go` blob | `97316e591698792ad3bece18063e4883aac4e3c2` |
| Supplied `internal/service/aud4_auth_test.go` blob | `156a1caa47479d234319f6227be51781c6070900` |

The detached worktree is
`/workspace/scratch/9ceb45b04668/audit-c-auth-qualification`. It is clean and
contains the exact fixed head. A Go overlay replaces only `auth.go` with the
baseline blob. The existing three supplied regressions are unchanged. No new
tests or adversarial probes were created, and no tracked source was modified.

The fixed `auth.go` is byte-identical to the previously qualified `0a7ef6e`
source. The test-file delta from that commit removes the separate F02 policy
test; the three tests selected below are retained.

## Independent baseline result

All three required regressions fail against the baseline implementation:

| Regression | Observed failure |
|---|---|
| `TestLoginStraddlingPasswordChangeGetsNoSession` | A session issued for the superseded password survives after both self-service password change and administrator reset; observed `must_change=false` and `true`, respectively |
| `TestChangePasswordReplacesOnlyTheVerifiedHash` | A change verified against a superseded password was applied |
| `TestChangePasswordGuessesAreThrottled` | All 30 current-password guesses were evaluated |

The command exits 1 with the expected assertion failures; package test time
is 0.073 seconds. There is no compilation failure in this baseline run.
Evidence: `PR82-auth-regressions-before.log`.

## Commands and overlay

Go 1.26.11 was used. All commands ran from the detached worktree with:

```sh
export PATH=/workspace/scratch/b4e4b2242853/toolchain/go/bin:$PATH
export GOTOOLCHAIN=local
export GOMAXPROCS=1
```

Baseline materialization and identity check:

```sh
git show cdc8112cf80ca51fdd220cd2f86c5180180cbe2a:internal/service/auth.go > /workspace/scratch/9ceb45b04668/audit-c/PR82-cdc8112-auth.go
git hash-object /workspace/scratch/9ceb45b04668/audit-c/PR82-cdc8112-auth.go
```

The hash-object output matches `527afe551387125448e0198eb945a24fa95e5448`.
`PR82-before-overlay.json` maps the worktree's `internal/service/auth.go` to
that baseline file.

Exact baseline command:

```sh
go test -overlay=/workspace/scratch/9ceb45b04668/audit-c/PR82-before-overlay.json -p=1 -parallel=1 ./internal/service -run '^(TestLoginStraddlingPasswordChangeGetsNoSession|TestChangePasswordReplacesOnlyTheVerifiedHash|TestChangePasswordGuessesAreThrottled)$' -count=1 -v > /workspace/scratch/9ceb45b04668/audit-c/PR82-auth-regressions-before.log 2>&1
```

Exact fixed-head control command, using the same existing tests without the
overlay:

```sh
go test -p=1 -parallel=1 ./internal/service -run '^(TestLoginStraddlingPasswordChangeGetsNoSession|TestChangePasswordReplacesOnlyTheVerifiedHash|TestChangePasswordGuessesAreThrottled)$' -count=1 -v > /workspace/scratch/9ceb45b04668/audit-c/PR82-auth-regressions-after.log 2>&1
```

## Fixed-head control limitation

The control command did not execute any tests. Its compiler process received
`signal: killed` while compiling `internal/service_test`, and the command
exited 1 with `[build failed]`. The cause is not established. This is preserved
in `PR82-auth-regressions-after.log`; no passing result is inferred from this
attempt and it was not repeated while the integration lead's aggregate suite
was running.

The requested independent fail-before evidence is complete. Fixed-source race
qualification reported by the other reviewer remains separate evidence. This
workstream ran no broader test selection and has no remaining test processes.
