# Bounded continuation and handoff

This record preserves the user's authorized master consolidation. It permits reviewed repairs and normal expected-head merges when applicable checks and approvals pass. It does not permit deployment, release, branch deletion, shared force pushes, protection bypasses, copying unfinished external work or future-module activation.

## Published candidates and ownership

The exact source heads, whole trees, original target baselines and actual commit parents are in [PUBLICATIONS.json](PUBLICATIONS.json). Refresh each live PR before acting.

| PR | Owned scope | Target |
| --- | --- | --- |
| [104](https://github.com/djshellshoxxx/circula/pull/104) | Core Go/Python export verification, dedicated native Python workflow, this audit directory | main |
| [97](https://github.com/djshellshoxxx/circula/pull/97) | Recurring calendar repair and dedicated workflow | isolated recurring target |
| [118](https://github.com/djshellshoxxx/circula/pull/118) | Complete recovered recurring module, chronology and parser repairs, preserved old tests | our #97 source branch |
| [103](https://github.com/djshellshoxxx/circula/pull/103) | Web Push/SMTP transport repairs and regressions | isolated notifications target |
| [105](https://github.com/djshellshoxxx/circula/pull/105) | Federation signing-input repair and dedicated workflow | isolated federation target |
| [117](https://github.com/djshellshoxxx/circula/pull/117) | Complete recovered federation module with both protocol test families | our #105 source branch |
| [108](https://github.com/djshellshoxxx/circula/pull/108) | Recovered analytics | isolated analytics target |
| [113](https://github.com/djshellshoxxx/circula/pull/113) | Recovered receipts plus exact typed-number fidelity | isolated receipts target |
| [115](https://github.com/djshellshoxxx/circula/pull/115) | Recovered areas plus strict JSON name equivalence | isolated areas target |
| [114](https://github.com/djshellshoxxx/circula/pull/114) | Recovered advisory governance plus strict JSON name equivalence | isolated advisory-governance target |
| [116](https://github.com/djshellshoxxx/circula/pull/116) | Recovered messaging plus parser and CLI input repairs | isolated messaging target |
| [120](https://github.com/djshellshoxxx/circula/pull/120) | QR camera/image cancellation, native Node workflow and exact accepted-main baseline | isolated QR target |

All future modules remain `enabled:false` and `integration:HOLD`. A module-target merge is not Core activation.

### Work still being completed at this checkpoint

The Windows private-storage follow-up is isolated in `circula-messaging-windows`, based on local messaging head `d0e28040c27f3d336076fdaee51b84869f5f4153`. Its initial candidate is now locally frozen at `f8dea465300cd76ab3db74af0fbd6284e4d847a4`; it is not published and a third reproduced cancellation-rollback lifetime defect is under repair. It validates a protected owner-only directory and SQLite files, preserves existing unsafe paths without changing their ACLs or bytes, and retains path guards. Independent review reproduced premature guard release during concurrent close and an active SQLite transaction. The repair must pass the new regressions, complete final-source Linux race/vet and Windows compilation/vet, then receive frozen-source review. Windows cross-compilation is not native execution; actual Windows CI remains mandatory.

The QR camera candidate is isolated in `circula-qr-camera`, based on `9db8a323d7dcf430d1a58a4a4d10ca3628aeb401`. Local checkpoint `2ee089e36c9f2036c797d2be5bcce5a0650f78cb` repaired late camera/import/decode completion and scanner media ownership; the same 23-case suite failed 21 cases before and passed all 23 after, including two using the exact bundled scanner. Further independent regressions show that an oversized replacement image and a manual form submission must also cancel earlier work. Those two follow-ups are repaired and independently reviewed: the expanded suite passes all 27 cases, and both independent probes pass. Final QR publication is #120 at `2f0c83bc34a52551281769ec17561b18920bd2cf`; it adds the exact two accepted-main baseline files and otherwise retains the qualified camera blobs. The scope is camera lifetime and preview navigation; no payment or authorization bypass is claimed. The final publication passed exact full-tree, parent, six-path and blob readback; current native/provider gates remain required.

PR #103 now publishes its reviewed native transport workflow and qualification note at `f831c45de9137b20abe3d31590433bda6348112b`. The four previously reviewed runtime/test blobs and dependency graph remain unchanged. This new workflow is separate from the #100-owned module workflow and shared Core workflow. The existing `ci:cross-platform` label was added before publication; all three new workflow runs were observed queued, not passed.

## Resume procedure

1. Read live repository instructions, [OWNERSHIP.md](OWNERSHIP.md), this continuation record and the current master report. Refresh main, owned PR heads and targets, open PR file ownership, configured jobs, reviews and merge state. A saved SHA is provenance, not proof that a live branch is unchanged.
2. Inspect current worktrees before editing. Keep each source owner isolated. Preserve new external commits. Never force-update a shared ref, overwrite another actor's changes or delete a source branch.
3. Verify actual current-head jobs and failed-step logs. A queued workflow, empty legacy status array, skipped placeholder, previous-head pass, successful cross-build or local subset is not a successful current native/full gate.
4. Repair demonstrated failures within the owned scope with a meaningful failing regression, minimal change, relevant qualification and independent source review. Retain failed and interrupted attempts. Publish only frozen reviewed paths; verify exact Git blobs, complete tree, parent and changed-path set; use a guarded non-force update.
5. For informational provider results, update the owned PR body with exact source, commands and job links. Do not create a source commit solely to record green CI and invalidate the checked head.
6. Mark a PR ready only after its own work is complete and actual normal gates justify readiness. Merge through the ordinary PR API with the expected head and verified target; no administrative bypass. Independent agent review is recorded source review and must not be represented as a submitted GitHub reviewer approval.
7. After an eligible integration or target movement, reconcile conflicts in the owned integration worktree and qualify the resulting source. Keep every valuable original test unless there is a demonstrated reason to change it.

## External dependencies and integration order

API-036 remains with #90 and its #109/#111/#112 continuation stack. Core consolidation #99/#119 and #102, feed #92, PWA #95, notifications Core/workflow refresh #100 and administrative recovery #107 are also independently active. Their unfinished source is reserved. Read their final handoff and normal approvals before incorporating anything.

Recurring #110 is an external three-file CI repair targeting our #97 source. It restores two accepted-main blobs and rewrites a public digest fixture's representation without changing its expected value or assertions. Its current jobs and ordinary requirements must pass first. After a verified handoff, reconcile that fixture with the recovered module's preserved `due_with_schedule_test.go`; do not discard either original test family or silently copy unfinished #110.

Federation #106 owns a README correction targeting our #105 source. Recovery #117 deliberately preserves that current README and stores the archived README separately. Do not overwrite #106. Its existence does not by itself prohibit an unrelated, independently qualified #117 stack merge; reconcile the final README handoff before completing the module-target consolidation.

Six recovery candidates and QR #120 include exactly two completed-main baseline files: `cmd/circula/auditanchor.go` with the missing `os` import and `internal/service/departure_pagination_regression_test.go` with its fixture-only cap correction. Their exact blobs and accepted-main provenance are recorded in the recovery reviews. This limited allowance does not authorize copying other unfinished Core changes. Recurring's equivalent handoff belongs to #110.

Core #104 leaves the shared CI workflow and `docs/TRACEABILITY.md` under their active owners. The preserved `pending-traceability.patch` is an older generated proposal, not a patch to apply blindly. After eligible handoffs, regenerate the complete combined traceability output without removing other workers' mappings. Check migrations, interfaces, source containment and full current-head gates.

## Archive and divergent-source reconciliation

The verified recovery ZIP and Git bundle retain all 48 archived refs and complete history. The recovered immutable baseline now has 15/15 Linux module qualifications: race, vet, formatting, dependency checksums and source-integrity checks. The final notification retry passed 35 top-level tests / 58 test events without skips on exact archive head `ba454e1c14d35bdd90dca8379e890f2bca6281d7`. The previous 14/15 checkpoint and failed/resource-limited attempts remain historical evidence.

These results qualify the recorded archived source only. They do not qualify later numeric, Unicode, chronology, transport or Windows repairs, and they do not establish native-platform, browser, provider, product-policy or Core integration acceptance.

The four-module current/archive reconciliation covers favorites, QR, discovery and onboarding. Current targets contain substantial features absent from their stale prototype narratives. Do not replace complete folders. Preserve current Store/API/CLI behavior and port compatible defects or tests independently. Actual schema, replay, limits, catalog-authority and retention-policy conflicts are documented separately from routine parser or lifecycle work. Discovery/onboarding parser and fractional-time retention candidates remain hypotheses until reproduced against the current implementation. The onboarding author owns `circula-onboarding-retention` at current base `473cab2fb45a7b67ea0001f9fdc7833ffda45eb9`, has refreshed all 35 open PR file lists without overlap, and will use the actual pinned mattn SQLite driver/tag after the Windows author releases the Go slot.

Archived notifications have a competing persistence, preference and unsubscribe design. Preserve both versions and the reviewed portable UTF-8 and JSON patches. Do not replace the current session/keyring/AAD/transports implementation or choose conflicting delivery/unsubscribe policy without a resolved contract.

## Evidence and execution recovery

The latest provider snapshot shows #103's complete executed Linux Core/module jobs passing, with its optional native Core matrix skipped. #97 and #105 dedicated module matrices passed on all three native systems, while inherited Core failures remain recorded. New recovery and #104 heads still require their own provider outcomes.

Local full Core attempts failed under resource pressure or storage exhaustion; those failures remain in `verification/`. The preceding #104 provider head passed full Linux/macOS/Windows tests, target builds, reproducibility, fuzz and contract/load jobs, but failed traceability freshness and an external APT download before browser testing. That older success does not qualify a later head.

The executor briefly disconnected during the final Windows close-lifetime run. Reattachment confirmed all source and untracked audit files survived. Read the pending receipt and exact source hashes; do not infer a pass from the interrupted session or overwrite an incomplete receipt. Only one heavy Go job may use the shared qualification cache at a time. The Windows author owns that slot until explicitly released.

Temporary execution storage can expire. Commit and publish important source and receipts promptly. Clean only this workstream's unused reproducible caches, never another actor's cache or source. Preserve raw logs or a lossless evidence archive with a manifest where feasible.

When all currently actionable work is complete and only provider queues, normal approvals, verified owner handoffs or genuine manual acceptance remain, use a bounded continuation check scoped to these owned candidates. Refresh ownership on every run. Notify only meaningful changes; never report unchanged queued work as progress or relax a gate merely to merge more branches. No such continuation automation is claimed as created by this checkpoint.
