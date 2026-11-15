# Beta readiness report

## 2026-11-14 follow-up (PR #35, branch `claude/cool-ramanujan-n2peag`)

Code tested: the branch head that carries this section (later commits change only documentation). Toolchain go1.26.11
linux/amd64 (fetched by `GOTOOLCHAIN=auto` from `go.mod`), golangci-lint v2.15.0 built with go1.26.11, Playwright
1.63.0 with the pre-installed Chromium, Node 22. Machine: 4 vCPUs.

| Check | Command | Result |
|---|---|---|
| Format | `gofmt -l .` | empty |
| Vet | `go vet ./...` | clean |
| Lint | `golangci-lint run ./...` (v2.15.0, including the new depguard rules) | 0 issues |
| Hygiene | `scripts/check-hygiene.sh` | ok |
| Unit and integration | `go test -count=1 ./...` | all 12 packages ok |
| Race | `go test -race -count=1 -timeout=20m ./...` | all ok; `internal/service` 460 s |
| e2e | `make e2e` equivalent (`npx playwright test`) | 578 passed (9.5 min) |
| Fuzz | `FuzzAPICommandBody`, `FuzzOpenCursor`, 15 s each | no findings |
| Deploy smoke | `scripts/smoke-deploy.sh` | `smoke-deploy: ok` |
| Dependency scan | govulncheck in CI (lint job) | passed on PR #35 |

Notes:
- A first race run, started while e2e shared the same 4 CPUs, hit Go's 10-minute per-package limit in
  `internal/service`. On its own the package takes about 8 minutes under `-race`, so the Linux race step in `ci.yml` now
  sets `-timeout=20m` (as the macOS step already did), and QA-007 runs 8 rounds under `-race` (100 without).
- Backup file names have one-second resolution. A restore started within the same second as a pre-migration backup is
  refused ("backup destination ... already exists") and changes nothing. Found while writing the QA-026 test; low.
- Not run here: the QA-001 long run at release size (100,000 sequences, DEC-023), a tagged release, the manual
  screen-reader run, an independent security review.

Classification is unchanged: **beta candidate**, pending the owner gates in `HANDOFF.md`.

Date: 2026-11-11. Branch `claude/circula-beta-completion-1npjw6`.
Code tested: commit `3cd1442` (`3cd1442` and later commits change only documentation).
Toolchain: go1.26.10 linux/amd64 (from `go.mod`), Chromium 1243 (Playwright 1.63.0), Node 22.

## Classification

**3. Beta candidate.** Not yet beta qualified.

The core is complete and tested: authentication, mutual-credit transfers, accurate balances and history, and the
ledger's guarantees under failures, retries, concurrency and restarts. Every engineering acceptance criterion below
passed in this environment. It is not *qualified* because of four gaps:

- CI has not yet run on this head. It runs when the PR is opened.
- Dependency scanning (govulncheck) could not run here.
- The container image was not built.
- The owner gates are open: licence, the DEC-* decisions, the independent security review SEC-033 and the manual
  screen-reader run.

These are not production-ready claims.

## Acceptance criteria

| Criterion | Result | How verified | Level |
|---|---|---|---|
| Application builds | Pass | `go build ./...`; `make build`; static `CGO_ENABLED=0` build in `smoke-deploy.sh` | build |
| Clean installation works | Pass | `scripts/smoke-deploy.sh`: fresh data dir, systemd-equivalent settings, `/healthz` and `/readyz` → `smoke-deploy: ok` | manual/scripted |
| Existing databases migrate | Pass | `TestUpgradeFromEverySchemaVersion` (each of v1–v15 → v16); golden v1 fixture with 10,000 transactions → v16; migration 0016 backfill and blocking tests | integration |
| Users can authenticate | Pass | service, HTTP and e2e sign-in, MFA and recovery tests | unit, integration, e2e |
| Accounts can be created or provisioned | Pass | `circula admin create-operator`, invitation/approval/open join, e2e registration | integration, e2e |
| Authorized transfers work | Pass | `TestTradeScenarioBalances`, acceptance scenario, e2e no-script proposal → acceptance | unit, e2e |
| Unauthorized transfers fail | Pass | `TestAuthorizationMatrix`, API 403/404 tests, refusals leave the ledger unchanged | unit, integration |
| Ledger balances reconcile | Pass | `VerifyLedger` after each scenario; sum of balances = 0; tamper detection | unit, CLI |
| Duplicate submissions do not double-post | Pass | idempotency tests; replay of reversal and adjustment posts nothing | unit |
| Concurrent transfers preserve credit limits | Pass | `TestParallelAcceptsNeverBreachCreditLimit` under `-race` | unit (race) |
| Transaction history is accurate | Pass | history checks across reopen; acceptance steps | unit, CLI |
| Restart persistence | Pass | `TestReopenPreservesLedger`; acceptance steps 16–17 | integration |
| Essential HTML works without JavaScript | Pass | e2e `zz-noscript.spec.js` (6 flows) and `noscript-focus.spec.js` | e2e |
| Security-critical tests pass | Pass | see [../security/security-review.md](../security/security-review.md) | all |
| Backup and restore exercised | Pass | online `circula backup` of a live e2e instance → `scripts/restore-drill.sh` (restore ok, signature valid, ledger ok) → `circula restore` and `verify-ledger` on a separate copy | manual |
| Deployment instructions verified | Partial | `smoke-deploy.sh` follows `deploy/circula.service`; Docker/compose not built here | scripted |
| CI passes on the release candidate | **Pending** | CI runs when the PR is opened | CI |

## Commands and results

All runs were in this environment on the commits named. Counts include subtests (`go test -json` events).

| Check | Command | Commit | Result |
|---|---|---|---|
| Format | `gofmt -l .` | `3cd1442` | no files |
| Vet | `go vet ./...` | `3cd1442` | pass |
| Unit and integration tests, race detector | `go test -count=1 -race ./...` | `3cd1442` | **1227 pass, 0 fail, 1 skip** (`TestLoadConcurrentTrades`, opt-in with `CIRCULA_LOAD=1`) |
| ↳ per package | | | cmd/circula 178 · api 234 · contract 1 (85 operations, 164 operation/status pairs) · domain 149 · histbal 2 · httpx 7 · service 513 (+1 skip) · store 35 · web 101 · scripts/sbom 7 |
| Lint | `golangci-lint run ./...` (v2.15.0, the CI pin, built with go1.26.10) | `9217825` | 0 issues |
| End-to-end (Playwright, axe WCAG 2.2 AA light and dark, contrast, target size, responsive, keyboard, no-script) | `make e2e` | `7569285` (later commits change only test code and docs) | **578 pass, 0 fail** (7.5 min) |
| Fuzz (boundary slice) | `go test -run '^$' -fuzz '^FuzzAPICommandBody$' -fuzztime 15s ./internal/api`, same for `FuzzOpenCursor` | slice on `86b8a4a` | pass, no findings |
| Mutation check, DOM-017 | old `OpenDispute` logic restored temporarily | `e127db8` | `TestDisputeDeadlineFixedAtCompletion` **fails** as intended; passes with the fix |
| Deployment smoke | `scripts/smoke-deploy.sh` | `9217825` | `smoke-deploy: ok` |
| Backup and restore drill | see below | `9217825` binary | pass |
| Release packaging (offline, one target) | `VERSION=v0.0.0-test OUT=… TARGETS=linux/amd64 scripts/package-release.sh` | release-chain slice | pass: archive, SBOM and SHA256SUMS verified, scan clean |
| Dependency vulnerabilities | `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` | | **not run**: `vuln.go.dev` refused by this environment's proxy (HTTP 403). CI runs it |
| Container | `docker build -f deploy/Dockerfile .` | | **not run**: Docker is not available here |

Level of evidence by area:

- **Unit-tested:** accounting rules, limits, validation, authorization matrix, state transitions, idempotency, faults.
- **Integration-tested** (real SQLite file, HTTP handlers, CLI): migrations (each of v1–v15 and a golden fixture
  with 10,000 transactions), API contract, sessions and cookies, restart persistence, the 20-step CLI acceptance
  scenario.
- **End-to-end** (real binary in Chromium): registration, sign-in, MFA, recovery, trades, disputes,
  administration, accessibility, no-script.
- **Manually verified here:** online backup → restore drill → independent restore and `verify-ledger`; smoke
  deployment.
- **Not verified:** CI on this head, govulncheck, the container image, a tagged release run, a manual screen-reader
  run, an independent security review.

## Restore drill record

Source: an online backup of the e2e instance while it was running (1 network, 8 members, 17 transactions: 6
completed, 5 disputed, 4 declined, 1 proposed, 1 reversed; 24 entries; 89 audit events). Schema v16.

```
backup:  circula backup <dir>        sha256 45c1d33a…  schema v16  ledger ok  (warning printed: backup not encrypted, no encryption key set for the drill)
drill:   scripts/restore-drill.sh     {"restore":"ok","ledger":"ok"}  backup signature: valid (backup-key)
restore: circula restore + verify-ledger on a separate path: network demo ok; sum(balances)=0; 0 completed trades without a dispute deadline
```

The drill backup was signed but not encrypted. Production must set `backup-encryption-key`
(see [../ops/backup-restore.md](../ops/backup-restore.md)).

## Known issues and remaining blockers

| # | Item | Severity | Owner |
|---|---|---|---|
| 1 | CI on this head (Linux race gate, macOS 15, Windows, e2e, govulncheck) | Release blocker | CI |
| 2 | govulncheck not run locally (vuln.go.dev refused by this environment's proxy) | Release blocker until CI passes | CI |
| 3 | Licence; DEC-005, 011, 013, 014, 021, 022, 023, 025; A3-F02 break-glass scope | Release blocker | owner |
| 4 | Independent security review (SEC-033); manual screen-reader run (UX-056, QA-035) | Release blocker for production; recommended for beta | owner |
| 5 | Export spool slice not integrated (BR-06 findings) | Not a blocker; exports remain in memory with the existing caps | eng |
| 6 | BR-04: admin full-export artefacts survive member erasure for up to 24 h | Low | eng |
| 7 | web-ux, arch-guards, sec-ops-docs slices not integrated (conflicts, unreviewed) | Not a blocker | eng |
| 8 | Docker image and compose file not built or run here | Before a container-based beta | eng |
| 9 | Re-review window reads the policy in force at decision time (not covered by DOM-017) | Low; owner to confirm intent | owner |
