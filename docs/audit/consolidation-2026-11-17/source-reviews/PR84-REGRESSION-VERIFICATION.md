# PR84 narrow independent regression verification

Date: 2026-11-18. Reviewer: core_recovery (C). Caveman investigate-first / verify-and-stop workflow. Scope was limited to the three supplied regression tests, run one package at a time with `GOMAXPROCS=2`, `GOGC=50`, `-p=1`, `-count=1`, and a three-minute test timeout. No broad suite, new test/probe code, tracked source edit or remote mutation was performed.

## Exact inputs and result

| Input | Exact value |
|---|---|
| PR84 head | `7f98b3db8d032fc107599a3da4a6e1747c3e00aa` |
| PR84 tree | `c7077653183021a006821c71926be3a089da502b` |
| Baseline main | `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a` |
| Baseline tree | `f41365f8ee4f42f4cbd33fb4e1a78cad85caedd1` |
| Go executable | `/workspace/scratch/b4e4b2242853/toolchain/go/bin/go` |
| Source snapshots | `/workspace/scratch/9ceb45b04668/audit-c-pr84/` |

**All requested fail-before / pass-after checks completed.** These are narrow regression results, not an aggregate-suite or release approval.

| Check | Before | PR84 fixed control |
|---|---|---|
| `TestProblemCodesHaveCatalogueRows` | Literal main snapshot plus only the exact supplied PR84 test: all four expected missing-row failures; exit 1; package 0.018 s | Pass; exit 0; package 0.014 s |
| `TestPruneKeepsFutureDatedBackupsOutOfCounts` | Behavior-restoring mutation: all four subtests fail as expected | All four subtests pass |
| `TestJobRetentionDoesNotCountFutureDatedBackup` | Same mutation: expected failure because the future-dated file consumes retention capacity and the newest older backup is removed | Pass; new backup manifest also verified by the supplied test |
| Combined two CLI test functions | Exit 1; package 0.677 s | Exit 0; package 0.539 s |

No Go process was terminated during these checks. An initial CLI setup attempt encountered a full cache filesystem before tests could execute; the environment correction and that preserved failure are documented below.

## Snapshot and mutation integrity

`pr84-regression-setup.py` used `git archive` to create untracked source snapshots. The fixed snapshot is the exact PR84 tree. The API baseline snapshot is the exact baseline main tree with only `internal/api/problem_catalogue_test.go` copied verbatim from PR84. Thus the API fail-before ran against the actual baseline implementation and catalogue, with the existing supplied regression added.

The old `pruneBackups` implementation has no `pruneBackupsAt` seam, so the retention fail-before is explicitly a **behavior-restoring mutation**, not literal old-tree execution. It starts from the exact PR84 snapshot, keeps the seam and every supplied test unchanged, and removes only:

```go
if e.at.After(now) {
    keep[e.path] = true
    continue
}
```

Every entry then participates in the daily/monthly selection again, restoring the prior counting behavior. The exact four-line removal is saved in `PR84-retention-prior-algorithm.diff`. Snapshot SHA-256 checksums and exact Git blob IDs are in `PR84-regression-snapshots.json`.

| File | PR84 Git blob |
|---|---|
| `internal/api/problem_catalogue_test.go` | `4eff2343bd7293754ef4a2ea00f286a081a96fde` |
| `cmd/circula/backup_test.go` | `4b7aa65aed259800226a512e8a68b7bb67c5ef7b` |
| `cmd/circula/jobs_regress_test.go` | `5eae6f3f9d3ab66f11612c6206f1574658403ca2` |
| `cmd/circula/snapshot.go` | `fcec927773bf4f965744c9e709ef0e45ddcd2b4a` |
| `docs/specs/07-api-contract.md` | `4f6b1a626efbd72d72a083e1e987ce26ca063474` |
| Baseline catalogue | `3dcf4f2da6af1ffac015f207511c3ce8ad5a27e6` |

The pruning subtests cover a backup 50 years ahead with default 30-daily/12-monthly counts, one hour ahead in the same month, a backup exactly at now, and several future-dated files including a one-nanosecond boundary. The fixed control preserves future files and their manifest/anchor sidecars without counting them, while pruning eligible backups according to the configured limits.

## Exact test commands

API before, working directory `/workspace/scratch/9ceb45b04668/audit-c-pr84/api-baseline`:

```sh
GOMAXPROCS=2 GOGC=50 /workspace/scratch/b4e4b2242853/toolchain/go/bin/go test -p=1 -count=1 -timeout=3m -run '^TestProblemCodesHaveCatalogueRows$' -v ./internal/api
```

API after used the identical command in `/workspace/scratch/9ceb45b04668/audit-c-pr84/fixed`. These two runs preceded cache relocation and used the existing default cache.

Retention before, working directory `/workspace/scratch/9ceb45b04668/audit-c-pr84/retention-prior-algorithm`:

```sh
GOMAXPROCS=2 GOGC=50 GOCACHE=/workspace/scratch/9ceb45b04668/audit-c-pr84/go-build-cache /workspace/scratch/b4e4b2242853/toolchain/go/bin/go test -p=1 -count=1 -timeout=3m -run '^(TestPruneKeepsFutureDatedBackupsOutOfCounts|TestJobRetentionDoesNotCountFutureDatedBackup)$' -v ./cmd/circula
```

Retention after used the identical command in `/workspace/scratch/9ceb45b04668/audit-c-pr84/fixed`.

Logs: `PR84-api-before.log`, `PR84-api-after.log`, `PR84-retention-before-mutation.log`, `PR84-retention-after.log`.

## A2-08 disposition correction

The original `docs/audit/2026-10-28/A2-circula-adapters.md`, present on baseline main with blob `f8126e266f17f936c744f694b85687e82db40518`, explicitly includes the missing catalogue rows in **A2-08**, not a separate newly discovered finding:

- Line 133 identifies `network-frozen` (423), `approval-required` (403), `password-change-required` (403), and `terms-acceptance-required` (409) as absent from section 7.2.
- Line 135 explicitly calls for adding those four codes, alongside the OpenAPI response fixes and contract cases.
- The exact original excerpt is preserved in `PR84-A2-08-original-evidence.txt`.

The independent baseline regression produces exactly those four failures. Therefore, on main `cdc8112`, A2-08 must be **partially fixed — catalogue documentation outstanding**, even if the OpenAPI response-status portion is already fixed. PR84 supplies and verifies the remaining catalogue rows. Its branch-local “fixed here” description is not evidence that canonical main already contains them. Update the canonical disposition after the verified change actually lands; do not call this missing portion a false positive or a newly expanded audit scope.

## Cache setup failure, correction and handoff

The first CLI attempt, before relocation, failed at setup with `write /home/agent/.cache/go-build/...: no space left on device`; no retention test executed. Its complete output remains in `PR84-retention-cache-setup-failure.log`. This was not a failed regression assertion or a process-termination signal.

Read-only disk inspection showed `/home/agent` on a 4.9 GB tmpfs with zero available space; the generated Go build cache occupied 3.3 GB. The workspace disk had 45 GB available. The existing cache was copied without alteration to:

`/workspace/scratch/9ceb45b04668/audit-c-pr84/go-build-cache`

`cp -a` completed with exit 0. Both retention runs then used that explicit GOCACHE and completed normally. After the checks, root explicitly authorized clearing only the old generated build cache. The following completed with exit 0:

```sh
GOMAXPROCS=2 GOGC=50 GOCACHE=/home/agent/.cache/go-build /workspace/scratch/b4e4b2242853/toolchain/go/bin/go clean -cache
```

No module cache, source, worktree, evidence or branch was removed. The workspace cache remains intact at 3.3 GB; the old generated cache is 264 KB. `/home/agent` now has 3.3 GB available, and the workspace disk has 42 GB available. Diagnostics are preserved in `PR84-disk-before-cache-clean.txt`, `PR84-disk-after-cache-clean.txt`, and `PR84-old-cache-clean.log`.

This concrete full-filesystem observation explains this attempt's setup failure only. It does **not** establish the cause of earlier `signal: killed` runs. No Go process remains from this workstream. Root has been handed the explicit workspace cache path for its outstanding qualification, without overlapping test jobs.
