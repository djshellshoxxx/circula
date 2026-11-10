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

## Follow-up source-link check (same run)

Compared all 30 pinned GitHub blob links across the four wiki pages to the recursive exact-source tree (724 paths). Found one invalid security spec path (`docs/specs/09-security-privacy-abuse.md`) and corrected it to the existing `docs/specs/09-security-privacy.md` in commit `87d0afe63d5b60d1b917848eb4a17158470845ca`. The `docs/releases/` directory link was corrected from GitHub's blob URL form to the source-pinned tree URL in commit `e41d48f91c9972b8332fb42a213bb64f30904ef2`. Wiki-local links still target the four proposed page names. New documentation-branch CI runs may be pending; no passing claim until exact latest-head result is returned.

## Next hourly qualification and traceability review (2026-11-10)

The current source of truth remains unmerged PR #19 at `d98734cf1b2cc9a826397a6e43f7902de2304109`; `main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` is still prompt-only. Current stacked draft heads at this check: #21 `4c9f04cff60fd881734445dfa34d5c4d8f12c64d`, #22 `b8dcd058814155a9b5d44bbbdd2afb66ed4fa892`, #23 `9edbba110b89cc266bfc5358969f575ca352ff7d`.

**Fresh exact-head Actions:** all three draft heads have nine jobs in their latest workflow, of which **eight succeeded and the macOS-14 test job was cancelled**; the workflow conclusion is `cancelled`, not `success` and not a reported Go test assertion failure. Evidence: [PR #21 run 42819837307](https://github.com/djshellshoxxx/circula/actions/runs/42819837307), [PR #22 run 42819960398](https://github.com/djshellshoxxx/circula/actions/runs/42819960398), [PR #23 run 42820373582](https://github.com/djshellshoxxx/circula/actions/runs/42820373582). The cancelled macOS job's log download returned GitHub blob-not-found; cancellation cause remains **unverified**. Prior PR #19 exact-head green runs from November 4 are historical baseline evidence only. Do not mark the November 10 drafts or release gate fully passed. No local runtime tests were run by this documentation phase.

**Sample requirement-to-test source traceability (existence verified, not newly executed):**

| Requirement / concern | Pinned source | Pinned test / evidence | Limit |
|---|---|---|---|
| SEC-011 adjustment dual control | `internal/service/dualcontrol.go` | `TestDualControlProposerCannotApprove`, `TestDualControlTargetAdminCannotApprove`, `TestDualControlConcurrentApprove` in `internal/service/dualcontrol_test.go` | Production approval and policy settings remain owner-controlled |
| QA-034 / DEF-001 invitation return link | `internal/web/h_auth.go`, templates | `TestReturnLinksDecodeToTheOriginalPath` in `internal/web/next_link_test.go`; PR #19 historical CI | Separate terms-form 422 return-path defect remains a new draft fix |
| Governance terms retry navigation | `internal/web/h_gov.go:59-76` | New regression added in [draft PR #24](https://github.com/djshellshoxxx/circula/pull/24), first test-only SHA `22b80982813e8d51595f27b0b4a1aecbbc213d94` | No successful exact-head test or merged fix confirmed at this documentation checkpoint |
| QA-035 / UX-056 manual screen reader | `docs/qa/SCREEN-READER.md` | No verified human screen-reader evidence | Must remain explicitly manual, never inferred from axe |
| Release QA-039 | `docs/qa/RELEASE-GATE.md` | Historical examples and script-reported checks | Owner thresholds, licence, external review and deployment authorization remain open |

`docs/TRACEABILITY.md` explicitly identifies itself as a **2026-10-21 snapshot** at `925f637`; it cannot be treated as current completion evidence for PR #19. `docs/qa/RELEASE-GATE.md` also contains a historical sample and environment-specific failures; neither was silently rewritten. Wiki-local Home/Sidebar navigation still links both top-level User and Developer guides, with the prior zero-missing-target source checks. No live wiki publishing capability was used, and the live wiki is **not verified published**.

**Next documentation action:** after reviewing later PR #24 test/CI evidence, update only affected trace rows, verify the macOS cancellation independently (runner availability/concurrency/billing logs if accessible), and reconcile obsolete operator Known gaps individually against exact source. No release document should claim readiness until all manual and owner gates are satisfied.

This documentation activity was sequential; there was no separately spawned concurrent documentation worker.
