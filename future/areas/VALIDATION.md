# Standalone validation — 2026-11-16

Toolchain: Go 1.26.11 linux/amd64. No root build or ledger test was run by this isolated module work.

```
GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...
ok github.com/djshellshoxxx/circula/future/areas 2.402s coverage: 68.3% of statements
GOMAXPROCS=1 go vet ./...
PASS (exit 0)
go mod verify
all modules verified
```

Parser-focused bounded fuzz command avoids instrumenting the unrelated SQLite test driver:

```
go test -run '^$' -fuzz FuzzValidate -fuzztime=3s -parallel=1 areas.go service.go fuzz_test.go
PASS; 32,889 executions; 50 new interesting inputs
```

Dependency advisory command, Go vulnerability tool pinned to v1.1.4:

```
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
No vulnerabilities found.
```

Observed tests cover current member/admin/MFA policy, tenant/owner rejection, nullable area assignment, two same-area/one different/unset fixture, hidden/archived projection, no auto-restore, stale/concurrent revisions, exact idempotent retry/conflict, retry revocation, departure cleanup after retry-quota exhaustion, audit failure rollback, clean reopen/backup restore, scoped composite foreign keys, default-off sharing/admin-only revocation, strict request limits/unknown coordinates, HTML escaping, no-store/CSP, native form/CLI paths, encrypted cursor scope/tamper/expiry and unsupported labels (control/bidi/Unicode separators/invalid UTF-8).

Statement coverage does not establish correctness/security; current independent audit remains pending. Browser/keyboard/zoom/screen-reader checks were not run. No image is uploaded here, so EXIF/GPS removal is an explicit future media-adapter acceptance gate. Actual Circula auth/store transaction compatibility, root API/CLI registration, production backup/restore, replay-preserving retention, full ledger regression and release disposition remain unqualified. No integration, publication or activation occurred; HOLD remains.

## Fresh independent audit continuation

See AUDIT.md for findings and exact matrix. Final Go1.26.11 nested race/coverage suite passed70.6% in4.487s; vet and module verification pass. Added ignored-write/schema/FK/corrupt replay/restore-after-archive/cancellation/strict Unicode+duplicate JSON/route/content-type/privacy-cleanup and saturated sharing-quota regressions. Parser-only strict JSON fuzz passed11,378 executions. Frozen-source runtime advisory scan result is recorded below. Actual core/browser/manual gates remain unqualified and HOLD persists.

Frozen-source runtime govulncheckv1.1.4 rerun: **No vulnerabilities found.** No production/core adapter or manual browser qualification is inferred from this result.

## Master continuation — current HTTP correction

Runtime source `9d6f9e2844e7c7624c3c7c2c494030213ca3cf71`, tested checkout `0f8a02e51078911485bb99cb9dc8329c32709992` (documentation-only successor). Go1.26.11 linux/amd64 at `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`; shared cache `/workspace/scratch/0afb7942ff6b/go-build-cache`; `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1`.

- `go test -p 1 -run '^TestMaster' -count=1 -timeout 90s ./...`: PASS0.048s.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS4.746s,71.4% statements.
- `go vet -p 1 ./...`: PASS, exit0/no diagnostics.
- `go mod verify`: PASS, all modules verified.
- `git diff --check`: PASS.
- Notice inventory:25 module records,32 listed files; all present.

Raw fresh command output is retained under recovery-evidence/. No runtime source changed after these checks. Regressions exercise strict pre-auth queries, query-free mutation URLs, form callback session/member revocation/replacement/cancellation, and preserved single-empty cursor/listing compatibility. Coordinator independently reviewed the exact correction, recorded in SECURITY-AUDIT.md.

This run adds no dependency and does not claim to rerun the predecessor parser fuzz/advisory scan. Browser/keyboard/zoom/screen-reader, deployed principal/MFA/CSRF/store, replay-retention operations and full core regression remain unexecuted. The current84.75% criterion score is incomplete; all flags remain disabled and integration HOLD.

## Fresh form-cache audit continuation

Tested runtime source `232fd26e0a2f905931c221a80291a9bcd4d9fe0e`; Go1.26.11 with the same shared cache, GOGC20, GOMAXPROCS1 and GOMEMLIMIT450MiB limits.

- Before the correction, `TestMalformedFormRejectedBeforeHostParseCache` failed in all three cases: malformed discarded field, malformed duplicate area, and semicolon field. Each returned200, invoked the host once and persisted assignment revision1. Raw failure: recovery-evidence/form-cache-red.log.
- After moving capped strict form parsing before the host callback, focused malformed/native-control tests PASS0.010s. Invalid forms return400 before host invocation and leave no assignment.
- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS4.870s,71.7% statements.
- `go vet -p 1 ./...`: PASS, no diagnostics.
- `go mod verify`: PASS, all modules verified.

Raw output and exit codes are retained in recovery-evidence/form-cache-*.log with SHA256SUMS. Existing JSON/native form tests also passed in the full suite. No dependency/schema/core change or new browser/manual evidence. The84.75% matrix remains unchanged because this correction restores a previously counted strict-input guarantee rather than closing an external gate.
