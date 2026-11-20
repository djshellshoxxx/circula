# Independent review: receipts, governance, areas and QR

Reviewed by `/root/password_admission` on 2026-11-20. Final read-only provider snapshot: **2026-11-20 16:49:29 UTC**. This review used its own worktree and made no remote or application-source changes.

## Disposition

**No blocking source finding remains for merging these exact candidates into their existing isolated future targets.** All four candidates remain disabled and on integration **HOLD**. The disposition does not approve Core activation, choose outstanding product or retention policy, or qualify a release.

At the final provider readback, all four PRs were open, draft and mergeable; each had **seven applicable Core jobs and three native module jobs successful**. Each Core run also returned one skipped conditional-matrix placeholder. No submitted GitHub reviews were returned. This independent review is evidence for the coordinator; it does not impersonate a provider approval or determine repository permission requirements. The coordinator must handle draft/review/merge policy using the pinned source and target, with a final drift check before any mutation.

| PR | Source branch and exact head | Existing target and exact base | Changed paths / unique commits |
| --- | --- | --- | --- |
| [#113 receipts](https://github.com/djshellshoxxx/circula/pull/113) | `master/recover-receipts-20261119` — `6186416ddd83c365405a798130999cb1e4df0f31` | `future/fnet-203-receipts-20261116` — `bd2fbdbd3adf77b27fa12335ff2827359423aea3` | 67 / 3 |
| [#114 governance](https://github.com/djshellshoxxx/circula/pull/114) | `master/recover-governance-20261119` — `711833c510b14e84c3a7441b3b63f2f990bc8368` | `future/fnet-208-governance-20261116` — `b4f511372ec26dc63d005bb50d87c56e18fc1a88` | 77 / 2 |
| [#115 areas](https://github.com/djshellshoxxx/circula/pull/115) | `master/recover-areas-20261119` — `52cf9c3afadb950e4dd428807319e3c96f3004a9` | `future/fnet-206-areas-20261116` — `8b444b20139e3e687db494b3fe36bb3643148e5b` | 108 / 2 |
| [#120 QR](https://github.com/djshellshoxxx/circula/pull/120) | `master/fux102-camera-lifecycle-20261119` — `a91bb5fdc97fc540a1617538073e92509ceba485` | `future/fux-102-qr-20261116` — `9db8a323d7dcf430d1a58a4a4d10ca3628aeb401` | 10 / 2 |

The complete unique commit histories and path lists are in [source-inventory.json](evidence/future-four-review-20261120/source-inventory.json). The review read the changed production surfaces, relevant contracts and tests, every dedicated workflow, and the common Core restoration. Original regression sources were inspected without executing pre-fix reproductions.

## Source, merge and isolation verification

For every PR, `git merge-base BASE HEAD` returned the exact current target base. Read-only `git merge-tree --write-tree BASE HEAD` exited zero, reported no conflict, and returned the exact head tree:

| PR | Simulated merge tree, also the reviewed head tree |
| --- | --- |
| #113 | `f611a03c49b5862052271789e9b353e09ecbb7da` |
| #114 | `3c776a8d327c37798cffdcc27b6b8ed6c6c5cddd` |
| #115 | `0f4b70bcf024b5993d41165ef1905f0b9b64d8f3` |
| #120 | `05ea33f8cadd386e717b7ed836e76bc7e8abea90` |

Source branch, source SHA, target branch and target SHA were identical between initial and final provider reads. All four candidates are descendants of their target. There is no outstanding merge conflict at these pins.

Every manifest has `enabled: false`, `integration: HOLD`, and an empty future-feature dependency array. No changed path belongs to another future module. Source scanning found no Core/foreign-future import literal or automatic `init` registration in these modules. Explicit constructors and migration entry points remain unregistered. The root `go.mod`, `go.sum` and shared `.github/workflows/ci.yml` are byte-identical to each corresponding base.

Each PR changes exactly seven paths outside its owned module: its one dedicated workflow and the same six bounded Core restoration paths. All six final Core blobs are identical across the four candidates:

| Core path | Provenance verified |
| --- | --- |
| `cmd/circula/auditanchor.go` | Exact accepted blob `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` from `dc97f7d5b0ae64306a8e2b1e57ec83f233d3d09d`; missing `os` import restored. |
| `internal/service/departure_pagination_regression_test.go` | Exact accepted blob `3b117ecfe2f01e7e2363b9d201503cde6e0f5595` from `7457b618057a8b1757ec643806e9afbfc5ba907e`; fixture cap supports the existing 501-proposal assertions. |
| `internal/store/sqlite_erasure.go` | Exact accepted blob from `28326597340a4f0efad33f37feecc380db0de66f`; resource-related notification redaction remains scoped to the erased member/network. |
| `internal/service/notification_erasure_test.go` | Exact accepted regression blob from `28326597340a4f0efad33f37feecc380db0de66f`, including exact-link, prefix, same-name and cross-network controls. |
| `internal/service/erasure_test.go` | Exact accepted deterministic privacy/credential regression blob from `fc31dc9a6b8c102fe612f47bc543e94e8114df0f`. |
| `cmd/circula/util.go` | `writeAtomicPrivateFile` exactly matches its accepted function at `fc31dc9a6b8c102fe612f47bc543e94e8114df0f`. Base-to-head changes are the two narrow G703 comments only; executable code and the older target's `storeOptions` helper are unchanged. |

Older recovery documents describe the earlier two-file baseline restoration. The current PR bodies and this report record the later four-file follow-up as well; those historical statements are not a current six-path inventory.

## Module-specific findings and remaining gates

### #113 — receipts

**Disposition: ready for isolated-target consolidation at the pinned head.**

Read the complete recovered receipt projection, canonical parsing, certificate issuer/verifier, local trust handling, service, HTTP, CLI and health surfaces. The plain-document numeric correction preserves bounded integer semantics using the original numeric tokens, without changing the certificate wire format, Core code paths or dependency graph. The recovery preserves the current completed-trade, tenant, party/scoped-admin and consent boundaries. Service rereads after signing remain in place.

The module requires host-authenticated actors, an authorized consistent read projection, current network enablement and optional signing consent. Signing uses the injected signer and local trust metadata; verification performs no external key fetch. Generated receipts remain ephemeral. The HTTP adapter bounds input before host callbacks and retains no-store, escaped output and host CSRF requirements.

The recorded archive inventory has 44 retained original paths, with 42 current blobs equal to their recorded archive blobs. The two intentional changed originals are README and `canonical.go`; all six original test files, manifest and both dependency files remain exact. The added regression is preserved. Dependencies stay limited to the receipts module, pinned JCS and pinned go-jose; the workflow checks owning module paths, including generated test packages.

Remaining gates are the actual Core authority/read-transaction/key adapters, complete verifier interoperability, production trust/key operations, browser/manual accessibility, and later explicit feature integration. The historical completion percentage is not a present activation claim.

### #114 — governance

**Disposition: ready for isolated-target consolidation at the pinned head.**

Read the recovered transactional service, schema checks, proposal lifecycle, frozen electorate, unique ballots, own-ballot read boundary, advisory results, comments/moderation/process disputes, independent certification, policy references, idempotent replay checks, strict parser, HTTP and CLI.

Authority callbacks receive the same SQL transaction as the decision. The checked schema-row writer reservation precedes mutable authority reads. Current active/enabled authority still applies before replay, and the service retains additive module-owned tables and checked writes. The module provides no binding adoption rule or policy writer. Ballot privacy is expressly limited to own-ballot API visibility; database operators can read stored choices.

The parser correction matches the JSON decoder's Unicode simple-fold field equivalence while preserving valid single aliases and distinct multi-rune spellings. The recorded archive inventory has 48 retained paths, 46 exact; README and `strict.go` are the only changed originals. All six original tests, dependency files and disabled manifest match recorded archive blobs.

Actual Core authority/admin-MFA/store/policy-history adapters, deployment retention and backup policy, manual browser/accessibility/translation acceptance and separate feature selection remain open. This is FNET-208 advisory governance; it does not replace Core GOV-021 administrative recovery.

### #115 — areas

**Disposition: ready for isolated-target consolidation at the pinned head.**

Read named-area validation, durable revisions and idempotency, transaction-scoped policy calls, association cleanup, retained replay tombstones, default-off sharing, quota-independent opt-out, encrypted actor/network/area-bound pagination, strict parsing and HTTP/CLI surfaces.

Private association and listing visibility decisions remain host policy calls within the service transaction. Cleanup removes association payloads and retires cached replies while preserving replay refusal. The sharing export is limited to the enabled named taxonomy. No coordinates, location inference, geometry, map integration or cross-feature import is introduced.

The recorded archive inventory has 76 retained original paths, 74 exact; README and `json.go` are the two changed originals. All seven original tests, manifest, dependencies, notices and historical material are retained. The same simple-fold comparison correction preserves the established unambiguous JSON contract.

Actual Core principal/MFA/listing/store/session/CSRF integration, replay-preserving retention/backup policy and browser/keyboard/zoom/screen-reader acceptance remain open. A photo adapter, if selected later, would require its own media/EXIF qualification; this candidate does not accept image uploads.

### #120 — QR

**Disposition: ready for isolated-target consolidation at the pinned head.**

Read the complete camera wrapper delta and 27-case deterministic lifecycle suite. Attempt ownership now covers pending permission, playback, imports, native/fallback results, images, manual submit and cancellation. A stale operation cannot mutate a newer attempt. Immediate attached-track release and per-attempt video elements keep delayed bundled-scanner cleanup confined to the old element. Existing successful preview navigation, manual fallback, decoded-text/file bounds and empty-selection behavior remain covered.

All QR Go files, module dependency files, disabled manifest and bundled scanner/worker distributions are byte-identical to the target base. This candidate selectively adapts cancellation handling; it does not replace the target's service, store, scanner, route or payment/authorization semantics.

The dedicated current workflow qualifies Node syntax and the lifecycle suite on three native operating systems. It does not run the QR Go module suite, a real camera or a real mobile browser. Actual mobile camera/CSP/manual accessibility, host-session/CSRF integration and transaction/ledger acceptance remain future gates. The old manifest/README scope description remains documented historical inventory debt.

## Current-head provider evidence

The following job IDs were read directly from each exact-head PR workflow run's latest job list. All listed applicable jobs concluded **success**.

| Applicable Core job | #113 / run 44555363075 | #114 / run 44555364562 | #115 / run 44555371649 | #120 / run 44555388643 |
| --- | --- | --- | --- | --- |
| Ubuntu race test | 135669882378 | 135669875564 | 135670089371 | 135669970129 |
| Fuzz smoke | 135669882895 | 135669875338 | 135670089664 | 135669970512 |
| Vet, formatting, lint, vulnerability check | 135669882915 | 135669875809 | 135670089347 | 135669970642 |
| Release-target builds | 135669882989 | 135669874765 | 135670089607 | 135669970667 |
| Reproducible builds | 135669883035 | 135669876118 | 135670089444 | 135669970826 |
| API contract/load smoke | 135669883047 | 135669876534 | 135670089646 | 135669970885 |
| Playwright/axe/keyboard | 135669883085 | 135669875624 | 135670089772 | 135669970762 |

| Native module job | #113 / run 44555363061 | #114 / run 44555364365 | #115 / run 44555371753 | #120 / run 44555388749 |
| --- | --- | --- | --- | --- |
| Ubuntu | 135669874925 | 135669873285 | 135669900699 | 135669966801 |
| macOS Intel | 135669876038 | 135669872740 | 135669900507 | 135669966356 |
| Windows | 135669875665 | 135669872290 | 135669900087 | 135669965853 |

Run links:

- Receipts: [Core](https://github.com/djshellshoxxx/circula/actions/runs/44555363075), [module](https://github.com/djshellshoxxx/circula/actions/runs/44555363061).
- Governance: [Core](https://github.com/djshellshoxxx/circula/actions/runs/44555364562), [module](https://github.com/djshellshoxxx/circula/actions/runs/44555364365).
- Areas: [Core](https://github.com/djshellshoxxx/circula/actions/runs/44555371649), [module](https://github.com/djshellshoxxx/circula/actions/runs/44555371753).
- QR: [Core](https://github.com/djshellshoxxx/circula/actions/runs/44555388643), [lifecycle](https://github.com/djshellshoxxx/circula/actions/runs/44555388749).

A job URL appends `/jobs/JOB_ID` to its run URL. This yields **40 applicable successful jobs, zero failed/cancelled/pending applicable jobs**, across eight successful runs.

Core's `test-cross` has `if: github.event_name != 'pull_request'` and a two-target matrix (macOS Intel and Windows). On each PR the provider returned a single skipped unexpanded placeholder named `test (${{ matrix.os }})`:

| PR | Skipped placeholder job ID |
| --- | --- |
| #113 | 135669885251 |
| #114 | 135669877407 |
| #115 | 135670091440 |
| #120 | 135669972762 |

These are **four returned skipped job records representing eight configured native Core target executions that did not run**. They are not eight successful native Core jobs or eight separate skipped IDs. The three native module jobs per PR are separate actual successes. Receipts/governance/areas run module race on Linux/macOS, ordinary module tests on Windows and bounded module fuzzing on Linux. QR's three jobs run the Node lifecycle suite. Conditional platform steps and failure-screenshot uploads were skipped as expected; no required applicable test step was silently skipped.

## Historical standalone qualification and receipt verification

This review executed no Go/Node suite, fuzz campaign, vulnerability reproduction or load traffic. It inspected current provider outcomes and the source-bound historical evidence, avoiding compilation contention with the ongoing Core campaign.

- **Receipts:** verified all 17 source/test/dependency SHA-256 entries in each repaired-source ordinary/vet/checksum receipt against current files, with zero mismatch. The committed ordinary receipt is exit 0, 84.6% coverage, 0.044 seconds package / 5.148 seconds wall; vet and checksum receipts also exit 0. The earlier archive race log and later PR-body race/fuzz qualification are historical evidence; the final current-head native jobs independently provide the applicable race/ordinary/fuzz gate outcomes. No source-equivalent result is presented as a newly executed local test.
- **Governance:** verified all 15 runtime/test/dependency hashes and six portable log hashes with zero mismatch. The historical real-driver race receipt records 59 passing test events, zero failed/skipped, 75.7% coverage; the actual strict parser/original fuzz target completed 100,000 executions. The portable race file is explicitly a result/coverage excerpt, not the entire original log.
- **Areas:** verified all 16 runtime/test/dependency hashes and six portable log hashes with zero mismatch. The historical real-driver race receipt records 68 passing events, zero failed/skipped, 72.1% coverage. Parser fuzz completed 100,000 executions. Carried-forward results are explicitly source-equivalent. The earlier mixed assertion/resource failure remains preserved; it is not reclassified as a clean run. The portable race file is likewise an explicitly labelled excerpt.
- **QR:** reviewed the historical 27-pass/zero-skip local qualification and current exact-head native provider successes. The suite evaluates the unchanged production script in a synthetic DOM/media harness; two cases additionally exercise the actual bundled scanner's delayed lifecycle. It does not establish physical-camera or browser acceptance.

The three archive commit objects and QR's historical local qualification commit are not present in this checkout. This reviewer compared all current candidate blobs against the committed per-file recovery receipts, verified current source hashes and read historical review records; it did not reopen the recovery bundle or independently reconstruct those absent historical Git commits. All **168 recorded original files** remain retained; **162** match their recorded archive blobs, with only the three README changes and three reviewed parser changes differing. All **19 original test files** and each recorded manifest/dependency pair remain exact.

Current provider metadata, source inventories and verified receipt hashes are preserved in [the evidence directory](evidence/future-four-review-20261120/README.md). No branch was deleted or rewritten, no future module activated, and no deployment/release action occurred.

## Handoff

The coordinator can consider normal gated merges into the four existing future targets at the exact pins above. There is no source repair requested by this review. Check branch/head drift and current repository merge requirements immediately before any merge. Retain all archived/remaining development streams and the HOLD boundaries; Core integration remains separate work.
