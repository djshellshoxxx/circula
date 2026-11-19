# PR #88 documentation review

Reviewed 2026-11-18, read-only. Complete three-file diff inspected; no Go builds,
tests, probes, source edits or remote mutations performed.

## Exact revision and author intent

- PR: https://github.com/djshellshoxxx/circula/pull/88
- Head: `f4cbaddd4d40f690529ab33c88f302be9d0a1752`
- Tree: `63656a9e0696fb2f30b756c50a9c38261e9c6288`
- Branch: `nodata/doc-sweep-20261118`
- Base and sole parent: `b4398074ae4ca72559235ca50242cfcb7a934eec` (`main`).
- One commit ahead, zero behind that exact base; 3 insertions / 4 deletions.
- API snapshot: open, draft, not merged, mergeable true; no discussion comments.
- Author account: `djshellshoxxx`; commit author `Claude`; session
  `https://claude.ai/code/session_01KtKj8mSbGMEnwFJH6PMZrC`.
- PR body explicitly limits this to three code-backed documentation updates,
  flags the old wiki links, and says PR/branch states were not checked. It
  deliberately leaves the owner-controlled DEC-014 record unchanged.

Raw normalized API snapshots: `PR88-API-EVIDENCE.json`.

## Complete change assessment

| File / changed line | Claim | Source verification on exact base |
|---|---|---|
| `HANDOFF.md:20` | E-02 and NEW-01 are fixed; A3-F02 and PRIV-01..05 remain open | Matches `docs/security/security-review.md:85–96`; normal/raw store open paths set `trusted_schema(0)` at `internal/store/sqlite.go:109` and `inspect.go:77`; backup hardening already present |
| `docs/TODO.md:73` | NEW-01 staging uses a leased private `os.MkdirTemp` directory and exclusive copy creation | `cmd/circula/backup_work.go:14–31` creates and leases the directory; `snapshot.go:205,244` uses it; `util.go:24` opens copies with `os.O_CREATE|os.O_EXCL|os.O_WRONLY` and mode 0600 |
| `docs/wiki/Developer-Guide.md:117` | Native CI runs on PRs carrying `ci:cross-platform` | `.github/workflows/ci.yml:109–137` has the label predicate and native matrix; `docs/ops/ci.md:9–17` explains that adding the label alone does not trigger a run |

The three new statements describe already merged behavior. They do not depend
on unmerged PR #82, #84 or #87, introduce future-module claims, claim release
qualification, or resolve DEC-014. The concise NEW-01 statement is supported;
its `O_EXCL` implementation is in `util.go`, with fuller provenance in the
linked security-review table. `git diff --check` passes for the exact diff.

## Required documentation correction

`docs/wiki/Developer-Guide.md:114–117`: the newly updated CI claim still cites
`claude/cool-gauss-vdxlod` and the page banner says the instructions describe
that branch. The branch exists at `64fa4f685780b93a349a862e3741df5b4db067cc`,
whose workflow at lines 81–83 unconditionally excludes PRs from `test-cross`.
Point the CI workflow link to `main` (or the verified `b4398074...` revision)
and explicitly identify this CI subsection as current-main guidance. This is
a small citation/scope correction, not a runtime or policy blocker; it does
not require rewriting every older wiki link in this PR.

Suggested sentence: "The current `main` branch's
[`ci.yml`](https://github.com/djshellshoxxx/circula/blob/main/.github/workflows/ci.yml)
defines:". The label sentence can retain its existing pointer to
`docs/ops/ci.md`, which already explains the next-event requirement.

## Overlap and disposition

- PR #78 API head at review is
  `bcacd848dea66cac8b7909aa1e9e9dac7fdf84e4`; its eight added files are confined
  to `docs/audit/consolidation-completion-2026-11/`. It has no file overlap with
  #88 and is authored by a different Claude session. No coordination comments
  were present on #88. Treat #88 as active useful work, not as obsolete.
- The root's `final-evidence-publish` working tree currently changes
  `.github/workflows/ci.yml`, `docs/security/security-review.md` and adds its
  separate evidence directory. It has not changed #88's three files. #88's
  factual corrections therefore remain unique document improvements.
- The surrounding old `HANDOFF.md` inventory still calls its pre-#69 snapshot
  current and says nothing has merged. Those statements are pre-existing,
  explicitly outside this PR author's verified scope, and should be reconciled
  by the root's final evidence publication, using the actual ending inventory.
  Do not expand this small PR into a competing consolidation ledger.
- Preserve all three changes, correct the CI citation, and retain the existing
  owner-policy holds. Normal author draft/readiness and exact-head CI decisions
  remain parent-owned; this source review does not claim they passed. If these
  lines are folded into the final evidence PR instead, only supersede #88 after
  the replacement is committed and its exact files/commit are documented.

Disposition: valuable documentation-only PR; compatible with the reviewed
core consolidation after the narrow CI source-link correction.
