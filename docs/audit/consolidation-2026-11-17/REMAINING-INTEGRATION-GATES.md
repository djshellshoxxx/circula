# Remaining integration gates

This is a resumable checkpoint, not an authorization request. The repository owner already authorized verified core merges and preservation of valuable work. The remaining immediate blocker is execution of required GitHub CI on the reviewed current heads. No gate was bypassed and no automatic merge was enabled.

## Verified endpoints

- Integrated main: `b4398074ae4ca72559235ca50242cfcb7a934eec`, tree `1e2923efdff1adcf60c2718fe2a6a56edb3c1ed9`.
- Locally qualified combined candidate: `17152dadad91d4e1c994ea807e7d6e2497513aa8`, tree `1e47c9c30165ca27866887533586c90323e1630e`. It preserves that main and includes the reviewed changes from #82, #84 and #87. The exact constituent ancestry/equivalence mapping is in [integration receipts](integration-receipts.json).
- [Full race result](evidence/final-candidate-qualification.json): one completed invocation, 14 test-bearing packages and two helpers, exit 0. [Pinned uncapped lint](evidence/final-password-proof-lint.json) and [module/vet/traceability checks](evidence/final-candidate-static-checks.json) also pass on that candidate.
- [Provider execution evidence](evidence/final-ci/queue-current-20261118T102432.json): at 10:24:32 UTC, #84 had four required jobs passed, two running and one queued. The other four reviewed PRs each had seven queued jobs. The earlier wholly-unassigned observation remains a dated record. No billing, capacity, approval or service-outage cause is inferred.

## Source-reviewed work awaiting provider execution

| PR | Exact reviewed head | Current required CI run | Remaining gate |
|---|---|---|---|
| [#82](https://github.com/djshellshoxxx/circula/pull/82) | `e127acc9877edebeca818b75a5d48ce12511eca8` | [44279484396](https://github.com/djshellshoxxx/circula/actions/runs/44279484396) | Seven required jobs. Existing authority/throttle policy is preserved. |
| [#84](https://github.com/djshellshoxxx/circula/pull/84) | `eb4bb08b15d41295aa2c4b9f93040124ce4d66e9` | [44272452596](https://github.com/djshellshoxxx/circula/actions/runs/44272452596) | Seven required jobs. Only after merge may A2-08/A3-F10 be promoted from partial on main. |
| [#87](https://github.com/djshellshoxxx/circula/pull/87) | `679a3016d77c3cf34e218ecb296f1d42cc0969c8` | [44275665191](https://github.com/djshellshoxxx/circula/actions/runs/44275665191) | Seven required jobs. Merge only as partial remediation; both timestamp residuals stay open. |
| [#88](https://github.com/djshellshoxxx/circula/pull/88) | `c774d5e412e039d514c37c7b69826d5306d7930c` | [44281537794](https://github.com/djshellshoxxx/circula/actions/runs/44281537794) | Seven required jobs. Documentation only; pin citation correction and preserve the three useful original edits. |
| [#78](https://github.com/djshellshoxxx/circula/pull/78) | `20d42bb6980d3d0e2c5bab7085e9b01a211dfaa8` | [44283037811](https://github.com/djshellshoxxx/circula/actions/runs/44283037811) | Seven required jobs, plus #82/#87 documentation dependencies. Keep the audit record last among these changes. |

Before each merge, fetch remote refs and re-read the PR's head, base, review state and named job outcomes. Require successful current-head results; a queued run, superseded run, local aggregate, skipped optional matrix or successful different head is not a substitute. If a worker has advanced a head, inspect the delta and establish its source/test evidence before proceeding.

Use focused merge commits with an expected-head check. The intended core order is #82, #84, then #87. Their generated `docs/TRACEABILITY.md` headers may need regeneration when assembled; the locally qualified candidate already records that generated-only resolution. Do not overwrite production code or tests to resolve a generated-file conflict. If a new published source or resolution changes the reviewed content, run affected checks and the required current-head CI before merging. #88 is independently applicable documentation. Merge #78 only after its referenced core audit notes are present. Recheck this evidence PR against the resulting main and its own required checks before merging it.

After an observed or performed merge, record the exact merge SHA and verify that its ordered parents and resulting tree preserve the reviewed changes. Update the current disposition rather than rewriting initial historical records. Recompute branch and PR counts after the evidence PR itself exists; this package deliberately captures the prior endpoint to avoid a self-referential receipt.

## Work that remains intentionally held

| PR or finding | Preserved disposition |
|---|---|
| #83 / AUD4-F02 | Reviewer-reset exclusion is an owner-policy proposal. It is separate from A3-F02 / DEC-014 and was removed from the compatible #82 slice. |
| #85 / GOV-021 | Permanent authority, authenticated operator independence, waiting/objection/expiry and break-glass/time-bound-access/notice choices remain unapproved. Source fixes for audit/output, indexing, pagination and the closing test do not settle those choices. |
| #86 / SEC-026 | Stacked on #85. Enabled defaults/thresholds, member-policy visibility exception, subject visibility and retention need decisions. Migration numbering is resolved to0020; dependency and policy gates remain. |
| #67 and #80 | FNET-204 future work, with #80 stacked on #67. Preserve their branches, `enabled: false` and `integration: HOLD`. Do not merge into core. |
| A3-F02 / DEC-014 | Unresolved. The assignment does not supply an owner decision on lasting break-glass actions, second approval or notices. |
| AUD4-F03 | Same-millisecond and host-clock-rollback cases are independently reproduced. Durable policy-at-opening binding and compatible legacy-data handling remain separate work even after #87 merges. |

The [security disposition](SECURITY-DISPOSITIONS.md) retains the historical 48 addressed, five policy/design and two partial findings on this main checkpoint, as well as the additional concurrent findings. The [initial/final branch inventories](README.md#inventory-scope-and-concurrency), historical closure evidence and preserved archive prevent a missing merge from being confused with missing source material.

No deployment, release/tag publication, branch deletion, remote history rewrite or force push is part of this checkpoint or its remaining instructions.
