# PR129 guarded integration and Core exact-head qualification — 2026-11-21 UTC

This is a later, additive observation. The dated 2026-11-20 qualification and original failure/success records at evidence commit `80e3a58860f2f5533787585e3af241b9c0207ad9` remain unchanged. This report does **not** claim Core/main integration.

## Repair qualification (exact published source)

- PR: [#129](https://github.com/djshellshoxxx/circula/pull/129), `repair/core127-e2e-observer-20261120` into `integration/unique-core-completion-20261120`
- Source SHA: `11739c7f8f59455b4dc480294cab4d79f7933cb9`; source tree: `43e65ee91a90ff70b70cc69d0b517b8367a2585f`
- Exact-head provider [CI run 44838129155](https://github.com/djshellshoxxx/circula/actions/runs/44838129155), attempt 1: completed, **success**. All **seven applicable** jobs completed successfully and report the same source SHA. Conditional Core-native placeholder job `136639528278` was **skipped** and is not a pass.
- Successful jobs: build release targets `136639526238`; reproducibility `136639526519`; Ubuntu Go race tests `136639526601`; API contract/load `136639526703`; browser/Playwright/axe/keyboard `136639526717`; vet/gofmt/lint/vulnerability `136639526727`; fuzz smoke `136639526729`.
- Independent production/observer and CI-workflow source reviews, local negative control, prior browser provider execution, and the original Linux timeout failure remain recorded in the [loading follow-up README](README.md) and [continuation report](../../LOADING_AND_EVIDENCE_CONTINUATION_20261120.md). GitHub PR review submissions were absent at the observed check; repository rulesets were empty, branch protection was not independently accessible to the installed integration (HTTP 403). GitHub indicated a clean/mergeable pull request. The independent reviews were documentary reviews, not falsely counted as submitted PR approvals.
- The complete PR source delta to the unchanged Core target had two paths only: `e2e/tests/loading.spec.js` and `.github/workflows/ci.yml`. Compared source `11739c7…` against `bea5de25…`: ahead 2, behind 0, no other changed paths.
- Confirmed `delete_branch_on_merge=false` before the merge; no force push, release, deployment or source branch deletion was requested.

## Actual merge and independent readback

- Normal `merge` of PR129 used `expected_head_sha=11739c7f8f59455b4dc480294cab4d79f7933cb9`. GitHub returned successful merge commit `3ef43ac1e05360d2a1641742dc056afe8871879b`.
- Read actual Git commit: **ordered parents** `bea5de25cf76b66d87689034e2529f9b40c6dff6` (target) and `11739c7f8f59455b4dc480294cab4d79f7933cb9` (source). **Merge tree** `43e65ee91a90ff70b70cc69d0b517b8367a2585f`, exactly equal to the independently reviewed repair source tree. The actual merge changed only the two reviewed paths.
- Fetched actual Core target ref `integration/unique-core-completion-20261120` = `3ef43ac1e05360d2a1641742dc056afe8871879b`. The original repair source ref remains `11739c7f8f59455b4dc480294cab4d79f7933cb9`. PR129 is closed/merged; PR127 remains draft/open.
- Accepted `main` remains `aa01563fcc891da60fefebb8335eca27855678e8` at this observation; no main merge was performed.

## Fresh Core PR127 provider gate

The new exact source of [Core PR127](https://github.com/djshellshoxxx/circula/pull/127) is merge SHA `3ef43ac1e05360d2a1641742dc056afe8871879b`. Its base is unchanged main `aa01563fcc891da60fefebb8335eca27855678e8`. `ci:cross-platform` is present.

- [Shared Core run 44843655337](https://github.com/djshellshoxxx/circula/actions/runs/44843655337): **nine** individually enumerated jobs queued, each exact `3ef43ac…` SHA at observation. Matrix includes Ubuntu/macOS/Windows Go, browser, API/load, fuzz, lint/vulnerability, build and reproducibility.
- [Native reader run 44843655247](https://github.com/djshellshoxxx/circula/actions/runs/44843655247): **three** native Python-reader jobs (Ubuntu/macOS/Windows) queued at the same SHA.
- **Zero new Core qualification successes claimed** while queued. The previously successful repair-subset run does not substitute for the Core exact-head 9+3 gate. Older Core runs, historical skips and failures remain unchanged and source-specific.
- Before considering main: independently refresh refs, all twelve individual job outcomes, review/rules and merge simulation. If any exact-head job fails, review its complete log and correct only the observed accepted-scope defect, then requalify changed source. Keep PR127 draft until required gates and review are satisfied.

## Scope and preservation

This merge did not activate any future module, import future-module code or alter future manifests (only the two reviewed test/workflow paths changed). The previously verified independent future-module integrations are unaffected. The fifteen future manifests remain recorded as `enabled:false`/`integration:HOLD` in the preserved evidence; recheck their live values after actual main integration before declaring completed Core integration. Proof-based older-Core PR reconciliation (#90/#102/#109/#112/#125/#128, superseded #111 and separately recovered documentary #104) is still conditional on qualified **actual-main** integration. Policy holds and all future modules' host/production acceptance remain excluded. No release or deployment.
