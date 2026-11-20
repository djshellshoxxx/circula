# Federation history and README composition — 2026-11-20

**Disabled; integration HOLD. Local documentation/history composition only.** The coordinator owns remote publication, retargeting and current-source provider qualification. No Core integration, feature activation, deployment or release is performed by this change.

## Exact inputs and normal merge

| Input | Commit | Role |
| --- | --- | --- |
| Actual isolated future target | `1b3c92e03007e35e3f190e719a1ea78a807d4102` | `future/fnet-209-federation-20261116`; observed target of [PR #105](https://github.com/djshellshoxxx/circula/pull/105) |
| Published signing repair | `059b0e4508a6c4f8e29f6a4cccd06f92b14ea115` | PR #105 head; common ancestor of the recovered service and README correction |
| Reviewed recovery source | `267aa7d08bbc41e89a429eec7a077c5c0b1f9778` | [PR #117](https://github.com/djshellshoxxx/circula/pull/117) head; complete tree `1160058f6a9a1c8a5a495449a73e0d0771cf04ea` |
| README acknowledgement | `d55554408ef591371768259905c70ca58d68facb` | [PR #106](https://github.com/djshellshoxxx/circula/pull/106) head; one-line correction acknowledging the dedicated module workflow |
| Normal merge | `d0feef13d9b3249617a73c5c748520b0c0b50dd9` | First parent is the reviewed recovery source; second parent is the README acknowledgement; tree `62b50eb42848c37c5d65e8dc911e32d11986a297` |

The remote PR inputs were read again at 2026-11-20 17:43 UTC and matched those pins. PR #117 still targeted `master/fnet209-utf8-20261119` at the signing-repair commit; PR #106 targeted the same branch. The actual future target is already an ancestor of the signing repair and the recovered service. It therefore needs no additional merge into this composition.

A read-only three-way preview used the signing repair as the merge base. It reported no conflict and only the README change. The subsequent normal merge confirmed that result. The intermediate README is exactly PR #106's blob `fa60487165f4229a67910c923ed14e90ab71ff24`. Both original histories remain reachable. No cherry-pick, squash, shared-history rewrite or branch deletion is used.

The following documentation commit reconciles the README against the recovered standalone service. It preserves PR #106's intent that the module's dedicated workflow be acknowledged, credits the already implemented v2 protocol/persistence/outbox/transport/interfaces, and states the actual host, operational and independent acceptance gates. It removes obsolete prototype claims that no third-party dependencies or persistence exist. The original archive README and all original recovery, signing, qualification and ownership receipts remain unchanged.

## Complete change boundary

Relative to the reviewed PR #117 source, the final composition changes only `future/federation/README.md` and adds this receipt. Every other tracked path retains its original Git blob and mode. This includes all eight production Go files, every test and fuzz assertion, dependencies and notices, the disabled/HOLD manifest, the module workflow, shared Core workflows, and the previously restored Core files.

| Preserved path | Exact Git blob |
| --- | --- |
| `future/federation/feature.json` | `547ebe612535c520a037fc3facb516159b863376` |
| `future/federation/go.mod` | `756f8c12174a36d8c9d329e705bacc3eca079db8` |
| `future/federation/go.sum` | `8fe352e67715300cad443dc9398fed14b4e7ee4b` |
| `.github/workflows/future-federation.yml` | `ba5cf2dec6e2947d3a2444c55133d713bc91616a` |
| `.github/workflows/ci.yml` | `3e864a60c771fe83d2a6016abd832ebdddf2e194` |
| `cmd/circula/auditanchor.go` | `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` |
| `cmd/circula/util.go` | `975b54f052508b5bdf8fa627ac4d80c9cf05366a` |
| `internal/service/departure_pagination_regression_test.go` | `3b117ecfe2f01e7e2363b9d201503cde6e0f5595` |
| `internal/service/erasure_test.go` | `e5e4b3131e887ea7535140b75224c506f656490d` |
| `internal/service/notification_erasure_test.go` | `683bee4899d66803e4f2b4193d47e492ea2baa08` |
| `internal/store/sqlite_erasure.go` | `fe452fc13c626aa70fdfa9df6d8fac26a068f796` |

Relative to the actual future target, the existing reviewed recovery source already changes the module, its dedicated workflow and the six Core paths listed above. This composition retains that complete scope. The audit-anchor import and departure-fixture repairs were recorded in [RECOVERY-20261119.md](RECOVERY-20261119.md); the notification redaction and regression restoration is retained in commit `bb8d113d3a8e80597f611417410e7d78151247e2`. No new production or workflow change is introduced here.

The manifest still declares `enabled: false`, `integration: HOLD`, and no dependency on another future feature. Its `documentation_handoff` and provider-status strings describe the earlier checkpoint and are intentionally preserved for provenance. This receipt records the later local incorporation of PR #106; it does not rewrite an earlier observation or claim that GitHub has merged that PR.

## Observed predecessor failures and qualification boundary

The actual logs for all five failed Core jobs in each of PR #105 and PR #106 were inspected. Both predecessors retain the old `cmd/circula/auditanchor.go` blob and fail to compile because `os` is not imported. Their Linux race jobs also fail while creating the 501-proposal departure fixture because its test cap is too low. The reviewed recovery source already contains the exact accepted corrections for both files, verified against accepted main snapshot `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.

| Predecessor | Observed run and evidence |
| --- | --- |
| PR #105 | [Core run 44513782075](https://github.com/djshellshoxxx/circula/actions/runs/44513782075): failed jobs `135524416644`, `135524417085`, `135524417127`, `135524417158`, `135524417189`; [race log](https://github.com/djshellshoxxx/circula/actions/runs/44513782075/job/135524417158) records both causes. |
| PR #106 | [Core run 44517329467](https://github.com/djshellshoxxx/circula/actions/runs/44517329467): failed jobs `135536630618`, `135536630629`, `135536630752`, `135536630835`, `135536630931`; [race log](https://github.com/djshellshoxxx/circula/actions/runs/44517329467/job/135536630931) records both causes. |

Their conditional Core native jobs were skipped. Their failing vet steps prevented golangci-lint and govulncheck from running; those skipped steps are not passes. Their reproducible-build failures also prevented the later local deployment-smoke step. The existing module successes and the later PR #117 input jobs qualify only their recorded source, not this new composition.

For this documentation composition, verification is limited to source/tree identity, ancestry and merge simulation, links, whitespace, and unchanged-configuration scanning. No new Go suite, race run, fuzz campaign, dependency check, browser session or external-peer test is claimed. Historical runtime evidence remains in [RECOVERY-QUALIFICATION-20261119.md](RECOVERY-QUALIFICATION-20261119.md).

Executed static checks verified all 26 links in the two changed documents, all 11 recorded preserved-file identities, the complete two-path change boundary and every other tracked blob/mode. The manifest and all four input ancestries passed their checks. The full directory scan with the unchanged repository Gitleaks configuration found zero findings. Existing workflow action-pin and repository-hygiene checks passed, as did whitespace checks. These results qualify the documentation composition only.

## Intended publication route and remaining gates

The resulting descendant can advance the existing PR #117 head by a guarded fast-forward and target `future/fnet-209-federation-20261116` directly. It retains the original target, PR #105, PR #106 and the recovered PR #117 source in its ancestry. Re-read the remote head and actual target before publication; any changed input requires renewed comparison. The existing module workflow already accepts the actual future target, so no workflow routing edit is required.

Before merging the composed PR into that isolated branch, require its own three native module jobs and every applicable existing Core job to succeed, with normal review requirements met. Report conditional Core native jobs explicitly if skipped. Do not infer qualification from the preceding composition or use the old failed predecessor sources as the final merge candidate. Preserve original branches and commit references until inclusion is verified.

Core integration and the independent acceptance gates in [README.md](README.md) and [INTEGRATION.md](INTEGRATION.md) remain on HOLD. The coordinator handles any remote PR disposition after verifying the final composition; no remote mutation accompanies this local work.
