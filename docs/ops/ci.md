# Continuous integration

Workflow: `.github/workflows/ci.yml`. The tag-triggered `release.yml` gates are separate and unchanged.

## When each job runs

| Trigger | Jobs |
|---|---|
| Pull request (every push to the PR) | `lint` (vet, gofmt, golangci-lint, govulncheck, gitleaks, pins, hygiene, licences), `test (ubuntu-24.04)` with `-race`, `build`, `reproducible`, `fuzz`, `contract`, `e2e` |
| Push to `main` | All of the above, plus `test-cross`: `test (macos-15-intel)` with `-race`, `test (windows-2022)` |
| Nightly (03:17 UTC) and manual `workflow_dispatch` | Same as a push to `main` |

Rationale: measured timings were macOS race about 7.9 min, Ubuntu race about 6.8 min and e2e about 4.7 min. The
macOS and Windows runs add wall time and runner cost on every PR push; running them on `main` and nightly still
catches platform regressions within a day. A PR that touches platform-specific code can run them early: Actions,
ci, "Run workflow", pick the branch.

## No duplicate runs

`push` is limited to `main`, so a PR branch is not built twice for the same commit (previously `push` and
`pull_request` both fired). The concurrency group is keyed on the PR number; a newer push to a PR cancels the
superseded run. Runs for `main`, schedule and dispatch are never cancelled by a later one. A branch pushed without
an open PR gets no CI until a PR is opened or the workflow is dispatched by hand.

## Branch protection

Required checks for PRs should be the Linux jobs (for example `test (ubuntu-24.04)`). Do not require
`test (macos-15-intel)` or `test (windows-2022)`; they do not run on PRs.
