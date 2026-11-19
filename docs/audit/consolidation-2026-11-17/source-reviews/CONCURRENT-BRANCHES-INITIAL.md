# Supplemental branch audit: concurrent work discovered at final preflight

Pinned main: `78c695f202e4b9440a4d78ac3b4b181aa81cabac` (PR77 merge).
GitHub branches collection at inspection: **99**. All original **89** branches remain present. Of those original branches, only `main` has a changed head; the other 88 original heads are unchanged.

## New branches

| Branch / PR | Head | Merge base | Ahead / behind main | Classification |
|---|---|---|---|---|
| `claude/circula-consolidation-audit-0sydm1` / #78 | `bc9fc56c3ca33f6ef93df7e37410f944c1eb6147` | `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4` | 1 / 3 | Historical audit baseline, one file/49 lines |
| `claude/fnet-204-signed-cursors` / #80 | `0952c94a5dbab4044d7673ce9d51cec9063ab97f` | `2eacf7718f5bb3da3be5a288024af879076a9597` | 5 / 125 | Valuable future-only module work; keep isolated |
| `claude/ux-047-auth-catalogue` / #79 | `dd3e82e27ddb3d1cbf03460e6a5fc879f16691db` | `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4` | 1 / 3 | Valuable new core work; qualify before integration |

## PR79 content review

Changes migrate authentication/account-security text to 121 catalogue entries across nine templates and four handlers. The added tests exercise catalogue completeness, plural/grouped counts, uniform authentication refusal wording, rendered accessible labels, and escaping of hostile interpolated values. UX047 explicitly remains partial; no locale choice is made.

All four handler files (`h_account.go`, `h_auth.go`, `h_recover.go`, `h_security.go`) were compared with the base after substituting `messages.Text` calls with the literal catalogue values and normalizing concatenated literals/whitespace. Each comparison is identical. There is no changed service call, status code, throttle branch, CSRF path, redirect, or authorization logic in those handlers.

Template review retains endpoints, hidden next/challenge values, field names, autocomplete, CSRF includes, and html/template escaping. Dynamic strings remain plain strings; no trusted markup is introduced. Counted notices/recovery codes now use existing catalogue plural and grouping behavior. No ledger, database migration, dependency, workflow or future-module file is included.

Recommendation: preserve this unique core work. Obtain current-head CI, focused/full web and messages qualification and independent review, then merge the verified PR. Check ownership/latest head before mutation because another worker created this during the consolidation.

## PR80 isolation review

The commit adds signed-cursor viewer/filter binding, domain separation, expiry and configurable TTL, key IDs/rotation, `Page` wiring, regression tests and fuzz targets. Changes remain entirely under `future/feed/`. Its `feature.json` retains `enabled:false` and `integration:HOLD`; TTL and key custody remain owner decisions. It is a PR67 follow-up stacked on `chatgpt/feed-signed-cursors-20261116`. No core integration is recommended in this task.

## PR78 overlap review

The single `docs/audit/consolidation-completion-2026-11/BASELINE.md` document records another worker's start at main `ffa2758`. It overlaps the live consolidation inventory and is useful only as dated historical evidence. It must not replace the current final inventory. Its statement that all other branches match the old branch ledger omits the already-documented current PR39 source-tip mismatch. Keep provenance clear if incorporating it.

## Evidence and limits

`original-89-preservation.json` records current GitHub branch-preservation comparison. `supplemental-new-branches.json` includes complete commit IDs, ahead/behind counts, merge bases, changed paths and classifications. `ux047-full.diff` preserves the reviewed diff. Focused tests ran on an isolated git-archive snapshot of exact PR79 head `dd3e82e`, using Go 1.26.11, GOTOOLCHAIN=local and CGO_ENABLED=1: all four `Test.*PagesCatalogue` render tests passed (0.398 s); all five top-level tests in `internal/web/messages` passed (0.007 s), including catalogue completeness and authentication refusals. No full web, race or browser qualification is claimed. Root checkout, repository files, refs and remote state were not changed.

B confirmed PR78/79/80 share Claude Code session `session_01KtKj8n2B7N1ZkoQQdz85gS`; PR78 plans further SEC026/GOV021/UX047/FNET204 work. Keep active drafts intact unless coordinating integration; recent code should not be silently declared superseded.
