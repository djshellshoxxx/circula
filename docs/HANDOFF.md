# Circula: handoff and completion plan

Written 2026-10-22 at the head of `claude/circula-completeness` (draft PR #7 into `claude/circula-build`).
Audience: the next engineer or agent swarm. Read this first, then `docs/audit/COMPLETENESS.md`.

## 1. State in one paragraph

Circula is a Go 1.26 (static, `CGO_ENABLED=0`) single-binary service for community mutual-credit networks: SQLite
(embedded, one writer), server-rendered HTML UI plus a versioned JSON API, hash-chained append-only audit, per-network
roles. Two swarms built it. Swarm 1 (branch `claude/circula-build`) did the core, the security features (MFA, account
recovery, dual control, break-glass, erasure, backup encryption, cookie hardening, login throttle, setup secret) and
release plumbing. Swarm 2 (this branch, `claude/circula-completeness`) did product completeness, ops/architecture and
quality work, then governance and hardening. At the last measured point the Go test suite (`-race`), golangci-lint
(0 issues) and `make e2e` (102 passed) were green and GitHub CI had no failing check on `9a86ea6`.
The PR is a **draft and has never been reviewed by a human**.

Do not read this as "MVP finished". See section 5 for what is not done and section 6 for how to finish it.

## 2. Branches, PR, and rules of engagement

| Branch | Purpose |
|---|---|
| `claude/circula-build` | integration branch, includes swarm 1 and its security features (do not push here directly) |
| `claude/circula-completeness` | swarm 2 work; PR #7 targets `claude/circula-build` |
| `claude/circula-security-features` | swarm 1 feature branch (already merged into build) |
| `main` | never touched by swarm 2 |
| `claude/cc-*` | short-lived agent branches in worktrees; all merged, safe to delete |

Constraints that were in force and should stay unless the owner lifts them:
- No releases, tags or publishing. `release.yml`, signing, SBOM and provenance belong to the security swarm's area.
- Security swarm's code was treated as off-limits except additive tests: `internal/service/auth.go`, MFA, account recovery,
  dual control, break-glass, erasure, backup encryption, cookie hardening, login throttle.
- Never use Fable models. Haiku for easy work, Sonnet for hard work, Opus only if truly needed.
- Gate before every commit: `gofmt -l .` empty, `go vet ./...`, `go test -race ./...`, `golangci-lint run` 0 issues
  (v2.15.0; gosec false positives need `//nolint:gosec // reason`). Run `make e2e` for anything touching templates, routes or setup.
- `GOTOOLCHAIN=go1.26.10`. Lint binary: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`.

## 3. Repository map

```
cmd/circula/        CLI + server wiring, jobs, config, backups, TLS, maintenance mode, audit-anchor, verify-export, suspend, break-glass
internal/domain/    types, errors, money (int64 minor units), policy, state machines, FoldName (unicode search)
internal/service/   all business logic; one *Service per process; adapters never pass roles. See docs/dev/SERVICE.md
internal/store/     SQLite repos, migrations (embedded, must be contiguous), inspect.go (schema/pragma helpers), golden fixture test
internal/api/       JSON API (/api/v1), idempotency layer, problem+json, ETag, scoped tokens
internal/web/       HTML handlers + templates + static; CSRF, sessions, rate limits, time zone display
internal/contract/  contract test: ~80 operations validated against docs/api/openapi.yaml (kin-openapi)
internal/httpx, internal/testutil
e2e/                Playwright + axe-core + keyboard-only tests against the real binary (Node 22+)
deploy/             hardened systemd unit, sysusers/tmpfiles, Dockerfile, compose
scripts/            release-gate.sh, restore-drill*.sh, smoke-deploy.sh, check-*.sh, licenses.sh, read-export.py
docs/               specs/ (requirements), api/openapi.yaml, dev/SERVICE.md, ops/, guides/, qa/, security/, platform/, audit/
```

Key docs: `docs/dev/SERVICE.md` (service contract, error codes, idempotency, freeze/observer), `docs/specs/*` (requirements; status
lines are partly stale), `docs/TRACEABILITY.md`, `docs/audit/COMPLETENESS.md` (requirement-by-requirement status),
`docs/security/RAINBOW-BOOKS-SPEC.md` and `GOV-STANDARDS-MAPPING.md` (security research, see caveats), `docs/ops/*`, `docs/guides/*`.

Migrations (0001..0014, contiguous; the loader fails if a number is missing):
0001 init, 0002 hardening, 0003 mfa, 0004 recovery, 0005 dualcontrol, 0006 breakglass, 0007 erasure (swarm 1);
0008 notify/reports/dispute responses, 0009 overview/announcements/person prefs, 0010 announcement supersede,
0011 terms + policy notice, 0012 suspension, 0013 person activity, 0014 closure/challenges/deadline notices (swarm 2).
**Next free number is 0015.** Parallel agents must be assigned numbers up front and the lead renumbers on merge.

## 4. What exists (shipped in PR #7 on top of swarm 1)

Features: dispute response (FR-076) with a response-period gate (policy, default 7 days, reviewer waiver with reason), one
re-review (FR-078), reports (FR-024), in-app notifications and inbox (FR-038), limit-warning notice and hourly job (UX-012),
fixed-text notices to affected members for admin actions (SEC-012), 90-day notification retention, admin overview (FR-067),
admin reversal UI with confirm step, reviewer review step and paged queues (WEB-02), `GET /listings/{id}` with ETag/304,
member display time zones, announcements with supersede, unicode-safe directory and listing search, terms versioning and
acceptance (GOV-009), policy-change notice period and member policy log (GOV-003/002), member departure form (FR-014),
network suspension by admins and by the platform operator (GOV-023/024), scheduled network closure with 30-day notice,
final checksummed export and 90-day erasure (GOV-025), deadlines/overdue reminders for disputes (GOV-014), challenges API
(GOV-016), admin-required read-only state (GOV-020), inactivity flagging (GOV-012), member and reviewer audit views
(FR-082), join-request rate limit (FR-012), noindex (FR-025).

Ops/architecture: SQL moved out of `cmd` into `internal/store`; maintenance mode (reads stay up, writes 503 with
Retry-After); per-network freeze on ledger failure instead of whole-instance refusal, with admin banner and member notice;
pre-migration backup verification; golden 10,000-transaction migration fixture with timing; posting/replay metrics;
idempotency key length 255; verify job records runs without audit flooding; `circula verify-export`; restore audit event;
restore-drill scripts; 10 s shutdown grace; log retention setting and `prune-logs`; audit anchor (`audit-anchor write|verify`,
optional HMAC, written beside backups); native TLS (min 1.2, SIGHUP reload); `secure_delete`; optional audit mirror file;
8 fuzz targets (one found a real decoder bug).

Quality: contract test, axe + keyboard e2e (102 tests, includes security pages, frozen 423 page, maintenance), ARIA proxy
tests, concurrency tests (100 disjoint trades), gated load test (measured 250 to 360 trades/s on a 4 vCPU VM, one run each,
no HTTP; see `docs/qa/LOAD-TEST.md`), release-gate script, hygiene checks (PII scan, migration numbering), gitleaks step,
licence manifest, reproducible-build check, deploy smoke, Scorecard workflow, user and admin guides.

## 5. What is NOT done

### 5.1 Hard gaps (code or process missing)

| Item | State | Notes |
|---|---|---|
| GOV-006 policy-change objections | not built | L; needs objection record/vote on a pending change (pending-change mechanism from GOV-003 exists) |
| GOV-016 challenges web UI | API only | add pages for challenge, decide, list |
| GOV-001 policy version stored per transaction | not built | migration needed; needed for fairness of notice-period changes |
| GOV-010 self-export offer, GOV-011 removal challenge path | partial | GOV-011 should reuse the challenges feature |
| GOV-021 lost-admin recovery over break-glass | partial | security-swarm area |
| GOV-018 write-off flow, GOV-022 import | not built / deferred | import is deferred in the spec |
| SEC-026 velocity/reciprocal signals | not built | needs thresholds (open question 09 #10) |
| UX-047 string externalisation | not built | translation scope is a product decision; L |
| API-010 `admin` token scope | 501-class gap | M |
| FR-054 balance-at API+UI, FR-064 default-flag API route | partial | S each |
| FR-095 streaming export, API-065 async export jobs | partial | exports are buffered in memory; per-person quota missing |
| ARC-002/004/010/018/020 | partial | store contract suite, fuzz coverage, depguard (could not be shown to fire), reference reader tests |
| RB-001/RB-014 permission table, separate ops listener | skipped | not small |
| Frozen state persistence | in memory only | re-derived at start-up and hourly; no DB flag |
| person_activity erasure | gap | table has no FK to persons, so erasure does not clear it (security-swarm code) |
| Closure erasure of challenges/reports free text | check | verify `EraseClosedNetworks` covers every free-text table added in 0008..0014 |
| Platform policy advance-notice test (GOV-026) | not written | docs/platform/*.md are drafts pending counsel |
| Production hosting items | none shipped | no managed deployment; RPO/RTO unconfirmed |

### 5.2 Evidence gaps (built but not proven)

- No human screen-reader pass (UX-056/QA-035). `docs/qa/SCREEN-READER.md` is a manual script plus an automated ARIA proxy.
- No independent security review (SEC-033). Everything was reviewed by the same agent family. Do not describe it as audited.
- `govulncheck` has never run in the build sandbox (proxy blocks `vuln.go.dev`); it runs in CI only.
- Hardened systemd unit never started under real sandboxing; Dockerfile never built (no daemon in the sandbox).
  `docs/ops/hardening.md` has the commands to run on a real host.
- Load test: one run per row on a shared VM, service called directly (no HTTP/TLS/reads). No soak test, no restore under load.
- Restore drill record in `docs/qa/evidence/` is from an empty migrated database (procedure only). Run one on real data.
- DEC-023 quality thresholds are proposed, unconfirmed. The release-gate script fails them by design.
- `docs/security/RAINBOW-BOOKS-SPEC.md` and `GOV-STANDARDS-MAPPING.md`: sources were mostly unreachable from the sandbox.
  Content is from search snippets and model knowledge, marked UNVERIFIED. Re-verify against the official documents before relying on it.
- Spec status lines, `docs/TRACEABILITY.md` and the counts in `docs/audit/COMPLETENESS.md` lag the code. The summary table
  (268 built / 103 partial / 21 not built / 8 deviations of 400 P0+P1) was computed before the last waves; rows were
  updated by hand but the totals were not recomputed. **Task 1 below fixes that.**
- Windows and macOS CI exposed bugs the Linux-only sandbox missed (zip handle leak, CRLF, case-insensitive file systems).
  Always wait for the three-OS CI run, not just the local gate.

### 5.3 Security-swarm-owned items (not touched by swarm 2)

Argon2 benchmark on minimum hardware (SEC-001/DEC-006), breached-password list (SEC-002), role-specific session timeouts
(SEC-003), per-network and invitation limits (SEC-006), session key rotation (SEC-027/OPS-010), release notes file (OPS-026),
artefact signing and SBOM and provenance (OPS-035, SEC-029, RB-024/025), verify admin MFA enforcement and dual-control
self-approval refusal (RB-007/RB-022).

### 5.4 Decisions only the owner can make

| Decision | Options / default | Blocks |
|---|---|---|
| Confirm DEC-016 RPO/RTO (hosted 1 h / 4 h, self-hosted 24 h / 4 h) and record in spec 13 | accept or change | OPS-034 |
| DEC-023 quality thresholds (`docs/qa/THRESHOLDS.md`) | accept or change | QA-040, release gate |
| OQ-FR-2 response period (default 7 days) and DEC-012 decision target (+14 d) need counsel review | accept or change | FR-076 closure |
| Is the re-review window / limit-warning percent / retention (90 d) policy or constant | now policy fields except notification retention | none |
| SEC-026 thresholds and signals | decide or drop for MVP | SEC-026 |
| UX-047 translation scope | decide languages | UX-047 |
| Platform policy documents and SECURITY.md contacts (marked `OWNER:`) | counsel + owner | GOV-026, disclosure process |
| Approve the proposed fix to `TestSecurityHeadersAndRequestID` (see 7.2) | edit the assertion | flaky CI |
| Repo settings: branch protection, private vulnerability reporting, public repo for Scorecard publishing, pin Docker base images by digest | owner action | Scorecard score |

## 6. Plan to finish

Estimates are agent-days with the swarm approach below; "wall" assumes 4 to 6 parallel agents.

### Phase 0: truth and hygiene (1 day)
1. **Recompute the audit.** One Sonnet agent re-reads the code and tests, regenerates `docs/audit/COMPLETENESS.md`
   (fix the 361 vs 400 count mismatch noted in its first version), refreshes spec status lines and `docs/TRACEABILITY.md`.
   Done when counts match row statuses and every "built" row cites a test or file.
2. Delete merged `claude/cc-*` branches and stale worktrees under `.claude/worktrees`.
3. Fix the request-ID flake once the owner approves (7.2).
4. Run `scripts/release-gate.sh` and commit the evidence file; list failing/MANUAL checks.

### Phase 1: remaining P0 gaps, free of the security swarm (3 to 5 days wall)
Work packages (each a worktree branch, one migration number assigned in advance, next free = 0015):
- **A. Governance finish:** GOV-001 (policy version on each transaction), GOV-006 objections, GOV-016 web pages, GOV-010 self-export,
  GOV-011 removal challenge via challenges, GOV-018 write-off via adjustment with dual control, API-010 admin scope.
- **B. Architecture debts:** ARC-002 (CLI reads through service), ARC-004 (store contract test suite), ARC-010 (fuzz coverage + trust-boundary
  checklist), ARC-018 (prove depguard fires), ARC-020 reference reader tests, FR-054, FR-064 routes, FR-095 streaming export with per-person
  quota (replace `MemSink` buffering), SEC-032 default-limit cap.
- **C. Tests and evidence:** QA-001/004/007/010/011 (property/fault/contention), QA-013/015/017/019/022/023 (notice, terms, deadline tests),
  QA-024 restore under load, QA-026 forced migration failure, QA-028 backup under load, QA-030 reconciliation script, UX-033..040 and UX-057
  (320px, reflow, target size, orientation, dark contrast) as Playwright specs in CI, QA-034 scripting-disabled e2e.
- **D. Erasure and privacy follow-ups (coordinate with the security swarm):** `person_activity` and every table added in 0008..0014 must be
  cleared or pseudonymised by member erasure and by closure erasure; add a canary test that scans the database file and logs after erasure (SEC-020/RB-003).
Done when: gate green on all three OS, contract test covers every new operation, guides updated, COMPLETENESS rows updated.

### Phase 2: P1 and polish (2 to 3 days wall)
OPS-029 optional mail/webhook channel (or document as out of scope), DAT-024/032/045, OPS-004/015/030, UX-017/049, SEC-014 personal-data inventory,
SEC-030 incident drill record, OPS-003 smoke each release target, GOV-027 link support boundary, GOV-026 advance-notice test.

### Phase 3: verification that agents cannot do (owner + humans, parallel with phases 1/2)
1. Independent security review or penetration test (SEC-033); give the reviewers `docs/security/*`, `docs/dev/SERVICE.md` and the audit.
2. Human screen-reader pass using `docs/qa/SCREEN-READER.md` (NVDA, VoiceOver at minimum) and record results.
3. Run `docs/ops/hardening.md` on a real systemd host (seccomp, MemoryDenyWriteExecute) and build/run the container image.
4. Restore drill on a realistic dataset; record; confirm RPO/RTO.
5. Longer soak and load test with HTTP, TLS and mixed reads on target hardware.
6. Counsel review of platform policies, terms, privacy notice and dispute process.
7. Decide DEC-023, OQ-FR-2, SEC-026, UX-047 (section 5.4).

### Phase 4: release readiness (security swarm + owner)
Signing, SBOM, provenance, release notes (OPS-026/035, SEC-029), session key rotation, branch protection, vulnerability disclosure contacts,
tag only after `scripts/release-gate.sh` is green with human items recorded. Merge PR #7 after review (see 7.1 for order).

## 7. Working notes and lessons

### 7.1 Merging and PR
- PR #7 is a draft into `claude/circula-build`. Review in logical chunks (commit history is merge-heavy). Merge strategy is the owner's call.
  Expect doc conflicts (spec status lines, COMPLETENESS counts, `docs/ops/README.md`).
- Rebases were avoided; every branch was merged. Keep it that way for anything already pushed.

### 7.2 Known flaky or open test issue
`TestSecurityHeadersAndRequestID` (`internal/api/authn_test.go` near line 46) asserts the replacement request id does not contain the
substring "bad". The id is random hex, so it can contain "bad" by chance (rare, seen once under load). Proposed fix: assert that the id
is not the hostile input and contains no space or control character. An automated permission check blocked the edit because it touches a
security assertion; the owner should approve or make the change.

### 7.3 Process lessons from the swarm (apply to the next one)
- **Assign migration numbers up front** and tell agents. The loader requires contiguous versions, so each agent invented untracked
  placeholder migrations locally; the lead renumbered on merge. Better: reserve a block per agent and fill gaps with real files.
- **Parallel agents merge cleanly textually but break semantically.** Every wave produced integration failures that only CI caught: e2e seed
  vs the response-period gate, contract test closing a network with an open proposal, policy-form test missing new fields, duplicate test
  helpers, two scripts with the same name. After each merge run the full gate and e2e before declaring green.
- **CI is three-OS.** Linux-only local runs missed the Windows zip-handle leak and CRLF issue and the macOS case-insensitive time zone bug.
- **Shared scratchpad collides.** Agents truncated each other's logs. Tell agents to use unique log names.
- **Never run `pkill` with a pattern that matches your own command line**; it killed the lead's shell twice.
- Subagents repeat their final report many times. Tell them explicitly to stop after merge; one needed a stop message.
- Agents that did web research could mostly not reach primary sources (egress policy). Treat research docs as drafts.
- Background `go test -race ./...` takes 4 to 10 minutes; use `-timeout 40m` and poll a log file.

### 7.4 Useful commands
```
export GOTOOLCHAIN=go1.26.10
gofmt -l . ; go vet ./... ; go test -race -timeout 40m ./... ; ~/go/bin/golangci-lint run
make e2e              # builds the binary, runs Playwright/axe/keyboard (needs Node 22+, Chromium)
make contract         # OpenAPI contract test
CIRCULA_LOAD=1 make load
scripts/release-gate.sh   # writes an evidence file under docs/qa/evidence
scripts/check-hygiene.sh ; scripts/check-action-pins.sh ; scripts/check-reproducible.sh
circula serve -check-config ; circula verify-ledger ; circula audit-anchor write <path>
```
Configuration reference: `docs/ops/configuration.md` (env `CIRCULA_*`, flags, config file). Operations: `docs/ops/runbook.md`.

## 8. Suggested first hour for the next engineer
1. Read sections 1, 5 and 6 above, then `docs/dev/SERVICE.md` sections 1 and 2.
2. Check out `claude/circula-completeness`, run the gate and `make e2e`, confirm green on your machine.
3. Open PR #7, skim the file list by directory, and decide the review/merge order.
4. Start Phase 0 task 1 (re-audit); its output is the true backlog.
5. Take the owner decisions in 5.4 to the product owner this week; several packages wait on them.
