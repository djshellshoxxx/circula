# CI workflow qualification — 2026-11-11

Scope: `.github/workflows/ci.yml` on branch `haiku/h08` (base `d40ef1d`). Item 4 of the
duplicate-run and matrix review. This record covers duplicate-run removal, dependency-cache
keys, explicit test parallelism and a permission and pin audit. It does not change any Go
code, any job name, any scanner, or the macOS and Windows matrix.

**Limits of this record.** GitHub Actions was not run for the changed workflow, so no green
run is claimed for it. The only runs cited are existing runs on other commits, read through
the GitHub Actions API. Nothing here is a release or merge gate.

## Before (measured)

Source: GitHub Actions runs for `2078bf0964353892e80214b4601211e36818b889`. The `ci.yml` at
that commit is byte-identical to `ci.yml` at this branch's base, so these times apply to the
same workflow. Each commit produced two runs of the full 9-job matrix:

| Run | Event | Workflow wall clock | Notes |
|---|---|---|---|
| 42880207869 | push | about 16.3 min | macOS job 16.1 min; race step 14.7 min |
| 42880245458 | pull_request | about 12.5 min | macOS job 12.2 min; race step 11.4 min |

Step times from the same runs (pull_request run, push run in brackets):

| Step | Time |
|---|---|
| macOS Intel race suite (`-p 2 -timeout=25m`) | 11.4 min [14.7] |
| Ubuntu race suite | 7.5 min [7.7] |
| e2e (`make e2e`) | 5.0 min [5.6] |
| Windows `go test` | 3.3 min |

Approximate job-minutes for one full run, summed from job timestamps: about 36 (macOS 12.2,
Ubuntu test 7.7, e2e 5.6, Windows 3.8, build 2.5, fuzz 1.6, reproducible 1.4, lint 0.8,
contract about 0.2). Before this change a PR commit therefore cost about 72 job-minutes,
because the push run and the pull_request run both executed the full matrix.

The coordinator's brief quoted macOS race 7.9 min, Ubuntu race 6.8 min and e2e 4.7 min. Those
figures were not reproduced from the runs above and their source was not found in this
repository, so they are not used here.

## After (changed)

| Change | Where | Effect |
|---|---|---|
| `push` limited to `branches: [main]`; `pull_request` unchanged; `workflow_dispatch` added | `on:` | A PR commit runs the matrix once (pull_request). Main still gets a post-merge run. Manual runs remain possible on any branch. |
| Job ids and names unchanged | all jobs | Required PR check names do not move. |
| `cache-dependency-path: go.sum` on every `cache: true` setup-go step (6 jobs) | lint, test, build, fuzz, contract, e2e | Dependency cache key is explicit and fixed to go.sum. |
| `reproducible` job keeps `cache: false`, now commented | reproducible | Two-build comparison is unchanged. |
| `go test ... -p 4` on the Linux race step and the Windows test step | test matrix | Package parallelism explicit. macOS keeps `-p 2`. |
| `concurrency` group and `cancel-in-progress: true` | top level | Unchanged. |
| `permissions: contents: read` (top level) | ci.yml | Unchanged, verified. |
| release.yml and scorecard.yml | — | Unchanged. Release keeps `cache: false`, no `concurrency` block and no cancel-in-progress. |

Expected effect, not measured: about 36 job-minutes per PR commit instead of about 72. MacOS
remains the largest share. Times for the changed workflow must be taken from its first real
run.

## Verification performed locally

- All three workflows parsed with `python3 -I` and PyYAML 6.0.1 (`scratchpad/h08/verify_wf.py`,
  output in `scratchpad/h08/verify.log`).
- `needs:` references checked. No `needs:` target is missing.
- Job ids and names compared with `d40ef1d`: identical for ci.yml (7 jobs), release.yml (3)
  and scorecard.yml (1).
- Every `uses:` is pinned to a 40-character SHA, and `scripts/check-action-pins.sh` passes.
- Permissions: each workflow has a top-level `permissions` block. None was loosened.

## Not verified

- No GitHub Actions run of the changed ci.yml exists yet.
- The setup-go cache contents were not checked. actions/setup-go could not be read through the
  GitHub connector from this session. The claim here is only that the key is go.sum.
- actionlint was not available. Expression and context validity was checked only by reading.
- Branch protection rules were not read. Required check names are unchanged, but the owner
  should confirm them.

## Owner decisions

1. **MacOS and Windows test matrix on main only (not done).** This would cut PR coverage. It
   would save about 16 job-minutes per PR commit (macOS 12.2 plus Windows 3.8). Reserved to the
   owner.
2. **Feature-branch pushes without a PR no longer run CI automatically.** Work on a branch
   without a PR runs only after `workflow_dispatch` or after a PR is opened. The alternative,
   a push job that skips when a PR is open, needs `pull-requests: read` and extra logic. Owner
   to choose.
3. **Path filters for docs-only PRs (not done).** Skipping required checks by path would leave
   them pending. Only consider this with a companion job that reports the same check names.

## Duplicates left in place (within one run)

- The fuzz job's seed-corpus step runs the same seeds as the Linux race `go test ./...` step.
  It costs about 11 s and is kept as RB-017 evidence.
- The contract job's `make contract` step is covered by the test job's `go test ./...`. It is
  kept because the same job runs the load smoke, which the test job skips.
- The e2e job uses `npm ci` without a cache. Caching it needs `actions/setup-node`, which is not
  on the allowlist in `scripts/check-action-pins.sh`. Left as a follow-up.
