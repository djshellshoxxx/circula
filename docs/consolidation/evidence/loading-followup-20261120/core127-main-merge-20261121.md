# Circula Core PR #127 — actual main integration receipt (2026-11-21 UTC)

This documentary-only receipt supplements the frozen historical checkpoints; it does not modify qualified runtime source, retry historical failures or declare all future modules or held policy decisions complete.

## Exact-head pre-merge qualification

- Core source: `3ef43ac1e05360d2a1641742dc056afe8871879b`, whole tree `43e65ee91a90ff70b70cc69d0b517b8367a2585f`.
- Accepted pre-merge `main`: `aa01563fcc891da60fefebb8335eca27855678e8`.
- Run [44843655337](https://github.com/djshellshoxxx/circula/actions/runs/44843655337): **9/9 applicable jobs completed successfully** at the exact Core source: Linux/macOS/Windows Go; Playwright/axe/keyboard; API/load; fuzz; lint/vulnerability; build targets; reproducible build.
- Run [44843655247](https://github.com/djshellshoxxx/circula/actions/runs/44843655247): **3/3 native Python-reader jobs completed successfully** on Linux/macOS/Windows, same exact source.
- PR [#129](https://github.com/djshellshoxxx/circula/pull/129) had previously merged as `3ef43ac1e05360d2a1641742dc056afe8871879b`. Its source branch `11739c7f8f59455b4dc480294cab4d79f7933cb9` remains preserved.
- The target and source were refreshed, PR127 was marked ready only after the exact-source gate was satisfied. GitHub reported a clean merge; the repository's `delete_branch_on_merge` was false, merge commits allowed, and the public active ruleset list empty. Branch protection could not be read through the integration (403); GitHub's normal merge endpoint nevertheless accepted the guarded merge. The independent reviews and earlier failure logs remain in the prior evidence package.

## Actual normal merge and readback

- [PR #127](https://github.com/djshellshoxxx/circula/pull/127): GitHub returned normal expected-head merge commit `87651d36558a8e421e41fbd76c514045d88c74ed` (no force, squash or rebase).
- Ordered actual parents: (1) `aa01563fcc891da60fefebb8335eca27855678e8`; (2) `3ef43ac1e05360d2a1641742dc056afe8871879b`.
- Actual merge tree: `43e65ee91a90ff70b70cc69d0b517b8367a2585f`, **exactly equal** to the reviewed Core source tree and GitHub's prematched merge simulation.
- Readback `refs/heads/main` equals the merge SHA; preserved Core source `refs/heads/integration/unique-core-completion-20261120` remains `3ef43ac1e05360d2a1641742dc056afe8871879b`; repair branch remains `11739c7f8f59455b4dc480294cab4d79f7933cb9`. All original commit ancestry remains reachable.
- Full actual main tree was independently inspected (2,140 entries, nontruncated). It contains **no `future/`, `features/`, `modules/`, `internal/future/` or `cmd/future/` paths**, so no future module implementation was imported by this merge.
- Independently fetched each of the **15 actual target branches** and their corresponding `future/<module>/feature.json` blobs. All 15 contain `"enabled": false` and `"integration": "HOLD"`. No activation/deployment occurred.

## Post-merge provider verification still required

The actual main commit triggered three **push** workflow runs, all on `87651d36558a8e421e41fbd76c514045d88c74ed`:

- [Core CI #44856110228](https://github.com/djshellshoxxx/circula/actions/runs/44856110228) — nine jobs, **queued at the observation**.
- [Native export-reader #44856110230](https://github.com/djshellshoxxx/circula/actions/runs/44856110230) — three jobs, **queued**.
- [Scorecard #44856110197](https://github.com/djshellshoxxx/circula/actions/runs/44856110197) — one job, **queued**.

These runs have not been declared successful here. The merge is verified; post-merge main qualification and historical PR reconciliation remain open until actual-head provider outcomes are known. Preserve all historical failures/skips.

## Historical ancestry and conditional reconciliation

Live original heads for PRs #90, #102, #109, #112, #125 and #128 are unchanged from the reviewed containment proof. Actual main is a descendant of **all six**: exact GitHub compare status `ahead` with `behind_by=0` and merge-base equal to each original source SHA. GitHub automatically marked #90, #102, #125 and #128 merged/closed in response to main integration; #109 and #112 remain open (their targets are historical intermediates). #128 ancestry preserves original audit evidence but does **not** mean every audit finding is fixed. Inspect individual PR states and their branches again before any manual closure.

PR #111 (`45356ca10e1ecd6e6441e0171910dd918b36f5d9`) and #104 (`bcdc569cbc483761bf2d0e2d7594636ac1713d7d`) are **diverged**, not ancestors. Their reviewed alternative-fixture/runtime supersession and the complete 102-document PR104 historical archive are preserved on this evidence branch, not wholesale merged. Do not claim whole-tree inclusion. Reconfirm source and policy of supersession before closing after post-merge qualification.

The five held policy PRs #83/#85/#86/#94/#107, seven policy/interface branches, OPS-04/03, LED-R02, GOV-021, AUD-4F-02, SEC-026, all-conflicted/system-subject escalation, CLI competition and DEC-006 remain held. Future-module activation, adapters and production/device/peer acceptance remain outside this finalization scope.
