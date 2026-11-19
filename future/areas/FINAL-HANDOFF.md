# FNET-206 final checkpoint handoff

This is a partial standalone milestone, not final feature qualification. Integration HOLD; disabled.

- Branch: `codex/future-areas-master-20261116`.
- Worktree: `/workspace/scratch/4f14005536ba/work-areas`.
- Recovered predecessor: `a5f28203a5aa389f81f4e9aad1ab97d612bef74d`.
- Current runtime correction: `232fd26e0a2f905931c221a80291a9bcd4d9fe0e`.
- Previous master checkpoint: `8120ee12afbecab5c65893d87b0e79a9e7e94ee5`; this successor adds the independently investigated form-cache correction.

## Completed implementation

Route-specific strict queries are bounded before parser/authentication callbacks. Duplicate/unknown/malformed fields fail; POST query fields are rejected; a single empty cursor/listing remains valid. Existing native mutation form actions contain no query string. Capped strict URL-encoded body parsing now precedes host Auth so a CSRF helper cannot cache partial malformed input and hide its original error. Form token generation precedes fresh same-identity authentication and current transactional private reads. Regressions cover malformed/duplicate/oversized input, forbidden POST query fields, session/member revocation, session replacement, callback cancellation and compatibility controls.

The coordinator reviewed the correction independently; see SECURITY-AUDIT.md. Current form-cache focused red/green tests passed0.010s after reproducing three mutations from malformed forms; full fresh race suite passed4.870s with71.7% statements; vet and module verification passed. Full commands and raw evidence are in VALIDATION.md and recovery-evidence/. No runtime source changed after those tests. The standard weighted matrix is84.75%, still incomplete.

## Reuse and architecture

No new dependency or upstream code migration. Existing test-only modernc.org/sqlite v1.60.1 and25-module graph remain;32 notice files exist and are inventoried. Production remains standard-library-only. Exact dependency versions/checksums/notices are in go.mod/go.sum, third_party/dependencies.json and THIRD_PARTY_NOTICES.md. Native named-area scope intentionally omits optional maps/geometry/coordinates.

## Remaining acceptance

Real browser keyboard/reflow/zoom, screen-reader and platform geolocation-denial evidence remain unavailable. Production replay-preserving retention/restore and operator acceptance need the actual environment. Concrete core principal/MFA/store adapters, deployment/session/CSRF policy and full ledger regression belong to separately authorized integration. An image-capable host must independently qualify EXIF/GPS sanitization; this module accepts no images. Current percentage and uncounted criteria are in COMPLETION.md.

## Resume

From future/areas, use `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go` with `GOCACHE=/workspace/scratch/0afb7942ff6b/go-build-cache GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1`. Run `go test -p 1 -race -cover -count=1 -timeout 90s ./...`, `go vet -p 1 ./...`, `go mod verify` in the coordinator's serialized compiler slot. Do not reset the predecessor worktree. A fresh team should verify this handoff and existing acceptance evidence before the next standalone change.

No push, remote write, merge, activation, core/root edit, route/migration registration, deployment or release occurred.
