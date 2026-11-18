# Code and process audit notes (2026-11-18)

This is a targeted review, not a full maintainability audit. It records what came up while integrating and fixing
code in this pass. Items marked *fixed* changed in a merged or open PR. The others are recommendations.

| # | Area | Finding | State |
|---|---|---|---|
| C1 | Correctness | Two places built SQLite DSNs by string concatenation, without escaping the path (AUD4-D01) | Fixed: one `fileURI` helper (#81) |
| C2 | Correctness | `LoginStep` and `ChangePassword` checked the password with Argon2 outside a write unit, then wrote without re-checking it (AUD4-F06) | Fixed, pending CI (#82). Related paths are not covered by #82: `BeginMFAEnrolment`, `ConfirmMFAEnrolment`, `DisableMFA` and `GenerateRecoveryCodes` check the current password, then write in separate units (`PutMFA`, recovery-code replacement, session revocation) without carrying the verified hash forward. This is a potential verify/write race, under separate review by the parallel integration pass, which owns `internal/service/mfa.go` and `GenerateRecoveryCodes` in `recovery.go` for it. No exploit has been reproduced |
| C3 | Correctness | Dispute deadlines were derived from the current policy in five places (AUD4-F03) | Partial, pending CI (#87): `disputePolicy` / `disputeDeadlines` is now the single source, but it reconstructs the in-force version from `effective_at`. The same-millisecond and clock-rollback cases remain until the version is stored on the dispute |
| C4 | Correctness | GOV-021 first draft: request lookups were bounded at 500, and a lifecycle test masked the closing-network guard | Fixed in #85 (`a5b258e`) after independent review |
| C5 | Resource limits | `go test -race ./internal/service` took 620–960 s in this container, longer than Go's default 10-minute timeout | CI sets its own limits: the Linux test recorder uses `-timeout 20m`, native macOS and Windows use `-timeout=25m -p 2`. `make test-race` passes no `-timeout`, so it uses the 10-minute default and can hit it on slower machines. Local runs here used `-timeout 40m`. Recommendation: add an explicit `-timeout` to `make test-race` |
| C6 | Process | Parallel branches each added a migration numbered `0019`; the loader and hygiene check reject gaps and duplicates | Resolved by stacking #86 on #85 and renumbering signals to `0020`. Recommendation: claim migration numbers in `docs/TODO.md` before starting parallel schema work |
| C7 | Process | A test that cites a spec ID needs `make traceability` before pushing. On #83 the GitHub lint job failed for this reason (job [134669246743](https://github.com/djshellshoxxx/circula/actions/runs/44265609316/job/134669246743), head `cb95bb1`). On #82 a local `traceability -check` found the same gap before any GitHub run finished; those #82 runs were cancelled when newer heads were pushed, not failed | Regenerated on both. Recommendation: run `make traceability-check hygiene` in every local pre-push check |
| C8 | Tooling | `/usr/local/bin/golangci-lint` in the cloud image is built with go1.25 and cannot lint this go1.26.11 module | Worked around with `go install …golangci-lint/v2@v2.15.0` into `/root/go/bin`. Recommendation: add that to the environment setup script |
| C9 | Portability | New Unix-only file checks (`O_NOFOLLOW`, link count, owner/mode) have `!unix` fallbacks | The runbook and security register state the Windows NTFS ACL boundary (#81, review G04) |
| C10 | Future modules | All `future/*` bases fork from `main@2eacf77` and no longer pass root CI (`cmd/circula/auditanchor.go: undefined: os`) | Owner-scheduled rebase before any module integration. Module-local tests still pass |
| C11 | API consistency | Session-only network routes now resolve the network before the cookie check, so outsiders get not-found, as API-017 requires | In #85 |
| C12 | Docs drift | `docs/ops/backup-restore.md` still described `*.db.enc.partial` staging after #72 | Fixed (#81) |

Not reviewed in this pass:

- dead code and unused abstractions;
- query plans beyond the new GOV-021 index check;
- goroutine and context-cancellation coverage beyond the export download watcher, which was checked and is sound;
- the full UI/API parity matrix.
