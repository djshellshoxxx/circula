# Circula Continuation Prompt — audit remediation and integration

Copy this prompt into a coding agent session with `djshellshoxxx/circula` checked out.
Written 2026-10-30 from a review of every branch and PR. Re-verify live state before acting; anything below may have moved.

---

## Role

Act as Circula's maintainer: Go engineer, security fixer, and release coordinator. Continue the existing build. Do not restart, redesign, or re-audit from scratch.

## Where the code actually is

`main` holds only `prompts/`. **All product work lives on a stacked PR chain. Never base work on `main`.**

| Ref | Role |
|---|---|
| `claude/awesome-darwin-y6eaiz` @ `ae512e5` | **Integration tip.** Contains the whole build line (circula-build → security-features → finish → completeness → work/spec-completion → codex/circula-resume → support-boundaries). Draft [PR #10](https://github.com/djshellshoxxx/circula/pull/10). |
| `claude/sharp-archimedes-tu76rh` @ `fe11179` | Draft [PR #12](https://github.com/djshellshoxxx/circula/pull/12): 2026-10-28 code audit, docs only. 55 verified findings, PoC tests, unapplied WIP patches. Last activity 2026-10-29 03:15 UTC. |
| `checkpoint/circula-handoff-20261025` @ `3d0080a` | Draft [PR #13](https://github.com/djshellshoxxx/circula/pull/13): frozen archive. 6 commits not in tip, incl. C-09 pending-policy ETag regression test (overlaps audit A2-07). Keep draft. Do not merge over newer work. |
| PRs [#1](https://github.com/djshellshoxxx/circula/pull/1), [#2](https://github.com/djshellshoxxx/circula/pull/2), [#3](https://github.com/djshellshoxxx/circula/pull/3), [#5](https://github.com/djshellshoxxx/circula/pull/5) | Early competing spec/README branches against `main`, **not** in tip. Likely superseded. #1 adds crypto/tokenization ideas that contradict spec scope (credit unit is not a cryptocurrency; tokenization deferred). |
| PRs #4, #7, #8, #9 | Open drafts whose heads are already ancestors of tip. Stack plumbing only. |

Stack: `main ← #4 ← #7 ← #9 ← … ← #10 ← #12`. Nothing has reached `main`.

## Verified state (2026-10-30, tip `ae512e5`)

Go 1.26.10 (from `go.mod`; CI uses `go-version-file`). `go build ./cmd/circula`, `go vet ./...`, `gofmt -l .` (empty), `go test -count=1 ./...`: **all pass**. Last full CI (9/9 jobs, incl. E2E, lint, govulncheck, gitleaks, reproducibility, fuzz) passed on source `5000a5e`; tip adds docs only.

Product: self-hosted Go + SQLite mutual-credit ledger. Web UI + JSON API, append-only double entry, hash-chained audit, MFA, recovery, dual control, break-glass, erasure, encrypted signed backups. Specs in `docs/specs/00–13`, decisions in `docs/DECISIONS.md`, traceability in `docs/TRACEABILITY.md`.

## Setup

1. Use the branch your session assigns. If it has no unique commits, reset it onto the tip:
   `git fetch origin claude/awesome-darwin-y6eaiz && git checkout -B <your-branch> origin/claude/awesome-darwin-y6eaiz`.
   If it has unique commits, stop and reconcile; never discard them.
2. Fetch the audit: `git fetch origin claude/sharp-archimedes-tu76rh`. Read, do not edit:
   `docs/audit/2026-10-28/interim-verification-circula.md` (findings table), `…/poc/**` (PoC tests as `.txt`), `…/wip-fixes/**` (unreviewed patches against `bb5a36f`).
3. Skim `HANDOFF.md`, `CONTRIBUTING.md`, `docs/qa/RELEASE-GATE.md`. Do not repeat broad repository scans; this prompt is the coverage map.
4. Check ownership of PR #12 before starting. If it shows new commits or comments after 2026-10-29 03:15 UTC, coordinate: take only findings it has not claimed. If idle, its remediation is yours. Never push to its branch.

## Work, in priority order

### 1. Repository hygiene (small, do first)

- A 21 MB compiled ELF `circula` is committed at repo root (added in `e7677f4`). `git rm --cached circula` and add `/circula` to `.gitignore`. Removing it from history needs a force push: **do not** without explicit owner approval. Note it in the PR instead.
- `scripts/check-hygiene.sh` does not catch committed binaries. Add a check (e.g. fail on tracked ELF/Mach-O/PE files or files > 5 MB).

### 2. Fix the six Medium audit findings

One commit per finding (or per tightly coupled pair). TDD: port the PoC from `poc/` into a real `_test.go`, watch it fail, fix, watch it pass. WIP patches are reference only; review each adversarially before reuse (`disputes.patch.txt` is known incomplete).

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

- `docs/specs/13-assumptions-risks-decisions.md` §9 and `docs/DECISIONS.md` (~lines 823, 830) still say MFA (DEC-007) and break-glass (DEC-014) are "not built". They are built. Mark "built with recommended default; owner confirmation outstanding". Do not mark the decision itself as made.
- `README.md` says "Go 1.24 or newer"; `go.mod` requires 1.26.10.
- `docs/audit/COMPLETENESS.md` is stale (e.g. FR-054 done in PR #11). Refresh statuses you can verify. `docs/TRACEABILITY.md` (last updated 2026-10-21) has no committed generator: update affected rows by hand, or add a small script if cheaper.
- Replace `HANDOFF.md` with a short current handoff (≤60 lines): tip SHA, branch map, done, open, blocked. Move receipts into `docs/audit/`.

### 5. Remaining P0/P1 gaps (only after 2–4)

From `docs/audit/COMPLETENESS.md` and `docs/qa/GAP-ANALYSIS.md`: axe a11y run in `ci.yml` (ARC-017, QA-033), import-layer lint via `depguard` (ARC-018), export-package verifier command (DAT-040), member and reviewer audit views (FR-082, DAT-033, DAT-035), API `admin` token scope currently 501 (API-010). Pick one at a time; each gets spec cross-ref, tests, traceability update.

### 6. Integration proposal (owner decides)

Write `docs/INTEGRATION-PLAN.md`: how to land the stack on `main` (e.g. one squash or merge PR from tip → `main`, then close #4/#7/#8/#9 as merged-through; close #1/#2/#3/#5 after salvaging anything useful, listing what). Include exact SHAs and required checks. **Propose only. Do not merge or close PRs without the owner's explicit per-PR confirmation.**

## Owner-blocked: ask, never decide

Licence (DEC-004; no LICENSE file, README says do not redistribute), DEC-072 (who may decide disputes), DEC-074 (visible `accepted` state), UX-047 locale scope, confirmation of built defaults for DEC-007/009/014, pilot jurisdictions, manual screen-reader test (`docs/qa/SCREEN-READER.md`), independent professional security review. Assistant reviews do not satisfy these gates.

## Gates before every push

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
