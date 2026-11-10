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

## Follow-up documentation maintenance (2026-11-10, run-3)

- **Source pin remains:** application draft [PR #19](https://github.com/djshellshoxxx/circula/pull/19) `d98734cf1b2cc9a826397a6e43f7902de2304109`, unmerged; `main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7` has no runtime implementation.
- **Wiki-source update:** `docs/wiki/User-Guide.md` at commit `fe879807c0b4ae6cd907cf7268011babca9afdba` adds a focused recovery workflow supported by `internal/web/h_recover.go`, `internal/service/recovery.go` and `internal/web/recover_test.go`. Explains 10 single-use codes, one-time display, code rotation, recovery without implicit login, session/token revocation, role/network limits on admin reset, and explicitly marks the functionality draft/unreleased. It does not change technical or financial requirements.
- **Link check performed on updated branch:** fetched all four wiki source pages and checked **10 wiki-local Markdown links, 0 missing page targets**. Home and sidebar each link User Guide and Developer Guide. This is a source check; GitHub's live wiki has **not** been written to or verified published.
- **Traceability re-check:** `TestWebRecoveryFlow` and `TestWebAdminReset` exist in pinned `internal/web/recover_test.go`; tests were **reviewed, not executed** during this documentation phase. Existing PR #19 hosted baseline runs are historical, not proof of the new docs head.
- **Current development regression:** test-only PR #24 commit `22b80982813e8d51595f27b0b4a1aecbbc213d94` was proven red from [run 42829236389](https://github.com/djshellshoxxx/circula/actions/runs/42829236389): both Ubuntu and Windows Go tests failed at `terms_next_test.go:49`, with empty `next` instead of `/n/town/trades/new`. The overall run was cancelled after one macOS job cancellation. Minimal handler change was committed as `7620d7f51913363dd013b80cc682eb670eeff3a5` on the **separate** development PR #24. [Implementation run 42838956843](https://github.com/djshellshoxxx/circula/actions/runs/42838956843) was still in progress when inspected; **no green implementation claim**.
- **Unresolved:** finish exact-head check, continue source-pinned operator Known-gap verification, maintain manual QA-035/UX-056 and SEC-033/DEC-014 holds, and obtain explicit permission/tool support before publishing the GitHub wiki. No local Go, browser or accessibility tests were executed by this phase.

This was a distinct **sequential** documentation phase; no independent third ChatGPT worker/subagent was spawned, and no application or other worker's branch was modified.


## Follow-up CI evidence and wiki maintenance (2026-11-10, run-4)

- **Pinned application source:** unmerged PR #19 at `d98734cf1b2cc9a826397a6e43f7902de2304109`; prompt-only `main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7`. Only this isolated documentation branch was edited; no application source changed.
- **Development PR #24 exact-head qualification:** `7620d7f51913363dd013b80cc682eb670eeff3a5`, [run 42838956843](https://github.com/djshellshoxxx/circula/actions/runs/42838956843), completed overall **cancelled** with eight successes and the macOS-14 job cancelled. A targeted macOS job rerun was accepted but its second attempt also finished **cancelled**, with no steps reported and no retrievable job log. Root cause is unknown; do not mark this run green, red for code assertion, or release qualified. Ubuntu and Windows tests, e2e browser, lint/vuln, API contract/load, fuzz and reproducible builds did succeed.
- **Companion audit/docs/handoff latest prior heads:** [audit PR #22 run 42839227272](https://github.com/djshellshoxxx/circula/actions/runs/42839227272) at `53f9e8b589c04fc65ca87930561e669be8eac06a`, [docs PR #23 run 42839406303](https://github.com/djshellshoxxx/circula/actions/runs/42839406303) at `f8843f410515f20ade3026f3271d2dab78bdaead`, and [handoff PR #21 run 42839509766](https://github.com/djshellshoxxx/circula/actions/runs/42839509766) at `e7fde39c66ac389f4f2e45f5246214b673febb0b` each completed with eight successful jobs and one cancelled macOS job. Not full CI passes.
- **Developer wiki guide update:** `docs/wiki/Developer-Guide.md` at commit `f450a706d90112a3609f633b8355f43a3504a41e` now explains the exact-head CI ambiguity, successful test platforms, explicit macOS cancellation, and the need to retain all gates. This is an evidence and operator-information update only; no requirements/security/financial policies were changed.
- **Wiki navigation verification:** fetched all four pages on the isolated docs branch; examined ten wiki-local Markdown links, zero missing page targets. Home and sidebar both link **User Guide** and **Developer Guide**. The live GitHub wiki was **not published** and its content has not been verified.
- **Test status:** read GitHub-hosted CI metadata only; no local Go, browser, race, screen-reader or security tests were executed by the docs phase. The new docs-head CI, if triggered, must be checked separately before asserting acceptance.
- **Open gaps / next action:** determine why macOS jobs are cancelled before claiming a complete 9/9 qualification; reconcile historical operator Known gaps and traceability only against actual source/tests; retain DEC-014/A3-F02, SEC-033, manual QA-035/UX-056, licence, and owner release gates. A distinct sequential docs-maintenance phase was performed; **no concurrent third worker was spawned**.

## Follow-up documentation drift correction — 2026-11-10, fifth rotation

- **Source pinned:** application draft PR #19 at `d98734cf1b2cc9a826397a6e43f7902de2304109` (unmerged, prompt-only `main@4b422ad1e8578f3e36c5a38c68eafe81f6ee0aa7`). No runtime code or external owner's branch changed.
- **Source-supported correction:** `docs/ops/README.md` updated at commit `40bfedec13ea612c460a494cd1da4a3508ff9dbe`. Removed the obsolete statement that `cookie-secure` was not wired into the HTML adapter: `cmd/circula/serve.go:newHandlers` creates a shared `httpx.Edge` with `ForceSecure: cfg.CookieSecure` for both API and web, and `internal/web/session.go`, `internal/web/hostprefix_test.go:TestSecureDeploymentIgnoresPlainCookies`, `internal/httpx/hostprefix_test.go:TestHostPrefixOnlyWhenSecure` enforce secure-mode behavior. Also corrected the obsolete claim that secret file settings do not exist: `cmd/circula/config.go` defines `setup-secret-file`, `metrics-token-file`, `backup-key-file`, `mfa-secret-file` and `backup-encryption-key-file`; `docs/ops/configuration.md` documents them. Operator-password input remains env/stdin. These corrections are *documentation about an existing draft feature*, not new security guarantees or release approval.
- **Verification performed:** fetched all four source-controlled wiki pages from this docs branch and checked **10 wiki-local Markdown links, zero missing destinations**; Home and _Sidebar both link User Guide and Developer Guide. Confirmed `docs/ops/configuration.md#secrets` has a `## Secrets` anchor and both obsolete assertions are absent from the updated ops README. No newly run Go, browser, CLI, lint, or security tests. Live GitHub Wiki remains unmodified and unverified.
- **CI provenance:** at inspection, previous exact-head #21/#22/#23 runs [42850067014](https://github.com/djshellshoxxx/circula/actions/runs/42850067014), [42849655805](https://github.com/djshellshoxxx/circula/actions/runs/42849655805), [42849875256](https://github.com/djshellshoxxx/circula/actions/runs/42849875256) were each **overall cancelled** with 8 successful jobs and 1 cancelled `test (macos-14)` job (no steps). PR #24's two-attempt [42838956843](https://github.com/djshellshoxxx/circula/actions/runs/42838956843) also remained cancelled. New docs edits and this ledger commit have **no independently established exact-head pass** as of writing; inspect newly triggered runs at the next checkpoint.
- **Remaining gaps:** macOS runner cancellation diagnostics, owner financial/cryptographic decisions (including DEC-014/A3-F02), licence, manual accessibility (QA-035/UX-056), independent SEC-033 review, stale historical traceability/QA snapshots, and live wiki publication. Do not mark cancelled checks or manual gates as passing.
- **Concurrency:** no third concurrent worker/subagent was available; this was a distinct sequential documentation phase, not a separately running agent. No Codex or paid OpenAI API was used. Next docs pass: inspect source-pinned config/operations assertions and repeat wiki link checks after any source change.
