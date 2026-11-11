# Branch consolidation report (beta)

Date: 2026-11-11. Integration branch: `claude/circula-beta-completion-1npjw6`. The session that did this work may
push only to that branch, so it is the integration branch; the repository has no `integration/*` convention.
Base: `main@4b422ad`, which holds only `prompts/create-foundational-specs.md`. Every product commit is on unmerged
branches. Method: `git merge-base --is-ancestor` for every pair of the 34 remote branches, then a diff of each tip
against the most complete tip.

## Summary

| Measure | Count |
|---|---|
| Remote branches inspected | 34, including this one |
| Branches with commits not in the integration base `3c2a577` | 17 |
| Contained in the base (fast-forward), no unique work | 14 product branches, plus the base itself and `main` |
| Of the 17: merged | 8 (PRs #21–#28) |
| Of the 17: partly ported | 1 (`checkpoint/circula-handoff-20261025`, PR #13) |
| Of the 17: not integrated (superseded, out of scope or owner decision) | 8 |
| Preserved WIP slices: integrated / not integrated | 4 / 4 |
| Open PRs at start | 25 (all drafts except #3; #6, #10, #11, #12, #14 were already merged between branches). Nothing was merged to `main`, and no PR was closed. |

No branch was deleted and no history was rewritten. Every branch not integrated is still on the remote.

## Lineage

```
main 4b422ad (prompt only)
 └─ security-features → build → finish → work/spec-completion → support-boundaries → resume-20261024
     → awesome-darwin → sharp-archimedes (PR #11, #12) → consolidate-20261030 (#17) → staging-refresh-docs (#18)
     → responsive-qa → noscript-qa → happy-noether (#19) → gracious-ptolemy (#30)          ← integration base
                                        └─ d98734c ─┬─ chatgpt terms-return → macOS15 → race-budget → decidable-expiry → cookie-policy (#24-#28)
                                                    ├─ chatgpt audit-20261110-a (#22)
                                                    ├─ chatgpt docs-20261110-a (#23)
                                                    └─ chatgpt qualify-pr19-20261110-a (#21)
 build ─ circula-completeness (#7)      side line; only its last docs commit is not in the base
 main ─ docs-only roots: foundational-specifications (#1), sweet-turing (#2), spec-shells (#3), readme-and-wiki (#5),
        relaxed-hopper (#29), zero-codex-continuation-spec (#20); integration-plan (#16) and checkpoint (#13) are docs tips
```

## Per-branch decisions

| Branch (PR) | State vs integration base | Decision | Evidence |
|---|---|---|---|
| `claude/gracious-ptolemy-dscc7v` (#30) | Most complete product tip: happy-noether plus the 20-step acceptance test | **Base (fast-forward)** | `3c2a577`. Build, vet and `go test ./...` passed, 9/9 packages |
| `claude/happy-noether-444cum` (#19), `codex/circula-noscript-qa`, `codex/circula-responsive-qa`, `codex/circula-staging-refresh-docs` (#18), `codex/circula-consolidate-20261030` (#17), `claude/sharp-archimedes-tu76rh`, `claude/awesome-darwin-y6eaiz`, `codex/circula-resume-20261024`, `codex/circula-support-boundaries-20261028`, `work/spec-completion` (#9), `claude/circula-finish` (#8), `claude/circula-build` (#4), `claude/circula-security-features`, `claude/vibrant-brahmagupta-3uofnj` (#15) | Ancestors of the base | **Integrated (contained)** | `merge-base --is-ancestor` true for each |
| `chatgpt/zero-codex/cookie-policy-adapter-20261110-b` (#28), which contains `decidable-expiry` (#27), `ci-macos-race-budget` (#26), `ci-macos15-retirement` (#25) and `terms-return` (#24) | 18 commits past fork `d98734c`. Fixes: terms return path, expired dual-control hints, secure-cookie adapter test, macOS 15 runner and race budget | **Merged** | `a2cff88`, no conflicts. Tests pass at the merge |
| `chatgpt/zero-codex/audit-20261110-a` (#22), `docs-20261110-a` (#23), `qualify-pr19-20261110-a` (#21) | Docs only: audit ledger, wiki sources (`docs/wiki/`), handoffs | **Merged** | `aa7a5a6`, `f17fe1c`, `c67a662` |
| `checkpoint/circula-handoff-20261025` (#13) | 6 commits from `30d18a1` | **Partly ported.** `763603f` (C-09 pending-policy ETag regression test and audit note) ported; it **reproduced a real defect**, fixed in `addc229`. `001f09a` e2e fixture rejected: superseded by the A3-01 rewrite of the same test in PR #11. Handoff/RESUME/CI-evidence docs not ported (superseded) | `addc229`, `2cd1ec8` |
| `claude/circula-completeness` (#7) | 1 commit not in base: `docs/HANDOFF.md`, written 2026-10-22 | **Not integrated** (superseded). Its code reached the base through `circula-build` | `git rev-list base..branch` = 1 |
| `codex/circula-integration-plan-20261030` (#16) | 1 docs commit (handoff, consolidation receipt, prompt) | **Not integrated** (superseded by this report) | |
| `claude/relaxed-hopper-8q4vhq` (#29) | Independent rewrite of all 14 specs under different file names, plus `tools/check_specs.py`, opened 2026-11-11 | **Not integrated: owner decision.** It would duplicate and contradict `docs/specs/`, which the code's requirement IDs and traceability use | 17 files, 8487 lines from `main` |
| `docs/foundational-specifications` (#1) | Early specs plus brainstorming on crypto and crypto-libertarian design | **Not integrated: out of scope** (no blockchain/crypto scope; specs superseded) | |
| `claude/sweet-turing-m3ftja` (#2), `docs/foundational-spec-shells` (#3), `docs/readme-and-wiki` (#5) | Early spec, plan and README drafts from `main` | **Not integrated** (superseded by `docs/specs/` and README in the base) | |
| `chatgpt/zero-codex-continuation-spec-20261109` (#20) | Process document for a parallel agent lane | **Not integrated** (process, not product) | |

## Preserved WIP slices (`docs/audit/wip-slices-2026-11-11/`)

| Slice | Applies to base? | Decision | Evidence |
|---|---|---|---|
| boundary-fuzz | yes | **Integrated** (`62ab2ce`, `7648a51`). Tests only; gofmt fix in `e127db8` and lint fix in `9217825` | `go test ./internal/api ./internal/service` pass. `FuzzAPICommandBody` and `FuzzOpenCursor` ran 15 s each with no findings |
| release-chain | yes | **Integrated with fix** (`d1ca324`, `72c7409`). The independent review's one medium finding (the release workflow did not run the new checks) is fixed in `release.yml` | `go test ./scripts/sbom` passes; an offline single-target package and verify run came out clean |
| e2e-axe | yes | **Integrated** (`a5be4e8`, `8779719`) | `make e2e`: 578/578 with e2e-flows stacked |
| e2e-flows | yes | **Integrated** (`cb1fe5b`, `7569285`). The production change is link additions and one CSS word-wrap rule, reviewed | Same e2e run, plus `ux002_reach_test.go` |
| export | yes | **Not integrated: accept-with-fixes, unresolved.** Medium findings: in-flight exports are not bounded before the cap check; the spool directory defaults to `os.TempDir()` and silently falls back to unbounded memory; the new manifest field is undocumented | Independent review. Tests pass in an isolated worktree |
| web-ux | conflict in `internal/web/refusal.go` | Not integrated (unreviewed, conflicts) | `git am -3` |
| arch-guards | conflict in `cmd/circula/config.go` | Not integrated (unreviewed, conflicts) | `git am -3` |
| sec-ops-docs | conflicts in `cmd/circula/serve.go`, `internal/web/web.go` | Not integrated (unreviewed; the implementer's tests were never confirmed) | `git am -3` |

## New work on the integration branch (not from any branch)

| Commit | Change |
|---|---|
| `addc229`, `2cd1ec8` | C-09: the policy ETag includes the pending policy version, so a stale `If-Match` gets 412 |
| `86b8a4a` | Ledger and session beta qualification tests |
| `e127db8` | DOM-017/DAT-051 (P0 required correction): the dispute deadline and its policy version are fixed at completion; migration 0016 backfills them |
| `72c7409` | Release workflow enforces the release-notes and release-set checks |
| `9217825` | golangci-lint v2.15.0: 0 issues |

## Unresolved

1. PR #29: should the rewritten spec set replace `docs/specs/`? Owner decision.
2. Export slice: fix the three medium findings before integrating.
3. web-ux, arch-guards, sec-ops-docs: rebase onto this branch, then review.
4. Open draft PRs #1–#30 stay open. After this branch merges, close those whose content is contained or superseded
   (the table above lists which).
