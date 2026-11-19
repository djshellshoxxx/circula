# FUX-105 milestone F — callback and form-boundary audit

Overall partial; 90% weighted evidence unchanged. Feature disabled; integration HOLD; no remote writes.

- Branch: `work/requalify-messaging-20261116`.
- Worktree: `/workspace/scratch/3ef42c6ef5d8/feature-worktrees/messaging`.
- Recovered milestone: `ea06857109b145d986d7b6f3ac69f7cf72efb97e`.
- Test-only red checkpoint: `a7837aeee91d0ec2d53d517a43b4db34bb58a651`.
- Callback/form correction: `b81473dd929b541a6d937cdf71c3a157fa32227f`.
- Exact final tested source: `0b1861f50ef1fcecb65eaf3e4227a647ce2dcd58`.
- This handoff is a documentation-only successor; resolve `git rev-parse HEAD` for the final documentation checkpoint.

## Completed and verified

Closed cached malformed-form acceptance, stale session identity across Token/VerifyCSRF, search disclosure after token callback revocation, and query parsing before a byte/syntax guard. Raw red/green evidence demonstrates the failures and repairs. No store/schema/dependency change; only future/messaging files. Existing pinned SQLite dependencies and notices are unchanged; exact upstream revisions/licenses remain in UPSTREAM-REUSE-MATRIX.md and THIRD_PARTY_NOTICES.md.

Final focused tests passed 0.263 s; full fresh race library 76.3% / 5.259 s and CLI 78.4% / 1.280 s; vet no diagnostics; module verification all verified. SECURITY-AUDIT.md and VALIDATION.md document scope, exact commands, prior observations and limits. The coordinator independently reviewed both callback/form and final query corrections and found no further issue in those bounded changes. No browser or current advisory/performance acceptance claimed.

## Remaining criteria and next owner

Real browser/no-JS/WCAG/manual screen-reader acceptance and broad retained-history/concurrency performance qualification remain. Missing-substring search scans under the existing writer reservation; a result cap does not bound work. Actual host session/membership/retention/audit/core integration remains separately unauthorized. The 90% score is not production approval.

Next fresh performance team should define and execute retained-history/contended-client workloads and in-flight cancellation, without adding a private-message index by assumption. Preserve current source and historical handoffs; no reset from stale remote branches. No push, main merge, routes, root migrations/dependencies, activation, deployment or release.

Resume from this worktree's future/messaging using Go 1.26.11 and shared cache at `/workspace/scratch/0afb7942ff6b/go-build-cache`, GOGC=20 / GOMAXPROCS=1 / GOMEMLIMIT=450MiB and `go test -p 1 -race -count=1 ./...`. Coordinate the compiler slot first.
