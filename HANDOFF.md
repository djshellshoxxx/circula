# Archive restart receipt — 2026-10-28

Repository: `djshellshoxxx/circula`, checkout `/workspace/circula`.
Owned archive branch: `checkpoint/circula-handoff-20261025`.
Frozen source/test/handoff checkpoint: `59f7af9bcb39162c94756560178f71e439aef474`.
This later receipt changes documentation only; resolve its exact commit with
`git rev-parse HEAD`. Before this receipt the worktree was clean with no untracked
files, and local/remote archive refs both matched the frozen checkpoint.

The exact owner approval resolved the historical publication denial for that
checkpoint: it was normally pushed and verified, and [PR #13](https://github.com/djshellshoxxx/circula/pull/13)
is draft/open/unmerged into `work/spec-completion` at
`d39bfef12d4ac1430815fa49ad799fb1c4bf9964`. The older local-only and approval-gate
statements below describe their authoring time, not the current publication state.
This documentation backup is authorized by the later branch-handoff instruction.

Later CI provides genuine failing-before evidence at the frozen source:
[run 40711327156](https://github.com/djshellshoxxx/circula/actions/runs/40711327156)
has five successful and four failed jobs. All three OS tests fail C-09: the
currency-NAME-only pending update retains the ETag and stale `If-Match` returns
409 instead of the asserted 412. Lint also reports staticcheck QF1003 at
`internal/api/more_test.go:205`. Formatting, vet and the pinned secret-scan step
passed; govulncheck was skipped after lint failure. E2E, cross-build,
reproducibility, contract/load and fuzz jobs passed. No production fix is claimed.
Original assertions, financial values, thresholds and dual-control rules remain.

This archive is **not an integration candidate**. The newer integrated branch
`claude/awesome-darwin-y6eaiz@bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f`
already contains historical balances and the independent-founder fixture.
External [PR #12](https://github.com/djshellshoxxx/circula/pull/12) owns broad audit
and A2-07/C-09 work, last verified at
`8f4e7709267ac4de70a650b4f7ce114a9796c797`. Preserve that ownership. Current
coordinator work is separately backed up on
`codex/circula-support-boundaries-20261028`, source
`5000a5e39dbfa224af4dd91123ca0b3a78e86a4c`, [draft PR #14](https://github.com/djshellshoxxx/circula/pull/14).
Its own handoff carries current acceptance evidence and next steps.

Local `codex/circula-resume-20261024@91851e0560c4c7a656968aa56a0f14ecec511d99`
is a retired recovery ref, fully reachable from this published archive. Its remote
remains `30d18a1626427fa6aa14f70c8df05f3e807efab5`, the base of externally owned
PR #10; do not advance it during this handoff or overwrite unique work. Missing
historical `c134253`/`ee78a7f` remain unrecovered.

Next action here is preservation, not implementation or merge. Before resuming,
read this receipt and the detailed records below, inspect status and live remote
refs, and reconcile with PR #12's owner. Local package/browser/full-scan gates
remain blocked by uncached dependencies and the existing dependency/artifact
HTTP 403 denials; no denied route was retried. Toolchain prerequisites and exact
commands/results remain in the detailed restart record. Manual screen-reader and
independent professional review remain open. No worker or paid Claude call was
started for this handoff; remaining credit/broker balance is not available here.

The documentation-only commit uses `[skip ci]` to honor the bounded handoff's
no-expensive-rerun instruction. No workflows or gates are changed; skipped checks
may remain pending and do not establish acceptance or permit a merge. No approval
review, merge, release, deployment or credential/security change was performed.

---

# Circula handoff — 2026-10-25 (historical)

**LOCAL ONLY.** Branch: `checkpoint/circula-handoff-20261025`.
Repository: `djshellshoxxx/circula`; integration owner: this session's coordinator.
One existing inexpensive worker prepared the regression. No new team was created.

## Exact checkpoints

- Source/test payload: `763603fdbfb4a4718923de555e603dec187b5019`.
  Current HEAD adds handoff documentation: resolve with `git rev-parse HEAD`.
- Preserved local base: `91851e0560c4c7a656968aa56a0f14ecec511d99`, still on
  `codex/circula-resume-20261024` and an ancestor of this branch.
- Last verified public work branch: `30d18a1626427fa6aa14f70c8df05f3e807efab5`.
- Last verified integration target `work/spec-completion`:
  `d39bfef12d4ac1430815fa49ad799fb1c4bf9964`.

## Completed, current and remaining

Earlier specs/MFA work is published. The local independent-founder approval
fixture, software-artwork descriptor and all earlier handoffs are preserved.
Current C-09 work adds a fixed-clock regression for a pending currency-NAME-only
policy change: changed strong ETag, stale `If-Match` 412, unchanged active/pending
state. No production fix, amounts, thresholds, financial policy or roles changed.

C-09 remains a candidate, with **no runtime failing-before evidence**. The focused
`GOPROXY=off` API command failed dependency setup before compilation. Formatting,
whitespace, hygiene and preservation checks passed. Broader/browser checks remain
unexecuted locally; existing HTTP 403 denials were not retried or bypassed.
Published CI results do not validate these local commits.
The seven-path outgoing diff passed a limited credential-pattern/private-data
review; no research packs or tools are outgoing. Full Gitleaks remains unexecuted.

## Blocked publication and merge decision

Automatic review denied draft-PR creation from `codex/circula-resume-20261024` into
`work/spec-completion`, citing the earlier publication restriction. A generic
checkpoint request does not resolve that rejection. No remote write was retried.

Exact approval needed: publish source plus handoffs on the named checkpoint
branch and, if intended, create a draft PR into `work/spec-completion`, explicitly
disclosing unexecuted gates. **Leave unmerged:** the source and target above lack
required runtime/browser validation. No merge was planned or attempted. Record
full live source/target SHAs, ownership, tests and intended action before any
later merge. No credentials/security changes, private research-pack uploads,
funds, purchases, destructive resets, deployments or releases.

## Recovery and next range

Read [the detailed restart record](docs/audit/RESUME-2026-10-24.md),
[the source-pinned C-09 packet and exact command/output](docs/audit/C-09-POLICY-ETAG-2026-10-25.md),
and [CONTRIBUTING.md](CONTRIBUTING.md). Check `git status` and preserve all local
commits; do not reset to old main. Historical missing checkpoints remain missing.

When dependencies are lawfully available, run the focused C-09 test unchanged
before editing production. Next range: `internal/api/h_policy.go` and policy
scheduling in `internal/service/networks.go`. Separately validate the preserved
founder browser fixture and [artifact evidence](docs/qa/evidence/ci-40103567118-dual-control.md).
Resolve the exact publication approval before any push, PR/comment, merge or CI
trigger; then inspect outgoing data, required gates and actual remote SHAs.
Recheck workflow triggers and side effects before an authorized push.
Pause new feature batches while preparing the handoff.
