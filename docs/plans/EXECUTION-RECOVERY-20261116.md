# Execution recovery checkpoint — 2026-11-16

**BLOCKED: the shared execution environment became unavailable. No feature is fully qualified. Integration remains HOLD.**

This checkpoint is saved through the GitHub connector after shell access failed with `No usable execution environment is available`. It records evidence and recovery locations; it does NOT contain or back up the uncommitted implementation source.

## Safely published
- All 15 upstream research reports, with exact pins/license evidence and adaptation plans.
- Initial upstream-backed PWA implementation at `344b87bdbc2f03d4b7bd43f090e027b490f2e907` on `future/fux-101-pwa-20261116`.
- Research catalog and fresh-agent workflow on this catalog branch.
- No feature merged, enabled, released or deployed. Root core files were not modified by this work.

## Team phases
22 distinct agents were used: 15 research agents, six fresh implementation agents, and one fresh PWA auditor. Remaining implementation/audit phases have not run. Do not describe the intended 45-phase workflow as completed.

## Workspace-only changes requiring recovery
Paths are under `/workspace/scratch/3ef42c6ef5d8/feature-worktrees/<slug>/future/<slug>/`. Workspace availability and durability cannot currently be verified.

| Module | Source written before disconnect | Last confirmed evidence | Unfinished recovery |
|---|---|---|---|
| pwa | Independent audit fixes and AUDIT.md; NOT committed/published | Go race/vet passed after fixes, nine Node tests, deterministic rebuild, npm audit zero known findings | Commit fixes and update validation/README; real browser/mobile/BFCache/accessibility and core adapter qualification remain open |
| qr | Encoder/scanner, aliases/requests, atomic claim interface, SQLite runner, HTTP/confirmation/PNG/CLI, notices | Real SQLite test-only cgo race suite passed, 75.3% library coverage; bundled jsQR decoded four generated cases | README/feature.json/VALIDATION stale; commit; fresh audit; default modernc/vet/fuzz/SCA/browser/core-ledger acceptance unresolved |
| discovery | Image sanitizer, SQLite metadata/taxonomy/media/retention, signed snapshots, API, no-JS gallery/editor/admin/forms, CLI | Initial test-only cgo race suite passed, 54.1%; expanded forms/CLI suite result unknown | Apply pending metadata Updated/touch patch; review filter bounds and expiry-aware media reads; rerun final suite/vet; docs; commit; fresh audit |
| notifications | SQLite outbox/preferences/encrypted targets, dispatch/retry/revoke, SMTP/Push, SSRF policy, management UI/API/CLI, tests | Non-SQL policy/verified-TLS SMTP/Web Push/SSRF tests passed | Final SQL suite unknown; last docs/target-verifier regression command may not have executed; recover, test, document, commit, fresh audit |
| messaging | Durable messages/blocks/reports/moderation/holds/read-hide positions, HTTP/API/signed cursors, optional WS, CLI, notices | Test-only cgo race suite passed, service 70.3%, CLI 72%; later retention/fuzz additions untested | Browser/vet outcome unknown; README/feature.json/VALIDATION stale; rerun latest source, commit, fresh audit |
| onboarding | Durable progress, admin copy, no-JS templates, isolated synthetic practice, default-off opt-in metrics, harness/notices | Pure domain/practice race tests passed; final SQL suite result unknown | Recover results/rerun, docs, commit, fresh audit |

The other nine modules (matching, favorites, receipts, feed, analytics, areas, recurring, governance, federation) have completed/published research but no new upstream-adaptation implementation phase.

## PWA audit findings not yet safely published
The auditor reported fixes for privacy lifecycle initialization depending on service-worker/BroadcastChannel availability, and unbounded precache response reads. The published implementation predates these fixes. Do not integrate it. Recover the audited worktree, inspect the diff and publish only after verifying the recorded changes.

## Test environment limitations and known working configuration
Default `modernc.org/sqlite v1.60.1` compilation repeatedly exceeded memory, including serialized low-memory attempts. Production driver was not replaced. Test-only `sqlite_cgo_test` uses `mattn/go-sqlite3 v1.14.53`, exact `93c7e0a2ec6762a38ab371755d49a2b3f4a12e67`, with MIT notices. It exercises real SQL but does not qualify the production driver.

Known successful QR command:
```sh
CGO_CFLAGS='-O0 -g0' GOGC=20 GOMAXPROCS=1 CGO_ENABLED=1 \
GOTOOLCHAIN=go1.26.11 /tmp/circula-toolchain/go/bin/go \
test -p 1 -race -tags=sqlite_cgo_test -cover ./...
```

Chromium failed at socket creation in worker runtime. Headless-shell download was invalid/truncated. Root cloud-browser tab creation also crashed. No real-browser acceptance was established.

## Resume procedure
1. Restore access to the SAME workspace if possible; inspect all six worktrees before changing/resetting anything. Do not overwrite uncommitted files.
2. Fetch remote branch heads; preserve concurrent remote changes. The published source and this checkpoint are authoritative only for what was actually published.
3. Inspect and checkpoint recovered source promptly on each independent feature branch, with WIP/unverified labels where tests are pending. Publish small checkpoints before lengthy tests to avoid another workspace-only interval.
4. Reconcile stale documentation and rerun tests against exact final files. Separate cgo-test evidence from production-driver qualification.
5. Continue with fresh agents for remaining feature implementation and each feature's independent audit. Fix audit findings and keep full requirement-level checklists.
6. Keep all features separate and HOLD. No automatic main merge or activation; one-feature integration remains deferred until both that feature and core are complete and selected by the owner.
