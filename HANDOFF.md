# Circula continuation — 2026-10-28

Branch: `codex/circula-support-boundaries-20261028`.
Feature commit: `ae3d0928d077196de20b594bce6adaf555f290f9`.
Base: `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f` on
`claude/awesome-darwin-y6eaiz`, after merged PR #11. This is not main.
The current HEAD adds this handoff; use `git rev-parse HEAD` for its exact SHA.

## Preserved work and ownership

- Archive checkpoint `59f7af9bcb39162c94756560178f71e439aef474` is now published
  on `checkpoint/circula-handoff-20261025`; [PR #13](https://github.com/djshellshoxxx/circula/pull/13)
  is verified **draft**, unmerged, head that exact SHA, base `work/spec-completion`
  at `d39bfef12d4ac1430815fa49ad799fb1c4bf9964`. It discloses all unexecuted gates.
  Its later CI now confirms C-09's unchanged ETag and 409-versus-412 failure;
  lint also flags staticcheck QF1003 in the candidate test. E2E and the full
  pinned secret-scan step passed. Keep the archive draft and unmerged.
- Archive and current base diverge from `30d18a1626427fa6aa14f70c8df05f3e807efab5`:
  five archive-only commits and ten base-only commits. Neither was overwritten.
- PR #11 already implements historical balances and preserves the independent
  approver/24-hour rules. Do not redo those changes.
- Active external [audit PR #12](https://github.com/djshellshoxxx/circula/pull/12),
  last read at `8f4e7709267ac4de70a650b4f7ce114a9796c797`, owns the broad audit
  and A2-07 pending-policy ETags. Local C-09 stays preserved in PR #13, not ported
  into this feature branch. No duplicate audit or fix was started.

## New accepted slice: GOV-027

Existing Help now explains support responsibilities and is linked from the admin
home. It distinguishes operator procedures from incomplete technical enforcement
and acknowledges host operators' ability to read stored data. Spec 11 has the
matching support-boundary summary. No new route, handler, permission, ledger,
threshold, credential, financial policy or dependency was added.

One worker implemented; an independent assistant reviewed the delta and resolved
a copy blocker. It is not a professional security audit. Detailed evidence:
[GOV-027 record](docs/audit/GOV-027-2026-10-28.md).

## Tests and limits

```sh
PATH="$PWD/.tools/go1.26.10/go/bin:$PATH" GO111MODULE=off GOTOOLCHAIN=local \
GOPROXY=off GOSUMDB=off GOCACHE=/tmp/circula-go-cache \
go test -count=1 internal/web/support_boundaries_template_test.go
```

Actual template tests failed before implementation and pass now. Running the final
test against pinned original templates also fails both cases. Formatting, diff
whitespace, hygiene and workflow pins pass. This standalone test compiles only
standard-library code; it does not prove full application or browser integration.
Full web/backend/native tests and browser runs remain unavailable locally because
module caches are absent and prior dependency/artifact HTTP 403 denials remain
binding. No denied downloads were retried. Screen-reader and independent review
gates remain open. The broad code/security audit is still owned and in progress
in PR #12; its snapshot excludes this new delta.

## Next action and publication boundary

Verify source publication with `git ls-remote --heads origin
refs/heads/codex/circula-support-boundaries-20261028`; never assume a push succeeded.
No feature PR or merge has been attempted. **A new draft PR requires specific user
confirmation** under the renewed instructions. Intended review target, if approved:
`claude/awesome-darwin-y6eaiz`, last read at the base SHA above. Recheck live ownership,
target/source SHAs, CI and required reviews before proposing any merge; do not
merge PR #13 over newer work. No force push, deployment, release, paid Claude call,
purchase or credentials/security-setting change is authorized here.
