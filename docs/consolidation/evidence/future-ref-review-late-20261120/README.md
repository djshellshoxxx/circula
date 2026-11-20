# Late branch and future integration verification — 2026-11-20

This independent read-only review refreshes the repository at approximately
23:08–23:15 UTC. It reads the published finalization receipt at
`49dc8acc781790635fca8ed32882a7a27971c8a1` and verifies its three future-target
merges against live GitHub jobs, Git data and freshly fetched local objects.
No runtime source was edited, no product tests were executed, and this worker
performed no remote mutations. These files are dated receipts, not a promise
that concurrent repository activity stops after the observations.

## Three completed future-target integrations

| PR | Actual target | Verified merge | Exact tree | Successful jobs |
|---|---|---|---|---:|
| [116](https://github.com/djshellshoxxx/circula/pull/116) | `future/fux-105-messaging-20261116` | `54d3af51aa32c022562e8041d1baa149da071db3` | `c58b7a38c802190229b26dc64be05af355b416b9` | 7 shared + 3 native |
| [117](https://github.com/djshellshoxxx/circula/pull/117) | `future/fnet-209-federation-20261116` | `0aeb5a1446798019141a3f6f5bcb1e7a24b70c47` | `6709ccb99b01574dedb554e1723dac90f959cb47` | 7 shared + 3 native |
| [118](https://github.com/djshellshoxxx/circula/pull/118) | `future/fnet-207-recurring-20261116` | `89c6aeb6b4f73292d38a4a0803146ff5b9386e6f` | `64ca1e44dc878d7fb73bd8e30f52966813a78837` | 9 shared + 3 native |

The [individual-job snapshot](gates.json) verifies **32 successful jobs** on
the three exact source heads. Messaging and federation each also have one
conditional native Core matrix placeholder explicitly **skipped**. No skipped
job is included in the successful count. Native module jobs ran on all three
configured platforms; their workflow instrumentation remains source-specific.

The [GitHub readback](merge-api-proof.json) records each actual target ref,
unchanged source ref, exact tree and ordered parents. Independent local Git
checks match those objects and establish that each merge tree equals its
qualified source tree, with no differing file. The full original source and
target histories remain reachable.

All **15 actual future-target manifests** were read at their branch tips and
still have `enabled:false`, `integration:HOLD`, and empty declared future-module
dependencies. Main is `aa01563fcc891da60fefebb8335eca27855678e8`. Core candidate
`bea5de25cf76b66d87689034e2529f9b40c6dff6` has no changed `future/` path relative
to main and no Go source reference containing a future-module import path.
This review establishes preservation/isolation, not Core activation or
production/device/peer acceptance.

## Ref refresh and new work

The two-page live GitHub branch listing and local fetched remote refs match
exactly: **152 nonsymbolic remote heads**. [refs-prs.json](refs-prs.json) records
the current branch identities and the 17 PRs open at that observation.

Two historical comparisons have different observation boundaries:

| Comparison | Unchanged | Moved | New | Removed |
|---|---:|---:|---:|---:|
| Original 148-head branch inventory published in `2eb576a…` | 134 | 14 | 4 | 0 |
| Later 18:23:46 preservation snapshot bundled in the same publication | 143 | 5 | 4 | 0 |

Every moved historical head remains an ancestor of its current tip. The
[original-inventory delta](inventory-delta.json) and [later-ref proof](local-ref-proof.json)
retain exact old/new commits and changed-path inventories. The latter snapshot
predates the analytics merge and creation of the separate evidence branch;
it must not be mistaken for the repository state after that publication.

The four added names are the known evidence branch and three loading-repair
branches:

- `audit/core127-final-evidence-20261120` at `49dc8acc781790635fca8ed32882a7a27971c8a1`.
- `repair/core127-sync-loading-20261120` at `e5f7664c505a67fc1aa21c8d4609b55708991457`.
- `repair/core127-loading-regression-syntax-20261120` at `bea5de25cf76b66d87689034e2529f9b40c6dff6`.
- `repair/core127-e2e-observer-20261120` at `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef`, proposed in PR129.

No new unrelated feature stream or unclassified runtime delta was found in
this refresh. The loading changes are independently owned by the Core review
workers; this receipt does not qualify them or replace PR127/129 gates.

## Historical future PR reconciliation

PR105 is already merged, and PR106 is already closed without a second merge.
Their original sources, respectively
`059b0e4508a6c4f8e29f6a4cccd06f92b14ea115` and
`d55554408ef591371768259905c70ca58d68facb`, are ancestors of the actual federation
target `0aeb5a1446798019141a3f6f5bcb1e7a24b70c47`. The source branches remain.

**PR80 was the sole still-open future-module PR in the initial snapshot. This
review established complete inclusion through PR92; the coordinator then
closed PR80 at 23:14:40 UTC while retaining its source branch.** This worker's
[independent later readback](pr80-closure-readback.json) confirms closed state,
`merged:false`, and the unchanged source ref. The original snapshot remains
unaltered.

PR80 source `0952c94a5dbab4044d7673ce9d51cec9063ab97f` is an ancestor of actual
feed target `108107435645db687f8ac380e6571319f419fa32`. The complete eight-path
original change was accounted for:

- All three original test files are byte-identical in the accepted target:
  `cursor_auth_test.go`, `cursor_fuzz_test.go`, and `feed_test.go`.
- The entire original `VALIDATION.md` body is retained; later qualification
  records were prepended with separate attribution.
- The README reconciles the expanded stored-pagination implementation while
  retaining signed-v2 expiry, context binding, domain separation, rotation
  seams, uniform errors, compatibility and remaining key/TTL/preference holds.
- The two implementation paths retain the original v2 contract and add later
  reviewed key-copy, codec-readiness, canonical-JSON and contradictory-source
  checks. The descriptive manifest updates retain disabled/HOLD state.

The accepted feed source `5f46b0cfeadda140b1b29d120766100d7e3da735` has live
**9/9 shared successes** in run44715092592 and **1/1 dedicated success** in
run44715092710. These are provider records for the accepted feed composition,
not a newly executed test run or qualification of PR80's old source.
[historical-pr-proof.json](historical-pr-proof.json) records those jobs and PR
states; [verification.json](verification.json) records ancestry and all eight
file identities. No new merge or module activation was needed for this cleanup.

The other 16 open PRs are Core consolidation/repair, audits or held policy
alternatives. This review makes no recommendation to close them before the
current Core candidate is integrated and its inclusion is verified.

## Evidence verification

`SHA256SUMS` covers every file in this directory except itself. Run
`sha256sum -c SHA256SUMS` from this directory. These JSON receipts preserve
individual source identities, job conclusions, refs and comparisons. They do
not replace the original evidence or convert pending, failed or skipped checks
into passes.
