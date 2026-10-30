# Circula Continuation Prompt — audit remediation and integration

Copy this prompt into a coding agent session with `djshellshoxxx/circula` checked out.
Updated 2026-10-30 after reviewing all 16 remote branches and merging approved PRs #10 and #12. Re-verify live state before acting; anything below may have moved.

---

## Role

Act as Circula's maintainer: Go engineer, security fixer, and release coordinator. Continue the existing build. Do not restart, redesign, or re-audit from scratch.

## Where the code actually is

`main` holds only `prompts/`. **All product work lives on a stacked PR chain. Never base work on `main`.**

| Ref | Role |
|---|---|
| `claude/awesome-darwin-y6eaiz` @ `827e651fe151f49659cc20aefe85d4af67bdaa97` | **Integration tip.** Contains the build line, historical balances, independent-approver UI/fixtures, support guidance and merged interim audit evidence. [PR #12](https://github.com/djshellshoxxx/circula/pull/12) is merged; no audit fix was applied. |
| `codex/circula-resume-20261024` @ `f4bfe6bc3f81c94b859abeae902ae862ea6e143e` | [PR #10](https://github.com/djshellshoxxx/circula/pull/10) is merged here. Its tree equals tested source `ae512e51d4f7a611119a5567225b862969770207`; it does not yet include PR #12's audit documents. |
| `claude/sharp-archimedes-tu76rh` @ `fe111799ffabb2427d6d94feb32a037fa3ed2af8` | Preserved external audit source. Interim results contain 55 batch-verifier entries, incomplete extra-lens checks, PoC text and unapplied WIP patches. Verification/coverage/consolidation remain unfinished; this is not a professional audit. |
| `checkpoint/circula-handoff-20261025` @ `3d0080a` | Draft [PR #13](https://github.com/djshellshoxxx/circula/pull/13): frozen archive. 6 commits not in tip, incl. C-09 pending-policy ETag regression test (overlaps audit A2-07). Keep draft. Do not merge over newer work. |
| PRs [#1](https://github.com/djshellshoxxx/circula/pull/1), [#2](https://github.com/djshellshoxxx/circula/pull/2), [#3](https://github.com/djshellshoxxx/circula/pull/3), [#5](https://github.com/djshellshoxxx/circula/pull/5) | Early competing spec/README branches against `main`, **not** in tip. Likely superseded. #1 adds crypto/tokenization ideas that contradict spec scope (credit unit is not a cryptocurrency; tokenization deferred). |
| PRs #4, #7, #8, #9 | Open drafts whose heads are already ancestors of tip. Stack plumbing only. |

Product code has not reached `main`. PR #14 was already merged before this continuation. Read [the proposed integration plan](../docs/INTEGRATION-PLAN.md) for exact source/target receipts and preserved branches; do not repeat the already completed merges.

## Verified state (2026-10-30)

Go 1.26.10 (from `go.mod`; CI uses `go-version-file`). At source `ae512e51d4f7a611119a5567225b862969770207`, push run `40820305135` and PR run `40820312600` each passed all nine jobs, including E2E, OS tests, lint/vulnerability/secret checks, cross-build, reproducibility, contract/load and fuzz. Audit source `fe11179` also passed nine jobs in each of runs `40811138499` and `40811136958`. Later post-merge results belong to their exact merge SHAs; see the integration plan for current receipts.

The existing integration handoff records another session's local build/vet/test and 103-browser-test results. This review used source inspection, existing remote CI and conflict simulations; it did not rerun full application tests locally. This executor's prior uncached-module and artifact-byte HTTP 403 denials remain unresolved. Do not retry denied downloads, use alternate hosts or treat another session's access as permission here.

Product: self-hosted Go + SQLite mutual-credit ledger. Web UI + JSON API, append-only double entry, hash-chained audit, MFA, recovery, dual control, break-glass, erasure, encrypted signed backups. Specs in `docs/specs/00–13`, decisions in `docs/DECISIONS.md`, traceability in `docs/TRACEABILITY.md`.

## Setup

1. Inspect repository identity, instructions, `git status`, worktrees and branch ownership. Preserve dirty/untracked files and every unique commit. Fetch the tip with an explicit refspec (some checkouts otherwise fetch only `main`):
   `git fetch --no-tags origin refs/heads/claude/awesome-darwin-y6eaiz:refs/remotes/origin/claude/awesome-darwin-y6eaiz`.
   Create a new branch/worktree from the verified tip, for example `git worktree add -b <new-branch> <new-worktree> origin/claude/awesome-darwin-y6eaiz`. Do not reset an existing branch or overwrite another owner's worktree.
2. The audit packets are now in the integration tip. Read, do not apply blindly:
   `docs/audit/2026-10-28/interim-verification-circula.md` (findings table), `…/poc/**` (PoC tests as `.txt`), `…/wip-fixes/**` (unreviewed patches against `bb5a36f`).
3. Skim `HANDOFF.md`, `CONTRIBUTING.md`, `docs/qa/RELEASE-GATE.md`. Do not repeat broad repository scans; this prompt is the coverage map.
4. Reconcile live audit/remediation branches, handoffs and outstanding ownership before starting. A merged PR or a period of inactivity does not transfer ownership. Reserve only explicitly unowned findings through the coordinator; if ownership is uncertain, continue an independent accepted task or report the blocker. Never push to another owner's source branch.

## Work, in priority order

### 1. Repository hygiene (small, do first)

- A 21 MB compiled ELF `circula` is committed at repo root (added in `e7677f4`). `git rm --cached circula` and add `/circula` to `.gitignore`. Removing it from history needs a force push: **do not** without explicit owner approval. Note it in the PR instead.
- `scripts/check-hygiene.sh` does not catch committed binaries. Add a check (e.g. fail on tracked ELF/Mach-O/PE files or files > 5 MB).

### 2. Fix the six Medium audit findings

The interim verifier reports six Medium findings; extra-lens review is incomplete and includes a refuted skeptic result for A3-F02. Reproduce and try to disprove each candidate at the current source before claiming a runtime defect or starting an owned fix. Preserve original evidence. One commit per finding (or per tightly coupled pair). Port the PoC into a real `_test.go`, preserve failing-before output, fix, then verify. A1/A3 PoCs assert desired behavior and fail before a fix; A2 PoCs assert defect presence and pass before a fix, so adapt their regression direction explicitly. WIP patches are reference only; review each adversarially before reuse (`disputes.patch.txt` is known incomplete).

| ID | Defect |
|---|---|
| A1-01 | Settle-to-zero departure posts an adjustment with no network-state gate (closed/closing/suspended/admin-required). |
| A1-02 (+A1-11) | Decimals guard checked only when a change is requested; a scheduled change lands after postings exist. Decimal scale must be fixed at creation; `TestPolicyChanges` asserts the wrong behaviour; FR-005/GOV-004 wrongly marked built. |
| A1-03 | Re-review posts a reversal on a trade another live dispute holds, stranding a `posting_pending` dispute. |
| A3-F01 | With `backup-encryption-key` set, pre-migration gate runs `integrity_check` on `.db.enc` ciphertext; `migrate` and `serve` (auto mode) can never proceed. |
| A3-F02 | Break-glass "no lasting changes" (`requireLasting`) covers ~10 methods; a break-glass admin can post adjustments and approve dual control. Invert to deny-by-default. |
| A3-F16 | Second-factor throttle renewable by one failure at expiry and shared with session-only `DisableMFA`; a stolen session locks the owner out. |

### 3. Low findings (batch by subsystem)

- Ledger/disputes/lifecycle: A1-04, A1-05, A1-06, A1-07.
- API/auth: A2-01, A2-02, A2-03, A2-04, A2-05, A2-06, A2-07 (reconcile with C-09 test in PR #13: port the test, fix the ETag, credit the source), A2-09, A2-10, A2-11, A2-12, A2-14.
- Ops/security: A3-F03, A3-F05, A3-F06, A3-F09, A3-F10, A3-F11, A3-F13, A3-F14, A3-F15, A3-F17, A3-F18, A3-F19, A3-F25.
- Also A2-08: add the missing 413/415/423/429/503 responses to `openapi.yaml` so contract validation matches runtime.

Info findings (A1-08..10, A2-13, A2-15..21, A3-F07, F12, F20..24): fix the cheap ones; record the rest in `docs/qa/DEFECTS.md` with rationale. A2-21 (unkeyed SHA-256 for tokens/session ids vs spec SEC requirement) needs a migration plan; write it, don't rush it.

### 4. Documentation drift (fix as you touch areas)

- Reconcile MFA/break-glass implementation wording with both source and interim findings. Distinguish implemented features, remaining enforcement defects and owner-unconfirmed defaults; do not claim the entire control is complete or the decision made.
- `README.md` says "Go 1.24 or newer"; `go.mod` requires 1.26.10.
- `docs/audit/COMPLETENESS.md` is stale (e.g. FR-054 done in PR #11). Refresh statuses you can verify. `docs/TRACEABILITY.md` (last updated 2026-10-21) has no committed generator: update affected rows by hand, or add a small script if cheaper.
- Replace `HANDOFF.md` with a short current handoff (≤60 lines): tip SHA, branch map, done, open, blocked. Move receipts into `docs/audit/`.

### 5. Revalidate remaining P0/P1 candidates (only after 2–4)

`docs/audit/COMPLETENESS.md` and `docs/qa/GAP-ANALYSIS.md` contain historical statuses. Automated axe/keyboard accessibility tests already run through the `e2e` job in `.github/workflows/ci.yml` and `e2e/tests/a11y.spec.js` (ARC-017, QA-033); do not reimplement them. Manual screen-reader evidence remains distinct. Revalidate candidates such as import-layer lint (ARC-018), export verification (DAT-040), member/reviewer audit views (FR-082, DAT-033, DAT-035) and admin-token scope (API-010) against the current source before calling them missing. Select only an accepted, unowned gap; each gets a spec cross-reference, behavioral tests and traceability update.

### 6. Integration proposal (owner decides)

Maintain `docs/INTEGRATION-PLAN.md`: the owner approved and this continuation completed development merges #10 and #12. Product integration to `main`, release and old-PR closure remain proposed only. Prefer a normal history-preserving product integration once findings, ownership, reviews and required checks are reconciled. Salvage unique older documentation only after content comparison; do not close or overwrite those branches here.

## Owner-blocked: ask, never decide

Licence (DEC-004; no LICENSE file, README says do not redistribute), DEC-072 (who may decide disputes), DEC-074 (visible `accepted` state), UX-047 locale scope, confirmation of built defaults for DEC-007/009/014, pilot jurisdictions, manual screen-reader test (`docs/qa/SCREEN-READER.md`), independent professional security review. Assistant reviews do not satisfy these gates.

## Gates for product changes before every push

Use these on an application checkout. A prompt-only branch has no `go.mod`,
Makefile or application CI; record its documentation checks separately and do
not claim an application test pass for it. Preserve all product gates below.

```sh
make check                       # gofmt, vet, race tests, golangci-lint, govulncheck, action pins, hygiene
make e2e                         # when web/API behaviour changed (Playwright + axe; see docs/qa/E2E.md)
make contract                    # when API shape changed
gitleaks detect --log-opts="<base>..HEAD"
git diff --check <base>
```

Report actual output. If a gate cannot run (network, tool missing), say so; never claim it passed.

## Hard rules

- No force push, merge, PR close, release, tag, deploy, credential or repo-setting change without explicit owner confirmation.
- Never skip, delete, or weaken a test to get green. Never edit another session's branch.
- Ledger invariants are sacred: equal-and-opposite postings, append-only entries, every command audited, `verify-ledger` stays green.
- No legal or compliance claims. No crypto, tokenization, or currency redemption features (deferred by spec).
- Keep each change to what the finding needs. Update the finding's status (fixed + commit SHA) in a remediation log at `docs/audit/2026-10-28/REMEDIATION.md`.
- No model identifiers in commits, PRs or files.

## Finish

Push your branch, open a **draft** PR based on `claude/awesome-darwin-y6eaiz` (not `main`), mirroring the PR description style of #10. Summarise: findings fixed (IDs + SHAs), findings deferred with reasons, doc drift fixed, gate results with real output, owner questions still open, and the recommended next slice.
