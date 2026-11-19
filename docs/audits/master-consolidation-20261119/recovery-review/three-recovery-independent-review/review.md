# Independent review of areas, governance and messaging recovery

**Disposition: no blocking source or dedicated-workflow finding in the reviewed recovery changes.** One evidence-classification defect was found and is now corrected. This is source/provenance review; current runtime, native/provider and integration gates remain separate.

## Exact reviewed candidates

| Module | Commit | Tree | Archived files retained | Original test files retained byte-for-byte |
|---|---|---|---:|---:|
| areas | `0ad43e4a414945e1351e7509dce7f214516d199b` | `d784637417003e1a18b6ad3ee279aa8f38569fa3` | 76 | 7 |
| governance | `8f7457836e05c9ffc7ae85e9c4ad48977403140b` | `95b0efe8617e5125614b68507e50c55a1ee975ef` | 48 | 6 |
| messaging | `7e9708d8e50a1959af399aa5956fb2a54ab880b6` | `a5556af33aca22f91ee5371d1d979217bf4936b8` | 65 | 9 |

The areas conclusion includes the docs/evidence-only correction from `b7659583e01b127d65c6a127030b9d3389f5d95c` through `0ad43e4a414945e1351e7509dce7f214516d199b`. All three worktrees were clean at final readback.

## Scope and preservation

I reviewed the complete new parser and CLI production deltas, all new regressions, relevant production callers, original fuzz sources and each dedicated workflow. A complete Git blob comparison independently confirmed that every archived module path remains present. All original test files, licenses/notices, migrations, historical evidence, module manifests, `go.mod` and `go.sum` remain unchanged. README recovery notices explicitly distinguish preserved historical claims from current qualification.

The only production changes relative to each archive are:

| Module | Production delta |
|---|---|
| Areas | `future/areas/json.go`: duplicate-field comparison uses Unicode simple-fold equivalence. |
| Governance | `future/governance/strict.go`: the same narrow comparison repair. |
| Messaging | `future/messaging/http.go`: the same duplicate comparison repair; `future/messaging/cmd/circula-messages/main.go`: reject malformed UTF-8 after bounded stdin reads and before creating a request. |

Outside each module, the complete candidate diff contains its new dedicated workflow and exactly two authorized accepted-main baseline files. Their blobs match `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`:

| Path | Accepted-main blob |
|---|---|
| `cmd/circula/auditanchor.go` | `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` |
| `internal/service/departure_pagination_regression_test.go` | `3b117ecfe2f01e7e2363b9d201503cde6e0f5595` |

No active worker source, shared workflow, root dependency file or unrelated module is part of these changes.

## Parser semantics and regression coverage

The comparison helper walks each `unicode.SimpleFold` cycle and chooses its minimum rune. This matches the equivalence relation used by Go's JSON field matching without applying multi-rune expansions. The helper supplies only a comparison key; it does not rewrite input names or data. Per-object tracking still applies to nested objects. Existing unknown-field, depth, trailing-data and Unicode handling is retained.

The new tests call the actual production parser. Their ordinary `encoding/json` controls show that long-s/final-sigma aliases can assign the same typed field twice. Both alias orders, escaped long-s and nesting are covered. ASCII and Kelvin rejection controls remain present. Single valid aliases remain accepted; sharp-s versus `SS` and ligature versus expanded ASCII remain distinct. Valid Unicode, decomposed text and literal U+FFFD are preserved.

The retained fail-before logs show five newly failing areas parser subcases, six governance parser subcases and six messaging parser subcases. Messaging also has eighteen malformed-stdin cases. Its tests require rejection before any HTTP transport call or output and retain valid Unicode controls. The tests do not run in parallel while replacing the process-global default transport.

### Corrected evidence finding

The original areas fail-before log contains both all five intended assertion failures and a killed `modernc.org/sqlite/lib` vet subprocess with a `build-fail` event. Its initial assertion-only classification was incomplete. Commit `0ad43e4a414945e1351e7509dce7f214516d199b` corrects the JSON classification and RECOVERY prose to a mixed assertion/resource result.

Independent readback verified that `fail-before-original-receipt.json` is byte-identical to the prior receipt and has SHA-256 `a0d46b85cb641ee7cf2444aee162016d89e359eb0981987cf184023181e1b380`. The raw log remains byte-identical with SHA-256 `695d74e8a3f56ab9ec9c83afa85456074e9a1481cffb19e5828454427564fc67`. Only the three documented evidence/prose paths changed. Production, tests, dependencies and workflow remain unchanged. Governance and messaging fail-before logs contain no analogous build/resource diagnostic in this review. The finding is closed.

## Dedicated workflows

Static readback and checks covered YAML syntax, immutable action pins, token permissions, checkout credential handling, matrix/native commands, shell syntax, embedded Python syntax and literal dependency allowlists. All passed. The allowlists exactly match the archived module dependency graph, including required indirectness, SQLite 1.60.1 and x/sys 0.48.0; no dependency or driver substitution is introduced.

Each workflow keeps the feature disabled and on integration HOLD, forbids replacement/exclusion or unintended Core/other-feature imports, records exact Git/module metadata, verifies checksums and requires formatting/vet. Linux and macOS run the complete real-driver race suite; Windows runs the ordinary suite. Job, test and fuzz execution is bounded. No `continue-on-error` or skip masks the required native checks.

Areas and governance fuzz the actual production parser plus the original fuzz source as a source-file command. Their full-module real-driver race/vet gates are separate and remain required. Messaging retains full-package fuzz over its original SQLite graph. The recovery prose preserves earlier resource failures and does not equate parser-only fuzz with whole-module qualification. `go list -buildvcs=false` is limited to dependency-boundary enumeration; explicit Git metadata is recorded separately and real test/vet commands retain their ordinary flags.

## Native Windows limitation

Messaging's unchanged `Open` constructor rejects group/world mode bits with `info.Mode().Perm() & 0077`. The pinned Go 1.26.11 Windows implementation reports writable regular files as 0666 and read-only files as 0444; both fail that guard. The recovery document correctly describes a source-based expected portability gate, retains the fail-closed check and runs the actual Windows suite. This review did not execute Windows tests and does not approve a permission-check relaxation. A later reviewed ACL or host-pool adaptation and actual native result remain necessary.

## Qualification boundary

This reviewer ran read-only provenance, hash, YAML, shell-syntax and embedded-Python checks. No Go build, test, vet, fuzz or provider run was started during this review because the serial qualification slot belongs to the author. After-change runtime receipts must be evaluated separately against their actual source blobs; archived passes remain historical evidence. This signoff does not claim a full independent security audit of all unchanged archived implementation, full Core qualification, native/provider success, real browser/accessibility acceptance, policy selection, feature activation, integration, merge or deployment approval.

The machine-readable `review.json` includes every archived path and blob mapping, complete changed-path sets, exact source/test/workflow observations, the original raw-log checks and the closed evidence finding. `provenance-and-static-review.json` preserves the initial review state before the evidence correction.
