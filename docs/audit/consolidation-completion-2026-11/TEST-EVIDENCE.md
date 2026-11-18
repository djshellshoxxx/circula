# Test evidence (2026-11-18)

Toolchain: go1.26.11, golangci-lint v2.15.0. The linter was built locally with go1.26.11 because the image's own
binary refuses this module; CI pins the same version. Local runs were on a Linux container. CI results are linked
by run ID. Local results and CI results are recorded separately, and a local run that was cut short is not counted
as a pass.

## Merged into `main`

| Tree / revision | Local | CI (exact head) |
|---|---|---|
| main + #71 + #72 (tree `bddf803`, now `a62071d`) | `go build`, `go vet`, `go test -count=1 ./...` all pass; `-race` passes for `cmd/circula` (250 s) and `internal/store` (109 s); golangci-lint reports 0 issues | #71: [44256822964](https://github.com/djshellshoxxx/circula/actions/runs/44256822964), 9/9. #72: [44256964805](https://github.com/djshellshoxxx/circula/actions/runs/44256964805), 9/9 (including native Windows and macOS) |
| #77 | test-only diff, reviewed | [44256413287](https://github.com/djshellshoxxx/circula/actions/runs/44256413287), 7/7 |
| main + #79 (tree `f41365f`, now `cdc8112`) | `go test` for `internal/web/...`, `internal/api/...` and `cmd/circula` passes; traceability up to date | [44260832164](https://github.com/djshellshoxxx/circula/actions/runs/44260832164), 7/7 (the browser/axe suite passed) |
| main + #81 (tree `1e2923e`, now `b439807`) | build and vet pass; `cmd/circula`, `internal/store` and `internal/web` tests pass; traceability and hygiene pass. Branch head: full `go test ./...`; `-race` for cmd and store; `GOOS=windows` and `GOOS=darwin` vet | [44264463917](https://github.com/djshellshoxxx/circula/actions/runs/44264463917), 9/9 (including native Windows and macOS) |

## Open PRs (author or coordinator runs; CI decides)

| PR | Local evidence |
|---|---|
| #82 AUD4-F06/F04 | New tests fail before the fix and pass after it. `go test ./...` passes. `-race ./internal/service` passes with `-timeout 40m` (777 s); with the default 10-minute timeout the race run hit the limit. Traceability and hygiene pass after regeneration |
| #85 GOV-021 | Merged with main at the time (`b20c118`): golangci-lint reports 0 issues; `go test ./...` passes; `-race` passes for service (958 s), api (603 s), web (490 s), store (321 s) and cmd (367 s). Review fixes at `e11d292`: the agent ran the same suite and it passed |
| #86 SEC-026 | Stacked tree `2e2e6f0`: golangci-lint reports 0 issues; `go test ./...` passes; `-race` passes for service (628 s), api (311 s), web (232 s) and store (124 s). At `4d349c4`: focused SEC026/GOV021/migration/contract/catalogue tests, traceability and hygiene pass |
| #87 AUD4-F03 | Author run: lint, `go test ./...`, race for service/store/api/web, traceability and hygiene pass. After merging main: `go test` passes for service, web, store, repository and api |
| #80 FNET-204 (future) | Module: `go test`, `-race` and vet pass; coverage 98.8%; `FuzzOpenCursor` ran 60 s (1,361,653 executions, 178 new inputs, no failures). Root CI fails on the stale base (`auditanchor.go: undefined: os`), not on this change |

## Fail-before evidence for fixes

| Finding | Regression tests | Fail-before result |
|---|---|---|
| AUD4-D01 | `TestSQLiteOpensExactPathWithURIMetacharacters`, `TestBackupDirectoryWithURIMetacharacters`, `TestFileURIEscaping` | The store test failed; the backup subtests `bk?x` and `bk%41x` failed. `bk#x` did not reproduce through `backupTo`; the store-level `#` case covers it |
| AUD4-D02 | `TestSecureRemoveLeavesHardLinkedDataIntact`, `TestSecureRemoveAllDoesNotFollowDirectorySymlink`, `TestBackupStagingRequiresPrivateOrStickyParent`, `TestLeasedWorkCleanupRefusesSubstitutedDirectory`, `TestBackupStagingRefusesForeignOwnedParent` (root only) | All failed. `TestZeroFileRefusesReplacedName` tests a new helper, so it has no fail-before run |
| AUD4-F06/F04 | `TestLoginStraddlingPasswordChangeGetsNoSession`, `TestChangePasswordReplacesOnlyTheVerifiedHash`, `TestChangePasswordGuessesAreThrottled` | All failed, including the subtests. The parallel reviewer reproduced this independently through a Go overlay |
| AUD4-F03 | `TestDisputeTimelinesFixedAtOpen` and three related tests | All failed with the observed behaviour |
| AUD4-F02 (draft) | `TestAdminCannotResetReviewerPassword` | Failed on `main` |

## Not run

- Browser e2e locally: Chromium cannot open its socket in this container, so CI is the only browser evidence.
- Manual screen-reader checks.
- Long QA-001 property runs (not re-run in this pass).
- The manual stress workflows.
- Native macOS and Windows locally: CI only.
- Real OS-level fault injection beyond the existing tests.
