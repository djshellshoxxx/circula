# FNET-207 form-cache audit handoff

Branch: codex/future-recurring-master-20261116. Worktree: /workspace/scratch/4f14005536ba/work-recurring. Prior clean milestone: 053ffac736f3ba5d7aef04e17dd6f7ec0c5fac1f. Exact tested runtime/test commit: b4b8aec884c53a78a8586e0d94fad95e9416dc53. The documentation successor does not modify Go/module source.

## Completed correction

A host Session using PostFormValue could hide ParseForm's first error and allow malformed body fields to be discarded before a mutation. Red regression reproduced pause commit with HTTP 200. The handler now fully parses/validates the 64 KiB-capped form before Session and uses the retained command after authorization. Session receives populated PostForm for CSRF. Malformed/duplicate/unknown fields reject 400 before host; oversized form rejects 413. Valid native form and JSON compatibility pass. Only http.go and form_cache_test.go change executable source.

## Evidence

Go 1.26.11, GOGC=20/GOMAXPROCS=1/GOMEMLIMIT=450MiB and shared build cache: focused red reproduction confirmed; focused fixed suite PASS 0.013s; uncached full race PASS 2.611s, 78.6%; vet and module verification PASS. Raw logs and checksums: form-audit-evidence/. The coordinator independently reviewed the exact source correction and regressions and identified no issue. The previous query milestone review and nine-finding historical audit remain separately recorded.

Weighted criteria remain 84.25/100; this fix repairs an already counted requirement rather than adding percentage. No new browser/manual, scheduler integration, provider, operational erasure/restore or advisory acceptance is claimed. See COMPLETION.md and previous HANDOFF-MILESTONE-MASTER.md for unchanged gates and retained upstream pins/notices. No dependency change; robfig/cron remains unmodified v3.0.1; SQLite remains fixture-only.

## Resume

Fresh next team must verify this source and the final audit record before continuing. Preserve the current branch/worktree; no reset. Only future/recurring modifications are authorized. Keep disabled and integration HOLD. Never automatically post/accept trades. No publication, root dependencies, core routes/migrations, main merge, activation or deployment. Operational archive/erasure/retention, real browser/accessibility and future separately authorized host conformance remain gates.

From future/recurring, coordinate compiler slot then use GOGC=20 GOMAXPROCS=1 GOMEMLIMIT=450MiB GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache /workspace/scratch/0afb7942ff6b/toolchain/go/bin/go test -p 1 -race -cover -count=1 -timeout 90s ./... only for new changes or unresolved concerns.
