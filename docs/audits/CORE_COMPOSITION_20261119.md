# Combined core review and qualification checkpoint

Recorded 2026-11-19 UTC. This checkpoint composes the compatible core streams in
[PR #90](https://github.com/djshellshoxxx/circula/pull/90). Publication and full
provider qualification are separate steps. Main was still
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` at this review; no new main merge or
release is claimed. The current PR head, its checks and the eventual normal
merge receipt govern adoption.

## Exact source and preserved history

| Source | Reviewed head | Preserved contribution |
|---|---|---|
| Original combined core, PR90 | `f25d10b6b4085b80d7de53ecbb55efffeb9d8969` | PR #78/#82/#87/#88/#89 ancestry; credential-proof repairs, immutable dispute-policy binding, exact ledger aggregates, historical policy compatibility, recovery throttling, onboarding, bounded sessions/tokens and corrected session links |
| Core recovery, PR99 | `47446b33b01f4993a1caa406516746858bbf0a11` | Token mutation authority inside the write transaction, rollback/discard recovery, migration error handling, UUIDv7 allocation and retained query-plan assertions |
| Later PR99 preservation merge | `0a73ee42b96255601cf2227b1ddf6599ec07a0ef` | Original PR96 history, earlier PR101/102 ancestry, 71 added evidence/documentation files and the updated continuation record |
| Session completion, PR101 | `f7ac6d22a1bd1718f6f4e3a840da7a28c83d0cb0` | Previous/First navigation, current-preserving bulk revocation, atomic MFA/audit handling, migration 0021, canonical/bounded cursors, insertion regression, contracts and browser/fuzz fixtures |
| Export/UI recovery, PR102 | `f29ebb01ff681a1276f385b8b568e41355b83505` | Snapshot authorization, aborted incomplete responses including real-client regression, listing lookahead, validation/marketplace/browser catalogue and duplicate-definition checking |
| Export verifier, PR104 | `c3c991aa3c12a11edce2874e9444b9d2a99f746c` | Unambiguous ZIP/manifest membership, safe paths and file types, bounded reads, actual size/content verification and Go/Python regressions |
| Administrator reset and requirement evidence | local `83f1d981cc78599e80c218a7ee9846e84151d94b` and `12d6fabf649fc80403d26616a30136f5b9753e20` | Complete reset messages and escaping/ARIA regressions; honest requirement citations and SEC-031 status correction |

The reviewed runtime checkpoint is local `65242f32afab0b47e836e8fbe37cd323d3df75e7`,
tree `5cb5698c619f9a1028778ed241bee2d881adc07d`. After the branch inventory and
later PR99 evidence merge, local `ec0efb8a818378c9233b7ec6da1b7598da24c657` has tree
`79d3284c6890c65635adefcf5262e320271409d3`. Every non-documentation path is identical
between these two checkpoints. Local identifiers describe engineering provenance;
they are not represented as remotely retrievable commits unless publication
preserves that exact object.

The initial five source PR heads and all reviewed concurrent core heads remain
ancestors of the local composition. No original branch is rewritten or deleted.
Git-data publication must compare the complete local and remote tree, retain
the remote source parents and advance PR90 only from its freshly checked old
head without force.

## Conflict resolutions and security review

Independent security and branch reviewers read the source and regression diffs.
The security review found all 36 selected critical blobs identical to their
approved contributing heads. The ledger reviewer independently checked the
resolved queries and token mutation authority. Receipts are retained under
[evidence/final-core-20261119](evidence/final-core-20261119/).

1. **Session queries:** preserve all six query-plan cases: first, continuation,
   previous, owned lookup, bulk revocation and token inventory. Each retains its
   expected indexed `SEARCH` and rejection of full `SCAN`/temporary B-tree plans.
   Both token write paths still call live authority validation inside their
   serialized write unit. Core migrations are 0001–0021.
2. **Catalogue:** retain all earlier onboarding/session messages and use the
   broader recovered validation/admin/marketplace namespace. Removing the 14
   duplicate administrator-reset keys leaves 511 distinct keys after PR102;
   PR101 adds 14 session keys for 525. There are no conflicting retained values.
   The shared 403 reset message is `admin.reset.forbidden`, which correctly covers
   self-reset and other prohibited targets; the narrower self-reset-only message
   would misdescribe some refusals. Both reset regression families remain.
3. **Export authority and interruption:** authorization is repeated inside the
   data snapshot before any sink write. Role/MFA/break-glass/erasure eligibility
   is checked there; eligible suspended/departed members retain own-history
   access. This is not a promise of continuous cancellation after a snapshot
   opens. Stream/finalization errors propagate HTTP abort rather than producing
   a misleading successful partial ZIP. The final composition keeps the newer
   real-client regression from PR102.
4. **Export verification:** keep PR104's newer validators and boundary tests.
   Declared archive membership, schema, paths, regular-file metadata, bounded
   data and actual lengths must agree; Python verifies complete physical member
   content before accepting the declared length. Ordinary empty file lists
   remain supported.
5. **Later PR99 merge:** only documentation/evidence is added to the approved
   runtime. Its three conflicts were the generated traceability file, catalogue
   and message documentation. The current corrected catalogue/documentation
   were retained and traceability is regenerated from the final source. All
   historical evidence keeps its source/timeout/incomplete-run limits. One
   archived Python EOF blank was removed with unchanged syntax tree; its original
   hash and parent commit are recorded alongside updated copy checksums.

The future tree is unchanged from main, with no Go import into a future module.
The 15 independently developed modules remain separate. GOV-021 and SEC-026 are
technical refreshes on isolated branches; their feature migration must follow
core 0021 as 0022, and their existing product/authority decisions remain open.

## Executed qualification and its limits

| Check | Observed result | Source/scope limit |
|---|---|---|
| All catalogue, financial, session and support templates using actual production web files | Race PASS, 24.517s | Local `d836a3a` before the documentation-only PR99 merge; full command/file list in `focused-tests.json`. Excludes SQLite-backed handler integration. |
| Complete message package | Race PASS, 1.055s | Same focused source; includes the reconciled525-key catalogue and both reset regression families. |
| Traceability generator unit suite | Race PASS, 8.056s | Same focused source; includes duplicate-definition behavior. Generated-document freshness subsequently passed in 1.201s, with 531 IDs, 15 uncited built rows, zero unknown IDs and zero duplicates. |
| Python export reader | 22 tests PASS, 11.981s | Executed after PR104 integration; its reader/test blobs are unchanged by the later core merges. |
| Production browser-message script | 7 Node tests PASS, zero failures/skips | Executed after PR102 catalogue reconciliation; script/test blobs are unchanged by subsequent merges. |
| Session SQL scaling | 21 migrations and integrity/FK checks PASS at 100, 10,000 and 100,000 retained rows | Python SQLite 3.53.1 using production SQL, not the Go driver. First/deep/previous/direct work was 316/449/443/22 VM instructions; repeated bulk revoke was 21. Initial bulk work scales with rows changed. |
| Session service pagination/position tests | PASS, 1.794s | Production service files at reviewed PR101 predecessor `baedf4b`; no SQLite adapter. Latest PR101 preserves these production blobs and adds the committed-insert regression. |
| Earlier native Windows full suite | PASS at PR90 `f25d10b` | Run 44506774451 / job 135500348332; validates the previous session-link correction, not this newer composition. |
| Full local combined Go race attempt | Interrupted, exit 130; no terminal package/test result | Attempt at `8c47cdf` was stopped after an environment reset invalidated the inspected warm-cache state. It is not a pass or a product-assertion failure. |
| Final static checks | PASS | All 117 changed Go files formatted; generated traceability current; action pins/permissions, diff whitespace and repository hygiene passed. |
| Full provider suite at the published combined head | PENDING at this checkpoint | Linux race/property/migration, native macOS/Windows, browser, lint/vulnerability, API/load, bounded fuzz, builds and reproducibility remain required. |

Earlier failed, incomplete and superseded runs remain recorded. A check name,
test citation, source review, successful package subset or skipped optional
native placeholder cannot qualify the final combined head. The normal merge
operation must respect any required provider checks/reviews; the unreadable
legacy protection configuration is not presumed absent.

## Inventory, completed merges and remaining work

The initial 107 real branch refs remain preserved. The
[concurrent snapshot](CONCURRENT_BRANCH_SNAPSHOT_20261119.md) records 122 refs at
11:16:04 UTC and a later 123-ref cutoff at 11:27:08 UTC. These are dated additive
inventories, not evidence that every later incoming branch has completed
runtime qualification. Newly observed future work receives a separate review
and remains isolated.

Two ordinary topic merges are observed: PR #91 at
`0d480a5b348f3d385df0dc9c9a3e43a0b03a8332` into the GOV-021 topic, and PR #93 at
`797dddf21adddfb34a49e7bda1871a202968dcda` into notifications. Both exact resulting
trees were compared with their qualified source trees. Neither is a new main
merge. Original branches and unmerged work remain preserved.

The catalogue/validation/marketplace/browser-warning continuation is implemented,
but full UX-047 still includes other templates, API/service message text and
complete locale handling. Three missing automated citations were supplied and
SEC-031 was corrected to partial because an approved reporting/contact channel
is still absent; 15 built rows remain without a test citation. Their inspection,
manual and deployment criteria are classified in the evidence review rather
than covered by vacuous tests. Specification declarations are not completion
percentages or passing test counts.

Next adoption steps are to publish this exact composed tree, obtain its full
current-head checks, fix any actual failures, and use a normal merge if required
reviews and repository policy permit. Separately qualify future/topic repairs
on their own targets. The remaining owner, manual, external review and release
gates stay explicit in [REMAINING_BLOCKERS.md](REMAINING_BLOCKERS.md). No deployment,
release tag, history rewrite or change to historical financial records is part
of this checkpoint.
