# Circula handoff

## 2026-11-16: core completion and consolidation (current)

Branch `codex/circula-core-completion-20261116` continues `claude/happy-babbage-dcumwz@dc97f7d`,
which already joined the beta follow-up (#33) with main. Documentation upkeep (#55) is merged,
with the newer export, logging and QA evidence preserved. The 15 `future/*` modules remain
isolated; this consolidation does not enable them.

Implemented and regression-tested in this pass:
- Export expiry actively releases disk space. Admitted downloads pin resources until completion;
  erasure aborts downloads and invalidates every affected network export, including member archives.
  Running exports cannot commit snapshots taken before an erasure; shutdown cleans up spools.
- Cached API retries recheck current scopes, membership, roles and MFA without repeating commands.
  Network erasure invalidates cached responses that could name counterparties.
- Migration 0017 preserves insertion order for immutable trade/dispute histories across equal or
  backward timestamps and VACUUM. Guards also reject REPLACE; invalid schema versions fail safely.
- Nanosecond backup names support rapid successive snapshots and retain old filename compatibility.
  Future-dated backups cannot suppress the scheduled cadence.
- The standalone export reader validates reconciliation for CSV/JSON and all exported scopes.
- Logout failures are reported without falsely clearing a still-valid session. Unicode login
  throttles clear correctly. Erasure removes counterpart notification content without broad text matches.
- Closure final-export failures are retried before erasure; completion/audit events are not duplicated.
- QA-019 now covers service, JSON API and HTML forms with a controlled clock; browser coverage is separate.
- Listing descriptions are labelled optional. The race target enables CGO. CI checks traceability
  freshness and runs the standalone reference-reader tests.

Additional audit completion:
- Frozen networks no longer starve proposal/adjustment expiry in other networks; batches stay bounded.
- Policy logs and challenges have signed, bounded cursor pagination and query validation.
- Re-review and pending adjustment approval require signed review/back/confirm flows, with current
  service checks and reviewer balance privacy retained.
- Administrator reads/navigation use effective roles and current MFA, including expiring break-glass grants.
- Network-scoped tokens cannot manage global credentials/preferences. JSON field spellings are exact;
  decoder failures no longer reveal Go types. Public static responses do not set CSRF cookies.
- OpenAPI covers common runtime statuses; temporary authentication storage failures preserve retryable 503 responses.
- Credit-limit ETags represent override/expiry configuration and are compared again inside the write transaction.
- Restore invalidates copied sessions, API tokens, MFA login challenges and break-glass sessions on the
  verified staging copy before swapping; failed invalidation leaves the original database untouched.
- Operator dual-control edits preserve pending policy notice timing and unrelated future values.
  Break-glass login lookup uses the same normalization as sign-in.
- Export byte order remains backwards-compatible with pre-migration closure checksums. Only history
  presentation uses migration 0017's insertion sequence.

Final historical-branch follow-up: restored the tested >500 reversal-history pagination from
`claude/cool-ramanujan-n2peag`; fixed RESTORE-01 with durable interruption markers and seven
subprocess crash tests; added versioned HMAC coverage for backup verification summaries (CRYPTO-02),
keeping legacy manifests readable. Temporary-admin adjustment proposals remain approvable while
the grant is live. Invitation-only join pages hide network existence from nonmembers.

Further completion: optional adjustment links (migration 0018) now flow through pending requests,
approval, audit, history, API and exports while preserving legacy export bytes. Repository contracts
are driver-neutral with old-store aliases. The MVP equivalence suite covers 43 paired HTML/API
operations and six explicit API-only lifecycle/token operations. Policy revisions are atomic;
throttle settings are bounded; scheduled verification is read-only; operator provisioning and displayed
reversal balances are correctly audited. UUID and query validation, forced-password navigation,
recovery-throttle erasure, reversal independence/audit history, timestamps and migration guards have
current regressions. Real SQLite disk-full tests verify rollback, retry and reopen. Traceability now
separates source citations from actual dated test evidence and uploads CI evidence artifacts.
The property harness now isolates each seed, records complete failures and provides a bounded manual
long-run workflow; its 50-sequence smoke passed. Release thresholds are still owner decisions.

Validation is recorded for each candidate in [PR #56 checks](https://github.com/djshellshoxxx/circula/pull/56/checks).
Published checkpoint `e299459` passed Linux race, build targets, lint/vulnerability, fuzz and contract/load
checks; its browser run passed 578 and failed one stale optional-description label expectation, corrected
in `e1b67e6`. Later changes have focused regression/race evidence and require combined validation.
Local Chromium cannot launch in this execution environment; the browser gate runs in GitHub Actions.
No claim of manual screen-reader testing, independent security review or tagged release is made.

See [docs/TODO.md](docs/TODO.md) for current work. All dated sections below are historical records.
They can contain superseded next steps, failures and merge instructions.

## 2026-11-16: beta follow-up

Branch `claude/quirky-einstein-l9ni8m`, draft PR #33 stacked on `claude/circula-beta-completion-1npjw6` (PR #31).
It works through the "Next" list of the 2026-11-11 section below. Every preserved WIP slice is now integrated.

Merged and reviewed (each slice reviewed, findings fixed with regression tests):
- **CI (O4):** one run per commit; macOS 15/Windows tests on `main`, nightly and dispatch only. See [docs/ops/ci.md](docs/ops/ci.md).
- **Go 1.26.11** (GO-2026-6859, ported from PR #30) and the QA-037 marker on migration 0016's temporary-table drop.
- **Export (BR-06, BR-04):** disk spool with reserved bytes and running-export slots, `export-spool-dir`, manifest
  `reconciliation` documented; erasure drops the network's admin-scope exports. QA-030 built.
- **web-ux:** UX-021, UX-049, SEC-016, GOV-010 built. UX-017 partial (adjustments cannot link to an original).
- **arch-guards:** DAT-045 built; ARC-002/ARC-004 adapter and store-contract guards; QA-012 matrix extended.
- **sec-ops-docs:** SEC-020, OPS-030 built (redacting log, request ids aligned); SEC-014, SEC-030, OPS-015 partial.
- **qa-robustness:** QA-007, QA-010, QA-024, QA-026, QA-028 built; QA-001, QA-019 partial. Bug fixed: expired
  proposals now notify both parties.
- **BR-05:** equal Argon2id work for an unusable stored hash; same-network visibility tests (A4).
- **Container (O2):** image and compose run for the first time: [docs/ops/container-verification-2026-11-12.md](docs/ops/container-verification-2026-11-12.md).
- **Traceability (QA-042, partial):** generated by `make traceability`; `make traceability-check` is in `make check`.

Verified at `927d2ca` (later commits change only docs), Go 1.26.11:
`gofmt`, `go vet`, golangci-lint v2.15.0 (0 issues), `check-hygiene.sh`, `check-action-pins.sh`, `go test -count=1 ./...`
and `go test -race -timeout 40m ./...` all pass in every package; `make e2e` 578 passed, 0 failed. CI passed on the PR.
govulncheck ran only in CI.

Next:
1. Owner: merge PR #31, then this PR.
2. Owner decisions raised in this pass: an original link for adjustments (UX-017, schema change); the timing threshold
   (DEC-023, provisional 10 %); QA-001 sequence count in CI; erasure of `person_prefs.time_zone`,
   `person_activity.last_seen_at`, `challenges.reason`; BR-04 scope (all admin scopes, not only `full`).
3. Engineering: backup names have one-second resolution (second backup in the same second is refused); status history
   orders by `occurred_at` (wrong if the clock steps back); 68 built requirements still have no test citation
   (`docs/TRACEABILITY.md`); ARC-002 dual-route test, ARC-004 driver split, ARC-008 operator audit; `read-export.py`
   does not check `reconciliation`; SEC-014 "(optional)" labels on listing forms; run `traceability-check` in CI.
4. Branch protection: drop `test (macos-15-intel)` and `test (windows-2022)` from required checks if they are set.
5. Owner gates below are unchanged: licence, DEC-*, SEC-033, manual screen-reader run.

## 2026-11-11: beta consolidation

## 2026-11-16: repo upkeep

- `docs/wiki` source pages refreshed in the repo. The GitHub wiki is NOT synced yet: the wiki git repo does not accept clones (likely no first page saved in the web UI). Owner action needed, then sync.
- `docs/specs` status refresh in progress (matrix is the reference).
- [docs/TODO.md](docs/TODO.md) is the single open-work list. The "Next" and "Paused" lists in older sections below are history.
- PR triage done. The bot closed no PRs; the owner decides which to close. Checked against the GitHub REST API on 2026-11-16:
  - 0 open issues.
  - 37 open PRs: 5 Dependabot (#46 to #50), #3 (not a draft), and 31 draft chained PRs.
  - PR #31 (head `claude/circula-beta-completion-1npjw6`, base `main`) merged 2026-11-16T01:12:49Z (merge commit `bffcede`). The 2026-11-11 section describes that branch as the draft PR into `main`. No open PR has that head now.
  - PR #19 (named as a draft in the 2026-11-04 section) merged to `main` on 2026-11-16 (`34ccb81`).
- Repo health checks run on 2026-11-16 with go1.26.11 (`go.mod`, `deploy/Dockerfile` and the CI workflows agree on 1.26.11):
  - `gofmt -l .`, `go build ./...`, `go vet ./...`, `scripts/check-hygiene.sh`, `scripts/check-action-pins.sh`: clean.
  - `go test -count=1 ./...`: fails in `internal/service`. `TestDepartureCancelsMoreThanFiveHundredProposals` panics with `rate-limited: too many open proposals` (`testutil.Must`). All other packages pass. Commit `2eacf77` is the last commit that touched it. Owner or engineering to triage.
  - Not run: race detector, e2e, load test, govulncheck (needs vuln.go.dev), release gate.
- Docs corrected in this pass: the release archive contents (`README.md`), the PR #19 status (`docs/ops/README.md`), the container build status (`docs/ops/hardening.md`), the lockout description (fixed 15 minutes, not doubling; `docs/dev/recovery.md`, `docs/security/RAINBOW-BOOKS-SPEC.md`), the release-gate notes (`docs/qa/EVIDENCE.md`, `docs/qa/RELEASE-GATE.md`), a stale unit path (`docs/security/GOV-STANDARDS-MAPPING.md`), and the QA-042 pending note in `docs/TRACEABILITY.md`.
- Not changed: owner decision rows in `docs/DECISIONS.md`. DEC-014 still says "Break-glass not built", which no longer matches `cmd/circula/breakglass.go`. Owner to update the status.


Branch `claude/circula-beta-completion-1npjw6`, a draft PR into `main`. It is the integration branch: it starts from
`claude/gracious-ptolemy-dscc7v@3c2a577` (PR #30) and merges PRs #21–#28, part of #13 and four preserved WIP
slices. Read these first:

- [docs/plans/branch-consolidation-report.md](docs/plans/branch-consolidation-report.md): every branch and the decision on it.
- [docs/plans/circula-completion-matrix.md](docs/plans/circula-completion-matrix.md): feature states with evidence, and the dependency graph.
- [docs/security/security-review.md](docs/security/security-review.md): findings BR-01..BR-06.
- [docs/releases/beta-readiness-report.md](docs/releases/beta-readiness-report.md): exact SHA, commands and results, blockers.

Done: C-09 policy ETag fix; DOM-017/DAT-051 dispute deadline fixed at completion (migration 0016); beta ledger and
session tests; release workflow gates; e2e WCAG/contrast/flow coverage (578 tests); lint at 0 issues.

Next, in order:
1. Owner: review and merge this PR into `main`, then close the PRs it supersedes (see the consolidation report).
2. Export slice: fix review findings BR-06 (bound in-flight bytes, configurable spool directory, document the
   manifest field), then integrate.
3. Rebase web-ux, arch-guards and sec-ops-docs onto this branch, review them, integrate.
4. CI: cut duplicate runs; move the macOS/Windows matrix to main or nightly (measure first).
5. BR-04: drop admin full-export artefacts on member erasure. BR-05: timing check for enumeration.
6. Owner gates below; independent security review (SEC-033); manual screen-reader run.

---

## Earlier handoff (2026-11-04, kept for history)

Updated 2026-11-04. Owned branch: `claude/happy-noether-444cum`, draft PR
https://github.com/djshellshoxxx/circula/pull/19 (base `main`, which holds prompts only).
Started from the product tip `codex/circula-noscript-qa@ba57512`, which contains the #17 → #18
chain and the responsive/no-script QA branches. Earlier handoffs are preserved in Git history
and under `docs/audit/`.

## Done in this pass
- `94728b1` DEF-001: the invitation return link (`next`) was double-escaped, so the return path
  was lost after Create an account / Sign in. This was the only failing e2e case at `b231dc5`.
  Fixed in the templates. Regression test: `TestReturnLinksDecodeToTheOriginalPath`.
- `c34d947`: remediates audit 2026-10-28 Medium findings A1-01, A1-02/A1-11, A1-03, A3-F01 and
  A3-F16. Each PoC is kept as a regression test, and those tests fail with the fix reverse-applied.
  A3-F02 is confirmed but waits on owner decision DEC-014. Report:
  [docs/audit/2026-10-28/REMEDIATION-2026-11-04.md](docs/audit/2026-10-28/REMEDIATION-2026-11-04.md).

- `ce7dcf0`, `d98734c`: merged reviewed slices `wf/auth-sessions` and `wf/gov-policy`. They add
  SEC-027/OPS-010 `circula revoke-sessions` (with `-rotate-web-key`), SEC-003 role session
  lifetimes, SEC-006 configurable web/API limits and `network-join-limit`, SEC-032 grant cap plus
  audited open-mode confirmation, GOV-011 listed removal reasons and conflicted-admin refusal, and
  DOM-031 stored scale-change detection. Defaults keep the earlier behaviour.
- `a2d20b1`: fixes the four blocking findings from an independent review of those merges, each
  with a regression test that fails without the fix.

## Verified locally (Go 1.26.10, this container)
- At `ba57512`: `go build ./...`, `go vet ./...` and `go test -count=1 ./...` pass in all 9 packages.
- At `94728b1`: the full Playwright suite passes, 325 of 325.
- At `c34d947`: `go vet ./...` passes; `go test` passes for service, store, cmd/circula, web, api and domain.

## Open owner questions raised this pass
- GOV-011: should `transaction_id` be mandatory for fraud or breach removals? The conflict check
  only sees the trade named in the request plus undecided disputes and re-reviews.
- DOM-031: should a scale change stored by an earlier build be neutralised once it is in force?
  It is currently detected, audited and logged as a warning only.
- SEC-032: should open self-joins be excluded from the grant even after confirmation? Currently
  they receive it, per DOM-027/DEC-051. Also: how should a lowered cap affect stored grants?
- SEC-006: per-login throttle values, numeric bounds for the rate settings (OPS-008) and a
  per-network login limit.
- DEC-005, DEC-006 and DEC-011 values. DECISIONS.md rows DEC-051, DEC-064 and DEC-065 are stale
  against the built mechanisms; they are owner records and were not edited.

## Paused 2026-11-11 (owner instruction)
- Work stopped at the owner's request. No agents or workflows are running.
- Unmerged, unreviewed slice work is preserved as patch series in
  [docs/audit/wip-slices-2026-11-11/](docs/audit/wip-slices-2026-11-11/README.md), with base, head,
  IDs and state for each.
- Next steps to resume:
  1. Review each implemented slice, then fix and merge it: web-ux, e2e-axe, e2e-flows, arch-guards, sec-ops-docs.
  2. Finish boundary-fuzz, export and release-chain.
  3. Implement qa-robustness.
  4. Cut duplicate CI runs and move the macOS/Windows test matrix to main only, measured as in
     the scratch notes: macOS race 7.9 min, Ubuntu race 6.8 min, e2e 4.7 min.
  5. Regenerate docs/TRACEABILITY.md (QA-042).
- Last CI result: `76e2c89` passed every job (push and PR runs). CI for later heads had not
  finished when work was paused.

## Owner gates (not decided here)
- Licence; DEC-005, DEC-011, DEC-013, DEC-014, DEC-021, DEC-022, DEC-023 and DEC-025; UX-047 locale scope.
- Release thresholds; pilot jurisdictions; the manual screen-reader run (UX-056, QA-035).
- Independent security review (SEC-033); A3-F02 break-glass scope.
- Merging to `main`, releases and deployments.
