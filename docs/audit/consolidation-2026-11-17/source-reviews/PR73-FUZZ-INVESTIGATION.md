# PR 73: timed fuzz job investigation

## Recommendation

One retry of the failed job on unchanged head
`cb25b41febd43273cd4aa856072406b210089d81` is supported by the observations
below. Preserve the original failure and require the retried gate to pass. No
timeout, target, corpus, application source, or workflow changes were made.
If the same failure recurs, investigate further rather than repeating retries.

## Exact-head evidence

- Provider run: `44247617122`; failed fuzz job: `134610272040`.
- Commit tree: `aa7dd4e4520ac5c293f80f5b6f1acdb79fc2c409`, also the tree of
  local commit `4c26a7ed443d47e339d3bdac7b8f17d3b4e6fe58`.
- `git diff HEAD^ HEAD -- internal/domain .github/workflows/ci.yml` is empty.
- The provider used Go `1.26.11` on Linux/amd64. The command was:

  ```sh
  go test -run '^$' -fuzz '^FuzzParseAmount$' -fuzztime 10s ./internal/domain
  ```

- CI completed all 21 seed cases, fuzzed with four workers, and recorded
  1,158,695 executions with 69 new interesting inputs. It failed at 10.098s
  with only `context deadline exceeded`. The complete log contains no target
  assertion failure, crash stack, or failing-input corpus path.
- Exactly one local reproduction ran that command on the exact head in the
  isolated detached worktree `audit-f-fuzz`, with the pinned Go 1.26.11
  toolchain and `GOMAXPROCS=4` matching the provider's four workers. All 21
  seed cases completed, followed by 1,718,765 executions and 64 new interesting
  inputs. It passed at 10.088s. The working tree remains clean.

## Source-supported explanation and its limit

The target in `internal/domain/fuzz_test.go` parses an amount and, when valid,
checks non-negativity and an exact formatting/parsing round trip. The target
and `internal/domain/money.go` have no context, timer, background goroutine,
or external dependency that returns this context error.

In the pinned Go 1.26.11 source:

- `src/internal/fuzz/fuzz.go:107` creates the timeout context from the fuzz
  budget, and line 112 creates its worker child context.
- At line 228, expiration of the parent invokes `stop(ctx.Err())`.
- At line 129, `stop` suppresses the error if it equals the child context's
  error. Otherwise a non-nil error can be retained and returned at line 235.
- `src/context/context.go:561-571` stores the parent's error and closes its
  done channel before propagating cancellation to child contexts.
- `src/context/context.go:463-470` allows a child with no stored cancellation
  error yet to return nil from `Err()`.

Consequently, a scheduling interval between parent notification and child
cancellation is a plausible explanation for the exact budget-expiration
message. These source observations do not prove which interleaving occurred
in the provider job, and this report does not claim a confirmed Go defect.
A passing randomized local run also cannot rule out every input-dependent
defect. The timing, absent target diagnostic, unchanged target, and successful
single reproduction jointly support the limited same-head retry.

## Preserved evidence

- `audit-f/pr73-fuzz-job.log`: complete provider job log.
- `audit-f/pr73-fuzz-local.log`: complete local reproduction output.
- Pinned toolchain source:
  `/workspace/scratch/b4e4b2242853/toolchain/go/src/internal/fuzz/fuzz.go`
  and `/workspace/scratch/b4e4b2242853/toolchain/go/src/context/context.go`.

No provider write or repository change was performed by this investigation.
