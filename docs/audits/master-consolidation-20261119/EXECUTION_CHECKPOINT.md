# Master consolidation execution checkpoint

This is an execution checkpoint for the autonomous consolidation directive, not a completion or release declaration. The full inventory, recovery dispositions and final receipts are being consolidated separately.

## Reviewed Core repair

The code source is local commit `3cc231542e7c46793f27b2fc406b9220fabf1bff` (tree `ca441d471000c1ae24ca87c28255a0c467c6e2f9`), based on main `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. Its three Go verifier files retain the independently reviewed export-package boundary repair. Its two Python files add the reviewed complete-ZIP-read, integer/type, ordered field-alias and strict-JSON-constant corrections. Valid legacy forms, unknown extensions and applicable last-value semantics remain covered.

The final Python suite has 38 top-level tests. It passed again on Linux with Python 3.12.14 in 17.199 seconds (17.360 seconds wall), with both source files unchanged. The independent source review and original fail-before fixtures are preserved in the manifest-parity evidence. Local compiler/storage failures from earlier complete Core commands remain recorded as failures.

A new dedicated native reader workflow runs the same suite on Linux, macOS and Windows. It uses the hosted interpreter, records its actual version and source blobs, and introduces no action outside the existing repository allowlist. The source/workflow review found no remaining blocking issue after an initial unapproved-action choice was removed. No provider execution of this new workflow is claimed yet.

## Provider results at the preceding published source

PR #104's earlier head `c3c991aa3c12a11edce2874e9444b9d2a99f746c` was tested as merge `ae7b54a77f24b7ea71099220fe6b50c17b6dc2dc` into the recorded main. Run [44511241324](https://github.com/djshellshoxxx/circula/actions/runs/44511241324) completed with seven successful jobs: Linux race, macOS race, native Windows tests, release-target builds, reproducible-build/local smoke, bounded fuzz, and API contract/load.

Two jobs failed. Generated DAT-040 traceability is stale and remains a real merge blocker owned by the active shared-document stream. Browser dependency installation received Ubuntu package-server HTTP 503 errors and stopped before the browser suite ran. Vet, action pins, hygiene, licences and secret scanning passed before the traceability failure; later Python, golangci-lint and vulnerability steps in that job were skipped. The recorded build-smoke job is not a production deployment. See `CORE_PROVIDER_CHECKPOINT.json` for the exact jobs and failure excerpts.

Those results qualify that earlier revision and its merge only. This follow-up needs current-head checks; neither previous passes nor local source equality bypass that requirement.

## Isolated recovery continues

The verified recovery ZIP and complete Git bundle retain all 48 saved refs. Fresh unchanged-archive Linux race/vet/format/checksum qualification passed for 14 of 15 modules; the notifications archive's resource-killed race attempt remains unqualified. These are source-specific engineering checks, not full-feature release acceptance.

Reviewed module work stays isolated and disabled/HOLD. PR #97's narrow recurring calendar change has passed its three native module jobs. PR #103's notification transport repairs have passed the actual pinned-driver provider module gate. PR #108 now contains the recovered analytics implementation at `38b9bcd0e049d97167d793baba933d5e79d8030d`, with tree `926205aae18016ef55f882fda6415e1ac2b7fd94`; its current-head checks are pending.

Further recovered recurring, federation, receipts, areas, advisory governance and messaging candidates are frozen for source review and staged qualification. This session has reproduced and repaired additional recurrence timestamp ordering, Unicode JSON-alias ambiguity, messaging stdin UTF-8 normalization and offline receipt numeric-rounding defects. Each repair has its own actual evidence and remaining qualification scope; no unexecuted full suite is described as passing. The receipts rounding finding concerns the plain offline parser, not a demonstrated signature or authorization bypass.

The original notification archive also has two portable reviewed repair patches. Its competing storage/subscription design is preserved for reconciliation; it does not replace the later session/AAD/transport implementation.

## Active ownership and integration boundary

The latest observed active Core umbrella is PR #90 at `839e86935767288e594914ad711486616a604eca`. New draft PRs #109 (UI/catalogue continuation) and #111 (API-036 qualification) target that branch. PRs #99/#102 and the notification/PWA refreshes remain separately owned. Feed PR #92 remains reserved. No unfinished worker source, shared migration, contract, generated traceability or shared CI change is copied here.

PR #106 owns the narrow federation README handoff. PR #110 owns the recurring accepted-baseline/golden-digest CI repair and remains subject to its current-head gates. Recovered federation preserves that README boundary, and recurring does not duplicate PR #110.

Every merge must follow normal repository checks and approvals at the actual head. No future module is activated or merged into Core by this checkpoint. No release, deployment, branch deletion, shared force push or protection bypass is authorized by these receipts.
