# Analytics Core fuzz qualification continuation — 2026-11-20

## Exact source and scope

This follow-up starts at PR #108 source
`9eb06fc9b56905133aa9ec2d2786b8f80dbcf94c`, tree
`68657bdf61b1b14f81092de379e2b2945795e5a2`. Its isolated target is
`future/fnet-205-analytics-20261116` at
`0b2965101e1d5d1e3e6447e7c24197efc7f0cb9f`, also the merge base.
The existing candidate has three unique commits and 43 changed paths.

The complete path/commit inventory, exact component and test-file identities,
original feature manifest, and source-review scope are recorded in
[qualification-evidence-20261120/source-review.json](qualification-evidence-20261120/source-review.json).
The original manifest blob is
`2507b08b687d64c875480af1b993cfa97f9cfa1a` and remains unchanged:
`enabled` is false, integration is HOLD, and no other future module is a
required dependency. Its completion percentage and tested-source field remain
historical provenance, not a fresh readiness assessment.

## Observed failure and repair

[Core run 44555360653, job 135669862841](https://github.com/djshellshoxxx/circula/actions/runs/44555360653/job/135669862841)
ran the full fuzz seed step successfully, then reported `context deadline
exceeded` at the end of the ten-second amount campaign. The last progress line
reported 1,154,206 executions; the failure showed no parser assertion or saved
counterexample. The subsequent strict JSON campaign did not run in that job.

The repair uses the execution-budget approach already present on accepted
main `aa01563fcc891da60fefebb8335eca27855678e8`. The amount campaign requests
1,200,000 executions, exceeding the observed failed campaign's work count;
the JSON campaign requests 100,000 executions. Each has an independent
three-minute package deadline. Errors still fail the job normally.

Only the two existing fuzz step names and command invocations change in the
shared workflow. The complete target bodies, seeds and assertions, initial
all-target seed step, ten-minute job deadline, action pins, permissions,
triggers, platform conditions and all other jobs remain unchanged. The analytics
implementation, tests, dependencies, native module workflow and historical
recovery evidence are also unchanged. This is an isolated branch qualification
repair; it introduces no Core feature integration or module activation.

## Executed local checks

Both commands ran on Linux/amd64 with Go 1.26.11, one worker,
`GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=-p=1`, `GOGC=20` and
`GOMEMLIMIT=512MiB`. Each command had an additional 240-second outer cap,
including downloads, compilation and execution.

```sh
go test -run '^$' -fuzz '^FuzzParseAmount$' -fuzztime 1200000x -timeout 3m ./internal/domain
go test -run '^$' -fuzz '^FuzzStrictUnmarshal$' -fuzztime 100000x -timeout 3m ./internal/api
```

| Campaign | Observed executions | Package time | Total wall time | Result |
| --- | ---: | ---: | ---: | --- |
| Amount parser | 1,200,000 | 32.397 s | 88.786 s | PASS, exit 0 |
| Strict JSON | 100,000 | 23.189 s | 219.005 s | PASS, exit 0 |

The observed baseline phases completed all 21 and 14 entries respectively.
These are two successful fuzz campaigns, with no failing campaign; they are
not a fresh full unit/race-suite count. The exact commands, environment,
result metadata, raw logs and their SHA-256 digests are in
[qualification-evidence-20261120/qualification.json](qualification-evidence-20261120/qualification.json).
No race instrumentation was requested for these fuzz commands.

YAML semantic comparison confirms that only the two stated existing steps
changed. Workflow pin/permission checks, repository hygiene, whitespace checks,
and the complete directory scan with the repository's Gitleaks v8.28.0 all
passed. An initial scan of the new receipt interpreted a Git object ID as a
credential; the receipt now records object IDs in explicitly named tree fields.
The final
scanner report contains zero findings. Git comparison verifies that runtime,
tests, original feature manifest and dependency files retain their source
blobs. No scanner rule or ignore annotation was added.

## Review and remaining qualification

The continuation reviewed the complete unique path/commit inventory, all three
analytics production diffs and four added test diffs, the dedicated native
workflow, the six inherited accepted Core repair paths, the complete shared
workflow and both unchanged Core fuzz bodies. The review identified no further
blocking defect within this qualification-repair scope. The source receipt
states its limits: it does not repeat the original archive-bundle verification
or claim a provider approval.

Before this follow-up, Core run 44555360653 passed six jobs, failed the amount
fuzz job and skipped its conditional native Core matrix. Dedicated module run
44555359311 passed all three native platform jobs. Those are dated results for
the preceding source, not qualification of a new published head. Fresh current-
head applicable Core and dedicated module checks are required. The older
workflow's conditional native Core skips must remain explicitly reported.
No full unit/race suite, native platform run, browser workflow or vulnerability
scan was rerun locally by this narrowly scoped continuation.

The original feature holds remain: immutable-release storage, query-budget and
correction policy, production performance budgets, browser/accessibility
acceptance, the authenticated historical Core adapter, full ledger integration
and explicit feature selection. All original manifests and development paths
remain independent. No release or deployment is included.
