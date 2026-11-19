# Code and process audit notes (2026-11-18)

This is a targeted review, not a full maintainability audit. It records what came up while integrating and fixing
code in this pass. Items marked *fixed* changed in a merged or open PR. The others are recommendations.

| # | Area | Finding | State |
|---|---|---|---|
| C1 | Correctness | Two places built SQLite DSNs by string concatenation, without escaping the path (AUD4-D01) | Fixed: one `fileURI` helper (#81) |
| C2 | Correctness | `LoginStep` and `ChangePassword` checked the password with Argon2 outside a write unit, then wrote without re-checking it (AUD4-F06). The later review reproduced the same proof-binding defect in MFA enrolment/confirmation/disabling and account-recovery generation | Repaired in open #82, including the follow-up at `e127acc`. Independent source review and focused regressions pass. The full local combined race suite and pinned lint passed on candidate `17152dad`; current-head provider CI remains pending. See the chronology below and [TEST-EVIDENCE](TEST-EVIDENCE.md) |
| C3 | Correctness | Dispute deadlines were derived from the current policy in five places (AUD4-F03) | Partial, pending CI (#87): `disputePolicy` / `disputeDeadlines` is now the single source, but it reconstructs the in-force version from `effective_at`. The same-millisecond and clock-rollback cases remain until the version is stored on the dispute |
| C4 | Correctness | GOV-021 first draft: request lookups were bounded at 500, and a lifecycle test masked the closing-network guard | Fixed in #85 (`a5b258e`) after independent review |
| C5 | Resource limits | `go test -race ./internal/service` took 620–960 s in this container, longer than Go's default 10-minute timeout | CI sets its own limits: the Linux test recorder uses `-race -timeout 20m`; native macOS uses `go test -race -count=1 -p 2 -timeout=25m ./...`; Windows uses `go test -count=1 ./...`, without race instrumentation or an explicit Go timeout. `make test-race` passes no `-timeout`, so it uses the 10-minute default and can hit it on slower machines. Local runs here used `-timeout 40m`. Recommendation: add an explicit `-timeout` to `make test-race` |
| C6 | Process | Parallel branches each added a migration numbered `0019`; the loader and hygiene check reject gaps and duplicates | Resolved by stacking #86 on #85 and renumbering signals to `0020`. Recommendation: claim migration numbers in `docs/TODO.md` before starting parallel schema work |
| C7 | Process | A test that cites a spec ID needs `make traceability` before pushing. On #83 the GitHub lint job failed for this reason (job [134669246743](https://github.com/djshellshoxxx/circula/actions/runs/44265609316/job/134669246743), head `cb95bb1`). On #82 a local `traceability -check` found the same gap before any GitHub run finished; those #82 runs were cancelled when newer heads were pushed, not failed | Regenerated on both. Recommendation: run `make traceability-check hygiene` in every local pre-push check |
| C8 | Tooling | `/usr/local/bin/golangci-lint` in the cloud image is built with go1.25 and cannot lint this go1.26.11 module | Worked around with `go install …golangci-lint/v2@v2.15.0` into `/root/go/bin`. Recommendation: add that to the environment setup script |
| C9 | Portability | New Unix-only file checks (`O_NOFOLLOW`, link count, owner/mode) have `!unix` fallbacks | The runbook and security register state the Windows NTFS ACL boundary (#81, review G04) |
| C10 | Future modules | All `future/*` bases fork from `main@2eacf77` and no longer pass root CI (`cmd/circula/auditanchor.go: undefined: os`) | Owner-scheduled rebase before any module integration. Module-local tests still pass |
| C11 | API consistency | Session-only network routes now resolve the network before the cookie check, so outsiders get not-found, as API-017 requires | In #85 |
| C12 | Docs drift | `docs/ops/backup-restore.md` still described `*.db.enc.partial` staging after #72 | Fixed (#81) |

## Password-proof follow-up (C2)

The earlier review recorded the MFA and recovery paths as a potential verify/write race. The parallel
integration pass then reproduced it using authenticated request actors and actual self-service password
changes or administrator resets. `BeginMFAEnrolment` (including resumption), `ConfirmMFAEnrolment`,
`DisableMFA` and `GenerateRecoveryCodes` could accept a proof of the superseded password. Confirmation
and disabling still required the valid factor supplied by the regression.

PR #82 now carries the compatible repair at `e127acc9877edebeca818b75a5d48ce12511eca8`, tree
`84ea230a25cf30a56f4f43b1a401abea6dc1b7fb`: compare the verified hash inside the protected write before
secret disclosure, factor consumption or security-state mutation. All 12 deterministic baseline cases failed;
the same 12 pass after the repair, and the focused race selection passes all 21 top-level tests.
[Independent review](https://github.com/djshellshoxxx/circula/pull/82#pullrequestreview-5954151240)
accepted the source and regression evidence. The subsequent full local combined race suite passed at
2026-11-18T10:07:00.462200Z on candidate `17152dadad91d4e1c994ea807e7d6e2497513aa8`, tree
`1e47c9c30165ca27866887533586c90323e1630e`; pinned golangci-lint v2.15.0 also passed with zero issues on
that candidate. This qualifies the local combination of #82, #84 and #87. All three PRs remained unmerged
and all seven required provider jobs per PR were still queued at 10:07:24Z. See [TEST-EVIDENCE](TEST-EVIDENCE.md)
for the command, source pins and receipts. This additional finding is separate from the original 55-finding
ledger; reset authority and A3-F02 / DEC-014 are unchanged.

## Not reviewed in this pass

- dead code and unused abstractions;
- query plans beyond the new GOV-021 index check;
- goroutine and context-cancellation coverage beyond the export download watcher, which was checked and is sound;
- the full UI/API parity matrix.
