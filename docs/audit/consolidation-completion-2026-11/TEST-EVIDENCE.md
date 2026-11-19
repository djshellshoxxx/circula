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
| #87 AUD4-F03 (partial) | Author run: lint, `go test ./...`, race for service/store/api/web, traceability and hygiene pass. After merging main: `go test` passes for service, web, store, repository and api |
| #80 FNET-204 (future) | Module: `go test`, `-race` and vet pass; coverage 98.8%; `FuzzOpenCursor` ran 60 s (1,361,653 executions, 178 new inputs, no failures). Root CI fails on the stale base (`auditanchor.go: undefined: os`), not on this change |

### Subsequent PR #82 password-proof follow-up

The parallel integration worker reproduced the MFA/account-recovery proof-binding defect against
`6785863252d2910553b1f4a26396f14b4405d6fa` and supplied
`TestPasswordProtectedChangesRejectSupersededPassword`. It uses a logged-in actor and commits actual
self-service password changes or administrator resets during password verification. Both change paths
cover fresh/resumed enrolment, confirmation, disabling with TOTP or a recovery code, and generating
account-recovery codes.

| Exact source or control | Executed result |
|---|---|
| Baseline production plus the new regression | All 12 cases fail on stale-proof acceptance; service `0.189s` |
| Repaired source, published as `e127acc9877edebeca818b75a5d48ce12511eca8` (tree `84ea230a25cf30a56f4f43b1a401abea6dc1b7fb`) | All 12 cases pass; service `0.179s` |
| Same repaired source, focused race selection | 21 top-level tests pass: the new regression plus 20 existing MFA/password/recovery tests; service `14.099s`, no race diagnostic |

The separate reviewer checked source order, actual change/reset setup, state preservation, factor reuse,
committed file bytes and all three retained log hashes. The reviewer did not rerun Go; the executions above
belong to the implementation worker. [Independent review](https://github.com/djshellshoxxx/circula/pull/82#pullrequestreview-5954151240)
accepted the fix. The committed [repair note](https://github.com/djshellshoxxx/circula/blob/e127acc9877edebeca818b75a5d48ce12511eca8/docs/audit/PASSWORD-PROOF-BINDING-2026-11-18.md)
records its scope and commands. These focused executions preceded the separate completed full combined
race run below. PR #82 remains unmerged and current-head provider CI remains pending.

### Completed local combined qualification (2026-11-18T10:07Z)

The integration coordinator ran the full suite once on local candidate
`17152dadad91d4e1c994ea807e7d6e2497513aa8`, tree `1e47c9c30165ca27866887533586c90323e1630e`.
The result qualifies that exact candidate; `main` was still `b4398074ae4ca72559235ca50242cfcb7a934eec`.

| Included change | Exact source |
|---|---|
| #82, including the MFA/account-recovery repair | Local authoring commit `d3fcc6831f81eea3344a03113e27532a07996bcf`; published `e127acc9877edebeca818b75a5d48ce12511eca8` has the identical tree `84ea230a25cf30a56f4f43b1a401abea6dc1b7fb` |
| #84, retention/logging and remaining audit repairs | `eb4bb08b15d41295aa2c4b9f93040124ce4d66e9` |
| #87, partial prospective dispute timelines | `679a3016d77c3cf34e218ecb296f1d42cc0969c8` |

Command: `go test -race -count=1 -p 1 -timeout=25m ./...`, with Go 1.26.11 Linux/amd64,
`GOTOOLCHAIN=local`, `GOMAXPROCS=2`, `GOGC=50`, `GOMEMLIMIT=512MiB`, `CGO_ENABLED=1` and an explicit
workspace build cache. It ran from 09:49:16.576031Z to 10:07:00.462200Z, exited 0 after 1063.886s, and
reported all 14 test-bearing packages passing plus two helper packages with no test files. No race diagnostic
was reported. This is one completed full invocation, distinct from earlier interrupted or multi-invocation
qualification attempts. Existing privileged-test skips and the manual/native limitations below remain.

The same candidate passed pinned golangci-lint v2.15.0 with `--max-issues-per-linter=0`,
`--max-same-issues=0` and `--timeout=10m`: zero issues, exit 0, 40.920s. Both receipts record matching
before/after HEAD and tree values and a clean post-run worktree. Producer records are
`aggregate-final-password-proof-race.json`/`.log` and `final-password-proof-lint.json`/`.log`;
the race log SHA-256 is `4dc27ca06074b9bf74cbb9d5ea34f15c2929bd519112561bf72ee09cbfd0b4dd`.

Provider CI is separate: at 2026-11-18T10:07:24.719Z, #82/#84/#87 were all open and unmerged. Each had
zero of seven required jobs successful, all seven queued, no assigned runner and no executed step:
[#82 run 44279484396](https://github.com/djshellshoxxx/circula/actions/runs/44279484396),
[#84 run 44272452596](https://github.com/djshellshoxxx/circula/actions/runs/44272452596),
[#87 run 44275665191](https://github.com/djshellshoxxx/circula/actions/runs/44275665191).
The local pass does not claim provider CI, deployment/release qualification or a merge.

## Fail-before evidence for fixes

| Finding | Regression tests | Fail-before result |
|---|---|---|
| AUD4-D01 | `TestSQLiteOpensExactPathWithURIMetacharacters`, `TestBackupDirectoryWithURIMetacharacters`, `TestFileURIEscaping` | The store test failed; the backup subtests `bk?x` and `bk%41x` failed. `bk#x` did not reproduce through `backupTo`; the store-level `#` case covers it |
| AUD4-D02 | `TestSecureRemoveLeavesHardLinkedDataIntact`, `TestSecureRemoveAllDoesNotFollowDirectorySymlink`, `TestBackupStagingRequiresPrivateOrStickyParent`, `TestLeasedWorkCleanupRefusesSubstitutedDirectory`, `TestBackupStagingRefusesForeignOwnedParent` (root only) | All failed. The root-only test's fail-before and pass-after runs were made by the coordinator, as root (UID 0) in this session's Linux container, with no pinned log. The hosted #81 Linux job and the parallel reviewer ran as non-root, so they skipped it; it has no independent or CI execution. `TestZeroFileRefusesReplacedName` tests a new helper, so it has no fail-before run |
| AUD4-F06/F04 | `TestLoginStraddlingPasswordChangeGetsNoSession`, `TestChangePasswordReplacesOnlyTheVerifiedHash`, `TestChangePasswordGuessesAreThrottled` | All failed, including the subtests. The parallel reviewer reproduced this independently through a Go overlay |
| AUD4-F03 (partial) | `TestDisputeTimelinesFixedAtOpen` and three related tests | All failed with the observed behaviour. Test-only same-millisecond and clock-rollback variants still fail on the fix (independent review); see the residual |
| AUD4-F02 (draft) | `TestAdminCannotResetReviewerPassword` | Failed on `main` |

## Not run

- Browser e2e locally: Chromium cannot open its socket in this container, so CI is the only browser evidence.
- Manual screen-reader checks.
- Long QA-001 property runs (not re-run in this pass).
- The manual stress workflows.
- Native macOS and Windows locally: CI only.
- Real OS-level fault injection beyond the existing tests.
