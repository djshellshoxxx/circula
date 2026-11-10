# Circula ChatGPT docs maintenance — 2026-11-10

## Source and ownership
- Application source: PR #19, exact SHA `d98734cf1b2cc9a826397a6e43f7902de2304109` (`claude/happy-noether-444cum`), open draft; default `main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` contains prompts only.
- Proposed coordination spec: PR #20 `3d9c56196625fc0c158b4652cbe2b9716c7af97b`, draft, unmerged.
- Isolated documentation branch: `chatgpt/zero-codex/docs-20261110-a` targeting the PR #19 branch; **does not modify Claude-owned source**.
- Wiki content completed through documentation commit `e2bfe76d718909dbfd429f60696909719e734575` (the ledger commit follows this SHA).

## Changes
- Created `docs/wiki/Home.md` and `docs/wiki/_Sidebar.md`, both with prominent, working User Guide and Developer Guide navigation.
- Created `docs/wiki/User-Guide.md` with overview/status, getting started, installation/access, trades, balances, disputes, troubleshooting/FAQs, privacy/security limits and release-stage distinction.
- Created `docs/wiki/Developer-Guide.md` with architecture/module map, Go/Node prerequisites, local setup, CLI/build/test commands, API/contract links, security/contribution rules, CI, debugging, release procedure, specs/decisions and unqualified work.
- Avoided duplicating detailed canonical guides; links resolve to source-pinned application documentation rather than prompt-only `main`.

## Verification performed
- Source paths were inspected via direct GitHub connector: README, `docs/guides/user-guide.md`, `docs/guides/admin-guide.md`, `docs/dev/SERVICE.md`, `docs/ops`, `docs/qa/E2E.md`, `docs/qa/RELEASE-GATE.md`, `docs/specs/README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `Makefile` and CI metadata.
- Wiki internal link-target check: **Home, User-Guide, Developer-Guide and _Sidebar have zero missing wiki-local targets**; check uses GitHub wiki page-name form without `.md` extension.
- Historical CI at PR #19 exact head: runs [41806043004](https://github.com/djshellshoxxx/circula/actions/runs/41806043004) and [41806032535](https://github.com/djshellshoxxx/circula/actions/runs/41806032535), each 9 jobs successful.
- **No local Go/npm/browser tests were run for this docs-only change**; remote wiki publication/link behavior was **not** verified.

## Important drift and remaining work
- Existing `docs/ops/README.md` Known gaps and `docs/qa/E2E.md` contain historical snapshots that may disagree with currently implemented runtime features. Reconcile case by case against current source/CI before editing, not by copying a newer unverified status.
- The current source has no `docs/wiki/` until this PR; the live GitHub Wiki **was not modified** because this connector does not expose direct wiki-write capability. Publisher must create/update Home, User-Guide, Developer-Guide and _Sidebar using the wiki repository/UI, then confirm links. No publication claim.
- Source branch is an unmerged draft and may move. Update pinned URLs only after re-reading exact head and relevant tests. Do not describe it as released.
- Requirement-to-test traceability remains limited by historical `docs/TRACEABILITY.md`; next docs pass should reconcile individual high-priority rows and cite exact test methods.
- Security contact placeholders and release/financial/cryptographic/legal choices are **owner-gated**; do not fill them speculatively.

## Next action
Review stacked draft PR and its CI, publish live wiki only with separately authorized wiki write capability, verify navigation, then reconcile the highest-impact operator/QA documentation drift and release-gate evidence without interfering with active owner work.

No separate documentation subagent was available. This was a third **sequential** maintenance phase, not a concurrent worker.
