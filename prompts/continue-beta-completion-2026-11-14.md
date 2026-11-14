# Circula: continue beta completion from the latest branches (2026-11-14)

You are continuing Circula, a mutual-credit ledger written in Go (SQLite store, HTML web UI, JSON API, CLI, Playwright
e2e). Work at high effort and finish as much as possible in this one session. Verify everything you claim by running
it. Do not merge to `main`, tag, release or deploy: those are owner gates.

## 1. Where things stand (read before acting)

`main@4b422ad` holds only a prompt. All product code lives on unmerged branches. There are **two frontier branches**
that diverged from a common ancestor (`claude/happy-noether-444cum@34ccb81`, PR #19). Neither contains the other.

| Line | Branch (PR) | Head | Notes |
|---|---|---|---|
| **B (primary)** | `claude/quirky-einstein-l9ni8m` (#33) | `8b4c8ba` | Builds on `claude/circula-beta-completion-1npjw6` (#31), which integrated PRs #21–#28 and four WIP slices. Has `HANDOFF.md`, `docs/plans/circula-completion-matrix.md`, `docs/plans/branch-consolidation-report.md`, `docs/security/security-review.md`, `docs/releases/beta-readiness-report.md`, `docs/ops/container-verification-2026-11-12.md`. Adds CI dedupe (`b31714d`), QA-037 hygiene fix (`689bf84`), Go 1.26.11 (`c22ada1`), spec status updates (`870dd08`), container verification (O2). |
| **A** | `chatgpt/circula-consolidation-review-20261111` (#34) | `02f9f0a` | Builds on `haiku/circula-consolidation`. Its unique work (by `git cherry` against B) is listed below. |

Unique patches on A that B lacks (`git cherry -v origin/claude/quirky-einstein-l9ni8m origin/chatgpt/circula-consolidation-review-20261111`):

- `2b01510` web: reversal links, member references, leave export, refusal coverage. This is the **web-ux slice**
  (UX-017, UX-049, GOV-010, SEC-016, UX-021) that B's consolidation report marks "not integrated, conflicts in
  `internal/web/refusal.go`".
- `f8ce749` store: persistence regression tests (H05).
- `f83ae1c` test(web): removal above the dual-control threshold sends a valid reason code.
- `fbdcfc1` + `d8e4e0b` reversal history pagination: `TestReversalsOfResultLimit` (501 reversals) and `ReversalsOf`
  paging over `TxFilter.AfterID` in 500-row pages. **Never executed.** Verify.
- `df89562`, `3c3ebeb` CI changes (push on main only, cache keys, macos-15-intel). These overlap B's `b31714d`;
  reconcile, do not stack both.
- `d900316`, `026c6ee`, `1a06ae4`, `02f9f0a` docs: `docs/audit/CONSOLIDATION-QUALIFICATION-2026-11-11.md`,
  `docs/audit/CI-WORKFLOW-QUALIFICATION-20261111.md`.
- A also has e2e/web commits that are the same slices as B's (`eeb2851`, `f38647c`, `519e731`, `504c1fe`); git cherry
  shows them as different patches, so diff the trees before choosing a version. Prefer B's unless A's is a strict
  improvement.

Other branches with work in neither A nor B (check, then decide):

- `claude/gracious-ptolemy-dscc7v@1d16598` (#30): `0a33e3c` HTTP-level acceptance scenario test in `cmd/circula`.
  Check whether an equivalent is already on B (B's base was `3c2a577`); integrate if missing.
- `docs/future-usability-network-specs-20261111` (#32), `planning/future-cli-specs-20261110`: future-feature docs.
  Out of beta scope; leave alone, mention in the handoff.
- `claude/relaxed-hopper-8q4vhq` (#29), `claude/sweet-turing-m3ftja`, `docs/foundational-*`, `docs/readme-and-wiki`,
  `claude/circula-completeness`, `codex/circula-integration-plan-20261030`, `chatgpt/zero-codex-continuation-spec-20261109`:
  superseded or owner decision per `docs/plans/branch-consolidation-report.md`. Do not integrate.

Still open per B's `HANDOFF.md`, the completion matrix and the security review:

1. **Export slice** (`docs/audit/wip-slices-2026-11-11/`, ARC-020 GOV-022 QA-030 FR-095), review finding BR-06:
   bound in-flight export bytes before the cap check, make the spool directory configurable with no silent fallback
   to unbounded memory, document the new manifest field.
2. **arch-guards** (ARC-008 QA-012 DAT-045 ARC-002 ARC-004; conflict in `cmd/circula/config.go`) and
   **sec-ops-docs** (SEC-014 SEC-020 SEC-030 OPS-030 OPS-015; conflicts in `cmd/circula/serve.go`,
   `internal/web/web.go`; tests never confirmed): rebase, review, fix, integrate.
3. **qa-robustness** slice, never started: QA-001 QA-007 QA-010 QA-024 QA-028 QA-026 QA-019 (definitions in
   `docs/specs/12-quality-test-plan.md`).
4. **BR-04**: drop a network's admin full-export artefacts when a member is erased (`internal/api/erasure_hook.go`).
   **BR-05**: test timing equivalence of unknown-login vs wrong-password (sign-in and recovery).
5. **QA-042**: regenerate `docs/TRACEABILITY.md` from source and tests.
6. Stale spec status columns in `docs/specs/` (the matrix notes several; `docs/specs/12-quality-test-plan.md` lists
   28 P0 items as partial). Re-check each P0 item against code and tests; implement what is missing, or correct the
   status with evidence.
7. Container observation 1 (`docs/ops/container-verification-2026-11-12.md`): `serve` migrates a brand-new database
   even with `migrate-mode=explicit`. Decide whether that is the intended behaviour per the specs; fix code or the
   comment, with a test.

## 2. Setup

- Create a working branch from B: `git fetch origin` then branch from `origin/claude/quirky-einstein-l9ni8m`.
  Push only to the branch your session is assigned. Never rewrite other branches.
- `go.mod` wants Go 1.26.11. If the local toolchain is older, let `GOTOOLCHAIN=auto` fetch it; if the proxy blocks
  that, report it and use the newest available toolchain, noting the deviation.
- e2e: `make e2e` (Playwright; use the pre-installed Chromium, do not reinstall Playwright). Lint: `golangci-lint`
  v2.15.0 as pinned in CI.
- Baseline first: on the unmodified B head run `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`,
  `go test -race ./...` (Linux), lint and `make e2e`. Record exact results. Any red result on the baseline is task 0.

## 3. Work, in this order

Use test-driven development for every behaviour change: write the failing test, see it fail for the right reason,
fix, see it pass. For each integrated slice, review the diff as a hostile reviewer (ledger invariants, authorization,
idempotency, audit durability, CSRF, IDOR, resource limits) before merging it.

1. **Port A's unique work onto B.** Merge or cherry-pick `2b01510` (resolve `internal/web/refusal.go` against B),
   `f8ce749`, `f83ae1c`, `fbdcfc1`, `d8e4e0b`. Run `go test ./internal/service -run '^TestReversalsOfResultLimit$'
   -count=1 -timeout=15m`. If posting 501 reversals is refused by a domain rule, document the limit and reclassify the
   test; otherwise make it pass and check the web history view stays bounded. Reconcile CI: keep one coherent
   `ci.yml` (B's main/nightly macOS-Windows policy plus A's `go.sum` cache keys if still useful). Carry over A's two
   qualification docs, updated to the facts you verified.
2. **Gracious-ptolemy acceptance test** (`0a33e3c`), if not already equivalent on B.
3. **Export slice + BR-06**, then **BR-04**.
4. **arch-guards**, then **sec-ops-docs**.
5. **boundary-fuzz / release-chain leftovers**: confirm nothing from those WIP series remains unintegrated.
6. **qa-robustness** slice (implement).
7. **BR-05** timing test (statistical, non-flaky; skip under `-short` if slow).
8. **Migrate-mode observation** (item 7 above).
9. **Spec P0 sweep** (item 6 above), then **QA-042** traceability regeneration.

After each step: gofmt, vet, full `go test`, and e2e if web/templates changed. Commit in small, well-described
commits. Push after each completed step so nothing is lost if the session ends.

## 4. Finish

- Final full gate on the exact head: gofmt, vet, `go test -count=1 ./...`, `-race`, lint, `make e2e`, fuzz targets
  briefly (`FuzzAPICommandBody`, `FuzzOpenCursor`, 15 s each), `scripts/smoke-deploy.sh` if runnable.
- Update `docs/plans/circula-completion-matrix.md`, `docs/security/security-review.md` (BR-04/05/06 status),
  `docs/plans/branch-consolidation-report.md` (A and the new decisions), `docs/releases/beta-readiness-report.md`
  (exact SHA, commands, results) and the top section of `HANDOFF.md` (done, next, owner gates). Record real outcomes,
  including failures and anything skipped.
- Open or update a **draft** PR into `main` for your branch. In its body list which PRs it supersedes (#19, #30, #31,
  #33, #34, and others per the consolidation report) so the owner can close them. Do not close PRs yourself.

## 5. Owner gates (do not decide; list them in the handoff)

Licence; DEC-005, DEC-011, DEC-013, DEC-014 (A3-F02 break-glass scope), DEC-021, DEC-022, DEC-023, DEC-025, DEC-072;
UX-047 locale scope; release thresholds and pilot jurisdictions; GOV-011, DOM-031, SEC-032, SEC-006 questions in
`HANDOFF.md`; whether PR #29's spec rewrite replaces `docs/specs/`; independent security review (SEC-033); manual
screen-reader run (UX-056, QA-035); merging to `main`, tags, releases, deployments.

## 6. Rules

- Evidence before claims: never mark something COMPLETE without a test you ran on the final head.
- Never skip, disable or weaken a test to get green. Never use floats for money. Keep migrations forward-only and
  contiguous (next is 0017).
- If you run low on context or time, stop at a clean commit, push, and write the handoff so the next session can
  resume from it.
