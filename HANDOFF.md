# Circula continuation — 2026-10-28

Repository: `djshellshoxxx/circula`, checkout `/workspace/circula`.
Branch: `codex/circula-support-boundaries-20261028`.
Feature commit: `ae3d0928d077196de20b594bce6adaf555f290f9`.
Latest tested source: `5000a5e39dbfa224af4dd91123ca0b3a78e86a4c`.
Base: `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f` on
`claude/awesome-darwin-y6eaiz`, after merged PR #11. This is not main.
The later handoff checkpoint changes documentation only; use `git rev-parse HEAD`
for its exact SHA. Do not attribute source CI to this newer document-only head.

## Restart state and recovery refs

At handoff intake the worktree had no dirty or untracked files and local/remote
feature refs both matched the tested source above. This bounded pass updates only
handoff/evidence Markdown, then commits, normally pushes and verifies owned refs.
Run `git status --short --branch`, `git rev-parse HEAD` and `git ls-remote --heads
origin` before resuming; preserve any later edits or external authors' commits.
No other worktree or active worker was present; the coordinator alone owns this
handoff. No new implementation, worker, paid model call or expensive test rerun
was started. No paid Claude calls were made by this coordinator; remaining global
credits and the shared broker balance are not visible here. Keep a checkpointing
reserve rather than treating an unknown balance as available spending.

- Archive documentation head: `3d0080a9c43680f69f81217af3b42374e7160bf8`,
  verified on remote `checkpoint/circula-handoff-20261025`. Frozen archive source
  remains `59f7af9bcb39162c94756560178f71e439aef474`.
- Local `codex/circula-resume-20261024` remains the retired recovery ref
  `91851e0560c4c7a656968aa56a0f14ecec511d99`, three commits ahead of its remote
  `30d18a1626427fa6aa14f70c8df05f3e807efab5`. All three are reachable from the
  published archive; no unique work is unbacked. Do not advance that remote here:
  it is the base of externally owned PR #10.
- `work@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` is the old prompt baseline,
  not a current development branch. Missing `c134253` and `ee78a7f` remain
  unrecovered; nothing here recreates or claims to restore them.

The documentation commits use `[skip ci]` solely to avoid expensive reruns for
this handoff. Workflows and gates are unchanged. Skipped checks may remain pending
and cannot establish acceptance or permit a merge. Both PRs remain draft/unmerged.

## Preserved work and ownership

- Archive checkpoint `59f7af9bcb39162c94756560178f71e439aef474` is now published
  on `checkpoint/circula-handoff-20261025`; [PR #13](https://github.com/djshellshoxxx/circula/pull/13)
  is verified **draft**, unmerged, frozen source that exact SHA, base `work/spec-completion`
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

The installed compiler is `.tools/go1.26.10/go/bin/go`, official Go 1.26.10;
`go.mod` requires that version. The system `/usr/bin/go` is a board game, not
a compiler. The ignored toolchain is not backed up as source; provenance and
the verified official checksum are in `.tools/go1.26.10-provenance.json` and the
detailed [restart record](docs/audit/RESUME-2026-10-24.md). `/tmp` caches are
ephemeral. No network installation or denied download is authorized by this
handoff. Run the following from the repository root only with that compiler:

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
Draft [PR #14](https://github.com/djshellshoxxx/circula/pull/14) now exists and was
read back: draft/open/unmerged, source `5000a5e39dbfa224af4dd91123ca0b3a78e86a4c`
before this documentation-only receipt,
base `claude/awesome-darwin-y6eaiz` at the base SHA above. Creating draft source PRs
is within the current development request; submitting a GitHub approval review
requires specific per-PR user confirmation. No approval review was submitted.
The earlier requirement for confirmation merely to create this draft was an
overly broad interpretation and is superseded by this clarification.

Source-push CI [40713730369](https://github.com/djshellshoxxx/circula/actions/runs/40713730369)
completed successfully: all nine jobs at exact source `912598d...`, including all
OS tests, E2E, lint/vulnerability/secret checks, build/reproducibility, contract/load
and fuzz. Separate PR CI `40715371415` finished with eight successful jobs and a macOS failure in
`TestSecurityHeadersAndRequestID`. Its old substring check can reject a valid
random hex ID containing `bad`. Source `5000a5e...` corrects only that assertion,
with deterministic counterexample, predicate checks and independent review. See
[request-ID CI evidence](docs/audit/REQUEST-ID-CI-2026-10-28.md).

Final CI at corrected source `5000a5e39dbfa224af4dd91123ca0b3a78e86a4c`, read
back again for this handoff:

- [PR run 40718272756](https://github.com/djshellshoxxx/circula/actions/runs/40718272756):
  completed success, **9/9 jobs passed**, including all three OS tests/macOS race,
  E2E, static/vulnerability/secret checks, cross-build, reproducibility,
  contract/load and fuzz.
- [Push run 40718262628](https://github.com/djshellshoxxx/circula/actions/runs/40718262628):
  overall failure, **8 jobs passed, macOS cancelled while queued**. No assertion
  failure is reported for that cancelled job. This coordinator did not cancel,
  rerun or dispatch either workflow. Do not describe both workflows as green.

These remote results cover the tested source, not this newer documentation-only
head. Local full-package dependency and artifact access limits are unchanged.

The earlier documentation receipt was held while CI was active. After that run's
completed failure, source `5000a5e...` checkpointed the assertion correction and
receipt together. No running workflow was restarted or cancelled manually. No
merge has been attempted.
Recheck live ownership, full source/target SHAs and all required checks before
integration, including implications of the cancelled push job and skipped docs
checks. Record the proposed action and exact source/target before any merge.
The integration target remains `claude/awesome-darwin-y6eaiz@bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f`;
external PR #12 remains at the audit SHA recorded above. Neither branch was edited.

Ready next action: an incoming owner can read these receipts, reconcile live
required checks/review and external audit ownership, then decide integration.
No additional small accepted/unowned feature was identified in the last triage:
FR-054, FR-067 and UX-024 already exist; UX-047 needs an owner decision on initial
locale/catalogue scope. This handoff did not repeat broad scans or start it.
Manual screen-reader and independent professional review remain blocked on their
separate evidence; assistant delta reviews do not satisfy those gates. Formal
GitHub approval-review submission still requires specific per-PR confirmation.
Do not merge PR #13 over newer work. No force push, deployment, release, paid
Claude call, purchase or credentials/security-setting change is authorized here.

## Integration receipt — 2026-10-29

PR #14 (`codex/circula-support-boundaries-20261028@837d51b0d5374822b28f1b3a9206f9e2d23bd60c`)
was fast-forwarded into `claude/awesome-darwin-y6eaiz` (from `bb5a36f`). No source was
changed by the integration. Local gates at `837d51b`, in a cloud session where module
downloads work (official Go 1.26.10 via toolchain download):

- `gofmt -l .` empty; `go vet ./...` pass; golangci-lint v2.15.0: 0 issues;
  `go build -trimpath ./cmd/circula` pass.
- `go test -count=1 ./...`: all packages ok.
- `go test -count=200 -run '^TestSecurityHeadersAndRequestID$' ./internal/api`: ok.
- `make build` + Playwright (pinned 1.63.0, preinstalled Chromium): 103 passed.
- gitleaks v8.28.0 over `bb5a36f..837d51b`: no leaks; hygiene, action pins and
  `git diff --check bb5a36f`: pass.

External audit PR #12 (`claude/sharp-archimedes-tu76rh`, last read at
`fe111799ffabb2427d6d94feb32a037fa3ed2af8`) was not edited and shares no files with
PR #14. PR #13 stays draft and unmerged. The integration is in PR #10. Screen-reader
and independent professional review gates remain open.
