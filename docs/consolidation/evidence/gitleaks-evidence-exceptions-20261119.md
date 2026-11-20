# Exact exceptions for verified evidence false positives

Base source: `e0d27cc7cb73299219439919b260991b824d594a`.
Scanner: the existing CI pin, `github.com/zricethezav/gitleaks/v8@v8.28.0`.
Prepared on 2026-11-19. No runtime source, inventory content, or historical
execution evidence is changed by this repair.

## Reported failures and provenance

The [PR 99 quality job](https://github.com/djshellshoxxx/circula/actions/runs/44513300863/job/135522757126)
reported ten findings at source `0a73ee42b96255601cf2227b1ddf6599ec07a0ef`.
The [PR 102 quality job](https://github.com/djshellshoxxx/circula/actions/runs/44512804697/job/135521077519)
reported one finding at source `f29ebb01`. All eleven used `generic-api-key`.
The provider excerpts were redacted; the following classifications were checked
against the actual repository content, not inferred solely from the redactions.

| Original document | Observed lines | Count | Independent verification |
| --- | --- | ---: | --- |
| `docs/audits/evidence/post-snapshot-20261119/core-cursor-security-merged-a50f90b.execution.json` | 66, 67, 89, 94, 100 | 5 | Each recorded value exactly equals SHA-256 of the corresponding API source file at the base source above. |
| `docs/audits/evidence/post-snapshot-20261119/core-catalogue-traceability-merged-12af211.execution.json` | 36 | 1 | The recorded value exactly equals SHA-256 of `internal/web/messages/auth_test.go` at the base source above. |
| `docs/audits/BRANCH_INVENTORY_2026-11-19.json` | 1 | 4 | Each value in `dependency_node_shas` equals the corresponding recorded remote branch head and resolves to a local Git commit object. |
| `docs/consolidation/BRANCH_REFRESH_20261119.md` | 133 | 1 | The matched text is the ordinary phrase `CSRF/idempotency` in the catalogue scope table. |
| `docs/audits/evidence/closing-publication-20261119/remote-heads.json` | 18, 36, 65, 87 | 4 | The same four verified Git commit IDs appear in a later preserved head snapshot. |
| `docs/consolidation/evidence/shared-pages-receipt.json` | 102, 255, 298 | 3 | The first value equals the Git blob ID of `internal/web/h_auth.go` at the receipt's baseline; the other two equal SHA-256 of that source file at this repair's base. |

The last two rows were found by the subsequent whole-directory scans of both
this repair worktree and the clean integration source
`789f082a337eac18f9628976846536e6230f5975`. Both scans found the same seven
additional matches after the first eleven were addressed. These two documents
already exist in the base; the matches were not introduced by that later
integration. The shared-page receipt's baseline is
`40277310094ac16b34969fc3b002b2182983d8b5`, and its recorded source blob was
reproduced directly from that Git tree.

The five API paths are `internal/api/api.go`, `internal/api/auth.go`,
`internal/api/h_password.go`, `internal/api/h_tokens.go`, and
`internal/api/replay_auth.go`. The two execution records name unpublished local
source commits that are not present in this checkout; their six source digests
were independently reproduced from the available base source. No execution
claim is inferred from the cursor record, which itself records an interruption.

The four branch names are `chatgpt/circula-ledger-auth-guard-20261115`,
`claude/aud4-auth-fixes`, `claude/ux-047-auth-catalogue`, and
`dependabot/go_modules/github.com/getkin/kin-openapi-0.153.0`. The values remain in
their original evidence and in the exact allowlist expressions; they are not
credentials and are not replaced or redacted in those originals.

The first four documents are byte-identical to their respective reported
source heads and to this repair's base. The two additional documents are
byte-identical to both the base and the later integration source. The original
114-branch and later 128-branch consolidation inventories are also unchanged.
Their preserved bytes are recorded in the
[machine receipt](gitleaks-evidence-exceptions-20261119.json).

## Exception scope

Six additive `[[rules.allowlists]]` blocks in `.gitleaks.toml` use all of these
constraints:

- One `[[rules]]` block with `id = "generic-api-key"` scopes all six exceptions
  to the observed rule. Its default matcher, entropy, keywords and existing
  allowlists are inherited unchanged.
- `condition = "AND"` requires both the file path and the matched value.
- An anchored literal path names exactly one original document.
- `regexTarget = "secret"` and anchored literal values match only the verified
  nonsecret values, not surrounding text or an entire line. There are 13 distinct
  values in 17 path/value pairs, accounting for the 18 observed findings.

The existing two fixture allowlists and the default-rule extension are retained.
No rule is disabled, no documentation directory is skipped, and no general
hexadecimal, digest, Git-ID, or prose pattern is exempted.

The implementation was checked against the pinned upstream source, specifically
`config/config.go` (`parseAllowlist` and rule extension),
`config/allowlist.go` (predicate matching), and `detect/detect.go`
(`checkCommitOrPathAllowed` and `checkFindingAllowed`). An AND allowlist with a
value predicate does not skip a file merely because its path matches. The
generic rule's 3.5 entropy threshold and broad credential-label matching explain
the source-digest findings; its ordinary-word exemption does not cover the
slash-containing phrase.

The first draft used global `targetRules` allowlists. The real-scanner regression
reproduced all eleven findings both before and after that draft: in this pinned
version, default extension increments `extendDepth`, while target-rule allowlist
population is gated on depth zero. The implemented rule-local form uses the
existing extension merge directly, preserving the default rule while adding its
six allowlists. No scanner upgrade or source modification is needed.

## Regression checks

`scripts/check_gitleaks_config.py` uses the real scanner with temporary copies of
the six documents. It does not edit them. The test removes only the marked new
exceptions to construct the original configuration, and statically rejects
broader rule, path, value, target, or condition predicates.

| Real-scanner case | Required result |
| --- | --- |
| Original documents, original configuration | All 18 observed findings, with exact rule/path/line multiplicities. |
| Original documents, repaired configuration | Zero findings. |
| Replace the values in their original contexts with generated synthetic credentials | All 18 replacements detected, including 40-character and 64-character hexadecimal material. |
| Extend each original value by one character | All 18 changed values detected; literal-value anchors matter. |
| Prefix each exact document path | All 18 original matches detected outside the allowed paths. |
| Suffix each exact document path | All 18 original matches detected outside the allowed paths. |
| Add a generated generic credential and GitHub-token-shaped control beside a known match on the same line in every document | All 12 added controls detected under the actual default generic and GitHub rules. |
| Add a test-only rule matching the same verified nonsecret values | Six representative matches detected under that other rule; the exceptions affect only the targeted rule. |

Synthetic material is generated deterministically from public fixture labels at
runtime. No real credential is used or stored as a test fixture. Scanner reports
are redacted, and the harness prints only rule, path, line and count metadata on
failure. Configuration files and reports live outside each scanned input tree.

The existing CI secret-scan step runs this harness with the same pinned scanner
command before the unchanged whole-repository scan. This changes neither
workflow permissions nor event scope.

## Execution status

**All eight real-scanner controls passed** on 2026-11-19, in 25.294 seconds.
The [complete control output](gitleaks-evidence-controls-20261119.log) records the
18-to-zero baseline/fixed contrast and the expected detections in every negative
control. Python syntax and TOML parsing passed, and exact scope validation
reproduced the original configuration when the new marked section was removed.

Whole-directory scans with the repaired configuration returned exit zero and
zero findings for both the repair worktree (23.318 seconds) and the clean
integration source `789f082a337eac18f9628976846536e6230f5975` (25.714 seconds).
The latter checkout remained clean and at the same head before and after the
read-only scan; no root source files were edited. The machine receipt records
exact arguments, environment, configuration/source digests, scan heads and
timestamps. Those two directory scans preceded addition of this final receipt;
the added receipt files require their final scan before the repair is handed off.

The prebuilt scanner's `go version -m` output identifies the pinned module
version and checksum, built by Go 1.26.11. Its CLI version string is not injected
by `go install`, so the record uses module build information instead. No scanner
compilation or full Go package build was performed by this repair task.

This repair addresses verified false positives in directory scanning. It does
not claim that a different commit, Git history, external system, or unscanned
environment is free of credentials.
