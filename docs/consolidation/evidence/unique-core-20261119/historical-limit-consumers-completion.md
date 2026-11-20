# Historical credit-limit coverage: consumer completion

## Source delivered

Consumer commit: `db27af0bd48cc04b5f51bcf5793e52e4cf1e16d2`

Tree: `6dbfdfdec70fdcefc92be0a6d3251278a73fd0e5`

Worktree: `/workspace/scratch/4d30fa819181/circula-historical-consumers`

The commit contains 16 consumer, contract, documentation and test paths. Its
core dependency is the single cherry-pick `cbcbaeb` of
`48bb4c69b43b392cdbaf3db81dbc186120d3a9ba`; parent already integrated that core
as `2d3a163`. Apply only the consumer commit to the parent's integrated tree.
The worktree was clean before and after qualification.

## Resulting behavior

API consistency checks return optional `historical_limit_coverage`, containing
the status and all four counts. Creation and retained cached reads use the
same projection. The OpenAPI schema documents the optional object without
changing existing required fields or administrator authorization.

CLI network checks show coverage independently of success/failure. Backup
output shows the aggregated coverage. Aggregation counts only financial
networks and excludes platform audit. Any unknown financial coverage keeps
the aggregate unknown; an empty/platform-only selection also has no financial
coverage claim. Counts are checked for negative values and addition overflow.
The existing pass/fail return values and backup decisions continue to depend
on invariant results.

The administrator check page and retained overview share one localized
coverage template. They distinguish a complete run, partial evidence, an old
saved run with no coverage metadata, and no saved run. Complete coverage does
not mean the ledger passed; legacy-only partial coverage does not mean it
failed. The service overview now carries the optional saved-run pointer.

Backup `LedgerSummary` appends an optional pointer with `omitempty`. Old
manifests retain field omission and exact canonical ordering. The version 1
signature payload remains unchanged, and version 2 naturally authenticates
the new object in the existing whole-manifest payload. SQLite snapshots
retain the witness table. Export schema 1 and closure checksum code/bytes
remain unchanged; the backup documentation states the portable export limit.

No authentication, posting decisions, freeze policy, response authorization,
runtime metrics, export versions or closure versions are changed by this
consumer commit.

## Focused qualification

Executed on the exact clean consumer commit with Go 1.26.11, `-race -p 1
-count=1`, one CPU, a 256 MiB memory target and workspace-disk build temporary
files. All tracked source hashes, the commit and the clean state were checked
before and after the command.

**23 top-level tests and 22 subcases passed.** No failure, skip, race warning or
missing selected test was recorded. Command wall time: 76.524 seconds.

| Full package compiled; selected tests executed | Package elapsed |
|---|---:|
| `cmd/circula` | 13.735 s |
| `internal/api` | 2.256 s |
| `internal/web` | 6.109 s |
| `internal/web/messages` | 1.059 s |
| `internal/contract` | 6.937 s |

The selection covers nine new top-level tests plus fourteen existing
compatibility/contract tests. It checks aggregation with unknown and partial
history, platform exclusion, result independence, real service-to-snapshot
propagation, old version 1 signed vectors, old version 2 canonical bytes both
before and after platform coverage existed, plain/encrypted signatures,
authentication of every new coverage field, API creation and retained read,
member refusal, localized counts, a retained old NULL-coverage fixture, a new
admin check, and the running API against OpenAPI. JSON/YAML structure checks,
gofmt and `git diff --check` also passed.

The old saved-run HTML fixture uses the normal repository append operation
with absent optional metadata. It does not claim to independently exercise
all migration paths; those belong to the core worker and final integrated
suite.

## Retained evidence and limits

- Raw output: `analysis/historical-consumers-focused.jsonl`
- Raw SHA-256: `ea43a1bc5c977b17179c419d60f4eb3baff739183f7ad6d30001d64726a7354c`
- Detailed receipt: `analysis/historical-consumers-focused-receipt.json`
- Runner: `analysis/run_historical_consumer_tests.py`
- Compact commit/receipt manifest: `analysis/historical_limit_consumers_verification.json`

The runner refuses to overwrite retained logs and records the full command,
environment, selected test names, source hashes, pass/fail events, and elapsed
times. The compile slot was released to the report-context worker after the
command exited. This is focused author-run qualification, not an independent
review of the consumer implementation or a full integrated suite pass.

The separate core source review identified an unusable-policy decoder issue;
the core author confirmed it and is preparing the parent-approved repair.
That finding and its follow-up are tracked in
`analysis/historical_limit_core_independent_review.md`. It does not change the
consumer type contract. Parent integration, the repaired core and final
frozen-tree qualification remain separately tracked.
